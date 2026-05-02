package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type Protocol string

const (
	ProtocolSSH   Protocol = "ssh"
	ProtocolHTTPS Protocol = "https"
)

type OwnerType string

const (
	OwnerTypeDID    OwnerType = "did"
	OwnerTypeHandle OwnerType = "handle"
)

type URLKind string

const (
	URLKindFetch URLKind = "fetch"
	URLKindPush  URLKind = "push"
)

type Remote struct {
	Owner      string    `json:"owner"`
	OwnerType  OwnerType `json:"ownerType"`
	Name       string    `json:"name"`
	RemoteName string    `json:"remoteName"`
	RemoteURL  string    `json:"remoteUrl"`
	Protocol   Protocol  `json:"protocol"`
	Knot       string    `json:"knot"`
	URLKind    URLKind   `json:"urlKind"`
}

type Runner interface {
	Run(ctx context.Context, cwd string, args ...string) ([]byte, error)
}

type GitRunner struct{}

func (GitRunner) Run(ctx context.Context, cwd string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- executable is fixed; args are produced by tang command wrappers.
	cmd.Dir = cwd
	return cmd.Output()
}

func ListTangledRemotes(ctx context.Context, cwd string, knotHosts []string) ([]Remote, error) {
	return ListTangledRemotesWithRunner(ctx, cwd, knotHosts, GitRunner{})
}

func ListTangledRemotesWithRunner(ctx context.Context, cwd string, knotHosts []string, runner Runner) ([]Remote, error) {
	out, err := runner.Run(ctx, cwd, "remote", "-v")
	if err != nil {
		return nil, nil
	}
	return ParseRemoteList(out, knotHosts), nil
}

func ParseRemoteList(out []byte, knotHosts []string) []Remote {
	type remoteURLs struct {
		fetch []string
		push  []string
	}
	remotes := map[string]*remoteURLs{}
	order := []string{}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		name, rawURL, kind := fields[0], fields[1], strings.Trim(fields[2], "()")
		if _, ok := remotes[name]; !ok {
			remotes[name] = &remoteURLs{}
			order = append(order, name)
		}
		switch kind {
		case string(URLKindFetch):
			remotes[name].fetch = append(remotes[name].fetch, rawURL)
		case string(URLKindPush):
			remotes[name].push = append(remotes[name].push, rawURL)
		}
	}

	var result []Remote
	for _, remoteName := range order {
		urls := remotes[remoteName]
		if parsed, ok := firstParsed(remoteName, urls.fetch, URLKindFetch, knotHosts); ok {
			result = append(result, parsed)
			continue
		}
		if parsed, ok := firstParsed(remoteName, urls.push, URLKindPush, knotHosts); ok {
			result = append(result, parsed)
		}
	}
	return result
}

func ParseRemoteURL(raw string, knotHosts []string) (Remote, bool) {
	host, owner, repo, protocol, ok := splitRemoteURL(raw)
	if !ok || !hostAllowed(host, knotHosts) {
		return Remote{}, false
	}
	ownerType := OwnerTypeHandle
	if strings.HasPrefix(owner, "did:") {
		ownerType = OwnerTypeDID
	}
	if owner == "" || repo == "" {
		return Remote{}, false
	}
	return Remote{
		Owner:     owner,
		OwnerType: ownerType,
		Name:      strings.TrimSuffix(repo, ".git"),
		RemoteURL: raw,
		Protocol:  protocol,
		Knot:      host,
	}, true
}

func firstParsed(remoteName string, urls []string, kind URLKind, knotHosts []string) (Remote, bool) {
	for _, raw := range urls {
		parsed, ok := ParseRemoteURL(raw, knotHosts)
		if ok {
			parsed.RemoteName = remoteName
			parsed.URLKind = kind
			return parsed, true
		}
	}
	return Remote{}, false
}

func splitRemoteURL(raw string) (host, owner, repo string, protocol Protocol, ok bool) {
	if strings.HasPrefix(raw, "git@") {
		parts := strings.SplitN(strings.TrimPrefix(raw, "git@"), ":", 2)
		if len(parts) != 2 {
			return "", "", "", "", false
		}
		owner, repo, ok = splitPath(parts[1])
		return parts[0], owner, repo, ProtocolSSH, ok
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", "", "", "", false
	}
	switch parsed.Scheme {
	case "ssh":
		if parsed.User.Username() != "git" {
			return "", "", "", "", false
		}
		owner, repo, ok = splitPath(parsed.Path)
		return parsed.Hostname(), owner, repo, ProtocolSSH, ok
	case "https":
		owner, repo, ok = splitPath(parsed.Path)
		return parsed.Hostname(), owner, repo, ProtocolHTTPS, ok
	default:
		return "", "", "", "", false
	}
}

func splitPath(rawPath string) (owner, repo string, ok bool) {
	cleaned := strings.Trim(path.Clean("/"+rawPath), "/")
	parts := strings.Split(cleaned, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	owner = parts[0]
	repo = strings.TrimSuffix(parts[1], ".git")
	if !validOwner(owner) || repo == "" {
		return "", "", false
	}
	return owner, repo, true
}

func hostAllowed(host string, allowed []string) bool {
	for _, candidate := range allowed {
		candidate = strings.TrimSpace(candidate)
		candidate = strings.TrimPrefix(candidate, "https://")
		candidate = strings.TrimPrefix(candidate, "http://")
		candidate = strings.TrimSuffix(candidate, "/")
		if strings.EqualFold(host, candidate) {
			return true
		}
	}
	return false
}

var handlePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*[a-zA-Z0-9]$`)

func validOwner(owner string) bool {
	if strings.HasPrefix(owner, "did:") {
		return strings.HasPrefix(owner, "did:plc:") || strings.HasPrefix(owner, "did:web:")
	}
	return handlePattern.MatchString(owner)
}

func CurrentBranch(ctx context.Context, cwd string, runner Runner) (string, error) {
	out, err := runner.Run(ctx, cwd, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := string(bytes.TrimSpace(out))
	if branch == "" {
		return "", errors.New("not on a branch")
	}
	return branch, nil
}

func Clone(ctx context.Context, url string, dir string) error {
	args := []string{"clone", url}
	if dir != "" {
		args = append(args, dir)
	}
	cmd := exec.CommandContext(ctx, "git", args...) // #nosec G204 -- executable is fixed; URL and dir are explicit CLI inputs.
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", bytes.TrimSpace(out), err)
	}
	return nil
}

func CheckoutBranchFromRemote(ctx context.Context, cwd, remote, branch string) error {
	fetch := exec.CommandContext(ctx, "git", "fetch", remote, branch) // #nosec G204 -- executable is fixed; remote/branch are explicit CLI inputs.
	fetch.Dir = cwd
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", bytes.TrimSpace(out), err)
	}
	checkout := exec.CommandContext(ctx, "git", "checkout", "-B", branch, "FETCH_HEAD") // #nosec G204 -- executable is fixed; branch is an explicit CLI input.
	checkout.Dir = cwd
	if out, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %s: %w", bytes.TrimSpace(out), err)
	}
	return nil
}
