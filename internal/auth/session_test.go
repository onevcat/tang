package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSessionRequiresPDS(t *testing.T) {
	_, err := NewSession("did:plc:test", "alice.test", "", "access", "refresh")
	if !errors.Is(err, ErrMissingPDS) {
		t.Fatalf("NewSession error = %v", err)
	}
}

func TestSessionExtractsJWTExpiry(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	token := fakeJWT(map[string]any{"exp": exp})
	session, err := NewSession("did:plc:test", "alice.test", "https://pds.example.com/", token, "refresh")
	if err != nil {
		t.Fatalf("NewSession error = %v", err)
	}
	if !session.ExpiresAt.Equal(time.Unix(exp, 0)) {
		t.Fatalf("ExpiresAt = %s", session.ExpiresAt)
	}
	if session.PDS != "https://pds.example.com" {
		t.Fatalf("PDS = %q", session.PDS)
	}
}

func TestNeedsRefreshAtFiveMinuteBoundary(t *testing.T) {
	now := time.Now()
	if !NeedsRefresh(&Session{ExpiresAt: now.Add(5 * time.Minute)}, now) {
		t.Fatal("expected refresh at five minute boundary")
	}
	if NeedsRefresh(&Session{ExpiresAt: now.Add(6 * time.Minute)}, now) {
		t.Fatal("did not expect refresh with more than five minutes remaining")
	}
}

func fakeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	body, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(body)
	return strings.Join([]string{header, payload, ""}, ".")
}
