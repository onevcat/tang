package tangled

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/git"
)

type Repo struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Knot        string `json:"knot"`
	RepoDID     string `json:"repoDid,omitempty"`
	CreatedAt   string `json:"createdAt"`
	URI         string `json:"uri"`
	CID         string `json:"cid,omitempty"`
	CloneSSH    string `json:"cloneSsh"`
	CloneHTTPS  string `json:"cloneHttps"`
}

type CreateRepoOptions struct {
	Name          string
	Description   string
	Knot          string
	DefaultBranch string
}

type RepoService struct {
	Config     *config.Config
	HTTPClient *http.Client
}

func NewRepoService(cfg *config.Config, httpClient *http.Client) *RepoService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &RepoService{Config: cfg, HTTPClient: httpClient}
}

func (s *RepoService) ListRepos(ctx context.Context, owner string) ([]Repo, error) {
	ownerDID, pds, err := resolveOwner(ctx, owner)
	if err != nil {
		return nil, err
	}
	client := NewAnonymousPDSClient(pds, s.HTTPClient)
	out, err := client.ListRecords(ctx, ownerDID, core.RepoNSID, 100, "")
	if err != nil {
		return nil, err
	}
	repos := make([]Repo, 0, len(out.Records))
	for _, rec := range out.Records {
		record, ok := rec.Value.Val.(*core.Repo)
		if !ok {
			continue
		}
		cid := ""
		if rec.Cid != "" {
			cid = rec.Cid
		}
		repos = append(repos, repoFromRecord(owner, rec.Uri, cid, record))
	}
	return repos, nil
}

func (s *RepoService) GetRepo(ctx context.Context, owner, name string) (*Repo, error) {
	repos, err := s.ListRepos(ctx, owner)
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		if repo.Name == name {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("repository %s/%s not found", owner, name)
}

func (s *RepoService) CreateRepo(ctx context.Context, session *auth.Session, opts CreateRepoOptions) (*Repo, error) {
	knot := opts.Knot
	if knot == "" {
		knot = config.DefaultKnotHost
		if len(s.Config.Knot.Hosts) > 0 {
			knot = s.Config.Knot.Hosts[0]
		}
	}
	rkey := syntax.NewTIDNow(0).String()
	pdsClient := NewPDSClient(session, s.HTTPClient)
	token, err := pdsClient.GetServiceAuth(ctx, knot, core.RepoCreateNSID, 60*time.Second)
	if err != nil {
		return nil, err
	}
	knotClient := NewKnotClient(knot, WithKnotHTTPClient(s.HTTPClient), WithServiceAuthToken(token))
	createOut, err := knotClient.CreateRepo(ctx, &core.RepoCreate_Input{
		Name:          opts.Name,
		Rkey:          rkey,
		DefaultBranch: optionalString(opts.DefaultBranch),
	})
	if err != nil {
		return nil, err
	}
	record := &core.Repo{
		LexiconTypeID: core.RepoNSID,
		Name:          opts.Name,
		Knot:          knot,
		Description:   optionalString(opts.Description),
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	if createOut.RepoDid != nil {
		record.RepoDid = createOut.RepoDid
	}
	out, err := pdsClient.PutRecord(ctx, session.DID, core.RepoNSID, rkey, record, nil)
	if err != nil {
		return nil, err
	}
	return ptr(repoFromRecord(session.Handle, out.Uri, out.Cid, record)), nil
}

func (s *RepoService) Clone(ctx context.Context, owner, name, dir string) error {
	repo, err := s.GetRepo(ctx, owner, name)
	if err != nil {
		return err
	}
	return git.Clone(ctx, repo.CloneSSH, dir)
}

func repoFromRecord(owner, uri, cid string, record *core.Repo) Repo {
	description := ""
	if record.Description != nil {
		description = *record.Description
	}
	repoDID := ""
	if record.RepoDid != nil {
		repoDID = *record.RepoDid
	}
	cloneHost := cloneHostForKnot(record.Knot)
	return Repo{
		Owner:       owner,
		Name:        record.Name,
		Description: description,
		Knot:        record.Knot,
		RepoDID:     repoDID,
		CreatedAt:   record.CreatedAt,
		URI:         uri,
		CID:         cid,
		CloneSSH:    fmt.Sprintf("git@%s:%s/%s", cloneHost, owner, record.Name),
		CloneHTTPS:  fmt.Sprintf("https://%s/%s/%s", cloneHost, owner, record.Name),
	}
}

func cloneHostForKnot(knot string) string {
	if knot == "knot1.tangled.sh" {
		return "tangled.org"
	}
	return knot
}

func resolveOwner(ctx context.Context, owner string) (did, pds string, err error) {
	if strings.HasPrefix(owner, "did:") {
		ident, err := atproto.ResolveDID(ctx, owner)
		if err != nil {
			return "", "", err
		}
		return owner, ident.PDS, nil
	}
	ident, err := atproto.ResolveHandle(ctx, owner)
	if err != nil {
		return "", "", err
	}
	return ident.DID, ident.PDS, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func ptr[T any](value T) *T {
	return &value
}
