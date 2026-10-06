package api

// HostKeyState is what a probe found for one hop.
type HostKeyState string

const (
	HostKeyConfirmed   HostKeyState = "confirmed"   // in the confirmed store or OpenSSH known_hosts
	HostKeyUnknown     HostKeyState = "unknown"     // nobody confirmed it; ProbeID lets a person do so
	HostKeyMismatch    HostKeyState = "mismatch"    // contradicts a key on record: a hard stop
	HostKeyUnreachable HostKeyState = "unreachable" // no handshake
)

// HostKeyProbeRequest is POST /api/hostkeys/probe's body.
type HostKeyProbeRequest struct {
	SSH SSHView `json:"ssh"`
}

// HostKeyHop is one host on the path, jump host first.
type HostKeyHop struct {
	HostPort    string       `json:"hostPort"`
	Fingerprint string       `json:"fingerprint,omitempty"` // OpenSSH SHA256 form, as `ssh-keygen -lf` prints it
	KeyType     string       `json:"keyType,omitempty"`
	State       HostKeyState `json:"state"`
	ProbeID     string       `json:"probeId,omitempty"` // only with HostKeyUnknown
	Error       string       `json:"error,omitempty"`
}

// HostKeyProbe is the probe's answer. It stops at the first hop that is not
// confirmed: the next hop is only ever reached through a confirmed one.
type HostKeyProbe struct {
	Hops         []HostKeyHop `json:"hops"`
	AllConfirmed bool         `json:"allConfirmed"`
}

// Pending is the first hop that is not confirmed, or nil.
func (p HostKeyProbe) Pending() *HostKeyHop {
	for i := range p.Hops {
		if p.Hops[i].State != HostKeyConfirmed {
			return &p.Hops[i]
		}
	}
	return nil
}

// HostKeyConfirm is POST /api/hostkeys/confirm's body: the probe and the
// fingerprint the person compared. The server records the key it captured
// for that probe, never a key a client sends, and only if the fingerprint is
// exactly that key's.
type HostKeyConfirm struct {
	ProbeID     string `json:"probeId"`
	Fingerprint string `json:"fingerprint"`
}

// HostKeyStore names where a recorded key lives.
type HostKeyStore string

const (
	HostKeyStoreJumpgate HostKeyStore = "jumpgate" // ~/.jumpgate/confirmed_hosts: jumpgate can forget it
	HostKeyStoreOpenSSH  HostKeyStore = "openssh"  // ~/.ssh/known_hosts: read only, never edited
)

// RecordedHostKey is one key a store holds for a host.
type RecordedHostKey struct {
	Fingerprint string       `json:"fingerprint"` // OpenSSH SHA256 form
	KeyType     string       `json:"keyType"`
	Store       HostKeyStore `json:"store"`
}

// RecordedHostKeys is GET /api/hostkeys?hostPort=…'s answer: every key the
// Strict policy trusts for the host, jumpgate's store first.
type RecordedHostKeys struct {
	HostPort string            `json:"hostPort"`
	Keys     []RecordedHostKey `json:"keys"`
}

// HostKeyForget is POST /api/hostkeys/forget's body: the host and the
// fingerprint of the one confirmed key to remove, as the person typed it.
// Replacing a key that changed is forget, then probe and confirm.
type HostKeyForget struct {
	HostPort    string `json:"hostPort"`
	Fingerprint string `json:"fingerprint"`
}
