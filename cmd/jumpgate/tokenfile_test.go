package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func tokenEnv(name, file string) func(string) string {
	return func(k string) string {
		if k == name+"_FILE" {
			return file
		}
		return ""
	}
}

// On every platform: a token file only this user can read is accepted, and
// the same file once another user is granted read access (mode bits on unix,
// an Everyone ACE on Windows) is refused with the file named.
func TestTokenFilePrivacyIsCheckedOnEveryPlatform(t *testing.T) {
	f := filepath.Join(t.TempDir(), "relay")
	if err := fsperm.WriteFilePrivate(f, []byte("tok\n")); err != nil {
		t.Fatal(err)
	}
	got, err := readToken(tokenEnv("JUMPGATE_RELAY_TOKEN", f), "JUMPGATE_RELAY_TOKEN")
	if err != nil || got != "tok" {
		t.Fatalf("private token file: readToken = %q, %v", got, err)
	}

	testutil.Loosen(t, f)
	_, err = readToken(tokenEnv("JUMPGATE_RELAY_TOKEN", f), "JUMPGATE_RELAY_TOKEN")
	if err == nil {
		t.Fatal("a token file other users can read was accepted")
	}
	if !strings.Contains(err.Error(), f) {
		t.Fatalf("error %q does not name the file", err)
	}
}

// A directory is never a token file.
func TestTokenFileThatIsADirectoryIsRefusedOnEveryPlatform(t *testing.T) {
	d := filepath.Join(t.TempDir(), "dir")
	if err := fsperm.MkdirPrivate(d); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(tokenEnv("JUMPGATE_ADMIN_TOKEN", d), "JUMPGATE_ADMIN_TOKEN"); err == nil {
		t.Fatal("a directory was accepted as a token file")
	}
}
