---
description: Identify drift between specification markdown files and JSON schemas
---

You are tasked with identifying any drift between the GLX specification markdown files and the JSON schemas.

## Task

Analyze all entity types and compare:

1. **specification/4-entity-types/*.md** - The human-readable specification
2. **specification/schema/v1/*.schema.json** - The machine-readable schemas

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

✅ No drift detected

OR

⚠️ Drift detected:

### Required Fields
- [Issue description]

### Optional Fields
- [Issue description]

### Field Types
- [Issue description]

### Descriptions
- [Issue description]

### Constraints
- [Issue description]
```

## Summary

At the end, provide:
- Total entity types checked
- Count of entity types with drift
- List of entity types that need attention
- Severity assessment (minor/major)

## Notes

- Be thorough but practical - minor wording differences in descriptions are acceptable
- Focus on structural and semantic differences that could cause confusion or validation issues
- If a field is marked as "required" but has a complex `anyOf` constraint, document this clearly
- Check both directions: markdown → schema AND schema → markdown
