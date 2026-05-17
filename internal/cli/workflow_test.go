package cli

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/tangled"

	keyringlib "github.com/zalando/go-keyring"
)

func TestRepositoryIssuePullAndSSHKeyCommandsWithLocalATProto(t *testing.T) {
	keyringlib.MockInit()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", "")

	patchCID := "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"
	knot := newCLIWorkflowKnot(t)
	oldDefaultTransport := http.DefaultTransport
	http.DefaultTransport = knot.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = oldDefaultTransport })
	knotHost := strings.TrimPrefix(knot.URL, "https://")
	pds := newCLIWorkflowPDS(t, map[string]cliWorkflowRecord{
		core.RepoNSID + "/r1": {
			URI:   "at://did:plc:alice/sh.tangled.repo/r1",
			CID:   "repo-cid",
			Value: `{"$type":"sh.tangled.repo","name":"tang","knot":"` + knotHost + `","repoDid":"did:plc:repo","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueNSID + "/i1": {
			URI:   "at://did:plc:alice/sh.tangled.repo.issue/i1",
			CID:   "issue-cid",
			Value: `{"$type":"sh.tangled.repo.issue","repo":"at://did:plc:alice/sh.tangled.repo/r1","title":"Issue","body":"Body","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueCommentNSID + "/ic1": {
			URI:   "at://did:plc:alice/sh.tangled.repo.issue.comment/ic1",
			CID:   "issue-comment-cid",
			Value: `{"$type":"sh.tangled.repo.issue.comment","issue":"at://did:plc:alice/sh.tangled.repo.issue/i1","body":"Comment","createdAt":"2026-05-02T00:01:00Z"}`,
		},
		core.RepoPullNSID + "/p1": {
			URI:   "at://did:plc:alice/sh.tangled.repo.pull/p1",
			CID:   "pull-cid",
			Value: `{"$type":"sh.tangled.repo.pull","title":"Pull","body":"Pull body","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"source":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"feature"},"rounds":[{"createdAt":"2026-05-02T00:00:00Z","patchBlob":{"$type":"blob","ref":{"$link":"` + patchCID + `"},"mimeType":"application/gzip","size":32}}]}`,
		},
		core.RepoPullNSID + "/p2": {
			URI:   "at://did:plc:alice/sh.tangled.repo.pull/p2",
			CID:   "pull-cid-2",
			Value: `{"$type":"sh.tangled.repo.pull","title":"Patch only","createdAt":"2026-05-02T00:01:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"rounds":[]}`,
		},
		core.PublicKeyNSID + "/key1": {
			URI:   "at://did:plc:alice/sh.tangled.publicKey/key1",
			CID:   "key-cid",
			Value: `{"$type":"sh.tangled.publicKey","name":"main","key":"ssh-ed25519 AAAA test","createdAt":"2026-05-02T00:00:00Z"}`,
		},
	}, map[string][]byte{patchCID: gzipForCLITest(t, "diff --git a/README.md b/README.md\n")})
	constellation := newCLIWorkflowConstellation(t)
	restoreResolvers := tangled.SetResolversForTesting(
		func(context.Context, string) (*atproto.Identity, error) {
			return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: pds.URL}, nil
		},
		func(context.Context, string) (*atproto.Identity, error) {
			return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: pds.URL}, nil
		},
	)
	t.Cleanup(restoreResolvers)
	oldResolveDIDForCLI := resolveDIDForCLI
	oldResolveHandleForCLI := resolveHandleForCLI
	resolveDIDForCLI = func(context.Context, string) (*atproto.Identity, error) {
		return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: pds.URL}, nil
	}
	resolveHandleForCLI = func(context.Context, string) (*atproto.Identity, error) {
		return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: pds.URL}, nil
	}
	t.Cleanup(func() {
		resolveDIDForCLI = oldResolveDIDForCLI
		resolveHandleForCLI = oldResolveHandleForCLI
	})
	var openedURLs []string
	oldOpenBrowserForCLI := openBrowserForCLI
	openBrowserForCLI = func(target string) error {
		openedURLs = append(openedURLs, target)
		return nil
	}
	t.Cleanup(func() { openBrowserForCLI = oldOpenBrowserForCLI })

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load config error = %v", err)
	}
	if err := cfg.Set("constellation.url", constellation.URL); err != nil {
		t.Fatalf("Set constellation error = %v", err)
	}
	if err := cfg.Set("knot.hosts", knotHost+",tangled.org"); err != nil {
		t.Fatalf("Set knot hosts error = %v", err)
	}
	if err := cfg.Set("appview.url", "https://app.example.com"); err != nil {
		t.Fatalf("Set appview error = %v", err)
	}
	session, err := auth.NewSession("did:plc:alice", "onev.cat", pds.URL, "access", "refresh")
	if err != nil {
		t.Fatalf("NewSession error = %v", err)
	}
	if err := auth.Save(session); err != nil {
		t.Fatalf("Save session error = %v", err)
	}

	repoRoot := t.TempDir()
	runGitForCLITest(t, repoRoot, "init")
	runGitForCLITest(t, repoRoot, "remote", "add", "origin", "git@tangled.org:onev.cat/tang.git")
	t.Chdir(repoRoot)

	assertCLIWorkflowOutput(t, "repo list", []string{"repo", "list", "onev.cat"}, "onev.cat/tang")
	assertCLIWorkflowOutput(t, "repo list authenticated", []string{"repo", "list"}, "onev.cat/tang")
	assertCLIWorkflowOutput(t, "repo view", []string{"repo", "view", "onev.cat/tang"}, "SSH: git@"+knotHost+":onev.cat/tang")
	assertCLIWorkflowOutput(t, "repo create", []string{"repo", "create", "new-repo"}, "Created repository onev.cat/new-repo")
	assertCLIWorkflowOutput(t, "issue list", []string{"issue", "list", "--state", "all"}, "#1\tIssue")
	assertCLIWorkflowOutput(t, "issue view", []string{"issue", "view", "#1"}, "Comment by did:plc:alice")
	assertCLIWorkflowOutput(t, "issue view web", []string{"issue", "view", "#1", "--web"}, "")
	assertCLIWorkflowOutput(t, "issue create", []string{"issue", "create", "New", "--body", "New body"}, "Created issue")
	assertCLIWorkflowOutput(t, "issue close", []string{"issue", "close", "#1"}, "Issue #1 is now closed")
	assertCLIWorkflowOutput(t, "issue edit", []string{"issue", "edit", "#1", "--title", "Updated"}, "Updated issue")
	assertCLIWorkflowOutput(t, "issue comment", []string{"issue", "comment", "#1", "--body", "hello"}, "Commented on issue #1")
	assertCLIWorkflowOutput(t, "pr list", []string{"pr", "list", "--state", "all"}, "#1\tPull")
	assertCLIWorkflowOutput(t, "pr view web", []string{"pr", "view", "#1", "--web"}, "")
	assertCLIWorkflowOutput(t, "pr view", []string{"pr", "view", "#1"}, "Merge: clean")
	assertCLIWorkflowOutput(t, "pr create", []string{"pr", "create", "--title", "New PR", "--head", "feature"}, "Created pull")
	assertCLIWorkflowOutput(t, "pr diff", []string{"pr", "diff", "#1"}, "diff --git")
	assertCLIWorkflowOutput(t, "pr close", []string{"pr", "close", "#1"}, "Pull #1 is now closed")
	assertCLIWorkflowOutput(t, "pr comment", []string{"pr", "comment", "#1", "--body", "hello"}, "Commented on pull #1")
	assertCLIWorkflowOutput(t, "ssh-key list", []string{"ssh-key", "list"}, "key1\tmain")

	keyPath := filepath.Join(t.TempDir(), "id_ed25519.pub")
	if err := os.WriteFile(keyPath, []byte("ssh-ed25519 BBBB new-key"), 0o600); err != nil {
		t.Fatalf("WriteFile key error = %v", err)
	}
	assertCLIWorkflowOutput(t, "ssh-key add", []string{"ssh-key", "add", keyPath}, "Added SSH key")
	assertCLIWorkflowOutput(t, "ssh-key delete", []string{"ssh-key", "delete", "key1"}, "Deleted SSH key key1")

	assertCLIWorkflowError(t, "pr merge", []string{"pr", "merge", "#1"})
	assertCLIWorkflowError(t, "pr checkout patch-only", []string{"pr", "checkout", "#2"})
	assertCLIWorkflowError(t, "pr diff no rounds", []string{"pr", "diff", "#2"})
	assertCLIWorkflowError(t, "issue view missing", []string{"issue", "view", "missing"})
	assertCLIWorkflowError(t, "pr diff missing", []string{"pr", "diff", "missing"})
	assertCLIWorkflowOutput(t, "browse", []string{"browse"}, "")
	assertCLIWorkflowOutput(t, "browse issue", []string{"browse", "issue", "#1"}, "")
	if len(openedURLs) != 4 {
		t.Fatalf("opened URLs = %#v", openedURLs)
	}

	browseCmd := NewRootCommand(BuildInfo{})
	browseCmd.SetContext(context.Background())
	cfgForBrowse, contextForBrowse, err := browseContext(browseCmd)
	if err != nil {
		t.Fatalf("browseContext error = %v", err)
	}
	if cfgForBrowse.AppView.URL != "https://app.example.com" || contextForBrowse.Identifier() != "onev.cat/tang" {
		t.Fatalf("browse context = %#v %#v", cfgForBrowse, contextForBrowse)
	}
}

func assertCLIWorkflowOutput(t *testing.T, name string, args []string, want string) {
	t.Helper()
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, args...); err != nil {
		t.Fatalf("%s failed: %v\noutput:\n%s", name, err, out.String())
	}
	if !strings.Contains(out.String(), want) {
		t.Fatalf("%s output missing %q:\n%s", name, want, out.String())
	}
}

func assertCLIWorkflowError(t *testing.T, name string, args []string) {
	t.Helper()
	cmd := NewRootCommand(BuildInfo{})
	var out bytes.Buffer
	if err := executeForTest(cmd, &out, args...); err == nil {
		t.Fatalf("%s unexpectedly succeeded:\n%s", name, out.String())
	}
}

type cliWorkflowRecord struct {
	URI   string
	CID   string
	Value string
}

func newCLIWorkflowPDS(t *testing.T, records map[string]cliWorkflowRecord, blobs map[string][]byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.repo.listRecords":
			collection := r.URL.Query().Get("collection")
			out := struct {
				Records []json.RawMessage `json:"records"`
			}{}
			for key, record := range records {
				if strings.HasPrefix(key, collection+"/") {
					out.Records = append(out.Records, cliWorkflowRecordJSON(t, record))
				}
			}
			writeJSONForCLITest(t, w, out)
		case "/xrpc/com.atproto.repo.getRecord":
			key := r.URL.Query().Get("collection") + "/" + r.URL.Query().Get("rkey")
			record, ok := records[key]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(cliWorkflowRecordJSON(t, record))
		case "/xrpc/com.atproto.repo.createRecord":
			var input struct {
				Collection string `json:"collection"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSONForCLITest(t, w, map[string]string{"uri": "at://did:plc:alice/" + input.Collection + "/created", "cid": "created-cid"})
		case "/xrpc/com.atproto.repo.putRecord":
			var input struct {
				Collection string `json:"collection"`
				RKey       string `json:"rkey"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			writeJSONForCLITest(t, w, map[string]string{"uri": "at://did:plc:alice/" + input.Collection + "/" + input.RKey, "cid": "put-cid"})
		case "/xrpc/com.atproto.repo.deleteRecord":
			writeJSONForCLITest(t, w, map[string]any{})
		case "/xrpc/com.atproto.repo.uploadBlob":
			_, _ = io.ReadAll(r.Body)
			writeJSONForCLITest(t, w, map[string]any{
				"blob": map[string]any{
					"$type":    "blob",
					"ref":      map[string]string{"$link": "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"},
					"mimeType": "application/gzip",
					"size":     32,
				},
			})
		case "/xrpc/com.atproto.server.getServiceAuth":
			writeJSONForCLITest(t, w, map[string]string{"token": "service-token"})
		case "/xrpc/com.atproto.sync.getBlob":
			data, ok := blobs[r.URL.Query().Get("cid")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(data)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newCLIWorkflowKnot(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/sh.tangled.repo.create":
			writeJSONForCLITest(t, w, map[string]string{"repoDid": "did:plc:repo"})
		case "/xrpc/sh.tangled.repo.compare":
			_, _ = w.Write([]byte(`{"patch":"From abc\nSubject: New PR\n\nPatch"}`))
		case "/xrpc/sh.tangled.repo.merge":
			writeJSONForCLITest(t, w, map[string]any{})
		case "/xrpc/sh.tangled.repo.mergeCheck":
			writeJSONForCLITest(t, w, map[string]any{"is_conflicted": false})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func cliWorkflowRecordJSON(t *testing.T, record cliWorkflowRecord) []byte {
	t.Helper()
	data, err := json.Marshal(struct {
		URI   string          `json:"uri"`
		CID   string          `json:"cid"`
		Value json.RawMessage `json:"value"`
	}{URI: record.URI, CID: record.CID, Value: json.RawMessage(record.Value)})
	if err != nil {
		t.Fatalf("Marshal record error = %v", err)
	}
	return data
}

func newCLIWorkflowConstellation(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		collection := r.URL.Query().Get("collection")
		var links []map[string]string
		switch collection {
		case core.RepoIssueNSID:
			links = append(links, map[string]string{"did": "did:plc:alice", "collection": core.RepoIssueNSID, "rkey": "i1"})
		case core.RepoIssueCommentNSID:
			links = append(links, map[string]string{"did": "did:plc:alice", "collection": core.RepoIssueCommentNSID, "rkey": "ic1"})
		case core.RepoPullNSID:
			links = append(links, map[string]string{"did": "did:plc:alice", "collection": core.RepoPullNSID, "rkey": "p1"})
			links = append(links, map[string]string{"did": "did:plc:alice", "collection": core.RepoPullNSID, "rkey": "p2"})
		}
		writeJSONForCLITest(t, w, map[string]any{"total": len(links), "linking_records": links})
	}))
	t.Cleanup(server.Close)
	return server
}

func gzipForCLITest(t *testing.T, input string) []byte {
	t.Helper()
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	if _, err := io.WriteString(zw, input); err != nil {
		t.Fatalf("gzip write error = %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close error = %v", err)
	}
	return out.Bytes()
}
