package intent

import (
	"encoding/json"
	"testing"
)

func TestJobTerminalStates(t *testing.T) {
	for s, want := range map[string]bool{
		JobWaiting: false, JobRunning: false, JobNeedsDecision: false,
		JobSucceeded: true, JobFailed: true, JobAborted: true, JobDamaged: true,
	} {
		if JobTerminal(s) != want {
			t.Errorf("JobTerminal(%s) = %v", s, !want)
		}
	}
}

// The record is the agent's signed answer and the manifest's core; its JSON
// names are a wire contract with every controller version.
func TestJobRecordWireNames(t *testing.T) {
	b, _ := json.Marshal(JobRecord{ID: "0123456789abcdef", Kind: KindNodeMigrateNames, State: JobWaiting,
		Lease:    JobLease{Owner: "0xabc", HeartbeatAt: 5, TTLSeconds: 90},
		Question: &JobQuestion{ID: "q", Tier: TierRoutine, Options: []JobOption{{ID: OptRetry, Label: "Retry"}}}})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for _, k := range []string{"format", "id", "kind", "title", "planVersion", "driver", "steps", "step", "attempt", "state", "lease", "question", "createdBy", "createdAt", "updatedAt", "logLen", "logHead", "agentVersion"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JobRecord JSON lacks %q: %s", k, b)
		}
	}
}
