package updatecheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLatestSuccess(t *testing.T) {
	var gotPath, gotAccept, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.4.0",
			"html_url": "https://github.com/owner/repo/releases/tag/v0.4.0",
			"published_at": "2026-08-01T12:30:00Z",
			"extra_field": "ignored"
		}`))
	}))
	defer srv.Close()

	c := &Client{repo: "owner/repo", apiBase: srv.URL, http: srv.Client()}
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest returned error: %v", err)
	}

	if gotPath != "/repos/owner/repo/releases/latest" {
		t.Errorf("request path = %q, want /repos/owner/repo/releases/latest", gotPath)
	}
	if gotAccept != "application/vnd.github+json" {
		t.Errorf("Accept header = %q, want application/vnd.github+json", gotAccept)
	}
	if gotUA == "" {
		t.Error("User-Agent header is empty; GitHub rejects requests without a UA")
	}

	if rel.Version != "v0.4.0" {
		t.Errorf("Version = %q, want v0.4.0", rel.Version)
	}
	if rel.URL != "https://github.com/owner/repo/releases/tag/v0.4.0" {
		t.Errorf("URL = %q, want the release page", rel.URL)
	}
	want := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)
	if !rel.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt = %v, want %v", rel.PublishedAt, want)
	}
}

func TestLatestNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := &Client{repo: "owner/repo", apiBase: srv.URL, http: srv.Client()}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("Latest returned nil error for a 404 response")
	}
}

func TestLatestRateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer srv.Close()

	c := &Client{repo: "owner/repo", apiBase: srv.URL, http: srv.Client()}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("Latest returned nil error for a 403 response")
	}
}

func TestLatestMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	c := &Client{repo: "owner/repo", apiBase: srv.URL, http: srv.Client()}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("Latest returned nil error for a malformed JSON body")
	}
}

func TestLatestCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name":"v0.4.0"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := &Client{repo: "owner/repo", apiBase: srv.URL, http: srv.Client()}
	if _, err := c.Latest(ctx); err == nil {
		t.Fatal("Latest returned nil error for a canceled context")
	}
}

func TestNew(t *testing.T) {
	c := New("owner/repo")
	if c.repo != "owner/repo" {
		t.Errorf("repo = %q, want owner/repo", c.repo)
	}
	if c.apiBase != "https://api.github.com" {
		t.Errorf("apiBase = %q, want https://api.github.com", c.apiBase)
	}
	if c.http == nil {
		t.Fatal("http client is nil")
	}
	if c.http.Timeout != 10*time.Second {
		t.Errorf("http timeout = %v, want 10s", c.http.Timeout)
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal plain", "1.2.3", "1.2.3", 0},
		{"equal empty", "", "", 0},
		{"v-prefix vs none", "v1.2.3", "1.2.3", 0},
		{"both v-prefix", "v1.2.3", "v1.2.3", 0},
		{"major greater", "2.0.0", "1.9.9", 1},
		{"major less", "1.9.9", "2.0.0", -1},
		{"minor greater", "1.3.0", "1.2.9", 1},
		{"minor less", "1.2.0", "1.3.0", -1},
		{"patch greater", "1.2.4", "1.2.3", 1},
		{"patch less", "1.2.3", "1.2.4", -1},
		{"missing patch equals zero", "1.2", "1.2.0", 0},
		{"missing minor and patch", "1", "1.0.0", 0},
		{"missing component less", "1.2", "1.2.1", -1},
		{"prerelease below release", "1.2.3-rc1", "1.2.3", -1},
		{"release above prerelease", "1.2.3", "1.2.3-rc1", 1},
		{"two prereleases rc1 rc2", "1.2.3-rc1", "1.2.3-rc2", -1},
		{"two prereleases rc2 rc1", "1.2.3-rc2", "1.2.3-rc1", 1},
		{"two prereleases equal", "1.2.3-rc1", "1.2.3-rc1", 0},
		{"garbage reads as zero", "abc", "0.0.0", 0},
		{"garbage component", "1.x.3", "1.0.3", 0},
		{"garbage vs real", "1.2.garbage", "1.2.5", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CompareSemver(tt.a, tt.b); got != tt.want {
				t.Errorf("CompareSemver(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name                    string
		current, latest         string
		releaseURL              string
		wantUpdateAvailable     bool
		wantCurrent, wantLatest string
		wantReleaseURL          string
	}{
		{
			name:                "update available",
			current:             "0.3.0",
			latest:              "0.4.0",
			releaseURL:          "https://example/rel",
			wantUpdateAvailable: true,
			wantCurrent:         "0.3.0",
			wantLatest:          "0.4.0",
			wantReleaseURL:      "https://example/rel",
		},
		{
			name:                "not available when equal",
			current:             "0.4.0",
			latest:              "0.4.0",
			wantUpdateAvailable: false,
			wantCurrent:         "0.4.0",
			wantLatest:          "0.4.0",
		},
		{
			name:                "not available when current newer",
			current:             "0.5.0",
			latest:              "0.4.0",
			wantUpdateAvailable: false,
			wantCurrent:         "0.5.0",
			wantLatest:          "0.4.0",
		},
		{
			name:                "not available when current is dev",
			current:             "dev",
			latest:              "0.4.0",
			wantUpdateAvailable: false,
			wantCurrent:         "dev",
			wantLatest:          "0.4.0",
		},
		{
			name:                "not available when current is empty",
			current:             "",
			latest:              "0.4.0",
			wantUpdateAvailable: false,
			wantCurrent:         "",
			wantLatest:          "0.4.0",
		},
		{
			name:                "not available when latest is empty",
			current:             "0.3.0",
			latest:              "",
			wantUpdateAvailable: false,
			wantCurrent:         "0.3.0",
			wantLatest:          "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Evaluate(tt.current, tt.latest, tt.releaseURL)
			if got.UpdateAvailable != tt.wantUpdateAvailable {
				t.Errorf("UpdateAvailable = %v, want %v", got.UpdateAvailable, tt.wantUpdateAvailable)
			}
			if got.Current != tt.wantCurrent {
				t.Errorf("Current = %q, want %q", got.Current, tt.wantCurrent)
			}
			if got.Latest != tt.wantLatest {
				t.Errorf("Latest = %q, want %q", got.Latest, tt.wantLatest)
			}
			if got.ReleaseURL != tt.wantReleaseURL {
				t.Errorf("ReleaseURL = %q, want %q", got.ReleaseURL, tt.wantReleaseURL)
			}
		})
	}
}
