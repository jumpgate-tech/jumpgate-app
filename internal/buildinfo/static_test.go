package buildinfo

import (
	"bytes"
	"debug/elf"
	"os"
	"runtime"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// SelfIsStaticLinux must never claim a dynamically linked binary is static:
// a controller that believed it would upload a binary that cannot start on a
// headless box (B-2). The converse is allowed to be conservative.
func TestSelfIsStaticLinuxIsNeverWrong(t *testing.T) {
	if runtime.GOOS != "linux" {
		if SelfIsStaticLinux() {
			t.Fatal("SelfIsStaticLinux() is true off Linux")
		}
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f, err := elf.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dynamic := false
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			dynamic = true
		}
	}
	if SelfIsStaticLinux() && dynamic {
		t.Fatal("SelfIsStaticLinux() is true for a dynamically linked binary")
	}
	if os.Getenv("CGO_ENABLED") == "0" && !SelfIsStaticLinux() {
		t.Fatal("a CGO_ENABLED=0 Linux build must report SelfIsStaticLinux() = true")
	}
}

// Only a fixed-address executable with no interpreter is static. A PIE
// (ET_DYN) needs the loader to relocate it, and PT_INTERP names a dynamic
// loader the box may not have (B-2).
func TestIsStaticELF(t *testing.T) {
	for _, c := range []struct {
		name   string
		image  []byte
		static bool
	}{
		{"static executable", testutil.ELF(t, elf.ET_EXEC, elf.EM_X86_64, false), true},
		{"dynamically linked", testutil.ELF(t, elf.ET_EXEC, elf.EM_X86_64, true), false},
		{"PIE", testutil.ELF(t, elf.ET_DYN, elf.EM_X86_64, true), false},
		{"static PIE", testutil.ELF(t, elf.ET_DYN, elf.EM_X86_64, false), false},
		{"not ELF", []byte("#!/bin/sh\n"), false},
	} {
		if got := IsStaticELF(bytes.NewReader(c.image)); got != c.static {
			t.Errorf("%s: IsStaticELF = %v, want %v", c.name, got, c.static)
		}
	}
}
