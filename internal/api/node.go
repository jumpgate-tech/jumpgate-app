package api

import "time"

// LogHit is one classified journal line, as the logs routes send it. Its JSON
// matches internal/logwatch.Hit, which the server sends unchanged.
type LogHit struct {
	Unit      string    `json:"unit"`
	Line      string    `json:"line"`
	At        time.Time `json:"at"`
	Signature string    `json:"signature"`
	Severity  string    `json:"severity"` // info|warn|error|critical
	Explain   string    `json:"explain"`
	LearnURL  string    `json:"learnUrl,omitempty"`
}

// ServiceResult answers a service action: the unit's state afterwards.
type ServiceResult struct {
	Active bool `json:"Active"`
}

// Endpoints is a node's RPC endpoints and whether they answer. Its JSON names
// are ops.EndpointInfo's, which the web UI reads.
type Endpoints struct {
	ExecHTTP        string `json:"ExecHTTP"`
	BeaconHTTP      string `json:"BeaconHTTP"`
	ExecReachable   bool   `json:"ExecReachable"`
	BeaconReachable bool   `json:"BeaconReachable"`
	ChainIDMatches  bool   `json:"ChainIDMatches"`
	Access          string `json:"Access"` // "local" | "ssh"
	TunnelHint      string `json:"TunnelHint"`
}

// CheckItem is one firewall checklist line (ops.CheckItem's JSON names).
type CheckItem struct {
	ID     string `json:"ID"`
	Title  string `json:"Title"`
	Why    string `json:"Why"`
	Status string `json:"Status"` // pass | fail | warn | unknown
	Detail string `json:"Detail"`
	Fix    string `json:"Fix"`
}

// SyncState is one word for how a client, or the node, is doing.
type SyncState string

const (
	SyncSynced      SyncState = "synced"
	SyncSyncing     SyncState = "syncing"
	SyncStopped     SyncState = "stopped"
	SyncNoData      SyncState = "no_data"     // running, no head yet
	SyncUnavailable SyncState = "unavailable" // no reading: shown as such, never as 0
)

// ClientStatus is one client's reading. Head is the exec block or the beacon
// slot; Distance is the beacon's slots behind (0 for exec).
type ClientStatus struct {
	Active   bool      `json:"active"`
	Syncing  bool      `json:"syncing"`
	Head     uint64    `json:"head"`
	Distance uint64    `json:"distance,omitempty"`
	Peers    int       `json:"peers"`
	State    SyncState `json:"state"`
}

// NodeStatus is a status reading with the server's verdicts. HeadLag is nil
// when the public reference head is unknown; DiskUsedPct is nil when the
// disk probe failed.
type NodeStatus struct {
	At          time.Time    `json:"at"`
	Exec        ClientStatus `json:"exec"`
	Beacon      ClientStatus `json:"beacon"`
	RefHead     uint64       `json:"refHead,omitempty"`
	HeadLag     *uint64      `json:"headLag,omitempty"`
	Overall     SyncState    `json:"overall"`
	DiskUsedPct *float64     `json:"diskUsedPct,omitempty"`
}

// FitVerdict says whether the data fits the disk it is on.
type FitVerdict string

const (
	FitOK      FitVerdict = "ok"      // room for the expected size with FitMargin headroom
	FitTight   FitVerdict = "tight"   // room for the expected size, not the headroom
	FitShort   FitVerdict = "short"   // not even the expected size
	FitUnknown FitVerdict = "unknown" // no estimate for this chain
)

// DiskSample is one disk reading in a target's history.
type DiskSample struct {
	At        time.Time `json:"at"`
	UsedBytes uint64    `json:"usedBytes"`
	FreeBytes uint64    `json:"freeBytes"`
}

// DiskView is a disk reading with the server's verdicts. Every Expected*
// figure, GrowthPerDay and DaysToFull is an estimate; ExpectedLabel is always
// "estimate" so no front end can forget to say so.
type DiskView struct {
	At                  time.Time    `json:"at"`
	ExecBytes           uint64       `json:"execBytes"`
	BeaconBytes         uint64       `json:"beaconBytes"`
	FreeBytes           uint64       `json:"freeBytes"`
	ExpectedExecBytes   uint64       `json:"expectedExecBytes"`
	ExpectedBeaconBytes uint64       `json:"expectedBeaconBytes"`
	ExpectedLabel       string       `json:"expectedLabel"`
	SyncLabel           string       `json:"syncLabel,omitempty"`
	Fit                 FitVerdict   `json:"fit"`
	History             []DiskSample `json:"history,omitempty"`
	GrowthPerDay        *int64       `json:"growthPerDay,omitempty"`
	DaysToFull          *float64     `json:"daysToFull,omitempty"`
}

// UsedBytes is what the two clients hold.
func (d DiskView) UsedBytes() uint64 { return d.ExecBytes + d.BeaconBytes }

// FirewallSummary counts a checklist and grades it: fail if anything fails,
// warn if anything warns, unknown if nothing could be checked, else ok.
type FirewallSummary struct {
	Pass    int    `json:"pass"`
	Warn    int    `json:"warn"`
	Fail    int    `json:"fail"`
	Unknown int    `json:"unknown"`
	Grade   string `json:"grade"`
}
