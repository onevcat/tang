package tangled

import (
	"strings"
	"testing"
)

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

func TestResolveIssueIdentifierErrors(t *testing.T) {
	issues := []Issue{{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/r1"}}
	if _, err := ResolveIssueIdentifier("#0", issues); err == nil || !strings.Contains(err.Error(), "greater than 0") {
		t.Fatalf("zero issue error = %v", err)
	}
	if _, err := ResolveIssueIdentifier("#2", issues); err == nil || !strings.Contains(err.Error(), "#2") {
		t.Fatalf("missing issue number error = %v", err)
	}
	if _, err := ResolveIssueIdentifier("missing", issues); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing issue rkey error = %v", err)
	}
}

func TestAssignIssueNumbersSortsByCreationTime(t *testing.T) {
	issues := []Issue{
		{Title: "third", CreatedAt: "2026-01-03T00:00:00Z"},
		{Title: "first", CreatedAt: "2026-01-01T00:00:00Z"},
		{Title: "second", CreatedAt: "2026-01-02T00:00:00Z"},
	}
	assignIssueNumbers(issues)
	for i, issue := range issues {
		if issue.Number != i+1 {
			t.Fatalf("issue %d = %#v", i, issue)
		}
	}
	if issues[0].Title != "first" || issues[2].Title != "third" {
		t.Fatalf("issues sorted as %#v", issues)
	}
}
