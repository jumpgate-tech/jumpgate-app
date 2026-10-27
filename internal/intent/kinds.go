package intent

// Kinds in this sub-project. All are routine tier.
const (
	KindAgentInfo     = "agent.info"
	KindStatusRead    = "status.read"
	KindDiskRead      = "disk.read"
	KindEndpointsRead = "endpoints.read"
	KindFirewallRead  = "firewall.read"
	KindLogsRead      = "logs.read"
	KindLogsSince     = "logs.since"
	KindServiceAction = "service.action"
	// KindModulesRead reads the box's box modules. Like agent.info it needs
	// no node.json, so a bootnode-only box reports status.
	KindModulesRead = "modules.read"
	// KindAgentUpgrade replaces the box's agent binary. It is sent to
	// /v1/upgrade with the binary as the request body, never to /v1/intent.
	KindAgentUpgrade = "agent.upgrade"
)

// Reason codes carried by a rejected receipt's result. They are stable: the
// CLI maps each to a one-line remedy.
const (
	ReasonWrongAgent       = "wrong_agent"
	ReasonExpired          = "expired"
	ReasonClockSkew        = "clock_skew"
	ReasonBadSignature     = "bad_signature"
	ReasonUnauthorizedKind = "unauthorized_kind"
	ReasonUnknownKind      = "unknown_kind"
	ReasonStaleSeq         = "stale_seq"
	ReasonReplayedNonce    = "replayed_nonce"
	ReasonBusy             = "busy"
	ReasonInvalidPayload   = "invalid_payload"
	ReasonValidation       = "validation"
	ReasonNotSetUp         = "not_set_up"
	ReasonReplayState      = "replay_state"
	ReasonScopeLocal       = "scope_local" // a local-only controller arrived through the tunnel
)

// Rejection is the result of a rejected intent. LastSeq lets a controller
// whose counter fell behind resynchronise; AgentTime lets it report skew.
type Rejection struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	LastSeq   uint64 `json:"lastSeq,omitempty"`
	AgentTime int64  `json:"agentTime,omitempty"`
}

// Failure is the result of an intent that ran and failed.
type Failure struct {
	Message string `json:"message"`
}

// Payloads.
type (
	LogsReadPayload struct {
		N int `json:"n"`
	}
	EndpointsReadPayload struct {
		SSHLogin string `json:"sshLogin"`
	}
	// LogsSincePayload asks for journal lines after Cursor (empty: the last N).
	LogsSincePayload struct {
		Cursor string `json:"cursor"`
		N      int    `json:"n"`
	}
	// AgentUpgradePayload describes the binary an agent.upgrade streams.
	AgentUpgradePayload struct {
		Arch    string `json:"arch"`
		Version string `json:"version"`
		SHA256  string `json:"sha256"` // hex
		Size    int64  `json:"size"`
	}
	// AgentUpgradeResult answers agent.upgrade. Upgraded is false when the
	// binary sent is the one already installed.
	AgentUpgradeResult struct {
		Upgraded bool   `json:"upgraded"`
		From     string `json:"from"`
		To       string `json:"to"`
	}
	ServiceActionPayload struct {
		Service string `json:"service"` // "exec" | "beacon"
		Action  string `json:"action"`  // "start" | "stop" | "restart"
	}
)

// AgentInfo answers agent.info.
type AgentInfo struct {
	Version      string   `json:"version"`
	Arch         string   `json:"arch"` // runtime.GOARCH of the agent binary
	Address      string   `json:"address"`
	LastSeq      uint64   `json:"lastSeq"` // for the asking controller
	PlanVersions []string `json:"planVersions"`
	Signers      int      `json:"signers"`
	SetUp        bool     `json:"setUp"`
	ChainID      int      `json:"chainId,omitempty"` // the box's node; 0 before setup
	Names        string   `json:"names,omitempty"`   // catalog names id of the box's node; "" is legacy
	DataDir      string   `json:"dataDir,omitempty"` // the node's data dir, after any names marker
	JWTPath      string   `json:"jwtPath,omitempty"` // the node's JWT secret, after any names marker
	// SetUpError is why the box's node config could not be read, when a
	// node.json or names.json exists but is refused (SetUp is then false).
	// It is never set for a box with no node.json: a caller must not take a
	// box whose config is broken for a new one.
	SetUpError *Rejection `json:"setUpError,omitempty"`
}

// Logs limits.
const (
	LogsDefaultN = 200
	LogsMaxN     = 2000
)
