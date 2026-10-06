package apiclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/buildinfo"
	"github.com/valve-tech/jumpgate/internal/daemon"
	"github.com/valve-tech/jumpgate/internal/intent"
)

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return newHTTP(ts.URL, "tok", buildinfo.Version())
}

func TestDoSendsTheTokenAndDecodesJSON(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", 401)
			return
		}
		fmt.Fprint(w, `{"status":0,"result":{"x":1}}`)
	})
	r, err := c.Intent(context.Background(), "box", intent.KindStatusRead, nil)
	if err != nil || r.Status != intent.StatusOK || string(r.Result) != `{"x":1}` {
		t.Fatalf("Intent = %+v, %v", r, err)
	}
}

func TestErrorsAreAPIErrors(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(504)
		fmt.Fprint(w, `{"error":"box down","code":"unreachable"}`)
	})
	_, err := c.Intent(context.Background(), "box", intent.KindStatusRead, nil)
	var e *api.Error
	if !errors.As(err, &e) || e.Status != 504 || e.Code != api.CodeUnreachable || e.Hint == "" {
		t.Fatalf("err = %#v", err)
	}
}

func TestAServerThatIsGoneIsServerUnreachable(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	c := newHTTP(ts.URL, "tok", buildinfo.Version())
	ts.Close()
	err := c.Do(context.Background(), http.MethodGet, "/api/health", nil, nil)
	if !errors.Is(err, ErrServerUnreachable) {
		t.Fatalf("err = %v", err)
	}
}

// The real transport: HTTP over the server's unix socket.
func TestNewTalksOverTheUnixSocket(t *testing.T) {
	dir, _ := os.MkdirTemp("", "jgc")
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"status":0}`)
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	info := daemon.Info{Socket: sock, Token: "tok", Version: "v0.0.0-test"}
	c := New(info)
	if _, err := c.Intent(context.Background(), "box", "agent.info", nil); err != nil {
		t.Fatal(err)
	}
	if c.ServerVersion() != "v0.0.0-test" {
		t.Fatalf("version %q", c.ServerVersion())
	}
	if c.Info() != info {
		t.Fatalf("Info() = %+v, want %+v", c.Info(), info)
	}
}

// Ruling T2a: the client asks daemon.SkewWarning, the one version
// comparison, so an unknown server version (an older server.json) counts as
// a skew there and here alike.
func TestSkew(t *testing.T) {
	c := New(daemon.Info{Version: "v0.0.1-old"})
	server, mine, differs := c.Skew()
	if !differs || server != "v0.0.1-old" || mine == "" {
		t.Fatalf("Skew = %q %q %v", server, mine, differs)
	}
	var b strings.Builder
	WarnSkew(&b, c)
	if !strings.Contains(b.String(), "jumpgate stop") || b.String() != daemon.SkewWarning(c.Info())+"\n" {
		t.Fatalf("warning %q is not daemon.SkewWarning's", b.String())
	}
	if _, _, d := New(daemon.Info{}).Skew(); !d {
		t.Fatal("an unknown server version is a skew (daemon.SkewWarning's rule)")
	}
	same := New(daemon.Info{Version: buildinfo.Version()})
	b.Reset()
	WarnSkew(&b, same)
	if _, _, d := same.Skew(); d || b.Len() != 0 {
		t.Fatalf("same version: differs %v, warning %q", d, b.String())
	}
}

func TestPairStreamsEventsUntilDone(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"step\":\"upload\",\"line\":\"ok\"}\n\n")
		fmt.Fprint(w, "data: {\"done\":true,\"agent\":\"0xabc\"}\n\n")
	})
	ch, err := c.Pair(context.Background(), "box", api.PairRequest{Sudo: true})
	if err != nil {
		t.Fatal(err)
	}
	var evs []api.PairEvent
	for ev := range ch {
		evs = append(evs, ev)
	}
	if len(evs) != 2 || evs[0].Step != "upload" || !evs[1].Done || evs[1].Agent != "0xabc" {
		t.Fatalf("events %+v", evs)
	}
}

// A real client re-reads server.json to rediscover a restarted server.
func TestNewCanRediscover(t *testing.T) {
	if New(daemon.Info{}).rediscover == nil {
		t.Fatal("New's client cannot rediscover a restarted server")
	}
}

func TestTargets(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"a","mode":"ssh","agent":{"address":"0x1","transport":"ssh"},"link":"agent"}]`)
	})
	ts, err := c.Targets(context.Background())
	if err != nil || len(ts) != 1 || ts[0].Link != api.LinkAgent || ts[0].Agent.Address != "0x1" {
		t.Fatalf("Targets = %+v, %v", ts, err)
	}
}

func TestNodeMethodsUseTheNodeRoutes(t *testing.T) {
	var paths []string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		switch {
		case strings.HasSuffix(r.URL.Path, "/firewall"):
			fmt.Fprint(w, `[{"ID":"p2p","Status":"pass"}]`)
		case strings.HasSuffix(r.URL.Path, "/endpoints"):
			fmt.Fprint(w, `{"ExecHTTP":"http://127.0.0.1:8545","ExecReachable":true}`)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			fmt.Fprint(w, `{"Active":true}`)
		}
	})
	ctx := context.Background()
	if r, err := c.ServiceAction(ctx, "box", "beacon", "restart"); err != nil || !r.Active {
		t.Fatalf("ServiceAction %+v %v", r, err)
	}
	if e, err := c.Endpoints(ctx, "box"); err != nil || !e.ExecReachable {
		t.Fatalf("Endpoints %+v %v", e, err)
	}
	if f, err := c.Firewall(ctx, "box"); err != nil || f[0].Status != "pass" {
		t.Fatalf("Firewall %+v %v", f, err)
	}
	if err := c.RemoveTarget(ctx, "box"); err != nil {
		t.Fatal(err)
	}
	want := "POST /api/targets/box/services/beacon/restart GET /api/targets/box/endpoints GET /api/targets/box/firewall DELETE /api/targets/box"
	if strings.Join(paths, " ") != want {
		t.Fatalf("paths %v", paths)
	}
}

func TestHostKeyMethods(t *testing.T) {
	var bodies []string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, r.Method+" "+r.URL.Path+" "+string(b))
		if r.URL.Path == "/api/hostkeys/probe" {
			fmt.Fprint(w, `{"hops":[{"hostPort":"h:22","state":"unknown","probeId":"p1","fingerprint":"SHA256:x"}]}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	ctx := context.Background()
	p, err := c.ProbeHostKeys(ctx, api.SSHView{Host: "h", User: "u"})
	if err != nil || p.Pending() == nil || p.Pending().ProbeID != "p1" {
		t.Fatalf("probe %+v %v", p, err)
	}
	if err := c.ConfirmHostKey(ctx, "p1", "SHA256:x"); err != nil {
		t.Fatal(err)
	}
	if err := c.AddTarget(ctx, api.AddTarget{ID: "b", Mode: "ssh", SSH: &api.SSHView{Host: "h", User: "u"}}); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 3 ||
		!strings.HasPrefix(bodies[0], "POST /api/hostkeys/probe ") || !strings.Contains(bodies[0], `"ssh":{"Host":"h"`) ||
		!strings.HasPrefix(bodies[1], "POST /api/hostkeys/confirm ") || !strings.Contains(bodies[1], `"probeId":"p1"`) || !strings.Contains(bodies[1], `"fingerprint":"SHA256:x"`) ||
		!strings.HasPrefix(bodies[2], "POST /api/targets ") || !strings.Contains(bodies[2], `"Host":"h"`) {
		t.Fatalf("bodies %q", bodies)
	}
}

func TestForgetHostKeyMethods(t *testing.T) {
	var reqs []string
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		reqs = append(reqs, r.Method+" "+r.URL.RequestURI()+" "+string(b))
		if r.URL.Path == "/api/hostkeys" {
			fmt.Fprint(w, `{"hostPort":"[::1]:22","keys":[{"fingerprint":"SHA256:x","keyType":"ssh-ed25519","store":"jumpgate"}]}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	ctx := context.Background()
	rec, err := c.RecordedHostKeys(ctx, "[::1]:22")
	if err != nil || len(rec.Keys) != 1 || rec.Keys[0].Store != api.HostKeyStoreJumpgate {
		t.Fatalf("recorded %+v %v", rec, err)
	}
	if err := c.ForgetHostKey(ctx, "[::1]:22", "SHA256:x"); err != nil {
		t.Fatal(err)
	}
	if len(reqs) != 2 || !strings.HasPrefix(reqs[0], "GET /api/hostkeys?hostPort=%5B%3A%3A1%5D%3A22 ") ||
		!strings.HasPrefix(reqs[1], "POST /api/hostkeys/forget ") || !strings.Contains(reqs[1], `"hostPort":"[::1]:22","fingerprint":"SHA256:x"`) {
		t.Fatalf("requests %q", reqs)
	}
}
