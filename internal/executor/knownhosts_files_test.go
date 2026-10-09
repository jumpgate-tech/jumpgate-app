package executor

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// M-10: the system-wide known_hosts counts as confirmed too, when present: its
// content is offered through a private snapshot of the checked file.
func TestOpenSSHKnownHostsIncludesTheSystemFileWhenPresent(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	old := systemKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts = old })

	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("without a system file: %v", got)
	}
	if err := os.WriteFile(sys, []byte("host ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := OpenSSHKnownHosts(home)
	if len(got) != 2 || got[0] != user {
		t.Fatalf("with a system file: %v", got)
	}
	if b, ok := memKnownHostsBytes(got[1]); !ok || string(b) != "host ssh-ed25519 AAAA\n" {
		t.Fatalf("held bytes = %q, %v", b, ok)
	}
}

// An untrusted system file must not vouch for hosts: it is skipped, the user's
// file stays.
func TestOpenSSHKnownHostsSkipsAnUntrustedSystemFile(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	if err := os.WriteFile(sys, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })

	openTrustedKnownHosts = func(string) (*os.File, error) { return nil, errors.New("writable by Everyone") }
	user := filepath.Join(home, ".ssh", "known_hosts")
	if got := OpenSSHKnownHosts(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("untrusted file kept: %v", got)
	}
}

// What is trusted is what was checked: if the path is swapped for another file
// after the check, the snapshot still holds the checked file's content, and
// changing the original later does not change the snapshot.
func TestOpenSSHKnownHostsSnapshotsTheCheckedFile(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	sys := filepath.Join(dir, "ssh_known_hosts")
	if err := os.WriteFile(sys, []byte("checked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })
	openTrustedKnownHosts = func(p string) (*os.File, error) {
		f, err := fsperm.OpenTrustedWritable(p) // the real check, which opens the file
		if err != nil {
			return nil, err
		}
		// An attacker swaps the path after the check.
		other := filepath.Join(dir, "evil")
		if err := os.WriteFile(other, []byte("evil\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = os.Remove(p)
		if err := os.Rename(other, p); err != nil {
			f.Close()
			t.Skipf("cannot swap a file in place here: %v", err) // Windows refuses while the checked file is open, which is the point
		}
		return f, nil
	}
	got := OpenSSHKnownHosts(home)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if b, _ := memKnownHostsBytes(got[1]); string(b) != "checked\n" {
		t.Fatalf("held bytes = %q, want the checked file's content", b)
	}
}

// The real check: a file this user owns passes while only they can write it,
// and fails once everyone can.
func TestCheckKnownHostsFileRefusesAWorldWritableFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "ssh_known_hosts")
	if err := os.WriteFile(f, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := fsperm.CheckTrustedWritable(f); err != nil {
		t.Fatalf("owner-writable file refused: %v", err)
	}
	makeWorldWritable(t, f)
	if err := fsperm.CheckTrustedWritable(f); err == nil {
		t.Fatal("a world-writable file was trusted")
	}
}

// No temp path outlives building a checker and nothing re-reads one by name:
// deleting or planting files in the temp area after the entries are loaded
// changes no decision, and a @revoked line in the system file stays enforced.
func TestSystemKnownHostsSurviveTheTempAreaBeingTamperedWith(t *testing.T) {
	tmp := t.TempDir()
	for _, k := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(k, tmp)
	}
	pub := func() ssh.PublicKey {
		p, _, _ := ed25519.GenerateKey(rand.Reader)
		k, _ := ssh.NewPublicKey(p)
		return k
	}
	revoked, good, stranger := pub(), pub(), pub()
	line := func(marker string, k ssh.PublicKey, host string) string {
		return marker + " " + host + " " + string(ssh.MarshalAuthorizedKey(k))
	}
	sysFile := filepath.Join(t.TempDir(), "ssh_known_hosts")
	content := line("@revoked", revoked, "bad.example") + line("", good, "good.example")
	if err := os.WriteFile(sysFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	oldSys := systemKnownHosts
	systemKnownHosts = sysFile
	t.Cleanup(func() { systemKnownHosts = oldSys })

	files := OpenSSHKnownHosts(t.TempDir())
	cb := Strict(filepath.Join(t.TempDir(), "confirmed"), files...)
	addr := func(h string) (string, net.Addr) { return h, &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 22} }

	check := func(when string) {
		t.Helper()
		if h, a := addr("good.example:22"); cb(h, a, good) != nil {
			t.Errorf("%s: a key in the system file was refused: %v files=%v", when, cb(h, a, good), files)
		}
		if h, a := addr("bad.example:22"); cb(h, a, revoked) == nil {
			t.Errorf("%s: a @revoked key was accepted", when)
		}
		if h, a := addr("good.example:22"); cb(h, a, stranger) == nil {
			t.Errorf("%s: a different key for a known host was accepted", when)
		}
	}
	check("fresh")
	if left, _ := os.ReadDir(tmp); len(left) != 0 {
		t.Fatalf("temp files outlived the call: %v", left)
	}
	// Plant, in the temp area, files named the way a snapshot would be.
	evil := line("", stranger, "good.example")
	for _, name := range []string{"0.known_hosts", "1.known_hosts", "x.known_hosts"} {
		_ = os.WriteFile(filepath.Join(tmp, name), []byte(evil), 0o644)
	}
	_ = os.Mkdir(filepath.Join(tmp, "jumpgate-known-hosts-1"), 0o755)
	check("after planting")
	_ = os.RemoveAll(tmp)
	_ = os.MkdirAll(tmp, 0o755)
	check("after deleting")
	if got := KnownHostKeyAlgorithms(filepath.Join(t.TempDir(), "confirmed"), files...)("good.example:22"); len(got) == 0 {
		t.Error("the algorithms of a system-file key were lost")
	}
}

// The files an ssh child is pointed at are the same set strict checking
// uses: the user's file, and the system file when it passes the same trust
// check, by its path (an ssh child cannot read the in-memory snapshot).
func TestOpenSSHKnownHostsPathsMatchTheStrictSet(t *testing.T) {
	home := t.TempDir()
	sys := filepath.Join(t.TempDir(), "ssh_known_hosts")
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = sys
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })
	user := filepath.Join(home, ".ssh", "known_hosts")

	if got := OpenSSHKnownHostsPaths(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("without a system file: %v", got)
	}
	if err := os.WriteFile(sys, []byte("host ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	openTrustedKnownHosts = fsperm.OpenTrustedWritable
	resolved, err := filepath.EvalSymlinks(sys) // ssh gets the resolved path (a temp dir may itself sit behind a link)
	if err != nil {
		t.Fatal(err)
	}
	if got := OpenSSHKnownHostsPaths(home); !slices.Equal(got, []string{user, resolved}) {
		t.Fatalf("with a trusted system file: %v", got)
	}
	if n := len(OpenSSHKnownHosts(home)); n != 2 {
		t.Fatalf("strict set has %d entries", n)
	}
	openTrustedKnownHosts = func(string) (*os.File, error) { return nil, errors.New("writable by Everyone") }
	if got := OpenSSHKnownHostsPaths(home); !slices.Equal(got, []string{user}) {
		t.Fatalf("untrusted system file kept: %v", got)
	}
	if n := len(OpenSSHKnownHosts(home)); n != 1 {
		t.Fatalf("strict set has %d entries", n)
	}
}

// The path handed to an ssh child is the file the trust check judged: when the
// system file is a symlink, ssh gets the resolved target, not the link.
func TestOpenSSHKnownHostsPathsGivesSSHTheResolvedPath(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	real := filepath.Join(dir, "real_known_hosts")
	if err := os.WriteFile(real, []byte("h ssh-ed25519 AAAA\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "ssh_known_hosts")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("cannot symlink here: %v", err)
	}
	oldSys := systemKnownHosts
	systemKnownHosts = link
	t.Cleanup(func() { systemKnownHosts = oldSys })

	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	got := OpenSSHKnownHostsPaths(home)
	if len(got) != 2 || got[1] != want {
		t.Fatalf("got %q, want the resolved %q", got, want)
	}
}

// The path handed to ssh comes from the file the trust check opened, not from
// a second lookup of the link: retargeting the link after the check passed
// does not change it.
func TestOpenSSHKnownHostsPathsIgnoresARetargetAfterTheCheck(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	checked := filepath.Join(dir, "checked_known_hosts")
	other := filepath.Join(dir, "other_known_hosts")
	for _, p := range []string{checked, other} {
		if err := os.WriteFile(p, []byte("h ssh-ed25519 AAAA\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(dir, "ssh_known_hosts")
	if err := os.Symlink(checked, link); err != nil {
		t.Skipf("cannot symlink here: %v", err)
	}
	oldSys, oldOpen := systemKnownHosts, openTrustedKnownHosts
	systemKnownHosts = link
	t.Cleanup(func() { systemKnownHosts, openTrustedKnownHosts = oldSys, oldOpen })
	openTrustedKnownHosts = func(p string) (*os.File, error) {
		f, err := fsperm.OpenTrustedWritable(p)
		if err != nil {
			return nil, err
		}
		// The check has passed; now point the link somewhere else.
		if err := os.Remove(link); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(other, link); err != nil {
			t.Fatal(err)
		}
		return f, nil
	}

	want, err := filepath.EvalSymlinks(checked)
	if err != nil {
		t.Fatal(err)
	}
	got := OpenSSHKnownHostsPaths(home)
	if len(got) != 2 || got[1] != want {
		t.Fatalf("got %q, want the checked %q", got, want)
	}
}
