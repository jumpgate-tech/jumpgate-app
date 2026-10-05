//go:build !windows

package signer

import (
	"context"
	"strings"
	"testing"
)

// The keychain and 1Password helpers run without the server's tokens.
func TestHelperCommandsDoNotSeeSecretTokens(t *testing.T) {
	t.Setenv("JUMPGATE_ADMIN_TOKEN", "admin-secret-value")
	t.Setenv("OP_SESSION_test", "op-session-kept")
	out, err := runCmd(context.Background(), "", "env")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "admin-secret-value") {
		t.Fatalf("a helper saw a token:\n%s", out)
	}
	if !strings.Contains(out, "OP_SESSION_test=op-session-kept") {
		t.Fatalf("a helper lost the 1Password session:\n%s", out)
	}
}
