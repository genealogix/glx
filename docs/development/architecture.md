# Architecture Guide

This guide explains the overall architecture of the GENEALOGIX specification and implementation.

## System Overview

GENEALOGIX is a family history archive format with three main components:

1. **Specification**: Defines the data model and file format
2. **Schemas**: JSON Schema validation for all entity types
3. **CLI Tool**: Validation and management utilities

```
┌─────────────────────────────────────────────────────────────────┐
│                        GENEALOGIX System                        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│             │    │             │    │             │
│    Spec     │ ─> │  Schemas    │ -> │  CLI Tool   │
│             │    │             │    │             │
│ • Entity    │    │ • JSON      │    │ • glx       │
│   Types     │    │   Schema    │    │   validate  │
│ • File      │    │ • Validation│    │ • glx init  │
│   Format    │    │ • Tests     │    │ • glx check │
│ • Git       │    │             │    │   schemas   │
│   Workflow  │    │             │    │             │
└─────────────┘    └─────────────┘    └─────────────┘
         │                   │                   │
         │                   │                   │
         ▼                   ▼                   ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│             │    │             │    │             │
│ Examples    │    │ Test Suite  │    │ Validation  │
│             │    │             │    │             │
│ • Complete  │    │ • Valid     │    │ • Schema    │
│   Family    │    │   Tests     │    │   Check     │
│ • Minimal   │    │ • Invalid   │    │ • Reference │
│ • Basic     │    │   Tests     │    │   Integrity │
│ • Migration │    │ • Edge      │    │ • File      │
│             │    │   Cases     │    │   Structure │
└─────────────┘    └─────────────┘    └─────────────┘
```

## Specification Architecture

### Entity Type System

GENEALOGIX defines 9 core entity types:

| Entity Type | Directory | Purpose | Key Fields |
|-------------|-----------|---------|------------|
| Person | `persons/` | Individual people | name, birth, death |
| Relationship | `relationships/` | Family connections | participants, type |
| Event | `events/` | Life events | type, date, place |
| Place | `places/` | Geographic locations | name, hierarchy, coordinates |
| Source | `sources/` | Original materials | title, creator, repository |
| Citation | `citations/` | Evidence references | source, locator, quality |
| Repository | `repositories/` | Archives/libraries | name, contact, holdings |
| Assertion | `assertions/` | Evidence-based claims | subject, claim, citations |
| Media | `media/` | Supporting files | format, description |

### File Format Design

**YAML Structure:**
```yaml
# Standard entity header
id: person-a1b2c3d4        # Unique identifier
version: "1.0"            # Schema version
type: person              # Entity type

# Entity-specific content
name:
  given: John
  surname: Smith
  display: John Smith

# Evidence and metadata
citations: [citation-123] # Evidence references
notes: "Research notes..." # Additional context
```

**ID System:**
- **Format**: `{type}-{8hex}` (e.g., `person-a1b2c3d4`)
- **Generation**: Random 8-character hex strings
- **Validation**: Pattern matching and uniqueness
- **References**: Cross-entity linking

## Schema Architecture

### JSON Schema Hierarchy

**Meta Schema:**
```json
schema/meta/schema.schema.json
├── Defines schema structure
├── Validates all entity schemas
└── Ensures consistency
```

**Entity Schemas:**
```json
schema/v1/
├── person.schema.json      # Person entity
├── event.schema.json       # Event entity
├── place.schema.json       # Place entity
├── source.schema.json      # Source entity
├── citation.schema.json    # Citation entity
├── repository.schema.json  # Repository entity
├── assertion.schema.json   # Assertion entity
├── relationship.schema.json # Relationship entity
└── media.schema.json       # Media entity
```

### Schema Features

**Common Schema Elements:**
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://schema.genealogix.org/v1/person",
  "title": "Person",
  "description": "An individual in the family archive",
  "type": "object",
  "required": ["id", "version", "type"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^person-[a-f0-9]{8}$"
    },
    "version": {
      "type": "string",
      "pattern": "^\\d+\\.\\d+$"
    },
    "type": {
      "type": "string",
      "enum": ["person"]
    }
  }
}
```

**Validation Features:**
- **Pattern matching**: ID format validation
- **Required fields**: Ensures completeness
- **Type checking**: Data type validation
- **Reference validation**: Cross-entity integrity
- **Enum validation**: Controlled vocabularies

## CLI Tool Architecture

### Command Structure

**Main Commands:**
```bash
glx init          # Initialize new repository
glx validate      # Validate .glx files
glx check-schemas # Validate schema files
```

**Command Implementation:**
```go
// glx/main.go
func main() {
    cmd := os.Args[1]
    switch cmd {
    case "init":
        runInit()
    case "validate":
        runValidate(os.Args[2:])
    case "check-schemas":
        runCheckSchemas()
    }
}
```

### Validation Pipeline

**Multi-stage validation:**

1. **File Discovery**
   - Scan directories for `.glx` files
   - Filter by entity type directories
   - Handle glob patterns and paths

2. **Syntax Validation**
   - YAML parsing
   - Basic structure checking
   - Schema compliance

3. **Reference Validation**
   - Cross-entity reference checking
   - ID existence validation
   - Circular reference detection

4. **Business Rule Validation**
   - Date logic (birth before death)
   - Place hierarchy validation
   - Evidence chain completeness

### Repository Initialization

**Directory Structure Creation:**
```go
func runInit() error {
    dirs := []string{
        "persons", "relationships", "events", "places",
        "sources", "citations", "repositories",
        "assertions", "media",
    }

    for _, dir := range dirs {
        os.MkdirAll(dir, 0755)
    }

    // Create .gitignore and README.md
    writeGitignore()
    writeReadme()
}
```

## Test Suite Architecture

### Test Organization

**Valid Tests:**
```
test-suite/valid/
├── person-minimal.glx      # Minimal valid person
├── person-complete.glx     # All optional fields
├── event-standard.glx      # Standard event
├── place-hierarchy.glx     # Place with parent
├── citation-quality.glx    # Citation with quality
└── assertion-chain.glx     # Complete evidence chain
```

**Invalid Tests:**
```
test-suite/invalid/
├── person-missing-id.glx   # Missing required field
├── person-bad-id.glx       # Invalid ID format
├── event-no-place.glx      # Missing required place
├── citation-no-source.glx  # Missing source reference
└── place-circular.glx      # Circular reference
```

### Test Execution

**Test Runner:**
```bash
# test-suite/run-tests.sh
#!/bin/bash
VALIDATOR=${VALIDATOR:-glx}
TEST_DIR=$1

for test_file in $TEST_DIR/*.glx; do
    echo "Testing $test_file"
    $VALIDATOR validate "$test_file"

    # Check expected result (valid/invalid)
    # Report pass/fail
done
```

**Test File Format:**
```yaml
# TEST: person-minimal
# EXPECT: valid
# DESCRIPTION: Minimal valid person with only required fields

id: person-a1b2c3d4
version: "1.0"
type: person
name:
  given: John
  surname: Smith
```

## Evidence Chain Architecture

### Data Flow

**Evidence Flow:**
```
Repository → Source → Citation → Assertion → Entity
     ↓         ↓         ↓         ↓         ↓
Physical  Original  Specific  Evidence-  Person/
Archive   Material  Reference backed    Event/
         (Document) (Page/Entry) Claim    Place
```

**Implementation:**
```yaml
# Complete evidence chain
repositories/repo-library.glx:
  name: "Leeds Library"

sources/source-census.glx:
  title: "1851 Census"
  repository: repository-leeds-library

citations/citation-schedule.glx:
  source: source-census
  locator: "HO107/2319/234/23"
  quality: 2

assertions/assertion-occupation.glx:
  subject: person-john-smith
  claim: occupation
  value: blacksmith
  citations: [citation-schedule]
```

## Git Integration Architecture

### Repository Structure

**Standard Layout:**
```
family-archive/
├── .git/
├── .gitignore
├── README.md (auto-generated)
├── persons/
├── relationships/
├── events/
├── places/
├── sources/
├── citations/
├── repositories/
├── assertions/
└── media/
```

### Workflow Integration

**Branching Strategy:**
```
main (stable)
├── feature/add-evidence-quality
├── research/1851-census
├── fix/validation-bug
└── docs/update-specification
```

**Merge Strategy:**
- **Feature branches**: For new functionality
- **Research branches**: For evidence investigation
- **Hotfix branches**: For urgent fixes
- **Release branches**: For stable releases

## Validation Architecture

### Multi-Layer Validation

**1. File System Validation:**
- Directory structure correctness
- File extensions (`.glx`)
- ID format compliance
- Entity type organization

**2. YAML Validation:**
- Syntax correctness
- Schema compliance
- Required field presence
- Data type validation

**3. Reference Validation:**
- Cross-entity reference integrity
- ID existence checking
- Circular reference detection
- Required relationship validation

**4. Business Logic Validation:**
- Date consistency (birth before death)
- Place hierarchy validity
- Evidence chain completeness
- Quality rating appropriateness

### Error Reporting

**Structured Error Messages:**
```go
type ValidationError struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Column   int    `json:"column"`
    Message  string `json:"message"`
    Code     string `json:"code"`
    Severity string `json:"severity"`
}
```

**Error Categories:**
- **Syntax**: YAML parsing errors
- **Schema**: JSON Schema violations
- **Reference**: Broken entity references
- **Logic**: Business rule violations
- **Structure**: File organization issues

## Extension Architecture

### Schema Extensibility

**Version Management:**
```json
{
  "version": "1.0",
  "extends": "1.0-base",
  "additions": {
    "properties": {
      "custom_field": {
        "type": "string",
        "description": "Custom extension field"
      }
    }
  }
}
```

### Custom Entity Types

**Future Extension:**
```yaml
# Custom entity types (v2.0+)
custom-entities/
├── dna-matches/
│   └── dna-match-a1b2c3d4.glx
├── military-records/
│   └── military-service-b2c3d4e5.glx
└── medical-history/
    └── medical-condition-c3d4e5f6.glx
```

## Performance Architecture

### Validation Performance

**Optimization Strategies:**
- **Parallel validation**: Multiple files simultaneously
- **Incremental validation**: Only changed files
- **Caching**: Schema compilation results
- **Streaming**: Large file processing

**Performance Metrics:**
- **Validation speed**: Files per second
- **Memory usage**: Peak memory consumption
- **Schema compile time**: JSON Schema compilation
- **Reference resolution**: Cross-file dependency checking

### Large Archive Support

**Scaling Considerations:**
- **File count**: Support for 10,000+ files
- **Archive size**: Multi-gigabyte archives
- **Validation time**: Sub-second for small changes
- **Git performance**: Efficient with large repositories

## Security Architecture

### Data Validation
- **Input sanitization**: All user inputs validated
- **Schema enforcement**: Strict JSON Schema compliance
- **Reference validation**: Prevent broken references
- **File system safety**: Directory traversal protection

### Git Integration Safety
- **Safe file operations**: No arbitrary command execution
- **Path validation**: Canonical path checking
- **Repository integrity**: Git state preservation
- **Error handling**: Graceful failure modes

## Migration Architecture

### From GEDCOM

**Mapping Strategy:**
```
GEDCOM INDI → GENEALOGIX Person
GEDCOM FAM → GENEALOGIX Relationship
GEDCOM EVEN → GENEALOGIX Event
GEDCOM PLAC → GENEALOGIX Place
GEDCOM SOUR → GENEALOGIX Source
```

**Quality Translation:**
```
GEDCOM QUAY 0 → GENEALOGIX Quality 0
GEDCOM QUAY 1 → GENEALOGIX Quality 1
GEDCOM QUAY 2 → GENEALOGIX Quality 2
GEDCOM QUAY 3 → GENEALOGIX Quality 3
```

### Data Preservation

**Migration Guarantees:**
- **No data loss**: All GEDCOM data preserved
- **Evidence tracking**: Source information maintained
- **Quality assessment**: GEDCOM QUAY ratings converted
- **Structure mapping**: Hierarchical relationships preserved

## Testing Architecture

### Test Categories

**1. Unit Tests:**
- Individual function testing
- Schema validation logic
- CLI command testing
- Error handling verification

**2. Integration Tests:**
- End-to-end validation
- Cross-entity reference checking
- Git workflow integration
- Example validation

**3. Performance Tests:**
- Large archive validation
- Memory usage testing
- Speed benchmarking
- Scalability testing

### Test Data Management

**Test Data Sources:**
- **Valid tests**: Minimal and complete examples
- **Invalid tests**: Common error patterns
- **Edge cases**: Boundary condition testing
- **Regression tests**: Historical bug patterns

## Documentation Architecture

### Documentation Structure

**User Documentation:**
```
docs/
├── quickstart.md           # 5-minute tutorial
├── guides/
│   ├── best-practices.md   # Workflow guidelines
│   ├── common-pitfalls.md  # Error avoidance
│   ├── migration-from-gedcom.md # Import guide
│   └── glossary.md         # Terminology
└── diagrams/
    ├── entity-relationship.md # Entity connections
    ├── evidence-chain.md     # Evidence flow
    └── git-workflow.md       # Collaboration
```

**Developer Documentation:**
```
docs/development/
├── setup.md              # Development environment
├── architecture.md       # System architecture
├── testing-guide.md      # Testing framework
└── schema-development.md # Schema conventions
```

### Cross-Reference System

**Internal Linking:**
- Specification sections reference examples
- Examples link to relevant specifications
- CLI documentation references schema definitions
- Error messages link to troubleshooting guides

## Future Architecture

### Version 2.0 Considerations

**Potential Enhancements:**
- **Custom entity types**: User-defined entity schemas
- **Advanced relationships**: Complex family structures
- **Temporal modeling**: Time-aware data
- **Collaboration features**: Real-time editing support
- **API interfaces**: REST and GraphQL APIs
- **Plugin system**: Extension mechanisms

### Backwards Compatibility

**Compatibility Strategy:**
- **Schema versioning**: Clear version progression
- **Migration tools**: Automated upgrade paths
- **Graceful degradation**: Support for older formats
- **Feature flags**: Optional new functionality

This architecture provides a solid foundation for reliable, extensible family history data management while maintaining simplicity and performance.
