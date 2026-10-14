#!/usr/bin/env bash
#
# golangci-lint-staged.sh — lefthook pre-commit job for Go (see lefthook.yml).
#
# Lints what the commit will contain, the way CI will see it:
#
#   - The staged snapshot, not the working tree. The index is exported to a
#     scratch directory, so unstaged edits and untracked files can neither
#     block the commit nor hide a staged finding.
#   - Issues new since the branch point (merge-base with origin/main), which is
#     what CI's `only-new-issues` reports on the PR diff. Using HEAD as the
#     baseline would let `git commit --amend` and commits made with the hook
#     skipped launder their findings into the baseline.
#   - Only the packages that branch diff touches, not the whole module.
#   - The golangci-lint version pinned in .golangci-lint-version, which CI uses.
#
# Run from the repository root: bash scripts/hooks/golangci-lint-staged.sh
set -euo pipefail

want=$(tr -d '[:space:]' < .golangci-lint-version)
have=$(golangci-lint version --short 2>/dev/null || true)
if [ "v${have#v}" != "$want" ]; then
  echo "golangci-lint $want is required (pinned in .golangci-lint-version); found ${have:-none}." >&2
  echo "Install it with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$want" >&2
  exit 1
fi

# Branch point; HEAD when main is unknown (a fresh repo or a clone without
# origin), and nothing at all before the first commit.
base=
if git rev-parse -q --verify HEAD >/dev/null; then
  base=$(git merge-base HEAD origin/main 2>/dev/null || git merge-base HEAD main 2>/dev/null || echo HEAD)
fi

diff_opts=(--cached --no-color --no-ext-diff --src-prefix=a/ --dst-prefix=b/)
if [ -n "$base" ]; then
  changed=$(git diff "${diff_opts[@]}" --name-only --diff-filter=ACMR "$base" -- '*.go')
else
  changed=$(git ls-files -- '*.go')
fi

snap=$(git rev-parse --path-format=absolute --git-path glx-lint-snapshot)
rm -rf "$snap"
mkdir -p "$snap"
trap 'rm -rf "$snap" "$snap.patch"' EXIT

# Export the index. docs/ and website/ hold no Go code and most of the bytes.
git ls-files -z | grep -zv -E '^(docs|website)/' | git checkout-index -z --stdin --prefix="$snap/"

pkgs=()
while IFS= read -r dir; do
  if compgen -G "$snap/$dir/*.go" >/dev/null; then
    pkgs+=("./$dir")
  fi
done < <(printf '%s\n' "$changed" | sed -e '/^$/d' -e 's|/[^/]*$||' -e 's|^[^/]*\.go$|.|' | sort -u)
if [ ${#pkgs[@]} -eq 0 ]; then
  exit 0
fi

args=(run --timeout=5m)
if [ -n "$base" ]; then
  git diff "${diff_opts[@]}" "$base" -- '*.go' > "$snap.patch"
  args+=(--new-from-patch="$snap.patch")
fi

cd "$snap"
golangci-lint "${args[@]}" "${pkgs[@]}"
