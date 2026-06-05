#!/usr/bin/env bash
#
# pre-commit-golangci.sh — advisory golangci-lint gate for Claude Code sessions.
#
# Wired from .claude/settings.json as a PreToolUse hook that fires before the
# Bash tool runs a `git commit` (matcher "Bash", handler `if: "Bash(git commit:*)"`).
# When the commit stages Go source, it runs golangci-lint scoped to the new code
# and surfaces any findings before the commit is created. See
# scripts/claude-hooks/README.md and genealogix/glx#869.
#
# Advisory only: findings are reported via golangci-lint's own exit code, which
# Claude Code treats as a NON-blocking error for PreToolUse (only exit 2 blocks a
# tool call). Whether to promote this to a hard, blocking gate — and how it
# relates to the lefthook/CI lint passes — is the strategy decision tracked in
# genealogix/glx#870; this script deliberately stays advisory.
#
# Runnable standalone from the repo root:
#   bash scripts/claude-hooks/pre-commit-golangci.sh
set -euo pipefail

# Run from the project root so golangci-lint resolves .golangci.yml and the module.
cd "${CLAUDE_PROJECT_DIR:-.}"

# Nothing to lint unless this commit stages Go source.
staged_go=$(git diff --cached --name-only --diff-filter=ACMR -- '*.go')
if [ -z "$staged_go" ]; then
  exit 0
fi

# Linting walks the working tree, not the index, so unstaged edits can produce
# findings for code that is not part of this commit. Warn rather than fail.
if [ -n "$(git diff --name-only -- '*.go')" ]; then
  echo "Warning: unstaged Go changes detected — lint may flag code not in this commit" >&2
fi

# Scope to changes new since the branch point; fall back to the previous commit
# when main is not reachable (e.g. a shallow checkout or a brand-new repo).
base=$(git merge-base HEAD main 2>/dev/null || echo 'HEAD~1')

# Send findings to stderr so Claude Code surfaces them; the exit status reflects
# the lint result (non-blocking for PreToolUse, per the note above).
golangci-lint run --new-from-rev="$base" --timeout=5m >&2
