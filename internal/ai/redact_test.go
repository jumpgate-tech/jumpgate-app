package ai

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
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
			`Cookie: <secret-1>`},
		{"json secret", `{"password":"hunter2","user":"bob"}`, `{"password":"<secret-1>","user":"bob"}`},
		{"api keys", `keys sk-ant-api03-abcdefghijklmnop1234 glpat-abcdefghij1234567890 gsk_abcdefghijklmnop1234`,
			`keys <secret-1> <secret-2> <secret-3>`},
		{"jwt", `got ` + jwt + ` back`, `got <secret-1> back`},
		{"short sk word kept", `the task-force and sk-1 stay`, `the task-force and sk-1 stay`},
		{"url userinfo", `curl http://user:pass@host:8545/p`, `curl http://<secret-1>@host:8545/p`},
		{"url userinfo ip host", `dial http://admin:s3cret@10.0.0.5:8545/`, `dial http://<secret-1>@<ip-1>:8545/`},
		{"url without userinfo", `http://host:8545/p and ws://h:1/x`, `http://host:8545/p and ws://h:1/x`},
		{"auth basic", `Authorization: Basic dXNlcjpwYXNz`, `Authorization: Basic <secret-1>`},
		{"auth token scheme", `Authorization: Token abcdefghijklmnop`, `Authorization: Token <secret-1>`},
		{"auth other scheme", `authorization: Digest abcd1234`, `authorization: Digest <secret-1>`},
		{"auth no scheme", `Authorization: rawcredential123`, `Authorization: <secret-1>`},
		{"auth short bearer", `Authorization: Bearer abc`, `Authorization: Bearer <secret-1>`},
		{"colon password", `Password: hunter2`, `Password: <secret-1>`},
		{"colon token", `token: abcdef`, `token: <secret-1>`},
		{"header api key", `X-Api-Key: abc123`, `X-Api-Key: <secret-1>`},
		{"spaced equals", `api_key = x`, `api_key = <secret-1>`},
		{"spaced colon", `secret : y`, `secret : <secret-1>`},
		{"json no opening quote", `access_token":"x"`, `access_token":"<secret-1>"`},
		{"json access token", `{"access_token":"x","n":1}`, `{"access_token":"<secret-1>","n":1}`},
		{"github pat", `pat github_pat_11ABCDEFG0abcdefghijkl_abcdefghijklmnopqrstuvwxyz0123456789 end`, `pat <secret-1> end`},
		{"google key", `key AIzaSyA-abcdefghijklmnopqrstuvwxyz01234 end`, `key <secret-1> end`},
		{"infura path", `POST https://mainnet.infura.io/v3/0123456789abcdef0123456789abcdef failed`, `POST https://mainnet.infura.io/v3/<secret-1> failed`},
		{"alchemy path", `GET https://eth-mainnet.g.alchemy.com/v2/AbCdEfGhIjKlMnOpQrStUvWxYz012345?x=1`, `GET https://eth-mainnet.g.alchemy.com/v2/<secret-1>?x=1`},
		{"key before trailing ip segment", `https://h/AbCdEfGhIjKlMnOpQrStUvWxYz012345/1.1.1.1`, `https://h/<secret-1>/<ip-1>`},
		{"slug url kept", `see https://docs.example.com/how-to-run-a-node-without-any-problems-at-all`, `see https://docs.example.com/how-to-run-a-node-without-any-problems-at-all`},
		{"tx url kept", `https://etherscan.io/tx/0x` + strings.Repeat("ab", 32), `https://etherscan.io/tx/0x` + strings.Repeat("ab", 32)},
		{"bare session token", `session 0123456789abcdef0123456789abcdef ok`, `session <secret-1> ok`},
		{"0x 32 hex kept", `val 0x0123456789abcdef0123456789abcdef ok`, `val 0x0123456789abcdef0123456789abcdef ok`},
		{"64 hex kept", `hash ` + strings.Repeat("ab", 32), `hash ` + strings.Repeat("ab", 32)},
		{"ipv6 zone", `peer fe80::1%eth0`, `peer <ip-1>`},
		{"ipv6 zone punctuation", `on fe80::1%eth0, ok`, `on <ip-1>, ok`},
		{"set-cookie", `Set-Cookie: sid=abc123; Path=/; HttpOnly`, `Set-Cookie: <secret-1>`},
		{"quoted password", `password="correct horse battery staple" next=1`, `password="<secret-1>" next=1`},
		{"quoted header", `x-api-key: "zzz Secret" ok`, `x-api-key: "<secret-1>" ok`},
		{"single quoted", `secret : 'my pass phrase' ok`, `secret : '<secret-1>' ok`},
		{"json escaped quote", `{"password":"a\"bcdefsecretX","u":"bob"}`, `{"password":"<secret-1>","u":"bob"}`},
		{"unterminated quote", `password="never closed here`, `password="<secret-1>`},
		{"cli password", `run --password hunter2pass --verbose`, `run --password <secret-1> --verbose`},
		{"cli token", `--token x`, `--token <secret-1>`},
		{"cli api-key", `--api-key x`, `--api-key <secret-1>`},
		{"cli flag without value", `--token --verbose and -p hunter2 --token-file /etc/t`, `--token --verbose and -p hunter2 --token-file /etc/t`},
		{"private key 0x", `private_key=0x` + strings.Repeat("ab", 32), `private_key=<secret-1>`},
		{"json private key", `{"privateKey":"0x` + strings.Repeat("ab", 32) + `"}`, `{"privateKey":"<secret-1>"}`},
		{"mnemonic and seed", `seed=xyz123 mnemonic: abandon ability able`, `seed=<secret-2> mnemonic: <secret-1>`},
		{"url password with slash", `postgres://u:p4ss/w0rd@db:5432/x`, `postgres://<secret-1>@db:5432/x`},
		{"url empty user", `redis://:pw@cache`, `redis://<secret-1>@cache`},
		{"upper key prefixes", `SK-ANT-API03-abcdefghijklmnop1234 GHP_abcdefghijklmnopqrstuvwxyz0123456789 AIZASyA-abcdefghijklmnopqrstuvwxyz01234`, `<secret-1> <secret-2> <secret-3>`},
		{"40 hex sha kept", `commit ` + strings.Repeat("ab", 20) + ` built`, `commit ` + strings.Repeat("ab", 20) + ` built`},
		{"placeholder-looking secret", `password="<hunter2 not a placeholder"`, `password="<secret-1>"`},
		{"cookie lone cr", "Cookie: a=1\rsid=supersecretsessionvalue", `Cookie: <secret-1>`},
		{"cookie crlf then next header", "Cookie: a=1\r\nHost: node.example", "Cookie: <secret-1>\nHost: node.example"},
		{"mnemonic lone cr", "mnemonic: w1 w2\rw3 w4 w5 w6", `mnemonic: <secret-1>`},
		{"glued anthropic", `xxxsk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789`, `xxx<secret-1>`},
		{"glued after word", `tokensk-ant-api03-abcdefghijklmnop1234 ok`, `token<secret-1> ok`},
		{"glued github", `Xghp_abcdefghijklmnopqrstuvwxyz0123456789 Ygho_abcdefghijklmnopqrstuvwxyz0123456789 Zghs_abcdefghijklmnopqrstuvwxyz0123456789`, `X<secret-1> Y<secret-2> Z<secret-3>`},
		{"glued gitlab slack google", `aglpat-abcdefghij1234567890 bxoxb-abcdefghijklmnop1234 cAIzaSyA-abcdefghijklmnopqrstuvwxyz01234`, `a<secret-1> b<secret-2> c<secret-3>`},
		{"glued openai style", `Xsk-abcdefghijklmnopqrstuvwxyz0123`, `X<secret-1>`},
		{"hyphenated words kept", `disk-usage-statistics-total-report risk-assessment-for-every-node`, `disk-usage-statistics-total-report risk-assessment-for-every-node`},
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
		"Authorization: Basic abc", "http://u:p@h/", "https://h/v3/" + strings.Repeat("a1", 16), "fe80::1%",
	} {
		f.Add(s, uint8(10), uint8(80), uint8(120), uint8(200), uint8(250), uint8(30))
	}
	// Known secrets planted at random positions must never survive, whatever
	// text surrounds them.
	planted := []struct{ text, secret string }{
		{"203.0.113.77", "203.0.113.77"},
		{"2001:db8:85a3::8a2e:370:7334", "2001:db8:85a3::8a2e:370:7334"},
		{"token=SECRETVALUE123", "SECRETVALUE123"},
		{"sk-abcdefghijklmnop1234567", "abcdefghijklmnop1234567"},
		{"0123456789abcdef0123456789abcdef", "0123456789abcdef0123456789abcdef"},
		{`password="zebra quokka walrus llama"`, "quokka walrus"},
	}
	f.Fuzz(func(t *testing.T, s string, a, b, c, d, e, cutBack uint8) {
		if len(s) > 2000 {
			t.Skip("keeps the planted secrets inside the per-line cap")
		}
		offs := []int{int(a), int(b), int(c), int(d), int(e), int(cutBack)}
		sort.Ints(offs)
		var sb strings.Builder
		last := 0
		for i, o := range offs {
			at := o * (len(s) + 1) / 256
			sb.WriteString(s[last:at])
			sb.WriteString(" " + planted[i].text + " ")
			last = at
		}
		sb.WriteString(s[last:])
		in := sb.String()

		once := Redact([]string{in, in})
		if once[0] != once[1] {
			t.Fatalf("same input, different output: %q vs %q", once[0], once[1])
		}
		// An unmatched quote in s legitimately swallows up to the quoted
		// plant's opening quote, so that plant is only checked without quotes.
		quoted := strings.ContainsAny(s, "\"'")
		for i, p := range planted {
			if i == len(planted)-1 && quoted {
				continue
			}
			if strings.Contains(once[0], p.secret) {
				t.Fatalf("%q survived:\n  in %q\nout %q", p.secret, in, once[0])
			}
		}
		if twice := Redact(once); !reflect.DeepEqual(once, twice) {
			t.Fatalf("not idempotent:\n  in %q\nonce %q\ntwice %q", in, once, twice)
		}
		// A key straddling the per-line cut must not leave a piece behind.
		const key = "sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789"
		pad := maxLineBytes - len(key)/2 - len(s) + int(cutBack)%len(key) - len(key)/2
		cutIn := s + strings.Repeat("x", max(pad, 0)) + "=" + key
		if got := Redact([]string{cutIn})[0]; strings.Contains(got, key[:8]) || strings.Contains(got, key[len(key)-8:]) {
			t.Fatalf("key at the cut survived: %q", got[max(len(got)-100, 0):])
		}
		// Raw input too, which has no planted secrets but may be anything.
		raw := Redact([]string{s})
		if again := Redact(raw); !reflect.DeepEqual(raw, again) {
			t.Fatalf("not idempotent:\n  in %q\nonce %q\ntwice %q", s, raw, again)
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

func TestRedactCapsLongLinesAndInputs(t *testing.T) {
	long := strings.Repeat("x", 3<<20) + " tail 203.0.113.7"
	got := Redact([]string{long, "ok 10.0.0.1"})
	if len(got[0]) > maxLineBytes || !strings.HasSuffix(got[0], truncMarker) || got[1] != "ok <ip-1>" {
		t.Fatalf("long line: len %d, got %q", len(got[0]), got[1])
	}
	many := make([]string, maxLines+500)
	for i := range many {
		many[i] = "peer 10.0.0.1"
	}
	out := Redact(many)
	if len(out) != maxLines || !strings.Contains(out[maxLines-1], "omitted") {
		t.Fatalf("got %d lines, last %q", len(out), out[len(out)-1])
	}
	if again := Redact(out); !reflect.DeepEqual(again, out) {
		t.Fatal("capped output is not idempotent")
	}
	if again := Redact(got); !reflect.DeepEqual(again, got) {
		t.Fatal("truncated output is not idempotent")
	}
}

// secretShapes are credential formats the cut points must not leak a piece of.
var secretShapes = map[string]string{
	"anthropic":  "sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789",
	"github":     "ghp_abcdefghijklmnopqrstuvwxyz0123456789",
	"github pat": "github_pat_11ABCDEFG0abcdefghijkl_abcdefghijklmnopqrstuvwxyz0123456789",
	"gitlab":     "glpat-abcdefghij1234567890",
	"google":     "AIzaSyA-abcdefghijklmnopqrstuvwxyz01234",
	"jwt":        "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r",
	"session":    "0123456789abcdef0123456789abcdef",
	"bearer":     "Bearer AbCdEfGh1234567890zyx",
	"token=":     "token=SECRETVALUE1234567890",
	"quoted":     `password="correct horse battery staple"`,
}

// prefixed lists the shapes recognised by their prefix alone, even glued to
// a word character.
var prefixed = map[string]bool{"anthropic": true, "github": true, "github pat": true, "gitlab": true, "google": true}

func leaksRun(out, secret string, n int) (string, bool) {
	for i := 0; i+n <= len(secret); i++ {
		if strings.Contains(out, secret[i:i+n]) {
			return secret[i : i+n], true
		}
	}
	return "", false
}

// A secret that straddles the per-line cut must be redacted whole, not cut
// in half with its prefix left behind.
func TestRedactSecretsAtTheLineCut(t *testing.T) {
	for name, secret := range secretShapes {
		// The part of the secret that is the value, for the 8-character check.
		value := secret
		if i := strings.IndexAny(secret, "= "); i >= 0 {
			value = strings.TrimLeft(secret[i:], `= "`)
		}
		for _, sep := range []string{"=", ""} {
			if sep == "" && !prefixed[name] {
				continue // only keys with a distinctive prefix can be glued
			}
			for start := maxLineBytes - len(secret) - 2; start <= maxLineBytes+2; start++ {
				in := strings.Repeat("x", start) + sep + secret
				out := Redact([]string{in})[0]
				if run, bad := leaksRun(out, value, 8); bad {
					t.Fatalf("%s (sep %q) at offset %d leaked %q", name, sep, start, run)
				}
			}
		}
	}
}

// The hard bound before redaction cuts mid-secret, so a key-shaped run that
// touches that cut is dropped instead of kept as a prefix.
func TestRedactSecretsAtTheHardCut(t *testing.T) {
	defer func(old int) { maxRawLineBytes = old }(maxRawLineBytes)
	maxRawLineBytes = 1000
	for name, secret := range secretShapes {
		value := secret
		if i := strings.IndexAny(secret, "= "); i >= 0 {
			value = strings.TrimLeft(secret[i:], `= "`)
		}
		for start := maxRawLineBytes - len(truncMarker) - len(secret) - 2; start <= maxRawLineBytes+2; start++ {
			in := strings.Repeat("x", start) + "=" + secret
			out := Redact([]string{in})[0]
			if run, bad := leaksRun(out, value, 8); bad {
				t.Fatalf("%s at raw offset %d leaked %q", name, start, run)
			}
		}
	}
}

func TestRedactPathologicalInputsAreFast(t *testing.T) {
	const n = 3 << 20
	inputs := map[string]string{
		"dots":      strings.Repeat("1.", n/2),
		"colons":    strings.Repeat(":", n),
		"hexcolons": strings.Repeat("a:", n/2),
		"tokens":    strings.Repeat("token=", n/6),
		"bearers":   strings.Repeat("Bearer ", n/7),
		"urls":      strings.Repeat("http://a:b@", n/11),
		"hex":       strings.Repeat("0123456789abcdef", n/16),
		"slashes":   "https://h" + strings.Repeat("/", n),
	}
	single := 100 * time.Millisecond
	if raceEnabled {
		single = time.Second // the race detector costs about 10x
	}
	for name, in := range inputs {
		start := time.Now()
		Redact([]string{in})
		if d := time.Since(start); d > single {
			t.Errorf("%s: %v", name, d)
		}
	}
	// This input is match-dense on every line, so the bound is looser than
	// for the single-line cases, and looser still under the race detector.
	bound := 600 * time.Millisecond
	if raceEnabled {
		bound = 5 * time.Second
	}
	lines := make([]string, maxLines)
	for i := range lines {
		lines[i] = strings.Repeat("1.2.3.4 token=a http://u:p@h/ ", 200)
	}
	start := time.Now()
	Redact(lines)
	if d := time.Since(start); d > bound {
		t.Errorf("max-size input: %v", d)
	}
}
