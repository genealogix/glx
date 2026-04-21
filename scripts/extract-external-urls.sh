#!/usr/bin/env bash
# extract-external-urls.sh - Extract unique external (https?://) URLs from markdown.
# Sibling to check-links.sh (which handles internal links). Used by lychee.yml
# to pre-filter URLs — lychee's own markdown parser cannot handle VitePress
# root-relative routes (e.g. /specification/...) and errors out on them.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

extract_urls() {
  local file="$1"
  awk '
    /^[ \t]*```/ { in_code = !in_code; next }
    in_code { next }
    {
      gsub(/`[^`]+`/, "")
      line = $0
      while (match(line, /\[[^]]*\]\(([^)]+)\)/, arr)) {
        if (arr[1] ~ /^https?:\/\//) print arr[1]
        line = substr(line, RSTART + RLENGTH)
      }
    }
  ' "$file"
}

{
  while IFS= read -r -d '' file; do
    extract_urls "$file"
  done < <(find specification docs -name "*.md" \
    -not -path "*/node_modules/*" \
    -not -path "*/gedcom-spec/*" \
    -print0 2>/dev/null)

  while IFS= read -r -d '' root_file; do
    extract_urls "$root_file"
  done < <(find . -maxdepth 1 -type f -name "*.md" -print0 2>/dev/null)
} | sed 's/[.,;:!?]*$//' | sort -u
