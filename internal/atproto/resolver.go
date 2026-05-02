package atproto

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	coreid "tangled.org/core/idresolver"
)

const defaultPLCURL = "https://plc.directory"

var (
	ErrHandleNotFound = errors.New("handle not found")
	ErrPDSMissing     = errors.New("DID document is missing PDS endpoint")
)

type Directory interface {
	ResolveAtIdentifier(ctx context.Context, input string) (did, handle, pds string, err error)
	ResolveDID(ctx context.Context, did string) (handle, pds string, err error)
}

type CoreDirectory struct {
	resolver *coreid.Resolver
}

func NewCoreDirectory() *CoreDirectory {
	return &CoreDirectory{resolver: coreid.DefaultResolver(defaultPLCURL)}
}

func (d *CoreDirectory) ResolveAtIdentifier(ctx context.Context, input string) (string, string, string, error) {
	ident, err := d.resolver.ResolveAtIdentifier(ctx, input)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: %v", ErrHandleNotFound, err)
	}
	pds := strings.TrimRight(ident.PDSEndpoint(), "/")
	if pds == "" {
		return "", "", "", fmt.Errorf("%w: %s", ErrPDSMissing, ident.DID.String())
	}
	return ident.DID.String(), ident.Handle.String(), pds, nil
}

func (d *CoreDirectory) ResolveDID(ctx context.Context, did string) (string, string, error) {
	parsed, err := syntax.ParseDID(did)
	if err != nil {
		return "", "", err
	}
	ident, err := d.resolver.Directory().LookupDID(ctx, parsed)
	if err != nil {
		return "", "", err
	}
	pds := strings.TrimRight(ident.PDSEndpoint(), "/")
	if pds == "" {
		return "", "", fmt.Errorf("%w: %s", ErrPDSMissing, did)
	}
	handle := ident.Handle.String()
	if handle == "" || handle == syntax.HandleInvalid.String() {
		declared, err := ident.DeclaredHandle()
		if err == nil {
			handle = declared.String()
		}
	}
	return handle, pds, nil
}

func ResolveHandle(ctx context.Context, handle string) (*Identity, error) {
	return ResolveHandleWithDirectory(ctx, NewCoreDirectory(), handle)
}

func ResolveHandleWithDirectory(ctx context.Context, directory Directory, handle string) (*Identity, error) {
	did, resolvedHandle, pds, err := directory.ResolveAtIdentifier(ctx, handle)
	if err != nil {
		return nil, err
	}
	return &Identity{DID: did, Handle: resolvedHandle, PDS: pds}, nil
}

func ResolveDID(ctx context.Context, did string) (*Identity, error) {
	return ResolveDIDWithDirectory(ctx, NewCoreDirectory(), did)
}

func ResolveDIDWithDirectory(ctx context.Context, directory Directory, did string) (*Identity, error) {
	handle, pds, err := directory.ResolveDID(ctx, did)
	if err != nil {
		return nil, err
	}
	return &Identity{DID: did, Handle: handle, PDS: pds}, nil
}
