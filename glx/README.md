# GLX - GENEALOGIX CLI Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx)](https://goreportcard.com/report/github.com/genealogix/glx)

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 📥 **GEDCOM Import** - Import GEDCOM 5.5.1 and 7.0 files to GLX format
- 📤 **GEDCOM Export** - Export GLX archives back to GEDCOM 5.5.1 or 7.0 format
- 🔍 **Validate Files** - Structural and referential integrity validation
- 🔄 **Split/Join** - Convert between single-file and multi-file formats
- 📊 **Stats** - Display a summary dashboard of entity counts, assertion confidence, and coverage
- 📍 **Places** - Analyze places for data quality issues (duplicates, missing coordinates, hierarchy gaps)
- 🔎 **Query** - Filter and list entities from an archive by name, date, type, source, and more
- 👤 **Vitals** - Display vital records (birth, death, burial) for a person
- 📅 **Timeline** - Show chronological events for a person, including family events
- 📝 **Summary** - Comprehensive person profile with auto-generated life history narrative
- 🌳 **Ancestors/Descendants** - Display ancestor and descendant trees with box-drawing characters
- 📎 **Cite** - Generate formatted citation text from structured citation data
- 🔬 **Analyze** - Research gap analysis: evidence gaps, quality issues, chronological inconsistencies, and suggestions
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

# Export back to GEDCOM
glx export family-archive -o family.ged
glx export family-archive -o family70.ged --format 70

# Show a stats dashboard for an archive
glx stats family-archive

# Analyze places for data quality issues
glx places family-archive

# Run research gap analysis
glx analyze family-archive
glx analyze --check consistency
glx analyze "John Smith"

# Look up a person's vital records
glx vitals "John Smith"

# Show a chronological timeline of events
glx timeline "John Smith"

# Display a comprehensive person profile
glx summary "John Smith"

# Display ancestor and descendant trees
glx ancestors person-abc123
glx descendants person-abc123 --generations 3

# Generate formatted citation text
glx cite citation-abc123

# Query persons born before 1850
glx query persons --born-before 1850

# Find all marriage events
glx query events --type marriage

# Find assertions from a specific source
glx query assertions --source source-abc123

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

Validate GLX files for structural and referential integrity.

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

> **Note:** Temporal consistency checks (death before birth, parent younger than child, etc.) are handled by `glx analyze`, not `glx validate`. This keeps validation focused on structural correctness.

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

### `glx export`

Export a GLX archive to GEDCOM format.

**Usage:**
```bash
glx export <glx-archive> -o <output> [flags]
```

**Options:**
- `-o, --output <path>` - Output GEDCOM file path (required)
- `-f, --format <format>` - GEDCOM version: `551` or `70` (default: `551`)
- `-v, --verbose` - Verbose output

**Supported Output Formats:**
- GEDCOM 5.5.1 (default)
- GEDCOM 7.0

**Features:**
- Accepts single-file (`.glx`) or multi-file archive directories as input
- Reconstructs GEDCOM FAM records from GLX relationships
- Converts dates, places, and names back to GEDCOM format
- Preserves sources, repositories, media, citations, and notes
- Exports inline SOUR citations on individual events
- Handles single-spouse families, multiple marriage events, and multi-family children

**Examples:**

```bash
# Export to GEDCOM 5.5.1 (default)
glx export family-archive -o family.ged

# Export a single-file archive
glx export family.glx -o family.ged

# Export to GEDCOM 7.0
glx export family-archive -o family.ged --format 70

# Export with verbose output
glx export family-archive -o family.ged --verbose
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

| Entity type    | Supported flags                                        |
| -------------- | ------------------------------------------------------ |
| `persons`      | `--name`, `--born-before`, `--born-after`               |
| `events`       | `--type`, `--before`, `--after`                         |
| `assertions`   | `--confidence`, `--status`, `--source`, `--citation`    |
| `sources`      | `--name`, `--type`                                      |
| `relationships`| `--type`                                                |
| `places`       | `--name`                                                |
| `repositories` | `--name`                                                |
| `citations`    | _(no filters)_                                          |
| `media`        | _(no filters)_                                          |

**All filter options:**
- `--name <string>` - Filter by name (substring match, case-insensitive). For persons, searches all name variants including birth names, married names, maiden names, and as-recorded forms
- `--born-before <year>` - Filter persons born before this year
- `--born-after <year>` - Filter persons born after this year
- `--type <string>` - Filter by type (event type, relationship type, or source type)
- `--before <year>` - Filter events with date before this year
- `--after <year>` - Filter events with date after this year
- `--confidence <string>` - Filter assertions by confidence level (e.g. `high`, `medium`, `low`)
- `--status <string>` - Filter assertions by status
- `--source <id>` - Filter assertions by source ID (matches assertions referencing the source directly or via a citation)
- `--citation <id>` - Filter assertions by citation ID

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

# Find all assertions citing a specific source
glx query assertions --source source-1860-census

# Find assertions using a specific citation
glx query assertions --citation citation-abc123

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

> **Note:** Name matching is case-insensitive and matches any substring. For persons, `--name` searches all name variants including birth names, married names, maiden names, and as-recorded forms — not just the primary name. Results show alternate names with an "aka:" suffix. Year filters use the first four-digit year found in a date string, so formats like `ABT 1850`, `BEF 1920-01-15`, and `BET 1880 AND 1890` are all supported.

### `glx vitals`

Display vital records for a person.

**Usage:**
```bash
glx vitals <person> [flags]
```

**Arguments:**
- `<person>` - Person ID (e.g., `person-d-lane`) or name to search for (case-insensitive substring match)

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)

**Shows:**
- Name, Sex, Birth, Christening, Death, Burial
- Any other life events the person participated in (marriages, census records, etc.)

If the name matches multiple persons, all matches are listed for disambiguation.

**Examples:**

```bash
# Look up by person ID
glx vitals person-d-lane

# Look up by name
glx vitals "Mary Green"

# Specify archive path
glx vitals "Mary Green" --archive my-archive
```

### `glx timeline`

Display a chronological timeline of all events in a person's life.

**Usage:**
```bash
glx timeline <person> [flags]
```

**Arguments:**
- `<person>` - Person ID or name to search for

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)
- `--no-family` - Exclude family events (show only direct events)

**Shows:**
- Direct events where the person is a participant
- Family events discovered through relationship traversal (spouse births/deaths, children's births/deaths, parent deaths)
- Undated events in a separate section

**Examples:**

```bash
# Timeline by person ID
glx timeline person-john-smith

# Timeline by name
glx timeline "John Smith"

# Direct events only (no family events)
glx timeline "John Smith" --no-family

# Specify archive path
glx timeline "John Smith" --archive my-archive
```

### `glx summary`

Display a comprehensive profile for a person, including an auto-generated life history narrative.

**Usage:**
```bash
glx summary <person> [flags]
```

**Arguments:**
- `<person>` - Person ID or name to search for

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)

**Sections displayed:**
- **Identity** - Name, sex, alternate names (birth, married, maiden, AKA, etc.)
- **Vital Events** - Birth, christening, death, burial
- **Life Events** - Census, immigration, naturalization, military service, etc.
- **Family** - Spouse(s) with marriage info, parents, siblings
- **Relationships** - Godparent, neighbor, household, employment, etc.
- **Life History** - Auto-generated biographical narrative

**Examples:**

```bash
# Summary by person ID
glx summary person-abc123

# Summary by name search
glx summary "Mary Lane"

# Summary in a specific archive
glx summary "John Smith" --archive my-family-archive
```

### `glx ancestors`

Display the ancestor tree for a person by traversing parent-child relationships.

**Usage:**
```bash
glx ancestors <person-id> [flags]
```

**Arguments:**
- `<person-id>` - Person entity ID

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)
- `-g, --generations <n>` - Maximum number of generations (0 for unlimited, default: 0)

Traverses all parent-child relationship variants (biological, adoptive, foster, step). Non-default relationship types are annotated in the output. Includes cycle detection.

**Examples:**

```bash
# Show ancestors
glx ancestors person-abc123

# Limit to 3 generations
glx ancestors person-abc123 --generations 3

# Use a specific archive
glx ancestors person-abc123 --archive my-archive
```

### `glx descendants`

Display the descendant tree for a person by traversing parent-child relationships.

**Usage:**
```bash
glx descendants <person-id> [flags]
```

**Arguments:**
- `<person-id>` - Person entity ID

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)
- `-g, --generations <n>` - Maximum number of generations (0 for unlimited, default: 0)

Traverses all parent-child relationship variants (biological, adoptive, foster, step). Non-default relationship types are annotated in the output. Includes cycle detection.

**Examples:**

```bash
# Show descendants
glx descendants person-abc123

# Limit to 3 generations
glx descendants person-abc123 --generations 3

# Use a specific archive
glx descendants person-abc123 --archive my-archive
```

### `glx cite`

Generate formatted citation text from structured citation data.

**Usage:**
```bash
glx cite [citation-id] [flags]
```

**Arguments:**
- `[citation-id]` - Optional citation entity ID. If omitted, prints all citations in the archive.

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)

Assembles citations from the source title, source type, repository name, URL, and accessed date already stored in the archive. This eliminates repetitive manual writing of the `citation_text` property.

**Examples:**

```bash
# Format a specific citation
glx cite citation-1860-census-lane-household

# Format all citations in the archive
glx cite

# Use a specific archive
glx cite --archive my-archive
```

### `glx analyze`

Analyze a GLX archive for research gaps, evidence quality issues, chronological inconsistencies, and research suggestions.

**Usage:**
```bash
glx analyze [person] [flags]
```

**Arguments:**
- `[person]` - Optional person ID or name to filter results to a single person

**Options:**
- `-a, --archive <path>` - Archive path (directory or single file; defaults to current directory)
- `-c, --check <category>` - Run a single analysis category: `gaps`, `evidence`, `consistency`, or `suggestions`
- `-f, --format <format>` - Output format (`json` for machine-readable)
- `-p, --person <id-or-name>` - Filter results to a specific person (alternative to positional argument)

**Analysis categories:**

| Category | What it checks |
| --- | --- |
| `gaps` | Missing data that should be findable (no birth date, no parents, no events) |
| `evidence` | Unsupported or weakly supported claims (no citations, single-source persons, orphaned citations/sources) |
| `consistency` | Chronological cross-checks (death before birth, parent younger than child, implausible lifespan) |
| `suggestions` | Research recommendations (census years to search, vital records to locate) |

**Examples:**

```bash
# Full analysis of current directory
glx analyze

# Focus on one person
glx analyze person-mary-lane

# Run only gap analysis
glx analyze --check gaps

# JSON output for tooling
glx analyze --format json

# Analyze a specific archive
glx analyze --archive my-archive
```

**Output:**
```
=== Research Gap Analysis: 42 issues found ===

EVIDENCE GAPS (15)
  HIGH person-john-smith             No birth date
  HIGH person-mary-brown             No parents
  MED  person-thomas-smith           No death date (born before 1930)

EVIDENCE QUALITY (12)
  HIGH person-john-smith             No assertions backed by citations
  MED  person-mary-brown             Only one source

CONSISTENCY (5)
  HIGH person-jane-doe               Death year (1810) before birth year (1842)

SUGGESTIONS (10)
  →   person-john-smith              Search 1850 census (born ~1842)
  →   person-mary-brown              Look for vital records in Leeds
```

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

