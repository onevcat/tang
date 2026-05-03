package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseRemoteURLSupportsConfiguredKnotHosts(t *testing.T) {
	got, ok := ParseRemoteURL("git@knot.example.com:onev.cat/repo.git", []string{"tangled.org", "knot.example.com"})
	if !ok {
		t.Fatal("expected configured knot host to parse")
	}
	if got.Knot != "knot.example.com" || got.Owner != "onev.cat" || got.Name != "repo" || got.Protocol != ProtocolSSH {
		t.Fatalf("remote = %#v", got)
	}
}

func TestParseRemoteURLRejectsUnconfiguredHost(t *testing.T) {
	if _, ok := ParseRemoteURL("git@github.com:onevcat/repo.git", []string{"tangled.org"}); ok {
		t.Fatal("expected unconfigured host to be rejected")
	}
}

func TestParseRemoteListPrefersFetchOverPush(t *testing.T) {
	input := []byte("origin\tgit@tangled.org:onev.cat/fetch-repo.git (fetch)\norigin\tgit@tangled.org:onev.cat/push-repo.git (push)\n")
	got := ParseRemoteList(input, []string{"tangled.org"})
	if len(got) != 1 {
		t.Fatalf("remote count = %d", len(got))
	}
	if got[0].Name != "fetch-repo" || got[0].URLKind != URLKindFetch {
		t.Fatalf("remote = %#v", got[0])
	}
}

func TestParseRemoteListFallsBackToPush(t *testing.T) {
	input := []byte("origin\tgit@github.com:onevcat/repo.git (fetch)\norigin\tgit@tangled.org:onev.cat/repo.git (push)\n")
	got := ParseRemoteList(input, []string{"tangled.org"})
	if len(got) != 1 {
		t.Fatalf("remote count = %d", len(got))
	}
	if got[0].RemoteName != "origin" || got[0].URLKind != URLKindPush || got[0].Knot != "tangled.org" {
		t.Fatalf("remote = %#v", got[0])
	}
}

func TestParseRemoteURLSupportsHTTPSAndDIDOwner(t *testing.T) {
	got, ok := ParseRemoteURL("https://tangled.org/did:plc:abc123/repo", []string{"tangled.org"})
	if !ok {
		t.Fatal("expected https remote to parse")
	}
	if got.OwnerType != OwnerTypeDID || got.Protocol != ProtocolHTTPS {
		t.Fatalf("remote = %#v", got)
	}
}

func TestListTangledRemotesWithRunner(t *testing.T) {
	runner := &fakeRunner{out: []byte("origin\tgit@tangled.org:onev.cat/tang.git (fetch)\n")}
	got, err := ListTangledRemotesWithRunner(context.Background(), "/tmp", []string{"tangled.org"}, runner)
	if err != nil {
		t.Fatalf("ListTangledRemotesWithRunner error = %v", err)
	}
	if len(got) != 1 || got[0].RemoteName != "origin" {
		t.Fatalf("remotes = %#v", got)
	}
	if runner.args[0] != "remote" || runner.args[1] != "-v" {
		t.Fatalf("runner args = %#v", runner.args)
	}

	got, err = ListTangledRemotesWithRunner(context.Background(), "/tmp", []string{"tangled.org"}, &fakeRunner{err: errors.New("git failed")})
	if err != nil {
		t.Fatalf("erroring runner should be ignored, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("remotes on error = %#v", got)
	}
}

func TestListTangledRemotesUsesGitRunner(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "remote", "add", "origin", "git@tangled.org:onev.cat/tang.git")
	got, err := ListTangledRemotes(context.Background(), root, []string{"tangled.org"})
	if err != nil {
		t.Fatalf("ListTangledRemotes error = %v", err)
	}
	if len(got) != 1 || got[0].RemoteName != "origin" {
		t.Fatalf("remotes = %#v", got)
	}
}

func TestCurrentBranch(t *testing.T) {
	branch, err := CurrentBranch(context.Background(), "/tmp", &fakeRunner{out: []byte("feature\n")})
	if err != nil {
		t.Fatalf("CurrentBranch error = %v", err)
	}
	if branch != "feature" {
		t.Fatalf("branch = %q", branch)
	}
	if _, err := CurrentBranch(context.Background(), "/tmp", &fakeRunner{out: []byte("\n")}); err == nil {
		t.Fatal("expected detached branch error")
	}
}

func TestCloneUsesLocalRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	runGit(t, root, "init", "--bare", origin)
	target := filepath.Join(root, "clone")
	if err := Clone(context.Background(), origin, target); err != nil {
		t.Fatalf("Clone error = %v", err)
	}
	if _, err := exec.Command("git", "-C", target, "status").Output(); err != nil {
		t.Fatalf("cloned repo status error = %v", err)
	}
}

func TestCheckoutBranchFromRemoteUsesFetchHead(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote")
	work := filepath.Join(root, "work")
	runGit(t, root, "init", remote)
	runGit(t, remote, "config", "user.email", "test@example.com")
	runGit(t, remote, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	runGit(t, remote, "add", "README.md")
	runGit(t, remote, "commit", "-m", "initial")
	runGit(t, remote, "checkout", "-b", "feature")
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("feature"), 0o600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}
	runGit(t, remote, "commit", "-am", "feature")

	runGit(t, root, "init", work)
	runGit(t, work, "remote", "add", "origin", remote)
	if err := CheckoutBranchFromRemote(context.Background(), work, "origin", "feature"); err != nil {
		t.Fatalf("CheckoutBranchFromRemote error = %v", err)
	}
	branch, err := CurrentBranch(context.Background(), work, GitRunner{})
	if err != nil {
		t.Fatalf("CurrentBranch error = %v", err)
	}
	if branch != "feature" {
		t.Fatalf("branch = %q", branch)
	}
}

type fakeRunner struct {
	out  []byte
	err  error
	args []string
}

func (r *fakeRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	r.args = args
	return r.out, r.err
}

func runGit(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, out, err)
	}
}
