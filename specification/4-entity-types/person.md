---
title: Person Entity
description: Individual representation in GENEALOGIX - identity, properties, and lifecycle
layout: doc
---

# Person Entity

[← Back to Entity Types](README.md)

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
| `tags` | array | Tags for categorization |

## Required Fields

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Recommended formats:
  - Descriptive: `person-john-smith-1850`, `person-mary-jones`
  - Random hex: `person-a1b2c3d4` (for collaboration)
  - Sequential: `person-001`, `person-002`

## Optional Fields

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

**Key Points:**
- All properties are optional
- Property names and types are validated against the `person_properties` vocabulary
- Properties can be temporal (change over time) - see [Data Types](../6-data-types.md#temporal-values)
- Custom properties can be added by extending the vocabulary
- Living status is implied by the presence/absence of `died_on`

## Complete Example

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
    tags:
      - maternal-line
      - smith-family
      - ohio-branch
```

## Schema Reference

See [person.schema.json](../schema/v1/person.schema.json) for the
complete JSON Schema definition.

## See Also

- [Assertion Entity](assertion.md)
- [Relationship Entity](relationship.md)
- [Data Types](../6-data-types.md)
- [Provenance Tracking](../2-core-concepts#provenance-tracking)
