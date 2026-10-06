package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/agentclient"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

const (
	defaultStateDir  = "/var/lib/jumpgate"
	defaultConfigDir = "/etc/jumpgate"
)

func cmdAgent(args []string) int {
	// The agent runs on Linux boxes; macOS keeps it for its tests (spec D19).
	if hostGOOS == "windows" {
		return failed("`jumpgate agent` runs on the Linux boxes jumpgate manages, not on Windows")
	}
	if len(args) == 0 {
		return usage("usage: jumpgate agent init|enroll|run|reset-replay")
	}
	fset := flag.NewFlagSet("agent "+args[0], flag.ContinueOnError)
	state := fset.String("state-dir", defaultStateDir, "agent state directory")
	conf := fset.String("config-dir", defaultConfigDir, "agent config directory")
	address := fset.String("address", "", "signer address to enroll")
	tier := fset.String("tier", "routine", "signer tier")
	label := fset.String("label", "", "signer label")
	localUID := fset.Int("local-uid", -1, "also allow this local uid on the socket")
	yes := fset.Bool("yes", false, "confirm reset-replay")
	resetReplay := fset.Bool("reset-replay", false, "init: start an empty replay record when the key exists but the record is gone")
	if err := fset.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	var err error
	switch args[0] {
	case "init":
		err = agentInit(os.Stdout, *state, *conf, *resetReplay)
	case "enroll":
		err = agentEnroll(*conf, agentclient.DefaultSocket, *address, *tier, *label, *localUID)
	case "run":
		err = agentRun(*state, *conf)
	case "reset-replay":
		if !*yes {
			return usage("reset-replay reopens a replay window for intents captured before now; re-run with --yes once you know why the record broke")
		}
		err = agentResetReplay(*state)
	default:
		return usage("unknown agent subcommand %q", args[0])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "jumpgate agent:", err)
		return 1
	}
	return 0
}

// agentInit creates the agent key and replay record if absent and prints the
// agent's address. It never replaces a key: the box keeps its identity across
// re-pairing, so only a key file that does not exist is generated. Any other
// failure to read it (damaged, wrong mode, a symlink) stops here.
//
// An existing key with no replay record is refused unless resetReplay is
// set: a fresh, empty record would reopen the replay window the running agent
// deliberately fails closed on (replay_state), and re-pairing must not do that
// silently.
func agentInit(out io.Writer, stateDir, configDir string, resetReplay bool) error {
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(configDir, 0o700); err != nil {
		return err
	}
	keyPath := filepath.Join(stateDir, "agent.key")
	replayPath := filepath.Join(stateDir, "replay.json")
	k, err := signer.LoadKeyFile(keyPath)
	if errors.Is(err, fs.ErrNotExist) {
		k, err = signer.GenerateKeyFile(keyPath)
	} else if err == nil && !resetReplay {
		if _, serr := os.Lstat(replayPath); errors.Is(serr, fs.ErrNotExist) {
			return fmt.Errorf("%s exists but %s does not; starting an empty replay record reopens a replay window for intents captured before now. Find out why it went missing, then re-run with --reset-replay", keyPath, replayPath)
		}
	}
	if err != nil {
		return err
	}
	if err := agent.InitReplay(replayPath); err != nil {
		return err
	}
	fmt.Fprintln(out, k.Address().Hex())
	return nil
}

// agentEnroll adds a controller (and optionally a local uid) to the policy. It
// appends and is idempotent: enrolling an address already present changes
// nothing, including its tier. A running agent's socket is re-chmodded to the
// new policy's mode (R23) so an enrolled local uid can connect at once; an
// empty socketPath skips that.
func agentEnroll(configDir, socketPath, address, tier, label string, localUID int) error {
	a, err := eip712.ParseAddress(address)
	if err != nil {
		return err
	}
	if tier != string(agent.TierRoutine) && tier != string(agent.TierApproval) {
		return fmt.Errorf("tier must be routine or approval")
	}
	path := filepath.Join(configDir, "policy.json")
	p, err := agent.LoadPolicy(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	p.AddSigner(agent.SignerEntry{Address: a.Hex(), Tier: agent.Tier(tier), Label: label})
	if localUID >= 0 {
		p.AddLocalUID(localUID)
	}
	if err := p.Save(path); err != nil {
		return err
	}
	if socketPath == "" {
		return nil
	}
	return agent.ApplySocketMode(socketPath, p)
}

// agentResetReplay moves a broken replay record aside and starts an empty one.
func agentResetReplay(stateDir string) error {
	path := filepath.Join(stateDir, "replay.json")
	if err := os.Rename(path, path+".broken-"+time.Now().UTC().Format("20060102T150405")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return agent.InitReplay(path)
}

// agentRun serves until SIGTERM. systemd hands the key in through
// LoadCredential; outside systemd it is read from the state directory.
func agentRun(stateDir, configDir string) error {
	keyPath := filepath.Join(stateDir, "agent.key")
	if d := os.Getenv("CREDENTIALS_DIRECTORY"); d != "" {
		keyPath = filepath.Join(d, "agent.key")
	}
	k, err := signer.LoadKeyFile(keyPath)
	if err != nil {
		return err
	}
	a := agent.New(agent.Config{
		Key: k, Exec: executor.NewLocal(),
		PolicyPath: filepath.Join(configDir, "policy.json"),
		ReplayPath: filepath.Join(stateDir, "replay.json"),
		NodePath:   filepath.Join(configDir, "node.json"),
	})
	// The policy only sets the socket mode here; Handle re-reads it for every
	// intent. An unreadable policy keeps the stricter 0660.
	p, err := agent.LoadPolicy(filepath.Join(configDir, "policy.json"))
	if err != nil {
		p = agent.Policy{}
	}
	ln, err := agent.Listen(agentclient.DefaultSocket, agent.JumpgateGID(), agent.SocketMode(p))
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintf(os.Stderr, "jumpgate agent %s listening on %s\n", a.Address().Hex(), agentclient.DefaultSocket)
	return agent.Serve(ctx, a, ln)
}
