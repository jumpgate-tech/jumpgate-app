// internal/joblog/bodies.go
package joblog

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"github.com/valve-tech/jumpgate/internal/intent"
)

// Entry kinds.
const (
	KindCreated          = "job.created"       // IntentBody: the request that created the job
	KindIntent           = "intent"            // IntentBody: a later request that changed the job
	KindStepStarted      = "step.started"      // StepBody
	KindStepFinished     = "step.finished"     // StepBody with OK and Error
	KindStepLost         = "step.lost"         // StepBody: the unit ended without a result
	KindDecisionAsked    = "decision.asked"    // intent.JobQuestion
	KindDecisionAnswered = "decision.answered" // AnswerBody
	KindLeaseTaken       = "lease.taken"       // LeaseBody
	KindLeaseReleased    = "lease.released"    // LeaseBody
	KindFinished         = "job.finished"      // FinishedBody
)

// IntentBody records a request exactly as the controller signed it.
type IntentBody struct {
	Envelope intent.Envelope `json:"envelope"`
	Outcome  string          `json:"outcome"` // "created", "lease take", "advance", "decide", …
}

type StepBody struct {
	Step    string `json:"step"`
	Attempt int    `json:"attempt"`
	Unit    string `json:"unit,omitempty"`
	OK      bool   `json:"ok,omitempty"`
	Error   string `json:"error,omitempty"`
}

type AnswerBody struct {
	Question   string   `json:"question"`
	Option     string   `json:"option"`
	Controller string   `json:"controller"`
	Signers    []string `json:"signers"`
}

type LeaseBody struct {
	Owner    string `json:"owner"`
	Previous string `json:"previous,omitempty"`
}

type FinishedBody struct {
	State string `json:"state"`
	Error string `json:"error,omitempty"`
}

// Describe is an entry as one line for a person. It reads only verified
// entries' bodies; an unreadable body says so rather than guessing.
func Describe(e Entry) string {
	return clean(describe(e))
}

// clean drops control characters. A step's error text comes from unsigned
// step output and reaches a terminal, where an escape sequence could rewrite
// what the person sees.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func describe(e Entry) string {
	bad := fmt.Sprintf("%s (unreadable body)", e.Kind)
	switch e.Kind {
	case KindCreated, KindIntent:
		var b IntentBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		return fmt.Sprintf("%s by %s: %s", b.Envelope.Intent.Kind, b.Envelope.Intent.Controller, b.Outcome)
	case KindStepStarted, KindStepFinished, KindStepLost:
		var b StepBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		switch e.Kind {
		case KindStepStarted:
			return fmt.Sprintf("step %s attempt %d started (%s)", b.Step, b.Attempt, b.Unit)
		case KindStepLost:
			return fmt.Sprintf("step %s attempt %d stopped without a result", b.Step, b.Attempt)
		}
		if b.OK {
			return fmt.Sprintf("step %s attempt %d finished", b.Step, b.Attempt)
		}
		return fmt.Sprintf("step %s attempt %d failed: %s", b.Step, b.Attempt, b.Error)
	case KindDecisionAsked:
		var q intent.JobQuestion
		if json.Unmarshal(e.Body, &q) != nil {
			return bad
		}
		// Describe is for one-line lists: a long prompt (a process list) is
		// cut to 160 runes; the inbox shows the whole question.
		p := []rune(strings.ReplaceAll(q.Prompt, "\n", " "))
		if len(p) > 160 {
			p = append(p[:159], '…')
		}
		return fmt.Sprintf("asked (%s tier): %s", q.Tier, string(p))
	case KindDecisionAnswered:
		var b AnswerBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		return fmt.Sprintf("answered %s by %s", b.Option, b.Controller)
	case KindLeaseTaken, KindLeaseReleased:
		var b LeaseBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		if e.Kind == KindLeaseReleased {
			return "lease released by " + b.Owner
		}
		if b.Previous != "" {
			return fmt.Sprintf("lease taken by %s from %s", b.Owner, b.Previous)
		}
		return "lease taken by " + b.Owner
	case KindFinished:
		var b FinishedBody
		if json.Unmarshal(e.Body, &b) != nil {
			return bad
		}
		if b.Error != "" {
			return fmt.Sprintf("job %s: %s", b.State, b.Error)
		}
		return "job " + b.State
	}
	return e.Kind
}
