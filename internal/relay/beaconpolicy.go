package relay

import (
	"errors"
	"net/http"
	"strings"
)

// beaconMethod is the name a beacon call is priced under. The price book has no
// row for it by default, so it bills at the operator's "*" price until they set
// one; every beacon call costs the same, whatever its path.
const beaconMethod = "beacon_api"

var (
	errBeaconPath   = errors.New("relay: beacon path not served")
	errBeaconMethod = errors.New("relay: beacon method not allowed")
)

// beaconDeniedSections are /eth/vN/<section> trees the relay never serves to a
// customer, even read-only:
//
//   - debug: full state dumps, hundreds of megabytes for one charge.
//   - events: a server-sent event stream, which one per-request charge cannot
//     meter.
//   - validator: duties and block production belong to the operator's own
//     validator client, not to a customer key.
var beaconDeniedSections = map[string]bool{
	"debug":     true,
	"events":    true,
	"validator": true,
}

// checkBeaconRequest narrows the beacon proxy to the standard read API.
//
// The beacon client exposes more than the spec: client-specific admin trees
// (/lighthouse/...) and write endpoints that publish blocks or exits. A customer
// key is for reading the chain, so everything else is refused before any charge
// and before the node sees it.
func checkBeaconRequest(method, rest string) error {
	segs := strings.Split(strings.TrimPrefix(rest, "/"), "/")
	for _, s := range segs {
		// An empty, "." or ".." segment is never a real resource. Refusing it
		// here means the node's own path normalisation never decides where a
		// request lands.
		if s == "" || s == "." || s == ".." {
			return errBeaconPath
		}
	}
	if len(segs) < 3 || segs[0] != "eth" || (segs[1] != "v1" && segs[1] != "v2") {
		return errBeaconPath
	}
	if beaconDeniedSections[segs[2]] {
		return errBeaconPath
	}

	switch method {
	case http.MethodGet, http.MethodHead:
		return nil
	case http.MethodPost:
		// The spec offers POST variants of two state queries so a caller can ask
		// about many validators at once. They read; nothing else POSTed does.
		if isBeaconStateQuery(segs) {
			return nil
		}
	}
	return errBeaconMethod
}

// isBeaconStateQuery matches /eth/v1/beacon/states/{id}/validators and
// .../validator_balances.
func isBeaconStateQuery(segs []string) bool {
	return len(segs) == 6 && segs[1] == "v1" && segs[2] == "beacon" && segs[3] == "states" &&
		(segs[5] == "validators" || segs[5] == "validator_balances")
}

func writeBeaconPolicyError(w http.ResponseWriter, err error) {
	if errors.Is(err, errBeaconMethod) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed on this beacon path")
		return
	}
	writeError(w, http.StatusNotFound, "beacon path not served")
}
