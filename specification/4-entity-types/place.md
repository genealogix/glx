---
title: Place Entity
description: Geographic locations with hierarchical organization for genealogical research
layout: doc
---

# Place Entity

[← Back to Entity Types](README.md)

## Overview

A Place entity represents a geographic location relevant to the family archive. Places form a hierarchical structure that supports genealogical research across varying levels of granularity (country, region, county, town, street, etc.).

## Core Concepts

### Place Hierarchy

Places form a tree structure where each place can have a parent place, enabling representation of administrative hierarchies and geographic containment relationships.

```yaml
# places/place-england.glx
places:
  place-england:
    name: "England"
    type: country

# places/place-yorkshire.glx
places:
  place-yorkshire:
    name: "Yorkshire"
    type: county
    parent: place-england

# places/place-leeds.glx
places:
  place-leeds:
    name: "Leeds"
    type: city
    parent: place-yorkshire
```

### Place Names

Places support multiple names to represent:
- Historical name changes
- Alternative spellings and transliterations
- Native language vs. colonial names
- Informal/colloquial names

Each name can be classified and dated.

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
| `alternative_names` | array | Historical/alternative names for this place |
| `latitude` | number | WGS84 latitude coordinate |
| `longitude` | number | WGS84 longitude coordinate |
| `jurisdiction` | string | Formal jurisdiction identifier or code |
| `place_format` | string | Standard format for place hierarchy (GEDCOM PLAC.FORM style) |
| `notes` | string | Free-form notes about the place |
| `tags` | array | Tags for categorization |

## Place Types

Place types are defined in `vocabularies/place-types.glx` within each archive.

**See [Vocabularies - Place Types](vocabularies.md#place-types-vocabulary) for:**
- Complete list of standard place types
- How to add custom place types
- Vocabulary file structure and examples
- Validation requirements

### Alternative Names Structure

```yaml
alternative_names:
  - name: "York"
    type: "historical"
    language: "en"
    date_range:
      start: "1066"
      end: "present"
  - name: "Jorvik"
    type: "historical"
    language: "en"
    date_range:
      start: "867"
      end: "1066"
```

## Usage Patterns

### In Events/Facts

Places are referenced in events to indicate where the event occurred:

```yaml
type: "birth"
place: "place-leeds123"
```

### In Addresses

Places can be components of addresses within person records or residence events:

```yaml
residence:
  place: "place-leeds123"
  date: "FROM 1850 TO 1900"
```

## GEDCOM Mapping

| GLX Property | GEDCOM Element | Notes |
|--------------|----------------|-------|
| Entity ID (map key) | (synthetic) | Not in GEDCOM; generated from place data |
| `name` | PLAC | Text value of PLAC tag |
| `parent` | (implicit) | Represented in hierarchical PLAC structure |
| `type` | PLAC.TYPE | Non-standard; used in extended GEDCOM |
| `latitude` | PLAC.MAP.LATI | WGS84 latitude |
| `longitude` | PLAC.MAP.LONG | WGS84 longitude |

## Examples

### Simple Location

```yaml
# places/place-paris.glx
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
# places/place-leeds-district.glx
places:
  place-leeds-registration:
    name: "Leeds Registration District"
    type: registration_district
    parent: place-yorkshire
    alternative_names:
      - name: "Leeds"
        type: "informal"
      - name: "West Riding of Yorkshire"
        type: "historical"
    latitude: 53.8008
    longitude: -1.5491
    jurisdiction: "england.yorkshire.leeds"
    place_format: "City, County, Country"
    notes: "Historical registration district for civil registration purposes"
```

### Place with Temporal Properties

```yaml
# places/place-new-york.glx
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

## Validation Rules

- Place hierarchy must be acyclic (no circular parent references)
- Coordinates, if present, must be valid WGS84 values
- Parent place must exist before referencing it
- Type should follow standardized taxonomy

## See Also

- [Event Entity](event.md)
- [Vocabularies](vocabularies.md#place-types-vocabulary)
- [Data Types](../6-data-types.md)

