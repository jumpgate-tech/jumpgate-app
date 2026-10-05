package agentbin

import "fmt"

// Reporting wraps Path for bootstrap.Options.AgentBinary and reports which
// agent was picked, so a stale developer override is visible to the person
// pairing (spec D14). The server's pair handler and the CLI's foreground
// local pairing both use it: the first sends the line on the pairing stream,
// the second prints it.
func Reporting(report func(line string)) func(arch string) (string, error) {
	return func(arch string) (string, error) {
		p, src, err := Path(arch)
		if err == nil {
			report(fmt.Sprintf("agent binary for linux/%s: %s", arch, src))
		}
		return p, err
	}
}
