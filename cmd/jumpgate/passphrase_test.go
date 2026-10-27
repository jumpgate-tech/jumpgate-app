package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// fakeTTY answers prompts with lines and records what was written to it.
type fakeTTY struct {
	r   *bufio.Reader
	out *bytes.Buffer
}

func (f fakeTTY) Read(p []byte) (int, error)  { return f.r.Read(p) }
func (f fakeTTY) Write(p []byte) (int, error) { return f.out.Write(p) }
func (f fakeTTY) Close() error                { return nil }

// useFakeTTY points openTTY at a terminal that answers with lines, in order,
// across every open. It returns what was written to the terminal and how
// many times it was opened.
func useFakeTTY(t *testing.T, lines ...string) (written *bytes.Buffer, opens *int) {
	t.Helper()
	r := bufio.NewReader(strings.NewReader(strings.Join(lines, "\n") + "\n"))
	out := &bytes.Buffer{}
	n := 0
	old := openTTY
	openTTY = func() (io.ReadWriteCloser, uintptr, error) {
		n++
		return fakeTTY{r: r, out: out}, 0, nil
	}
	t.Cleanup(func() { openTTY = old })
	return out, &n
}

func TestReadPassphraseFromStdin(t *testing.T) {
	old := stdinReader
	t.Cleanup(func() { stdinReader = old })
	for _, c := range []struct{ in, want, err string }{
		{"pw\n", "pw", ""},
		{"pw\r\n", "pw", ""},
		{" p w \n", " p w ", ""},
		{"pw", "pw", ""},
		{"pw\nsecond\n", "pw", ""},
		{"\n", "", "empty passphrase"},
		{"", "", "empty passphrase"},
		{strings.Repeat("a", 1025) + "\n", "", "too long"},
	} {
		stdinReader = strings.NewReader(c.in)
		got, err := readPassphrase("passphrase: ", true, false)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%q: err %v, want %q", c.in, err, c.err)
			}
			if got != nil {
				t.Errorf("%q: returned %q with an error", c.in, got)
			}
			continue
		}
		if err != nil || string(got) != c.want {
			t.Errorf("%q: got %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestReadPassphraseFromTTY(t *testing.T) {
	written, _ := useFakeTTY(t, "secret", "secret")
	got, err := readPassphrase("new passphrase: ", false, true)
	if err != nil || string(got) != "secret" {
		t.Fatalf("got %q, %v", got, err)
	}
	if !strings.Contains(written.String(), "new passphrase: ") {
		t.Fatalf("prompt not shown: %q", written.String())
	}
	if strings.Contains(written.String(), "secret") {
		t.Fatal("the passphrase was echoed to the terminal")
	}
}

func TestReadPassphraseConfirmMismatch(t *testing.T) {
	useFakeTTY(t, "a", "b")
	got, err := readPassphrase("passphrase: ", false, true)
	if err == nil || !strings.Contains(err.Error(), "passphrases do not match") {
		t.Fatalf("err %v", err)
	}
	if got != nil {
		t.Fatalf("returned %q", got)
	}
}

func TestReadPassphraseNeedsATerminal(t *testing.T) {
	old := openTTY
	openTTY = func() (io.ReadWriteCloser, uintptr, error) { return nil, 0, errors.New("no tty") }
	t.Cleanup(func() { openTTY = old })
	_, err := readPassphrase("passphrase: ", false, false)
	if err == nil || !strings.Contains(err.Error(), "--stdin") {
		t.Fatalf("err %v, want one naming --stdin", err)
	}
}
