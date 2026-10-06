package agentbin

import "fmt"

// Reporting wraps Load for bootstrap.Options.AgentBinary and reports which
// agent was picked, so a stale developer override is visible to the person
// pairing (spec D14). A build with embedded agents that was told to use the
// developer ones instead (JUMPGATE_DEV_AGENTS=1, ruling P35) also warns, on
// the report and on stderr. The server's pair handler and the CLI's
// foreground local pairing both use it: the first sends the lines on the
// pairing stream, the second prints them.
func Reporting(report func(line string)) func(arch string) ([]byte, error) {
	return func(arch string) ([]byte, error) {
		b, src, err := Load(arch)
		if err != nil {
			return nil, err
		}
		if src == SourceDevDir && embeddedFS != nil {
			dir, _ := devDir()
			warning := fmt.Sprintf("WARNING: using development agents from %s, not the agents built into this jumpgate", dir)
			report(warning)
			fmt.Fprintln(stderr, warning)
		}
		report(fmt.Sprintf("agent binary for linux/%s: %s", arch, src))
		return b, nil
	}
}
