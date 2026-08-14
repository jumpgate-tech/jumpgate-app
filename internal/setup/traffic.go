package setup

// Reading a gateway's own request counters off the target.
//
// WHY this is here rather than in internal/metrics: that package is pure —
// bytes in, values out — so it can be tested exhaustively against a captured
// dump with no process and no network anywhere near it. This file is the impure
// half, and it is the half that has to know about executors, curl and the two
// backends. Splitting them that way is the same arrangement ops.ERPCRunArgs and
// ops.DockerRun already use.
//
// WHY curl through the executor rather than a Go http.Client: the gateway may
// be on another machine. The counters bind loopback ON THE TARGET, deliberately
// (see ops.ERPCRunSpec.MetricsPort), so from this process there is nothing to
// dial for an SSH target — the only path to 127.0.0.1 over there is a command
// run over there. gatewayCheck's readiness probe already reaches the gateway
// this way for exactly the same reason, and using one mechanism means a
// cross-machine gateway is not a special case that quietly went untested.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/executor"
	"github.com/valve-tech/valve-node-app/internal/metrics"
)

// ErrMetricsOff is returned when the operator has turned the gateway's
// counters off. It is a typed error rather than an empty reading because those
// are different facts: "nobody has called this gateway" and "this gateway is
// not counting" would otherwise both render as a row of zeroes, and only one of
// them is fixed by a setting.
var ErrMetricsOff = errors.New("this gateway's request counters are turned off, so there is nothing to read")

// trafficScrapeTimeout bounds one scrape. It is short on purpose: this runs
// behind a screen that polls, and a wedged metrics listener must fail fast and
// visibly rather than hold the request open and make the whole screen feel
// broken.
const trafficScrapeTimeout = 5

// ReadGatewayTraffic scrapes one gateway's counters and folds them into
// per-network, per-upstream totals.
//
// g is the OPERATOR's view of the config, not the container's. That matters:
// ops.GatewayContainerConfig rewrites the metrics port to the fixed in-container
// one, and scraping that value would dial a port the host has nothing on. The
// host-side port is the operator's, on both backends — for docker it is the host
// half of the -p mapping, for systemd it is simply where the process bound.
func ReadGatewayTraffic(ctx context.Context, e executor.Executor, g catalog.GatewayConfig) (metrics.Traffic, error) {
	samples, err := ReadGatewaySamples(ctx, e, g)
	if err != nil {
		return metrics.Traffic{}, err
	}
	return metrics.FromSamples(samples, g.ProjectIDOrDefault()), nil
}

// ReadGatewayAnalytics scrapes one gateway and folds the same dump into the
// diagnosis view: client-facing latency per chain, and per-endpoint error,
// lag and selection state.
//
// It is a second fold of ONE scrape, not a second scrape. The dump carries
// every family either view needs, so a caller that wants both — the analytics
// screen wants share alongside latency — reads once with ReadGatewaySamples
// and folds twice, rather than curling the gateway twice a poll for two halves
// of the same reading.
func ReadGatewayAnalytics(ctx context.Context, e executor.Executor, g catalog.GatewayConfig) (metrics.Analytics, error) {
	samples, err := ReadGatewaySamples(ctx, e, g)
	if err != nil {
		return metrics.Analytics{}, err
	}
	return metrics.AnalyticsFromSamples(samples, g.ProjectIDOrDefault()), nil
}

// ReadGatewaySamples performs the scrape itself: one curl on the gateway's own
// machine, parsed into samples and handed back unfolded.
//
// Every failure wording lives here rather than in each caller, which is the
// point of the split: "the gateway publishes its counters on loopback only"
// is the same explanation whichever view the operator was looking at, and two
// copies of it would drift.
func ReadGatewaySamples(ctx context.Context, e executor.Executor, g catalog.GatewayConfig) ([]metrics.Sample, error) {
	if !g.MetricsEnabled() {
		return nil, ErrMetricsOff
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/metrics", g.MetricsHTTP())
	cmd := fmt.Sprintf("curl -s --max-time %d %s", trafficScrapeTimeout, shQuote(url))

	res, err := e.Run(ctx, cmd, nil)
	if err != nil {
		return nil, fmt.Errorf("traffic: scrape %s: %w", url, err)
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("traffic: %s did not answer (curl exit %d): %s — the gateway publishes its counters on loopback only, so this is read on the machine it runs on",
			url, res.ExitCode, strings.TrimSpace(res.Stderr))
	}

	body := strings.TrimSpace(res.Stdout)
	if body == "" {
		// An empty body with a zero exit is what a listener that accepted the
		// connection and served nothing looks like. Saying so beats handing a
		// parser an empty string and reporting "no networks", which reads as a
		// healthy gateway nobody has called.
		return nil, fmt.Errorf("traffic: %s answered with an empty body — something is listening there, but it is not serving Prometheus metrics", url)
	}

	samples, err := metrics.ParseText(strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("traffic: %s did not answer with Prometheus text: %w", url, err)
	}
	return samples, nil
}

// IntentsFor derives what the routing configuration says each upstream of one
// chain SHOULD be carrying.
//
// It reads Local, and nothing else, because that is genuinely all eRPC has —
// but Local is a preference WITHIN a network, not an absolute rank: upstreams
// are filed in the 0.2-scored fallback tier only on a chain that also has a
// local upstream, and on a public-only chain every one of them is rendered at
// full weight. metrics.Shares reproduces both cases from these flags, which is
// why the whole slice goes over rather than a per-upstream verdict. The
// multiplier is emphatically not a target percentage — eRPC picks the
// best-scoring upstream rather than splitting traffic by weight — so
// normalising 1.0 and 0.2 into "83% / 17%" would be inventing a number and then
// drawing a bar against it. metrics.Shares turns these flags into the actual
// intended fractions; the rule lives there, next to the arithmetic.
func IntentsFor(n catalog.GatewayNetwork) []metrics.Intent {
	out := make([]metrics.Intent, 0, len(n.Upstreams))
	for i, u := range n.Upstreams {
		id := u.ID
		if id == "" {
			// The same fallback RenderGatewayConfig uses when an upstream was
			// never named. It has to match: the id in the rendered config is
			// the id eRPC labels its counters with, so a different guess here
			// would silently attribute every request to nothing.
			id = catalog.GeneratedUpstreamID(n.ChainID, u.Endpoint, u.Local, i+1)
		}
		out = append(out, metrics.Intent{Upstream: id, Local: u.Local})
	}
	return out
}
