package relay

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Category names the kind of API a request addresses. Arch (evm, svm, btc)
// names the chain family. They are different dimensions: every category carries
// an arch slot, so a future chain family costs a value and not a new route.
//
// Category names and arch names share one namespace, because level three of a
// health path accepts either. So never add an arch named "beacon", and never
// add a category named "evm". The constraint is cheap to hold and silent when
// broken, which is why it is written here beside the two sets.
type Category string

const (
	CategoryRPC    Category = "rpc"
	CategoryBeacon Category = "beacon"
	CategoryHealth Category = "health"
)

// HealthDepth is how many dimensions a health request pins. Health filters the
// category x arch x chain matrix, so the depth decides the answer's shape.
type HealthDepth int

const (
	HealthNone     HealthDepth = iota // not a health route
	HealthLiveness                    // /health — unkeyed, bare up or down
	HealthAll                         // /health/<key>
	HealthSelector                    // /health/<key>/<arch|category>
	HealthCell                        // /health/<key>/<arch>/<chainId>
)

var (
	// ErrNotFound is an unknown category. Nothing here is addressable.
	ErrNotFound = errors.New("relay: no such route")
	// ErrBadShape is a known category addressed wrongly.
	ErrBadShape = errors.New("relay: malformed path")
	// ErrUnsupportedArch is a real arch this build does not serve yet. It is a
	// different fact from a malformed path and must stay a different status:
	// a caller who sends svm has the grammar right and the timing wrong.
	ErrUnsupportedArch = errors.New("relay: arch not served")
)

// knownArches is every arch the grammar recognises. servedArches is the subset
// this build actually proxies. The gap between them is deliberate — the relay
// carries arch as a variable from the first line of code, so widening it later
// changes a value and not a shape.
var (
	knownArches  = map[string]bool{"evm": true, "svm": true, "btc": true}
	servedArches = map[string]bool{"evm": true}
)

// knownCategories maps the first segment to a Category.
var knownCategories = map[string]Category{
	"rpc":    CategoryRPC,
	"beacon": CategoryBeacon,
	"health": CategoryHealth,
}

// Route is a parsed request path.
type Route struct {
	Category Category
	Key      string
	Arch     string
	ChainID  int
	// Rest is the beacon remainder — the REST tree under /eth/... that the
	// beacon client owns. Empty for every other category.
	Rest string
	// Sel is a health request's arch-or-category selector, raw and unclassified.
	Sel string
	// Depth is how many dimensions a health request pins. HealthNone elsewhere.
	Depth HealthDepth
}

// Keyed reports whether the route carries a key to authenticate. Only the bare
// liveness level does not, and nothing downstream may treat it as authenticated.
func (r Route) Keyed() bool {
	return r.Category != CategoryHealth || r.Depth != HealthLiveness
}

// UpstreamPath is the path the relay forwards. It never carries the key — that
// strip is the whole point of the relay, and one test asserts it directly for
// every category.
func (r Route) UpstreamPath(projectID string) string {
	switch r.Category {
	case CategoryRPC:
		return fmt.Sprintf("/%s/%s/%d", projectID, r.Arch, r.ChainID)
	case CategoryBeacon:
		if r.Rest == "" {
			return "/"
		}
		return r.Rest
	default:
		// Health is answered by the relay itself. It has no upstream.
		return ""
	}
}

// ParseRoute parses /<category>/<key>/<arch>/<chainId>[/...].
//
// It splits the RAW path before it unescapes each segment, so an encoded slash
// cannot smuggle an extra segment past the depth check: %2F decodes inside one
// segment and is rejected there rather than silently becoming a separator.
func ParseRoute(path string) (Route, error) {
	raw := splitPath(path)
	if len(raw) == 0 {
		return Route{}, ErrNotFound
	}

	category, ok := knownCategories[raw[0]]
	if !ok {
		return Route{}, ErrNotFound
	}
	if category == CategoryHealth {
		return parseHealth(raw)
	}
	return parseChainRoute(category, raw)
}

// parseChainRoute handles /rpc and /beacon. Both pin all four dimensions; they
// differ only in whether a remainder is legal.
func parseChainRoute(category Category, raw []string) (Route, error) {
	// /rpc is fixed depth: eRPC addresses a chain by the whole path, so an
	// extra segment is not a sub-resource but a malformed request.
	if category == CategoryRPC && len(raw) != 4 {
		return Route{}, fmt.Errorf("%w: rpc needs /rpc/<key>/<arch>/<chainId>", ErrBadShape)
	}
	if len(raw) < 4 {
		return Route{}, fmt.Errorf("%w: need /<category>/<key>/<arch>/<chainId>", ErrBadShape)
	}

	key, err := segment(raw[1], "key")
	if err != nil {
		return Route{}, err
	}
	arch, err := segment(raw[2], "arch")
	if err != nil {
		return Route{}, err
	}
	chainID, err := chainSegment(raw[3])
	if err != nil {
		return Route{}, err
	}
	if err := checkArch(arch); err != nil {
		return Route{}, err
	}

	r := Route{Category: category, Key: key, Arch: arch, ChainID: chainID}
	// The beacon API is a REST tree, so a remainder is the normal case. It is
	// joined from the raw segments, because the beacon client owns whatever
	// encoding the caller sent and the relay must not rewrite it.
	if category == CategoryBeacon && len(raw) > 4 {
		r.Rest = "/" + strings.Join(raw[4:], "/")
	}
	return r, nil
}

// parseHealth handles the variable-depth rollup. Each level pins one more
// dimension of the category x arch x chain matrix.
func parseHealth(raw []string) (Route, error) {
	r := Route{Category: CategoryHealth}

	switch len(raw) {
	case 1:
		r.Depth = HealthLiveness
		return r, nil
	case 2, 3, 4:
		// handled below
	default:
		return Route{}, fmt.Errorf("%w: health pins at most arch and chain", ErrBadShape)
	}

	key, err := segment(raw[1], "key")
	if err != nil {
		return Route{}, err
	}
	r.Key = key
	r.Depth = HealthAll
	if len(raw) == 2 {
		return r, nil
	}

	// Level three accepts an arch OR a category, which is why the two share one
	// namespace. Health is a query, so an arch this build does not serve is a
	// legitimate question with an empty answer rather than a 501.
	sel, err := segment(raw[2], "selector")
	if err != nil {
		return Route{}, err
	}
	if !knownArches[sel] && knownCategories[sel] == "" {
		return Route{}, fmt.Errorf("%w: %q is neither an arch nor a category", ErrBadShape, sel)
	}
	r.Sel = sel
	r.Depth = HealthSelector
	if knownArches[sel] {
		r.Arch = sel
	}
	if len(raw) == 3 {
		return r, nil
	}

	chainID, err := chainSegment(raw[3])
	if err != nil {
		return Route{}, err
	}
	r.ChainID = chainID
	r.Depth = HealthCell
	return r, nil
}

// splitPath trims one trailing slash and splits the raw path. A trailing slash
// is the same route: callers append one by habit, and a 400 there would be a
// support ticket rather than a defence.
func splitPath(path string) []string {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// segment unescapes one path segment and rejects an empty or slash-bearing
// result. The slash check is what stops %2F from becoming a separator.
func segment(rawSeg, name string) (string, error) {
	decoded, err := url.PathUnescape(rawSeg)
	if err != nil {
		return "", fmt.Errorf("%w: %s is not valid percent-encoding", ErrBadShape, name)
	}
	if decoded == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrBadShape, name)
	}
	if strings.Contains(decoded, "/") {
		return "", fmt.Errorf("%w: %s contains an encoded separator", ErrBadShape, name)
	}
	return decoded, nil
}

// chainSegment parses the chain id. Zero is legal — btc addresses chain 0 — but
// a negative id is not a chain.
func chainSegment(rawSeg string) (int, error) {
	decoded, err := segment(rawSeg, "chainId")
	if err != nil {
		return 0, err
	}
	chainID, err := strconv.Atoi(decoded)
	if err != nil {
		return 0, fmt.Errorf("%w: chainId %q is not a number", ErrBadShape, decoded)
	}
	if chainID < 0 {
		return 0, fmt.Errorf("%w: chainId %d is negative", ErrBadShape, chainID)
	}
	return chainID, nil
}

// checkArch separates "never heard of it" from "not yet". The first is a bad
// shape; the second is a caller who has the grammar right and the timing wrong.
func checkArch(arch string) error {
	if !knownArches[arch] {
		return fmt.Errorf("%w: unknown arch %q", ErrBadShape, arch)
	}
	if !servedArches[arch] {
		return fmt.Errorf("%w: %q", ErrUnsupportedArch, arch)
	}
	return nil
}
