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
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/jobs"
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
		return usage("usage: jumpgate agent init|enroll|run|reset-replay|job-step|job-abandon")
	}
	fset := flag.NewFlagSet("agent "+args[0], flag.ContinueOnError)
	state := fset.String("state-dir", defaultStateDir, "agent state directory")
	conf := fset.String("config-dir", defaultConfigDir, "agent config directory")
	address := fset.String("address", "", "signer address to enroll")
	tier := fset.String("tier", "routine", "signer tier")
	label := fset.String("label", "", "signer label")
	localUID := fset.Int("local-uid", -1, "also allow this local uid on the socket")
	yes := fset.Bool("yes", false, "confirm reset-replay or job-abandon")
	resetReplay := fset.Bool("reset-replay", false, "init: start an empty replay record when the key exists but the record is gone")
	// job-step's own arguments are read by jobs.ParseStepArgs, which checks
	// every one; they are declared here only so the flag set accepts them.
	fset.String("job", "", "job id (job-step)")
	fset.String("step", "", "step id (job-step)")
	fset.String("attempt", "", "attempt (job-step)")
	if err := fset.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	// job-abandon takes its job id first; flags after it still count.
	var jobID string
	if args[0] == "job-abandon" && fset.NArg() > 0 {
		jobID = fset.Arg(0)
		if err := fset.Parse(fset.Args()[1:]); err != nil || fset.NArg() > 0 {
			return usage("usage: jumpgate agent job-abandon ID --yes")
		}
	}
	var err error
	switch args[0] {
	case "init":
		err = agentInit(os.Stdout, *state, *conf, *resetReplay)
	case "enroll":
		err = agentEnroll(*conf, agentclient.DefaultSocket, *address, *tier, *label, *localUID)
	case "run":
		err = agentRun(*state, *conf)
	case "job-step":
		var a jobs.StepArgs
		if a, err = jobs.ParseStepArgs(args[1:]); err == nil {
			err = agentJobStep(*state, *conf, a)
		}
	case "reset-replay":
		if !*yes {
			return usage("reset-replay reopens a replay window for intents captured before now; re-run with --yes once you know why the record broke")
		}
		err = agentResetReplay(*state)
	case "job-abandon":
		if !jobs.ValidID(jobID) {
			return usage("usage: jumpgate agent job-abandon ID --yes (ID is the job's 16 hex digits)")
		}
		if !*yes {
			return usage("job-abandon ends job %s as failed, whatever its steps left undone; re-run with --yes once you know why no controller can move it", jobID)
		}
		err = agentJobAbandon(os.Stdout, *state, jobID)
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

// agentJobStep is the body of a job step's transient unit. It holds no key
// and never opens the job store, whose lock the running agent holds: it
// reads its manifest and writes one result file (spec D-J4). Its arguments
// were checked by jobs.ParseStepArgs.
func agentJobStep(stateDir, configDir string, a jobs.StepArgs) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return jobs.RunStep(ctx, filepath.Join(stateDir, "jobs"), a, jobs.Env{
		Exec: executor.NewLocal(), NodePath: filepath.Join(configDir, "node.json"),
		NamesPath: filepath.Join(configDir, "names.json"), Out: os.Stdout,
	})
}

// agentKeyPath is where the agent's key is read from: systemd hands it in
// through LoadCredential; outside systemd it is in the state directory.
func agentKeyPath(stateDir string) string {
	if d := os.Getenv("CREDENTIALS_DIRECTORY"); d != "" {
		return filepath.Join(d, "agent.key")
	}
	return filepath.Join(stateDir, "agent.key")
}

// agentJobAbandon ends a job no signed request can move, as failed, with an
// entry the agent's own key signs (jobs.Abandon). It opens the job store, so
// the agent must be stopped first.
func agentJobAbandon(out io.Writer, stateDir, id string) error {
	k, err := signer.LoadKeyFile(agentKeyPath(stateDir))
	if err != nil {
		return err
	}
	rec, err := jobs.Abandon(context.Background(), jobs.ManagerConfig{
		Dir: filepath.Join(stateDir, "jobs"), Key: k, Runner: jobs.DefaultRunner(executor.NewLocal()), Version: buildinfo.Version(),
	}, id)
	if errors.Is(err, jobs.ErrStoreInUse) {
		return fmt.Errorf("the running agent holds the job store; stop the agent (systemctl stop jumpgate-agent), run job-abandon again, then start it: %w", err)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "job %s (%s) is now %s: %s\n", rec.ID, rec.Kind, rec.State, rec.Error)
	return nil
}

// agentJobManager is the job store agentRun serves with. Only another agent
// holding the store stops the agent. Any other failure to open it (a broken
// directory, a full disk) is logged and returned as unavailable, and the
// agent runs without durable jobs, so status and logs keep working. That is
// not "no jobs": a job already in the store may own the node services, so
// the agent refuses service.action while the store holds anything (J1), and
// answers every job kind with the reason.
func agentJobManager(log io.Writer, stateDir, configDir string, k signer.Signer, ex executor.Executor) (mgr *jobs.Manager, unavailable, err error) {
	mgr, err = openJobManager(stateDir, configDir, k, ex)
	if errors.Is(err, jobs.ErrStoreInUse) {
		return nil, nil, err
	}
	if err != nil {
		fmt.Fprintf(log, "jumpgate agent: running without durable jobs, the job store did not open: %v\n", err)
		return nil, err, nil
	}
	return mgr, nil, nil
}

// openJobManager opens the job store, which it locks until Close. A second
// agent on the box fails here, saying so, before it could touch the running
// agent's socket.
func openJobManager(stateDir, configDir string, k signer.Signer, ex executor.Executor) (*jobs.Manager, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	mgr, err := jobs.NewManager(jobs.ManagerConfig{
		Dir: filepath.Join(stateDir, "jobs"), Key: k, Runner: jobs.DefaultRunner(ex), Version: buildinfo.Version(),
		StepArgv: func(a jobs.StepArgs) []string { return jobs.StepArgv(self, stateDir, configDir, a) },
	})
	if errors.Is(err, jobs.ErrStoreInUse) {
		return nil, fmt.Errorf("another jumpgate agent is already running on this box: %w", err)
	}
	return mgr, err
}

// agentRun serves until SIGTERM. systemd hands the key in through
// LoadCredential; outside systemd it is read from the state directory.
func agentRun(stateDir, configDir string) error {
	k, err := signer.LoadKeyFile(agentKeyPath(stateDir))
	if err != nil {
		return err
	}
	ex := executor.NewLocal()
	// The job store first: its lock is what tells a second agent to stop,
	// and Listen below replaces whatever socket is at the path.
	mgr, unavailable, err := agentJobManager(os.Stderr, stateDir, configDir, k, ex)
	if err != nil {
		return err
	}
	if mgr != nil {
		defer mgr.Close()
	}
	a := agent.New(agent.Config{
		Key: k, Exec: ex, Jobs: mgr, Log: os.Stderr,
		JobsUnavailable: unavailable, JobsDir: filepath.Join(stateDir, "jobs"),
		PolicyPath: filepath.Join(configDir, "policy.json"),
		ReplayPath: filepath.Join(stateDir, "replay.json"),
		NodePath:   filepath.Join(configDir, "node.json"),
		NamesPath:  filepath.Join(configDir, "names.json"),
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
	// The reconcile loop writes to the job store, so it must end before the
	// deferred mgr.Close above releases it. Defers run last first: this one
	// stops the loop and waits for it, then the store closes.
	jobsDone := make(chan struct{})
	go func() { defer close(jobsDone); a.RunJobs(ctx) }()
	defer func() { stop(); <-jobsDone }()
	fmt.Fprintf(os.Stderr, "jumpgate agent %s listening on %s\n", a.Address().Hex(), agentclient.DefaultSocket)
	return agent.Serve(ctx, a, ln)
}
