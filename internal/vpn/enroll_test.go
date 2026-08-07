package vpn

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/valve-tech/valve-node-app/internal/executor"
)

// a valid base64-std WireGuard key (32 bytes) built from a byte pattern, so the
// tests need no external tooling to mint one.
func testKey(fill byte) string {
	b := bytes.Repeat([]byte{fill}, 32)
	return base64.StdEncoding.EncodeToString(b)
}

func TestGenerateKeyFrom_DeterministicAndClamped(t *testing.T) {
	seed := bytes.Repeat([]byte{0x42}, 32)

	k1, err := generateKeyFrom(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("generateKeyFrom: %v", err)
	}
	k2, err := generateKeyFrom(bytes.NewReader(seed))
	if err != nil {
		t.Fatalf("generateKeyFrom (again): %v", err)
	}
	// Same randomness in => same keypair out. This is what makes the whole
	// engine testable.
	if k1 != k2 {
		t.Fatalf("same seed produced different keys:\n%+v\n%+v", k1, k2)
	}

	// Both keys are 44-char base64-std (32 bytes) ending in the single pad char.
	for _, s := range []string{k1.PrivateKey, k1.PublicKey} {
		if len(s) != 44 || !strings.HasSuffix(s, "=") {
			t.Errorf("key %q is not 44-char base64 ending '='", s)
		}
	}

	// The private key must carry Curve25519 clamping: low 3 bits of byte 0
	// clear, top bit of byte 31 clear, bit 6 of byte 31 set.
	priv, err := base64.StdEncoding.DecodeString(k1.PrivateKey)
	if err != nil {
		t.Fatalf("private key is not base64: %v", err)
	}
	if len(priv) != 32 {
		t.Fatalf("private key decoded to %d bytes, want 32", len(priv))
	}
	if priv[0]&7 != 0 {
		t.Errorf("byte 0 low bits not cleared: %08b", priv[0])
	}
	if priv[31]&128 != 0 {
		t.Errorf("byte 31 high bit not cleared: %08b", priv[31])
	}
	if priv[31]&64 == 0 {
		t.Errorf("byte 31 bit 6 not set: %08b", priv[31])
	}
}

func TestGenerateKey_RealRandDiffers(t *testing.T) {
	a, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	b, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if a.PrivateKey == b.PrivateKey || a.PublicKey == b.PublicKey {
		t.Errorf("two real-rand keys collided: %+v vs %+v", a, b)
	}
}

func TestNextPeerIP(t *testing.T) {
	// .1 is the server, .2 and .3 are taken (one with a mask, one without) —
	// the next free host is .4.
	got, err := nextPeerIP("10.9.0.1/24", []string{"10.9.0.2/32", "10.9.0.3"})
	if err != nil {
		t.Fatalf("nextPeerIP: %v", err)
	}
	if got != "10.9.0.4/32" {
		t.Errorf("nextPeerIP = %q, want 10.9.0.4/32", got)
	}

	// The server's own address is skipped even when nothing else is taken:
	// first free is .2, not .1.
	got, err = nextPeerIP("10.9.0.1/24", nil)
	if err != nil {
		t.Fatalf("nextPeerIP (empty taken): %v", err)
	}
	if got != "10.9.0.2/32" {
		t.Errorf("nextPeerIP with no taken = %q, want 10.9.0.2/32 (server .1 skipped)", got)
	}

	// A /30 has host addresses .1 and .2 (.0 network, .3 broadcast). Server is
	// .1, .2 is taken => exhausted.
	if _, err := nextPeerIP("10.0.0.1/30", []string{"10.0.0.2"}); err == nil {
		t.Errorf("expected exhaustion error for a fully-taken /30")
	}
}

func TestAddPeer_HappyPath(t *testing.T) {
	pub := testKey(0x11)
	// dump: interface line + a peer line whose first field is our public key.
	dump := "PRIVKEY\tSRVPUB\t51820\toff\n" +
		pub + "\t(none)\t203.0.113.7:51820\t10.9.0.2/32\t0\t0\t0\t0\n"
	f := newFake().script("wg show", executor.Result{ExitCode: 0, Stdout: dump})

	err := AddPeer(context.Background(), f, AddPeerParams{
		Iface: "jumpgate0", PeerPublicKey: pub, AllowedIP: "10.9.0.2/32",
	})
	if err != nil {
		t.Fatalf("AddPeer: %v", err)
	}
	if !f.called("wg set 'jumpgate0' peer '" + pub + "'") {
		t.Errorf("wg set was not invoked with the quoted iface and peer; calls=%v", f.calls)
	}
	if !f.called("wg-quick save 'jumpgate0'") {
		t.Errorf("wg-quick save was not invoked to persist the peer; calls=%v", f.calls)
	}
}

// The load-bearing test: `wg set` exits 0 but the dump does NOT list the peer.
// AddPeer must FAIL — an exit code is not proof the peer landed.
func TestAddPeer_FailsWhenPeerAbsentDespiteExit0(t *testing.T) {
	pub := testKey(0x22)
	f := newFake().script("wg show", executor.Result{ExitCode: 0, Stdout: wgDumpUp}) // wgDumpUp has "peerpub", not our key

	err := AddPeer(context.Background(), f, AddPeerParams{
		Iface: "jumpgate0", PeerPublicKey: pub, AllowedIP: "10.9.0.2/32",
	})
	if err == nil {
		t.Fatalf("AddPeer reported success even though the dump does not list the peer")
	}
}

func TestAddPeer_RejectsBadKeyAndNon32_WithoutRunning(t *testing.T) {
	// Not valid base64 of a 32-byte key.
	f := newFake()
	if err := AddPeer(context.Background(), f, AddPeerParams{
		Iface: "jumpgate0", PeerPublicKey: "not-a-real-key", AllowedIP: "10.9.0.2/32",
	}); err == nil {
		t.Errorf("expected AddPeer to reject a malformed public key")
	}
	if len(f.calls) != 0 {
		t.Errorf("AddPeer ran %d command(s) before rejecting a bad key; want 0: %v", len(f.calls), f.calls)
	}

	// A wider-than-/32 AllowedIP must be rejected too, again before running.
	f2 := newFake()
	if err := AddPeer(context.Background(), f2, AddPeerParams{
		Iface: "jumpgate0", PeerPublicKey: testKey(0x33), AllowedIP: "10.9.0.0/24",
	}); err == nil {
		t.Errorf("expected AddPeer to reject a non-/32 AllowedIP")
	}
	if len(f2.calls) != 0 {
		t.Errorf("AddPeer ran %d command(s) before rejecting a non-/32; want 0: %v", len(f2.calls), f2.calls)
	}
}

func TestRenderClientConfig(t *testing.T) {
	srvPub := testKey(0x44)
	out, err := RenderClientConfig(ClientConfigParams{
		PrivateKey:          testKey(0x55),
		Address:             []string{"10.9.0.2/32"},
		DNS:                 []string{"1.1.1.1"},
		ServerPublicKey:     srvPub,
		Endpoint:            "vpn.example.com:51820",
		AllowedIPs:          []string{"0.0.0.0/0"},
		PersistentKeepalive: 25,
	})
	if err != nil {
		t.Fatalf("RenderClientConfig: %v", err)
	}
	for _, want := range []string{"[Interface]", "PrivateKey", "[Peer]", srvPub, "vpn.example.com:51820"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered config missing %q:\n%s", want, out)
		}
	}

	// Round-trips: what we render parses and validates as a real client config.
	cfg, err := ParseConfig(out)
	if err != nil {
		t.Fatalf("ParseConfig of rendered output: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("rendered config did not validate: %v", err)
	}
}
