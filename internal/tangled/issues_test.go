package tangled

import "testing"

func TestResolveIssueIdentifierNumberHashAndRKey(t *testing.T) {
	issues := []Issue{
		{Title: "Second", CreatedAt: "2026-01-02T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/r2"},
		{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/r1"},
	}
	got, err := ResolveIssueIdentifier("#1", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier #1 error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("#1 resolved to %#v", got)
	}
	got, err = ResolveIssueIdentifier("2", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier 2 error = %v", err)
	}
	if got.Title != "Second" {
		t.Fatalf("2 resolved to %#v", got)
	}
	got, err = ResolveIssueIdentifier("r1", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier r1 error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("r1 resolved to %#v", got)
	}
}
