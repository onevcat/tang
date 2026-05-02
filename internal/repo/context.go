package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tangled.org/onev.cat/tang/internal/config"
	tanggit "tangled.org/onev.cat/tang/internal/git"
)

var ErrNoRepositoryContext = errors.New("not in a Tangled repository")
var ErrInvalidRepositorySelector = errors.New("invalid repository selector")

type RepositoryContext struct {
	Owner      string            `json:"owner"`
	OwnerType  tanggit.OwnerType `json:"ownerType"`
	Name       string            `json:"name"`
	RemoteName string            `json:"remoteName"`
	RemoteURL  string            `json:"remoteUrl"`
	Protocol   tanggit.Protocol  `json:"protocol"`
	Knot       string            `json:"knot"`
	URLKind    tanggit.URLKind   `json:"urlKind"`
}

func Resolve(ctx context.Context, cwd string, cfg *config.Config) (*RepositoryContext, error) {
	remotes, err := tanggit.ListTangledRemotes(ctx, cwd, cfg.Knot.Hosts)
	if err != nil {
		return nil, err
	}
	return SelectRemote(remotes, cfg.Remote)
}

func ResolveSelector(selector string, cfg *config.Config) (*RepositoryContext, error) {
	selector = strings.Trim(strings.TrimSpace(selector), "/")
	selector = strings.TrimPrefix(selector, "https://")
	selector = strings.TrimPrefix(selector, "http://")
	parts := strings.Split(selector, "/")
	knot := defaultKnotHost(cfg)
	var owner, name string
	switch len(parts) {
	case 2:
		owner, name = parts[0], parts[1]
	case 3:
		knot, owner, name = strings.TrimSuffix(parts[0], "/"), parts[1], parts[2]
	default:
		return nil, fmt.Errorf("%w: repository must be [HOST/]OWNER/NAME", ErrInvalidRepositorySelector)
	}
	if owner == "" || name == "" || knot == "" {
		return nil, fmt.Errorf("%w: repository must be [HOST/]OWNER/NAME", ErrInvalidRepositorySelector)
	}
	ownerType := tanggit.OwnerTypeHandle
	if strings.HasPrefix(owner, "did:") {
		ownerType = tanggit.OwnerTypeDID
	}
	return &RepositoryContext{
		Owner:     owner,
		OwnerType: ownerType,
		Name:      strings.TrimSuffix(name, ".git"),
		Knot:      knot,
	}, nil
}

func SelectRemote(remotes []tanggit.Remote, configuredRemote string) (*RepositoryContext, error) {
	if len(remotes) == 0 {
		return nil, ErrNoRepositoryContext
	}
	if configuredRemote != "" {
		for _, remote := range remotes {
			if remote.RemoteName == configuredRemote {
				return fromRemote(remote), nil
			}
		}
	}
	for _, remote := range remotes {
		if remote.RemoteName == "origin" {
			return fromRemote(remote), nil
		}
	}
	return fromRemote(remotes[0]), nil
}

func (r RepositoryContext) Identifier() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

func fromRemote(remote tanggit.Remote) *RepositoryContext {
	return &RepositoryContext{
		Owner:      remote.Owner,
		OwnerType:  remote.OwnerType,
		Name:       remote.Name,
		RemoteName: remote.RemoteName,
		RemoteURL:  remote.RemoteURL,
		Protocol:   remote.Protocol,
		Knot:       remote.Knot,
		URLKind:    remote.URLKind,
	}
}

func defaultKnotHost(cfg *config.Config) string {
	if cfg != nil && len(cfg.Knot.Hosts) > 0 {
		return cfg.Knot.Hosts[0]
	}
	return config.DefaultKnotHost
}
