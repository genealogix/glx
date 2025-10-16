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
- Minimal valid entities
- Complete entities with all optional fields
- Edge cases that should be accepted

### `invalid/`

Files that MUST fail validation:
- Missing required fields
- Invalid field types
- Malformed references
- Schema violations

## Test File Naming

```
{entity}-{scenario}.glx
```

Examples:
- `person-minimal.glx` - Minimal valid person
- `person-missing-id.glx` - Person without required id
- `person-bad-version.glx` - Person with invalid version format

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


