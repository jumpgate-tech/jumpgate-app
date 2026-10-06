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
	"unicode/utf8"
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
	lines := make([]string, MaxExplainLines+50)
	for i := range lines {
		lines[i] = "line"
	}
	lines[len(lines)-1] = "THE MOST RECENT"

	got := CapLines(lines)
	if len(got) > MaxExplainLines {
		t.Fatalf("kept %d lines, want at most %d", len(got), MaxExplainLines)
	}
	if got[len(got)-1] != "THE MOST RECENT" {
		t.Error("the newest line was dropped, which is the one being asked about")
	}
}

func TestCapLines_ShortInputIsUntouched(t *testing.T) {
	in := []string{"a", "b", "c"}
	got := CapLines(in)
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("got %v, want it left alone", got)
	}
}

// A single line larger than the byte budget cannot be trimmed into it, and
// the loop must terminate rather than spin or panic on the empty slice.
func TestCapLines_ASingleOversizeLineTerminates(t *testing.T) {
	got := CapLines([]string{strings.Repeat("x", MaxExplainBytes*2)})
	if len(got) > 1 {
		t.Errorf("got %d lines out of one", len(got))
	}
}

func TestCapLines_EmptyInput(t *testing.T) {
	if got := CapLines(nil); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}

// The regression for capLines dropping everything: when the newest line on
// its own exceeds the byte budget, popping whole lines from the front empties
// the batch and the provider is asked to explain nothing, so it invents a
// diagnosis. The newest line must survive, cut down to fit, with both of its
// ends kept: the head carries the timestamp, level and message that say what
// failed, and the tail carries the trailing error= detail that says why.
func TestCapLines_AnOversizeNewestLineIsTruncatedNotDropped(t *testing.T) {
	head := "2026-07-23T03:00:00Z ERROR rpc: request failed payload="
	tail := ` error="401 Unauthorized"`
	huge := head + strings.Repeat("ab", MaxExplainBytes) + tail

	got := CapLines([]string{"older line", huge})
	if len(got) != 1 {
		t.Fatalf("got %d lines, want exactly the truncated newest one", len(got))
	}
	if n := len(got[0]) + 1; n > MaxExplainBytes {
		t.Errorf("truncated line is %d bytes with its newline, want <= %d", n, MaxExplainBytes)
	}
	if !strings.HasPrefix(got[0], head) {
		t.Errorf("truncation lost the line's head (level and message): %.80q", got[0])
	}
	if !strings.HasSuffix(got[0], tail) {
		t.Errorf("truncation lost the line's tail (the error detail): %q", got[0][max(0, len(got[0])-80):])
	}
	if !strings.Contains(got[0], "truncated") {
		t.Error("truncated line does not say it was truncated, so the model may read the join as real text")
	}
}

// Truncation must not split a multi-byte character, or the prompt carries
// invalid UTF-8 to the provider.
func TestCapLines_TruncationKeepsValidUTF8(t *testing.T) {
	got := CapLines([]string{strings.Repeat("é", MaxExplainBytes)})
	if len(got) != 1 {
		t.Fatalf("got %d lines, want 1", len(got))
	}
	if !utf8.ValidString(got[0]) {
		t.Error("truncated line is not valid UTF-8")
	}
	if n := len(got[0]) + 1; n > MaxExplainBytes {
		t.Errorf("truncated line is %d bytes with its newline, want <= %d", n, MaxExplainBytes)
	}
}

// A non-empty input never yields an empty batch, whatever its shape.
func TestCapLines_NonEmptyInputNeverYieldsAnEmptyBatch(t *testing.T) {
	inputs := [][]string{
		{strings.Repeat("x", MaxExplainBytes)},
		{strings.Repeat("x", MaxExplainBytes-1)},
		{"short", strings.Repeat("y", 3*MaxExplainBytes)},
	}
	for _, in := range inputs {
		if got := CapLines(in); len(got) == 0 {
			t.Errorf("CapLines(%d lines, newest %d bytes) returned nothing", len(in), len(in[len(in)-1]))
		}
	}
}

// The Gemini key travels in a header, not the URL. A key in the query string
// rides along into every *url.Error the transport returns, and from there into
// logs and API responses.
func TestGemini_KeyIsAHeaderNotAQueryParameter(t *testing.T) {
	var gotQuery, gotKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotKey = r.Header.Get("x-goog-api-key")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	t.Cleanup(ts.Close)

	p, err := New("gemini", "sk-secret-gemini", ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Explain(context.Background(), explainReq()); err != nil {
		t.Fatalf("Explain: %v", err)
	}
	if strings.Contains(gotQuery, "sk-secret-gemini") {
		t.Errorf("key in the query string: %q", gotQuery)
	}
	if gotKey != "sk-secret-gemini" {
		t.Errorf("x-goog-api-key = %q, want the key", gotKey)
	}
}

func TestGemini_TransportErrorsDoNotCarryTheKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := ts.URL
	ts.Close()

	p, err := New("gemini", "sk-secret-gemini", url)
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Explain(context.Background(), explainReq())
	if err == nil {
		t.Fatal("expected an error from a closed server")
	}
	if strings.Contains(err.Error(), "sk-secret-gemini") {
		t.Errorf("error leaks the key: %v", err)
	}
}
