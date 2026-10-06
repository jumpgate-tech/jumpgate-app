package agentclient

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
	"github.com/valve-tech/jumpgate/internal/testutil"
)

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func mustDial(t *testing.T, tg Target, s signer.Signer, seqs SeqStore) *Client {
	t.Helper()
	c, err := Dial(context.Background(), tg, s, seqs)
	must(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

func mustKey(t *testing.T) *signer.Key {
	t.Helper()
	k, err := signer.GenerateKey()
	must(t, err)
	return k
}

type stubExec struct{}

func (stubExec) Run(context.Context, string, *executor.RunOpts) (executor.Result, error) {
	return executor.Result{}, nil
}
func (stubExec) WriteFile(context.Context, string, []byte, os.FileMode) error { return nil }
func (stubExec) ReadFile(context.Context, string) ([]byte, error)             { return nil, nil }
func (stubExec) Close() error                                                 { return nil }

// startAgent runs a real agent on a temp socket and returns its socket, the
// agent's address and the enrolled controller key.
func startAgent(t *testing.T) (sock string, agentAddr eip712.Address, controller *signer.Key, a *agent.Agent) {
	t.Helper()
	testutil.RequireUnix(t) // a real agent needs peer credentials
	dir := testutil.ShortTempDir(t)
	agentKey, err := signer.GenerateKey()
	must(t, err)
	controller, err = signer.GenerateKey()
	must(t, err)
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	must(t, p.Save(filepath.Join(dir, "policy.json")))
	must(t, agent.InitReplay(filepath.Join(dir, "replay.json")))
	a = agent.New(agent.Config{Key: agentKey, Exec: stubExec{}, PolicyPath: filepath.Join(dir, "policy.json"),
		ReplayPath: filepath.Join(dir, "replay.json"), NodePath: filepath.Join(dir, "node.json")})
	sock = filepath.Join(dir, "a.sock")
	ln, err := agent.Listen(sock, -1, 0o660)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go agent.Serve(ctx, a, ln)
	return sock, agentKey.Address(), controller, a
}

func TestDoLocalAgentInfo(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	c, err := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: agentAddr}, controller, NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Do = %+v, %v", res, err)
	}
	var info intent.AgentInfo
	must(t, json.Unmarshal(res.Result, &info))
	if info.Address != agentAddr.Hex() {
		t.Fatalf("info = %+v", info)
	}
}

// A receipt signed by anyone other than the paired agent is a security error,
// never a result.
func TestDoRejectsAReceiptFromTheWrongAgent(t *testing.T) {
	sock, _, controller, _ := startAgent(t)
	other := mustKey(t)
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: other.Address()}, controller, NewMemorySeqStore())
	_, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if !errors.Is(err, ErrBadReceipt) {
		t.Fatalf("err = %v, want ErrBadReceipt", err)
	}
}

// A controller whose counter fell behind (restored config, second machine)
// resynchronises from the signed stale_seq rejection and retries once.
func TestDoResyncsAStaleSequence(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	seqs := NewMemorySeqStore()
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: agentAddr}, controller, seqs)
	for i := 0; i < 3; i++ {
		if _, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	must(t, seqs.Set(agentAddr, 1)) // forget
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("after resync: %+v, %v", res, err)
	}
}

func TestDoSeparatesUnreachableFromRefused(t *testing.T) {
	k := mustKey(t)
	c, err := Dial(context.Background(), Target{Local: true, Socket: "/tmp/does-not-exist.sock", Agent: k.Address()}, k, NewMemorySeqStore())
	if err == nil {
		_, err = c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	}
	if !errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want ErrUnreachable", err)
	}
}

// The remote path: SSH as the tunnel user, then a direct-streamlocal channel
// to the agent's socket, exactly as OpenSSH provides it.
func TestDoOverSSHStreamlocal(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	host, port, hostKey, keyPath := startStreamlocalSSHD(t)
	target := Target{Socket: sock, Agent: agentAddr, SSH: executor.SSHConfig{
		Host: host, Port: port, User: "jumpgate", KeyPath: keyPath, HostKey: ssh.FixedHostKey(hostKey),
	}}
	c, err := Dial(context.Background(), target, controller, NewMemorySeqStore())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("Do over SSH = %+v, %v", res, err)
	}
}

func writePrivateKey(t *testing.T, priv ed25519.PrivateKey) string {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "id_ed25519")
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	return path
}

// testKeys returns an ed25519 host signer and the path of a client key.
func testKeys(t *testing.T) (ssh.Signer, string) {
	t.Helper()
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return hostSigner, writePrivateKey(t, clientPriv)
}

// startStreamlocalSSHD is a gliderlabs server whose only capability is the
// direct-streamlocal@openssh.com channel, registered by hand.
func startStreamlocalSSHD(t *testing.T) (string, int, ssh.PublicKey, string) {
	t.Helper()
	hostSigner, clientKeyPath := testKeys(t)
	srv := &gliderssh.Server{
		PublicKeyHandler: func(gliderssh.Context, gliderssh.PublicKey) bool { return true },
		ChannelHandlers: map[string]gliderssh.ChannelHandler{
			"direct-streamlocal@openssh.com": func(_ *gliderssh.Server, _ *ssh.ServerConn, newChan ssh.NewChannel, _ gliderssh.Context) {
				var req struct {
					SocketPath string
					Reserved0  string
					Reserved1  uint32
				}
				if err := ssh.Unmarshal(newChan.ExtraData(), &req); err != nil {
					newChan.Reject(ssh.ConnectionFailed, "bad request")
					return
				}
				conn, err := net.Dial("unix", req.SocketPath)
				if err != nil {
					newChan.Reject(ssh.ConnectionFailed, err.Error())
					return
				}
				ch, reqs, err := newChan.Accept()
				if err != nil {
					conn.Close()
					return
				}
				go ssh.DiscardRequests(reqs)
				go func() { io.Copy(ch, conn); ch.CloseWrite() }()
				go func() { io.Copy(conn, ch); conn.Close() }()
			},
		},
	}
	srv.AddHostKey(hostSigner)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	a := ln.Addr().(*net.TCPAddr)
	return "127.0.0.1", a.Port, hostSigner.PublicKey(), clientKeyPath
}

// A correctly signed receipt that does not answer this exact request, or
// whose result bytes were swapped, is discarded.
func TestDoRejectsReceiptsThatDoNotAnswerThisRequest(t *testing.T) {
	cases := map[string]func(rc *intent.Receipt, in intent.Intent, result *[]byte){
		"wrong request hash": func(rc *intent.Receipt, _ intent.Intent, _ *[]byte) { rc.RequestHash[0] ^= 1 },
		"wrong seq":          func(rc *intent.Receipt, _ intent.Intent, _ *[]byte) { rc.Seq++ },
		"swapped result":     func(_ *intent.Receipt, _ intent.Intent, r *[]byte) { *r = []byte(`{"tampered":true}`) },
		"wrong agent field":  func(rc *intent.Receipt, _ intent.Intent, _ *[]byte) { rc.Agent = eip712.Address{1} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			agentKey, controller := mustKey(t), mustKey(t)
			sock := fakeAgent(t, func(w http.ResponseWriter, r *http.Request) {
				var env intent.Envelope
				if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
					t.Error(err)
					return
				}
				in, err := env.Intent.Parse()
				if err != nil {
					t.Error(err)
					return
				}
				digest, err := in.Digest()
				if err != nil {
					t.Error(err)
					return
				}
				result := []byte(`{"ok":true}`)
				rc := intent.Receipt{Agent: agentKey.Address(), RequestHash: digest, Seq: in.Seq, Status: intent.StatusOK, ResultHash: intent.Hash(result)}
				mutate(&rc, in, &result)
				sig, err := agentKey.SignTypedData(r.Context(), rc.TypedData())
				if err != nil {
					t.Error(err)
					return
				}
				json.NewEncoder(w).Encode(intent.ReceiptEnvelope{Receipt: rc.JSON(), Result: result, Sig: sig.Hex()})
			})
			c := mustDial(t, Target{Local: true, Socket: sock, Agent: agentKey.Address()}, controller, NewMemorySeqStore())
			res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
			if !errors.Is(err, ErrBadReceipt) || res.Result != nil {
				t.Fatalf("Do = %+v, %v; want ErrBadReceipt and no result", res, err)
			}
		})
	}
}

// fakeAgent serves handler on a fresh unix socket and returns its path.
func fakeAgent(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	dir := testutil.ShortTempDir(t) // AF_UNIX works on Windows too
	sock := filepath.Join(dir, "a.sock")
	ln, err := net.Listen("unix", sock)
	must(t, err)
	srv := &http.Server{Handler: handler}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
	return sock
}

// An endless reply is cut off and refused, not buffered without bound.
func TestDoRefusesAnOversizedReply(t *testing.T) {
	k, controller := mustKey(t), mustKey(t)
	sock := fakeAgent(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"receipt":{},"result":"`))
		chunk := bytes.Repeat([]byte("A"), 1<<20)
		for i := 0; i < 64; i++ { // 64 MiB offered, limit is 16
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: k.Address()}, controller, NewMemorySeqStore())
	start := time.Now()
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if !errors.Is(err, ErrBadReceipt) || res.Result != nil {
		t.Fatalf("Do = %+v, %v; want ErrBadReceipt", res, err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Fatalf("took %v", d)
	}
}

// Non-200 answers are unsigned; they are an HTTP error, not unreachable and
// not a Rejection.
func TestDoReportsHTTPErrorsFromTheAgent(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusRequestEntityTooLarge, http.StatusBadRequest} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			k, controller := mustKey(t), mustKey(t)
			sock := fakeAgent(t, func(w http.ResponseWriter, r *http.Request) { http.Error(w, "nope here", code) })
			c := mustDial(t, Target{Local: true, Socket: sock, Agent: k.Address()}, controller, NewMemorySeqStore())
			res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
			if !errors.Is(err, ErrAgentHTTP) || errors.Is(err, ErrUnreachable) || res.Rejection != nil {
				t.Fatalf("Do = %+v, %v; want ErrAgentHTTP", res, err)
			}
			if !strings.Contains(err.Error(), strconv.Itoa(code)) || !strings.Contains(err.Error(), "nope here") {
				t.Fatalf("message lacks status or body: %v", err)
			}
		})
	}
}

// Concurrent Do calls on one client never sign the same sequence.
func TestDoIsSafeForConcurrentUse(t *testing.T) {
	sock, agentAddr, controller, _ := startAgent(t)
	c := mustDial(t, Target{Local: true, Socket: sock, Agent: agentAddr}, controller, NewMemorySeqStore())
	const n = 16
	errs := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
			if err == nil && res.Status != intent.StatusOK {
				err = fmt.Errorf("status %d: %+v", res.Status, res.Rejection)
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
}
