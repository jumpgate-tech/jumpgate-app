// Package testutil holds helpers that keep tests portable across macOS, Linux
// and Windows. Only test files import it; it imports internal/fsperm, so
// fsperm's own tests are external (package fsperm_test).
package testutil

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/valve-tech/jumpgate/internal/fsperm"
)

// ShortTempDir returns a fresh directory, removed when the test ends, whose
// path is short enough to hold unix sockets a few levels down. t.TempDir()
// embeds the test name under TMPDIR, which on macOS (/var/folders/…) already
// eats most of sun_path's 104 bytes, so macOS uses /tmp. Linux's TMPDIR is
// /tmp, and Windows' %TEMP% is short enough without the test name.
func ShortTempDir(t testing.TB) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS == "darwin" {
		base = "/tmp"
	}
	d, err := os.MkdirTemp(base, "jg")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(d) })
	return d
}

// Home points both HOME and USERPROFILE at a fresh short directory for the
// rest of the test. os.UserHomeDir reads USERPROFILE on Windows and HOME
// elsewhere; setting only HOME on Windows silently shares one home between
// every test in the package.
func Home(t testing.TB) string {
	t.Helper()
	h := ShortTempDir(t)
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	return h
}

// RequirePOSIXShell skips a test that executes `sh`. The controller never
// shells out locally on Windows (local mode is refused there), so a test that
// needs sh tests code Windows does not run.
func RequirePOSIXShell(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell; local commands never run on a Windows controller")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no sh on PATH")
	}
}

// RequireUnix skips a test of unix-only behaviour: permission bits on
// non-secret files, FIFOs, peer credentials, the agent itself.
func RequireUnix(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("unix-only behaviour")
	}
}

// AssertPrivate fails the test unless only the file's owner can read or
// change path. On unix that is the mode bits. On Windows it checks the DACL
// through fsperm.CheckPrivate.
func AssertPrivate(t testing.TB, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if err := fsperm.CheckPrivate(path); err != nil {
			t.Fatal(err)
		}
		return
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := fi.Mode().Perm()
	if perm&0o077 != 0 || perm&0o600 != 0o600 {
		t.Fatalf("%s has mode %o; want owner read/write and nothing for group or others", path, perm)
	}
}

// ELF returns a minimal 64-bit little-endian ELF image for machine: a header
// of type typ and one program header, PT_INTERP when interp is set (a
// dynamically linked binary) and PT_LOAD otherwise. Tests that decide
// whether a binary is static parse it without anything being built or run.
func ELF(t testing.TB, typ elf.Type, machine elf.Machine, interp bool) []byte {
	t.Helper()
	prog := elf.Prog64{Type: uint32(elf.PT_LOAD)}
	if interp {
		prog.Type = uint32(elf.PT_INTERP)
	}
	hdr := elf.Header64{
		Type: uint16(typ), Machine: uint16(machine), Version: uint32(elf.EV_CURRENT),
		Phoff: 64, Ehsize: 64, Phentsize: 56, Phnum: 1, Shentsize: 64,
	}
	copy(hdr.Ident[:], elf.ELFMAG)
	hdr.Ident[elf.EI_CLASS] = byte(elf.ELFCLASS64)
	hdr.Ident[elf.EI_DATA] = byte(elf.ELFDATA2LSB)
	hdr.Ident[elf.EI_VERSION] = byte(elf.EV_CURRENT)
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, hdr); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, prog); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
