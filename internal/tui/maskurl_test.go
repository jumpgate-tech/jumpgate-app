package tui

import (
	"strings"
	"testing"
	"time"
)

const k32 = "0123456789abcdef0123456789abcdef"

func TestMaskURLFailsClosed(t *testing.T) {
	for in, want := range map[string]string{
		"h:8545/v3/" + k32 + "?next=http://x":           "h:8545/v3/***?next=***",
		"/abcdefghijklmnopqrstuvwxyzabcdef":             "/***",
		"https://h/0123456789abcdef%2F0123456789abcdef": "https://h/***",
		"https://h/x;key=SECRET":                        "https://h/x;***",
		"https://h/v1/abcd1234":                         "https://h/v1/***",
		"https://us er:SECRET@h":                        "https://***",
		"https:u:SECRET@h":                              "***",
		"//u:SECRET@h/":                                 "//***/",
		"u:SECRET@h":                                    "***",
		"https://h/369/rpc/ws":                          "https://h/369/rpc/ws",
		"https://h/?SECRET":                             "https://h/?***",
		"https://h/?Ab3dEfGh1jKlMnOpQrStUv=1":           "https://h/?***=***",
		"wss://h:8546/v1":                               "wss://h:8546/v1",
		"http://127.0.0.1:8545":                         "http://127.0.0.1:8545",
		"http://[::1]:8545/rpc":                         "http://[::1]:8545/rpc",
		"https://user:hunter2@rpc.example.org:8443/x":   "https://***/x",
		"https://user:p@ss/w@rpc.example.org/":          "https://***/***/",
		"https://tok3nvalue@rpc.example.org/":           "https://***/",
		"https://user:SECRET%2540h/":                    "https://***/",
		"user%3ASECRET%40h/x":                           "***/x",
		"https://u:SECRET\uff20h/":                      "https://***/",
		"https://u:SE?CRET@h/":                          "https://***/",
		"https://u:SE#CRET@h/":                          "https://***/",
		"https://h:80:90/x":                             "https://***/x",
		"https://h\u00e9/x":                             "https://***/x",
	} {
		if got := maskURL(in); got != want {
			t.Errorf("maskURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskTextFailsClosed(t *testing.T) {
	for _, in := range []string{
		"https:u:SECRET@h", "//u:SECRET@h", "u:SECRET@h", "https://us er:SECRET@h",
		"https://u:SECRET\tx@h/", "see https://u:SECRET x@h/ now", "(https://h/SECRET1234567890abcdef)",
		"host:8545/v3/SECRET1234567890abcdef", "url=https://u:SECRET@h", "x https://h/a;key=SECRET y",
		"u:SECRET @h", "see u:SECRET @h now", "u:SECRET\uff20h", "u:SECRET%40h", "https:// u:SECRET @h",
	} {
		if got := maskText(in); strings.Contains(got, "SECRET") {
			t.Errorf("maskText(%q) = %q leaks", in, got)
		}
	}
	for _, in := range []string{"the certificate expires in 6 days", "sudo ufw deny 8545", "8545 is bound to 0.0.0.0"} {
		if got := maskText(in); got != in {
			t.Errorf("maskText(%q) = %q, plain text changed", in, got)
		}
	}
}

func TestMaskTextUnicodeSpacesDoNotSplitAToken(t *testing.T) {
	for _, sp := range []string{"\u00a0", "\u2003", "\u3000"} {
		in := "https://h/v3/" + sp + "Zk9SECRETQx7"
		if got := maskText(in); strings.Contains(got, "SECRET") {
			t.Errorf("maskText(%q) = %q leaks", in, got)
		}
	}
}

var invisibles = []string{"\u200b", "\u200c", "\u200d", "\u2060", "\ufeff", "\u00ad"}

func TestMaskTextHidesASecretBehindInvisibleCharactersBeforeAt(t *testing.T) {
	var ins []string
	for _, ch := range append([]string{"", "\u2003", "\uff20", "%40"}, invisibles...) {
		at := "@"
		if ch == "\uff20" || ch == "%40" {
			at = ""
		}
		ins = append(ins, "u:SECRET "+ch+at+"h", "u:SECRET "+ch+" "+at+"h", "u:SECRET \u2003 "+ch+at+"h")
	}
	for _, in := range ins {
		if got := maskText(in); strings.Contains(got, "SECRET") {
			t.Errorf("maskText(%q) = %q leaks", in, got)
		}
		if got := maskText(sanitizeLine(in)); strings.Contains(got, "SECRET") {
			t.Errorf("sanitize+maskText(%q) = %q leaks", in, got)
		}
	}
}

func TestSanitizeDropsFormatCharacters(t *testing.T) {
	for _, ch := range invisibles {
		if got := sanitize("a" + ch + "b"); got != "ab" {
			t.Errorf("sanitize kept %q: %q", ch, got)
		}
	}
	if got := sanitize("caf\u00e9 \u2003x"); got != "caf\u00e9 \u2003x" {
		t.Errorf("sanitize changed ordinary text: %q", got)
	}
}

// The secret walks across every offset around the cut; the token before the
// cut's whitespace is masked too, because its "@" partner may be dropped.
func TestMaskTextCutWalksAcrossASecretWithASpaceBeforeAt(t *testing.T) {
	for _, secret := range []string{"u:SECRET @h", "u:SECRET  @h", "https://u:SECRET @h/"} {
		for off := 0; off <= len(secret)+2; off++ {
			pad := maxMaskText - off
			in := strings.Repeat("a ", pad/2) + strings.Repeat("b", pad%2) + secret + strings.Repeat(" tail", 20)
			if got := maskText(in); strings.Contains(got, "SECRET") {
				t.Fatalf("secret %q offset %d leaks: %q", secret, off, got[max(len(got)-30, 0):])
			}
		}
	}
}

// A cut at the size cap must not leave part of a secret behind.
func TestMaskTextCapsItsInputWithoutExposingACutSecret(t *testing.T) {
	secret := "https://u:SECRETSECRETSECRET@h/"
	for cut := 1; cut < len(secret); cut++ {
		in := strings.Repeat("a ", (maxMaskText-cut)/2) + strings.Repeat("b", (maxMaskText-cut)%2) + secret
		got := maskText(in)
		if strings.Contains(got, "SEC") || !strings.HasSuffix(got, "\u2026") || len(got) > maxMaskText+4 {
			t.Fatalf("cut %d: tail %q len %d", cut, got[max(len(got)-40, 0):], len(got))
		}
	}
}

func TestMaskTextIsLinear(t *testing.T) {
	if raceEnabled {
		t.Skip("timing is meaningless under the race detector")
	}
	for _, in := range []string{strings.Repeat("https:// ", 20000), strings.Repeat("https:// ", 20000) + "x@h", strings.Repeat("a @b ", 20000)} {
		start := time.Now()
		maskTokens(in)
		if d := time.Since(start); d > 50*time.Millisecond {
			t.Errorf("maskTokens took %v on %d bytes", d, len(in))
		}
	}
}

func FuzzMaskURL(f *testing.F) {
	for _, s := range []string{"a", "k3y", "0123456789abcdef", "x/y", "p@ss", "%40?#", "\uff20\u00e9 :"} {
		f.Add(s, uint8(0), uint8(0), uint8(0))
	}
	alphabet := []rune("0123456789abcdefXYZ%?#@: \uff20\u00e9/")
	f.Fuzz(func(t *testing.T, raw string, scheme, where, shape uint8) {
		var b []rune
		for _, r := range raw {
			b = append(b, alphabet[int(r)%len(alphabet)])
		}
		if len(b) > 24 {
			b = b[:24]
		}
		mid := string(b)
		secret := "Zk9" + mid + "Qx7" // never an allowlisted word
		plain := strings.NewReplacer("/", "_").Replace(secret)
		sch := []string{"https://", "http://", "wss://", "", "//", "https:"}[int(scheme)%6]
		var u string
		switch where % 6 {
		case 0: // the password, which may hold "/"
			u = sch + "user:" + secret + "@host.example:8545/v3/rpc"
		case 1: // a bare token as the user (a "/" in it would read as a host)
			secret = plain
			u = sch + secret + "@host.example/v1"
		case 2:
			u = sch + "host.example:8545/v3/" + secret + "/x"
		case 3:
			u = sch + "host.example:8545/v3?a=1&key=" + secret + "&b=2"
		case 4:
			u = sch + "host.example:8545/v3?a=1#" + secret
		default:
			u = sch + "host.example:8545/v3?a=1&b=" + secret
		}
		if where%6 == 0 && shape >= 128 {
			gaps := []string{" ", "  ", "\t", "\u00a0", "\u2003", "\u3000", "\u200b", "\u200c", "\u200d", "\u2060", "\ufeff", "\u00ad", " \u200b", "\u2003 ", " \u2003 "}
			u = sch + "user:" + secret + gaps[int(shape)%len(gaps)] + "@host.example/v1"
		}
		if shape%2 == 1 && where%6 >= 2 {
			u += "?later=" + secret
		}
		for _, in := range []string{u, "note " + u + " end", "(" + u + ")", "u=" + u + ","} {
			if got := maskURL(in); strings.Contains(got, secret) {
				t.Fatalf("maskURL(%q) = %q", in, got)
			}
			// What the screen does: sanitize, then mask. The secret itself
			// is checked after the same cleaning.
			clean := sanitizeLine(secret)
			if got := maskText(sanitizeLine(in)); strings.Contains(got, clean) {
				t.Fatalf("sanitize+maskText(%q) = %q", in, got)
			}
			if got := maskText(in); strings.Contains(got, secret) {
				t.Fatalf("maskText(%q) = %q", in, got)
			}
		}
	})
}
