---
title: Migration from GEDCOM
description: Guide for converting GEDCOM files to GENEALOGIX format.
layout: doc
---

# Migration from GEDCOM Guide

Guide for converting GEDCOM files to GENEALOGIX format using the automated import tool.

## Key Differences

Understanding how GENEALOGIX differs from GEDCOM helps you get the most out of your migration.

| Aspect | GEDCOM | GENEALOGIX |
|--------|--------|------------|
| **Format** | Custom tag-based text format | YAML |
| **Evidence model** | Source citations attached to facts | Full evidence chains (Source → Citation → Assertion) |
| **Version control** | Monolithic file | Git-native, multi-file archives |
| **IDs** | Sequential references (`@I1@`, `@F1@`) | Typed, descriptive IDs (`person-a3f8d2c1`) |
| **Places** | Flat comma-separated strings | Hierarchical entities with coordinates |
| **Media** | File paths or embedded BLOBs | Media entities with MIME types and metadata |
| **Notes** | Inline or shared text blocks | First-class entity notes |
| **Extensibility** | Underscore-prefixed custom tags | Custom vocabularies |

## Before You Start

### Prerequisites

- The `glx` CLI tool installed ([installation instructions](https://github.com/genealogix/glx/blob/main/glx/README.md#installation))
- Your GEDCOM file (`.ged`)
- Git installed (recommended for version control)

### Supported GEDCOM Versions

| Version | Support Level |
|---------|--------------|
| GEDCOM 5.5.1 | Full support |
| GEDCOM 7.0 | Full support |

The importer auto-detects the GEDCOM version from the `HEAD.GEDC.VERS` tag. Unknown versions are treated as GEDCOM 5.5.1.

::: tip Check your version
Open your `.ged` file in a text editor and look near the top for a line like `2 VERS 5.5.1` or `2 VERS 7.0`.
:::

## Automated Import

### Basic Usage

```bash
# Import to multi-file archive (default)
glx import family.ged -o family-archive/

# Import to single-file archive
glx import family.ged -o family.glx --format single
```

### CLI Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | (required) | Output file or directory |
| `--format` | `-f` | `multi` | Output format: `multi` or `single` |
| `--no-validate` | | `false` | Skip validation before saving |
| `--verbose` | `-v` | `false` | Show detailed import progress |
| `--show-first-errors` | | `10` | Number of validation errors to show (0 for all) |

### Single-File vs Multi-File

**Single-file** (`--format single`): All entities in one `.glx` file. Best for small archives or sharing.

**Multi-file** (`--format multi`): One entity per file in a directory structure. Best for Git tracking, collaboration, and large archives.

```bash
# Single-file output
glx import family.ged -o family.glx --format single

# Multi-file output creates a directory structure:
# family-archive/
# ├── persons/
# ├── events/
# ├── relationships/
# ├── places/
# ├── sources/
# ├── citations/
# ├── repositories/
# ├── media/
# ├── assertions/
# └── vocabularies/
glx import family.ged -o family-archive/
```

### Import Statistics

After a successful import, the CLI prints a summary of what was created:

```
✓ Successfully imported to family.glx

Import statistics:
  Persons:       31
  Events:        77
  Relationships: 49
  Places:        5
  Sources:       3
  Citations:     12
  Repositories:  1
  Media:         0
  Assertions:    150
```

### What Gets Imported

The importer processes records in dependency order across multiple passes, handling all standard GEDCOM record types.

**Individuals**: Names (with parsed components), gender, 20+ event types, 8+ property types, external IDs, notes, media references

**Events**: Each imported event receives an auto-generated `title` field for human readability. The format varies by event type:
- Individual events: "Birth of Robert Webb (1815)"
- Couple events: "Marriage of John Smith and Jane Doe (1850)"
- Date-only: "Census (1860)"
- Name-only: "Death of Jane Miller"

Unknown event types fall back to Title Case of the snake_case type (e.g., `military_service` → "Military Service"). Titles are generated from participant names and the event date — they are not extracted from the GEDCOM source.

**Families**: Spouse relationships, parent-child relationships (with pedigree types), 9 family event types, media

**Sources and evidence**: Source records with metadata, inline citations, evidence chain construction (Source → Citation → Assertion)

**Places**: Hierarchical place entities built from comma-separated GEDCOM places, with coordinates from MAP/LATI/LONG tags

**Repositories**: Repository records with address, contact info, and type detection

**Media**: Media entities with MIME type resolution, file path rewriting, and BLOB decoding

**Notes**: Both shared notes (GEDCOM 5.5.1 `NOTE` records, GEDCOM 7.0 `SNOTE` records) and inline notes

### Media File Handling

The importer handles three types of media references:

- **Relative file paths**: Copied to `media/files/` in your archive, with paths rewritten automatically. Duplicate filenames are deduplicated (e.g., `photo.jpg` → `photo-2.jpg`).
- **URLs and absolute paths**: Preserved as-is in the media entity's URI field.
- **BLOB data** (GEDCOM 5.5.1): Binary data is decoded and written to files in `media/files/`.

::: warning Media source directory
Relative file paths in the GEDCOM are resolved from the directory containing the `.ged` file. Make sure media files are accessible at those paths before importing.
:::

## Field Mapping

### Individual Records (INDI)

#### Events

| GEDCOM Tag | GLX Event Type | Notes |
|------------|---------------|-------|
| `BIRT` | `birth` | Date/place also propagated to person properties |
| `DEAT` | `death` | Date/place also propagated to person properties |
| `CHR` | `christening` | |
| `BURI` | `burial` | |
| `CREM` | `cremation` | |
| `ADOP` | `adoption` | |
| `BAPM` | `baptism` | |
| `BARM` | `bar_mitzvah` | |
| `BATM` | `bat_mitzvah` | |
| `BLES` | `blessing` | |
| `CHRA` | `adult_christening` | |
| `CONF` | `confirmation` | |
| `FCOM` | `first_communion` | |
| `ORDN` | `ordination` | |
| `NATU` | `naturalization` | |
| `EMIG` | `emigration` | |
| `IMMI` | `immigration` | |
| `PROB` | `probate` | |
| `WILL` | `will` | |
| `GRAD` | `graduation` | |
| `RETI` | `retirement` | |

#### Properties

| GEDCOM Tag | GLX Property | Notes |
|------------|-------------|-------|
| `NAME` | `name` | Parsed into structured fields (see [Name Conversion](#name-conversion)) |
| `SEX` | `gender` | M→male, F→female, U→unknown, X→other |
| `OCCU` | `occupation` | Temporal property |
| `RELI` | `religion` | |
| `EDUC` | `education` | |
| `NATI` | `nationality` | |
| `CAST` | `caste` | |
| `SSN` | `ssn` | |
| `TITL` | `title` | Handles CONT/CONC for long values |
| `RESI` | `residence` | Temporal property with date and place |
| `FACT` | (varies) | Mapped to properties or generic events based on content |
| `EXID` | `external_ids` | GEDCOM 7.0 external identifiers |
| `NOTE` | `notes` | Inline or shared note text |
| `OBJE` | `media` | Media references or embedded media |

#### Special Handling

| GEDCOM Tag | GLX Mapping | Notes |
|------------|-------------|-------|
| `CENS` | Temporal properties + synthetic source | Census records create residence properties and synthetic sources/citations — not events |
| `FAMC` | Parent-child relationship | Deferred until all individuals are processed; uses PEDI for relationship type |
| `FAMS` | Spouse link | Used during family processing |
| `NO` | Negative assertion | GEDCOM 7.0 only; creates assertion with `no_` prefix |

### Family Records (FAM)

#### Participants

| GEDCOM Tag | GLX Mapping | Notes |
|------------|-------------|-------|
| `HUSB` | Participant (role: spouse) | |
| `WIFE` | Participant (role: spouse) | |
| `CHIL` | (via INDI FAMC) | Parent-child relationships created from individual records |

#### Events

| GEDCOM Tag | GLX Event Type | Notes |
|------------|---------------|-------|
| `MARR` | `marriage` | Also sets `start_event` on the relationship |
| `DIV` | `divorce` | Also sets `end_event` on the relationship |
| `ENGA` | `engagement` | |
| `MARB` | `marriage_banns` | |
| `MARC` | `marriage_contract` | |
| `MARL` | `marriage_license` | |
| `MARS` | `marriage_settlement` | |
| `ANUL` | `annulment` | |
| `DIVF` | `divorce_filed` | |
| `EVEN` | `event` | Generic family event |

### Source Records (SOUR)

| GEDCOM Tag | GLX Field | Notes |
|------------|-----------|-------|
| `TITL` | `source.title` | |
| `AUTH` | `source.authors` | |
| `PUBL` | `source.properties.publication_info` | |
| `ABBR` | `source.properties.abbreviation` | |
| `REPO` | `source.repository` | With `CALN` stored as `call_number` |
| `TEXT` | `source.description` | |
| `NOTE` | `source.notes` | |
| `DATA.EVEN` | `source.properties.events_recorded` | Event types this source records |
| `DATA.AGNC` | `source.properties.agency` | Responsible agency |
| `DATA.DATE` | `source.date` | |
| `TYPE` | `source.type` | Mapped via source type vocabulary |
| `OBJE` | `source.media` | |
| `EXID` | `source.properties.external_ids` | GEDCOM 7.0 |

::: info Source type inference
If no `TYPE` tag is present, the importer infers the source type from keywords in the title: "census" → `census`, "birth certificate" → `vital_record`, "parish register" → `church_register`, "newspaper" → `newspaper`, and so on.
:::

### Citation Subrecords (SOUR within events)

| GEDCOM Tag | GLX Field | Notes |
|------------|-----------|-------|
| `PAGE` | `citation.properties.locator` | Location within the source |
| `DATA.DATE` | `citation.properties.source_date` | When the source recorded the information |
| `DATA.TEXT` | `citation.properties.text_from_source` | Transcription from source |
| `TEXT` | `citation.properties.text_from_source` | GEDCOM 5.5.1 direct text |
| `QUAY` | `citation.notes` | Quality assessment preserved as note |
| `NOTE` | `citation.notes` | |
| `OBJE` | `citation.media` | |

### Repository Records (REPO)

| GEDCOM Tag | GLX Field | Notes |
|------------|-----------|-------|
| `NAME` | `repository.name` | |
| `ADDR` | `repository.address`, `.city`, `.state_province`, `.postal_code`, `.country` | Full address with subfields |
| `PHON` | `repository.properties.phones` | |
| `EMAIL` | `repository.properties.emails` | |
| `WWW` | `repository.website` | |
| `NOTE` | `repository.notes` | |
| `TYPE` | `repository.type` | GEDCOM 7.0 |
| `EXID` | `repository.properties.external_ids` | GEDCOM 7.0 |

::: info Repository deduplication
Repositories are automatically deduplicated by name, city, and country. If two GEDCOM REPO records match on these fields, they are merged into a single repository entity.
:::

### Media Records (OBJE)

| GEDCOM Tag | GLX Field | Notes |
|------------|-----------|-------|
| `FILE` | `media.uri` | Relative paths rewritten to `media/files/` |
| `FILE.FORM` | (MIME inference) | GEDCOM 5.5.1 format |
| `FILE.MIME` | `media.mime_type` | GEDCOM 7.0 explicit MIME |
| `FILE.TITL` | `media.title` | |
| `FORM` | (MIME inference) | GEDCOM 5.5.1 format at OBJE level |
| `FORM.MEDI` | `media.properties.medium` | Medium type (photo, document, etc.) |
| `TITL` | `media.title` | |
| `CROP` | `media.properties.crop` | GEDCOM 7.0 crop coordinates |
| `NOTE` | `media.notes` | |
| `BLOB` | (decoded to file) | GEDCOM 5.5.1 deprecated binary data |

## Common Challenges

### Name Conversion

GEDCOM names use slash delimiters for surnames and quotes for nicknames. The importer parses these into structured name fields.

**GEDCOM:**
```
1 NAME Dr. John "Jack" /von Smith/ Jr.
```

**GENEALOGIX:**
```yaml
properties:
  name:
    value: "Dr. John \"Jack\" von Smith Jr."
    fields:
      prefix: "Dr."
      given: "John"
      nickname: "Jack"
      surname_prefix: "von"
      surname: "Smith"
      suffix: "Jr."
```

The importer also handles GEDCOM name substructure tags (`NPFX`, `GIVN`, `NICK`, `SPFX`, `SURN`, `NSFX`) which override the parsed values when present.

**Multiple NAME records** on a single individual are imported as a temporal name list. The `TYPE` subrecord (e.g., `birth`, `married`, `aka`) is preserved as the `type` field. See [Name Variations](/specification/4-entity-types/person#name-variation-examples) for all supported type values.

Recognized surname prefixes include: von, van, de, der, den, del, della, di, da, le, la, du, des, af, av.

### Place Hierarchy

GEDCOM stores places as flat, comma-separated strings (specific to general). The importer builds a proper hierarchy of Place entities with parent references.

**GEDCOM:**
```
2 PLAC Leeds, Yorkshire, England
```

**GENEALOGIX:**
```yaml
places:
  place-3:
    name: "England"
    type: country

  place-2:
    name: "Yorkshire"
    type: county
    parent: place-3

  place-1:
    name: "Leeds"
    type: city
    parent: place-2
```

Place types are inferred from hierarchy depth (city, county, state, country) and keyword detection (cemetery, church, hospital, etc.). Coordinates from `MAP`/`LATI`/`LONG` subrecords are preserved.

Places are deduplicated by name and parent, so "Leeds, Yorkshire, England" appearing in multiple records creates only one set of place entities.

### Date Formats

GEDCOM dates are converted to ISO 8601 format where possible. Qualified and range dates preserve GEDCOM keywords.

| GEDCOM | GENEALOGIX | Description |
|--------|------------|-------------|
| `15 JAN 1850` | `1850-01-15` | Exact date |
| `JAN 1850` | `1850-01` | Month precision |
| `1850` | `1850` | Year precision |
| `ABT 1850` | `ABT 1850` | Approximate |
| `BEF 1920` | `BEF 1920` | Before |
| `AFT 15 MAR 1900` | `AFT 1900-03-15` | After |
| `CAL 1850` | `CAL 1850` | Calculated |
| `BET 1849 AND 1851` | `BET 1849 AND 1851` | Between range |
| `FROM 1900 TO 1950` | `FROM 1900 TO 1950` | Period range |
| `@#DJULIAN@ 15 MAR 1731` | `JULIAN 1731-03-15` | Julian calendar date |
| `@#DHEBREW@ 15 TSH 5765` | `HEBREW 15 TSH 5765` | Hebrew calendar (raw preserved) |
| `@#DFRENCH R@ 1 VEND 0012` | `FRENCH_R 1 VEND 0012` | French Republican (raw preserved) |
| `@#DGREGORIAN@ 15 MAR 1731` | `1731-03-15` | Gregorian (default, no prefix) |

See [Core Concepts - Data Types](/specification/2-core-concepts#data-types) for the complete date format specification.

### Evidence Chains

GEDCOM attaches source citations directly to facts. The importer expands these into complete evidence chains with separate Source, Citation, and Assertion entities.

**GEDCOM:**
```
0 @I1@ INDI
1 BIRT
2 DATE 15 JAN 1850
2 SOUR @S1@
3 PAGE Page 23
3 DATA
4 TEXT "Born January 15, 1850"
```

**GENEALOGIX:**
```yaml
sources:
  source-1:
    title: "Birth Certificate"
    type: vital_record

citations:
  citation-1:
    source: source-1
    properties:
      locator: "Page 23"
      text_from_source: "Born January 15, 1850"

assertions:
  assertion-1:
    subject:
      event: event-person-1-birth
    property: date
    value: "1850-01-15"
    citations: [citation-1]
```

::: tip Assertions require citations
The importer only creates assertions when citations exist. Properties without source citations are stored directly on the entity without an assertion wrapper.
:::

### Pedigree Types

GEDCOM `FAMC.PEDI` tags specify the nature of parent-child relationships. The importer maps these to relationship types.

| PEDI Value | GLX Relationship Type |
|------------|----------------------|
| `birth` | `biological_parent_child` |
| `adopted` | `adoptive_parent_child` |
| `foster` | `foster_parent_child` |
| (empty or `unknown`) | `parent_child` |
| Any other value | `parent_child` |

### Address Handling

GEDCOM `ADDR` records with subfields (`ADR1`, `ADR2`, `CITY`, `STAE`, `POST`, `CTRY`) are handled in two ways:

1. **Full address text**: Preserved in the entity's address properties
2. **Place hierarchy fallback**: When no `PLAC` tag is present on an event, `ADDR` subfields (`CITY`, `STAE`, `CTRY`) are used to build a place hierarchy

### Census Records

Census records receive special handling. Instead of creating events, the importer:

1. Creates a synthetic Source and Citation (titled "Census of {date}" or using the `TYPE` subrecord)
2. Creates temporal residence properties on the person when a place is present
3. Links everything through assertions backed by the census citation
4. For family-level `CENS` records, applies the same data to both spouses

## Post-Migration Workflow

### Validate Your Import

```bash
# Validate the imported archive
glx validate family-archive/

# Or for single-file
glx validate family.glx
```

Fix any reported errors and re-validate. Use `--show-first-errors 0` to see all errors at once.

### Review Import Results

After importing, review the results:

- **Entity counts**: Check the import statistics match expectations for your tree
- **Relationships**: Verify parent-child and spouse relationships were created correctly
- **Places**: Review the inferred place hierarchy and types
- **Evidence chains**: Spot-check that source citations created proper assertions

### Enhancement Opportunities

The automated import creates a solid foundation. Consider enhancing:

- **Confidence levels**: Add `confidence: high/medium/low` to assertions
- **Assertion status**: Add `status: proven/speculative/disproven` to track research verification
- **Transcriptions**: Add `text_from_source` to citations
- **Place details**: Add coordinates and refine place types
- **Research notes**: Add notes to entities documenting your analysis
- **Custom vocabularies**: Extend vocabulary files for domain-specific event or relationship types

### Git Tracking

Initialize version control for your archive:

```bash
cd family-archive/
git init
git add .
git commit -m "Import from GEDCOM: family.ged"
```

## Troubleshooting

### Common Issues

**"Failed to open GEDCOM file"**
Check that the file path is correct and the file is readable.

**Validation errors after import**
Run with `--no-validate` to skip validation and import anyway, then fix issues manually. Use `--show-first-errors 0` to see all errors.

**Missing media files**
Media files referenced by relative paths must exist alongside the `.ged` file. The importer copies them to `media/files/` in your archive. Check that the files exist at the paths specified in your GEDCOM.

**Garbled text or encoding issues**
The parser handles UTF-8 (with or without BOM) and standard line endings (LF, CRLF, CR). If your file uses a different encoding, convert it to UTF-8 first.

**Large GEDCOM files**
The parser supports files up to 1MB line buffer size. Very large files with long continuation lines should import without issues.

**Extension tags not imported**
Custom tags starting with `_` (e.g., `_MARNM`, `_PRIM`) are recognized but not stored in the GLX output. These are logged as warnings in verbose mode.

## GEDCOM Version Differences

Most differences are handled transparently by the importer, but it helps to know what to expect.

| Feature | GEDCOM 5.5.1 | GEDCOM 7.0 |
|---------|-------------|------------|
| **Shared notes** | `NOTE` records with XRef | `SNOTE` tag |
| **Media MIME types** | Inferred from `FORM` tag | Explicit `MIME` tag |
| **External IDs** | Not supported | `EXID` tag with optional `TYPE` |
| **Negative assertions** | Not supported | `NO` tag (e.g., `NO BIRT`) |
| **Crop coordinates** | Not supported | `CROP` tag on media |
| **Extension schemas** | Convention only (`_` prefix) | `SCHMA` tag with URI definitions |
| **Binary data** | `BLOB` tag (deprecated) | Not supported |
| **Void pointers** | Not used | `@VOID@` for embedded structures |

## See Also

- [Quickstart Guide](/quickstart) - Create a new archive from scratch
- [Entity Types](/specification/4-entity-types/) - Entity specifications
- [Best Practices](best-practices) - Workflow recommendations
- [CLI Documentation](/cli) - Command reference
