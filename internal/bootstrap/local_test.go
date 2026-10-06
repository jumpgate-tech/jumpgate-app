package bootstrap

import (
	"errors"
	"testing"
)

func TestLocalSupportedOnlyOnLinux(t *testing.T) {
	if err := LocalSupported("linux"); err != nil {
		t.Fatalf("linux: %v", err)
	}
	for _, goos := range []string{"darwin", "windows", "freebsd"} {
		if err := LocalSupported(goos); !errors.Is(err, ErrLocalUnsupported) {
			t.Fatalf("%s: %v, want ErrLocalUnsupported", goos, err)
		}
	}
}
