package api

// The module routes' shapes as clients read them (plan Task 8). The server
// builds them from catalog.Module and monitor.ModuleStatus; these mirror the
// JSON so a client needs neither package.

// CatalogModule is one element of /api/catalog's modules.
type CatalogModule struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Summary   string          `json:"summary"`
	SiteURL   string          `json:"siteUrl"`
	DocsStep  string          `json:"docsStep"`
	Placement string          `json:"placement"` // "node" | "box" | "gateway"
	Resources ModuleResources `json:"resources"`
	Applies   []ModuleApplies `json:"applies"`
	Methods   []ModuleMethod  `json:"methods"`
	// Public is the exposure choices a node module offers ("read", "post");
	// empty offers none.
	Public []string `json:"public"`
}

// ModuleResources is a module's rough footprint.
type ModuleResources struct {
	RAMBytes  int64 `json:"ramBytes"`
	DiskBytes int64 `json:"diskBytes"`
}

// ModuleApplies is the server's Applies verdict for one chain and client.
// OK false with no Reason hides the module; a Reason disables it and is
// shown verbatim. Clients never re-derive it.
type ModuleApplies struct {
	ChainID int    `json:"chainId"`
	Client  string `json:"client"`
	OK      bool   `json:"ok"`
	Beta    bool   `json:"beta"`
	Reason  string `json:"reason,omitempty"`
}

// ModuleMethod is one RPC method a module adds.
type ModuleMethod struct {
	Name  string `json:"name"`
	Class string `json:"class"`
}

// ModuleStatus is one module's reading (monitor.ModuleStatus).
type ModuleStatus struct {
	ID        string `json:"id"`
	Instance  string `json:"instance,omitempty"`
	Served    bool   `json:"served"`
	Active    bool   `json:"active"`
	Accepting bool   `json:"accepting"`
	Count     uint64 `json:"count"`
	Size      uint64 `json:"size"`
	HeadBlock uint64 `json:"headBlock"`
	Identity  string `json:"identity,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ModulesView is GET /api/targets/{id}/modules.
type ModulesView struct {
	Modules   []ModuleStatus `json:"modules"`
	GateStale bool           `json:"gateStale"`
	Message   string         `json:"message"`
}

// SetModule is PUT /api/targets/{id}/modules/{module}'s body.
type SetModule struct {
	Enabled bool   `json:"enabled"`
	Public  string `json:"public,omitempty"` // node modules: "" | "read" | "post"
	ChainID int    `json:"chainId,omitempty"`
	Port    int    `json:"port,omitempty"`
	ExtIP   string `json:"extIp,omitempty"`
}

// SetModuleResult is its answer: "started" (follow the setup stream),
// "saved", or "saved-gateways-failed" (sent as a 502; the change was saved).
type SetModuleResult struct {
	Status   string          `json:"status"`
	Gateways []string        `json:"gateways,omitempty"`
	Failed   []GatewayFailed `json:"failed,omitempty"`
}

// GatewayFailed is a gateway that did not re-provision after a change.
type GatewayFailed struct {
	ID    string `json:"id"`
	Error string `json:"error"`
}

// SetupEvent is one event of GET /api/targets/{id}/setup/stream
// (setup.Event). Done with an empty StepID ends the run.
type SetupEvent struct {
	StepID string `json:"stepId"`
	Line   string `json:"line,omitempty"`
	Done   bool   `json:"done,omitempty"`
	Err    string `json:"err,omitempty"`
}
