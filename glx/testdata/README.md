# GENEALOGIX Test Data

This directory contains test fixtures for the GLX CLI tool validation tests.

## Directory Structure

```
testdata/
├── valid/         # Files that MUST pass validation
│   ├── Single-entity files (person-minimal.glx, etc.)
│   └── Multi-entity archives (archive-*.glx)
└── invalid/       # Files that MUST fail validation
    ├── Single-entity files with errors
    └── Multi-entity archives with errors
```

## Valid Test Files

### Single-Entity Tests
Files testing individual entity types with all required fields:
- `person-minimal.glx` - Minimal person with only required fields
- `person-complete.glx` - Person with all optional fields
- `event-minimal.glx` - Basic event
- `event-complete.glx` - Event with participants and descriptions
- `place-minimal.glx` - Basic place
- `place-hierarchy.glx` - Place with parent relationship
- `relationship-marriage.glx` - Marriage relationship
- `relationship-parent-child.glx` - Parent-child relationship
- `source-complete.glx` - Source with publication details
- `citation-minimal.glx` - Basic citation
- `citation-quality2.glx` - Citation with quality rating
- `citation-quality3.glx` - High-quality citation
- `repository-minimal.glx` - Basic repository
- `repository-archive.glx` - Archive repository with details
- `assertion-minimal.glx` - Basic assertion
- `assertion-confidence-medium.glx` - Assertion with confidence
- `assertion-evidence-chain.glx` - Assertion with citations
- `media-photo.glx` - Media reference
- `event-occupation.glx` - Occupation event

### Multi-Entity Archive Tests
Complete GLX archives with multiple entity types demonstrating real-world usage:

#### `archive-small-family.glx`
Complete family with cross-references:
- 3 persons (John, Mary, Jane)
- 2 relationships (marriage, parent-child)
- 2 events (birth, marriage)
- 3 places (England, Yorkshire, Leeds) with hierarchy
- 1 source (parish register)
- 1 repository (Leeds Library)
- 1 citation with quality rating
- 1 assertion backed by citation

Tests: Entity references, place hierarchies, event participants, citation quality.

#### `archive-evidence-chain.glx`
Complete evidence chain from repository through assertions:
- Repository → Source → Citation → Assertion
- Multiple assertions from same citation
- Quality ratings and transcriptions
- Repository details with location/website

Tests: Evidence chain integrity, citation quality, assertion confidence.

#### `archive-with-media.glx`
Archive demonstrating media references:
- Person with associated media
- Photo and document media types
- Media linked to citations
- Both local (file://) and remote (https://) URIs

Tests: Media URI handling, media types, media-citation links.

## Invalid Test Files

### Single-Entity Validation Errors
- `person-missing-id.glx` - Entity has 'id' field (should use map key)
- `person-bad-id-format.glx` - Invalid ID format (not alphanumeric/hyphens)
- `event-missing-type.glx` - Event without required type
- `event-missing-place.glx` - Event without place reference
- `place-missing-name.glx` - Place without required name
- `source-missing-title.glx` - Source without required title
- `citation-missing-source.glx` - Citation without source reference
- `citation-invalid-quality.glx` - Citation with invalid quality rating
- `repository-missing-name.glx` - Repository without required name
- `relationship-missing-type.glx` - Relationship without type
- `assertion-missing-property.glx` - Assertion missing required fields
- `assertion-invalid-confidence.glx` - Invalid confidence level
- `media-invalid-type.glx` - Media with invalid type

### Multi-Entity Validation Errors

#### `archive-broken-references.glx`
Multiple cross-reference errors:
- Relationship references non-existent person
- Event references non-existent place
- Event participant references non-existent person
- Citation references non-existent source
- Assertion references non-existent entity
- Assertion references non-existent citation

Tests: Cross-reference validation across all entity types.

#### `archive-duplicate-ids.glx`
Duplicate entity ID errors:
- Multiple persons with same ID
- Multiple places with same ID

Tests: Duplicate ID detection.

#### `archive-missing-fields.glx`
Multiple entities with missing required fields:
- Event without type
- Place without name
- Source without title
- Relationship without type
- Repository without name

Tests: Required field validation across entity types.

## Test Usage

These files are used by:
- `validate_test.go::TestValidateGLXFile_ValidTestFiles` - Validates all files in `valid/`
- `validate_test.go::TestValidateGLXFile_InvalidTestFiles` - Ensures all files in `invalid/` fail
- `validate_test.go::TestValidateRepositoryReferences` - Tests cross-reference validation

## Adding New Test Files

### Valid Test Files
1. Create `.glx` file in `testdata/valid/`
2. Ensure it follows GLX specification
3. Run `glx validate testdata/valid/your-file.glx` to verify
4. File will automatically be tested by `TestValidateGLXFile_ValidTestFiles`

### Invalid Test Files
1. Create `.glx` file in `testdata/invalid/`
2. Add comment at top explaining what makes it invalid
3. Ensure it fails validation: `glx validate testdata/invalid/your-file.glx` (should error)
4. File will automatically be tested by `TestValidateGLXFile_InvalidTestFiles`

### Naming Convention
- Single-entity: `{entity-type}-{scenario}.glx`
  - Examples: `person-minimal.glx`, `event-missing-type.glx`
- Multi-entity: `archive-{scenario}.glx`
  - Examples: `archive-small-family.glx`, `archive-broken-references.glx`

## Related Tests

- **Examples Tests** (`examples_test.go`) - Validates `docs/examples/` directory
- **Validator Tests** (`validate_test.go`) - Unit tests for validation logic
- **Main Tests** (`main_test.go`) - Tests for CLI commands

## Coverage

These test files contribute to:
- Entity validation coverage
- Cross-reference validation coverage
- Error detection coverage
- Multi-entity archive validation
- Real-world use case validation

Current coverage: ~69% of validation code paths

