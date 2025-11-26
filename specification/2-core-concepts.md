---
title: Core Concepts
description: Fundamental principles and architecture of GENEALOGIX - assertions, evidence chains, and version control
layout: doc
---

# Core Concepts

This section explains the fundamental principles and architecture that make GENEALOGIX different from traditional genealogy formats.

## Assertion-Aware Data Model

> **See Also:** For complete assertion entity specification, see [Assertion Entity](4-entity-types/assertion.md)

### The Problem with Traditional Models
Traditional genealogy software stores conclusions directly:
```
Person: John Smith
Birth: January 15, 1850
Place: Leeds, Yorkshire
```

This approach loses the critical distinction between **evidence** (what sources say) and **conclusions** (what we believe). If conflicting evidence emerges, there's no clear way to represent uncertainty or evaluate source quality.

### GENEALOGIX Solution: Assertions
GENEALOGIX separates evidence from conclusions using **assertions**:

```yaml
# assertions/assertion-john-birth.glx - Conclusion based on evidence
assertions:
  assertion-john-birth:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
      - citation-baptism-record
    confidence: high

# citations/citation-birth-certificate.glx - Specific evidence
citations:
  citation-birth-certificate:
    source: source-gro-register
    locator: "Certificate 1850-LEEDS-00145"
    text_from_source: "John Smith, born January 15, 1850"
```

### Benefits of This Approach
1. **Multiple Evidence**: One assertion can reference multiple citations
2. **Conflicting Evidence**: Multiple assertions can exist for the same fact
3. **Research Transparency**: Clear audit trail from source to conclusion
4. **Confidence Levels**: Assertions can express certainty based on evidence

### Entity Properties vs. Assertions

In addition to assertions (which provide evidence), GENEALOGIX entities can have **properties** that represent the researcher's current conclusions:

```yaml
# Direct properties on entity (quick recording, no evidence yet)
persons:
  person-john:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      born_on: "1850-01-15"
      occupation: "blacksmith"
    notes: "Initial data entry"

# Assertions provide evidence for these conclusions
assertions:
  assertion-john-occupation:
    subject: person-john
    claim: occupation
    value: blacksmith
    citations:
      - citation-1851-census
    confidence: high
```

**Key Points:**
- **Properties** are quick ways to record current conclusions without evidence
- **Assertions** formally document the evidence supporting those properties
- Properties can be set before assertions are created (rapid data entry)
- Assertions provide the research trail explaining WHY properties have certain values
- Both mechanisms work together: properties for quick recording, assertions for research documentation

## Evidence Hierarchy

### Evidence Chain Structure

GENEALOGIX organizes genealogical evidence in a hierarchical chain from physical sources to conclusions:

**Complete Evidence Chain:**
1. **Repository** - Physical or digital institution holding sources
2. **Source** - Original document, record, or material
3. **Citation** - Specific reference within the source
4. **Media** (optional) - Digital images, scans, or recordings of the source
5. **Assertion** - Evidence-based conclusion about a fact

Each level provides context and traceability for the research:

```
Repository → Source → Citation → [Media] → Assertion
   ↓           ↓         ↓          ↓          ↓
Physical   Original  Specific   Digital   Researcher's
Location   Material  Reference  Evidence  Conclusion
```

**Media as Optional Link:**
- Media entities (photographs, scans, audio) can document sources
- Not required but highly recommended for preservation
- Links between citation and assertion or directly to sources
- See [Media Entity](4-entity-types/media.md) for details

### Evidence Chain Example

```yaml
# repositories/repository-gro.glx
repositories:
  repository-gro:
    name: General Register Office
    address: "London, England"

# sources/source-birth-register.glx
sources:
  source-birth-register:
    title: England Birth Register 1850
    repository: repository-gro

# citations/citation-john-birth.glx
citations:
  citation-john-birth:
    source: source-birth-register
    locator: "Volume 23, Page 145, Entry 23"

# assertions/assertion-john-born.glx
assertions:
  assertion-john-born:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations: [citation-john-birth]
    confidence: high
```

## Provenance Tracking

### What is Provenance?
Provenance is the complete history of how information came to be known, including:
- **Source**: Where the information originated
- **Chain of Custody**: How it was preserved and transmitted
- **Author**: Who recorded or interpreted the information
- **Timestamp**: When the information was recorded
- **Context**: Circumstances surrounding the creation

### GENEALOGIX Provenance Features

#### 1. Source Attribution
Every assertion must reference specific citations:
```yaml
# assertions/assertion-smith-occupation.glx
assertions:
  assertion-smith-occupation:
    subject: person-john-smith
    claim: occupation
    value: blacksmith
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
```

#### 2. Change Tracking with Git
Since GENEALOGIX archives are Git repositories, all changes are automatically tracked:
```bash
# Git provides complete change history
git log --oneline -- persons/person-john-smith.glx

# See who made what changes
git blame persons/person-john-smith.glx

# Track research progress over time
git log --since="2024-01-01" --until="2024-03-31"
```

#### 3. Research Notes and Analysis
Structured fields for documenting research decisions:
```yaml
# assertions/assertion-disputed-birth.glx
assertions:
  assertion-disputed-birth:
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    confidence: medium
    notes: |
      Two conflicting sources:
      - Birth certificate: January 15, 1850 (preferred, higher quality)
      - Baptism record: January 20, 1850 (5-day delay common)

      Certificate takes precedence as primary direct evidence.
      More research needed on baptism delay practices.
    citations:
      - citation-birth-cert
      - citation-baptism-record
```

#### 4. Confidence Assessment
Assertions include confidence levels based on evidence quality:
```yaml
confidence_levels:
  high:    "Multiple high-quality sources agree, minimal uncertainty"
  medium:  "Some evidence supports conclusion, but conflicts or gaps exist"
  low:     "Limited evidence, significant uncertainty"
  disputed: "Multiple sources conflict, resolution unclear"
```

## Version Control Integration

### Git-Native Architecture
GENEALOGIX is designed from the ground up for Git version control:

#### 1. File-Based Structure
Each entity is a separate YAML file, perfect for Git tracking:
```
family-archive/
├── persons/
│   ├── person-john-smith.glx
│   ├── person-mary-brown.glx
│   └── person-jane-smith.glx
├── events/
│   ├── event-birth-john.glx
│   └── event-marriage.glx
└── citations/
    └── citation-1851-census.glx
```

#### 2. Branch-Based Research
Isolate research investigations in branches:
```bash
# Research specific time period
git checkout -b research/1851-census-analysis

# Add census data and analysis
# ... make changes ...

# Validate research branch
glx validate

# Merge findings when complete
git checkout main
git merge research/1851-census-analysis
```

#### 3. Collaborative Workflows
Multiple researchers can work simultaneously:
```bash
# Researcher A: Focus on vital records
git checkout -b research/vital-records
# ... work on birth, marriage, death records ...

# Researcher B: Focus on census data
git checkout -b research/census-records
# ... work on census information ...

# Integrate both research streams
git checkout main
git merge research/vital-records
git merge research/census-records  # May need conflict resolution
```

#### 4. Evidence Conflict Resolution
Git helps resolve conflicting evidence:
```bash
# Scenario: Two researchers find different birth dates
git merge research/birth-certificate-data
# CONFLICT: persons/person-john-smith.glx

# Manual resolution: Create assertion with both citations
# Git tracks the resolution process
git add persons/person-john-smith.glx
git commit -m "Resolve birth date conflict

Evidence:
- Birth certificate: Jan 15, 1850 (quality 3)
- Census record: age 25 in 1875 (implies 1850, quality 2)

Resolution: Accept certificate date, census supports approximately"
```

### Advanced Git Features

#### 1. Bisect for Research Errors
Find when incorrect information was introduced:
```bash
# Archive shows person born in wrong place
git bisect start
git bisect bad HEAD  # Current state has error
git bisect good v1.0.0  # Known good state

# Git finds the commit that introduced the error
git bisect run glx validate
```

#### 2. Stash for Experimentation
Try research hypotheses without committing:
```bash
# Start investigating alternative theory
git stash push -m "Trying alternative birth place theory"

# Make experimental changes
# ... modify files ...

# Validate hypothesis
glx validate

# If promising, commit; if not, restore
git stash pop  # Discard changes
# or
git stash pop && git add . && git commit  # Keep changes
```

#### 3. Interactive Rebase for Research History
Clean up research process:
```bash
# Clean up messy research commits
git rebase -i v1.0.0

# Combine related research steps
# Edit commit messages for clarity
# Remove dead-end investigations
```

## Entity Relationships

### Core Entity Connections
The 9 GENEALOGIX entity types form an interconnected web:

```
Person ←→ Relationship ←→ Person
Person ←→ Event ←→ Place
Source ←→ Citation → Assertion ←→ Person/Event/Place
Repository → Source
```

### Validation Dependencies
These relationships create validation requirements:
- Citations must reference existing sources
- Assertions must reference existing citations
- Events must reference existing places (if place specified)
- Participants must reference existing persons

### Reference Integrity
GENEALOGIX enforces referential integrity:
```yaml
# Valid: All referenced entities exist
places:
  place-leeds-parish-church:
    name: "Leeds Parish Church"

persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"

  person-mary-brown:
    properties:
      name:
        value: "Mary Brown"
        fields:
          given: "Mary"
          surname: "Brown"

events:
  event-wedding:
    type: marriage
    place: place-leeds-parish-church  # Must exist in places/
    participants:
      - person: person-john-smith     # Must exist in persons/
      - person: person-mary-brown     # Must exist in persons/

# Invalid: Referenced entities don't exist
# glx validate will catch these errors
```

## Repository-Owned Vocabularies

### Archive-Level Type Definitions

Unlike traditional genealogy formats with fixed type systems, GENEALOGIX uses **repository-owned controlled vocabularies**. Each archive defines its own valid types in the `vocabularies/` directory, combining standardization with flexibility.

### Why Archive-Level Vocabularies?

This is one of GENEALOGIX's most powerful features: **each archive controls its own type system**.

#### Freedom and Flexibility
Unlike formats with centrally-defined types, GLX lets each archive define exactly what it needs:

- **Genealogy**: Use standard relationship types (marriage, parent-child, adoption)
- **Colonial History**: Add custom types (indenture, manumission, land_grant)
- **Religious Studies**: Define custom events (ordination, investiture, pilgrimage)
- **Biography**: Create domain-specific relationships (mentor-mentee, patron-artist)
- **Local History**: Track community roles (town_selectman, guild_master)
- **And more**: Any research domain with people, events, and relationships

#### Autonomy Without Chaos
You get both standardization AND flexibility:

1. **Standard starter vocabularies**: New archives begin with common genealogy types
2. **Extend as needed**: Add custom types specific to your research
3. **Archive-owned**: No central committee, no approval process
4. **Git-versioned**: Vocabulary changes tracked with your data
5. **Validated**: The CLI ensures all used types are defined
6. **Collaborative**: Teams discuss and agree on types within the repository

### Standard Vocabulary Files

When you initialize a new archive with `glx init`, standard vocabulary files are automatically created:

```
vocabularies/
  relationship-types.glx   # Marriage, parent-child, adoption, etc.
  event-types.glx         # Birth, death, baptism, occupation, etc.
  place-types.glx         # Country, city, parish, etc.
  repository-types.glx    # Archive, library, church, etc.
  participant-roles.glx   # Principal, witness, godparent, etc.
  media-types.glx         # Photo, document, audio, etc.
  confidence-levels.glx   # High, medium, low, disputed
```

### Examples of Custom Vocabularies

#### Maritime Genealogy
```yaml
# vocabularies/event-types.glx
event_types:
  ship_departure:
    label: "Ship Departure"
    description: "Departure on a sea voyage"

  shipwreck:
    label: "Shipwreck"
    description: "Vessel lost at sea"
```

#### Academic Biography
```yaml
# vocabularies/relationship-types.glx
relationship_types:
  doctoral_advisor:
    label: "Doctoral Advisor"
    description: "PhD thesis advisor relationship"

  academic_rivalry:
    label: "Academic Rivalry"
    description: "Documented professional competition"
```

#### Enslaved Persons Research
```yaml
# vocabularies/event-types.glx
event_types:
  manumission:
    label: "Manumission"
    description: "Legal grant of freedom"
    gedcom: "EVEN"

  sale:
    label: "Sale"
    description: "Record of person being sold"
    gedcom: "EVEN"
```

### No Limits, No Boundaries

**Traditional genealogy formats say**: "You can only use these predefined types"

**GENEALOGIX says**: "Define whatever types your research needs"

This makes GLX suitable for:
- Traditional family history
- Local and community history
- Biographical research
- Prosopography (collective biography)
- Historical demography
- Any research involving people, events, and relationships

### Validation

The `glx validate` command ensures all entity types used in your archive are defined in the vocabulary files, preventing typos and maintaining consistency.

This core concept architecture ensures that GENEALOGIX archives are reliable, verifiable, and maintainable over time.


