package cli

import (
	"testing"

	"tangled.org/onev.cat/tang/internal/config"
	tangrepo "tangled.org/onev.cat/tang/internal/repo"
	"tangled.org/onev.cat/tang/internal/tangled"
)

func TestBrowseURLs(t *testing.T) {
	cfg := config.Defaults()
	context := &tangrepo.RepositoryContext{Owner: "onev.cat", Name: "tang-playground"}
	if got := repoURL(cfg, context); got != "https://tangled.org/onev.cat/tang-playground" {
		t.Fatalf("repoURL = %q", got)
	}
	issue := tangled.Issue{Number: 12}
	if got := issueURL(cfg, context, issue); got != "https://tangled.org/onev.cat/tang-playground/issues/12" {
		t.Fatalf("issueURL = %q", got)
	}
}

func TestOpenBrowserReportsStartError(t *testing.T) {
	t.Setenv("PATH", "")
	if err := openBrowser("https://app.example.com"); err == nil {
		t.Fatal("expected browser start error")
	}
}
