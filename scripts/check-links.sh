#!/usr/bin/env bash
# check-links.sh - Validate internal markdown links
#
# Convention (#1196): links between markdown files are relative repository
# paths that include the .md extension, e.g. `../guides/best-practices.md`.
# That is the one form GitHub renders correctly; the website maps the same
# links through its rewrites table (website/.vitepress/relative-links.js).
# Two forms that only ever worked on the website are therefore reported as
# broken here:
#   - absolute VitePress routes such as `/quickstart` (GitHub resolves them
#     against github.com, so they 404), and
#   - extensionless targets such as `4-entity-types/person` (GitHub does not
#     append .md).
# A directory link resolves if the directory has a README.md or index.md.
#
# Uses POSIX awk only (no gawk extensions) so it runs on macOS as well as CI.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

ERRORS=0
CHECKED=0

LINKS_FILE=$(mktemp)
trap 'rm -f "$LINKS_FILE"' EXIT

# Extract links from a markdown file, skipping fenced code blocks and inline code.
# Uses a single awk pass to strip code and extract link targets with line numbers.
extract_links() {
  local file="$1"
  if ! awk '
    /^[ \t]*```/ { in_code = !in_code; next }
    in_code { next }
    {
      # Remove inline code spans
      gsub(/`[^`]+`/, "")
      # Find all [text](link) patterns on this line
      line = $0
      while (match(line, /\[[^]]*\]\([^)]+\)/)) {
        link = substr(line, RSTART, RLENGTH)
        sub(/^\[[^]]*\]\(/, "", link)
        sub(/\)$/, "", link)
        print FILENAME "|" NR "|" link
        line = substr(line, RSTART + RLENGTH)
      }
    }
  ' "$file"; then
    echo "ERROR: awk failed processing $file" >&2
    exit 1
  fi
}

# Collect links from specification/ and docs/
while IFS= read -r -d '' file; do
  extract_links "$file"
done < <(find specification docs -name "*.md" \
  -not -path "*/node_modules/*" \
  -not -path "*/gedcom-spec/*" \
  -print0 2>/dev/null) > "$LINKS_FILE"

# Collect links from root markdown files and the GitHub-rendered .github/ pages
while IFS= read -r -d '' root_file; do
  extract_links "$root_file"
done < <(find . .github -maxdepth 1 -type f -name "*.md" \
  -not -name "CLAUDE.md" -not -name "PULL_REQUEST_TEMPLATE.md" \
  -print0 2>/dev/null) >> "$LINKS_FILE"

# Validate each relative link
while IFS='|' read -r source_file lineno link; do
  [[ "$link" =~ ^https?:// ]] && continue
  [[ "$link" =~ ^# ]] && continue
  [[ "$link" =~ ^mailto: ]] && continue

  target="${link%%#*}"
  [[ -z "$target" ]] && continue

  CHECKED=$((CHECKED + 1))

  if [[ "$target" =~ ^/ ]]; then
    echo "  BROKEN: $source_file:$lineno -> $link (absolute site route; use a relative repo path ending in .md)"
    ERRORS=$((ERRORS + 1))
    continue
  fi

  source_dir="$(dirname "$source_file")"
  resolved="$source_dir/$target"

  if [[ -f "$resolved" ]] || \
     [[ -e "$resolved/README.md" ]] || \
     [[ -e "$resolved/index.md" ]]; then
    continue
  fi

  if [[ -e "${resolved}.md" ]]; then
    echo "  BROKEN: $source_file:$lineno -> $link (missing .md extension; extensionless links 404 on GitHub)"
  else
    echo "  BROKEN: $source_file:$lineno -> $link"
  fi
  ERRORS=$((ERRORS + 1))
done < "$LINKS_FILE"

echo "Checked $CHECKED relative links across specification/, docs/, root and .github/ markdown files"
if [[ $ERRORS -gt 0 ]]; then
  echo "Found $ERRORS broken link(s)!"
  exit 1
else
  echo "All links OK"
fi
