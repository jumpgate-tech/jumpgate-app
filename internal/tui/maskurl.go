package tui

import (
	"regexp"
	"strings"
)

// maskURL hides what a URL may carry as a secret, and fails closed: when it
// is unsure it masks, so it over-masks on purpose.
//
//   - userinfo (everything up to the last "@" before the query) becomes
//     user:*** (a bare token becomes ***);
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
func maskURL(s string) string { return maskURLWith(s, false) }

var schemeRE = regexp.MustCompile(`^(?:[A-Za-z][A-Za-z0-9+.-]*:)?//`)

func maskURLWith(s string, fullUser bool) string {
	scheme, rest := "", s
	if loc := schemeRE.FindString(s); loc != "" {
		scheme, rest = loc, s[len(loc):]
	}
	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		end = len(rest)
	}
	// A password may hold "/" and "@": the userinfo runs to the last "@"
	// before the query when it comes before the first "/" or looks like
	// user:password.
	pre := rest
	if q := strings.IndexAny(rest, "?#"); q >= 0 {
		pre = rest[:q]
	}
	if at := strings.LastIndex(pre, "@"); at > end && strings.Contains(pre[:at], ":") {
		end = len(rest)
		if i := strings.IndexAny(rest[at:], "/?#"); i >= 0 {
			end = at + i
		}
	}
	auth, tail := rest[:end], rest[end:]
	if at := strings.LastIndex(auth, "@"); at >= 0 {
		user := auth[:at]
		if c := strings.Index(user, ":"); c >= 0 && !fullUser {
			user = user[:c] + ":***"
		} else {
			user = "***"
		}
		auth = user + auth[at:]
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

var (
	wordRE    = regexp.MustCompile(`^[a-z]{1,8}$`)
	versionRE = regexp.MustCompile(`^v[0-9]{1,2}$`)
	digitsRE  = regexp.MustCompile(`^[0-9]{1,10}$`)
	keyNameRE = regexp.MustCompile(`^[A-Za-z_-]{1,16}$`)
)

// plainSegment: empty, or on the small allowlist described on maskURL.
func plainSegment(seg string) bool {
	return seg == "" || wordRE.MatchString(seg) || versionRE.MatchString(seg) || digitsRE.MatchString(seg)
}

// keyName: a segment that announces a key in the next one (/key/short).
func keyName(seg string) bool {
	switch strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(seg)) {
	case "key", "apikey", "token", "auth", "secret", "accesstoken":
		return true
	}
	return false
}

func plainKey(k string) bool { return keyNameRE.MatchString(k) }

var tokenRE = regexp.MustCompile(`\s+|\S+`)

// maskText masks the URL-like parts of free text (a warning, a check's
// detail or fix), fail closed. It reads whitespace-separated tokens: one that
// holds "://", "@" or "/" is masked as a URL with the userinfo hidden
// entirely. A "scheme://" token that ends before any "/" (a space inside the
// userinfo) is joined with the following tokens up to the first one holding
// "@". Plain words and numbers are left alone.
func maskText(s string) string {
	toks := tokenRE.FindAllString(s, -1)
	var out strings.Builder
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		if strings.TrimSpace(t) == "" {
			out.WriteString(t)
			continue
		}
		if k := strings.Index(t, "://"); k >= 0 && !strings.ContainsAny(t[k+3:], "/?#@") {
			for j := i + 1; j < len(toks); j++ {
				if strings.Contains(toks[j], "@") {
					t = strings.Join(toks[i:j+1], "")
					i = j
					break
				}
			}
		}
		if strings.ContainsAny(t, "@/") || strings.Contains(t, "://") {
			t = maskURLWith(t, true)
		}
		out.WriteString(t)
	}
	return out.String()
}
