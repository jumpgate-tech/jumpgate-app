package relay

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// beaconProbeTimeout bounds one health probe. A node that accepts a connection
// and never answers must not stall the probe loop for every other chain.
const beaconProbeTimeout = 3 * time.Second

// beaconHealthPath is the beacon API's own health route. It answers 200 healthy,
// 206 syncing, and 503 down — the spec's own three states, so the pool reads
// them rather than inventing a probe of its own.
const beaconHealthPath = "/eth/v1/node/health"

// BeaconHealth is one chain's pool status, reported into the health rollup so it
// can say WHY a chain is degraded rather than only that it is.
type BeaconHealth struct {
	Total   int  `json:"total"`
	Usable  int  `json:"usable"`
	Syncing int  `json:"syncing"`
	OK      bool `json:"ok"`
}

// beaconNodeState is one upstream and what the last probe found.
type beaconNodeState struct {
	url *url.URL
	// usable means the node answers 200 or 206. A syncing node is degraded, not
	// dead: dropping every syncing node would empty the pool exactly when a
	// chain is catching up and the operator most needs it to answer.
	usable bool
	// syncing records the 206 case so the rollup can distinguish the two.
	syncing bool
}

// BeaconPool is one chain's beacon upstreams, round-robined and health-checked.
//
// A chain's beacon route is a POOL rather than one endpoint: every target
// running that chain with a beacon. The pool exists so a customer never lands on
// a node that has already told us it is down.
type BeaconPool struct {
	hc *http.Client

	mu sync.Mutex
	// nodes is per chain, in a stable order so round-robin is even.
	nodes map[int][]*beaconNodeState
	// cursor is the round-robin position per chain.
	cursor map[int]int
	// probed records that a first probe has completed for a chain. Before it,
	// every node is served: a relay that answered 501 for the first probe
	// interval after startup would look broken on every deploy.
	probed map[int]bool

	stopOnce sync.Once
	stopped  chan struct{}
}

// NewBeaconPool builds a pool from the catalog's per-chain beacon endpoints.
func NewBeaconPool(endpoints map[int][]*url.URL) *BeaconPool {
	p := &BeaconPool{
		hc:      &http.Client{Timeout: beaconProbeTimeout},
		nodes:   make(map[int][]*beaconNodeState),
		cursor:  make(map[int]int),
		probed:  make(map[int]bool),
		stopped: make(chan struct{}),
	}
	for chainID, urls := range endpoints {
		for _, u := range urls {
			p.nodes[chainID] = append(p.nodes[chainID], &beaconNodeState{url: u})
		}
	}
	return p
}

// Next returns the upstream to serve, or false when the chain has no usable
// beacon. False becomes a definite 501 rather than a proxy attempt that is
// certain to fail.
func (p *BeaconPool) Next(chainID int) (*url.URL, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nodes := p.nodes[chainID]
	if len(nodes) == 0 {
		return nil, false
	}

	// Walk from the cursor so traffic spreads evenly, and stop at the first
	// node the last probe found usable.
	for i := 0; i < len(nodes); i++ {
		idx := (p.cursor[chainID] + i) % len(nodes)
		node := nodes[idx]
		if node.usable || !p.probed[chainID] {
			p.cursor[chainID] = (idx + 1) % len(nodes)
			return node.url, true
		}
	}
	return nil, false
}

// Probe refreshes every node's health. A node that recovers returns to rotation,
// which is why this repeats rather than running once — a pool that never
// re-probed would shrink to nothing over a long uptime and never grow back.
func (p *BeaconPool) Probe(ctx context.Context) {
	p.mu.Lock()
	type target struct {
		chainID int
		node    *beaconNodeState
	}
	var targets []target
	for chainID, nodes := range p.nodes {
		for _, node := range nodes {
			targets = append(targets, target{chainID, node})
		}
	}
	p.mu.Unlock()

	var wg sync.WaitGroup
	results := make([]struct{ usable, syncing bool }, len(targets))
	for i, t := range targets {
		wg.Add(1)
		go func(i int, u *url.URL) {
			defer wg.Done()
			results[i].usable, results[i].syncing = p.probeOne(ctx, u)
		}(i, t.node.url)
	}
	wg.Wait()

	p.mu.Lock()
	for i, t := range targets {
		t.node.usable = results[i].usable
		t.node.syncing = results[i].syncing
		p.probed[t.chainID] = true
	}
	p.mu.Unlock()
}

// probeOne reads one node's own health endpoint.
func (p *BeaconPool) probeOne(ctx context.Context, base *url.URL) (usable, syncing bool) {
	ctx, cancel := context.WithTimeout(ctx, beaconProbeTimeout)
	defer cancel()

	target := *base
	target.Path = beaconHealthPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false, false
	}
	resp, err := p.hc.Do(req)
	if err != nil {
		return false, false
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, false
	case http.StatusPartialContent:
		return true, true
	default:
		return false, false
	}
}

// Health reports one chain's pool status for the rollup.
func (p *BeaconPool) Health(chainID int) BeaconHealth {
	p.mu.Lock()
	defer p.mu.Unlock()

	var h BeaconHealth
	for _, node := range p.nodes[chainID] {
		h.Total++
		if node.usable {
			h.Usable++
		}
		if node.syncing {
			h.Syncing++
		}
	}
	h.OK = h.Usable > 0
	return h
}

// Run re-probes on an interval until ctx is cancelled.
func (p *BeaconPool) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	p.Probe(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopped:
			return
		case <-ticker.C:
			p.Probe(ctx)
		}
	}
}

// Stop ends the probe loop.
func (p *BeaconPool) Stop() {
	p.stopOnce.Do(func() { close(p.stopped) })
}
