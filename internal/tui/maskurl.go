package tui

import (
	"regexp"
	"strings"
)

// maskURL hides what a URL carries as a secret: the userinfo password
// (user:***@), key-like path segments (Infura and Alchemy put the API key in
// the path), and every query value and fragment. Scheme, host, port and
// short readable path segments stay. It works on text, not net/url, so a
// malformed URL is masked too instead of shown whole. Callers sanitize first.
func maskURL(s string) string {
	scheme, rest := "", s
	if i := strings.Index(s, "://"); i >= 0 {
		scheme, rest = s[:i+3], s[i+3:]
	}
	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		end = len(rest)
	}
	// A password may hold "/" and "@": the userinfo runs to the last "@"
	// before the query, when what precedes it looks like user:password.
	pre := rest
	if q := strings.IndexAny(rest, "?#"); q >= 0 {
		pre = rest[:q]
	}
	if at := strings.LastIndex(pre, "@"); at > end && strings.Contains(pre[:at], ":") {
		end = strings.IndexAny(rest[at:], "/?#")
		if end < 0 {
			end = len(rest)
		} else {
			end += at
		}
	}
	auth, tail := rest[:end], rest[end:]
	if at := strings.LastIndex(auth, "@"); at >= 0 {
		user := auth[:at]
		if c := strings.Index(user, ":"); c >= 0 {
			user = user[:c] + ":***"
		} else {
			user = "***" // a bare token as the user
		}
		auth = user + auth[at:]
	}
	frag := ""
	if i := strings.Index(tail, "#"); i >= 0 {
		tail, frag = tail[:i], "#***"
	}
	query := ""
	if i := strings.Index(tail, "?"); i >= 0 {
		pairs := strings.Split(tail[i+1:], "&")
		for j, p := range pairs {
			if k := strings.Index(p, "="); k >= 0 {
				pairs[j] = p[:k] + "=***"
			} else if p != "" {
				pairs[j] = "***"
			}
		}
		tail, query = tail[:i], "?"+strings.Join(pairs, "&")
	}
	segs := strings.Split(tail, "/")
	for i, seg := range segs {
		if keyLikeSegment(seg) || (i > 0 && seg != "" && keyNameSegment(segs[i-1])) {
			segs[i] = "***"
		}
	}
	return scheme + auth + strings.Join(segs, "/") + query + frag
}

// keyLikeSegment: a long run of key characters with a digit in it. Words
// ("execution-layer") and chain ids ("369") are not.
func keyLikeSegment(seg string) bool {
	if len(seg) < 20 {
		return false
	}
	digit := false
	for i := 0; i < len(seg); i++ {
		switch c := seg[i]; {
		case c >= '0' && c <= '9':
			digit = true
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_', c == '-', c == '.', c == '=', c == '~':
		default:
			return false
		}
	}
	return digit
}

// keyNameSegment: a segment that announces a key in the next one.
func keyNameSegment(seg string) bool {
	switch strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(seg)) {
	case "key", "apikey", "token", "auth", "secret", "accesstoken":
		return true
	}
	return false
}

var urlInText = regexp.MustCompile(`[A-Za-z][A-Za-z0-9+.-]*://[^\s"'<>]+`)

// maskText masks each URL inside free text (a warning, a check's detail).
func maskText(s string) string {
	return urlInText.ReplaceAllStringFunc(s, maskURL)
}
