#!/usr/bin/env bash
#
# eslint-staged.sh — lefthook pre-commit job for the website (see lefthook.yml).
#
# Lints the staged content of each file passed in, not the working copy: the
# blob and configuration are exported from the index. That keeps unstaged
# edits out of the result, and it lints the LF text the repository stores
# rather than a CRLF Windows checkout, which `linebreak-style` would reject.
# The flags match `npm run lint`, which CI runs.
#
# Run from website/ with paths relative to it:
#   bash ../scripts/hooks/eslint-staged.sh .vitepress/config.js
set -euo pipefail

eslint=node_modules/.bin/eslint
if [ ! -f "$eslint" ]; then
  echo "website/node_modules is not installed; run 'make install-deps' first." >&2
  exit 1
fi

# Export website configuration as well as sources: an unstaged config edit
# must not change which staged findings are reported. Keep installed packages
# available without copying them into the snapshot.
root=$(git rev-parse --show-toplevel)
snap=$(mktemp -d "$(git rev-parse --path-format=absolute --git-path glx-eslint-snapshot).XXXXXX")
trap 'rm -rf "$snap"' EXIT
git -C "$root" ls-files -z -- website/ |
  git -C "$root" checkout-index -z --stdin --prefix="$snap/"
ln -s "$root/website/node_modules" "$snap/website/node_modules"
cd "$snap/website"
eslint="$root/website/$eslint"

status=0
for file in "$@"; do
  git -C "$root/website" show ":./$file" |
    "$eslint" --max-warnings 0 --no-warn-ignored --stdin --stdin-filename "$file" ||
    status=1
done
exit "$status"
