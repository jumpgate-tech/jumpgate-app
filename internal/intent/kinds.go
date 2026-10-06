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
	ServiceActionPayload struct {
		Service string `json:"service"` // "exec" | "beacon"
		Action  string `json:"action"`  // "start" | "stop" | "restart"
	}
)

// AgentInfo answers agent.info.
type AgentInfo struct {
	Version      string   `json:"version"`
	Address      string   `json:"address"`
	LastSeq      uint64   `json:"lastSeq"` // for the asking controller
	PlanVersions []string `json:"planVersions"`
	Signers      int      `json:"signers"`
	SetUp        bool     `json:"setUp"`
	ChainID      int      `json:"chainId,omitempty"` // the box's node; 0 before setup
}

// Logs limits.
const (
	LogsDefaultN = 200
	LogsMaxN     = 2000
)
