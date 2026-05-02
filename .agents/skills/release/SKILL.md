---
name: release
description: Release tang end-to-end: infer the next version from git history, update CHANGELOG.md, test, tag, build the macOS arm64 binary, upload the GitHub release asset, update onevcat/homebrew-tap, and verify Homebrew installation.
---

# tang Release Skill

Use this skill when the user says `$release`, "发布", "release", "发版", or asks to ship a new tang version.

The goal is to complete the whole release, not just describe it. The agent owns the release bookkeeping, but must not invent a version blindly.
If the user does not provide a version, infer it from history and proceed. Ask
only when the history supports multiple materially different release choices,
such as a possible breaking release versus an ordinary patch release.

## Release Policy

- Current distribution target: Homebrew formula in `onevcat/homebrew-tap`.
- Current binary target: macOS arm64 only.
- Source version tag in this repo: `vX.Y.Z`.
- GitHub release tag in `onevcat/homebrew-tap`: `tang-X.Y.Z`.
- Homebrew asset: `tang-X.Y.Z-darwin-arm64.tar.gz`.
- The release script performs the mechanical release after changelog and version are ready:

```sh
scripts/release.sh X.Y.Z
```

The default tap checkout path is:

```text
/Users/onevcat/Sync/github/homebrew-tap
```

## Decide The Version

1. Inspect the latest release tag:

```sh
git tag --list 'v*' --sort=-version:refname | head -1
```

2. Inspect history since that tag:

```sh
git log --oneline --decorate <latest-tag>..HEAD
git diff --stat <latest-tag>..HEAD
```

3. Choose the next SemVer version:

- Patch: bug fixes, docs, release tooling, internal cleanup, small CLI behavior fixes.
- Minor: new commands, new user-visible workflows, config keys, or meaningful capability additions.
- Major: incompatible CLI/data/config changes after `1.0.0`. Before `1.0.0`, prefer minor for breaking user-facing changes unless the user explicitly wants `1.0.0`.

For this early project, most ordinary releases after `0.0.1` should be patch releases unless there is a clear new capability.

## Prepare CHANGELOG.md

Before running the release script, update `CHANGELOG.md` with a top-level entry:

```markdown
## X.Y.Z

- User-facing change.
- Another user-facing change.
```

Guidelines:

- Derive bullets from git history since the previous tag.
- Prefer user-facing language over commit-message wording.
- Include release/distribution changes when they affect installation.
- Do not include every internal refactor.
- Keep the existing `0.0.1` entry intact.

Commit and push the changelog before tagging:

```sh
go test ./...
git add CHANGELOG.md
git commit -m "Prepare X.Y.Z release"
git push origin main
```

If other files are intentionally part of the release preparation, include them in the same commit. Do not include unrelated dirty files.

## Run The Release

Run:

```sh
scripts/release.sh X.Y.Z
```

The script will:

- require clean `tang` and `homebrew-tap` worktrees,
- require `CHANGELOG.md` to contain `## X.Y.Z`,
- run `go test ./...`,
- run `make build`,
- create and push `vX.Y.Z`,
- call `scripts/release-homebrew.sh` to build, ad-hoc sign, package, checksum, and upload the macOS arm64 asset,
- update `Formula/tang.rb` in `onevcat/homebrew-tap`,
- commit and push the tap,
- update the installed `onevcat/tap` checkout if present,
- run `brew audit --strict --online onevcat/tap/tang`,
- run `brew reinstall onevcat/tap/tang`,
- run `brew test onevcat/tap/tang`,
- print `tang version`.

## Verification To Report

After release, report:

- source tag, for example `vX.Y.Z`,
- source commit SHA,
- GitHub release URL,
- Homebrew tap commit SHA,
- asset SHA256,
- final `tang version`,
- the install command:

```sh
brew install onevcat/tap/tang
```

## Failure Handling

- If the tag push succeeds but GitHub release or tap update fails, stop and report the exact completed steps. Do not delete published tags or releases automatically.
- If Homebrew audit fails after the tap commit, fix `Formula/tang.rb`, commit, push, and rerun audit/install/test.
- If SSH auth to Tangled fails, retry only after checking that the relevant public key is present with `tang ssh-key list`; do not force-push unrelated refs.
- If a GitHub write is required, use the currently authenticated `gh` account only if it is `onevcat`; otherwise stop and ask the user to switch auth.
