package server

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/valve-tech/jumpgate/internal/api"
	"github.com/valve-tech/jumpgate/internal/config"
	"github.com/valve-tech/jumpgate/internal/intent"
	"github.com/valve-tech/jumpgate/internal/signer"
)

// pairedTarget is the saved target box, with its agent.
func pairedTarget(t *testing.T) config.Target {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	tg, ok := findTarget(cfg, "box")
	if !ok || tg.Agent == nil {
		t.Fatal("box is not paired")
	}
	return tg
}

// Verification is one signed agent.info round trip. On a re-pair the agent
// already holds a higher sequence for this controller; the next sequence
// follows its signed answer, or every intent after pairing would be refused
// as stale.
func TestVerifyPairingFollowsTheAgentsSequence(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{onServer: func(s *Server) { srv = s }})
	tg := pairedTarget(t)
	next, ev := srv.verifyPairing(context.Background(), tg)
	if ev != nil || next != 2 {
		t.Fatalf("first verify: next %d, %+v", next, ev)
	}
	cfg, _ := srv.loadConfig()
	for i := 0; i < 3; i++ {
		if _, err := srv.agentResult(context.Background(), cfg, tg, intent.KindAgentInfo, struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	next, ev = srv.verifyPairing(context.Background(), tg)
	if ev != nil || next < 5 {
		t.Fatalf("re-pair verify: next %d (the agent has seen more), %+v", next, ev)
	}
}

// An agent that answers with another key than the one paired is not the box
// that was installed: a security failure, never a pairing.
func TestVerifyPairingRefusesAnotherAgentsKey(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{onServer: func(s *Server) { srv = s }})
	tg := pairedTarget(t)
	other, _ := signer.GenerateKey()
	ap := *tg.Agent
	ap.Address = other.Address().Hex()
	tg.Agent = &ap
	if _, ev := srv.verifyPairing(context.Background(), tg); ev == nil || ev.Code != api.CodeBadReceipt || ev.Hint == "" {
		t.Fatalf("verify against another agent's key: %+v", ev)
	}
}

// A controller the agent does not know is refused agent.info: the pairing
// did not enroll it, and verification says so.
func TestVerifyPairingReportsAnAgentThatRefusesThisController(t *testing.T) {
	requireAgentPeer(t)
	stranger, _ := signer.GenerateKey()
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{
		server:   func(c *Config) { c.Signer = stranger },
		onServer: func(s *Server) { srv = s },
	})
	if _, ev := srv.verifyPairing(context.Background(), pairedTarget(t)); ev == nil || ev.Code != api.CodeRejected {
		t.Fatalf("verify by an unenrolled controller: %+v", ev)
	}
}

// No agent at the socket: unreachable, with its hint.
func TestVerifyPairingReportsAMissingAgent(t *testing.T) {
	requireAgentPeer(t)
	var srv *Server
	pairedBoxWith(t, nopExec{}, pairedOpts{onServer: func(s *Server) { srv = s }})
	tg := pairedTarget(t)
	ap := *tg.Agent
	ap.Socket = filepath.Join(t.TempDir(), "gone.sock")
	tg.Agent = &ap
	if _, ev := srv.verifyPairing(context.Background(), tg); ev == nil || ev.Code != api.CodeUnreachable || ev.Hint == "" {
		t.Fatalf("verify with no agent: %+v", ev)
	}
}
