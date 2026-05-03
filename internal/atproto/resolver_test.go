package atproto

import (
	"context"
	"errors"
	"testing"
)

type fakeDirectory struct {
	did    string
	handle string
	pds    string
	err    error
}

func (d fakeDirectory) ResolveAtIdentifier(context.Context, string) (string, string, string, error) {
	return d.did, d.handle, d.pds, d.err
}

func (d fakeDirectory) ResolveDID(context.Context, string) (string, string, error) {
	return d.handle, d.pds, d.err
}

func TestResolveHandleDoesNotFallbackWhenPDSMissing(t *testing.T) {
	_, err := ResolveHandleWithDirectory(context.Background(), fakeDirectory{err: ErrPDSMissing}, "alice.test")
	if !errors.Is(err, ErrPDSMissing) {
		t.Fatalf("ResolveHandle error = %v", err)
	}
}

func TestResolveHandleReturnsIdentity(t *testing.T) {
	ident, err := ResolveHandleWithDirectory(context.Background(), fakeDirectory{
		did:    "did:plc:test",
		handle: "alice.test",
		pds:    "https://pds.example.com",
	}, "alice.test")
	if err != nil {
		t.Fatalf("ResolveHandle error = %v", err)
	}
	if ident.DID != "did:plc:test" || ident.Handle != "alice.test" || ident.PDS != "https://pds.example.com" {
		t.Fatalf("identity = %#v", ident)
	}
}

func TestResolveDIDWithDirectory(t *testing.T) {
	ident, err := ResolveDIDWithDirectory(context.Background(), fakeDirectory{
		handle: "alice.test",
		pds:    "https://pds.example.com",
	}, "did:plc:test")
	if err != nil {
		t.Fatalf("ResolveDID error = %v", err)
	}
	if ident.DID != "did:plc:test" || ident.Handle != "alice.test" || ident.PDS != "https://pds.example.com" {
		t.Fatalf("identity = %#v", ident)
	}
	_, err = ResolveDIDWithDirectory(context.Background(), fakeDirectory{err: ErrHandleNotFound}, "did:plc:test")
	if !errors.Is(err, ErrHandleNotFound) {
		t.Fatalf("ResolveDID error = %v", err)
	}
}

func TestCoreDirectoryRejectsInvalidDIDBeforeNetwork(t *testing.T) {
	dir := NewCoreDirectory()
	if dir == nil {
		t.Fatal("NewCoreDirectory returned nil")
	}
	if _, _, err := dir.ResolveDID(context.Background(), "not-a-did"); err == nil {
		t.Fatal("expected invalid DID error")
	}
	if _, err := ResolveDID(context.Background(), "not-a-did"); err == nil {
		t.Fatal("expected invalid DID error from package helper")
	}
}

func TestCoreDirectoryResolveAtIdentifierHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err := NewCoreDirectory().ResolveAtIdentifier(ctx, "onev.cat")
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}
