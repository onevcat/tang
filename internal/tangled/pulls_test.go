package tangled

import "testing"

func TestResolvePullIdentifier(t *testing.T) {
	pulls := []Pull{
		{Title: "Second", CreatedAt: "2026-01-02T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/r2"},
		{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/r1"},
	}
	got, err := ResolvePullIdentifier("#1", pulls)
	if err != nil {
		t.Fatalf("ResolvePullIdentifier error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("#1 resolved to %#v", got)
	}
	got, err = ResolvePullIdentifier("r2", pulls)
	if err != nil {
		t.Fatalf("ResolvePullIdentifier r2 error = %v", err)
	}
	if got.Title != "Second" {
		t.Fatalf("r2 resolved to %#v", got)
	}
}
