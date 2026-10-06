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
