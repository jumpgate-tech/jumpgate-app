package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// maskURL hides what a URL may carry as a secret, and fails closed: when it
// is unsure it masks, so it over-masks on purpose.
//
//   - the authority is shown only when it is entirely a host[:port] or a
//     bracketed IPv6 address; any userinfo, "%", non-ASCII or odd character
//     makes all of it ***;
//   - every path segment is masked unless it is all digits (up to 10, a chain
//     id or version) or a short lowercase word (up to 8 letters: rpc, ws,
//     api, eth) or v and 1-2 digits; so are ";params" and anything encoded
//     (a segment with "%" may hide a "/");
//   - every query value, a query key that is not a plain short word, and the
//     fragment.
//
// The scheme is recognised only as a leading "scheme://" (or "//"); input
// without one is host[:port] and the rest. Scheme, host and port stay
// visible. It works on text, not net/url, so a malformed URL is masked too.
// Callers sanitize first.
// leadingScheme is a leading "scheme://" or "//", or "".
func leadingScheme(s string) string {
	if strings.HasPrefix(s, "//") {
		return "//"
	}
	if s == "" || !(s[0] >= 'a' && s[0] <= 'z' || s[0] >= 'A' && s[0] <= 'Z') {
		return ""
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '+', c == '.', c == '-':
		case c == ':':
			if strings.HasPrefix(s[i:], "://") {
				return s[:i+3]
			}
			return ""
		default:
			return ""
		}
	}
	return ""
}

func maskURL(s string) string {
	scheme, rest := "", s
	if loc := leadingScheme(s); loc != "" {
		scheme, rest = loc, s[len(loc):]
	}
	// The authority runs to the first "/" and nothing else ends it: a "?", "#"
	// or space in it may be part of a password. It is shown only when it is
	// plainly a host (with a port); with userinfo, encoding or anything odd,
	// the whole of it is hidden.
	end := strings.IndexByte(rest, '/')
	if end < 0 {
		end = len(rest)
	}
	auth, tail := rest[:end], rest[end:]
	if !plainAuthority(auth) {
		auth = "***"
	}
	frag := ""
	if i := strings.Index(tail, "#"); i >= 0 {
		tail, frag = tail[:i], "#***"
	}
	query := ""
	if i := strings.Index(tail, "?"); i >= 0 {
		pairs := strings.FieldsFunc(tail[i+1:], func(r rune) bool { return r == '&' || r == ';' })
		for j, p := range pairs {
			if k := strings.Index(p, "="); k >= 0 {
				key := p[:k]
				if !plainKey(key) {
					key = "***"
				}
				pairs[j] = key + "=***"
			} else {
				pairs[j] = "***"
			}
		}
		tail, query = tail[:i], "?"+strings.Join(pairs, "&")
	}
	segs := strings.Split(tail, "/")
	for i, seg := range segs {
		head, params, hasParams := strings.Cut(seg, ";")
		if !plainSegment(head) || (i > 0 && keyName(segs[i-1])) {
			head = "***"
		}
		if hasParams && params != "" {
			head += ";***"
		}
		segs[i] = head
	}
	return scheme + auth + strings.Join(segs, "/") + query + frag
}

// plainAuthority: a host (letters, digits, dots, hyphens) or a bracketed
// IPv6 address, with an optional :port, and nothing else.
func plainAuthority(a string) bool {
	host, port := a, ""
	if strings.HasPrefix(a, "[") {
		end := strings.IndexByte(a, ']')
		if end < 2 {
			return false
		}
		for _, r := range a[1:end] {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' || r == ':' || r == '.') {
				return false
			}
		}
		host, port = "", a[end+1:]
	} else if c := strings.IndexByte(a, ':'); c >= 0 {
		host, port = a[:c], a[c:]
	}
	for i := 0; i < len(host); i++ {
		c := host[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-') {
			return false
		}
	}
	if port == "" {
		return true
	}
	if len(port) < 2 || port[0] != ':' {
		return false
	}
	for i := 1; i < len(port); i++ {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	return true
}

// plainSegment: empty, or on the small allowlist described on maskURL.
func plainSegment(seg string) bool {
	if len(seg) == 0 {
		return true
	}
	lower, digits := 0, 0
	for i := 0; i < len(seg); i++ {
		switch c := seg[i]; {
		case c >= 'a' && c <= 'z':
			lower++
		case c >= '0' && c <= '9':
			digits++
		default:
			return false
		}
	}
	switch {
	case digits == 0:
		return lower <= 8
	case lower == 0:
		return digits <= 10
	}
	return seg[0] == 'v' && lower == 1 && digits <= 2 // v1, v3
}

var keyNameStrip = strings.NewReplacer("-", "", "_", "")

// keyName: a segment that announces a key in the next one (/key/short).
func keyName(seg string) bool {
	if len(seg) > 12 {
		return false
	}
	switch strings.ToLower(keyNameStrip.Replace(seg)) {
	case "key", "apikey", "token", "auth", "secret", "accesstoken":
		return true
	}
	return false
}

func plainKey(k string) bool {
	if len(k) == 0 || len(k) > 16 {
		return false
	}
	for i := 0; i < len(k); i++ {
		if c := k[i]; !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// tokens splits s into alternating runs of whitespace and non-whitespace,
// which join back to s.
func tokens(s string) []string {
	var out []string
	start, space := 0, false
	for i, r := range s {
		if sp := unicode.IsSpace(r); i > 0 && sp != space {
			out, start = append(out, s[start:i]), i
			space = sp
		} else if i == 0 {
			space = sp
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// maxMaskText caps what maskText reads, so a hostile field cannot make it
// slow. The cut is masked.
const maxMaskText = 4096

// maskText masks the URL-like parts of free text (a warning, a check's
// detail or fix), fail closed, in one left-to-right pass over its
// whitespace-separated tokens (see maskTokens). Input beyond 4 KB is cut and
// ends in an ellipsis; the cut cannot leave part of a secret, because the
// last partial token and anything after the last "://" are masked.
func maskText(s string) string {
	if len(s) <= maxMaskText {
		return maskTokens(s)
	}
	cut := maxMaskText
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	head := s[:cut]
	// Mask the last (possibly partial) token, and everything from the last
	// "://" on: its authority may be cut before the "@" that would hide it.
	const space = " \t\r\n\v\f"
	start := strings.LastIndexAny(head, space) + 1
	if i := strings.LastIndex(head, "://"); i >= 0 {
		start = min(start, strings.LastIndexAny(head[:i], space)+1)
	}
	return maskTokens(head[:start]) + "***\u2026"
}

// maskTokens: a token holding "://", "@", "/" or an encoded or fullwidth @
// is masked as a URL. A "scheme://" token that ends before any "/" is joined
// with the following tokens up to the first one holding "@" (a space inside
// the userinfo); a token followed, across whitespace, by one that starts with
// "@" is joined with it. The next "@" token is found by one backward pass, so
// the work is linear. Plain words and numbers are left alone.
func maskTokens(s string) string {
	toks := tokens(s)
	nextAt := make([]int, len(toks)+1) // first token at or after i holding "@"
	nextAt[len(toks)] = -1
	for i := len(toks) - 1; i >= 0; i-- {
		nextAt[i] = nextAt[i+1]
		if strings.Contains(toks[i], "@") {
			nextAt[i] = i
		}
	}
	var out strings.Builder
	out.Grow(len(s))
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if strings.TrimSpace(t) == "" {
			out.WriteString(t)
			continue
		}
		switch k := strings.Index(t, "://"); {
		case k >= 0 && !strings.ContainsAny(t[k+3:], "/?#@") && nextAt[i] > i:
			t = strings.Join(toks[i:nextAt[i]+1], "")
			i = nextAt[i]
		case i+2 < len(toks) && strings.HasPrefix(toks[i+2], "@") && strings.TrimSpace(toks[i+1]) == "":
			t += toks[i+1] + toks[i+2]
			i += 2
		}
		if sensitiveToken(t) {
			t = maskURL(t)
		}
		out.WriteString(t)
	}
	return out.String()
}

func sensitiveToken(t string) bool {
	if strings.ContainsAny(t, "@/?#&=\uff20\ufe6b") || strings.Contains(t, "://") {
		return true
	}
	return strings.Contains(strings.ToLower(t), "%40")
}
