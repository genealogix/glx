#!/usr/bin/env bash
#
# eslint-staged.sh — lefthook pre-commit job for the website (see lefthook.yml).
#
# Lints the staged content of each file passed in, not the working copy: the
# blob is read from the index and fed to eslint on stdin. That keeps unstaged
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

status=0
for file in "$@"; do
  git show ":./$file" |
    "$eslint" --max-warnings 0 --no-warn-ignored --stdin --stdin-filename "$file" ||
    status=1
done
exit "$status"
