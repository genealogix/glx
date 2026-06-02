---
description: Identify drift between Go code (go-glx/types.go) and JSON schemas/specification
allowed-tools:
  - Read
  - Grep
  - Glob
model: claude-opus-4-7
---

You are tasked with identifying any drift between the GLX Go code implementation and the JSON schemas/specification.

## Allowlisted Drift (read this first)

Before analyzing, read **`.claude/drift-allowlist.yaml`** (validated by
`.claude/drift-allowlist.schema.json`). It is the single source of truth for
known, per-symbol drift that has already been triaged. Check every finding you
would otherwise report against it:

- If a finding concerns the same `file` and Go symbol as an allowlist entry,
  **suppress it** — do not report it as drift. Match on symbol *identity*, not
  exact string: an entry's `symbol` is written qualified (e.g.
  `GLXFile.ImportMetadata`), so a finding that names the field bare
  (`ImportMetadata`) or by its yaml tag (`metadata`) still matches it.
  - `permanent: true` entries are by-design and will never be "fixed" (e.g. a Go
    field name that intentionally differs from its yaml tag). Treat them as *not
    drift*.
  - Entries with a `tracking_issue` are temporary deferrals. Don't re-report them
    as new findings; refer to the tracking issue instead.
- Anything **not** in the allowlist is reported normally.

Only genuine *per-symbol* exceptions live in the allowlist. Class-level
methodology (e.g. "the Go validator intentionally does not duplicate JSON-schema
constraints", below) stays in this prompt — it is not an allowlist entry. The
allowlist is human-curated: do **not** invent entries. If you believe a new
exception is warranted, report the drift and recommend adding an allowlist entry.

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

Supporting types:
- Metadata (includes Submitter) — checked against `glx-file.schema.json` property `metadata`
- EntityRef (used by Assertion.Subject) — checked against `assertion.schema.json` `subject` oneOf

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
- `oneOf: [string, array]` in schema → `NoteList` in Go (custom YAML marshal/unmarshal)
- `string` in schema for dates → `DateString` in Go (type alias)
- nested `object` in schema → `*Submitter` in Go (pointer for optional nested struct)

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

#### EntityRef (Assertion.Subject)
- Mutually exclusive fields: Person, Event, Relationship, Place
- Schema enforces via `oneOf` — exactly one field must be set
- All fields have `omitempty` — Go serialization produces correct YAML

#### NoteList
- Go type: `NoteList` (alias for `[]string`) with custom YAML marshal/unmarshal
- Schema: `oneOf: [{type: string}, {type: array, items: {type: string}}]`
- Single note marshals as plain string; multiple notes marshal as array
- Present on ALL entity types and Participant — check all 9 entities + Metadata

#### Properties Field
- All entities have `properties map[string]any` with `omitempty`
- This is documented as "Vocabulary-defined properties"

#### Metadata and Submitter
- `Metadata` struct has 11 fields — check against `glx-file.schema.json` property `metadata`
- `Submitter` is nested via `*Submitter` pointer — check against schema's submitter object

#### GLXFile Top-Level
- Check that GLXFile struct has all entity type maps
- Verify yaml tags match schema (e.g., `persons`, `events`, etc.)
- Check `ImportMetadata *Metadata` field against `metadata` property in schema (the Go field name vs yaml-tag difference is allowlisted — see "Allowlisted Drift" above)
- Check all vocabulary definition maps (10 type vocabs + 8 property vocabs)

### 8. Vocabulary Struct Types

All 10 type-vocabulary schemas share a single Go struct `VocabularyEntry`
(unified in #727). Its fields are the superset of every vocabulary's
needs: `Label, Description, Category, AppliesTo, MimeType, GEDCOM`.
Unused fields are elided via `omitempty` per schema.

| Schema File | Fields actually populated |
|-------------|---------------------------|
| `event-types.schema.json` | Label, Description, Category, GEDCOM |
| `relationship-types.schema.json` | Label, Description, GEDCOM |
| `place-types.schema.json` | Label, Description, Category |
| `source-types.schema.json` | Label, Description, GEDCOM |
| `repository-types.schema.json` | Label, Description, GEDCOM |
| `media-types.schema.json` | Label, Description, MimeType |
| `sex-types.schema.json` | Label, Description, GEDCOM |
| `gender-types.schema.json` | Label, Description, GEDCOM (usually empty) |
| `confidence-levels.schema.json` | Label, Description, GEDCOM |
| `participant-roles.schema.json` | Label, Description, AppliesTo, GEDCOM |

Drift checks:
- `VocabularyEntry` YAML field order is load-bearing — guarded by `TestVocabularyEntryYAMLFieldOrder`. The declared order `label, description, category, applies_to, mime_type, gedcom` matches on-disk vocab files.
- Each `GLXFile` vocabulary map must use `map[string]*VocabularyEntry`.

Also check:
- `FieldDefinition` struct (Label, Description, ValueType) against property vocabulary schemas
- `PropertyDefinition` struct against all 8 property vocabulary schemas
- `go-glx/constants.go` for vocabulary constant coverage (event types, roles, etc.)

### 9. Validation Logic and Constraints

**IMPORTANT — Two-Layer Validation Architecture:**

The CLI (`glx validate`) runs validation in two passes:

1. **Pass 1 — JSON Schema validation** (`glx/validator.go` → `ValidateGLXFileStructure` using `santhosh-tekuri/jsonschema/v6`):
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

### 10. Common Issues to Look For

- Missing `omitempty` on optional fields
- Wrong yaml tag names (e.g., `state_province` vs `state`)
- Type mismatches (e.g., `string` vs `[]string`)
- Missing fields entirely
- Extra fields in Go that aren't in schema
- Reference types that should have `refType` tags but don't
- Required fields that have `omitempty` (wrong!)
- Validation logic in code not documented in specification
- Validation requirements in specification not implemented in code
- GEDCOM converter still emits or consumes a renamed/removed schema field (silent round-trip data loss)
- New schema field has no `gedcom_*.go` reference (verify intentionally not mapped)

### 11. GEDCOM Converter Drift

The GEDCOM converter (`go-glx/gedcom_*.go`) is the I/O boundary to external
genealogy software; field drift here causes silent round-trip data loss.
Importer is `gedcom_<entity>.go`, exporter is `gedcom_export_<entity>.go`,
with these exceptions:

- Person — `gedcom_individual.go` (INDI tag) / `gedcom_export_person.go`
- Relationship — `gedcom_family.go` (FAM tag) / `gedcom_export_family.go`
- Citation, Assertion, source-citation chains — `gedcom_evidence.go`
- Event — nested in Person/Relationship importers (no dedicated file)

Search all `go-glx/gedcom_*.go` (including helpers and `_test.go` goldens),
case-insensitive:

- For each renamed or retyped field, search for the old field name. Any
  match is drift — the converter still reads/writes the old name.
- For each removed field, search for the field name. An active read/write
  is drift. An `assert.NotEqual` or "should be skipped" guard assertion is
  the post-cleanup state, not drift.
- For each added field, search for the new field name. Zero matches means
  the GEDCOM layer ignores the field; some GLX fields legitimately have no
  GEDCOM tag, so the right question is "is this intentionally not
  exported?"

Severity for each of these cases is in the Severity Rubric below.

## Severity Rubric

Assign one of **critical / major / minor / info** to every finding. The
table below is the authoritative source — don't invent severities outside it
or vary across runs. Where a row says "info", the finding is informational
and does not require action unless investigation confirms a problem.

| Category | Condition | Severity |
|---|---|---|
| Field presence | Required schema field has no Go counterpart | **critical** |
| Field presence | Go field has no schema counterpart, schema has `additionalProperties: false` | **critical** |
| Field presence | Optional schema field has no Go counterpart | **major** |
| Field presence | Internal Go field (e.g., `validation *ValidationResult`) not in schema | **info** |
| Required/optional | Schema-required field has `omitempty` in Go | **critical** |
| Required/optional | Schema-optional field missing `omitempty` in Go | **major** |
| Field types | Type mismatch that breaks marshaling (`string` vs `[]string`) | **critical** |
| Field types | Type widening covered by custom marshaler (`NoteList` vs `oneOf[string,array]`) | **info** |
| YAML tags | Tag name mismatches schema property name (`state` vs `state_province`) | **critical** |
| Reference types | Missing `refType` tag on a reference field | **major** |
| Reference types | Wrong `refType` target (e.g., `persons` vs `events`) | **critical** |
| GEDCOM converter | Removed schema field still read or written by `gedcom_*.go` (importer or exporter) — silent round-trip data loss | **critical** |
| GEDCOM converter | Renamed schema field still read/written under old name | **critical** |
| GEDCOM converter | Added schema field with zero `gedcom_*.go` references — verify intentionally not exported | **info** |
| Validation | Spec requires constraint neither JSON schema nor Go enforces | **major** |
| Validation | Go enforces constraint not documented in spec | **minor** |
| Documentation | Go comment doesn't match schema description | **info** |

This rubric is shape-only; row-level severities are starting points and
will be tuned by the eval harness (#796) once hand-graded cases exist.

## Output Format

For each entity type, report:

```
## [Entity Type]

✅ No drift detected - Go code matches schema and specification

OR

⚠️ Drift detected:

Each finding bullet MUST begin with its severity tag (`**critical**`,
`**major**`, `**minor**`, or `**info**`) immediately after the `-` list
marker, drawn from the Severity Rubric above. `Fix:` continuation lines
do not get their own tag.

### Field Presence
- **critical** — Go struct missing field for schema property `property_name` (required field)
- **major** — Go struct missing field for schema property `property_name` (optional field)
- **critical** — Go field `FieldName` exists but not in schema (schema has `additionalProperties: false`)

### Field Types
- **critical** — Go field `FieldName` has type `[]string` but schema defines type `string`
- Fix: Update Go type to match schema

### Required vs Optional
- **critical** — Go field `FieldName` is required in schema but has `omitempty` tag (REMOVE omitempty)
- **major** — Go field `FieldName` is optional in schema but missing `omitempty` tag (ADD omitempty)

### YAML Tags
- **critical** — Go field `FieldName` has yaml tag `field` but schema property is `field_name`
- Fix: Change yaml tag to match schema property name

### Reference Types
- **major** — Go field `FieldName` references entities but missing `refType:"entity_type"` tag
- Fix: Add appropriate refType tag

### Validation Drift (Code → Specification)
- **minor** — Validation logic in go-glx/validation.go:123 not documented in specification
- Fix: Document validation requirement in specification/4-entity-types/[entity].md

### Validation Drift (Specification → Code)
- **major** — Validation requirement in specification/4-entity-types/[entity].md:45 not implemented
- Fix: Implement validation in go-glx/validation.go

### Documentation
- **info** — Go field `FieldName` comment doesn't match schema description
- Fix: Update comment to match schema

### GEDCOM Converter Drift
- **critical** — Schema field `field_name` renamed to `new_name`; old name still present in go-glx/gedcom_export_repository.go:NN
- Fix: Update GEDCOM importer/exporter to use new field name
- **critical** — Schema field `removed_field` removed; still emitted by go-glx/gedcom_export_person.go:NN (silent round-trip data loss)
- **info** — Schema field `new_field` added; not referenced in any gedcom_*.go (confirm intentionally not exported, or wire through the converter)
```

Severity tags come from the Severity Rubric above. Findings without a
severity tag fail the format contract.

**Remember**:
- Frame struct/field drift as "what the Go code needs to change" to match the schema
- Frame validation drift BIDIRECTIONALLY - both code and specification may need updates

## Special Focus Areas

### Check These Known Patterns:

1. **DateString type**: Used for date fields in Event, Source, Media, Assertion, Metadata
2. **PlaceID vs Place**: Event uses `PlaceID string` with yaml tag `place`
3. **Repository state field**: Go uses `State` field with yaml:`state_province`
4. **Media arrays**: Check if media references use `[]string` with `refType:"media"`
5. **Participant type**: Unified Participant struct used by Event, Relationship, and Assertion
6. **NoteList type**: Used on ALL entity types and Participant — `oneOf` in schema
7. **Vocabulary GEDCOM fields**: Recently added to 5 vocabulary structs — verify all present

## Summary

At the end, provide:
- Total entity types checked
- Count of entity types with structural drift (field/type/yaml tag issues)
- List of Go types that need updates to match schema
- Any validation gaps not covered by EITHER JSON schema or Go validator
- Count of findings per severity (critical/major/minor/info), per the Severity Rubric above
- Recommended actions

## Notes

- **Schema is the source of truth** - Go code should be updated to match it
- **Two-layer validation** - JSON schema (pass 1) handles structural constraints; Go validator (pass 2) handles cross-references and semantic checks. Do NOT flag schema-covered constraints as missing from Go code.
- Internal fields (like `validation *ValidationResult` in GLXFile) are expected to not be in schemas
- Comment differences are informational only unless significantly misleading
- Focus on structural issues that could cause marshaling/unmarshaling problems
- Check both directions: schema → Go (missing in Go) AND Go → schema (not in schema, may need removal)
- Pay special attention to required fields - these are critical for validation
- See the Severity Rubric above for how each finding category maps to severity
