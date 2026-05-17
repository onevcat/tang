# Changelog

All notable user-facing changes are recorded here.

## Unreleased

## 0.0.2

- Align numeric issue and pull request arguments with Tangled AppView numbers.
- Added `--atproto` raw record mode for issue and pull request list/view/action
  commands; it accepts AT URIs, rkeys, or unique rkey prefixes and rejects
  numeric AppView IDs.
- Issue and pull request list output now shows stable rkeys instead of local
  synthetic numbers.

## 0.0.1

- Initial Homebrew release for macOS arm64.
- Added authentication, SSH key management, repository, issue, pull request, and
  browser workflows for Tangled.
- Added configurable clone protocol support.
- Documented current Tangled AppView synchronization and patch-merge
  limitations.
