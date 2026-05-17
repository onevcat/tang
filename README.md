# tang

`tang` is a command-line client for [Tangled](https://tangled.org/).
It is designed for day-to-day repository work from a terminal: checking your
Tangled identity, managing SSH keys, finding the current repository, listing and
creating issues, and working with pull requests.

The tool talks to Tangled through its AT Protocol data model:

- Authentication is performed against the user's PDS.
- Issues, pull requests, comments, status records, and SSH keys are stored as
  ATProto records.
- Repository and pull request operations use the Tangled AppView, Constellation,
  and knot APIs as needed.

> [!WARNING]
>
> **Project status:** `tang` is early and experimental software.
> I use it for my own Tangled repositories and daily workflows, but it is still
> new and relatively unproven. See [Known Limitations](#known-limitations)
> before relying on it for critical work.

## Status

This project is usable for the implemented workflows, but still early. The
current command set covers:

- `auth`: login, logout, token refresh, token inspection, auth status.
- `status`: combined auth, repository, and service status.
- `config`: local configuration for knot, AppView, Constellation, and preferred
  git remote.
- `ssh-key`: list, add, and delete Tangled SSH public keys.
- `repo`: view, list, create, and clone repositories.
- `issue`: list, create, view, edit, comment, close, and reopen issues.
- `pr`: list, create, view, diff, comment, checkout, close, reopen, and merge
  pull requests.
- `browse`: open Tangled pages from the current repository context.
- `completion`: generate shell completions.

Some Tangled server-side behavior is eventually consistent. After creating,
closing, reopening, or commenting on records, AppView and Constellation results
may take a short time to reflect the latest state.

## Installation

Install with Homebrew on macOS arm64:

```sh
brew install onevcat/tap/tang
```

Build a local binary:

```sh
make build
./bin/tang --help
```

Install into your Go binary path:

```sh
make install
tang --help
```

The module currently targets Go 1.25.

## First-Time Setup

Log in with a Tangled handle and app password:

```sh
tang auth login --handle alice.example
```

The command prompts for the app password. Session credentials are stored through
the operating system keyring when available. The PDS URL is resolved from the
handle's DID document; `pds.url` is intentionally not a persistent config key.

Check the session:

```sh
tang auth status
tang auth refresh
tang auth token --json
```

Manage SSH keys:

```sh
tang ssh-key list
tang ssh-key add ~/.ssh/id_ed25519.pub
tang ssh-key delete <key-rkey>
```

Use `ssh-key list --json` when you need stable record identifiers for deletion
or automation.

## Repository Context

Most repository-scoped commands can infer the target repository from the current
git worktree. `tang` scans git remotes and selects a Tangled remote using this
order:

1. The configured preferred remote from `tang config set remote <name>`.
2. `origin`, when it is a Tangled remote.
3. The first Tangled remote found.

Supported repository selectors include:

```sh
tang status
tang repo view
tang issue list
tang pr list
```

You can override the inferred repository with `-R`:

```sh
tang issue list -R onev.cat/tang
tang pr view 1 -R tangled.org/core
```

Numeric issue and pull request arguments match Tangled AppView URLs. For
example, `tang issue view 5` resolves the same object as
`/OWNER/REPO/issues/5`; `tang pr view 3` resolves `/OWNER/REPO/pulls/3`.
Use `--atproto` when you need to inspect raw PDS/Constellation records that
may not be projected by AppView.

The selector format is `[HOST/]OWNER/REPO`. If no host is provided, configured
knot hosts are used.

## Core Workflows

### Inspect Your Environment

```sh
tang status
tang status --section auth
tang status --section repo
tang config list
```

Use JSON output for scripts:

```sh
tang status --json
tang status --json=auth,repository
tang repo view --json=name,knot,uri
```

`--json` without a value prints all fields. `--json=a,b,c` prints only selected
fields.

### Work With Repositories

View the current repository:

```sh
tang repo view
```

List repositories for an owner:

```sh
tang repo list onev.cat
```

Clone a repository:

```sh
tang repo clone onev.cat/tang-playground
tang repo clone onev.cat/tang-playground ./playground
tang repo clone git@tangled.org:onev.cat/tang-playground ./playground
```

When the repository argument is `OWNER/REPO`, `repo clone` chooses the clone URL
from `clone.protocol`. The default is `https`, matching GitHub CLI's configured
protocol model. Set it to `ssh` when you want Tangled's SSH clone URL:

```sh
tang config get clone.protocol
tang config set clone.protocol ssh
tang config set clone.protocol https
```

When the repository argument is an explicit clone URL, `repo clone` passes it
directly to `git clone` and does not consult `clone.protocol`.

Create a repository:

```sh
tang repo create my-tool --description "A small Tangled tool" --knot knot1.tangled.sh
```

Repository creation depends on the selected knot. If the default AppView does
not expose a create-capable route, pass an explicit `--knot`.

### Work With Issues

List issues:

```sh
tang issue list
tang issue list --state all
tang issue list --state closed --limit 20
tang issue list --atproto
```

Create an issue:

```sh
tang issue create "Fix repository context detection" --body "Remote parsing misses push URLs."
```

Read a body from a file or stdin:

```sh
tang issue create "Document release process" --body-file ./issue.md
printf 'Body from stdin\n' | tang issue create "CLI polish" --body-file -
```

View, comment, edit, and change state:

```sh
tang issue view 1
tang issue view 1 --web
tang issue comment 1 --body "Confirmed on main."
tang issue edit 1 --title "Fix remote context detection"
tang issue close 1
tang issue reopen 1
```

Numeric issue arguments such as `1` or `#1` are AppView issue numbers. They are
resolved through the configured AppView and match URLs like
`/onev.cat/tang/issues/5`.

For raw protocol inspection, use `--atproto`. In that mode numeric IDs are not
accepted, because they only have meaning in AppView. Pass a full AT URI, a full
rkey, or a unique rkey prefix instead:

```sh
tang issue list --atproto
tang issue view 3mkuteff --atproto
tang issue comment 3mkuteffbxa2b --atproto --body "More detail."
```

New records may need a short time before they appear in `issue list`, because
Tangled discovers cross-record links through Constellation indexing. After
creating an issue, keep the returned AT URI or rkey and use it directly for
follow-up commands:

```sh
tang issue view 3mkuteffbxa2b
tang issue comment 3mkuteffbxa2b --body "More detail."
```

### Work With Pull Requests

List and inspect pull requests:

```sh
tang pr list
tang pr list --state all
tang pr list --atproto
tang pr view 1
tang pr view 1 --web
tang pr diff 1
```

Create a pull request from a pushed branch:

```sh
git push origin my-branch
tang pr create --base main --head my-branch --title "Add repository docs" --body-file ./pr.md
```

When the patch contains a mail-style subject, `--fill` can derive the title:

```sh
tang pr create --base main --head my-branch --fill
```

Review and update a pull request:

```sh
tang pr comment 1 --body "Looks good after the latest patch."
tang pr checkout 1
tang pr close 1
tang pr reopen 1
```

Merge a pull request:

```sh
tang pr merge 1 --subject "Merge pull request #1"
```

`pr merge` sends the pull request patch to the repository's knot through
Tangled's `sh.tangled.repo.merge` endpoint, then records the pull request as
merged. It follows the same remote merge mechanism used by the Tangled web UI,
with a narrower CLI surface: one pull request patch is merged at a time, and
GitHub-style `--squash` / `--rebase` strategies are not part of the current
Tangled endpoint.

Numeric pull request arguments such as `1` or `#1` are AppView pull numbers.
They are resolved through the configured AppView and match URLs like
`/onev.cat/tang/pulls/3`.

For raw protocol inspection, use `--atproto`. In that mode numeric IDs are not
accepted; pass a full AT URI, a full rkey, or a unique rkey prefix instead:

```sh
tang pr list --atproto
tang pr view 3mkuu6 --atproto
tang pr diff 3mkuu6q672u22 --atproto
```

### Open Tangled In A Browser

Open the current repository page:

```sh
tang browse
```

Open an issue:

```sh
tang browse issue 1
```

`browse issue` uses AppView issue numbers. For raw ATProto records that are not
projected by AppView, use `issue view --atproto` and the AT URI/rkey instead.

### Configure Services

Configuration is stored under the user config directory as `tang/config.toml`
for the current platform.

Common settings:

```sh
tang config list
tang config get knot.hosts
tang config set knot.hosts knot1.tangled.sh,tangled.org
tang config get clone.protocol
tang config set clone.protocol ssh
tang config set appview.url https://tangled.org
tang config set constellation.url https://constellation.microcosm.blue
tang config set remote origin
```

`TANG_CONSTELLATION_URL` overrides `constellation.url` at runtime. This is useful
for testing against a local or staging Constellation instance.

## Global Flags

All commands support:

```text
--json[=fields]   Output JSON, optionally filtered by comma-separated fields.
--pds URL         Override PDS URL for auth and testing.
-R, --repo REPO   Select another repository using [HOST/]OWNER/REPO.
```

Use `--pds` for tests and diagnostics. Normal login and write workflows should
let `tang` resolve the PDS from the active identity.

## Shell Completion

Generate shell completion scripts:

```sh
tang completion zsh
tang completion bash
tang completion fish
tang completion powershell
```

## Development

Useful commands:

```sh
make test
make lint
make build
./bin/tang --help
```

Prepare a release by updating `CHANGELOG.md`, then run the full release script:

```sh
scripts/release.sh 0.0.2
```

The release script runs tests, tags and pushes `vX.Y.Z`, builds a precompiled
macOS arm64 binary, applies an ad-hoc code signature when `codesign` is
available, uploads the asset to `onevcat/homebrew-tap`, updates the Homebrew
formula, and verifies installation with `brew`.

The main implementation areas are:

- `cmd/tang`: binary entry point.
- `internal/cli`: Cobra command wiring and user-facing behavior.
- `internal/auth`: session and keyring handling.
- `internal/config`: TOML config loading and supported keys.
- `internal/git`: git remote parsing.
- `internal/repo`: repository context resolution.
- `internal/tangled`: Tangled AppView, PDS, knot, issue, repository, and pull
  request clients.
- `internal/constellation`: Constellation API client.

Before changing behavior, prefer adding focused tests near the package that owns
the logic. For command behavior, use `internal/cli` tests. For record parsing,
resolution, and API mapping, use `internal/tangled`, `internal/git`, or
`internal/repo` tests.

## Known Limitations

- Tangled AppView and Constellation indexing may lag behind successful PDS
  writes. For newly created records, keep the returned AT URI or rkey and use it
  directly while indexes catch up.
- Numeric issue and pull request IDs are AppView IDs. If AppView misses a
  valid PDS record, numeric resolution cannot find it; use `--atproto` with the
  record rkey, rkey prefix, or full AT URI to inspect raw protocol data.
- AppView is not yet a complete projection of Tangled's ATProto records.
  Protocol-level issue state records (`sh.tangled.repo.issue.state`) and pull
  request status records (`sh.tangled.repo.pull.status`) can be visible to
  `tang` while the web UI still shows stale state. Track upstream progress in
  [tangled.org/core#462](https://tangled.org/tangled.org/core/issues/462),
  [tangled.org/core#282](https://tangled.org/tangled.org/core/issues/282), and
  [tangled.org/core#517](https://tangled.org/tangled.org/core/issues/517) for
  status records that are not synchronized into AppView; this is an upstream
  AppView projection issue rather than a `tang` CLI state bug.
- AppView-created pull requests and CLI-created pull requests do not currently
  exercise the same ingestion path. The web UI writes its own AppView database
  rows when it creates a pull request, while `tang pr create` writes an
  ATProto `sh.tangled.repo.pull` record and depends on AppView's firehose
  ingester to project it. We have observed CLI-created pull records that exist
  on the author's PDS and are indexed by Constellation, with valid gzipped patch
  blobs, but never receive a `/pulls/N` AppView page. Track the concrete case in
  [tangled.org/core#576](https://tangled.org/tangled.org/core/issues/576).
  Also see [tangled.org/core#575](https://tangled.org/tangled.org/core/issues/575)
  for related skipped pull/issue projection and
  [the pull status ingestion record](https://pdsls.dev/at://did:plc:kl2ejrmz5zmxnno3ll4luz76/sh.tangled.repo.issue/3mkuyuh6t3l2k).
- Some older pull request records do not contain patch rounds, so `pr diff` and
  `pr checkout` cannot operate on them.
- SSH clone depends on Tangled's SSH key authorization index. If a freshly added
  key is rejected, retry after the key appears in `tang ssh-key list`, or use
  the default HTTPS clone protocol.
- `repo create` requires a create-capable knot. Pass `--knot` when the default
  service route is not sufficient.
- `pr merge` depends on the repository knot's `sh.tangled.repo.merge` endpoint.
  It is a remote patch merge, not a local worktree merge. The knot applies the
  pull request patch on top of the target branch and creates new commits, so the
  commits that land on the target branch do not keep the same object IDs as the
  source branch. This is closer to replaying patches than to a GitHub-style merge
  commit, and the source branch is not automatically advanced or deleted.
