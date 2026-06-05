#!/usr/bin/env bash
# Validate changie changelog fragments (#858, implements #321):
#   1. every fragment parses and renders (catches malformed YAML / unknown kind)
#   2. every fragment carries an issue/PR reference in its Issue field
# Run via `make changelog-check` or the changelog-fragments CI workflow.
set -euo pipefail

FRAG_DIR=".changes/unreleased"

if ! command -v changie >/dev/null 2>&1; then
  echo "ERROR: changie not found on PATH. Run 'make changelog-check' (auto-installs) or 'go install github.com/miniscruff/changie@v1.24.0'." >&2
  exit 1
fi

shopt -s nullglob
frags=("$FRAG_DIR"/*.yaml "$FRAG_DIR"/*.yml)
if [ ${#frags[@]} -eq 0 ]; then
  echo "No unreleased changelog fragments to validate."
  exit 0
fi

# 1. Parse + render check. A throwaway version keeps this independent of the
#    real next version; --dry-run does not delete fragments or write files.
changie batch 9999.0.0 --dry-run >/dev/null

# 2. Each fragment must carry an Issue reference like #NNN.
fail=0
for f in "${frags[@]}"; do
  if ! grep -Eq '^[[:space:]]*Issue:[[:space:]]*.*#[0-9]+' "$f"; then
    echo "::error file=$f::changelog fragment is missing an issue/PR reference (expected an 'Issue:' field containing #NNN)"
    fail=1
  fi
done

if [ "$fail" -eq 0 ]; then
  echo "All ${#frags[@]} changelog fragment(s) valid."
fi
exit "$fail"
