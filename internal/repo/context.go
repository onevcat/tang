package repo

import (
	"context"
	"errors"
	"fmt"

	"tangled.org/onev.cat/tang/internal/config"
	tanggit "tangled.org/onev.cat/tang/internal/git"
)

var ErrNoRepositoryContext = errors.New("not in a Tangled repository")

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
