---
title: Assertion Entity
description: Source-backed conclusions in GENEALOGIX - the core of the evidence model
layout: doc
---

# Assertion Entity

[← Back to Entity Types](README)

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
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
    confidence: high
```

**Key Points:**
- Entity ID is the map key (`assertion-john-birth-date`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `subject` | object | Typed reference to the entity this assertion is about |
| `property` OR `participant` | string/object | Either a property string or participant object (mutually exclusive) |
| `citations`, `sources`, or `media` | array | **At least one required** (enforced by JSON Schema and CLI validation) |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `value` | string | The concluded value of the property (**required** when `property` is present; not used with `participant`) |
| `date` | string | Date or date range when this property value applies (for temporal properties) |
| `confidence` | string | Confidence level (defined in archive vocabulary) |
| `notes` | string | General notes about the assertion |
| `status` | string | Research status of this assertion (e.g., proven, disproven, speculative) |

### Subject Object

The `subject` field uses a typed reference to avoid entity ID collisions. Exactly one key must be present:

| Field | Type | Description |
|-------|------|-------------|
| `person` | string | Reference to a Person entity |
| `event` | string | Reference to an Event entity |
| `relationship` | string | Reference to a Relationship entity |
| `place` | string | Reference to a Place entity |

### Participant Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `person` | string | Yes | Reference to the person entity |
| `role` | string | No | Role of the participant |
| `notes` | string | No | Notes about this participant |

## Required Fields

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Example formats:
  - Descriptive: `john-birth-date`, `mary-occupation`
  - Random hex: `a1b2c3d4`
  - Prefixed: `assertion-a1b2c3d4`
  - Sequential: `001`, `002`

### `subject`

- Type: Object with exactly one typed reference field
- Required: Yes
- Description: The entity this assertion is about (person, event, relationship, or place)

The typed reference structure prevents entity ID collisions in large archives where different entity types might have the same ID.

Examples:
```yaml
# Asserting about a person
subject:
  person: john-smith

# Asserting about an event
subject:
  event: marriage-1880

# Asserting about a relationship
subject:
  relationship: parent-child-001

# Asserting about a place
subject:
  place: leeds-yorkshire
```

### `property` (or `participant`)

- Type: String (for `property`) or Object (for `participant`)
- Required: One of `property` or `participant` must be present (mutually exclusive)
- Description: Either a property name being asserted, or a participant object for event/relationship participation

> **Note:** The `property` field corresponds to property names defined in [property vocabularies](vocabularies#property-vocabularies). For example, `property: born_on` references the `born_on` property from the person properties vocabulary. Unknown properties generate validation warnings.

Common property types:
- `born_on` - Birth date
- `died_on` - Death date
- `born_at` - Birth place
- `occupation` - Occupation/profession
- `residence` - Residence location
- `name` - Name form

Example:
```yaml
property: occupation
```

### Evidence Requirement

At least ONE of the following is required:
- `citations` - Array of citation IDs
- `sources` - Array of source IDs (direct source references)
- `media` - Array of media IDs (direct visual or documentary evidence)

**When to use each:**

- **`citations`** (preferred): When you have specific details about where in a source the evidence is found - page 23, entry 145, specific URL. This is rigorous and allows others to find the exact evidence.

- **`sources`** (direct): When the source doesn't need sub-location details:
  - Single-page documents (birth certificate) where a citation adds no value
  - Photographs or brief documents without meaningful subdivisions
  - Preliminary research where you'll add specific citations later

- **`media`** (direct visual evidence): When media itself is the evidence, without a formal source or citation:
  - Gravestone photos directly evidencing dates and names
  - Family photographs evidencing relationships or locations
  - Handwritten documents or letters where the media _is_ the primary evidence

Example:
```yaml
citations:
  - citation-birth-cert
  - citation-baptism-record
media:
  - media-gravestone-photo
```

## Optional Fields

### `participant`

- Type: Object
- Required: No (mutually exclusive with `property` and `value`)
- Description: Used for assertions about a person's participation in an event or relationship (instead of asserting a property value)

Structure:
```yaml
participant:
  person: "person-id"    # Reference to the person (required)
  role: "participant-role"  # Role of the participant (optional)
  notes: "string"        # Notes about the participant (optional)
```

**Key Points:**
- When `participant` is present, `property` and `value` must NOT be present
- Useful for representing conflicting evidence about who participated in an event or relationship

Example:
```yaml
assertions:
  assertion-witness-john:
    subject:
      event: event-marriage-1880
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
- Required: Yes when `property` is present; not used with `participant`
- Description: The concluded value of the property

Example:
```yaml
property: occupation
value: blacksmith
```

### `date`

- Type: String
- Required: No
- Description: Date or date range when this property value applies. Used with temporal properties (occupation, residence, etc.) to specify **when** the asserted value was true, matching the temporal value format on entities. This field is strictly for temporal targeting — it is NOT an "evidence date" or "observation date".

When a temporal property on an entity has multiple dated values:
```yaml
# Person's temporal property
occupation:
  - value: "blacksmith"
    date: "FROM 1870 TO 1890"
  - value: "farmer"
    date: "FROM 1890 TO 1920"
```

Each assertion can target a specific temporal entry using `date`:
```yaml
assertions:
  assertion-occupation-blacksmith:
    subject:
      person: person-john-smith
    property: occupation
    value: "blacksmith"
    date: "FROM 1870 TO 1890"
    citations: [citation-1881-census]
    confidence: high
```

Date formats follow the standard [date format](../2-core-concepts#date-format-standard). If omitted, the assertion applies to the property value without temporal context.

### `confidence`

- Type: String
- Required: No
- Description: Confidence level based on evidence quality

Confidence levels and their criteria are defined in each archive's `vocabularies/confidence-levels.glx` file. The standard vocabulary provides these defaults:
- `high` - Multiple high-quality sources agree, minimal uncertainty
- `medium` - Some evidence supports conclusion, but conflicts or gaps exist
- `low` - Limited evidence, significant uncertainty
- `disputed` - Multiple sources conflict, resolution unclear

Archives can customize these descriptions or add additional levels to match their research methodology.

**See [Vocabularies - Confidence Levels](vocabularies#confidence-levels-vocabulary) for:**
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

## Participant Assertions

Participant assertions represent evidence about who participated in an event or relationship, including conflicting evidence about participation and roles.

### Participant Assertion Example

```yaml
assertions:
  assertion-john-married:
    subject:
      event: event-marriage-1880
    participant:
      person: person-john-smith
      role: groom
    citations:
      - citation-marriage-cert
    confidence: high

  assertion-jane-married:
    subject:
      event: event-marriage-1880
    participant:
      person: person-jane-doe
      role: bride
    citations:
      - citation-marriage-cert
    confidence: high

  assertion-witness-thomas:
    subject:
      event: event-marriage-1880
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
assertions:
  # One source claims person-john is the father
  assertion-john-father-cert:
    subject:
      event: event-birth-1850
    participant:
      person: person-john-smith
      role: parent
      notes: "Listed as father on birth certificate"
    citations:
      - citation-birth-cert
    confidence: high

  # Another source claims person-thomas is the father
  assertion-thomas-father-letter:
    subject:
      event: event-birth-1850
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
assertions:
  assertion-john-birth-date:
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
    confidence: high
```

### Assertion with Multiple Evidence Sources

```yaml
assertions:
  assertion-john-occupation:
    subject:
      person: person-john-smith
    property: occupation
    value: blacksmith
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
    confidence: high
```

### Assertion with Conflicting Evidence

```yaml
assertions:
  assertion-mary-birth-disputed:
    subject:
      person: person-mary-jones
    property: born_on
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

### Temporal Property Assertion

For temporal properties like residence and occupation, the `date` field specifies when the value applies:

```yaml
assertions:
  assertion-john-residence-1851:
    subject:
      person: person-john-smith
    property: residence
    value: "place-leeds"
    date: "FROM 1850 TO 1870"
    citations:
      - citation-1851-census
      - citation-directory-1851
    confidence: high

  assertion-john-occupation-blacksmith:
    subject:
      person: person-john-smith
    property: occupation
    value: "blacksmith"
    date: "FROM 1870 TO 1890"
    citations:
      - citation-1881-census
      - citation-trade-directory
    confidence: high
```

### Assertion with Direct Media Evidence

```yaml
assertions:
  assertion-john-death-date:
    subject:
      person: person-john-smith
    property: died_on
    value: "1920-06-20"
    media:
      - media-gravestone-photo
    confidence: medium
    notes: "Date read directly from gravestone inscription"
```

Media can also be combined with citations and sources:

```yaml
assertions:
  assertion-john-death-confirmed:
    subject:
      person: person-john-smith
    property: died_on
    value: "1920-06-20"
    citations:
      - citation-death-cert
    media:
      - media-gravestone-photo
    confidence: high
    notes: "Death certificate corroborated by gravestone inscription"
```

### Low Confidence Assertion

```yaml
assertions:
  assertion-thomas-birth-estimated:
    subject:
      person: person-thomas-brown
    property: born_on
    value: "ABT 1825"
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
```

## Evidence Quality and Confidence

Assertions connect evidence (from citations) to conclusions with a certain level of confidence.

### Assertion Confidence

The standard vocabulary defines these default confidence levels (archives may customize):

| Confidence | Criteria | Example |
|------------|----------|---------|
| `high` | Multiple high-quality sources agree, minimal uncertainty | 3 birth certificates with same date |
| `medium` | Some evidence supports, but conflicts or gaps exist | 2 sources agree, 1 disagrees |
| `low` | Limited evidence, significant uncertainty | Only one low-quality source |
| `disputed` | Multiple sources conflict, resolution unclear | Multiple primary sources disagree |

## Validation Rules

- `subject` must be an object with exactly one typed reference field (person, event, relationship, or place)
- The referenced entity must exist in the archive
- **At least one of `citations`, `sources`, or `media` must be present** (this is validated as an error if missing)
- All citation references must point to existing Citation entities
- All source references must point to existing Source entities
- All media references must point to existing Media entities
- `property` values should match properties defined in the appropriate [property vocabulary](vocabularies#property-vocabularies) (unknown properties generate warnings)
- Confidence must be from the [confidence levels vocabulary](vocabularies#confidence-levels-vocabulary)

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
    ├── subject → references Person, Event, Relationship, or Place (typed reference)
    ├── citations → array of Citation IDs (evidence)
    ├── sources → array of Source IDs (direct reference)
    └── media → array of Media IDs (direct visual/documentary evidence)

Citation
    └── supports → Assertion (via assertion's citations array)

Media
    └── supports → Assertion (via assertion's media array)
    └── documents → Source or Citation (via media's source field or citation's media array)

Person/Event/Relationship/Place
    └── documented by → Assertion (subject reference)
```

## GEDCOM Mapping

GENEALOGIX assertions are implicit in GEDCOM:

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Assertion | Implicit in INDI/FAM + SOUR structure | |
| `subject` | INDI or FAM record | GLX uses typed reference |
| `property` | Property tag (BIRT, DEAT, OCCU, etc.) | |
| `value` | Property value | |
| `citations` | SOUR tags on property | |
| `confidence` | Derived from QUAY values | |

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
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-cert
    confidence: high
```

## Schema Reference

See [assertion.schema.json](../schema/v1/assertion.schema.json) for the complete JSON Schema definition.

## See Also

- [Core Concepts - Assertion-Aware Data Model](../2-core-concepts#assertion-aware-data-model) - Overview of assertion philosophy
- [Core Concepts - Evidence Chain](../2-core-concepts#evidence-chain) - Understanding evidence quality
- [Citation Entity](citation) - Evidence references that support assertions
- [Source Entity](source) - Original sources cited by assertions
- [Person Entity](person) - Common subject of assertions
- [Core Concepts - Data Types](../2-core-concepts#data-types)
