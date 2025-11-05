---
title: Best Practices
description: Recommended workflows and patterns for GENEALOGIX research.
layout: doc
---

# Best Practices Guide

This guide provides recommended workflows and conventions for maintaining high-quality GENEALOGIX archives.

## Research Workflow

### 1. Evidence-First Approach

**Always start with sources, not conclusions:**

```bash
# ❌ Wrong: Start with assumptions without evidence

# ✅ Correct: Evidence-based approach

# sources/birth-cert.glx - Document the source first
sources:
  source-a1b2c3d4:
    version: "1.0"
    title: Birth Certificate
    type: vital_record

# citations/birth-citation.glx - Cite specific evidence
citations:
  citation-12345678:
    version: "1.0"
    source: source-a1b2c3d4
    quality: 3

# assertions/birth-assertion.glx - Then make evidence-based claims
assertions:
  assertion-abc12345:
    version: "1.0"
    subject: person-a1b2c3d4
    claim: born_on
    value: "1850-01-15"
    citations: [citation-12345678]
    confidence: high
```

### 2. Complete Evidence Chains

**Every assertion needs a complete evidence chain:**

```yaml
# Complete chain example
repositories/repository-gro.glx:
  name: General Register Office
  location: London

sources/source-birth-register.glx:
  title: Birth Register 1850
  repository: repository-gro

citations/citation-entry-145.glx:
  source: source-birth-register
  locator: "Entry 145, Page 23"
  quality: 3

assertions/assertion-john-born.glx:
  citations: [citation-entry-145]
  claim: "John Smith was born January 15, 1850"
```

## File Organization

### 1. Directory Structure Recommendations

GENEALOGIX is flexible about file organization. You can use any structure that works for your workflow. The example below shows one common pattern (one-entity-per-file with dedicated directories), but many other approaches are equally valid.

**Example: One-entity-per-file organization (good for collaboration):**
```
family-archive/
├── persons/           # Person entities
├── relationships/     # Relationship entities
├── events/           # Event entities
├── places/           # Place entities
├── sources/          # Source entities
├── citations/        # Citation entities
├── repositories/     # Repository entities
├── assertions/       # Assertion entities
└── media/           # Media entities
```

**Other valid approaches:**
- **Single file**: All entities in one `family.glx` file (good for small archives)
- **Hybrid**: Mix of single files and directories based on logical groupings
- **Custom**: Any structure that makes sense for your research process

**Key principles that matter:**
- Use `.glx` file extension (required)
- Entity type prefixes on IDs match actual entity types (required)
- All references point to existing entities (required)
- Consistent naming within your chosen structure (recommended)

### 2. ID Management

**Generate IDs systematically:**
```bash
# Good: Structured IDs
person-a1b2c3d4  # Person record
event-b2c3d4e5   # Birth event
place-c3d4e5f6   # Birth place
source-d4e5f6g7  # Birth certificate
citation-e5f6g7h8 # Specific citation

# Avoid: Human-readable but fragile
john-smith-person
john-birth-event
john-birth-place
```

**ID generation methods:**
```bash
# Command line
echo "person-$(openssl rand -hex 4)"

# Python
import secrets
f"person-{secrets.token_hex(4)}"

# Node.js
require('crypto').randomBytes(4).toString('hex')
```

## Quality Standards

### 1. Evidence Quality Guidelines

| Quality | When to Use | Example |
|---------|-------------|---------|
| **3** | Original records created at time of event | Birth certificates, baptism records |
| **2** | Records created later but based on original | Census records, marriage indexes |
| **1** | Family records, contemporary accounts | Family Bibles, letters |
| **0** | Compiled records, oral history | Published genealogies, interviews |

**Quality assessment checklist:**
- Is this the original record or a copy?
- Was it created at the time of the event or later?
- Does it directly state the fact or require inference?
- Are there multiple sources that agree?

### 2. Citation Standards

**Be specific in citations:**
```yaml
# ✅ Good citation
citations/citation-census.glx:
  source: source-1851-census
  locator: "HO107, Piece 2319, Folio 234, Page 23, Schedule 145"
  quality: 2
  transcription: |
    "John Smith, Head, Married, 25, Blacksmith, born Leeds"

# ❌ Vague citation
citations/citation-bad.glx:
  source: source-1851-census
  locator: "somewhere in the census"
  quality: 2
```

**Include transcriptions for important evidence:**
```yaml
citations/citation-parish.glx:
  source: source-st-pauls-register
  locator: "Baptisms 1850, Entry 145"
  quality: 3
  transcription: |
    "January 20th, 1850. John, son of Thomas Smith, blacksmith,
    and Mary Smith, of 23 Wellington Street. Born January 15th."
  notes: |
    5-day delay between birth and baptism is typical for working families
```

## Git Workflow Best Practices

### 1. Branch Strategy

**Use feature branches for research:**
```bash
# Research-specific investigations
git checkout -b research/1851-census
git checkout -b research/vital-records
git checkout -b research/smith-occupation

# Evidence integration
git checkout -b evidence/1850-1900-integration

# Documentation updates
git checkout -b docs/update-place-hierarchy
```

**Branch naming conventions:**
- `research/[topic]` - Research investigations
- `evidence/[period]` - Evidence integration
- `feature/[description]` - New functionality
- `fix/[issue]` - Bug fixes
- `docs/[area]` - Documentation updates

### 2. Commit Message Standards

**Clear, descriptive commit messages:**
```bash
# ✅ Good commit messages
git commit -m "Add John Smith birth evidence

- Birth certificate from GRO (quality 3)
- Parish baptism record (quality 3)
- 1851 Census confirmation (quality 2)
- Confidence: high (multiple primary sources)"

# ❌ Poor commit messages
git commit -m "update"
git commit -m "added stuff"
git commit -m "john smith"
```

**Structure for complex changes:**
```bash
git commit -m "Integrate 1851 Census data for Smith family

Added census evidence for 5 family members:
- John Smith: occupation (blacksmith), residence
- Mary Smith: age, birthplace
- Jane Smith: birth year, school attendance
- Thomas Smith: relationship to head, occupation

Source: HO107, Piece 2319, Yorkshire
Quality: 2 (secondary source)
Validated: All citations reference existing sources"
```

### 3. Validation Before Commit

**Always validate before committing:**
```bash
# Validate entire archive
glx validate

# Check specific directories
glx validate persons/ events/ places/

# Validate specific files
glx validate persons/person-john-smith.glx

# Run full test suite for major changes
cd /path/to/genealogix-spec
glx validate examples/complete-family/
```

## Naming Conventions

### 1. Person Names

**Use structured name format:**
```yaml
# ✅ Consistent format
name:
  given: John
  surname: Smith
  display: John Smith

# Include middle names
name:
  given: John Henry
  surname: Smith
  display: John Henry Smith

# Handle name changes
name_changes:
  - date: "1880-03-15"
    former: Mary Brown
    new: Mary Smith
```

**Standardize common variations:**
- Use "Elizabeth" not "Elisabeth" (unless source-specific)
- Include generational suffixes: "John Smith Jr.", "Mary Smith III"
- Document alternative spellings in notes

### 2. Place Names

**Use hierarchical place structure:**
```yaml
# ✅ Hierarchical places
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
  coordinates:
    latitude: 53.7960
    longitude: -1.5479
```

**Include alternative names:**
```yaml
places/place-yorkshire.glx:
  name: Yorkshire
  alternative_names:
    - West Riding
    - County of York
  type: county
```

### 3. Event Types

**Use standard event types:**
```yaml
# Standard life events
event_types:
  - birth
  - baptism
  - marriage
  - death
  - burial
  - occupation
  - residence
  - education
  - military_service
  - immigration
  - emigration
```

**Document custom events clearly:**
```yaml
events/event-custom.glx:
  event_type: family_reunion
  date: "1990-07-15"
  place: place-leeds
  description: |
    Annual Smith family reunion at Roundhay Park.
    Attended by 45 family members from 3 generations.
```

## Data Quality Standards

### 1. Date Formats

**Use ISO 8601 date formats:**
```yaml
# ✅ Standard formats
date: "1850-01-15"        # Complete date
date: "1850-01"           # Year and month
date: "1850"              # Year only
date: "1850?"             # Uncertain year
date: "1849/1850"         # Between years
date: "1850-01-15/1850-01-20"  # Date range
```

**Document date uncertainty:**
```yaml
events/event-birth.glx:
  date: "1850-01-15?"  # Questionable date
  notes: |
    Birth certificate shows January 15, but family tradition
    says January 20. Certificate takes precedence as primary evidence.
```

### 2. Place Accuracy

**Be specific about place precision:**
```yaml
# ✅ Good place documentation
places/place-23-wellington-st.glx:
  name: 23 Wellington Street
  type: address
  parent: place-leeds
  coordinates:
    latitude: 53.7960
    longitude: -1.5479
  notes: |
    Residential address from 1851 Census.
    Street no longer exists, site now occupied by modern building.
```

**Document place name changes:**
```yaml
places/place-leeds.glx:
  name: Leeds
  alternative_names:
    - Loidis (historical)
    - Ledes (medieval)
  type: city
  notes: |
    Place name evolution: Ledes (1086 Domesday Book) → Leeds (modern)
```

## Collaboration Guidelines

### 1. Research Coordination

**Use GitHub Issues for research planning:**
```markdown
# Research Plan: Smith Family Origins

## Goals
- Document Smith family in Leeds, 1800-1900
- Find immigration/emigration patterns
- Connect to other Smith families in Yorkshire

## Assigned Tasks
- [ ] 1851 Census analysis (@researcher1)
- [ ] Parish register search (@researcher2)
- [ ] Occupation records (@researcher3)

## Evidence Standards
- Primary sources only for vital events
- Quality 2+ for residence/occupation data
- All assertions must have citations
```

### 2. Conflict Resolution

**Handle conflicting evidence systematically:**

```yaml
# Conflicting birth dates
assertions/assertion-john-birth-disputed.glx:
  subject: person-john-smith
  claim: birth_date
  value: "1850-01-15"  # Preferred date
  confidence: medium
  research_notes: |
    Conflicting evidence:
    - Birth certificate: Jan 15, 1850 (quality 3) - PREFERRED
    - Baptism record: Jan 20, 1850 (quality 3) - ALTERNATIVE
    - Census age: 25 in 1875 (quality 2) - SUPPORTS 1850

    Resolution: Birth certificate takes precedence as primary direct evidence.
    5-day baptism delay is within normal range for working families.
  citations:
    - citation-birth-cert
    - citation-baptism
    - citation-census-1875
```

### 3. Review Process

**Pull request review checklist:**
- [ ] All new assertions have citations
- [ ] Citation quality ratings are appropriate
- [ ] Evidence chains are complete
- [ ] Place hierarchy is correct
- [ ] All files pass validation
- [ ] ID formats are correct
- [ ] No conflicting evidence without resolution

## Maintenance Best Practices

### 1. Regular Validation
```bash
# Daily validation during active research
glx validate

# Weekly comprehensive check
glx validate examples/complete-family/

# Monthly evidence review
git log --since="1 month ago" --oneline
glx validate  # Ensure no regressions
```

### 2. Archive Backup
```bash
# Create regular backups
git tag backup-$(date +%Y-%m-%d)
git push origin backup-$(date +%Y-%m-%d)

# Archive to external storage
tar -czf smith-family-$(date +%Y-%m-%d).tar.gz .
```

### 3. Documentation Updates
```bash
# Update research status
git checkout -b docs/update-research-status
# Edit README.md with current status
git commit -m "Update research status - 2024 Q1

Completed:
- 1851 Census integration
- Birth certificate evidence
- Place hierarchy setup

In Progress:
- 1861 Census research
- Marriage record search
- Occupation timeline"
```

## Quality Metrics

### Research Completeness
Track these metrics for archive quality:

- **Evidence Coverage**: Percentage of assertions with citations
- **Quality Distribution**: Breakdown of citation quality ratings
- **Chain Completeness**: Percentage of complete evidence chains
- **Validation Success**: Pass rate of `glx validate`

**Example quality report:**
```yaml
archive_quality:
  total_assertions: 150
  cited_assertions: 145  # 96.7% coverage
  quality_distribution:
    quality_3: 45  # 30% primary direct
    quality_2: 60  # 40% secondary direct
    quality_1: 25  # 17% primary indirect
    quality_0: 15  # 10% secondary indirect
  complete_chains: 135  # 90% complete evidence chains
  validation_pass: true
```

Following these best practices ensures that your GENEALOGIX archive maintains high quality, reliability, and research integrity over time.
