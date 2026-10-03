package agent

// Reject is a refusal the agent signs and returns. Code is one of the
// intent.Reason* constants.
type Reject struct {
	Code    string
	Message string
	LastSeq uint64
}

func (r *Reject) Error() string { return r.Code + ": " + r.Message }

func reject(code, msg string) *Reject { return &Reject{Code: code, Message: msg} }
