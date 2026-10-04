package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"testing"
	"testing/fstest"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/executor"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// remotePaired saves box (from pairServer) as paired over SSH, with the
// transport key in place so the dial reaches the host-key check.
func remotePaired(t *testing.T) config.Target {
	t.Helper()
	t.Setenv("SSH_AUTH_SOCK", "")
	if _, err := ensureTransportKey(); err != nil {
		t.Fatal(err)
	}
	agentKey, _ := signer.GenerateKey()
	c, err := config.Update(func(c *config.Config) error {
		c.Targets[0].Agent = &config.AgentPairing{Address: agentKey.Address().Hex(), Transport: "ssh"}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return c.Targets[0]
}

func otherHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	k, _ := ssh.NewPublicKey(pub)
	return k
}

// I5: a host key nobody confirmed, or one that changed since, is a security
// failure (CLI exit 4), never "could not reach the box".
func TestIntentEndpointReportsHostKeyFailuresAsSecurityErrors(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	ts, token := pairServer(t, d)
	remotePaired(t)

	res, out := postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != 409 || out["code"] != "unknown_host" || out["hint"] == "" || out["fingerprint"] != executor.Fingerprint(d.hostKey) {
		t.Fatalf("unconfirmed host: %d %v", res.StatusCode, out)
	}

	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, d.hostname(), otherHostKey(t)); err != nil {
		t.Fatal(err)
	}
	res, out = postIntent(t, ts, token, "/api/targets/box/intent/agent.info", `{}`)
	if res.StatusCode != 502 || out["code"] != "host_key" || out["hint"] == "" {
		t.Fatalf("changed host key: %d %v", res.StatusCode, out)
	}
}

func TestVerifyPairingReportsHostKeyFailuresAsSecurityErrors(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	pairServer(t, d)
	tg := remotePaired(t)
	ctrl, _ := signer.GenerateKey()
	s := New(Config{Token: NewSessionToken(), UI: fstest.MapFS{}, Signer: ctrl})

	if _, ev := s.verifyPairing(context.Background(), tg); ev == nil || ev.Code != "unknown_host" || ev.Hint == "" {
		t.Fatalf("unconfirmed host: %+v", ev)
	}
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, d.hostname(), otherHostKey(t)); err != nil {
		t.Fatal(err)
	}
	if _, ev := s.verifyPairing(context.Background(), tg); ev == nil || ev.Code != "host_key" || ev.Hint == "" {
		t.Fatalf("changed host key: %+v", ev)
	}
}

// The privileged login that pairing opens maps a changed key the same way.
func TestPairReportsAChangedHostKeyAsHostKey(t *testing.T) {
	d := startPairTestSSHD(t, nil)
	ts, token := pairServer(t, d)
	confirmed, _ := config.ConfirmedHostsFile()
	if err := executor.RecordHostKey(confirmed, d.hostname(), otherHostKey(t)); err != nil {
		t.Fatal(err)
	}
	res := postPair(t, ts, token, `{}`)
	var out map[string]any
	_ = jsonDecode(res, &out)
	if res.StatusCode != 502 || out["code"] != "host_key" {
		t.Fatalf("%d %v", res.StatusCode, out)
	}
}

func jsonDecode(res *http.Response, v any) error { return json.NewDecoder(res.Body).Decode(v) }
