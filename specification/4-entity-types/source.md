---
title: Source Entity
description: Original documents and records - the foundation of evidence-based genealogical research
layout: doc
---

# Source Entity

[← Back to Entity Types](README)

## Overview

A Source entity represents an original document, record, publication, or material that contains genealogical information. Sources are the foundation of evidence-based research and form the middle layer of the evidence chain between repositories (where sources are held) and citations (specific references within sources).

Sources can include:
- Vital records (birth, marriage, death certificates)
- Census records and population registers
- Church registers (baptisms, marriages, burials)
- Military records and service files
- Newspapers and periodicals
- Published family histories and genealogies
- Wills, probate records, and court documents
- Land records and deeds
- Immigration and naturalization records
- Oral histories and interviews
- Online databases and digital collections

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in sources/ directory)
sources:
  source-parish-register:
    title: "St. Paul's Parish Register, 1840-1860"
    type: church_register
    authors:
      - "Church of England"
    repository: repository-leeds-library
    date: "FROM 1840 TO 1860"
```

**Key Points:**
- Entity ID is the map key (`source-parish-register`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `title` | string | Full title of the source |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Source type from vocabulary |
| `authors` | array | Author(s) or creator(s) — personal names or institutional names |
| `date` | string | Date or date range of the source |
| `description` | string | Description of the source |
| `repository` | string | Reference to Repository entity |
| `language` | string | Language of the source |
| `media` | array | References to Media entities |
| `properties` | object | Vocabulary-defined properties (see [Properties](#properties)) |
| `notes` | string | Free-form notes |

## Required Fields (Detailed)

### Entity ID (map key)

- Format: Any alphanumeric string with hyphens, 1-64 characters
- Must be unique within the archive
- Example formats:
  - Descriptive: `birth-register`, `1851-census`
  - Random hex: `a1b2c3d4`
  - Prefixed: `source-a1b2c3d4`
  - Sequential: `001`, `002`

### `title`

- Type: String
- Required: Yes
- Description: Full title of the source

Example:
```yaml
title: "1851 Census of England and Wales"
```

## Optional Fields

### `type`

- Type: String
- Required: No (but recommended)
- Description: Classification of the source type

Common source types:
- `vital_record` - Birth, marriage, death certificates
- `census` - Census records and enumerations
- `church_register` - Parish registers, baptisms, marriages, burials
- `military` - Military service records, pension files
- `newspaper` - Newspapers, periodicals, gazettes
- `probate` - Wills, probate records, estate files
- `land` - Deeds, land grants, property records
- `court` - Court records, legal proceedings
- `immigration` - Passenger lists, naturalization records
- `directory` - City directories, telephone books
- `book` - Published genealogies, family histories
- `database` - Online databases, compiled records
- `oral_history` - Interviews, recorded memories
- `correspondence` - Letters, emails, personal papers
- `photograph` - Photo collections
- `population_register` - Civil population registers, household registration records
- `tax_record` - Tax rolls, assessments, tithes
- `notarial_record` - Notarial acts, contracts, legal instruments
- `other` - Other source types

Example:
```yaml
type: church_register
```

### `authors`

- Type: Array of Strings
- Required: No
- Description: Author(s) or creator(s) of the source. Maps to GEDCOM `SOUR.AUTH`.

Use for both personal authors and institutional creators:

Example:
```yaml
# Personal author
authors:
  - "Elizabeth Shown Mills"

# Institutional creator
authors:
  - "General Register Office"

# Multiple authors
authors:
  - "Elizabeth Shown Mills"
  - "John Doe"
```

### `date`

- Type: String
- Required: No
- Description: Publication date or date range covered by the source

Formats:
- Single date: `"1851"`
- Date range: `"FROM 1840 TO 1860"`
- Publication date: `"2015-06-20"`

Example:
```yaml
date: "FROM 1850 TO 1855"
```

### `repository`

- Type: String
- Required: No (but recommended)
- Description: Repository ID where this source is held

Example:
```yaml
repository: repository-national-archives
```

### `description`

- Type: String
- Required: No
- Description: Detailed description of the source content and scope

Example:
```yaml
description: |
  Parish registers for St. Paul's Church, Leeds, covering baptisms,
  marriages, and burials from 1840 to 1860. Includes entries for
  families in the Wellington Street area.
```

### `language`

- Type: String
- Required: No
- Description: Language(s) of the source

Example:
```yaml
language: "English"
```

### `media`

- Type: Array of Strings
- Required: No
- Description: References to media entities (scans, photos) of this source

Example:
```yaml
media:
  - media-register-scan-page-1
  - media-register-scan-page-2
```

### `properties`

- Type: Object
- Required: No
- Description: Vocabulary-defined properties for additional source metadata

The following are standard properties from the default vocabulary; archives can define additional properties by extending the vocabulary:

| Property | Type | Description |
|----------|------|-------------|
| `abbreviation` | string | Short reference name (from GEDCOM ABBR) |
| `call_number` | string | Repository catalog number (from GEDCOM CALN) |
| `events_recorded` | string[] | Types of events documented (from GEDCOM EVEN) |
| `agency` | string | Responsible agency (from GEDCOM AGNC) |
| `coverage` | string | Geographic/temporal scope |
| `external_ids` | string[] | External system identifiers |
| `publication_info` | string | Publication details: publisher, place, edition (from GEDCOM PUBL) |
| `url` | string | Web address where the source can be accessed online |

Example:
```yaml
sources:
  source-parish-register:
    title: "St. Paul's Parish Register"
    type: church_register
    repository: repository-leeds-archives
    properties:
      abbreviation: "StP-Reg"
      call_number: "PR/LEE/123"
      events_recorded:
        - "Baptisms"
        - "Marriages"
        - "Burials"
      coverage: "Leeds, Yorkshire, 1840-1860"
```

**See [Vocabularies - Source Properties](vocabularies#source-properties-vocabulary) for the full vocabulary definition.**

## Usage Patterns

### Vital Record

```yaml
sources:
  source-birth-cert-john-smith:
    title: "Birth Certificate - John Smith, 1850"
    type: vital_record
    authors:
      - "General Register Office"
    date: "1850-01-15"
    repository: repository-gro
    description: "Original birth certificate for John Smith, born Leeds"
    language: "English"
    media:
      - media-birth-cert-scan
```

### Census Record

```yaml
sources:
  source-census-1851-yorkshire:
    title: "1851 Census of England and Wales"
    type: census
    authors:
      - "UK Census Office"
    date: "1851-03-30"
    repository: repository-national-archives
    description: |
      Census enumeration for Leeds, Yorkshire, England.
      Enumeration District 5, covering Wellington Street area.
    language: "English"
```

### Church Register

```yaml
sources:
  source-st-pauls-register:
    title: "St. Paul's Cathedral Parish Register, 1840-1860"
    type: church_register
    authors:
      - "Church of England"
    date: "FROM 1840 TO 1860"
    repository: repository-leeds-archives
    description: |
      Parish registers for St. Paul's Cathedral, Leeds.
      Includes baptisms, marriages, and burials.

      Coverage:
      - Baptisms: 1840-1860
      - Marriages: 1840-1860
      - Burials: 1845-1860
    language: "English"
    media:
      - media-register-volume-1
    notes: "Well preserved, some water damage to pages 45-50"
```

### Published Book

```yaml
sources:
  source-smith-family-book:
    title: "The Smith Family of Yorkshire: A Genealogy"
    type: book
    authors:
      - "Elizabeth Brown"
    date: "1985"
    repository: repository-family-history-library
    description: |
      Comprehensive genealogy of the Smith family of Leeds
      and surrounding areas, 1750-1950. Includes source
      citations and family group sheets.
    language: "English"
    properties:
      publication_info: "Yorkshire Genealogical Society, Leeds, Yorkshire. 1st Edition, 324 pages."
```

### Online Database

```yaml
sources:
  source-ancestry-uk-census:
    title: "UK Census Collection, 1841-1911"
    type: database
    authors:
      - "Ancestry.com"
    date: "FROM 1841 TO 1911"
    repository: repository-ancestry
    description: |
      Digitized and indexed UK census records from 1841-1911.
      Images and transcriptions available online.

      Subscription required for access.
    language: "English"
    notes: "Digital images of original records"
```

### Newspaper

```yaml
sources:
  source-leeds-mercury:
    title: "Leeds Mercury"
    type: newspaper
    authors:
      - "Leeds Mercury Publishing Company"
    date: "1890-06-15"
    repository: repository-british-library
    description: "Daily newspaper published in Leeds, Yorkshire"
    language: "English"
```

### Oral History

```yaml
sources:
  source-mary-smith-interview:
    title: "Interview with Mary Smith"
    type: oral_history
    authors:
      - "Jane Researcher (interviewer)"
      - "Mary Smith (interviewee)"
    date: "2020-03-15"
    description: |
      Oral history interview with Mary Smith discussing her
      childhood in Leeds during the 1940s and 1950s. Family
      stories, local history, and genealogical information.
      
      Duration: 60 minutes
      Location: Leeds, Yorkshire
    media:
      - media-interview-audio
      - media-interview-transcript
    language: "English"
    notes: "Recorded with permission, transcript available"
```

## Source Types

Source types are defined in the archive's `vocabularies/source-types.glx` file.

**See [Vocabularies - Source Types](vocabularies#source-types-vocabulary) for:**
- Complete list of standard source types
- How to add custom source types
- Vocabulary file structure and examples

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Source files are typically organized by type or repository:

```
sources/
├── vital-records/
│   ├── source-birth-john.glx
│   ├── source-marriage-john-mary.glx
│   └── source-death-john.glx
├── census/
│   ├── source-1841-census.glx
│   ├── source-1851-census.glx
│   └── source-1861-census.glx
├── church-registers/
│   ├── source-st-pauls-register.glx
│   └── source-holy-trinity-register.glx
├── newspapers/
│   └── source-leeds-mercury.glx
├── books/
│   └── source-smith-family-history.glx
└── online/
    └── source-ancestry-census.glx
```

## Relationship to Other Entities

```
Source
    ├── held in → Repository (via repository field)
    ├── referenced by → Citations (citations point to sources)
    └── documented by → Media (via media array)

Repository
    └── holds → Sources (sources reference repository)

Citation
    └── references → Source (via source field)

Media
    └── documents → Source (media can be scans/photos of sources)
```

## Validation Rules

- `title` must be present and non-empty
- If `repository` is specified, it must reference an existing Repository entity
- If `media` array is present, all IDs must reference existing Media entities
- `date` should follow standard date formats (YYYY, YYYY-MM-DD, or `FROM YYYY TO YYYY` for ranges)
- Type must be from the [source types vocabulary](vocabularies#source-types-vocabulary)

## GEDCOM Mapping

Source entities map to GEDCOM source records:

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| Entity ID | `@SOUR@` | Source record ID |
| `title` | `SOUR.TITL` | Source title |
| `authors[0]` | `SOUR.AUTH` | Author/creator (first only in GEDCOM) |
| `date` | `SOUR.DATE` | Publication date |
| `repository` | `SOUR.REPO` | Repository reference |
| `description` | `SOUR.TEXT` or `SOUR.NOTE` | Source text/notes |
| `properties.abbreviation` | `SOUR.ABBR` | Short title |
| `properties.publication_info` | `SOUR.PUBL` | Publication info |
| `properties.call_number` | `SOUR.REPO.CALN` | Call number at repository |
| `properties.events_recorded` | `SOUR.DATA.EVEN` | Events in source |
| `properties.agency` | `SOUR.DATA.AGNC` | Responsible agency |
| `properties.external_ids` | `SOUR.EXID` | External identifiers (GEDCOM 7.0) |

GEDCOM Example:
```
0 @S1@ SOUR
1 TITL St. Paul's Parish Register
1 AUTH Church of England
1 DATE 1840/1860
1 REPO @R1@
1 NOTE Parish registers for baptisms, marriages, and burials
```

GENEALOGIX Equivalent:
```yaml
sources:
  source-st-pauls:
    title: "St. Paul's Parish Register"
    authors:
      - "Church of England"
    date: "FROM 1840 TO 1860"
    repository: repository-leeds-archives
    description: "Parish registers for baptisms, marriages, and burials"
```

## Best Practices

### Complete Source Documentation

Include as much bibliographic information as possible:
- Full title
- Author(s)
- Date or date range
- Repository location
- Repository catalog number
- Physical description (if relevant)

### Consistent Citation Format

Use consistent citation styles within your archive:
- Follow established citation standards (Evidence Explained, Chicago Manual of Style, etc.)
- Document your chosen citation style in archive documentation

### Source Characteristics

Documenting source characteristics in notes helps researchers evaluate evidence:
- Primary vs. secondary nature
- Original vs. derivative
- Completeness and condition
- Known limitations or biases

### Digital Preservation

Link media entities to sources for digital preservation:
- Scan or photograph original sources
- Store digital copies with source metadata
- Include hash values for integrity verification

## Schema Reference

See [source.schema.json](../schema/v1/source.schema.json) for the complete JSON Schema definition.

## See Also

- [Core Concepts - Evidence Chain](../2-core-concepts#evidence-chain) - Understanding the evidence chain
- [Repository Entity](repository) - Where sources are held
- [Citation Entity](citation) - Specific references within sources
- [Media Entity](media) - Digital preservation of sources
- [Assertion Entity](assertion) - Conclusions drawn from sources
