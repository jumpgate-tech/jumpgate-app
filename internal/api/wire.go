package api

import (
	"encoding/json"

	"github.com/valve-tech/jumpgate/internal/intent"
)

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
