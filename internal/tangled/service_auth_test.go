package tangled

import (
	"testing"
	"time"
)

func TestBuildServiceAuthRequest(t *testing.T) {
	now := time.Unix(1000, 0)
	req := BuildServiceAuthRequest("knot.example.com", "sh.tangled.repo.create", now, 20*time.Minute)
	if req.Audience != "did:web:knot.example.com" {
		t.Fatalf("audience = %q", req.Audience)
	}
	if req.Lexicon != "sh.tangled.repo.create" {
		t.Fatalf("lexicon = %q", req.Lexicon)
	}
	if req.Expires.Unix() != 2200 {
		t.Fatalf("expires = %d", req.Expires.Unix())
	}
}
