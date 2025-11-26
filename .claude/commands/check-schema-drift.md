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

## Entity Types to Check

- assertion
- citation
- event
- media
- person
- place
- relationship
- repository
- source

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

### 5. Special Constraints
- Check for complex validation rules (patterns, minItems, anyOf, allOf, not)
- Verify these are documented in the markdown
- Look for undocumented constraints in schemas

### 6. Entity ID Patterns
- Verify entity ID pattern constraints match between docs and schemas
- Check that the pattern `^[a-zA-Z0-9-]{1,64}$` is consistently applied where needed

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
- Be thorough but practical - minor wording differences in descriptions are acceptable
- Focus on structural and semantic differences that could cause confusion or validation issues
- If a field is marked as "required" but has a complex `anyOf` constraint, document this clearly
- Check both directions: specification → schema (missing in schema) AND schema → specification (undocumented in spec)
