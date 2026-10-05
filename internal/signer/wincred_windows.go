//go:build windows

package signer

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// advapi32's credential functions are not wrapped by x/sys/windows, so they
// are loaded lazily from the system directory (never from the search path).
var (
	advapi32        = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2 // this user, this machine; never roams to other machines
)

// credential is CREDENTIALW.
type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type winCredManager struct{}

func platformCreds() credStore { return winCredManager{} }

func (winCredManager) read(target string) ([]byte, error) {
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return nil, err
	}
	var pc *credential
	r, _, callErr := procCredReadW.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&pc)))
	if r == 0 {
		if errors.Is(callErr, windows.ERROR_NOT_FOUND) {
			return nil, errCredNotFound
		}
		return nil, fmt.Errorf("CredReadW: %w", callErr)
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(pc)))
	if pc.CredentialBlobSize == 0 || pc.CredentialBlob == nil {
		return []byte{}, nil
	}
	blob := unsafe.Slice(pc.CredentialBlob, pc.CredentialBlobSize)
	out := make([]byte, len(blob))
	copy(out, blob)
	// CredFree releases the buffer without wiping it; wipe the secret first.
	clear(blob)
	return out, nil
}

func (winCredManager) write(target string, blob []byte) error {
	if len(blob) == 0 {
		return errors.New("CredWriteW: refusing to store an empty credential")
	}
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	user, _ := windows.UTF16PtrFromString("jumpgate")
	comment, _ := windows.UTF16PtrFromString("jumpgate controller key")
	c := credential{
		Type:               credTypeGeneric,
		TargetName:         t,
		Comment:            comment,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            credPersistLocalMachine,
		UserName:           user,
	}
	if r, _, callErr := procCredWriteW.Call(uintptr(unsafe.Pointer(&c)), 0); r == 0 {
		return fmt.Errorf("CredWriteW: %w", callErr)
	}
	return nil
}

func (winCredManager) del(target string) error {
	t, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	if r, _, callErr := procCredDeleteW.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0); r == 0 && !errors.Is(callErr, windows.ERROR_NOT_FOUND) {
		return fmt.Errorf("CredDeleteW: %w", callErr)
	}
	return nil
}
