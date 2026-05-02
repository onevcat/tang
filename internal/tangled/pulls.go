package tangled

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/atproto"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/constellation"
)

var ErrPatchOnlyCheckout = errors.New("pull request has no source branch; checkout only works for branch-based pull requests")

type Pull struct {
	Number    int    `json:"number,omitempty"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Status    string `json:"status"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
	CID       string `json:"cid,omitempty"`
	Target    string `json:"target"`
	Source    string `json:"source,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Mergeable string `json:"mergeable,omitempty"`
}

type PullCreateOptions struct {
	Repo       Repo
	RepoURI    string
	BaseBranch string
	HeadBranch string
	Title      string
	Body       string
	Fill       bool
}

type PullService struct {
	Config        *config.Config
	Constellation *constellation.Client
	HTTPClient    *http.Client
}

func NewPullService(cfg *config.Config, httpClient *http.Client) *PullService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &PullService{
		Config:        cfg,
		Constellation: constellation.NewClient(cfg.Constellation.URL, httpClient),
		HTTPClient:    httpClient,
	}
}

func (s *PullService) ListPulls(ctx context.Context, repoURI string, status string, limit int) ([]Pull, error) {
	if limit <= 0 {
		limit = 50
	}
	backlinks, err := s.Constellation.GetBacklinks(ctx, repoURI, core.RepoPullNSID, ".target.repo", limit, "")
	if err != nil {
		return nil, err
	}
	pulls := make([]Pull, 0, len(backlinks.Records))
	for _, link := range backlinks.Records {
		pull, err := s.getPullByParts(ctx, link.DID, link.Collection, link.RKey)
		if err != nil {
			continue
		}
		if st, err := s.GetPullStatus(ctx, pull.URI); err == nil {
			pull.Status = st
		}
		pulls = append(pulls, *pull)
	}
	assignPullNumbers(pulls)
	if status != "" && status != "all" {
		filtered := pulls[:0]
		for _, pull := range pulls {
			if pull.Status == status {
				filtered = append(filtered, pull)
			}
		}
		pulls = filtered
	}
	return pulls, nil
}

func (s *PullService) CreatePull(ctx context.Context, session *auth.Session, opts PullCreateOptions) (*Pull, error) {
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}
	if opts.HeadBranch == "" {
		return nil, fmt.Errorf("head branch is required")
	}
	repoIdentifier := opts.Repo.RepoDID
	if repoIdentifier == "" {
		ownerDID, _, err := resolveOwner(ctx, opts.Repo.Owner)
		if err != nil {
			return nil, err
		}
		repoIdentifier = ownerDID + "/" + opts.Repo.Name
	}
	compare, err := NewKnotClient(opts.Repo.Knot, WithKnotHTTPClient(s.HTTPClient)).Compare(ctx, repoIdentifier, opts.BaseBranch, opts.HeadBranch)
	if err != nil {
		return nil, err
	}
	var comparison struct {
		Patch            string `json:"patch"`
		CombinedPatchRaw string `json:"combined_patch_raw"`
	}
	if err := json.Unmarshal(compare, &comparison); err != nil {
		return nil, err
	}
	patch := comparison.Patch
	if patch == "" {
		patch = comparison.CombinedPatchRaw
	}
	if patch == "" {
		return nil, fmt.Errorf("compare returned no patch")
	}
	title := opts.Title
	body := opts.Body
	if opts.Fill && title == "" {
		title, body = fillTitleBodyFromPatch(patch, body)
	}
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	if _, err := zw.Write([]byte(patch)); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	pds := NewPDSClient(session, s.HTTPClient)
	blob, err := pds.UploadBlob(ctx, &gz)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	record := &core.RepoPull{
		LexiconTypeID: core.RepoPullNSID,
		Title:         title,
		Body:          optionalString(body),
		CreatedAt:     now,
		Target: &core.RepoPull_Target{
			Repo:    &opts.RepoURI,
			RepoDid: optionalString(opts.Repo.RepoDID),
			Branch:  opts.BaseBranch,
		},
		Source: &core.RepoPull_Source{
			Repo:   &opts.RepoURI,
			Branch: opts.HeadBranch,
		},
		Rounds: []*core.RepoPull_Round{{
			CreatedAt: now,
			PatchBlob: blob.Blob,
		}},
	}
	rkey := syntax.NewTIDNow(0).String()
	out, err := pds.PutRecord(ctx, session.DID, core.RepoPullNSID, rkey, record, nil)
	if err != nil {
		return nil, err
	}
	return &Pull{Title: title, Body: body, Status: "open", Author: session.DID, CreatedAt: now, URI: out.Uri, CID: out.Cid, Target: opts.BaseBranch, Source: opts.RepoURI, Branch: opts.HeadBranch}, nil
}

func (s *PullService) GetPull(ctx context.Context, pullURI string) (*Pull, error) {
	parsed, err := ParseATURI(pullURI)
	if err != nil {
		return nil, err
	}
	return s.getPullByParts(ctx, parsed.DID, parsed.Collection, parsed.RKey)
}

func (s *PullService) SetPullStatus(ctx context.Context, session *auth.Session, pullURI, status string) error {
	statusValue := core.RepoPullStatusOpen
	switch status {
	case "closed":
		statusValue = core.RepoPullStatusClosed
	case "merged":
		statusValue = core.RepoPullStatusMerged
	}
	record := &core.RepoPullStatus{LexiconTypeID: core.RepoPullStatusNSID, Pull: pullURI, Status: statusValue}
	_, err := NewPDSClient(session, s.HTTPClient).CreateRecord(ctx, session.DID, core.RepoPullStatusNSID, record, nil)
	return err
}

func (s *PullService) GetPullStatus(ctx context.Context, pullURI string) (string, error) {
	backlinks, err := s.Constellation.GetBacklinks(ctx, pullURI, core.RepoPullStatusNSID, ".pull", 100, "")
	if err != nil || len(backlinks.Records) == 0 {
		return "open", err
	}
	sort.Slice(backlinks.Records, func(i, j int) bool { return backlinks.Records[i].RKey < backlinks.Records[j].RKey })
	latest := backlinks.Records[len(backlinks.Records)-1]
	ident, err := atproto.ResolveDID(ctx, latest.DID)
	if err != nil {
		return "open", err
	}
	out, err := NewAnonymousPDSClient(ident.PDS, s.HTTPClient).GetRecord(ctx, latest.DID, latest.Collection, latest.RKey)
	if err != nil {
		return "open", err
	}
	record, ok := out.Value.Val.(*core.RepoPullStatus)
	if !ok {
		return "open", nil
	}
	switch record.Status {
	case core.RepoPullStatusClosed:
		return "closed", nil
	case core.RepoPullStatusMerged:
		return "merged", nil
	default:
		return "open", nil
	}
}

func (s *PullService) AddPullComment(ctx context.Context, session *auth.Session, pullURI, body string) (*Comment, error) {
	record := &core.RepoPullComment{LexiconTypeID: core.RepoPullCommentNSID, Pull: pullURI, Body: body, CreatedAt: time.Now().UTC().Format(time.RFC3339)}
	out, err := NewPDSClient(session, s.HTTPClient).CreateRecord(ctx, session.DID, core.RepoPullCommentNSID, record, nil)
	if err != nil {
		return nil, err
	}
	return &Comment{Author: session.DID, Body: body, CreatedAt: record.CreatedAt, URI: out.Uri, CID: out.Cid}, nil
}

func (s *PullService) FetchPullPatch(ctx context.Context, pullURI string) (string, error) {
	pull, err := s.rawPull(ctx, pullURI)
	if err != nil {
		return "", err
	}
	if len(pull.record.Rounds) == 0 {
		return "", fmt.Errorf("pull has no rounds")
	}
	latest := pull.record.Rounds[len(pull.record.Rounds)-1]
	ident, err := atproto.ResolveDID(ctx, pull.author)
	if err != nil {
		return "", err
	}
	data, err := NewAnonymousPDSClient(ident.PDS, s.HTTPClient).GetBlob(ctx, pull.author, latest.PatchBlob.Ref.String())
	if err != nil {
		return "", err
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer func() { _ = zr.Close() }()
	out, err := io.ReadAll(zr)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *PullService) MergeCheck(ctx context.Context, repo Repo, ownerDID string, pull Pull) (string, error) {
	patch, err := s.FetchPullPatch(ctx, pull.URI)
	if err != nil {
		return "", err
	}
	out, err := NewKnotClient(repo.Knot, WithKnotHTTPClient(s.HTTPClient)).MergeCheck(ctx, &core.RepoMergeCheck_Input{
		Did:    ownerDID,
		Name:   repo.Name,
		Branch: pull.Target,
		Patch:  patch,
	})
	if err != nil {
		return "", err
	}
	if out.Is_conflicted {
		return "conflicted", nil
	}
	if out.Error != nil && *out.Error != "" {
		return "error: " + *out.Error, nil
	}
	return "clean", nil
}

func (s *PullService) getPullByParts(ctx context.Context, did, collection, rkey string) (*Pull, error) {
	raw, err := s.getRawPullByParts(ctx, did, collection, rkey)
	if err != nil {
		return nil, err
	}
	return pullFromRecord(raw.author, raw.uri, raw.cid, raw.record), nil
}

type rawPull struct {
	author string
	uri    string
	cid    string
	record *core.RepoPull
}

func (s *PullService) rawPull(ctx context.Context, uri string) (*rawPull, error) {
	parsed, err := ParseATURI(uri)
	if err != nil {
		return nil, err
	}
	return s.getRawPullByParts(ctx, parsed.DID, parsed.Collection, parsed.RKey)
}

func (s *PullService) getRawPullByParts(ctx context.Context, did, collection, rkey string) (*rawPull, error) {
	ident, err := atproto.ResolveDID(ctx, did)
	if err != nil {
		return nil, err
	}
	out, err := NewAnonymousPDSClient(ident.PDS, s.HTTPClient).GetRecord(ctx, did, collection, rkey)
	if err != nil {
		return nil, err
	}
	record, ok := out.Value.Val.(*core.RepoPull)
	if !ok {
		return nil, fmt.Errorf("record is not a pull: %s", out.Uri)
	}
	cid := ""
	if out.Cid != nil {
		cid = *out.Cid
	}
	return &rawPull{author: did, uri: out.Uri, cid: cid, record: record}, nil
}

func pullFromRecord(author, uri, cid string, record *core.RepoPull) *Pull {
	body := ""
	if record.Body != nil {
		body = *record.Body
	}
	pull := &Pull{Title: record.Title, Body: body, Status: "open", Author: author, CreatedAt: record.CreatedAt, URI: uri, CID: cid}
	if record.Target != nil {
		pull.Target = record.Target.Branch
	}
	if record.Source != nil {
		pull.Branch = record.Source.Branch
		if record.Source.Repo != nil {
			pull.Source = *record.Source.Repo
		}
	}
	return pull
}

func ResolvePullIdentifier(input string, pulls []Pull) (Pull, error) {
	normalized := strings.TrimPrefix(input, "#")
	if n, err := strconv.Atoi(normalized); err == nil {
		assignPullNumbers(pulls)
		for _, pull := range pulls {
			if pull.Number == n {
				return pull, nil
			}
		}
		return Pull{}, fmt.Errorf("pull #%d not found", n)
	}
	for _, pull := range pulls {
		if RKeyFromURI(pull.URI) == normalized || pull.URI == input {
			return pull, nil
		}
	}
	return Pull{}, fmt.Errorf("pull %q not found", input)
}

func assignPullNumbers(pulls []Pull) {
	sort.Slice(pulls, func(i, j int) bool { return pulls[i].CreatedAt < pulls[j].CreatedAt })
	for i := range pulls {
		pulls[i].Number = i + 1
	}
}

func fillTitleBodyFromPatch(patch, existingBody string) (string, string) {
	title := ""
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "Subject: ") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "Subject: "))
			break
		}
	}
	if title == "" {
		title = "Pull request"
	}
	return title, existingBody
}
