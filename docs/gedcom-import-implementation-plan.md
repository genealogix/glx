# GEDCOM Import Implementation Plan

**Version**: 1.0
**Date**: 2025-11-18
**Status**: Planning

---

## Executive Summary

This document provides a comprehensive implementation plan for importing GEDCOM files (versions 5.5.1 and 7.0) into the GLX (Genealogix Archive) format. The implementation is designed to handle real-world genealogical data with maximum fidelity while maintaining GLX's evidence-based genealogy principles.

### Objectives

1. **Full GEDCOM 5.5.1 Support**: Import all standard GEDCOM 5.5.1 records and tags
2. **GEDCOM 7.0 Support**: Import GEDCOM 7.0 with modern features (shared notes, extensions, etc.)
3. **Lossless Conversion**: Preserve all meaningful genealogical data during conversion
4. **Evidence Chain Creation**: Transform GEDCOM sources into proper GLX evidence chains
5. **Validation**: Ensure imported data validates against GLX schema

### Test Data Available

**GEDCOM 5.5.1 Examples**:
- `shakespeare.ged` - Small family (434 lines) - Good for basic testing
- `kennedy.ged` - Medium family (1,426 lines) - Real-world complex data
- `british-royalty.ged` - Large family (3,733 lines) - Complex relationships
- `bullinger.ged` - Very large family (17,862 lines) - Stress testing
- `torture-test-551.ged` - Edge cases and specification coverage

**GEDCOM 7.0 Examples**:
- `minimal70.ged` - Minimal valid file (4 lines) - Parser baseline
- `same-sex-marriage.ged` - Modern features (15 lines) - GEDCOM 7.0 flexibility
- `age-all.ged` - All age value formats (410 lines) - Age parsing
- `date-all.ged` - All date formats (10,337 lines!) - Comprehensive date testing
- `maximal70.ged` - Full spec coverage (870 lines) - All GEDCOM 7.0 features

---

## Table of Contents

1. [Unified Setup](#unified-setup)
2. [GEDCOM Structure Overview](#gedcom-structure-overview)
3. [GEDCOM 5.5.1 Implementation](#gedcom-551-implementation)
4. [GEDCOM 7.0 Implementation](#gedcom-70-implementation)
5. [Implementation Phases](#implementation-phases)
6. [Testing Strategy](#testing-strategy)
7. [Error Handling](#error-handling)
8. [Performance Considerations](#performance-considerations)

---

## Unified Setup

### Architecture Overview

```
┌─────────────────┐
│  GEDCOM File    │
│  (.ged)         │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Parser Layer                   │
│  - Lexer (line-by-line)        │
│  - Structure Builder           │
│  - Version Detection           │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Conversion Layer               │
│  - Entity Converters           │
│  - Evidence Chain Builder      │
│  - ID Generator                │
│  - Place Hierarchy Builder     │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────┐
│  GLX Archive    │
│  (GLXFile)      │
└─────────────────┘
```

### Core Components

#### 1. **Parser** (`lib/gedcom_import.go`)

**Responsibilities**:
- Tokenize GEDCOM file line-by-line
- Build hierarchical record structure
- Detect GEDCOM version (5.5.1 vs 7.0)
- Handle character encoding (UTF-8, ANSEL, etc.)
- Validate basic GEDCOM structure

**Key Functions**:
```go
func ImportGEDCOMFromFile(filepath string) (*GLXFile, error)
func ImportGEDCOM(r io.Reader) (*GLXFile, error)
func parseGEDCOM(r io.Reader) ([]*GEDCOMRecord, GEDCOMVersion, error)
func parseGEDCOMLine(text string, lineNum int) (*GEDCOMLine, error)
func buildRecords(lines []*GEDCOMLine) ([]*GEDCOMRecord, error)
```

#### 2. **Converter** (`lib/gedcom_converter.go` - new file)

**Responsibilities**:
- Convert GEDCOM records to GLX entities
- Generate unique IDs for GLX entities
- Build evidence chains from GEDCOM sources
- Create place hierarchies
- Handle custom tags and extensions

**Key Functions**:
```go
func convertToGLX(records []*GEDCOMRecord, version GEDCOMVersion) (*GLXFile, error)
func convertIndividual(record *GEDCOMRecord, ctx *ConversionContext) error
func convertFamily(record *GEDCOMRecord, ctx *ConversionContext) error
func convertSource(record *GEDCOMRecord, ctx *ConversionContext) error
func convertRepository(record *GEDCOMRecord, ctx *ConversionContext) error
func convertMedia(record *GEDCOMRecord, ctx *ConversionContext) error
```

#### 3. **Utilities** (`lib/gedcom_util.go` - new file)

**Responsibilities**:
- Date parsing and conversion
- Name parsing (handle `/surname/` notation)
- Place hierarchy parsing
- ID generation and mapping

**Key Functions**:
```go
func parseGEDCOMDate(gedcomDate string) (interface{}, error)
func parseGEDCOMName(gedcomName string) (given, surname, prefix, suffix string)
func parseGEDCOMPlace(placeStr string, placeForm string) (*PlaceHierarchy, error)
func generateGLXID(gedcomXRef string, entityType string) string
func normalizeDate(date string) (string, error)
```

### Conversion Context

To track state during conversion, we'll use a context object:

```go
type ConversionContext struct {
    GLX          *GLXFile
    Version      GEDCOMVersion

    // ID mapping: GEDCOM XRef -> GLX ID
    PersonIDMap      map[string]string
    FamilyIDMap      map[string]string
    SourceIDMap      map[string]string
    RepositoryIDMap  map[string]string
    MediaIDMap       map[string]string
    PlaceIDMap       map[string]string // place name -> place ID
    NoteIDMap        map[string]string // GEDCOM 7.0 shared notes

    // Counters for generating unique IDs
    PersonCounter      int
    EventCounter       int
    RelationshipCounter int
    PlaceCounter       int
    CitationCounter    int

    // Deferred processing (for forward references)
    DeferredFamilies []*GEDCOMRecord

    // Errors and warnings
    Errors   []string
    Warnings []string
}
```

### ID Generation Strategy

**GEDCOM IDs**: `@I1@`, `@F1@`, `@S1@`, etc.
**GLX IDs**: Human-readable, descriptive, unique

**Strategy**:
1. Extract GEDCOM XRef (`@I1@` → `I1`)
2. Generate GLX ID based on entity type and name/description
3. Maintain mapping for references
4. Ensure uniqueness with counters if needed

**Examples**:
- `@I1@` (William Shakespeare) → `person-william-shakespeare-i1`
- `@F1@` (William & Anne) → `relationship-william-anne-f1`
- `@S1@` → `source-s1` (if no title available)
- `Stratford-upon-Avon` → `place-stratford-upon-avon`

---

## GEDCOM Structure Overview

### Common Elements (Both Versions)

#### Line Format

```
<level> [<xref_id>] <tag> [<value>]
```

**Examples**:
```gedcom
0 @I1@ INDI              # Level 0, XRef @I1@, Tag INDI
1 NAME John /Smith/      # Level 1, Tag NAME, Value "John /Smith/"
2 GIVN John              # Level 2, Tag GIVN, Value "John"
```

#### Hierarchy

```
0 HEAD                   # Header (required)
1 GEDC                   # GEDCOM spec info
2 VERS 5.5.1            # Version
0 @I1@ INDI             # Individual record
1 NAME ...              # Name substructure
2 GIVN ...              # Given name
0 @F1@ FAM              # Family record
1 HUSB @I1@             # Reference to individual
0 TRLR                  # Trailer (required)
```

### Record Types

| Record Type | 5.5.1 | 7.0 | GLX Mapping |
|-------------|-------|-----|-------------|
| `HEAD` | ✓ | ✓ | Metadata (not entity) |
| `INDI` | ✓ | ✓ | Person + Events |
| `FAM` | ✓ | ✓ | Relationship(s) + Events |
| `SOUR` | ✓ | ✓ | Source |
| `REPO` | ✓ | ✓ | Repository |
| `OBJE` | ✓ | ✓ | Media |
| `NOTE` | ✓ | - | Inline notes |
| `SNOTE` | - | ✓ | Shared notes (7.0 only) |
| `SUBM` | ✓ | ✓ | Submitter info (metadata) |
| `TRLR` | ✓ | ✓ | End marker |

---

## GEDCOM 5.5.1 Implementation

### Version Detection

```gedcom
0 HEAD
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
```

**Detection Strategy**: Check for `GEDC.VERS` containing `5.5` or `5.5.1`

### Individual Records (INDI)

#### Structure

```gedcom
0 @I1@ INDI
1 NAME John Fitzgerald "Jack" /Kennedy/
2 TYPE birth
2 GIVN John Fitzgerald
2 NICK Jack
2 SURN Kennedy
2 NSFX Jr
1 SEX M
1 BIRT
2 DATE 29 MAY 1917
2 PLAC Brookline, MA
1 DEAT
2 DATE 22 NOV 1963
2 PLAC Dallas, TX
1 BURI
2 DATE 25 NOV 1963
2 PLAC Arlington National, VA
1 OCCU President
2 DATE FROM 1961 TO 1963
1 NOTE Some biographical note
1 FAMC @F1@
1 FAMS @F0@
1 CHAN
2 DATE 1 MAR 2020
3 TIME 16:17:25.798
```

#### Conversion to GLX

**Person Entity**:
```yaml
persons:
  person-john-kennedy-i1:
    properties:
      name: "John Fitzgerald Kennedy"
      given_name: "John Fitzgerald"
      surname: "Kennedy"
      nickname: "Jack"
      sex: "M"
    notes: "Some biographical note"
    tags: []
```

**Events Created**:
1. **Birth Event**:
```yaml
events:
  event-birth-john-kennedy:
    type: birth
    date: "1917-05-29"
    place: place-brookline-ma
    participants:
      - person: person-john-kennedy-i1
        role: principal
```

2. **Death Event**:
```yaml
events:
  event-death-john-kennedy:
    type: death
    date: "1963-11-22"
    place: place-dallas-tx
    participants:
      - person: person-john-kennedy-i1
        role: principal
```

3. **Burial Event**:
```yaml
events:
  event-burial-john-kennedy:
    type: burial
    date: "1963-11-25"
    place: place-arlington-va
    participants:
      - person: person-john-kennedy-i1
        role: principal
```

4. **Occupation** (stored as temporal property or event):
```yaml
persons:
  person-john-kennedy-i1:
    properties:
      occupation:
        - value: "President"
          date: "1961/1963"
```

**Relationships** (from FAMC/FAMS):
- `FAMC @F1@` → Person is child in family F1 (process when converting FAM record)
- `FAMS @F0@` → Person is spouse in family F0 (process when converting FAM record)

#### INDI Tag Mapping

| GEDCOM Tag | GLX Mapping | Implementation Priority | Notes |
|------------|-------------|-------------------------|-------|
| `NAME` | `properties.name` | Phase 1 (MVP) | Parse `/surname/` notation |
| `GIVN` | `properties.given_name` | Phase 1 | Direct mapping |
| `SURN` | `properties.surname` | Phase 1 | Direct mapping |
| `NICK` | `properties.nickname` | Phase 2 | Simple property |
| `NSFX` | `properties.name_suffix` | Phase 2 | Jr, Sr, III, etc. |
| `NPFX` | `properties.name_prefix` | Phase 2 | Dr, Rev, etc. |
| `SEX` | `properties.sex` | Phase 1 | M, F, U |
| `BIRT` | Event (type: `birth`) | Phase 1 | Create event entity |
| `CHR`/`BAPM` | Event (type: `christening`/`baptism`) | Phase 1 | Create event entity |
| `DEAT` | Event (type: `death`) | Phase 1 | Create event entity |
| `BURI` | Event (type: `burial`) | Phase 1 | Create event entity |
| `CREM` | Event (type: `cremation`) | Phase 2 | Create event entity |
| `ADOP` | Event (type: `adoption`) | Phase 2 | Create event entity |
| `BAPM` | Event (type: `baptism`) | Phase 1 | Create event entity |
| `BARM` | Event (type: `bar_mitzvah`) | Phase 3 | Custom event type |
| `BASM` | Event (type: `bas_mitzvah`) | Phase 3 | Custom event type |
| `BLES` | Event (type: `blessing`) | Phase 3 | Custom event type |
| `CHRA` | Event (type: `adult_christening`) | Phase 3 | Custom event type |
| `CONF` | Event (type: `confirmation`) | Phase 2 | Create event entity |
| `FCOM` | Event (type: `first_communion`) | Phase 3 | Custom event type |
| `ORDN` | Event (type: `ordination`) | Phase 3 | Custom event type |
| `NATU` | Event (type: `naturalization`) | Phase 2 | Create event entity |
| `EMIG` | Event (type: `emigration`) | Phase 2 | Create event entity |
| `IMMI` | Event (type: `immigration`) | Phase 2 | Create event entity |
| `CENS` | Event (type: `census`) | Phase 2 | Create event entity |
| `PROB` | Event (type: `probate`) | Phase 3 | Custom event type |
| `WILL` | Event (type: `will`) | Phase 3 | Custom event type |
| `GRAD` | Event (type: `graduation`) | Phase 2 | Create event entity |
| `RETI` | Event (type: `retirement`) | Phase 3 | Custom event type |
| `RESI` | Event (type: `residence`) or temporal property | Phase 2 | Can be repeating |
| `OCCU` | Temporal property `occupation` | Phase 2 | Can change over time |
| `EDUC` | Temporal property `education` | Phase 3 | Educational background |
| `RELI` | Temporal property `religion` | Phase 3 | Can change over time |
| `NOTE` | `notes` | Phase 1 | Append to notes field |
| `SOUR` | Create Citation + link to subject | Phase 2 | Evidence chain |
| `OBJE` | Media reference (deferred) | Phase 3 | Link to media entity |
| `FAMC` | Create relationship (child) | Phase 1 | Process with FAM |
| `FAMS` | Create relationship (spouse) | Phase 1 | Process with FAM |
| `ASSO` | Store in `notes` or custom | Phase 3 | Associations |
| `REFN` | `properties.reference_number` | Phase 3 | User reference |
| `RIN` | `properties.record_id` | Phase 3 | Automated record ID |
| `CHAN` | Store in `notes` or ignore | Phase 3 | Change history |
| `_CUSTOM` | `properties._custom_tagname` | Phase 3 | Custom tags |

### Family Records (FAM)

#### Structure

```gedcom
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
1 MARR
2 DATE 12 OCT 1953
2 PLAC Newport, RI
1 DIV
2 DATE 1970
1 NOTE Family note
```

#### Conversion to GLX

**Marriage Relationship** (Spouse-Spouse):
```yaml
relationships:
  relationship-john-jackie-f1:
    type: marriage
    participants:
      - person: person-john-kennedy-i1
        role: partner
      - person: person-jacqueline-kennedy-i2
        role: partner
    start_event: event-marriage-john-jackie
    end_event: event-divorce-john-jackie
    notes: "Family note"
```

**Marriage Event**:
```yaml
events:
  event-marriage-john-jackie:
    type: marriage
    date: "1953-10-12"
    place: place-newport-ri
    participants:
      - person: person-john-kennedy-i1
        role: partner
      - person: person-jacqueline-kennedy-i2
        role: partner
```

**Divorce Event**:
```yaml
events:
  event-divorce-john-jackie:
    type: divorce
    date: "1970"
    participants:
      - person: person-john-kennedy-i1
        role: partner
      - person: person-jacqueline-kennedy-i2
        role: partner
```

**Parent-Child Relationships** (one per child):
```yaml
relationships:
  relationship-parents-child1-f1:
    type: parent_child
    participants:
      - person: person-john-kennedy-i1
        role: parent
      - person: person-jacqueline-kennedy-i2
        role: parent
      - person: person-child1-i3
        role: child

  relationship-parents-child2-f1:
    type: parent_child
    participants:
      - person: person-john-kennedy-i1
        role: parent
      - person: person-jacqueline-kennedy-i2
        role: parent
      - person: person-child2-i4
        role: child
```

#### FAM Tag Mapping

| GEDCOM Tag | GLX Mapping | Implementation Priority | Notes |
|------------|-------------|-------------------------|-------|
| `HUSB` | Relationship participant (role: `partner`) | Phase 1 | First partner |
| `WIFE` | Relationship participant (role: `partner`) | Phase 1 | Second partner |
| `CHIL` | Separate relationship per child | Phase 1 | Parent-child relationship |
| `MARR` | `start_event` (Event: marriage) | Phase 1 | Create marriage event |
| `DIV` | `end_event` (Event: divorce) | Phase 2 | Create divorce event |
| `ANUL` | `end_event` (Event: annulment) | Phase 2 | Create annulment event |
| `MARB` | Event (type: `marriage_banns`) | Phase 3 | Marriage banns |
| `MARC` | Event (type: `marriage_contract`) | Phase 3 | Marriage contract |
| `MARL` | Event (type: `marriage_license`) | Phase 3 | Marriage license |
| `MARS` | Event (type: `marriage_settlement`) | Phase 3 | Marriage settlement |
| `ENGA` | Event (type: `engagement`) | Phase 2 | Engagement event |
| `NOTE` | Relationship `notes` | Phase 1 | Direct mapping |
| `SOUR` | Create citations | Phase 2 | Evidence chain |
| `NCHI` | `properties.num_children` | Phase 3 | Number of children |

### Source Records (SOUR)

#### Structure

```gedcom
0 @S1@ SOUR
1 TITL Birth Certificate
1 AUTH County Clerk
1 PUBL Published in 1920
1 DATE 1920
1 REPO @R1@
1 NOTE Source note
```

#### Conversion to GLX

```yaml
sources:
  source-birth-cert-s1:
    title: "Birth Certificate"
    authors: ["County Clerk"]
    publication_info: "Published in 1920"
    date: "1920"
    repository: repository-r1
    notes: "Source note"
```

### Repository Records (REPO)

#### Structure

```gedcom
0 @R1@ REPO
1 NAME Family History Library
1 ADDR 35 North West Temple Street
2 CITY Salt Lake City
2 STAE Utah
2 POST 84150
2 CTRY USA
1 PHON +1-801-555-1234
1 EMAIL info@familysearch.org
1 WWW https://www.familysearch.org
```

#### Conversion to GLX

```yaml
repositories:
  repository-fhl-r1:
    name: "Family History Library"
    address: "35 North West Temple Street"
    city: "Salt Lake City"
    state_province: "Utah"
    postal_code: "84150"
    country: "USA"
    phone: "+1-801-555-1234"
    email: "info@familysearch.org"
    website: "https://www.familysearch.org"
```

### Media Records (OBJE)

#### Structure

```gedcom
0 @O1@ OBJE
1 FILE photo.jpg
2 FORM jpeg
1 TITL Family Photo
```

#### Conversion to GLX

```yaml
media:
  media-family-photo-o1:
    uri: "photo.jpg"
    mime_type: "image/jpeg"
    title: "Family Photo"
```

### Event Substructures (Common)

#### Date Handling

GEDCOM dates need conversion to GLX format:

| GEDCOM Date | GLX Date | Notes |
|-------------|----------|-------|
| `25 DEC 1800` | `1800-12-25` | Full date (ISO 8601) |
| `DEC 1800` | `1800-12` | Year-month |
| `1800` | `1800` | Year only |
| `ABT 1800` | `1800?` or `~1800` | Approximate |
| `BEF 1800` | `<1800` | Before |
| `AFT 1800` | `>1800` | After |
| `BET 1800 AND 1805` | `1800/1805` | Range |
| `FROM 1800 TO 1805` | `1800/1805` | Range |
| `CAL 1800` | `1800` + note "calculated" | Calculated |
| `EST 1800` | `1800` + note "estimated" | Estimated |

#### Place Handling

GEDCOM places are hierarchical comma-separated:

**GEDCOM**: `Leeds, Yorkshire, England`

**GLX** (create hierarchy):
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

**Strategy**:
1. Split by comma
2. Create places from most general to most specific
3. Link with `parent` references
4. Reuse existing places (check `PlaceIDMap`)

#### Source Citations (Embedded)

```gedcom
1 BIRT
2 DATE 1917
2 SOUR @S1@
3 PAGE Page 23
3 QUAY 2
3 NOTE Citation note
```

**Conversion**:
1. Create Citation entity
2. Link to Source (`@S1@`)
3. Link to subject (Person or Event)

```yaml
citations:
  citation-birth-john-s1:
    source: source-birth-cert-s1
    page: "Page 23"
    quality: 2
    notes: "Citation note"
```

Then create Assertion (GLX idiomatic approach):
```yaml
assertions:
  assertion-birth-date-john:
    subject: person-john-kennedy-i1
    claim: birth_date
    value: "1917-05-29"
    confidence: high  # Map from QUAY 2
    citations: [citation-birth-john-s1]
```

### Custom Tags (5.5.1)

GEDCOM 5.5.1 allows custom tags starting with `_`:

```gedcom
1 _UID 9BAE88A3BE7E4AC58C1CCAF92A768780D111
1 _TITL The Kennedy Family
```

**Conversion Strategy**:
- Store in `properties._tagname` (lowercase)
- Example: `properties._uid`, `properties._titl`

---

## GEDCOM 7.0 Implementation

### Version Detection

```gedcom
0 HEAD
1 GEDC
2 VERS 7.0
```

**Detection Strategy**: Check for `GEDC.VERS` containing `7.` or `7.0`

### Key Differences from 5.5.1

| Feature | GEDCOM 5.5.1 | GEDCOM 7.0 | Implementation Impact |
|---------|--------------|------------|----------------------|
| **Shared Notes** | Inline only | `SNOTE` record | Need note resolution |
| **Extensions** | Custom tags (`_TAG`) | `SCHMA` definitions | Parse extension URIs |
| **Time Values** | Dates only | `TIME` subtag | Combine date+time |
| **Phrases** | - | `PHRASE` clarifications | Store in properties/notes |
| **Age Values** | Simple | Complex formats | Enhanced age parsing |
| **NO Tag** | - | Explicitly "no data" | Handle negative assertions |
| **Media** | Simple | Enhanced with `CROP`, multiple titles | Extended media handling |
| **Associations** | Basic `ASSO` | Enhanced with `ROLE.PHRASE` | Richer associations |
| **Restrictions** | `RESN` single | `RESN` comma-separated list | Parse list |
| **Enumerations** | Fixed | Extensible with `PHRASE` | Store phrases |

### Shared Notes (SNOTE)

#### Structure

```gedcom
0 @N1@ SNOTE This is a shared note that can be referenced from multiple records.
2 MIME text/plain
2 LANG en-US
2 TRAN Esta es una nota compartida.
3 LANG es
```

Referenced as:
```gedcom
1 SNOTE @N1@
```

#### Conversion Strategy

1. **First Pass**: Collect all `SNOTE` records into `NoteIDMap`
2. **Second Pass**: Resolve `SNOTE` references and append to entity notes

```yaml
persons:
  person-example:
    notes: |
      This is a shared note that can be referenced from multiple records.

      [Translation - es]: Esta es una nota compartida.
```

### Extension Schema (SCHMA)

#### Structure

```gedcom
1 SCHMA
2 TAG _SKYPEID http://xmlns.com/foaf/0.1/skypeID
2 TAG _JABBERID http://xmlns.com/foaf/0.1/jabberID
```

Then used as:
```gedcom
1 _SKYPEID john.doe
```

#### Conversion Strategy

1. Parse `SCHMA` in header
2. Map extension tags to URIs
3. Store in properties with full context

```yaml
persons:
  person-example:
    properties:
      _skypeid: "john.doe"  # or
      extensions:
        "http://xmlns.com/foaf/0.1/skypeID": "john.doe"
```

### PHRASE Tags

#### Structure

```gedcom
1 SEX M
2 PHRASE Male at birth, non-binary
```

#### Conversion Strategy

Store phrase as additional context:

```yaml
persons:
  person-example:
    properties:
      sex: "M"
      sex_phrase: "Male at birth, non-binary"
```

### Time Values

#### Structure

```gedcom
2 DATE 10 JUN 2022
3 TIME 15:43:20.48Z
3 PHRASE Afternoon
```

#### Conversion Strategy

Combine date and time:

```yaml
events:
  event-example:
    date: "2022-06-10T15:43:20.48Z"
    properties:
      date_phrase: "Afternoon"
```

### NO Tag (Negative Assertions)

#### Structure

```gedcom
1 NO DIV
2 DATE FROM 1700 TO 1800
```

Means: "Explicitly no divorce during 1700-1800"

#### Conversion Strategy

Create assertion with negative value:

```yaml
assertions:
  assertion-no-divorce:
    subject: relationship-example
    claim: divorce
    value: "false"
    properties:
      date_range: "1700/1800"
    confidence: high
```

### Enhanced Media (OBJE)

#### Structure

```gedcom
0 @O1@ OBJE
1 FILE photo.jpg
2 MIME image/jpeg
2 TITL Original photo title
1 CROP
2 TOP 50
2 LEFT 25
2 HEIGHT 200
2 WIDTH 300
```

#### Conversion Strategy

```yaml
media:
  media-photo-o1:
    uri: "photo.jpg"
    mime_type: "image/jpeg"
    title: "Original photo title"
    properties:
      crop:
        top: 50
        left: 25
        height: 200
        width: 300
```

### Same-Sex Marriage Support

GEDCOM 7.0 removes gender assumptions:

```gedcom
0 @F1@ FAM
1 HUSB @I1@  # Can be any gender
1 WIFE @I2@  # Can be any gender
```

**GLX Handling**: Already gender-neutral with `partner` role.

```yaml
relationships:
  relationship-f1:
    type: marriage
    participants:
      - person: person-i1
        role: partner
      - person: person-i2
        role: partner
```

---

## Implementation Phases

### Phase 1: MVP (Minimum Viable Product)

**Goal**: Import basic genealogical data from simple GEDCOM files

**Entities**:
- ✅ Persons (INDI) with basic fields
- ✅ Events (BIRT, DEAT, MARR)
- ✅ Relationships (FAM → parent-child, spouse)
- ✅ Places (basic, no hierarchy yet)

**Features**:
- ✅ GEDCOM 5.5.1 and 7.0 version detection
- ✅ Line-by-line parser
- ✅ Hierarchical record builder
- ✅ Basic name parsing (`/surname/`)
- ✅ Simple date conversion (year, full dates)
- ✅ ID generation and mapping
- ✅ Basic place creation

**Test Files**:
- `minimal70.ged` ✅
- `same-sex-marriage.ged` ✅
- `shakespeare.ged` (partial)

**Success Criteria**:
- Import minimal GEDCOM files without errors
- Create valid GLX archive with persons, events, relationships
- Pass basic validation

**Estimated Effort**: 2-3 days

---

### Phase 2: Core Features

**Goal**: Handle real-world GEDCOM files with common features

**Entities**:
- ✅ Sources (SOUR)
- ✅ Repositories (REPO)
- ✅ Citations (from embedded SOUR)
- ✅ Assertions (from citations)
- ✅ Media (OBJE)

**Features**:
- ✅ Place hierarchies (comma-separated places)
- ✅ Advanced date parsing (ABT, BEF, AFT, ranges)
- ✅ Event type expansion (CHR, BURI, CENS, GRAD, etc.)
- ✅ Source citations and evidence chains
- ✅ Custom properties (OCCU, EDUC, RELI as temporal)
- ✅ Shared notes (GEDCOM 7.0)
- ✅ Basic custom tags (`_TAG`)

**Test Files**:
- `shakespeare.ged` ✅
- `kennedy.ged` ✅
- `british-royalty.ged` (partial)

**Success Criteria**:
- Import medium-sized family trees
- Create proper evidence chains
- Handle most common tags
- Produce valid, well-structured GLX archives

**Estimated Effort**: 4-5 days

---

### Phase 3: Advanced Features

**Goal**: Full GEDCOM spec compliance and edge cases

**Entities**:
- ✅ All entity types complete
- ✅ All event types
- ✅ Complex assertions

**Features**:
- ✅ All event types (30+ types)
- ✅ Extension schema parsing (GEDCOM 7.0)
- ✅ PHRASE tag handling (GEDCOM 7.0)
- ✅ TIME values (GEDCOM 7.0)
- ✅ NO tag (negative assertions)
- ✅ Enhanced media (CROP, multiple titles)
- ✅ Complex date formats (all variations)
- ✅ ASSO (associations)
- ✅ REFN, RIN (reference numbers)
- ✅ CHAN (change history) - store or ignore
- ✅ All custom tags

**Test Files**:
- `british-royalty.ged` ✅
- `bullinger.ged` ✅ (stress test)
- `maximal70.ged` ✅ (full spec coverage)
- `date-all.ged` ✅ (all date formats)
- `age-all.ged` ✅ (all age values)
- `torture-test-551.ged` ✅ (edge cases)

**Success Criteria**:
- 100% GEDCOM 5.5.1 tag coverage
- 100% GEDCOM 7.0 tag coverage
- Handle very large files (15K+ lines)
- Handle all edge cases
- Comprehensive test suite

**Estimated Effort**: 5-7 days

---

### Phase 4: Polish & Optimization

**Goal**: Production-ready import with excellent UX

**Features**:
- ✅ Streaming parser (memory efficient for huge files)
- ✅ Progress reporting
- ✅ Detailed error messages with line numbers
- ✅ Warning system (non-fatal issues)
- ✅ Import report (statistics, skipped items)
- ✅ Encoding detection (UTF-8, ANSEL, etc.)
- ✅ Performance optimization
- ✅ CLI integration (`glx import gedcom`)

**Quality**:
- ✅ 100% test coverage
- ✅ Documentation
- ✅ Examples and tutorials
- ✅ Error recovery strategies

**Estimated Effort**: 3-4 days

---

## Testing Strategy

### Unit Tests

**Coverage**:
- Line parser: `TestParseGEDCOMLine`
- Record builder: `TestBuildRecords`
- Date parser: `TestParseGEDCOMDate` (all formats)
- Name parser: `TestParseGEDCOMName`
- Place parser: `TestParseGEDCOMPlace`
- ID generation: `TestGenerateGLXID`

### Integration Tests

**Test Files** (by phase):

**Phase 1**:
- `minimal70.ged` - Minimal valid file
- `same-sex-marriage.ged` - Basic modern features
- Custom minimal 5.5.1 file

**Phase 2**:
- `shakespeare.ged` - Small real family
- `kennedy.ged` - Medium real family
- Custom file with sources and citations

**Phase 3**:
- `british-royalty.ged` - Large real family
- `bullinger.ged` - Very large family (stress test)
- `maximal70.ged` - Full GEDCOM 7.0 spec
- `date-all.ged` - All date formats
- `age-all.ged` - All age values
- `torture-test-551.ged` - Edge cases

### Validation Tests

After import, validate GLX archive:

```bash
glx validate imported-archive.glx
```

Ensure:
- ✅ No validation errors
- ✅ All references valid
- ✅ Schema compliance

### Comparison Tests

For known files, compare output:

```bash
# Import GEDCOM
glx import gedcom shakespeare.ged > shakespeare.glx

# Validate expected entities exist
# - Expect X persons
# - Expect Y relationships
# - Expect Z events
```

### Round-Trip Tests (Future)

```bash
# Import GEDCOM
glx import gedcom input.ged > output.glx

# Export to GEDCOM
glx export gedcom output.glx > re-exported.ged

# Compare (with normalization)
diff <(normalize-gedcom input.ged) <(normalize-gedcom re-exported.ged)
```

---

## Error Handling

### Error Categories

#### 1. **Fatal Errors** (Stop import)

- Invalid GEDCOM structure (malformed lines)
- Unsupported GEDCOM version (< 5.5)
- File encoding errors
- Circular references

**Handling**: Return error, do not create GLX file

#### 2. **Conversion Errors** (Skip item, continue)

- Invalid XRef (reference to non-existent record)
- Invalid date format (unparseable)
- Missing required fields

**Handling**: Log error, skip item, continue import, report at end

#### 3. **Warnings** (Non-fatal issues)

- Unknown custom tags
- Deprecated tags
- Unusual data (e.g., birth after death)
- Missing optional fields

**Handling**: Log warning, best-effort conversion, report at end

### Error Reporting

```go
type ImportResult struct {
    GLX      *GLXFile
    Errors   []ImportError
    Warnings []ImportWarning
    Stats    ImportStatistics
}

type ImportError struct {
    Line    int
    Record  string // XRef or type
    Field   string
    Message string
}

type ImportWarning struct {
    Line    int
    Record  string
    Message string
}

type ImportStatistics struct {
    PersonsImported      int
    RelationshipsCreated int
    EventsCreated        int
    PlacesCreated        int
    SourcesImported      int
    CitationsCreated     int
    LinesProcessed       int
    RecordsProcessed     int
}
```

**Output**:
```
GEDCOM Import Report
====================

✓ Successfully imported: kennedy.ged

Statistics:
  - 45 persons imported
  - 23 relationships created
  - 156 events created
  - 12 places created
  - 8 sources imported
  - 34 citations created
  - 1,426 lines processed

Warnings (3):
  - Line 234: Unknown custom tag _UIDX on person @I12@
  - Line 567: Date format unusual: "Abt. 1850s" - stored as-is
  - Line 890: Birth date after death date for person @I45@

Output: kennedy.glx
```

---

## Performance Considerations

### Memory Management

**Challenge**: Large GEDCOM files (17K+ lines) can use significant memory

**Strategies**:

1. **Streaming Parser**: Process line-by-line, don't load entire file
2. **Lazy Record Building**: Build records on-demand
3. **ID Mapping**: Use efficient maps, clear after conversion
4. **Deferred Processing**: Process families after all individuals loaded

### Optimization Targets

| File Size | Lines | Target Time | Memory Limit |
|-----------|-------|-------------|--------------|
| Small | < 1K | < 1s | < 50 MB |
| Medium | 1K - 5K | < 5s | < 100 MB |
| Large | 5K - 20K | < 30s | < 500 MB |
| Huge | 20K+ | < 2 min | < 1 GB |

### Profiling

```bash
# Run with profiling
go test -bench=BenchmarkImportLarge -cpuprofile=cpu.prof -memprofile=mem.prof

# Analyze
go tool pprof cpu.prof
go tool pprof mem.prof
```

---

## Implementation Checklist

### Phase 1: MVP

- [ ] Enhance parser to handle GEDCOM 7.0 features
- [ ] Implement `convertIndividual()` for basic INDI fields
- [ ] Implement `convertFamily()` for FAM records
- [ ] Implement basic date parser (year, full dates)
- [ ] Implement name parser (`/surname/` extraction)
- [ ] Implement ID generator with mapping
- [ ] Create basic place entities
- [ ] Write tests for Phase 1 files
- [ ] Update mapping doc with Phase 1 status

### Phase 2: Core

- [ ] Implement `convertSource()` for SOUR records
- [ ] Implement `convertRepository()` for REPO records
- [ ] Implement `convertMedia()` for OBJE records
- [ ] Implement citation creation from embedded SOUR
- [ ] Implement assertion creation from citations
- [ ] Implement place hierarchy parsing
- [ ] Implement advanced date parser (qualifiers, ranges)
- [ ] Implement shared notes (GEDCOM 7.0)
- [ ] Expand event types (CHR, BURI, CENS, etc.)
- [ ] Write tests for Phase 2 files
- [ ] Update mapping doc with Phase 2 status

### Phase 3: Advanced

- [ ] Implement all event types (30+)
- [ ] Implement SCHMA parsing (GEDCOM 7.0)
- [ ] Implement PHRASE handling (GEDCOM 7.0)
- [ ] Implement TIME values (GEDCOM 7.0)
- [ ] Implement NO tag (negative assertions)
- [ ] Implement enhanced media (CROP, etc.)
- [ ] Implement comprehensive date parser (all formats)
- [ ] Implement ASSO handling
- [ ] Implement REFN/RIN
- [ ] Implement custom tag handling
- [ ] Write tests for Phase 3 files
- [ ] Stress test with bullinger.ged
- [ ] Update mapping doc with Phase 3 status

### Phase 4: Polish

- [ ] Implement streaming parser
- [ ] Add progress reporting
- [ ] Improve error messages
- [ ] Implement warning system
- [ ] Create import report
- [ ] Encoding detection
- [ ] Performance profiling
- [ ] CLI integration
- [ ] Write documentation
- [ ] Create examples and tutorials
- [ ] Final review and testing

---

## Appendix: Quick Reference

### GEDCOM 5.5.1 Record Types

- `HEAD` - Header
- `INDI` - Individual
- `FAM` - Family
- `SOUR` - Source
- `REPO` - Repository
- `OBJE` - Multimedia object
- `NOTE` - Note (can be shared)
- `SUBM` - Submitter
- `SUBN` - Submission
- `TRLR` - Trailer

### GEDCOM 7.0 New Features

- `SNOTE` - Shared note (replaces shared NOTE)
- `SCHMA` - Extension schema
- `PHRASE` - Clarification phrases
- `TIME` - Time values
- `NO` - Explicit negative assertions
- Enhanced `OBJE` - Cropping, multiple titles
- Enhanced `ASSO` - Rich associations
- `EXID` - External identifiers

### GLX Entity Types

- Person
- Relationship
- Event
- Place
- Source
- Citation
- Repository
- Assertion
- Media

### Date Format Quick Reference

| GEDCOM | GLX | Phase |
|--------|-----|-------|
| `1850` | `1850` | 1 |
| `25 DEC 1850` | `1850-12-25` | 1 |
| `DEC 1850` | `1850-12` | 2 |
| `ABT 1850` | `1850?` | 2 |
| `BEF 1850` | `<1850` | 2 |
| `AFT 1850` | `>1850` | 2 |
| `BET 1849 AND 1851` | `1849/1851` | 2 |
| `FROM 1849 TO 1851` | `1849/1851` | 2 |

---

**End of Implementation Plan**

*This is a living document. Update as implementation progresses.*
