package auth

import (
	"context"
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

func TestSessionMarshalRoundTrip(t *testing.T) {
	exp := time.Now().Add(time.Hour).Unix()
	session, err := NewSession("did:plc:test", "alice.test", "https://pds.example.com", fakeJWT(map[string]any{"exp": exp}), "refresh")
	if err != nil {
		t.Fatalf("NewSession error = %v", err)
	}

	data, err := session.Marshal()
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}
	got, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}
	if got.DID != session.DID || got.Handle != session.Handle || got.PDS != session.PDS || got.RefreshJWT != session.RefreshJWT {
		t.Fatalf("round-tripped session = %#v", got)
	}
	if !got.ExpiresAt.Equal(time.Unix(exp, 0)) {
		t.Fatalf("ExpiresAt = %s", got.ExpiresAt)
	}
}

func TestSessionValidationFailures(t *testing.T) {
	session := &Session{DID: "did:plc:test", Handle: "alice.test"}
	if !errors.Is(session.Validate(), ErrMissingPDS) {
		t.Fatalf("Validate missing PDS error = %v", session.Validate())
	}
	session.PDS = "https://pds.example.com"
	session.DID = ""
	if err := session.Validate(); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("Validate missing identity error = %v", err)
	}
	if _, err := Unmarshal(`{"pds":"https://pds.example.com"}`); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("Unmarshal missing identity error = %v", err)
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

func TestEnsureFreshOnlyRefreshesExpiredSessions(t *testing.T) {
	now := time.Now()
	fresh := &Session{ExpiresAt: now.Add(10 * time.Minute)}
	refresher := &fakeRefresher{next: &Session{ExpiresAt: now.Add(time.Hour)}}

	got, err := EnsureFresh(context.Background(), fresh, refresher, now)
	if err != nil {
		t.Fatalf("EnsureFresh fresh error = %v", err)
	}
	if got != fresh {
		t.Fatalf("EnsureFresh returned %#v", got)
	}
	if refresher.calls != 0 {
		t.Fatalf("refresher called %d times", refresher.calls)
	}

	expiring := &Session{ExpiresAt: now.Add(time.Minute)}
	got, err = EnsureFresh(context.Background(), expiring, refresher, now)
	if err != nil {
		t.Fatalf("EnsureFresh expiring error = %v", err)
	}
	if got != refresher.next {
		t.Fatalf("EnsureFresh refreshed session = %#v", got)
	}
	if refresher.calls != 1 {
		t.Fatalf("refresher called %d times", refresher.calls)
	}
}

type fakeRefresher struct {
	next  *Session
	err   error
	calls int
}

func (r *fakeRefresher) RefreshSession(context.Context, *Session) (*Session, error) {
	r.calls++
	return r.next, r.err
}

func fakeJWT(claims map[string]any) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	body, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(body)
	return strings.Join([]string{header, payload, ""}, ".")
}
