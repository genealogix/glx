#!/usr/bin/env bash
set -euo pipefail

cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.."

version=$(tr -d '[:space:]' < .golangci-lint-version)
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  printf 'Invalid .golangci-lint-version: %s (expected vMAJOR.MINOR.PATCH)\n' "$version" >&2
  exit 1
fi

bin_dir=$(go env GOBIN)
if [[ -z "$bin_dir" ]]; then
  go_path=$(go env GOPATH)
  bin_dir="${go_path%%:*}/bin"
fi
target="$bin_dir/golangci-lint"

# Check the copy this script manages, not whichever one PATH finds first.
installed=$("$target" version --short 2>/dev/null || true)
if [[ "v${installed#v}" != "$version" ]]; then
  # The versioned upstream installer checks the release archive's SHA-256.
  installer=$(mktemp)
  trap 'rm -f "$installer"' EXIT
  curl --fail --silent --show-error --location --retry 3 \
    "https://raw.githubusercontent.com/golangci/golangci-lint/$version/install.sh" -o "$installer"
  sh "$installer" -b "$bin_dir" "$version"
fi

resolved=$(command -v golangci-lint || true)
if [[ "$resolved" != "$target" ]]; then
  printf 'Warning: golangci-lint %s is in %s, but PATH resolves %s first.\n' \
    "$version" "$bin_dir" "${resolved:-nothing}" >&2
fi
