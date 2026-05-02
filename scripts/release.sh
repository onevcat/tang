#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/release.sh <version> [options]

Release tang and update the Homebrew tap.

Arguments:
  <version>             Release version without a leading "v", for example 0.0.2.

Options:
  --tap-path <path>     Local checkout of onevcat/homebrew-tap.
                        Defaults to /Users/onevcat/Sync/github/homebrew-tap.
  --release-repo <repo> GitHub repository that receives release assets.
                        Defaults to onevcat/homebrew-tap.
  --remote <name>       Git remote used for the tang repository. Defaults to origin.
  --dry-run             Validate and prepare local changes without pushing tags,
                        creating GitHub releases, or pushing the tap.

The script performs:
  1. Validate version, clean worktrees, and CHANGELOG.md entry.
  2. Run go test ./... and make build.
  3. Create and push tag v<version>.
  4. Build, sign, package, checksum, and upload the macOS arm64 asset.
  5. Update Formula/tang.rb in the Homebrew tap.
  6. Commit and push the Homebrew tap.
  7. Pull the installed tap checkout when present.
  8. Run brew audit, reinstall, and brew test.
USAGE
}

version=""
tap_path="/Users/onevcat/Sync/github/homebrew-tap"
release_repo="onevcat/homebrew-tap"
remote="origin"
dry_run="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --tap-path)
      tap_path="${2:-}"
      if [[ -z "$tap_path" ]]; then
        echo "error: --tap-path requires a path" >&2
        exit 2
      fi
      shift 2
      ;;
    --release-repo)
      release_repo="${2:-}"
      if [[ -z "$release_repo" ]]; then
        echo "error: --release-repo requires owner/repo" >&2
        exit 2
      fi
      shift 2
      ;;
    --remote)
      remote="${2:-}"
      if [[ -z "$remote" ]]; then
        echo "error: --remote requires a git remote name" >&2
        exit 2
      fi
      shift 2
      ;;
    --dry-run)
      dry_run="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    -*)
      echo "error: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [[ -n "$version" ]]; then
        echo "error: version was provided more than once" >&2
        usage >&2
        exit 2
      fi
      version="$1"
      shift
      ;;
  esac
done

if [[ -z "$version" ]]; then
  usage >&2
  exit 2
fi

if [[ "$version" == v* ]]; then
  echo "error: version must not start with v" >&2
  exit 2
fi
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: version must be X.Y.Z" >&2
  exit 2
fi

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

tag="v${version}"
release_tag="tang-${version}"
asset_name="tang-${version}-darwin-arm64"
asset_url="https://github.com/${release_repo}/releases/download/${release_tag}/${asset_name}.tar.gz"
tmp_dir="$(mktemp -d)"
notes_file="${tmp_dir}/release-notes-${version}.md"
temporary_tag_created="false"

cleanup() {
  rm -rf "$tmp_dir"
  if [[ "$temporary_tag_created" == "true" ]]; then
    git tag -d "$tag" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

require_clean_worktree() {
  local path="$1"
  if [[ -n "$(git -C "$path" status --porcelain)" ]]; then
    echo "error: working tree is not clean: $path" >&2
    git -C "$path" status --short >&2
    exit 1
  fi
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

changelog_entry() {
  awk -v version="$version" '
    $0 == "## " version { found=1; next }
    found && /^## / { exit }
    found { print }
  ' CHANGELOG.md
}

update_formula() {
  local formula="$1"
  local sha="$2"
  cat >"$formula" <<FORMULA
class Tang < Formula
  desc "Command-line client for Tangled"
  homepage "https://tangled.org/onev.cat/tang"
  url "${asset_url}"
  version "${version}"
  sha256 "${sha}"
  license "MIT"

  depends_on arch: :arm64
  depends_on :macos

  def install
    bin.install "tang"
    generate_completions_from_executable(bin/"tang", "completion")
  end

  test do
    assert_match "tang version #{version}", shell_output("#{bin}/tang version")
  end
end
FORMULA
}

require_command git
require_command go
require_command gh
require_command shasum
require_command brew

gh_login="$(gh api user --jq .login)"
if [[ "$gh_login" != "onevcat" ]]; then
  echo "error: gh must be authenticated as onevcat, got ${gh_login}" >&2
  exit 1
fi

tap_path="$(cd "$tap_path" && pwd)"
if [[ ! -d "$tap_path/.git" ]]; then
  echo "error: --tap-path is not a git checkout: $tap_path" >&2
  exit 1
fi

if [[ ! -f CHANGELOG.md ]]; then
  echo "error: CHANGELOG.md is required" >&2
  exit 1
fi

notes="$(changelog_entry)"
if [[ -z "$(printf "%s" "$notes" | tr -d '[:space:]')" ]]; then
  echo "error: CHANGELOG.md does not contain a ## ${version} entry" >&2
  exit 1
fi

require_clean_worktree "$repo_root"
require_clean_worktree "$tap_path"

git fetch "$remote" main --tags
if [[ "$(git branch --show-current)" != "main" ]]; then
  echo "error: release must run from tang main" >&2
  exit 1
fi
if [[ "$(git rev-parse HEAD)" != "$(git rev-parse "${remote}/main")" ]]; then
  echo "error: local main is not equal to ${remote}/main" >&2
  exit 1
fi
git -C "$tap_path" fetch origin main --tags
if [[ "$(git -C "$tap_path" branch --show-current)" != "main" ]]; then
  echo "error: tap checkout must be on main" >&2
  exit 1
fi
git -C "$tap_path" merge --ff-only origin/main

if git rev-parse -q --verify "refs/tags/${tag}" >/dev/null; then
  echo "error: local tag already exists: ${tag}" >&2
  exit 1
fi
if git ls-remote --exit-code --tags "$remote" "$tag" >/dev/null 2>&1; then
  echo "error: remote tag already exists: ${tag}" >&2
  exit 1
fi
if gh release view "$release_tag" --repo "$release_repo" >/dev/null 2>&1; then
  echo "error: GitHub release already exists: ${release_tag}" >&2
  exit 1
fi

go test ./...
make build

mkdir -p dist
{
  echo "macOS arm64 prebuilt binary for tang ${version}."
  echo
  echo "Changes:"
  printf "%s\n" "$notes"
} >"$notes_file"

if [[ "$dry_run" == "true" ]]; then
  git tag -a "$tag" -m "tang ${tag}"
  temporary_tag_created="true"
  scripts/release-homebrew.sh "$version" --repo "$release_repo"
else
  git tag -a "$tag" -m "tang ${tag}"
  git push "$remote" "$tag"
  scripts/release-homebrew.sh "$version" --upload --repo "$release_repo" --notes-file "$notes_file"
fi

sha="$(shasum -a 256 "dist/${asset_name}.tar.gz" | awk '{print $1}')"
formula_path="$tap_path/Formula/tang.rb"
if [[ "$dry_run" == "true" ]]; then
  formula_path="dist/tang.rb"
fi
update_formula "$formula_path" "$sha"

if [[ "$dry_run" == "true" ]]; then
  echo "dry-run: wrote $formula_path"
  exit 0
fi

git -C "$tap_path" add Formula/tang.rb README.md
git -C "$tap_path" commit -m "Update tang to ${version}"
git -C "$tap_path" push origin main

if brew tap | grep -qx "onevcat/tap"; then
  installed_tap_path="$(brew --repo onevcat/tap)"
  git -C "$installed_tap_path" pull --ff-only
fi

HOMEBREW_NO_AUTO_UPDATE=1 brew audit --strict --online onevcat/tap/tang
HOMEBREW_NO_AUTO_UPDATE=1 brew reinstall onevcat/tap/tang
HOMEBREW_NO_AUTO_UPDATE=1 brew test onevcat/tap/tang
tang version
