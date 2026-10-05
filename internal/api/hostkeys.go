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
