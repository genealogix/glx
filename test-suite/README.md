# GENEALOGIX Conformance Test Suite

This test suite validates implementations of the GENEALOGIX specification.

## Running Tests

```bash
# Run all tests
./run-tests.sh

# Run specific category
./run-tests.sh valid
./run-tests.sh invalid

# With specific implementation
./run-tests.sh --validator /path/to/glx
```

## Test Categories

### `valid/`

Files that MUST pass validation:
- **Minimal entities**: Only required fields populated
- **Complete entities**: All optional fields populated
- **Edge cases**: Boundary conditions and complex scenarios
- **Real-world examples**: Practical usage patterns
- **Evidence chains**: Complete citation and assertion relationships

### `invalid/`

Files that MUST fail validation:
- **Missing required fields**: Core fields like id, version, type
- **Invalid formats**: Wrong ID patterns, date formats, quality ratings
- **Broken references**: Nonexistent entity references
- **Schema violations**: Wrong field types, invalid values
- **Business rule violations**: Date inconsistencies, circular references

## Test File Naming

```
{entity}-{scenario}.glx
```

### Valid Test Examples:
- `person-minimal.glx` - Minimal valid person (required fields only)
- `person-complete.glx` - Person with all optional fields
- `event-complete.glx` - Event with participants and descriptions
- `place-hierarchy.glx` - Place with parent and coordinates
- `source-complete.glx` - Source with full publication details
- `citation-quality3.glx` - High-quality citation with transcription
- `assertion-evidence-chain.glx` - Assertion with complete evidence trail

### Invalid Test Examples:
- `person-missing-id.glx` - Person without required id
- `person-bad-id-format.glx` - Person with invalid ID format
- `person-invalid-version.glx` - Person with malformed version
- `event-missing-place.glx` - Event without required place reference
- `person-broken-reference.glx` - Person referencing nonexistent place
- `citation-invalid-quality.glx` - Citation with out-of-range quality

## Expected Results

Each test file includes a comment header:

```yaml
# TEST: person-minimal
# EXPECT: valid
# DESCRIPTION: Minimal valid person with only required fields
```

Or for invalid tests:

```yaml
# TEST: person-missing-id
# EXPECT: invalid
# ERROR: "id field is required"
# DESCRIPTION: Person entity must have an id field
```

## Implementing Conformance

To claim GENEALOGIX conformance, an implementation MUST:

1. Pass all tests in `valid/` category
2. Reject all tests in `invalid/` category
3. Report appropriate error messages for invalid files

## Contributing Tests

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines on adding tests.


