package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/executor"
)

func cmdHosts(args []string) int {
	if len(args) == 0 {
		return usage("usage: jumpgate hosts add|list")
	}
	switch args[0] {
	case "list":
		return hostsList(context.Background(), os.Stdout)
	case "add":
		return hostsAdd(args[1:])
	}
	return usage("unknown hosts subcommand %q", args[0])
}

// hostsList prints every target from the server, which is the one reader and
// writer of config.json (the CLI used to read the file itself).
func hostsList(ctx context.Context, w io.Writer) int {
	c, err := connect(ctx)
	if err != nil {
		return failed("%v", err)
	}
	ts, err := c.Targets(ctx)
	var e *api.Error
	if errors.As(err, &e) {
		return reportServerErrorFrom(os.Stderr, "hosts list", c.Info(), *e)
	}
	if err != nil {
		return failed("%v", err)
	}
	for _, t := range ts {
		paired := "not paired"
		if t.Agent != nil {
			paired = "agent " + t.Agent.Address
		}
		fmt.Fprintf(w, "%-16s %-6s %s\n", t.ID, t.Mode, paired)
	}
	return 0
}

func hostsAdd(args []string) int {
	const usageText = "usage: jumpgate hosts add NAME (--ssh USER@HOST[:PORT] [--key PATH] [--jump USER@HOST[:PORT]] [--sudo] | --local)"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usage(usageText)
	}
	name := args[0]
	fset := flag.NewFlagSet("hosts add", flag.ContinueOnError)
	sshArg := fset.String("ssh", "", "user@host[:port]")
	key := fset.String("key", "", "private key (optional with ssh-agent)")
	jumpArg := fset.String("jump", "", "jump host user@host[:port]")
	sudo := fset.Bool("sudo", false, "run setup commands through sudo -n")
	local := fset.Bool("local", false, "pair this machine")
	if err := fset.Parse(args[1:]); err != nil || (*local == (*sshArg != "")) {
		return usage("give exactly one of --ssh or --local")
	}

	if err := checkExistingTarget(name, *local, *sshArg, *jumpArg); err != nil {
		if errors.Is(err, errBadAddress) {
			return usage("%v", err)
		}
		return failed("%v", err)
	}

	ctx := context.Background()
	target := api.AddTarget{ID: name, Mode: "local"}
	if !*local {
		keyPath := *key
		if keyPath != "" {
			// The server resolves paths from its own working directory.
			abs, err := filepath.Abs(keyPath)
			if err != nil {
				return failed("--key: %v", err)
			}
			keyPath = abs
		}
		if _, _, _, err := parseSSHTarget(*sshArg); err != nil {
			return usage("--ssh: %v", err)
		}
		if *jumpArg != "" {
			if _, _, _, err := parseSSHTarget(*jumpArg); err != nil {
				return usage("--jump: %v", err)
			}
		}
		// The host-key prompt is the only defence against trust-on-first-use,
		// so it must be answered by a person: never from a pipe or a file.
		if !stdinIsTerminal() {
			return failed("host keys must be confirmed by a person at a terminal; stdin is not one, so nothing was trusted. Run `jumpgate hosts add` from an interactive shell")
		}
		cfg, err := sshConfigFrom(*sshArg, keyPath, *jumpArg)
		if err != nil {
			return failed("%v", err)
		}
		view := sshViewOf(cfg)
		target = api.AddTarget{ID: name, Mode: "ssh", SSH: &view}
	}

	// The server captures and records host keys, so it starts before the
	// prompt; the person at this terminal still decides.
	c, err := connect(ctx)
	if err != nil {
		return failed("%v", err)
	}
	apiclient.WarnSkew(os.Stderr, c)
	if target.SSH != nil {
		if code := confirmHostKeys(ctx, c, *target.SSH, newConsentReader(os.Stdin), os.Stdout, os.Stderr); code != 0 {
			return code
		}
	}
	if e := call(c, "/api/targets", target, nil); e != nil {
		// target_exists is the server's answer for a name already on record.
		if e.Status != http.StatusConflict || e.Code != api.CodeTargetExists {
			return reportServerErrorFrom(os.Stderr, "add "+name, c.Info(), *e)
		}
		// The name is already a target: pairing it again is how a box is
		// re-paired or an interrupted pairing finished.
		fmt.Printf("target %s already exists; pairing it again\n", name)
	}
	return streamPair(c, name, *sudo)
}

// errBadAddress marks an --ssh or --jump value that does not parse.
var errBadAddress = errors.New("bad address")

// checkExistingTarget refuses to re-run hosts add on an existing name with a
// different address. Re-pairing pairs the address on record, so accepting a
// new --ssh would confirm host keys for one box and pair another while the
// operator believes the host was re-pointed.
func checkExistingTarget(name string, local bool, sshArg, jumpArg string) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	t, ok := findTargetByID(c, name)
	if !ok {
		return nil
	}
	recorded, want := describeTarget(t), "this machine (--local)"
	if !local {
		u, h, p, err := parseSSHTarget(sshArg)
		if err != nil {
			return fmt.Errorf("%w: --ssh: %v", errBadAddress, err)
		}
		want = sshAddr(u, h, p)
		if jumpArg != "" {
			ju, jh, jp, err := parseSSHTarget(jumpArg)
			if err != nil {
				return fmt.Errorf("%w: --jump: %v", errBadAddress, err)
			}
			want += " via " + sshAddr(ju, jh, jp)
		}
	}
	if recorded == want {
		return nil
	}
	return fmt.Errorf("target %s already exists for %s, not %s; re-running hosts add re-pairs the address on record. To point %s at a new address, remove it in the web app first (which forgets its pairing), then run `jumpgate hosts add %s` again", name, recorded, want, name, name)
}

func findTargetByID(c config.Config, id string) (config.Target, bool) {
	for _, t := range c.Targets {
		if t.ID == id {
			return t, true
		}
	}
	return config.Target{}, false
}

// describeTarget renders a target's address the way checkExistingTarget
// renders the flags, so equal addresses compare equal.
func describeTarget(t config.Target) string {
	if t.Mode != "ssh" || t.SSH == nil {
		return "this machine (--local)"
	}
	s := sshAddr(t.SSH.User, t.SSH.Host, t.SSH.Port)
	if j := t.SSH.Jump; j != nil {
		s += " via " + sshAddr(j.User, j.Host, j.Port)
	}
	return s
}

func sshAddr(user, host string, port int) string {
	if port == 0 {
		port = 22
	}
	return user + "@" + net.JoinHostPort(host, strconv.Itoa(port))
}

// maxConsentLine bounds what the confirmation prompt will read while waiting
// for a newline, so an endless stdin (/dev/zero) cannot grow memory.
const maxConsentLine = 4 << 10

// newConsentReader reads at most maxConsentLine bytes in all. Past that the
// reader hits EOF without a newline, which confirmHostKeys counts as "no".
func newConsentReader(r io.Reader) *bufio.Reader {
	return bufio.NewReader(io.LimitReader(r, maxConsentLine))
}

// stdinIsTerminal reports whether stdin is a character device. A variable so
// tests can stand in for a terminal; nothing else may replace it.
var stdinIsTerminal = func() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// sshConfigFrom builds the address hosts add sends the server. It carries no
// host-key policy and no known_hosts path: the server applies Strict itself
// whenever it dials, and picks its own files.
func sshConfigFrom(login, key, jump string) (executor.SSHConfig, error) {
	user, host, port, err := parseSSHTarget(login)
	if err != nil {
		return executor.SSHConfig{}, err
	}
	cfg := executor.SSHConfig{Host: host, Port: port, User: user, KeyPath: key}
	if jump != "" {
		ju, jh, jp, err := parseSSHTarget(jump)
		if err != nil {
			return executor.SSHConfig{}, fmt.Errorf("--jump: %w", err)
		}
		cfg.Jump = &executor.SSHConfig{Host: jh, Port: jp, User: ju, KeyPath: key}
	}
	return cfg, nil
}

// sshViewOf is the request form of a dial configuration.
func sshViewOf(c executor.SSHConfig) api.SSHView {
	v := api.SSHView{Host: c.Host, User: c.User, KeyPath: c.KeyPath, Port: c.Port}
	if c.Jump != nil {
		j := sshViewOf(*c.Jump)
		v.Jump = &j
	}
	return v
}

// hostKeyAPI is the part of the server confirmHostKeys needs. Info lets an
// error from an older server (one without these routes) be reported as the
// version skew it is.
type hostKeyAPI interface {
	ProbeHostKeys(ctx context.Context, ssh api.SSHView) (api.HostKeyProbe, error)
	ConfirmHostKey(ctx context.Context, probeID, fingerprint string) error
	Info() daemon.Info
}

// confirmHostKeys shows the operator every unconfirmed host key on the path,
// the jump host first, and confirms one only on an explicit "yes". The server
// captures and records the keys, and records only the key whose fingerprint
// was shown here; consent stays at a terminal. A key that contradicts one on
// record is a hard stop, never offered for trust.
func confirmHostKeys(ctx context.Context, hk hostKeyAPI, addr api.SSHView, in *bufio.Reader, out, errw io.Writer) int {
	report := func(what string, err error) int {
		var e *api.Error
		if errors.As(err, &e) {
			return reportServerErrorFrom(errw, what, hk.Info(), *e)
		}
		fmt.Fprintf(errw, "jumpgate: %s: %v\n", what, err)
		return exitCode("failed")
	}
	// Each round confirms at most one hop, so a path of n hops settles in
	// n+1 probes; anything longer means the keys keep changing under us.
	hops := 1
	for j := addr.Jump; j != nil; j = j.Jump {
		hops++
	}
	for round := 0; round <= hops; round++ {
		p, err := hk.ProbeHostKeys(ctx, addr)
		if err != nil {
			return report("host keys", err)
		}
		hop := p.Pending()
		if hop == nil {
			if p.AllConfirmed {
				return 0
			}
			fmt.Fprintln(errw, "jumpgate: the server did not probe every host on the path; nothing was trusted")
			return exitCode("failed")
		}
		switch hop.State {
		case api.HostKeyUnreachable:
			fmt.Fprintf(errw, "jumpgate: cannot reach %s: %s\n", hop.HostPort, hop.Error)
			return exitCode("unreachable")
		case api.HostKeyMismatch:
			fmt.Fprintf(errw, "jumpgate: SECURITY: %s\n  -> %s\n", hop.Error, api.HintFor(api.CodeHostKey))
			return exitCode("host_key")
		case api.HostKeyUnknown:
		default:
			fmt.Fprintf(errw, "jumpgate: %s: the server answered %q, which this jumpgate does not know; nothing was trusted\n", hop.HostPort, hop.State)
			return exitCode("failed")
		}
		fmt.Fprintf(out, "%s presents %s host key %s\nCompare it with the box's console (`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub`).\nType yes to trust it: ", hop.HostPort, hop.KeyType, hop.Fingerprint)
		// Only a complete line reading exactly "yes" is consent: any read
		// error (EOF after a bare "yes", a closed stdin) counts as no.
		answer, rerr := in.ReadString('\n')
		if rerr != nil || strings.TrimSpace(answer) != "yes" {
			fmt.Fprintln(errw, "jumpgate: not trusted; nothing was changed")
			return exitCode("failed")
		}
		if err := hk.ConfirmHostKey(ctx, hop.ProbeID, hop.Fingerprint); err != nil {
			return report("confirm "+hop.HostPort, err)
		}
	}
	fmt.Fprintln(errw, "jumpgate: the host keys did not settle after confirming every hop; run hosts add again")
	return exitCode("failed")
}

// call POSTs body to the local server and decodes a success into out. It
// returns nil on success, otherwise the server's error.
func call(c *apiclient.Client, path string, body, out any) *apiError {
	err := c.Do(context.Background(), http.MethodPost, path, body, out)
	var e *api.Error
	switch {
	case err == nil:
		return nil
	case errors.As(err, &e):
		return e
	default:
		return &apiError{Message: "server: " + err.Error()}
	}
}

// pairFailureCodes are the pairing stream's own failure codes: plain failures
// (exit 1), unlike the server codes reportServerError maps.
var pairFailureCodes = map[api.Code]bool{
	"": true, api.CodeStepFailed: true, api.CodeVerifyFailed: true, api.CodeRecordFailed: true, api.CodeTransportKey: true,
}

// streamPair prints each pairing event as it arrives.
func streamPair(c *apiclient.Client, name string, sudo bool) int {
	ch, err := c.Pair(context.Background(), name, api.PairRequest{Sudo: sudo})
	var e *api.Error
	if errors.As(err, &e) {
		if e.Code == api.CodeUnknownHost {
			fmt.Fprintf(os.Stderr, "jumpgate: %s presents an unconfirmed key %s\n", e.Host, e.Fingerprint)
		}
		return reportServerErrorFrom(os.Stderr, "pair", c.Info(), *e)
	}
	if err != nil {
		return failed("server: %v", err)
	}
	for ev := range ch {
		switch {
		case ev.Done:
			fmt.Printf("paired: agent %s\nRecommended now: disable root SSH login on the box (PermitRootLogin no). Keep console access as the way back in.\n", ev.Agent)
			return 0
		case ev.Error != "":
			code := reportServerError(os.Stderr, "pairing failed at "+ev.Step, apiError{Message: ev.Error, Code: ev.Code, Hint: ev.Hint})
			if pairFailureCodes[ev.Code] {
				return exitCode("failed")
			}
			return code
		default:
			fmt.Printf("[%s] %s\n", ev.Step, ev.Line)
		}
	}
	// A server of another version (the skew warning has said so) may send
	// events this CLI cannot read, such as an older server's "err" field: say what to
	// do rather than leave a bare "closed".
	if _, _, differs := c.Skew(); differs {
		return failed("the running server is an older jumpgate; run `jumpgate stop` and retry")
	}
	return failed("the server closed the stream before pairing finished")
}
