package signer

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeCreds struct {
	m       map[string][]byte
	readErr error
	drop    bool // write reports success but stores nothing
}

func (f *fakeCreds) read(target string) ([]byte, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}
	b, ok := f.m[target]
	if !ok {
		return nil, errCredNotFound
	}
	return append([]byte(nil), b...), nil
}

// write copies the blob, as CredWriteW does: the store zeroes its buffer
// after writing.
func (f *fakeCreds) write(target string, blob []byte) error {
	if !f.drop {
		f.m[target] = append([]byte(nil), blob...)
	}
	return nil
}
func (f *fakeCreds) del(target string) error { delete(f.m, target); return nil }

func withCreds(t *testing.T, f *fakeCreds) {
	t.Helper()
	old := winCreds
	winCreds = f
	t.Cleanup(func() { winCreds = old })
}

func TestWinCredCreateOpenRoundTrip(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{}}
	withCreds(t, f)
	k, err := Create(context.Background(), StoreWinCred, "controller")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.m["jumpgate/controller"]; !ok {
		t.Fatalf("stored under %v, want target jumpgate/controller", f.m)
	}
	k2, err := Open(context.Background(), StoreWinCred, "controller")
	if err != nil || k2.Address() != k.Address() {
		t.Fatalf("Open = %v, %v; want the created key", k2, err)
	}
}

func TestWinCredNeverReplacesAKey(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{"jumpgate/controller": []byte("00")}}
	withCreds(t, f)
	if _, err := Create(context.Background(), StoreWinCred, "controller"); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("Create over an existing credential = %v, want ErrKeyExists", err)
	}
	if string(f.m["jumpgate/controller"]) != "00" {
		t.Fatal("the existing credential was changed")
	}
}

// Any read failure other than "not found" fails closed: treating it as absent
// could replace a real key.
func TestWinCredExistenceCheckFailsClosed(t *testing.T) {
	f := &fakeCreds{m: map[string][]byte{}, readErr: errors.New("access denied")}
	withCreds(t, f)
	if _, err := Create(context.Background(), StoreWinCred, "controller"); err == nil || errors.Is(err, ErrKeyExists) {
		t.Fatalf("Create = %v, want a hard error", err)
	}
	if len(f.m) != 0 {
		t.Fatal("a key was stored after a failed existence check")
	}
}

// A write that reports success without storing anything must not hand back a
// key that exists nowhere.
func TestWinCredCreateVerifiesTheWrite(t *testing.T) {
	withCreds(t, &fakeCreds{m: map[string][]byte{}, drop: true})
	if _, err := Create(context.Background(), StoreWinCred, "controller"); err == nil || !strings.Contains(err.Error(), "could not be verified") {
		t.Fatalf("Create with a lost write = %v, want a verification error", err)
	}
}

func TestWinCredRejectsUnsafeNames(t *testing.T) {
	withCreds(t, &fakeCreds{m: map[string][]byte{}})
	if _, err := Create(context.Background(), StoreWinCred, "a b"); err == nil || !strings.Contains(err.Error(), "may only contain") {
		t.Fatalf("Create(\"a b\") = %v", err)
	}
}

func TestWinCredOpenMissingNamesKeysInit(t *testing.T) {
	withCreds(t, &fakeCreds{m: map[string][]byte{}})
	if _, err := Open(context.Background(), StoreWinCred, "controller"); err == nil || !strings.Contains(err.Error(), "--store wincred") {
		t.Fatalf("Open of a missing credential = %v", err)
	}
}

func TestWinCredOpenRejectsGarbage(t *testing.T) {
	withCreds(t, &fakeCreds{m: map[string][]byte{"jumpgate/controller": []byte("not hex")}})
	if _, err := Open(context.Background(), StoreWinCred, "controller"); err == nil {
		t.Fatal("Open of a non-hex credential succeeded")
	}
}
