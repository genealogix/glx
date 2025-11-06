# GENEALOGIX CLI Tool (glx)

[![Go Reference](https://pkg.go.dev/badge/github.com/genealogix/spec/glx.svg)](https://pkg.go.dev/github.com/genealogix/spec/glx)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](../LICENSE)

The official command-line tool for working with GENEALOGIX family archives. Provides validation, repository initialization, and schema checking capabilities.

## Features

- **📋 Archive Validation**: Comprehensive validation of .glx files
- **🔧 Repository Setup**: Initialize new genealogy repositories
- **✅ Schema Checking**: Validate JSON Schema files
- **🌳 Git Integration**: Works seamlessly with Git workflows
- **🔍 Reference Integrity**: Cross-entity validation and error reporting

## Installation

### From Source (Recommended for Development)

```bash
# Clone the repository
git clone https://github.com/genealogix/spec.git
cd spec

# Install CLI tool
go install ./glx

# Verify installation
glx --help
```

### Using Go Install

```bash
# Install latest version
go install github.com/genealogix/spec/glx@latest

# Verify installation
glx --help
```

### Binary Download (Future Release)

```bash
# Download from releases page
wget https://github.com/genealogix/spec/releases/download/v1.0.0/glx-linux-amd64
chmod +x glx-linux-amd64
sudo mv glx-linux-amd64 /usr/local/bin/glx
```

## Quick Start

### Initialize a New Repository

```bash
# Create directory for your family archive
mkdir my-family-history
cd my-family-history

# Initialize with standard structure
glx init

# Your repository now has:
# persons/     - Individual records
# events/      - Life events
# places/      - Geographic locations
# sources/     - Original materials
# citations/   - Evidence references
# repositories/ - Archives and libraries
# assertions/  - Evidence-based conclusions
# media/       - Supporting files
# .gitignore   - Git ignore rules
# README.md    - Repository documentation
```

### Validate Your Archive

```bash
# Validate entire archive
glx validate

# Validate specific directories
glx validate persons/
glx validate events/ places/

# Validate specific files
glx validate persons/person-john-smith.glx
glx validate events/event-birth.glx

# Validate with glob patterns
glx validate persons/*.glx
glx validate **/*.glx
```

### Check Schema Files

```bash
# Validate all JSON schemas
glx check-schemas

# This verifies:
# - All required schema files exist
# - Schemas are valid JSON
# - Required metadata is present
# - Schema references are correct
```

## Commands

### `glx init`

Initialize a new GENEALOGIX repository with standard directory structure.

```bash
glx init [directory]

Options:
  directory    Target directory (default: current directory)

Examples:
  glx init                    # Initialize current directory
  glx init my-family-archive  # Initialize specific directory
  glx init ../family-project  # Initialize parent directory
```

**What it creates:**
- Directory structure for all entity types
- `.gitignore` file optimized for genealogy archives
- `README.md` with repository documentation
- Proper permissions for collaboration

### `glx validate`

Validate .glx files against JSON schemas and business rules.

```bash
glx validate [paths...]

Arguments:
  paths    Files or directories to validate (default: current directory)

Options:
  -v, --verbose    Show detailed validation output
  -q, --quiet      Suppress non-error output
  -s, --strict     Fail on warnings as well as errors

Examples:
  glx validate                    # Validate current directory
  glx validate persons/           # Validate persons directory
  glx validate *.glx              # Validate specific files
  glx validate examples/          # Validate all examples
  glx validate --verbose          # Show detailed output
```

**Validation checks:**
- ✅ YAML syntax correctness
- ✅ JSON Schema compliance
- ✅ Cross-entity reference integrity
- ✅ ID format validation
- ✅ Business rule compliance
- ✅ Evidence chain completeness

### `glx check-schemas`

Validate JSON Schema files and their metadata.

```bash
glx check-schemas

Examples:
  glx check-schemas              # Check all schemas
```

**Schema validation includes:**
- ✅ Schema file existence (all 9 entity types)
- ✅ JSON syntax correctness
- ✅ Required metadata presence ($id, $schema, title)
- ✅ Schema reference validity

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GLX_SCHEMA_DIR` | Directory containing schema files | `specification/schema/v1/` |
| `GLX_STRICT_MODE` | Enable strict validation | `false` |
| `GLX_CACHE_SCHEMAS` | Cache compiled schemas | `true` |
| `GLX_MAX_ERRORS` | Maximum errors to report | `100` |

### Configuration File (Future)

```yaml
# .glxconfig.yaml
strict_mode: true
schema_dir: "custom-schemas/"
cache_schemas: true
max_errors: 50
custom_validators:
  - "plugins/custom-rules.so"
```

## Validation Rules

### File Structure Validation

**Directory Structure:**
```
Valid directories:
├── persons/
├── relationships/
├── events/
├── places/
├── sources/
├── citations/
├── repositories/
├── assertions/
└── media/

Invalid:
├── people/      # Wrong name
├── events       # Missing trailing slash
├── Events/      # Wrong case
└── mixed/       # Contains wrong entity types
```

**File Extensions:**
- ✅ Valid: `person-a1b2c3d4.glx`
- ❌ Invalid: `person-a1b2c3d4.yaml`, `person-a1b2c3d4.json`

### ID Format Validation

**Pattern:** `{type}-{8hex}`
- `person-` + 8 hex characters
- `event-` + 8 hex characters
- `place-` + 8 hex characters
- etc.

**Valid IDs:**
- ✅ `person-a1b2c3d4`
- ✅ `event-1a2b3c4d`
- ✅ `place-2b3c4d5e`

**Invalid IDs:**
- ❌ `john-smith` (no prefix)
- ❌ `person-abc` (too short)
- ❌ `person-ABCDEFGH` (uppercase)
- ❌ `person-a1b2c3d4e` (too long)

### Reference Validation

**Cross-entity references must exist:**
```yaml
# ✅ Valid references
places:
  place-leeds:
    version: "1.0"
    name: "Leeds"

persons:
  person-john:
    version: "1.0"
    concluded_identity:
      primary_name: "John"

events:
  event-wedding:
    version: "1.0"
    type: marriage
    place: place-leeds  # place-leeds exists above
    participants:
      - person: person-john  # person-john exists above

# ❌ Invalid references
events:
  event-bad:
    version: "1.0"
    type: marriage
    place: place-missing  # place-missing doesn't exist!
```

### Evidence Chain Validation

**Complete evidence chains required:**
```yaml
# ✅ Complete chain
sources:
  source-cert:
    version: "1.0"
    title: "Birth Certificate"

citations:
  citation-cert:
    version: "1.0"
    source: source-cert

assertions:
  assertion-birth:
    version: "1.0"
    citations: [citation-cert]

# ❌ Broken chain
assertions:
  assertion-broken:
    version: "1.0"
    citations: [citation-missing]  # Citation doesn't exist!
```

## Error Messages

### Validation Error Format

```bash
# Standard error format
ERROR: file.glx:line:column: error_code: message
  details and suggestions

# Example
ERROR: persons/person-john.glx:15:23: REF_NOT_FOUND: place 'place-missing' not found
  Did you mean: place-leeds, place-yorkshire, place-england?
  Create the missing place or fix the reference.
```

### Common Error Codes

| Code | Description | Example |
|------|-------------|---------|
| `SYNTAX_ERROR` | YAML parsing failed | Invalid indentation |
| `SCHEMA_VIOLATION` | JSON Schema violation | Missing required field |
| `REF_NOT_FOUND` | Broken entity reference | Nonexistent ID |
| `ID_INVALID` | Invalid ID format | Wrong pattern |
| `DATE_INVALID` | Invalid date format | Wrong date syntax |
| `QUALITY_INVALID` | Invalid quality rating | Out of 0-3 range |

## Performance

### Validation Speed

**Typical performance:**
- Small archive (10-50 files): < 1 second
- Medium archive (100-500 files): 1-3 seconds
- Large archive (1000+ files): 5-15 seconds

**Optimization tips:**
```bash
# Validate only changed files
glx validate $(git diff --name-only)

# Use caching
export GLX_CACHE_SCHEMAS=true

# Parallel validation (future)
glx validate --parallel
```

### Memory Usage

**Memory requirements:**
- Base memory: ~10MB
- Per file: ~1KB (including schema compilation)
- Large archives: ~50MB for 1000+ files

## Integration

### Git Workflows

**Pre-commit validation:**
```bash
# .git/hooks/pre-commit
#!/bin/bash
glx validate
if [ $? -ne 0 ]; then
    echo "Validation failed. Fix errors before committing."
    exit 1
fi
```

**GitHub Actions:**
```yaml
# .github/workflows/validate.yml
- name: Validate Archive
  run: glx validate

- name: Schema Check
  run: glx check-schemas
```

### IDE Integration

**Visual Studio Code:**
```json
// settings.json
{
  "glx.validateOnSave": true,
  "glx.validateOnType": false,
  "glx.showValidationErrors": true
}
```

**Vim:**
```vim
" .vimrc
autocmd BufWritePost *.glx silent !glx validate <afile>
```

## Troubleshooting

### Common Issues

**"Command not found: glx"**
```bash
# Check Go installation
go version

# Reinstall CLI
go install github.com/genealogix/spec/glx@latest

# Check PATH
which glx
echo $PATH
```

**Validation errors:**
```bash
# Check YAML syntax
python3 -c "import yaml; yaml.safe_load(open('file.glx'))"

# Validate against schema
glx validate --verbose file.glx

# Check file permissions
ls -la file.glx
```

**Schema compilation errors:**
```bash
# Check schema syntax
ajv compile -s specification/schema/v1/person.schema.json

# Validate schema files
glx check-schemas

# Update schema URLs
# Ensure GitHub URLs are accessible
```

### Debug Mode

**Enable debug output:**
```bash
# Set debug environment variable
export GLX_DEBUG=true

# Run with debug
glx validate --verbose

# Check debug logs
cat /tmp/glx-debug.log  # If available
```

### Getting Help

**Community support:**
- [GitHub Issues](https://github.com/genealogix/spec/issues) - Bug reports
- [GitHub Discussions](https://github.com/genealogix/spec/discussions) - Q&A
- [Documentation](../docs/) - Guides and tutorials

**Development:**
- [CLI Source Code](main.go) - Implementation details
- [Test Suite](../test-suite/) - Validation tests
- [Development Guide](../docs/development/) - Contributing guide

## Contributing

### Development Setup

```bash
# Clone repository
git clone https://github.com/genealogix/spec.git
cd spec

# Set up development environment
go mod download
go build -o bin/glx glx/main.go

# Run tests
go test ./glx/...

# Install locally
go install ./glx
```

### Adding New Commands

**Command structure:**
```go
func main() {
    switch os.Args[1] {
    case "new-command":
        runNewCommand()
    case "validate":
        runValidate(os.Args[2:])
    // ... existing commands
    }
}

func runNewCommand() error {
    // Implementation
    return nil
}
```

### Testing CLI Changes

```bash
# Test new functionality
go test ./glx/...

# Manual testing
glx --help
glx validate examples/complete-family/

# Integration testing
cd /tmp && glx init test-repo && cd test-repo && glx validate
```

## Changelog

### v1.0.0
- Initial release
- Basic validation functionality
- Repository initialization
- Schema checking

### v1.1.0 (Planned)
- Enhanced error reporting
- Performance optimizations
- Plugin system for custom validation
- Batch processing capabilities

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](../LICENSE) for details.

## Related Projects

- [GENEALOGIX Specification](../README.md) - Core specification
- [Examples](../docs/examples/) - Practical usage examples
- [Test Suite](tests/) - Validation tests
- [Schemas](../specification/schema/) - JSON Schema definitions

---

**Part of the GENEALOGIX ecosystem** • [Main Repository](https://github.com/genealogix/spec)
