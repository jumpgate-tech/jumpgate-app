//go:build !windows

package executor

import (
	"context"
	"strings"
	"testing"
)

// A child the local executor launches never inherits the server's secrets,
// even if one is still in the process environment.
func TestLocalChildDoesNotSeeSecretTokens(t *testing.T) {
	t.Setenv("JUMPGATE_ADMIN_TOKEN", "admin-secret-value")
	t.Setenv("JUMPGATE_RELAY_TOKEN", "relay-secret-value")
	t.Setenv("JUMPGATE_HARMLESS", "visible")
	res, err := NewLocal().Run(context.Background(), "env", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(res.Stdout, "secret-value") {
		t.Fatalf("the child saw a token:\n%s", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "JUMPGATE_HARMLESS=visible") {
		t.Fatalf("the child lost an ordinary variable:\n%s", res.Stdout)
	}
}
