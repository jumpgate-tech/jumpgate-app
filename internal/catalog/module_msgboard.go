package catalog

// msgboard is the PulseChain message board served by valve-tech/reth: a
// node module that adds the msgboard namespace to reth's --http.api on 369
// and 943. Local by default; public exposure is a per-node choice made at
// the gateway, and msgboard_content (unbounded) is never public.
func init() {
	modules = append(modules, Module{
		ID:        "msgboard",
		Name:      "Message board",
		Summary:   "Serves the PulseChain message board from your node.",
		SiteURL:   "https://msgboard.xyz",
		Placement: PlaceNode,

		Chains:     []int{369, 943},
		Clients:    []string{"reth"},
		Namespaces: []string{"msgboard"},
		Methods: []ModuleMethod{
			{Name: "msgboard_addMessage", Class: MethodWrite},
			{Name: "msgboard_status", Class: MethodRead},
			{Name: "msgboard_categories", Class: MethodRead},
			{Name: "msgboard_contentPage", Class: MethodRead},
			{Name: "msgboard_getMessage", Class: MethodRead},
			{Name: "msgboard_content", Class: MethodUnbounded},
			{Name: "msgboard_subscribe", Class: MethodStream},
			{Name: "msgboard_unsubscribe", Class: MethodStream},
		},

		Probe:     ModuleProbe{HelpContains: "--msgboard.block-range", StatusMethod: "msgboard_status"},
		NeedsSync: true,
		// The 80 MB board plus one 167 MB content build, rounded.
		Resources: Resources{RAMBytes: 256 << 20, DiskBytes: 128 << 20},

		ClientReason: func(client string) string {
			return "The message board needs reth (valve fork). This node runs " + client + "."
		},
		PollHint: "poll msgboard_contentPage",
		DocsStep: "msgboard-enable",
	})
}
