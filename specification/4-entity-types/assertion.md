---
title: Assertion Entity
description: Source-backed conclusions in GENEALOGIX - the core of the evidence model
layout: doc
---

# Assertion Entity

[← Back to Entity Types](README.md)

## Overview

An Assertion entity represents a source-backed conclusion about a specific genealogical fact. Assertions form the core of the GENEALOGIX evidence model, separating **what sources say** (citations) from **what we conclude** (assertions).

This separation enables:
- Multiple evidence sources supporting a single conclusion
- Conflicting evidence representation
- Confidence assessment based on evidence quality
- Clear research transparency and audit trails

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in assertions/ directory)
assertions:
  assertion-john-birth-date:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
    confidence: high
```

**Key Points:**
- Entity ID is the map key (`assertion-john-birth-date`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Required Fields

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Recommended formats:
  - Descriptive: `assertion-john-birth-date`, `assertion-mary-occupation`
  - Random hex: `assertion-a1b2c3d4` (for collaboration)
  - Sequential: `assertion-001`, `assertion-002`

### `subject`

- Type: String
- Required: Yes
- Description: The entity this assertion is about (person, event, relationship, etc.)

Example:
```yaml
subject: person-john-smith
```

### `claim`

- Type: String
- Required: Yes
- Description: The property or fact being claimed

Common claim types:
- `born_on` - Birth date
- `died_on` - Death date
- `born_at` - Birth place
- `occupation` - Occupation/profession
- `residence` - Residence location
- `name` - Name form

Example:
```yaml
claim: occupation
```

### Evidence Requirement

At least ONE of the following is required:
- `citations` - Array of citation IDs
- `sources` - Array of source IDs (direct source references)

Example:
```yaml
citations:
  - citation-birth-cert
  - citation-baptism-record
```

## Optional Fields

### `participant`

- Type: Object
- Required: No (mutually exclusive with `claim` and `value`)
- Description: Used for assertions about a person's participation in an event or relationship (instead of claiming a property value)

Structure:
```yaml
participant:
  person: "person-id"    # Reference to the person (required)
  role: "participant-role"  # Role of the participant (optional)
  notes: "string"        # Notes about the participant (optional)
```

**Key Points:**
- When `participant` is present, `claim` and `value` must NOT be present
- When `participant` is present, the implicit claim is "participant"
- Useful for representing conflicting evidence about who participated in an event or relationship

Example:
```yaml
assertions:
  assertion-witness-john:
    subject: event-marriage-1880
    participant:
      person: person-john-smith
      role: witness
      notes: "Listed as witness on marriage certificate"
    citations:
      - citation-marriage-cert
    confidence: high
```

### `value`

- Type: String
- Required: No for participant assertions; recommended for claim assertions
- Description: The concluded value of the claim (not used with `participant`)

Example:
```yaml
claim: occupation
value: blacksmith
```

### `confidence`

- Type: String
- Required: No
- Description: Confidence level based on evidence quality

Confidence levels and their criteria are defined in each archive's `vocabularies/confidence-levels.glx` file. The standard vocabulary provides these defaults:
- `high` - Multiple primary sources agree
- `medium` - Some conflicting evidence, but preponderance supports
- `low` - Limited evidence, requires more research
- `disputed` - Multiple sources conflict, resolution unclear

Archives can customize these descriptions or add additional levels to match their research methodology.

**See [Vocabularies - Confidence Levels](vocabularies.md#confidence-levels-vocabulary) for:**
- Customizing confidence level definitions for your archive
- Adding custom confidence levels
- Vocabulary file structure and validation

Example:
```yaml
confidence: high
```

### `notes`

- Type: String
- Required: No
- Description: General notes about the assertion

### `tags`

- Type: Array of Strings
- Required: No
- Description: Tags for categorization

Example:
```yaml
tags:
  - needs-review
  - conflicting-evidence
  - high-priority
```

## Participant Assertions

Participant assertions represent evidence about who participated in an event or relationship, including conflicting evidence about participation and roles.

### Participant Assertion Example

```yaml
# assertions/assertion-marriage-participants.glx
assertions:
  assertion-john-married:
    subject: event-marriage-1880
    participant:
      person: person-john-smith
      role: groom
    citations:
      - citation-marriage-cert
    confidence: high
  
  assertion-jane-married:
    subject: event-marriage-1880
    participant:
      person: person-jane-doe
      role: bride
    citations:
      - citation-marriage-cert
    confidence: high
  
  assertion-witness-thomas:
    subject: event-marriage-1880
    participant:
      person: person-thomas-brown
      role: witness
      notes: "Witnessed marriage ceremony"
    citations:
      - citation-marriage-cert
    confidence: high
```

### Conflicting Participant Evidence

```yaml
# assertions/assertion-conflicting-parents.glx
assertions:
  # One source claims person-john is the father
  assertion-john-father-cert:
    subject: event-birth-1850
    participant:
      person: person-john-smith
      role: parent
      notes: "Listed as father on birth certificate"
    citations:
      - citation-birth-cert
    confidence: high
  
  # Another source claims person-thomas is the father
  assertion-thomas-father-letter:
    subject: event-birth-1850
    participant:
      person: person-thomas-brown
      role: parent
      notes: "Family letter suggests Thomas was the father"
    citations:
      - citation-family-letter
    confidence: low
    notes: |
      Conflicting evidence about paternity:
      - Birth certificate (primary source): John Smith
      - Family letter (secondary source): Thomas Brown
      
      Certificate is more reliable, but letter provides alternative possibility.
      Needs further research.
```

## Usage Patterns

### Basic Biographical Assertion

```yaml
# assertions/assertion-john-birth.glx
assertions:
  assertion-john-birth-date:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
    confidence: high
```

### Assertion with Multiple Evidence Sources

```yaml
# assertions/assertion-john-occupation.glx
assertions:
  assertion-john-occupation:
    subject: person-john-smith
    claim: occupation
    value: blacksmith
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
    confidence: high
```

### Assertion with Conflicting Evidence

```yaml
# assertions/assertion-mary-birth.glx
assertions:
  assertion-mary-birth-disputed:
    subject: person-mary-jones
    claim: born_on
    value: "1852-03-10"
    citations:
      - citation-birth-cert       # Says March 10
      - citation-family-bible      # Says March 12
    confidence: medium
    notes: |
      Birth certificate (primary source) says March 10, 1852.
      Family Bible (secondary source) says March 12, 1852.
      
      Certificate is more reliable as primary direct evidence.
      Bible entry may have been written from memory later.
      
      Conclusion: March 10, 1852 (with some uncertainty)
```

### Complex Residence Assertion

```yaml
# assertions/assertion-residence.glx
assertions:
  assertion-john-residence-1851:
    subject: person-john-smith
    claim: residence
    value: "Wellington Street, Leeds, Yorkshire"
    citations:
      - citation-1851-census
      - citation-directory-1851
    confidence: high
    notes: "Residence at time of 1851 census"
    tags:
      - census-derived
      - verified
```

### Low Confidence Assertion

```yaml
# assertions/assertion-estimated-birth.glx
assertions:
  assertion-thomas-birth-estimated:
    subject: person-thomas-brown
    claim: born_on
    value: "circa 1825"
    citations:
      - citation-death-cert-age
    confidence: low
    notes: |
      No birth record found. Age at death (1900) reported as 75,
      suggesting birth around 1825. However, age reporting in 
      death certificates is often approximate.
      
      Need to search:
      - Parish registers 1820-1830
      - Census records for age progression
    tags:
      - estimated
      - needs-research
```

## Evidence Quality and Confidence

Assertions connect evidence (from citations) to conclusions with a certain level of confidence.

### Assertion Confidence

The standard vocabulary defines these default confidence levels (archives may customize):

| Confidence | Criteria | Example |
|------------|----------|---------|
| `high` | Multiple primary sources agree | 3 birth certificates with same date |
| `medium` | Preponderance of evidence, some conflict | 2 sources agree, 1 disagrees |
| `low` | Limited evidence, needs research | Only one low-quality source |
| `disputed` | Conflicting evidence, no clear resolution | Multiple primary sources disagree |

## Validation Rules

- `subject` must reference an existing entity ID
- At least one of `citations` or `sources` must be present
- All citation references must point to existing Citation entities
- All source references must point to existing Source entities
- `confidence` should be one of: `high`, `medium`, `low`, `disputed`

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Assertion files are typically organized by subject or topic:

```
assertions/
├── biographical/
│   ├── births/
│   │   ├── assert-john-birth.glx
│   │   └── assert-mary-birth.glx
│   ├── deaths/
│   │   └── assert-john-death.glx
│   └── occupations/
│       └── assert-john-occupation.glx
├── relationships/
│   └── assert-parentage.glx
└── residences/
    └── assert-1851-residence.glx
```

Or by entity:

```
assertions/
├── person-john-smith/
│   ├── assert-birth.glx
│   ├── assert-occupation.glx
│   └── assert-death.glx
├── person-mary-jones/
│   └── assert-birth.glx
└── rel-marriage-001/
    └── assert-marriage-date.glx
```

## Relationship to Other Entities

```
Assertion
    ├── subject → references Person, Event, Relationship, or other entity
    ├── citations → array of Citation IDs (evidence)
    └── sources → array of Source IDs (direct reference)

Citation
    └── supports → Assertion (via assertion's citations array)

Person/Event/Relationship
    └── documented by → Assertion (subject reference)
```

## GEDCOM Mapping

GENEALOGIX assertions are implicit in GEDCOM:

| GENEALOGIX | GEDCOM Equivalent |
|------------|-------------------|
| Assertion | Implicit in INDI/FAM + SOUR structure |
| `subject` | INDI or FAM record |
| `claim` | Property tag (BIRT, DEAT, OCCU, etc.) |
| `value` | Property value |
| `citations` | SOUR tags on property |
| `confidence` | Derived from QUAY values |

GEDCOM Example:
```
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 JAN 1850
2 SOUR @S1@
3 QUAY 3
```

GENEALOGIX Equivalent:
```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"

assertions:
  assertion-john-birth:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-cert
    confidence: high
```

## Schema Reference

See [assertion.schema.json](../schema/v1/assertion.schema.json) for the complete JSON Schema definition.

## See Also

- [Core Concepts - Assertion-Aware Data Model](../2-core-concepts.md#assertion-aware-data-model) - Overview of assertion philosophy
- [Core Concepts - Evidence Hierarchy](../2-core-concepts.md#evidence-hierarchy) - Understanding evidence quality
- [Citation Entity](citation.md) - Evidence references that support assertions
- [Source Entity](source.md) - Original sources cited by assertions
- [Person Entity](person.md) - Common subject of assertions
- [Data Types](../6-data-types.md)
