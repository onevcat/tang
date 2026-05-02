package tangled

import (
	"context"
	"fmt"
	"time"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
)

const defaultServiceAuthTTL = 60 * time.Second

type ServiceAuthRequest struct {
	Audience string
	Lexicon  string
	Expires  time.Time
}

func BuildServiceAuthRequest(knotHost, lexicon string, now time.Time, ttl time.Duration) ServiceAuthRequest {
	if ttl <= 0 {
		ttl = defaultServiceAuthTTL
	}
	return ServiceAuthRequest{
		Audience: fmt.Sprintf("did:web:%s", knotHost),
		Lexicon:  lexicon,
		Expires:  now.Add(ttl),
	}
}

func (c *PDSClient) GetServiceAuth(ctx context.Context, knotHost, lexicon string, ttl time.Duration) (string, error) {
	req := BuildServiceAuthRequest(knotHost, lexicon, time.Now(), ttl)
	out, err := comatproto.ServerGetServiceAuth(ctx, c.client, req.Audience, req.Expires.Unix(), req.Lexicon)
	if err != nil {
		return "", err
	}
	return out.Token, nil
}
