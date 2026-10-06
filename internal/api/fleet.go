package api

import "time"

// Stale marks a row's sections whose last good reading is older than three
// of their polling intervals. A stale value is shown greyed with its age.
type Stale struct {
	Status   bool `json:"status"`
	Disk     bool `json:"disk"`
	Firewall bool `json:"firewall"`
}

// JobsSummary counts a box's durable jobs. Always null until sub-project 2.
type JobsSummary struct {
	Running       int `json:"running"`
	NeedsDecision int `json:"needsDecision"`
}

// FleetRow is one box in the fleet. A nil section is "unavailable" (never
// probed, or never answered), which a front end must show as such, not as 0.
// Every value carries its age: Status.At and Disk.At are when they were read,
// FirewallAt when the firewall was, LastSeen when the box last answered and
// CheckedAt when the poller last asked (nil: not polled yet, so "loading").
type FleetRow struct {
	ID          string           `json:"id"`
	Link        Link             `json:"link"`
	ThisMachine bool             `json:"thisMachine"`
	Reachable   bool             `json:"reachable"`
	LastSeen    *time.Time       `json:"lastSeen,omitempty"`
	CheckedAt   *time.Time       `json:"checkedAt,omitempty"`
	ChainID     int              `json:"chainId,omitempty"`
	Network     string           `json:"network,omitempty"`
	Agent       string           `json:"agent,omitempty"`
	Status      *NodeStatus      `json:"status"`
	Disk        *DiskView        `json:"disk"`
	Firewall    *FirewallSummary `json:"firewall"`
	FirewallAt  *time.Time       `json:"firewallAt,omitempty"`
	Jobs        *JobsSummary     `json:"jobs"`
	Error       *Error           `json:"error"`
	Stale       Stale            `json:"stale"`
}

// Fleet is GET /api/fleet and each /api/fleet/stream event.
type Fleet struct {
	At              time.Time  `json:"at"`
	IntervalSeconds int        `json:"intervalSeconds"`
	Rows            []FleetRow `json:"rows"`
}

// Row finds a box by id.
func (f Fleet) Row(id string) (FleetRow, bool) {
	for _, r := range f.Rows {
		if r.ID == id {
			return r, true
		}
	}
	return FleetRow{}, false
}
