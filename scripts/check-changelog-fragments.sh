#!/usr/bin/env bash
# Validate changie changelog fragments (#858, implements #321):
#   1. every fragment parses and renders (catches malformed YAML / unknown kind)
#   2. every fragment carries an issue/PR reference in its Issue field
# Run via `make changelog-check` or the changelog-fragments CI workflow.
set -euo pipefail

FRAG_DIR=".changes/unreleased"

# `make changelog-check` exports $CHANGIE as an absolute path (see the
# ensure_changie comment in the Makefile for why it is not put on PATH); a bare
# run, and CI, fall back to a PATH lookup.
CHANGIE="${CHANGIE:-changie}"

if ! command -v "$CHANGIE" >/dev/null 2>&1; then
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
"$CHANGIE" batch 9999.0.0 --dry-run >/dev/null

# 2. Each fragment must carry a `#NNN` reference in its top-level `custom.Issue`
#    field — which is what `changeFormat` renders. Delegated to
#    tools/fragmentcheck, which reads the fragment with a YAML parser: an
#    `Issue:`-looking line inside the `body: |-` block scalar is body text and
#    cannot satisfy the gate, while every spelling YAML allows for the real
#    field (block map, inline flow map, quoted or not) is accepted. Text
#    pattern-matching got the first property right but rejected the second.
exec go run ./tools/fragmentcheck "$FRAG_DIR"
