package server

import (
	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
)

// targetView is how a stored target is shown to clients.
func targetView(t config.Target) api.TargetView {
	v := api.TargetView{ID: t.ID, Mode: t.Mode, SSH: sshView(t.SSH), Wire: t.Wire, Devnet: t.Devnet, ThisMachine: t.Mode == "local"}
	switch {
	case t.Agent != nil && t.Agent.Transport == "local":
		v.Link = api.LinkAgentLocal
	case t.Agent != nil:
		v.Link = api.LinkAgent
	case t.Mode == "local":
		v.Link = api.LinkLocalOnly
	default:
		v.Link = api.LinkSSHOnly
	}
	if t.Agent != nil {
		v.Agent = &api.AgentView{Address: t.Agent.Address, Transport: t.Agent.Transport, PairedAt: t.Agent.PairedAt}
	}
	return v
}

func sshView(c *executor.SSHConfig) *api.SSHView {
	if c == nil {
		return nil
	}
	return &api.SSHView{Host: c.Host, User: c.User, KeyPath: c.KeyPath, HostKeyFile: c.HostKeyFile, Port: c.Port, Jump: sshView(c.Jump)}
}

// sshConfig is the reverse, for requests that carry an address.
func sshConfig(v *api.SSHView) *executor.SSHConfig {
	if v == nil {
		return nil
	}
	return &executor.SSHConfig{Host: v.Host, User: v.User, KeyPath: v.KeyPath, HostKeyFile: v.HostKeyFile, Port: v.Port, Jump: sshConfig(v.Jump)}
}
