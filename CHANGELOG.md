# Changelog

All notable user-facing changes are recorded here.

## Unreleased

- Repository lookup now supports web-created repository records whose
  `record.name` is empty by falling back to the repo record rkey.

## 0.0.2

- Updated `tangled.org/core` to `v1.14.0-alpha` to match the current Tangled
  record schema.
- Align numeric issue and pull request arguments with Tangled AppView numbers.
- Added `--atproto` raw record mode for issue and pull request list/view/action
  commands; it accepts AT URIs, rkeys, or unique rkey prefixes and rejects
  numeric AppView IDs.
- Issue and pull request list output now shows stable rkeys instead of local
  synthetic numbers.
- Issue and pull request records now include repository DIDs, and related
  backlink records are written so AppView can associate records with the
  correct repository identity.
- Expanded coverage for repository resolution, CLI workflows, Tangled service
  interactions, authentication, config, and output formatting.

## 0.0.1

- Initial Homebrew release for macOS arm64.
- Added authentication, SSH key management, repository, issue, pull request, and
  browser workflows for Tangled.
- Added configurable clone protocol support.
- Documented current Tangled AppView synchronization and patch-merge
  limitations.
