package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestPrintStatusAndExpiryFormatting(t *testing.T) {
	result := statusResult{
		Authentication: authStatus{
			Authenticated: true,
			Handle:        "onev.cat",
			DID:           "did:plc:alice",
			PDS:           "https://pds.example.com",
			ExpiresAt:     time.Now().Add(time.Hour),
		},
		Repository: repoStatus{
			Detected:   true,
			Identifier: "onev.cat/tang",
			Knot:       "knot1.tangled.sh",
			Remote:     "origin",
			URLKind:    "hosted",
		},
		Services: []serviceCheck{
			{Name: "Constellation", URL: "https://constellation.example.com", Reachable: true},
			{Name: "AppView", URL: "https://app.example.com", Reachable: false},
		},
	}
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := printStatus(cmd, result); err != nil {
		t.Fatalf("printStatus error = %v", err)
	}
	got := out.String()
	for _, want := range []string{"Logged in as onev.cat", "onev.cat/tang", "reachable", "unreachable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status output missing %q:\n%s", want, got)
		}
	}
	if formatExpiry(time.Time{}) != "unknown" {
		t.Fatalf("zero expiry = %q", formatExpiry(time.Time{}))
	}
	if formatExpiry(time.Now().Add(-time.Minute)) != "expired" {
		t.Fatalf("expired = %q", formatExpiry(time.Now().Add(-time.Minute)))
	}
}

func TestPrintStatusUnauthenticatedAndNoRepo(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	cmd.SetOut(&out)
	err := printStatus(cmd, statusResult{
		Services: []serviceCheck{{Name: "AppView", URL: "https://app.example.com"}},
	})
	if err != nil {
		t.Fatalf("printStatus error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "(not authenticated)") || !strings.Contains(got, "(not in a Tangled repository)") {
		t.Fatalf("status output = %s", got)
	}
}
