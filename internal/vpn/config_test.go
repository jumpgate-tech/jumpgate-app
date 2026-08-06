package vpn

import (
	"reflect"
	"testing"
)

// A ProtonVPN-style config: full-tunnel single peer, /32 interface address.
const protonConf = `[Interface]
# Bouncing = 3
PrivateKey = QFhTdU5rZXlQcml2YXRlS2V5UHJpdmF0ZUtleVByaXY=
Address = 10.2.0.2/32
DNS = 10.2.0.1

[Peer]
PublicKey = eFRJQkE1cHVibGljS2V5cHVibGljS2V5cHVibGljS2U=
AllowedIPs = 0.0.0.0/0
Endpoint = 203.0.113.7:51820
`

func TestParseConfig_Proton(t *testing.T) {
	cfg, err := ParseConfig(protonConf)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Interface.PrivateKey != "QFhTdU5rZXlQcml2YXRlS2V5UHJpdmF0ZUtleVByaXY=" {
		t.Errorf("PrivateKey = %q", cfg.Interface.PrivateKey)
	}
	if !reflect.DeepEqual(cfg.Interface.Address, []string{"10.2.0.2/32"}) {
		t.Errorf("Address = %v", cfg.Interface.Address)
	}
	if !reflect.DeepEqual(cfg.Interface.DNS, []string{"10.2.0.1"}) {
		t.Errorf("DNS = %v", cfg.Interface.DNS)
	}
	if len(cfg.Peers) != 1 {
		t.Fatalf("peers = %d, want 1", len(cfg.Peers))
	}
	p := cfg.Peers[0]
	if p.Endpoint != "203.0.113.7:51820" {
		t.Errorf("Endpoint = %q", p.Endpoint)
	}
	if !reflect.DeepEqual(p.AllowedIPs, []string{"0.0.0.0/0"}) {
		t.Errorf("AllowedIPs = %v", p.AllowedIPs)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

// A self-hosted mesh style: bare interface IP (no mask), comma lists, keepalive,
// two peers — proves BYO configs beyond the vendor shape parse too.
const meshConf = `[Interface]
PrivateKey = bWVzaFByaXZhdGVLZXltZXNoUHJpdmF0ZUtleW1lc2hQcml2
Address = 10.9.0.5, fd00::5
ListenPort = 51820
MTU = 1420

[Peer]
PublicKey = cGVlck9uZVB1YmxpY0tleXBlZXJPbmVQdWJsaWNLZXlwZWU=
Endpoint = hub.example.internal:51820
AllowedIPs = 10.9.0.0/24, fd00::/64
PersistentKeepalive = 25

[Peer]  ; a second mesh peer
PublicKey = cGVlclR3b1B1YmxpY0tleXBlZXJUd29QdWJsaWNLZXlwZWU=
AllowedIPs = 10.9.0.7/32
`

func TestParseConfig_Mesh(t *testing.T) {
	cfg, err := ParseConfig(meshConf)
	if err != nil {
		t.Fatalf("ParseConfig: %v", err)
	}
	if cfg.Interface.MTU != 1420 || cfg.Interface.ListenPort != 51820 {
		t.Errorf("MTU/ListenPort = %d/%d", cfg.Interface.MTU, cfg.Interface.ListenPort)
	}
	if !reflect.DeepEqual(cfg.Interface.Address, []string{"10.9.0.5", "fd00::5"}) {
		t.Errorf("Address = %v", cfg.Interface.Address)
	}
	if len(cfg.Peers) != 2 {
		t.Fatalf("peers = %d, want 2", len(cfg.Peers))
	}
	if cfg.Peers[0].PersistentKeepalive != 25 {
		t.Errorf("keepalive = %d", cfg.Peers[0].PersistentKeepalive)
	}
	if !reflect.DeepEqual(cfg.Peers[0].AllowedIPs, []string{"10.9.0.0/24", "fd00::/64"}) {
		t.Errorf("AllowedIPs = %v", cfg.Peers[0].AllowedIPs)
	}
}

func TestOverlayCIDRs_NormalizesBareIPs(t *testing.T) {
	cfg, err := ParseConfig(meshConf)
	if err != nil {
		t.Fatal(err)
	}
	got := cfg.OverlayCIDRs()
	want := []string{"10.9.0.5/32", "fd00::5/128"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("OverlayCIDRs = %v, want %v", got, want)
	}
}

func TestValidate_RejectsIncomplete(t *testing.T) {
	cases := map[string]string{
		"no private key": "[Interface]\nAddress = 10.0.0.1/32\n[Peer]\nPublicKey = k\nEndpoint = x:1\n",
		"no address":     "[Interface]\nPrivateKey = k\n[Peer]\nPublicKey = k\nEndpoint = x:1\n",
		"no peer":        "[Interface]\nPrivateKey = k\nAddress = 10.0.0.1/32\n",
		"peer no pubkey": "[Interface]\nPrivateKey = k\nAddress = 10.0.0.1/32\n[Peer]\nEndpoint = x:1\n",
	}
	for name, conf := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, err := ParseConfig(conf)
			if err != nil {
				t.Fatalf("ParseConfig: %v", err)
			}
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate accepted an incomplete config")
			}
		})
	}
}

func TestRenderRoundTrip(t *testing.T) {
	cfg, err := ParseConfig(protonConf)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseConfig(cfg.Render())
	if err != nil {
		t.Fatalf("re-parse rendered: %v", err)
	}
	if !reflect.DeepEqual(cfg, again) {
		t.Errorf("round-trip changed the config:\n first: %+v\n again: %+v", cfg, again)
	}
}

func TestParseConfig_KeyBeforeSectionIsError(t *testing.T) {
	if _, err := ParseConfig("PrivateKey = k\n[Interface]\n"); err == nil {
		t.Errorf("expected error for a key before any section header")
	}
}
