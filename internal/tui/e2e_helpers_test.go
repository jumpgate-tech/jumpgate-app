package tui_test

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
	"testing"
)

// Only "this Windows has no AF_UNIX" skips the end-to-end test; any other
// ServeUnix failure is a failure.
func TestUnixSocketUnsupportedIsNarrow(t *testing.T) {
	for _, err := range []error{
		errors.New("server: listen on /x: address already in use"),
		fmt.Errorf("server: listen: %w", &net.OpError{Op: "listen", Net: "unix", Err: os.NewSyscallError("bind", syscall.EADDRINUSE)}),
		nil,
	} {
		if unixSocketUnsupported(err) {
			t.Errorf("%v counted as unsupported", err)
		}
	}
	afNoSupport := fmt.Errorf("server: listen on x: %w (the local socket needs Windows 10 version 1803)",
		&net.OpError{Op: "listen", Net: "unix", Err: os.NewSyscallError("socket", syscall.Errno(wsaEAFNOSUPPORT))})
	if got, want := unixSocketUnsupported(afNoSupport), runtime.GOOS == "windows"; got != want {
		t.Errorf("WSAEAFNOSUPPORT on %s: %v", runtime.GOOS, got)
	}
}
