//go:build unix

package vault

import (
	"errors"
	"strings"
	"testing"
)

// TestRootRefusesToWriteAnotherUsersVault replaces the plan's
// TestRootWriteKeepsTheOwner: root never writes in another user's directory;
// it is told to run as that user.
func TestRootRefusesToWriteAnotherUsersVault(t *testing.T) {
	err := writeOwnerProblem("v", 990, 0)
	if !errors.Is(err, ErrUnsafeFile) || !strings.Contains(err.Error(), "uid 990") {
		t.Fatalf("root on uid 990: %v", err)
	}
	if err := writeOwnerProblem("v", 0, 0); err != nil {
		t.Fatalf("root on its own vault: %v", err)
	}
	if err := writeOwnerProblem("v", 501, 501); err != nil {
		t.Fatalf("own vault: %v", err)
	}
}
