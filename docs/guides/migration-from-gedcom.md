---
title: Migration from GEDCOM
description: Guide for converting GEDCOM files to GENEALOGIX format.
layout: doc
---

# Migration from GEDCOM Guide

Guide for converting GEDCOM files to GENEALOGIX format.

## Key Differences

| Aspect | GEDCOM | GENEALOGIX |
|--------|--------|------------|
| **Format** | Tag-based | YAML |
| **Evidence** | Basic sources | Complete chains |
| **Quality** | QUAY (0-3) | Quality (0-3) |
| **Version Control** | File-based | Git-native |

## Migration Process

### 1. Initialize Archive

```bash
mkdir my-family-archive
cd my-family-archive
glx init
git init
git add .
git commit -m "Initial: GENEALOGIX archive structure"
```

### 2. Convert Entities

**Manual conversion process:**

1. Extract individuals → `persons/`
2. Extract families → `relationships/`
3. Extract events → `events/`
4. Extract places → `places/`
5. Extract sources → `sources/`

### 3. Create Evidence Chains

GEDCOM sources need to be expanded into complete evidence chains.

**GEDCOM:**
```
0 @I1@ INDI
1 BIRT
2 DATE 15 JAN 1850
2 SOUR @S1@
3 QUAY 2
3 PAGE Page 23
```

**GENEALOGIX:**
```yaml
sources:
  source-birth-cert:
    title: "Birth Certificate"
    type: vital_record

citations:
  citation-birth:
    source: source-birth-cert
    quality: 2
    page: "Page 23"

assertions:
  assertion-birth-date:
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    citations: [citation-birth]
```

## Field Mapping

### Individual Records (INDI)

| GEDCOM | GENEALOGIX | Notes |
|--------|------------|-------|
| `INDI` | Person | Core entity |
| `NAME` | Person.properties.primary_name | Structured format |
| `BIRT` | Event (birth) | Separate entity |
| `DEAT` | Event (death) | Separate entity |
| `OCCU` | Event (occupation) | Separate entity |

### Family Records (FAM)

| GEDCOM | GENEALOGIX | Notes |
|--------|------------|-------|
| `FAM` | Relationship | Family connections |
| `HUSB`/`WIFE` | Relationship.participants | Role-based |
| `MARR` | Event (marriage) | Separate entity |

### Source Records (SOUR)

| GEDCOM | GENEALOGIX | Notes |
|--------|------------|-------|
| `SOUR` | Source | Original material |
| `TITL` | Source.title | Title |
| `REPO` | Source.repository | Repository reference |
| `PAGE` | Citation.page | Moved to citation |
| `QUAY` | Citation.quality | 1:1 mapping |

## Quality Translation

GEDCOM QUAY maps 1:1 to GENEALOGIX citation quality field (maintained for compatibility). However, the idiomatic GLX approach is to use assertion confidence levels instead:

```yaml
# GEDCOM QUAY preserved in citation (optional)
citations:
  citation-from-gedcom:
    source: source-census
    quality: 2  # From GEDCOM QUAY

# Idiomatic GLX: Use assertion confidence
assertions:
  assertion-birth:
    subject: person-john
    claim: birth_date
    value: "1850-01-15"
    confidence: high  # Preferred approach
    citations: [citation-from-gedcom]
```

See [Confidence Levels](../../specification/5-standard-vocabularies/confidence-levels.glx) for the vocabulary.

## Common Challenges

### Name Conversion

**GEDCOM:**
```
1 NAME John /Smith/
```

**GENEALOGIX:**
```yaml
properties:
  primary_name: "John Smith"
```

### Place Hierarchy

**GEDCOM:**
```
2 PLAC Leeds, Yorkshire, England
```

**GENEALOGIX:**
```yaml
places:
  place-england:
    name: "England"
    type: country

  place-yorkshire:
    name: "Yorkshire"
    type: county
    parent: place-england

  place-leeds:
    name: "Leeds"
    type: city
    parent: place-yorkshire
```

### Date Formats

**GEDCOM:**
```
2 DATE 15 JAN 1850
2 DATE ABT 1850
2 DATE BET 1849 AND 1851
```

**GENEALOGIX:**
```yaml
date: "1850-01-15"
date: "1850?"
date: "1849/1851"
```

## Post-Migration

### Validation

```bash
# Validate converted data
glx validate

# Fix any errors
# Re-validate
glx validate
```

### Enhancement

After migration, enhance evidence quality:
- Add transcriptions
- Verify quality ratings
- Complete evidence chains
- Add research notes

### Git Tracking

```bash
git add .
git commit -m "Migration from GEDCOM complete

Migrated:
- 150 individuals → persons/
- 45 families → relationships/
- 200 events → events/
- 50 sources → sources/

Next steps:
- Enhance evidence quality
- Add transcriptions
- Complete place hierarchy"
```

## See Also

- [Entity Types](../../specification/4-entity-types/README.md) - Entity specifications
- [Best Practices](best-practices.md) - Workflow recommendations
- [CLI Documentation](../../glx/README.md) - Command reference
