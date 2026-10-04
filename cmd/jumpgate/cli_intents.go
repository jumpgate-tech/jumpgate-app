package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func cmdIntent(name, kind string) func([]string) int {
	return func(args []string) int {
		if len(args) != 1 {
			return fail("usage: jumpgate %s HOST", name)
		}
		return runIntent(args[0], kind, struct{}{})
	}
}

func cmdLogs(args []string) int {
	fset := flag.NewFlagSet("logs", flag.ContinueOnError)
	n := fset.Int("n", intent.LogsDefaultN, "lines per unit (max 2000)")
	if len(args) == 0 {
		return fail("usage: jumpgate logs HOST [-n N]")
	}
	if err := fset.Parse(args[1:]); err != nil {
		return exitCode("usage")
	}
	return runIntent(args[0], intent.KindLogsRead, intent.LogsReadPayload{N: *n})
}

func cmdService(args []string) int {
	if len(args) != 3 {
		return fail("usage: jumpgate service HOST exec|beacon start|stop|restart")
	}
	return runIntent(args[0], intent.KindServiceAction, intent.ServiceActionPayload{Service: args[1], Action: args[2]})
}

// reply is the server's answer to an intent: the verified receipt on success,
// otherwise the error JSON.
type reply struct {
	Status    uint8             `json:"status"`
	Result    json.RawMessage   `json:"result"`
	Rejection *intent.Rejection `json:"rejection"`
	Failure   *intent.Failure   `json:"failure"`
	RefHead   uint64            `json:"refHead"`
}

func runIntent(host, kind string, payload any) int {
	ctx := context.Background()
	exe, _ := os.Executable()
	info, err := daemon.EnsureRunning(ctx, exe)
	if err != nil {
		return fail("%v", err)
	}
	b, _ := json.Marshal(payload)
	res, err := info.Client().Do(mustRequest(ctx, info, "/api/targets/"+host+"/intent/"+kind, b))
	if err != nil {
		return fail("server: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return reportServerError(os.Stderr, host, readAPIError(res))
	}
	var r reply
	if err := json.NewDecoder(io.LimitReader(res.Body, 32<<20)).Decode(&r); err != nil {
		return fail("could not read the server's answer: %v", err)
	}
	return printReply(os.Stdout, os.Stderr, host, r)
}

func printReply(stdout, stderr io.Writer, host string, r reply) int {
	switch {
	case r.Rejection != nil:
		fmt.Fprintf(stderr, "jumpgate: %s refused (%s): %s\n  -> %s\n", host, r.Rejection.Code, r.Rejection.Message, remedies[r.Rejection.Code])
		if r.Rejection.Code == intent.ReasonClockSkew && r.Rejection.AgentTime != 0 {
			fmt.Fprintf(stderr, "  this machine: %s, the box: %s\n", time.Now().UTC().Format(time.RFC3339), time.Unix(int64(r.Rejection.AgentTime), 0).UTC().Format(time.RFC3339))
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
