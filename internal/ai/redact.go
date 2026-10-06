package ai

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Patterns for what identifies a box, its peers or its credentials in client
// logs. Order matters: secrets go first so their values are never mistaken
// for addresses, and an enode contains an IP, so enodes go before bare IPs.
// quotedValue matches a quoted string with backslash escapes, closed or
// running to the end of the line.
const quotedValue = `"(?:[^"\\]|\\(?s:.))*\\?(?:"|$)|'(?:[^'\\]|\\(?s:.))*\\?(?:'|$)`

var (
	// Secret shapes. Each replaces only the secret, keeping the key or scheme
	// so the excerpt still reads ("password=<secret-1>").
	bearerRE = regexp.MustCompile(`(?i)(\bBearer\s+)([A-Za-z0-9._~+/=-]{8,})`)
	// authRE covers "Authorization: <scheme> <credential>" for any scheme
	// (Basic, Token, Digest, ...), and a bare credential with no scheme.
	authRE = regexp.MustCompile(`(?i)(\bAuthorization["']?\s*[:=]\s*"?(?:[A-Za-z][A-Za-z0-9_-]*[ \t]+)?)([^\s",;]+)`)
	// keyedRE covers key=value, key: value, key = value and "key":"value"
	// where the key names a credential (jumpgate_token, X-Api-Key,
	// access_token, private_key, ...), whatever the value looks like. A value
	// that opens with a quote runs to the matching unescaped quote, spaces
	// and all, or to the end of the line when it is never closed.
	keyedRE = regexp.MustCompile(`(?i)(\b[a-z0-9_-]*(?:token|password|passwd|secret|api[_-]?key|private[_-]?key|priv[_-]?key|seed)["']?\s*[:=]\s*)(` + quotedValue + `|[^\s&"',;]+)`)
	// lineValueRE covers values that are not one word: a cookie header and a
	// recovery phrase run to the end of the line. A lone \r does not end the
	// line here (it is a carriage return, not a new log record), so text
	// after one is still part of the value; only \n ends it.
	lineValueRE = regexp.MustCompile(`(?i)(\b(?:set-)?cookie["']?\s*[:=]\s*|\b(?:mnemonic|seed[_ -]?phrase|recovery[_ -]?phrase)["']?\s*[:=]\s*)(` + quotedValue + `|[^\n]+)`)
	// cliFlagRE covers "--password hunter2" (a space, not "="). A value that
	// starts with "-" is the next flag, not a secret.
	cliFlagRE = regexp.MustCompile(`(?i)(--[a-z0-9_-]*(?:token|password|passwd|secret|api[_-]?key)[ \t]+)(` + quotedValue + `|[^\s-]\S*)`)
	// urlUserRE covers scheme://user:password@host (and ://:password@host).
	// The password may hold "/" and "@"; the userinfo ends at the last "@"
	// before any query or fragment.
	urlUserRE = regexp.MustCompile(`(\b[A-Za-z][A-Za-z0-9+.-]*://)([^\s/@:<>"']*:[^\s<>"'?#]*)(@)`)
	urlRE     = regexp.MustCompile(`\b(?:https?|wss?)://[^\s"'<>]+`)
	// keyRE matches API keys. Formats with a distinctive prefix match on the
	// prefix alone, even glued to a word character ("tokensk-ant-..."), since
	// a leading \b would let any preceding letter hide the key. The generic
	// sk-/gsk_ forms keep the boundary unless the body is long and plain, so
	// hyphenated words such as "disk-usage-statistics" are left alone.
	keyRE = regexp.MustCompile(`(?i)(?:sk-(?:ant|proj|or)-|gsk_|gh[pousr]_|xox[abp]-|glpat-)[A-Za-z0-9_-]{16,}` +
		`|github_pat_[A-Za-z0-9_]{20,}|AIza[0-9A-Za-z_-]{35}|sk-[A-Za-z0-9]{20,}` +
		`|\b(?:sk|gsk|ghp|gho|xox[abp])[-_][A-Za-z0-9_-]{16,}`)
	jwtRE = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]*`)
	// sessionRE is a bare 32-hex run: jumpgate's session token. The word
	// boundaries keep 0x-prefixed values and 40/64-hex addresses and hashes.
	sessionRE = regexp.MustCompile(`\b[0-9a-fA-F]{32}\b`)

	enodeRE = regexp.MustCompile(`enode://[0-9a-fA-F]{128}@[^\s"',]+`)
	enrRE   = regexp.MustCompile(`\benr:-[A-Za-z0-9_-]+`)
	peerRE  = regexp.MustCompile(`\b(?:16Uiu2HAm|12D3KooW|Qm)[1-9A-HJ-NP-Za-km-z]{40,}\b`)
	// A bare 128-hex node id, as geth and erigon print for a remote peer.
	nodeIDRE = regexp.MustCompile(`\b[0-9a-fA-F]{128}\b`)
	// Candidates only: net.ParseIP and the boundary checks decide.
	ipv6RE = regexp.MustCompile(`[0-9A-Fa-f.]*:[0-9A-Fa-f:.]*(?:%[A-Za-z0-9_-]+)?`)
	ipv4RE = regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)
)

// Redact prepares log lines for a third-party AI provider. The same value
// gets the same placeholder within one call, so a reader can still follow
// "the same peer" through the excerpt. Redacting already-redacted text changes
// nothing, and the input is not modified.
//
// Redacted (placeholder kind in parentheses):
//   - IPv4 and IPv6 addresses, including v4-mapped, bracketed host:port,
//     zone ids (fe80::1%eth0) and addresses inside /ip4 and /ip6 multiaddrs (ip)
//   - enode:// URLs (enode), bare 128-hex node ids (node), ENR records (enr)
//     and libp2p peer ids (peer)
//   - credentials (secret): Authorization headers of any scheme, Bearer
//     tokens, key=value, key: value and "key":"value" pairs whose key names a
//     token, password, secret, api key, private key or seed (a quoted value
//     is redacted to its closing quote), --password style CLI flags, Cookie
//     and Set-Cookie values, mnemonic phrases, URL userinfo (user:pass@), sk-,
//     gsk_, ghp_, github_pat_, glpat- and Google AIza keys, JWTs, URL path
//     segments shaped like API keys (Infura and Alchemy /v3/<key>), and bare
//     32-hex runs (jumpgate session tokens)
//
// Not redacted: hostnames and domain names (they cannot be told from prose
// reliably), Ethereum 0x addresses and 0x or 64-hex block and transaction
// hashes (public chain data), ports on their own, timestamps, and version
// strings such as v1.2.3.4. A bare credential with no key name next to it
// and no known prefix is not recognised, and neither is a bare 40-hex run:
// that is a git SHA, valuable when diagnosing a build, and a 40-hex value
// next to a credential keyword is already caught by the keyword.
//
// Limits: a line is cut to maxLineBytes with a "...[truncated]" marker, at
// most maxLines lines and about maxTotalBytes of output are kept, and the
// rest is replaced by one "[N more lines omitted]" line, so the result can
// have fewer lines than the input.
func Redact(lines []string) []string {
	seen := map[string]string{}
	count := map[string]int{}
	sub := func(kind, v string) string {
		key := kind + "\x00" + v
		if p, ok := seen[key]; ok {
			return p
		}
		count[kind]++
		p := fmt.Sprintf("<%s-%d>", kind, count[kind])
		seen[key] = p
		return p
	}
	whole := func(re *regexp.Regexp, kind, l string) string {
		return re.ReplaceAllStringFunc(l, func(v string) string { return sub(kind, v) })
	}
	// keepKey replaces group 2 of a pattern, leaving groups 1 (and 3) as they
	// were. A value that is already a secret placeholder is left alone, which
	// is what makes the whole function idempotent; anything else that merely
	// looks like one (password="<hunter2") is still redacted. A quoted value
	// keeps its quotes around the placeholder.
	keepKey := func(re *regexp.Regexp, l string) string {
		return re.ReplaceAllStringFunc(l, func(m string) string {
			g := re.FindStringSubmatch(m)
			val, tail := g[2], strings.Join(g[3:], "")
			quote, closing := "", ""
			if q := val[0]; q == '"' || q == '\'' {
				quote = string(q)
				val = val[1:]
				if closed(val, q) {
					val, closing = val[:len(val)-1], quote
				}
			}
			if val == "" || secretPlaceholderRE.MatchString(val) {
				return m
			}
			return g[1] + quote + sub("secret", val) + closing + tail
		})
	}
	// Bound the work: the loop below is linear in what it reads, so reading
	// at most maxLines lines of maxLineBytes and about maxTotalBytes of
	// output keeps a hostile log from stalling a request.
	limit := maxLines
	if len(lines) > maxLines {
		limit = maxLines - 1
	}
	out := make([]string, 0, min(len(lines), maxLines))
	total, work := 0, 0
	for i, l := range lines {
		if i >= limit {
			break
		}
		// Redact the whole line and only then clip it, so a secret that
		// straddles the 4 KB cut is replaced, not left as a prefix. The hard
		// bound keeps the redaction itself linear.
		l = hardCut(l)
		// An omission marker from an earlier pass is exempt from the budgets,
		// so redacting redacted text keeps it as it was.
		if work+len(l) > maxWorkBytes && !omittedRE.MatchString(l) {
			break
		}
		raw := len(l)
		// Go's regexp is slow next to a substring search, so each pattern
		// runs only on lines that contain something it could match.
		lower := strings.ToLower(l)
		if strings.Contains(lower, "authorization") {
			l = keepKey(authRE, l)
		}
		if strings.Contains(lower, "bearer") {
			l = keepKey(bearerRE, l)
		}
		if strings.Contains(lower, "cookie") || containsAny(lower, "mnemonic", "phrase") {
			l = keepKey(lineValueRE, l)
		}
		if strings.Contains(l, "--") {
			l = keepKey(cliFlagRE, l)
		}
		if containsAny(lower, "token", "password", "passwd", "secret", "key", "seed") {
			l = keepKey(keyedRE, l)
		}
		if strings.Contains(l, "://") {
			l = keepKey(urlUserRE, l)
			l = redactURLPaths(l, func(v string) string { return sub("secret", v) })
		}
		if containsAny(lower, "sk-", "sk_", "gsk_", "ghp_", "gho_", "ghs_", "ghu_", "ghr_", "xox", "glpat-", "github_pat_", "aiza") {
			l = whole(keyRE, "secret", l)
		}
		if strings.Contains(l, "eyJ") {
			l = whole(jwtRE, "secret", l)
		}
		if strings.Contains(l, "enode://") {
			l = whole(enodeRE, "enode", l)
		}
		if strings.Contains(l, "enr:-") {
			l = whole(enrRE, "enr", l)
		}
		if containsAny(l, "16Uiu2HAm", "12D3KooW", "Qm") {
			l = whole(peerRE, "peer", l)
		}
		l = whole(nodeIDRE, "node", l)
		l = whole(sessionRE, "secret", l)
		l = replaceBounded(ipv6RE, l, trimIPv6, func(v string) string { return sub("ip", v) })
		l = replaceBounded(ipv4RE, l, trimIPv4, func(v string) string { return sub("ip", v) })
		l = clipLine(l)
		if total+len(l) > maxTotalBytes && !omittedRE.MatchString(l) {
			break
		}
		total += len(l)
		work += max(raw, len(l))
		out = append(out, l)
	}
	if dropped := len(lines) - len(out); dropped > 0 {
		out = append(out, fmt.Sprintf("[%d more lines omitted]", dropped))
	}
	return out
}

// Limits on what one Redact call will read. Explain excerpts are a few
// hundred lines, so these only bite on hostile or pathological input.
const (
	maxLineBytes  = 4096
	maxLines      = 2000
	maxTotalBytes = 128 << 10
	// maxWorkBytes bounds what redaction reads of all lines together.
	maxWorkBytes = 128 << 10
	truncMarker  = " ...[truncated]"
)

// maxRawLineBytes bounds what redaction reads of one line. It is a variable
// only so a test can walk a secret across the cut without 32 KB of filler.
var maxRawLineBytes = 32 << 10

var (
	omittedRE           = regexp.MustCompile(`^\[\d+ more lines omitted\]$`)
	secretPlaceholderRE = regexp.MustCompile(`^<secret-\d+>$`)
)

// closed reports whether s ends in an unescaped q: an even number of
// backslashes (including none) before it.
func closed(s string, q byte) bool {
	if s == "" || s[len(s)-1] != q {
		return false
	}
	n := 0
	for i := len(s) - 2; i >= 0 && s[i] == '\\'; i-- {
		n++
	}
	return n%2 == 0
}

func isTokenByte(c byte) bool {
	return isWordByte(c) || c == '-' || c == '.' || c == '~' || c == '+' || c == '/'
}

// hardCut bounds the text redaction reads of one line. The cut can fall in
// the middle of a secret whose tail is then gone and can no longer be
// recognised, so a run of eight or more token characters touching the cut is
// dropped as a possible key prefix.
func hardCut(l string) string {
	if len(l) <= maxRawLineBytes {
		return l
	}
	cut := maxRawLineBytes - len(truncMarker)
	for cut > 0 && !utf8.RuneStart(l[cut]) {
		cut--
	}
	run := cut
	for run > 0 && isTokenByte(l[run-1]) {
		run--
	}
	if cut-run >= 8 {
		cut = run
	}
	return l[:cut] + truncMarker
}

// clipLine cuts a line to maxLineBytes, marker included, at a rune boundary
// and, when one is near, at a space so a half-cut token is not left behind.
func clipLine(l string) string {
	if len(l) <= maxLineBytes {
		return l
	}
	cut := maxLineBytes - len(truncMarker)
	for cut > 0 && !utf8.RuneStart(l[cut]) {
		cut--
	}
	if sp := strings.LastIndexByte(l[max(cut-64, 0):cut], ' '); sp >= 0 {
		cut = max(cut-64, 0) + sp
	}
	return l[:cut] + truncMarker
}

// redactURLPaths replaces a URL path segment that has the shape of an API key
// (32 or more URL-safe characters with a digit, as in Infura's /v3/<key> and
// Alchemy's /v2/<key>), except 0x-prefixed values and 64-hex hashes.
func redactURLPaths(l string, repl func(string) string) string {
	return urlRE.ReplaceAllStringFunc(l, func(u string) string {
		start := strings.Index(u, "://") + 3
		path := strings.IndexByte(u[start:], '/')
		if path < 0 {
			return u
		}
		path += start
		end := len(u)
		if q := strings.IndexAny(u[path:], "?#"); q >= 0 {
			end = path + q
		}
		segs := strings.Split(u[path:end], "/")
		for i, seg := range segs {
			if looksLikeKey(seg) {
				segs[i] = repl(seg)
			}
		}
		return u[:path] + strings.Join(segs, "/") + u[end:]
	})
}

func looksLikeKey(seg string) bool {
	if len(seg) < 32 || strings.HasPrefix(seg, "0x") {
		return false
	}
	digit, hex := false, true
	for i := 0; i < len(seg); i++ {
		c := seg[i]
		switch {
		case isDigit(c):
			digit = true
		case c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F':
		case c >= 'g' && c <= 'z' || c >= 'G' && c <= 'Z' || c == '_' || c == '-':
			hex = false
		default:
			return false
		}
	}
	return digit && !(hex && len(seg) == 64)
}

// replaceBounded replaces the regexp matches that fit(...) accepts as a whole
// token. fit may shrink the match (returning the new start and end offsets)
// or reject it. Go's regexp has no lookaround, so the boundary logic that
// keeps "v1.2.3.4" and "reth::cli" intact lives here.
func replaceBounded(re *regexp.Regexp, s string, fit func(s string, start, end int) (int, int, bool), repl func(string) string) string {
	var b strings.Builder
	last := 0
	for _, m := range re.FindAllStringIndex(s, -1) {
		start, end, ok := fit(s, m[0], m[1])
		if !ok || start < last {
			continue
		}
		b.WriteString(s[last:start])
		b.WriteString(repl(s[start:end]))
		last = end
	}
	b.WriteString(s[last:])
	return b.String()
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// trimIPv4 accepts a dotted quad that is not part of a longer number, a word
// or a version string such as 1.2.3.4.5.
func trimIPv4(s string, start, end int) (int, int, bool) {
	if net.ParseIP(s[start:end]) == nil {
		return 0, 0, false
	}
	if start > 0 && (isWordByte(s[start-1]) || s[start-1] == '.' && start > 1 && isDigit(s[start-2])) {
		return 0, 0, false
	}
	if end < len(s) && (isWordByte(s[end]) || s[end] == '.' && end+1 < len(s) && isDigit(s[end+1])) {
		return 0, 0, false
	}
	return start, end, true
}

// trimIPv6 accepts a candidate that parses as an address once sentence
// punctuation is trimmed, and that is not a Rust path (reth::cli), a clock
// time or a word that merely looks hexadecimal.
func trimIPv6(s string, start, end int) (int, int, bool) {
	// The longest address is under 50 bytes plus a zone id; a longer run is
	// garbage, and refusing it keeps the trimming loop below cheap.
	if end-start > 100 {
		return 0, 0, false
	}
	for start < end {
		v := s[start:end]
		addr := v
		if z := strings.IndexByte(v, '%'); z >= 0 {
			addr = v[:z]
		}
		switch {
		case net.ParseIP(addr) != nil && strings.Count(addr, ":") >= 2:
			if start > 0 && isWordByte(s[start-1]) || end < len(s) && isWordByte(s[end]) {
				return 0, 0, false
			}
			if addr != "::" && !strings.ContainsAny(addr, "0123456789") {
				return 0, 0, false
			}
			return start, end, true
		case strings.HasSuffix(v, ".") || strings.HasSuffix(v, ":") && !strings.HasSuffix(v, "::"):
			end--
		case strings.HasPrefix(v, ":") && !strings.HasPrefix(v, "::"):
			// "ip:2001:db8::1": the first colon is a label separator.
			start++
		default:
			return 0, 0, false
		}
	}
	return 0, 0, false
}

// Local reports a provider that runs on the operator's own machine.
func Local(provider string) bool { return provider == "ollama" }

// Disclosure is the one line both front ends show next to the provider.
func Disclosure(provider string) string {
	switch provider {
	case "":
		return ""
	case "ollama":
		return "Explain sends log lines to your Ollama unredacted; nothing leaves this machine unless Ollama runs elsewhere."
	}
	name := map[string]string{"gemini": "Gemini", "groq": "Groq"}[provider]
	if name == "" {
		name = provider
	}
	return "Explain sends the selected log lines to " + name + " after removing IP addresses, enode and ENR records, peer IDs and obvious secrets."
}
