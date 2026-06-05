#!/usr/bin/env bash
# Fail-closed wrapper around gh-api-gate.py for the PreToolUse hook (genealogix/glx#912).
#
# Locates a Python 3 interpreter and runs the gate, passing the hook's stdin
# through. If no compatible Python is available (or the gate script is missing),
# it emits an "ask" decision so a gh api call is never silently auto-approved on
# a machine that cannot evaluate the gate — degrading gracefully to the pre-#911
# behaviour (every gh api call prompts) rather than opening a hole.
#
# The interpreter is version-checked before exec: a bare `python` that is really
# Python 2 would fail to run this Python 3 script, and that exec failure would
# surface as a non-zero exit (tripping the settings.json `|| exit 2` hard block)
# instead of the intended fail-closed "ask". So we only accept Python >= 3.
set -u

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GATE="$DIR/gh-api-gate.py"

PY=""
for cand in python3 python; do
  p="$(command -v "$cand" 2>/dev/null)" || continue
  if [ -n "$p" ] && "$p" -c 'import sys; raise SystemExit(0 if sys.version_info[0] >= 3 else 1)' 2>/dev/null; then
    PY="$p"
    break
  fi
done

if [ -n "$PY" ] && [ -f "$GATE" ]; then
  exec "$PY" "$GATE"
fi

printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"ask","permissionDecisionReason":"gh-api gate unavailable (no Python 3 on PATH or gate script missing); gating gh api for manual review."}}'
