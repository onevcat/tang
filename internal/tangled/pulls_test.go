package tangled

import (
	"strings"
	"testing"

	core "tangled.org/core/api/tangled"
)

func TestResolvePullIdentifier(t *testing.T) {
	pulls := []Pull{
		{Title: "Second", CreatedAt: "2026-01-02T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/def456"},
		{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/abc123"},
	}
	got, err := ResolvePullIdentifier("def456", pulls)
	if err != nil {
		t.Fatalf("ResolvePullIdentifier r2 error = %v", err)
	}
	if got.Title != "Second" {
		t.Fatalf("r2 resolved to %#v", got)
	}
	got, err = ResolvePullIdentifier("abc", pulls)
	if err != nil {
		t.Fatalf("ResolvePullIdentifier prefix error = %v", err)
	}
	if got.Title != "First" {
		t.Fatalf("prefix resolved to %#v", got)
	}
}

func TestResolvePullIdentifierErrors(t *testing.T) {
	pulls := []Pull{{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/r1"}}
	if _, err := ResolvePullIdentifier("#1", pulls); err == nil || !strings.Contains(err.Error(), "requires AppView resolution") {
		t.Fatalf("numeric pull error = %v", err)
	}
	if _, err := ResolvePullIdentifier("missing", pulls); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing pull rkey error = %v", err)
	}
	ambiguous := []Pull{
		{Title: "First", URI: "at://did:plc:a/sh.tangled.repo.pull/abc123"},
		{Title: "Second", URI: "at://did:plc:a/sh.tangled.repo.pull/abc456"},
	}
	if _, err := ResolvePullIdentifier("abc", ambiguous); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous pull prefix error = %v", err)
	}
}

func TestAssignPullNumbersPreservesAppViewNumbers(t *testing.T) {
	pulls := []Pull{
		{Title: "third", Number: 30, CreatedAt: "2026-01-03T00:00:00Z"},
		{Title: "first", Number: 10, CreatedAt: "2026-01-01T00:00:00Z"},
		{Title: "second", Number: 20, CreatedAt: "2026-01-02T00:00:00Z"},
	}
	assignPullNumbers(pulls)
	if pulls[0].Number != 10 || pulls[1].Number != 20 || pulls[2].Number != 30 {
		t.Fatalf("pull numbers = %#v", pulls)
	}
	if pulls[0].Title != "first" || pulls[2].Title != "third" {
		t.Fatalf("pulls sorted as %#v", pulls)
	}
}

func TestPullFromRecordMapsOptionalFields(t *testing.T) {
	body := "Body"
	sourceRepo := "at://did:plc:a/sh.tangled.repo/repo"
	pull := pullFromRecord("did:plc:a", "at://did:plc:a/sh.tangled.repo.pull/r1", "cid", &core.RepoPull{
		Title:     "Title",
		Body:      &body,
		CreatedAt: "2026-01-01T00:00:00Z",
		Target:    &core.RepoPull_Target{Branch: "main"},
		Source:    &core.RepoPull_Source{Repo: &sourceRepo, Branch: "feature"},
	})
	if pull.Title != "Title" || pull.Body != body || pull.Status != "open" || pull.Author != "did:plc:a" {
		t.Fatalf("pull = %#v", pull)
	}
	if pull.Target != "main" || pull.Source != sourceRepo || pull.Branch != "feature" || pull.CID != "cid" {
		t.Fatalf("pull refs = %#v", pull)
	}
}

func TestFillTitleBodyFromPatch(t *testing.T) {
	title, body := fillTitleBodyFromPatch("From abc\nSubject: Make clone configurable\n\nPatch", "existing")
	if title != "Make clone configurable" || body != "existing" {
		t.Fatalf("filled title/body = %q/%q", title, body)
	}
	title, body = fillTitleBodyFromPatch("no subject", "")
	if title != "Pull request" || body != "" {
		t.Fatalf("fallback title/body = %q/%q", title, body)
	}
}
