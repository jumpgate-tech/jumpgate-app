//go:build trustdistro

package setup

import (
	"os"
	"testing"
)

// TestPrintLinuxTrustCommand writes the exact command TrustStoreCommand returns
// for Linux to $JUMPGATE_TRUST_CMD_OUT. scripts/test-trust-linux.sh runs it on
// real distros, because the command runs on the box and a string test cannot
// tell whether Debian's or Fedora's tooling accepts it.
func TestPrintLinuxTrustCommand(t *testing.T) {
	out := os.Getenv("JUMPGATE_TRUST_CMD_OUT")
	if out == "" {
		t.Skip("JUMPGATE_TRUST_CMD_OUT is not set")
	}
	i, err := TrustStoreCommand("linux", "/tmp/ca.crt", "edge")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, []byte(i.Command), 0o600); err != nil {
		t.Fatal(err)
	}
}
