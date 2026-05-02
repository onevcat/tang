package git

import "testing"

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
