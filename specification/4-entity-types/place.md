---
title: Place Entity
description: Geographic locations with hierarchical organization for genealogical research
layout: doc
---

# Place Entity

[← Back to Entity Types](README)

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
| `parent` | string | Reference to parent place in hierarchy |
| `type` | string | Place type from `vocabularies/place-types.glx` |
| `latitude` | number | WGS84 latitude coordinate |
| `longitude` | number | WGS84 longitude coordinate |
| `notes` | string | Free-form notes about the place |

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

**See [Vocabularies - Place Properties](vocabularies#place-properties-vocabulary) for the full vocabulary definition.**

## Place Types

Place types are defined in `vocabularies/place-types.glx` within each archive.

**See [Vocabularies - Place Types](vocabularies#place-types-vocabulary) for:**
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
      existed_from: "1624"
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

```
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
| `parent` | (implicit) | Represented in hierarchical PLAC structure |
| `type` | PLAC.TYPE | Non-standard; used in extended GEDCOM |
| `latitude` | PLAC.MAP.LATI | WGS84 latitude |
| `longitude` | PLAC.MAP.LONG | WGS84 longitude |
| `properties.place_format` | PLAC.FORM | Place hierarchy format string |

## Validation Rules

- Place hierarchy must be acyclic (no circular parent references)
- Coordinates, if present, must be valid WGS84 values
- Parent place must reference an existing Place entity
- Type must be from the [place types vocabulary](vocabularies#place-types-vocabulary)

## Schema Reference

See [place.schema.json](../schema/v1/place.schema.json) for the complete JSON Schema definition.

## See Also

- [Event Entity](event) - Events that occur at places
- [Person Entity](person) - Residence and birth/death places
- [Vocabularies](vocabularies#place-types-vocabulary) - Place types vocabulary
- [Core Concepts - Data Types](../2-core-concepts#data-types) - Coordinate and date formats

