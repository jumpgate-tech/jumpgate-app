package setup

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/executor/argvfake"
)

func TestProbeListeners_LocalDialsInsteadOfAShell(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	f := argvfake.New().Route("127.0.0.1:4000", ln.Addr().String())
	found, err := probeListeners(context.Background(), f, 4000)
	if err != nil || !strings.Contains(found, "127.0.0.1:4000") {
		t.Fatalf("busy port: %q, %v", found, err)
	}
	if found, err := probeListeners(context.Background(), f, 4001); err != nil || found != "" {
		t.Fatalf("free port: %q, %v", found, err)
	}
	if len(f.ShellCalls()) != 0 {
		t.Fatalf("shell used: %q", f.ShellCalls())
	}
}
