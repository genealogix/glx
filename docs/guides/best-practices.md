---
title: Best Practices
description: Practical recommendations for maintaining GENEALOGIX archives.
layout: doc
---

# Best Practices Guide

Practical recommendations for maintaining GENEALOGIX archives.

## Evidence Documentation

### Complete Evidence Chains

Link assertions to their supporting evidence:

```yaml
citations:
  citation-birth-cert:
    version: "1.0"
    source: source-birth-register
    quality: 3
    transcription: "John Smith, born January 15, 1850"

assertions:
  assertion-john-birth:
    version: "1.0"
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    citations: [citation-birth-cert]
    confidence: high
```

### Quality Assessment

Use the 0-3 quality scale consistently:
- **3**: Original records created at time of event
- **2**: Records created later from originals
- **1**: Contemporary accounts, family records
- **0**: Compiled records, oral history

See [Citation Entity](../../specification/4-entity-types/citation.md#evidence-quality) for details.

### Transcribe Key Evidence

Include transcriptions for important sources:

```yaml
citations:
  citation-parish:
    version: "1.0"
    source: source-parish-register
    quality: 3
    transcription: |
      "January 20th, 1850. John, son of Thomas Smith, blacksmith,
      and Mary Smith, of 23 Wellington Street. Born January 15th."
```

## Git Workflow

### Validation Before Commit

Always validate before committing:

```bash
# Validate entire archive
glx validate

# Validate specific changes
glx validate persons/ events/
```

### Descriptive Commits

Write clear commit messages:

```bash
# Good commit message
git commit -m "Add 1851 Census evidence for Smith family

- John Smith: occupation (blacksmith), residence
- Mary Smith: age, birthplace
- Source: HO107, Piece 2319, Yorkshire
- Quality: 2 (secondary source)"
```

### Branch for Research

Use branches for research investigations:

```bash
git checkout -b research/1851-census
git checkout -b evidence/vital-records
```

## Conflicting Evidence

Document resolution when sources conflict:

```yaml
assertions:
  assertion-birth-date:
    version: "1.0"
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    confidence: medium
    notes: |
      Birth certificate: Jan 15 (quality 3) - preferred
      Baptism record: Jan 20 (quality 3) - 5 day delay typical
      Census age: supports 1850 (quality 2)
    citations:
      - citation-birth-cert
      - citation-baptism
      - citation-census
```

## File Organization

GENEALOGIX is flexible about file organization. Choose what works for your workflow:

**One-entity-per-file** (good for collaboration):
```
persons/person-john.glx
events/event-birth.glx
```

**Single file** (good for small archives):
```
family.glx  # All entities in one file
```

**Hybrid** (logical groupings):
```
core-family.glx
sources/vital-records.glx
```

What matters:
- Use `.glx` extension
- Correct entity type prefixes on IDs
- Valid references between entities

## ID Generation

Generate IDs systematically:

```bash
# Random hex (recommended for collaboration)
person-a1b2c3d4
event-b2c3d4e5

# Command line generation
echo "person-$(openssl rand -hex 4)"
```

Avoid human-readable IDs that may collide during collaboration.

## Validation

Run validation regularly:

```bash
# During active research
glx validate

# Before commits
glx validate
git add .
git commit -m "..."

# After merges
git merge research-branch
glx validate
```

## See Also

- [Common Pitfalls](common-pitfalls.md) - Avoid common mistakes
- [Entity Types](../../specification/4-entity-types/README.md) - Entity specifications
- [CLI Documentation](../../glx/README.md) - Command reference
