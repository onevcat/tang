package repo

import (
	"errors"
	"testing"

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
