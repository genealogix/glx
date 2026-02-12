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
| `properties` | object | Vocabulary-defined properties (name, gender, dates, etc.) |
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
      given: String
      surname: String
      prefix: String
      suffix: String
  gender: String            # From person_properties vocabulary
  born_on: Date            # From person_properties vocabulary
  born_at: "place-id"      # From person_properties vocabulary (reference)
  died_on: Date            # From person_properties vocabulary
  died_at: "place-id"      # From person_properties vocabulary (reference)
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
  born_on: "1850-01-15"
  born_at: "place-leeds"
  died_on: "1920-06-20"
  died_at: "place-london"
  occupation: "blacksmith"
  residence:
    - value: "place-leeds"
      date: "FROM 1850 TO 1900"
    - value: "place-london"
      date: "FROM 1900 TO 1920"
```

#### Temporal Property Examples

Properties marked as `temporal: true` can change over time. Here are common examples:

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

> **See Also:** [Temporal Properties Example](../../docs/examples/temporal-properties/) for a complete working archive demonstrating temporal values with assertions and evidence chains.

**Key Points:**
- All properties are optional
- Property names and types are validated against the `person_properties` vocabulary
- Properties can be temporal (change over time) - see [Core Concepts - Data Types](../2-core-concepts#temporal-properties)
- Custom properties can be added by extending the vocabulary
- Whether a person is living or deceased is implied by the presence/absence of `died_on`

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
      born_on: "1825-04-10"
      died_on: "1890-11-22"
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
| `properties.born_on` | `INDI.BIRT.DATE` | Birth date |
| `properties.born_at` | `INDI.BIRT.PLAC` | Birth place (reference) |
| `properties.died_on` | `INDI.DEAT.DATE` | Death date |
| `properties.died_at` | `INDI.DEAT.PLAC` | Death place (reference) |
| `properties.occupation` | `INDI.OCCU` | Occupation |
| `properties.residence` | `INDI.RESI` | Residence |
| `notes` | `INDI.NOTE` | Notes |

**Note:** GEDCOM stores birth/death as events with dates and places. GLX imports these as person properties for convenience, while the full event details are preserved in Event entities.

## Validation Rules

- Properties must be from the [person properties vocabulary](vocabularies#person-properties-vocabulary)
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
