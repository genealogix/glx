---
title: Place Entity
description: Geographic locations with hierarchical organization for genealogical research
layout: doc
---

# Place Entity

[← Back to Entity Types](README.md)

## Overview

A Place entity represents a geographic location relevant to the family archive. Places form a hierarchical structure that supports genealogical research across varying levels of granularity (country, region, county, town, street, etc.).

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in places/ directory)
places:
  place-leeds:
    name: "Leeds"
    type: city
    parent: place-yorkshire
```

**Key Points:**

- Entity ID is the map key (`place-leeds`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Core Concepts

### Place Hierarchy

Places form a tree structure where each place can have a parent place, enabling representation of administrative hierarchies and geographic containment relationships.

```yaml
places:
  place-england:
    name: "England"
    type: country

  place-yorkshire:
    name: "Yorkshire"
    type: county
    parent: place-england

  place-leeds:
    name: "Leeds"
    type: city
    parent: place-yorkshire
```

### Hierarchy That Changes Over Time

Jurisdictions change: counties move between states, territories become states, districts are merged and reorganized. When a place's parent changed, write `parent` as a list of entries, each with the parent place as `value` and the period it applied as `date` — the same `{value, date}` shape as [temporal properties](../2-core-concepts.md#temporal-properties):

```yaml
places:
  place-indiana-territory:
    name: "Indiana Territory"
    parent: place-usa

  place-indiana:
    name: "Indiana"
    type: state
    parent: place-usa

  place-wayne-in:
    name: "Wayne County"
    type: county
    parent:
      - value: place-indiana-territory
        date: "FROM 1811 TO 1816-12-10"
      - value: place-indiana
        date: "FROM 1816-12-11"
```

A marriage in Wayne County dated 1814 is then "Wayne County, Indiana Territory", and one dated 1817 is "Wayne County, Indiana". The plain string form remains valid and means the place had one parent throughout; use the list only where the hierarchy actually changed.

**Resolving the parent for a date.** Tools that have a date in hand (an event's date, a residence period) use the entry whose period overlaps that date the most; on a tie, the earlier entry in the list. A year-only date such as `1816` therefore resolves to Indiana Territory, which covers most of 1816. When no period overlaps the date, an undated entry is used if there is one, otherwise the entry whose period is nearest to the date.

**Default parent.** Where no date applies (a list of all places, the place's own page, JSON-LD export), tools use the default parent: the first entry without a `date`, if any; otherwise the most recent entry — the one whose period ends last, an open-ended `FROM` period counting as ending last. In the example above the default parent of Wayne County is Indiana.

Entry dates use the [date format](../2-core-concepts.md#date-format-standard); `FROM … TO …`, open-ended `FROM …` and `TO …` ranges are the natural fit. Periods for different parents should not overlap: boundaries that touch at the stated precision (`TO 1972` followed by `FROM 1972`) are fine, but a genuine overlap is reported as a validation warning.

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `name` | string | Current/primary place name |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `properties` | object | Vocabulary-defined properties of the place |
| `parent` | string \| object[] | Reference to parent place in hierarchy: a place ID, or a list of `{value, date}` entries when the parent changed over time (see [Hierarchy That Changes Over Time](#hierarchy-that-changes-over-time)) |
| `type` | string | Place type from `vocabularies/place-types.glx` |
| `latitude` | number | WGS84 latitude coordinate |
| `longitude` | number | WGS84 longitude coordinate |
| `notes` | string \| string[] | Free-form notes about the place |

### Properties

Place properties allow capturing historical information that doesn't fit into the standard structural fields. The following are standard properties from the default vocabulary; archives can define additional properties by extending the vocabulary.

| Property | Type | Description |
|----------|------|-------------|
| `existed_from` | date | When the place came into existence |
| `existed_to` | date | When the place ceased to exist |
| `population` | integer | Population count (supports temporal values) |
| `description` | string | Detailed description of the place |
| `jurisdiction` | string | Formal jurisdiction identifier or code (e.g., ISO 3166, FIPS code) |
| `place_format` | string | Standard format for place hierarchy (GEDCOM PLAC.FORM style) |
| `alternative_names` | string (temporal, multi-value) | Historical or alternate names for a place |
| `external_ids` | string (multi-value) | Identifiers from external geographic databases (e.g., GeoNames, Wikidata, OpenStreetMap) |

Example:

```yaml
places:
  place-new-amsterdam:
    name: "New Amsterdam"
    type: city
    properties:
      existed_from: "1626"
      existed_to: "1664"
      population:
        - value: 270
          date: "1630"
        - value: 1500
          date: "1664"
      description: "Dutch colonial settlement on Manhattan Island"
      alternative_names:
        - value: "Nieuw-Amsterdam"
          date: "FROM 1626 TO 1664"
```

**See [Vocabularies - Place Properties](vocabularies.md#place-properties-vocabulary) for the full vocabulary definition.**

## Place Types

Place types are defined in `vocabularies/place-types.glx` within each archive.

**See [Vocabularies - Place Types](vocabularies.md#place-types-vocabulary) for:**

- Complete list of standard place types
- How to add custom place types
- Vocabulary file structure and examples
- Validation requirements

## Usage Patterns

### Simple Location

```yaml
places:
  place-paris:
    name: "Paris"
    type: city
    parent: place-france
    latitude: 48.8566
    longitude: 2.3522
```

### Complex Hierarchical Location

```yaml
places:
  place-leeds-registration:
    name: "Leeds Registration District"
    type: district
    parent: place-yorkshire
    latitude: 53.8008
    longitude: -1.5491
    properties:
      jurisdiction: "england.yorkshire.leeds"
      place_format: "City, County, Country"
    notes: "Historical registration district for civil registration purposes"
```

### Place with Temporal Properties

```yaml
places:
  place-new-york-city:
    name: "New York City"
    type: city
    parent: place-new-york-state
    latitude: 40.7128
    longitude: -74.0060
    properties:
      population:
        - value: 60515
          date: "1800"
        - value: 202589
          date: "1830"
        - value: 3437202
          date: "1900"
        - value: 8336817
          date: "2020"
      existed_from: "1626"
```

### Referencing Places

Places are referenced in events and person properties:

```yaml
# In events
events:
  event-birth-john:
    type: birth
    place: place-leeds
    participants:
      - person: person-john
        role: subject

# In person properties
persons:
  person-john:
    properties:
      residence:
        - value: place-leeds
          date: "FROM 1850 TO 1900"
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Place files are typically stored in a `places/` directory:

```text
places/
├── countries/
│   ├── place-england.glx
│   ├── place-scotland.glx
│   └── place-usa.glx
├── regions/
│   ├── place-yorkshire.glx
│   ├── place-lancashire.glx
│   └── place-massachusetts.glx
└── cities/
    ├── place-leeds.glx
    ├── place-liverpool.glx
    └── place-boston.glx
```

## GEDCOM Mapping

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Entity ID (map key) | (synthetic) | Not in GEDCOM; generated from place data |
| `name` | PLAC | Text value of PLAC tag |
| `parent` | (implicit) | Represented in hierarchical PLAC structure; for a parent that changed over time, each event's PLAC uses the parent that applied at the event's date |
| `type` | PLAC.TYPE | Non-standard; used in extended GEDCOM |
| `latitude` | PLAC.MAP.LATI | WGS84 latitude |
| `longitude` | PLAC.MAP.LONG | WGS84 longitude |
| `properties.place_format` | PLAC.FORM | Place hierarchy format string |

## Validation Rules

- Place hierarchy must be acyclic (no circular parent references), counting the parents of every period of a temporal `parent`
- Coordinates, if present, must be valid WGS84 values
- `latitude` and `longitude` must be supplied together — setting one without the other is rejected by schema validation
- Parent place must reference an existing Place entity, including every entry of a temporal `parent`
- Entries of a temporal `parent` naming different parents should not have overlapping periods (warning)
- If `type` is specified, it must be from the [place types vocabulary](vocabularies.md#place-types-vocabulary)

## Schema Reference

See [place.schema.json](../schema/v1/place.schema.json) for the complete JSON Schema definition.

## See Also

- [Event Entity](event.md) - Events that occur at places
- [Person Entity](person.md) - Residence and birth/death places
- [Vocabularies](vocabularies.md#place-types-vocabulary) - Place types vocabulary
- [Core Concepts - Data Types](../2-core-concepts.md#data-types) - Date formats and reference types
