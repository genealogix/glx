---
title: Person Entity
description: Individual representation in GENEALOGIX - identity, properties, and lifecycle
layout: doc
---

# Person Entity

[← Back to Entity Types](README.md)

## Overview

The Person entity represents an individual in the family archive. Person entities can be stored in any `.glx` file in the archive under the `persons` key.

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
| `properties` | object | Vocabulary-defined properties (name, sex, gender, occupation, etc.) |
| `notes` | string \| string[] | Free-form notes about the person |

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
  sex: String               # From person_properties vocabulary (recorded in source)
  gender: String            # From person_properties vocabulary (self-identified)
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
  sex: "male"
  occupation: "blacksmith"
  residence:
    - value: "place-leeds"
      date: "FROM 1850 TO 1900"
    - value: "place-london"
      date: "FROM 1900 TO 1920"
```

#### Sex and Gender Properties

GLX splits "recorded sex" and "gender identity" into two separate properties (issue #528).

- **`sex`** — What the source recorded. Maps to GEDCOM `SEX` on import/export. Constrained by the `sex_types` vocabulary via `vocabulary_type: sex_types`. Standard values: `male` (GEDCOM `M`), `female` (`F`), `unknown` (`U`), `not_recorded` (`N` or empty `SEX` value), `other` (`X`). A GEDCOM record with no `SEX` tag at all imports without a `sex` property — `not_recorded` is reserved for sources that explicitly indicate the sex field is absent. Use `unknown` when the source was consulted but the sex could not be determined.
- **`gender`** — Self-identified gender identity. Constrained by the `gender_types` vocabulary. Standard values: `male`, `female`, `nonbinary`, `other`. No direct GEDCOM mapping (GEDCOM 7.0 defers identity to `FACT`). Primarily useful for modern records and living persons; for most historical genealogy only `sex` will be populated.

Both properties are temporal, allowing changes over time. Out-of-vocabulary values produce a **warning**, not an error — archives may use custom values before adding them to the vocabulary.

#### `living` — Privacy Marker

`living` is a non-temporal boolean property used to opt a person into export-time privacy filters such as `glx export --privatize-living`. Setting it to `true` is the explicit way to say "redact this person's data on export"; setting it to `false` is the explicit way to opt out of the date-based heuristic for someone the tool would otherwise guess is living. When the property is absent, privacy filters fall back to a heuristic — no recorded death, burial, or cremation event AND a birth event less than 100 years before the current wall clock implies living. The property has no GEDCOM mapping; it is GLX-specific and never written to a GEDCOM file.

#### `status` — Research Placeholders

`status` is a non-temporal property that marks a person as a research placeholder: someone the research implies exists but has not identified, such as an unknown ancestor pair predicted from a DNA match cluster. Its value comes from the [person statuses vocabulary](vocabularies.md#person-statuses-vocabulary). Leave it unset for an ordinary person. See [Research Placeholders](#research-placeholders) for how to use it.

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

The authoritative list of name `type` values is defined in the [person properties vocabulary](vocabularies.md#person-properties-vocabulary). The table below highlights common values and how to use them.

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

> **See Also:** [Temporal Properties Example](../../docs/examples/temporal-properties/README.md) for a complete working archive demonstrating temporal values with assertions and evidence chains.

**See [Vocabularies - Person Properties](vocabularies.md#person-properties-vocabulary) for the full vocabulary definition.**

**Key Points:**

- All properties are optional
- Property names and types are validated against the `person_properties` vocabulary
- The `sex` and `gender` properties are constrained by the [sex types](../5-standard-vocabularies/#sex-types) and [gender types](../5-standard-vocabularies/#gender-types) vocabularies — out-of-vocabulary values produce a warning
- Properties can be temporal (change over time) - see [Core Concepts - Data Types](../2-core-concepts.md#temporal-properties)
- Custom properties can be added by extending the vocabulary
- Birth and death information is stored on [Event entities](event.md) of type `birth` and `death`, not as person properties

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
      sex: "male"
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
      sex: "female"
    notes: |
      Family tradition says she was named after her grandmother.
      Need to verify with census records.
```

### Research Placeholders

DNA-driven methods such as the Leeds Method build a predicted family tree before anyone in it has a name. A placeholder gives each unknown node an entity ID, so relationships, assertions and research logs can point at it and the tree can be refined as evidence accumulates:

```yaml
persons:
  person-unknown-ggp-8:
    properties:
      name:
        value: "Unknown Great-Grandparent #8 (Green parents)"
      status: predicted
    notes: "Predicted from DNA cluster 8. Geography: Brooke County, VA."
```

| Status | Meaning |
|--------|---------|
| `hypothetical` | Proposed as a working assumption; nothing points to this person directly yet |
| `predicted` | Inferred from indirect evidence such as DNA clusters; not yet identified |
| `resolved` | Began as a placeholder and has since been identified |

`status` says whether a person has been identified, not how strong the evidence is. Confidence belongs on [assertions](assertion.md), as it does for [hypothetical relationships](relationship.md); competing candidates for a placeholder's identity belong in [research log leads](research-log.md#researchlead).

**Resolving a placeholder.** When the person is identified, do one of the following:

- Update the placeholder in place, giving it the real name and `status: resolved`. This keeps a visible record that the person began as a prediction.
- Merge it with `glx merge-persons`, if the identified person already has a record. An unresolved status never survives a merge with an identified person, whichever side is kept.

**How tools treat placeholders.** `glx analyze` does not report missing births, deaths, events or parents for an unresolved placeholder, and does not suggest record searches for it. It lists the placeholder as open and gives an archive-wide count of open and resolved placeholders. A person whose only recorded parents are unresolved placeholders is still reported as having no identified parents. `glx query persons --status` filters by status, and `glx summary` shows it. A `resolved` person is treated like any other person.

Only the standard `hypothetical` and `predicted` values mark a person as unresolved; tools treat a custom status like any other person. GEDCOM cannot express this distinction, so `glx export` adds a `NOTE` to each unresolved placeholder's `INDI` record. That note imports back as an ordinary note, not as a `status`.

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Person files are typically stored in a `persons/` directory:

```text
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
| `properties.sex` | `INDI.SEX` | M→male, F→female, U→unknown, X→other; absent/N→not_recorded |
| `properties.gender` | (none) | No direct mapping; GEDCOM 7.0 defers gender identity to `FACT` |
| `properties.status` | `INDI.NOTE` | Export only, for unresolved placeholders; see [Research Placeholders](#research-placeholders) |
| `properties.occupation` | `INDI.OCCU` | Occupation |
| `properties.residence` | `INDI.RESI` | Residence |
| `notes` | `INDI.NOTE` | Notes |

**Note:** GEDCOM birth/death events (`INDI.BIRT`, `INDI.DEAT`) are imported as [Event entities](event.md) of type `birth` and `death`, not as person properties.

## Validation Rules

- Properties should be from the [person properties vocabulary](vocabularies.md#person-properties-vocabulary) (unknown properties generate warnings)
- `status` should be from the [person statuses vocabulary](vocabularies.md#person-statuses-vocabulary) (out-of-vocabulary values generate warnings)
- All place references must point to existing Place entities
- Date formats must follow the [date format standard](../2-core-concepts.md#date-format-standard) (invalid formats generate warnings)

## Schema Reference

See [person.schema.json](../schema/v1/person.schema.json) for the
complete JSON Schema definition.

## See Also

- [Event Entity](event.md) - Life events for this person
- [Relationship Entity](relationship.md) - Connections to other people
- [Assertion Entity](assertion.md) - Evidence for person properties
- [Core Concepts - Data Types](../2-core-concepts.md#data-types) - Date and property formats
- [Vocabularies](vocabularies.md#person-properties-vocabulary) - Person properties vocabulary
