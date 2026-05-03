package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"tangled.org/onev.cat/tang/internal/config"
	tangrepo "tangled.org/onev.cat/tang/internal/repo"
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

func TestSplitOwnerRepoAndRepoSelector(t *testing.T) {
	owner, name, err := splitOwnerRepo("/onev.cat/tang/")
	if err != nil {
		t.Fatalf("splitOwnerRepo error = %v", err)
	}
	if owner != "onev.cat" || name != "tang" {
		t.Fatalf("owner/name = %q/%q", owner, name)
	}
	if _, _, err := splitOwnerRepo("onev.cat"); err == nil {
		t.Fatal("expected invalid repository error")
	}

	cmd := &cobra.Command{Use: "test"}
	gotOwner, gotName, err := repoSelector(cmd, config.Defaults(), []string{"onev.cat/tang"})
	if err != nil {
		t.Fatalf("repoSelector args error = %v", err)
	}
	if gotOwner != "onev.cat" || gotName != "tang" {
		t.Fatalf("repoSelector = %q/%q", gotOwner, gotName)
	}

	root := &cobra.Command{Use: "root"}
	root.SetContext(context.Background())
	root.PersistentFlags().StringP("repo", "R", "", "Select another repository")
	child := &cobra.Command{Use: "child"}
	root.AddCommand(child)
	if err := root.PersistentFlags().Set("repo", "onev.cat/from-flag"); err != nil {
		t.Fatalf("Set repo flag error = %v", err)
	}
	gotOwner, gotName, err = repoSelector(child, config.Defaults(), nil)
	if err != nil {
		t.Fatalf("repoSelector flag error = %v", err)
	}
	if gotOwner != "onev.cat" || gotName != "from-flag" {
		t.Fatalf("repoSelector flag = %q/%q", gotOwner, gotName)
	}
}

func TestCurrentRepoContextWithoutRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	cmd := &cobra.Command{Use: "test"}
	cmd.SetContext(context.Background())
	if _, err := currentRepoContext(cmd, config.Defaults()); !errors.Is(err, tangrepo.ErrNoRepositoryContext) {
		t.Fatalf("currentRepoContext error = %v", err)
	}
}

func TestRenderJSONIfRequested(t *testing.T) {
	root := NewRootCommand(BuildInfo{})
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		rendered, err := renderJSONIfRequested(cmd, &RootOptions{JSONFields: "name"}, map[string]any{"name": "tang", "state": "open"})
		if err != nil {
			return err
		}
		if !rendered {
			t.Fatal("expected JSON render")
		}
		return nil
	}
	var out bytes.Buffer
	if err := executeForTest(root, &out, "--json=name"); err != nil {
		t.Fatalf("execute error = %v", err)
	}
	if !strings.Contains(out.String(), `"name": "tang"`) || strings.Contains(out.String(), "state") {
		t.Fatalf("json output = %s", out.String())
	}

	root = NewRootCommand(BuildInfo{})
	root.RunE = func(cmd *cobra.Command, _ []string) error {
		rendered, err := renderJSONIfRequested(cmd, &RootOptions{}, map[string]any{"name": "tang"})
		if err != nil {
			return err
		}
		if rendered {
			t.Fatal("did not expect JSON render")
		}
		return nil
	}
	out.Reset()
	if err := executeForTest(root, &out); err != nil {
		t.Fatalf("execute without json error = %v", err)
	}
}
