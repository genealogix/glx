---
description: Identify drift between Go code (go-glx/types.go) and JSON schemas/specification
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
- Go code in `go-glx/types.go` is **derived from** the schemas
- Any drift detected means the **Go code needs to be updated** to match the schema
- When reporting drift, frame it as "Go code X needs to be updated because schema says Y"

**IMPORTANT - Bidirectional Validation Checking**:
Validation logic and constraints must be synchronized between specification and code:
- **Code has validation NOT in spec** → Specification needs to document this validation
- **Spec has validation NOT in code** → Code needs to implement this validation
- This is BIDIRECTIONAL - check both directions!

## Task

Analyze the Go type definitions in **go-glx/types.go** and compare them with:

1. **specification/schema/v1/*.schema.json** - The source schemas (machine-readable)
2. **specification/4-entity-types/*.md** - The ultimate source specification (for context)

## Entity Types to Check

Core entities:
- Person
- Event (uses Participant)
- Relationship (uses Participant)
- Place
- Source
- Citation
- Repository
- Media
- Assertion (uses Participant)

## Code Files to Check

In addition to **go-glx/types.go**, also check:
- **go-glx/validation.go** - Contains validation logic and constraint checking
- Any other `go-glx/*.go` files with validation functions (e.g., `validation_temporal.go`)

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
- Check Participant struct (used by Event, Relationship, and Assertion)
- Verify it matches the schema's object definitions
- Check required fields in nested types

### 7. Special Cases

#### Assertion Entity
- Verify mutually exclusive fields: `property`/`participant`, `value`/`participant`
- Check that required constraint `anyOf: [sources, citations, media]` is handled
- Verify `subject` field allows multiple entity types

#### Properties Field
- All entities have `properties map[string]any` with `omitempty`
- This is documented as "Vocabulary-defined properties"

#### GLXFile Top-Level
- Check that GLXFile struct has all entity type maps
- Verify yaml tags match schema (e.g., `persons`, `events`, etc.)
- Check vocabulary definition fields

### 8. Validation Logic and Constraints

**IMPORTANT — Two-Layer Validation Architecture:**

The CLI (`glx validate`) runs validation in two passes:

1. **Pass 1 — JSON Schema validation** (`glx/validator.go` → `ValidateGLXFileStructure` using `gojsonschema`):
   Enforces ALL structural constraints from the JSON schemas:
   - `required` fields
   - `minLength`, `minItems`, `minimum`/`maximum` constraints
   - `allOf`/`anyOf`/`not` constraints (e.g., Assertion mutual exclusivity)
   - `additionalProperties: false`
   - `pattern` on entity ID references
   - `format` constraints (e.g., URI)

2. **Pass 2 — Go cross-reference validation** (`go-glx/validation.go` → `archive.Validate()`):
   Handles things JSON schema CANNOT check:
   - Entity/vocabulary reference existence (does the referenced ID actually exist?)
   - Place hierarchy cycle detection
   - Property vocabulary validation (is this property name defined?)
   - Property value type validation (does the value match the vocabulary's value_type?)
   - Date format validation
   - Temporal property structure validation

**DO NOT flag constraints already enforced by JSON schema as "missing from Go code."**
The Go validator intentionally does NOT duplicate JSON schema constraints. Only flag
validation gaps where NEITHER layer covers a requirement from the specification.

#### What to Check

Look for validation rules documented in **specification/4-entity-types/*.md** that are NOT
enforced by EITHER the JSON schema OR the Go validator:

- **Business rules** not expressible in JSON schema (e.g., "birth date must be before death date")
- **Cross-entity constraints** beyond simple reference existence
- **Semantic validation** that requires understanding entity relationships

Also check the reverse direction — validation logic in Go code that is NOT documented in the specification:

- **Custom validation rules** in `go-glx/validation.go` not mentioned in spec
- **Warning-level checks** that users should know about

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
- Validation logic in go-glx/validation.go:123 not documented in specification
- Fix: Document validation requirement in specification/4-entity-types/[entity].md

### Validation Drift (Specification → Code)
- Validation requirement in specification/4-entity-types/[entity].md:45 not implemented
- Fix: Implement validation in go-glx/validation.go

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
5. **Participant type**: Unified Participant struct used by Event, Relationship, and Assertion

## Summary

At the end, provide:
- Total entity types checked
- Count of entity types with structural drift (field/type/yaml tag issues)
- List of Go types that need updates to match schema
- Any validation gaps not covered by EITHER JSON schema or Go validator
- Severity assessment (critical/major/minor)
- Recommended actions

## Notes

- **Schema is the source of truth** - Go code should be updated to match it
- **Two-layer validation** - JSON schema (pass 1) handles structural constraints; Go validator (pass 2) handles cross-references and semantic checks. Do NOT flag schema-covered constraints as missing from Go code.
- Internal fields (like `validation *ValidationResult` in GLXFile) are expected to not be in schemas
- Comment differences are informational only unless significantly misleading
- Focus on structural issues that could cause marshaling/unmarshaling problems
- Check both directions: schema → Go (missing in Go) AND Go → schema (not in schema, may need removal)
- Pay special attention to required fields - these are critical for validation
- Required field with `omitempty` is a **CRITICAL** error
- Go fields that exist but are NOT in the schema (with `additionalProperties: false`) are **CRITICAL** — they produce schema-invalid YAML
