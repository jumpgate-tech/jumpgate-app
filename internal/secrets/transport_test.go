package secrets

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/valve-tech/jumpgate/internal/vault"
	"github.com/valve-tech/jumpgate/internal/vault/vaulttest"
)

func TestEnsureTransportKeyIsStableInTheVault(t *testing.T) {
	v := vaulttest.Unlocked(t)
	lines := make([]string, 4)
	var wg sync.WaitGroup
	for i := range lines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l, err := EnsureTransportKey(v)
			if err != nil {
				t.Error(err)
			}
			lines[i] = l
		}()
	}
	wg.Wait()
	a := lines[0]
	for _, b := range lines[1:] {
		if a != b {
			t.Fatalf("lines differ: %q %q", a, b)
		}
	}
	if !strings.HasSuffix(a, " jumpgate-controller") {
		t.Fatalf("line %q", a)
	}
	s, err := TransportSigner(v)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(s.PublicKey()))) + " jumpgate-controller"; want != a {
		t.Fatalf("signer %q, line %q", want, a)
	}
	v.Lock()
	if _, err := TransportSigner(v); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("locked: %v", err)
	}
	if _, err := EnsureTransportKey(nil); !errors.Is(err, vault.ErrLocked) {
		t.Fatalf("nil: %v", err)
	}
}
