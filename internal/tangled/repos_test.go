package tangled

import (
	"context"
	"errors"
	"testing"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/config"
)

func TestRepoFromRecordBuildsCloneURLs(t *testing.T) {
	name := "tang"
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:      &name,
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
	name := "tang"
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:      &name,
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

func TestCloneHostForKnotLeavesCustomHostUntouched(t *testing.T) {
	if got := cloneHostForKnot("knot.example.com"); got != "knot.example.com" {
		t.Fatalf("clone host = %q", got)
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

func TestRepoFromRecordCopiesOptionalFields(t *testing.T) {
	name := "tang"
	description := "Command-line client"
	repoDID := "did:plc:repo"
	repo := repoFromRecord("onev.cat", "at://did/sh.tangled.repo/r", "cid", &core.Repo{
		Name:        &name,
		Knot:        "knot.example.com",
		Description: &description,
		RepoDid:     &repoDID,
		CreatedAt:   "2026-05-02T00:00:00Z",
	})
	if repo.Description != description || repo.RepoDID != repoDID || repo.CID != "cid" || repo.URI == "" {
		t.Fatalf("repo = %#v", repo)
	}
}

func TestRepoCloneURLDefaultsToHTTPS(t *testing.T) {
	repo := Repo{CloneSSH: "ssh", CloneHTTPS: "https"}
	got, err := NewRepoService(nil, nil).CloneURL(repo)
	if err != nil {
		t.Fatalf("CloneURL error = %v", err)
	}
	if got != "https" {
		t.Fatalf("CloneURL = %q", got)
	}
}

func TestOptionalStringAndPtr(t *testing.T) {
	if optionalString("") != nil {
		t.Fatal("optionalString empty should be nil")
	}
	if got := optionalString("value"); got == nil || *got != "value" {
		t.Fatalf("optionalString value = %#v", got)
	}
	if got := ptr(42); got == nil || *got != 42 {
		t.Fatalf("ptr value = %#v", got)
	}
}

func TestResolverTestHookAndResolveOwnerDID(t *testing.T) {
	restore := SetResolversForTesting(
		func(context.Context, string) (*atproto.Identity, error) {
			return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: "https://pds.example.com"}, nil
		},
		func(context.Context, string) (*atproto.Identity, error) {
			return &atproto.Identity{DID: "did:plc:alice", Handle: "onev.cat", PDS: "https://pds.example.com"}, nil
		},
	)
	defer restore()

	did, pds, err := resolveOwner(context.Background(), "did:plc:alice")
	if err != nil {
		t.Fatalf("resolveOwner DID error = %v", err)
	}
	if did != "did:plc:alice" || pds != "https://pds.example.com" {
		t.Fatalf("resolved owner = %q %q", did, pds)
	}
	did, pds, err = resolveOwner(context.Background(), "onev.cat")
	if err != nil {
		t.Fatalf("resolveOwner handle error = %v", err)
	}
	if did != "did:plc:alice" || pds != "https://pds.example.com" {
		t.Fatalf("resolved handle owner = %q %q", did, pds)
	}
}
