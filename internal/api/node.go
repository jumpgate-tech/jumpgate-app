package api

import "time"

// LogHit is one classified journal line, as the logs routes send it. Its JSON
// matches internal/logwatch.Hit, which the server sends unchanged.
type LogHit struct {
	Unit      string    `json:"unit"`
	Line      string    `json:"line"`
	At        time.Time `json:"at"`
	Signature string    `json:"signature"`
	Severity  string    `json:"severity"` // info|warn|error|critical
	Explain   string    `json:"explain"`
	LearnURL  string    `json:"learnUrl,omitempty"`
}

// ServiceResult answers a service action: the unit's state afterwards.
type ServiceResult struct {
	Active bool `json:"Active"`
}

// Endpoints is a node's RPC endpoints and whether they answer. Its JSON names
// are ops.EndpointInfo's, which the web UI reads.
type Endpoints struct {
	ExecHTTP        string `json:"ExecHTTP"`
	BeaconHTTP      string `json:"BeaconHTTP"`
	ExecReachable   bool   `json:"ExecReachable"`
	BeaconReachable bool   `json:"BeaconReachable"`
	ChainIDMatches  bool   `json:"ChainIDMatches"`
	Access          string `json:"Access"` // "local" | "ssh"
	TunnelHint      string `json:"TunnelHint"`
}

// CheckItem is one firewall checklist line (ops.CheckItem's JSON names).
type CheckItem struct {
	ID     string `json:"ID"`
	Title  string `json:"Title"`
	Why    string `json:"Why"`
	Status string `json:"Status"` // pass | fail | warn | unknown
	Detail string `json:"Detail"`
	Fix    string `json:"Fix"`
}
