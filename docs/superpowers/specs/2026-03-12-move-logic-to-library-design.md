# Move Core Logic from CLI to go-glx Library

**Date**: 2026-03-12
**Status**: Draft
**Motivation**: Enable a commercial app to consume go-glx as a library for schema validation, archive validation, and archive loading without depending on the CLI.

---

## Scope

Move three areas of logic from the CLI (`glx/`) to the library (`go-glx/`):

1. Schema/structural validation
2. Validation orchestration (structural + cross-reference pipeline)
3. Archive loading (streaming reader-based API)

**Out of scope**: Analysis (gaps, evidence, consistency, suggestions), querying/filtering, tree traversal, statistics, output formatting. These stay in the CLI.

**Constraints**:
- Existing library API remains unchanged (additive only)
- `go-glx` must never do filesystem I/O (no `os.*`, no `fs.FS`)
- Accept `io.Reader`, `[]byte`, `map[string]any` — never file paths

**New dependency note**: Moving schema validation to the library means `go-glx` will import `specification/schema/v1` (same Go module, already used by CLI) and `github.com/xeipuuv/gojsonschema`. Library consumers will transitively depend on these.

---

## 1. Schema Validation (`schema_validation.go`)

### What moves

The CLI's `validator.go` contains JSON schema validation logic that is pure data transformation:

- Embedded JSON schema files (via `go:embed`) — the library already embeds vocabularies using this pattern. The schemas live in `specification/schema/v1/` and are imported as `github.com/genealogix/glx/specification/schema/v1`.
- Schema reference resolution (`resolveRefs`, `resolveJSONPointerRef`, `resolveFileRef`) — graph traversal over in-memory schema data
- Document validation against resolved schemas
- Entity ID format validation

### New public API

```go
// ValidateGLXDocument validates a parsed YAML/JSON document against the
// embedded GLX JSON schemas. The document should be the result of
// unmarshaling a single GLX file (YAML or JSON) into a map.
// Returns errors and warnings using the existing ValidationResult type.
func ValidateGLXDocument(doc map[string]any) *ValidationResult

// IsValidEntityID returns true if the given string is a valid GLX entity ID
// (alphanumeric or hyphens, 1-64 characters).
func IsValidEntityID(id string) bool
```

Uses the existing `ValidationResult`, `ValidationError`, and `ValidationWarning` types — no new validation types needed. Schema validation issues populate `SourceType`/`SourceField`/`Message` with the JSON path and description; `TargetType`/`TargetID` are left empty (they're only relevant for cross-reference errors).

### What stays in CLI

- `ParseYAMLFile(path string)` — opens file from disk, unmarshals, then calls `ValidateGLXDocument`
- Error formatting and output

---

## 2. Validation Pipeline (`validation_pipeline.go`)

### What moves

The CLI's `validation_runner.go` orchestrates a multi-phase validation pipeline. The pure logic phases move to the library.

### Design

Schema validation and cross-reference validation operate at different stages:
- Schema validation runs on raw parsed data (`map[string]any`) *before* deserialization into typed structs
- Cross-reference validation runs on the deserialized `*GLXFile` *after* loading

These two phases stay separate in the library API rather than being combined into a single function. The `ArchiveLoader` (Section 3) can run both at the appropriate stages when `LoadOptions.Validate` is true:
1. Each `Load()` call runs `ValidateGLXDocument` on the raw parsed map before merging entities
2. `Finalize()` runs the existing `Validate()` for cross-reference checks

There is no new `ValidateArchive` function. Instead, the loader handles orchestration, and consumers who load data through other paths can call `ValidateGLXDocument` and `Validate()` independently.

### What stays in CLI

- Directory walking and file collection
- Media file existence checks (requires filesystem access)
- Output formatting

---

## 3. Archive Loading (`archive.go`)

### Design

A streaming loader that accepts `io.Reader`s one at a time. Each reader contains a GLX document (single-file or one entity file from a multi-file archive). The loader parses and accumulates entities incrementally.

This unifies single-file and multi-file loading behind one API. The distinction between archive formats becomes a CLI concern (is this a file or a directory?).

### Relationship to existing API

`DeserializeSingleFileBytes` and `DeserializeMultiFileFromMap` remain unchanged. The `ArchiveLoader` is a higher-level API built on the same underlying deserialization logic. Existing consumers are unaffected.

### New public API

```go
// LoadOptions configures archive loading behavior.
type LoadOptions struct {
    Validate bool // Run schema + cross-reference validation
}

// ArchiveLoader accumulates GLX entities from one or more readers.
// Not safe for concurrent use — call Load sequentially.
type ArchiveLoader struct {
    // unexported fields
}

// NewArchiveLoader creates a new loader with the given options.
func NewArchiveLoader(opts LoadOptions) *ArchiveLoader

// Load parses a GLX document from the reader and accumulates its entities
// into the archive being built. Can be called once (single-file archive)
// or many times (multi-file archive, one call per entity file).
//
// If LoadOptions.Validate is true, runs schema validation on the parsed
// document before merging. Returns an error if the reader contains
// unparseable data. Schema validation issues are accumulated and
// returned from Finalize.
//
// If an entity ID appears in multiple Load calls, the first occurrence
// wins. Duplicates are accumulated as warnings and returned from Finalize.
func (l *ArchiveLoader) Load(r io.Reader) error

// Finalize returns the completed GLXFile and a ValidationResult containing
// any schema validation issues and duplicate warnings accumulated during
// Load calls. If LoadOptions.Validate is true, also runs cross-reference
// validation and includes those results.
//
// Returns (*GLXFile, *ValidationResult, error). The GLXFile is always
// returned if parsing succeeded, even if validation found issues — the
// caller decides whether to use it. The error is only for fatal problems
// (e.g., no data loaded).
//
// After Finalize, the loader should not be reused.
func (l *ArchiveLoader) Finalize() (*GLXFile, *ValidationResult, error)
```

### CLI usage pattern

```go
// Single-file
f, _ := os.Open("archive.glx")
defer f.Close()
loader := glx.NewArchiveLoader(glx.LoadOptions{Validate: true})
loader.Load(f)
archive, result, err := loader.Finalize()

// Multi-file — feed readers as files are discovered
loader := glx.NewArchiveLoader(glx.LoadOptions{Validate: true})
filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
    if !d.IsDir() && strings.HasSuffix(path, ".glx") {
        f, _ := os.Open(path)
        defer f.Close()
        loader.Load(f)
    }
    return nil
})
archive, result, err := loader.Finalize()
```

### What stays in CLI

- File/directory detection
- Opening files and passing readers
- Directory walking
- Writing archives to disk (serialization output path unchanged)

---

## Migration Strategy

1. Add new library code alongside existing API (no breaking changes)
2. Update CLI to use new library functions where applicable
3. Remove duplicated logic from CLI
4. Existing tests continue to pass throughout

---

## Testing

- Library tests use `strings.NewReader` / `bytes.NewReader` for all reader-based APIs
- Schema validation tests use known-good and known-bad document maps
- `ArchiveLoader` tests verify single-call and multi-call accumulation, including duplicate handling
- CLI tests remain as integration/E2E tests
