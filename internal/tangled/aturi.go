package tangled

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/repo"
)

var ErrInvalidATURI = errors.New("invalid AT-URI")

type ATURI struct {
	DID        string
	Collection string
	RKey       string
}

func ParseATURI(uri string) (ATURI, error) {
	match := aturiPattern.FindStringSubmatch(uri)
	if match == nil {
		return ATURI{}, fmt.Errorf("%w: %s", ErrInvalidATURI, uri)
	}
	return ATURI{DID: match[1], Collection: match[2], RKey: match[3]}, nil
}

func BuildRepoATURI(ctx context.Context, context *repo.RepositoryContext) (string, error) {
	ownerDID := context.Owner
	pds := ""
	if !strings.HasPrefix(ownerDID, "did:") {
		ident, err := atproto.ResolveHandle(ctx, ownerDID)
		if err != nil {
			return "", err
		}
		ownerDID = ident.DID
		pds = ident.PDS
	} else {
		ident, err := atproto.ResolveDID(ctx, ownerDID)
		if err != nil {
			return "", err
		}
		pds = ident.PDS
	}
	client := NewAnonymousPDSClient(pds, http.DefaultClient)
	records, err := client.ListRecords(ctx, ownerDID, core.RepoNSID, 100, "")
	if err != nil {
		return "", err
	}
	for _, record := range records.Records {
		value, ok := record.Value.Val.(*core.Repo)
		if ok && value.Name == context.Name {
			return record.Uri, nil
		}
	}
	return "", fmt.Errorf("repository %s not found for %s", context.Name, context.Owner)
}

func RKeyFromURI(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) == 0 {
		return uri
	}
	return parts[len(parts)-1]
}

var aturiPattern = regexp.MustCompile(`^at://(did:[a-z]+:[a-zA-Z0-9._:%-]+)/([a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9._-]+)*)(?:/([a-zA-Z0-9._-]+))?$`)
