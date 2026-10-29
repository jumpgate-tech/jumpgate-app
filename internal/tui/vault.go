package tui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
)

// The Vault view (inventory B): its state, slots and idle lock; unlock and
// lock; a token put through a masked field; and the vault commands, which
// re-run this jumpgate in the foreground (runSelfPause) so passphrases and a
// printed recovery code stay on the real terminal.

var vaultKeys = struct{ Unlock, Lock, Idle, Put, Init, Migrate, Backup, Restore, Recovery, Rekdf, SealTPM, SealHost key.Binding }{
	Unlock:   key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "unlock")),
	Lock:     key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "lock")),
	Idle:     key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "idle lock minutes")),
	Put:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "put relay-token or admin-token")),
	Init:     key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "create the vault")),
	Migrate:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move plain secrets into the vault")),
	Backup:   key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "back up to a file")),
	Restore:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restore from a file, typed confirmation")),
	Recovery: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add a recovery code")),
	Rekdf:    key.NewBinding(key.WithKeys("K"), key.WithHelp("K", "raise the KDF cost")),
	SealTPM:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "seal with the TPM2 (sudo)")),
	SealHost: key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "seal with the host key (sudo)")),
}

// vaultSecrets are the names PUT /api/vault/secrets takes.
var vaultSecrets = []string{"relay-token", "admin-token"}

const maxIdleLock = 1440

type (
	vaultSettingsMsg struct {
		gen uint64
		s   api.Settings
		err error
	}
	vaultPutMsg struct {
		name string
		err  error
	}
)

type vaultScreen struct {
	linux   bool // synced from the App: keys() has no App
	idle    int
	idleHas bool
	idleErr error
	note    string // the last action's outcome, never a secret
	visit
}

func (s *vaultScreen) capturing() bool { return false }

func (s *vaultScreen) keys() []key.Binding {
	ks := []key.Binding{vaultKeys.Unlock, vaultKeys.Lock, vaultKeys.Idle, vaultKeys.Put, vaultKeys.Init, vaultKeys.Migrate,
		vaultKeys.Backup, vaultKeys.Restore, vaultKeys.Recovery, vaultKeys.Rekdf}
	if s.linux {
		ks = append(ks, vaultKeys.SealTPM, vaultKeys.SealHost)
	}
	return ks
}

func (s *vaultScreen) load(a *App) tea.Cmd {
	s.begin(a.ctx)
	gen, ctx := s.gen, s.ctx
	return tea.Batch(loadGuide(a), a.do("settings", func(_ context.Context, be Backend) tea.Msg {
		st, err := be.Settings(ctx)
		return vaultSettingsMsg{gen: gen, s: st, err: err}
	}))
}

func (s *vaultScreen) update(a *App, msg tea.Msg) tea.Cmd {
	s.linux = a.o.GOOS == "linux"
	switch msg := msg.(type) {
	case enterMsg:
		return s.load(a)
	case leaveMsg:
		s.invalidate()
	case vaultSettingsMsg:
		if !s.current(msg.gen) {
			return nil
		}
		s.idleErr = msg.err
		if msg.err == nil {
			s.idle, s.idleHas = msg.s.VaultIdleLockMinutes, true
		}
	case vaultPutMsg:
		if msg.err != nil {
			s.note = "FAIL store " + msg.name + ": " + a.errText(msg.err)
		} else {
			s.note = "stored " + msg.name + " in the vault"
		}
		a.flash = s.note
		return loadGuide(a)
	case tea.KeyPressMsg:
		if a.active != scrVault || a.detail != nil {
			return nil
		}
		return s.key(a, msg)
	}
	return nil
}

func (s *vaultScreen) key(a *App, k tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(k, vaultKeys.Unlock):
		a.modal = newUnlock()
	case key.Matches(k, vaultKeys.Lock):
		return a.lockVault()
	case key.Matches(k, vaultKeys.Idle):
		cur := ""
		if s.idleHas {
			cur = strconv.Itoa(s.idle)
		}
		a.modal = newAsk("Idle lock", "Minutes without use before the vault locks: 0 (off) to 1440.", cur, func(a *App, v string) (string, tea.Cmd) {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > maxIdleLock || strings.TrimLeft(v, "0123456789") != "" {
				return "the idle lock is 0 to 1440 minutes", nil
			}
			return "", s.saveIdle(a, n)
		})
	case key.Matches(k, vaultKeys.Put):
		a.modal = newAsk("Put a secret in the vault", "Which one: relay-token or admin-token.", "", func(a *App, name string) (string, tea.Cmd) {
			if !slices.Contains(vaultSecrets, name) {
				return "the name is relay-token or admin-token", nil
			}
			a.modal = newSecret("Put "+name, "The value goes to the unlocked vault and is never shown or logged.", name, false, func(a *App, v []byte) tea.Cmd {
				return putSecret(a, name, v)
			})
			return "", nil
		})
	case key.Matches(k, vaultKeys.Init):
		return a.runSelfPause("vault", "init")
	case key.Matches(k, vaultKeys.Migrate):
		return a.runSelfPause("vault", "migrate")
	case key.Matches(k, vaultKeys.Backup):
		a.modal = newAsk("Back up the vault", "The file to write. The backup stays encrypted.", "", func(a *App, p string) (string, tea.Cmd) {
			if why := checkPath(p); why != "" {
				return why, nil
			}
			return "", a.runSelfPause("vault", "backup", p)
		})
	case key.Matches(k, vaultKeys.Restore):
		a.modal = newAsk("Restore the vault", "The backup file to restore from.", "", func(a *App, p string) (string, tea.Cmd) {
			if why := checkPath(p); why != "" {
				return why, nil
			}
			a.modal = newConfirm("Restore the vault from "+p,
				"The vault is replaced by the backup: secrets added since it was made are lost.", "restore",
				func() tea.Cmd { return a.runSelfPause("vault", "restore", p) })
			return "", nil
		})
	case key.Matches(k, vaultKeys.Recovery):
		return a.runSelfPause("vault", "add-recovery")
	case key.Matches(k, vaultKeys.Rekdf):
		return a.runSelfPause("vault", "rekdf")
	case key.Matches(k, vaultKeys.SealTPM) && s.linux:
		return a.runSelfPause("vault", "seal", "--tpm2")
	case key.Matches(k, vaultKeys.SealHost) && s.linux:
		return a.runSelfPause("vault", "seal", "--allow-host-key")
	}
	return nil
}

func (s *vaultScreen) saveIdle(a *App, n int) tea.Cmd {
	gen, ctx := s.gen, s.ctx
	if ctx == nil {
		return nil
	}
	return a.do("idle lock", func(_ context.Context, be Backend) tea.Msg {
		st, err := be.SaveSettings(ctx, api.SettingsUpdate{VaultIdleLockMinutes: &n})
		return vaultSettingsMsg{gen: gen, s: st, err: err}
	})
}

// putSecret sends v as name and clears it and the encoded body.
func putSecret(a *App, name string, v []byte) tea.Cmd {
	path := "/api/vault/secrets/" + url.PathEscape(name)
	return a.do("put "+name, func(ctx context.Context, be Backend) tea.Msg {
		defer clear(v)
		body := secretBody("", "value", v)
		defer clear(body)
		return vaultPutMsg{name: name, err: be.Do(ctx, http.MethodPut, path, apiclient.RawJSON(body), nil)}
	})
}

func (s *vaultScreen) view(a *App, w, h int) string {
	s.linux = a.o.GOOS == "linux"
	row := func(label, v string) string { return truncate(fmt.Sprintf(" %-10s %s", label, v), w, a.gl.Ellipsis) }
	na := a.th.Dim.Render("n/a")
	var lines []string
	if !a.vaultHas {
		lines = append(lines, row("state", na))
	} else {
		v := a.vault
		st := sanitizeLine(v.State)
		switch v.State {
		case "unlocked":
			st = a.th.OK.Render(st)
		case "unusable":
			st = a.th.Bad.Render("FAIL unusable")
		case "none":
			st = a.th.Warn.Render("not created yet")
		default:
			st = a.th.Warn.Render(st)
		}
		lines = append(lines, row("state", st))
		slots := make([]string, 0, len(v.Slots))
		for _, sl := range v.Slots {
			slots = append(slots, sanitizeLine(sl))
		}
		if len(slots) == 0 {
			slots = []string{"none"}
		}
		lines = append(lines, row("slots", strings.Join(slots, ", ")))
		if r := sanitizeLine(v.Reason); r != "" {
			lines = append(lines, row("reason", maskText(r)))
		}
		if len(v.Plain) > 0 {
			var ps []string
			for _, p := range v.Plain {
				ps = append(ps, sanitizeLine(p))
			}
			lines = append(lines, row("plain", a.th.Warn.Render(strings.Join(ps, ", "))+" (not in the vault yet: m)"))
		}
	}
	idle := na
	switch {
	case s.idleErr != nil:
		idle = a.th.Bad.Render("FAIL") + " " + a.errText(s.idleErr)
	case s.idleHas && s.idle <= 0:
		idle = "off"
	case s.idleHas:
		idle = fmt.Sprintf("%d min", s.idle)
	}
	lines = append(lines, row("idle lock", idle))
	if s.note != "" {
		lines = append(lines, "", " "+truncate(s.note, max(w-1, 1), a.gl.Ellipsis))
	}
	lines = append(lines, "",
		" [u] unlock   [l] lock   [t] idle lock minutes   [p] put relay-token or admin-token",
		"",
		" "+a.th.Title.Render("VAULT COMMANDS")+a.th.Dim.Render(" (take over the terminal and ask for passphrases there)"),
		" [I] create the vault   [m] move plain secrets in   [b] back up   [r] restore",
		" [a] add a recovery code (prints once)   [K] raise the KDF cost")
	if s.linux {
		lines = append(lines, " [s] seal with the TPM2   [S] seal with the host key (sudo)")
	} else {
		lines = append(lines, a.th.Dim.Render(" Sealing for auto-unlock is for the Linux service."))
	}
	return strings.Join(lines, "\n")
}
