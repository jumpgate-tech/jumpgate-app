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
