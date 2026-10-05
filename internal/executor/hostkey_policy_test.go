package executor

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
)

// B1/E2: a dial with no host-key policy is refused before any connection is
// made. Trust-on-first-use is never a silent default; a caller that wants it
// says so with TOFUHostKeyCallback.
func TestDialSSHRefusesANilHostKeyPolicy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			accepted <- struct{}{}
			c.Close()
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	cfg := SSHConfig{Host: "127.0.0.1", Port: port, User: "x", KeyPath: writeTempKey(t), HostKeyFile: filepath.Join(t.TempDir(), "known_hosts")}

	if _, err := DialSSH(context.Background(), cfg); !errors.Is(err, ErrNoHostKeyPolicy) {
		t.Fatalf("DialSSH err = %v, want ErrNoHostKeyPolicy", err)
	}
	if _, err := NewSSH(cfg); !errors.Is(err, ErrNoHostKeyPolicy) {
		t.Fatalf("NewSSH err = %v, want ErrNoHostKeyPolicy", err)
	}
	select {
	case <-accepted:
		t.Fatal("a dial with no host-key policy connected anyway")
	default:
	}
}
