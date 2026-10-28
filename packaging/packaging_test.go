// Package packaging holds tests for the release packaging files; it has no
// code of its own.
package packaging

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// section returns the YAML block that starts at the line equal to header
// and runs until the next line at the same or a shallower indent.
func section(t *testing.T, y, header string) string {
	t.Helper()
	lines := strings.Split(y, "\n")
	indent := len(header) - len(strings.TrimLeft(header, " "))
	for i, l := range lines {
		if l != header {
			continue
		}
		j := i + 1
		for ; j < len(lines); j++ {
			s := strings.TrimSpace(lines[j])
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			if len(lines[j])-len(strings.TrimLeft(lines[j], " ")) <= indent {
				break
			}
		}
		return strings.Join(lines[i:j], "\n")
	}
	t.Fatalf("no %q block", header)
	return ""
}

func TestInstallScriptIsAReleaseAsset(t *testing.T) {
	y := read(t, "../.goreleaser.yaml")
	for _, block := range []string{"checksum:", "release:"} {
		if s := section(t, y, block); !strings.Contains(s, "./packaging/install/install.sh") {
			t.Errorf(".goreleaser.yaml %s does not list ./packaging/install/install.sh in extra_files", block)
		}
	}
}

func TestPackagesJobIsGatedAndScoped(t *testing.T) {
	job := section(t, read(t, "../.github/workflows/release.yml"), "  packages:")
	for _, want := range []string{
		"needs: [provenance]",
		"environment: packages",
		"vars.PKG_REPO_ENABLED == 'true'",
		"startsWith(github.ref, 'refs/tags/v')",
		"contents: read",
		"4d950372162cef595726db63c21035ab.r2.cloudflarestorage.com",
		"scripts/build-pkg-repo.sh",
		"if: always()",
		`rm -rf "$GNUPGHOME"`,
	} {
		if !strings.Contains(job, want) {
			t.Errorf("packages job lacks %q", want)
		}
	}
	// The signing key reaches gpg only through stdin, never as an argument
	// (which ps and /proc show): it appears as an env mapping or a here-string.
	for _, l := range strings.Split(job, "\n") {
		if !strings.Contains(l, "PKG_SIGNING_KEY") {
			continue
		}
		s := strings.TrimSpace(l)
		if s == "PKG_SIGNING_KEY: ${{ secrets.PKG_SIGNING_KEY }}" || strings.Contains(s, `<<<"$PKG_SIGNING_KEY"`) {
			continue
		}
		t.Errorf("PKG_SIGNING_KEY used outside an env mapping or stdin: %q", s)
	}
	// Packages go up before the indexes that name them.
	before := func(a, b string) {
		ia, ib := strings.Index(job, a), strings.Index(job, b)
		if ia < 0 || ib < 0 || ia > ib {
			t.Errorf("packages job must sync %q before %q", a, b)
		}
	}
	before("repo/apt/pool/", "repo/apt/dists/")
	before("--exclude 'repodata/*'", "repo/rpm/repodata/")
	before("repo/apt/dists/", "repo/jumpgate.asc")
	before("repo/rpm/repodata/", "repo/jumpgate.repo")
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNFPMShipsTheServiceFiles(t *testing.T) {
	y := read(t, "../.goreleaser.yaml")
	for _, want := range []string{
		"nfpms:",
		"formats: [deb, rpm]",
		"dst: /usr/lib/systemd/system/jumpgate.service",
		"dst: /usr/lib/sysusers.d/jumpgate.conf",
		"postinstall: packaging/linux/server/postinst.sh",
	} {
		if !strings.Contains(y, want) {
			t.Errorf(".goreleaser.yaml lacks %q", want)
		}
	}
	// The box's agent binary is never the package's to own or replace.
	if strings.Contains(y, "/usr/local/lib/jumpgate") {
		t.Error(".goreleaser.yaml ships into /usr/local/lib/jumpgate, the agent's path")
	}
}

func TestPostinstNeverEnablesOrRestartsAManualService(t *testing.T) {
	const path = "linux/server/postinst.sh"
	s := read(t, path)
	for i, line := range strings.Split(s, "\n") {
		code, _, _ := strings.Cut(line, "#")
		if strings.Contains(code, "enable") {
			t.Errorf("line %d enables something: %q", i+1, line)
		}
		if strings.Contains(code, "restart") {
			if strings.Contains(strings.ReplaceAll(code, "try-restart", ""), "restart") {
				t.Errorf("line %d restarts without try-restart: %q", i+1, line)
			}
			if !strings.Contains(code, "50-vault-seal.conf") {
				t.Errorf("line %d try-restarts a service that is not sealed: %q", i+1, line)
			}
		}
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh to check the syntax with")
	}
	if out, err := exec.Command("sh", "-n", path).CombinedOutput(); err != nil {
		t.Errorf("sh -n %s: %v\n%s", path, err, out)
	}
}
