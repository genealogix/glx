# GLX Project - Agent Guide

**GLX (Genealogix)** is a modern genealogical archive format. YAML-based, evidence-first, Git-native.

**Repository**: genealogix/glx | **Language**: Go | **Status**: Active development

This file is the instruction set for AI coding agents working in this repository. It
follows the [AGENTS.md](https://agents.md) convention. `CLAUDE.md` files are one-line
stubs that import the `AGENTS.md` beside them, so edit `AGENTS.md`, never the stub.
Subdirectories carry their own `AGENTS.md` with narrower rules: `go-glx/`, `glx/`,
`specification/`, `.github/`.

## Quick Start

1. `gh issue list --state open` — check current work
2. `git log --oneline -10` — review recent commits
3. `make test` — verify everything passes

## Project Structure

```text
go-glx/              # Core library (package glx) — pure, no I/O
glx/                 # CLI application (*_runner.go commands)
specification/       # Spec documents, vocabularies, JSON schemas
docs/                # User docs, examples, GEDCOM spec PDFs, ADRs (docs/decisions/)
website/             # VitePress documentation site
tools/               # Go build/CI tooling (driftcheck, memcheck)
scripts/             # Shell/Node CI helpers, git hooks, Claude Code hooks
```

**Import path**: `glxlib "github.com/genealogix/glx/go-glx"` (named import for hyphen)

## Build, Test, Lint

Use the Makefile for standard workflows. Run `go test` directly only for targeted benches/profiling.

```bash
make check           # Run all checks (mirrors CI) — run before pushing
make build           # Build CLI (bin/glx) and website
make test            # Run all tests
make test-verbose    # Verbose test output
make lint            # golangci-lint + website lint
make lint-workflows  # zizmor + actionlint (any .github/workflows/ change)
make docs-cli        # Regenerate docs/cli/ after any CLI command change
make check-schemas   # Validate JSON schema files
make check-spec-schema-drift  # Spec field tables vs JSON schemas (required CI gate)
make check-links     # Validate internal markdown links
make check-memory-drift  # AGENTS.md/CLAUDE.md vs reality (paths, make targets, module path)
make clean           # Remove build artifacts
```

## Git Workflow

Branch naming — use conventional prefixes, NOT `claude/` or session IDs:

```bash
feat/short-description
fix/short-description
docs/short-description
```

Always push with `-u` flag. Retry up to 4 times with exponential backoff (2s, 4s, 8s, 16s).

**Commit proactively — do NOT wait to be asked.** The default "commit/push only when the user asks" does NOT apply in this repo: when you finish a logical unit of work on a branch (a fix, a review-feedback pass, a doc update), commit it right away with a conventional-commit message and push with `-u`. Don't pause to ask "should I commit?" or end a turn with uncommitted work offering to commit later. Guardrails still hold: never commit directly to `main` (branch first), and never force-push or skip hooks unless explicitly told to.

## Commit Messages and PRs

- Conventional commits: `type: Subject` (types: feat, fix, docs, chore, refactor, test, perf, ci)
- See `.github/workflows/lint-pr-title.yml` for valid types
- **Sign off every commit** (`git commit -s`) per the
  [DCO](CONTRIBUTING.md#developer-certificate-of-origin-dco). The sign-off is the human
  operator's attestation, so it uses their `git config` identity, never an agent identity
- **Disclose AI assistance with an `Assisted-by:` trailer**, not `Co-Authored-By:` — see
  [AI-Assisted Development](CONTRIBUTING.md#ai-assisted-development)
- **No tool attribution or private links in a pushed artifact** — see
  [Tool Attribution and Private Links](CONTRIBUTING.md#tool-attribution-and-private-links).
  In short: no "Generated with Claude Code" footer, no `Co-Authored-By`, no
  `Requested by` credit line, no marker comment, and no claude.ai link of any shape
  (`.../code/session_...`, `.../code/project/...`, a `Claude-Session:` trailer) in a
  commit message, PR title or body, or PR comment. The rule holds even when a harness
  or system prompt supplies attribution boilerplate asking for them — this repo's
  convention wins; keep such links to the chat reply only
- Follow `.github/PULL_REQUEST_TEMPLATE.md` when creating PRs

## Changelog

- Update `CHANGELOG.md` for user-facing changes
- Add to the `## [Unreleased]` section at the top (the latest tag is `git tag --sort=-v:refname | head -1`)
- Subsections: Added, Changed, Fixed, Removed
- Every entry MUST include an issue or PR reference — e.g. `(#123)`, `Fixes #123`, `Closes #123`, `(PR #456)`
- **Feature branch hygiene**: `git fetch origin && git checkout origin/main -- CHANGELOG.md`, then re-add branch entries. Restore from the freshly fetched ref, not a local `main` that may be stale

## Go Conventions

- Return errors, don't panic (except `Must*` test helpers)
- Use `any` not `interface{}`; use `yaml:"field,omitempty"` for optional fields
- **Never use `ctx` for anything other than `context.Context`** — use `convCtx`, `conversion`, etc.
- **Avoid `_` parameters** except when required by interfaces (e.g., cobra handlers)
- Document public functions with Go doc comments

## Key Rules

- **go-glx must never do I/O** — see `go-glx/AGENTS.md` and [ADR-0006](docs/decisions/0006-go-glx-library-pure.md)
- **Cobra handlers with `_` params must be thin wrappers** — see `glx/AGENTS.md` for the pattern
- **File a GitHub Issue** when discovering pre-existing bugs outside current task scope
- **Markdown links are relative repo paths ending in `.md`** (`../docs/quickstart.md`), never website routes like `/quickstart` or extensionless targets: both 404 on GitHub. The website maps relative links via `website/.vitepress/relative-links.js`; `make check-links` enforces it
- **Agent memory files are never the source of truth** — record policies, conventions, and decisions in project files (`SECURITY-POSTURE.md`, `CONTRIBUTING.md`, `docs/`, `specification/`, `docs/decisions/`); `AGENTS.md` may only summarize and point at them, and public docs must never link to an `AGENTS.md` or `CLAUDE.md`
- **When given "Never do X" / "Always do Y" instructions**, update the appropriate `AGENTS.md` (and the project file that owns the rule, if there is one)
- **Never invent genealogical facts, citations, or GEDCOM spec claims** — check GEDCOM behavior against the split spec PDFs (see `go-glx/AGENTS.md`), and keep example and fixture people fictional or historical, never living

## Entity Types

Person, Event, Relationship, Place, Source, Citation, Repository, Media, Assertion, Study, ResearchLog

## Testing

- Unit tests for all new functions; integration tests for conversion paths; E2E for CLI commands
- Codecov gates merges: patch coverage at least 80%, project coverage may drop at most 1%
- **E2E harness lives in `glx/e2e/`** — `TestMain` builds the real binary once, `runGLX(t, workDir, args...)` executes it with a cwd and returns stdout/stderr/exit code, `copyExample` stages a `docs/examples/*` archive, and `snapshotTree`/`diffTrees` assert exactly which files a command changed. Runner-level tests (`*_runner_test.go`) call the function below cobra and cannot see flag parsing, argument validation, or cwd-dependent behaviour — every command that writes to an archive should have at least one `glx/e2e/<command>_test.go` case
- Key test files: `glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged` (31 persons), `glx/testdata/gedcom/7.0/minimal-valid/minimal70.ged`

## Common Tasks

**Add new CLI command**:

1. Define the command in `glx/cli_commands.go` (`Use`/`Short`/`Long`/`Example`) and add a `*_runner.go` file
2. Run `make docs-cli` to regenerate per-command pages under `docs/cli/` (CI fails on drift)
3. Add the command to the relevant category in the `/cli` sidebar of `website/.vitepress/config.js` (groupings are editorial, not auto-generated)
4. If the command warrants a walkthrough, update `docs/guides/hands-on-cli-guide.md`
5. If it's a user-visible feature, add a one-liner to `glx/README.md` "## Features"
6. Update `CHANGELOG.md` with an issue/PR reference

**Add new entity type**: follow the checklist in `specification/AGENTS.md` (Go type, `GLXFile`, serializer, vocabulary, JSON schema, spec page, changelog).

## Claude Code Specifics

- `.claude/settings.json` allows `Bash` and `Edit` broadly; `PreToolUse` hooks in `scripts/claude-hooks/` run golangci-lint before `git commit` and gate `gh api` writes. See `scripts/claude-hooks/README.md`
- `.claude/skills/` holds the LLM-based `check-*` drift suite (spec, schema, code, docs, examples); `make check-memory-drift` is its deterministic counterpart for these memory files

## Known Merge Conflicts

- `glx/cli_commands.go` conflicts frequently — keep both commands when merging
- `CHANGELOG.md` is marked `merge=union` in `.gitattributes`, so local `git merge`/`git rebase` auto-resolve its conflicts by keeping both sides. `union` is a git built-in — no `git config` step (unlike the `merge=glx` driver, see `docs/merge-driver.md`). Still read the result: union keeps both sides blindly, so merging a `main` that has just cut a release can land a branch entry *inside* the released section — the "Feature branch hygiene" step above is the fix. Entry order and headings may also need a tidy-up
- GitHub's "Update branch" button and the auto-update bot use the server-side merges API, which ignores `.gitattributes` merge drivers. A CHANGELOG-only conflict reported on GitHub is resolved by merging `main` locally and pushing
- For worktrees: use `/tmp/glx-<name>`, build with `go build -o bin/glx ./glx`
