package ai

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Patterns for what identifies a box, its peers or its credentials in client
// logs. Order matters: secrets go first so their values are never mistaken
// for addresses, and an enode contains an IP, so enodes go before bare IPs.
var (
	// Secret shapes. Each replaces only the secret, keeping the key or scheme
	// so the excerpt still reads ("password=<secret-1>").
	bearerRE = regexp.MustCompile(`(?i)(\bBearer\s+)([A-Za-z0-9._~+/=-]{8,})`)
	// assignRE covers key=value pairs whose key names a credential, including
	// jumpgate_token= and ?token= in URLs and cookies.
	assignRE = regexp.MustCompile(`(?i)(\b[a-z0-9_-]*(?:token|password|passwd|secret|api[_-]?key)=)([^\s&"',;]+)`)
	// jsonSecretRE covers "password":"value" in structured logs.
	jsonSecretRE = regexp.MustCompile(`(?i)("[a-z0-9_-]*(?:token|password|passwd|secret|api[_-]?key)"\s*:\s*")([^"]*)(")`)
	keyRE        = regexp.MustCompile(`\b(?:sk|gsk|ghp|gho|xox[abp])[-_][A-Za-z0-9_-]{16,}|\bglpat-[A-Za-z0-9_-]{16,}`)
	jwtRE        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]*`)

	enodeRE = regexp.MustCompile(`enode://[0-9a-fA-F]{128}@[^\s"',]+`)
	enrRE   = regexp.MustCompile(`\benr:-[A-Za-z0-9_-]+`)
	peerRE  = regexp.MustCompile(`\b(?:16Uiu2HAm|12D3KooW|Qm)[1-9A-HJ-NP-Za-km-z]{40,}\b`)
	// A bare 128-hex node id, as geth and erigon print for a remote peer.
	nodeIDRE = regexp.MustCompile(`\b[0-9a-fA-F]{128}\b`)
	// Candidates only: net.ParseIP and the boundary checks decide.
	ipv6RE = regexp.MustCompile(`[0-9A-Fa-f.]*:[0-9A-Fa-f:.]*`)
	ipv4RE = regexp.MustCompile(`(?:\d{1,3}\.){3}\d{1,3}`)
)

// Redact replaces IP addresses (v4, v6, v4-mapped, inside multiaddrs and
// bracketed host:port), enode URLs, ENR records, libp2p and node peer ids and
// obvious credentials with placeholders before lines go to a third-party
// provider. The same value gets the same placeholder within one call, so a
// reader can still follow "the same peer" through the excerpt. Redacting
// already-redacted text changes nothing. The input is not modified.
//
// Ethereum 0x addresses and hostnames are deliberately left alone: they are
// public chain data or not reliably distinguishable from prose.
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
	// were. A value that is already a placeholder is left alone, which is
	// what makes the whole function idempotent.
	keepKey := func(re *regexp.Regexp, l string) string {
		return re.ReplaceAllStringFunc(l, func(m string) string {
			g := re.FindStringSubmatch(m)
			if strings.HasPrefix(g[2], "<") {
				return m
			}
			return g[1] + sub("secret", g[2]) + strings.Join(g[3:], "")
		})
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		l = keepKey(bearerRE, l)
		l = keepKey(assignRE, l)
		l = keepKey(jsonSecretRE, l)
		l = whole(keyRE, "secret", l)
		l = whole(jwtRE, "secret", l)
		l = whole(enodeRE, "enode", l)
		l = whole(enrRE, "enr", l)
		l = whole(peerRE, "peer", l)
		l = whole(nodeIDRE, "node", l)
		l = replaceBounded(ipv6RE, l, trimIPv6, func(v string) string { return sub("ip", v) })
		l = replaceBounded(ipv4RE, l, trimIPv4, func(v string) string { return sub("ip", v) })
		out[i] = l
	}
	return out
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
	for start < end {
		v := s[start:end]
		switch {
		case net.ParseIP(v) != nil && strings.Count(v, ":") >= 2:
			if start > 0 && isWordByte(s[start-1]) || end < len(s) && isWordByte(s[end]) {
				return 0, 0, false
			}
			if v != "::" && !strings.ContainsAny(v, "0123456789") {
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
