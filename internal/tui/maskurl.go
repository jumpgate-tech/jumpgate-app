package tui

import (
	"strings"
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

// maxMaskText caps what maskText reads, so a hostile field stays cheap.
const maxMaskText = 4096

// maskText masks free text (a warning, a check's why, detail or fix) with a
// blunt rule that over-masks ordinary prose on purpose. Callers sanitize
// first. Per whitespace-separated token:
//
//   - a token holding any of @ : / % \ = ? # & or any non-ASCII rune is
//     suspicious and becomes *** whole;
//   - a token holding @ or % or any non-ASCII rune is at-like: the two
//     tokens before it go too (a space in userinfo, an encoded or look-alike
//     @ after it);
//   - the one token after a token holding "://" goes too (a key after a
//     space in the path);
//   - a word holding "://" followed later on the same line by an at-like word
//     takes every word between them, and when the cut drops the rest of the
//     line, every word after it;
//   - input over 4 KB is cut on a rune boundary, its last two tokens go and
//     it ends in an ellipsis (the cut may have dropped the @ that hides them).
//
// Tokens split on ASCII whitespace only, which is kept as it was.
func maskText(s string) string {
	cut := len(s) > maxMaskText
	if cut {
		n := maxMaskText
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	// parts alternates runs of whitespace and words; words indexes the words.
	var parts []string
	var words []int
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || (i > start && isSpace(s[i]) != isSpace(s[start])) {
			if i > start {
				if !isSpace(s[start]) {
					words = append(words, len(parts))
				}
				parts = append(parts, s[start:i])
			}
			start = i
		}
	}
	mask := make([]bool, len(words))
	// open is the first word after the earliest "://" word on this line that
	// has not met an at-like word yet; -1 when none.
	open := -1
	for w, p := range words {
		if p > 0 && !isSpace(parts[p][0]) && strings.ContainsAny(parts[p-1], "\n\r") {
			open = -1
		}
		t := parts[p]
		ascii := asciiOnly(t)
		mask[w] = mask[w] || !ascii || strings.ContainsAny(t, "@:/%\\=?#&")
		if !ascii || strings.ContainsAny(t, "@%") {
			for k := max(w-2, 0); k < w; k++ {
				mask[k] = true
			}
		}
		if !ascii || strings.ContainsAny(t, "@%") {
			// An at-like word closes a URL opened earlier on the line: every
			// word between them may be userinfo with spaces in it.
			if open >= 0 {
				for k := open; k < w; k++ {
					mask[k] = true
				}
				open = -1
			}
		}
		if strings.Contains(t, "://") {
			if w+1 < len(words) {
				mask[w+1] = true
			}
			if open < 0 {
				open = w + 1
			}
		}
	}
	if cut {
		// The at-like word may have been cut off: an unclosed URL takes the
		// rest of its line with it.
		if open >= 0 {
			for k := open; k < len(words); k++ {
				mask[k] = true
			}
		}
		for k := max(len(words)-2, 0); k < len(words); k++ {
			mask[k] = true
		}
	}
	for w, p := range words {
		if mask[w] {
			parts[p] = "***"
		}
	}
	out := strings.Join(parts, "")
	if cut {
		out += "\u2026"
	}
	return out
}

func isSpace(c byte) bool { return strings.IndexByte(" \t\n\r\v\f", c) >= 0 }

func asciiOnly(t string) bool {
	for i := 0; i < len(t); i++ {
		if t[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
