package tangled

import (
	"context"
	"net/http"
	"strings"

	core "tangled.org/core/api/tangled"

	indigoxrpc "github.com/bluesky-social/indigo/xrpc"
)

type KnotClient struct {
	client *indigoxrpc.Client
}

type KnotClientOption func(*indigoxrpc.Client)

func WithKnotHTTPClient(client *http.Client) KnotClientOption {
	return func(c *indigoxrpc.Client) {
		c.Client = client
	}
}

func WithServiceAuthToken(token string) KnotClientOption {
	return func(c *indigoxrpc.Client) {
		c.Auth = &indigoxrpc.AuthInfo{AccessJwt: token}
	}
}

func WithKnotBaseURL(baseURL string) KnotClientOption {
	return func(c *indigoxrpc.Client) {
		c.Host = strings.TrimRight(baseURL, "/")
	}
}

func NewKnotClient(knotHost string, opts ...KnotClientOption) *KnotClient {
	client := &indigoxrpc.Client{Host: "https://" + strings.TrimRight(knotHost, "/")}
	for _, opt := range opts {
		opt(client)
	}
	return &KnotClient{client: client}
}

func (c *KnotClient) Compare(ctx context.Context, repo, rev1, rev2 string) ([]byte, error) {
	return core.RepoCompare(ctx, c.client, repo, rev1, rev2)
}

func (c *KnotClient) Diff(ctx context.Context, repo, ref string) ([]byte, error) {
	return core.RepoDiff(ctx, c.client, ref, repo)
}

func (c *KnotClient) CreateRepo(ctx context.Context, input *core.RepoCreate_Input) (*core.RepoCreate_Output, error) {
	return core.RepoCreate(ctx, c.client, input)
}

func (c *KnotClient) Merge(ctx context.Context, input *core.RepoMerge_Input) error {
	return core.RepoMerge(ctx, c.client, input)
}
