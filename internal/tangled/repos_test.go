package tangled

import (
	"testing"

	core "tangled.org/core/api/tangled"
)

func TestRepoFromRecordBuildsCloneURLs(t *testing.T) {
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:      "tang",
		Knot:      "knot.example.com",
		CreatedAt: "2026-05-02T00:00:00Z",
	})
	if repo.CloneSSH != "git@knot.example.com:onev.cat/tang.git" {
		t.Fatalf("CloneSSH = %q", repo.CloneSSH)
	}
	if repo.CloneHTTPS != "https://knot.example.com/onev.cat/tang" {
		t.Fatalf("CloneHTTPS = %q", repo.CloneHTTPS)
	}
}
