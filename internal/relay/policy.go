package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/valve-tech/jumpgate/internal/catalog"
)

var (
	// ErrBadRequest is a body the relay cannot read. It maps to 400.
	ErrBadRequest = errors.New("relay: malformed JSON-RPC request")
	// ErrMethodDenied is a call the key may not make. It maps to 403.
	ErrMethodDenied = errors.New("relay: method not allowed for this key")
	// ErrOriginDenied is a browser origin the key does not list.
	ErrOriginDenied = errors.New("relay: origin not allowed for this key")
	// ErrIPDenied is a caller address the key does not permit.
	ErrIPDenied = errors.New("relay: address not allowed for this key")
	// ErrNetworkDenied is a chain the key is not scoped to.
	ErrNetworkDenied = errors.New("relay: chain not allowed for this key")
)

// moduleWrites and moduleStreams are the catalog's module method tables,
// behind package vars so tests can stand in a fake module.
var (
	moduleWrites  = catalog.ModuleWriteMethods
	moduleStreams = catalog.ModuleStreamMethods
)

// moduleGrantError is a module write the key does not name. It is still an
// ErrMethodDenied, but its message is shown to the caller verbatim, because
// "method not allowed" alone would not say how to fix it.
type moduleGrantError struct{ method string }

func (e moduleGrantError) Error() string { return ErrMethodDenied.Error() + ": " + e.message() }
func (e moduleGrantError) Unwrap() error { return ErrMethodDenied }
func (e moduleGrantError) message() string {
	return e.method + " needs to be on this key's method allow list"
}

// DeniedMessage returns the caller-facing text for a refusal that carries its
// own, and false for every other refusal, which keeps its fixed body.
func DeniedMessage(err error) (string, bool) {
	var g moduleGrantError
	if errors.As(err, &g) {
		return g.message(), true
	}
	return "", false
}

// deniedBody is the refusal text for a CheckMethods error.
func deniedBody(err error) string {
	if msg, ok := DeniedMessage(err); ok {
		return msg
	}
	return "method not allowed for this key"
}

// traceNamespaces are the expensive namespaces that AllowTrace gates as a pair.
// An operator should not have to enumerate every trace method one at a time to
// keep a cheap key cheap.
var traceNamespaces = []string{"trace", "debug"}

// rpcCall is the only part of a JSON-RPC request the relay reads. It never
// decodes params: the relay is a gate, not an interpreter, and params can be
// large.
type rpcCall struct {
	Method string `json:"method"`
}

// ParseMethods reads the JSON-RPC method names from a request body. It accepts
// a single call or a batch array, because eRPC serves both on the same path.
//
// A body the relay cannot read is refused rather than forwarded. Forwarding an
// unparsed body would mean an upstream sees a call the policy never checked.
func ParseMethods(body []byte) ([]string, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: empty body", ErrBadRequest)
	}

	if trimmed[0] == '[' {
		var batch []rpcCall
		if err := json.Unmarshal(trimmed, &batch); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
		}
		// An empty batch asks nothing. Forwarding it spends an upstream call on
		// a caller who wanted no answer.
		if len(batch) == 0 {
			return nil, fmt.Errorf("%w: empty batch", ErrBadRequest)
		}
		methods := make([]string, 0, len(batch))
		for i, call := range batch {
			if call.Method == "" {
				return nil, fmt.Errorf("%w: batch entry %d has no method", ErrBadRequest, i)
			}
			methods = append(methods, call.Method)
		}
		return methods, nil
	}

	var single rpcCall
	if err := json.Unmarshal(trimmed, &single); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	if single.Method == "" {
		return nil, fmt.Errorf("%w: no method", ErrBadRequest)
	}
	return []string{single.Method}, nil
}

// CheckMethods applies the key's method policy to every call in a request.
//
// One denied entry refuses the whole batch. A partial batch would make
// per-request metering ambiguous in the next slice, for no real gain to the
// caller.
func CheckMethods(rec KeyRecord, methods []string) error {
	for _, method := range methods {
		if err := checkOneMethod(rec, method); err != nil {
			return err
		}
	}
	return nil
}

func checkOneMethod(rec KeyRecord, method string) error {
	// The block list wins over everything. A deny is a deliberate act; an allow
	// may be a broad default a later edit widened by accident.
	if containsExact(rec.MethodBlock, method) {
		return fmt.Errorf("%w: %s is blocked", ErrMethodDenied, method)
	}
	if !rec.AllowTrace && inTraceNamespace(method) {
		return fmt.Errorf("%w: %s needs allow_trace", ErrMethodDenied, method)
	}
	// A module write gossips or changes node state, so the empty allow list's
	// "everything" never covers it: the key has to name it. The lookup folds
	// case, because eRPC folds method case in places (architecture/evm/hooks.go)
	// and a write spelled Msgboard_addMessage must not slip through.
	if isModuleWrite(method) && !containsFold(rec.MethodAllow, method) {
		return moduleGrantError{method: method}
	}
	// An empty allow list means "everything except the block list", so a key
	// with no policy still works. A non-empty one is exclusive.
	if len(rec.MethodAllow) > 0 && !containsExact(rec.MethodAllow, method) {
		return fmt.Errorf("%w: %s is not on the allow list", ErrMethodDenied, method)
	}
	return nil
}

// isModuleWrite reports whether method, in any case, is a module write.
func isModuleWrite(method string) bool {
	for w := range moduleWrites() {
		if strings.EqualFold(w, method) {
			return true
		}
	}
	return false
}

// containsFold is containsExact ignoring case.
func containsFold(list []string, want string) bool {
	for _, item := range list {
		if strings.EqualFold(item, want) {
			return true
		}
	}
	return false
}

// inTraceNamespace reports whether a method belongs to trace or debug. It
// matches on the namespace separator rather than a bare prefix, so a method
// named tracker_getThing is not mistaken for the trace namespace.
func inTraceNamespace(method string) bool {
	for _, ns := range traceNamespaces {
		if strings.HasPrefix(method, ns+"_") {
			return true
		}
	}
	return false
}

// CheckOrigin applies the key's origin policy. This is the control that makes a
// browser-exposed key safe: the key itself ships to every visitor, so the
// origin is what actually bounds who may spend it.
func CheckOrigin(rec KeyRecord, origin string) error {
	if len(rec.Origins) == 0 {
		return nil
	}
	for _, allowed := range rec.Origins {
		if allowed == "*" || allowed == origin {
			return nil
		}
	}
	// A missing Origin header fails a key that sets a policy. A caller that
	// sends no origin cannot be shown to satisfy one.
	return fmt.Errorf("%w: %q", ErrOriginDenied, origin)
}

// CheckIP applies the key's address policy. Deny is evaluated first, so a deny
// entry cannot be undone by a broader allow entry.
func CheckIP(rec KeyRecord, ip string) error {
	if len(rec.IPAllow) == 0 && len(rec.IPDeny) == 0 {
		return nil
	}

	addr := net.ParseIP(ip)
	if addr == nil {
		// A policy exists and the caller's address cannot be read. Refuse
		// rather than wave it through — an unreadable address is exactly what
		// an attacker would arrange if the fallback were "allow".
		return fmt.Errorf("%w: %q is not an address", ErrIPDenied, ip)
	}

	for _, entry := range rec.IPDeny {
		if ipMatches(entry, addr) {
			return fmt.Errorf("%w: %s is denied", ErrIPDenied, ip)
		}
	}
	if len(rec.IPAllow) == 0 {
		return nil
	}
	for _, entry := range rec.IPAllow {
		if ipMatches(entry, addr) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s is not on the allow list", ErrIPDenied, ip)
}

// ipMatches accepts a bare address or a CIDR block.
func ipMatches(entry string, addr net.IP) bool {
	if strings.Contains(entry, "/") {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return false
		}
		return network.Contains(addr)
	}
	return net.ParseIP(entry).Equal(addr)
}

// CheckNetwork applies the key's chain policy. The store keeps chain ids as
// strings in key_constraint rows, so the comparison happens on the string.
func CheckNetwork(rec KeyRecord, chainID int) error {
	if len(rec.Networks) == 0 {
		return nil
	}
	want := strconv.Itoa(chainID)
	if containsExact(rec.Networks, want) {
		return nil
	}
	return fmt.Errorf("%w: chain %d", ErrNetworkDenied, chainID)
}

// containsExact compares case-sensitively, as JSON-RPC defines method names. A
// case-insensitive compare would let ETH_SENDRAWTRANSACTION slip past a block
// list that names the real method.
func containsExact(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
