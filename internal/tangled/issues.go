package tangled

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	core "tangled.org/core/api/tangled"
	"tangled.org/onev.cat/tang/internal/auth"
	"tangled.org/onev.cat/tang/internal/config"
	"tangled.org/onev.cat/tang/internal/constellation"
)

type Issue struct {
	Number    int    `json:"number,omitempty"`
	Title     string `json:"title"`
	Body      string `json:"body,omitempty"`
	Repo      string `json:"repo,omitempty"`
	State     string `json:"state"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
	CID       string `json:"cid"`
}

type Comment struct {
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	URI       string `json:"uri"`
	CID       string `json:"cid"`
}

type IssueListOptions struct {
	State  string
	Limit  int
	Cursor string
}

type IssueService struct {
	Constellation *constellation.Client
	HTTPClient    *http.Client
}

func NewIssueService(cfg *config.Config, httpClient *http.Client) *IssueService {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &IssueService{
		Constellation: constellation.NewClient(cfg.Constellation.URL, httpClient),
		HTTPClient:    httpClient,
	}
}

func (s *IssueService) ListIssues(ctx context.Context, repoURI string, opts IssueListOptions) ([]Issue, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	links, err := repoBacklinks(ctx, s.Constellation, s.HTTPClient, repoURI, core.RepoIssueNSID, ".repo", limit, opts.Cursor)
	if err != nil {
		return nil, err
	}
	issues := make([]Issue, 0, len(links))
	for _, link := range links {
		issue, err := s.getIssueByParts(ctx, link.DID, link.Collection, link.RKey)
		if err != nil {
			continue
		}
		state, err := s.GetIssueState(ctx, issue.URI)
		if err == nil {
			issue.State = state
		}
		issues = append(issues, *issue)
	}
	assignIssueNumbers(issues)
	if opts.State != "" && opts.State != "all" {
		filtered := issues[:0]
		for _, issue := range issues {
			if issue.State == opts.State {
				filtered = append(filtered, issue)
			}
		}
		issues = filtered
	}
	return issues, nil
}

func (s *IssueService) CreateIssue(ctx context.Context, session *auth.Session, repoURI, title, body string) (*Issue, error) {
	repoDID, err := ResolveRepoDID(ctx, repoURI, s.HTTPClient)
	if err != nil {
		return nil, err
	}
	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}
	record := &core.RepoIssue{
		LexiconTypeID: core.RepoIssueNSID,
		Repo:          repoDID,
		Title:         title,
		Body:          bodyPtr,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	client := NewPDSClient(session, s.HTTPClient)
	out, err := client.CreateRecord(ctx, session.DID, core.RepoIssueNSID, record, nil)
	if err != nil {
		return nil, err
	}
	cid := ""
	if out.Cid != "" {
		cid = out.Cid
	}
	return &Issue{Title: title, Body: body, Repo: repoDID, State: "open", Author: session.DID, CreatedAt: record.CreatedAt, URI: out.Uri, CID: cid}, nil
}

func (s *IssueService) GetIssue(ctx context.Context, issueURI string) (*Issue, error) {
	parsed, err := ParseATURI(issueURI)
	if err != nil {
		return nil, err
	}
	return s.getIssueByParts(ctx, parsed.DID, parsed.Collection, parsed.RKey)
}

func (s *IssueService) UpdateIssue(ctx context.Context, session *auth.Session, issueURI, title, body string, updateTitle, updateBody bool) (*Issue, error) {
	current, err := s.GetIssue(ctx, issueURI)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseATURI(issueURI)
	if err != nil {
		return nil, err
	}
	if parsed.DID != session.DID {
		return nil, fmt.Errorf("cannot edit issue authored by %s", parsed.DID)
	}
	if !updateTitle {
		title = current.Title
	}
	if !updateBody {
		body = current.Body
	}
	var bodyPtr *string
	if body != "" {
		bodyPtr = &body
	}
	record := &core.RepoIssue{
		LexiconTypeID: core.RepoIssueNSID,
		Title:         title,
		Body:          bodyPtr,
		CreatedAt:     current.CreatedAt,
	}
	if current.Repo != "" {
		record.Repo = current.Repo
	}
	client := NewPDSClient(session, s.HTTPClient)
	var swap *string
	if current.CID != "" {
		swap = &current.CID
	}
	out, err := client.PutRecord(ctx, session.DID, core.RepoIssueNSID, parsed.RKey, record, swap)
	if err != nil {
		return nil, err
	}
	current.Title = title
	current.Body = body
	current.CID = out.Cid
	return current, nil
}

func (s *IssueService) SetIssueState(ctx context.Context, session *auth.Session, issueURI, state string) error {
	recordState := core.RepoIssueStateOpen
	if state == "closed" {
		recordState = core.RepoIssueStateClosed
	}
	record := &core.RepoIssueState{
		LexiconTypeID: core.RepoIssueStateNSID,
		Issue:         issueURI,
		State:         recordState,
	}
	client := NewPDSClient(session, s.HTTPClient)
	_, err := client.CreateRecord(ctx, session.DID, core.RepoIssueStateNSID, record, nil)
	return err
}

func (s *IssueService) GetIssueState(ctx context.Context, issueURI string) (string, error) {
	backlinks, err := s.Constellation.GetBacklinks(ctx, issueURI, core.RepoIssueStateNSID, ".issue", 100, "")
	if err != nil {
		return "open", err
	}
	if len(backlinks.Records) == 0 {
		return "open", nil
	}
	sort.Slice(backlinks.Records, func(i, j int) bool {
		return backlinks.Records[i].RKey < backlinks.Records[j].RKey
	})
	latest := backlinks.Records[len(backlinks.Records)-1]
	ident, err := resolveDIDFunc(ctx, latest.DID)
	if err != nil {
		return "open", err
	}
	client := NewAnonymousPDSClient(ident.PDS, s.HTTPClient)
	out, err := client.GetRecord(ctx, latest.DID, latest.Collection, latest.RKey)
	if err != nil {
		return "open", err
	}
	record, ok := out.Value.Val.(*core.RepoIssueState)
	if !ok {
		return "open", nil
	}
	if record.State == core.RepoIssueStateClosed {
		return "closed", nil
	}
	return "open", nil
}

func (s *IssueService) ListComments(ctx context.Context, issueURI string) ([]Comment, error) {
	backlinks, err := s.Constellation.GetBacklinks(ctx, issueURI, core.RepoIssueCommentNSID, ".issue", 100, "")
	if err != nil {
		return nil, err
	}
	comments := make([]Comment, 0, len(backlinks.Records))
	for _, link := range backlinks.Records {
		ident, err := resolveDIDFunc(ctx, link.DID)
		if err != nil {
			continue
		}
		client := NewAnonymousPDSClient(ident.PDS, s.HTTPClient)
		out, err := client.GetRecord(ctx, link.DID, link.Collection, link.RKey)
		if err != nil {
			continue
		}
		record, ok := out.Value.Val.(*core.RepoIssueComment)
		if !ok {
			continue
		}
		cid := ""
		if out.Cid != nil {
			cid = *out.Cid
		}
		comments = append(comments, Comment{Author: link.DID, Body: record.Body, CreatedAt: record.CreatedAt, URI: out.Uri, CID: cid})
	}
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].CreatedAt < comments[j].CreatedAt
	})
	return comments, nil
}

func (s *IssueService) AddComment(ctx context.Context, session *auth.Session, issueURI, body string) (*Comment, error) {
	record := &core.RepoIssueComment{
		LexiconTypeID: core.RepoIssueCommentNSID,
		Issue:         issueURI,
		Body:          body,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	client := NewPDSClient(session, s.HTTPClient)
	out, err := client.CreateRecord(ctx, session.DID, core.RepoIssueCommentNSID, record, nil)
	if err != nil {
		return nil, err
	}
	return &Comment{Author: session.DID, Body: body, CreatedAt: record.CreatedAt, URI: out.Uri, CID: out.Cid}, nil
}

func ResolveIssueIdentifier(input string, issues []Issue) (Issue, error) {
	normalized := strings.TrimPrefix(input, "#")
	if n, err := strconv.Atoi(normalized); err == nil {
		if n < 1 {
			return Issue{}, fmt.Errorf("issue number must be greater than 0")
		}
		assignIssueNumbers(issues)
		for _, issue := range issues {
			if issue.Number == n {
				return issue, nil
			}
		}
		return Issue{}, fmt.Errorf("issue #%d not found", n)
	}
	for _, issue := range issues {
		if RKeyFromURI(issue.URI) == normalized || issue.URI == input {
			return issue, nil
		}
	}
	return Issue{}, fmt.Errorf("issue %q not found", input)
}

func (s *IssueService) getIssueByParts(ctx context.Context, did, collection, rkey string) (*Issue, error) {
	ident, err := resolveDIDFunc(ctx, did)
	if err != nil {
		return nil, err
	}
	client := NewAnonymousPDSClient(ident.PDS, s.HTTPClient)
	out, err := client.GetRecord(ctx, did, collection, rkey)
	if err != nil {
		return nil, err
	}
	record, ok := out.Value.Val.(*core.RepoIssue)
	if !ok {
		return nil, fmt.Errorf("record is not an issue: %s", out.Uri)
	}
	body := ""
	if record.Body != nil {
		body = *record.Body
	}
	cid := ""
	if out.Cid != nil {
		cid = *out.Cid
	}
	return &Issue{Title: record.Title, Body: body, Repo: record.Repo, State: "open", Author: did, CreatedAt: record.CreatedAt, URI: out.Uri, CID: cid}, nil
}

func assignIssueNumbers(issues []Issue) {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].CreatedAt < issues[j].CreatedAt
	})
	for i := range issues {
		issues[i].Number = i + 1
	}
}
