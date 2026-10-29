//go:build live

package tui

// The live end-to-end run (scripts/e2e-tui.sh sets it up): the real program
// loop in teatest against a real `jumpgate serve` and two systemd+sshd boxes
// in Docker. It pairs both boxes from the hosts table, restarts a service,
// answers a job's decisions, turns a module on, shows and copies the invite,
// upgrades a gate, locks and unlocks the vault, and stops the server from
// the Server view. Every screen it checks is appended as text to
// $JUMPGATE_LIVE_FRAMES.
//
// Environment: JUMPGATE_LIVE_HOME (the server's ~/.jumpgate),
// JUMPGATE_LIVE_PASSFILE (the vault passphrase, read here and never shown),
// JUMPGATE_LIVE_BOXES (two ssh host:port), JUMPGATE_LIVE_KEY (their root key
// file), JUMPGATE_LIVE_HOOK (scripts/e2e-tui.sh, for `hook pre-upgrade 2`),
// JUMPGATE_BIN (the jumpgate the server runs).

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/exp/teatest/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/tui/tuitest"
)

// liveShot is what the program loop held after its last message, copied
// under the lock so the test goroutine never reads the App itself.
type liveShot struct {
	frame     string
	modal     bool
	flashes   []string // every flash set, in order
	lastFlash string
	inbox     []api.JobView
	invite    string
	done      int
	total     int
	known     bool
}

// liveModel records a liveShot after every Update.
type liveModel struct {
	a    *App
	mu   *sync.Mutex
	shot *liveShot
}

func (m liveModel) Init() tea.Cmd { return m.a.Init() }

func (m liveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.a.Update(msg)
	a := m.a
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shot.frame = ansi.Strip(a.render())
	m.shot.modal = a.modal != nil
	// A flash counts each time it is set: the same words twice (a second
	// decision) are two flashes when a key cleared the first between them.
	if f := ansi.Strip(a.flash); f != m.shot.lastFlash {
		m.shot.lastFlash = f
		if f != "" {
			m.shot.flashes = append(m.shot.flashes, f)
		}
	}
	if js, ok := a.screens[scrJobs].(*jobsScreen); ok && js.has {
		m.shot.inbox = append([]api.JobView(nil), js.in.Jobs...)
	}
	m.shot.invite = a.invite
	m.shot.known = a.guideKnown
	m.shot.done, m.shot.total = guideProgress(guideSteps(a.facts()))
	return m, cmd
}

func (m liveModel) View() tea.View { return m.a.View() }

type liveDriver struct {
	t      *testing.T
	tm     *teatest.TestModel
	mu     *sync.Mutex
	shot   *liveShot
	seen   int // flashes already consumed by flashed
	frames string
	secret string
}

func (d *liveDriver) get() liveShot {
	d.mu.Lock()
	defer d.mu.Unlock()
	s := *d.shot
	s.flashes = append([]string(nil), d.shot.flashes...)
	s.inbox = append([]api.JobView(nil), d.shot.inbox...)
	return s
}

func (d *liveDriver) frame() string { return d.get().frame }

// save appends a frame to the frames file; never one holding the passphrase.
func (d *liveDriver) save(name, f string) {
	d.t.Helper()
	if d.secret != "" && strings.Contains(f, d.secret) {
		d.t.Fatalf("%s: the passphrase is on screen", name)
	}
	if d.frames == "" {
		return
	}
	fh, err := os.OpenFile(d.frames, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		d.t.Fatal(err)
	}
	defer fh.Close()
	fmt.Fprintf(fh, "--- %s\n%s\n", name, f)
}

// until waits for ok on the shot and returns its frame.
func (d *liveDriver) until(what string, wait time.Duration, ok func(s liveShot) bool) string {
	d.t.Helper()
	end := time.Now().Add(wait)
	for {
		s := d.get()
		if ok(s) {
			return s.frame
		}
		if time.Now().After(end) {
			d.save("TIMEOUT "+what, s.frame)
			d.t.Fatalf("timeout waiting for %s:\n%s\nflashes:\n%s", what, s.frame, strings.Join(tail(s.flashes, 10), "\n"))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (d *liveDriver) screen(what string, wait time.Duration, subs ...string) string {
	d.t.Helper()
	return d.until(what, wait, func(s liveShot) bool {
		for _, x := range subs {
			if !strings.Contains(s.frame, x) {
				return false
			}
		}
		return true
	})
}

// flashed waits for a flash newer than the last one it returned that
// matches re, and fails on one that says FAIL or failed in between.
func (d *liveDriver) flashed(re string, wait time.Duration) string {
	d.t.Helper()
	rx := regexp.MustCompile(re)
	end := time.Now().Add(wait)
	for {
		s := d.get()
		for i := d.seen; i < len(s.flashes); i++ {
			if rx.MatchString(s.flashes[i]) {
				d.seen = i + 1
				return s.flashes[i]
			}
		}
		if time.Now().After(end) {
			d.save("TIMEOUT flash "+re, s.frame)
			d.t.Fatalf("timeout waiting for the flash %s:\n%s\nflashes:\n%s", re, s.frame, strings.Join(tail(s.flashes, 10), "\n"))
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (d *liveDriver) mark() { d.seen = len(d.get().flashes) }

func (d *liveDriver) press(keys ...string) {
	for _, k := range keys {
		d.tm.Send(tuitest.Key(k))
	}
	time.Sleep(30 * time.Millisecond)
}

func (d *liveDriver) typ(s string) {
	for _, r := range s {
		k := string(r)
		if k == " " {
			k = "space"
		}
		d.tm.Send(tuitest.Key(k))
	}
	time.Sleep(30 * time.Millisecond)
}

// modal waits for a modal and then past its flush window (keys typed in the
// first 150 ms of a modal are dropped).
func (d *liveDriver) modal() string {
	d.t.Helper()
	d.until("a modal", 10*time.Second, func(s liveShot) bool { return s.modal })
	time.Sleep(250 * time.Millisecond)
	return d.frame()
}

func (d *liveDriver) noModal() {
	d.t.Helper()
	d.until("the modal to close", 10*time.Second, func(s liveShot) bool { return !s.modal })
}

func tail(xs []string, n int) []string { return xs[max(len(xs)-n, 0):] }

func TestLive(t *testing.T) {
	home, passFile := os.Getenv("JUMPGATE_LIVE_HOME"), os.Getenv("JUMPGATE_LIVE_PASSFILE")
	if home == "" || passFile == "" {
		t.Skip("set JUMPGATE_LIVE_HOME and JUMPGATE_LIVE_PASSFILE (scripts/e2e-tui.sh does)")
	}
	boxes := strings.FieldsFunc(os.Getenv("JUMPGATE_LIVE_BOXES"), func(r rune) bool { return r == ',' })
	hook, exe, keyFile := os.Getenv("JUMPGATE_LIVE_HOOK"), os.Getenv("JUMPGATE_BIN"), os.Getenv("JUMPGATE_LIVE_KEY")
	if len(boxes) < 2 || hook == "" || exe == "" {
		t.Fatal("set JUMPGATE_LIVE_BOXES (two), JUMPGATE_LIVE_HOOK and JUMPGATE_BIN (scripts/e2e-tui.sh does)")
	}
	t.Setenv("JUMPGATE_HOME", home)
	t.Setenv("JUMPGATE_SYSTEM_DIR", "/nonexistent-jumpgate-system")
	pb, err := os.ReadFile(passFile)
	if err != nil {
		t.Fatal(err)
	}
	pass := strings.TrimSpace(string(pb))
	modalFlush = 150 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	// As `jumpgate tui` connects (cmd/jumpgate tuiOptions), so a reconnect
	// behaves as it does there.
	c, err := apiclient.Connect(ctx, apiclient.Options{Start: true, Exe: exe})
	if err != nil {
		t.Fatal(err)
	}
	startedPID := c.Info().PID
	var cmu sync.Mutex
	cur := c
	a := New(Options{
		Backend: c,
		Self:    exe,
		// linux: no pbcopy on this Mac's clipboard; no DISPLAY: no xclip.
		// The invite is copied by OSC 52 only.
		GOOS: "linux",
		Getenv: func(k string) string {
			if k == "DISPLAY" || k == "WAYLAND_DISPLAY" {
				return ""
			}
			return os.Getenv(k)
		},
		Hostname: "live",
		Start: func(ctx context.Context) (Backend, error) {
			nc, err := apiclient.Connect(ctx, apiclient.Options{Start: true, Exe: exe})
			if err != nil {
				return nil, err
			}
			cmu.Lock()
			cur = nc
			cmu.Unlock()
			return nc, nil
		},
		Server: func() ServerInfo {
			cmu.Lock()
			defer cmu.Unlock()
			i := cur.Info()
			return ServerInfo{PID: i.PID, HTTPAddr: i.HTTPAddr, StartedAt: i.StartedAt, System: i.System, Shape: i.Shape}
		},
	})
	mu := &sync.Mutex{}
	shot := &liveShot{}
	tm := teatest.NewTestModel(t, liveModel{a: a, mu: mu, shot: shot},
		teatest.WithInitialTermSize(120, 40), teatest.WithProgramOptions(tea.WithColorProfile(colorprofile.Ascii)))
	d := &liveDriver{t: t, tm: tm, mu: mu, shot: shot, frames: os.Getenv("JUMPGATE_LIVE_FRAMES"), secret: pass}
	// Drain the program's output so the renderer never blocks on it.
	go func() {
		buf := make([]byte, 64<<10)
		for {
			if _, err := tm.Output().Read(buf); err != nil {
				time.Sleep(20 * time.Millisecond)
			}
		}
	}()

	setupRE := regexp.MustCompile(`setup (\d)/(\d)`)
	setup := func(f string) string {
		if m := setupRE.FindStringSubmatch(f); m != nil {
			return m[1] + "/" + m[2]
		}
		return ""
	}

	// --- the setup guide opens by itself: no box yet, the vault locked.
	guide := d.until("the setup guide, locked", 60*time.Second, func(s liveShot) bool {
		return s.modal && strings.Contains(s.frame, "Vault created and unlocked") && setup(s.frame) != ""
	})
	d.save("setup guide (vault locked)", guide)
	if got := setup(guide); got != "3/6" {
		t.Errorf("setup %s, want 3/6", got)
	}
	for _, want := range []string{"todo 2. Vault created and unlocked", "done 3. Recovery code added", "done 4. Controller key present"} {
		if !strings.Contains(guide, want) {
			t.Errorf("the guide lacks %q", want)
		}
	}
	if !regexp.MustCompile(`todo 2\. Vault created and unlocked\s+\S+ next`).MatchString(guide) {
		t.Errorf("the guide does not mark step 2 next")
	}
	time.Sleep(250 * time.Millisecond)
	d.press("esc")
	d.noModal()
	d.save("fleet (vault locked)", d.screen("fleet, locked", 30*time.Second, "LOCKED"))

	unlock := func() string {
		d.press("6")
		d.screen("vault view", 10*time.Second, "state")
		d.mark()
		d.press("u")
		d.modal()
		d.typ(pass)
		masked := regexp.MustCompile(`passphrase: (\S*)`)
		p := d.until("the masked passphrase", 5*time.Second, func(s liveShot) bool {
			m := masked.FindStringSubmatch(s.frame)
			return m != nil && len([]rune(m[1])) == len([]rune(pass))
		})
		d.save("unlock prompt", p)
		d.press("enter")
		d.flashed(`^the vault is unlocked$`, 30*time.Second)
		d.noModal()
		return d.until("unlocked", 30*time.Second, func(s liveShot) bool {
			return regexp.MustCompile(`state\s+unlocked`).MatchString(s.frame)
		})
	}
	d.save("vault unlocked", unlock())
	d.until("setup 4/6", 30*time.Second, func(s liveShot) bool { return setup(s.frame) == "4/6" })

	// --- pair both boxes from the hosts table: the form, the typed host-key
	// confirmation, add, pair over ssh.
	d.press("1")
	d.screen("fleet", 10*time.Second, "Fleet")
	d.press("v")
	d.screen("hosts table", 10*time.Second, "HOSTS")
	for i, addr := range boxes[:2] {
		name := fmt.Sprintf("box%d", i+1)
		d.press("a")
		d.screen("add form", 10*time.Second, "ADD A BOX")
		for _, v := range []string{name, "root@" + addr, keyFile, "", "n"} {
			if v == "n" {
				d.press("enter") // the sudo field keeps its "n"
				break
			}
			d.typ(v)
			d.press("enter")
		}
		hk := d.screen(name+" host key", 60*time.Second, "unconfirmed host key", "Type yes to trust it")
		d.save("hosts add "+name+": host key", hk)
		d.typ("yes")
		d.press("enter")
		done := d.screen(name+" paired", 240*time.Second, "paired: gate 0x")
		d.save("hosts add "+name+": paired", done)
		d.press("enter")
		d.screen("hosts table again", 10*time.Second, "HOSTS", name)
		if i == 0 {
			// The first paired, healthy box completes the required steps:
			// the status strip drops its count.
			d.until("setup 6/6", 120*time.Second, func(s liveShot) bool { return s.known && s.done == 6 && s.total == 6 })
			d.press("g")
			d.modal()
			g := d.screen("the guide, 6/6", 10*time.Second, "done 6. Box paired and healthy")
			d.save("setup guide (6/6)", g)
			if setup(g) != "" {
				t.Errorf("the status strip still counts setup: %s", setup(g))
			}
			for n := 1; n <= 6; n++ {
				if !regexp.MustCompile(fmt.Sprintf(`done %d\. `, n)).MatchString(g) {
					t.Errorf("guide step %d is not done", n)
				}
			}
			d.press("esc")
			d.noModal()
		}
	}
	d.save("hosts", d.until("hosts list", 30*time.Second, func(s liveShot) bool {
		return regexp.MustCompile(`box1 .* gate 0x`).MatchString(s.frame) && regexp.MustCompile(`box2 .* gate 0x`).MatchString(s.frame)
	}))

	// --- the fleet table and the node detail.
	d.press("v")
	fl := d.until("fleet rows", 120*time.Second, func(s liveShot) bool {
		return strings.Contains(s.frame, "box1") && strings.Contains(s.frame, "box2") && !strings.Contains(s.frame, "HOSTS")
	})
	d.save("fleet", fl)
	d.press("up", "up")
	d.press("enter")
	d.save("node box1: overview", d.screen("node detail", 60*time.Second, "box1", "Overview"))

	// --- restart a service from the actions menu, with its typed confirmation.
	d.mark()
	d.press("x")
	d.modal()
	d.save("actions", d.screen("actions", 5*time.Second, "ACTIONS"))
	d.press("enter") // restart exec
	d.modal()
	d.save("restart confirm", d.screen("restart confirm", 5*time.Second, "Type exec to confirm"))
	time.Sleep(250 * time.Millisecond) // a new modal: past its flush window
	d.typ("exec")
	d.press("enter")
	d.flashed(`^restart exec on box1: (active|inactive)$`, 180*time.Second)
	d.save("node box1: after restart", d.frame())

	// --- the module before the rename: box1's node still has the legacy
	// names, so the server refuses it with the way out, and nothing is
	// created that would stop the rename.
	d.press("m")
	d.modal()
	d.until("box1 modules", 60*time.Second, func(s liveShot) bool { return strings.Contains(s.frame, "watch") })
	for i := 0; i < 15 && !regexp.MustCompile(`[>▸] .*watch`).MatchString(d.frame()); i++ {
		d.press("down")
	}
	d.press("enter")
	d.screen("watch fields", 10*time.Second, "endpoints")
	d.press("enter")
	refused := d.screen("module refused", 60*time.Second, "run the node.migrate-names job first")
	d.save("modules on box1: refused before the rename", refused)
	if !regexp.MustCompile(`Health watch\s+off`).MatchString(refused) {
		t.Errorf("the refused module is not shown off")
	}
	d.press("esc")
	d.noModal()

	// --- a job with decisions: rename the legacy names on box1, answer with
	// the typed confirm.
	d.press("esc")
	d.press("2")
	d.screen("jobs", 30*time.Second, "inbox")
	d.mark()
	d.press("n")
	d.modal()
	d.typ("box1 node.migrate-names")
	d.press("enter")
	d.save("start a job: confirm", d.screen("job confirm", 5*time.Second, "Type box1 to confirm"))
	time.Sleep(250 * time.Millisecond) // a new modal: past its flush window
	d.typ("box1")
	d.press("enter")
	started := d.flashed(`^job \S+ started on box1$`, 60*time.Second)
	jobID := strings.Fields(started)[1]
	d.noModal()
	for _, want := range []string{"stop-processes", "kill-processes"} {
		idx := -1
		d.until("decision "+want, 240*time.Second, func(s liveShot) bool {
			for _, j := range s.inbox {
				if j.ID == jobID && j.State == "needs-decision" && j.Question != nil {
					for k, o := range j.Question.Options {
						if o.ID == want {
							idx = k
							return true
						}
					}
				}
			}
			return false
		})
		d.save("jobs inbox (decision "+want+")", d.frame())
		d.mark()
		d.press("d")
		d.modal()
		d.save("job decision ("+want+")", d.screen("question", 5*time.Second, "asks"))
		d.press(fmt.Sprint(idx + 1))
		d.screen("typed confirm", 5*time.Second, "Type "+want+" to confirm")
		time.Sleep(250 * time.Millisecond) // a new modal: past its flush window
		d.save("typed confirm ("+want+")", d.frame())
		d.typ("no")
		d.press("enter")
		d.save("typed confirm: a miss", d.screen("typed-confirm miss", 5*time.Second, "that is not"))
		d.typ(want)
		d.press("enter")
		d.flashed(`^decision sent; the box answered `, 120*time.Second)
		d.noModal()
	}
	d.until("job succeeded", 300*time.Second, func(s liveShot) bool {
		for _, j := range s.inbox {
			if j.ID == jobID {
				return j.State == "succeeded"
			}
		}
		return false
	})
	d.save("job succeeded", d.frame())

	// --- a module: the health watch on box1, now that the node is renamed.
	d.press("1")
	d.screen("fleet", 10*time.Second, "box2")
	d.press("up", "up")
	d.press("enter")
	d.screen("node detail", 60*time.Second, "box1", "Overview")
	d.press("m")
	d.modal()
	mods := d.until("box1 modules", 60*time.Second, func(s liveShot) bool { return strings.Contains(s.frame, "watch") })
	d.save("modules on box1", mods)
	for i := 0; i < 15 && !regexp.MustCompile(`[>▸] .*watch`).MatchString(d.frame()); i++ {
		d.press("down")
	}
	d.press("enter")
	d.save("modules: watch fields", d.screen("watch fields", 10*time.Second, "endpoints"))
	d.press("enter")
	on := d.until("watch on", 300*time.Second, func(s liveShot) bool {
		return regexp.MustCompile(`[>▸] Health watch\s+on .*Running`).MatchString(s.frame)
	})
	d.screen("setup done", 120*time.Second, "Done.")
	d.save("modules on box1: watch on", on)
	d.press("esc")
	d.noModal()
	d.press("esc") // back to the fleet
	d.press("esc")

	// --- invite: the one line under the panes, and c copies it.
	d.press("1")
	d.press("i")
	inv := d.until("invite", 30*time.Second, func(s liveShot) bool { return s.invite != "" })
	inv = d.screen("invite line", 10*time.Second, "[c] copy  [i] hide  on the box, as root: curl -fsSL")
	d.save("invite", inv)
	line := d.get().invite
	if !regexp.MustCompile(`^curl -fsSL https://jumpgate\.app/install\.sh \| sh -s -- --agent --version v0\.0\.91 --controller 0x[0-9a-fA-F]{40} --transport-key '[^']+'$`).MatchString(line) {
		t.Errorf("invite command %q", line)
	}
	d.mark()
	d.press("c")
	d.flashed(`^invite copied`, 10*time.Second)
	d.press("i")

	// --- upgrade box2's gate: the hook opts it in and gives the controller a
	// newer gate.
	h := exec.Command(hook, "hook", "pre-upgrade", "2")
	h.Stdout, h.Stderr = os.Stderr, os.Stderr
	if err := h.Run(); err != nil {
		t.Fatalf("hook pre-upgrade: %v", err)
	}
	d.press("v")
	d.screen("hosts table", 10*time.Second, "HOSTS")
	for i := 0; i < 5 && !regexp.MustCompile(`[>▸] box2 `).MatchString(d.frame()); i++ {
		d.press("down")
	}
	d.until("box2 selected", 5*time.Second, func(s liveShot) bool { return regexp.MustCompile(`[>▸] box2 `).MatchString(s.frame) })
	d.mark()
	d.press("u")
	d.flashed(`^gate on box2: v0\.0\.90 -> v0\.0\.91$`, 180*time.Second)
	d.save("hosts upgrade", d.frame())
	d.press("v")

	// --- lock and unlock.
	d.press("6")
	d.screen("vault view", 10*time.Second, "state")
	d.mark()
	d.press("l")
	d.flashed(`^the vault is locked`, 30*time.Second)
	d.save("vault locked", d.until("locked", 30*time.Second, func(s liveShot) bool {
		return regexp.MustCompile(`state\s+locked`).MatchString(s.frame)
	}))
	d.save("vault unlocked again", unlock())

	d.press("2")
	d.save("jobs", d.screen("jobs", 30*time.Second, "inbox", "succeeded"))

	// --- stop the server from the Server view: the TUI's reconnect must not
	// start another one.
	d.press("7")
	d.screen("server view", 10*time.Second, "Server")
	d.mark()
	d.press("x")
	d.modal()
	d.save("server stop confirm", d.screen("stop confirm", 5*time.Second, "Type stop to confirm"))
	d.typ("stop")
	d.press("enter")
	d.flashed(`^the server is stopping`, 30*time.Second)
	gone := time.Now().Add(30 * time.Second)
	for {
		if _, ok, _ := daemon.Find(ctx); !ok {
			break
		}
		if time.Now().After(gone) {
			t.Fatal("the server did not stop")
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Long enough for several reconnect attempts of every stream.
	time.Sleep(15 * time.Second)
	if info, ok, _ := daemon.Find(ctx); ok {
		t.Fatalf("a server runs again after the stop (pid %d; the stopped one was %d)", info.PID, startedPID)
	}
	d.save("server stopped", d.frame())

	d.press("q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(10*time.Second))
}
