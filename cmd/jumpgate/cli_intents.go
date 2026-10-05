package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func cmdIntent(name, kind string) func([]string) int {
	return func(args []string) int {
		if len(args) != 1 {
			return usage("usage: jumpgate %s HOST", name)
		}
		return runIntent(args[0], kind, struct{}{})
	}
}

func cmdLogs(args []string) int {
	fset := flag.NewFlagSet("logs", flag.ContinueOnError)
	n := fset.Int("n", intent.LogsDefaultN, "lines per unit (max 2000)")
	if len(args) == 0 {
		return usage("usage: jumpgate logs HOST [-n N]")
	}
	if err := fset.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	return runIntent(args[0], intent.KindLogsRead, intent.LogsReadPayload{N: *n})
}

func cmdService(args []string) int {
	if len(args) != 3 {
		return usage("usage: jumpgate service HOST exec|beacon start|stop|restart")
	}
	return runIntent(args[0], intent.KindServiceAction, intent.ServiceActionPayload{Service: args[1], Action: args[2]})
}

func runIntent(host, kind string, payload any) int {
	ctx := context.Background()
	c, err := connect(ctx)
	if err != nil {
		return failed("%v", err)
	}
	apiclient.WarnSkew(os.Stderr, c)
	r, err := c.Intent(ctx, host, kind, payload)
	var e *api.Error
	if errors.As(err, &e) {
		return reportServerErrorFrom(os.Stderr, host, c.Info(), *e)
	}
	if err != nil {
		return failed("%v", err)
	}
	return printReply(os.Stdout, os.Stderr, host, r)
}

func printReply(stdout, stderr io.Writer, host string, r api.IntentReply) int {
	switch {
	case r.Rejection != nil:
		hint := r.Hint
		if hint == "" {
			hint = api.RejectionHint(r.Rejection.Code) // an older server sends none
		}
		fmt.Fprintf(stderr, "jumpgate: %s refused (%s): %s\n  -> %s\n", host, r.Rejection.Code, r.Rejection.Message, hint)
		if r.Rejection.Code == intent.ReasonClockSkew && r.Rejection.AgentTime != 0 {
			fmt.Fprintf(stderr, "  this machine: %s, the box: %s\n", time.Now().UTC().Format(time.RFC3339), time.Unix(r.Rejection.AgentTime, 0).UTC().Format(time.RFC3339))
		}
		return exitCode("refused")
	case r.Failure != nil:
		fmt.Fprintf(stderr, "jumpgate: %s ran it and it failed: %s\n", host, r.Failure.Message)
		return exitCode("failed")
	}
	var pretty any
	_ = json.Unmarshal(r.Result, &pretty)
	if r.RefHead != 0 {
		pretty = map[string]any{"node": pretty, "refHead": r.RefHead}
	}
	out, _ := json.MarshalIndent(pretty, "", "  ")
	fmt.Fprintln(stdout, string(out))
	return 0
}
