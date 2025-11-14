---
title: Common Pitfalls
description: Avoid common mistakes when working with GENEALOGIX archives.
layout: doc
---

# Common Pitfalls Guide

Avoid common mistakes when working with GENEALOGIX archives.

## Evidence Issues

### Missing Evidence Chains

**Problem:** Claims without supporting evidence

```yaml
# ❌ Wrong: No evidence
persons:
  person-john:
    version: "1.0"
    # Claims without citations

# ✅ Correct: Evidence-backed
assertions:
  assertion-birth:
    version: "1.0"
    subject: person-john
    claim: birth_date
    value: "1850-01-15"
    citations: [citation-birth-cert]
```

### Broken References

**Problem:** References to non-existent entities

```yaml
# ❌ Wrong: Reference doesn't exist
citations:
  citation-bad:
    version: "1.0"
    source: source-nonexistent  # Doesn't exist!

# ✅ Correct: Valid reference
sources:
  source-census:
    version: "1.0"
    title: "1851 Census"

citations:
  citation-good:
    version: "1.0"
    source: source-census  # Exists
```

Validation catches this:
```bash
glx validate
# ERROR: citations[citation-bad].source references non-existent sources: source-nonexistent
```

### Incorrect Quality Ratings

Use appropriate quality ratings:
- **3**: Original records (birth certificates)
- **2**: Later records (census)
- **1**: Family records (Bibles)
- **0**: Compiled records (genealogies)

## File Format Issues

### Wrong Field Names

**Problem:** Using `_id` suffix on reference fields

```yaml
# ❌ Wrong: Old naming
citations:
  citation-bad:
    version: "1.0"
    source_id: source-census  # Wrong field name

# ✅ Correct: Singular names
citations:
  citation-good:
    version: "1.0"
    source: source-census  # Correct field name
```

### Invalid ID Formats

**Problem:** IDs that don't follow the pattern

```yaml
# ❌ Wrong: Invalid formats
id: john-smith  # Missing type prefix
id: person_123  # Underscore not allowed

# ✅ Correct: Valid format
id: person-a1b2c3d4  # Alphanumeric and hyphens
```

IDs must be alphanumeric with hyphens, 1-64 characters.

### Wrong File Extensions

All files must use `.glx` extension:

```bash
# ❌ Wrong
person.yaml
event.txt

# ✅ Correct
person.glx
event.glx
```

## YAML Syntax Issues

### Indentation

Use consistent spacing (not tabs):

```yaml
# ❌ Wrong: Inconsistent
persons:
  person-john:
    version: "1.0"
     name: "John"  # Wrong indentation

# ✅ Correct: Consistent
persons:
  person-john:
    version: "1.0"
    concluded_identity:
      primary_name: "John"
```

### Quoting

Quote strings with special characters:

```yaml
# ✅ Correct quoting
date: "1850-01-15"
name: "O'Connor"
notes: "Contains: special chars"
```

## Reference Issues

### Circular References

**Problem:** Self-referencing entities

```yaml
# ❌ Wrong: Self-reference
places:
  place-leeds:
    version: "1.0"
    name: "Leeds"
    parent: place-leeds  # Can't be own parent!

# ✅ Correct: Proper hierarchy
places:
  place-yorkshire:
    version: "1.0"
    name: "Yorkshire"

  place-leeds:
    version: "1.0"
    name: "Leeds"
    parent: place-yorkshire
```

### Duplicate IDs

**Problem:** Same ID used multiple times

Validation will fail:
```bash
glx validate
# ERROR: duplicate persons ID: person-john
```

Ensure all IDs are unique within their entity type.

## Git Workflow Issues

### Unvalidated Commits

Always validate before committing:

```bash
# ✅ Correct workflow
glx validate
git add .
git commit -m "Add validated data"
```

### Poor Commit Messages

Write descriptive messages:

```bash
# ❌ Wrong
git commit -m "update"

# ✅ Correct
git commit -m "Add 1851 Census evidence for Smith family

- John Smith: occupation, residence
- Source: HO107, Piece 2319"
```

## Troubleshooting

### Validation Failures

When `glx validate` fails:

1. Check file extensions (must be `.glx`)
2. Verify ID formats (alphanumeric + hyphens)
3. Check all references exist
4. Review field names (use singular: `source`, not `source_id`)

### Common Errors

```bash
# Missing required field
ERROR: version is required
FIX: Add version: "1.0"

# Broken reference
ERROR: citations[citation-1].source references non-existent sources: source-missing
FIX: Create the source or fix the reference

# Invalid ID
ERROR: persons[john_smith]: invalid entity ID
FIX: Use person-a1b2c3d4 format

# Duplicate ID
ERROR: duplicate persons ID: person-john
FIX: Use unique IDs
```

## See Also

- [Best Practices](best-practices.md) - Recommended workflows
- [Entity Types](../../specification/4-entity-types/README.md) - Entity specifications
- [CLI Documentation](../../glx/README.md) - Command reference
