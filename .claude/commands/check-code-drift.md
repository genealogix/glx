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

### 8. Common Issues to Look For

- Missing `omitempty` on optional fields
- Wrong yaml tag names (e.g., `state_province` vs `state`)
- Type mismatches (e.g., `string` vs `[]string`)
- Missing fields entirely
- Extra fields in Go that aren't in schema
- Reference types that should have `refType` tags but don't
- Required fields that have `omitempty` (wrong!)

## Output Format

For each entity type, report:

```
## [Entity Type]

✅ No drift detected - Go code matches schema

OR

⚠️ Drift detected - Go code needs updates:

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

### Documentation
- Go field `FieldName` comment doesn't match schema description
- Fix: Update comment to match schema
```

**Remember**: Frame all drift as "what the Go code needs to change" to match the schema.

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
- Count of entity types with drift
- List of Go types that need updates to match schema
- Severity assessment (critical/major/minor)
- Recommended actions: "Update lib/types.go [specific types] to match schema"

## Notes

- **Schema is the source of truth** - Go code should be updated to match it
- Internal fields (like `validation *ValidationResult` in GLXFile) are expected to not be in schemas
- Comment differences are informational only unless significantly misleading
- Focus on structural issues that could cause marshaling/unmarshaling problems
- Check both directions: schema → Go (missing in Go) AND Go → schema (not in schema, may need removal)
- Pay special attention to required fields - these are critical for validation
- Required field with `omitempty` is a **CRITICAL** error
