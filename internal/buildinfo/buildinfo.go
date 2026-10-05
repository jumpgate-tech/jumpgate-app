// Package buildinfo carries the app's identity at build time: the version it
// was built as, and the GitHub repository its releases are published to. The
// release build injects both with linker flags; a plain `go build` gets the
// defaults below.
package buildinfo

import (
	"debug/elf"
	"io"
	"os"
	"runtime"
	"sync"
)

// version is the app version. The default "dev" marks an unversioned local
// build — the update check treats "dev" as "never newer", so a developer
// build is never told to update itself.
//
// The release build overrides it:
//
//	-ldflags "-X github.com/valve-tech/jumpgate/internal/buildinfo.version=v0.4.0"
var version = "dev"

// releaseRepo is the "owner/repo" the update check polls for new releases. It
// is the PUBLIC release repo (jumpgate-app), not this module's own path — the
// two differ because this tree is published under a second identity. It is a
// var, not a const, so a build can repoint it with -ldflags -X if the release
// repo ever moves.
var releaseRepo = "jumpgate-tech/jumpgate-app"

// Version returns the version this binary was built as, e.g. "v0.4.0", or
// "dev" for an unversioned local build.
func Version() string { return version }

// ReleaseRepo returns the "owner/repo" whose GitHub releases the update check
// reads.
func ReleaseRepo() string { return releaseRepo }

// SelfIsStaticLinux reports whether this binary can itself run as the agent
// on a Linux box of the same arch. Only a Linux build without cgo can be
// statically linked. A cgo build (every tray build, and a source build with
// Go's default CGO_ENABLED=1) links the build host's glibc and desktop
// libraries and would fail to start on a headless box, so it must never be
// uploaded as the agent. A build without cgo can still be a PIE or be linked
// externally, so the running executable's own ELF is checked too, once.
func SelfIsStaticLinux() bool {
	selfStaticOnce.Do(func() {
		if runtime.GOOS != "linux" || cgoEnabled {
			return
		}
		exe, err := os.Executable()
		if err != nil {
			return
		}
		f, err := os.Open(exe)
		if err != nil {
			return
		}
		defer f.Close()
		selfStatic = IsStaticELF(f)
	})
	return selfStatic
}

var (
	selfStaticOnce sync.Once
	selfStatic     bool
)

// IsStaticELF reports whether r holds a fixed-address ELF executable with no
// program interpreter: one that starts without a dynamic loader. A PIE
// (ET_DYN) is refused even without PT_INTERP, as only the loader or a
// static-pie start-up relocates it, and the agent is never built as one.
func IsStaticELF(r io.ReaderAt) bool {
	f, err := elf.NewFile(r)
	if err != nil {
		return false
	}
	if f.Type != elf.ET_EXEC {
		return false
	}
	for _, p := range f.Progs {
		if p.Type == elf.PT_INTERP {
			return false
		}
	}
	return true
}
