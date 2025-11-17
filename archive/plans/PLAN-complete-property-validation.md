# Implementation Plan: Complete Property Validation & Standard Vocabulary Adoption

## Status: ✅ COMPLETED
**Started:** 2025-11-17
**Completed:** 2025-11-17

## Overview
Complete the property vocabulary implementation by fixing non-standard property usage and implementing full temporal property validation.

## Context
- Standard vocabularies exist and use `given_name`/`family_name` (correct)
- Test files and examples incorrectly use `primary_name` (non-standard)
- Temporal validation exists for `reference_type` but NOT for `value_type`
- Need comprehensive temporal property testing

## Implementation Checklist

### Phase 1: Fix Standard Vocabulary Usage ✅ CRITICAL
**Estimated Time:** 1 hour

#### 1.1 Update Test Data Files
- [ ] Fix `glx/testdata/valid/assertion-with-participant.glx`
  - Replace `primary_name` with `given_name` and `family_name`
  - Update embedded vocabulary if it defines primary_name
- [ ] Fix `glx/testdata/valid/person-with-properties.glx`
  - Replace `primary_name` with `given_name` and `family_name`
  - Update embedded vocabulary
- [ ] Fix `glx/testdata/invalid/assertion-participant-and-claim/archive.glx`
- [ ] Fix `glx/testdata/invalid/assertion-participant-and-value/archive.glx`
- [ ] Fix `glx/testdata/invalid/assertion-participant-invalid-role/archive.glx`
- [ ] Fix `glx/testdata/invalid/assertion-unknown-claim/archive.glx`

#### 1.2 Update Test Expectations
- [ ] Update `glx/cmd_validate_test.go` if it expects `primary_name`
- [ ] Run tests to ensure they pass with new property names

**Success Criteria:**
- All testdata uses standard vocabulary (`given_name`/`family_name`)
- All tests pass

---

### Phase 2: Implement Temporal Value Validation ⚠️ HIGH PRIORITY
**Estimated Time:** 2 hours

#### 2.1 Add Value Type Validation Function
**File:** `lib/validation.go`

- [ ] Create `validatePropertyValue()` function:
  ```go
  func (glx *GLXFile) validatePropertyValue(
      entityType, entityID, propName string,
      propValue interface{},
      propDef *PropertyDefinition,
      result *ValidationResult,
  ) {
      isTemporal := propDef.Temporal != nil && *propDef.Temporal

      // Handle non-temporal: must be simple value
      if !isTemporal {
          // TODO: Validate against value_type (date, string, integer, boolean)
          // For now, just accept any value
          return
      }

      // Handle temporal: can be simple value OR list of {value, date} objects
      // Simple value is OK for temporal properties
      if _, isSimple := propValue.(string); isSimple {
          return
      }
      if _, isSimple := propValue.(float64); isSimple {
          return
      }
      if _, isSimple := propValue.(bool); isSimple {
          return
      }

      // List structure for temporal properties
      if valueList, ok := propValue.([]interface{}); ok {
          for i, item := range valueList {
              itemMap, ok := item.(map[string]interface{})
              if !ok {
                  result.Errors = append(result.Errors, ValidationError{
                      SourceType:  entityType,
                      SourceID:    entityID,
                      SourceField: fmt.Sprintf("properties.%s[%d]", propName, i),
                      Message: fmt.Sprintf("%s[%s].properties.%s[%d]: temporal list items must be objects with 'value' and optional 'date' fields",
                          entityType, entityID, propName, i),
                  })
                  continue
              }

              // Check for required 'value' field
              if _, hasValue := itemMap["value"]; !hasValue {
                  result.Errors = append(result.Errors, ValidationError{
                      SourceType:  entityType,
                      SourceID:    entityID,
                      SourceField: fmt.Sprintf("properties.%s[%d]", propName, i),
                      Message: fmt.Sprintf("%s[%s].properties.%s[%d]: temporal list items must have 'value' field",
                          entityType, entityID, propName, i),
                  })
              }

              // Optional: validate date field format
              if dateVal, hasDate := itemMap["date"]; hasDate {
                  if _, isString := dateVal.(string); !isString {
                      result.Warnings = append(result.Warnings, ValidationWarning{
                          SourceType: entityType,
                          SourceID:   entityID,
                          Field:      fmt.Sprintf("properties.%s[%d].date", propName, i),
                          Message:    fmt.Sprintf("%s[%s].properties.%s[%d].date should be a string", entityType, entityID, propName, i),
                      })
                  }
              }
          }
          return
      }

      // Unknown structure for temporal property
      result.Warnings = append(result.Warnings, ValidationWarning{
          SourceType: entityType,
          SourceID:   entityID,
          Field:      fmt.Sprintf("properties.%s", propName),
          Message:    fmt.Sprintf("%s[%s].properties.%s: unexpected value type for temporal property (expected simple value or list)", entityType, entityID, propName),
      })
  }
  ```

#### 2.2 Integrate into validateProperties
- [ ] Update `validateProperties()` to call `validatePropertyValue()`:
  ```go
  func (glx *GLXFile) validateProperties(...) {
      for propName, propValue := range properties {
          propDef, exists := propVocab[propName]
          if !exists {
              // ... existing warning ...
              continue
          }

          // Validate references
          if propDef.ReferenceType != "" {
              glx.validatePropertyReference(entityType, entityID, propName, propValue, propDef.ReferenceType, result)
          } else if propDef.ValueType != "" {
              // ADD THIS: Validate value types
              glx.validatePropertyValue(entityType, entityID, propName, propValue, propDef, result)
          }
      }
  }
  ```

**Success Criteria:**
- Temporal properties with `value_type` are validated
- Both simple values and temporal lists are accepted
- Malformed temporal lists generate errors
- Validation works for all value types (string, date, integer, boolean)

---

### Phase 3: Add Comprehensive Tests ⚠️ HIGH PRIORITY
**Estimated Time:** 1.5 hours

#### 3.1 Create Temporal Property Test Files

- [ ] Create `glx/testdata/valid/temporal-properties-simple.glx`:
  ```yaml
  person_properties:
    occupation:
      label: "Occupation"
      value_type: string
      temporal: true

  persons:
    person-simple-temporal:
      properties:
        occupation: "blacksmith"  # Simple value for temporal property - OK
  ```

- [ ] Create `glx/testdata/valid/temporal-properties-list.glx`:
  ```yaml
  person_properties:
    occupation:
      label: "Occupation"
      value_type: string
      temporal: true
    residence:
      label: "Residence"
      reference_type: places
      temporal: true

  places:
    place-leeds:
      name: "Leeds"

  persons:
    person-temporal-list:
      properties:
        occupation:
          - value: "blacksmith"
            date: "1870"
          - value: "farmer"
            date: "FROM 1880 TO 1920"
        residence:
          - value: "place-leeds"
            date: "FROM 1850 TO 1900"
  ```

- [ ] Create `glx/testdata/invalid/temporal-properties-malformed.glx`:
  ```yaml
  person_properties:
    occupation:
      label: "Occupation"
      value_type: string
      temporal: true

  persons:
    person-bad-temporal:
      properties:
        occupation:
          - invalid_structure  # ERROR: should be {value, date}
          - value: "blacksmith"  # OK
          - date: "1870"  # ERROR: missing 'value'
  ```

#### 3.2 Add Unit Tests
- [ ] Create/update `lib/validation_test.go`:
  - Test simple temporal value validation
  - Test temporal list validation
  - Test malformed temporal structures
  - Test non-temporal property validation
  - Test mixed temporal/non-temporal in same entity

#### 3.3 Update Existing Tests
- [ ] Run all tests and fix any failures due to property name changes
- [ ] Ensure validation tests cover new temporal scenarios

**Success Criteria:**
- All temporal property scenarios have test coverage
- Tests pass for valid temporal structures
- Tests fail for invalid temporal structures
- Test coverage remains above 79%

---

### Phase 4: Value Type Format Validation (Optional Enhancement)
**Estimated Time:** 1 hour

#### 4.1 Implement Format Validators
- [ ] Add `validateDateFormat()` - Validate FamilySearch normalized dates
- [ ] Add `validateIntegerFormat()` - Validate integer values
- [ ] Add `validateBooleanFormat()` - Validate boolean values
- [ ] Integrate into `validatePropertyValue()`

**Note:** This is optional and can be deferred. Current plan (line 888) says to defer format validation.

---

### Phase 5: Documentation & Examples
**Estimated Time:** 30 minutes

#### 5.1 Update Test Documentation
- [ ] Document temporal property test files in `glx/testdata/README.md`
- [ ] Add comments explaining temporal validation behavior

#### 5.2 Verify Examples (if time permits)
- [ ] Check if any docs/examples still use `primary_name`
- [ ] Update if necessary (low priority - examples aren't tested)

---

## Testing Strategy

### Test Cases to Cover
1. **Standard Vocabulary Usage:**
   - ✅ Properties use `given_name`, `family_name` (not `primary_name`)
   - ✅ All testdata validates successfully

2. **Temporal Property Validation:**
   - ✅ Simple value for temporal property: `occupation: "blacksmith"`
   - ✅ Temporal list: `occupation: [{value: "blacksmith", date: "1870"}]`
   - ✅ Mixed: some temporal, some non-temporal properties
   - ❌ Missing `value` field in temporal list item
   - ❌ Non-object item in temporal list
   - ✅ Temporal reference property: `residence: [{value: "place-id", date: "1900"}]`

3. **Non-Temporal Property Validation:**
   - ✅ Simple value for non-temporal property: `born_on: "1850-01-15"`
   - ⚠️ List for non-temporal property (should warn/error)

### Validation Levels
- **Errors (Hard):** Missing required fields, invalid structure, broken references
- **Warnings (Soft):** Unknown properties, unexpected formats (where recoverable)

---

## Success Metrics

### Code Quality
- [x] All tests passing (100%) ✅
- [x] Code coverage ≥ 79% (maintain current level) ✅ **80.8%** (improved from 79.2%)
- [x] No regression in existing functionality ✅

### Correctness
- [x] All testdata uses standard vocabulary ✅
- [x] Temporal properties validated correctly ✅
- [x] Both simple and list temporal values accepted ✅
- [x] Malformed temporal structures rejected ✅

### Completeness
- [x] Standard vocabulary (`given_name`/`family_name`) used throughout ✅
- [x] Temporal validation implemented for all value types ✅
- [x] Comprehensive test coverage for temporal properties ✅
- [ ] Documentation updated (low priority)

---

## Implementation Notes

### Design Decisions
1. **Temporal validation approach:** Accept both simple values and lists for temporal properties
2. **Error vs Warning:** Structural errors (missing `value` field) are errors; format issues are warnings
3. **Value type formats:** Defer strict format validation (dates, integers) - accept any valid YAML type for now

### Key Files to Modify
1. `lib/validation.go` - Add `validatePropertyValue()`, update `validateProperties()`
2. `glx/testdata/valid/*.glx` - Replace `primary_name` with standard properties
3. `glx/testdata/invalid/*.glx` - Replace `primary_name` with standard properties
4. `glx/testdata/valid/temporal-*.glx` - New test files (3 files)
5. `glx/testdata/invalid/temporal-*.glx` - New test file (1 file)
6. `lib/validation_test.go` - Add temporal validation tests

### Testing Approach
1. Fix all `primary_name` uses first
2. Run existing tests - should pass
3. Implement temporal validation
4. Add temporal test files
5. Run all tests - should pass
6. Check coverage - should maintain ≥79%

---

## Progress Tracking

### Phase 1: Standard Vocabulary ✅ COMPLETED
- [x] 6/6 testdata files updated
  - Updated all valid and invalid testdata to use `given_name`/`family_name`
  - Updated embedded vocabularies
- [x] Test files updated
- [x] Tests passing: ✅ All 79.2% coverage maintained

### Phase 2: Temporal Validation ✅ COMPLETED
- [x] validatePropertyValue() function implemented in [lib/validation.go:342-423](lib/validation.go#L342-L423)
- [x] Integration into validateProperties() at [lib/validation.go:296-298](lib/validation.go#L296-L298)
- [x] Tests passing: ✅

### Phase 3: Test Coverage ✅ COMPLETED
- [x] 2/2 valid test files created
  - `glx/testdata/valid/temporal-properties-simple/archive.glx`
  - `glx/testdata/valid/temporal-properties-list/archive.glx`
- [x] 1/1 invalid test file created
  - `glx/testdata/invalid/temporal-properties-malformed/archive.glx`
- [x] Unit tests added:
  - 6 unit tests in `lib/validation_temporal_test.go`
  - 2 integration tests in `glx/cmd_validate_temporal_test.go`
- [x] Coverage ≥79%: ✅ 79.2% maintained

### Phase 4: Format Validation ✅ COMPLETED
- [x] Implemented `validateValueType()` function in [lib/validation.go:518-571](lib/validation.go#L518-L571)
- [x] Implemented `validateDateFormat()` function in [lib/validation.go:437-485](lib/validation.go#L437-L485)
- [x] Validates string, integer, boolean, and date types
- [x] Date validation supports ISO 8601 format (YYYY, YYYY-MM, YYYY-MM-DD)
- [x] Date validation supports FamilySearch-inspired keywords (FROM, TO, ABT, BEF, AFT, BET, CAL, INT)
- [x] Added comprehensive tests in `lib/validation_format_test.go` (4 test functions, 40+ test cases)
- [x] All tests passing

### Phase 5: Documentation ✅ COMPLETED
- [x] Updated date format documentation in [specification/6-data-types.md](specification/6-data-types.md)
  - Clarified we use ISO 8601-style dates with FamilySearch-inspired keywords
  - Added important notes about 4-digit year requirement
  - Added validation behavior documentation
- [x] Removed all `primary_name` references (19 files updated)
  - Updated all documentation files
  - Updated all specification files
  - Updated all example READMEs
  - Updated migration guides
- [x] All examples now use standard `given_name`/`family_name` vocabulary

---

## Rollback Plan
If issues arise:
1. Revert testdata files: `git checkout glx/testdata/`
2. Revert validation changes: `git checkout lib/validation.go`
3. All tests should pass at previous state

---

## Known Issues & Risks
1. **Risk:** Changing property names might break existing user archives
   - **Mitigation:** No users yet (per plan line 864)
2. **Risk:** Temporal validation might be too strict
   - **Mitigation:** Use warnings for recoverable issues, errors only for structural problems

---

## Next Steps After Completion
1. Update docs/examples to use standard vocabulary (if time)
2. Consider implementing strict value type format validation
3. Add more comprehensive date format validation (FamilySearch standard)
4. Performance testing with large archives

---

## References
- Main Plan: `/workspaces/spec/PLAN-property-vocabularies.md`
- Standard Vocabularies: `/workspaces/spec/specification/5-standard-vocabularies/`
- Validation Code: `/workspaces/spec/lib/validation.go`
- Final Coverage: **80.8%** (improved from initial 79.2%)
