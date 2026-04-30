# GLX - GENEALOGIX CLI Tool

[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx)](https://goreportcard.com/report/github.com/genealogix/glx)

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 📥 **GEDCOM Import** - Import GEDCOM 5.5.1 and 7.0 files to GLX format
- 📤 **GEDCOM Export** - Export GLX archives back to GEDCOM 5.5.1 or 7.0 format
- 🔍 **Validate Files** - Structural and referential integrity validation
- 🔄 **Split/Join** - Convert between single-file and multi-file formats
- 🔀 **Merge** - Combine two GLX archives with duplicate detection and dry-run support
- 📊 **Stats** - Display a summary dashboard of entity counts, assertion confidence, and coverage
- 📍 **Places** - Analyze places for data quality issues (duplicates, missing coordinates, hierarchy gaps)
- 🔍 **Search** - Full-text search across all entity types with case-sensitive and type-filter options
- 🔎 **Query** - Filter and list entities from an archive by name, date, type, source, and more
- 👤 **Vitals** - Display vital records (birth, death, burial) for a person
- 📅 **Timeline** - Show chronological events for a person, including family events
- 📝 **Summary** - Comprehensive person profile with auto-generated life history narrative
- 🌳 **Ancestors/Descendants** - Display ancestor and descendant trees with box-drawing characters
- 📎 **Cite** - Generate formatted citation text from structured citation data
- 🔗 **Cluster** - FAN club analysis identifying associates through census, events, and place overlap
- 🔗 **Path** - Find the shortest relationship path between two people using BFS
- 🔬 **Analyze** - Research gap analysis: evidence gaps, quality issues, chronological inconsistencies, and suggestions
- 📋 **Census Import** - Generate GLX entities from structured census templates with person matching, assertions, and dry-run preview
- 🔄 **Migrate** - Convert deprecated person properties to birth/death events
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

# FAN club analysis for brickwall research
glx cluster person-mary-lane --place place-ironton-sauk-wi --before 1860
# Import a census template into an archive
glx census add --from 1860-census-lane.yaml --archive my-archive

# Preview without writing files
glx census add --from 1860-census-lane.yaml --archive my-archive --dry-run

# Find the relationship path between two people
glx path "Mary Lane" "John Smith"

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

The `glx` CLI groups its commands into archive management, import/export, exploration, data entry, and analysis. The full per-command reference (flags, examples, aliases) is auto-generated from the live Cobra command tree on every build:

- Browse online: <https://genealogix.dev/cli/>
- Read in this repo: [`docs/cli/glx.md`](../docs/cli/glx.md)
- Regenerate locally: `make docs-cli`

CI fails on any drift between `glx/cli_commands.go` and the committed pages under `docs/cli/`. To change the docs for a command, edit its `Use`/`Short`/`Long`/`Example` strings (or its `*_runner.go` file) and re-run `make docs-cli`.

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

