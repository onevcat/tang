package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tangled.org/onev.cat/tang/internal/auth"

	keyringlib "github.com/zalando/go-keyring"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{Version: "v0.1.0", Commit: "abc123"})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "version"); err != nil {
		t.Fatalf("version command failed: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "v0.1.0") || !strings.Contains(got, "abc123") {
		t.Fatalf("version output = %q", got)
	}
}

func TestHelpIncludesCoreCommands(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{Version: "dev", Commit: "none"})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "--help"); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
	got := out.String()
	for _, want := range []string{"version", "completion"} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output missing %q:\n%s", want, got)
		}
	}
}

func TestConfigCommandsUseIsolatedUserConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "config", "set", "clone.protocol", "ssh"); err != nil {
		t.Fatalf("config set failed: %v", err)
	}
	if !strings.Contains(out.String(), "Set clone.protocol") {
		t.Fatalf("set output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "config", "get", "clone.protocol"); err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if strings.TrimSpace(out.String()) != "ssh" {
		t.Fatalf("get output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "config", "list", "--json=*"); err != nil {
		t.Fatalf("config list json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"clone.protocol": "ssh"`) {
		t.Fatalf("list json output = %q", out.String())
	}

	if _, err := os.Stat(t.TempDir()); err != nil {
		t.Fatalf("Stat sanity check error = %v", err)
	}
}

func TestCompletionCommandRejectsUnsupportedShell(t *testing.T) {
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "completion", "tcsh"); err == nil || !strings.Contains(err.Error(), "unsupported shell") {
		t.Fatalf("completion error = %v", err)
	}
}

func TestAuthCommandsUseMockKeyringAndFakePDS(t *testing.T) {
	keyringlib.MockInit()
	pds := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			var input struct {
				Identifier string `json:"identifier"`
				Password   string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if input.Identifier != "onev.cat" || input.Password != "password" {
				http.Error(w, "bad credentials", http.StatusBadRequest)
				return
			}
			writeJSONForCLITest(t, w, map[string]string{
				"did":        "did:plc:alice",
				"handle":     "onev.cat",
				"accessJwt":  "access",
				"refreshJwt": "refresh",
			})
		case "/xrpc/com.atproto.server.refreshSession":
			writeJSONForCLITest(t, w, map[string]string{
				"did":        "did:plc:alice",
				"handle":     "onev.cat",
				"accessJwt":  "new-access",
				"refreshJwt": "new-refresh",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(pds.Close)

	cmd := NewRootCommand(BuildInfo{})
	cmd.SetIn(strings.NewReader("password\n"))
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "--pds", pds.URL, "auth", "login", "--handle", "onev.cat"); err != nil {
		t.Fatalf("auth login failed: %v", err)
	}
	if !strings.Contains(out.String(), "Logged in as onev.cat") {
		t.Fatalf("login output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "auth", "token", "--json=*"); err != nil {
		t.Fatalf("auth token failed: %v", err)
	}
	if !strings.Contains(out.String(), `"token": "access"`) {
		t.Fatalf("token output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "auth", "refresh"); err != nil {
		t.Fatalf("auth refresh failed: %v", err)
	}
	if !strings.Contains(out.String(), "Token refreshed") {
		t.Fatalf("refresh output = %q", out.String())
	}
	session, err := auth.Load()
	if err != nil {
		t.Fatalf("Load refreshed session error = %v", err)
	}
	if session.AccessJWT != "new-access" {
		t.Fatalf("refreshed session = %#v", session)
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "auth", "status"); err != nil {
		t.Fatalf("auth status failed: %v", err)
	}
	if !strings.Contains(out.String(), "Logged in as onev.cat") {
		t.Fatalf("status output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "auth", "logout"); err != nil {
		t.Fatalf("auth logout failed: %v", err)
	}
	if !strings.Contains(out.String(), "Logged out") {
		t.Fatalf("logout output = %q", out.String())
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "auth", "status"); err != nil {
		t.Fatalf("auth status after logout failed: %v", err)
	}
	if !strings.Contains(out.String(), "(not authenticated)") {
		t.Fatalf("logged out status output = %q", out.String())
	}
}

func writeJSONForCLITest(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("Encode response error = %v", err)
	}
}

func TestStatusCommandWithMockAuthAndLocalServices(t *testing.T) {
	keyringlib.MockInit()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Chdir(t.TempDir())
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(service.Close)

	session, err := auth.NewSession("did:plc:alice", "onev.cat", "https://pds.example.com", "access", "refresh")
	if err != nil {
		t.Fatalf("NewSession error = %v", err)
	}
	if err := auth.Save(session); err != nil {
		t.Fatalf("Save session error = %v", err)
	}

	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "config", "set", "constellation.url", service.URL); err != nil {
		t.Fatalf("config set constellation failed: %v", err)
	}
	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "config", "set", "appview.url", service.URL); err != nil {
		t.Fatalf("config set appview failed: %v", err)
	}

	cmd = NewRootCommand(BuildInfo{})
	out.Reset()
	if err := executeForTest(cmd, &out, "status", "--json=*"); err != nil {
		t.Fatalf("status json failed: %v", err)
	}
	if !strings.Contains(out.String(), `"authenticated": true`) || !strings.Contains(out.String(), `"reachable": true`) {
		t.Fatalf("status json output = %q", out.String())
	}
}

func TestRepoCloneCommandAcceptsExplicitLocalURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	root := t.TempDir()
	origin := filepath.Join(root, "origin.git")
	runGitForCLITest(t, root, "init", "--bare", origin)
	target := filepath.Join(root, "clone")

	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, "repo", "clone", "file://"+origin, target); err != nil {
		t.Fatalf("repo clone failed: %v", err)
	}
	if _, err := exec.Command("git", "-C", target, "status").Output(); err != nil { // #nosec G204 -- fixed git binary with test-owned temp path.
		t.Fatalf("cloned repo status error = %v", err)
	}
}

func runGitForCLITest(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) // #nosec G204 -- fixed git binary with test-controlled arguments.
	cmd.Dir = cwd
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s: %v", args, out, err)
	}
}
