package api

import (
	"net"
	"strconv"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

// Link is how this controller reaches a target, as the fleet shows it.
type Link string

const (
	LinkAgent      Link = "agent"       // paired; agent over the SSH tunnel
	LinkAgentLocal Link = "agent_local" // paired; this machine's agent socket
	LinkSSHOnly    Link = "ssh_only"    // not paired; legacy root SSH
	LinkLocalOnly  Link = "local_only"  // this machine, not paired
)

// Paired reports whether intents reach this target's agent.
func (l Link) Paired() bool { return l == LinkAgent || l == LinkAgentLocal }

// SSHView is a target's SSH address. Its JSON names are executor.SSHConfig's,
// which the web UI already reads; it carries no runtime fields.
type SSHView struct {
	Host        string   `json:"Host"`
	User        string   `json:"User"`
	KeyPath     string   `json:"KeyPath"`
	HostKeyFile string   `json:"HostKeyFile"`
	Port        int      `json:"Port"`
	Jump        *SSHView `json:"jump,omitempty"`
}

// Address renders user@host:port and its jump chain, for people.
func (v SSHView) Address() string {
	port := v.Port
	if port == 0 {
		port = 22
	}
	s := v.User + "@" + net.JoinHostPort(v.Host, strconv.Itoa(port))
	if v.Jump != nil {
		s += " via " + v.Jump.Address()
	}
	return s
}

// AgentView is a target's pairing, without the controller's private
// bookkeeping (sequence numbers, test socket overrides).
type AgentView struct {
	Address   string    `json:"address"`
	Transport string    `json:"transport"` // "ssh" | "local"
	PairedAt  time.Time `json:"pairedAt"`
}

// TargetView is GET /api/targets' element: the stored target with today's
// field names, plus how it is reached.
type TargetView struct {
	ID          string                `json:"id"`
	Mode        string                `json:"mode"` // "local" | "ssh"
	SSH         *SSHView              `json:"ssh,omitempty"`
	Wire        *catalog.WireConfig   `json:"wire,omitempty"`
	Devnet      *catalog.DevnetConfig `json:"devnet,omitempty"`
	Agent       *AgentView            `json:"agent"`
	Link        Link                  `json:"link"`
	ThisMachine bool                  `json:"thisMachine"`
}

// AddTarget is POST /api/targets' body.
type AddTarget struct {
	ID   string   `json:"id"`
	Mode string   `json:"mode"`
	SSH  *SSHView `json:"ssh,omitempty"`
}
