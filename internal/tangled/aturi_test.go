package tangled

import (
	"errors"
	"testing"
)

func TestParseATURI(t *testing.T) {
	got, err := ParseATURI("at://did:plc:abc/sh.tangled.repo.issue/3abc")
	if err != nil {
		t.Fatalf("ParseATURI error = %v", err)
	}
	if got.DID != "did:plc:abc" || got.Collection != "sh.tangled.repo.issue" || got.RKey != "3abc" {
		t.Fatalf("ATURI = %#v", got)
	}
}

func TestParseATURIRejectsInvalid(t *testing.T) {
	_, err := ParseATURI("https://example.com")
	if !errors.Is(err, ErrInvalidATURI) {
		t.Fatalf("ParseATURI error = %v", err)
	}
}
