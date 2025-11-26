---
title: Citation Entity
description: Specific references to evidence with source linkage
layout: doc
---

# Citation Entity

[← Back to Entity Types](README.md)

## Overview

A Citation entity represents a specific reference to evidence that supports genealogical conclusions. Citations link to Source entities and provide detailed information about where evidence was found, including page numbers and data dates.

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
| `page` | string | Page number or locator within source |
| `data_date` | string | Date the data was recorded (for documentary sources) |
| `text_from_source` | string | Transcription or excerpt from the source |
| `locator` | string | Location within source (e.g., 'Page 45, Entry 123', 'Film 1234567, Image 87') |
| `repository` | string | Reference to Repository entity |
| `media` | array | References to Media entities |
| `notes` | string | Free-form notes about the citation |
| `tags` | array | User-defined tags for organization |

## Usage Patterns

### Simple Citation to Book

```yaml
# citations/citation-book.glx
citations:
  citation-marriage-record:
    source: source-parish-register
    page: "125"
    text_from_source: "John Smith married to Mary Jones, 15 May 1850"
```

Note: The `id` is the map key (`citation-marriage-record`), not a separate field.

### Citation with Online Source

```yaml
# citations/citation-online.glx
citations:
  citation-census-online:
    source: source-ancestry-census
    data_date: "1851"
    page: "Schedule 7, piece 1123"
    locator: "https://www.ancestry.com/..., Image 87342534"
    text_from_source: |
      Name: John Smith
      Age: 35
      Occupation: Blacksmith
      Place of Birth: Leeds, Yorkshire, England
```

### Citation to Archive Document

```yaml
# citations/citation-will.glx
citations:
  citation-will-john:
    source: source-probate-wills
    repository: repository-probate
    page: "23"
    locator: "Item 1876/X/150, Film 100234"
    data_date: "1876"
    text_from_source: |
      I, John Smith, being of sound mind, do hereby
      bequeath all my goods and chattels...
```

### Citation with Media References

```yaml
# citations/citation-photo.glx
citations:
  citation-photo:
    source: source-photo-collection
    locator: "Album 1, page 5"
    media:
      - media-john-photo
    notes: "Photo provides visual evidence of person's appearance"
```

## Citation in Assertions

Citations are primarily used within Assertions to provide evidence:

```yaml
# assertions/assertion-birth.glx
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
| `page` | SOUR.PAGE | Page within source |
| `data_date` | SOUR.DATA.DATE | Date data was recorded |
| `text_from_source` | SOUR.TEXT | Transcribed text |
| `locator` | SOUR.OBJE.FILE, SOUR.REPO.CALN | Combined from multiple GEDCOM elements |

## Validation Rules

- Source ID must reference an existing Source entity
- Page information should be concise and meaningful
- Text transcriptions should accurately represent source material
- Repository, if specified, must exist

## Evidence Hierarchy

Citations are part of the GENEALOGIX evidence hierarchy. See [Core Concepts - Evidence Hierarchy](../2-core-concepts#evidence-hierarchy) for the complete evidence chain from Repository → Source → Citation → Assertion.

## See Also

- [Source Entity](source.md) - Bibliographic resource
- [Assertion Entity](assertion.md) - Evidence conclusions
- [Repository Entity](repository.md) - Where sources are held
