//go:build windows

package executor

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

// pipeConn is a named pipe opened for overlapped I/O, so every read and write
// can be cancelled: an os.File on a pipe is a synchronous handle, which a
// wedged server can block for ever with nothing able to interrupt it.
type pipeConn struct {
	h windows.Handle

	mu       sync.Mutex
	deadline time.Time
	closed   bool
}

// openPipe opens the named pipe for overlapped reading and writing.
func openPipe(name string) (*pipeConn, error) {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil,
		windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED|windows.SECURITY_SQOS_PRESENT|windows.SECURITY_ANONYMOUS, 0)
	if err != nil {
		return nil, err
	}
	return &pipeConn{h: h}, nil
}

// SetDeadline bounds every later read and write, as net.Conn's does.
func (p *pipeConn) SetDeadline(t time.Time) error {
	p.mu.Lock()
	p.deadline = t
	p.mu.Unlock()
	return nil
}

func (p *pipeConn) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	n, err := p.do(func(ov *windows.Overlapped, n *uint32) error {
		return windows.ReadFile(p.h, b, n, ov)
	})
	if err == windows.ERROR_BROKEN_PIPE || (err == nil && n == 0) {
		return n, io.EOF
	}
	return n, err
}

func (p *pipeConn) Write(b []byte) (int, error) {
	total := 0
	for total < len(b) {
		chunk := b[total:]
		n, err := p.do(func(ov *windows.Overlapped, n *uint32) error {
			return windows.WriteFile(p.h, chunk, n, ov)
		})
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Close cancels any I/O in flight, then closes the handle.
func (p *pipeConn) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()
	_ = windows.CancelIoEx(p.h, nil)
	return windows.CloseHandle(p.h)
}

var errPipeTimeout = os.ErrDeadlineExceeded

// do runs one overlapped operation and waits for it, up to the deadline; on
// timeout the operation is cancelled.
func (p *pipeConn) do(start func(ov *windows.Overlapped, n *uint32) error) (int, error) {
	p.mu.Lock()
	deadline, closed := p.deadline, p.closed
	p.mu.Unlock()
	if closed {
		return 0, os.ErrClosed
	}
	ev, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(ev)
	ov := &windows.Overlapped{HEvent: ev}
	var n uint32
	err = start(ov, &n)
	if err == nil {
		return int(n), nil // completed at once
	}
	if err != windows.ERROR_IO_PENDING {
		return int(n), err
	}
	wait := uint32(windows.INFINITE)
	if !deadline.IsZero() {
		d := time.Until(deadline)
		if d < 0 {
			d = 0
		}
		wait = uint32(d / time.Millisecond)
	}
	ev2, err := windows.WaitForSingleObject(ev, wait)
	if err != nil {
		return 0, err
	}
	if ev2 == uint32(windows.WAIT_TIMEOUT) {
		_ = windows.CancelIoEx(p.h, ov)
		// Wait for the cancellation to land: ov and the buffer must outlive it.
		_ = windows.GetOverlappedResult(p.h, ov, &n, true)
		return int(n), errPipeTimeout
	}
	if err := windows.GetOverlappedResult(p.h, ov, &n, false); err != nil {
		if errors.Is(err, windows.ERROR_OPERATION_ABORTED) {
			return int(n), os.ErrClosed
		}
		return int(n), err
	}
	return int(n), nil
}
