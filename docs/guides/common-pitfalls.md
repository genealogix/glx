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
    # Claims without citations

# ✅ Correct: Evidence-backed
assertions:
  assertion-birth:
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
    source: source-nonexistent  # Doesn't exist!

# ✅ Correct: Valid reference
sources:
  source-census:
    title: "1851 Census"

citations:
  citation-good:
    source: source-census  # Exists
```

Validation catches this:
```bash
glx validate
# ERROR: citations[citation-bad].source references non-existent sources: source-nonexistent
```

### Missing Confidence Levels

**Problem:** Not expressing certainty about conclusions

```yaml
# ❌ Missing confidence assessment
assertions:
  assertion-birth:
    subject: person-john
    claim: birth_date
    value: "1850-01-15"
    citations: [citation-single-source]

# ✅ Correct: Express confidence
assertions:
  assertion-birth:
    subject: person-john
    claim: birth_date
    value: "1850-01-15"
    confidence: high  # Multiple corroborating sources
    citations: [citation-birth-cert, citation-baptism, citation-census]
```

Use assertion confidence levels (high, medium, low, disputed) rather than citation quality ratings for expressing certainty.

## File Format Issues

### Wrong Field Names

**Problem:** Using `_id` suffix on reference fields

```yaml
# ❌ Wrong: Old naming
citations:
  citation-bad:
    source_id: source-census  # Wrong field name

# ✅ Correct: Singular names
citations:
  citation-good:
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
     name: "John"  # Wrong indentation

# ✅ Correct: Consistent
persons:
  person-john:
    properties:
      given_name: "John"
      family_name: "Smith"
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
    name: "Leeds"
    parent: place-leeds  # Can't be own parent!

# ✅ Correct: Proper hierarchy
places:
  place-yorkshire:
    name: "Yorkshire"

  place-leeds:
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
