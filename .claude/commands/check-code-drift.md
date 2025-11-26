---
description: Identify drift between Go code (lib/types.go) and JSON schemas/specification
---

You are tasked with identifying any drift between the GLX Go code implementation and the JSON schemas/specification.

## Source of Truth Flow

**IMPORTANT**: The source of truth hierarchy is:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
     SOURCE OF TRUTH         DERIVED FROM SPEC      DERIVED FROM SCHEMA
```

**This means:**
- The **JSON schemas (and ultimately the specification) are the source of truth**
- Go code in `lib/types.go` is **derived from** the schemas
- Any drift detected means the **Go code needs to be updated** to match the schema
- When reporting drift, frame it as "Go code X needs to be updated because schema says Y"

**IMPORTANT - Bidirectional Validation Checking**:
Validation logic and constraints must be synchronized between specification and code:
- **Code has validation NOT in spec** → Specification needs to document this validation
- **Spec has validation NOT in code** → Code needs to implement this validation
- This is BIDIRECTIONAL - check both directions!

## Task

Analyze the Go type definitions in **glx/lib/types.go** and compare them with:

1. **specification/schema/v1/*.schema.json** - The source schemas (machine-readable)
2. **specification/4-entity-types/*.md** - The ultimate source specification (for context)

## Entity Types to Check

Core entities:
- Person
- Event
- Relationship (including RelationshipParticipant)
- Place (including AlternativeName)
- Source
- Citation
- Repository
- Media
- Assertion (including AssertionParticipant)

## Code Files to Check

In addition to **glx/lib/types.go**, also check:
- **glx/lib/validator.go** - Contains validation logic and constraint checking
- Any other lib files with validation functions

## What to Check

For each entity type, verify:

### 1. Field Presence
- All fields in JSON schema `properties` exist in Go struct
- All fields in Go struct exist in JSON schema (except internal fields)
- No missing fields in either direction

### 2. Field Types
Compare Go types with JSON schema types:
- `string` in schema → `string` in Go
- `array` in schema → `[]string` or `[]Type` in Go
- `object` in schema → `struct` or `map[string]any` in Go
- `number` in schema → `float64` or `*float64` in Go
- `boolean` in schema → `bool` in Go

### 3. Required vs Optional
- Required fields in schema should NOT have `omitempty` in yaml tag
- Optional fields in schema should have `omitempty` in yaml tag
- Check pointer types for truly optional fields (e.g., `*float64` for latitude/longitude)

### 4. YAML Tag Names
- Go struct field `yaml:"field_name"` must match JSON schema property name
- Check for snake_case vs camelCase mismatches
- Verify all yaml tags are present and correct

### 5. Reference Types
- Check that `refType` tags in Go code match the schema's reference patterns
- Example: `refType:"persons"` should correspond to pattern `^[a-zA-Z0-9-]{1,64}$` in schema
- Verify reference arrays have correct `refType` tags

### 6. Nested Types
- Check nested structs (EventParticipant, RelationshipParticipant, AssertionParticipant, AlternativeName, DateRange)
- Verify these match the schema's object definitions
- Check required fields in nested types

### 7. Special Cases

#### Assertion Entity
- Verify mutually exclusive fields: `claim`/`participant`, `value`/`participant`
- Check that required constraint `anyOf: [sources, citations]` is handled
- Verify `subject` field allows multiple entity types

#### Properties Field
- All entities have `properties map[string]any` with `omitempty`
- This is documented as "Vocabulary-defined properties"

#### GLXFile Top-Level
- Check that GLXFile struct has all entity type maps
- Verify yaml tags match schema (e.g., `persons`, `events`, etc.)
- Check vocabulary definition fields

### 8. Validation Logic and Constraints (BIDIRECTIONAL CHECK)

This is where validation logic often lives ONLY in the code. Check both directions:

#### Code → Specification (Documentation Gaps)
Look for validation logic in **glx/lib/validator.go** and other validation code that is NOT documented in the specification:

- **Field format validation** (e.g., regex patterns, length constraints)
  - Example: Email format validation, ID format validation
  - If code validates format, specification should document the format

- **Cross-field constraints** (e.g., mutually exclusive fields, conditional requirements)
  - Example: "If field A is present, field B is required"
  - If code enforces constraint, specification should document it

- **Business rules** (e.g., date ranges, logical constraints)
  - Example: "Birth date must be before death date"
  - If code validates rule, specification should document it

- **Reference validation** (e.g., checking that referenced entities exist)
  - Example: "person_id must reference a valid Person entity"
  - If code validates references, specification should document the requirement

- **Enumeration constraints** (e.g., allowed values for fields)
  - Example: "type field must be one of: [value1, value2, value3]"
  - If code validates enums, specification should document allowed values

#### Specification → Code (Implementation Gaps)
Look for validation rules documented in **specification/4-entity-types/*.md** that are NOT implemented in the code:

- **Required fields** specified in prose but not validated in code
- **Format requirements** described in specification but not checked in validator
- **Constraints** documented in specification but not enforced in code
- **Business rules** written in specification but missing from validation logic
- **Edge cases** described in specification but not handled in code

#### What to Report
For each validation rule or constraint:

**Code has validation NOT in specification**:
```
⚠️ Validation Gap in Specification

Location: lib/validator.go:123
Validation: Email field must match regex pattern `^[a-z]+@[a-z]+\.[a-z]+$`
Issue: This validation exists in code but is NOT documented in specification/4-entity-types/person.md
Action: Add format requirement to specification
```

**Specification has validation NOT in code**:
```
⚠️ Validation Missing in Code

Location: specification/4-entity-types/event.md (line 45)
Requirement: "The end_date, if present, must be after the date field"
Issue: This constraint is documented in specification but NOT enforced in lib/validator.go
Action: Implement validation in code
```

### 9. Common Issues to Look For

- Missing `omitempty` on optional fields
- Wrong yaml tag names (e.g., `state_province` vs `state`)
- Type mismatches (e.g., `string` vs `[]string`)
- Missing fields entirely
- Extra fields in Go that aren't in schema
- Reference types that should have `refType` tags but don't
- Required fields that have `omitempty` (wrong!)
- Validation logic in code not documented in specification
- Validation requirements in specification not implemented in code

## Output Format

For each entity type, report:

```
## [Entity Type]

✅ No drift detected - Go code matches schema and specification

OR

⚠️ Drift detected:

### Field Presence
- Go struct missing field for schema property `property_name`
- Go field `FieldName` exists but not in schema (may need removal or schema update)

### Field Types
- Go field `FieldName` has type `[]string` but schema defines type `string`
- Fix: Update Go type to match schema

### Required vs Optional
- Go field `FieldName` is required in schema but has `omitempty` tag (REMOVE omitempty)
- Go field `FieldName` is optional in schema but missing `omitempty` tag (ADD omitempty)

### YAML Tags
- Go field `FieldName` has yaml tag `field` but schema property is `field_name`
- Fix: Change yaml tag to match schema property name

### Reference Types
- Go field `FieldName` references entities but missing `refType:"entity_type"` tag
- Fix: Add appropriate refType tag

### Validation Drift (Code → Specification)
- Validation logic in lib/validator.go:123 not documented in specification
- Fix: Document validation requirement in specification/4-entity-types/[entity].md

### Validation Drift (Specification → Code)
- Validation requirement in specification/4-entity-types/[entity].md:45 not implemented
- Fix: Implement validation in lib/validator.go

### Documentation
- Go field `FieldName` comment doesn't match schema description
- Fix: Update comment to match schema
```

**Remember**:
- Frame struct/field drift as "what the Go code needs to change" to match the schema
- Frame validation drift BIDIRECTIONALLY - both code and specification may need updates

## Special Focus Areas

### Check These Known Patterns:

1. **DateString type**: Used for date fields in Event, Source
2. **PlaceID vs Place**: Event uses `PlaceID string` with yaml tag `place`
3. **Repository state field**: Go uses `State` field with yaml:`state_province`
4. **Media arrays**: Check if media references use `[]string` with `refType:"media"`
5. **Participant types**: EventParticipant, RelationshipParticipant, AssertionParticipant

## Summary

At the end, provide:
- Total entity types checked
- Count of entity types with structural drift (field/type/yaml tag issues)
- Count of entity types with validation drift
- List of Go types that need updates to match schema
- List of validation gaps found:
  - Validation in code but not in specification (needs documentation)
  - Validation in specification but not in code (needs implementation)
- Severity assessment (critical/major/minor)
- Recommended actions:
  - "Update lib/types.go [specific types] to match schema"
  - "Document validation logic in specification/4-entity-types/[files]"
  - "Implement missing validation in lib/validator.go"

## Notes

- **Schema is the source of truth** - Go code should be updated to match it
- **Validation drift is BIDIRECTIONAL** - both code and specification may need updates
- Internal fields (like `validation *ValidationResult` in GLXFile) are expected to not be in schemas
- Comment differences are informational only unless significantly misleading
- Focus on structural issues that could cause marshaling/unmarshaling problems
- Check both directions: schema → Go (missing in Go) AND Go → schema (not in schema, may need removal)
- Check both directions for validation: code → spec (missing documentation) AND spec → code (missing implementation)
- Pay special attention to required fields - these are critical for validation
- Required field with `omitempty` is a **CRITICAL** error
- Validation logic that exists in code but not in specification is a **MAJOR** documentation issue
- Validation requirements in specification but not in code is a **CRITICAL** implementation issue
