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
