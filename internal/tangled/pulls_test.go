package tangled

import (
	"strings"
	"testing"

	core "tangled.org/core/api/tangled"
)

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

func TestResolvePullIdentifierErrors(t *testing.T) {
	pulls := []Pull{{Title: "First", CreatedAt: "2026-01-01T00:00:00Z", URI: "at://did:plc:a/sh.tangled.repo.pull/r1"}}
	if _, err := ResolvePullIdentifier("#2", pulls); err == nil || !strings.Contains(err.Error(), "#2") {
		t.Fatalf("missing pull number error = %v", err)
	}
	if _, err := ResolvePullIdentifier("missing", pulls); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing pull rkey error = %v", err)
	}
}

func TestAssignPullNumbersSortsByCreationTime(t *testing.T) {
	pulls := []Pull{
		{Title: "third", CreatedAt: "2026-01-03T00:00:00Z"},
		{Title: "first", CreatedAt: "2026-01-01T00:00:00Z"},
		{Title: "second", CreatedAt: "2026-01-02T00:00:00Z"},
	}
	assignPullNumbers(pulls)
	for i, pull := range pulls {
		if pull.Number != i+1 {
			t.Fatalf("pull %d = %#v", i, pull)
		}
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
