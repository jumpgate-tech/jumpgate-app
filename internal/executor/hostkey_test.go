package executor

// Trust-on-first-use host key verification.
//
// This is the file that decides whether an SSH connection is talking to the
// machine it talked to last time, so the negative cases matter more than the
// positive one: a mismatch that is accepted is exactly the man-in-the-middle
// this exists to catch, and a malformed record that is treated as "no record"
// silently downgrades to trusting whatever answers next.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

// testHostKey generates a distinct public key per call.
func testHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("wrap key: %v", err)
	}
	return key
}

var testAddr = &net.TCPAddr{IP: net.IPv4(10, 0, 0, 9), Port: 22}

// First contact records the key and accepts. That is the "trust" in
// trust-on-first-use, and the file is created 0600 because it is the record
// everything afterwards is checked against.
func TestTOFU_FirstContactRecordsTheKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "known_hosts")
	key := testHostKey(t)

	cb := tofuHostKeyCallback(path)
	if err := cb("10.0.0.9:22", testAddr, key); err != nil {
		t.Fatalf("first contact was refused: %v", err)
	}

	got, err := lookupHostKey(path, "10.0.0.9:22")
	if err != nil {
		t.Fatalf("lookupHostKey: %v", err)
	}
	if got == nil {
		t.Fatal("first contact accepted the host without recording its key")
	}
	if string(got.Marshal()) != string(key.Marshal()) {
		t.Error("the recorded key is not the one presented")
	}

	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("mode = %04o, want 0600", perm)
		}
	}
}

// The second connection to the same host with the same key is accepted, and
// nothing is appended — a file that grows one line per connection eventually
// makes the mismatch check ambiguous.
func TestTOFU_AKnownHostIsAcceptedWithoutRerecording(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	key := testHostKey(t)
	cb := tofuHostKeyCallback(path)

	if err := cb("10.0.0.9:22", testAddr, key); err != nil {
		t.Fatalf("first contact: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := cb("10.0.0.9:22", testAddr, key); err != nil {
		t.Fatalf("a known host was refused: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("the record was appended again:\n%s\nthen\n%s", before, after)
	}
}

// The whole point: a DIFFERENT key for a recorded host is refused, and the
// error says both possibilities, because "the host was rebuilt" and "someone
// is in the middle" look identical from here.
func TestTOFU_AChangedKeyIsRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	cb := tofuHostKeyCallback(path)

	if err := cb("10.0.0.9:22", testAddr, testHostKey(t)); err != nil {
		t.Fatalf("first contact: %v", err)
	}

	err := cb("10.0.0.9:22", testAddr, testHostKey(t)) // a different key
	if err == nil {
		t.Fatal("a changed host key was accepted — this is the attack this file exists to catch")
	}
	msg := err.Error()
	for _, want := range []string{"10.0.0.9:22", "man-in-the-middle", "rebuilt", path} {
		if !strings.Contains(msg, want) {
			t.Errorf("the error does not mention %q: %s", want, msg)
		}
	}
}

// Records are per host:port, so two hosts do not share a verdict.
func TestTOFU_HostsAreRecordedIndependently(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	cb := tofuHostKeyCallback(path)

	keyA, keyB := testHostKey(t), testHostKey(t)
	if err := cb("10.0.0.9:22", testAddr, keyA); err != nil {
		t.Fatalf("first host: %v", err)
	}
	if err := cb("10.0.0.10:22", testAddr, keyB); err != nil {
		t.Fatalf("a second host was refused on the first host's record: %v", err)
	}

	// And each still verifies against its own key.
	if err := cb("10.0.0.9:22", testAddr, keyA); err != nil {
		t.Errorf("the first host stopped verifying: %v", err)
	}
	if err := cb("10.0.0.10:22", testAddr, keyA); err == nil {
		t.Error("the second host accepted the first host's key")
	}
}

// A missing file is first contact for every host, not an error.
func TestLookupHostKey_AMissingFileIsNoRecord(t *testing.T) {
	got, err := lookupHostKey(filepath.Join(t.TempDir(), "does-not-exist"), "10.0.0.9:22")
	if err != nil {
		t.Fatalf("a missing known_hosts was an error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v out of a missing file", got)
	}
}

// A malformed entry for the host asked about is an ERROR, not "no record".
// Treating it as absent would re-record whatever answers next, which turns a
// corrupt file into a silent downgrade to trusting a stranger.
func TestLookupHostKey_AMalformedEntryIsRefusedNotIgnored(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{name: "the key is not base64", line: "10.0.0.9:22 ssh-ed25519 !!!not-base64!!!"},
		{name: "the key is base64 but not a key", line: "10.0.0.9:22 ssh-ed25519 " + base64.StdEncoding.EncodeToString([]byte("nonsense"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "known_hosts")
			if err := os.WriteFile(path, []byte(tc.line+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := lookupHostKey(path, "10.0.0.9:22")
			if err == nil {
				t.Fatalf("a corrupt record read as %v instead of failing", got)
			}
			if !strings.Contains(err.Error(), "malformed") {
				t.Errorf("the error does not say the record is malformed: %v", err)
			}

			// And the callback refuses the connection rather than
			// re-recording over the corrupt line.
			if err := tofuHostKeyCallback(path)("10.0.0.9:22", testAddr, testHostKey(t)); err == nil {
				t.Error("a corrupt record was overwritten by trusting whatever answered")
			}
		})
	}
}

// Blank lines and entries for other hosts are skipped, and a line with the
// wrong number of fields is not mistaken for a record.
func TestLookupHostKey_SkipsNoiseAndOtherHosts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	key := testHostKey(t)
	want := "10.0.0.9:22 " + key.Type() + " " + base64.StdEncoding.EncodeToString(key.Marshal())

	body := strings.Join([]string{
		"",
		"   ",
		"10.0.0.99:22 ssh-ed25519 " + base64.StdEncoding.EncodeToString(testHostKey(t).Marshal()),
		"10.0.0.9:22 two-fields-only",
		want,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := lookupHostKey(path, "10.0.0.9:22")
	if err != nil {
		t.Fatalf("lookupHostKey: %v", err)
	}
	if got == nil {
		t.Fatal("the valid record was not found past the noise")
	}
	if string(got.Marshal()) != string(key.Marshal()) {
		t.Error("the wrong host's key was returned")
	}
}

// An unreadable file is an error rather than first contact, for the same
// reason a malformed entry is.
func TestLookupHostKey_AnUnreadableFileIsAnError(t *testing.T) {
	dir := t.TempDir()
	// A directory where the file should be: reading it fails with something
	// other than ErrNotExist.
	path := filepath.Join(dir, "known_hosts")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}

	if _, err := lookupHostKey(path, "10.0.0.9:22"); err == nil {
		t.Fatal("an unreadable known_hosts read as first contact")
	}
}

func TestAppendHostKey_ReportsAPathItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := appendHostKey(filepath.Join(blocker, "sub", "known_hosts"), "10.0.0.9:22", testHostKey(t))
	if err == nil {
		t.Fatal("recording a host key under a regular file reported success")
	}
}
