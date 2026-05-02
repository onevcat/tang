package auth

import (
	"context"
	"time"
)

const refreshSkew = 5 * time.Minute

type Refresher interface {
	RefreshSession(ctx context.Context, session *Session) (*Session, error)
}

func NeedsRefresh(session *Session, now time.Time) bool {
	if session == nil {
		return false
	}
	if session.ExpiresAt.IsZero() {
		return true
	}
	return !session.ExpiresAt.After(now.Add(refreshSkew))
}

func EnsureFresh(ctx context.Context, session *Session, refresher Refresher, now time.Time) (*Session, error) {
	if !NeedsRefresh(session, now) {
		return session, nil
	}
	return refresher.RefreshSession(ctx, session)
}
