# Architecture Guide

This guide explains the technical architecture of the GENEALOGIX CLI implementation.

## System Overview

The GENEALOGIX CLI is a Go-based tool that validates and manages GLX archives:

1. **Specification**: Formal data model (see [specification/](../../specification/))
2. **JSON Schemas**: Embedded validation rules
3. **Go Structs**: Type-safe parsing and validation
4. **CLI Tool**: Command-line interface

## CLI Tool Architecture

### Command Structure

```bash
glx init [directory]      # Initialize new archive
glx validate [paths...]   # Validate .glx files
glx check-schemas         # Validate schema files
```

See [CLI README](../../glx/README.md) for command details.

### Validation Pipeline

The validator performs multi-stage validation:

1. **JSON Schema Validation**
   - Parse YAML files
   - Validate against embedded JSON schemas
   - Check required fields and types

2. **Struct-Based Parsing**
   - Parse files into Go structs (`lib.GLXFile`)
   - Merge all files into single `GLXFile`
   - Detect duplicate IDs (fatal error)

3. **Reference Validation**
   - Use reflection with `refType` struct tags
   - Validate entity references (e.g., `person`, `place`)
   - Validate vocabulary references (e.g., `event_types`)
   - Report all reference errors at once

See [validator.go](../../glx/validator.go) for implementation.

## Type System

### Go Structs

Located in `lib/types.go`:
- Structs for all entity and vocabulary types
- YAML tags for serialization
- `refType` tags for reference validation
- `Merge` method for combining files

**Naming Conventions:**
- **Go fields**: Use `ID` suffix for readability (e.g., `SourceID`, `PersonID`)
- **YAML tags**: Singular entity names (e.g., `yaml:"source"`, `yaml:"person"`)
- **refType tags**: Match GLXFile map keys (e.g., `refType:"sources"`, `refType:"persons"`)

Example:
```go
type Citation struct {
    SourceID     string   `yaml:"source" refType:"sources"`
    RepositoryID string   `yaml:"repository,omitempty" refType:"repositories"`
    Media        []string `yaml:"media,omitempty" refType:"media"`
}
```

### Reference Validation

Uses reflection to validate references:
- `refType` struct tags define reference targets
- Comma-delimited for multiple valid types (e.g., `refType:"persons,events,relationships,places"`)
- Validates both entity and vocabulary references uniformly
- Reports all errors at once

Implementation in `validateEntityReferences()` function.

## Schema Embedding

Schemas are embedded in the Go binary using `go:embed`:

```go
// specification/schema/v1/embed.go
//go:embed person.schema.json
var PersonSchema []byte

var EntitySchemas = map[string][]byte{
    "person": PersonSchema,
    "relationship": RelationshipSchema,
    // ...
}
```

**Benefits:**
- No file I/O at runtime
- Single binary distribution
- Guaranteed schema availability

After modifying schemas, rebuild the CLI:
```bash
cd glx
go build
```

## Vocabulary Loading

Vocabularies can be defined in any `.glx` file in the archive.

The `LoadArchiveVocabularies()` function:
- Walks all `.glx` files
- Parses each into `lib.GLXFile`
- Merges vocabulary definitions
- Returns unified vocabulary set

This allows flexible vocabulary organization without hardcoded paths.

## Validation Flow

### File-by-File Validation

1. Parse YAML
2. Validate against JSON schema
3. Parse into Go struct
4. Merge into master `GLXFile`
5. Check for duplicate IDs

### Cross-File Validation

After all files are merged:
1. Build lookup maps for all entities and vocabularies
2. Walk all structs using reflection
3. Check each `refType` tagged field
4. Validate references exist
5. Report all errors

See `ValidateReferencesWithStructs()` in [validator.go](../../glx/validator.go).

## Performance

### Optimization Strategies

- Parallel file parsing
- Embedded schemas (no file I/O)
- Single-pass reference validation
- Efficient struct-based validation

### Tested Scale

- 1000+ files
- Multi-megabyte archives
- Complex reference graphs

## Testing

Tests are in `glx/` directory:

```
glx/
├── main_test.go          # CLI tests
├── validate_test.go      # Validation tests
└── testdata/
    ├── valid/            # Valid test files
    └── invalid/          # Invalid test files
```

Tests run in CI on every commit.

See [Testing Guide](testing-guide.md) for details.

## See Also

- [Setup Guide](setup.md) - Development environment
- [Schema Development](schema-development.md) - Schema maintenance
- [Testing Guide](testing-guide.md) - Testing framework
- [Specification](../../specification/README.md) - Formal specification
