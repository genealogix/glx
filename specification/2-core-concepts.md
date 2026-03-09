---
title: Core Concepts
description: Fundamental principles and architecture of GENEALOGIX - flexibility, entities, properties, assertions, and evidence chains
layout: doc
---

# Core Concepts

This section explains how GENEALOGIX works: the flexible vocabularies, entities, and evidence model that make GLX different from traditional genealogy formats.

## Archive-Owned Vocabularies

### Why Archive-Level Control?

Unlike traditional genealogy formats with fixed type systems, GENEALOGIX uses **archive-owned controlled vocabularies**. Each archive defines its own valid types in vocabulary files, combining standardization with flexibility.

**This is GENEALOGIX's most powerful feature: each archive controls its own type system.**

#### Freedom and Flexibility

Unlike formats with centrally-defined types, GLX lets each archive define exactly what it needs. You can add custom entries to any vocabulary — event types, relationship types, place types, and more:

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

When you create an archive with `glx init` or `glx import`, standard vocabulary files are automatically created (by default in a `vocabularies/` directory, though they can live anywhere in the archive):

| File | Contents |
|------|----------|
| `event-types.glx` | Birth, death, baptism, immigration, etc. |
| `relationship-types.glx` | Marriage, parent-child, adoption, etc. |
| `place-types.glx` | Country, city, parish, etc. |
| `source-types.glx` | Vital records, census, church registers, etc. |
| `repository-types.glx` | Archive, library, church, etc. |
| `media-types.glx` | Photo, document, audio, etc. |
| `participant-roles.glx` | Subject, witness, godparent, etc. |
| `confidence-levels.glx` | High, medium, low, disputed |
| `person-properties.glx` | Person properties (name, occupation, etc.) |
| `event-properties.glx` | Event properties |
| `relationship-properties.glx` | Relationship properties |
| `place-properties.glx` | Place properties |
| `media-properties.glx` | Media properties |
| `repository-properties.glx` | Repository properties |
| `source-properties.glx` | Source properties |
| `citation-properties.glx` | Citation properties |

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

**1. Edit the appropriate vocabulary files**

```yaml
# participant-roles.glx
participant_roles:
  # ... standard roles ...

  # Your custom roles
  apprentice:
    label: "Apprentice"
    description: "Person learning a trade"
    applies_to:
      - event
  master:
    label: "Master"
    description: "Person teaching a trade"
    applies_to:
      - event
```

```yaml
# event-types.glx
event_types:
  # ... standard types ...

  # Your custom type
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"
```

**2. Use the new types in your data**

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
      - person: person-thomas-brown
        role: master
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
- Unknown assertion properties not defined in property vocabularies
- Warnings allow rapid data entry and emerging properties without breaking validation

This policy balances strictness (broken references are errors) with flexibility (unknown properties generate warnings, not errors).

This flexibility makes GLX suitable for traditional family history, local and community history, biographical research, prosopography (collective biography), historical demography, and any research involving people, events, and relationships.

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
3. **Event** - Occurrences in time and place (birth, death, marriage, immigration)
4. **Place** - Geographic locations with hierarchical structure
5. **Source** - Information sources (books, records, websites, databases)
6. **Citation** - Specific references within sources
7. **Repository** - Institutions holding sources (archives, libraries, churches)
8. **Media** - Digital objects (photos, documents, audio, video)
9. **Assertion** - Evidence-based conclusions about facts

### Validation Dependencies

These relationships create validation requirements that ensure archive integrity. The `glx validate` command enforces referential integrity (citations → sources, assertions → citations/sources/media, events → places, participants → persons, relationships → persons). See [Validation Behavior](#validation-behavior) for complete validation policy.

## Data Types

GENEALOGIX uses fundamental data types throughout the specification for entity properties and values.

### Primitive Types

#### String

A sequence of Unicode characters. Strings are the default type when no specific type is specified in property definitions.

**Example:**
```yaml
name:
  value: "John Smith"
  fields:
    given: "John"
    surname: "Smith"
occupation: "blacksmith"
```

#### Integer

A whole number (positive, negative, or zero). Used for numeric values like population counts.

**Example:**
```yaml
population: 5000
```

#### Boolean

A true/false value.

**Example:**
```yaml
verified: true
primary_source: false
```

#### Date

A calendar date or fuzzy date specification. GENEALOGIX uses YYYY-MM-DD format for precise dates combined with FamilySearch-inspired keywords for fuzzy dates.

### Date Format Standard

GENEALOGIX uses a **hybrid date format** combining:
- **YYYY-MM-DD dates** for precise calendar dates
- **FamilySearch-inspired keywords** for approximate, ranged, and calculated dates

This format supports both precise dates and fuzzy/approximate dates commonly encountered in genealogical research.

#### Format Specification

**Simple Dates:**
- `YYYY` - Year only (4 digits required, e.g., `1850`, `2020`, `0047`)
- `YYYY-MM` - Year and month (e.g., `1850-03`, `2020-12`)
- `YYYY-MM-DD` - Full date (e.g., `1850-03-15`, `2020-12-31`)

**Keyword Modifiers (FamilySearch-inspired):**
- **Approximate Dates:**
  - `ABT YYYY` - About/approximately (e.g., `ABT 1850`)
  - `BEF YYYY` - Before (e.g., `BEF 1920`)
  - `AFT YYYY` - After (e.g., `AFT 1880`)
  - `CAL YYYY` - Calculated (e.g., `CAL 1850`)

- **Date Ranges:**
  - `BET YYYY AND YYYY` - Between two dates (e.g., `BET 1880 AND 1890`)
  - `FROM YYYY TO YYYY` - Range with start and end (e.g., `FROM 1900 TO 1950`)
  - `FROM YYYY` - Open-ended range from a start date (e.g., `FROM 1900`)

- **Interpreted Dates:**
  - `INT YYYY-MM-DD (original text)` - Interpreted from original source (e.g., `INT 1850-03-15 (March 15th, 1850)`)

#### Important Notes

1. **Year Format:** Years must be exactly 4 digits. Pad with zeros for years before 1000 CE (e.g., `0047` for year 47, `0800` for year 800).

2. **Date Format:** GENEALOGIX uses YYYY-MM-DD format (e.g., `1850-03-15` for March 15, 1850). This is the international standard for date representation, chosen for its clarity and sortability.

3. **Keywords vs Full Format:** GENEALOGIX uses **keywords inspired by** the FamilySearch Normalized Date Format, but the underlying date representation uses YYYY-MM-DD, not the full FamilySearch format.

4. **Keyword Combinations:** Keywords can be combined with any simple date format (e.g., `ABT 1850`, `ABT 1850-03`, `ABT 1850-03-15`).

#### Date Examples

```yaml
# Precise dates
born_on: "1850-03-15"      # Full date
born_on: "1850-03"          # Year and month
born_on: "1850"             # Year only
born_on: "0047"             # Year 47 AD (zero-padded)

# Approximate dates
born_on: "ABT 1850"         # About 1850
died_on: "BEF 1920"         # Before 1920
born_on: "AFT 1880-06"      # After June 1880

# Date ranges
residence:
  - value: "place-leeds"
    date: "FROM 1900 TO 1950"  # Lived in Leeds 1900-1950
  - value: "place-london"
    date: "FROM 1950"           # Lived in London from 1950 onward

# Fuzzy dates
born_on: "BET 1880 AND 1890"   # Born between 1880 and 1890

# Calculated dates
born_on: "CAL 1850"             # Birth year calculated from other evidence

# Interpreted dates
born_on: "INT 1850-03-15 (15th March 1850)"  # Original text preserved
```

#### Date Validation

GENEALOGIX validates date formats at two levels:
1. **Structure:** Dates must follow the format specifications above
2. **Keywords:** Only the defined keywords (FROM, TO, ABT, BEF, AFT, BET, AND, CAL, INT) are recognized

Invalid date formats will generate validation warnings (not errors), allowing archives with imperfect dates to still load while alerting researchers to potential data quality issues.

### Reference Types

Reference types indicate that a property value is a string identifier that must exist as an entity in the archive. References are validated at runtime against the actual entities in the archive.

#### Supported Reference Types

- **persons** - Reference to a person entity
- **places** - Reference to a place entity
- **events** - Reference to an event entity
- **relationships** - Reference to a relationship entity
- **sources** - Reference to a source entity
- **citations** - Reference to a citation entity
- **repositories** - Reference to a repository entity
- **media** - Reference to a media entity

**Example:**
```yaml
properties:
  born_at: "place-leeds"  # Reference to a place
  residence:              # Temporal reference to a place
    - value: "place-london"
      date: "FROM 1900 TO 1920"
```

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
      residence: "place-leeds"  # Single-value shorthand; see Temporal Properties for list format
```

### Defined by Property Vocabularies

All properties are defined in property vocabularies (see Archive-Owned Vocabularies above). The `person-properties.glx` vocabulary defines:
- What properties exist (`name`, `gender`, `born_on`, `occupation`, etc.)
- Their data types (string, date, place reference)
- Whether they can change over time (temporal)
- Whether they have structured fields

### Properties Can Exist Without Assertions

Properties can be set without assertions, supporting rapid data entry. You can add assertions later as you research sources. See [How Properties and Assertions Work Together](#how-properties-and-assertions-work-together) for examples.

### Temporal Properties

Properties marked as `temporal: true` in vocabularies can hold multiple values — either dated (for values that change over time) or undated (for multiple values without known dates). They support three formats:

**Single Value** (for properties that don't change or represent a point in time):
```yaml
properties:
  gender: "male"
  born_on: "1850-01-15"
```

**Dated List** (for values that change over time):

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

**Undated List** (for multiple values without date information):

```yaml
# An obituary lists occupations but no dates
properties:
  occupation:
    - value: "teacher"
    - value: "school principal"
    - value: "county superintendent"
```

This is common when a source (like an obituary, biographical sketch, or family letter) mentions multiple values but doesn't specify when each applied. The list format captures all known values without forcing artificial dates.

Each list entry includes:
- `value` - The property value, conforming to the property's `value_type` or `reference_type`
- `date` - Optional date string specifying when the value applied

Dated and undated entries can be mixed in the same list — use dates where you have them, omit where you don't.

### Structured Properties

Properties can have structured fields for complex data. There are three usage patterns:

**1. Value only** (simple properties):
```yaml
properties:
  occupation: "blacksmith"
  religion: "Church of England"
```

**2. Value + Fields** (preserve original while providing structure):
```yaml
properties:
  name:
    value: "Dr. John Smith Jr."
    fields:
      type: "birth"
      prefix: "Dr."
      given: "John"
      surname: "Smith"
      suffix: "Jr."
```

The `value` field preserves the original recorded form, while `fields` provide structured access to components. This is the recommended approach for most structured properties.

**3. Fields only** (when there's no natural single-value representation):
```yaml
properties:
  crop:
    fields:
      top: 450
      left: 100
      width: 800
      height: 200
```

### Notes Field

All entities support an optional `notes` field for free-form text:

```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
    notes: |
      Research notes about this person.
      Questions for future investigation.
```

Use notes to:
- Document research decisions and uncertainties
- Record questions for future investigation
- Provide context not captured elsewhere

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
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations:
      - citation-birth-certificate
      - citation-baptism-record
    confidence: high
```

### How Assertions Work

**Core fields:**
- `subject`: Typed reference to the entity this assertion is about (person, event, relationship, place)
- `property`: The property being asserted (references property vocabulary)
- `value`: The concluded value of the property
- `citations`, `sources`, or `media`: Evidence supporting this assertion (at least one required)
- `confidence`: How certain we are based on evidence quality
- `status`: Research state of the assertion (e.g., `proven`, `speculative`, `disproven`) — independent of confidence

**The `property` field references property vocabularies:**

```yaml
# The property "born_on" must be defined in person-properties.glx
assertion-john-birth:
  subject:
    person: person-john-smith
  property: born_on  # Validated against person_properties vocabulary
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

Every assertion requires at least one citation, source, or media reference, creating an audit trail from conclusion back to original evidence.

### Conflicting Evidence

Multiple assertions can exist for the same fact, representing conflicting evidence:

```yaml
assertions:
  # Assertion based on birth certificate
  assertion-mary-birth-cert:
    subject:
      person: person-mary-jones
    property: born_on
    value: "1852-03-10"
    citations: [citation-birth-cert]
    confidence: high
    notes: "Primary direct evidence"

  # Assertion based on family Bible
  assertion-mary-birth-bible:
    subject:
      person: person-mary-jones
    property: born_on
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
  high:
    label: "High Confidence"
    description: "Multiple high-quality sources agree, minimal uncertainty"
  medium:
    label: "Medium Confidence"
    description: "Some evidence supports conclusion, but conflicts or gaps exist"
  low:
    label: "Low Confidence"
    description: "Limited evidence, significant uncertainty"
  disputed:
    label: "Disputed"
    description: "Multiple sources conflict, resolution unclear"
```

Confidence levels are defined in `confidence-levels.glx` and can be customized per archive.

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
    subject:
      person: person-john
    property: born_on
    value: "1850-01-15"
    citations: [citation-birth-cert]
    confidence: high

  assertion-john-occupation:
    subject:
      person: person-john
    property: occupation
    value: "blacksmith"
    date: "FROM 1870 TO 1890"
    citations:
      - citation-1851-census
      - citation-trade-directory
    confidence: high
```

Properties record **what** we believe. Assertions document **why** we believe it, with evidence. For temporal properties like occupation, the assertion's `date` field specifies **when** the value applies.

### Existential Assertions

An assertion with only a `subject` and evidence — no `property`, `value`, or `participant` — is an **existential assertion**. It simply says: "this entity is evidenced by these sources."

```yaml
assertions:
  assertion-john-alice-parentage:
    subject:
      relationship: rel-john-alice-parent-child
    citations:
      - citation-1880-census
    confidence: high
    notes: "Census shows John Smith as head of household with Alice listed as daughter"
```

Adding a `date` makes it temporal — "this entity existed at this time":

```yaml
assertions:
  assertion-john-alice-parentage:
    subject:
      relationship: rel-john-alice-parent-child
    date: "1880"
    citations:
      - citation-1880-census
    confidence: high
```

**When to use existential assertions:**
- **Relationships** — evidence that a parent-child or marriage relationship existed, before asserting specific property values
- **Events** — confirming an event occurred without yet asserting its date or place
- **Places** — documenting that a place existed at a given time

Existential assertions are useful during early research phases when you have evidence that an entity exists but haven't yet established specific property values. They let you document the evidence chain immediately, then add property assertions later as research progresses.

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

Each level provides context and traceability for research. Here's a complete example showing all links in the chain:

```yaml
# 1. Repository - Physical institution
repositories:
  repository-gro:
    name: General Register Office
    address: "London, England"

# 2. Source - Original document
sources:
  source-birth-register:
    title: England Birth Register 1850
    repository: repository-gro

# 3. Citation - Specific reference
citations:
  citation-john-birth:
    source: source-birth-register
    properties:
      locator: "Volume 23, Page 145, Entry 23"
      text_from_source: "John Smith, born 15 January 1850, Leeds"

# 4. Assertion - Evidence-based conclusion
assertions:
  assertion-john-born:
    subject:
      person: person-john-smith
    property: born_on
    value: "1850-01-15"
    citations: [citation-john-birth]
    confidence: high

# 5. Property - Concluded value
persons:
  person-john-smith:
    properties:
      born_on: "1850-01-15"
```

### Multiple Citations and Corroboration

Assertions can reference multiple citations, showing corroboration from independent sources:

```yaml
assertions:
  assertion-smith-occupation:
    subject:
      person: person-john-smith
    property: occupation
    value: "blacksmith"
    citations:
      - citation-1851-census
      - citation-trade-directory
      - citation-parish-record
    confidence: high
```

### Research Notes

Use the `notes` field to document research decisions, conflicting evidence, and uncertainties:

```yaml
assertions:
  assertion-disputed-birth:
    subject:
      person: person-john-smith
    property: born_on
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


## Collaboration

### Version Control Ready

GENEALOGIX is designed from the ground up for Git version control:

- **File-per-entity structure**: Each entity in a separate file enables clean, focused diffs
- **YAML format**: Human-readable, merge-friendly, and Git-optimized
- **Entity-level granularity**: Changes to one person don't affect other files
- **Branch-based research**: Isolate hypotheses and investigations in branches
- **Collaborative workflows**: Multiple researchers can work simultaneously with standard Git merge tools
- **Complete audit trail**: Git tracks every change, who made it, and when

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

## Next Steps

Now that you understand the core concepts and architecture, the next step is understanding how to organize your archive files. See [Archive Organization](3-archive-organization) for details on file formats, directory structures, and organization strategies.