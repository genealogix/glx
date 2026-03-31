---
title: Person Entity
description: Individual representation in GENEALOGIX - identity, properties, and lifecycle
layout: doc
---

# Person Entity

[← Back to Entity Types](README)

## Overview

The Person entity represents an individual in the family archive. Person entities can be stored in any `.glx` file in the repository under the `persons` key.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in persons/ directory)
persons:
  person-john-smith-1850:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
```

**Key Points:**
- Entity ID is the map key (`person-john-smith-1850`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `properties` | object | Vocabulary-defined properties (name, gender, occupation, etc.) |
| `notes` | string | Free-form notes about the person |

> **Note:** While no properties are technically required, the `name` property is recommended for most person records as it enables meaningful identification and display.

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Example formats:
  - Descriptive: `john-smith-1850`, `mary-jones`
  - Random hex: `a1b2c3d4`
  - Prefixed: `person-a1b2c3d4`
  - Sequential: `001`, `002`

### `properties`

- Type: Object
- Required: No
- Description: Vocabulary-defined properties representing the concluded/accepted values for this person

Structure:
```yaml
properties:
  name:                      # Unified name property with optional fields
    value: String           # Full name as recorded
    fields:                 # Optional structured breakdown
      type: String          # Name classification (birth, married, alias, etc.)
      prefix: String
      given: String
      nickname: String
      surname_prefix: String  # e.g., von, van, de
      surname: String
      suffix: String
  gender: String            # From person_properties vocabulary
  occupation: String        # From person_properties vocabulary
  residence: "place-id"    # From person_properties vocabulary (reference)
```

Example:
```yaml
properties:
  name:
    value: "John Smith"
    fields:
      given: "John"
      surname: "Smith"
  gender: "male"
  occupation: "blacksmith"
  residence:
    - value: "place-leeds"
      date: "FROM 1850 TO 1900"
    - value: "place-london"
      date: "FROM 1900 TO 1920"
```

#### Gender Property

The `gender` property is constrained by the `gender_types` vocabulary via `vocabulary_type: gender_types`. The standard vocabulary includes `male`, `female`, `unknown`, and `other` (with GEDCOM SEX mappings), but archives may extend it with additional entries. Values not found in the vocabulary produce a **warning**, not an error — this allows archives to use custom values before adding them to the vocabulary. The property is temporal, allowing changes to be recorded over time.

#### Temporal Property Examples

Properties marked as `temporal: true` can hold multiple values. Dates are optional — use them when you know when each value applied, omit them when you don't.

**Occupation Changes:**
```yaml
properties:
  occupation:
    - value: "farm laborer"
      date: "1865"
    - value: "blacksmith apprentice"
      date: "FROM 1867 TO 1870"
    - value: "blacksmith"
      date: "FROM 1870 TO 1890"
    - value: "farmer"
      date: "FROM 1890 TO 1920"
```

**Occupations Without Dates** (e.g., from an obituary):
```yaml
properties:
  occupation:
    - value: "teacher"
    - value: "school principal"
    - value: "county superintendent"
```

**Name Changes (e.g., marriage):**
```yaml
properties:
  name:
    - value: "Mary Jones"
      date: "FROM 1855 TO 1880"
      fields:
        given: "Mary"
        surname: "Jones"
    - value: "Mary Smith"
      date: "FROM 1880"
      fields:
        given: "Mary"
        surname: "Smith"
```

#### Name Variation Examples

The `type` field on name entries distinguishes why multiple names exist. Use it not just for temporal changes (birth to married) but also for alternate spellings, abbreviations, and names as recorded in specific documents.

The authoritative list of name `type` values is defined in the [person properties vocabulary](vocabularies#person-properties-vocabulary). The table below highlights common values and how to use them.

**Common name type values:**

| Type | Usage |
|------|-------|
| `birth` | Name at birth |
| `married` | Name after marriage |
| `maiden` | Name used before marriage (typically pre-marriage surname) |
| `alias` | Known alternate identity |
| `aka` | Also known as (general alternate name) |
| `immigrant` | Name used when immigrating |
| `anglicized` | Anglicized or localized form of a foreign name |
| `religious` | Name taken for religious purposes |
| `formal` | Formal or legal name |
| `professional` | Professional or stage name |
| `as_recorded` | Name exactly as it appears in a source document |

**Alternate Spellings and Abbreviations:**
```yaml
properties:
  name:
    - value: "Robert Webb"
      fields:
        type: "birth"
        given: "Robert"
        surname: "Webb"
    - value: "R. Webb"
      fields:
        type: "as_recorded"
        given: "R."
        surname: "Webb"
```

**Anglicized Name:**
```yaml
properties:
  name:
    - value: "Johann Schmidt"
      fields:
        type: "birth"
        given: "Johann"
        surname: "Schmidt"
    - value: "John Smith"
      date: "FROM 1885"
      fields:
        type: "anglicized"
        given: "John"
        surname: "Smith"
```

**Multiple Known Aliases:**
```yaml
properties:
  name:
    - value: "William Henry McCarty"
      fields:
        type: "birth"
        given: "William Henry"
        surname: "McCarty"
    - value: "William H. Bonney"
      fields:
        type: "alias"
        given: "William H."
        surname: "Bonney"
    - value: "Billy the Kid"
      fields:
        type: "aka"
```

> **Tip:** Dates are optional on name entries. Use them when you know when a name was adopted; omit them for variations that aren't tied to a specific time period. The `type` field clarifies why each entry exists without implying temporal succession.

**Multiple Residences:**
```yaml
properties:
  residence:
    - value: "place-leeds"
      date: "FROM 1850 TO 1870"
    - value: "place-manchester"
      date: "FROM 1870 TO 1885"
    - value: "place-london"
      date: "FROM 1885 TO 1920"
```

**Nationality/Citizenship Changes:**
```yaml
properties:
  nationality:
    - value: "British Subject"
      date: "FROM 1850 TO 1895"
    - value: "American Citizen"
      date: "FROM 1895"
```

> **See Also:** [Temporal Properties Example](/examples/temporal-properties/) for a complete working archive demonstrating temporal values with assertions and evidence chains.

**See [Vocabularies - Person Properties](vocabularies#person-properties-vocabulary) for the full vocabulary definition.**

**Key Points:**
- All properties are optional
- Property names and types are validated against the `person_properties` vocabulary
- The `gender` property is constrained by the [gender types vocabulary](vocabularies#property-definition-structure) — out-of-vocabulary values produce a warning
- Properties can be temporal (change over time) - see [Core Concepts - Data Types](../2-core-concepts#temporal-properties)
- Custom properties can be added by extending the vocabulary
- Birth and death information is stored on [Event entities](event) of type `birth` and `death`, not as person properties

## Usage Patterns

### Basic Person

```yaml
# persons/person-john.glx
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"
```

### Person with Full Details

```yaml
# persons/person-margaret-smith.glx
persons:
  person-margaret-smith-1825:
    properties:
      name:
        value: "Margaret Eleanor Smith"
        fields:
          given: "Margaret Eleanor"
          surname: "Smith"
      gender: "female"
    notes: |
      Family tradition says she was named after her grandmother.
      Need to verify with census records.
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Person files are typically stored in a `persons/` directory:

```
persons/
├── person-john-smith.glx
├── person-mary-jones.glx
├── person-margaret-smith.glx
└── person-thomas-brown.glx
```

## GEDCOM Mapping

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Entity ID (map key) | `@INDI@` | Individual record ID |
| `properties.name` | `INDI.NAME` | Person's name |
| `properties.name.fields.given` | `INDI.NAME.GIVN` | Given name |
| `properties.name.fields.surname` | `INDI.NAME.SURN` | Surname |
| `properties.gender` | `INDI.SEX` | M/F mapped to male/female |
| `properties.occupation` | `INDI.OCCU` | Occupation |
| `properties.residence` | `INDI.RESI` | Residence |
| `notes` | `INDI.NOTE` | Notes |

**Note:** GEDCOM birth/death events (`INDI.BIRT`, `INDI.DEAT`) are imported as [Event entities](event) of type `birth` and `death`, not as person properties.

## Validation Rules

- Properties should be from the [person properties vocabulary](vocabularies#person-properties-vocabulary) (unknown properties generate warnings)
- All place references must point to existing Place entities
- Date formats must follow genealogical date conventions

## Schema Reference

See [person.schema.json](../schema/v1/person.schema.json) for the
complete JSON Schema definition.

## See Also

- [Event Entity](event) - Life events for this person
- [Relationship Entity](relationship) - Connections to other people
- [Assertion Entity](assertion) - Evidence for person properties
- [Core Concepts - Data Types](../2-core-concepts#data-types) - Date and property formats
- [Vocabularies](vocabularies#person-properties-vocabulary) - Person properties vocabulary
