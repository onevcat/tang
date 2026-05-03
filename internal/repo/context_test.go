package repo

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"tangled.org/onev.cat/tang/internal/config"
	tanggit "tangled.org/onev.cat/tang/internal/git"
)

func TestSelectRemoteUsesConfiguredRemote(t *testing.T) {
	remotes := []tanggit.Remote{
		{RemoteName: "origin", Name: "origin-repo"},
		{RemoteName: "upstream", Name: "upstream-repo"},
	}
	got, err := SelectRemote(remotes, "upstream")
	if err != nil {
		t.Fatalf("SelectRemote error = %v", err)
	}
	if got.RemoteName != "upstream" {
		t.Fatalf("selected remote = %#v", got)
	}
}

func TestSelectRemoteFallsBackToOrigin(t *testing.T) {
	remotes := []tanggit.Remote{
		{RemoteName: "mirror", Name: "mirror-repo"},
		{RemoteName: "origin", Name: "origin-repo"},
	}
	got, err := SelectRemote(remotes, "missing")
	if err != nil {
		t.Fatalf("SelectRemote error = %v", err)
	}
	if got.RemoteName != "origin" {
		t.Fatalf("selected remote = %#v", got)
	}
}

func TestSelectRemoteNoRemotes(t *testing.T) {
	_, err := SelectRemote(nil, "")
	if !errors.Is(err, ErrNoRepositoryContext) {
		t.Fatalf("SelectRemote error = %v", err)
	}
}

func TestResolveSelectorOwnerRepo(t *testing.T) {
	got, err := ResolveSelector("tangled.org/core", config.Defaults())
	if err != nil {
		t.Fatalf("ResolveSelector error = %v", err)
	}
	if got.Owner != "tangled.org" || got.Name != "core" {
		t.Fatalf("context = %#v", got)
	}
	if got.Knot != config.DefaultKnotHost {
		t.Fatalf("knot = %q", got.Knot)
	}
}

func TestResolveSelectorHostOwnerRepo(t *testing.T) {
	got, err := ResolveSelector("tangled.org/onev.cat/tang", config.Defaults())
	if err != nil {
		t.Fatalf("ResolveSelector error = %v", err)
	}
	if got.Knot != "tangled.org" || got.Owner != "onev.cat" || got.Name != "tang" {
		t.Fatalf("context = %#v", got)
	}
}

func TestResolveSelectorTrimsURLAndGitSuffix(t *testing.T) {
	got, err := ResolveSelector("https://knot.example.com/onev.cat/tang.git/", config.Defaults())
	if err != nil {
		t.Fatalf("ResolveSelector error = %v", err)
	}
	if got.Knot != "knot.example.com" || got.Owner != "onev.cat" || got.Name != "tang" {
		t.Fatalf("context = %#v", got)
	}
}

func TestResolveSelectorDIDOwnerType(t *testing.T) {
	got, err := ResolveSelector("did:plc:abc/tang", config.Defaults())
	if err != nil {
		t.Fatalf("ResolveSelector error = %v", err)
	}
	if got.OwnerType != tanggit.OwnerTypeDID {
		t.Fatalf("owner type = %q", got.OwnerType)
	}
	if got.Identifier() != "did:plc:abc/tang" {
		t.Fatalf("identifier = %q", got.Identifier())
	}
}

func TestResolveSelectorRejectsInvalidInput(t *testing.T) {
	for _, input := range []string{"", "onev.cat", "onev.cat/", "host/owner/repo/extra"} {
		if _, err := ResolveSelector(input, config.Defaults()); !errors.Is(err, ErrInvalidRepositorySelector) {
			t.Fatalf("ResolveSelector(%q) error = %v", input, err)
		}
	}
}

func TestSelectRemoteFallsBackToFirstRemote(t *testing.T) {
	remotes := []tanggit.Remote{
		{RemoteName: "mirror", Owner: "onev.cat", Name: "mirror-repo", Knot: "knot.example.com"},
	}
	got, err := SelectRemote(remotes, "missing")
	if err != nil {
		t.Fatalf("SelectRemote error = %v", err)
	}
	if got.RemoteName != "mirror" || got.Identifier() != "onev.cat/mirror-repo" {
		t.Fatalf("selected remote = %#v", got)
	}
}

func TestResolveUsesLocalGitRemote(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "remote", "add", "origin", "git@tangled.org:onev.cat/tang.git")
	got, err := Resolve(context.Background(), root, config.Defaults())
	if err != nil {
		t.Fatalf("Resolve error = %v", err)
	}
	if got.Identifier() != "onev.cat/tang" || got.RemoteName != "origin" {
		t.Fatalf("context = %#v", got)
	}
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) // #nosec G204 -- fixed git binary with test-controlled arguments.
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, out, err)
	}
}
