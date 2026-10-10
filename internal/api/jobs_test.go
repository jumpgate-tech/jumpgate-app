package api

import (
	"encoding/json"
	"testing"
)

func TestInboxCountsJobsThatNeedADecision(t *testing.T) {
	in := JobsInbox{Jobs: []JobView{{State: "needs-decision"}, {State: "running"}, {State: "needs-decision"}}}
	if in.NeedsDecision() != 2 {
		t.Fatal(in.NeedsDecision())
	}
}

func TestJobViewWireNames(t *testing.T) {
	b, err := json.Marshal(JobView{Target: "box-a", ID: "0123456789abcdef", Question: &JobQuestion{ID: "q", Signable: true}})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"target", "id", "kind", "title", "state", "step", "steps", "attempt", "question", "lease", "createdAt", "updatedAt", "driving"} {
		if _, ok := m[k]; !ok {
			t.Errorf("JobView JSON lacks %q: %s", k, b)
		}
	}
}

func TestJobCodesAreRegisteredWithTheirExitClass(t *testing.T) {
	for c, exit := range map[Code]int{
		CodeJobNotFound: ExitFailed, CodeLeaseHeld: ExitFailed, CodeJobActive: ExitFailed, CodeJobsUnsupported: ExitFailed,
		CodeJobState: ExitFailed, CodeApprovalUnavailable: ExitUsage, CodeJobLogInvalid: ExitSecurity,
		CodeUnknownPlan: ExitFailed, CodeJobKindNotAllowed: ExitUsage,
	} {
		if !Known(c) || c.Exit() != exit || HintFor(c) == "" {
			t.Errorf("%s: known %v exit %d hint %q", c, Known(c), c.Exit(), HintFor(c))
		}
	}
}
