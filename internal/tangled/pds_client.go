package tangled

import (
	"context"
	"net/http"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/lex/util"
	indigoxrpc "github.com/bluesky-social/indigo/xrpc"
	cbg "github.com/whyrusleeping/cbor-gen"
	"tangled.org/onev.cat/tang/internal/auth"
)

type PDSClient struct {
	client *indigoxrpc.Client
}

func NewPDSClient(session *auth.Session, httpClient *http.Client) *PDSClient {
	return &PDSClient{client: &indigoxrpc.Client{
		Host:   session.PDS,
		Client: httpClient,
		Auth: &indigoxrpc.AuthInfo{
			AccessJwt:  session.AccessJWT,
			RefreshJwt: session.RefreshJWT,
			Handle:     session.Handle,
			Did:        session.DID,
		},
	}}
}

func NewAnonymousPDSClient(host string, httpClient *http.Client) *PDSClient {
	return &PDSClient{client: &indigoxrpc.Client{Host: host, Client: httpClient}}
}

func (c *PDSClient) CreateSession(ctx context.Context, identifier, password string) (*comatproto.ServerCreateSession_Output, error) {
	return comatproto.ServerCreateSession(ctx, c.client, &comatproto.ServerCreateSession_Input{
		Identifier: identifier,
		Password:   password,
	})
}

func (c *PDSClient) RefreshSession(ctx context.Context, session *auth.Session) (*auth.Session, error) {
	refreshClient := &indigoxrpc.Client{
		Host:   session.PDS,
		Client: c.client.Client,
		Auth: &indigoxrpc.AuthInfo{
			AccessJwt:  session.RefreshJWT,
			RefreshJwt: session.RefreshJWT,
			Handle:     session.Handle,
			Did:        session.DID,
		},
	}
	out, err := comatproto.ServerRefreshSession(ctx, refreshClient)
	if err != nil {
		return nil, err
	}
	return auth.NewSession(out.Did, out.Handle, session.PDS, out.AccessJwt, out.RefreshJwt)
}

func (c *PDSClient) CreateRecord(ctx context.Context, repo, collection string, record cbg.CBORMarshaler, rkey *string) (*comatproto.RepoCreateRecord_Output, error) {
	validate := false
	return comatproto.RepoCreateRecord(ctx, c.client, &comatproto.RepoCreateRecord_Input{
		Repo:       repo,
		Collection: collection,
		Rkey:       rkey,
		Record:     &util.LexiconTypeDecoder{Val: record},
		Validate:   &validate,
	})
}

func (c *PDSClient) DeleteRecord(ctx context.Context, repo, collection, rkey string) error {
	_, err := comatproto.RepoDeleteRecord(ctx, c.client, &comatproto.RepoDeleteRecord_Input{
		Repo:       repo,
		Collection: collection,
		Rkey:       rkey,
	})
	return err
}

func (c *PDSClient) ListRecords(ctx context.Context, repo, collection string, limit int64, cursor string) (*comatproto.RepoListRecords_Output, error) {
	return comatproto.RepoListRecords(ctx, c.client, collection, cursor, limit, repo, false)
}
