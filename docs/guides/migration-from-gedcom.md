---
title: Migration from GEDCOM
description: Guide for converting GEDCOM files to GENEALOGIX format
layout: doc
---

# Migration from GEDCOM Guide

This guide helps you convert existing GEDCOM files to GENEALOGIX format while preserving evidence and maintaining data quality.

## Understanding the Differences

### GEDCOM vs GENEALOGIX

| Aspect | GEDCOM | GENEALOGIX |
|--------|--------|------------|
| **Format** | Custom binary-like | YAML (human-readable) |
| **Structure** | Tag-based hierarchy | Entity-relationship |
| **Evidence** | Basic source records | Complete evidence chains |
| **Quality** | QUAY field | Structured quality (0-3) |
| **Version Control** | File-based | Git-native |
| **Validation** | Syntax only | Schema validation |

### Evidence Model Comparison

**GEDCOM Evidence:**
```
0 SOUR @S1@
1 QUAY 2
1 PAGE Page 23
```

**GENEALOGIX Evidence:**
```yaml
# Complete evidence chain
sources/source-census.glx:
  title: 1851 England Census
  type: census

citations/citation-schedule.glx:
  source: source-census
  locator: "HO107, Piece 2319, Folio 234, Page 23, Schedule 145"
  quality: 2  # Secondary source
  transcription: "John Smith, Head, 25, Blacksmith"

assertions/assertion-occupation.glx:
  subject: person-john-smith
  claim: occupation
  value: blacksmith
  citations: [citation-schedule]
```

## Migration Process

### 1. Pre-Migration Assessment

**Evaluate your GEDCOM file:**
```bash
# Check GEDCOM structure
head -50 family.ged

# Count records
grep "^0" family.ged | wc -l

# Check for evidence
grep "SOUR" family.ged | wc -l

# Look for quality indicators
grep "QUAY" family.ged | wc -l
```

**Plan migration strategy:**
- Identify high-quality records to migrate first
- Note records needing additional research
- Plan evidence chain completion

### 2. Basic Structure Migration

**Create GENEALOGIX repository:**
```bash
# Initialize repository
mkdir my-family-archive
cd my-family-archive
glx init
git init
git add .
git commit -m "Initial: Set up GENEALOGIX archive structure"
```

**Convert basic entities:**
```bash
# Use migration tools (when available)
# glx migrate from-gedcom ../family.ged

# Or manual conversion process:
# 1. Extract individuals → persons/
# 2. Extract families → relationships/
# 3. Extract events → events/
# 4. Extract places → places/
```

### 3. Evidence Migration

**Convert GEDCOM sources to evidence chains:**

**GEDCOM Source:**
```
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Leeds, Yorkshire, England
2 SOUR @S1@
3 QUAY 2
3 PAGE Page 23
0 @S1@ SOUR
1 TITL Birth Certificate
1 REPO @R1@
```

**GENEALOGIX Evidence Chain:**
```yaml
# Repository
repositories/repository-gro.glx:
  repositories:
    repository-gro:
      version: "1.0"
      name: General Register Office
      location: London, England

# Source
sources/source-birth-cert.glx:
  sources:
    source-birth-cert:
      version: "1.0"
      title: Birth Certificate
      type: vital_record
      repository: repository-gro

# Citation
citations/citation-birth-page23.glx:
  citations:
    citation-birth-page23:
      version: "1.0"
      source: source-birth-cert
      locator: "Page 23"
      quality: 2  # GEDCOM QUAY 2 = secondary evidence

# Person with evidence
persons/person-john-smith.glx:
  persons:
    person-john-smith:
      version: "1.0"
      name:
        given: John
        surname: Smith
        display: John Smith

# Assertion (not direct field)
assertions/assertion-john-birth.glx:
  assertions:
    assertion-john-birth:
      version: "1.0"
      subject: person-john-smith
      claim: born_on
      value: "1850-01-15"
      citations: [citation-birth-page23]
```

## GEDCOM Field Mapping

### Individual Records (INDI)

| GEDCOM Field | GENEALOGIX Entity | Notes |
|--------------|-------------------|-------|
| `INDI` | Person | Core biographical information |
| `NAME` | Person.name | Structured name format |
| `BIRT` | Event (birth) | Separate event entity |
| `DEAT` | Event (death) | Separate event entity |
| `BURI` | Event (burial) | Separate event entity |
| `OCCU` | Event (occupation) | Employment events |
| `RESI` | Event (residence) | Residence events |

### Family Records (FAM)

| GEDCOM Field | GENEALOGIX Entity | Notes |
|--------------|-------------------|-------|
| `FAM` | Relationship | Family connections |
| `HUSB` | Relationship.participants | Role-based participants |
| `WIFE` | Relationship.participants | Role-based participants |
| `CHIL` | Relationship.participants | Role-based participants |
| `MARR` | Event (marriage) | Separate marriage event |

### Source Records (SOUR)

| GEDCOM Field | GENEALOGIX Entity | Notes |
|--------------|-------------------|-------|
| `SOUR` | Source | Original material |
| `TITL` | Source.title | Title or name |
| `AUTH` | Source.creator | Author/creator |
| `PUBL` | Source.publication_info | Publication details |
| `REPO` | Source.repository | Where source is held |

### Citation Fields

| GEDCOM Field | GENEALOGIX Entity | Notes |
|--------------|-------------------|-------|
| `SOUR` | Citation.source | Reference to source |
| `PAGE` | Citation.locator | Location within source |
| `QUAY` | Citation.quality | Evidence quality |
| `TEXT` | Citation.transcription | Source text |
| `NOTE` | Citation.notes | Additional notes |

## Quality Translation

### GEDCOM QUAY to GENEALOGIX Quality

| GEDCOM QUAY | GENEALOGIX Quality | Description |
|-------------|-------------------|-------------|
| `0` | 0 | Unreliable evidence |
| `1` | 1 | Questionable reliability |
| `2` | 2 | Secondary evidence, reliable |
| `3` | 3 | Primary evidence, direct |

**Translation examples:**
```yaml
# GEDCOM: 2 SOUR @S1@ 3 QUAY 2
citations/citation-example.glx:
  source: source-example
  quality: 2  # Secondary evidence
  notes: "GEDCOM QUAY: 2"

# GEDCOM: 2 SOUR @S1@ 3 QUAY 3
citations/citation-primary.glx:
  source: source-primary
  quality: 3  # Primary evidence
  notes: "GEDCOM QUAY: 3"
```

## Migration Tools

### Available Tools

**When tools are available:**
```bash
# Automated conversion (future)
glx migrate from-gedcom family.ged

# Convert specific sections
glx migrate from-gedcom family.ged --individuals
glx migrate from-gedcom family.ged --families
glx migrate from-gedcom family.ged --sources

# Validate converted data
glx validate
```

### Manual Migration Process

**Step 1: Extract Individuals**
```bash
# Extract individual records
grep "^0 @I" family.ged | cut -d'@' -f2 > individuals.txt

# For each individual, create person file
# Convert NAME, BIRT, DEAT, etc.
```

**Step 2: Extract Sources**
```bash
# Extract source records
grep "^0 @S" family.ged | cut -d'@' -f2 > sources.txt

# Create source entities
# Convert TITL, AUTH, PUBL, etc.
```

**Step 3: Create Evidence Chains**
```bash
# Link citations to sources
# Convert QUAY to quality ratings
# Add transcriptions where available
```

## Common Migration Challenges

### 1. Name Format Conversion

**GEDCOM names:**
```
1 NAME John /Smith/
1 NAME Mary Anne /O'Connor/
1 NAME Juan /García López/
```

**GENEALOGIX names:**
```yaml
# Simple name
name:
  given: John
  surname: Smith
  display: John Smith

# Complex name
name:
  given: Mary Anne
  surname: O'Connor
  display: Mary Anne O'Connor

# Multi-part surname
name:
  given: Juan
  surname: García López
  display: Juan García López
```

### 2. Place Hierarchy

**GEDCOM places:**
```
2 PLAC Leeds, Yorkshire, England
2 PLAC New York City, New York, USA
```

**GENEALOGIX places:**
```yaml
# Hierarchical structure
places/place-england.glx:
  name: England
  type: country

places/place-yorkshire.glx:
  name: Yorkshire
  type: county
  parent: place-england

places/place-leeds.glx:
  name: Leeds
  type: city
  parent: place-yorkshire

# Usage in events
events/event-birth.glx:
  place: place-leeds  # Reference, not text
```

### 3. Date Standardization

**GEDCOM dates:**
```
2 DATE 15 JAN 1850
2 DATE ABT 1850
2 DATE BET 1849 AND 1851
2 DATE BEF 1860
```

**GENEALOGIX dates:**
```yaml
# Standard formats
date: "1850-01-15"     # Complete date
date: "1850"          # Year only
date: "1849/1851"     # Between years
date: "1850?"         # Uncertain
```

### 4. Evidence Completeness

**Problem:** GEDCOM sources without citations
```
0 @I1@ INDI
1 BIRT
2 DATE 1850
2 SOUR @S1@  # Source attached but no citation details
```

**Solution:** Create basic citation structure
```yaml
# Basic citation from minimal GEDCOM data
citations/citation-basic.glx:
  source: source-basic
  quality: 2  # Default for GEDCOM sources
  notes: |
    Migrated from GEDCOM source.
    Additional research needed for:
    - Specific page/entry numbers
    - Transcription of relevant text
    - Quality assessment verification
```

## Post-Migration Cleanup

### 1. Validation and Quality Check

**Validate migrated data:**
```bash
# Check all files
glx validate

# Check specific directories
glx validate persons/
glx validate sources/ citations/

# Compare with original GEDCOM
# Verify all individuals migrated
# Check all sources converted
```

### 2. Evidence Enhancement

**Improve evidence quality:**
```bash
# Research missing details
# Add transcriptions where missing
# Upgrade quality ratings with better sources
# Complete evidence chains

# Example enhancement
citations/citation-enhanced.glx:
  source: source-birth-certificate
  locator: "Certificate #1850-LEEDS-00145"
  quality: 3  # Upgraded from GEDCOM QUAY 2
  transcription: |
    "Registration District: Leeds
    Birth: January 15, 1850
    Name: John Smith
    Father: Thomas Smith, Blacksmith
    Mother: Mary Smith, formerly Brown"
```

### 3. Git Integration

**Track migration process:**
```bash
# Commit migration steps
git add .
git commit -m "Migrate from GEDCOM family.ged

Migration completed:
- 150 individuals → persons/
- 45 families → relationships/
- 200 events → events/
- 50 sources → sources/
- Basic citations created

TODO:
- Enhance evidence quality ratings
- Add transcriptions for key sources
- Complete place hierarchy
- Validate all evidence chains"

# Create migration branch for cleanup
git checkout -b migration-cleanup
# ... enhance evidence quality ...
git commit -m "Enhanced evidence quality

- Upgraded 20 citations from quality 2 to 3
- Added transcriptions for vital records
- Completed evidence chains for birth events
- Added research notes for uncertain data"
```

## Migration Best Practices

### 1. Incremental Migration

**Migrate in phases:**
```bash
# Phase 1: Basic structure
glx migrate from-gedcom family.ged --basic
git commit -m "Phase 1: Basic migration complete"

# Phase 2: Evidence chains
glx migrate from-gedcom family.ged --evidence
git commit -m "Phase 2: Evidence chains added"

# Phase 3: Quality enhancement
# Manual research and quality improvements
git commit -m "Phase 3: Evidence quality enhanced"
```

### 2. Quality Preservation

**Document original GEDCOM quality:**
```yaml
# Track original GEDCOM quality
sources/source-gedcom-original.glx:
  title: Original GEDCOM Import
  type: digital_file
  notes: |
    Migrated from family.ged on 2024-03-15
    Original QUAY ratings preserved in citation notes
    Evidence enhancement needed for research standards

citations/citation-from-gedcom.glx:
  source: source-gedcom-original
  quality: 2  # From GEDCOM QUAY 2
  notes: "Original GEDCOM QUAY: 2, Page: 23"
```

### 3. Research Notes

**Document migration decisions:**
```yaml
assertions/assertion-migrated.glx:
  research_notes: |
    Migrated from GEDCOM on 2024-03-15
    Original: 2 DATE 15 JAN 1850, 2 SOUR @S1@, 3 QUAY 2
    Quality upgraded from 2 to 3 based on source verification
    Additional research confirmed birth certificate exists
```

## Migration Validation

### 1. Completeness Check

**Verify all data migrated:**
```bash
# Count entities
ls -1 persons/ | wc -l    # Should match GEDCOM individuals
ls -1 sources/ | wc -l    # Should match GEDCOM sources
ls -1 events/ | wc -l     # Should match GEDCOM events

# Check for broken references
glx validate

# Verify evidence coverage
# Count assertions with citations
# Compare with GEDCOM QUAY usage
```

### 2. Quality Assessment

**Compare quality before/after:**
```yaml
migration_quality:
  gedcom_sources: 50
  quality_distribution:
    quay_0: 5   # 10%
    quay_1: 10  # 20%
    quay_2: 25  # 50%
    quay_3: 10  # 20%

  genealogix_quality:
    quality_0: 3   # 6% (reduced)
    quality_1: 8   # 16% (similar)
    quality_2: 20  # 40% (similar)
    quality_3: 19  # 38% (increased - research enhancement)
```

### 3. Evidence Chain Validation

**Verify evidence completeness:**
```bash
# Check for assertions without citations
glx validate --strict

# Verify citation-source links
glx validate citations/ sources/

# Check repository references
glx validate sources/ repositories/
```

## Advanced Migration Topics

### 1. Custom Event Types

**Handle GEDCOM custom events:**
```yaml
# GEDCOM: 1 EVEN Graduation
events/event-graduation.glx:
  event_type: education_graduation
  date: "1870"
  place: place-leeds-university
  description: "Graduated from Leeds Mechanics Institute"
```

### 2. Complex Relationships

**Migrate non-standard relationships:**
```yaml
# GEDCOM: 1 FAMC @F2@ (adoptive family)
relationships/rel-adoptive-family.glx:
  relationship_type: adoption
  participants:
    - person: person-child
      role: adopted_child
    - person: person-parent
      role: adoptive_parent
```

### 3. Media Files

**Migrate associated media:**
```yaml
# GEDCOM: 1 OBJE @O1@
media/media-birth-cert.jpg:
  # File: media-a1b2c3d4.jpg
  # Link to citation: citation-birth-cert
```

## Troubleshooting Migration

### Common Issues

**1. Character encoding problems:**
```bash
# Check GEDCOM encoding
file family.ged

# Convert if needed
iconv -f iso-8859-1 -t utf-8 family.ged > family-utf8.ged
```

**2. Malformed GEDCOM:**
```bash
# Validate GEDCOM structure
# Look for unmatched tags
# Check level consistency
```

**3. Missing required fields:**
```bash
# GENEALOGIX requires IDs, versions, types
# GEDCOM may have missing data
# Add research notes for missing information
```

### Migration Tools (Future)

When available, use automated tools:
```bash
# Complete migration
glx migrate from-gedcom family.ged

# Interactive migration
glx migrate from-gedcom family.ged --interactive

# Batch migration with quality enhancement
glx migrate from-gedcom *.ged --batch --enhance-quality

# Validate migration completeness
glx validate --migration-report
```

## Post-Migration Workflow

### 1. Enhanced Research

**Improve evidence quality:**
```bash
# Research original sources
# Upgrade quality ratings
# Add transcriptions
# Complete evidence chains

# Example enhancement
citations/citation-enhanced.glx:
  source: source-original-document
  locator: "Document #1850-LEEDS-00145"
  quality: 3  # Upgraded with research
  transcription: "Full document text..."
  research_notes: "Located original at Leeds Library, verified 2024-03-15"
```

### 2. Git History Integration

**Preserve research timeline:**
```bash
# Create migration history
git log --oneline --grep="migration" --grep="GEDCOM"

# Track research progress
git log --since="migration" --oneline

# Compare quality improvements
git diff HEAD migration-complete -- citations/
```

### 3. Collaboration Setup

**Prepare for collaborative research:**
```bash
# Set up for team research
git checkout -b research/vital-records
git checkout -b research/census-data
git checkout -b research/place-verification

# Create research plan
# Assign tasks via GitHub Issues
# Track progress with branches
```

Migration from GEDCOM to GENEALOGIX provides an opportunity to enhance evidence quality, complete research gaps, and establish better research practices for the future.
