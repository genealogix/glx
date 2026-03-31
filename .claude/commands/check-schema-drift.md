---
description: Identify drift between specification markdown files and JSON schemas
---

You are tasked with identifying any drift between the GLX specification markdown files and the JSON schemas.

## Source of Truth Flow

**IMPORTANT**: The source of truth hierarchy is:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
     SOURCE OF TRUTH         DERIVED FROM SPEC      DERIVED FROM SCHEMA
```

**This means:**
- The **specification markdown files are the source of truth**
- JSON schemas are **derived from** the specification
- Any drift detected means the **schema needs to be updated** to match the specification
- When reporting drift, frame it as "Schema X needs to be updated because specification says Y"

## Task

Analyze all entity types and compare:

1. **specification/4-entity-types/*.md** - The source of truth (human-readable specification)
2. **specification/schema/v1/*.schema.json** - Derived schemas (machine-readable)

## Schemas to Check

### Entity Schemas
- assertion
- citation
- event
- media
- person
- place
- relationship
- repository
- source

### Top-Level Archive Schema
- `glx-file.schema.json` — the root schema defining overall GLX file structure, compare against `specification/3-archive-organization.md`:
  - `metadata` object fields match spec
  - All 9 entity map sections are present with correct `patternProperties` references
  - All vocabulary map sections are present
  - Entity ID pattern (`^[a-zA-Z0-9-]{1,64}$`) matches spec
  - Vocabulary key pattern matches spec

### Vocabulary Schemas
Compare `specification/4-entity-types/vocabularies.md` and `specification/5-standard-vocabularies/*.glx` against `specification/schema/v1/vocabularies/*.schema.json`:

**Type vocabularies** (9 schemas):
- event-types, relationship-types, place-types, source-types, media-types, repository-types, confidence-levels, participant-roles, gender-types

**Property vocabularies** (8 schemas):
- person-properties, event-properties, relationship-properties, place-properties, media-properties, repository-properties, source-properties, citation-properties

For each vocabulary schema, verify:
- Schema structure matches the vocabulary format documented in `vocabularies.md` (label, description, gedcom, fields, etc.)
- Required fields in the schema match what the spec says is required for vocabulary entries
- For property vocabulary schemas specifically, property definition schemas enforce `value_type`/`reference_type`/`vocabulary_type` mutual exclusivity per spec
- All vocabulary `.glx` template files validate against their corresponding schema

## What to Check

For each entity type, verify:

### 1. Required Fields Alignment
- Compare "Required Fields" table in markdown with `required` array in JSON schema
- Check that all markdown-listed required fields appear in schema
- Check that schema doesn't have additional required fields not documented

### 2. Optional Fields Alignment
- Compare "Optional Fields" table in markdown with `properties` in JSON schema
- Verify all markdown-listed optional fields are in schema
- Verify schema doesn't have properties missing from markdown

### 3. Field Types
- Compare field types in markdown tables with JSON schema types
- Check for type mismatches (e.g., markdown says "string", schema says "array")

### 4. Field Descriptions
- Check that descriptions in markdown tables roughly match schema descriptions
- Flag significant discrepancies
- **IGNORE** inline lists of vocabulary-defined values in description strings (e.g., a `properties` field description listing common property names like "locator, text_from_source, ..."). These are informational hints, not normative. They do not need to be updated every time a vocabulary entry is added.

### 5. Special Constraints
- Check for complex validation rules (patterns, minItems, anyOf, allOf, not)
- Verify these are documented in the markdown
- Look for undocumented constraints in schemas

### 6. Entity ID Patterns
- Verify entity ID pattern constraints match between docs and schemas
- Check that the pattern `^[a-zA-Z0-9-]{1,64}$` is consistently applied where needed

### 7. additionalProperties Severity
All top-level entity schemas and the archive root in `glx-file.schema.json` set `additionalProperties: false` on the entity objects. Some nested map fields (e.g., properties maps) intentionally use `additionalProperties: true` to allow arbitrary keys. For fields covered by `additionalProperties: false`, drift direction is critical:
- **Spec documents a field, schema missing it** → **CRITICAL** — `glx validate` will reject valid archives using that field (data loss risk)
- **Schema has a field, spec doesn't document it** → **MAJOR** — undocumented but functional, no data loss

### 8. `$ref` Resolution
- Verify all `$ref` values in schemas point to files that actually exist (e.g., `"$ref": "person.schema.json"`, `"$ref": "vocabularies/event-types.schema.json"`)
- Check for orphaned schemas — schema files with no `$ref` pointing to them and no corresponding spec section

### 9. File-Level Existence
- Every entity type documented in the spec should have a corresponding schema file
- Every schema file should have a corresponding spec section
- Every vocabulary `.glx` file should have a corresponding vocabulary schema
- Flag any mismatches in either direction

### 10. Special Focus Areas
These patterns are known to be complex and drift-prone:
- **Assertion mutual exclusivity** — `allOf`/`anyOf`/`not` constraints enforcing that `property`/`value` and `participant` are mutually exclusive
- **Participant object** — shared nested structure used by Event, Relationship, and Assertion; must stay consistent across all three schemas
- **Temporal property structure** — `value`/`date` object pattern for properties that change over time
- **Evidence requirement** — at least one of `citations`, `sources`, or `media` required on assertions
- **Generic `properties` field** — `additionalProperties: true` on entity properties maps (intentional, not drift)

## Cross-References

- If drift is found in a schema, it likely also affects Go code — see `/check-code-drift` for downstream impact
- For specification-internal issues (contradictions, ambiguity), use `/check-spec` instead
- Schema-related issues found by `/check-spec` should be redirected here

## Output Format

For each entity type, report:

```
## [Entity Type]

✅ No drift detected - Schema matches specification

OR

⚠️ Drift detected - Schema needs updates:

### Required Fields
- Schema missing required field `field_name` documented in specification
- Schema has undocumented required field `field_name` not in specification

### Optional Fields
- Schema missing optional field `field_name` documented in specification
- Schema has undocumented field `field_name` not in specification

### Field Types
- Schema has `field_name` as type X but specification documents it as type Y

### Descriptions
- Schema description for `field_name` doesn't match specification

### Constraints
- Specification documents constraint X but schema doesn't enforce it
- Schema enforces undocumented constraint X
```

**Remember**: Frame all drift as "what the schema needs to change" to match the specification.

## Summary

At the end, provide:
- Total entity types checked
- Count of entity types with drift
- List of schemas that need updates to match specification
- Severity assessment (minor/major/critical)
- Recommended actions: "Update [schema files] to match specification"

## Notes

- **Specification is the source of truth** - schemas should be updated to match it
- Be thorough but practical - focus on structural and semantic differences that could cause confusion or validation issues
- If a field is marked as "required" but has a complex `anyOf` constraint, document this clearly
- Check both directions: specification → schema (missing in schema) AND schema → specification (undocumented in spec)
- **Description comparison guidance**: Acceptable differences include minor rewording, added examples, or extra detail in the schema. Flag differences where the schema description contradicts the spec, omits a key constraint, or describes different behavior
