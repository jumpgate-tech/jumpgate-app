package agentclient

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	gliderssh "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/agent"
	"github.com/valve-tech/jumpgate/internal/eip712"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

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
	dir, _ := os.MkdirTemp("/tmp", "jgc")
	t.Cleanup(func() { os.RemoveAll(dir) })
	agentKey, _ := signer.GenerateKey()
	controller, _ = signer.GenerateKey()
	p := agent.Policy{Signers: []agent.SignerEntry{{Address: controller.Address().Hex(), Tier: agent.TierRoutine}}}
	_ = p.Save(filepath.Join(dir, "policy.json"))
	_ = agent.InitReplay(filepath.Join(dir, "replay.json"))
	a = agent.New(agent.Config{Key: agentKey, Exec: stubExec{}, PolicyPath: filepath.Join(dir, "policy.json"),
		ReplayPath: filepath.Join(dir, "replay.json"), NodePath: filepath.Join(dir, "node.json")})
	sock = filepath.Join(dir, "a.sock")
	ln, err := agent.Listen(sock, -1)
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
	_ = json.Unmarshal(res.Result, &info)
	if info.Address != agentAddr.Hex() {
		t.Fatalf("info = %+v", info)
	}
}

// A receipt signed by anyone other than the paired agent is a security error,
// never a result.
func TestDoRejectsAReceiptFromTheWrongAgent(t *testing.T) {
	sock, _, controller, _ := startAgent(t)
	other, _ := signer.GenerateKey()
	c, _ := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: other.Address()}, controller, NewMemorySeqStore())
	defer c.Close()
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
	c, _ := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: agentAddr}, controller, seqs)
	defer c.Close()
	for i := 0; i < 3; i++ {
		if _, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	_ = seqs.Set(agentAddr, 1) // forget
	res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
	if err != nil || res.Status != intent.StatusOK {
		t.Fatalf("after resync: %+v, %v", res, err)
	}
}

func TestDoSeparatesUnreachableFromRefused(t *testing.T) {
	k, _ := signer.GenerateKey()
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
			agentKey, _ := signer.GenerateKey()
			controller, _ := signer.GenerateKey()
			dir, _ := os.MkdirTemp("/tmp", "jgf")
			t.Cleanup(func() { os.RemoveAll(dir) })
			sock := filepath.Join(dir, "a.sock")
			ln, err := net.Listen("unix", sock)
			if err != nil {
				t.Fatal(err)
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var env intent.Envelope
				_ = json.NewDecoder(r.Body).Decode(&env)
				in, _ := env.Intent.Parse()
				digest, _ := in.Digest()
				result := []byte(`{"ok":true}`)
				rc := intent.Receipt{Agent: agentKey.Address(), RequestHash: digest, Seq: in.Seq, Status: intent.StatusOK, ResultHash: intent.Hash(result)}
				mutate(&rc, in, &result)
				sig, _ := agentKey.SignTypedData(r.Context(), rc.TypedData())
				_ = json.NewEncoder(w).Encode(intent.ReceiptEnvelope{Receipt: rc.JSON(), Result: result, Sig: sig.Hex()})
			})}
			go srv.Serve(ln)
			t.Cleanup(func() { srv.Close() })
			c, _ := Dial(context.Background(), Target{Local: true, Socket: sock, Agent: agentKey.Address()}, controller, NewMemorySeqStore())
			defer c.Close()
			res, err := c.Do(context.Background(), intent.KindAgentInfo, struct{}{})
			if !errors.Is(err, ErrBadReceipt) || res.Result != nil {
				t.Fatalf("Do = %+v, %v; want ErrBadReceipt and no result", res, err)
			}
		})
	}
}
