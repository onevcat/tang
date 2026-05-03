package auth

import (
	"errors"
	"fmt"
	"testing"

	keyringlib "github.com/zalando/go-keyring"
)

func TestKeyringStoreSaveLoadAndClearWithMockProvider(t *testing.T) {
	keyringlib.MockInit()
	session, err := NewSession("did:plc:alice", "onev.cat", "https://pds.example.com", "access", "refresh")
	if err != nil {
		t.Fatalf("NewSession error = %v", err)
	}
	store := KeyringStore{}
	if err := store.Save(session); err != nil {
		t.Fatalf("Save error = %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if got.DID != session.DID || got.Handle != session.Handle || got.PDS != session.PDS {
		t.Fatalf("loaded session = %#v", got)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("Clear error = %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load after clear error = %v", err)
	}
	if err := Save(session); err != nil {
		t.Fatalf("package Save error = %v", err)
	}
	if _, err := Load(); err != nil {
		t.Fatalf("package Load error = %v", err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("package Clear error = %v", err)
	}
}

func TestClassifyKeyringError(t *testing.T) {
	if classifyKeyringError(nil) != nil {
		t.Fatal("nil error should stay nil")
	}
	if !errors.Is(classifyKeyringError(keyringlib.ErrNotFound), ErrNotFound) {
		t.Fatalf("not found = %v", classifyKeyringError(keyringlib.ErrNotFound))
	}
	if !errors.Is(classifyKeyringError(fmt.Errorf("keyring locked")), ErrLocked) {
		t.Fatalf("locked = %v", classifyKeyringError(fmt.Errorf("keyring locked")))
	}
	if !errors.Is(classifyKeyringError(fmt.Errorf("permission denied")), ErrUnauthorized) {
		t.Fatalf("unauthorized = %v", classifyKeyringError(fmt.Errorf("permission denied")))
	}
	if err := classifyKeyringError(fmt.Errorf("other")); err == nil || err.Error() != "other" {
		t.Fatalf("other = %v", err)
	}
}
