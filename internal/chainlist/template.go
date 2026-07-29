package chainlist

import (
	"regexp"
	"strings"
)

// placeholderRe matches the feed's shell-style provider slot. isTemplated stays
// deliberately broader (a bare "${" counts), because a malformed placeholder is
// still not a usable endpoint and must not fall through to being probed.
var placeholderRe = regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

// PlaceholderName returns the name inside a URL's ${...} slot, or "" when there
// is none. The name is the identity an operator recognises — it is what they go
// and obtain, what the app stores a key under, and what a rejection names.
func PlaceholderName(raw string) string {
	m := placeholderRe.FindStringSubmatch(raw)
	if m == nil {
		return ""
	}
	return m[1]
}

// Resolve substitutes every placeholder in raw from keys, reporting whether the
// result is usable. An untemplated URL resolves to itself so callers need no
// special case.
//
// A missing OR empty key fails rather than substituting nothing: a URL with an
// empty path segment looks configured and answers nothing, which is worse than
// a rejection that says which key is missing.
func Resolve(raw string, keys map[string]string) (string, bool) {
	if !isTemplated(raw) {
		return raw, true
	}
	out := raw
	for _, m := range placeholderRe.FindAllStringSubmatch(raw, -1) {
		v := keys[m[1]]
		if strings.TrimSpace(v) == "" {
			return "", false
		}
		out = strings.ReplaceAll(out, m[0], v)
	}
	// A leftover "${" means a malformed slot the regex could not name.
	if isTemplated(out) {
		return "", false
	}
	return out, true
}
