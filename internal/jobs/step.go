package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/valve-tech/jumpgate/internal/fsatomic"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// StepArgs names one attempt of one step of one job.
type StepArgs struct {
	Job, Step string
	Attempt   int
}

// maxAttempt matches the attempt field of a unit name (1-4 digits).
const maxAttempt = 9999

// valid is the one check every step argv passes, built or parsed: a job id, a
// step id and a small integer are all a payload may put on a command line.
func (a StepArgs) valid() error {
	if !ValidID(a.Job) || !ValidStepID(a.Step) || a.Attempt < 0 || a.Attempt > maxAttempt {
		return fmt.Errorf("jobs: bad step arguments: job %q, step %q, attempt %d", a.Job, a.Step, a.Attempt)
	}
	return nil
}

// StepArgv is the command line of a step's unit: the agent's own binary, its
// own directories, and a, which the caller must have checked (valid).
func StepArgv(self, stateDir, configDir string, a StepArgs) []string {
	return []string{self, "agent", "job-step", "--state-dir", stateDir, "--config-dir", configDir,
		"--job", a.Job, "--step", a.Step, "--attempt", strconv.Itoa(a.Attempt)}
}

// attemptRE refuses the signs and spaces strconv.Atoi would accept.
var attemptRE = regexp.MustCompile(`^[0-9]{1,4}$`)

// ParseStepArgs reads --job, --step and --attempt from anywhere in argv.
func ParseStepArgs(argv []string) (StepArgs, error) {
	var a StepArgs
	attempt := ""
	for i := 0; i+1 < len(argv); i++ {
		switch argv[i] {
		case "--job":
			a.Job = argv[i+1]
		case "--step":
			a.Step = argv[i+1]
		case "--attempt":
			attempt = argv[i+1]
		}
	}
	if !attemptRE.MatchString(attempt) {
		return a, fmt.Errorf("jobs: bad step arguments %q", argv)
	}
	a.Attempt, _ = strconv.Atoi(attempt)
	if err := a.valid(); err != nil {
		return a, fmt.Errorf("jobs: bad step arguments %q", argv)
	}
	return a, nil
}

// StepResult is a step's result file: the only thing a step process writes.
type StepResult struct {
	OK         bool                `json:"ok"`
	Error      string              `json:"error,omitempty"`
	Result     json.RawMessage     `json:"result,omitempty"`
	Ask        *intent.JobQuestion `json:"ask,omitempty"`
	FinishedAt int64               `json:"finishedAt"`
}

// RunStep is `jumpgate agent job-step`: it runs one attempt and writes its
// result file. It reads the manifest without the store and never touches the
// manifest or the log, and it holds no key (D-J4). A step that fails still
// writes a result and returns nil, so the unit exits 0 and the result file
// alone says how the step ended.
func RunStep(ctx context.Context, dir string, a StepArgs, base Env) error {
	path, err := ResultPath(dir, a.Job, a.Step, a.Attempt)
	if err != nil {
		return err
	}
	m, err := LoadManifest(dir, a.Job)
	if err != nil {
		return err
	}
	plan, ok := PlanFor(m.PlanVersion)
	if !ok {
		return fmt.Errorf("jobs: this agent does not know plan %s", m.PlanVersion)
	}
	// The agent marks the job running, with this very unit, before it starts
	// the unit; anything else is a step run by hand or a stale unit.
	idx := plan.index(a.Step)
	if idx < 0 || idx != m.Step || a.Attempt != m.Attempt || m.State != intent.JobRunning || m.Unit != UnitName(a.Job, a.Step, a.Attempt) {
		return errors.New("jobs: the agent did not ask for this step; refusing to run it out of turn")
	}
	env := base
	// Inputs are never secrets: steps' commands and stdout are echoed to the
	// box journal (adm-readable, outliving job pruning) and inputs sit in the
	// manifest. A step that needs a secret reads a root-only file on the box.
	env.Inputs, env.Results = m.Inputs, m.Results
	if m.Answer != nil && m.Answer.Step == idx && m.Answer.Attempt == a.Attempt-1 {
		env.Answer = m.Answer
	}
	if env.Out != nil {
		fmt.Fprintf(env.Out, "jumpgate job %s: step %s (%s), attempt %d\n", a.Job, a.Step, plan.Steps[idx].Title, a.Attempt)
	}
	out, runErr := plan.Steps[idx].Run(ctx, env)
	res := StepResult{OK: runErr == nil, Result: out.Result, Ask: out.Ask, FinishedAt: time.Now().Unix()}
	if runErr != nil {
		res.Error = runErr.Error()
		if env.Out != nil {
			fmt.Fprintln(env.Out, "step failed:", runErr)
		}
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return fsatomic.WriteFile(path, b)
}
