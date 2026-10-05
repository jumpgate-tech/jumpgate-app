package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/server"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// claimAppInstance takes the per-user single-instance lock for the tray/web
// entry point, the same lock `jumpgate serve` takes (R25). When another
// jumpgate server already holds it, it returns that server's server.json
// instead, so the caller opens it rather than starting a second server on the
// same port. A holder that is still starting gets up to wait to answer; one
// that never does is an error naming the way out.
func claimAppInstance(ctx context.Context, wait time.Duration) (*daemon.Holder, *daemon.Info, error) {
	deadline := time.Now().Add(wait)
	for {
		h, err := daemon.Acquire()
		if err == nil {
			return h, nil, nil
		}
		if !errors.Is(err, daemon.ErrAlreadyRunning) {
			return nil, nil, err
		}
		info, ok, err := daemon.Find(ctx)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return nil, &info, nil
		}
		if !time.Now().Before(deadline) {
			return nil, nil, errors.New("another jumpgate server holds the lock but does not answer; stop it with `jumpgate stop`, or see ~/.jumpgate/run/server.log")
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// appURL is the login URL the browser or tray window opens: the token in the
// query is exchanged for a session cookie on first load.
func appURL(info daemon.Info) string {
	return fmt.Sprintf("http://%s/?token=%s", info.HTTPAddr, info.Token)
}

// serveAndPublish runs s on bind and on ~/.jumpgate/run/server.sock until ctx
// ends, and publishes server.json once both answer, so every jumpgate command
// finds this server whichever entry point started it.
func serveAndPublish(ctx context.Context, stop context.CancelFunc, s *server.Server, holder *daemon.Holder, bind, token string) error {
	dir, err := daemon.RunDir()
	if err != nil {
		return err
	}
	sock := filepath.Join(dir, "server.sock")
	return serveBoth(ctx, stop, s.ListenAndServe, func(ctx context.Context) error { return s.ServeUnix(ctx, sock) },
		func() error { return waitForSocket(ctx, sock, 2*time.Second) },
		func() error {
			if err := holder.Publish(daemon.Info{PID: os.Getpid(), Socket: sock, HTTPAddr: bind, Token: token, Version: buildinfo.Version(), StartedAt: time.Now().UTC()}); err != nil {
				return fmt.Errorf("publish: %w", err)
			}
			fmt.Fprintf(os.Stderr, "jumpgate server on %s and %s\n", bind, sock)
			return nil
		})
}

// loadServerConfig is the start of serve and of the app, once the server
// lock is held: it loads the config, then restricts the state directory and
// every secret already in it to this user (ruling P29). A file it cannot
// restrict is reported and the server still comes up, except a signing key
// (the controller key file, the transport key): that stops startup.
func loadServerConfig(stderr io.Writer) (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, fmt.Errorf("load config: %w", err)
	}
	keyFile := ""
	if cfg.Controller != nil && signer.Store(cfg.Controller.KeyStore) == signer.StoreFile {
		keyFile = cfg.Controller.KeyRef
	}
	tightened, warnings, err := config.TightenState(keyFile)
	for _, p := range tightened {
		fmt.Fprintf(stderr, "jumpgate: other users could read or change %s; it is now restricted to you (if it holds a key, treat that key as exposed)\n", p)
	}
	for _, w := range warnings {
		fmt.Fprintf(stderr, "jumpgate: %v\n", w)
	}
	return cfg, err
}

// openControllerKey opens the controller key named in the config, or returns
// nil when none has been made yet (the intent and pair routes then answer 503
// no_controller_key).
func openControllerKey(cfg config.Config) (signer.Signer, error) {
	if cfg.Controller == nil {
		return nil, nil
	}
	return signer.Open(context.Background(), signer.Store(cfg.Controller.KeyStore), cfg.Controller.KeyRef)
}
