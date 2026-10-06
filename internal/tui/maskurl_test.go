package tui

import (
	"strings"
	"testing"
)

const k32 = "0123456789abcdef0123456789abcdef"

func TestMaskURLFailsClosed(t *testing.T) {
	for in, want := range map[string]string{
		"h:8545/v3/" + k32 + "?next=http://x":           "h:8545/v3/***?next=***",
		"/abcdefghijklmnopqrstuvwxyzabcdef":             "/***",
		"https://h/0123456789abcdef%2F0123456789abcdef": "https://h/***",
		"https://h/x;key=SECRET":                        "https://h/x;***",
		"https://h/v1/abcd1234":                         "https://h/v1/***",
		"https://us er:SECRET@h":                        "https://us er:***@h",
		"https:u:SECRET@h":                              "https:***@h",
		"//u:SECRET@h/":                                 "//u:***@h/",
		"u:SECRET@h":                                    "u:***@h",
		"https://h/369/rpc/ws":                          "https://h/369/rpc/ws",
		"https://h/?SECRET":                             "https://h/?***",
		"https://h/?Ab3dEfGh1jKlMnOpQrStUv=1":           "https://h/?***=***",
		"wss://h:8546/v1":                               "wss://h:8546/v1",
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

func FuzzMaskURL(f *testing.F) {
	for _, s := range []string{"a", "k3y", "0123456789abcdef", "x/y", "p@ss"} {
		f.Add(s, uint8(0), uint8(0), uint8(0))
	}
	f.Fuzz(func(t *testing.T, raw string, scheme, where, shape uint8) {
		var b strings.Builder
		for _, r := range raw {
			if r < 128 && (r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
				b.WriteRune(r)
			}
		}
		secret := "Zk9" + b.String() + "Qx7" // never an allowlisted word
		sch := []string{"https://", "http://", "wss://", "", "//", "https:"}[int(scheme)%6]
		var u string
		switch where % 3 {
		case 0:
			u = sch + "user:" + secret + "@host.example:8545/v3/rpc"
		case 1:
			u = sch + "host.example:8545/v3/" + secret + "/x"
		default:
			u = sch + "host.example:8545/v3?key=" + secret + "&a=1"
		}
		if where%3 == 0 && shape%2 == 1 {
			u = sch + secret + "@host.example/v1" // a bare token as the user
		}
		for _, in := range []string{u, "note " + u + " end", "(" + u + ")", "u=" + u + ","} {
			if got := maskURL(in); strings.Contains(got, secret) {
				t.Fatalf("maskURL(%q) = %q", in, got)
			}
			if got := maskText(in); strings.Contains(got, secret) {
				t.Fatalf("maskText(%q) = %q", in, got)
			}
		}
	})
}
