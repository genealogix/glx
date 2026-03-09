# GLX - GENEALOGIX CLI Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx)](https://goreportcard.com/report/github.com/genealogix/glx)

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 📥 **GEDCOM Import** - Import GEDCOM 5.5.1 and 7.0 files to GLX format
- 🔍 **Validate Files** - Comprehensive validation with cross-reference checking
- 🔄 **Split/Join** - Convert between single-file and multi-file formats
- 📊 **Stats** - Display a summary dashboard of entity counts, assertion confidence, and coverage
- 📍 **Places** - Analyze places for data quality issues (duplicates, missing coordinates, hierarchy gaps)
- 🔎 **Query** - Filter and list entities from an archive by name, date, type, and more
- 📋 **Schema Validation** - Verify JSON schemas have required metadata
- 🧪 **Test Suite** - Comprehensive test fixtures with coverage reporting
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
glx import family.ged -o family-archive

# Validate all files in the new directory
cd family-archive
glx validate

# Join multi-file archive back to single file
glx join family-archive combined.glx

# Show a stats dashboard for an archive
glx stats family-archive

# Analyze places for data quality issues
glx places family-archive

# Query persons born before 1850
glx query persons --born-before 1850

# Find all marriage events
glx query events --type marriage

# List all sources
glx query sources

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
│   └── confidence-levels.glx
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
glx validate [path] --report
```

**Options:**
- `--report` - Generate confidence summary report (assertion coverage and gaps)

**Validation Checks:**
- ✓ YAML syntax correctness
- ✓ Required fields presence
- ✓ Entity ID format (alphanumeric + hyphens, 1-64 chars)
- ✓ No 'id' fields in entities (IDs are map keys)
- ✓ Entity type-specific validation
- ✓ Cross-reference integrity
- ✓ Duplicate ID detection
- ✓ Vocabulary validation (if vocabularies/ exists)

**Behavior:**
- **Directory or multi-path validation** performs full cross-reference checking across all files
- **Single-file validation** checks structure only (no cross-reference checks)

**Examples:**

```bash
# Validate current directory (with cross-reference checks)
glx validate

# Validate specific directory (with cross-reference checks)
glx validate persons/

# Validate multiple paths (with cross-reference checks)
glx validate persons/ events/ places/

# Validate single file (structure only, no cross-reference checks)
glx validate archive.glx

# Validate example archives
glx validate ../docs/examples/complete-family/

# Generate confidence summary report
glx validate --report
glx validate path/to/archive --report
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
glx import family.ged -o family.glx --format single

# Import to multi-file directory
glx import family.ged -o family-archive

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

### `glx stats`

Display a summary dashboard for any GLX archive, showing entity counts, assertion confidence distribution, and entity coverage.

**Usage:**
```bash
glx stats [path]
```

**Arguments:**
- `[path]` - Path to a multi-file archive directory or a single-file `.glx` archive (defaults to current directory)

**Output sections:**

- **Entity counts** — total number of each entity type in the archive
- **Assertion confidence** — breakdown of assertions by confidence level with percentages (only shown when assertions exist)
- **Entity coverage** — how many persons, events, relationships, and places are referenced by at least one assertion (only shown when assertions exist)

**Examples:**

```bash
# Stats for a multi-file archive directory
glx stats family-archive

# Stats for a single-file archive
glx stats family.glx
```

**Output:**
```
Entity counts:
  Persons:       31
  Events:        77
  Relationships: 42
  Places:        5
  Sources:       5
  Citations:     12
  Repositories:  2
  Media:         0
  Assertions:    20

Assertion confidence:
  high           8  ( 40.0%)
  medium         6  ( 30.0%)
  (unset)        6  ( 30.0%)

Entity coverage (referenced by assertions):
  Persons          12/31  (38.7%)
  Events           5/77  (6.5%)
  Relationships    3/42  (7.1%)
  Places           -
```

> **Note:** The confidence distribution lists standard levels first (high, medium, low, disputed), then any custom levels alphabetically, with `(unset)` last. The coverage section shows `-` for entity types with no entries in the archive.

### `glx places`

Analyze places in a GLX archive for data quality issues. Reports duplicate names, missing coordinates, missing types, hierarchy gaps, dangling parent references, and unreferenced places.

**Usage:**
```bash
glx places [path]
```

**Arguments:**
- `[path]` - Path to a multi-file archive directory or a single-file `.glx` archive (defaults to current directory)

**Reports:**

- **Duplicate names** — places that share the same name (ambiguous without context)
- **Missing coordinates** — places without latitude/longitude
- **Missing type** — places without a type classification
- **No parent** — places (other than countries and regions) missing a parent (hierarchy gap)
- **Dangling parent** — places referencing a parent that doesn't exist in the archive
- **Unreferenced** — places not used by any event, assertion, or as a parent

Each place is shown with its full canonical hierarchy path (e.g., "Leeds, Yorkshire, England").

**Examples:**

```bash
# Analyze places in current directory
glx places

# Analyze places in a specific archive
glx places my-family-archive

# Analyze a single-file archive
glx places family.glx
```

**Output:**
```
Place analysis: 106 places

Missing coordinates (106 of 106):
  place-acorn-hall  Acorn Hall, The Riverlands, Westeros
  place-astapor  Astapor, Essos
  place-braavos  Braavos, Essos
  ...

No parent (hierarchy gap):
  place-essos  Essos
  place-sothoryos  Sothoryos
  place-the-stepstones  The Stepstones
  place-westeros  Westeros

Unreferenced (not used by any event, assertion, or as parent):
  place-acorn-hall  Acorn Hall, The Riverlands, Westeros
  place-barrowton  Barrowton, The North, Westeros
  ...
```

> **Note:** Only sections with issues are shown. If all places have coordinates, parents, and are referenced, those sections are omitted and "No issues found." is printed.

### `glx query`

Filter and list entities from a GENEALOGIX archive. Supports all nine entity types with type-specific filter flags.

**Usage:**
```bash
glx query <entity-type> [flags]
```

**Arguments:**
- `<entity-type>` - One of: `persons`, `events`, `assertions`, `sources`, `relationships`, `places`, `citations`, `repositories`, `media`

**Common options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)

**Type-specific filter options:**

| Entity type    | Supported flags                          |
| -------------- | ---------------------------------------- |
| `persons`      | `--name`, `--born-before`, `--born-after` |
| `events`       | `--type`, `--before`, `--after`          |
| `assertions`   | `--confidence`, `--status`               |
| `sources`      | `--name`, `--type`                       |
| `relationships`| `--type`                                 |
| `places`       | `--name`                                 |
| `repositories` | `--name`                                 |
| `citations`    | _(no filters)_                           |
| `media`        | _(no filters)_                           |

**All filter options:**
- `--name <string>` - Filter by name (substring match, case-insensitive)
- `--born-before <year>` - Filter persons born before this year
- `--born-after <year>` - Filter persons born after this year
- `--type <string>` - Filter by type (event type, relationship type, or source type)
- `--before <year>` - Filter events with date before this year
- `--after <year>` - Filter events with date after this year
- `--confidence <string>` - Filter assertions by confidence level (e.g. `high`, `medium`, `low`)
- `--status <string>` - Filter assertions by status

**Examples:**

```bash
# List all persons in the current archive
glx query persons

# Find persons with "Smith" in their name
glx query persons --name "Smith"

# Find persons born before 1850
glx query persons --born-before 1850

# Find persons born between 1800 and 1860
glx query persons --born-after 1800 --born-before 1860

# List all marriage events
glx query events --type marriage

# Find events before 1900
glx query events --before 1900

# List low-confidence assertions
glx query assertions --confidence low

# Find disputed assertions with a specific status
glx query assertions --confidence disputed --status reviewed

# Find sources by title keyword
glx query sources --name "census"

# Find sources of a specific type
glx query sources --type vital-record

# Find parent-child relationships
glx query relationships --type parent-child

# Find places by name in a specific archive
glx query places --name "London" --archive family-archive

# List all citations in a single-file archive
glx query citations --archive family.glx

# List all repositories
glx query repositories
```

**Output:**
```
  person-a3f8d2c1  John Smith  (b. 1842-03-15 – d. 1901-07-22)
  person-b7c1e4f2  Mary Brown  (b. 1848)
  person-d9a2f6b3  Thomas Smith  (b. ABT 1870)

3 person(s) found
```

> **Note:** Name matching is case-insensitive and matches any substring. Year filters use the first four-digit year found in a date string, so formats like `ABT 1850`, `BEF 1920-01-15`, and `BET 1880 AND 1890` are all supported.

## File Format

GENEALOGIX uses YAML files with `.glx` extension. Entities are stored as maps where the key is the entity ID.

### Single-File Format

```yaml
# archive.glx
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"

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
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
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

## Testing

```bash
# Run all tests
make test

# Run all tests with verbose output
make test-verbose
```

See [testdata/README.md](https://github.com/genealogix/glx/blob/main/glx/testdata/README.md) for test data documentation.

## Development

### Prerequisites

- Go (see `go.mod` for minimum version)
- Git

### Building

```bash
make build
```

### Dependencies

See `go.mod` for the current dependency list.

### Contributing

Contributions are welcome! Please:

1. Write tests for new functionality
2. Run `make test` before submitting
3. Follow Go conventions and idioms
4. Update documentation

## Related Documentation

- [GENEALOGIX Specification](/specification/)
- [JSON Schemas](/specification/schema/)
- [Examples](/examples/)
- [Test Data Documentation](https://github.com/genealogix/glx/blob/main/glx/testdata/README.md)
- [Contributing Guide](/development/contributing)

## License

Apache License 2.0 - See [LICENSE](https://github.com/genealogix/glx/blob/main/LICENSE) for details.

## Support

- 📖 [Specification](/specification/)
- 💡 [Examples](/examples/)
- 🐛 [Issue Tracker](https://github.com/genealogix/glx/issues)
- 💬 [Discussions](https://github.com/genealogix/glx/discussions)

