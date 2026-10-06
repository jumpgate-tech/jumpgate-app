package ai

import (
	"reflect"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	enode := "enode://" + strings.Repeat("ab", 64) + "@198.51.100.4:30303"
	in := []string{
		"peer 203.0.113.7:30303 dropped; retry 203.0.113.7",
		"dialing " + enode,
		"Found enr: enr:-Iu4QJcpU1t2f5mFzHzvf4LHg0R",
		"libp2p peer 16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ disconnected",
		"listening on [fe80::1ff:fe23:4567:890a]:9000 and 2001:db8::1",
		"reth::cli started at 12:34:56 with 4 threads",
	}
	out := Redact(in)
	want := []string{
		"peer <ip-1>:30303 dropped; retry <ip-1>",
		"dialing <enode-1>",
		"Found enr: <enr-1>",
		"libp2p peer <peer-1> disconnected",
		"listening on [<ip-2>]:9000 and <ip-3>",
		"reth::cli started at 12:34:56 with 4 threads",
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("line %d:\n got %q\nwant %q", i, out[i], want[i])
		}
	}
	if in[0] != "peer 203.0.113.7:30303 dropped; retry 203.0.113.7" {
		t.Fatal("Redact changed its input")
	}
}

func TestRedactSamples(t *testing.T) {
	nodeID := strings.Repeat("0123456789abcdef", 8)
	jwt := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r"
	cases := []struct{ name, in, want string }{
		{"geth peer", `INFO [10-05|12:01:02.123] Looking for peers peercount=0 tried=12 static=0`,
			`INFO [10-05|12:01:02.123] Looking for peers peercount=0 tried=12 static=0`},
		{"geth rpc listen", `INFO [10-05|12:01:02.123] HTTP server started endpoint=127.0.0.1:8545 prefix= cors=localhost`,
			`INFO [10-05|12:01:02.123] HTTP server started endpoint=<ip-1>:8545 prefix= cors=localhost`},
		{"geth version string", `INFO [10-05|12:01:02.123] Starting Geth on Ethereum mainnet... version=v1.2.3.4`,
			`INFO [10-05|12:01:02.123] Starting Geth on Ethereum mainnet... version=v1.2.3.4`},
		{"geth user agent", `Geth/v1.13.5-stable-916d6a44/linux-amd64/go1.21.4`,
			`Geth/v1.13.5-stable-916d6a44/linux-amd64/go1.21.4`},
		{"geth self enode", `INFO [10-05|12:01:02.123] New local node record seq=1 id=` + nodeID[:16] + ` ip=10.1.2.3 udp=30303 tcp=30303`,
			`INFO [10-05|12:01:02.123] New local node record seq=1 id=` + nodeID[:16] + ` ip=<ip-1> udp=30303 tcp=30303`},
		{"bare node id", `Peer id=` + nodeID + ` addr=192.0.2.10:30303 conn=dyndial`,
			`Peer id=<node-1> addr=<ip-1>:30303 conn=dyndial`},
		{"enode with discport", `enode://` + nodeID + `@203.0.113.9:30303?discport=30301`,
			`<enode-1>`},
		{"reth", `2026-10-05T12:01:02.123456Z  INFO reth::cli: Connected to peer peer_id=` + "0x" + nodeID[:20] + ` remote_addr=203.0.113.5:30303`,
			`2026-10-05T12:01:02.123456Z  INFO reth::cli: Connected to peer peer_id=0x` + nodeID[:20] + ` remote_addr=<ip-1>:30303`},
		{"lighthouse", `Oct 05 12:01:02.123 INFO Libp2p Starting, service: libp2p, peer_id: 16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ, address: /ip4/203.0.113.1/tcp/9000/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ`,
			`Oct 05 12:01:02.123 INFO Libp2p Starting, service: libp2p, peer_id: <peer-1>, address: /ip4/<ip-1>/tcp/9000/p2p/<peer-1>`},
		{"lighthouse ip6 multiaddr", `Listening established, address: /ip6/2001:db8::7/tcp/9000`,
			`Listening established, address: /ip6/<ip-1>/tcp/9000`},
		{"prysm", `time="2026-10-05 12:01:02" level=info msg="Peer summary" activePeers=0 prefix=p2p multiAddr=/ip4/198.51.100.20/tcp/13000/p2p/16Uiu2HAmPLe7Mzm8TsYUubgCAW1aJoeFScxrLj8ppHFivPo97bUZ`,
			`time="2026-10-05 12:01:02" level=info msg="Peer summary" activePeers=0 prefix=p2p multiAddr=/ip4/<ip-1>/tcp/13000/p2p/<peer-1>`},
		{"prysm enr", `level=info msg="Running node with peer id of  ENR: enr:-MK4QAbCdEfGhIjKlMnOpQrStUvWxYz0123456789_abc"`,
			`level=info msg="Running node with peer id of  ENR: <enr-1>"`},
		{"erigon", `[INFO] [10-05|12:01:02.123] [p2p] GoodPeers eth66=3 eth67=2 remote=[::ffff:203.0.113.77]:30303`,
			`[INFO] [10-05|12:01:02.123] [p2p] GoodPeers eth66=3 eth67=2 remote=[<ip-1>]:30303`},
		{"bracketed loopback", `serving on [::1]:9545`, `serving on [<ip-1>]:9545`},
		{"port alone", `listening on :8545 and port 30303`, `listening on :8545 and port 30303`},
		{"timestamps", `12:34:56.789 2026-10-05T12:01:02Z 00:00:00`, `12:34:56.789 2026-10-05T12:01:02Z 00:00:00`},
		{"too many dots", `build 1.2.3.4.5 and 999.1.1.1`, `build 1.2.3.4.5 and 999.1.1.1`},
		{"ip at sentence end", `no route to 203.0.113.7.`, `no route to <ip-1>.`},
		{"ip6 at sentence end", `no route to 2001:db8::1.`, `no route to <ip-1>.`},
		{"block hash kept", `head hash=0x` + strings.Repeat("ab", 32), `head hash=0x` + strings.Repeat("ab", 32)},
		{"bearer", `Authorization: Bearer abcDEF123456789xyz.tok`, `Authorization: Bearer <secret-1>`},
		{"token and password", `GET /?token=0123456789abcdef0123456789abcdef&x=1 password=hunter2`,
			`GET /?token=<secret-1>&x=1 password=<secret-2>`},
		{"cookie", `Cookie: jumpgate_token=0123456789abcdef0123456789abcdef; theme=dark`,
			`Cookie: jumpgate_token=<secret-1>; theme=dark`},
		{"json secret", `{"password":"hunter2","user":"bob"}`, `{"password":"<secret-1>","user":"bob"}`},
		{"api keys", `keys sk-ant-api03-abcdefghijklmnop1234 glpat-abcdefghij1234567890 gsk_abcdefghijklmnop1234`,
			`keys <secret-1> <secret-2> <secret-3>`},
		{"jwt", `got ` + jwt + ` back`, `got <secret-1> back`},
		{"short sk word kept", `the task-force and sk-1 stay`, `the task-force and sk-1 stay`},
		{"empty", ``, ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Redact([]string{tc.in})[0]
			if got != tc.want {
				t.Errorf("\n got %q\nwant %q", got, tc.want)
			}
			if again := Redact([]string{got})[0]; again != got {
				t.Errorf("not idempotent:\n first %q\nsecond %q", got, again)
			}
		})
	}
}

// The same value keeps its placeholder across lines of one call, and
// numbering restarts on the next call.
func TestRedactPlaceholdersAreStablePerCall(t *testing.T) {
	got := Redact([]string{"a 10.0.0.1", "b 10.0.0.2", "c 10.0.0.1"})
	want := []string{"a <ip-1>", "b <ip-2>", "c <ip-1>"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	if got := Redact([]string{"x 10.0.0.2"}); got[0] != "x <ip-1>" {
		t.Fatalf("numbering did not restart: %q", got)
	}
	if got := Redact(nil); len(got) != 0 {
		t.Fatalf("nil in, %q out", got)
	}
}

func FuzzRedact(f *testing.F) {
	for _, s := range []string{
		"", "peer 203.0.113.7:30303", "[::1]:9000", "::", ":::", "1.2.3.4.5", "::ffff:1.2.3.4",
		"enode://" + strings.Repeat("ab", 64) + "@1.2.3.4:1", "enr:-Iu4Q", "token=", "Bearer ",
		"password=<secret-1>", "eyJ.eyJ.", "reth::cli 12:34:56", "/ip4/1.2.3.4/tcp/9000", "<ip-1> <enode-9>",
		"2001:db8::1.", "fe80::1%eth0", "sk-" + strings.Repeat("a", 20), "\xff\xfe 1.1.1.1",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		once := Redact([]string{s, s})
		if once[0] != once[1] {
			t.Fatalf("same input, different output: %q vs %q", once[0], once[1])
		}
		twice := Redact(once)
		if !reflect.DeepEqual(once, twice) {
			t.Fatalf("not idempotent:\n  in %q\nonce %q\ntwice %q", s, once, twice)
		}
	})
}

func TestLocalAndDisclosure(t *testing.T) {
	if !Local("ollama") || Local("gemini") || Local("groq") {
		t.Fatal("Local is wrong")
	}
	if d := Disclosure("gemini"); !strings.Contains(d, "IP addresses") || !strings.Contains(d, "Gemini") {
		t.Fatalf("gemini disclosure %q", d)
	}
	if d := Disclosure("ollama"); !strings.Contains(d, "unredacted") {
		t.Fatalf("ollama disclosure %q", d)
	}
	if Disclosure("") != "" {
		t.Fatal("no provider, no disclosure")
	}
}
