package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/valve-tech/jumpgate/internal/relay"
	"github.com/valve-tech/jumpgate/internal/server"
)

// cmdRelay runs the metered RPC data plane on its own, as `jumpgate relay`.
// This is the recommended way to sell RPC access: the internet-facing proxy
// then holds no authority over the fleet. It never opens config.json, the
// controller key or an executor, never takes the controller's lock, and is
// not migrated or discovered like the controller server. Its settings come
// only from flags and the JUMPGATE_* environment.
func cmdRelay(args []string) int {
	ctx, stop := shutdownContext(context.Background())
	defer stop()
	return relayMain(ctx, args, os.Getenv, os.Stderr)
}

// relayMain is cmdRelay with its context, environment and log injected.
func relayMain(ctx context.Context, args []string, getenv func(string) string, logw io.Writer) int {
	fset := flag.NewFlagSet("relay", flag.ContinueOnError)
	fset.SetOutput(logw)
	var o serverOptions
	if err := addRelayFlags(fset, getenv, &o); err != nil {
		fmt.Fprintf(logw, "jumpgate: relay: %v\n", err)
		return exitCode("usage")
	}
	if err := fset.Parse(args); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(logw, "jumpgate: relay: %v\n", err)
		}
		return exitCode("usage")
	}
	if fset.NArg() > 0 {
		fmt.Fprintf(logw, "jumpgate: relay: unexpected argument %q\n", fset.Arg(0))
		return exitCode("usage")
	}
	if o.RelayBind == "" {
		fmt.Fprintf(logw, "jumpgate: relay: --relay-bind (or %s) is required\n", envRelayBind)
		return exitCode("usage")
	}

	opts := relayBuildOptions(o)
	handler, runtime, err := relay.Build(opts)
	if err != nil {
		fmt.Fprintf(logw, "jumpgate: relay: %v\n", err)
		return exitCode("failed")
	}
	if w := relay.UnmeteredWarning(opts); w != "" {
		fmt.Fprintf(logw, "jumpgate: %s\n", w)
	}

	// The background loops are not optional: without them leased credits are
	// never settled back. They stop with ctx, and the listener's shutdown
	// waits for in-flight calls first.
	loops := make(chan struct{})
	go func() { runtime.Run(ctx); close(loops) }()
	fmt.Fprintf(logw, "jumpgate: metered RPC data plane on %s\n", o.RelayBind)
	err = server.ServeRelay(ctx, o.RelayBind, handler)
	if err != nil {
		fmt.Fprintf(logw, "jumpgate: relay: %v\n", err)
		return exitCode("failed")
	}
	<-loops
	return 0
}
