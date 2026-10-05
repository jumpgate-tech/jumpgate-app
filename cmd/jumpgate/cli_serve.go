package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/valve-tech/jumpgate/internal/daemon"
)

// cmdServe runs the controller server without opening a browser. It takes the
// same server flags and environment as the web app and builds the server
// through the same buildServer, so a server a CLI command auto-starts here is
// the server the app would have been.
func cmdServe(args []string) int {
	opts, err := parseServeArgs(args, os.Getenv)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitCode("usage")
		}
		return usage("serve: %v", err)
	}
	if warning := bindWarningLine(opts.Bind); warning != "" {
		fmt.Fprintln(os.Stderr, warning)
	}
	holder, err := daemon.Acquire()
	if err != nil {
		return failed("%v", err)
	}
	defer holder.Release()

	ctx, stop := shutdownContext(context.Background())
	defer stop()
	b, err := buildServer(opts, stop, os.Stderr)
	if err != nil {
		return failed("%v", err)
	}
	b.start(ctx, os.Stderr)
	if err := serveAndPublish(ctx, stop, b.srv, holder, opts.Bind, b.token, &b.shape); err != nil {
		return failed("%v", err)
	}
	return 0
}

// serveBoth runs the TCP and unix listeners, calls ready until the sockets
// accept, then publish, and runs until ctx ends or a listener fails. It
// returns only after BOTH listeners have returned: each waits for in-flight
// clear, wipe and pairing operations before it does, and the process must not
// exit under them. A failure in one stops the other. The first error wins.
func serveBoth(ctx context.Context, stop context.CancelFunc, tcp, unix func(context.Context) error, ready, publish func() error) error {
	errs := make(chan error, 2)
	go func() { errs <- tcp(ctx) }()
	go func() { errs <- unix(ctx) }()
	readyErr := make(chan error, 1)
	go func() { readyErr <- ready() }()

	var first error
	abort := func(err error) {
		if first == nil {
			first = err
		}
		stop()
	}
	for remaining := 2; remaining > 0; {
		select {
		case err := <-errs:
			remaining--
			if err != nil {
				abort(err)
			}
			stop() // one listener is finished, so the other must wind down too
		case err := <-readyErr:
			readyErr = nil // a nil channel is never selected again
			switch {
			case err != nil:
				abort(err)
			case first == nil:
				// Advertise the server only once its sockets answer and no
				// listener has already failed.
				if perr := publish(); perr != nil {
					abort(perr)
				}
			}
		}
	}
	return first
}

// waitForSocket dials the unix socket until it accepts, for at most timeout.
func waitForSocket(ctx context.Context, path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		c, err := net.DialTimeout("unix", path, 100*time.Millisecond)
		if err == nil {
			c.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the server socket %s did not come up within %s: %w", path, timeout, err)
		}
		select {
		case <-ctx.Done():
			return errors.Join(ctx.Err(), err)
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func cmdStop([]string) int {
	if err := daemon.Stop(context.Background()); err != nil {
		return failed("%v", err)
	}
	return 0
}
