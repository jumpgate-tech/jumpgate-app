//go:build embedagents

package agentbin

import (
	"bytes"
	"debug/elf"
	"testing"

	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

// The agents a release embeds are Linux ELF files of the right machine, and
// statically linked: no PT_INTERP, so they start on any distro (D1).
func TestEmbeddedAgentsAreStaticLinuxOfTheRightArch(t *testing.T) {
	for arch, machine := range map[string]elf.Machine{"amd64": elf.EM_X86_64, "arm64": elf.EM_AARCH64} {
		testutil.Home(t)
		b, src, err := Load(arch)
		if err != nil || src != SourceEmbedded {
			t.Fatalf("%s: Load = %d bytes, %q, %v; want the embedded agent", arch, len(b), src, err)
		}
		if !buildinfo.IsStaticELF(bytes.NewReader(b)) {
			t.Errorf("%s: not a static executable", arch)
		}
		f, err := elf.NewFile(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("%s: %v", arch, err)
		}
		if f.Machine != machine {
			t.Errorf("%s: machine %v, want %v", arch, f.Machine, machine)
		}
		for _, prog := range f.Progs {
			if prog.Type == elf.PT_INTERP {
				t.Errorf("%s: dynamically linked (has PT_INTERP)", arch)
			}
		}
	}
}
