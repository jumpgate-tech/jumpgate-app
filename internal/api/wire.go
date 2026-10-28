package api

import (
	"encoding/json"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/intent"
)

// PairingInfo is GET /api/pairing-info: what a CLI needs to install and
// enroll an agent in its own process (the jumpgate service never logs in to
// a box). Everything in it is public.
type PairingInfo struct {
	Controller   string              `json:"controller"`      // the recorded controller address
	TransportKey string              `json:"transportKey"`    // the tunnel key, an authorized_keys line
	System       bool                `json:"system"`          // serve --system
	ServiceUID   int                 `json:"serviceUid"`      // the service's uid in system mode, else -1
	Scope        string              `json:"scope,omitempty"` // "self" under --self-only
	Wire         *catalog.WireConfig `json:"wire,omitempty"`  // the named target's, when ?target= is given
}

// SystemInitKey is POST /api/system/init-key: `jumpgate setup server` asks
// the jumpgate service to create its controller key. Passphrase is a byte
// slice (base64 in JSON) so both ends can clear it; it is never logged. A
// file key's path is always the service's own: Ref is refused for it.
type SystemInitKey struct {
	Store         string `json:"store"` // "vault" (default) | "1password" | "file"
	Ref           string `json:"ref,omitempty"`
	Passphrase    []byte `json:"passphrase,omitempty"`
	Scope         string `json:"scope,omitempty"` // "" or "self"
	InsecurePlain bool   `json:"insecurePlain,omitempty"`
}

// IntentReply is the server's answer to POST /api/targets/{id}/intent/{kind}:
// the verified receipt. Hint is the server's remedy for a rejection.
type IntentReply struct {
	Status    uint8             `json:"status"`
	Result    json.RawMessage   `json:"result,omitempty"`
	Rejection *intent.Rejection `json:"rejection,omitempty"`
	Failure   *intent.Failure   `json:"failure,omitempty"`
	RefHead   uint64            `json:"refHead,omitempty"` // status.read only
	Hint      string            `json:"hint,omitempty"`
}

// PairRequest is POST /api/targets/{id}/pair's body. Installed is the agent
// address a foreground CLI already installed (platform Task 5).
type PairRequest struct {
	Sudo      bool   `json:"sudo"`
	Installed string `json:"installed,omitempty"`
}

// PairEvent is one SSE frame of a pairing: a step's progress line, the error
// that ended it, or the final {done, agent}.
type PairEvent struct {
	Step  string `json:"step,omitempty"`
	Line  string `json:"line,omitempty"`
	Error string `json:"error,omitempty"`
	Code  Code   `json:"code,omitempty"`
	Hint  string `json:"hint,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Agent string `json:"agent,omitempty"`
}

// AgentUpgradeView is POST /api/targets/{id}/agent/upgrade: whether the
// box's gate was replaced, and the versions before and after.
type AgentUpgradeView struct {
	Upgraded bool   `json:"upgraded"`
	From     string `json:"from"`
	To       string `json:"to"`
}
