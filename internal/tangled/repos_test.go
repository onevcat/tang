package tangled

import (
	"errors"
	"testing"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/config"
)

func TestRepoFromRecordBuildsCloneURLs(t *testing.T) {
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:      "tang",
		Knot:      "knot.example.com",
		CreatedAt: "2026-05-02T00:00:00Z",
	})
	if repo.CloneSSH != "git@knot.example.com:onev.cat/tang" {
		t.Fatalf("CloneSSH = %q", repo.CloneSSH)
	}
	if repo.CloneHTTPS != "https://knot.example.com/onev.cat/tang" {
		t.Fatalf("CloneHTTPS = %q", repo.CloneHTTPS)
	}
}

func TestRepoFromRecordUsesHostedCloneHostForDefaultHostedKnot(t *testing.T) {
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:      "tang",
		Knot:      "knot1.tangled.sh",
		CreatedAt: "2026-05-02T00:00:00Z",
	})
	if repo.CloneSSH != "git@tangled.org:onev.cat/tang" {
		t.Fatalf("CloneSSH = %q", repo.CloneSSH)
	}
	if repo.CloneHTTPS != "https://tangled.org/onev.cat/tang" {
		t.Fatalf("CloneHTTPS = %q", repo.CloneHTTPS)
	}
}

func TestRepoCloneURLUsesConfiguredProtocol(t *testing.T) {
	repo := Repo{
		CloneSSH:   "git@tangled.org:onev.cat/tang",
		CloneHTTPS: "https://tangled.org/onev.cat/tang",
	}
	service := NewRepoService(&config.Config{Clone: config.CloneConfig{Protocol: "https"}}, nil)
	got, err := service.CloneURL(repo)
	if err != nil {
		t.Fatalf("CloneURL https error = %v", err)
	}
	if got != repo.CloneHTTPS {
		t.Fatalf("CloneURL https = %q", got)
	}
	service.Config.Clone.Protocol = "ssh"
	got, err = service.CloneURL(repo)
	if err != nil {
		t.Fatalf("CloneURL ssh error = %v", err)
	}
	if got != repo.CloneSSH {
		t.Fatalf("CloneURL ssh = %q", got)
	}
}

func TestRepoCloneURLRejectsUnsupportedProtocol(t *testing.T) {
	service := NewRepoService(&config.Config{Clone: config.CloneConfig{Protocol: "git"}}, nil)
	if _, err := service.CloneURL(Repo{}); !errors.Is(err, config.ErrUnsupportedValue) {
		t.Fatalf("CloneURL invalid protocol error = %v", err)
	}
}
