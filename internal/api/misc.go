package api

// Explain answers POST /api/targets/{id}/explain. SentExcerpt is exactly what
// went to the provider, after redaction when Redacted is set.
type Explain struct {
	Text        string   `json:"text"`
	SentExcerpt []string `json:"sentExcerpt"`
	Redacted    bool     `json:"redacted"`
}

// Settings is GET /api/settings. Secrets are never returned, only whether
// they are set.
type Settings struct {
	AIProvider          string   `json:"aiProvider"`
	AIKeySet            bool     `json:"aiKeySet"`
	RefRPCBase          string   `json:"refRpcBase"`
	ProviderKeysSet     []string `json:"providerKeysSet"`
	UpdateNotifyEnabled bool     `json:"updateNotifyEnabled"`
	AIDisclosure        string   `json:"aiDisclosure"`
}

// SettingsUpdate is the part of PUT /api/settings the TUI sends; a nil field
// is left unchanged.
type SettingsUpdate struct {
	AIProvider *string `json:"aiProvider,omitempty"`
	AIKey      *string `json:"aiKey,omitempty"`
}

// ControllerView is GET /api/controller: the key this server signs with.
// State is ok, missing (no key made), unopened (made but not loaded; Reason
// says why) or mismatch (loaded key is not the recorded one).
type ControllerView struct {
	Recorded string `json:"recorded"`
	Address  string `json:"address"`
	Store    string `json:"store"`
	State    string `json:"state"`
	Reason   string `json:"reason,omitempty"`
}

// AgentCheck is POST /api/fleet/{id}/check: a signed agent.info round trip.
type AgentCheck struct {
	Agent     string `json:"agent"`
	Version   string `json:"version"`
	ChainID   int    `json:"chainId"`
	SetUp     bool   `json:"setUp"`
	Signers   int    `json:"signers"`
	ElapsedMs int64  `json:"elapsedMs"`
}

// SSHCommand is GET /api/fleet/{id}/ssh: an interactive shell's argv.
type SSHCommand struct {
	Argv    []string `json:"argv"`
	Display string   `json:"display"`
}

// GatewayNetwork is one chain a gateway serves.
type GatewayNetwork struct {
	ChainID  int    `json:"chainId"`
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`
	LocalURL string `json:"localUrl,omitempty"`
}

// GatewaySummary is the read-only part of /api/gateways' gateway view.
type GatewaySummary struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Placement struct {
		TargetID string `json:"targetId"`
	} `json:"placement"`
	Status struct {
		State string `json:"State"`
	} `json:"status"`
	BaseURL      string           `json:"baseUrl"`
	LocalBaseURL string           `json:"localBaseUrl,omitempty"`
	Networks     []GatewayNetwork `json:"networks"`
	Warnings     []string         `json:"warnings,omitempty"`
}
