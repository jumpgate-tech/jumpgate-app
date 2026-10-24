package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valve-tech/jumpgate/internal/catalog"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/jobs"
	"github.com/valve-tech/jumpgate/internal/monitor"
)

// DefaultInterval is the watch's poll interval when neither the config nor
// JUMPGATE_WATCH_INTERVAL sets one.
const DefaultInterval = 30 * time.Second

// Config is one watch's settings.
type Config struct {
	Wire      *catalog.WireConfig // the box's node, nil if none
	Endpoints []Endpoint
	Interval  time.Duration // 0 → JUMPGATE_WATCH_INTERVAL, else 30s
	StatePath string        // <DataDir>/state.json
	Webhook   string        // "" = none; https only; never logged (Task 16 delivers)
	RefRPC    string        // reference endpoint for exec-behind; "" = no lag rule
}

// LoadConfig builds a Config from the module's config.json (a
// catalog.BoxModule) and, when it exists, the node.json setup stages
// beside it (catalog.BoxModule.NodePath: the box's node, secrets stripped,
// names already resolved).
func LoadConfig(configPath string) (Config, error) {
	var cfg Config
	b, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, err
	}
	var m catalog.BoxModule
	if err := json.Unmarshal(b, &m); err != nil {
		return cfg, fmt.Errorf("%s does not parse: %w", configPath, err)
	}
	if m.ID != "watch" {
		return cfg, fmt.Errorf("%s is not a health watch record", configPath)
	}
	if cfg.Endpoints, err = catalog.WatchEndpoints(m); err != nil {
		return cfg, err
	}
	if err := catalog.ValidateWatchWebhook(m.Webhook); err != nil {
		return cfg, err
	}
	cfg.Webhook = m.Webhook
	cfg.StatePath = m.DataDir() + "/state.json"
	w, err := jobs.ReadNodeConfig(filepath.Join(filepath.Dir(configPath), "node.json"), "")
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		// The watch still covers the endpoints; say why the node is not.
		log.Printf("watch: not watching the node: %v", err)
	default:
		cfg.Wire = &w
	}
	return cfg, nil
}

func interval(cfg Config) time.Duration {
	if cfg.Interval > 0 {
		return cfg.Interval
	}
	if d, err := time.ParseDuration(os.Getenv("JUMPGATE_WATCH_INTERVAL")); err == nil && d > 0 {
		return d
	}
	return DefaultInterval
}

// deliver is where alert changes leave the box (Task 16: the webhook). It
// must never log cfg.Webhook.
var deliver = func(ctx context.Context, cfg Config, prev, next State) {}

// Run polls until ctx ends: the node (when the box has one) and every
// endpoint, then Evaluate, then the state file, then delivery.
func Run(ctx context.Context, cfg Config) error {
	every := interval(cfg)
	prev, err := ReadState(cfg.StatePath)
	if err != nil {
		log.Printf("watch: starting from an empty state: %v", err)
		prev = State{}
	}
	var mon *monitor.Monitor
	chainID := 0
	if cfg.Wire != nil {
		mon = monitor.New(monitor.Config{Exec: executor.NewLocal(), Wire: *cfg.Wire, RefRPC: cfg.RefRPC})
		chainID = cfg.Wire.ChainID
	}
	hc := &http.Client{Timeout: 5 * time.Second}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		in := Inputs{ChainID: chainID, EndpointList: cfg.Endpoints, Endpoints: map[string]error{}}
		if mon != nil {
			s := mon.PollOnce(ctx)
			in.Snap = &s
		}
		for _, e := range cfg.Endpoints {
			in.Endpoints[e.Name] = probe(ctx, hc, e)
		}
		if ctx.Err() != nil {
			return nil
		}
		next := Evaluate(prev, in, time.Now())
		if err := WriteState(cfg.StatePath, next); err != nil {
			log.Printf("watch: writing state: %v", err)
		}
		deliver(ctx, cfg, prev, next)
		prev = next
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// probe asks e whether it is answering. Errors name the endpoint, not
// what the transport says about its URL.
func probe(ctx context.Context, hc *http.Client, e Endpoint) error {
	var req *http.Request
	var err error
	if e.Kind == "beacon" {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.URL, "/")+"/eth/v1/node/health", nil)
	} else {
		body := []byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`)
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, e.URL, bytes.NewReader(body))
		if req != nil {
			req.Header.Set("Content-Type", "application/json")
		}
	}
	if err != nil {
		return err
	}
	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("%s did not answer", e.Name)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if e.Kind == "beacon" {
		if resp.StatusCode != 200 && resp.StatusCode != 206 {
			return fmt.Errorf("%s health is %d", e.Name, resp.StatusCode)
		}
		return nil
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s answered %d", e.Name, resp.StatusCode)
	}
	var r struct {
		Result string          `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if json.Unmarshal(b, &r) != nil || r.Result == "" || len(r.Error) > 0 {
		return fmt.Errorf("%s gave no block number", e.Name)
	}
	return nil
}
