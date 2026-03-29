# GLX Project - Claude Development Guide

This document provides context for Claude AI instances working on the GLX project.

---

## Project Overview

**GLX (Genealogix)** is a modern genealogical archive format. It uses YAML for human readability and supports advanced features like evidence-based assertions, comprehensive source citations, and flexible vocabularies.

**Repository**: genealogix/glx
**Primary Language**: Go
**Status**: Active development

---

## Quick Start for New Claude Instances

1. **Read the plans**: Check `.claude/plans/README.md` for current work
2. **Check the branch**: Development happens on feature branches (e.g., `claude/gedcom-import-function-*`)
3. **Review recent commits**: `git log --oneline -10` to see what's been done
4. **Check test status**: `go test ./...` to ensure everything passes
5. **Review todo list**: Active tasks are tracked in `.claude/plans/README.md`

---

## Project Structure

```
spec/
├── .claude/
│   ├── plans/              # Active planning documents
│   └── plans/old/          # Archived historical plans
├── go-glx/                 # Core library (package glx) — importable by external apps
│   │                       # import glxlib "github.com/genealogix/glx/go-glx"
│   ├── types.go           # Core GLX entity types
│   ├── gedcom_*.go        # GEDCOM import implementation
│   ├── serializer.go      # Single/multi-file serialization
│   ├── id_generator.go    # Entity ID generation
│   └── vocabularies.go    # Vocabulary embedding
├── glx/                    # Main CLI application
│   ├── cmd_*.go           # CLI command implementations (import, split, join, validate)
│   └── testdata/
│       └── gedcom/        # GEDCOM test files (180+ files)
├── specification/
│   ├── 5-standard-vocabularies/ # Standard vocabulary definitions
│   └── schema/v1/         # JSON schemas
├── docs/
│   ├── quickstart.md      # User documentation
│   ├── examples/          # Example GLX archives with symlinked vocabularies
│   └── gedcom-spec/       # GEDCOM specification PDFs
└── website/               # GLX website content
```

---

## Core Concepts

### GLX Entity Types

1. **Person** - Individual people
2. **Event** - Life events (birth, death, marriage, etc.)
3. **Relationship** - Connections between people (parent-child, marriage, etc.)
4. **Place** - Geographic locations with hierarchical structure
5. **Source** - Information sources (books, records, websites)
6. **Citation** - Specific citations of sources
7. **Repository** - Where sources are held (archives, libraries)
8. **Media** - Photos, documents, audio, video
9. **Assertion** - Researcher conclusions with evidence chains

### GLX File Formats

1. **Single-file**: All entities in one YAML file
2. **Multi-file**: Entity-per-file in directory structure (IMPLEMENTATION IN PROGRESS)

### Vocabularies

GLX uses controlled vocabularies for:
- Event types (birth, death, marriage, etc.)
- Relationship types (parent-child, spouse, etc.)
- Place types (city, county, state, etc.)
- Source types (book, record, website, etc.)
- And more...

Vocabularies are defined in `.glx` files and will be embedded in the binary using `go:embed`.


## Important Files to Read

### Before Starting Work

3. **`go-glx/types.go`** - Core GLX entity type definitions

### For GEDCOM Work

3. **`go-glx/gedcom_converter.go`** - Main GEDCOM conversion orchestrator

## Development Workflow

### Git Workflow

**Use standard branch naming conventions** — do NOT use `claude/` prefix or session IDs.

```bash
# Branch naming: use conventional prefixes
feat/short-description
fix/short-description
docs/short-description

# Examples:
feat/name-type-implementation
fix/gedcom-note-resolution
docs/update-quickstart

# Always push with -u flag:
git push -u origin <branch-name>

# Retry up to 4 times with exponential backoff if network issues (2s, 4s, 8s, 16s)
```

### Testing

**ALWAYS use the Makefile to run tests.**

```bash
# Run all tests (preferred)
make test

# Run all tests with verbose output
make test-verbose

# NEVER run tests directly with go test commands
# ALWAYS use the Makefile
```

### Building

**ALWAYS use the Makefile to build.**

```bash
# Build CLI (preferred)
make build

# Run CLI
./bin/glx import shakespeare.ged -o output.glx

# Clean build artifacts
make clean
```

---

## Key Design Decisions

### Performance Profiling

**Memory profiling workflow:**
```bash
# Generate allocation profile
go test -bench='BenchmarkName' -benchtime=1x -memprofile=/tmp/prof.out ./go-glx/
# Top allocators by flat bytes (direct allocators)
go tool pprof -top -flat /tmp/prof.out
# Top allocators by cumulative bytes (call tree)
go tool pprof -top -cum /tmp/prof.out
# Filter to package only
go tool pprof -top -flat -focus='glx' /tmp/prof.out
```

**Common allocation pitfalls:**
- Map literals inside function bodies allocate on every call — move to package-level vars
- `LogInfo(fmt.Sprintf(...))` allocates even when logging is disabled — use `LogInfof` instead
- `append`-based backing arrays increase `TotalAlloc` from doubling copies — only use `make([]T, exactSize)` when size is known
- Standard vocabularies are cached via `sync.Once` in `vocabularies.go` — don't add a second load path

### Critical Architectural Rule: go-glx Package Must Never Do I/O

**The `go-glx` library package (package glx) is a pure library and must NEVER perform filesystem I/O.**

This means:
- ❌ NO `os.ReadFile`, `os.WriteFile`, `os.Open`, `os.Create`
- ❌ NO `os.MkdirAll`, `os.Stat`, `os.ReadDir`
- ❌ NO `filepath.Join` with file operations
- ✅ YES to `io.Reader`, `io.Writer`, `[]byte` parameters
- ✅ YES to returning `[]byte` or accepting `[]byte`
- ✅ The `glx/` CLI package handles ALL filesystem operations

**Import path:** `glxlib "github.com/genealogix/glx/go-glx"` (named import needed because of hyphen)

**Correct Pattern:**

```go
// ❌ WRONG - library doing I/O
// go-glx/serializer.go
package glx

func SerializeSingleFile(glx *GLXFile, outputPath string) error {
    yamlBytes, _ := yaml.Marshal(glx)
    return os.WriteFile(outputPath, yamlBytes, 0o644) // NO!
}

// ✅ CORRECT - library returns bytes, CLI does I/O
// go-glx/serializer.go
package glx

func SerializeToBytes(glx *GLXFile) ([]byte, error) {
    return yaml.Marshal(glx)
}

// glx/ CLI package
func saveToFile(glx *glxlib.GLXFile, path string) error {
    data, err := glxlib.SerializeToBytes(glx)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0o644) // CLI does I/O
}
```

**Rationale:**
1. Makes library testable without filesystem
2. Enables library to be used in contexts where I/O isn't appropriate (web servers, embedded systems)
3. Separates concerns: library handles data transformation, CLI handles I/O
4. Prevents architectural violations that couple library code to filesystem

### Architectural Decisions (v0.3.0-beta Serializer)

1. **Vocabulary Embedding**: Use `go:embed` to embed standard vocabularies in binary
2. **Entity IDs**: Use random 8-character hex IDs for filenames (e.g., `person-a3f8d2c1.glx`)
3. **Write Strategy**: Sequential writes (no parallelization for now)
4. **Validation**: Default validate before save with `--no-validate` flag to override

### Go Best Practices

- Return errors, don't panic (except in test helpers with `Must*` prefix)
- Use `any` instead of `interface{}` (Go 1.18+)
- Use `yaml:"field,omitempty"` for optional fields
- Keep functions focused and testable

### Naming Conventions

**NEVER use the variable name `ctx` for anything other than `context.Context`.**

Using `ctx` for other types (like `*ConversionContext` in this codebase) creates extreme confusion since `ctx` is universally understood in Go to mean `context.Context`.

```go
// ❌ INCORRECT - ctx is not context.Context
func convertPerson(personRecord *GEDCOMRecord, ctx *ConversionContext) error {
	ctx.Logger.LogInfo("Converting person")
	// ...
}

// ✅ CORRECT - Use a descriptive name
func convertPerson(personRecord *GEDCOMRecord, convCtx *ConversionContext) error {
	convCtx.Logger.LogInfo("Converting person")
	// ...
}

// ✅ ALSO CORRECT - Even better, be explicit
func convertPerson(personRecord *GEDCOMRecord, conversion *ConversionContext) error {
	conversion.Logger.LogInfo("Converting person")
	// ...
}
```

**Rationale**: The Go community has strong conventions around `ctx` always meaning `context.Context`. Breaking this convention makes code harder to read and understand, especially when both `context.Context` and other context-like types appear in the same codebase.

### Updating This Document

**When given broad instructions like "Never do X" or "Always do Y", update CLAUDE.md to document the guideline.**

This ensures:
1. The guideline is preserved for future Claude instances
2. Patterns stay consistent across the codebase
3. New developers understand project conventions

Examples of instructions that should be documented:
- "Never use X pattern"
- "Always do Y when Z"
- "Prefer A over B"
- "Don't use _ parameters except for interfaces"
- "Never name variables ctx unless they're context.Context"

### Documenting Pre-Existing Issues

**When a pre-existing bug or issue is discovered during implementation, ALWAYS document it.**

If you discover a bug or architectural issue that exists in the codebase but is outside the scope of your current task:
1. **Add it to `todo.md`** under the appropriate category with priority marker
2. **Mention it in your summary** to the user so they're aware

This ensures issues don't get lost and can be prioritized appropriately.

### Cobra Command Handler Pattern

**Functions with `_` parameters must be thin wrappers with no logic.**

When implementing cobra.Command handlers that have unused parameters (like `cmd *cobra.Command`), follow this pattern:

```go
// ✅ CORRECT - Thin wrapper with no logic
func runValidate(_ *cobra.Command, args []string) error {
	return validatePaths(args)
}

func validatePaths(args []string) error {
	// All logic goes here
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}
	// ... rest of implementation
}

// ❌ INCORRECT - Logic in function with _ parameter
func runValidate(_ *cobra.Command, args []string) error {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}
	// ... this violates the pattern
}
```

**Rationale**: Functions with `_` parameters indicate unused parameters required by interfaces. Keeping them as thin wrappers makes it obvious that the parameter is truly unused and keeps all logic in testable, interface-free functions.

### Unused Parameters - General Rule

**AVOID `_` parameters in regular functions. Just remove the parameter from the signature.**

The `_` pattern is ONLY acceptable when required by an interface (like cobra.Command handlers). For regular functions:

```go
// ❌ INCORRECT - Unnecessary _ parameter
func validateNestedStructs(entityType, entityID, _ string, fieldVal reflect.Value, result *ValidationResult) {
	// fieldName is not used
}

// ✅ CORRECT - Remove the unused parameter entirely
func validateNestedStructs(entityType, entityID string, fieldVal reflect.Value, result *ValidationResult) {
	// Much cleaner!
}
```

**When to use `_`**:
- ✅ Required by interface (e.g., `func runValidate(_ *cobra.Command, args []string)`)
- ❌ Regular function with no interface constraint

**Why**: If there's no interface forcing the signature, there's no reason to keep unused parameters. Just remove them and update the call sites. It's clearer and more maintainable.

---

## Common Tasks

### Add a New Entity Type

1. Define type in `go-glx/types.go`
2. Add to `GLXFile` struct
3. Update serializer to handle new type
4. Add vocabulary if needed
5. Update documentation

### Add GEDCOM Tag Support

1. Find appropriate converter file (e.g., `go-glx/gedcom_individual.go`)
2. Add tag handling in `switch` statement
3. Extract data and map to GLX entity
4. Add test case in `go-glx/gedcom_test.go`
5. Update gap analysis if fixing a gap

### Add a New CLI Command

When adding a new CLI command, update all four documentation locations:
1. `glx/README.md` — Features list, Quick Start examples, and full command reference section
2. `website/.vitepress/config.js` — Add to appropriate sidebar menu group
3. `docs/guides/hands-on-cli-guide.md` — Add walkthrough section with Westeros examples
4. `CHANGELOG.md` — Add entry to the current unreleased beta section

### Debug GEDCOM Import

1. Enable verbose logging: `ctx.Logger.LogInfo(...)`
2. Check `ConversionContext` for entity maps
3. Run specific test: `make test` (always use Makefile)
4. Check error accumulation in `ctx.Errors`

---

## Testing Strategy

### Test Files

- **testdata/gedcom/minimal-70.ged** - GEDCOM 7.0 minimal test
- **testdata/gedcom/shakespeare.ged** - GEDCOM 5.5.1 comprehensive test (31 persons, 77 events)

### Test Coverage Requirements

- Unit tests for all new functions
- Integration tests for full conversion paths
- E2E tests for CLI commands

---

## Documentation Standards

### Planning Documents

- Store in `.claude/plans/`
- Include date, status, and purpose at top
- Use markdown with clear sections
- Update `.claude/plans/README.md` when adding new plans
- **DO NOT include time estimates** - They waste tokens and are meaningless
- Focus on task breakdown and dependencies, not hours/days estimates

### Code Comments

- Document public functions and types with Go doc comments
- Explain complex algorithms inline
- Reference GEDCOM spec for GEDCOM-specific code

### Specification Documents

**Internal Links**: Omit `.md` file extension for VitePress compatibility
- ✓ Good: `[Person Entity](4-entity-types/person)`
- ✗ Bad: `[Person Entity](4-entity-types/person.md)`

### Commit Messages and PR Titles

- Use conventional commits format: `type: Subject starting with uppercase` (see `.github/workflows/lint-pr-title.yml` for valid types: feat, fix, docs, chore, refactor, test, perf, ci)
- Keep messages brief - prefer single-line messages when possible
- Do NOT include "Generated with Claude Code", Co-Authored-By footers, or any AI attribution in commits, PRs, or any other output
- Examples:
  - `feat: Add GEDCOM 7.0 EXID support`
  - `fix: Handle family events ANUL, DIVF, CENS, EVEN`
  - `docs: Update serializer implementation plan`

### Pull Requests

- **Always read and follow `.github/PULL_REQUEST_TEMPLATE.md`** when creating PRs

### Changelog

- Always update `CHANGELOG.md` when making user-facing changes
- Add entries to the **latest unreleased version section** at the top. Verify which section is unreleased by checking `git tag --sort=-v:refname | head -1` — the latest tag's version has already been released, so add entries to the section above it
- Use appropriate subsections: Added, Changed, Fixed, Removed
- Group related changes under descriptive headers (e.g., `#### Citation Entity`)
- **Feature branch changelog hygiene**: Feature branches often have stale/mangled changelogs. To fix: `git checkout main -- CHANGELOG.md`, then add the branch's entries to the current unreleased section. Never try to merge a diverged changelog — restore and re-add.

### What NOT to Do

- **Don't produce time estimates** in plans or responses
- **Don't estimate hours/days** for tasks - focus on what needs to be done
- **Don't create speculative timelines** - they're meaningless and waste tokens
- Let the user decide scheduling and priorities

---

## Known Issues and Gotchas

### GEDCOM Import

- GEDCOM 5.5.1 uses `@REF@` for references, GEDCOM 7.0 uses `@VOID@`
- Shared notes work differently between versions (NOTE vs SNOTE)
- Some GEDCOM features have no GLX equivalent (use Properties field)

### Serialization

- Entity IDs are random, so filenames are not deterministic
- Must embed `_id` field in YAML to preserve entity IDs in multi-file format
- Vocabularies must be written to multi-file archives

### Feature Branch Merges

- `glx/cli_commands.go` and `CHANGELOG.md` conflict frequently when merging main into feature branches
- For `cli_commands.go`: keep both commands — add the new command's `rootCmd.AddCommand()` call and its full command block
- For worktree-based branch work: use `/tmp/glx-<name>` as worktree path, build with `go build -o bin/glx ./glx` (Makefile requires vitepress)

### Testing

- Shakespeare test file has complex family relationships - good stress test
- Some GEDCOM files have invalid data - parser should handle gracefully

---

## Resources

### GLX Specification

- See `docs/` directory for user-facing documentation
- Spec is evolving - check recent commits for latest changes

### GEDCOM Specification

**IMPORTANT**: Do NOT read full PDF specifications - they're too large and will cause errors.

**Use split PDFs instead**:
- GEDCOM 5.5.1: `docs/gedcom-spec/GEDCOM_5.5.1_Specification/part_*.pdf` (6 parts, ~20 pages each)
- GEDCOM 7.0: `docs/gedcom-spec/GEDCOM_7.0_Specification/part_*.pdf` (6 parts, ~20 pages each)

Also available:
- GEDCOM 3.0, 4.0, 5.0, 5.3, 5.4, 5.5, 5.5.5, 5.6 (all split)
- GEDZIP 0.1, 0.2 (all split)

**Usage**: Use the Read tool on specific part files based on page numbers you need

### Go Resources

- YAML library: gopkg.in/yaml.v3
- Standard library: crypto/rand, embed, filepath, os

---

## Contact and Collaboration

**Repository**: genealogix/glx
**Branch Pattern**: `feat/short-description`, `fix/short-description`, `docs/short-description`
**Workflow**: Feature branches → Push → (PR created manually by user)

---

Last Updated: 2026-02-11
