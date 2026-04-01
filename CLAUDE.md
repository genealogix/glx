# GLX Project - Claude Development Guide

**GLX (Genealogix)** is a modern genealogical archive format. YAML-based, evidence-first, Git-native.

**Repository**: genealogix/glx | **Language**: Go | **Status**: Active development

## Quick Start

1. `gh issue list --state open` — check current work
2. `git log --oneline -10` — review recent commits
3. `make test` — verify everything passes

## Project Structure

```
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

## Commit Messages and PRs

- Conventional commits: `type: Subject` (types: feat, fix, docs, chore, refactor, test, perf, ci)
- See `.github/workflows/lint-pr-title.yml` for valid types
- Do NOT include AI attribution (no "Generated with Claude Code", no Co-Authored-By)
- Follow `.github/PULL_REQUEST_TEMPLATE.md` when creating PRs

## Changelog

- Update `CHANGELOG.md` for user-facing changes
- Add to the **latest unreleased section** (check with `git tag --sort=-v:refname | head -1`)
- Subsections: Added, Changed, Fixed, Removed
- **Feature branch hygiene**: `git checkout main -- CHANGELOG.md`, then re-add branch entries

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
- **When given "Never do X" / "Always do Y" instructions**, update the appropriate CLAUDE.md

## Entity Types

Person, Event, Relationship, Place, Source, Citation, Repository, Media, Assertion

## Testing

- Unit tests for all new functions; integration tests for conversion paths; E2E for CLI commands
- Key test files: `testdata/gedcom/shakespeare.ged` (31 persons), `testdata/gedcom/minimal-70.ged`

## Common Tasks

**Add new CLI command** — update all four locations:
1. `glx/README.md` — features list and command reference
2. `website/.vitepress/config.js` — sidebar menu
3. `docs/guides/hands-on-cli-guide.md` — walkthrough with Westeros examples
4. `CHANGELOG.md`

**Add new entity type**: define in `go-glx/types.go` → add to `GLXFile` → update serializer → add vocabulary → update docs

## Known Merge Conflicts

- `glx/cli_commands.go` and `CHANGELOG.md` conflict frequently — keep both commands when merging
- For worktrees: use `/tmp/glx-<name>`, build with `go build -o bin/glx ./glx`

Last Updated: 2026-03-31
