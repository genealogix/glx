# GLX Project - Claude Development Guide

This document provides context for Claude AI instances working on the GLX project.

---

## Project Overview

**GLX (Genealogical Ledger eXchange)** is a modern genealogical archive format designed to replace GEDCOM. It uses YAML for human readability and supports advanced features like evidence-based assertions, comprehensive source citations, and flexible vocabularies.

**Repository**: genealogix/spec
**Primary Language**: Go
**Current Version**: v0.0.0-beta.2 (in progress)
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
├── glx/                    # Main CLI application
│   ├── cmd_*.go           # CLI command implementations (import, split, join, validate)
│   ├── lib/               # Core library code
│   │   ├── types.go       # Core GLX entity types
│   │   ├── gedcom_*.go    # GEDCOM import implementation
│   │   ├── serializer.go  # Single/multi-file serialization
│   │   ├── id_generator.go # Entity ID generation
│   │   └── vocabularies.go # Vocabulary embedding
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

---

## Current Development Status

### ✅ Completed Features (v0.2.0-beta)

1. **GEDCOM Import** - PRODUCTION-READY
   - Full GEDCOM 5.5.1 support
   - Full GEDCOM 7.0 support
   - Two-pass conversion (entities, then families)
   - Evidence chain mapping (SOUR → Citations → Assertions)
   - Place hierarchy building
   - 31 test persons, 77 events, 49 relationships imported successfully
   - Gap analysis complete: 100% critical, 94% high-priority coverage

2. **Schema Enhancements**
   - Added Properties fields to Source, Citation, Repository, Media, Assertion
   - Place coordinate support (MAP/LATI/LONG)
   - External ID support (GEDCOM 7.0 EXID)
   - Shared NOTE support (GEDCOM 5.5.1 and 7.0)

### 🚧 In Progress (v0.3.0-beta)

**GLX Serializer** - Currently implementing
- **Phase 1**: Core infrastructure (ID generator, vocabulary embedder, serializer interface)
- **Phase 2**: Single-file and multi-file serialization
- **Phase 3**: Archive loading
- **Phase 4**: Testing
- **Phase 5**: CLI integration (split, join commands)

**See**: `.claude/plans/glx-serializer-implementation-steps.md` for detailed task list

### 📋 Planned Features

- Property vocabularies for GEDCOM-specific fields
- Validation framework with vocabulary checking
- Export to GEDCOM
- Web viewer/editor
- Advanced querying

---

## Important Files to Read

### Before Starting Work

1. **`.claude/plans/README.md`** - Overview of all planning documents and current status
2. **`.claude/plans/glx-serializer-implementation-steps.md`** - Current implementation guide
3. **`lib/types.go`** - Core GLX entity type definitions

### For GEDCOM Work

1. **`.claude/plans/gedcom-import-complete-plan.md`** - Complete GEDCOM import plan (9,065 lines)
2. **`.claude/plans/gedcom-import-gap-analysis.md`** - What's implemented vs. what's planned
3. **`lib/gedcom_converter.go`** - Main GEDCOM conversion orchestrator

### For Serializer Work

1. **`.claude/plans/glx-serializer-plan.md`** - Architecture and design
2. **`.claude/plans/glx-vocabulary-embedding.md`** - Vocabulary embedding strategy
3. **`.claude/plans/glx-id-generation.md`** - Entity ID generation strategy

---

## Development Workflow

### Git Workflow

```bash
# Feature branches follow this pattern:
claude/feature-name-{sessionId}

# Example:
claude/gedcom-import-function-01EsRP3qVbnSKcufKho9gVhK

# Always push with -u flag:
git push -u origin <branch-name>

# Retry up to 4 times with exponential backoff if network issues (2s, 4s, 8s, 16s)
```

### Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test -v -run TestImportShakespeare ./lib

# Run with coverage
go test -cover ./...
```

### Building

```bash
# Build CLI
go build -o glx ./cmd/glx

# Run CLI
./glx import shakespeare.ged -o output.glx
```

---

## Key Design Decisions

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

---

## Common Tasks

### Add a New Entity Type

1. Define type in `lib/types.go`
2. Add to `GLXFile` struct
3. Update serializer to handle new type
4. Add vocabulary if needed
5. Update documentation

### Add GEDCOM Tag Support

1. Find appropriate converter file (e.g., `lib/gedcom_individual.go`)
2. Add tag handling in `switch` statement
3. Extract data and map to GLX entity
4. Add test case in `lib/gedcom_test.go`
5. Update gap analysis if fixing a gap

### Debug GEDCOM Import

1. Enable verbose logging: `ctx.Logger.LogInfo(...)`
2. Check `ConversionContext` for entity maps
3. Run specific test: `go test -v -run TestImportShakespeare ./lib`
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

### Commit Messages

- Use conventional commits format
- Examples:
  - `feat: Add GEDCOM 7.0 EXID support`
  - `fix: Handle family events ANUL, DIVF, CENS, EVEN`
  - `docs: Update serializer implementation plan`

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

### Testing

- Shakespeare test file has complex family relationships - good stress test
- Some GEDCOM files have invalid data - parser should handle gracefully

---

## Resources

### GLX Specification

- See `docs/` directory for user-facing documentation
- Spec is evolving - check recent commits for latest changes

### GEDCOM Specification

- GEDCOM 5.5.1: `docs/gedcom-spec/gedcom-5-5-1.pdf`
- GEDCOM 7.0: https://gedcom.io/specifications/

### Go Resources

- YAML library: gopkg.in/yaml.v3
- Standard library: crypto/rand, embed, filepath, os

---

## Contact and Collaboration

**Repository**: genealogix/spec
**Branch Pattern**: `claude/feature-name-{sessionId}`
**Workflow**: Feature branches → Push → (PR created manually by user)

---

## Quick Reference: Current Sprint

**Goal**: Implement GLX serializer (v0.3.0-beta)

**Current Task**: Task 1.1 - Create ID generator (lib/id_generator.go)

**Next Tasks**:
1. Create vocabulary embedder (lib/vocabularies.go)
2. Create serializer interface and types
3. Implement single-file YAML serializer
4. Implement multi-file serializer

**See**: `.claude/plans/glx-serializer-implementation-steps.md` for full task breakdown

---

Last Updated: 2025-11-18
