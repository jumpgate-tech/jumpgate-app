package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/peercred"
	"github.com/valve-tech/jumpgate/internal/server"
)

// adminGroup is the group whose members drive the jumpgate service over its
// admin socket (D6).
const adminGroup = "jumpgate-admin"

// systemRepublishEvery is how often the service compares its vault state to
// the one system.json shows.
const systemRepublishEvery = 2 * time.Second

// cmdServeSystem is `jumpgate serve --system`: the controller server as the
// jumpgate service under systemd. It differs from per-user serve in where it
// listens and how it is found: a 0660 admin socket in daemon.SystemDir that
// admits root, the service user and jumpgate-admin by peer credentials, and a
// world-readable, token-free system.json instead of server.json.
func cmdServeSystem(opts serverOptions) int {
	if hostGOOS != "linux" {
		return failed("`serve --system` runs under systemd on Linux")
	}
	holder, err := daemon.Acquire()
	if err != nil {
		return failed("%v", err)
	}
	defer holder.Release()

	gid, ok := peercred.GroupID(adminGroup)
	if !ok {
		return failed("group %s is missing; run `sudo systemd-sysusers`", adminGroup)
	}
	dir := daemon.SystemDir
	if err := os.Chown(dir, -1, gid); err != nil {
		return failed("%s: %v", dir, err)
	}
	if err := os.Chmod(dir, 0o750); err != nil {
		return failed("%s: %v", dir, err)
	}
	if err := disableDumpable(); err != nil {
		return failed("%v", err)
	}

	ctx, stop := shutdownContext(context.Background())
	defer stop()
	opts.AdminGID = gid
	b, err := buildServer(opts, stop, os.Stderr)
	if err != nil {
		return failed("%v", err)
	}
	var restartRequested atomic.Bool
	b.srv.SetRestart(func() {
		fmt.Fprintln(os.Stderr, "jumpgate: restart requested; exiting so systemd starts the service again (it comes back locked)")
		restartRequested.Store(true)
		stop()
	})
	if cfg, err := config.Load(); err == nil {
		for _, w := range systemStartWarnings(cfg) {
			fmt.Fprintf(os.Stderr, "jumpgate: %s\n", w)
		}
	}
	b.start(ctx, os.Stderr)
	defer daemon.RemoveSystem(dir)
	err = serveSystem(ctx, stop, b.srv, dir, opts.Bind)
	if err != nil {
		fmt.Fprintf(os.Stderr, "jumpgate: %v\n", err)
	}
	return systemExitCode(restartRequested.Load(), err)
}

// exitRestart is serve --system's status after POST /api/restart: a failure
// to systemd, so Restart=on-failure starts the service again.
const exitRestart = 75

// systemExitCode is serve --system's exit status.
func systemExitCode(restartRequested bool, err error) int {
	switch {
	case restartRequested:
		return exitRestart
	case err != nil:
		return 1
	}
	return 0
}

// serveSystem runs the TCP listener and the admin socket, publishes
// system.json once the socket accepts, and republishes it whenever the vault
// state changes.
func serveSystem(ctx context.Context, stop context.CancelFunc, s *server.Server, dir, bind string) error {
	sock := filepath.Join(dir, "server.sock")
	started := time.Now().UTC()
	publish := func(lock string) error {
		return daemon.PublishSystem(dir, daemon.SystemInfo{
			PID: os.Getpid(), Socket: sock, HTTPAddr: bind, Version: buildinfo.Version(),
			StartedAt: started, UID: os.Getuid(), Lock: lock,
		})
	}
	return serveBoth(ctx, stop, s.ListenAndServe, func(ctx context.Context) error { return s.ServeUnixSystem(ctx, sock) },
		func() error { return waitForSocket(ctx, sock, 2*time.Second) },
		func() error {
			last := s.VaultState().State
			if err := publish(last); err != nil {
				return fmt.Errorf("publish: %w", err)
			}
			fmt.Fprintf(os.Stderr, "jumpgate service on %s and %s\n", bind, sock)
			go func() {
				t := time.NewTicker(systemRepublishEvery)
				defer t.Stop()
				for {
					select {
					case <-ctx.Done():
						return
					case <-t.C:
					}
					if cur := s.VaultState().State; cur != last {
						if err := publish(cur); err != nil {
							fmt.Fprintf(os.Stderr, "jumpgate: publish %s: %v\n", filepath.Join(dir, "system.json"), err)
							continue
						}
						last = cur
					}
				}
			}()
			return nil
		})
}

// systemStartWarnings are the lines the service logs at start about how its
// secrets are kept.
func systemStartWarnings(cfg config.Config) []string {
	var w []string
	if plainSecretsWarning(cfg) != "" {
		w = append(w, "this service's secrets are stored unencrypted, readable from any backup or copy of the disk; run `jumpgate vault migrate`")
	}
	return w
}
