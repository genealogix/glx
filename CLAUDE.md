# GLX Project - Claude Development Guide

**GLX (Genealogix)** is a modern genealogical archive format. YAML-based, evidence-first, Git-native.

**Repository**: genealogix/glx | **Language**: Go | **Status**: Active development

## Quick Start

1. `gh issue list --state open` — check current work
2. `git log --oneline -10` — review recent commits
3. `make test` — verify everything passes

## Project Structure

```text
go-glx/              # Core library (package glx) — pure, no I/O
glx/                 # CLI application (*_runner.go commands)
specification/       # Spec documents, vocabularies, JSON schemas
docs/                # User docs, examples, GEDCOM spec PDFs
website/             # VitePress documentation site
```

**Import path**: `glxlib "github.com/genealogix/glx/go-glx"` (named import for hyphen)

## Build, Test, Lint

Use the Makefile for standard workflows. Run `go test` directly only for targeted benches/profiling.

```bash
make build           # Build CLI (bin/glx) and website
make test            # Run all tests
make test-verbose    # Verbose test output
make lint            # golangci-lint + website lint
make check-schemas   # Validate JSON schema files
make check-links     # Validate internal markdown links
make check-memory-drift  # CLAUDE.md/AGENTS.md vs reality (paths, make targets, module path)
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

Changes are tracked as **changie fragments** (`.changes/unreleased/*.yaml`), one file per change — never by hand-editing `CHANGELOG.md` (#858). `CHANGELOG.md` is regenerated at release time and must not be edited directly; doing so reintroduces the merge-conflict class file-per-change was adopted to remove.

- For each user-facing change, add a fragment: `make changelog` (or `changie new`). Kinds: Added, Changed, Deprecated, Removed, Fixed, Security (Keep a Changelog order).
- Every fragment MUST carry an issue/PR reference in its required `Issue` field — e.g. `#123` or `PR #456`. `changie new` refuses to create one without it.
  - Single ref: `changie new -k Added -b "Subject — detail." -m Issue="#123" --interactive=false`
  - Multi-ref (commas) must use the env var, since `-m` comma-splits: `CHANGIE_CUSTOM_Issue="#41, #775" changie new -k Added -b "..." --interactive=false`
- **Before adding a fragment, grep `.changes/unreleased/` for a sibling fragment about the same feature and edit that one instead of creating a duplicate.** changie does not consolidate `add X` + `fix X` + `refine X` — it emits three bullets. Folding follow-ups into the original fragment is the contributor's job (the release-time `/compact-changelog` pass is the backstop, not a substitute).
- No "feature branch hygiene" dance is needed anymore: fragments have unique filenames, so branches never collide on `CHANGELOG.md`.

## Go Conventions

- Return errors, don't panic (except `Must*` test helpers)
- Use `any` not `interface{}`; use `yaml:"field,omitempty"` for optional fields
- **Never use `ctx` for anything other than `context.Context`** — use `convCtx`, `conversion`, etc.
- **Avoid `_` parameters** except when required by interfaces (e.g., cobra handlers)
- Document public functions with Go doc comments

## Key Rules

- **go-glx must never do I/O** — see `go-glx/CLAUDE.md` for details
- **Cobra handlers with `_` params must be thin wrappers** — see `glx/CLAUDE.md` for the pattern
- **File a GitHub Issue** when discovering pre-existing bugs outside current task scope
- **Markdown links are relative repo paths ending in `.md`** (`../docs/quickstart.md`), never website routes like `/quickstart` or extensionless targets: both 404 on GitHub. The website maps relative links via `website/.vitepress/relative-links.js`; `make check-links` enforces it
- **CLAUDE.md files are never the source of truth** — record policies, conventions, and decisions in project files (`SECURITY-POSTURE.md`, `CONTRIBUTING.md`, `docs/`, `specification/`, `docs/decisions/`); CLAUDE.md may only summarize and point at them, and public docs must never link to a CLAUDE.md
- **When given "Never do X" / "Always do Y" instructions**, update the appropriate CLAUDE.md

## Entity Types

Person, Event, Relationship, Place, Source, Citation, Repository, Media, Assertion, Study, ResearchLog

## Testing

- Unit tests for all new functions; integration tests for conversion paths; E2E for CLI commands
- **E2E harness lives in `glx/e2e/`** — `TestMain` builds the real binary once, `runGLX(t, workDir, args...)` executes it with a cwd and returns stdout/stderr/exit code, `copyExample` stages a `docs/examples/*` archive, and `snapshotTree`/`diffTrees` assert exactly which files a command changed. Runner-level tests (`*_runner_test.go`) call the function below cobra and cannot see flag parsing, argument validation, or cwd-dependent behaviour — every command that writes to an archive should have at least one `glx/e2e/<command>_test.go` case
- Key test files: `glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged` (31 persons), `glx/testdata/gedcom/7.0/minimal-valid/minimal70.ged`

## Common Tasks

**Add new CLI command**:

1. Define the command in `glx/cli_commands.go` (`Use`/`Short`/`Long`/`Example`) and add a `*_runner.go` file
2. Run `make docs-cli` to regenerate per-command pages under `docs/cli/` (CI fails on drift)
3. Add the command to the relevant category in the `/cli` sidebar of `website/.vitepress/config.js` (groupings are editorial, not auto-generated)
4. If the command warrants a walkthrough, update `docs/guides/hands-on-cli-guide.md`
5. If it's a user-visible feature, add a one-liner to `glx/README.md` "## Features"
6. Add a changie fragment for the change (`make changelog`) with an issue/PR reference

**Add new entity type**: define in `go-glx/types.go` → add to `GLXFile` → update serializer → add vocabulary → update docs

## Known Merge Conflicts

- `glx/cli_commands.go` conflicts frequently — keep both commands when merging
- `CHANGELOG.md` no longer conflicts: changes are file-per-change changie fragments (#858), and `CHANGELOG.md` itself is regenerated, never hand-edited on a branch
- For worktrees: use `/tmp/glx-<name>`, build with `go build -o bin/glx ./glx`

Last Updated: 2026-06-05
