---
description: Review example GLX archives for compliance with specification and code
---

You are tasked with comprehensively reviewing the example GLX archives in `docs/examples/` for compliance with the GLX specification and implementation.

## Source of Truth Flow

**IMPORTANT**: Examples are derived from the specification and must be valid according to the code:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
                                    ↓
                              Example Archives
                            (docs/examples/*.glx)
```

**This means:**
- Examples must be **valid YAML** that the GLX CLI can process
- Examples must follow **schema definitions** in specification/schema/v1/
- Examples must use **valid vocabulary terms** from specification/5-standard-vocabularies/
- Examples should demonstrate **best practices** from the specification

## Task

Perform a comprehensive review of all example archives using both:

1. **CLI Validation** - Automated checks using `glx validate`
2. **Manual Review** - Human-readable analysis of content quality and compliance

## Example Directories to Review

```
docs/examples/
├── minimal/                # Bare minimum valid archive
├── basic-family/           # Simple family relationships
├── complete-family/        # Full-featured example with all entity types
├── single-file/            # Single-file archive format
├── assertion-workflow/     # Assertion workflow demonstration
├── temporal-properties/    # Temporal property demonstrations
└── participant-assertions/ # Assertion participant patterns
```

## Step 1: CLI Validation

Run the GLX CLI validator on all example archives:

```bash
# Validate all examples
./bin/glx validate docs/examples/

# Also validate each directory individually for detailed output
./bin/glx validate docs/examples/minimal/
./bin/glx validate docs/examples/basic-family/
./bin/glx validate docs/examples/complete-family/
./bin/glx validate docs/examples/single-file/
./bin/glx validate docs/examples/assertion-workflow/
./bin/glx validate docs/examples/temporal-properties/
./bin/glx validate docs/examples/participant-assertions/
```

**Record all validation errors and warnings.** These are machine-verified issues that must be fixed.

## Step 2: Manual Review

For each example archive, manually verify:

### 2.1 Entity Structure Compliance

Compare each entity against its specification in `specification/4-entity-types/`:

- **Required fields**: Are all required fields present?
- **Field types**: Do field values match expected types (string, array, etc.)?
- **Field names**: Do YAML keys match specification field names exactly?
- **Nested structures**: Are participant objects, properties, etc. correctly structured?

### 2.2 Vocabulary Usage

Compare vocabulary terms against `specification/5-standard-vocabularies/`:

- **Event types**: Are event type values from event-types.glx?
- **Relationship types**: Are relationship types from relationship-types.glx?
- **Place types**: Are place types from place-types.glx?
- **Participant roles**: Are participant roles from participant-roles.glx?
- **Confidence levels**: Are confidence values from confidence-levels.glx?
- **Custom vocabularies**: Are any custom vocabulary terms properly defined?

### 2.3 Cross-Reference Integrity

Check that all entity references are valid:

- **Person references**: Do referenced person IDs exist in the archive?
- **Event references**: Do referenced event IDs exist?
- **Place references**: Do referenced place IDs exist?
- **Source/Citation references**: Do source and citation IDs exist?
- **No dangling references**: All `*_id` and `*_ids` fields point to existing entities

### 2.4 Date Format Compliance

Verify all dates follow the GLX date format specification:

- ISO 8601 format (YYYY-MM-DD, YYYY-MM, YYYY)
- Valid date ranges (if using `start_date`/`end_date`)
- Approximate dates use proper notation
- No invalid date values (e.g., month 13, day 32)

### 2.5 Best Practices

Check that examples demonstrate good GLX practices:

- **Meaningful IDs**: IDs should be descriptive (e.g., `person-john-smith` not `p1`)
- **Complete examples**: Each example should demonstrate its intended feature
- **Realistic data**: Names, dates, and places should be realistic
- **Proper notes**: Notes field usage when appropriate
- **Properties usage**: Properties field for additional data

### 2.6 README Accuracy

For each example directory, verify its README.md:

- **Accurately describes contents**: Does the README match what's in the archive?
- **All entities listed**: Are all entity files mentioned?
- **Features described**: Does it explain what the example demonstrates?
- **No stale documentation**: README isn't out of date with actual files

## Step 3: Specification Alignment

Cross-check examples against specification documentation:

### 3.1 Read Specification Entity Types

Read these specification files:
- `specification/4-entity-types/person.md`
- `specification/4-entity-types/event.md`
- `specification/4-entity-types/relationship.md`
- `specification/4-entity-types/place.md`
- `specification/4-entity-types/source.md`
- `specification/4-entity-types/citation.md`
- `specification/4-entity-types/repository.md`
- `specification/4-entity-types/media.md`
- `specification/4-entity-types/assertion.md`

### 3.2 Compare Examples to Specification

For each entity type appearing in examples:

1. List all fields used in the example
2. Compare against specification's field table
3. Flag any fields in example but not in specification (drift)
4. Flag any required fields in specification but missing from example

### 3.3 Check Example Patterns

Verify examples match patterns described in specification:

- **Single-file format**: Matches specification/3-archive-organization.md
- **Multi-file format**: Follows directory structure from specification
- **Vocabulary embedding**: Vocabularies are properly linked or embedded

## Step 4: Code Alignment

Verify examples would serialize/deserialize correctly with the Go code:

### 4.1 Read Type Definitions

Read `glx/lib/types.go` and check that example entities would map correctly to Go structs:

- Field names match YAML tags in Go structs
- Field types are compatible
- Required fields (non-omitempty) are present

### 4.2 Check for Type Mismatches

Look for potential unmarshaling issues:

- Strings where arrays expected (or vice versa)
- Nested objects where primitives expected
- Missing required fields that would fail validation

## Output Format

### CLI Validation Results

```
## CLI Validation Results

### docs/examples/minimal/
✅ Passed - No validation errors

### docs/examples/complete-family/
⚠️ Issues found:
- persons/person-john-smith.glx: Missing required field 'properties'
- events/event-births.glx: Invalid date format on line 12

[Continue for all directories]
```

### Manual Review Results

```
## Manual Review Results

### docs/examples/minimal/

#### Entity Compliance
✅ Person entities follow specification
⚠️ Event entities - Issue: `location` field used instead of `place`

#### Vocabulary Usage
✅ All event types from standard vocabulary
⚠️ Relationship type `married` should be `spouse` per vocabulary

#### Cross-References
✅ All person references valid
⚠️ Event `event-birth-john` references `place-london` which doesn't exist

#### Date Formats
✅ All dates use valid ISO 8601 format

#### Best Practices
✅ Meaningful entity IDs used
⚠️ Missing notes on complex relationships

#### README Accuracy
✅ README accurately describes contents

[Continue for all directories]
```

### Specification Alignment Results

```
## Specification Alignment

### Person Entity
✅ Example fields match specification

### Event Entity
⚠️ Drift detected:
- Example uses `location` but specification defines `place`
- Example missing `end_date` which is a valid optional field

[Continue for all entity types]
```

### Code Alignment Results

```
## Code Alignment

### Person struct (lib/types.go)
✅ Examples would deserialize correctly

### Event struct (lib/types.go)
⚠️ Example field `location` doesn't exist in Go struct (yaml:"place")
```

## Summary Report

At the end, provide:

### Statistics
- Total example files reviewed: X
- CLI validation errors: X
- Manual review issues: X
- Specification alignment issues: X
- Code alignment issues: X

### Issues by Severity

**Critical** 🔴 (blocks usage)
- List critical issues

**Major** 🟡 (significant problems)
- List major issues

**Minor** 🔵 (improvements needed)
- List minor issues

### Top Priority Fixes

List the 3-5 most important issues to fix first, considering:
1. CLI validation errors (these break examples entirely)
2. Required field issues (examples won't validate)
3. Cross-reference errors (broken relationships)
4. Vocabulary mismatches (incorrect terminology)
5. Documentation drift (README inaccuracy)

### Recommendations

Concrete action items:
- Fix [specific file] to [specific change]
- Update [vocabulary term] to match standard vocabulary
- Add [missing required field] to [entity files]
- Update README in [directory] to reflect actual contents

## Important Notes

- **CLI validation errors are highest priority** - if `glx validate` fails, the example is broken
- **Cross-reference errors** cause cascading issues and should be fixed early
- **Vocabulary compliance** ensures examples teach correct usage
- **README accuracy** is important for users learning from examples
- Focus on issues that would confuse users learning GLX from examples
- Examples should be exemplary - they're teaching material

## Cross-Reference with Known Issues

Before finalizing your report, check if issues are already tracked in:
- `todo.md` - Project-wide known issues
- GitHub issues (if accessible)

Exclude already-tracked issues from the report to avoid duplicates.
