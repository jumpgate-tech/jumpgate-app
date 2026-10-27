package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"unicode/utf8"
)

// maxPassphrase bounds a passphrase line, from a terminal or from stdin.
const maxPassphrase = 1024

// tty is the controlling terminal: input with echo turned off while a
// passphrase is typed, and output for the prompt.
type tty struct{ in, out *os.File }

func (t *tty) Read(p []byte) (int, error)  { return t.in.Read(p) }
func (t *tty) Write(p []byte) (int, error) { return t.out.Write(p) }
func (t *tty) Close() error {
	err := t.in.Close()
	if t.out != t.in {
		if cerr := t.out.Close(); err == nil {
			err = cerr
		}
	}
	return err
}

// openTTY opens the controlling terminal and returns it with the input's
// file descriptor. A seam for tests.
var openTTY = func() (io.ReadWriteCloser, uintptr, error) {
	if runtime.GOOS == "windows" {
		in, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return nil, 0, err
		}
		out, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0)
		if err != nil {
			in.Close()
			return nil, 0, err
		}
		return &tty{in: in, out: out}, in.Fd(), nil
	}
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, 0, err
	}
	if !termIsTerminal(f.Fd()) {
		f.Close()
		return nil, 0, errors.New("/dev/tty is not a terminal")
	}
	return &tty{in: f, out: f}, f.Fd(), nil
}

// stdinReader is where --stdin reads its one line. A seam for tests.
var stdinReader io.Reader = os.Stdin

// readPassphrase reads a passphrase from the terminal, never echoing it, or
// with fromStdin one line from standard input. Exactly one trailing "\n" or
// "\r\n" is removed and nothing else. With confirm (terminal only) it asks
// twice and refuses a mismatch. An empty passphrase is refused. The caller
// clears the result.
func readPassphrase(prompt string, fromStdin, confirm bool) ([]byte, error) {
	if fromStdin {
		return checkPassphrase(readLine(stdinReader))
	}
	t, fd, err := openTTY()
	if err != nil {
		return nil, fmt.Errorf("no terminal to ask for the passphrase (%v); pipe it in with --stdin", err)
	}
	defer t.Close()
	first, err := checkPassphrase(askTTY(t, fd, prompt))
	if err != nil || !confirm {
		return first, err
	}
	second, err := askTTY(t, fd, "again: ")
	defer clear(second)
	if err != nil {
		clear(first)
		return nil, err
	}
	if subtle.ConstantTimeCompare(first, second) != 1 {
		clear(first)
		return nil, errors.New("passphrases do not match")
	}
	return first, nil
}

func askTTY(t io.ReadWriter, fd uintptr, prompt string) ([]byte, error) {
	if _, err := io.WriteString(t, prompt); err != nil {
		return nil, err
	}
	if real, ok := t.(*tty); ok {
		b, err := termReadPassword(fd)
		io.WriteString(real, "\n")
		if err != nil {
			clear(b)
			return nil, err
		}
		if len(b) > maxPassphrase {
			clear(b)
			return nil, fmt.Errorf("passphrase too long (over %d bytes)", maxPassphrase)
		}
		return b, nil
	}
	return readLine(t)
}

// readLine reads one line a byte at a time, so nothing past it is consumed.
func readLine(r io.Reader) ([]byte, error) {
	buf := make([]byte, 0, maxPassphrase+2) // never reallocated, so no copy is left uncleared
	var c [1]byte
	for {
		n, err := r.Read(c[:])
		if n == 1 {
			if c[0] == '\n' {
				break
			}
			if len(buf) == maxPassphrase+1 { // room for a final '\r'
				clear(buf)
				return nil, fmt.Errorf("passphrase too long (over %d bytes)", maxPassphrase)
			}
			buf = append(buf, c[0])
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			clear(buf)
			return nil, err
		}
	}
	if l := len(buf); l > 0 && buf[l-1] == '\r' {
		buf = buf[:l-1]
	}
	if len(buf) > maxPassphrase {
		clear(buf)
		return nil, fmt.Errorf("passphrase too long (over %d bytes)", maxPassphrase)
	}
	return buf, nil
}

// minNewPassphrase is the fewest characters (runes) a new vault passphrase
// may have. An existing vault unlocks with whatever passphrase it has.
const minNewPassphrase = 12

// readNewPassphrase is readPassphrase for a passphrase being chosen: one
// under minNewPassphrase characters is refused before anything is created.
func readNewPassphrase(prompt string, fromStdin, confirm bool) ([]byte, error) {
	b, err := readPassphrase(prompt, fromStdin, confirm)
	if err != nil {
		return nil, err
	}
	if n := utf8.RuneCount(b); n < minNewPassphrase {
		clear(b)
		return nil, fmt.Errorf("the passphrase has %d characters; a new vault passphrase needs at least %d", n, minNewPassphrase)
	}
	return b, nil
}

func checkPassphrase(b []byte, err error) ([]byte, error) {
	if err != nil {
		clear(b)
		return nil, err
	}
	if len(b) == 0 {
		return nil, errors.New("empty passphrase")
	}
	return b, nil
}
