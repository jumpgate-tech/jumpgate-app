package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/valve-tech/jumpgate/cmd/jumpgate/web"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
)

func cmdServe(args []string) int {
	fset := flag.NewFlagSet("serve", flag.ContinueOnError)
	bind := fset.String("bind", "127.0.0.1:8799", bindFlagUsage)
	_ = fset.Bool("no-open", true, "accepted for symmetry; serve never opens a browser")
	if err := fset.Parse(args); err != nil {
		return exitCode("usage")
	}
	holder, err := daemon.Acquire()
	if err != nil {
		return failed("%v", err)
	}
	defer holder.Release()

	cfg, err := loadServerConfig(os.Stderr)
	if err != nil {
		return failed("load config: %v", err)
	}
	sgn, err := openControllerKey(cfg)
	if err != nil {
		return failed("load controller key: %v", err)
	}
	ui, err := fs.Sub(web.FS, "dist")
	if err != nil {
		return failed("embedded UI: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	token := server.NewSessionToken()
	s := server.New(server.Config{Bind: *bind, Token: token, UI: ui, Signer: sgn, Shutdown: stop})

	err = serveAndPublish(ctx, stop, s, holder, *bind, token)
	if err != nil {
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
