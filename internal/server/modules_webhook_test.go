package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/config"
)

const savedHook = "https://ntfy.example/s3cret-topic"

// The saved webhook is a secret: target views say only that one is set.
func TestTargetViewMasksWatchWebhook(t *testing.T) {
	tg := config.Target{ID: "box", Mode: "local", Modules: []catalog.BoxModule{{ID: "watch", Webhook: savedHook, Upstreams: []string{"gw=https://rpc.example"}}}}
	v := targetView(tg)
	raw, _ := json.Marshal(v)
	if strings.Contains(string(raw), "s3cret") || !v.WatchWebhookSet || v.Modules[0].Upstreams[0] != "gw=https://rpc.example" {
		t.Fatalf("view = %s", raw)
	}
	if tg.Modules[0].Webhook != savedHook {
		t.Fatal("targetView changed the stored record")
	}
	if v := targetView(config.Target{ID: "x"}); v.WatchWebhookSet {
		t.Fatal("webhookSet without a watch")
	}
}

// An empty webhook in a change keeps the saved one; clearWebhook removes it.
func TestSetWatchKeepsOrClearsWebhook(t *testing.T) {
	fakeModules(t)
	a := newAPITestServer(t)
	moduleTarget(t, a, func(c *config.Config) {
		targetByID(c, "box").Modules = []catalog.BoxModule{{ID: "watch", Webhook: savedHook}}
	})
	put := func(body map[string]any) {
		t.Helper()
		res := a.do(t, "PUT", "/api/targets/box/modules/watch", body)
		if res.StatusCode != http.StatusAccepted {
			b, _ := io.ReadAll(res.Body)
			t.Fatalf("status %d: %s", res.StatusCode, b)
		}
		res.Body.Close()
		waitSetupIdle(t, a, "box")
	}
	put(map[string]any{"enabled": true, "upstreams": []string{"gw=https://rpc.example"}})
	if got := storedTarget(t, "box").Modules; len(got) != 1 || got[0].Webhook != savedHook || len(got[0].Upstreams) != 1 {
		t.Fatalf("after keep: %+v", got)
	}
	put(map[string]any{"enabled": true, "webhook": "https://ntfy.example/other"})
	if got := storedTarget(t, "box").Modules; got[0].Webhook != "https://ntfy.example/other" {
		t.Fatalf("after replace: %+v", got)
	}
	res := a.do(t, "PUT", "/api/targets/box/modules/watch", map[string]any{"enabled": true, "webhook": "http://ntfy.example/s3cret-plain"})
	if e := readAPIError(t, res); res.StatusCode != http.StatusBadRequest || strings.Contains(e.Message, "s3cret") {
		t.Fatalf("http webhook: %d %+v", res.StatusCode, e)
	}
	put(map[string]any{"enabled": true, "clearWebhook": true})
	if got := storedTarget(t, "box").Modules; got[0].Webhook != "" {
		t.Fatalf("after clear: %+v", got)
	}
}
