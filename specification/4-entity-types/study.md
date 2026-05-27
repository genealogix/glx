---
title: Study Entity
description: Formal scope of a research project — one place studies, one name studies, family reconstructions, and other focused inquiries
layout: doc
---

# Study Entity

[← Back to Entity Types](README)

## Overview

A Study entity declares the formal scope of a research project within a GENEALOGIX archive — for example, a One Place Study (OPS) targeting all records from a specific place within a date range, a One Name Study tracing every bearer of a surname, or a focused brick-wall investigation. Studies make scope machine-readable so tooling can report on coverage, progress, and which sources or places remain to be searched.

A Study does not by itself record evidence; it collects the boundaries of a project so that other entities (Sources, Citations, Assertions, Events) within the archive can be associated with that project's scope.

Studies are GLX-native. There is no GEDCOM equivalent.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in studies/ directory)
studies:
  study-pohl-goens-ops:
    title: "Pohl-Göns One Place Study"
    type: one_place_study
    status: active
    date_range: "FROM 1610 TO 1875"
    places:
      - place-pohl-goens
    notes: "Systematic review of all church-register and civil-records bearers in Pohl-Göns, Hesse-Darmstadt."
```

**Key Points:**

- Entity ID is the map key (`study-pohl-goens-ops`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `title` | string | Name of the study |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Study type from the [study types vocabulary](vocabularies#study-types-vocabulary) |
| `status` | string | Current status from the [study statuses vocabulary](vocabularies#study-statuses-vocabulary) |
| `date_range` | string | Temporal scope as a GLX date string, typically a range (`FROM YYYY TO YYYY`) |
| `places` | string[] | References to Place entities in scope |
| `sources` | string[] | References to Source entities in scope |
| `properties` | object | Vocabulary-extensible metadata (e.g., source types in scope, surname variants) |
| `notes` | string \| string[] | Free-form notes — research objectives, methodology, exclusions |

### `type`

Classification of the study. Standard types:

- `one_place_study` — All records and people associated with a specific place
- `one_name_study` — All bearers of a specific surname (and variants)
- `family_reconstruction` — Membership and connections of a single family
- `descendancy_study` — All descendants of a specific ancestor
- `ancestry_study` — Ancestors of a specific individual
- `brick_wall` — Focused investigation of a specific genealogical problem
- `other` — Other study type

Archives can extend this vocabulary by adding entries to `study-types.glx`. See [Study Types Vocabulary](vocabularies#study-types-vocabulary).

### `status`

Current state of the study. Standard values:

- `active` — Currently being researched
- `paused` — Temporarily on hold, expected to resume
- `completed` — Research goals met
- `abandoned` — Will not be continued

See [Study Statuses Vocabulary](vocabularies#study-statuses-vocabulary).

### `date_range`

A GLX date string defining the temporal scope. Most studies use a range:

```yaml
date_range: "FROM 1610 TO 1875"
```

Single dates and qualifiers (`ABT`, `BEF`, `AFT`) are also accepted but ranges are most common for study scope.

### `places`

Array of Place entity IDs that fall within the study's geographic scope. For a One Place Study this is typically a single place (and the [Place hierarchy](place#place-hierarchy) extends it implicitly to subordinate places). For a regional study, list each top-level place.

### `sources`

Array of Source entity IDs that are explicitly within scope. Useful when a study commits to systematically reviewing a fixed set of sources (e.g., a parish register series). For type-based scope (e.g., "all church registers"), use a custom property such as `source_types` instead — see [Custom Scope Metadata](#custom-scope-metadata).

### `properties`

Free-form metadata. There is no `study_properties` vocabulary in this revision, so property names are not validated and unknown keys do not produce warnings. Suggested uses:

- `source_types` — Source-type categories in scope, when individual sources are not enumerated
- `surname_variants` — For one-name studies, the spellings/variants in scope
- `researcher` — Name of the lead researcher
- `started_on` / `completed_on` — Project lifecycle dates separate from the genealogical `date_range`

### `notes`

Free-form text capturing research objectives, methodology, included/excluded record types, success criteria, and decisions made along the way.

## Usage Patterns

### One Place Study

```yaml
studies:
  study-pohl-goens-ops:
    title: "Pohl-Göns One Place Study"
    type: one_place_study
    status: active
    date_range: "FROM 1610 TO 1875"
    places:
      - place-pohl-goens
    notes: |
      Systematic reconstruction of every household in Pohl-Göns,
      Hesse-Darmstadt, between the start of surviving Lutheran
      registers (1610) and the introduction of Prussian civil
      registration (1876). In-scope record types: parish registers,
      land registers (Salbücher), tax rolls, and emigration lists.
```

### One Name Study

```yaml
studies:
  study-chiddick-ons:
    title: "Chiddick One Name Study"
    type: one_name_study
    status: active
    date_range: "FROM 1700 TO 1950"
    properties:
      surname_variants:
        - "Chiddick"
        - "Chiddock"
        - "Chiddik"
        - "Chedwick"
    notes: "Tracing all bearers of Chiddick and its spelling variants in England, Wales, and the Atlantic colonies."
```

### Family Reconstruction

```yaml
studies:
  study-webb-family:
    title: "Webb Family of Campbell County, Virginia"
    type: family_reconstruction
    status: active
    date_range: "FROM 1780 TO 1900"
    places:
      - place-campbell-co-va
    sources:
      - source-campbell-co-deeds
      - source-campbell-co-wills
      - source-1810-census-campbell-va
    notes: "Reconstructing the descendants of John Webb (b. ~1755) and his wife Sarah."
```

### Brick Wall Investigation

```yaml
studies:
  study-elizabeth-smith-origin:
    title: "Origin of Elizabeth Smith (b. ~1820)"
    type: brick_wall
    status: paused
    date_range: "FROM 1815 TO 1845"
    places:
      - place-yorkshire-england
      - place-pennsylvania-usa
    notes: |
      Determining whether Elizabeth Smith (m. 1842 Pittsburgh)
      arrived from Yorkshire ca. 1840 or was native-born to PA.
      Paused pending availability of indexed Yorkshire baptisms.
```

## Custom Scope Metadata

When the standard fields don't cover everything, use `properties` for archive-defined extensions. For example, scope by source type rather than enumeration:

```yaml
studies:
  study-leeds-ops:
    title: "Leeds Parish One Place Study"
    type: one_place_study
    status: active
    date_range: "FROM 1700 TO 1900"
    places: [place-leeds]
    properties:
      source_types:
        - church_register
        - census
        - directory
```

Properties on Studies are not validated against a vocabulary in this revision; archives can store any keys they like.

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Study files are typically stored in a `studies/` directory:

```text
studies/
├── study-pohl-goens-ops.glx
├── study-chiddick-ons.glx
└── study-webb-family.glx
```

## Validation Rules

- `title` must be present and non-empty
- `type`, if specified, must exist in the [study types vocabulary](vocabularies#study-types-vocabulary)
- `status`, if specified, must exist in the [study statuses vocabulary](vocabularies#study-statuses-vocabulary)
- All IDs in `places` must reference existing Place entities
- All IDs in `sources` must reference existing Source entities
- `date_range` should follow GLX date format (e.g., `FROM YYYY TO YYYY`)

## GEDCOM Mapping

Studies are GLX-native and have **no GEDCOM equivalent**. Research-project scope is not part of any GEDCOM specification (5.5.1 or 7.0). On GEDCOM export, Study entities are dropped; on GEDCOM import, no Study entities are created.

## Related Entities

- **Place** — Geographic scope of the study (`places` field)
- **Source** — Sources within the study's scope (`sources` field)
- **Citation, Assertion, Event** — Evidence and conclusions produced inside a study; these are not back-referenced from the Study, but tooling may surface them by intersecting their `places`/`sources`/`date` with a Study's scope

## Schema Reference

See [study.schema.json](../schema/v1/study.schema.json) for the complete JSON Schema definition.

## See Also

- [Place Entity](place) — Geographic scope
- [Source Entity](source) — In-scope sources
- [Vocabularies — Study Types](vocabularies#study-types-vocabulary)
- [Vocabularies — Study Statuses](vocabularies#study-statuses-vocabulary)
