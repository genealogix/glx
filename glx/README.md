# GLX - GENEALOGIX CLI Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/spec/glx)](https://goreportcard.com/report/github.com/genealogix/spec/glx)
[![Coverage](https://img.shields.io/badge/coverage-69.1%25-yellow.svg)](coverage.out)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](#running-tests)

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 🔍 **Validate Files** - Comprehensive validation with cross-reference checking
- 📋 **Schema Validation** - Verify JSON schemas have required metadata
- 🧪 **Test Suite** - 69.1% code coverage with comprehensive test fixtures
- 📚 **Examples Validation** - Automatically validates documentation examples

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/genealogix/spec.git
cd spec/glx

# Build the tool
go build -o glx .

# Optional: Install to PATH
go install
```

### Using Go Install

```bash
go install github.com/genealogix/spec/glx@latest
```

## Quick Start

```bash
# Create a new family archive
glx init

# Create a single-file archive
glx init --single-file

# Validate all files in current directory
glx validate

# Validate specific files or directories
glx validate persons/
glx validate archive.glx
glx validate persons/ events/

# Check JSON schemas
glx check-schemas
```

## Commands

### `glx init`

Initialize a new GENEALOGIX archive.

**Usage:**
```bash
glx init [--single-file]
```

**Options:**
- `--single-file`, `-s` - Create a single-file archive (default: multi-file)

**Examples:**

**Multi-file archive (default):**
```bash
glx init
```

Creates:
```
.
├── persons/
├── relationships/
├── events/
├── places/
├── sources/
├── citations/
├── repositories/
├── assertions/
├── media/
├── vocabularies/
│   ├── relationship-types.glx
│   ├── event-types.glx
│   ├── place-types.glx
│   ├── repository-types.glx
│   ├── participant-roles.glx
│   ├── media-types.glx
│   ├── confidence-levels.glx
│   └── quality-ratings.glx
├── .gitignore
└── README.md
```

**Single-file archive:**
```bash
glx init --single-file
```

Creates:
```
.
└── archive.glx
```

### `glx validate`

Validate GLX files and verify cross-references.

**Usage:**
```bash
glx validate [paths...]
```

**Validation Checks:**
- ✓ YAML syntax correctness
- ✓ Required fields presence
- ✓ Entity ID format (alphanumeric + hyphens, 1-64 chars)
- ✓ No 'id' fields in entities (IDs are map keys)
- ✓ Version field presence
- ✓ Entity type-specific validation
- ✓ Cross-reference integrity
- ✓ Duplicate ID detection
- ✓ Vocabulary validation (if vocabularies/ exists)

**Examples:**

```bash
# Validate current directory
glx validate

# Validate specific directory
glx validate persons/

# Validate multiple paths
glx validate persons/ events/ places/

# Validate single file
glx validate archive.glx

# Validate example archives
glx validate ../docs/examples/complete-family/
```

**Output:**
```
✓ persons/person-john-smith.glx
✓ events/event-john-birth.glx
✓ places/place-leeds.glx
✗ citations/citation-error.glx
  - source_id or source is required

Validated 3 file(s)
```

### `glx check-schemas`

Validate JSON schema files for required metadata.

**Usage:**
```bash
glx check-schemas
```

**Checks:**
- ✓ `$schema` field presence
- ✓ `$id` field presence

**Example:**
```bash
cd specification/
glx check-schemas
```

## File Format

GENEALOGIX uses YAML files with `.glx` extension. Entities are stored as maps where the key is the entity ID.

### Single-File Format

```yaml
# archive.glx
persons:
  person-john-smith:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"

relationships:
  rel-marriage:
    version: "1.0"
    type: "marriage"
    persons:
      - person-john-smith
      - person-mary-brown

events:
  event-john-birth:
    version: "1.0"
    type: "birth"
    date: "1850-01-15"
    place: place-leeds
```

### Multi-File Format

Each file contains one entity type:

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"
```

```yaml
# events/event-john-birth.glx
events:
  event-john-birth:
    version: "1.0"
    type: "birth"
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: "principal"
```

## Project Structure

```
glx/
├── main.go                  # CLI entry point
├── validate.go              # Validation orchestration
├── validator.go             # Core validation logic
├── check_schemas.go         # Schema validation
├── main_test.go            # Tests for main.go
├── validate_test.go        # Tests for validation
├── check_schemas_test.go   # Tests for schema checks
├── examples_test.go        # Tests for docs/examples
├── testdata/               # Test fixtures
│   ├── valid/             # Valid test files
│   │   ├── person-minimal.glx
│   │   ├── archive-small-family.glx
│   │   └── ...
│   ├── invalid/           # Invalid test files
│   │   ├── person-missing-id.glx
│   │   ├── archive-broken-references.glx
│   │   └── ...
│   └── README.md          # Test data documentation
└── README.md              # This file
```

## Testing

### Running Tests

```bash
# Run all tests
go test -v

# Run specific test
go test -v -run TestValidate

# Run with coverage
go test -cover

# Generate coverage report
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

Current coverage: **69.1%**

Covered areas:
- ✓ Entity validation (persons, events, places, sources, etc.)
- ✓ Cross-reference validation
- ✓ Duplicate ID detection
- ✓ Single-entity and multi-entity archives
- ✓ Examples validation (36 files)
- ✓ CLI commands (init, validate, check-schemas)
- ✓ Error handling

### Test Categories

#### Unit Tests (`validate_test.go`)
- Entity ID validation
- YAML parsing
- Entity type-specific validation
- Vocabulary loading
- Cross-reference checking
- Archive validation

#### Integration Tests (`main_test.go`)
- Repository initialization
- Single-file vs multi-file creation
- Vocabulary generation
- Directory structure validation

#### Examples Tests (`examples_test.go`)
- Validates all 36 example files in `docs/examples/`
- Tests minimal, basic-family, and complete-family examples
- Verifies cross-reference integrity across examples
- Checks example directory structure

#### Schema Tests (`check_schemas_test.go`)
- Schema metadata validation
- Missing field detection

### Test Fixtures

**Valid test files** (23 files in `testdata/valid/`):
- Single-entity tests (person-minimal.glx, event-complete.glx, etc.)
- Multi-entity archives (archive-small-family.glx, archive-evidence-chain.glx, etc.)

**Invalid test files** (18 files in `testdata/invalid/`):
- Missing fields (person-missing-id.glx, event-missing-type.glx, etc.)
- Broken references (archive-broken-references.glx)
- Duplicate IDs (archive-duplicate-ids.glx)
- Invalid formats (person-bad-id-format.glx)

See [testdata/README.md](testdata/README.md) for complete test data documentation.

## Development

### Prerequisites

- Go 1.25 or later
- Git

### Building

```bash
go build -o glx .
```

### Dependencies

```go
require (
    github.com/xeipuuv/gojsonschema v1.2.0
    github.com/stretchr/testify v1.11.1
    gopkg.in/yaml.v3 v3.0.1
)
```

### Code Organization

- **`main.go`** - CLI commands and initialization
- **`validate.go`** - Validation orchestration and reporting
- **`validator.go`** - Core entity validation, cross-references, vocabularies
- **`check_schemas.go`** - JSON schema metadata validation

### Adding New Validation Rules

1. Add logic to `validator.go::ValidateGLXFile()` or `basicValidateEntity()`
2. Add test cases to `validate_test.go`
3. Add test fixtures to `testdata/valid/` or `testdata/invalid/`
4. Update documentation

### Contributing

Contributions are welcome! Please:

1. Write tests for new functionality
2. Ensure `go test -cover` shows increased coverage
3. Run `go test -v` before submitting
4. Follow Go conventions and idioms
5. Update documentation

## Related Documentation

- [GENEALOGIX Specification](../specification/)
- [JSON Schemas](../specification/schema/v1/)
- [Examples](../docs/examples/)
- [Test Data Documentation](testdata/README.md)
- [Contributing Guide](../CONTRIBUTING.md)

## License

Apache License 2.0 - See [LICENSE](../LICENSE) for details.

## Support

- 📖 [Specification](../specification/)
- 💡 [Examples](../docs/examples/)
- 🐛 [Issue Tracker](https://github.com/genealogix/spec/issues)
- 💬 [Discussions](https://github.com/genealogix/spec/discussions)

