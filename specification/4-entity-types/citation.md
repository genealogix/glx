---
title: Citation Entity
description: Specific references to evidence with source linkage
layout: doc
---

# Citation Entity

[← Back to Entity Types](README.md)

## Overview

A Citation entity represents a specific reference to evidence that supports genealogical conclusions. Citations link to Source entities and provide detailed information about where evidence was found within the source.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in citations/ directory)
citations:
  citation-birth-record:
    source: source-parish-register
    properties:
      locator: "Page 45"
      text_from_source: "John Smith, born 15 January 1850"
```

**Key Points:**
- Entity ID is the map key (`citation-birth-record`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Core Concepts

### Citation vs. Source

- **Source**: A bibliographic resource (book, document, database, website)
- **Citation**: A specific reference to information within a source

One source can have many citations referencing different pages or sections.

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `source` | string | Reference to Source entity |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | Reference to Repository entity |
| `media` | array | References to Media entities |
| `properties` | object | Vocabulary-defined properties (see [Citation Properties](#properties)) |
| `notes` | string | Free-form notes about the citation |
| `tags` | array | User-defined tags for organization |

### Properties

Citation properties are defined in the `citation_properties` vocabulary. Standard properties include:

| Property | Type | Description |
|----------|------|-------------|
| `locator` | string | Location within source (e.g., 'Page 45', 'Film 1234567, Image 87', 'Entry 123') |
| `text_from_source` | string | Transcription or excerpt from the source |

**See Also:** [Citation Properties Vocabulary](../5-standard-vocabularies/README.md#citation-properties)

## Usage Patterns

### Simple Citation to Book

```yaml
citations:
  citation-marriage-record:
    source: source-parish-register
    properties:
      locator: "Page 125"
      text_from_source: "John Smith married to Mary Jones, 15 May 1850"
```

Note: The `id` is the map key (`citation-marriage-record`), not a separate field.

### Citation with Online Source

```yaml
citations:
  citation-census-online:
    source: source-ancestry-census
    properties:
      locator: "Schedule 7, piece 1123, Image 87342534"
      text_from_source: |
        Name: John Smith
        Age: 35
        Occupation: Blacksmith
        Place of Birth: Leeds, Yorkshire, England
```

### Citation to Archive Document

```yaml
citations:
  citation-will-john:
    source: source-probate-wills
    repository: repository-probate
    properties:
      locator: "Page 23, Item 1876/X/150, Film 100234"
      text_from_source: |
        I, John Smith, being of sound mind, do hereby
        bequeath all my goods and chattels...
```

### Citation with Media References

```yaml
citations:
  citation-photo:
    source: source-photo-collection
    media:
      - media-john-photo
    properties:
      locator: "Album 1, page 5"
    notes: "Photo provides visual evidence of person's appearance"
```

## Citation in Assertions

Citations are primarily used within Assertions to provide evidence:

```yaml
assertions:
  assertion-birth-john:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations:
      - citation-parish-birth
      - citation-census-birth
    confidence: high
    notes: "Primary source from parish register"
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Citation files are typically embedded in assertion documents or organized by source:

```
sources/
├── books/
│   └── source-book001.glx
│       └── citations/
│           ├── citation-01.glx
│           └── citation-02.glx
└── online/
    └── source-ancestry.glx
        └── citations/
            ├── citation-01.glx
            └── citation-02.glx
```

Or more commonly, citations are referenced by ID from assertions.

## GEDCOM Mapping

| GLX Field | GEDCOM Element | Notes |
|-----------|----------------|-------|
| Entity ID (map key) | (synthetic) | Not in GEDCOM |
| `source` | SOUR | Source reference |
| `properties.locator` | SOUR.PAGE | Location within source (GEDCOM PAGE is free-form text, not just page numbers) |
| `properties.text_from_source` | SOUR.TEXT, SOUR.DATA.TEXT | Transcribed text |

## Validation Rules

- Source ID must reference an existing Source entity
- Properties should follow the [citation_properties vocabulary](../5-standard-vocabularies/README.md#citation-properties)
- Text transcriptions should accurately represent source material
- Repository, if specified, must exist

## Evidence Hierarchy

Citations are part of the GENEALOGIX evidence hierarchy. See [Core Concepts - Evidence Hierarchy](../2-core-concepts#evidence-hierarchy) for the complete evidence chain from Repository → Source → Citation → Assertion.

## See Also

- [Source Entity](source.md) - Bibliographic resource
- [Assertion Entity](assertion.md) - Evidence conclusions
- [Repository Entity](repository.md) - Where sources are held
