package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/config"
)

func TestIsExplicitCloneURL(t *testing.T) {
	tests := map[string]bool{
		"git@tangled.org:onev.cat/tang":       true,
		"https://tangled.org/onev.cat/tang":   true,
		"ssh://git@tangled.org/onev.cat/tang": true,
		"onev.cat/tang":                       false,
		"did:plc:abc/tang":                    false,
	}
	for input, want := range tests {
		if got := isExplicitCloneURL(input); got != want {
			t.Fatalf("isExplicitCloneURL(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestCurrentRepoContextUsesRepoFlag(t *testing.T) {
	root := &cobra.Command{Use: "test"}
	root.SetContext(context.Background())
	root.PersistentFlags().StringP("repo", "R", "", "Select another repository")
	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)
	if err := root.PersistentFlags().Set("repo", "tangled.org/core"); err != nil {
		t.Fatalf("Set repo flag error = %v", err)
	}

	got, err := currentRepoContext(child, config.Defaults())
	if err != nil {
		t.Fatalf("currentRepoContext error = %v", err)
	}
	if got.Owner != "tangled.org" || got.Name != "core" {
		t.Fatalf("context = %#v", got)
	}
}
