#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/release-homebrew.sh <version> [--upload] [--repo owner/repo] [--notes-file path]

Build the macOS arm64 Homebrew release asset for tang.

Arguments:
  <version>          Release version without a leading "v", for example 0.0.1.

Options:
  --upload           Create a GitHub release and upload the generated asset.
  --repo owner/repo  GitHub repository that receives release assets.
                     Defaults to onevcat/homebrew-tap.
  --notes-file path  Release notes used when --upload creates the GitHub release.

The script expects tag v<version> to exist and point to HEAD. It creates:
  dist/tang-<version>-darwin-arm64.tar.gz
  dist/checksums.txt
USAGE
}

version=""
upload="false"
release_repo="onevcat/homebrew-tap"
notes_file=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --upload)
      upload="true"
      shift
      ;;
    --repo)
      release_repo="${2:-}"
      if [[ -z "$release_repo" ]]; then
        echo "error: --repo requires owner/repo" >&2
        exit 2
      fi
      shift 2
      ;;
    --notes-file)
      notes_file="${2:-}"
      if [[ -z "$notes_file" ]]; then
        echo "error: --notes-file requires a path" >&2
        exit 2
      fi
      shift 2
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

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree is not clean" >&2
  exit 1
fi

tag="v${version}"
release_tag="tang-${version}"
head_commit="$(git rev-parse HEAD)"
tag_commit="$(git rev-parse "${tag}^{commit}")"

if [[ "$head_commit" != "$tag_commit" ]]; then
  echo "error: ${tag} does not point to HEAD" >&2
  echo "HEAD: ${head_commit}" >&2
  echo "${tag}: ${tag_commit}" >&2
  exit 1
fi

short_commit="$(git rev-parse --short HEAD)"
asset_name="tang-${version}-darwin-arm64"
asset_dir="dist/${asset_name}"
archive="dist/${asset_name}.tar.gz"
checksums="dist/checksums.txt"

rm -rf dist
mkdir -p "$asset_dir"

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
  go build \
    -trimpath \
    -ldflags "-s -w -X main.version=${version} -X main.commit=${short_commit}" \
    -o "${asset_dir}/tang" \
    ./cmd/tang

if command -v codesign >/dev/null 2>&1; then
  codesign --force --sign - "${asset_dir}/tang"
  codesign --verify --verbose=2 "${asset_dir}/tang"
fi

if [[ "$(uname -s)" == "Darwin" && "$(uname -m)" == "arm64" ]]; then
  "${asset_dir}/tang" version
fi

tar -C "$asset_dir" -czf "$archive" tang
shasum -a 256 "$archive" | tee "$checksums"

echo
echo "Homebrew URL:"
echo "https://github.com/${release_repo}/releases/download/${release_tag}/${asset_name}.tar.gz"

if [[ "$upload" == "true" ]]; then
  release_args=(
    release create "$release_tag" "$archive" "$checksums"
    --repo "$release_repo"
    --title "tang ${version}"
  )
  if [[ -n "$notes_file" ]]; then
    release_args+=(--notes-file "$notes_file")
  else
    release_args+=(--notes "macOS arm64 prebuilt binary for tang ${version}.")
  fi
  gh "${release_args[@]}"
fi
