// Package metrics turns an eRPC gateway's own Prometheus counters into
// per-network, per-upstream traffic share: given what the routing config
// says an upstream is FOR (its preferred tier), does the traffic it is
// actually carrying match?
//
// The package is split into three layers that mirror the pipeline an
// operator cares about:
//
//   - ParseText: a minimal Prometheus text-exposition parser. Standard
//     library only — this is maybe 100 lines of scanning, and pulling in
//     client_golang's parser for it would be a dependency to save a
//     weekend.
//   - FromSamples: picks the handful of eRPC counters that describe
//     client-facing traffic (and only those — see its doc comment for why
//     erpc_upstream_request_total is explicitly NOT one of them) and sums
//     them into Traffic.
//   - Shares: compares that traffic against Intent, what the erpc.yaml
//     routing config (see internal/catalog.GatewayConfig) says an upstream
//     should carry, and flags the upstreams that have drifted.
//
// This package is pure: bytes in, values out. It does no I/O, opens no
// connections and never touches an executor.Executor. Fetching the counters
// from a live gateway is another package's job — this one only has to be
// right about what the numbers mean once they arrive.
package metrics

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Sample is one line of Prometheus text-exposition format, decoded: a
// metric name, its labels (nil when the line had none), and its value.
//
// Labels is a plain map rather than a sorted slice of pairs because every
// consumer in this package looks values up by key ("what is this sample's
// network?") and never needs label order — order matters for the ORIGINAL
// text (Prometheus's own tooling emits it consistently per family for
// diffability), but not for anything downstream of ParseText.
type Sample struct {
	Name   string
	Labels map[string]string
	Value  float64
}

// ParseText reads Prometheus text-exposition format and returns every
// metric sample in it, in file order.
//
// It implements only the subset of the format eRPC's own /metrics endpoint
// emits: HELP/TYPE metadata lines and blank lines are skipped, and each
// remaining line is either `name{label="value",...} number` or a bare
// `name number`. It does not implement exemplars, the OpenMetrics `# EOF`
// trailer, or per-sample timestamps (eRPC does not emit them; a trailing
// token after the value is tolerated and ignored rather than rejected, in
// case a future eRPC version adds one).
//
// A line ParseText cannot make sense of is a hard error naming the 1-based
// line number and the offending text, not a skip. A parser that silently
// dropped what it did not understand would turn a malformed line into a
// quietly wrong traffic share instead of a loud failure — and a wrong share
// bar is worse than no share bar, because it looks like a working one.
func ParseText(r io.Reader) ([]Sample, error) {
	var samples []Sample

	sc := bufio.NewScanner(r)
	// eRPC's real dump has label sets well under 1KB, but the default 64KB
	// token limit is one histogram bucket away from being a real ceiling on
	// a gateway with many networks tagged onto one line. Growing the buffer
	// costs nothing on the common case and avoids a truncation bug on a
	// busier gateway than the one that produced testdata/erpc.txt.
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)

	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		s, err := parseSampleLine(trimmed)
		if err != nil {
			return nil, fmt.Errorf("metrics: line %d: %q: %w", lineNo, line, err)
		}
		samples = append(samples, s)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("metrics: reading input: %w", err)
	}
	return samples, nil
}

// parseSampleLine parses one non-blank, non-comment line already trimmed of
// surrounding whitespace.
func parseSampleLine(line string) (Sample, error) {
	i := 0
	n := len(line)

	nameStart := i
	for i < n && line[i] != '{' && !isSpace(line[i]) {
		i++
	}
	if i == nameStart {
		return Sample{}, errors.New("no metric name")
	}
	name := line[nameStart:i]

	var labels map[string]string
	if i < n && line[i] == '{' {
		var err error
		labels, i, err = parseLabels(line, i)
		if err != nil {
			return Sample{}, err
		}
	}

	for i < n && isSpace(line[i]) {
		i++
	}
	if i >= n {
		return Sample{}, errors.New("no value after metric name")
	}

	valueStart := i
	for i < n && !isSpace(line[i]) {
		i++
	}
	valueTok := line[valueStart:i]

	// strconv.ParseFloat already accepts "NaN" and signed "Inf"/"Infinity"
	// (case-insensitively) per its documented grammar, so eRPC's
	// zero-division sentinels (+Inf, -Inf, NaN) and exponent form
	// (1.7e+09) need no special-casing here.
	value, err := strconv.ParseFloat(valueTok, 64)
	if err != nil {
		return Sample{}, fmt.Errorf("invalid value %q: %w", valueTok, err)
	}

	// Anything left on the line (an OpenMetrics timestamp, in the format
	// this package does not otherwise implement) is intentionally ignored:
	// see the ParseText doc comment.

	return Sample{Name: name, Labels: labels, Value: value}, nil
}

// parseLabels parses a `{...}` label block starting at line[open] == '{',
// and returns the decoded labels and the index just past the closing '}'.
//
// This is a hand-written scanner, not a regexp, because label VALUES are
// arbitrary quoted strings that may themselves contain commas, spaces, '='
// and '}' — exactly the characters a naive split on those characters would
// use as delimiters. eRPC's own dump proves the point: HELP text quoted
// into a label would break that approach immediately, and even without
// HELP text in labels, vendor strings like "unknown-pulsechain.com" are one
// comma away from being real ammunition for it. The only correct way to
// find the end of a quoted value is to walk it byte by byte and track
// whether an unescaped '"' has been seen.
func parseLabels(line string, open int) (map[string]string, int, error) {
	n := len(line)
	i := open + 1 // consume '{'

	labels := make(map[string]string)
	for {
		for i < n && isSpace(line[i]) {
			i++
		}
		if i < n && line[i] == '}' {
			return labels, i + 1, nil
		}
		if i >= n {
			return nil, 0, errors.New("unterminated label block: missing '}'")
		}

		keyStart := i
		for i < n && line[i] != '=' {
			i++
		}
		if i >= n {
			return nil, 0, errors.New("unterminated label: missing '='")
		}
		key := line[keyStart:i]
		i++ // consume '='

		if i >= n || line[i] != '"' {
			return nil, 0, fmt.Errorf("label %q: value must start with '\"'", key)
		}
		i++ // consume opening quote

		var val strings.Builder
		closed := false
		for i < n {
			c := line[i]
			if c == '\\' {
				if i+1 >= n {
					return nil, 0, fmt.Errorf("label %q: dangling '\\' at end of line", key)
				}
				switch line[i+1] {
				case '"':
					val.WriteByte('"')
				case '\\':
					val.WriteByte('\\')
				case 'n':
					val.WriteByte('\n')
				default:
					return nil, 0, fmt.Errorf("label %q: unknown escape '\\%c'", key, line[i+1])
				}
				i += 2
				continue
			}
			if c == '"' {
				closed = true
				i++
				break
			}
			val.WriteByte(c)
			i++
		}
		if !closed {
			return nil, 0, fmt.Errorf("label %q: unterminated quoted value", key)
		}
		labels[key] = val.String()

		for i < n && isSpace(line[i]) {
			i++
		}
		if i >= n {
			return nil, 0, errors.New("unterminated label block: missing '}'")
		}
		switch line[i] {
		case ',':
			i++
			continue
		case '}':
			return labels, i + 1, nil
		default:
			return nil, 0, fmt.Errorf("label %q: expected ',' or '}', found %q", key, string(line[i]))
		}
	}
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' }
