package tui

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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
		if strings.Contains(got, "SEC") || !strings.HasSuffix(got, "\u2026") || len(got) > 2*maxMaskText+3 {
			t.Fatalf("cut %d: tail %q len %d", cut, got[max(len(got)-40, 0):], len(got))
		}
	}
}

func TestMaskTextIsFastOnWorstCaseInput(t *testing.T) {
	if raceEnabled {
		t.Skip("timing is meaningless under the race detector")
	}
	worst := []string{
		strings.Repeat("@", maxMaskText),
		strings.Repeat("@ ", maxMaskText/2),
		strings.Repeat("https:// ", maxMaskText/9+1)[:maxMaskText],
		strings.Repeat("\u00e9 ", maxMaskText/3),
		strings.Repeat("a ", maxMaskText/2),
		strings.Repeat("@ ", 100000), // capped first
	}
	for _, in := range worst {
		start := time.Now()
		maskText(in)
		if d := time.Since(start); d > 5*time.Millisecond {
			t.Errorf("maskText took %v on %d bytes", d, len(in))
		}
	}
}

// Round 4 leaks: each must be masked, raw and after sanitize.
func TestMaskTextRound4Leaks(t *testing.T) {
	ins := []string{
		"u:SECRET %2540h", "u:SECRET%2540h", "u:SECRET \u0301@h",
		"https://u:A SECRET B\uff20h/", "https://h/v3/ Zk9SECRETQx7",
		"see https://h/v3/ SECRET now", "x SECRET \u00e9", "SECRET y %", "SECRET y z@h",
	}
	for _, ch := range []string{"\u034f", "\u115f", "\u3164", "\u2800"} {
		ins = append(ins, "u:SECRET "+ch+"@h", "SECRET "+ch+"@h", "a SECRET b "+ch+"@h", "SECRET b "+ch)
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

// The blunt rule, exactly: suspicious tokens become ***, at-like tokens take
// two before, "://" takes one after; whitespace is kept as it was.
func TestMaskTextBluntOutput(t *testing.T) {
	for in, want := range map[string]string{
		"see https://u:pw@h/p?k=v now":    "*** *** ***",
		"a b c d@e f":                     "a *** *** *** f",
		"open TCP/UDP 30303, e.g. now":    "open *** 30303, e.g. now",
		"key=v  x\ty":                     "***  x\ty",
		"caf\u00e9 is open":               "*** is open",
		"one two 50% three":               "*** *** *** three",
		"see https://h/x next then":       "see *** *** then",
		"\tleading and trailing \n":       "\tleading and trailing \n",
		"plain words stay as they are 42": "plain words stay as they are 42",
	} {
		if got := maskText(in); got != want {
			t.Errorf("maskText(%q) = %q, want %q", in, got, want)
		}
	}
}

// The 4 KB cut: u:SECRET @h walks across every offset around the cut, and
// the result is capped and ends in an ellipsis.
func TestMaskTextCutWalk(t *testing.T) {
	secret := "u:SECRET @h"
	for off := -20; off <= len(secret)+20; off++ {
		pad := maxMaskText - off
		in := strings.Repeat("a ", pad/2) + strings.Repeat("b", pad%2) + secret + strings.Repeat(" tail", 20)
		got := maskText(in)
		if strings.Contains(got, "SECRET") {
			t.Fatalf("offset %d leaks: %q", off, got[max(len(got)-30, 0):])
		}
		if len(in) > maxMaskText && !strings.HasSuffix(got, "\u2026") {
			t.Fatalf("offset %d: cut without ellipsis: %q", off, got[max(len(got)-30, 0):])
		}
	}
	// The cut lands on a rune boundary (one é of two survives) and masks
	// the last two tokens; the é is at-like, so it takes two before it.
	in := strings.Repeat("a ", maxMaskText/2-1) + "\u00e9\u00e9"
	got := maskText(in)
	if !strings.HasSuffix(got, " a *** *** ***\u2026") || !utf8.ValidString(got) {
		t.Fatalf("cut tail %q", got[max(len(got)-30, 0):])
	}
}

// Fuzz: random token sequences over a hostile alphabet, with SECRET planted
// as a whole token within two tokens before, or one after, an at-like or
// "://" token. sanitize then maskText must never show it.
func FuzzMaskText(f *testing.F) {
	f.Add([]byte("seed"), uint8(0), uint8(0))
	f.Add([]byte{1, 2, 3, 4, 5, 6, 7, 8}, uint8(1), uint8(3))
	f.Add([]byte{0xff, 0x10, 0x20}, uint8(2), uint8(7))
	pieces := []string{
		"a", "b", "Z", "k", "0", "9", "@", ":", "/", "%", "\uff20", "\ufe6b",
		"\u0301", "\u034f", "\u200b", "\u200d", "\u2060", "\ufeff", "\u00ad", "\u00a0",
		"\u115f", "\u3164", "\u2800", "%40", "%2540", "://", "x",
	}
	spaces := []string{" ", "  ", "\t", "\n", "\r", "\v", "\f", " \t "}
	atLike := []string{"@h", "\uff20h", "\ufe6bh", "%40h", "%2540h", "\u0301@h", "\u034f@h", "\u115f@h", "\u3164@h", "\u2800@h", "\u00a0@h", "\u00e9", "\u200b@h"}
	schemes := []string{"https://h/v3/", "x://y", "://", "wss://h:8546"}
	f.Fuzz(func(t *testing.T, raw []byte, trig, where uint8) {
		var toks []string
		for i := 0; i+1 < len(raw) && len(toks) < 40; i += 2 {
			n := int(raw[i]%4) + 1
			var tok strings.Builder
			for j := 0; j < n; j++ {
				tok.WriteString(pieces[int(raw[i+1]+uint8(j)*7)%len(pieces)])
			}
			toks = append(toks, tok.String())
		}
		at := len(toks) / 2
		trigger := atLike[int(trig)%len(atLike)]
		var planted []string
		switch where % 3 {
		case 0: // right before an at-like token
			planted = []string{"SECRET", trigger}
		case 1: // two before an at-like token
			planted = []string{"SECRET", "word", trigger}
		default: // right after a "://" token
			planted = []string{schemes[int(trig)%len(schemes)], "SECRET"}
		}
		toks = append(toks[:at], append(planted, toks[at:]...)...)
		var b strings.Builder
		for i, tok := range toks {
			if i > 0 {
				sp := 0
				if len(raw) > 0 {
					sp = int(raw[i%len(raw)])
				}
				b.WriteString(spaces[sp%len(spaces)])
			}
			b.WriteString(tok)
		}
		in := b.String()
		if got := maskText(sanitize(in)); strings.Contains(got, "SECRET") {
			t.Fatalf("sanitize+maskText(%q) = %q", in, got)
		}
	})
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

// A "://" word followed later on its line by an at-like word takes every word
// between them (userinfo with spaces), and when the 4 KB cut drops the at-like
// word, the rest of the line.
func TestMaskTextURLWithSpacesInUserinfo(t *testing.T) {
	in := "https://u:A B SECRET D E F@h/"
	if got := maskText(in); strings.Contains(got, "SECRET") {
		t.Errorf("maskText(%q) = %q leaks", in, got)
	}
	if got := maskText("ok\nhttps://u:A B SECRET D E F@h/ tail"); strings.Contains(got, "SECRET") || !strings.HasPrefix(got, "ok\n") {
		t.Errorf("multi-line: %q", got)
	}
	// A new line ends the search: a later line's @ does not reach back.
	if got := maskText("https://h/ x y z\nA B C d@e"); !strings.Contains(got, "y z\nA ***") {
		t.Errorf("a later line reached back: %q", got)
	}
	for cut := 0; cut < 40; cut++ {
		pad := maxMaskText - cut
		cutIn := strings.Repeat("a ", pad/2) + strings.Repeat("b", pad%2) + in + strings.Repeat(" tail", 20)
		if got := maskText(cutIn); strings.Contains(got, "SECRET") {
			t.Fatalf("cut offset %d leaks: %q", cut, got[max(len(got)-30, 0):])
		}
	}
	// Cut right before the @: nothing after "://" survives.
	pre := strings.Repeat("a ", (maxMaskText-len("https://u:A B SECRET D E F"))/2) + "https://u:A B SECRET D E F@h/"
	if got := maskText(pre); strings.Contains(got, "SECRET") {
		t.Errorf("cut before @ leaks: %q", got[max(len(got)-30, 0):])
	}
}
