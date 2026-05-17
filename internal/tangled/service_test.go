package tangled

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	localrepo "tangled.org/onev.cat/tang/internal/repo"
)

func TestRepoServiceListAndGetRepoUseResolvedPDS(t *testing.T) {
	const did = "did:plc:alice"
	server := newFakePDSServer(t, map[string]fakeRecord{
		core.RepoNSID + "/r1": {
			URI:        "at://did:plc:alice/sh.tangled.repo/r1",
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"tang","knot":"knot1.tangled.sh","createdAt":"2026-05-02T00:00:00Z","description":"CLI"}`,
		},
	})
	stubResolvers(t, did, "onev.cat", server.URL)

	service := NewRepoService(config.Defaults(), server.Client())
	repos, err := service.ListRepos(context.Background(), "onev.cat")
	if err != nil {
		t.Fatalf("ListRepos error = %v", err)
	}
	if len(repos) != 1 {
		t.Fatalf("repos = %#v", repos)
	}
	if repos[0].Owner != "onev.cat" || repos[0].Name != "tang" || repos[0].Description != "CLI" {
		t.Fatalf("repo = %#v", repos[0])
	}

	repo, err := service.GetRepo(context.Background(), "onev.cat", "tang")
	if err != nil {
		t.Fatalf("GetRepo error = %v", err)
	}
	if repo.CloneSSH != "git@tangled.org:onev.cat/tang" {
		t.Fatalf("CloneSSH = %q", repo.CloneSSH)
	}
	if _, err := service.GetRepo(context.Background(), "onev.cat", "missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("GetRepo missing error = %v", err)
	}
}

func TestIssueServiceCreateGetStateAndComments(t *testing.T) {
	const did = "did:plc:alice"
	records := map[string]fakeRecord{
		core.RepoNSID + "/r1": {
			URI:        "at://did:plc:alice/sh.tangled.repo/r1",
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"tang","knot":"knot1.tangled.sh","repoDid":"did:plc:repo","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueNSID + "/i1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/i1",
			CID:        "issue-cid",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"at://did:plc:alice/sh.tangled.repo/r1","title":"Existing","body":"Body","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueStateNSID + "/s1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue.state/s1",
			CID:        "state-cid",
			Lexicon:    core.RepoIssueStateNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue.state","issue":"at://did:plc:alice/sh.tangled.repo.issue/i1","state":"sh.tangled.repo.issue.state.closed"}`,
		},
		core.RepoIssueCommentNSID + "/c1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue.comment/c1",
			CID:        "comment-1",
			Lexicon:    core.RepoIssueCommentNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue.comment","issue":"at://did:plc:alice/sh.tangled.repo.issue/i1","body":"second","createdAt":"2026-05-02T00:02:00Z"}`,
		},
		core.RepoIssueCommentNSID + "/c2": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue.comment/c2",
			CID:        "comment-2",
			Lexicon:    core.RepoIssueCommentNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue.comment","issue":"at://did:plc:alice/sh.tangled.repo.issue/i1","body":"first","createdAt":"2026-05-02T00:01:00Z"}`,
		},
	}
	var createdIssue map[string]any
	pds := newFakePDSServer(t, records, withCreateRecordHook(func(collection string, record json.RawMessage) {
		if collection != core.RepoIssueNSID {
			return
		}
		if err := json.Unmarshal(record, &createdIssue); err != nil {
			t.Fatalf("Unmarshal created issue error = %v", err)
		}
	}))
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoIssueStateNSID: {
			{DID: did, Collection: core.RepoIssueStateNSID, RKey: "s1"},
		},
		core.RepoIssueCommentNSID: {
			{DID: did, Collection: core.RepoIssueCommentNSID, RKey: "c1"},
			{DID: did, Collection: core.RepoIssueCommentNSID, RKey: "c2"},
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)

	service := NewIssueService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())
	issue, err := service.GetIssue(context.Background(), "at://did:plc:alice/sh.tangled.repo.issue/i1")
	if err != nil {
		t.Fatalf("GetIssue error = %v", err)
	}
	if issue.Title != "Existing" || issue.Body != "Body" || issue.State != "open" || issue.CID != "issue-cid" {
		t.Fatalf("issue = %#v", issue)
	}
	state, err := service.GetIssueState(context.Background(), issue.URI)
	if err != nil {
		t.Fatalf("GetIssueState error = %v", err)
	}
	if state != "closed" {
		t.Fatalf("state = %q", state)
	}
	comments, err := service.ListComments(context.Background(), issue.URI)
	if err != nil {
		t.Fatalf("ListComments error = %v", err)
	}
	if len(comments) != 2 || comments[0].Body != "first" || comments[1].Body != "second" {
		t.Fatalf("comments = %#v", comments)
	}

	session := &auth.Session{DID: did, Handle: "onev.cat", PDS: pds.URL, AccessJWT: "access", RefreshJWT: "refresh"}
	created, err := service.CreateIssue(context.Background(), session, "at://did:plc:alice/sh.tangled.repo/r1", "New", "New body")
	if err != nil {
		t.Fatalf("CreateIssue error = %v", err)
	}
	if created.Title != "New" || created.Body != "New body" || created.State != "open" || created.URI == "" {
		t.Fatalf("created issue = %#v", created)
	}
	if createdIssue["repo"] != "did:plc:repo" {
		t.Fatalf("created issue repo = %#v", createdIssue)
	}
	comment, err := service.AddComment(context.Background(), session, issue.URI, "hello")
	if err != nil {
		t.Fatalf("AddComment error = %v", err)
	}
	if comment.Body != "hello" || comment.Author != did || comment.URI == "" {
		t.Fatalf("comment = %#v", comment)
	}
	if err := service.SetIssueState(context.Background(), session, issue.URI, "closed"); err != nil {
		t.Fatalf("SetIssueState error = %v", err)
	}
}

func TestIssueServiceListAndUpdateIssue(t *testing.T) {
	const did = "did:plc:alice"
	records := map[string]fakeRecord{
		core.RepoIssueNSID + "/i1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/i1",
			CID:        "issue-cid-1",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"at://did:plc:alice/sh.tangled.repo/r1","title":"First","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueNSID + "/i2": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/i2",
			CID:        "issue-cid-2",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"at://did:plc:alice/sh.tangled.repo/r1","title":"Second","createdAt":"2026-05-02T00:01:00Z"}`,
		},
		core.RepoIssueStateNSID + "/s1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue.state/s1",
			CID:        "state-cid",
			Lexicon:    core.RepoIssueStateNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue.state","issue":"at://did:plc:alice/sh.tangled.repo.issue/i2","state":"sh.tangled.repo.issue.state.closed"}`,
		},
	}
	pds := newFakePDSServer(t, records)
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoIssueNSID: {
			{DID: did, Collection: core.RepoIssueNSID, RKey: "i2"},
			{DID: did, Collection: core.RepoIssueNSID, RKey: "i1"},
		},
		core.RepoIssueStateNSID: {
			{DID: did, Collection: core.RepoIssueStateNSID, RKey: "s1"},
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewIssueService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())

	issues, err := service.ListIssues(context.Background(), "at://did:plc:alice/sh.tangled.repo/r1", IssueListOptions{State: "closed"})
	if err != nil {
		t.Fatalf("ListIssues error = %v", err)
	}
	if len(issues) != 2 || issues[0].Title != "First" || issues[1].Title != "Second" || issues[1].RKey != "i2" || issues[0].State != "closed" {
		t.Fatalf("issues = %#v", issues)
	}

	session := &auth.Session{DID: did, Handle: "onev.cat", PDS: pds.URL, AccessJWT: "access", RefreshJWT: "refresh"}
	updated, err := service.UpdateIssue(context.Background(), session, "at://did:plc:alice/sh.tangled.repo.issue/i1", "Updated", "", true, false)
	if err != nil {
		t.Fatalf("UpdateIssue error = %v", err)
	}
	if updated.Title != "Updated" || updated.Body != "" || updated.CID != "put-cid" {
		t.Fatalf("updated issue = %#v", updated)
	}
	other := *session
	other.DID = "did:plc:other"
	if _, err := service.UpdateIssue(context.Background(), &other, "at://did:plc:alice/sh.tangled.repo.issue/i1", "Updated", "", true, false); err == nil || !strings.Contains(err.Error(), "cannot edit") {
		t.Fatalf("UpdateIssue other author error = %v", err)
	}
}

func TestIssueServiceListIssuesQueriesRepoDIDAndLegacyATURI(t *testing.T) {
	const did = "did:plc:alice"
	const repoURI = "at://did:plc:alice/sh.tangled.repo/r1"
	const repoDID = "did:plc:repo"
	records := map[string]fakeRecord{
		core.RepoNSID + "/r1": {
			URI:        repoURI,
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"tang","knot":"knot1.tangled.sh","repoDid":"` + repoDID + `","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueNSID + "/legacy": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/legacy",
			CID:        "legacy-cid",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"` + repoURI + `","title":"Legacy","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoIssueNSID + "/canonical": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/canonical",
			CID:        "canonical-cid",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"` + repoDID + `","title":"Canonical","createdAt":"2026-05-02T00:01:00Z"}`,
		},
	}
	pds := newFakePDSServer(t, records)
	var issueTargets []string
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoIssueNSID: {
			{DID: did, Collection: core.RepoIssueNSID, RKey: "canonical", Target: repoDID},
			{DID: did, Collection: core.RepoIssueNSID, RKey: "canonical", Target: repoURI},
			{DID: did, Collection: core.RepoIssueNSID, RKey: "legacy", Target: repoURI},
		},
	}, withConstellationQueryHook(func(collection, target, _ string) {
		if collection == core.RepoIssueNSID {
			issueTargets = append(issueTargets, target)
		}
	}))
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewIssueService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())

	issues, err := service.ListIssues(context.Background(), repoURI, IssueListOptions{State: "all"})
	if err != nil {
		t.Fatalf("ListIssues error = %v", err)
	}
	if len(issues) != 2 || issues[0].Title != "Legacy" || issues[1].Title != "Canonical" {
		t.Fatalf("issues = %#v", issues)
	}
	if !containsString(issueTargets, repoDID) || !containsString(issueTargets, repoURI) {
		t.Fatalf("issue targets = %#v", issueTargets)
	}
}

func TestIssueServiceResolveIssueNumberFromAppView(t *testing.T) {
	const did = "did:plc:alice"
	records := map[string]fakeRecord{
		core.RepoIssueNSID + "/i4": {
			URI:        "at://did:plc:alice/sh.tangled.repo.issue/i4",
			CID:        "issue-cid",
			Lexicon:    core.RepoIssueNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.issue","repo":"did:plc:repo","title":"AppView issue","createdAt":"2026-05-02T00:00:00Z"}`,
		},
	}
	pds := newFakePDSServer(t, records)
	appview := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/onev.cat/tang/issues/4" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `<span data-aturi="at://did:plc:alice/sh.tangled.repo.issue/i4">issue</span>`)
	}))
	t.Cleanup(appview.Close)
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewIssueService(config.Defaults(), pds.Client())

	issue, err := service.ResolveIssueNumber(context.Background(), appview.URL, "onev.cat", "tang", 4)
	if err != nil {
		t.Fatalf("ResolveIssueNumber error = %v", err)
	}
	if issue.Number != 4 || issue.URI != "at://did:plc:alice/sh.tangled.repo.issue/i4" || issue.Title != "AppView issue" {
		t.Fatalf("issue = %#v", issue)
	}
}

func TestPullServiceGetStatusPatchAndMergeCheck(t *testing.T) {
	const did = "did:plc:alice"
	patch := "From abc\nSubject: Change README\n\nPatch body"
	blobCID := "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"
	records := map[string]fakeRecord{
		core.RepoPullNSID + "/p1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/p1",
			CID:        "pull-cid",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"Pull","body":"Body","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"source":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"feature"},"rounds":[{"createdAt":"2026-05-02T00:00:00Z","patchBlob":{"$type":"blob","ref":{"$link":"` + blobCID + `"},"mimeType":"application/gzip","size":32}}]}`,
		},
		core.RepoPullStatusNSID + "/s1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull.status/s1",
			CID:        "status-cid",
			Lexicon:    core.RepoPullStatusNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull.status","pull":"at://did:plc:alice/sh.tangled.repo.pull/p1","status":"sh.tangled.repo.pull.status.merged"}`,
		},
	}
	pds := newFakePDSServer(t, records, withBlob(blobCID, gzipBytes(t, patch)))
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoPullStatusNSID: {
			{DID: did, Collection: core.RepoPullStatusNSID, RKey: "s1"},
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)

	service := NewPullService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())
	pull, err := service.GetPull(context.Background(), "at://did:plc:alice/sh.tangled.repo.pull/p1")
	if err != nil {
		t.Fatalf("GetPull error = %v", err)
	}
	if pull.Title != "Pull" || pull.Branch != "feature" || pull.Target != "main" || pull.CID != "pull-cid" {
		t.Fatalf("pull = %#v", pull)
	}
	status, err := service.GetPullStatus(context.Background(), pull.URI)
	if err != nil {
		t.Fatalf("GetPullStatus error = %v", err)
	}
	if status != "merged" {
		t.Fatalf("status = %q", status)
	}
	gotPatch, err := service.FetchPullPatch(context.Background(), pull.URI)
	if err != nil {
		t.Fatalf("FetchPullPatch error = %v", err)
	}
	if gotPatch != patch {
		t.Fatalf("patch = %q", gotPatch)
	}
}

func TestPullServiceListAndMutations(t *testing.T) {
	const did = "did:plc:alice"
	records := map[string]fakeRecord{
		core.RepoPullNSID + "/p1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/p1",
			CID:        "pull-cid-1",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"First","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"source":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"feature-a"},"rounds":[]}`,
		},
		core.RepoPullNSID + "/p2": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/p2",
			CID:        "pull-cid-2",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"Second","createdAt":"2026-05-02T00:01:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"source":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"feature-b"},"rounds":[]}`,
		},
		core.RepoPullStatusNSID + "/s1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull.status/s1",
			CID:        "status-cid",
			Lexicon:    core.RepoPullStatusNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull.status","pull":"at://did:plc:alice/sh.tangled.repo.pull/p2","status":"sh.tangled.repo.pull.status.closed"}`,
		},
	}
	pds := newFakePDSServer(t, records)
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoPullNSID: {
			{DID: did, Collection: core.RepoPullNSID, RKey: "p2"},
			{DID: did, Collection: core.RepoPullNSID, RKey: "p1"},
		},
		core.RepoPullStatusNSID: {
			{DID: did, Collection: core.RepoPullStatusNSID, RKey: "s1"},
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewPullService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())

	pulls, err := service.ListPulls(context.Background(), "at://did:plc:alice/sh.tangled.repo/r1", "closed", 0)
	if err != nil {
		t.Fatalf("ListPulls error = %v", err)
	}
	if len(pulls) != 2 || pulls[0].Title != "First" || pulls[1].Title != "Second" || pulls[1].RKey != "p2" || pulls[0].Status != "closed" {
		t.Fatalf("pulls = %#v", pulls)
	}
	if _, err := service.FetchPullPatch(context.Background(), "at://did:plc:alice/sh.tangled.repo.pull/p1"); err == nil || !strings.Contains(err.Error(), "no rounds") {
		t.Fatalf("FetchPullPatch no rounds error = %v", err)
	}

	session := &auth.Session{DID: did, Handle: "onev.cat", PDS: pds.URL, AccessJWT: "access", RefreshJWT: "refresh"}
	comment, err := service.AddPullComment(context.Background(), session, "at://did:plc:alice/sh.tangled.repo.pull/p1", "hello")
	if err != nil {
		t.Fatalf("AddPullComment error = %v", err)
	}
	if comment.Body != "hello" || comment.Author != did || comment.URI == "" {
		t.Fatalf("comment = %#v", comment)
	}
	if err := service.SetPullStatus(context.Background(), session, "at://did:plc:alice/sh.tangled.repo.pull/p1", "merged"); err != nil {
		t.Fatalf("SetPullStatus error = %v", err)
	}
}

func TestPullServiceListPullsQueriesRepoDIDAndLegacyATURI(t *testing.T) {
	const did = "did:plc:alice"
	const repoURI = "at://did:plc:alice/sh.tangled.repo/r1"
	const repoDID = "did:plc:repo"
	records := map[string]fakeRecord{
		core.RepoNSID + "/r1": {
			URI:        repoURI,
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"tang","knot":"knot1.tangled.sh","repoDid":"` + repoDID + `","createdAt":"2026-05-02T00:00:00Z"}`,
		},
		core.RepoPullNSID + "/legacy": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/legacy",
			CID:        "legacy-cid",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"Legacy","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"` + repoURI + `","branch":"main"},"source":{"repo":"` + repoURI + `","branch":"feature-a"},"rounds":[]}`,
		},
		core.RepoPullNSID + "/canonical": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/canonical",
			CID:        "canonical-cid",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"Canonical","createdAt":"2026-05-02T00:01:00Z","target":{"repo":"` + repoDID + `","branch":"main"},"source":{"repo":"` + repoDID + `","branch":"feature-b"},"rounds":[]}`,
		},
	}
	pds := newFakePDSServer(t, records)
	var pullTargets []string
	constellation := newFakeConstellationServer(t, map[string][]fakeLink{
		core.RepoPullNSID: {
			{DID: did, Collection: core.RepoPullNSID, RKey: "canonical", Target: repoDID},
			{DID: did, Collection: core.RepoPullNSID, RKey: "canonical", Target: repoURI},
			{DID: did, Collection: core.RepoPullNSID, RKey: "legacy", Target: repoURI},
		},
	}, withConstellationQueryHook(func(collection, target, _ string) {
		if collection == core.RepoPullNSID {
			pullTargets = append(pullTargets, target)
		}
	}))
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewPullService(&config.Config{Constellation: config.ConstellationConfig{URL: constellation.URL}}, pds.Client())

	pulls, err := service.ListPulls(context.Background(), repoURI, "all", 0)
	if err != nil {
		t.Fatalf("ListPulls error = %v", err)
	}
	if len(pulls) != 2 || pulls[0].Title != "Legacy" || pulls[1].Title != "Canonical" {
		t.Fatalf("pulls = %#v", pulls)
	}
	if !containsString(pullTargets, repoDID) || !containsString(pullTargets, repoURI) {
		t.Fatalf("pull targets = %#v", pullTargets)
	}
}

func TestPullServiceResolvePullNumberFromAppView(t *testing.T) {
	const did = "did:plc:alice"
	records := map[string]fakeRecord{
		core.RepoPullNSID + "/p4": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/p4",
			CID:        "pull-cid",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"AppView pull","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"did:plc:repo","branch":"main"},"rounds":[]}`,
		},
	}
	pds := newFakePDSServer(t, records)
	appview := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/onev.cat/tang/pulls/4" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `<span data-aturi="at://did:plc:alice/sh.tangled.repo.pull/p4">pull</span>`)
	}))
	t.Cleanup(appview.Close)
	stubResolvers(t, did, "onev.cat", pds.URL)
	service := NewPullService(config.Defaults(), pds.Client())

	pull, err := service.ResolvePullNumber(context.Background(), appview.URL, "onev.cat", "tang", 4)
	if err != nil {
		t.Fatalf("ResolvePullNumber error = %v", err)
	}
	if pull.Number != 4 || pull.URI != "at://did:plc:alice/sh.tangled.repo.pull/p4" || pull.Title != "AppView pull" {
		t.Fatalf("pull = %#v", pull)
	}
}

func TestPDSClientSessionServiceAuthBlobAndDelete(t *testing.T) {
	const blobCID = "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"
	pds := newFakePDSServer(t, map[string]fakeRecord{}, withBlob(blobCID, []byte("blob-data")))
	client := NewPDSClient(&auth.Session{
		DID:        "did:plc:alice",
		Handle:     "onev.cat",
		PDS:        pds.URL,
		AccessJWT:  "access",
		RefreshJWT: "refresh",
	}, pds.Client())

	sessionOut, err := client.CreateSession(context.Background(), "onev.cat", "password")
	if err != nil {
		t.Fatalf("CreateSession error = %v", err)
	}
	if sessionOut.Did != "did:plc:alice" || sessionOut.Handle != "onev.cat" {
		t.Fatalf("session out = %#v", sessionOut)
	}
	refreshed, err := client.RefreshSession(context.Background(), &auth.Session{
		DID:        "did:plc:alice",
		Handle:     "onev.cat",
		PDS:        pds.URL,
		AccessJWT:  "old-access",
		RefreshJWT: "old-refresh",
	})
	if err != nil {
		t.Fatalf("RefreshSession error = %v", err)
	}
	if refreshed.AccessJWT != "new-access" || refreshed.RefreshJWT != "new-refresh" {
		t.Fatalf("refreshed = %#v", refreshed)
	}
	token, err := client.GetServiceAuth(context.Background(), "knot1.tangled.sh", core.RepoCreateNSID, 0)
	if err != nil {
		t.Fatalf("GetServiceAuth error = %v", err)
	}
	if token != "service-token" {
		t.Fatalf("service token = %q", token)
	}
	upload, err := client.UploadBlob(context.Background(), strings.NewReader("blob-data"))
	if err != nil {
		t.Fatalf("UploadBlob error = %v", err)
	}
	if upload.Blob == nil || upload.Blob.Ref.String() != blobCID {
		t.Fatalf("upload = %#v", upload)
	}
	gotBlob, err := client.GetBlob(context.Background(), "did:plc:alice", blobCID)
	if err != nil {
		t.Fatalf("GetBlob error = %v", err)
	}
	if string(gotBlob) != "blob-data" {
		t.Fatalf("blob = %q", gotBlob)
	}
	if err := client.DeleteRecord(context.Background(), "did:plc:alice", core.RepoNSID, "r1"); err != nil {
		t.Fatalf("DeleteRecord error = %v", err)
	}
}

func TestKnotClientMethods(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer service-token" {
			sawAuth = true
		}
		switch r.URL.Path {
		case "/xrpc/sh.tangled.repo.compare":
			_, _ = w.Write([]byte(`{"patch":"compare-patch"}`))
		case "/xrpc/sh.tangled.repo.diff":
			_, _ = w.Write([]byte("diff-patch"))
		case "/xrpc/sh.tangled.repo.create":
			writeJSON(t, w, map[string]string{"repoDid": "did:plc:repo"})
		case "/xrpc/sh.tangled.repo.merge":
			w.WriteHeader(http.StatusOK)
		case "/xrpc/sh.tangled.repo.mergeCheck":
			writeJSON(t, w, map[string]any{"is_conflicted": false})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client := NewKnotClient("ignored", WithKnotBaseURL(server.URL), WithKnotHTTPClient(server.Client()), WithServiceAuthToken("service-token"))
	compare, err := client.Compare(context.Background(), "did:plc:alice/tang", "main", "feature")
	if err != nil {
		t.Fatalf("Compare error = %v", err)
	}
	if string(compare) != `{"patch":"compare-patch"}` {
		t.Fatalf("compare = %s", compare)
	}
	diff, err := client.Diff(context.Background(), "did:plc:alice/tang", "main")
	if err != nil {
		t.Fatalf("Diff error = %v", err)
	}
	if string(diff) != "diff-patch" {
		t.Fatalf("diff = %s", diff)
	}
	created, err := client.CreateRepo(context.Background(), &core.RepoCreate_Input{Name: "tang", Rkey: "r1"})
	if err != nil {
		t.Fatalf("CreateRepo error = %v", err)
	}
	if created.RepoDid == nil || *created.RepoDid != "did:plc:repo" {
		t.Fatalf("created = %#v", created)
	}
	if err := client.Merge(context.Background(), &core.RepoMerge_Input{Did: "did:plc:alice", Name: "tang", Branch: "main", Patch: "patch"}); err != nil {
		t.Fatalf("Merge error = %v", err)
	}
	check, err := client.MergeCheck(context.Background(), &core.RepoMergeCheck_Input{Did: "did:plc:alice", Name: "tang", Branch: "main", Patch: "patch"})
	if err != nil {
		t.Fatalf("MergeCheck error = %v", err)
	}
	if check.Is_conflicted || !sawAuth {
		t.Fatalf("merge check = %#v sawAuth=%v", check, sawAuth)
	}
}

func TestBuildRepoATURIUsesResolvedOwnerRecords(t *testing.T) {
	const did = "did:plc:alice"
	pds := newFakePDSServer(t, map[string]fakeRecord{
		core.RepoNSID + "/r1": {
			URI:        "at://did:plc:alice/sh.tangled.repo/r1",
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"tang","knot":"knot1.tangled.sh","createdAt":"2026-05-02T00:00:00Z"}`,
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)

	uri, err := BuildRepoATURI(context.Background(), &localrepo.RepositoryContext{Owner: "onev.cat", Name: "tang"})
	if err != nil {
		t.Fatalf("BuildRepoATURI error = %v", err)
	}
	if uri != "at://did:plc:alice/sh.tangled.repo/r1" {
		t.Fatalf("uri = %q", uri)
	}
	if _, err := BuildRepoATURI(context.Background(), &localrepo.RepositoryContext{Owner: "onev.cat", Name: "missing"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("BuildRepoATURI missing error = %v", err)
	}
}

func TestBuildRepoATURIUsesRKeyWhenRecordNameIsEmpty(t *testing.T) {
	const did = "did:plc:alice"
	pds := newFakePDSServer(t, map[string]fakeRecord{
		core.RepoNSID + "/tang-manual-repo": {
			URI:        "at://did:plc:alice/sh.tangled.repo/tang-manual-repo",
			CID:        "repo-cid",
			Lexicon:    core.RepoNSID,
			RecordJSON: `{"$type":"sh.tangled.repo","name":"","knot":"knot1.tangled.sh","repoDid":"did:plc:repo","createdAt":"2026-05-02T00:00:00Z"}`,
		},
	})
	stubResolvers(t, did, "onev.cat", pds.URL)

	uri, err := BuildRepoATURI(context.Background(), &localrepo.RepositoryContext{Owner: "onev.cat", Name: "tang-manual-repo"})
	if err != nil {
		t.Fatalf("BuildRepoATURI error = %v", err)
	}
	if uri != "at://did:plc:alice/sh.tangled.repo/tang-manual-repo" {
		t.Fatalf("uri = %q", uri)
	}
}

func TestRepoServiceCreateRepo(t *testing.T) {
	server := newCombinedXRPCServer(t)
	host := strings.TrimPrefix(server.URL, "https://")
	session := &auth.Session{DID: "did:plc:alice", Handle: "onev.cat", PDS: server.URL, AccessJWT: "access", RefreshJWT: "refresh"}
	service := NewRepoService(&config.Config{Knot: config.KnotConfig{Hosts: []string{host}}}, server.Client())

	repo, err := service.CreateRepo(context.Background(), session, CreateRepoOptions{
		Name:          "tang",
		Description:   "CLI",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreateRepo error = %v", err)
	}
	if repo.Owner != "onev.cat" || repo.Name != "tang" || repo.Knot != host || repo.Description != "CLI" {
		t.Fatalf("repo = %#v", repo)
	}
	if repo.RepoDID != "did:plc:repo" || repo.CID != "put-cid" {
		t.Fatalf("repo identity = %#v", repo)
	}
}

func TestPullServiceCreatePullAndMergeCheck(t *testing.T) {
	var createdPull map[string]any
	server := newCombinedXRPCServer(t, withCombinedPutRecordHook(func(collection, _ string, record json.RawMessage) {
		if collection != core.RepoPullNSID {
			return
		}
		if err := json.Unmarshal(record, &createdPull); err != nil {
			t.Fatalf("Unmarshal created pull error = %v", err)
		}
	}))
	host := strings.TrimPrefix(server.URL, "https://")
	patch := "From abc\nSubject: Add tests\n\nPatch"
	blobCID := "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"
	pds := newFakePDSServer(t, map[string]fakeRecord{
		core.RepoPullNSID + "/p1": {
			URI:        "at://did:plc:alice/sh.tangled.repo.pull/p1",
			CID:        "pull-cid",
			Lexicon:    core.RepoPullNSID,
			RecordJSON: `{"$type":"sh.tangled.repo.pull","title":"Pull","createdAt":"2026-05-02T00:00:00Z","target":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"main"},"source":{"repo":"at://did:plc:alice/sh.tangled.repo/r1","branch":"feature"},"rounds":[{"createdAt":"2026-05-02T00:00:00Z","patchBlob":{"$type":"blob","ref":{"$link":"` + blobCID + `"},"mimeType":"application/gzip","size":32}}]}`,
		},
	}, withBlob(blobCID, gzipBytes(t, patch)))
	stubResolvers(t, "did:plc:alice", "onev.cat", pds.URL)

	session := &auth.Session{DID: "did:plc:alice", Handle: "onev.cat", PDS: server.URL, AccessJWT: "access", RefreshJWT: "refresh"}
	service := NewPullService(&config.Config{Constellation: config.ConstellationConfig{URL: "http://127.0.0.1"}}, server.Client())
	pull, err := service.CreatePull(context.Background(), session, PullCreateOptions{
		Repo:       Repo{Owner: "onev.cat", Name: "tang", Knot: host, RepoDID: "did:plc:repo"},
		RepoURI:    "at://did:plc:alice/sh.tangled.repo/r1",
		BaseBranch: "main",
		HeadBranch: "feature",
		Fill:       true,
	})
	if err != nil {
		t.Fatalf("CreatePull error = %v", err)
	}
	if pull.Title != "Add tests" || pull.Branch != "feature" || pull.Target != "main" {
		t.Fatalf("pull = %#v", pull)
	}
	target, ok := createdPull["target"].(map[string]any)
	if !ok || target["repo"] != "did:plc:repo" {
		t.Fatalf("created pull target = %#v", createdPull)
	}
	source, ok := createdPull["source"].(map[string]any)
	if !ok || source["repo"] != "did:plc:repo" {
		t.Fatalf("created pull source = %#v", createdPull)
	}

	mergeService := NewPullService(&config.Config{Constellation: config.ConstellationConfig{URL: "http://127.0.0.1"}}, server.Client())
	got, err := mergeService.MergeCheck(context.Background(), Repo{Name: "tang", Knot: host}, "did:plc:alice", Pull{URI: "at://did:plc:alice/sh.tangled.repo.pull/p1", Target: "main"})
	if err != nil {
		t.Fatalf("MergeCheck error = %v", err)
	}
	if got != "clean" {
		t.Fatalf("merge check = %q", got)
	}
}

type combinedXRPCOption func(*combinedXRPCState)

type combinedXRPCState struct {
	putRecordHook func(collection, rkey string, record json.RawMessage)
}

func withCombinedPutRecordHook(hook func(collection, rkey string, record json.RawMessage)) combinedXRPCOption {
	return func(s *combinedXRPCState) {
		s.putRecordHook = hook
	}
}

func newCombinedXRPCServer(t *testing.T, opts ...combinedXRPCOption) *httptest.Server {
	t.Helper()
	state := &combinedXRPCState{}
	for _, opt := range opts {
		opt(state)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.getServiceAuth":
			writeJSON(t, w, map[string]string{"token": "service-token"})
		case "/xrpc/com.atproto.repo.uploadBlob":
			_, _ = io.ReadAll(r.Body)
			writeJSON(t, w, map[string]any{
				"blob": map[string]any{
					"$type":    "blob",
					"ref":      map[string]string{"$link": "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"},
					"mimeType": "application/gzip",
					"size":     32,
				},
			})
		case "/xrpc/com.atproto.repo.putRecord":
			var input struct {
				Collection string          `json:"collection"`
				RKey       string          `json:"rkey"`
				Record     json.RawMessage `json:"record"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if state.putRecordHook != nil {
				state.putRecordHook(input.Collection, input.RKey, input.Record)
			}
			writeJSON(t, w, map[string]string{
				"uri": "at://did:plc:alice/" + input.Collection + "/" + input.RKey,
				"cid": "put-cid",
			})
		case "/xrpc/sh.tangled.repo.create":
			writeJSON(t, w, map[string]string{"repoDid": "did:plc:repo"})
		case "/xrpc/sh.tangled.repo.compare":
			_, _ = w.Write([]byte(`{"patch":"From abc\nSubject: Add tests\n\nPatch"}`))
		case "/xrpc/sh.tangled.repo.mergeCheck":
			writeJSON(t, w, map[string]any{"is_conflicted": false})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

type fakeRecord struct {
	URI        string
	CID        string
	Lexicon    string
	RecordJSON string
}

type fakeLink struct {
	DID        string
	Collection string
	RKey       string
	Target     string
}

type fakePDSOption func(*fakePDSState)

type fakePDSState struct {
	records          map[string]fakeRecord
	blobs            map[string][]byte
	createRecordHook func(collection string, record json.RawMessage)
	putRecordHook    func(collection, rkey string, record json.RawMessage)
}

func withBlob(cid string, data []byte) fakePDSOption {
	return func(s *fakePDSState) {
		s.blobs[cid] = data
	}
}

func withCreateRecordHook(hook func(collection string, record json.RawMessage)) fakePDSOption {
	return func(s *fakePDSState) {
		s.createRecordHook = hook
	}
}

func newFakePDSServer(t *testing.T, records map[string]fakeRecord, opts ...fakePDSOption) *httptest.Server {
	t.Helper()
	state := &fakePDSState{records: records, blobs: map[string][]byte{}}
	for _, opt := range opts {
		opt(state)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.server.createSession":
			writeJSON(t, w, map[string]string{
				"did":        "did:plc:alice",
				"handle":     "onev.cat",
				"accessJwt":  "access",
				"refreshJwt": "refresh",
			})
		case "/xrpc/com.atproto.server.refreshSession":
			writeJSON(t, w, map[string]string{
				"did":        "did:plc:alice",
				"handle":     "onev.cat",
				"accessJwt":  "new-access",
				"refreshJwt": "new-refresh",
			})
		case "/xrpc/com.atproto.server.getServiceAuth":
			if r.URL.Query().Get("aud") == "" || r.URL.Query().Get("lxm") == "" {
				http.Error(w, "missing query", http.StatusBadRequest)
				return
			}
			writeJSON(t, w, map[string]string{"token": "service-token"})
		case "/xrpc/com.atproto.repo.listRecords":
			collection := r.URL.Query().Get("collection")
			var out struct {
				Records []json.RawMessage `json:"records"`
			}
			for key, record := range state.records {
				if !strings.HasPrefix(key, collection+"/") {
					continue
				}
				out.Records = append(out.Records, recordJSON(t, record))
			}
			writeJSON(t, w, out)
		case "/xrpc/com.atproto.repo.getRecord":
			key := r.URL.Query().Get("collection") + "/" + r.URL.Query().Get("rkey")
			record, ok := state.records[key]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			if _, err := w.Write(recordJSON(t, record)); err != nil {
				t.Fatalf("Write response error = %v", err)
			}
		case "/xrpc/com.atproto.repo.createRecord":
			var input struct {
				Collection string          `json:"collection"`
				Record     json.RawMessage `json:"record"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if state.createRecordHook != nil {
				state.createRecordHook(input.Collection, input.Record)
			}
			rkey := "created"
			writeJSON(t, w, map[string]string{
				"uri": "at://did:plc:alice/" + input.Collection + "/" + rkey,
				"cid": "created-cid",
			})
		case "/xrpc/com.atproto.repo.putRecord":
			var input struct {
				Collection string          `json:"collection"`
				RKey       string          `json:"rkey"`
				Record     json.RawMessage `json:"record"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if state.putRecordHook != nil {
				state.putRecordHook(input.Collection, input.RKey, input.Record)
			}
			writeJSON(t, w, map[string]string{
				"uri": "at://did:plc:alice/" + input.Collection + "/" + input.RKey,
				"cid": "put-cid",
			})
		case "/xrpc/com.atproto.repo.deleteRecord":
			writeJSON(t, w, map[string]any{})
		case "/xrpc/com.atproto.repo.uploadBlob":
			_, _ = io.ReadAll(r.Body)
			writeJSON(t, w, map[string]any{
				"blob": map[string]any{
					"$type":    "blob",
					"ref":      map[string]string{"$link": "bafkreieqq463374bbcbeq7gpmet5rvrpeqow6t4rtjzrkhnlumdylagaqa"},
					"mimeType": "application/octet-stream",
					"size":     9,
				},
			})
		case "/xrpc/com.atproto.sync.getBlob":
			data, ok := state.blobs[r.URL.Query().Get("cid")]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			_, _ = w.Write(data)
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func recordJSON(t *testing.T, record fakeRecord) []byte {
	t.Helper()
	raw := json.RawMessage(record.RecordJSON)
	out := struct {
		URI   string          `json:"uri"`
		CID   string          `json:"cid"`
		Value json.RawMessage `json:"value"`
	}{URI: record.URI, CID: record.CID, Value: raw}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Marshal record error = %v", err)
	}
	return data
}

type fakeConstellationOption func(*fakeConstellationState)

type fakeConstellationState struct {
	queryHook func(collection, target, path string)
}

func withConstellationQueryHook(hook func(collection, target, path string)) fakeConstellationOption {
	return func(s *fakeConstellationState) {
		s.queryHook = hook
	}
}

func newFakeConstellationServer(t *testing.T, links map[string][]fakeLink, opts ...fakeConstellationOption) *httptest.Server {
	t.Helper()
	state := &fakeConstellationState{}
	for _, opt := range opts {
		opt(state)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/links" {
			http.NotFound(w, r)
			return
		}
		collection := r.URL.Query().Get("collection")
		target := r.URL.Query().Get("target")
		if state.queryHook != nil {
			state.queryHook(collection, target, r.URL.Query().Get("path"))
		}
		sourceLinks := links[collection]
		filterByTarget := false
		for _, link := range sourceLinks {
			if link.Target != "" {
				filterByTarget = true
				break
			}
		}
		wire := struct {
			Total          int `json:"total"`
			LinkingRecords []struct {
				DID        string `json:"did"`
				Collection string `json:"collection"`
				RKey       string `json:"rkey"`
			} `json:"linking_records"`
		}{}
		for _, link := range sourceLinks {
			if filterByTarget && link.Target != target {
				continue
			}
			wire.LinkingRecords = append(wire.LinkingRecords, struct {
				DID        string `json:"did"`
				Collection string `json:"collection"`
				RKey       string `json:"rkey"`
			}{DID: link.DID, Collection: link.Collection, RKey: link.RKey})
		}
		wire.Total = len(wire.LinkingRecords)
		writeJSON(t, w, wire)
	}))
	t.Cleanup(server.Close)
	return server
}

func stubResolvers(t *testing.T, did, handle, pds string) {
	t.Helper()
	oldDID := resolveDIDFunc
	oldHandle := resolveHandleFunc
	resolveDIDFunc = func(context.Context, string) (*atproto.Identity, error) {
		return &atproto.Identity{DID: did, Handle: handle, PDS: pds}, nil
	}
	resolveHandleFunc = func(context.Context, string) (*atproto.Identity, error) {
		return &atproto.Identity{DID: did, Handle: handle, PDS: pds}, nil
	}
	t.Cleanup(func() {
		resolveDIDFunc = oldDID
		resolveHandleFunc = oldHandle
	})
}

func gzipBytes(t *testing.T, input string) []byte {
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("Encode response error = %v", err)
	}
}
