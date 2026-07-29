package server

// Public-endpoint discovery: the HTTP face of internal/chainlist.
//
// This is how "add an endpoint" offers something real for a chain nobody in
// the fleet runs a node for. chainlist reads the canonical
// ethereum-lists/chains feed, drops the entries that are provider slots
// rather than endpoints (${INFURA_API_KEY} and friends — chain 1 alone
// carries two), and then PROBES what is left with eth_chainId, because the
// feed is advertising and roughly a third of what it lists for a popular
// chain is dead or serving a different chain than it claims.
//
// The probing is the entire value: offering an operator the raw feed would be
// offering them a list of upstreams a third of which fail, which is the same
// class of mistake as offering an action that can only fail. So this route
// returns the LIVE ones, and returns the rejected ones too — each with the
// reason — because "the endpoint I expected is missing" is a question the
// screen should be able to answer without a second trip.

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valve-tech/valve-node-app/internal/catalog"
	"github.com/valve-tech/valve-node-app/internal/chainlist"
	"github.com/valve-tech/valve-node-app/internal/config"
)

// chainlistTimeout bounds the whole discovery — one feed fetch plus a bounded
// fan-out of probes. It is generous compared with chainlist's own per-probe
// timeout because a chain with twenty endpoints and a cold feed fetch is a
// legitimate few seconds, and the caller is a human watching a spinner who
// would rather wait than retry.
const chainlistTimeout = 45 * time.Second

// chainlistEndpoint is one candidate as the UI needs it: the URL, whether it
// is usable, and — when it is not — why not, in words written for an
// operator.
type chainlistEndpoint struct {
	// URL is the endpoint in the form the FEED writes it, which for a provider
	// slot means the ${PLACEHOLDER} is still in it — see redactKeys. It is what
	// the browser gets and what the browser posts back to add the endpoint;
	// handleGatewayPutConfig fills the slot in again on the way to storage, so
	// the key itself never leaves this process by this route.
	URL    string `json:"url"`
	Kind   string `json:"kind"`   // "http" | "ws"
	Status string `json:"status"` // live | unprobed | rejected
	// ChainID is what the endpoint actually answered, which is the useful
	// detail when it is rejected: "that one is really chain 1" beats
	// "rejected".
	ChainID int `json:"chainId,omitempty"`
	// LatencyMS is the probe round trip. It is the only ordering signal an
	// operator has when picking between several live endpoints.
	LatencyMS int    `json:"latencyMs,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type chainlistResponse struct {
	ChainID int `json:"chainId"`
	// Source is "feed" or "vendored". A vendored answer is not a failure —
	// it is the built-in list standing in for an unreachable feed — but the
	// operator should know they are looking at a snapshot, so FetchError
	// travels with it rather than being swallowed.
	Source     string              `json:"source"`
	FetchError string              `json:"fetchError,omitempty"`
	Endpoints  []chainlistEndpoint `json:"endpoints"`
	// Live counts the endpoints that proved they serve this chain, so the UI
	// can lead with the number that matters.
	Live int `json:"live"`
}

// handleChainlist discovers and probes the public endpoints for one chain.
func (s *Server) handleChainlist(w http.ResponseWriter, r *http.Request) {
	chainID, err := strconv.Atoi(r.PathValue("chainId"))
	if err != nil || chainID <= 0 {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%q is not a chain id", r.PathValue("chainId")))
		return
	}

	cfg, err := s.loadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	keys := providerKeys(cfg)

	// Bounded by the request context, so a client that navigates away stops
	// the probes rather than leaving them running against public endpoints.
	ctx, cancel := context.WithTimeout(r.Context(), chainlistTimeout)
	defer cancel()

	d := s.newChainlist()
	// The keys are what turns a provider slot from a rejection into a
	// candidate. They are set here rather than in the Discoverer's constructor
	// because they are per-request state read from the stored config, and the
	// constructor is a test seam.
	d.Keys = keys

	res, err := d.Discover(ctx, chainID)
	if err != nil {
		// Only reached when the feed did not yield this chain AND there is no
		// vendored snapshot for it — i.e. genuinely nothing to offer. That
		// covers an unreachable feed and a perfectly healthy feed that simply
		// does not list the chain: chainlist deliberately treats those the
		// same, because from the operator's seat they have the same remedy.
		// 502, because the failure is upstream of this app rather than in the
		// request.
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	out := chainlistResponse{ChainID: res.ChainID, Source: string(res.Source)}
	if res.FetchErr != nil {
		out.FetchError = res.FetchErr.Error()
	}
	for _, ep := range res.Endpoints {
		e := chainlistEndpoint{
			// Both fields go through redactKeys: a probe failure can quote the
			// URL it failed on, so redacting only the URL would leave the key
			// in the sentence next to it.
			URL:     redactKeys(ep.URL, keys),
			Kind:    string(ep.Kind),
			Status:  string(ep.Status),
			ChainID: ep.ChainID,
			Reason:  redactKeys(ep.Reason, keys),
		}
		if ep.Latency > 0 {
			e.LatencyMS = int(ep.Latency / time.Millisecond)
		}
		if ep.Status == chainlist.StatusLive {
			out.Live++
		}
		out.Endpoints = append(out.Endpoints, e)
	}
	writeJSON(w, http.StatusOK, out)
}

// providerKeys is the key set this process will substitute into ${NAME} slots,
// by placeholder name.
//
// VALVE_API_KEY is defaulted to the shared demo key when nothing is stored, so
// a valve slot resolves on a box where the operator has set nothing up at all —
// the same zero-setup guarantee catalog.KnownSet makes. Every other placeholder
// has no default and simply stays unresolved, because there is no such thing as
// a demo Infura account.
func providerKeys(cfg config.Config) map[string]string {
	keys := make(map[string]string, len(cfg.ProviderKeys)+1)
	for name, v := range cfg.ProviderKeys {
		if v = strings.TrimSpace(v); v != "" {
			keys[strings.TrimSpace(name)] = v
		}
	}
	if keys[config.ValveKeyPlaceholder] == "" {
		keys[config.ValveKeyPlaceholder] = catalog.DefaultValveKey
	}
	return keys
}

// redactKeys is the inverse of chainlist.Resolve: it puts the ${PLACEHOLDER}
// back wherever a key was substituted in.
//
// This exists because a resolved provider URL carries the key as a path segment
// — https://mainnet.infura.io/v3/<the key> — and discovery's whole output is
// serialised straight to the browser. Sending the resolved URL would hand the
// operator's key to any script on the page and to anything that logs a
// response, for no gain: the browser does not dial these endpoints, this
// process does. Putting the placeholder back keeps the string USEFUL — it is
// still a stable, resolvable reference to exactly one endpoint, so the client
// can post it straight back to add it — while the secret stays here.
//
// Values are replaced longest-first so that one key which happens to contain
// another cannot leave a fragment behind, and so the result does not depend on
// map order.
func redactKeys(s string, keys map[string]string) string {
	if s == "" || len(keys) == 0 {
		return s
	}
	names := make([]string, 0, len(keys))
	for name, v := range keys {
		if v != "" {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		vi, vj := keys[names[i]], keys[names[j]]
		if len(vi) != len(vj) {
			return len(vi) > len(vj)
		}
		return names[i] < names[j]
	})
	for _, name := range names {
		s = strings.ReplaceAll(s, keys[name], "${"+name+"}")
	}
	return s
}
