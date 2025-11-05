# Citation Entity

[← Back to Entity Types](README.md)

## Overview

A Citation entity represents a specific reference to evidence that supports genealogical conclusions. Citations link to Source entities and provide detailed information about where evidence was found, including page numbers, data dates, and quality assessments.

## Core Concepts

### Citation vs. Source

- **Source**: A bibliographic resource (book, document, database, website)
- **Citation**: A specific reference to information within a source

One source can have many citations referencing different pages or sections.

### Evidence Quality

GENEALOGIX uses a 0-3 quality scale that maintains 1:1 compatibility with GEDCOM QUAY for interoperability.

**Quality Scale (GEDCOM QUAY Compatible):**
- **3**: Direct and primary evidence used, or by dominance of evidence
  - Examples: Original birth certificate, contemporary baptism record, firsthand diary entry
- **2**: Secondary evidence, data officially recorded sometime after event
  - Examples: Census record, compiled index, published vital records
- **1**: Questionable reliability of evidence
  - Examples: Undocumented oral history, conflicting sources, estimated data
- **0**: Unreliable evidence or estimated data
  - Examples: Unverified family tradition, unsourced online trees

**See [Vocabularies - Quality Ratings](vocabularies.md#quality-ratings-vocabulary) for:**
- Customizing quality rating definitions for your archive
- Alternative approaches using confidence levels
- Vocabulary file structure and validation

**GEDCOM Interoperability:**
This scale maps directly to GEDCOM 5.5.1 QUAY values (0→0, 1→1, 2→2, 3→3), ensuring lossless conversion between formats.

**Advanced Evidence Evaluation:**
For more sophisticated analysis beyond the 0-3 scale, use:
- `assertion.confidence` field (high/medium/low/disputed) for conclusion certainty
- `citation.research_notes` for detailed source analysis (primary vs. secondary, direct vs. indirect)
- Multiple citations per assertion to show corroboration

## Properties

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier (format: `citation-{id}`) |
| `version` | string | Schema version (e.g., "1.0") |
| `source_id` | string | Reference to Source entity |

### Optional Properties

| Property | Type | Description |
|----------|------|-------------|
| `page` | string | Page number or locator within source |
| `data_date` | string | Date the data was recorded (for documentary sources) |
| `text_from_source` | string | Transcription or excerpt from the source |
| `quality` | integer | Evidence quality (0-3, QUAY value) |
| `locator` | object | Structured locator information |
| `locator.film_number` | string | FamilySearch film number |
| `locator.item_number` | string | Item number or accession number |
| `locator.image_number` | string | Image or page identifier |
| `locator.url` | string | URL to online source |
| `repository_id` | string | Reference to Repository entity |
| `created_at` | datetime | Creation timestamp |
| `created_by` | string | User who created this record |
| `modified_at` | datetime | Last modification timestamp |
| `modified_by` | string | User who last modified this record |
| `notes` | string | Free-form notes about the citation |

## Usage Patterns

### Simple Citation to Book

```yaml
# citations/citation-book.glx
citations:
  citation-marriage-record:
    version: "1.0"
    source_id: source-parish-register
    page: "125"
    quality: 3
    text_from_source: "John Smith married to Mary Jones, 15 May 1850"
    created_at: "2025-01-15T10:30:00Z"
    created_by: "researcher@example.com"
```

### Citation with Online Source

```yaml
# citations/citation-online.glx
citations:
  citation-census-online:
    version: "1.0"
    source_id: source-ancestry-census
    data_date: "1851"
    page: "Schedule 7, piece 1123"
    quality: 2
    locator:
      url: "https://www.ancestry.com/..."
      image_number: "87342534"
    text_from_source: |
      Name: John Smith
      Age: 35
      Occupation: Blacksmith
      Place of Birth: Leeds, Yorkshire, England
    created_at: "2025-01-15T10:30:00Z"
    created_by: "researcher@example.com"
```

### Citation to Archive Document

```yaml
# citations/citation-will.glx
citations:
  citation-will-john:
    version: "1.0"
    source_id: source-probate-wills
    repository_id: repository-probate
    page: "23"
    quality: 3
    locator:
      item_number: "1876/X/150"
      film_number: "100234"
    data_date: "1876"
    text_from_source: |
      I, John Smith, being of sound mind, do hereby
      bequeath all my goods and chattels...
    created_at: "2025-01-15T10:30:00Z"
    created_by: "researcher@example.com"
```

## Citation in Assertions

Citations are primarily used within Assertions to provide evidence:

```yaml
# assertions/assertion-birth.glx
assertions:
  assertion-birth-john:
    version: "1.0"
    subject: person-john-smith
    claim: birth_date
    value: "1850-01-15"
    citations:
      - citation-parish-birth
      - citation-census-birth
    confidence: high
    research_notes: "Primary source from parish register"
```

## Locator Object

The locator object stores structured information about how to find the evidence:

```yaml
locator:
  # For online resources
  url: "https://www.familysearch.org/..."
  image_number: "4253453"
  
  # For archive materials
  film_number: "1234567"
  item_number: "P/152/1"
  
  # For databases
  record_id: "REC12345678"
  database_collection: "UK Census 1851"
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

| GLX Property | GEDCOM Element | Notes |
|--------------|----------------|-------|
| `id` | (synthetic) | Not in GEDCOM |
| `source_id` | SOUR | Source reference |
| `page` | SOUR.PAGE | Page within source |
| `data_date` | SOUR.DATA.DATE | Date data was recorded |
| `text_from_source` | SOUR.TEXT | Transcribed text |
| `quality` | SOUR.QUAY | Evidence quality (0-3) |
| `locator.url` | SOUR.OBJE.FILE | File/URL path |
| `locator.film_number` | SOUR.REPO.CALN.VALUE | Media number |

## Validation Rules

- Source ID must reference an existing Source entity
- Quality, if present, must be 0-3
- Page information should be concise and meaningful
- Locator URLs must be properly formed
- Text transcriptions should accurately represent source material
- Repository, if specified, must exist

## Evidence Hierarchy

Citations are the lowest level in the evidence hierarchy:

```
Archive/Collection (Source)
  └─ Specific Record/Document (Citation)
      └─ Specific Field/Statement (Assertion)
```

## Quality Assessment

When entering citations, consider:

- **Original vs. Derivative**: Is this from the original record or a copy?
- **Reliability of Source**: Who created/maintains the source?
- **Completeness**: Does the source provide all relevant information?
- **Corroboration**: Is the information supported by other sources?

## See Also

- [Source Entity](source.md) - Bibliographic resource
- [Assertion Entity](assertion.md) - Evidence conclusions
- [Repository Entity](repository.md) - Where sources are held




