# Testing Guide

This guide explains the testing framework and how to write tests for GENEALOGIX implementations.

## Test Suite Overview

The GENEALOGIX test suite validates implementations against the specification:

```
glx/tests/
├── README.md           # Test suite documentation
├── run-tests.sh        # Test runner script
├── valid/             # Files that MUST pass validation
├── invalid/           # Files that MUST fail validation
└── fixtures/          # Test data and utilities
```

## Test Categories

### Valid Tests

**Purpose**: Ensure implementations accept correct GENEALOGIX files

**Location**: `glx/tests/valid/`

**Characteristics**:
- Minimal valid entities (only required fields)
- Complete entities (all optional fields populated)
- Edge cases that should be accepted
- Real-world examples

**Example**:
```yaml
# glx/tests/valid/person-minimal.glx
# TEST: person-minimal
# EXPECT: valid
# DESCRIPTION: Minimal valid person with only required fields

id: person-a1b2c3d4
version: "1.0"
type: person
name:
  given: John
  surname: Smith
  display: John Smith
```

### Invalid Tests

**Purpose**: Ensure implementations reject incorrect files with appropriate errors

**Location**: `glx/tests/invalid/`

**Characteristics**:
- Missing required fields
- Invalid field types or formats
- Malformed references
- Schema violations
- Business rule violations

**Example**:
```yaml
# glx/tests/invalid/person-missing-id.glx
# TEST: person-missing-id
# EXPECT: invalid
# ERROR: "id field is required"
# DESCRIPTION: Person entity must have an id field

version: "1.0"
type: person
name:
  given: John
  surname: Smith
# Missing required id field
```

## Test File Format

### Test Header

Each test file must start with a header comment:

```yaml
# TEST: descriptive-test-name
# EXPECT: valid|invalid
# ERROR: "expected error message"
# DESCRIPTION: What this test validates
```

**Header Fields**:

| Field | Required | Description |
|-------|----------|-------------|
| `TEST` | Yes | Unique test identifier |
| `EXPECT` | Yes | Expected result (valid or invalid) |
| `ERROR` | Invalid only | Expected error message pattern |
| `DESCRIPTION` | Yes | Human-readable description |

### Test Naming Conventions

**Valid Tests**:
- `person-minimal` - Minimal required fields
- `person-complete` - All optional fields
- `person-edge-case` - Boundary conditions
- `event-standard` - Typical event structure

**Invalid Tests**:
- `person-missing-id` - Missing required field
- `person-bad-format` - Invalid format
- `person-broken-ref` - Invalid reference
- `event-no-place` - Missing required relationship

## Running Tests

### Basic Test Execution

**Run all tests**:
```bash
cd glx/tests
./run-tests.sh
```

**Run specific categories**:
```bash
./run-tests.sh valid     # Valid tests only
./run-tests.sh invalid   # Invalid tests only
```

**Run with custom validator**:
```bash
./run-tests.sh --validator /path/to/your-glx-implementation
./run-tests.sh --validator ../bin/glx
```

### Test Options

**Verbose output**:
```bash
./run-tests.sh --verbose
```

**Stop on first failure**:
```bash
./run-tests.sh --fail-fast
```

**Generate test report**:
```bash
./run-tests.sh --report test-report.json
```

## Writing Tests

### 1. Valid Test Guidelines

**Create minimal test**:
```yaml
# TEST: person-minimal
# EXPECT: valid
# DESCRIPTION: Person with only required fields

id: person-a1b2c3d4
version: "1.0"
type: person
name:
  given: John
  surname: Smith
  display: John Smith
```

**Create complete test**:
```yaml
# TEST: person-complete
# EXPECT: valid
# DESCRIPTION: Person with all optional fields populated

id: person-b2c3d4e5
version: "1.0"
type: person
name:
  given: John
  surname: Smith
  display: John Smith
  nickname: Johnny
alternative_names:
  - John H. Smith
  - J.H. Smith
birth: "1850-01-15"
death: "1920-03-10"
gender: male
occupation: blacksmith
education: elementary
religion: Church of England
nationality: British
residences:
  - place: place-leeds
    date: "1850-1920"
notes: |
  John was a blacksmith in Leeds.
  He worked at the ironworks on Wellington Street.
citations: [citation-birth-cert, citation-census]
```

**Create edge case test**:
```yaml
# TEST: person-maximal-names
# EXPECT: valid
# DESCRIPTION: Person with maximum name variations

id: person-c3d4e5f6
version: "1.0"
type: person
name:
  given: Jean-Pierre
  middle: Henri
  surname: O'Connor MacLeod
  display: Jean-Pierre Henri O'Connor MacLeod
  nickname: JP
alternative_names:
  - Jean Pierre Henri OConnor MacLeod
  - J.P.H. O'Connor-MacLeod
  - John Peter Henry Connor McLeod
notes: "Testing name handling with apostrophes, hyphens, and spaces"
```

### 2. Invalid Test Guidelines

**Test missing required fields**:
```yaml
# TEST: person-missing-name
# EXPECT: invalid
# ERROR: "name field is required"
# DESCRIPTION: Person must have a name field

id: person-d4e5f6g7
version: "1.0"
type: person
# Missing required name field
```

**Test invalid ID formats**:
```yaml
# TEST: person-bad-id-format
# EXPECT: invalid
# ERROR: "id must match pattern"
# DESCRIPTION: Person ID must follow person-{8hex} pattern

id: john-smith  # Invalid: no prefix, contains letters
version: "1.0"
type: person
name:
  given: John
  surname: Smith
```

**Test broken references**:
```yaml
# TEST: person-broken-place-ref
# EXPECT: invalid
# ERROR: "place 'place-nonexistent' not found"
# DESCRIPTION: Person birth place must reference existing place

id: person-e5f6g7h8
version: "1.0"
type: person
name:
  given: John
  surname: Smith
birth:
  place: place-nonexistent  # This place doesn't exist
```

**Test schema violations**:
```yaml
# TEST: person-invalid-version
# EXPECT: invalid
# ERROR: "version must match pattern"
# DESCRIPTION: Version must be in major.minor format

id: person-f6g7h8i9
version: "1"  # Invalid: missing minor version
type: person
name:
  given: John
  surname: Smith
```

## Test Implementation

### Test Runner Script

**Core test logic**:
```bash
#!/bin/bash
# glx/tests/run-tests.sh

VALIDATOR=${VALIDATOR:-glx}
TEST_DIR=$1

for test_file in $TEST_DIR/*.glx; do
    if [ ! -f "$test_file" ]; then
        continue
    fi

    # Parse test header
    TEST_NAME=$(grep "^# TEST:" "$test_file" | cut -d: -f2 | tr -d ' ')
    EXPECTED=$(grep "^# EXPECT:" "$test_file" | cut -d: -f2 | tr -d ' ')

    # Run validation
    if $VALIDATOR validate "$test_file" 2>/dev/null; then
        RESULT="valid"
    else
        RESULT="invalid"
    fi

    # Check result
    if [ "$RESULT" = "$EXPECTED" ]; then
        echo "✓ $TEST_NAME"
        PASSED=$((PASSED + 1))
    else
        echo "✗ $TEST_NAME (expected $EXPECTED, got $RESULT)"
        FAILED=$((FAILED + 1))
    fi
done
```

### Error Message Testing

**For invalid tests, verify error messages**:
```bash
# Extract expected error pattern
EXPECTED_ERROR=$(grep "^# ERROR:" "$test_file" | cut -d: -f2- | tr -d '"')

# Capture actual error
ACTUAL_ERROR=$($VALIDATOR validate "$test_file" 2>&1)

# Check if actual error matches expected pattern
if echo "$ACTUAL_ERROR" | grep -q "$EXPECTED_ERROR"; then
    echo "✓ Error message correct"
else
    echo "✗ Wrong error message. Expected: $EXPECTED_ERROR, Got: $ACTUAL_ERROR"
fi
```

## Test Coverage

### Entity Type Coverage

**Ensure all entity types are tested**:

| Entity Type | Valid Tests | Invalid Tests | Edge Cases |
|-------------|-------------|---------------|------------|
| Person | ✓ | ✓ | Name variations, dates |
| Event | ✓ | ✓ | Event types, participants |
| Place | ✓ | ✓ | Hierarchy, coordinates |
| Source | ✓ | ✓ | Publication info |
| Citation | ✓ | ✓ | Quality ratings, locators |
| Repository | ✓ | ✓ | Contact information |
| Assertion | ✓ | ✓ | Evidence chains |
| Relationship | ✓ | ✓ | Participant roles |
| Media | ✓ | ✓ | File formats |

### Validation Rule Coverage

**Test all validation rules**:

- **Required fields**: All required fields tested for absence
- **Field formats**: Invalid formats for all pattern-based fields
- **Reference integrity**: Broken references for all entity types
- **Business logic**: Date consistency, place hierarchy, etc.
- **Schema compliance**: All schema constraints tested

## Integration Testing

### Example Validation

**Test against real examples**:
```bash
# Validate complete family example
glx validate examples/complete-family/

# Check all examples
for dir in examples/*/; do
    echo "Testing $dir"
    glx validate "$dir"
done
```

### Cross-Entity Testing

**Test entity relationships**:
```bash
# Create test with complete evidence chain
# Validate that all references work
# Test circular reference prevention
# Verify hierarchy validation
```

## Performance Testing

### Large Archive Tests

**Test with realistic data sizes**:
```bash
# Generate large test data
python3 scripts/generate-large-test.py --persons 1000 --events 2000

# Time validation
time glx validate large-test-archive/

# Memory usage
/usr/bin/time -v glx validate large-test-archive/
```

### Concurrent Testing

**Test parallel validation**:
```bash
# Test multiple files simultaneously
glx validate persons/*.glx events/*.glx places/*.glx

# Test concurrent validation processes
glx validate archive1/ & glx validate archive2/ & wait
```

## CI/CD Testing

### GitHub Actions Integration

**Automated testing in CI**:
```yaml
# .github/workflows/test.yml
- name: Run Test Suite
  run: |
    cd test-suite
    ./run-tests.sh

- name: Validate Examples
  run: |
    glx validate examples/

- name: Schema Validation
  run: |
    ajv compile -s schema/meta/schema.schema.json
    find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;
```

### Pre-commit Testing

**Test before commits** (future):
```bash
# Validate changes
glx validate $(git diff --name-only HEAD)

# Run affected tests
./run-tests.sh --changed-only
```

## Test Maintenance

### Adding New Tests

**Process for new tests**:

1. **Identify test need**: New feature, edge case, or bug fix
2. **Choose category**: Valid or invalid test
3. **Write test file**: Follow naming and format conventions
4. **Add header**: Include TEST, EXPECT, ERROR, DESCRIPTION
5. **Test locally**: Run test to verify it works as expected
6. **Update documentation**: Add to test suite README

### Test Review Process

**Before adding tests**:
- [ ] Test follows naming conventions
- [ ] Header is complete and correct
- [ ] Test validates the intended behavior
- [ ] Error messages are specific (for invalid tests)
- [ ] Test is minimal and focused

### Regression Testing

**Prevent breaking changes**:
```bash
# Run tests before making changes
./run-tests.sh > before.txt

# Make changes
# ... edit code ...

# Run tests after changes
./run-tests.sh > after.txt

# Compare results
diff before.txt after.txt
```

## Debugging Tests

### Test Failure Analysis

**When tests fail**:
```bash
# Run with verbose output
./run-tests.sh --verbose --fail-fast

# Debug specific test
glx validate glx/tests/valid/person-minimal.glx

# Check test file format
head -10 glx/tests/valid/person-minimal.glx

# Verify validator behavior
glx validate --debug glx/tests/valid/person-minimal.glx
```

### Common Test Issues

**1. YAML Syntax Errors**:
```bash
# Check YAML syntax
python3 -c "import yaml; yaml.safe_load(open('test.glx'))"

# Fix indentation issues
# Use spaces, not tabs
# Match indentation levels
```

**2. Schema Validation**:
```bash
# Test against schema directly
ajv validate -s schema/v1/person.schema.json test.glx

# Check schema compilation
ajv compile -s schema/v1/person.schema.json
```

**3. Reference Issues**:
```bash
# Check for missing entities
# Verify all referenced IDs exist
# Check file locations
```

## Test Documentation

### Test Suite README

**Keep test documentation current**:
```markdown
# glx/tests/README.md

## Test Categories

### Valid Tests
- person-minimal: Minimal person record
- person-complete: Person with all optional fields
- event-standard: Typical life event
- place-hierarchy: Place with parent relationship

### Invalid Tests
- person-missing-id: Person without required ID
- person-bad-format: Invalid ID format
- event-no-place: Event without required place
- citation-no-source: Citation without source reference

## Running Tests

./run-tests.sh              # All tests
./run-tests.sh valid        # Valid tests only
./run-tests.sh invalid      # Invalid tests only
./run-tests.sh --verbose    # Detailed output
```

### Test Comments

**Document complex tests**:
```yaml
# TEST: person-complex-names
# EXPECT: valid
# DESCRIPTION: Person with complex name including apostrophes and hyphens
# NOTES: Tests name field handling for international names and punctuation

id: person-g7h8i9j0
version: "1.0"
type: person
name:
  given: Jean-Pierre
  surname: O'Connor-Smith
  display: Jean-Pierre O'Connor-Smith
alternative_names:
  - Jean Pierre OConnor Smith
  - J-P O'Connor-Smith
```

## Contributing Tests

### Test Contribution Process

**1. Create test files**:
```bash
# Add to appropriate directory
cp template.glx glx/tests/valid/new-test.glx

# Edit test file
vim glx/tests/valid/new-test.glx
```

**2. Test locally**:
```bash
# Run specific test
./run-tests.sh | grep new-test

# Verify behavior
glx validate glx/tests/valid/new-test.glx
```

**3. Update documentation**:
```bash
# Update README
vim glx/tests/README.md

# Add test description
# Update test counts
```

**4. Submit changes**:
```bash
git add glx/tests/valid/new-test.glx glx/tests/README.md
git commit -m "Add test for new validation rule

- Test: new-test-name
- Purpose: Validate new feature X
- Category: valid/invalid"
```

### Test Review Checklist

**Before submitting tests**:
- [ ] Test follows naming conventions
- [ ] Header is complete and accurate
- [ ] Test validates intended behavior
- [ ] Error messages are helpful (invalid tests)
- [ ] Test is minimal and focused
- [ ] Documentation is updated
- [ ] Test passes locally

This testing framework ensures that GENEALOGIX implementations are reliable, consistent, and maintainable.
