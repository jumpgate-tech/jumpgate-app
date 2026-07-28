package ai

// The ways a provider can answer 200 and still have told us nothing.
//
// A non-200 is the easy case and is already covered. These are the harder
// ones: a body that is not JSON, a well-formed reply with no candidates, and
// a candidate whose text is empty. All three must be errors — returning ""
// as an explanation puts an empty panel in front of the operator with nothing
// saying the provider is the reason.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func explainReq() ExplainRequest {
	return ExplainRequest{
		Lines:      []string{"ERROR something broke"},
		ExecClient: "reth",
		ChainName:  "PulseChain",
	}
}

// serving stands up a server returning one canned body and returns a provider
// pointed at it.
func serving(t *testing.T, id string, status int, body string) Provider {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ts.Close)

	p, err := New(id, "sk-test", ts.URL)
	if err != nil {
		t.Fatalf("New(%q): %v", id, err)
	}
	return p
}

func TestExplain_AnEmptyAnswerIsAnErrorNotAnEmptyExplanation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		body    string
		wantSay string
	}{
		{
			name: "gemini: no candidates",
			id:   "gemini", body: `{"candidates":[]}`,
			wantSay: "gemini",
		},
		{
			name: "gemini: a candidate with no parts",
			id:   "gemini", body: `{"candidates":[{"content":{"parts":[]}}]}`,
			wantSay: "gemini",
		},
		{
			name: "gemini: not JSON at all",
			id:   "gemini", body: `<html>200 but a proxy answered</html>`,
			wantSay: "decode",
		},
		{
			name: "groq: no choices",
			id:   "groq", body: `{"choices":[]}`,
			wantSay: "choices",
		},
		{
			name: "groq: a choice with empty content",
			id:   "groq", body: `{"choices":[{"message":{"role":"assistant","content":""}}]}`,
			wantSay: "content",
		},
		{
			name: "groq: not JSON at all",
			id:   "groq", body: `<html>200 but a proxy answered</html>`,
			wantSay: "decode",
		},
		{
			name: "ollama: empty content",
			id:   "ollama", body: `{"message":{"role":"assistant","content":""}}`,
			wantSay: "ollama",
		},
		{
			name: "ollama: not JSON at all",
			id:   "ollama", body: `<html>200 but a proxy answered</html>`,
			wantSay: "decode",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := serving(t, tc.id, http.StatusOK, tc.body)

			got, err := p.Explain(context.Background(), explainReq())
			if err == nil {
				t.Fatalf("an empty answer came back as the explanation %q", got)
			}
			if got != "" {
				t.Errorf("an error came with text attached: %q", got)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.wantSay) {
				t.Errorf("error does not say what happened (want %q): %v", tc.wantSay, err)
			}
		})
	}
}

// A provider that cannot be reached at all is an error naming the provider,
// so the operator knows which key or endpoint to go check.
func TestExplain_AnUnreachableProviderNamesItself(t *testing.T) {
	for _, id := range []string{"gemini", "groq", "ollama"} {
		t.Run(id, func(t *testing.T) {
			// A server that is closed immediately: the port is not listening.
			ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			url := ts.URL
			ts.Close()

			p, err := New(id, "sk-test", url)
			if err != nil {
				t.Fatalf("New(%q): %v", id, err)
			}
			if _, err := p.Explain(context.Background(), explainReq()); err == nil {
				t.Fatal("an unreachable provider returned an explanation")
			} else if !strings.Contains(err.Error(), id) {
				t.Errorf("error does not name the provider: %v", err)
			}
		})
	}
}

// A canceled context stops the call rather than blocking the request handler
// that is waiting on it.
func TestExplain_RespectsACanceledContext(t *testing.T) {
	p := serving(t, "groq", http.StatusOK, `{"choices":[{"message":{"content":"fine"}}]}`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := p.Explain(ctx, explainReq()); err == nil {
		t.Fatal("a canceled context still produced an explanation")
	}
}

// Every provider reports its own name, which is what the UI shows next to the
// explanation — an answer attributed to the wrong provider is worse than one
// attributed to none.
func TestProviders_ReportTheirOwnName(t *testing.T) {
	for _, id := range []string{"gemini", "groq", "ollama"} {
		p, err := New(id, "sk-test", "")
		if err != nil {
			t.Fatalf("New(%q): %v", id, err)
		}
		if got := p.Name(); got != id {
			t.Errorf("Name = %q, want %q", got, id)
		}
	}
}

// ---------------------------------------------------------------------
// capLines
// ---------------------------------------------------------------------

// The tail is kept, not the head: an operator asking "what is wrong right
// now" is served by the most recent lines, and these go to a third party, so
// sending more than the cap is a leak of volume as well as a cost.
func TestCapLines_KeepsTheTail(t *testing.T) {
	lines := make([]string, maxExplainLines+50)
	for i := range lines {
		lines[i] = "line"
	}
	lines[len(lines)-1] = "THE MOST RECENT"

	got := capLines(lines)
	if len(got) > maxExplainLines {
		t.Fatalf("kept %d lines, want at most %d", len(got), maxExplainLines)
	}
	if got[len(got)-1] != "THE MOST RECENT" {
		t.Error("the newest line was dropped, which is the one being asked about")
	}
}

func TestCapLines_ShortInputIsUntouched(t *testing.T) {
	in := []string{"a", "b", "c"}
	got := capLines(in)
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("got %v, want it left alone", got)
	}
}

// A single line larger than the byte budget cannot be trimmed into it, and
// the loop must terminate rather than spin or panic on the empty slice.
func TestCapLines_ASingleOversizeLineTerminates(t *testing.T) {
	got := capLines([]string{strings.Repeat("x", maxExplainBytes*2)})
	if len(got) > 1 {
		t.Errorf("got %d lines out of one", len(got))
	}
}

func TestCapLines_EmptyInput(t *testing.T) {
	if got := capLines(nil); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}
