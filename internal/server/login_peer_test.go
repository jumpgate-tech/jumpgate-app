package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/valve-tech/jumpgate/internal/testutil"
)

// procRow is one /proc/net/tcp{,6} line for a socket local -> remote owned
// by uid, in this machine's byte order.
func procRow(v6 bool, local, remote netip.AddrPort, uid int) string {
	return fmt.Sprintf("   0: %s %s 01 00000000:00000000 00:00000000 00000000 %5d        0 1 1\n",
		procAddr(local, v6), procAddr(remote, v6), uid)
}

const procHeader = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

func procTableOf(v6 bool, rows ...string) procTable {
	return procTable{data: []byte(procHeader + strings.Join(rows, "")), v6: v6}
}

// reads returns a table reader that serves each snapshot in turn (the last
// one repeatedly) and counts the reads.
func reads(snaps ...[]procTable) (func() []procTable, *int) {
	n := new(int)
	return func() []procTable {
		i := *n
		*n++
		if i >= len(snaps) {
			i = len(snaps) - 1
		}
		return snaps[i]
	}, n
}

func TestPeerFromTables(t *testing.T) {
	v4c, v4s := netip.MustParseAddrPort("127.0.0.1:54321"), netip.MustParseAddrPort("127.0.0.1:8791")
	v6c, v6s := netip.MustParseAddrPort("[::1]:54321"), netip.MustParseAddrPort("[::1]:8791")
	other := netip.MustParseAddrPort("127.0.0.1:1111")

	clientV4 := procRow(false, v4c, v4s, 1001)
	serverV4 := procRow(false, v4s, v4c, 1000)
	clientV6 := procRow(true, v6c, v6s, 1002)
	serverV6 := procRow(true, v6s, v6c, 1000)
	// A dual-stack listener: the server's accepted socket is in tcp6,
	// v4-mapped, while the IPv4 client's own socket is in tcp.
	serverMapped := procRow(true, v4s, v4c, 1000)
	noise := procRow(false, other, v4s, 1003)

	cases := []struct {
		name          string
		snaps         [][]procTable
		client, local netip.AddrPort
		uid           int
		want          peerVerdict
		reads         int
	}{
		{"ipv4 client row found", [][]procTable{{procTableOf(false, noise, serverV4, clientV4), procTableOf(true)}}, v4c, v4s, 1001, peerFound, 1},
		{"ipv6 client row found", [][]procTable{{procTableOf(false), procTableOf(true, serverV6, clientV6)}}, v6c, v6s, 1002, peerFound, 1},
		{"v4-mapped server row, ipv4 client row", [][]procTable{{procTableOf(false, clientV4), procTableOf(true, serverMapped)}}, v4c, v4s, 1001, peerFound, 1},
		{"client row missing twice: refused", [][]procTable{{procTableOf(false, noise, serverV4), procTableOf(true)}}, v4c, v4s, 0, peerMissing, 2},
		{"ipv6 client row missing twice: refused", [][]procTable{{procTableOf(false), procTableOf(true, serverV6)}}, v6c, v6s, 0, peerMissing, 2},
		{"v4-mapped server row, client missing twice: refused", [][]procTable{{procTableOf(false, noise), procTableOf(true, serverMapped)}}, v4c, v4s, 0, peerMissing, 2},
		{"re-read finds the client", [][]procTable{
			{procTableOf(false, serverV4), procTableOf(true)},
			{procTableOf(false, serverV4, clientV4), procTableOf(true)},
		}, v4c, v4s, 1001, peerFound, 2},
		{"tables unreadable", [][]procTable{nil}, v4c, v4s, 0, peerUnknown, 1},
		// Both tables readable and header-only: the environment does not
		// report connections (WSL1, gVisor). No torn read empties both.
		{"both tables header-only", [][]procTable{{procTableOf(false), procTableOf(true)}}, v4c, v4s, 0, peerNotReported, 1},
		{"one table has rows, client missing: refused", [][]procTable{{procTableOf(false), procTableOf(true, serverV6)}}, v4c, v4s, 0, peerMissing, 2},
		{"only tcp read, header-only: refused", [][]procTable{{procTableOf(false)}}, v4c, v4s, 0, peerMissing, 2},
		{"rows, then both header-only on the re-read: refused", [][]procTable{
			{procTableOf(false, noise), procTableOf(true)},
			{procTableOf(false), procTableOf(true)},
		}, v4c, v4s, 0, peerMissing, 2},
		// Readable tables that lack both rows are torn too: refused.
		{"server row missing too: refused", [][]procTable{{procTableOf(false, noise), procTableOf(true)}}, v4c, v4s, 0, peerMissing, 2},
		{"only tcp readable, neither row: refused", [][]procTable{{procTableOf(false, noise)}}, v4c, v4s, 0, peerMissing, 2},
		// Each decision uses one whole snapshot: the client row that only
		// the second read has is found there, with that read's uid.
		{"second snapshot used whole", [][]procTable{
			{procTableOf(false, noise), procTableOf(true, serverMapped)},
			{procTableOf(false, procRow(false, v4c, v4s, 1005)), procTableOf(true)},
		}, v4c, v4s, 1005, peerFound, 2},
		// Tables readable on the first read decide the request: becoming
		// unreadable on the re-read does not turn it into an allow.
		{"unreadable on the re-read: refused", [][]procTable{{procTableOf(false, serverV4)}, nil}, v4c, v4s, 0, peerMissing, 2},
		// A read error other than not-exist/permission (EMFILE from a
		// connection flood) yields an empty, non-nil snapshot: refused.
		{"transient read error: refused", [][]procTable{{}}, v4c, v4s, 0, peerMissing, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			read, n := reads(c.snaps...)
			uid, v := peerFromTables(read, c.client, c.local)
			if v != c.want || (v == peerFound && uid != c.uid) || *n != c.reads {
				t.Fatalf("got uid %d verdict %v after %d reads, want uid %d verdict %v after %d", uid, v, *n, c.uid, c.want, c.reads)
			}
		})
	}
}

// A client row that stays missing while the server's own row is there is a
// torn or forged read: the login is refused, and the code survives for the
// user's browser.
func TestLoginRefusesAPeerWhoseRowIsMissing(t *testing.T) {
	s, ts, _ := loginServer(t)
	s.peerUID = func(*http.Request) (int, peerVerdict) { return 0, peerMissing }
	code := s.NewLoginCode()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusForbidden {
		t.Fatalf("missing client row: %d, want 403", res.StatusCode)
	}
	s.peerUID = func(*http.Request) (int, peerVerdict) { return s.selfUID, peerFound }
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusOK {
		t.Fatalf("the refused attempt burned the code: %d, want 200", res.StatusCode)
	}
}

// Where the tables cannot be read at all (WSL1, gVisor, a restricted /proc)
// the login is allowed, with a warning in the log.
func TestLoginAllowsAnUnreadablePeerWithAWarning(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	s, ts, _ := loginServer(t)
	s.peerUID = func(*http.Request) (int, peerVerdict) { return 0, peerUnknown }
	if res, _ := login(t, ts, s.NewLoginCode()); res.StatusCode != http.StatusOK {
		t.Fatalf("unreadable peer tables: %d, want 200", res.StatusCode)
	}
	if !strings.Contains(buf.String(), "WARNING") {
		t.Fatalf("log %q: want a WARNING", buf.String())
	}
}

// fakeTables installs the real request-to-tables lookup on s, reading the
// given snapshots in place of /proc. Each snapshot is built for the request
// it is asked about, so tests can place (or leave out) its rows.
func fakeTables(s *Server, snap func(client, local netip.AddrPort) []procTable) {
	s.peerUID = func(r *http.Request) (int, peerVerdict) {
		client, local, _ := requestAddrs(r)
		return peerUIDFromTables(func() []procTable { return snap(client, local) })(r)
	}
}

// End to end through the handler: readable tables that list neither end of
// the connection are refused, and the code is not spent.
func TestLoginRefusesWhenReadableTablesLackBothRows(t *testing.T) {
	s, ts, _ := loginServer(t)
	other := netip.MustParseAddrPort("127.0.0.1:1111")
	fakeTables(s, func(client, local netip.AddrPort) []procTable {
		return []procTable{procTableOf(false, procRow(false, other, local, 1003)), procTableOf(true)}
	})
	code := s.NewLoginCode()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusForbidden {
		t.Fatalf("readable tables without our rows: %d, want 403", res.StatusCode)
	}
	fakeTables(s, func(client, local netip.AddrPort) []procTable {
		return []procTable{procTableOf(false, procRow(false, client, local, s.selfUID))}
	})
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusOK {
		t.Fatalf("the refused attempt burned the code: %d, want 200", res.StatusCode)
	}
}

// End to end: tables that cannot be read at all let the login through with
// a warning, and nothing is remembered: the next request with readable
// tables lacking the rows is refused again.
func TestLoginAllowsUnreadableTablesWithAWarningAndRemembersNothing(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	s, ts, _ := loginServer(t)
	fakeTables(s, func(netip.AddrPort, netip.AddrPort) []procTable { return nil })
	if res, _ := login(t, ts, s.NewLoginCode()); res.StatusCode != http.StatusOK {
		t.Fatalf("unreadable tables: %d, want 200", res.StatusCode)
	}
	if !strings.Contains(buf.String(), "WARNING") {
		t.Fatalf("log %q: want a WARNING", buf.String())
	}
	other := netip.MustParseAddrPort("127.0.0.1:1111")
	fakeTables(s, func(_, local netip.AddrPort) []procTable {
		return []procTable{procTableOf(false, procRow(false, other, local, 1003))}
	})
	if res, _ := login(t, ts, s.NewLoginCode()); res.StatusCode != http.StatusForbidden {
		t.Fatalf("readable tables after an unreadable request: %d, want 403", res.StatusCode)
	}
}

// Ruling P37: both tables readable and header-only means the environment
// does not report connections (WSL1, gVisor); a real kernel lists an
// established connection, and a torn read of a populated table still has
// rows. The login is allowed, with a warning naming those environments.
func TestLoginAllowsHeaderOnlyTablesWithAWarning(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	s, ts, _ := loginServer(t)
	fakeTables(s, func(netip.AddrPort, netip.AddrPort) []procTable {
		return []procTable{procTableOf(false), procTableOf(true)}
	})
	if res, _ := login(t, ts, s.NewLoginCode()); res.StatusCode != http.StatusOK {
		t.Fatalf("header-only tables: %d, want 200", res.StatusCode)
	}
	if !strings.Contains(buf.String(), "WARNING") || !strings.Contains(buf.String(), "WSL1") || !strings.Contains(buf.String(), "gVisor") {
		t.Fatalf("log %q: want a WARNING naming WSL1 and gVisor", buf.String())
	}
}

// One table with rows but none for this client: refused, code not spent.
func TestLoginRefusesWhenOneTableHasRowsButNotTheClient(t *testing.T) {
	s, ts, _ := loginServer(t)
	other := netip.MustParseAddrPort("[::1]:1111")
	fakeTables(s, func(client, local netip.AddrPort) []procTable {
		return []procTable{procTableOf(false), procTableOf(true, procRow(true, other, netip.MustParseAddrPort("[::1]:8791"), 1003))}
	})
	code := s.NewLoginCode()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusForbidden {
		t.Fatalf("one populated table without the client: %d, want 403", res.StatusCode)
	}
	fakeTables(s, func(client, local netip.AddrPort) []procTable {
		return []procTable{procTableOf(false, procRow(false, client, local, s.selfUID)), procTableOf(true)}
	})
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusOK {
		t.Fatalf("the refused attempt burned the code: %d, want 200", res.StatusCode)
	}
}

// A verdict the handler does not know (a value added later and not wired
// in) fails closed: 403, code not spent.
func TestLoginRefusesAnUnknownVerdict(t *testing.T) {
	s, ts, _ := loginServer(t)
	s.peerUID = func(*http.Request) (int, peerVerdict) { return s.selfUID, peerVerdict(99) }
	code := s.NewLoginCode()
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusForbidden {
		t.Fatalf("out-of-range verdict: %d, want 403", res.StatusCode)
	}
	s.peerUID = func(*http.Request) (int, peerVerdict) { return s.selfUID, peerFound }
	if res, _ := login(t, ts, code); res.StatusCode != http.StatusOK {
		t.Fatalf("the refused attempt burned the code: %d, want 200", res.StatusCode)
	}
}

// A request that is not over TCP (the owner-only unix socket) has no row to
// look up: the lookup says so without touching /proc, rather than claiming
// /proc cannot be read.
func TestPeerLookupOfANonTCPRequest(t *testing.T) {
	read := func() []procTable { t.Fatal("read /proc for a non-TCP request"); return nil }
	r := httptest.NewRequest(http.MethodGet, "/login", nil)
	r.RemoteAddr = "@"
	if _, v := peerUIDFromTables(read)(r); v != peerNotTCP {
		t.Fatalf("verdict %v, want peerNotTCP", v)
	}
}

// Login over the unix socket stays refused (it has no browser), without the
// misleading "/proc/net/tcp cannot be read" warning, and without spending
// the code.
func TestLoginOverTheUnixSocketIsRefusedQuietly(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })

	sock := filepath.Join(testutil.ShortTempDir(t), "s.sock")
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.ServeUnix(ctx, sock); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	code := s.NewLoginCode()
	var res *http.Response
	var err error
	for i := 0; i < 50; i++ {
		if res, err = client.Get("http://x/login?code=" + code); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("login over the unix socket: %d, want 403", res.StatusCode)
	}
	if strings.Contains(buf.String(), "/proc") {
		t.Fatalf("log %q: a unix-socket request is not a /proc matter", buf.String())
	}
	if ok := s.redeemLoginCode(code); !ok {
		t.Fatal("the refused unix-socket attempt burned the code")
	}
}

// procSnapshot sorts read errors: a table that does not exist or may not be
// read is "unreadable"; any other error (EMFILE in a connection flood) is
// not, so the snapshot stays non-nil and a missing row fails closed.
func TestProcSnapshotSortsReadErrors(t *testing.T) {
	emfile := errors.New("open /proc/net/tcp: too many open files")
	for _, c := range []struct {
		name    string
		errs    map[string]error
		wantNil bool
		tables  int
	}{
		{"both readable", nil, false, 2},
		{"both missing", map[string]error{"/proc/net/tcp": fs.ErrNotExist, "/proc/net/tcp6": fs.ErrNotExist}, true, 0},
		{"missing and not permitted", map[string]error{"/proc/net/tcp": fs.ErrPermission, "/proc/net/tcp6": fs.ErrNotExist}, true, 0},
		{"tcp6 missing (IPv6 off)", map[string]error{"/proc/net/tcp6": fs.ErrNotExist}, false, 1},
		{"EMFILE on both", map[string]error{"/proc/net/tcp": emfile, "/proc/net/tcp6": emfile}, false, 0},
		{"EMFILE and missing", map[string]error{"/proc/net/tcp": emfile, "/proc/net/tcp6": fs.ErrNotExist}, false, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := procSnapshot(func(path string) ([]byte, error) {
				if err := c.errs[path]; err != nil {
					return nil, err
				}
				return []byte(procHeader), nil
			})
			if (got == nil) != c.wantNil || len(got) != c.tables {
				t.Fatalf("snapshot %v (nil %v), want nil %v with %d tables", got, got == nil, c.wantNil, c.tables)
			}
		})
	}
}
