package buildinfo

import (
	"debug/elf"
	"os"
	"runtime"
	"testing"
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
