package peercred

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

func TestOfReportsThisProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no unix peer credentials on Windows")
	}
	path := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	done := make(chan net.Conn, 1)
	go func() {
		c, err := ln.Accept()
		if err != nil {
			done <- nil
			return
		}
		done <- c
	}()
	d, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	accepted := <-done
	if accepted == nil {
		t.Fatal("accept failed")
	}
	defer accepted.Close()
	c, err := Of(accepted)
	if err != nil {
		t.Fatalf("Of: %v", err)
	}
	if c.UID != os.Getuid() {
		t.Errorf("UID = %d, want %d", c.UID, os.Getuid())
	}
	if !c.InGroup(os.Getgid()) {
		t.Errorf("GIDs %v lack our gid %d", c.GIDs, os.Getgid())
	}
}

func TestOfRefusesANonUnixConn(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	_, err := Of(a)
	if err == nil || !strings.Contains(err.Error(), "not a unix socket") {
		t.Fatalf("err = %v, want not a unix socket", err)
	}
}

func TestInGroup(t *testing.T) {
	c := Cred{GIDs: []int{5, 7}}
	if !c.InGroup(7) {
		t.Error("InGroup(7) = false")
	}
	if c.InGroup(8) {
		t.Error("InGroup(8) = true")
	}
}

func TestGroupIDOfAMissingGroup(t *testing.T) {
	if _, ok := GroupID("jumpgate-no-such-group"); ok {
		t.Fatal("GroupID found a group that does not exist")
	}
}
