# GLX - GENEALOGIX CLI Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx/glx)](https://goreportcard.com/report/github.com/genealogix/glx/glx)
[![Coverage](https://img.shields.io/badge/coverage-70.5%25-yellow.svg)](coverage.out)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)](#running-tests)

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 📥 **GEDCOM Import** - Import GEDCOM 5.5.1 and 7.0 files to GLX format
- 🔍 **Validate Files** - Comprehensive validation with cross-reference checking
- 🔄 **Split/Join** - Convert between single-file and multi-file formats
- 📋 **Schema Validation** - Verify JSON schemas have required metadata
- 🧪 **Test Suite** - 70.5% code coverage with comprehensive test fixtures
- 📚 **Examples Validation** - Automatically validates documentation examples

## Installation

### From GitHub Releases (Recommended)

Download the latest pre-built binary for your platform from the [Releases page](https://github.com/genealogix/glx/releases):

**macOS (Apple Silicon):**
```bash
# Download and extract (replace VERSION with the version number)
curl -L https://github.com/genealogix/glx/releases/download/VERSION/glx_Darwin_arm64.tar.gz | tar xz

# Move to PATH
sudo mv glx /usr/local/bin/

# Verify installation
glx --version
```

**macOS (Intel):**
```bash
# Download and extract (replace VERSION with the version number)
curl -L https://github.com/genealogix/glx/releases/download/VERSION/glx_Darwin_x86_64.tar.gz | tar xz

# Move to PATH
sudo mv glx /usr/local/bin/

# Verify installation
glx --version
```

**Linux (ARM64):**
```bash
# Download and extract (replace VERSION with the version number)
curl -L https://github.com/genealogix/glx/releases/download/VERSION/glx_Linux_arm64.tar.gz | tar xz

# Move to PATH
sudo mv glx /usr/local/bin/

# Verify installation
glx --version
```

**Linux (x86_64):**
```bash
# Download and extract (replace VERSION with the version number)
curl -L https://github.com/genealogix/glx/releases/download/VERSION/glx_Linux_x86_64.tar.gz | tar xz

# Move to PATH
sudo mv glx /usr/local/bin/

# Verify installation
glx --version
```

**Windows (ARM64):**
- Download `glx_Windows_arm64.zip` from the [Releases page](https://github.com/genealogix/glx/releases)
- Extract the ZIP file
- Add the directory to your PATH or move `glx.exe` to a directory in your PATH

**Windows (x86_64):**
- Download `glx_Windows_x86_64.zip` from the [Releases page](https://github.com/genealogix/glx/releases)
- Extract the ZIP file
- Add the directory to your PATH or move `glx.exe` to a directory in your PATH

### Using Go Install

```bash
go install github.com/genealogix/glx/glx@latest
```

### From Source

```bash
# Clone the repository
git clone https://github.com/genealogix/glx.git
cd glx/glx

# Build the tool
go build -o glx .

# Optional: Install to PATH
go install
```

## Quick Start

```bash
# Create a new family archive in the `my-family-archive` directory
glx init my-family-archive

# Import a GEDCOM file
glx import family.ged -o family.glx

# Split single-file archive to multi-file format
glx split family.glx family-archive

# Validate all files in the new directory
cd family-archive
glx validate

# Join multi-file archive back to single file
glx join family-archive combined.glx

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
glx init [directory] [--single-file]
```

**Options:**
- `--single-file`, `-s` - Create a single-file archive (default: multi-file)

**Examples:**

**Multi-file archive (default):**
```bash
glx init my-family-archive
```

Creates:
```
my-family-archive/
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
glx init my-family-archive --single-file
```

Creates:
```
my-family-archive/
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

### `glx import`

Import a GEDCOM file and convert it to GLX format.

**Usage:**
```bash
glx import <gedcom-file> -o <output> [flags]
```

**Options:**
- `-o, --output <path>` - Output file or directory (required)
- `-f, --format <format>` - Output format: `single` or `multi` (default: `single`)
- `--no-validate` - Skip validation before saving
- `-v, --verbose` - Verbose output

**Supported Formats:**
- ✓ GEDCOM 5.5.1
- ✓ GEDCOM 7.0

**Features:**
- Converts all individuals to persons
- Converts all events (births, deaths, marriages, etc.)
- Converts all relationships (parent-child, spouse, etc.)
- Builds place hierarchies from GEDCOM locations
- Converts sources, citations, repositories, and media
- Creates evidence-based assertions
- Includes standard vocabularies

**Examples:**

```bash
# Import to single file
glx import family.ged -o family.glx

# Import to multi-file directory
glx import family.ged -o family-archive --format multi

# Import without validation (faster, less safe)
glx import family.ged -o family.glx --no-validate

# Verbose output to see import progress
glx import family.ged -o family.glx --verbose
```

**Output:**
```
✓ Successfully imported to family.glx

Import statistics:
  Persons:       31
  Events:        77
  Relationships: 49
  Places:        5
  Sources:       0
  Citations:     0
  Repositories:  0
  Media:         0
  Assertions:    150
```

### `glx split`

Split a single-file GLX archive into a multi-file directory structure.

**Usage:**
```bash
glx split <input-file> <output-directory> [flags]
```

**Options:**
- `--no-validate` - Skip validation before splitting
- `-v, --verbose` - Verbose output

**Creates:**
```
output-directory/
├── persons/
│   ├── person-{id}.glx
│   └── ...
├── events/
│   ├── event-{id}.glx
│   └── ...
├── relationships/
│   ├── relationship-{id}.glx
│   └── ...
├── places/
│   ├── place-{id}.glx
│   └── ...
├── sources/
├── citations/
├── repositories/
├── media/
├── assertions/
└── vocabularies/
    ├── event-types.glx
    ├── relationship-types.glx
    └── ...
```

**Examples:**

```bash
# Split an archive
glx split family.glx family-archive

# Split without validation
glx split family.glx family-archive --no-validate
```

### `glx join`

Join a multi-file GLX archive into a single YAML file.

**Usage:**
```bash
glx join <input-directory> <output-file> [flags]
```

**Options:**
- `--no-validate` - Skip validation before joining
- `-v, --verbose` - Verbose output

**Examples:**

```bash
# Join an archive
glx join family-archive family.glx

# Join without validation (faster)
glx join family-archive family.glx --no-validate

# Verbose output
glx join family-archive family.glx --verbose
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
    properties:
      given_name: "John"
      family_name: "Smith"

relationships:
  rel-marriage:
    type: "marriage"
    persons:
      - person-john-smith
      - person-mary-brown

events:
  event-john-birth:
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
    properties:
      given_name: "John"
      family_name: "Smith"
```

```yaml
# events/event-john-birth.glx
events:
  event-john-birth:
    
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
    github.com/spf13/cobra v1.10.1          // CLI framework
    github.com/xeipuuv/gojsonschema v1.2.0  // JSON Schema validation
    github.com/stretchr/testify v1.11.1      // Test assertions
    gopkg.in/yaml.v3 v3.0.1                  // YAML parsing
)
```

### Code Organization

- **`main.go`** - Entry point (calls Execute())
- **`cmd_root.go`** - Cobra root command setup
- **`cmd_init.go`** - `glx init` command implementation
- **`cmd_validate.go`** - `glx validate` command implementation
- **`cmd_check_schemas.go`** - `glx check-schemas` command implementation
- **`vocabularies_embed.go`** - Embedded standard vocabulary files
- **`validator.go`** - Core entity validation, cross-references, vocabularies

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
- 🐛 [Issue Tracker](https://github.com/genealogix/glx/issues)
- 💬 [Discussions](https://github.com/genealogix/glx/discussions)

