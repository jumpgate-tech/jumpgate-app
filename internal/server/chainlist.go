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
	"strconv"
	"time"

	"github.com/valve-tech/valve-node-app/internal/chainlist"
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

	// Bounded by the request context, so a client that navigates away stops
	// the probes rather than leaving them running against public endpoints.
	ctx, cancel := context.WithTimeout(r.Context(), chainlistTimeout)
	defer cancel()

	res, err := chainlist.New().Discover(ctx, chainID)
	if err != nil {
		// Only reached when the feed failed AND there is no vendored snapshot
		// for this chain — i.e. genuinely nothing to offer. 502, because the
		// failure is upstream of this app rather than in the request.
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}

	out := chainlistResponse{ChainID: res.ChainID, Source: string(res.Source)}
	if res.FetchErr != nil {
		out.FetchError = res.FetchErr.Error()
	}
	for _, ep := range res.Endpoints {
		e := chainlistEndpoint{
			URL:     ep.URL,
			Kind:    string(ep.Kind),
			Status:  string(ep.Status),
			ChainID: ep.ChainID,
			Reason:  ep.Reason,
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
