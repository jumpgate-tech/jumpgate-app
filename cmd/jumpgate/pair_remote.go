package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agentbin"
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/apiclient"
	"github.com/valve-tech/jumpgate/internal/bootstrap"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// newSSHExecutor dials the box for a push from this process. A seam for tests.
var newSSHExecutor = executor.NewSSHContext

// pairRemoteForeground installs and enrolls the gate on a box over the
// operator's own SSH login, in this process: the jumpgate service never
// reads the operator's key and never logs in to a box. The box's host key
// must be one the server has on record (hosts add confirmed it with the
// operator just before). It returns the new agent's address for the server
// to verify over its tunnel and record.
func pairRemoteForeground(ctx context.Context, out io.Writer, c *apiclient.Client, name string, login executor.SSHConfig, sudo bool) (string, int) {
	unpaired := func(format string, a ...any) (string, int) {
		return "", failed(format+"\nTarget %s is saved but not paired yet; run `jumpgate hosts add %s` again to finish.", append(a, name, name)...)
	}
	report := func(what string, err error) (string, int) {
		var e *api.Error
		if errors.As(err, &e) {
			return "", reportServerErrorFrom(os.Stderr, what, c.Info(), *e)
		}
		return unpaired("%s: %v", what, err)
	}
	info, err := c.PairingInfo(ctx, name)
	if err != nil {
		return report("pairing info", err)
	}
	controller, err := eip712.ParseAddress(info.Controller)
	if err != nil {
		return unpaired("controller address from the server: %v", err)
	}
	hostPort := sshHostPort(login)
	rec, err := c.RecordedHostKeys(ctx, hostPort)
	if err != nil {
		return report("host keys", err)
	}
	if len(rec.Keys) == 0 {
		return unpaired("no host key is confirmed for %s; confirm it first (`jumpgate hosts add` asks you to)", hostPort)
	}
	login.HostKey, login.HostKeyAlgorithms = recordedHostKeyCallback(hostPort, rec.Keys), nil

	fmt.Fprintln(out, api.CustodyWarning)
	ex, err := newSSHExecutor(ctx, login)
	if err != nil {
		return unpaired("ssh %s: %v", hostPort, err)
	}
	defer ex.Close()
	priv := ex
	if sudo || login.User != "root" {
		priv = executor.Sudo(ex)
	}
	label, _ := os.Hostname()
	if label == "" {
		label = "controller"
	}
	addr, err := runBootstrap(ctx, bootstrap.Options{
		Exec: priv, Local: false,
		AgentBinary: agentbin.Reporting(func(line string) { fmt.Fprintf(out, "[upload] %s\n", line) }),
		Controller:  controller, ControllerLabel: label, TransportKey: info.TransportKey, Wire: info.Wire,
		Event: func(step, line string) { fmt.Fprintf(out, "[%s] %s\n", step, line) },
	})
	if err != nil {
		return unpaired("pairing failed: %v", err)
	}
	return addr.Hex(), 0
}

// sshHostPort is the host:port a dial of c hands its host-key callback.
func sshHostPort(c executor.SSHConfig) string {
	port := c.Port
	if port == 0 {
		port = 22
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(port))
}

// recordedHostKeyCallback accepts hostPort presenting one of keys, and
// nothing else: no trust on first use from this process.
func recordedHostKeyCallback(hostPort string, keys []api.RecordedHostKey) ssh.HostKeyCallback {
	return func(hostname string, _ net.Addr, key ssh.PublicKey) error {
		if hostname != hostPort {
			return fmt.Errorf("host key for %s: only %s was confirmed", hostname, hostPort)
		}
		fp := executor.Fingerprint(key)
		for _, k := range keys {
			if k.Fingerprint == fp {
				return nil
			}
		}
		return fmt.Errorf("%s presented %s, which is not a confirmed key: %w", hostname, fp, executor.ErrHostKeyMismatch)
	}
}
