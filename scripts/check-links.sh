#!/usr/bin/env bash
# check-links.sh - Validate internal markdown links
# Handles VitePress extensionless link convention (e.g., "4-entity-types/person" resolves to person.md)
# Absolute VitePress routes (/specification/...) are skipped — those are checked by the website build.

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
      while (match(line, /\[[^]]*\]\(([^)]+)\)/, arr)) {
        print FILENAME "|" NR "|" arr[1]
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

# Collect links from root markdown files
for root_file in README.md CONTRIBUTING.md CODE_OF_CONDUCT.md; do
  [[ -f "$root_file" ]] || continue
  extract_links "$root_file"
done >> "$LINKS_FILE"

# Validate each relative link
while IFS='|' read -r source_file lineno link; do
  [[ "$link" =~ ^https?:// ]] && continue
  [[ "$link" =~ ^# ]] && continue
  [[ "$link" =~ ^mailto: ]] && continue

  target="${link%%#*}"
  [[ -z "$target" ]] && continue

  # Skip absolute VitePress routes
  [[ "$target" =~ ^/ ]] && continue

  source_dir="$(dirname "$source_file")"
  resolved="$source_dir/$target"

  CHECKED=$((CHECKED + 1))

  if [[ -e "$resolved" ]] || \
     [[ -e "${resolved}.md" ]] || \
     [[ -e "$resolved/README.md" ]] || \
     [[ -e "$resolved/index.md" ]]; then
    continue
  fi

  echo "  BROKEN: $source_file:$lineno -> $link"
  ERRORS=$((ERRORS + 1))
done < "$LINKS_FILE"

echo "Checked $CHECKED relative links across specification/, docs/, and root files"
if [[ $ERRORS -gt 0 ]]; then
  echo "Found $ERRORS broken link(s)!"
  exit 1
else
  echo "All links OK"
fi
