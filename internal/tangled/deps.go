package tangled

import (
	"context"

	"tangled.org/onev.cat/tang/internal/atproto"
)

var (
	resolveDIDFunc    = atproto.ResolveDID
	resolveHandleFunc = atproto.ResolveHandle
)

// SetResolversForTesting replaces AT Protocol identity resolution and returns a restore function.
func SetResolversForTesting(resolveDID, resolveHandle func(context.Context, string) (*atproto.Identity, error)) func() {
	oldDID := resolveDIDFunc
	oldHandle := resolveHandleFunc
	resolveDIDFunc = resolveDID
	resolveHandleFunc = resolveHandle
	return func() {
		resolveDIDFunc = oldDID
		resolveHandleFunc = oldHandle
	}
}
