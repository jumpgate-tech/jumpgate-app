package main

import (
	"bufio"
	"context"
	"encoding/json"
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

	"golang.org/x/crypto/ssh"

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
		c, err := config.Load()
		if err != nil {
			return failed("%v", err)
		}
		for _, t := range c.Targets {
			paired := "not paired"
			if t.Agent != nil {
				paired = "agent " + t.Agent.Address
			}
			fmt.Printf("%-16s %-6s %s\n", t.ID, t.Mode, paired)
		}
		return 0
	case "add":
		return hostsAdd(args[1:])
	}
	return usage("unknown hosts subcommand %q", args[0])
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

	ctx := context.Background()
	target := map[string]any{"id": name, "mode": "local"}
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
		if code := confirmHostKeys(ctx, cfg, bufio.NewReader(os.Stdin), os.Stdout, os.Stderr); code != 0 {
			return code
		}
		target = map[string]any{"id": name, "mode": "ssh", "ssh": cfg}
	}

	exe, _ := os.Executable()
	info, err := daemon.EnsureRunning(ctx, exe)
	if err != nil {
		return failed("%v", err)
	}
	if e := call(info, "/api/targets", target, nil); e != nil {
		if e.Status != http.StatusConflict || e.Code != "" {
			return reportServerError(os.Stderr, "add "+name, *e)
		}
		// The name is already a target: pairing it again is how a box is
		// re-paired or an interrupted pairing finished.
		fmt.Printf("target %s already exists; pairing it again\n", name)
	}
	return streamPair(info, name, *sudo)
}

// stdinIsTerminal reports whether stdin is a character device. A variable so
// tests can stand in for a terminal; nothing else may replace it.
var stdinIsTerminal = func() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// sshConfigFrom builds the dial configuration for hosts add. Every hop carries
// the strict host-key callback: pairing never trusts a key on first use.
func sshConfigFrom(login, key, jump string) (executor.SSHConfig, error) {
	user, host, port, err := parseSSHTarget(login)
	if err != nil {
		return executor.SSHConfig{}, err
	}
	check, err := strictCheck()
	if err != nil {
		return executor.SSHConfig{}, err
	}
	cfg := executor.SSHConfig{Host: host, Port: port, User: user, KeyPath: key, HostKeyFile: jgFile("known_hosts"), HostKey: check}
	if jump != "" {
		ju, jh, jp, err := parseSSHTarget(jump)
		if err != nil {
			return executor.SSHConfig{}, fmt.Errorf("--jump: %w", err)
		}
		cfg.Jump = &executor.SSHConfig{Host: jh, Port: jp, User: ju, KeyPath: key, HostKeyFile: jgFile("known_hosts"), HostKey: check}
	}
	return cfg, nil
}

// strictCheck is the host-key policy for pairing: keys a person confirmed
// (the confirmed-hosts file) or listed in the operator's OpenSSH known_hosts.
// It is deliberately not the trust-on-first-use file, which Strict never reads.
func strictCheck() (ssh.HostKeyCallback, error) {
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		return nil, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return executor.Strict(confirmed, filepath.Join(home, ".ssh", "known_hosts")), nil
}

// hostPort is the exact address string DialSSH hands the host-key callback,
// and so the one RecordHostKey must record.
func hostPort(c executor.SSHConfig) string {
	port := c.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

// confirmHostKeys shows the operator every unconfirmed host key on the path,
// the jump host first, and records one only on an explicit "yes". A key that
// contradicts one already on record is a hard stop. The target is reached
// through the jump host only after the jump host is confirmed, under the
// strict policy, never trust-on-first-use.
func confirmHostKeys(ctx context.Context, cfg executor.SSHConfig, in *bufio.Reader, out, errw io.Writer) int {
	failedTo := func(format string, a ...any) int {
		fmt.Fprintf(errw, "jumpgate: "+format+"\n", a...)
		return exitCode("failed")
	}
	check, err := strictCheck()
	if err != nil {
		return failedTo("%v", err)
	}
	confirmed, err := config.ConfirmedHostsFile()
	if err != nil {
		return failedTo("%v", err)
	}

	type hop struct{ probe executor.SSHConfig }
	var hops []hop
	if cfg.Jump != nil {
		j := *cfg.Jump
		j.Jump, j.HostKey = nil, nil // confirm the jump host by a direct connection
		hops = append(hops, hop{j})
	}
	target := cfg
	if cfg.Jump != nil {
		j := *cfg.Jump
		j.HostKey = check
		target.Jump = &j
	}
	target.HostKey = check
	hops = append(hops, hop{target})

	for _, h := range hops {
		key, err := executor.CaptureHostKey(ctx, h.probe)
		if err != nil {
			fmt.Fprintf(errw, "jumpgate: cannot reach %s: %v\n", h.probe.Host, err)
			return exitCode("unreachable")
		}
		hp := hostPort(h.probe)
		err = check(hp, nil, key)
		var unknown *executor.UnknownHostError
		switch {
		case err == nil:
			continue
		case errors.As(err, &unknown):
			fmt.Fprintf(out, "%s presents host key %s\nCompare it with the box's console (`ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub`).\nType yes to trust it: ", hp, unknown.Fingerprint)
			// Only a complete line reading exactly "yes" is consent: any read
			// error (EOF after a bare "yes", a closed stdin) counts as no.
			answer, rerr := in.ReadString('\n')
			if rerr != nil || strings.TrimSpace(answer) != "yes" {
				return failedTo("not trusted; nothing was changed")
			}
			if err := executor.RecordHostKey(confirmed, hp, key); err != nil {
				return failedTo("%v", err)
			}
		default:
			fmt.Fprintf(errw, "jumpgate: %v\n", err)
			return exitCode("host_key")
		}
	}
	return 0
}

// call POSTs body to the local server and decodes a success into out. It
// returns nil on success, otherwise the server's error.
func call(info daemon.Info, path string, body, out any) *apiError {
	b, _ := json.Marshal(body)
	res, err := info.Client().Do(mustRequest(context.Background(), info, path, b))
	if err != nil {
		return &apiError{Error: "server: " + err.Error()}
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		e := readAPIError(res)
		return &e
	}
	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return nil
}

// pairFailureCodes are the pairing stream's own failure codes: plain failures
// (exit 1), unlike the server codes reportServerError maps.
var pairFailureCodes = map[string]bool{
	"": true, "step_failed": true, "verify_failed": true, "record_failed": true, "transport_key": true,
}

// streamPair prints each pairing event as it arrives.
func streamPair(info daemon.Info, name string, sudo bool) int {
	b, _ := json.Marshal(map[string]bool{"sudo": sudo})
	res, err := info.Client().Do(mustRequest(context.Background(), info, "/api/targets/"+name+"/pair", b))
	if err != nil {
		return failed("server: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		e := readAPIError(res)
		if e.Code == "unknown_host" {
			fmt.Fprintf(os.Stderr, "jumpgate: %s presents an unconfirmed key %s\n", e.Host, e.Fingerprint)
		}
		return reportServerError(os.Stderr, "pair", e)
	}
	sc := bufio.NewScanner(res.Body)
	for sc.Scan() {
		line, ok := strings.CutPrefix(sc.Text(), "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Step  string `json:"step"`
			Line  string `json:"line"`
			Err   string `json:"err"`
			Code  string `json:"code"`
			Hint  string `json:"hint"`
			Done  bool   `json:"done"`
			Agent string `json:"agent"`
		}
		if json.Unmarshal([]byte(line), &ev) != nil {
			continue
		}
		switch {
		case ev.Done:
			fmt.Printf("paired: agent %s\nRecommended now: disable root SSH login on the box (PermitRootLogin no). Keep console access as the way back in.\n", ev.Agent)
			return 0
		case ev.Err != "":
			code := reportServerError(os.Stderr, "pairing failed at "+ev.Step, apiError{Error: ev.Err, Code: ev.Code, Hint: ev.Hint})
			if pairFailureCodes[ev.Code] {
				return exitCode("failed")
			}
			return code
		default:
			fmt.Printf("[%s] %s\n", ev.Step, ev.Line)
		}
	}
	return failed("the server closed the stream before pairing finished")
}
