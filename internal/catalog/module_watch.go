package catalog

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// The health watch is a chain-agnostic box module: the box's gate binary
// run as `jumpgate agent watch`, reading its own config.json (the full
// BoxModule record, root:jumpgate-node 0640) for its endpoints and webhook.
func init() {
	modules = append(modules, Module{
		ID:        "watch",
		Name:      "Health watch",
		Summary:   "Watches this box's node and your endpoints, and raises an alert when one stops working.",
		Placement: PlaceBox,
		Resources: Resources{RAMBytes: 50 << 20, DiskBytes: 1 << 20},
		Unit:      watchUnit,
	})
}

// gateBinary is where pairing installs the box's gate.
const gateBinary = "/usr/local/lib/jumpgate/jumpgate"

// WatchEndpoint is one endpoint the watch probes.
type WatchEndpoint struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Kind string `json:"kind"` // "rpc" (POST eth_blockNumber) | "beacon" (GET /eth/v1/node/health)
}

var watchNameRe = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// ParseWatchEndpoint parses one BoxModule.Upstreams entry of the watch:
// "name=url" probes a JSON-RPC endpoint, "name=beacon+url" a beacon API.
// The name is [A-Za-z0-9._-]; the url is http or https with a host.
func ParseWatchEndpoint(s string) (WatchEndpoint, error) {
	name, raw, ok := strings.Cut(s, "=")
	if !ok {
		return WatchEndpoint{}, fmt.Errorf("watch endpoint %q is not name=url", s)
	}
	if !watchNameRe.MatchString(name) {
		return WatchEndpoint{}, fmt.Errorf("watch endpoint name %q may use only letters, digits, '.', '_' and '-'", name)
	}
	kind := "rpc"
	if rest, ok := strings.CutPrefix(raw, "beacon+"); ok {
		kind, raw = "beacon", rest
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return WatchEndpoint{}, fmt.Errorf("watch endpoint %s: the URL must be http:// or https:// with a host", name)
	}
	return WatchEndpoint{Name: name, URL: raw, Kind: kind}, nil
}

// WatchEndpoints parses and checks every endpoint of a watch record.
func WatchEndpoints(b BoxModule) ([]WatchEndpoint, error) {
	var out []WatchEndpoint
	seen := map[string]bool{}
	for _, s := range b.Upstreams {
		e, err := ParseWatchEndpoint(s)
		if err != nil {
			return nil, err
		}
		if seen[e.Name] {
			return nil, fmt.Errorf("watch endpoint %s is listed twice", e.Name)
		}
		seen[e.Name] = true
		out = append(out, e)
	}
	return out, nil
}

// ValidateWatchWebhook accepts "" or an https:// URL with a host. The URL
// is never put in an error.
func ValidateWatchWebhook(s string) error {
	if s == "" {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("the watch webhook must be an https:// URL")
	}
	return nil
}

// watchUnit runs the box's gate as the health watch. Its endpoints and
// webhook stay in config.json; the unit names only that file.
//
// docs:covers watch-run
func watchUnit(b BoxModule) (BoxUnit, error) {
	if b.ChainID != 0 {
		return BoxUnit{}, fmt.Errorf("the health watch covers the whole box; it takes no chain")
	}
	if _, err := WatchEndpoints(b); err != nil {
		return BoxUnit{}, err
	}
	if err := ValidateWatchWebhook(b.Webhook); err != nil {
		return BoxUnit{}, err
	}
	cmd := gateBinary + " agent watch --config " + b.ConfigPath()
	u, err := renderServiceUnit("module "+b.Instance(), cmd, CurrentNames, b.DataDir(), false)
	if err != nil {
		return BoxUnit{}, err
	}
	return BoxUnit{Unit: u}, nil
}
