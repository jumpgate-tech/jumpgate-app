package api

import "testing"

func TestSSHViewAddress(t *testing.T) {
	v := SSHView{User: "root", Host: "10.0.0.5", Port: 2222, Jump: &SSHView{User: "ops", Host: "bastion"}}
	if got := v.Address(); got != "root@10.0.0.5:2222 via ops@bastion:22" {
		t.Fatalf("Address() = %q", got)
	}
	if got := (SSHView{User: "u", Host: "::1"}).Address(); got != "u@[::1]:22" {
		t.Fatalf("IPv6 Address() = %q", got)
	}
}

func TestLinkPaired(t *testing.T) {
	if !LinkAgent.Paired() || !LinkAgentLocal.Paired() || LinkSSHOnly.Paired() || LinkLocalOnly.Paired() {
		t.Fatal("Paired() is wrong")
	}
}
