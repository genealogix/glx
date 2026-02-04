---
title: Core Concepts
description: Fundamental principles and architecture of GENEALOGIX - flexibility, entities, properties, assertions, and evidence chains
layout: doc
---

# Core Concepts

This section explains how GENEALOGIX works: the flexible vocabularies, entities, and evidence model that make GLX different from traditional genealogy formats.

## Archive-Owned Vocabularies

### Why Archive-Level Control?

Unlike traditional genealogy formats with fixed type systems, GENEALOGIX uses **archive-owned controlled vocabularies**. Each archive defines its own valid types in the `vocabularies/` directory, combining standardization with flexibility.

**This is GENEALOGIX's most powerful feature: each archive controls its own type system.**

#### Freedom and Flexibility

Unlike formats with centrally-defined types, GLX lets each archive define exactly what it needs:

- **Traditional Genealogy**: Use standard types (marriage, parent-child, adoption)
- **Colonial History**: Add custom types (indenture, manumission, land_grant)
- **Religious Studies**: Define custom events (ordination, investiture, pilgrimage)
- **Biography**: Create domain-specific relationships (mentor-mentee, patron-artist)
- **Local History**: Track community roles (town_selectman, guild_master)
- **Maritime Research**: Add ship_departure, shipwreck, port_arrival events
- **Any research domain** with people, events, and relationships

#### Autonomy Without Chaos

You get both standardization AND flexibility:

1. **Standard starter vocabularies**: New archives begin with common genealogy types
2. **Extend as needed**: Add custom types specific to your research
3. **Archive-owned**: No central committee, no approval process
4. **Git-versioned**: Vocabulary changes tracked with your data
5. **Validated**: The CLI ensures all used types are defined
6. **Collaborative**: Teams discuss and agree on types within the archive

### Standard Vocabulary Files

When you initialize a new archive with `glx init`, standard vocabulary files are automatically created:

```
vocabularies/
├── event-types.glx           # Birth, death, baptism, occupation, etc.
├── relationship-types.glx    # Marriage, parent-child, adoption, etc.
├── place-types.glx           # Country, city, parish, etc.
├── source-types.glx          # Vital records, census, church registers, etc.
├── repository-types.glx      # Archive, library, church, etc.
├── media-types.glx           # Photo, document, audio, etc.
├── participant-roles.glx     # Principal, witness, godparent, etc.
├── confidence-levels.glx     # High, medium, low, disputed
└── property vocabularies/
    ├── person-properties.glx
    ├── event-properties.glx
    ├── relationship-properties.glx
    ├── place-properties.glx
    ├── media-properties.glx
    ├── repository-properties.glx
    ├── source-properties.glx
    └── citation-properties.glx
```

### Property Vocabularies

**Property vocabularies** are a special category that define the custom properties available for each entity type. These are critical for the assertion model (see sections below).

Property vocabularies define:
- What properties can exist on entities (name, occupation, residence, etc.)
- Property value types (string, date, integer, boolean, or references to other entities)
- Whether properties are temporal (can change over time)
- Whether properties support multiple values
- Structured fields for complex properties (name → given, surname, prefix, suffix)

Example from `person-properties.glx`:

```yaml
person_properties:
  name:
    label: "Name"
    description: "Person's name as recorded"
    value_type: string
    temporal: true
    fields:
      given:
        label: "Given Name"
        description: "Given/first name(s)"
      surname:
        label: "Surname"
        description: "Family name"

  occupation:
    label: "Occupation"
    description: "Profession or trade"
    value_type: string
    temporal: true

  residence:
    label: "Residence"
    description: "Place of residence"
    reference_type: places
    temporal: true
```

### How to Add Custom Types

Adding custom types is straightforward:

**1. Edit the appropriate vocabulary file**

```yaml
# vocabularies/event-types.glx
event_types:
  # ... standard types ...

  # Your custom type
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"
```

**2. Use the new type in your data**

```yaml
# events/event-john-apprentice.glx
events:
  event-john-apprentice:
    type: apprenticeship
    date: "1865-03-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: apprentice
```

**3. Validate your archive**

```bash
glx validate
```

The validator will confirm that all types are defined and all references are valid.

### Validation Behavior

The `glx validate` command enforces vocabulary consistency with different severity levels:

**Errors (must be fixed):**
- Entity references that don't exist (person, place, source, etc.)
- Vocabulary type references that aren't defined (event_types, relationship_types, etc.)
- Property references when properties are defined as `reference_type` but the referenced entity doesn't exist

**Warnings (flexible):**
- Unknown properties not defined in property vocabularies
- Unknown assertion claims not defined in property vocabularies
- Warnings allow rapid data entry and emerging properties without breaking validation

This policy balances strictness (broken references are errors) with flexibility (unknown properties generate warnings, not errors).

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

## Entity Relationships

GENEALOGIX uses 9 core entity types that form an interconnected web representing genealogical research:

### Core Entities

```
Person ←→ Relationship ←→ Person
Person ←→ Event ←→ Place
Source ←→ Citation → Assertion → Person/Event/Place
Repository → Source
Media → (any entity)
```

**The 9 Entity Types:**

1. **Person** - Individuals in the family archive
2. **Relationship** - Connections between people (marriage, parent-child, etc.)
3. **Event** - Occurrences in time and place (birth, death, marriage, occupation)
4. **Place** - Geographic locations with hierarchical structure
5. **Source** - Information sources (books, records, websites, databases)
6. **Citation** - Specific references within sources
7. **Repository** - Institutions holding sources (archives, libraries, churches)
8. **Media** - Digital objects (photos, documents, audio, video)
9. **Assertion** - Evidence-based conclusions about facts

### Validation Dependencies

These relationships create validation requirements that ensure archive integrity:

- Citations must reference existing sources
- Assertions must reference existing citations or sources
- Events must reference existing places (if place specified)
- Participants must reference existing persons
- Relationships must reference existing persons

GENEALOGIX enforces referential integrity through the `glx validate` command, preventing broken references and maintaining data consistency.

## Properties: Recording Conclusions

### What Are Properties?

Properties represent the **researcher's current conclusions** about an entity. They are the "accepted values" you record as you work:

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"
      born_on: "1850-01-15"
      born_at: "place-leeds"
      occupation: "blacksmith"
      residence: "place-leeds"
```

### Defined by Property Vocabularies

All properties are defined in property vocabularies (see Archive-Owned Vocabularies above). The `person-properties.glx` vocabulary defines:
- What properties exist (`name`, `gender`, `born_on`, `occupation`, etc.)
- Their data types (string, date, place reference)
- Whether they can change over time (temporal)
- Whether they have structured fields

### Properties Can Exist Without Assertions

**Critical point**: Properties can be set without assertions. This supports rapid data entry:

```yaml
# Quick data entry - just properties
persons:
  person-mary-jones:
    properties:
      name:
        value: "Mary Jones"
        fields:
          given: "Mary"
          surname: "Jones"
      born_on: "1852-03-10"
    notes: "Initial data entry from family records"
```

This person record is valid even without assertions documenting the evidence for these properties. You can add assertions later as you research sources.

### Temporal Properties

Properties marked as `temporal: true` in vocabularies can change over time:

```yaml
properties:
  occupation:
    - value: "blacksmith"
      date: "1880"
    - value: "farmer"
      date: "FROM 1885 TO 1920"
  residence:
    - value: "place-leeds"
      date: "FROM 1850 TO 1900"
    - value: "place-london"
      date: "FROM 1900 TO 1920"
```

### Structured Properties

Properties can have structured fields for complex data:

```yaml
properties:
  name:
    value: "Dr. John Smith Jr."
    fields:
      prefix: "Dr."
      given: "John"
      surname: "Smith"
      suffix: "Jr."
```

The `value` field preserves the original recorded form, while `fields` provide structured access to components.

### How Properties Complement Assertions

Properties and assertions work together:

- **Properties** = "What we currently believe"
- **Assertions** = "Why we believe it, with evidence"

Properties can be recorded quickly during initial data entry. Assertions document the research trail explaining why those properties have their values. Multiple assertions can support a single property, or present conflicting evidence about what the property value should be.

## Assertion-Aware Data Model

> **See Also:** For complete assertion entity specification, see [Assertion Entity](4-entity-types/assertion)

### The Problem with Traditional Models

Traditional genealogy software stores conclusions directly:
```
Person: John Smith
Birth: January 15, 1850
Place: Leeds, Yorkshire
```

This approach loses the critical distinction between **evidence** (what sources say) and **conclusions** (what we believe). If conflicting evidence emerges, there's no clear way to represent uncertainty or evaluate source quality.

### GENEALOGIX Solution: Assertions

GENEALOGIX separates evidence from conclusions using **assertions**. An assertion is an evidence-backed claim about a specific fact:

```yaml
# assertions/assertion-john-birth.glx
assertions:
  assertion-john-birth:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
      - citation-baptism-record
    confidence: high
```

### How Assertions Work

**Core fields:**
- `subject`: The entity this assertion is about (person, event, relationship, place)
- `claim`: The property being claimed (references property vocabulary)
- `value`: The concluded value of the claim
- `citations` or `sources`: Evidence supporting this claim (at least one required)
- `confidence`: How certain we are based on evidence quality

**The `claim` field references property vocabularies:**

```yaml
# The claim "born_on" must be defined in person-properties.glx
assertion-john-birth:
  subject: person-john-smith
  claim: born_on  # Validated against person_properties vocabulary
  value: "1850-01-15"
  citations: [citation-birth-cert]
```

The validator checks that `born_on` is defined in `person-properties.glx`. This ensures consistency between properties and assertions.

### Evidence-Based Claims

Assertions must cite their evidence:

```yaml
# citations/citation-birth-certificate.glx
citations:
  citation-birth-certificate:
    source: source-gro-register
    properties:
      locator: "Certificate 1850-LEEDS-00145"
      text_from_source: "John Smith, born January 15, 1850"
```

Every assertion requires at least one citation or source reference, creating an audit trail from conclusion back to original evidence.

### Conflicting Evidence

Multiple assertions can exist for the same fact, representing conflicting evidence:

```yaml
assertions:
  # Assertion based on birth certificate
  assertion-mary-birth-cert:
    subject: person-mary-jones
    claim: born_on
    value: "1852-03-10"
    citations: [citation-birth-cert]
    confidence: high
    notes: "Primary direct evidence"

  # Assertion based on family Bible
  assertion-mary-birth-bible:
    subject: person-mary-jones
    claim: born_on
    value: "1852-03-12"
    citations: [citation-family-bible]
    confidence: medium
    notes: |
      Family Bible entry conflicts with certificate.
      Bible likely written from memory later.
      Certificate takes precedence as primary source.
```

### Confidence Levels

Assertions include confidence levels based on evidence quality:

```yaml
confidence_levels:
  high:    "Multiple high-quality sources agree, minimal uncertainty"
  medium:  "Some evidence supports conclusion, but conflicts or gaps exist"
  low:     "Limited evidence, significant uncertainty"
  disputed: "Multiple sources conflict, resolution unclear"
```

Confidence levels are defined in `vocabularies/confidence-levels.glx` and can be customized per archive.

### Benefits of This Approach

1. **Multiple Evidence**: One assertion can reference multiple citations, showing corroboration
2. **Conflicting Evidence**: Multiple assertions can exist for the same property, documenting disagreements
3. **Research Transparency**: Clear audit trail from source to conclusion
4. **Confidence Tracking**: Assertions express certainty based on evidence quality
5. **Flexible Data Entry**: Properties can be recorded quickly, assertions added during research
6. **Source Quality**: Different evidence can be weighted differently via confidence levels

### How Properties and Assertions Work Together

```yaml
# 1. Quick data entry - just properties
persons:
  person-john:
    properties:
      born_on: "1850-01-15"
      occupation: "blacksmith"

# 2. Later: Add assertions documenting the evidence
assertions:
  assertion-john-birth:
    subject: person-john
    claim: born_on
    value: "1850-01-15"
    citations: [citation-birth-cert]
    confidence: high

  assertion-john-occupation:
    subject: person-john
    claim: occupation
    value: "blacksmith"
    citations:
      - citation-1851-census
      - citation-trade-directory
    confidence: high
```

Properties record **what** we believe. Assertions document **why** we believe it, with evidence.

## Evidence Chain

GENEALOGIX organizes genealogical evidence in a hierarchical chain from physical sources to conclusions:

### Complete Evidence Chain

```
Repository → Source → Citation → Assertion → Property
    ↓          ↓         ↓          ↓           ↓
 Physical   Original  Specific  Evidence-   Concluded
 Location   Material  Reference  Based      Value on
                                 Claim      Entity
```

Each level provides context and traceability for the research:

**1. Repository** - Physical or digital institution holding sources
```yaml
repositories:
  repository-gro:
    name: General Register Office
    address: "London, England"
```

**2. Source** - Original document, record, or material
```yaml
sources:
  source-birth-register:
    title: England Birth Register 1850
    repository: repository-gro
```

**3. Citation** - Specific reference within the source
```yaml
citations:
  citation-john-birth:
    source: source-birth-register
    properties:
      locator: "Volume 23, Page 145, Entry 23"
      text_from_source: "John Smith, born 15 January 1850, Leeds"
```

**4. Assertion** - Evidence-based conclusion
```yaml
assertions:
  assertion-john-born:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations: [citation-john-birth]
    confidence: high
```

**5. Property** - Concluded value on entity
```yaml
persons:
  person-john-smith:
    properties:
      born_on: "1850-01-15"
```

### Source Attribution

Every assertion must reference specific citations or sources, creating complete provenance:

```yaml
assertions:
  assertion-smith-occupation:
    subject: person-john-smith
    claim: occupation
    value: "blacksmith"
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
    confidence: high
```

This links the conclusion ("blacksmith") directly to three independent sources, enabling readers to verify the evidence.

### Research Notes and Analysis

Structured fields document research decisions:

```yaml
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

### Media Documentation

Media entities (photographs, scans, audio) can document sources:

```yaml
# Media documenting the source
media:
  media-birth-cert-scan:
    uri: "file:///media/birth-certificates/john-smith-1850.jpg"
    type: document
    title: "Birth Certificate Scan - John Smith"

# Citation references the media
citations:
  citation-birth-cert:
    source: source-gro-register
    media: [media-birth-cert-scan]
    properties:
      locator: "Certificate 1850-LEEDS-00145"
```

Media is not required but highly recommended for preservation and verification.

### Change Tracking with Git

Since GENEALOGIX archives are Git repositories, all changes are automatically tracked:

```bash
# See complete change history
git log --oneline -- persons/person-john-smith.glx

# See who made what changes
git blame persons/person-john-smith.glx

# Track research progress over time
git log --since="2024-01-01" --until="2024-03-31"
```

Git provides automatic provenance tracking for all research work, showing when conclusions were drawn and how they evolved over time.

## Version Control Ready

GENEALOGIX is designed from the ground up for Git version control:

- **File-per-entity structure**: Each entity in a separate file enables clean, focused diffs
- **YAML format**: Human-readable, merge-friendly, and Git-optimized
- **Entity-level granularity**: Changes to one person don't affect other files
- **Branch-based research**: Isolate hypotheses and investigations in branches
- **Collaborative workflows**: Multiple researchers can work simultaneously with standard Git merge tools
- **Complete audit trail**: Git tracks every change, who made it, and when

For detailed Git workflows, branching strategies, and collaboration patterns, see the [Git Workflow Guide](#) (coming soon).

This core concept architecture ensures that GENEALOGIX archives are reliable, verifiable, and maintainable over time.
