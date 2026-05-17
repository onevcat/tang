package tangled

import (
	"strings"
	"testing"
)

func TestResolveIssueIdentifierRKeyAndATURI(t *testing.T) {
	issues := []Issue{
		{Title: "Second", CreatedAt: "2026-01-02T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/def456"},
		{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/abc123"},
	}
	got, err := ResolveIssueIdentifier("abc123", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier r1 error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("r1 resolved to %#v", got)
	}
	got, err = ResolveIssueIdentifier("at://did:plc:a/sh.tangled.repo.issue/def456", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier at-uri error = %v", err)
	}
	if got.Title != "Second" {
		t.Fatalf("at-uri resolved to %#v", got)
	}
	got, err = ResolveIssueIdentifier("abc", issues)
	if err != nil {
		t.Fatalf("ResolveIssueIdentifier prefix error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("prefix resolved to %#v", got)
	}
}

func TestResolveIssueIdentifierErrors(t *testing.T) {
	issues := []Issue{{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.issue/r1"}}
	if _, err := ResolveIssueIdentifier("#0", issues); err == nil || !strings.Contains(err.Error(), "greater than 0") {
		t.Fatalf("zero issue error = %v", err)
	}
	if _, err := ResolveIssueIdentifier("#1", issues); err == nil || !strings.Contains(err.Error(), "requires AppView resolution") {
		t.Fatalf("numeric issue error = %v", err)
	}
	if _, err := ResolveIssueIdentifier("missing", issues); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing issue rkey error = %v", err)
	}
	ambiguous := []Issue{
		{Title: "First", URI: "at://did:plc:a/sh.tangled.repo.issue/abc123"},
		{Title: "Second", URI: "at://did:plc:a/sh.tangled.repo.issue/abc456"},
	}
	if _, err := ResolveIssueIdentifier("abc", ambiguous); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous issue prefix error = %v", err)
	}
}

func TestAssignIssueNumbersPreservesAppViewNumbers(t *testing.T) {
	issues := []Issue{
		{Title: "third", Number: 30, CreatedAt: "2026-01-03T00:00:00Z"},
		{Title: "first", Number: 10, CreatedAt: "2026-01-01T00:00:00Z"},
		{Title: "second", Number: 20, CreatedAt: "2026-01-02T00:00:00Z"},
	}
	assignIssueNumbers(issues)
	if issues[0].Number != 10 || issues[1].Number != 20 || issues[2].Number != 30 {
		t.Fatalf("issue numbers = %#v", issues)
	}
	if issues[0].Title != "first" || issues[2].Title != "third" {
		t.Fatalf("issues sorted as %#v", issues)
	}
}
