---
title: Migration from GEDCOM
description: Guide for converting GEDCOM files to GENEALOGIX format.
layout: doc
---

# Migration from GEDCOM Guide

Guide for converting GEDCOM files to GENEALOGIX format using the automated import tool or manual conversion.

**Note:** Automated GEDCOM import is now available! Use `glx import` to convert GEDCOM 5.5.1 and GEDCOM 7.0 files automatically. See [CLI Documentation](../../glx/README.md) for usage details.

## Key Differences

| Aspect | GEDCOM | GENEALOGIX |
|--------|--------|------------|
| **Format** | Tag-based | YAML |
| **Evidence** | Basic sources | Complete chains |
| **Version Control** | File-based | Git-native |

## Migration Process

### Automated Import (Recommended)

Convert GEDCOM files automatically using the GLX CLI:

```bash
# Import GEDCOM file
glx import family.ged -o family-archive.glx

# The import command handles:
# - Individual (INDI) → Person entities
# - Family (FAM) → Relationship entities
# - Events (BIRT, DEAT, MARR, etc.) → Event entities
# - Sources (SOUR) → Source entities + Citations + Assertions
# - Places (PLAC) → Hierarchical Place entities
# - Notes (NOTE/SNOTE) → Entity notes

# Initialize git tracking
git init
git add family-archive.glx
git commit -m "Import from GEDCOM: family.ged"
```

**Supported GEDCOM Versions:**
- ✅ GEDCOM 5.5.1 (full support)
- ✅ GEDCOM 7.0 (full support)

**What Gets Imported:**
- 31+ person attributes and events
- All family relationships (marriage, parent-child)
- **PEDI (pedigree) types**: Biological, adoptive, foster parent-child relationships
- Evidence chains (SOUR → Citation → Assertion)
- Place hierarchies (flat → hierarchical)
- **ADDR subfields**: Full address preservation and place hierarchy fallback
- Shared and inline notes
- Source citations with locators and transcriptions

For implementation details, see [GEDCOM Import Developer Docs](../development/gedcom-import.md).

### Manual Conversion Process

If you prefer manual conversion or need to customize the import:

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
| `NAME` | Person.properties.name | Unified name with optional fields |
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
| `QUAY` | Citation.notes | Preserved in notes |

## Common Challenges

### Name Conversion

**GEDCOM:**
```
1 NAME John /Smith/
```

**GENEALOGIX:**
```yaml
properties:
  name:
    value: "John Smith"
    fields:
      given: "John"
      surname: "Smith"
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
2 DATE FROM 1900 TO 1950
```

**GENEALOGIX:**
```yaml
date: "1850-01-15"
date: "ABT 1850"
date: "BET 1849 AND 1851"
date: "FROM 1900 TO 1950"
```

GLX uses YYYY-MM-DD format for exact dates and preserves GEDCOM keywords (ABT, BEF, AFT, CAL, BET, FROM, TO, AND) for qualified and range dates. See [Data Types](../../specification/6-data-types.md) for complete date format specification.

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
- Complete evidence chains
- Add research notes
- Set assertion confidence levels

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

## Testing Your Import

After importing, validate the results:

```bash
# Validate the imported archive
glx validate family-archive.glx

# Check what was imported
# - Count entities
# - Verify relationships
# - Review evidence chains
```

**Common Import Results:**
- Large family trees: 100-1000+ persons
- Comprehensive events: 2-3x person count
- Relationships: 1-2x person count
- Evidence chains automatically created from GEDCOM SOUR tags

## See Also

- [GEDCOM Import Developer Documentation](../development/gedcom-import.md) - Implementation details
- [Entity Types](../../specification/4-entity-types/README.md) - Entity specifications
- [Best Practices](best-practices.md) - Workflow recommendations
- [CLI Documentation](../../glx/README.md) - Command reference
