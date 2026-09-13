# GLX - GENEALOGIX CLI Tool

The official command-line tool for working with GENEALOGIX (GLX) family archives. Validates GLX files, initializes new archives, and checks schema conformance.

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
# Pin the release tag: GLX releases are prereleases, which @latest skips
go install github.com/genealogix/glx/glx@v0.0.0-beta.12
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
```

## Features

- ✅ **Initialize Archives** - Create new single-file or multi-file genealogy archives
- 📥 **GEDCOM Import** - Import GEDCOM 5.5.1 and 7.0 files (and GEDZIP `.gdz` archives with bundled media) to GLX format
- 📤 **GEDCOM Export** - Export GLX archives back to GEDCOM 5.5.1 or 7.0 format
- 🌐 **JSON-LD Export** - Export GLX archives as Schema.org-aligned JSON-LD for linked-data interop
- 🌐 **Publish** - Generate a self-contained static HTML site (person profiles, timelines, family links, source/place indexes, client-side search) for sharing with non-technical family
- 🔍 **Validate Files** - Structural and referential integrity validation
- 🔄 **Split/Join** - Convert between single-file and multi-file formats
- 🔀 **Merge** - Combine two GLX archives with duplicate detection and dry-run support
- 🔀 **Merge Driver** - Structural 3-way git merge for `.glx` files (`glx merge-driver`) that auto-resolves safe concurrent edits and falls back to text merge otherwise
- 📊 **Stats** - Display a summary dashboard of entity counts, assertion confidence, and coverage
- 📍 **Places** - Analyze places for data quality issues (duplicates, missing coordinates, hierarchy gaps)
- 🔍 **Search** - Full-text search across all entity types with case-sensitive and type-filter options
- 🔎 **Query** - Filter and list entities from an archive by name, date, type, source, and more
- 👤 **Vitals** - Display vital records (birth, death, burial) for a person
- 📅 **Timeline** - Show chronological events for a person, including family events
- 🧭 **Migrations** - Trace a person's geographic movement over time and find others with the same migration pattern
- 📝 **Summary** - Comprehensive person profile with auto-generated life history narrative
- 🌳 **Ancestors/Descendants** - Display ancestor and descendant trees with box-drawing characters
- 📎 **Cite** - Generate formatted citation text from structured citation data
- 🔗 **Cluster** - FAN club analysis identifying associates through census, events, and place overlap
- 🔗 **Path** - Find the shortest relationship path between two people using BFS
- 🔬 **Analyze** - Research gap analysis: evidence gaps, quality issues, chronological inconsistencies, and suggestions
- ⚖️ **Proof** - Compile evidence for a research question into a structured proof summary following the Genealogical Proof Standard (GPS)
- ⚖️ **Evidence** - Lay out every assertion for one person+property side-by-side, grouped by value, to weigh conflicting evidence
- 📋 **Census Import** - Generate GLX entities from structured census templates with person matching, assertions, and dry-run preview
- 🔗 **Link** - Create a FamilySearch citation (and repository/source scaffolding) from an ARK URL, offline
- ➕ **Add** - Create person, place, event, repository, source, citation, relationship, or assertion entities from CLI flags with vocabulary and reference validation
- 🔄 **Migrate** - Convert deprecated person properties to birth/death events
- 🖥️ **Serve** - Run a local web server with a browser-based read-only viewer (dashboard, person profiles, family tree, sources)
- ⚡ **Cache** - Build a binary archive cache (`.glx/cache.bin`) so repeated commands skip the YAML parse; transparently used by read commands, with git + filesystem staleness detection

## Development

Build, test, and contribution workflow — prerequisites, `make` targets, DCO sign-off, and the pull request process — are documented in the [Contributing Guide](../CONTRIBUTING.md).

## Related Documentation

- [GENEALOGIX Specification](../specification/README.md)
- [JSON Schemas](../specification/schema/README.md)
- [Examples](../docs/examples/README.md)
- [Test Data Documentation](https://github.com/genealogix/glx/blob/main/glx/testdata/README.md)
- [CLI Command Reference](../docs/cli/index.md)

## License

Apache License 2.0 - See [LICENSE](https://github.com/genealogix/glx/blob/main/LICENSE) for details.

## Support

- 📖 [Specification](../specification/README.md)
- 💡 [Examples](../docs/examples/README.md)
- 🐛 [Issue Tracker](https://github.com/genealogix/glx/issues)
- 💬 [Discussions](https://github.com/genealogix/glx/discussions)
