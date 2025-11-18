# GEDCOM Import - Complete Implementation Plan

**Version**: 2.0 - Unified Complete Coverage
**Date**: 2025-11-18
**Status**: Planning - Complete Coverage

---

## Executive Summary

This document provides a complete, unified implementation plan for importing GEDCOM files (versions 5.5.1 and 7.0) into GLX format. This plan covers 100% of both specifications without phased implementation - everything is planned and mapped comprehensively.

### Scope

- **GEDCOM 5.5.1**: Complete specification coverage (95+ tags)
- **GEDCOM 7.0**: Complete specification coverage (110+ tags)
- **Standard Vocabularies**: Additions needed for full GEDCOM support
- **Custom Properties**: None needed - all handled by standard vocab
- **Evidence Chains**: Complete transformation of GEDCOM sources
- **Test Coverage**: All 12 test files

---

## Table of Contents

1. [Architecture & Core Design](#architecture--core-design)
2. [Complete Tag Inventory](#complete-tag-inventory)
3. [Vocabulary Additions Required](#vocabulary-additions-required)
4. [Property Additions Required](#property-additions-required)
5. [Complete Entity Mappings](#complete-entity-mappings)
6. [Conversion Strategies](#conversion-strategies)
7. [Implementation Components](#implementation-components)
8. [Testing & Validation](#testing--validation)
9. [Performance & Optimization](#performance--optimization)

---

# Architecture & Core Design

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      GEDCOM File (.ged)                      │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                    LEXER & PARSER                            │
│  - Line tokenization (level, xref, tag, value)              │
│  - Hierarchical record building                             │
│  - Version detection (5.5.1 vs 7.0)                         │
│  - Character encoding handling (UTF-8, ANSEL, etc.)         │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              PREPROCESSING & RESOLUTION                      │
│  - Shared note resolution (GEDCOM 7.0)                      │
│  - Extension schema parsing (GEDCOM 7.0)                    │
│  - Cross-reference mapping (XRef → entity relationships)    │
│  - Place hierarchy extraction                               │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                  CONVERSION LAYER                            │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Entity Converters:                                     │  │
│  │ - Individual → Person + Events                         │  │
│  │ - Family → Relationships + Events                      │  │
│  │ - Source → Source                                      │  │
│  │ - Repository → Repository                              │  │
│  │ - Media → Media                                        │  │
│  │ - Place → Place (with hierarchy)                       │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Evidence Chain Builder:                                │  │
│  │ - Citations from embedded SOUR                         │  │
│  │ - Assertions for research conclusions                  │  │
│  └───────────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Utilities:                                             │  │
│  │ - Date parser (all GEDCOM formats → GLX dates)        │  │
│  │ - Name parser (/surname/ notation)                    │  │
│  │ - Place hierarchy builder                              │  │
│  │ - ID generator (GEDCOM XRef → GLX IDs)                │  │
│  └───────────────────────────────────────────────────────┘  │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              POST-PROCESSING & VALIDATION                    │
│  - Reference validation                                      │
│  - Evidence chain completion                                │
│  - Deferred family relationship processing                  │
│  - Place parent linking                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                      GLX Archive                             │
│  - Persons, Events, Relationships, Places                   │
│  - Sources, Citations, Repositories, Assertions             │
│  - Media, Vocabularies, Properties                          │
└─────────────────────────────────────────────────────────────┘
```

## Conversion Context

Central state tracker for the entire conversion process:

```go
type ConversionContext struct {
    // Target GLX archive
    GLX *GLXFile

    // GEDCOM version being processed
    Version GEDCOMVersion

    // ============================================================
    // ID MAPPING: GEDCOM XRef → GLX ID
    // ============================================================
    PersonIDMap     map[string]string  // @I1@ → person-john-smith-i1
    FamilyIDMap     map[string]string  // @F1@ → relationship-...
    SourceIDMap     map[string]string  // @S1@ → source-...
    RepositoryIDMap map[string]string  // @R1@ → repository-...
    MediaIDMap      map[string]string  // @O1@ → media-...
    PlaceIDMap      map[string]string  // "City, State" → place-city-state
    NoteIDMap       map[string]string  // @N1@ → resolved note text (GEDCOM 7.0)
    SubmitterIDMap  map[string]string  // @U1@ → submitter info

    // ============================================================
    // GEDCOM 7.0 SPECIFIC
    // ============================================================
    SharedNotes     map[string]*SharedNote     // @N1@ → note content
    ExtensionSchema map[string]string          // _TAG → URI

    // ============================================================
    // COUNTERS for unique ID generation
    // ============================================================
    PersonCounter       int
    EventCounter        int
    RelationshipCounter int
    PlaceCounter        int
    CitationCounter     int
    AssertionCounter    int

    // ============================================================
    // DEFERRED PROCESSING
    // ============================================================
    // Families must be processed after all individuals are loaded
    // to ensure XRefs can be resolved
    DeferredFamilies []*GEDCOMRecord

    // Place hierarchy must be built after all places extracted
    PlaceHierarchy map[string]*PlaceNode

    // ============================================================
    // METADATA from GEDCOM header
    // ============================================================
    HeaderMetadata map[string]interface{}

    // ============================================================
    // ERROR TRACKING
    // ============================================================
    Errors   []ImportError
    Warnings []ImportWarning

    // ============================================================
    // STATISTICS
    // ============================================================
    Stats ImportStatistics
}

type SharedNote struct {
    ID          string
    Content     string
    MimeType    string
    Language    string
    Translations map[string]string  // lang → translated text
}

type PlaceNode struct {
    Name     string
    Type     string
    Level    int       // 0=most specific, higher=more general
    Parent   *PlaceNode
    Children []*PlaceNode
}

type ImportError struct {
    Line     int
    Record   string  // XRef or tag
    Field    string
    Message  string
    Severity string  // "fatal", "error"
}

type ImportWarning struct {
    Line    int
    Record  string
    Field   string
    Message string
}

type ImportStatistics struct {
    // Input
    LinesProcessed   int
    RecordsProcessed int

    // Entities created
    PersonsImported      int
    RelationshipsCreated int
    EventsCreated        int
    PlacesCreated        int
    SourcesImported      int
    CitationsCreated     int
    RepositoriesImported int
    MediaImported        int
    AssertionsCreated    int

    // By event type
    EventTypeCount map[string]int

    // Issues
    ErrorCount   int
    WarningCount int

    // Skipped items
    SkippedRecords   int
    SkippedTags      int
    UnknownTags      []string
}
```

## ID Generation Strategy

**Principles**:
1. Human-readable when possible
2. Unique across entire archive
3. Reproducible from GEDCOM data
4. Preserve GEDCOM XRef for traceability

**Strategies by Entity Type**:

### Persons
```
GEDCOM: @I1@
Name: "John Fitzgerald Kennedy"

Strategy:
1. Sanitize name: "john-fitzgerald-kennedy"
2. Append GEDCOM XRef: "john-fitzgerald-kennedy-i1"
3. If collision, append counter: "john-fitzgerald-kennedy-i1-2"

Result: person-john-fitzgerald-kennedy-i1
```

### Events
```
Event type: birth
Person: person-john-kennedy-i1

Strategy:
1. Event type: "birth"
2. Primary participant: "john-kennedy"
3. Counter if needed

Result: event-birth-john-kennedy-i1
```

### Relationships
```
Type: marriage
Participants: John, Jacqueline

Strategy:
1. Type: "marriage"
2. Participant names: "john-jacqueline"
3. GEDCOM family ID: "f1"

Result: relationship-marriage-john-jacqueline-f1
```

### Places
```
Place: "Brookline, MA, USA"

Strategy:
1. Sanitize full name: "brookline-ma-usa"
2. Store in PlaceIDMap for reuse
3. If collision (rare), append counter

Result: place-brookline-ma-usa
```

### Sources
```
GEDCOM: @S1@
Title: "Birth Certificate"

Strategy:
1. Sanitize title: "birth-certificate"
2. Append GEDCOM XRef: "s1"
3. If no title, just use XRef

Result: source-birth-certificate-s1
```

### Citations
```
Source: source-birth-certificate-s1
Subject: person-john-kennedy-i1

Strategy:
1. Subject type: "person"
2. Subject ID snippet: "john-kennedy"
3. Source ID snippet: "s1"
4. Counter

Result: citation-person-john-kennedy-s1-1
```

---

# Complete Tag Inventory

## GEDCOM 5.5.1 Tags (Alphabetical)

### Level 0 Record Tags

| Tag | Name | Count in Tests | GLX Mapping | Notes |
|-----|------|----------------|-------------|-------|
| `FAM` | Family | ~20-50/file | Relationship(s) + Events | Multiple relationships per FAM |
| `HEAD` | Header | 1/file | Metadata only | Not an entity |
| `INDI` | Individual | ~50-500/file | Person + Events | Core entity |
| `NOTE` | Shared Note | 0-10/file | Resolve to inline notes | Can be shared |
| `OBJE` | Multimedia | 0-50/file | Media | Images, documents |
| `REPO` | Repository | 0-10/file | Repository | Archives, libraries |
| `SOUR` | Source | 0-20/file | Source | Original materials |
| `SUBM` | Submitter | 1-3/file | Metadata | File creator info |
| `SUBN` | Submission | 0-1/file | Metadata | Submission record |
| `TRLR` | Trailer | 1/file | End marker | Not an entity |

### Individual (INDI) Tags

| Tag | Description | Frequency | GLX Mapping | Priority |
|-----|-------------|-----------|-------------|----------|
| `NAME` | Personal name | Always | properties.name (parsed) | Critical |
| `SEX` | Biological sex | Very High | properties.sex | Critical |
| `BIRT` | Birth event | Very High | Event (type: birth) | Critical |
| `DEAT` | Death event | High | Event (type: death) | Critical |
| `BAPM` | Baptism | Medium | Event (type: baptism) | High |
| `BURI` | Burial | Medium | Event (type: burial) | High |
| `CHR` | Christening | Medium | Event (type: christening) | High |
| `CREM` | Cremation | Low | Event (type: cremation) | Medium |
| `ADOP` | Adoption | Low | Event (type: adoption) | Medium |
| `BARM` | Bar Mitzvah | Low | Event (type: bar_mitzvah) | Medium |
| `BASM` | Bat Mitzvah | Low | Event (type: bat_mitzvah) | Medium |
| `BLES` | Blessing | Low | Event (type: blessing) | Medium |
| `CHRA` | Adult Christening | Low | Event (type: adult_christening) | Medium |
| `CONF` | Confirmation | Medium | Event (type: confirmation) | Medium |
| `FCOM` | First Communion | Low | Event (type: first_communion) | Medium |
| `ORDN` | Ordination | Low | Event (type: ordination) | Medium |
| `NATU` | Naturalization | Low | Event (type: naturalization) | Medium |
| `EMIG` | Emigration | Low | Event (type: emigration) | Medium |
| `IMMI` | Immigration | Low | Event (type: immigration) | Medium |
| `CENS` | Census | Medium | Event (type: census) | Medium |
| `PROB` | Probate | Low | Event (type: probate) | Medium |
| `WILL` | Will | Low | Event (type: will) | Medium |
| `GRAD` | Graduation | Low | Event (type: graduation) | Medium |
| `RETI` | Retirement | Low | Event (type: retirement) | Low |
| `EVEN` | Generic Event | Low | Event (type from TYPE tag) | Medium |
| `CAST` | Caste | Very Low | properties.caste | Low |
| `DSCR` | Description | Low | properties.description | Low |
| `EDUC` | Education | Low | properties.education (temporal) | Medium |
| `IDNO` | ID Number | Low | properties.id_number | Low |
| `NATI` | Nationality | Low | properties.nationality (temporal) | Medium |
| `NCHI` | Children Count | Low | properties.num_children | Low |
| `NMR` | Marriage Count | Low | properties.num_marriages | Low |
| `OCCU` | Occupation | Medium | properties.occupation (temporal) | Medium |
| `PROP` | Property/Possessions | Very Low | properties.possessions | Low |
| `RELI` | Religion | Low | properties.religion (temporal) | Medium |
| `RESI` | Residence | Medium | Event (type: residence) or temporal property | Medium |
| `SSN` | Social Security Number | Low | properties.ssn | Low |
| `TITL` | Title (nobility) | Low | properties.title (temporal) | Low |
| `FACT` | Custom Fact | Low | Event (type from TYPE tag) | Low |
| `NOTE` | Note | High | notes field (append) | Critical |
| `SOUR` | Source Citation | Medium | Create Citation + Assertion | High |
| `OBJE` | Media Link | Low | Link to Media entity | Medium |
| `FAMC` | Child in Family | Very High | Relationship (child role) | Critical |
| `FAMS` | Spouse in Family | High | Relationship (spouse role) | Critical |
| `ASSO` | Association | Very Low | notes or custom relationship | Low |
| `ALIA` | Alias | Very Low | properties.alias | Low |
| `ANCI` | Ancestor Interest | Very Low | Ignore or notes | Very Low |
| `DESI` | Descendant Interest | Very Low | Ignore or notes | Very Low |
| `RFN` | Record File Number | Low | properties.record_file_number | Low |
| `AFN` | Ancestral File Number | Low | properties.ancestral_file_number | Low |
| `REFN` | User Reference | Low | properties.reference_number | Low |
| `RIN` | Automated Record ID | Low | properties.record_id | Low |
| `CHAN` | Change Date | Medium | Store in notes or ignore | Very Low |

### Family (FAM) Tags

| Tag | Description | Frequency | GLX Mapping | Priority |
|-----|-------------|-----------|-------------|----------|
| `HUSB` | Husband/Partner | Very High | Relationship participant (role: partner) | Critical |
| `WIFE` | Wife/Partner | Very High | Relationship participant (role: partner) | Critical |
| `CHIL` | Child | Very High | Separate parent-child relationship per child | Critical |
| `MARR` | Marriage | High | start_event (Event: marriage) | Critical |
| `DIV` | Divorce | Medium | end_event (Event: divorce) | High |
| `ANUL` | Annulment | Low | end_event (Event: annulment) | Medium |
| `ENGA` | Engagement | Low | Event (type: engagement) | Medium |
| `MARB` | Marriage Bann | Low | Event (type: marriage_banns) | Low |
| `MARC` | Marriage Contract | Low | Event (type: marriage_contract) | Low |
| `MARL` | Marriage License | Low | Event (type: marriage_license) | Low |
| `MARS` | Marriage Settlement | Low | Event (type: marriage_settlement) | Low |
| `DIVF` | Divorce Filed | Low | Event (type: divorce_filed) | Low |
| `CENS` | Census | Low | Event (type: census) | Medium |
| `EVEN` | Event | Low | Event (type from TYPE) | Medium |
| `NCHI` | Children Count | Low | properties.num_children | Low |
| `RESI` | Residence | Low | Event (type: residence) | Medium |
| `NOTE` | Note | Medium | notes field | High |
| `SOUR` | Source Citation | Medium | Create Citation | High |
| `OBJE` | Media Link | Low | Link to Media | Low |
| `REFN` | User Reference | Low | properties.reference_number | Low |
| `RIN` | Automated Record ID | Low | properties.record_id | Low |
| `CHAN` | Change Date | Medium | Ignore or notes | Very Low |

### Source (SOUR) Tags

| Tag | Description | Frequency | GLX Mapping | Priority |
|-----|-------------|-----------|-------------|----------|
| `TITL` | Title | Very High | title | Critical |
| `AUTH` | Author | High | authors (array) | Critical |
| `PUBL` | Publication Info | High | publication_info | High |
| `DATE` | Date | Medium | date | High |
| `TEXT` | Source Text | Low | description | Medium |
| `ABBR` | Abbreviation | Low | properties.abbreviation | Low |
| `REPO` | Repository Link | Medium | repository (reference) | High |
| `NOTE` | Note | Medium | notes | High |
| `OBJE` | Media Link | Low | media (array) | Medium |
| `REFN` | User Reference | Low | properties.reference_number | Low |
| `RIN` | Automated Record ID | Low | properties.record_id | Low |
| `CHAN` | Change Date | Low | Ignore | Very Low |

### Repository (REPO) Tags

| Tag | Description | Frequency | GLX Mapping | Priority |
|-----|-------------|-----------|-------------|----------|
| `NAME` | Repository Name | Always | name | Critical |
| `ADDR` | Address | High | address | High |
| `PHON` | Phone | Medium | phone | Medium |
| `EMAIL` | Email | Low | email | Medium |
| `FAX` | Fax | Low | properties.fax | Low |
| `WWW` | Website | Medium | website | High |
| `NOTE` | Note | Low | notes | Medium |
| `REFN` | User Reference | Low | properties.reference_number | Low |
| `RIN` | Automated Record ID | Low | properties.record_id | Low |
| `CHAN` | Change Date | Low | Ignore | Very Low |

### Multimedia (OBJE) Tags

| Tag | Description | Frequency | GLX Mapping | Priority |
|-----|-------------|-----------|-------------|----------|
| `FILE` | File Reference | Always | uri | Critical |
| `FORM` | Format | High | Convert to mime_type | Critical |
| `TITL` | Title | High | title | High |
| `NOTE` | Note | Low | notes | Medium |
| `REFN` | User Reference | Low | properties.reference_number | Low |
| `RIN` | Automated Record ID | Low | properties.record_id | Low |
| `CHAN` | Change Date | Low | Ignore | Very Low |

### Event/Attribute Substructure Tags (Common)

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `TYPE` | Event Type | event.type or event.value | High |
| `DATE` | Date | event.date (parsed) | Critical |
| `PLAC` | Place | event.place (create Place entity) | Critical |
| `ADDR` | Address | event.properties.address | Medium |
| `PHON` | Phone | event.properties.phone | Low |
| `EMAIL` | Email | event.properties.email | Low |
| `FAX` | Fax | event.properties.fax | Low |
| `WWW` | Website | event.properties.website | Low |
| `AGNC` | Agency | event.properties.agency | Medium |
| `RELI` | Religion | event.properties.religion | Low |
| `CAUS` | Cause | event.properties.cause or event.description | High |
| `RESN` | Restriction | event.properties.restriction | Low |
| `NOTE` | Note | event.notes | High |
| `SOUR` | Source | Create Citation linked to event | High |
| `OBJE` | Media | Link Media to event | Medium |
| `AGE` | Age at Event | participant.properties.age or participant.notes | Medium |

### Name Substructure Tags

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `NPFX` | Name Prefix | properties.name_prefix | Medium |
| `GIVN` | Given Name | properties.given_name | Critical |
| `NICK` | Nickname | properties.nickname | Medium |
| `SPFX` | Surname Prefix | properties.surname_prefix | Medium |
| `SURN` | Surname | properties.surname | Critical |
| `NSFX` | Name Suffix | properties.name_suffix | Medium |

### Place Substructure Tags

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `FORM` | Place Hierarchy Format | Used for parsing | High |
| `MAP` | Geographic Coordinates | Container for LAT/LONG | High |
| `LATI` | Latitude | place.latitude | High |
| `LONG` | Longitude | place.longitude | High |

### Source Citation Substructure Tags

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `PAGE` | Page/Location | citation.page | High |
| `EVEN` | Event Type Cited | citation.properties.event_type | Low |
| `ROLE` | Role in Event | citation.properties.role | Low |
| `DATA` | Data | Container for DATE/TEXT | Medium |
| `DATE` | Entry Date | citation.properties.entry_date | Low |
| `TEXT` | Text from Source | citation.text_from_source | High |
| `QUAY` | Quality | citation.quality (0-3) | High |
| `OBJE` | Media | citation.media | Medium |
| `NOTE` | Note | citation.notes | Medium |

### Header (HEAD) Tags

| Tag | Description | Storage | Priority |
|-----|-------------|---------|----------|
| `SOUR` | Source System | properties.gedcom_source | Low |
| `VERS` | Version | properties.gedcom_version | Low |
| `NAME` | Product Name | properties.gedcom_product_name | Low |
| `CORP` | Corporation | properties.gedcom_corporation | Low |
| `ADDR` | Address | properties.gedcom_corp_address | Low |
| `LANG` | Language | properties.gedcom_language | Low |
| `PLAC` | Place Format | Used for parsing, then discard | High |
| `FORM` | Place Form | Used for parsing | High |
| `DATE` | Transmission Date | properties.gedcom_date | Low |
| `TIME` | Time | properties.gedcom_time | Low |
| `SUBM` | Submitter Link | properties.gedcom_submitter | Low |
| `SUBN` | Submission Link | properties.gedcom_submission | Low |
| `FILE` | File Name | properties.gedcom_filename | Low |
| `COPR` | Copyright | properties.gedcom_copyright | Low |
| `GEDC` | GEDCOM | Container for version | Critical |
| `VERS` | GEDCOM Version | Used for version detection | Critical |
| `FORM` | Form (LINEAGE-LINKED) | Verify format | Critical |
| `CHAR` | Character Set | Used for encoding | Critical |
| `LANG` | Language | Store in metadata | Low |
| `DEST` | Destination | properties.gedcom_destination | Low |
| `NOTE` | Note | properties.gedcom_note | Low |

### Address Substructure Tags

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `ADR1` | Address Line 1 | Combine into address field | Medium |
| `ADR2` | Address Line 2 | Combine into address field | Medium |
| `ADR3` | Address Line 3 | Combine into address field | Medium |
| `CITY` | City | city field (if supported) | High |
| `STAE` | State/Province | state_province field | High |
| `POST` | Postal Code | postal_code field | Medium |
| `CTRY` | Country | country field | High |

### Continuation Tags

| Tag | Description | Handling | Priority |
|-----|-------------|----------|----------|
| `CONT` | Continuation (newline) | Append with newline | Critical |
| `CONC` | Concatenation (same line) | Append without newline | Critical |

---

## GEDCOM 7.0 Additional/Changed Tags

### New Record Types

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `SNOTE` | Shared Note | Resolve to inline notes | High |

### New Tags in 7.0

| Tag | Context | Description | GLX Mapping | Priority |
|-----|---------|-------------|-------------|----------|
| `EXID` | Any | External ID | properties.external_ids (array) | Medium |
| `UID` | Any | Unique ID | properties.uid (array, can be multiple) | Medium |
| `MIME` | NOTE/SNOTE | MIME type | properties.mime_type | Medium |
| `LANG` | NOTE/SNOTE | Language | properties.language | Medium |
| `TRAN` | NAME/NOTE/PLAC | Translation | Store as alternative with lang | Medium |
| `PHRASE` | Many | Clarification | Append to value or store in properties | High |
| `TIME` | DATE | Time of day | Combine with date | High |
| `SDATE` | Events | Sort Date | properties.sort_date | Low |
| `NO` | Events | Explicitly No Event | Create negative assertion | Medium |
| `SCHMA` | HEAD | Extension Schema | Parse and store mappings | Medium |
| `CREA` | Any | Creation Date | properties.creation_date | Low |
| `ROLE` | ASSO | Association Role | participant.role or properties.role | Medium |

### Changed Tags in 7.0

| Tag | Change | Impact | Handling |
|-----|--------|--------|----------|
| `RESN` | Can be comma-separated list | Multiple values | Parse list, store as array |
| `SEX` | Can have PHRASE | Clarification allowed | Store phrase separately |
| `HUSB/WIFE` | Gender-neutral | Can be any gender | Already handled (use "partner" role) |
| `OBJE` | Enhanced with CROP, multiple TITL | More metadata | Store in properties |
| `ASSO` | Enhanced with ROLE.PHRASE | Richer data | Store role phrase |
| `AGE` | More formats | Complex age values | Enhanced parsing needed |

### LDS Ordinance Tags (GEDCOM 7.0)

| Tag | Description | GLX Mapping | Priority |
|-----|-------------|-------------|----------|
| `BAPL` | LDS Baptism | Event (type: lds_baptism) | Low |
| `CONL` | LDS Confirmation | Event (type: lds_confirmation) | Low |
| `ENDL` | LDS Endowment | Event (type: lds_endowment) | Low |
| `SLGC` | LDS Sealing Child | Event (type: lds_sealing_child) | Low |
| `SLGS` | LDS Sealing Spouse | Event (type: lds_sealing_spouse) | Low |
| `TEMP` | LDS Temple | properties.lds_temple | Low |
| `STAT` | LDS Ordinance Status | properties.lds_status | Low |

---

# Vocabulary Additions Required

## Event Types to Add

The existing event types cover many GEDCOM events, but we need these additions:

```yaml
# File: specification/5-standard-vocabularies/event-types.glx
# ADD THESE:

event_types:
  # Already exist: birth, death, marriage, divorce, engagement, adoption,
  #                burial, cremation, baptism, confirmation, bar_mitzvah,
  #                bat_mitzvah (as BATM), christening, residence, occupation,
  #                title, nationality, religion, education

  # NEED TO ADD:

  # Religious Events (additions)
  adult_christening:
    label: "Adult Christening"
    description: "Christening of an adult"
    category: "religious"
    gedcom: "CHRA"

  first_communion:
    label: "First Communion"
    description: "First communion ceremony"
    category: "religious"
    gedcom: "FCOM"

  ordination:
    label: "Ordination"
    description: "Religious ordination"
    category: "religious"
    gedcom: "ORDN"

  blessing:
    label: "Blessing"
    description: "Religious blessing"
    category: "religious"
    gedcom: "BLES"

  # LDS Ordinances
  lds_baptism:
    label: "LDS Baptism"
    description: "LDS baptism ordinance"
    category: "religious"
    gedcom: "BAPL"

  lds_confirmation:
    label: "LDS Confirmation"
    description: "LDS confirmation ordinance"
    category: "religious"
    gedcom: "CONL"

  lds_endowment:
    label: "LDS Endowment"
    description: "LDS endowment ordinance"
    category: "religious"
    gedcom: "ENDL"

  lds_sealing_child:
    label: "LDS Sealing to Parents"
    description: "LDS sealing of child to parents"
    category: "religious"
    gedcom: "SLGC"

  lds_sealing_spouse:
    label: "LDS Sealing to Spouse"
    description: "LDS sealing of spouses"
    category: "religious"
    gedcom: "SLGS"

  # Migration Events
  emigration:
    label: "Emigration"
    description: "Emigration from a location"
    category: "migration"
    gedcom: "EMIG"

  immigration:
    label: "Immigration"
    description: "Immigration to a location"
    category: "migration"
    gedcom: "IMMI"

  naturalization:
    label: "Naturalization"
    description: "Obtaining citizenship"
    category: "legal"
    gedcom: "NATU"

  # Life Events
  graduation:
    label: "Graduation"
    description: "Educational graduation"
    category: "achievement"
    gedcom: "GRAD"

  retirement:
    label: "Retirement"
    description: "Retirement from work"
    category: "lifecycle"
    gedcom: "RETI"

  # Census
  census:
    label: "Census"
    description: "Enumeration in census"
    category: "official"
    gedcom: "CENS"

  # Legal Events
  probate:
    label: "Probate"
    description: "Probate of will or estate"
    category: "legal"
    gedcom: "PROB"

  will:
    label: "Will"
    description: "Creation or filing of will"
    category: "legal"
    gedcom: "WILL"

  # Marriage-Related Events (additions)
  annulment:
    label: "Annulment"
    description: "Annulment of marriage"
    category: "lifecycle"
    gedcom: "ANUL"

  marriage_banns:
    label: "Marriage Banns"
    description: "Publication of marriage banns"
    category: "lifecycle"
    gedcom: "MARB"

  marriage_contract:
    label: "Marriage Contract"
    description: "Marriage contract or prenuptial agreement"
    category: "legal"
    gedcom: "MARC"

  marriage_license:
    label: "Marriage License"
    description: "Obtaining marriage license"
    category: "legal"
    gedcom: "MARL"

  marriage_settlement:
    label: "Marriage Settlement"
    description: "Marriage settlement or property arrangement"
    category: "legal"
    gedcom: "MARS"

  divorce_filed:
    label: "Divorce Filed"
    description: "Filing for divorce"
    category: "legal"
    gedcom: "DIVF"
```

## Participant Roles to Add

```yaml
# File: specification/5-standard-vocabularies/participant-roles.glx
# ADD THESE:

participant_roles:
  # Already exist: principal, subject, witness, officiant, informant,
  #                groom, bride, spouse, parent, child,
  #                adoptive-parent, adopted-child, sibling

  # NEED TO ADD:

  partner:
    label: "Partner"
    description: "Partner in a relationship (gender-neutral)"
    applies_to:
      - relationship
      - event

  clergy:
    label: "Clergy"
    description: "Religious clergy or minister"
    applies_to:
      - event

  godparent:
    label: "Godparent"
    description: "Godparent or sponsor"
    applies_to:
      - event
      - relationship

  guardian:
    label: "Guardian"
    description: "Legal guardian"
    applies_to:
      - relationship

  executor:
    label: "Executor"
    description: "Executor of will or estate"
    applies_to:
      - event

  other:
    label: "Other"
    description: "Other unspecified role"
    applies_to:
      - event
      - relationship
```

## Person Properties to Add

```yaml
# File: specification/5-standard-vocabularies/person-properties.glx
# ADD THESE:

person_properties:
  # Already exist: given_name, family_name, gender, born_on, born_at,
  #                died_on, died_at, occupation, residence, religion,
  #                education, ethnicity, nationality, notes

  # NEED TO ADD:

  # Name components
  name_prefix:
    label: "Name Prefix"
    description: "Prefix before name (Dr., Rev., etc.)"
    value_type: string
    temporal: true

  surname_prefix:
    label: "Surname Prefix"
    description: "Prefix before surname (de, von, van, etc.)"
    value_type: string
    temporal: true

  name_suffix:
    label: "Name Suffix"
    description: "Suffix after name (Jr., Sr., III, etc.)"
    value_type: string
    temporal: true

  nickname:
    label: "Nickname"
    description: "Informal or pet name"
    value_type: string
    temporal: true

  alias:
    label: "Alias"
    description: "Alternative name or alias"
    value_type: string
    temporal: true

  # Name metadata
  name:
    label: "Full Name"
    description: "Complete formatted name"
    value_type: string
    temporal: true

  # Biological sex (vs gender identity)
  sex:
    label: "Biological Sex"
    description: "Biological sex (M/F/U/X)"
    value_type: string
    temporal: false

  # Personal attributes
  caste:
    label: "Caste"
    description: "Caste or social status"
    value_type: string
    temporal: true

  description:
    label: "Physical Description"
    description: "Physical description or appearance"
    value_type: string
    temporal: true

  title:
    label: "Title or Nobility"
    description: "Title of nobility or honor"
    value_type: string
    temporal: true

  # Identification numbers
  id_number:
    label: "Identification Number"
    description: "Government or institutional ID"
    value_type: string
    temporal: true

  ssn:
    label: "Social Security Number"
    description: "Social Security Number (use carefully - privacy)"
    value_type: string
    temporal: false

  # Counts
  num_children:
    label: "Number of Children"
    description: "Count of children"
    value_type: integer
    temporal: true

  num_marriages:
    label: "Number of Marriages"
    description: "Count of marriages"
    value_type: integer
    temporal: false

  # Property/possessions
  possessions:
    label: "Property or Possessions"
    description: "Property owned or possessions"
    value_type: string
    temporal: true

  # System IDs (for GEDCOM import tracking)
  record_file_number:
    label: "Record File Number"
    description: "Record file number (RFN)"
    value_type: string
    temporal: false

  ancestral_file_number:
    label: "Ancestral File Number"
    description: "Ancestral file number (AFN)"
    value_type: string
    temporal: false

  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string
    temporal: false

  record_id:
    label: "Automated Record ID"
    description: "Automated record identifier (RIN)"
    value_type: string
    temporal: false

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID, can have multiple)"
    value_type: string
    temporal: false

  external_ids:
    label: "External Identifiers"
    description: "External system identifiers (GEDCOM 7.0 EXID)"
    value_type: string
    temporal: false

  # LDS-specific
  lds_temple:
    label: "LDS Temple"
    description: "LDS temple code"
    value_type: string
    temporal: false

  lds_status:
    label: "LDS Ordinance Status"
    description: "Status of LDS ordinance"
    value_type: string
    temporal: true
```

## Relationship Properties to Add

```yaml
# File: specification/5-standard-vocabularies/relationship-properties.glx
# ADD THESE:

relationship_properties:
  # Already exist: started_on, ended_on, location, description, notes

  # NEED TO ADD:

  num_children:
    label: "Number of Children"
    description: "Count of children in this relationship"
    value_type: integer
    temporal: false

  # System IDs
  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string
    temporal: false

  record_id:
    label: "Automated Record ID"
    description: "Automated record identifier (RIN)"
    value_type: string
    temporal: false

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string
    temporal: false

  restriction:
    label: "Privacy Restriction"
    description: "Privacy or confidentiality restriction"
    value_type: string
    temporal: false
```

## Event Properties to Add

```yaml
# File: specification/5-standard-vocabularies/event-properties.glx
# ADD THESE:

event_properties:
  # Already exist: occurred_on, occurred_at, description, notes

  # NEED TO ADD:

  address:
    label: "Address"
    description: "Street address of event"
    value_type: string
    temporal: false

  phone:
    label: "Phone Number"
    description: "Phone number associated with event"
    value_type: string
    temporal: false

  email:
    label: "Email"
    description: "Email address associated with event"
    value_type: string
    temporal: false

  fax:
    label: "Fax Number"
    description: "Fax number associated with event"
    value_type: string
    temporal: false

  website:
    label: "Website"
    description: "Website URL associated with event"
    value_type: string
    temporal: false

  agency:
    label: "Responsible Agency"
    description: "Agency or organization responsible"
    value_type: string
    temporal: false

  religion:
    label: "Religion"
    description: "Religious affiliation for event"
    value_type: string
    temporal: false

  cause:
    label: "Cause"
    description: "Cause of event (esp. for death)"
    value_type: string
    temporal: false

  restriction:
    label: "Privacy Restriction"
    description: "Privacy or confidentiality restriction"
    value_type: string
    temporal: false

  sort_date:
    label: "Sort Date"
    description: "Date for sorting when actual date uncertain (GEDCOM 7.0)"
    value_type: date
    temporal: false

  event_type:
    label: "Event Type"
    description: "Type or subtype of event"
    value_type: string
    temporal: false

  entry_date:
    label: "Entry Date in Source"
    description: "Date when entered into source record"
    value_type: date
    temporal: false

  age:
    label: "Age at Event"
    description: "Age of person at time of event"
    value_type: string
    temporal: false

  # LDS-specific
  lds_temple:
    label: "LDS Temple"
    description: "LDS temple code"
    value_type: string
    temporal: false

  lds_status:
    label: "LDS Ordinance Status"
    description: "Status of LDS ordinance"
    value_type: string
    temporal: false

  # System IDs
  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string
    temporal: false
```

## Source Properties to Add

```yaml
# ADD to source-types.glx or create source-properties.glx:

source_properties:
  abbreviation:
    label: "Abbreviation"
    description: "Abbreviated source title"
    value_type: string

  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string

  record_id:
    label: "Automated Record ID"
    description: "Automated record identifier (RIN)"
    value_type: string

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string

  copyright:
    label: "Copyright Statement"
    description: "Copyright notice for source"
    value_type: string
```

## Repository Properties to Add

```yaml
# ADD to repository-types.glx or create repository-properties.glx:

repository_properties:
  fax:
    label: "Fax Number"
    description: "Fax number for repository"
    value_type: string

  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string

  record_id:
    label: "Automated Record ID"
    description: "Automated record identifier (RIN)"
    value_type: string

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string
```

## Media Properties to Add

```yaml
# File: specification/5-standard-vocabularies/media-properties.glx (NEW FILE)

media_properties:
  format:
    label: "File Format"
    description: "Original file format from GEDCOM"
    value_type: string

  crop_top:
    label: "Crop Top"
    description: "Top crop coordinate (GEDCOM 7.0)"
    value_type: integer

  crop_left:
    label: "Crop Left"
    description: "Left crop coordinate (GEDCOM 7.0)"
    value_type: integer

  crop_height:
    label: "Crop Height"
    description: "Crop height (GEDCOM 7.0)"
    value_type: integer

  crop_width:
    label: "Crop Width"
    description: "Crop width (GEDCOM 7.0)"
    value_type: integer

  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string

  record_id:
    label: "Automated Record ID"
    description: "Automated record identifier (RIN)"
    value_type: string

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string
```

## Citation Properties to Add

```yaml
# File: specification/5-standard-vocabularies/citation-properties.glx (NEW FILE)

citation_properties:
  event_type:
    label: "Event Type Cited"
    description: "Type of event being cited from source"
    value_type: string

  role:
    label: "Role in Event"
    description: "Role of subject in cited event"
    value_type: string

  entry_date:
    label: "Entry Date in Source"
    description: "Date when information was entered in source"
    value_type: date
```

## Place Properties to Add

```yaml
# File: specification/5-standard-vocabularies/place-properties.glx
# ADD THESE:

place_properties:
  # If not already present:

  abbreviation:
    label: "Abbreviation"
    description: "Abbreviated place name"
    value_type: string

  phonetic:
    label: "Phonetic"
    description: "Phonetic spelling of place name"
    value_type: string

  romanized:
    label: "Romanized"
    description: "Romanized version of place name"
    value_type: string

  language:
    label: "Language"
    description: "Language of place name"
    value_type: string

  reference_number:
    label: "User Reference Number"
    description: "User-defined reference number"
    value_type: string

  uid:
    label: "Unique Identifier"
    description: "Unique identifier (UID)"
    value_type: string
```

---

# Complete Entity Mappings

## Individual (INDI) → Person + Events

### Person Entity Creation

```yaml
persons:
  {generated-id}:
    properties:
      # From NAME tag
      name: {full_name}                    # Full parsed name
      given_name: {GIVN}                   # Given name(s)
      surname: {SURN}                      # Surname
      name_prefix: {NPFX}                  # Dr., Rev., etc.
      surname_prefix: {SPFX}               # de, von, van
      name_suffix: {NSFX}                  # Jr., Sr., III
      nickname: {NICK}                     # Nickname

      # From SEX tag
      sex: {M|F|U|X}                       # Biological sex

      # From attribute tags (stored as properties, not events)
      caste: {CAST value}
      description: {DSCR value}
      id_number: {IDNO value}
      num_children: {NCHI value}
      num_marriages: {NMR value}
      possessions: {PROP value}
      ssn: {SSN value}

      # Temporal properties (can have multiple with dates)
      occupation: [{value, date}, ...]     # From OCCU
      education: [{value, date}, ...]      # From EDUC
      nationality: [{value, date}, ...]    # From NATI
      religion: [{value, date}, ...]       # From RELI
      residence: [{place_ref, date}, ...]  # From RESI
      title: [{value, date}, ...]          # From TITL

      # System/tracking IDs
      record_file_number: {RFN}
      ancestral_file_number: {AFN}
      reference_number: {REFN}
      record_id: {RIN}
      uid: [{UID1}, {UID2}, ...]           # Can have multiple
      external_ids: [{type: url, id: val}] # GEDCOM 7.0 EXID

      # LDS-specific
      lds_temple: {TEMP}
      lds_status: {STAT}

    notes: |
      {Combined from all NOTE tags}
      {ASSO associations as text}
      {ALIA aliases}
      {CHAN change history if desired}

    tags: []
```

### Events Created from INDI

For each event tag on an Individual:

```yaml
events:
  {generated-event-id}:
    type: {event_type}                    # Mapped from tag
    date: {parsed_date}                   # From DATE subtag
    place: {place_id}                     # From PLAC subtag
    value: {TYPE value or tag value}      # Descriptive value

    participants:
      - person: {person_id}
        role: principal
        properties:
          age: {AGE value}                # If AGE subtag present
        notes: {AGE PHRASE or other participant notes}

    properties:
      address: {ADDR}
      phone: {PHON}
      email: {EMAIL}
      fax: {FAX}
      website: {WWW}
      agency: {AGNC}
      religion: {RELI}
      cause: {CAUS}
      restriction: {RESN}
      sort_date: {SDATE}                  # GEDCOM 7.0
      event_type: {TYPE}
      uid: [{UID}]

      # LDS-specific
      lds_temple: {TEMP}
      lds_status: {STAT}

      # GEDCOM 7.0
      phrase: {PHRASE}                    # Clarification

    description: {Combined CAUS or other descriptive text}
    notes: {Combined NOTE tags}
    tags: []
```

Event types created from INDI tags:
- `BIRT` → `birth`
- `DEAT` → `death`
- `BAPM` → `baptism`
- `BURI` → `burial`
- `CHR` → `christening`
- `CREM` → `cremation`
- `ADOP` → `adoption`
- `BARM` → `bar_mitzvah`
- `BASM` → `bat_mitzvah`
- `BLES` → `blessing`
- `CHRA` → `adult_christening`
- `CONF` → `confirmation`
- `FCOM` → `first_communion`
- `ORDN` → `ordination`
- `NATU` → `naturalization`
- `EMIG` → `emigration`
- `IMMI` → `immigration`
- `CENS` → `census`
- `PROB` → `probate`
- `WILL` → `will`
- `GRAD` → `graduation`
- `RETI` → `retirement`
- `RESI` → `residence` (can also be temporal property)
- `OCCU` → `occupation` (usually temporal property, but can be event)
- `EVEN` → Type from `TYPE` subtag
- `FACT` → Type from `TYPE` subtag
- `BAPL` → `lds_baptism`
- `CONL` → `lds_confirmation`
- `ENDL` → `lds_endowment`
- `SLGC` → `lds_sealing_child`

### Negative Assertions (GEDCOM 7.0)

For `NO` tags (e.g., `NO NATU`):

```yaml
assertions:
  {generated-assertion-id}:
    subject: {person_id}
    claim: {event_type}_occurred
    value: "false"
    confidence: high
    properties:
      date_range: {parsed DATE if present}
      phrase: {PHRASE if present}
    notes: {NOTE tags}
    citations: [{citation_ids from SOUR}]
```

### Citations from INDI Events

For each `SOUR` tag under an event:

```yaml
citations:
  {generated-citation-id}:
    source: {source_id}                   # From SOUR @ID@
    page: {PAGE}
    text_from_source: {TEXT}
    quality: {QUAY}                       # 0-3
    properties:
      event_type: {EVEN}                  # If EVEN subtag
      role: {ROLE}                        # If ROLE subtag
      entry_date: {DATA.DATE}             # Entry date in source
    notes: {NOTE}
    media: [{media_ids from OBJE}]
```

Then create assertion:

```yaml
assertions:
  {generated-assertion-id}:
    subject: {event_id}
    claim: {appropriate claim type}
    value: {extracted value}
    confidence: {mapped from QUAY: 0→very_low, 1→low, 2→medium, 3→high}
    citations: [{citation_id}]
```

---

## Family (FAM) → Relationship(s) + Events

A single GEDCOM FAM record creates:
1. **One spouse/partner relationship** (if HUSB and/or WIFE present)
2. **One parent-child relationship per CHIL** (linking parents to each child)
3. **Events** for marriage, divorce, and other family events

### Spouse Relationship

```yaml
relationships:
  {generated-rel-id}:
    type: marriage                        # Or "partner" if not married

    participants:
      - person: {husb_person_id}
        role: partner
        properties:
          phrase: {HUSB.PHRASE}           # GEDCOM 7.0
      - person: {wife_person_id}
        role: partner
        properties:
          phrase: {WIFE.PHRASE}           # GEDCOM 7.0

    start_event: {marriage_event_id}      # If MARR present
    end_event: {divorce_event_id}         # If DIV present

    properties:
      num_children: {NCHI}
      reference_number: {REFN}
      record_id: {RIN}
      uid: {UID}
      restriction: {RESN}

    description: {Descriptive text}
    notes: {Combined NOTE tags}
    tags: []
```

### Parent-Child Relationships

For each `CHIL` tag:

```yaml
relationships:
  {generated-rel-id}:
    type: parent-child

    participants:
      - person: {husb_person_id}
        role: parent
      - person: {wife_person_id}
        role: parent
      - person: {child_person_id}
        role: child
        properties:
          phrase: {CHIL.PHRASE}           # GEDCOM 7.0

    properties:
      # Adoption info if from ADOP event
      adoption_by: {BOTH|HUSB|WIFE}       # If adoption

    notes: {Any relevant notes}
```

### Family Events

For each family event tag:

```yaml
events:
  {generated-event-id}:
    type: {event_type}
    date: {DATE}
    place: {place_id from PLAC}

    participants:
      - person: {husb_person_id}
        role: partner                     # Or bride/groom if appropriate
        properties:
          age: {HUSB.AGE}
          phrase: {HUSB.AGE.PHRASE}
      - person: {wife_person_id}
        role: partner
        properties:
          age: {WIFE.AGE}
          phrase: {WIFE.AGE.PHRASE}

    properties:
      address: {ADDR}
      phone: {PHON}
      email: {EMAIL}
      fax: {FAX}
      website: {WWW}
      agency: {AGNC}
      religion: {RELI}
      cause: {CAUS}
      restriction: {RESN}
      sort_date: {SDATE}
      uid: {UID}
      phrase: {PHRASE}

    description: {TYPE or other descriptive text}
    notes: {NOTE}
    tags: []
```

Family event types:
- `MARR` → `marriage`
- `DIV` → `divorce`
- `ANUL` → `annulment`
- `ENGA` → `engagement`
- `MARB` → `marriage_banns`
- `MARC` → `marriage_contract`
- `MARL` → `marriage_license`
- `MARS` → `marriage_settlement`
- `DIVF` → `divorce_filed`
- `CENS` → `census` (family census)
- `RESI` → `residence` (family residence)
- `EVEN` → Type from `TYPE`
- `SLGS` → `lds_sealing_spouse`

### Negative Assertions for Family (GEDCOM 7.0)

For `NO DIV`, `NO ANUL`:

```yaml
assertions:
  {generated-assertion-id}:
    subject: {relationship_id}
    claim: {event_type}_occurred
    value: "false"
    confidence: high
    properties:
      date_range: {DATE if present}
    notes: {NOTE}
    citations: [{citation_ids}]
```

---

## Source (SOUR) → Source

```yaml
sources:
  {generated-source-id}:
    title: {TITL}
    type: {inferred from content or "other"}
    authors: [{AUTH split by commas or newlines}]
    date: {DATE}
    publication_info: {PUBL}
    repository: {repo_id from REPO}
    description: {TEXT}
    notes: {NOTE}
    media: [{media_ids from OBJE}]
    properties:
      abbreviation: {ABBR}
      reference_number: {REFN}
      record_id: {RIN}
      uid: {UID}
      copyright: {COPR}
    tags: []
```

### Source Type Inference

Try to infer source type from content:

| Indicators | Source Type |
|------------|-------------|
| "birth certificate", "death certificate" | `vital_record` |
| "census", "enumeration" | `census` |
| "baptism register", "parish register" | `church_register` |
| "military", "service record" | `military` |
| "newspaper", "gazette" | `newspaper` |
| "will", "probate", "estate" | `probate` |
| "deed", "land grant" | `land` |
| "court", "trial" | `court` |
| "passenger list", "naturalization" | `immigration` |
| "directory", "phone book" | `directory` |
| "book", "published" | `book` |
| "database", "ancestry", "familysearch" | `database` |
| "interview", "oral history" | `oral_history` |
| "letter", "correspondence" | `correspondence` |
| "photo", "photograph" | `photograph` |
| Default | `other` |

---

## Repository (REPO) → Repository

```yaml
repositories:
  {generated-repo-id}:
    name: {NAME}
    type: {inferred or "other"}
    address: {ADDR or combined ADR1/ADR2/ADR3}
    city: {CITY}
    state_province: {STAE}
    postal_code: {POST}
    country: {CTRY}
    phone: {PHON}
    email: {EMAIL}
    website: {WWW}
    notes: {NOTE}
    properties:
      fax: {FAX}
      reference_number: {REFN}
      record_id: {RIN}
      uid: {UID}
    tags: []
```

### Repository Type Inference

| Indicators in Name | Repository Type |
|-------------------|-----------------|
| "National Archives", "State Archives" | `archive` |
| "Library", "Public Library" | `library` |
| "Church", "Cathedral", "Parish" | `church` |
| "FamilySearch", "Ancestry", "MyHeritage" | `database` |
| "Museum" | `museum` |
| "Vital Records", "Registry", "Bureau" | `registry` |
| "Historical Society", "Genealogical Society" | `historical_society` |
| "University", "College" | `university` |
| "Department", "Office", "Bureau" | `government_agency` |
| Default | `other` |

---

## Multimedia (OBJE) → Media

```yaml
media:
  {generated-media-id}:
    uri: {FILE}
    mime_type: {converted from FORM or MIME}
    title: {TITL}
    notes: {NOTE}
    properties:
      format: {FORM}                      # Original GEDCOM format

      # GEDCOM 7.0 crop info
      crop_top: {CROP.TOP}
      crop_left: {CROP.LEFT}
      crop_height: {CROP.HEIGHT}
      crop_width: {CROP.WIDTH}

      reference_number: {REFN}
      record_id: {RIN}
      uid: {UID}
    tags: []
```

### Format to MIME Type Conversion

| GEDCOM FORM | MIME Type |
|-------------|-----------|
| `bmp` | `image/bmp` |
| `gif` | `image/gif` |
| `jpg`, `jpeg` | `image/jpeg` |
| `ole` | `application/ole` |
| `pcx` | `image/pcx` |
| `tif`, `tiff` | `image/tiff` |
| `png` | `image/png` |
| `pdf` | `application/pdf` |
| `wav` | `audio/wav` |
| `mp3` | `audio/mpeg` |
| `avi` | `video/avi` |
| `mpg`, `mpeg` | `video/mpeg` |
| `mp4` | `video/mp4` |
| `txt` | `text/plain` |
| `rtf` | `application/rtf` |
| `html`, `htm` | `text/html` |
| Default | `application/octet-stream` |

---

## Place (PLAC) → Place + Hierarchy

GEDCOM places are hierarchical strings: "City, County, State, Country"

### Parsing Strategy

1. **Extract place format** from `HEAD.PLAC.FORM` (if present)
   - Example: "City, County, State, Country"
   - Defines hierarchy levels

2. **Split place string** by commas
   - `"Brookline, Norfolk County, Massachusetts, USA"` →
   - `["Brookline", "Norfolk County", "Massachusetts", "USA"]`

3. **Create place hierarchy** from most general to most specific
   - Start with country (if present)
   - Build parent-child relationships

4. **Reuse existing places** using `PlaceIDMap`

### Place Hierarchy Creation

```yaml
places:
  # Level 0 (most general - country)
  place-usa:
    name: "USA"
    type: country
    parent: null

  # Level 1 (state/province)
  place-massachusetts:
    name: "Massachusetts"
    type: state
    parent: place-usa

  # Level 2 (county)
  place-norfolk-county-ma:
    name: "Norfolk County"
    type: county
    parent: place-massachusetts

  # Level 3 (most specific - city)
  place-brookline-ma:
    name: "Brookline"
    type: city
    parent: place-norfolk-county-ma
    latitude: {MAP.LATI if present}
    longitude: {MAP.LONG if present}
    properties:
      abbreviation: {if any}
      language: {LANG if GEDCOM 7.0}
      uid: {UID if present}
    notes: {NOTE if any}
```

### Place Type Inference

Infer type from:
1. **Position in hierarchy** (if format known)
2. **Keywords in name**:
   - "United States", "USA", "France" → `country`
   - "County", "Shire" → `county`
   - "Parish" → `parish`
   - "City", "Town", "Village" → `city` or `town`
   - "District" → `district`
   - "Region" → `region`
3. **Default by level**:
   - 4 parts: city, county, state, country
   - 3 parts: city, state, country
   - 2 parts: city, country
   - 1 part: city

### Alternative Names (GEDCOM 7.0)

If place has `TRAN` (translation):

```yaml
place-example:
  name: "Primary Name"
  alternative_names:
    - name: {TRAN value}
      type: translation
      language: {TRAN.LANG}
      date_range: {if temporal}
```

---

## Shared Notes (GEDCOM 7.0)

### Phase 1: Collection

During parsing, collect all `SNOTE` records:

```go
// In ConversionContext
SharedNotes map[string]*SharedNote

type SharedNote struct {
    ID           string
    Content      string
    MimeType     string
    Language     string
    Translations map[string]string
}
```

Example GEDCOM 7.0 shared note:

```gedcom
0 @N1@ SNOTE This is a shared note.
1 MIME text/plain
1 LANG en-US
1 TRAN Esta es una nota compartida.
2 LANG es
```

Stored as:

```go
SharedNotes["@N1@"] = &SharedNote{
    ID: "@N1@",
    Content: "This is a shared note.",
    MimeType: "text/plain",
    Language: "en-US",
    Translations: map[string]string{
        "es": "Esta es una nota compartida.",
    },
}
```

### Phase 2: Resolution

When encountering `SNOTE @N1@` reference:

1. Look up in `SharedNotes` map
2. Append to entity's `notes` field
3. Include translations if present

```yaml
persons:
  person-example:
    notes: |
      This is a shared note.

      [Translation - es]: Esta es una nota compartida.
```

---

# Conversion Strategies

## Date Parsing Strategy

### GEDCOM Date Formats

GEDCOM supports complex date expressions that must be parsed to GLX format.

#### Exact Dates

| GEDCOM | Description | GLX Output | Notes |
|--------|-------------|------------|-------|
| `1850` | Year only | `1850` | Direct |
| `JAN 1850` | Month and year | `1850-01` | ISO 8601 partial |
| `25 JAN 1850` | Full date | `1850-01-25` | ISO 8601 full |
| `DEC 1850` | Month and year | `1850-12` | Month name → number |

#### Approximate Dates

| GEDCOM | Meaning | GLX Output | Strategy |
|--------|---------|------------|----------|
| `ABT 1850` | About | `~1850` or `1850?` | Use `~` prefix or `?` suffix |
| `CAL 1850` | Calculated | `1850` | Store in properties.date_calculated |
| `EST 1850` | Estimated | `1850` | Store in properties.date_estimated |

#### Relative Dates

| GEDCOM | Meaning | GLX Output | Strategy |
|--------|---------|------------|----------|
| `BEF 1850` | Before | `<1850` | Use `<` prefix |
| `AFT 1850` | After | `>1850` | Use `>` prefix |
| `BET 1849 AND 1851` | Between | `1849/1851` | ISO 8601 interval |
| `FROM 1849 TO 1851` | From-to | `1849/1851` | ISO 8601 interval |

#### Interpreted Dates

| GEDCOM | Meaning | GLX Strategy |
|--------|---------|--------------|
| `INT 25 JAN 1850 (interpreted)` | Interpreted date with phrase | Store date + phrase in properties |
| `(phrase)` | Any phrase in parens | Store in properties.date_phrase |

#### GEDCOM 7.0 Extensions

| GEDCOM 7.0 | GLX Strategy |
|------------|--------------|
| `DATE 25 JAN 1850` + `TIME 14:30:00` | Combine: `1850-01-25T14:30:00` |
| `DATE 25 JAN 1850` + `PHRASE "late evening"` | Date + properties.date_phrase |

### Date Parser Implementation

```go
func parseGEDCOMDate(gedcomDate string, timeSuffix string, phrase string) (interface{}, map[string]interface{}) {
    properties := make(map[string]interface{})

    // Store phrase if present
    if phrase != "" {
        properties["date_phrase"] = phrase
    }

    // Clean up date string
    date := strings.TrimSpace(gedcomDate)

    // Parse modifiers
    if strings.HasPrefix(date, "ABT ") {
        date = strings.TrimPrefix(date, "ABT ")
        return "~" + parseExactDate(date, timeSuffix), properties
    }

    if strings.HasPrefix(date, "CAL ") {
        properties["date_calculated"] = true
        date = strings.TrimPrefix(date, "CAL ")
        return parseExactDate(date, timeSuffix), properties
    }

    if strings.HasPrefix(date, "EST ") {
        properties["date_estimated"] = true
        date = strings.TrimPrefix(date, "EST ")
        return parseExactDate(date, timeSuffix), properties
    }

    if strings.HasPrefix(date, "BEF ") {
        date = strings.TrimPrefix(date, "BEF ")
        return "<" + parseExactDate(date, timeSuffix), properties
    }

    if strings.HasPrefix(date, "AFT ") {
        date = strings.TrimPrefix(date, "AFT ")
        return ">" + parseExactDate(date, timeSuffix), properties
    }

    if strings.Contains(date, " AND ") {
        // BET ... AND ... or just range
        parts := strings.Split(date, " AND ")
        start := strings.TrimPrefix(parts[0], "BET ")
        end := parts[1]
        return parseExactDate(start, "") + "/" + parseExactDate(end, ""), properties
    }

    if strings.HasPrefix(date, "FROM ") && strings.Contains(date, " TO ") {
        parts := strings.Split(date, " TO ")
        start := strings.TrimPrefix(parts[0], "FROM ")
        end := parts[1]
        return parseExactDate(start, "") + "/" + parseExactDate(end, ""), properties
    }

    // Exact date
    return parseExactDate(date, timeSuffix), properties
}

func parseExactDate(date string, timeSuffix string) string {
    // Parse formats:
    // - "1850"
    // - "JAN 1850"
    // - "25 JAN 1850"

    parts := strings.Fields(date)

    if len(parts) == 1 {
        // Year only
        return parts[0]
    }

    if len(parts) == 2 {
        // Month Year
        month := monthToNumber(parts[0])
        year := parts[1]
        result := year + "-" + month
        if timeSuffix != "" {
            result += "T" + timeSuffix
        }
        return result
    }

    if len(parts) == 3 {
        // Day Month Year
        day := fmt.Sprintf("%02s", parts[0])
        month := monthToNumber(parts[1])
        year := parts[2]
        result := year + "-" + month + "-" + day
        if timeSuffix != "" {
            result += "T" + timeSuffix
        }
        return result
    }

    // Fallback: return as-is
    return date
}

func monthToNumber(month string) string {
    months := map[string]string{
        "JAN": "01", "FEB": "02", "MAR": "03", "APR": "04",
        "MAY": "05", "JUN": "06", "JUL": "07", "AUG": "08",
        "SEP": "09", "OCT": "10", "NOV": "11", "DEC": "12",
    }
    return months[strings.ToUpper(month)]
}
```

---

## Name Parsing Strategy

GEDCOM uses `/surname/` notation for names.

### Basic Format

```
GIVEN_NAMES /SURNAME/ SUFFIX
```

Examples:
- `John /Smith/`
- `Mary Jane /Doe/`
- `Robert /de La Cruz/ Jr.`
- `Dr. John Q. /Public/ III`

### With Substructure

```gedcom
1 NAME Lt. Cmndr. Joseph "John" /de Allen/ jr.
2 NPFX Lt. Cmndr.
2 GIVN Joseph
2 NICK John
2 SPFX de
2 SURN Allen
2 NSFX jr.
```

### Parsing Strategy

**Priority**:
1. If substructure present (GIVN, SURN, etc.), use those
2. Otherwise, parse the NAME value

**Parser**:

```go
func parseGEDCOMName(nameValue string, substructure *NameSubstructure) PersonName {
    name := PersonName{}

    // Use substructure if available
    if substructure != nil {
        name.Prefix = substructure.NPFX
        name.GivenName = substructure.GIVN
        name.Nickname = substructure.NICK
        name.SurnamePrefix = substructure.SPFX
        name.Surname = substructure.SURN
        name.Suffix = substructure.NSFX
        return name
    }

    // Parse from NAME value
    // Extract surname between slashes
    surnameRegex := regexp.MustCompile(`/([^/]+)/`)
    matches := surnameRegex.FindStringSubmatch(nameValue)

    if len(matches) > 1 {
        // Extract surname parts
        surnamePart := matches[1]
        // Check for surname prefix (de, von, van, etc.)
        surnameWords := strings.Fields(surnamePart)
        if len(surnameWords) > 1 && isSurnamePrefix(surnameWords[0]) {
            name.SurnamePrefix = surnameWords[0]
            name.Surname = strings.Join(surnameWords[1:], " ")
        } else {
            name.Surname = surnamePart
        }

        // Remove surname from name value
        nameValue = surnameRegex.ReplaceAllString(nameValue, "")
    }

    // Split remaining parts
    parts := strings.Fields(nameValue)

    // Extract prefix (Dr., Rev., Lt., etc.)
    for len(parts) > 0 && isNamePrefix(parts[0]) {
        name.Prefix += parts[0] + " "
        parts = parts[1:]
    }
    name.Prefix = strings.TrimSpace(name.Prefix)

    // Extract suffix (Jr., Sr., III, etc.)
    for len(parts) > 0 && isNameSuffix(parts[len(parts)-1]) {
        name.Suffix = parts[len(parts)-1] + " " + name.Suffix
        parts = parts[:len(parts)-1]
    }
    name.Suffix = strings.TrimSpace(name.Suffix)

    // Extract nickname (in quotes)
    nicknameRegex := regexp.MustCompile(`"([^"]+)"`)
    for _, part := range parts {
        if nicknameRegex.MatchString(part) {
            matches := nicknameRegex.FindStringSubmatch(part)
            name.Nickname = matches[1]
        }
    }

    // Remove nicknames from parts
    filteredParts := []string{}
    for _, part := range parts {
        if !nicknameRegex.MatchString(part) {
            filteredParts = append(filteredParts, part)
        }
    }

    // Remaining parts are given name(s)
    name.GivenName = strings.Join(filteredParts, " ")

    return name
}

func isSurnamePrefix(word string) bool {
    prefixes := []string{"de", "von", "van", "del", "la", "le", "di", "da", "den", "der", "ten", "ter", "te", "sur", "af", "av"}
    return contains(prefixes, strings.ToLower(word))
}

func isNamePrefix(word string) bool {
    prefixes := []string{"Dr", "Dr.", "Rev", "Rev.", "Mr", "Mr.", "Mrs", "Mrs.", "Ms", "Ms.", "Lt", "Lt.", "Col", "Col.", "Capt", "Capt.", "Sgt", "Sgt.", "Prof", "Prof."}
    return contains(prefixes, word)
}

func isNameSuffix(word string) bool {
    suffixes := []string{"Jr", "Jr.", "Sr", "Sr.", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "Esq", "Esq.", "PhD", "MD", "DDS"}
    return contains(suffixes, word)
}
```

### Storing Parsed Name

```yaml
persons:
  person-example:
    properties:
      # Full formatted name
      name: "Lt. Cmndr. Joseph 'John' de Allen Jr."

      # Components
      name_prefix: "Lt. Cmndr."
      given_name: "Joseph"
      nickname: "John"
      surname_prefix: "de"
      surname: "Allen"
      name_suffix: "Jr."
```

### Multiple Names

GEDCOM allows multiple NAME tags per person:

```gedcom
1 NAME John /Doe/
2 TYPE birth
1 NAME John /Smith/
2 TYPE aka
1 NAME Juan /Doe/
2 TYPE immigrant
```

**Strategy**:
- First NAME without TYPE or with TYPE=birth → primary name
- Additional NAMEs → store in `alternative_names` or notes

---

## Evidence Chain Strategy

GEDCOM has simple source citations. GLX has rich evidence chains with Sources, Citations, and Assertions.

### GEDCOM Source Citation Structure

```gedcom
0 @S1@ SOUR
1 TITL Birth Certificate

0 @I1@ INDI
1 NAME John Smith
1 BIRT
2 DATE 1850
2 SOUR @S1@
3 PAGE Page 23
3 QUAY 2
3 TEXT "Born 25 Jan 1850"
```

### GLX Evidence Chain

```yaml
# 1. Source (original material)
sources:
  source-birth-cert-s1:
    title: "Birth Certificate"
    type: vital_record

# 2. Event (what happened)
events:
  event-birth-john-smith:
    type: birth
    date: "1850"
    participants:
      - person: person-john-smith
        role: principal

# 3. Citation (specific reference to source)
citations:
  citation-birth-john-s1:
    source: source-birth-cert-s1
    page: "Page 23"
    text_from_source: "Born 25 Jan 1850"
    quality: 2

# 4. Assertion (what we conclude from citation)
assertions:
  assertion-birth-date-john:
    subject: event-birth-john-smith
    claim: occurred_on
    value: "1850-01-25"
    confidence: medium  # Mapped from QUAY 2
    citations: [citation-birth-john-s1]
```

### QUAY to Confidence Mapping

| QUAY | Label | GLX Confidence | Notes |
|------|-------|----------------|-------|
| 0 | Unreliable | very_low | Evidence is questionable |
| 1 | Questionable | low | Evidence is weak |
| 2 | Secondary | medium | Secondary evidence or inference |
| 3 | Direct | high | Direct and primary evidence |

### Citation Creation Algorithm

```go
func createCitationAndAssertion(sour *GEDCOMRecord, subject interface{}, claim string, value interface{}, ctx *ConversionContext) {
    // Get source XRef
    sourceXRef := sour.Value // @S1@
    sourceID := ctx.SourceIDMap[sourceXRef]

    // Generate citation ID
    citationID := generateCitationID(subject, sourceID, ctx)

    // Extract citation details
    citation := &Citation{
        Source: sourceID,
    }

    for _, sub := range sour.SubRecords {
        switch sub.Tag {
        case "PAGE":
            citation.Page = sub.Value
        case "TEXT":
            citation.TextFromSource = sub.Value
        case "QUAY":
            citation.Quality = parseIntOrNil(sub.Value)
        case "NOTE":
            citation.Notes += sub.Value + "\n"
        case "OBJE":
            mediaID := ctx.MediaIDMap[sub.Value]
            citation.Media = append(citation.Media, mediaID)
        }
    }

    // Add citation to GLX
    ctx.GLX.Citations[citationID] = citation
    ctx.Stats.CitationsCreated++

    // Create assertion if we have a claim
    if claim != "" && value != nil {
        assertionID := generateAssertionID(subject, claim, ctx)

        // Map QUAY to confidence
        confidence := mapQUAYtoConfidence(citation.Quality)

        assertion := &Assertion{
            Subject:    getSubjectID(subject),
            Claim:      claim,
            Value:      value,
            Confidence: confidence,
            Citations:  []string{citationID},
        }

        ctx.GLX.Assertions[assertionID] = assertion
        ctx.Stats.AssertionsCreated++
    }
}

func mapQUAYtoConfidence(quay *int) string {
    if quay == nil {
        return "medium" // Default
    }

    switch *quay {
    case 0:
        return "very_low"
    case 1:
        return "low"
    case 2:
        return "medium"
    case 3:
        return "high"
    default:
        return "medium"
    }
}
```

---

## Extension Schema Handling (GEDCOM 7.0)

GEDCOM 7.0 allows custom tags with URI definitions:

```gedcom
0 HEAD
1 SCHMA
2 TAG _SKYPEID http://xmlns.com/foaf/0.1/skypeID
2 TAG _JABBERID http://xmlns.com/foaf/0.1/jabberID

0 @I1@ INDI
1 _SKYPEID john.doe.1850
1 _JABBERID john@example.com
```

### Parsing Strategy

1. **Extract schema** from HEAD.SCHMA
2. **Store mappings**: `_SKYPEID` → `http://xmlns.com/foaf/0.1/skypeID`
3. **When encountering custom tag**, check if defined in schema
4. **Store with context**:

```yaml
persons:
  person-john-doe:
    properties:
      # Option 1: Store with underscore prefix
      _skypeid: "john.doe.1850"
      _jabberid: "john@example.com"

      # Option 2: Store in extensions namespace
      extensions:
        "http://xmlns.com/foaf/0.1/skypeID": "john.doe.1850"
        "http://xmlns.com/foaf/0.1/jabberID": "john@example.com"
```

**Recommendation**: Use Option 1 (underscore prefix) for simplicity, but document the URI mapping in import report.

---

# Implementation Components

## File Organization

```
lib/
├── gedcom_import.go          # Main entry point, parser
├── gedcom_converter.go       # Conversion logic
├── gedcom_individual.go      # INDI → Person conversion
├── gedcom_family.go          # FAM → Relationship conversion
├── gedcom_source.go          # SOUR → Source conversion
├── gedcom_repository.go      # REPO → Repository conversion
├── gedcom_media.go           # OBJE → Media conversion
├── gedcom_place.go           # Place parsing and hierarchy
├── gedcom_date.go            # Date parsing
├── gedcom_name.go            # Name parsing
├── gedcom_util.go            # Utilities (ID generation, etc.)
├── gedcom_evidence.go        # Citation and assertion creation
├── gedcom_gedcom7.go         # GEDCOM 7.0 specific features
├── gedcom_import_test.go     # Main tests
├── gedcom_date_test.go       # Date parsing tests
├── gedcom_name_test.go       # Name parsing tests
└── gedcom_integration_test.go # Integration tests with real files
```

## Core Functions

### lib/gedcom_import.go

```go
// Main entry points
func ImportGEDCOMFromFile(filepath string) (*ImportResult, error)
func ImportGEDCOM(r io.Reader) (*ImportResult, error)

// Parser
func parseGEDCOM(r io.Reader) ([]*GEDCOMRecord, GEDCOMVersion, error)
func parseGEDCOMLine(text string, lineNum int) (*GEDCOMLine, error)
func buildRecords(lines []*GEDCOMLine) ([]*GEDCOMRecord, error)

// Version detection
func detectGEDCOMVersion(records []*GEDCOMRecord) GEDCOMVersion
```

### lib/gedcom_converter.go

```go
// Main conversion orchestration
func convertToGLX(records []*GEDCOMRecord, version GEDCOMVersion) (*ImportResult, error)

// Record dispatch
func processRecord(record *GEDCOMRecord, ctx *ConversionContext) error
```

### lib/gedcom_individual.go

```go
func convertIndividual(record *GEDCOMRecord, ctx *ConversionContext) error
func extractPersonProperties(record *GEDCOMRecord) map[string]interface{}
func createEventFromIndividualTag(tag string, subrecords []*GEDCOMRecord, personID string, ctx *ConversionContext) error
```

### lib/gedcom_family.go

```go
func convertFamily(record *GEDCOMRecord, ctx *ConversionContext) error
func createSpouseRelationship(husb, wife string, famRecord *GEDCOMRecord, ctx *ConversionContext) error
func createParentChildRelationships(parents []string, children []string, ctx *ConversionContext) error
func createFamilyEvent(tag string, subrecords []*GEDCOMRecord, participants []string, ctx *ConversionContext) error
```

### lib/gedcom_source.go

```go
func convertSource(record *GEDCOMRecord, ctx *ConversionContext) error
func inferSourceType(title, text string) string
```

### lib/gedcom_repository.go

```go
func convertRepository(record *GEDCOMRecord, ctx *ConversionContext) error
func inferRepositoryType(name string) string
```

### lib/gedcom_media.go

```go
func convertMedia(record *GEDCOMRecord, ctx *ConversionContext) error
func formatToMimeType(format string) string
```

### lib/gedcom_place.go

```go
func parseGEDCOMPlace(placeStr string, placeFormat string) (*PlaceHierarchy, error)
func buildPlaceHierarchy(hierarchy *PlaceHierarchy, ctx *ConversionContext) string
func createOrGetPlace(name, placeType, parentID string, coords *Coordinates, ctx *ConversionContext) string
func inferPlaceType(name string, level int) string
```

### lib/gedcom_date.go

```go
func parseGEDCOMDate(gedcomDate string, timeSuffix string, phrase string) (interface{}, map[string]interface{})
func parseExactDate(date string, timeSuffix string) string
func monthToNumber(month string) string
func parseGEDCOM7Time(timeStr string) string
```

### lib/gedcom_name.go

```go
func parseGEDCOMName(nameValue string, substructure *NameSubstructure) PersonName
func isSurnamePrefix(word string) bool
func isNamePrefix(word string) bool
func isNameSuffix(word string) bool
func formatFullName(name PersonName) string
```

### lib/gedcom_util.go

```go
func generatePersonID(gedcomXRef string, name string, ctx *ConversionContext) string
func generateEventID(eventType string, personID string, ctx *ConversionContext) string
func generateRelationshipID(relType string, participants []string, gedcomXRef string, ctx *ConversionContext) string
func generatePlaceID(placeName string, ctx *ConversionContext) string
func generateSourceID(gedcomXRef string, title string, ctx *ConversionContext) string
func generateCitationID(subjectID string, sourceID string, ctx *ConversionContext) string
func generateAssertionID(subjectID string, claim string, ctx *ConversionContext) string

func sanitizeForID(s string) string
func combineNotes(notes []string) string
func parseRestriction(resn string) string
func parseAge(ageStr string) string
```

### lib/gedcom_evidence.go

```go
func createCitationFromSOUR(sourRecord *GEDCOMRecord, subjectID string, ctx *ConversionContext) string
func createAssertion(subjectID string, claim string, value interface{}, citationIDs []string, quay *int, ctx *ConversionContext) error
func mapQUAYtoConfidence(quay *int) string
```

### lib/gedcom_gedcom7.go

```go
func extractSharedNotes(records []*GEDCOMRecord) map[string]*SharedNote
func resolveSharedNote(noteXRef string, ctx *ConversionContext) string
func parseExtensionSchema(head *GEDCOMRecord) map[string]string
func handlePhraseTag(phrase string, properties map[string]interface{}, fieldName string)
func handleNegativeAssertion(noTag string, subrecords []*GEDCOMRecord, subjectID string, ctx *ConversionContext) error
```

---

# Testing & Validation

## Test File Strategy

### Small Files (Quick validation)
- `minimal70.ged` (4 lines) - Absolute minimum
- `same-sex-marriage.ged` (15 lines) - Modern features
- `shakespeare.ged` (434 lines) - Small real family

### Medium Files (Real-world testing)
- `kennedy.ged` (1,426 lines) - Real family with variety
- `age-all.ged` (410 lines) - All age formats
- `maximal70.ged` (870 lines) - Full GEDCOM 7.0 spec

### Large Files (Stress testing)
- `british-royalty.ged` (3,733 lines) - Complex relationships
- `date-all.ged` (10,337 lines!) - Every date format
- `bullinger.ged` (17,862 lines) - Massive family tree

### Edge Cases
- `torture-test-551.ged` - GEDCOM 5.5.1 edge cases

## Unit Tests

```go
// lib/gedcom_date_test.go
func TestParseExactDates(t *testing.T)
func TestParseApproximateDates(t *testing.T)
func TestParseRelativeDates(t *testing.T)
func TestParseRangeDates(t *testing.T)
func TestParseGEDCOM7TimeDates(t *testing.T)

// lib/gedcom_name_test.go
func TestParseSimpleName(t *testing.T)
func TestParseNameWithPrefix(t *testing.T)
func TestParseNameWithSuffix(t *testing.T)
func TestParseNameWithSurnamePrefix(t *testing.T)
func TestParseNameWithNickname(t *testing.T)
func TestParseComplexName(t *testing.T)

// lib/gedcom_place_test.go
func TestParsePlaceSimple(t *testing.T)
func TestParsePlaceHierarchy(t *testing.T)
func TestBuildPlaceHierarchy(t *testing.T)

// lib/gedcom_util_test.go
func TestGeneratePersonID(t *testing.T)
func TestGenerateEventID(t *testing.T)
func TestSanitizeForID(t *testing.T)
```

## Integration Tests

```go
// lib/gedcom_integration_test.go

func TestImportMinimal70(t *testing.T) {
    result, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/7.0/minimal-valid/minimal70.ged")
    require.NoError(t, err)
    assert.NotNil(t, result.GLX)
    assert.Equal(t, 0, len(result.Errors))
}

func TestImportShakespeare(t *testing.T) {
    result, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged")
    require.NoError(t, err)

    // Verify persons
    assert.Greater(t, len(result.GLX.Persons), 10)

    // Verify William Shakespeare exists
    var williamFound bool
    for id, person := range result.GLX.Persons {
        if strings.Contains(id, "william-shakespeare") {
            williamFound = true
            assert.Contains(t, person.Properties, "given_name")
            assert.Equal(t, "William", person.Properties["given_name"])
            assert.Equal(t, "Shakespeare", person.Properties["surname"])
        }
    }
    assert.True(t, williamFound)

    // Verify events created
    assert.Greater(t, len(result.GLX.Events), 20)

    // Verify relationships
    assert.Greater(t, len(result.GLX.Relationships), 10)
}

func TestImportKennedy(t *testing.T) {
    result, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/5.5.1/kennedy-family/kennedy.ged")
    require.NoError(t, err)

    // More complex validation
    assert.Greater(t, len(result.GLX.Persons), 30)
    assert.Greater(t, len(result.GLX.Events), 50)

    // Check statistics
    assert.Greater(t, result.Stats.PersonsImported, 30)
    assert.Greater(t, result.Stats.EventsCreated, 50)
}

func TestImportMaximal70(t *testing.T) {
    result, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/7.0/comprehensive-spec/maximal70.ged")
    require.NoError(t, err)

    // Verify GEDCOM 7.0 features
    // - Shared notes resolved
    // - Extension schema parsed
    // - PHRASE tags handled
    // - TIME values combined with dates
    // - NO tags → negative assertions
}

func TestImportBullinger(t *testing.T) {
    // Stress test with very large file
    result, err := ImportGEDCOMFromFile("../glx/testdata/gedcom/5.5.1/bullinger-family/bullinger.ged")
    require.NoError(t, err)

    // Performance check
    assert.Less(t, result.Stats.ProcessingTime, 30*time.Second)

    // Memory check (if profiling enabled)
    // assert.Less(t, result.Stats.PeakMemoryMB, 500)
}
```

## Validation Tests

After import, validate the GLX archive:

```go
func TestImportProducesValidGLX(t *testing.T) {
    files := []string{
        "../glx/testdata/gedcom/5.5.1/shakespeare-family/shakespeare.ged",
        "../glx/testdata/gedcom/5.5.1/kennedy-family/kennedy.ged",
        "../glx/testdata/gedcom/7.0/minimal-valid/minimal70.ged",
        "../glx/testdata/gedcom/7.0/same-sex-marriage/same-sex-marriage.ged",
    }

    for _, file := range files {
        t.Run(file, func(t *testing.T) {
            result, err := ImportGEDCOMFromFile(file)
            require.NoError(t, err)

            // Validate the GLX
            validationResult := result.GLX.Validate()
            assert.Equal(t, 0, len(validationResult.Errors), "Should have no validation errors")
        })
    }
}
```

---

# Performance & Optimization

## Memory Management

### Streaming Parser

Don't load entire file into memory:

```go
func parseGEDCOM(r io.Reader) ([]*GEDCOMRecord, GEDCOMVersion, error) {
    scanner := bufio.NewScanner(r)
    scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

    lines := []*GEDCOMLine{}
    lineNum := 0

    for scanner.Scan() {
        lineNum++
        line, err := parseGEDCOMLine(scanner.Text(), lineNum)
        if err != nil {
            return nil, "", err
        }
        lines = append(lines, line)
    }

    // Build records incrementally
    records, err := buildRecords(lines)
    return records, detectVersion(records), err
}
```

### ID Map Cleanup

After conversion, clear large maps:

```go
func convertToGLX(records []*GEDCOMRecord, version GEDCOMVersion) (*ImportResult, error) {
    ctx := NewConversionContext(version)

    // ... conversion ...

    // Clear large maps to free memory
    ctx.PersonIDMap = nil
    ctx.FamilyIDMap = nil
    // ... etc

    return result, nil
}
```

### Place Hierarchy Optimization

Reuse places efficiently:

```go
func createOrGetPlace(name string, placeType string, parentID string, coords *Coordinates, ctx *ConversionContext) string {
    // Check if place already exists
    key := buildPlaceKey(name, parentID)
    if existingID, exists := ctx.PlaceIDMap[key]; exists {
        return existingID
    }

    // Create new place
    placeID := generatePlaceID(name, ctx)
    // ... create place ...

    ctx.PlaceIDMap[key] = placeID
    return placeID
}
```

## Performance Targets

| File Size | Lines | Target Time | Memory Limit |
|-----------|-------|-------------|--------------|
| Tiny | < 100 | < 100ms | < 10 MB |
| Small | 100-1K | < 1s | < 50 MB |
| Medium | 1K-5K | < 5s | < 100 MB |
| Large | 5K-10K | < 15s | < 250 MB |
| Very Large | 10K-20K | < 30s | < 500 MB |
| Huge | 20K+ | < 2 min | < 1 GB |

## Profiling

```bash
# CPU profiling
go test -bench=BenchmarkImportLarge -cpuprofile=cpu.prof
go tool pprof cpu.prof

# Memory profiling
go test -bench=BenchmarkImportLarge -memprofile=mem.prof
go tool pprof mem.prof

# Benchmarks
go test -bench=. -benchmem
```

---

# Error Handling & Reporting

## Error Categories

### Fatal Errors (Stop Import)
- Malformed GEDCOM structure
- Unsupported version (< 5.5)
- File encoding errors
- Circular place references

### Conversion Errors (Skip Item)
- Invalid XRef (dangling reference)
- Unparseable date
- Missing required field

### Warnings (Continue)
- Unknown custom tags
- Unusual data (birth after death)
- Missing optional fields

## Import Report

```
================================================================================
GEDCOM Import Report
================================================================================

File: kennedy.ged
GEDCOM Version: 5.5.1
Encoding: UTF-8

--------------------------------------------------------------------------------
IMPORT STATISTICS
--------------------------------------------------------------------------------

Records Processed:
  - Individuals (INDI):        45
  - Families (FAM):            23
  - Sources (SOUR):            8
  - Repositories (REPO):       3
  - Multimedia (OBJE):         12
  - Total Records:             91
  - Total Lines:               1,426

Entities Created:
  - Persons:                   45
  - Relationships:             67  (23 spouse + 44 parent-child)
  - Events:                    156
  - Places:                    12
  - Sources:                   8
  - Citations:                 34
  - Repositories:              3
  - Media:                     12
  - Assertions:                34

Event Types:
  - birth:                     45
  - death:                     38
  - marriage:                  23
  - burial:                    22
  - christening:               12
  - census:                    8
  - other:                     8

Processing Time: 2.3 seconds
Peak Memory: 48 MB

--------------------------------------------------------------------------------
WARNINGS (3)
--------------------------------------------------------------------------------

  Line 234: Unknown custom tag _MSTAT on person @I12@ (John F. Kennedy)
    └─ Stored in properties._mstat

  Line 567: Date format unusual: "Abt. 1850s" for birth of @I34@
    └─ Stored as-is, may need manual review

  Line 890: Birth date (1970) after death date (1963) for person @I45@
    └─ Data imported but flagged for review

--------------------------------------------------------------------------------
ERRORS (0)
--------------------------------------------------------------------------------

  No errors encountered.

--------------------------------------------------------------------------------
GEDCOM-SPECIFIC NOTES
--------------------------------------------------------------------------------

  • Custom tags preserved:
    - _UID → properties.uid
    - _TITL → properties.gedcom_title
    - _MSTAT → properties._mstat

  • Place hierarchy created:
    - 12 places in 3-level hierarchy (city → state → country)

  • QUAY values mapped to confidence levels:
    - QUAY 0 (0 citations) → very_low
    - QUAY 1 (5 citations) → low
    - QUAY 2 (20 citations) → medium
    - QUAY 3 (9 citations) → high

================================================================================
OUTPUT
================================================================================

GLX Archive: kennedy.glx
Status: ✓ Valid (passes schema validation)

Next Steps:
  1. Review warnings above
  2. Validate: glx validate kennedy.glx
  3. Enhance evidence chains as needed

================================================================================
```

---

**End of Complete Implementation Plan**

This plan covers 100% of GEDCOM 5.5.1 and 7.0 specifications with complete vocabulary and property additions, comprehensive mappings, and detailed implementation strategies. The plan is unified and ready for implementation without phasing.

---

# Complete Conversion Examples

This section provides full, worked examples of converting complex GEDCOM structures to GLX.

## Example 1: Complete Individual with Events and Citations

### Input GEDCOM

```gedcom
0 @I1@ INDI
1 NAME John Fitzgerald "Jack" /Kennedy/
2 GIVN John Fitzgerald
2 NICK Jack
2 SURN Kennedy
1 SEX M
1 BIRT
2 DATE 29 MAY 1917
2 PLAC Brookline, Norfolk County, Massachusetts, USA
3 MAP
4 LATI N42.3317
4 LONG W71.1211
2 SOUR @S1@
3 PAGE Page 45, Entry 23
3 QUAY 3
3 TEXT Born May 29, 1917 at 83 Beals Street
1 DEAT
2 DATE 22 NOV 1963
2 PLAC Dallas, Dallas County, Texas, USA
2 CAUS Assassination
2 SOUR @S2@
3 PAGE Death Certificate #63-11796
3 QUAY 3
1 OCCU President of the United States
2 DATE FROM 20 JAN 1961 TO 22 NOV 1963
1 NOTE John Fitzgerald Kennedy was the 35th President of the United States.
1 FAMC @F1@
1 FAMS @F2@
```

### Output GLX

```yaml
persons:
  person-john-fitzgerald-kennedy-i1:
    properties:
      name: "John Fitzgerald Kennedy"
      given_name: "John Fitzgerald"
      nickname: "Jack"
      surname: "Kennedy"
      sex: "M"
      occupation:
        - value: "President of the United States"
          date: "1961-01-20/1963-11-22"
    notes: |
      John Fitzgerald Kennedy was the 35th President of the United States.

places:
  place-usa:
    name: "USA"
    type: country

  place-massachusetts:
    name: "Massachusetts"
    type: state
    parent: place-usa

  place-norfolk-county-ma:
    name: "Norfolk County"
    type: county
    parent: place-massachusetts

  place-brookline-ma-usa:
    name: "Brookline"
    type: city
    parent: place-norfolk-county-ma
    latitude: 42.3317
    longitude: -71.1211

  place-texas:
    name: "Texas"
    type: state
    parent: place-usa

  place-dallas-county-tx:
    name: "Dallas County"
    type: county
    parent: place-texas

  place-dallas-tx-usa:
    name: "Dallas"
    type: city
    parent: place-dallas-county-tx

events:
  event-birth-john-fitzgerald-kennedy-i1:
    type: birth
    date: "1917-05-29"
    place: place-brookline-ma-usa
    participants:
      - person: person-john-fitzgerald-kennedy-i1
        role: principal

  event-death-john-fitzgerald-kennedy-i1:
    type: death
    date: "1963-11-22"
    place: place-dallas-tx-usa
    participants:
      - person: person-john-fitzgerald-kennedy-i1
        role: principal
    properties:
      cause: "Assassination"

sources:
  source-s1:
    title: "Massachusetts Birth Records"
    type: vital_record

  source-s2:
    title: "Texas Death Certificate"
    type: vital_record

citations:
  citation-birth-kennedy-s1-1:
    source: source-s1
    page: "Page 45, Entry 23"
    text_from_source: "Born May 29, 1917 at 83 Beals Street"
    quality: 3

  citation-death-kennedy-s2-1:
    source: source-s2
    page: "Death Certificate #63-11796"
    quality: 3

assertions:
  assertion-birth-date-kennedy:
    subject: event-birth-john-fitzgerald-kennedy-i1
    claim: occurred_on
    value: "1917-05-29"
    confidence: high
    citations: [citation-birth-kennedy-s1-1]

  assertion-birth-place-kennedy:
    subject: event-birth-john-fitzgerald-kennedy-i1
    claim: occurred_at
    value: place-brookline-ma-usa
    confidence: high
    citations: [citation-birth-kennedy-s1-1]

  assertion-death-date-kennedy:
    subject: event-death-john-fitzgerald-kennedy-i1
    claim: occurred_on
    value: "1963-11-22"
    confidence: high
    citations: [citation-death-kennedy-s2-1]

  assertion-death-place-kennedy:
    subject: event-death-john-fitzgerald-kennedy-i1
    claim: occurred_at
    value: place-dallas-tx-usa
    confidence: high
    citations: [citation-death-kennedy-s2-1]

# Relationships will be created when processing FAM records @F1@ and @F2@
```

---

## Example 2: Family with Multiple Children and Events

### Input GEDCOM

```gedcom
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
1 CHIL @I5@
1 MARR
2 DATE 12 SEP 1953
2 PLAC Newport, Rhode Island, USA
2 SOUR @S3@
3 PAGE Marriage License #1953-456
3 QUAY 3
1 NOTE This was a highly publicized society wedding.
1 NCHI 3
```

### Output GLX

```yaml
relationships:
  # Spouse relationship
  relationship-marriage-john-jacqueline-f1:
    type: marriage
    participants:
      - person: person-john-kennedy-i1
        role: partner
      - person: person-jacqueline-bouvier-i2
        role: partner
    start_event: event-marriage-john-jacqueline-f1
    properties:
      num_children: 3
    notes: |
      This was a highly publicized society wedding.

  # Parent-child relationship 1
  relationship-parent-child-kennedy-i3-f1:
    type: parent-child
    participants:
      - person: person-john-kennedy-i1
        role: parent
      - person: person-jacqueline-bouvier-i2
        role: parent
      - person: person-caroline-kennedy-i3
        role: child

  # Parent-child relationship 2
  relationship-parent-child-kennedy-i4-f1:
    type: parent-child
    participants:
      - person: person-john-kennedy-i1
        role: parent
      - person: person-jacqueline-bouvier-i2
        role: parent
      - person: person-john-kennedy-jr-i4
        role: child

  # Parent-child relationship 3
  relationship-parent-child-kennedy-i5-f1:
    type: parent-child
    participants:
      - person: person-john-kennedy-i1
        role: parent
      - person: person-jacqueline-bouvier-i2
        role: parent
      - person: person-patrick-kennedy-i5
        role: child

events:
  event-marriage-john-jacqueline-f1:
    type: marriage
    date: "1953-09-12"
    place: place-newport-ri-usa
    participants:
      - person: person-john-kennedy-i1
        role: groom
      - person: person-jacqueline-bouvier-i2
        role: bride

sources:
  source-s3:
    title: "Rhode Island Marriage Records"
    type: vital_record

citations:
  citation-marriage-kennedy-s3-1:
    source: source-s3
    page: "Marriage License #1953-456"
    quality: 3

assertions:
  assertion-marriage-date-kennedy:
    subject: event-marriage-john-jacqueline-f1
    claim: occurred_on
    value: "1953-09-12"
    confidence: high
    citations: [citation-marriage-kennedy-s3-1]
```

---

## Example 3: GEDCOM 7.0 with Advanced Features

### Input GEDCOM

```gedcom
0 HEAD
1 GEDC
2 VERS 7.0
1 SCHMA
2 TAG _TWITTER http://xmlns.com/foaf/0.1/account
1 NOTE This is a test file for GEDCOM 7.0 features.
2 LANG en-US
2 TRAN Este es un archivo de prueba para GEDCOM 7.0.
3 LANG es

0 @N1@ SNOTE This person was a notable historical figure.
1 MIME text/plain
1 LANG en-US

0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
2 PHRASE Non-binary, assigned male at birth
1 BIRT
2 DATE 15 JAN 1850
3 TIME 14:30:00
3 PHRASE Afternoon
2 PLAC London, England
1 NO DEAT
2 DATE FROM 1900 TO 1950
3 PHRASE No death record found in this period
1 SNOTE @N1@
1 _TWITTER john.smith.1850
1 UID a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

### Output GLX

```yaml
persons:
  person-john-smith-i1:
    properties:
      name: "John Smith"
      given_name: "John"
      surname: "Smith"
      sex: "M"
      sex_phrase: "Non-binary, assigned male at birth"
      uid: ["a1b2c3d4-e5f6-7890-abcd-ef1234567890"]
      _twitter: "john.smith.1850"
    notes: |
      This person was a notable historical figure.

events:
  event-birth-john-smith-i1:
    type: birth
    date: "1850-01-15T14:30:00"
    place: place-london-england
    participants:
      - person: person-john-smith-i1
        role: principal
    properties:
      date_phrase: "Afternoon"

assertions:
  assertion-no-death-john-smith:
    subject: person-john-smith-i1
    claim: death_occurred
    value: "false"
    confidence: high
    properties:
      date_range: "1900/1950"
      phrase: "No death record found in this period"
```

---

# Edge Cases and Special Handling

## Circular References

### Problem

```gedcom
0 @P1@ PLAC
1 NAME City A
1 PARENT @P2@

0 @P2@ PLAC
1 NAME City B
1 PARENT @P1@
```

### Detection and Handling

```go
func buildPlaceHierarchy(placeID string, visited map[string]bool, ctx *ConversionContext) error {
    if visited[placeID] {
        return fmt.Errorf("circular reference detected in place hierarchy: %s", placeID)
    }

    visited[placeID] = true

    place := ctx.GLX.Places[placeID]
    if place.Parent != "" {
        if err := buildPlaceHierarchy(place.Parent, visited, ctx); err != nil {
            return err
        }
    }

    delete(visited, placeID)
    return nil
}
```

## Missing XRefs (Dangling References)

### Problem

```gedcom
0 @I1@ INDI
1 NAME John /Smith/
1 FAMC @F999@  # Family @F999@ doesn't exist
```

### Handling

```go
func resolvePersonReference(xref string, ctx *ConversionContext) (string, error) {
    personID, exists := ctx.PersonIDMap[xref]
    if !exists {
        // Log warning, create placeholder, or skip
        ctx.Warnings = append(ctx.Warnings, ImportWarning{
            Record:  xref,
            Message: fmt.Sprintf("Reference to non-existent person: %s", xref),
        })
        return "", fmt.Errorf("dangling reference: %s", xref)
    }
    return personID, nil
}
```

**Strategy**: Log warning and skip the reference, or create a placeholder entity.

## Ambiguous Dates

### Problem

Date strings that are unclear or malformed:

```gedcom
2 DATE Abt. 1850s
2 DATE Between 1840 and 1850, probably
2 DATE Unknown
2 DATE ????
```

### Handling

```go
func parseAmbiguousDate(dateStr string) (interface{}, map[string]interface{}) {
    properties := make(map[string]interface{})

    // Store original if uncertain
    if strings.Contains(strings.ToLower(dateStr), "unknown") ||
       strings.Contains(dateStr, "?") {
        properties["date_uncertain"] = true
        properties["date_original"] = dateStr
        return nil, properties
    }

    // Try to extract best guess
    if strings.Contains(strings.ToLower(dateStr), "between") {
        // Try to extract range
        // ... parsing logic ...
    }

    // If all else fails, store original and mark for review
    properties["date_parse_failed"] = true
    properties["date_original"] = dateStr
    ctx.Warnings = append(ctx.Warnings, ImportWarning{
        Message: fmt.Sprintf("Could not parse date: %s", dateStr),
    })

    return dateStr, properties
}
```

## Name Variations and Character Encoding

### Problem

Names with special characters, diacritics, or unusual formatting:

```gedcom
1 NAME François /Le Beau/
1 NAME Владимир /Иванов/
1 NAME 田中 /太郎/
1 NAME /Smith/, John  # Reversed format
```

### Handling

```go
func parseGEDCOMName(nameValue string, substructure *NameSubstructure) PersonName {
    // Handle UTF-8 properly
    nameValue = strings.TrimSpace(nameValue)

    // Handle reversed format (Surname, Given)
    if strings.Contains(nameValue, ",") && !strings.Contains(nameValue, "/") {
        parts := strings.Split(nameValue, ",")
        if len(parts) == 2 {
            return PersonName{
                Surname:   strings.TrimSpace(parts[0]),
                GivenName: strings.TrimSpace(parts[1]),
            }
        }
    }

    // Normal parsing...
    // Preserve all Unicode characters
}
```

## Multiple Values for Same Tag

### Problem

```gedcom
1 OCCU Farmer
1 OCCU Blacksmith
1 OCCU Mayor
```

### Handling

Store as temporal property with all values:

```yaml
persons:
  person-example:
    properties:
      occupation:
        - value: "Farmer"
        - value: "Blacksmith"
        - value: "Mayor"
```

Or if dates available:

```yaml
      occupation:
        - value: "Farmer"
          date: "1840/1850"
        - value: "Blacksmith"
          date: "1850/1860"
        - value: "Mayor"
          date: "1860/1870"
```

## Empty or Null Values

### Problem

```gedcom
1 BIRT
1 NAME //  # Empty name
1 DEAT Y  # Just "Y" (yes)
```

### Handling

```go
func handleEmptyEvent(tag string, value string, subrecords []*GEDCOMRecord) bool {
    // If value is "Y" or empty but subrecords exist, event occurred but details unknown
    if value == "Y" || (value == "" && len(subrecords) > 0) {
        return true // Create event with unknown details
    }

    // If completely empty, skip
    if value == "" && len(subrecords) == 0 {
        return false
    }

    return true
}
```

For empty name:

```yaml
persons:
  person-unknown-i1:
    properties:
      name: "[Unknown]"
      given_name: "[Unknown]"
```

## Very Long Notes

### Problem

Notes can be extremely long (thousands of characters):

```gedcom
1 NOTE This is a very long biographical note...
2 CONT ... continued for many lines ...
2 CONT ... and many more lines ...
# ... 100+ CONT lines ...
```

### Handling

No special handling needed - combine all CONT lines:

```go
func combineNotes(notes []string) string {
    return strings.Join(notes, "\n")
}
```

GLX has no note length limit.

## Duplicate Records

### Problem

Same person appears twice with different IDs:

```gedcom
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1850

0 @I99@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 1850
```

### Handling

**Detection**: Not automatic - requires fuzzy matching (out of scope for initial import)

**Strategy**: Import both as separate entities. User can merge later using GLX merge tools.

**Warning**: Could add warning if exact name and birth date match.

## LDS Ordinance Data

### Problem

LDS-specific tags that may not be relevant to all users:

```gedcom
1 BAPL
2 DATE 1 JAN 2000
2 TEMP LOGAN
2 STAT COMPLETED
```

### Handling

Import as events:

```yaml
events:
  event-lds-baptism-i1-1:
    type: lds_baptism
    date: "2000-01-01"
    properties:
      lds_temple: "LOGAN"
      lds_status: "COMPLETED"
```

**Note**: Users who don't need LDS data can filter by event type.

## Custom Tags Without Schema

### Problem (GEDCOM 5.5.1)

```gedcom
1 _CUSTOM Some custom value
1 _ANOTHER Another value
```

### Handling

Store with underscore prefix:

```yaml
persons:
  person-example:
    properties:
      _custom: "Some custom value"
      _another: "Another value"
```

**Warning**: Log which custom tags were encountered in import report.

---

# Implementation Checklist

This comprehensive checklist covers every aspect of the implementation.

## 1. Setup and Infrastructure

- [ ] Create `lib/gedcom_*.go` files structure
- [ ] Set up test file structure
- [ ] Add vocabulary files to `specification/5-standard-vocabularies/`
- [ ] Create `docs/gedcom-import-complete-plan.md` (this document)
- [ ] Set up benchmarking infrastructure
- [ ] Configure profiling tools

## 2. Vocabulary Additions

### Event Types

- [ ] Add `adult_christening` to `event-types.glx`
- [ ] Add `first_communion` to `event-types.glx`
- [ ] Add `ordination` to `event-types.glx`
- [ ] Add `blessing` to `event-types.glx`
- [ ] Add `lds_baptism` to `event-types.glx`
- [ ] Add `lds_confirmation` to `event-types.glx`
- [ ] Add `lds_endowment` to `event-types.glx`
- [ ] Add `lds_sealing_child` to `event-types.glx`
- [ ] Add `lds_sealing_spouse` to `event-types.glx`
- [ ] Add `emigration` to `event-types.glx`
- [ ] Add `immigration` to `event-types.glx`
- [ ] Add `naturalization` to `event-types.glx`
- [ ] Add `graduation` to `event-types.glx`
- [ ] Add `retirement` to `event-types.glx`
- [ ] Add `census` to `event-types.glx`
- [ ] Add `probate` to `event-types.glx`
- [ ] Add `will` to `event-types.glx`
- [ ] Add `annulment` to `event-types.glx`
- [ ] Add `marriage_banns` to `event-types.glx`
- [ ] Add `marriage_contract` to `event-types.glx`
- [ ] Add `marriage_license` to `event-types.glx`
- [ ] Add `marriage_settlement` to `event-types.glx`
- [ ] Add `divorce_filed` to `event-types.glx`

### Participant Roles

- [ ] Add `partner` to `participant-roles.glx`
- [ ] Add `clergy` to `participant-roles.glx`
- [ ] Add `godparent` to `participant-roles.glx`
- [ ] Add `guardian` to `participant-roles.glx`
- [ ] Add `executor` to `participant-roles.glx`
- [ ] Add `other` to `participant-roles.glx`

### Person Properties

- [ ] Add `name_prefix` to `person-properties.glx`
- [ ] Add `surname_prefix` to `person-properties.glx`
- [ ] Add `name_suffix` to `person-properties.glx`
- [ ] Add `nickname` to `person-properties.glx`
- [ ] Add `alias` to `person-properties.glx`
- [ ] Add `name` to `person-properties.glx`
- [ ] Add `sex` to `person-properties.glx`
- [ ] Add `caste` to `person-properties.glx`
- [ ] Add `description` to `person-properties.glx`
- [ ] Add `title` to `person-properties.glx`
- [ ] Add `id_number` to `person-properties.glx`
- [ ] Add `ssn` to `person-properties.glx`
- [ ] Add `num_children` to `person-properties.glx`
- [ ] Add `num_marriages` to `person-properties.glx`
- [ ] Add `possessions` to `person-properties.glx`
- [ ] Add `record_file_number` to `person-properties.glx`
- [ ] Add `ancestral_file_number` to `person-properties.glx`
- [ ] Add `reference_number` to `person-properties.glx`
- [ ] Add `record_id` to `person-properties.glx`
- [ ] Add `uid` to `person-properties.glx`
- [ ] Add `external_ids` to `person-properties.glx`
- [ ] Add `lds_temple` to `person-properties.glx`
- [ ] Add `lds_status` to `person-properties.glx`

### Event Properties

- [ ] Create `event-properties.glx` if not exists
- [ ] Add `address` to `event-properties.glx`
- [ ] Add `phone` to `event-properties.glx`
- [ ] Add `email` to `event-properties.glx`
- [ ] Add `fax` to `event-properties.glx`
- [ ] Add `website` to `event-properties.glx`
- [ ] Add `agency` to `event-properties.glx`
- [ ] Add `religion` to `event-properties.glx`
- [ ] Add `cause` to `event-properties.glx`
- [ ] Add `restriction` to `event-properties.glx`
- [ ] Add `sort_date` to `event-properties.glx`
- [ ] Add `event_type` to `event-properties.glx`
- [ ] Add `entry_date` to `event-properties.glx`
- [ ] Add `age` to `event-properties.glx`
- [ ] Add `lds_temple` to `event-properties.glx`
- [ ] Add `lds_status` to `event-properties.glx`
- [ ] Add `uid` to `event-properties.glx`

### Relationship Properties

- [ ] Add `num_children` to `relationship-properties.glx`
- [ ] Add `reference_number` to `relationship-properties.glx`
- [ ] Add `record_id` to `relationship-properties.glx`
- [ ] Add `uid` to `relationship-properties.glx`
- [ ] Add `restriction` to `relationship-properties.glx`

### Other Properties

- [ ] Create `source-properties.glx`
- [ ] Add source properties (abbreviation, reference_number, record_id, uid, copyright)
- [ ] Create `repository-properties.glx`
- [ ] Add repository properties (fax, reference_number, record_id, uid)
- [ ] Create `media-properties.glx`
- [ ] Add media properties (format, crop_*, reference_number, record_id, uid)
- [ ] Create `citation-properties.glx`
- [ ] Add citation properties (event_type, role, entry_date)
- [ ] Update `place-properties.glx` with additional properties

## 3. Core Parser Implementation

- [ ] Implement `parseGEDCOMLine()` with full tag support
- [ ] Implement `buildRecords()` with hierarchical structure
- [ ] Implement version detection (5.5.1 vs 7.0)
- [ ] Implement character encoding detection (UTF-8, ANSEL, etc.)
- [ ] Handle CONT/CONC tags for line continuation
- [ ] Implement error recovery for malformed lines
- [ ] Add line number tracking for error reporting

## 4. Conversion Context

- [ ] Implement `ConversionContext` struct
- [ ] Implement ID mapping (XRef → GLX ID) for all entity types
- [ ] Implement shared notes map (GEDCOM 7.0)
- [ ] Implement extension schema map (GEDCOM 7.0)
- [ ] Implement place hierarchy tracker
- [ ] Implement deferred processing queue
- [ ] Implement error/warning collectors
- [ ] Implement statistics tracker

## 5. Individual (INDI) Conversion

- [ ] Parse NAME tag with `/surname/` notation
- [ ] Parse NAME substructure (NPFX, GIVN, NICK, SPFX, SURN, NSFX)
- [ ] Handle multiple NAME tags
- [ ] Extract SEX with PHRASE (GEDCOM 7.0)
- [ ] Convert BIRT to birth event
- [ ] Convert DEAT to death event
- [ ] Convert all individual event tags (30+ types)
- [ ] Handle CAST, DSCR, EDUC, IDNO, NATI, NCHI, NMR, OCCU, PROP, RELI, SSN, TITL, FACT
- [ ] Store temporal properties (OCCU, EDUC, RELI, RESI, TITL)
- [ ] Handle ASSO associations
- [ ] Handle ALIA aliases
- [ ] Handle REFN, RIN, AFN, RFN
- [ ] Handle UID (can be multiple)
- [ ] Handle EXID (GEDCOM 7.0)
- [ ] Handle LDS ordinances (BAPL, CONL, ENDL, SLGC)
- [ ] Handle NO tag for negative assertions (GEDCOM 7.0)
- [ ] Defer FAMC/FAMS processing until families loaded
- [ ] Create citations from embedded SOUR
- [ ] Create assertions from citations
- [ ] Handle NOTE tags (combine multiple)
- [ ] Handle OBJE media references
- [ ] Handle CHAN change dates (optional)

## 6. Family (FAM) Conversion

- [ ] Extract HUSB and WIFE participants
- [ ] Handle HUSB/WIFE PHRASE (GEDCOM 7.0)
- [ ] Create spouse relationship
- [ ] Create parent-child relationships (one per CHIL)
- [ ] Handle CHIL PHRASE (GEDCOM 7.0)
- [ ] Convert MARR to marriage event
- [ ] Convert DIV to divorce event
- [ ] Convert all family event tags (ANUL, ENGA, MARB, MARC, MARL, MARS, DIVF, CENS, RESI, EVEN)
- [ ] Handle SLGS (LDS sealing spouse)
- [ ] Handle NO tag for negative assertions (GEDCOM 7.0)
- [ ] Handle NCHI (number of children)
- [ ] Create citations from embedded SOUR
- [ ] Handle NOTE tags
- [ ] Handle OBJE media references
- [ ] Handle REFN, RIN, UID
- [ ] Handle RESN restriction
- [ ] Handle CHAN change dates (optional)

## 7. Source (SOUR) Conversion

- [ ] Extract TITL
- [ ] Extract AUTH (split into array if multiple)
- [ ] Extract PUBL
- [ ] Extract DATE
- [ ] Extract TEXT → description
- [ ] Extract ABBR → properties.abbreviation
- [ ] Link REPO
- [ ] Handle NOTE
- [ ] Handle OBJE media
- [ ] Handle REFN, RIN, UID
- [ ] Handle COPR copyright
- [ ] Infer source type from title/content
- [ ] Handle CHAN (optional)

## 8. Repository (REPO) Conversion

- [ ] Extract NAME
- [ ] Extract ADDR (or combine ADR1/ADR2/ADR3)
- [ ] Extract CITY, STAE, POST, CTRY
- [ ] Extract PHON, EMAIL, FAX, WWW
- [ ] Handle NOTE
- [ ] Handle REFN, RIN, UID
- [ ] Infer repository type from name
- [ ] Handle CHAN (optional)

## 9. Media (OBJE) Conversion

- [ ] Extract FILE → uri
- [ ] Convert FORM to mime_type
- [ ] Extract TITL
- [ ] Handle MIME (GEDCOM 7.0)
- [ ] Handle CROP (GEDCOM 7.0)
- [ ] Handle NOTE
- [ ] Handle REFN, RIN, UID
- [ ] Handle CHAN (optional)

## 10. Place Parsing

- [ ] Extract place hierarchy format from HEAD.PLAC.FORM
- [ ] Split place names by comma
- [ ] Infer place types from hierarchy level
- [ ] Create place hierarchy from general to specific
- [ ] Link places with parent references
- [ ] Extract MAP.LATI and MAP.LONG
- [ ] Handle TRAN translations (GEDCOM 7.0)
- [ ] Reuse existing places (check PlaceIDMap)
- [ ] Handle NOTE on places
- [ ] Handle UID on places

## 11. Date Parsing

- [ ] Parse year-only dates (`1850`)
- [ ] Parse month-year dates (`JAN 1850`)
- [ ] Parse full dates (`25 JAN 1850`)
- [ ] Convert month names to numbers
- [ ] Handle ABT (about) dates
- [ ] Handle CAL (calculated) dates
- [ ] Handle EST (estimated) dates
- [ ] Handle BEF (before) dates
- [ ] Handle AFT (after) dates
- [ ] Handle BET...AND (between) dates
- [ ] Handle FROM...TO dates
- [ ] Handle TIME (GEDCOM 7.0)
- [ ] Handle PHRASE on dates (GEDCOM 7.0)
- [ ] Handle SDATE sort dates (GEDCOM 7.0)
- [ ] Handle ambiguous/malformed dates
- [ ] Store date qualifiers in properties
- [ ] Format output as ISO 8601

## 12. Name Parsing

- [ ] Extract surname from `/surname/` notation
- [ ] Extract surname prefix (de, von, van, etc.)
- [ ] Extract given name(s)
- [ ] Extract nickname from quotes
- [ ] Extract prefix (Dr., Rev., etc.)
- [ ] Extract suffix (Jr., Sr., III, etc.)
- [ ] Use substructure if available (NPFX, GIVN, NICK, SPFX, SURN, NSFX)
- [ ] Handle reversed format (Surname, Given)
- [ ] Handle UTF-8 and special characters
- [ ] Handle multiple NAME tags
- [ ] Handle NAME.TYPE (birth, aka, immigrant, etc.)
- [ ] Handle TRAN translations (GEDCOM 7.0)
- [ ] Format full name for display

## 13. Evidence Chain Creation

- [ ] Create Citation from SOUR tag
- [ ] Extract PAGE
- [ ] Extract TEXT
- [ ] Extract QUAY
- [ ] Extract DATA.DATE (entry date)
- [ ] Link to Source
- [ ] Link OBJE media to citation
- [ ] Handle NOTE on citation
- [ ] Create Assertion from Citation
- [ ] Map QUAY to confidence level
- [ ] Link assertion to subject
- [ ] Define claim type
- [ ] Extract value
- [ ] Link citation to assertion

## 14. GEDCOM 7.0 Features

- [ ] Parse SCHMA extension definitions
- [ ] Collect SNOTE shared notes
- [ ] Resolve SNOTE references
- [ ] Handle TRAN translations on various tags
- [ ] Handle PHRASE clarifications
- [ ] Handle TIME on dates
- [ ] Handle SDATE sort dates
- [ ] Handle NO negative assertions
- [ ] Handle EXID external IDs
- [ ] Handle multiple UID
- [ ] Handle MIME on notes
- [ ] Handle LANG on notes/names
- [ ] Handle RESN as comma-separated list
- [ ] Handle enhanced OBJE with CROP
- [ ] Handle ROLE.PHRASE on associations
- [ ] Store extension schema mappings
- [ ] Handle custom tags with schema URIs

## 15. ID Generation

- [ ] Implement person ID generation (name-based with XRef suffix)
- [ ] Implement event ID generation (type + person + counter)
- [ ] Implement relationship ID generation (type + participants + XRef)
- [ ] Implement place ID generation (hierarchical name)
- [ ] Implement source ID generation (title + XRef)
- [ ] Implement citation ID generation (subject + source + counter)
- [ ] Implement assertion ID generation (subject + claim + counter)
- [ ] Implement repository ID generation (name + XRef)
- [ ] Implement media ID generation (title + XRef)
- [ ] Handle ID collisions with counters
- [ ] Sanitize strings for IDs (lowercase, hyphens, no special chars)
- [ ] Store all ID mappings in context

## 16. Error Handling

- [ ] Define error types (fatal, error, warning)
- [ ] Implement error collection
- [ ] Implement warning collection
- [ ] Handle malformed GEDCOM structure
- [ ] Handle unsupported versions
- [ ] Handle character encoding errors
- [ ] Handle dangling references
- [ ] Handle circular references
- [ ] Handle unparseable dates
- [ ] Handle empty/null values
- [ ] Handle missing required fields
- [ ] Handle unknown custom tags
- [ ] Handle ambiguous data
- [ ] Generate detailed error messages with line numbers
- [ ] Provide suggestions for fixing errors

## 17. Statistics Tracking

- [ ] Track lines processed
- [ ] Track records processed
- [ ] Track persons imported
- [ ] Track relationships created
- [ ] Track events created (total and by type)
- [ ] Track places created
- [ ] Track sources imported
- [ ] Track citations created
- [ ] Track repositories imported
- [ ] Track media imported
- [ ] Track assertions created
- [ ] Track errors and warnings count
- [ ] Track skipped records/tags
- [ ] Track unknown tags
- [ ] Track processing time
- [ ] Track peak memory usage (if profiling enabled)

## 18. Import Report

- [ ] Generate comprehensive import report
- [ ] Include file information
- [ ] Include GEDCOM version and encoding
- [ ] Include statistics table
- [ ] List errors with line numbers and context
- [ ] List warnings with context
- [ ] List unknown/custom tags encountered
- [ ] Provide summary of GEDCOM-specific mappings
- [ ] Provide next steps guidance
- [ ] Format report for readability

## 19. Testing

### Unit Tests

- [ ] Test date parser with all formats
- [ ] Test name parser with all variations
- [ ] Test place parser and hierarchy builder
- [ ] Test ID generators
- [ ] Test GEDCOM line parser
- [ ] Test record builder
- [ ] Test version detection
- [ ] Test error handling
- [ ] Test utility functions

### Integration Tests

- [ ] Test import of `minimal70.ged`
- [ ] Test import of `same-sex-marriage.ged`
- [ ] Test import of `shakespeare.ged`
- [ ] Test import of `kennedy.ged`
- [ ] Test import of `british-royalty.ged`
- [ ] Test import of `bullinger.ged` (stress test)
- [ ] Test import of `maximal70.ged` (full GEDCOM 7.0)
- [ ] Test import of `date-all.ged` (all date formats)
- [ ] Test import of `age-all.ged` (all age formats)
- [ ] Test import of `torture-test-551.ged` (edge cases)
- [ ] Verify entity counts for known files
- [ ] Verify specific entities exist and have correct data
- [ ] Verify relationships created correctly
- [ ] Verify evidence chains created correctly
- [ ] Verify GEDCOM 7.0 features work

### Validation Tests

- [ ] Validate all imported GLX archives pass schema validation
- [ ] Verify no broken references
- [ ] Verify all vocabularies used are defined
- [ ] Verify all properties used are defined

### Performance Tests

- [ ] Benchmark small file import (< 1s)
- [ ] Benchmark medium file import (< 5s)
- [ ] Benchmark large file import (< 30s)
- [ ] Benchmark very large file import (< 2min)
- [ ] Profile memory usage
- [ ] Profile CPU usage
- [ ] Optimize hotspots

## 20. Documentation

- [ ] Complete this implementation plan
- [ ] Document vocabulary additions
- [ ] Document property additions
- [ ] Create user guide for GEDCOM import
- [ ] Create developer guide for extending import
- [ ] Document known limitations
- [ ] Document edge cases and their handling
- [ ] Create examples and tutorials
- [ ] Update main GLX documentation to reference GEDCOM import

## 21. CLI Integration

- [ ] Add `glx import gedcom` command
- [ ] Support file path argument
- [ ] Support output path argument
- [ ] Add `--version` flag to specify GEDCOM version (or auto-detect)
- [ ] Add `--validate` flag to validate after import
- [ ] Add `--report` flag to control report generation
- [ ] Add `--verbose` flag for detailed output
- [ ] Add progress indicator for large files
- [ ] Stream import report to stdout or file
- [ ] Return appropriate exit codes

## 22. Polish and Optimization

- [ ] Implement streaming parser for memory efficiency
- [ ] Optimize place hierarchy building
- [ ] Optimize ID generation
- [ ] Implement concurrent processing where safe
- [ ] Add caching for repeated lookups
- [ ] Profile and optimize bottlenecks
- [ ] Reduce memory allocations
- [ ] Final code review
- [ ] Final testing
- [ ] Performance validation

---

# Reference Tables

## Complete GEDCOM 5.5.1 Tag Reference

(continues in next section...)

## Complete GEDCOM 5.5.1 Tag Reference

### All Level-0 Record Tags

| Tag | Name | Frequency | Processing |
|-----|------|-----------|------------|
| `HEAD` | Header | 1 per file | Extract metadata, place format, version |
| `SUBM` | Submitter | 1-3 per file | Store metadata or ignore |
| `SUBN` | Submission | 0-1 per file | Store metadata or ignore |
| `INDI` | Individual | Many | Convert to Person + Events |
| `FAM` | Family | Many | Convert to Relationships + Events |
| `SOUR` | Source | 0-many | Convert to Source |
| `REPO` | Repository | 0-many | Convert to Repository |
| `OBJE` | Multimedia | 0-many | Convert to Media |
| `NOTE` | Note (shared) | 0-many | Resolve to inline notes |
| `TRLR` | Trailer | 1 per file | End marker, no processing |

### All Event/Fact Tags (Individual)

| Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Event Type | Category |
|-----|--------------|------------|----------------|----------|
| `ADOP` | ✓ | ✓ | adoption | lifecycle |
| `BAPM` | ✓ | ✓ | baptism | religious |
| `BARM` | ✓ | ✓ | bar_mitzvah | religious |
| `BASM` | ✓ | ✓ | bat_mitzvah | religious |
| `BIRT` | ✓ | ✓ | birth | lifecycle |
| `BLES` | ✓ | ✓ | blessing | religious |
| `BURI` | ✓ | ✓ | burial | lifecycle |
| `CENS` | ✓ | ✓ | census | official |
| `CHR` | ✓ | ✓ | christening | religious |
| `CHRA` | ✓ | ✓ | adult_christening | religious |
| `CONF` | ✓ | ✓ | confirmation | religious |
| `CREM` | ✓ | ✓ | cremation | lifecycle |
| `DEAT` | ✓ | ✓ | death | lifecycle |
| `EMIG` | ✓ | ✓ | emigration | migration |
| `EVEN` | ✓ | ✓ | [from TYPE] | custom |
| `FACT` | ✓ | ✓ | [from TYPE] | custom |
| `FCOM` | ✓ | ✓ | first_communion | religious |
| `GRAD` | ✓ | ✓ | graduation | achievement |
| `IMMI` | ✓ | ✓ | immigration | migration |
| `NATU` | ✓ | ✓ | naturalization | legal |
| `ORDN` | ✓ | ✓ | ordination | religious |
| `PROB` | ✓ | ✓ | probate | legal |
| `RETI` | ✓ | ✓ | retirement | lifecycle |
| `WILL` | ✓ | ✓ | will | legal |
| `BAPL` | - | ✓ | lds_baptism | lds |
| `CONL` | - | ✓ | lds_confirmation | lds |
| `ENDL` | - | ✓ | lds_endowment | lds |
| `SLGC` | - | ✓ | lds_sealing_child | lds |

### All Event Tags (Family)

| Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Event Type | Category |
|-----|--------------|------------|----------------|----------|
| `ANUL` | ✓ | ✓ | annulment | lifecycle |
| `CENS` | ✓ | ✓ | census | official |
| `DIV` | ✓ | ✓ | divorce | lifecycle |
| `DIVF` | ✓ | ✓ | divorce_filed | legal |
| `ENGA` | ✓ | ✓ | engagement | lifecycle |
| `EVEN` | ✓ | ✓ | [from TYPE] | custom |
| `MARR` | ✓ | ✓ | marriage | lifecycle |
| `MARB` | ✓ | ✓ | marriage_banns | lifecycle |
| `MARC` | ✓ | ✓ | marriage_contract | legal |
| `MARL` | ✓ | ✓ | marriage_license | legal |
| `MARS` | ✓ | ✓ | marriage_settlement | legal |
| `RESI` | ✓ | ✓ | residence | attribute |
| `SLGS` | - | ✓ | lds_sealing_spouse | lds |

### All Attribute Tags (Individual)

| Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Property | Type |
|-----|--------------|------------|--------------|------|
| `CAST` | ✓ | ✓ | caste | string |
| `DSCR` | ✓ | ✓ | description | string |
| `EDUC` | ✓ | ✓ | education | temporal |
| `IDNO` | ✓ | ✓ | id_number | string |
| `NATI` | ✓ | ✓ | nationality | temporal |
| `NCHI` | ✓ | ✓ | num_children | integer |
| `NMR` | ✓ | ✓ | num_marriages | integer |
| `OCCU` | ✓ | ✓ | occupation | temporal |
| `PROP` | ✓ | ✓ | possessions | string |
| `RELI` | ✓ | ✓ | religion | temporal |
| `RESI` | ✓ | ✓ | residence | temporal/event |
| `SSN` | ✓ | ✓ | ssn | string |
| `TITL` | ✓ | ✓ | title | temporal |

### All System Tags

| Tag | Description | GLX Storage | Notes |
|-----|-------------|-------------|-------|
| `REFN` | User reference number | properties.reference_number | Multiple allowed |
| `RIN` | Automated record ID | properties.record_id | System-generated |
| `RFN` | Record file number | properties.record_file_number | Permanent ID |
| `AFN` | Ancestral file number | properties.ancestral_file_number | FamilySearch ID |
| `UID` | Unique identifier | properties.uid (array) | Can have multiple |
| `EXID` | External ID (7.0) | properties.external_ids | Array of {type, id} |
| `CHAN` | Change date | Ignore or notes | Last modification |
| `CREA` | Creation date (7.0) | properties.creation_date | When created |

### All Header Tags

| Tag | Path | Description | Processing |
|-----|------|-------------|------------|
| `GEDC` | HEAD.GEDC | GEDCOM spec info | Container |
| `VERS` | HEAD.GEDC.VERS | GEDCOM version | Critical for version detection |
| `FORM` | HEAD.GEDC.FORM | Format (LINEAGE-LINKED) | Verify expected format |
| `SOUR` | HEAD.SOUR | Source system | Metadata |
| `VERS` | HEAD.SOUR.VERS | System version | Metadata |
| `NAME` | HEAD.SOUR.NAME | System name | Metadata |
| `CORP` | HEAD.SOUR.CORP | Corporation | Metadata |
| `DATA` | HEAD.SOUR.DATA | Source data | Metadata |
| `DATE` | HEAD.SOUR.DATA.DATE | Data date | Metadata |
| `COPR` | HEAD.SOUR.DATA.COPR | Copyright | Metadata |
| `DEST` | HEAD.DEST | Destination system | Metadata |
| `DATE` | HEAD.DATE | Transmission date | Metadata |
| `TIME` | HEAD.DATE.TIME | Transmission time | Metadata |
| `SUBM` | HEAD.SUBM | Submitter link | Metadata |
| `SUBN` | HEAD.SUBN | Submission link | Metadata |
| `FILE` | HEAD.FILE | Filename | Metadata |
| `COPR` | HEAD.COPR | Copyright | Metadata |
| `LANG` | HEAD.LANG | Language | Metadata |
| `PLAC` | HEAD.PLAC | Place format | Critical for place parsing |
| `FORM` | HEAD.PLAC.FORM | Place hierarchy format | Critical |
| `CHAR` | HEAD.CHAR | Character encoding | Critical for encoding |
| `NOTE` | HEAD.NOTE | Header note | Metadata |
| `SCHMA` | HEAD.SCHMA | Extension schema (7.0) | Parse extensions |

## GEDCOM Date Format Complete Reference

### Basic Formats

| Format | Example | ISO 8601 Output | Notes |
|--------|---------|-----------------|-------|
| Year | `1850` | `1850` | Direct |
| Month Year | `JAN 1850` | `1850-01` | Partial date |
| Month Year | `JANUARY 1850` | `1850-01` | Full month name |
| Day Month Year | `25 JAN 1850` | `1850-01-25` | Full date |
| Day Month Year | `25 JANUARY 1850` | `1850-01-25` | Full month name |

### Month Abbreviations

| GEDCOM | Number | Full Name |
|--------|--------|-----------|
| `JAN` | 01 | January |
| `FEB` | 02 | February |
| `MAR` | 03 | March |
| `APR` | 04 | April |
| `MAY` | 05 | May |
| `JUN` | 06 | June |
| `JUL` | 07 | July |
| `AUG` | 08 | August |
| `SEP` | 09 | September |
| `OCT` | 10 | October |
| `NOV` | 11 | November |
| `DEC` | 12 | December |

### Date Qualifiers

| Qualifier | Example | GLX Output | Properties |
|-----------|---------|------------|------------|
| `ABT` (about) | `ABT 1850` | `~1850` or `1850?` | date_approximate: true |
| `CAL` (calculated) | `CAL 1850` | `1850` | date_calculated: true |
| `EST` (estimated) | `EST 1850` | `1850` | date_estimated: true |

### Relative Dates

| Qualifier | Example | GLX Output | Notes |
|-----------|---------|------------|-------|
| `BEF` (before) | `BEF 1850` | `<1850` | Before date |
| `AFT` (after) | `AFT 1850` | `>1850` | After date |

### Date Ranges

| Format | Example | GLX Output | Notes |
|--------|---------|------------|-------|
| `BET...AND` | `BET 1849 AND 1851` | `1849/1851` | ISO 8601 interval |
| `FROM...TO` | `FROM 1849 TO 1851` | `1849/1851` | ISO 8601 interval |

### Date Periods

| Format | Example | GLX Output | Notes |
|--------|---------|------------|-------|
| `FROM` | `FROM 1850` | `1850/` | Open-ended start |
| `TO` | `TO 1850` | `/1850` | Open-ended end |

### Interpreted Dates

| Format | Example | Handling |
|--------|---------|----------|
| `INT...(...) ` | `INT 25 JAN 1850 (interpreted from census)` | Store date + phrase in properties.date_interpretation |

### GEDCOM 7.0 Time Extensions

| Format | Example | GLX Output |
|--------|---------|------------|
| `DATE` + `TIME` | `DATE 25 JAN 1850` + `TIME 14:30:00` | `1850-01-25T14:30:00` |
| `DATE` + `TIME` + `PHRASE` | + `PHRASE "afternoon"` | + properties.date_phrase: "afternoon" |

### Hebrew Calendar

| Format | Example | Notes |
|--------|---------|-------|
| `@#DHEBREW@` | `@#DHEBREW@ 5 TVT 5670` | Specify calendar, convert if possible |

### Julian Calendar

| Format | Example | Notes |
|--------|---------|-------|
| `@#DJULIAN@` | `@#DJULIAN@ 25 JAN 1850` | Specify calendar, convert to Gregorian |

### French Republican Calendar

| Format | Example | Notes |
|--------|---------|-------|
| `@#DFRENCH R@` | `@#DFRENCH R@ 5 VEND 10` | Specify calendar |

## Name Format Complete Reference

### Basic Formats

| GEDCOM Name | Given | Surname | Suffix | Full Name |
|-------------|-------|---------|--------|-----------|
| `John /Smith/` | John | Smith | - | John Smith |
| `Mary Jane /Doe/` | Mary Jane | Doe | - | Mary Jane Doe |
| `Robert /de La Cruz/` | Robert | de La Cruz | - | Robert de La Cruz |
| `John /Smith/ Jr.` | John | Smith | Jr. | John Smith Jr. |
| `Dr. John /Smith/` | John | Smith | - | Dr. John Smith |
| `John Q. /Public/ III` | John Q. | Public | III | John Q. Public III |

### With Surname Prefix

| GEDCOM Name | Prefix | Surname | Full Surname |
|-------------|--------|---------|--------------|
| `/de Gaulle/` | de | Gaulle | de Gaulle |
| `/von Braun/` | von | Braun | von Braun |
| `/van Gogh/` | van | Gogh | van Gogh |
| `/Le Beau/` | Le | Beau | Le Beau |
| `/di Medici/` | di | Medici | di Medici |
| `/del Rio/` | del | Rio | del Rio |

### With Nickname

| GEDCOM Name | Given | Nickname | Surname |
|-------------|-------|----------|---------|
| `John "Jack" /Kennedy/` | John | Jack | Kennedy |
| `William "Bill" /Clinton/` | William | Bill | Clinton |
| `Robert "Bob" /Smith/` | Robert | Bob | Smith |

### With Prefix

| GEDCOM Name | Prefix | Given | Surname |
|-------------|--------|-------|---------|
| `Dr. John /Smith/` | Dr. | John | Smith |
| `Rev. John /Smith/` | Rev. | John | Smith |
| `Lt. Cmndr. John /Smith/` | Lt. Cmndr. | John | Smith |
| `Prof. John /Smith/` | Prof. | John | Smith |

### With Suffix

| GEDCOM Name | Given | Surname | Suffix |
|-------------|-------|---------|--------|
| `John /Smith/ Jr.` | John | Smith | Jr. |
| `John /Smith/ Sr.` | John | Smith | Sr. |
| `John /Smith/ III` | John | Smith | III |
| `John /Smith/ Esq.` | John | Smith | Esq. |
| `John /Smith/ PhD` | John | Smith | PhD |

### Complex Names

| GEDCOM Name | Components |
|-------------|------------|
| `Lt. Cmndr. Joseph "Jack" /de La Cruz/ Jr.` | prefix: Lt. Cmndr., given: Joseph, nick: Jack, spfx: de, surname: La Cruz, suffix: Jr. |

### Substructure Format

```gedcom
1 NAME Lt. Cmndr. Joseph "John" /de Allen/ jr.
2 NPFX Lt. Cmndr.
2 GIVN Joseph
2 NICK John
2 SPFX de
2 SURN Allen
2 NSFX jr.
```

## Place Hierarchy Complete Reference

### Common Hierarchies

| Country | Typical Hierarchy | Example |
|---------|-------------------|---------|
| USA | City, County, State, Country | Brookline, Norfolk County, Massachusetts, USA |
| UK | Town, County, Country | Leeds, Yorkshire, England |
| France | City, Department, Region, Country | Paris, Paris, Île-de-France, France |
| Germany | City, State, Country | Munich, Bavaria, Germany |
| Canada | City, Province, Country | Toronto, Ontario, Canada |
| Australia | City, State, Country | Sydney, New South Wales, Australia |

### Place Type Inference by Level

For `City, County, State, Country`:

| Level | Inferred Type |
|-------|---------------|
| 0 (most specific) | city |
| 1 | county |
| 2 | state |
| 3 (most general) | country |

For `City, State, Country`:

| Level | Inferred Type |
|-------|---------------|
| 0 | city |
| 1 | state |
| 2 | country |

### Place Type Keywords

| Keyword in Name | Inferred Type |
|-----------------|---------------|
| County, Shire, Graf | county |
| Parish | parish |
| District | district |
| Region, Province | region |
| City, Town, Village, Borough | city/town |
| State, Land, Canton | state |
| Country, Nation, Kingdom | country |

## Quality (QUAY) to Confidence Mapping

| QUAY | Label | Description | GLX Confidence | Use Case |
|------|-------|-------------|----------------|----------|
| 0 | Unreliable | Evidence is questionable or conflicting | very_low | Known errors or conflicts |
| 1 | Questionable | Evidence is weak or indirect | low | Family lore, no documentation |
| 2 | Secondary | Secondary evidence or reasonable inference | medium | Census, family Bible, headstone |
| 3 | Primary | Direct and primary evidence | high | Birth certificate, contemporary document |
| (none) | Not specified | Quality not evaluated | medium (default) | Default when QUAY not present |

## MIME Type Conversion Reference

| GEDCOM FORM | File Extension | MIME Type | Category |
|-------------|----------------|-----------|----------|
| `bmp` | .bmp | image/bmp | Image |
| `gif` | .gif | image/gif | Image |
| `jpg`, `jpeg` | .jpg, .jpeg | image/jpeg | Image |
| `png` | .png | image/png | Image |
| `tif`, `tiff` | .tif, .tiff | image/tiff | Image |
| `pcx` | .pcx | image/pcx | Image |
| `pict` | .pict | image/pict | Image |
| `pdf` | .pdf | application/pdf | Document |
| `doc` | .doc | application/msword | Document |
| `docx` | .docx | application/vnd.openxmlformats-officedocument.wordprocessingml.document | Document |
| `txt` | .txt | text/plain | Text |
| `rtf` | .rtf | application/rtf | Text |
| `html`, `htm` | .html, .htm | text/html | Web |
| `wav` | .wav | audio/wav | Audio |
| `mp3` | .mp3 | audio/mpeg | Audio |
| `avi` | .avi | video/avi | Video |
| `mpg`, `mpeg` | .mpg, .mpeg | video/mpeg | Video |
| `mp4` | .mp4 | video/mp4 | Video |
| `mov` | .mov | video/quicktime | Video |
| `ole` | .ole | application/ole | Embedded Object |

## LDS Ordinance Status Values

| Status Code | Meaning |
|-------------|---------|
| `CHILD` | Died before 8 years old, no ordinances |
| `COMPLETED` | Completed ordinance |
| `EXCLUDED` | Excluded from ordinance |
| `PRE_1970` | Ordinance from before 1970 |
| `STILLBORN` | Stillborn, no ordinances |
| `SUBMITTED` | Submitted for ordinance |
| `UNCLEARED` | Uncleared for ordinance |
| `BIC` | Born in covenant |
| `DNS` | Do not seal |
| `DNS_CAN` | Do not seal, canceled |

## LDS Temple Codes

(Sample - there are 150+ temples)

| Code | Temple Name | Location |
|------|-------------|----------|
| `LOGAN` | Logan Utah Temple | Logan, UT |
| `SLAKE` | Salt Lake Temple | Salt Lake City, UT |
| `MANTI` | Manti Utah Temple | Manti, UT |
| `PROVO` | Provo Utah Temple | Provo, UT |
| `DENVE` | Denver Colorado Temple | Denver, CO |
| `SACRA` | Sacramento California Temple | Sacramento, CA |

---

# Advanced Implementation Patterns

## Streaming Parser Pattern

For very large files, use streaming to avoid loading entire file into memory:

```go
type StreamingParser struct {
    scanner *bufio.Scanner
    current *GEDCOMLine
    lineNum int
}

func NewStreamingParser(r io.Reader) *StreamingParser {
    scanner := bufio.NewScanner(r)
    scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
    return &StreamingParser{
        scanner: scanner,
        lineNum: 0,
    }
}

func (p *StreamingParser) Next() (*GEDCOMLine, error) {
    if !p.scanner.Scan() {
        if err := p.scanner.Err(); err != nil {
            return nil, err
        }
        return nil, io.EOF
    }

    p.lineNum++
    line, err := parseGEDCOMLine(p.scanner.Text(), p.lineNum)
    if err != nil {
        return nil, err
    }

    p.current = line
    return line, nil
}

func (p *StreamingParser) ParseRecords() ([]*GEDCOMRecord, error) {
    var records []*GEDCOMRecord
    var currentRecord *GEDCOMRecord
    stack := make([]*GEDCOMRecord, 0, 10)

    for {
        line, err := p.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }

        // Build hierarchical records...
    }

    return records, nil
}
```

## Deferred Family Processing Pattern

Families reference individuals that must be loaded first:

```go
func convertToGLX(records []*GEDCOMRecord, version GEDCOMVersion) (*ImportResult, error) {
    ctx := NewConversionContext(version)

    // First pass: Process all non-family records
    for _, record := range records {
        switch record.Tag {
        case "INDI":
            if err := convertIndividual(record, ctx); err != nil {
                // Handle error
            }
        case "SOUR":
            if err := convertSource(record, ctx); err != nil {
                // Handle error
            }
        case "REPO":
            if err := convertRepository(record, ctx); err != nil {
                // Handle error
            }
        case "OBJE":
            if err := convertMedia(record, ctx); err != nil {
                // Handle error
            }
        case "FAM":
            // Defer family processing
            ctx.DeferredFamilies = append(ctx.DeferredFamilies, record)
        }
    }

    // Second pass: Process families now that all individuals exist
    for _, famRecord := range ctx.DeferredFamilies {
        if err := convertFamily(famRecord, ctx); err != nil {
            // Handle error
        }
    }

    return &ImportResult{
        GLX:   ctx.GLX,
        Stats: ctx.Stats,
        Errors: ctx.Errors,
        Warnings: ctx.Warnings,
    }, nil
}
```

## Place Deduplication Pattern

Ensure places are created only once:

```go
func getOrCreatePlace(name string, placeType string, parentID string, coords *Coordinates, ctx *ConversionContext) string {
    // Generate unique key for this place
    key := generatePlaceKey(name, parentID)

    // Check if already exists
    if placeID, exists := ctx.PlaceIDMap[key]; exists {
        return placeID
    }

    // Create new place
    placeID := generatePlaceID(name, ctx)

    place := &Place{
        Name:   name,
        Type:   placeType,
        Parent: parentID,
    }

    if coords != nil {
        place.Latitude = &coords.Latitude
        place.Longitude = &coords.Longitude
    }

    ctx.GLX.Places[placeID] = place
    ctx.PlaceIDMap[key] = placeID
    ctx.Stats.PlacesCreated++

    return placeID
}

func generatePlaceKey(name string, parentID string) string {
    return fmt.Sprintf("%s|%s", sanitizeForID(name), parentID)
}
```

## Progressive Hierarchy Building

Build place hierarchies incrementally:

```go
func buildPlaceHierarchyFromString(placeStr string, placeFormat string, ctx *ConversionContext) (string, error) {
    // Split by comma
    parts := strings.Split(placeStr, ",")
    for i := range parts {
        parts[i] = strings.TrimSpace(parts[i])
    }

    // Reverse to go from general to specific
    reversed := make([]string, len(parts))
    for i := range parts {
        reversed[i] = parts[len(parts)-1-i]
    }

    // Build hierarchy
    var currentParent string
    var lastPlaceID string

    for level, partName := range reversed {
        placeType := inferPlaceType(partName, level, len(reversed))
        placeID := getOrCreatePlace(partName, placeType, currentParent, nil, ctx)
        currentParent = placeID
        lastPlaceID = placeID
    }

    // Return most specific place (leaf)
    return lastPlaceID, nil
}
```

## Citation Builder Pattern

Create evidence chains systematically:

```go
type CitationBuilder struct {
    ctx       *ConversionContext
    sourceID  string
    subjectID string
}

func NewCitationBuilder(sourceID string, subjectID string, ctx *ConversionContext) *CitationBuilder {
    return &CitationBuilder{
        ctx:       ctx,
        sourceID:  sourceID,
        subjectID: subjectID,
    }
}

func (cb *CitationBuilder) FromSOURRecord(sourRecord *GEDCOMRecord) (string, error) {
    citation := &Citation{
        Source: cb.sourceID,
    }

    // Extract citation details from subrecords
    for _, sub := range sourRecord.SubRecords {
        switch sub.Tag {
        case "PAGE":
            citation.Page = sub.Value
        case "TEXT":
            citation.TextFromSource = sub.Value
        case "QUAY":
            if quay, err := strconv.Atoi(sub.Value); err == nil {
                citation.Quality = &quay
            }
        case "NOTE":
            citation.Notes += sub.Value + "\n"
        case "OBJE":
            mediaID := cb.ctx.MediaIDMap[sub.Value]
            if mediaID != "" {
                citation.Media = append(citation.Media, mediaID)
            }
        }
    }

    // Generate citation ID
    citationID := generateCitationID(cb.subjectID, cb.sourceID, cb.ctx)

    // Store citation
    cb.ctx.GLX.Citations[citationID] = citation
    cb.ctx.Stats.CitationsCreated++

    return citationID, nil
}

func (cb *CitationBuilder) CreateAssertion(claim string, value interface{}, citationIDs []string) error {
    if claim == "" || value == nil {
        return nil // No assertion needed
    }

    assertionID := generateAssertionID(cb.subjectID, claim, cb.ctx)

    // Get quality from first citation
    var confidence string
    if len(citationIDs) > 0 {
        firstCitation := cb.ctx.GLX.Citations[citationIDs[0]]
        confidence = mapQUAYtoConfidence(firstCitation.Quality)
    } else {
        confidence = "medium"
    }

    assertion := &Assertion{
        Subject:    cb.subjectID,
        Claim:      claim,
        Value:      value,
        Confidence: confidence,
        Citations:  citationIDs,
    }

    cb.ctx.GLX.Assertions[assertionID] = assertion
    cb.ctx.Stats.AssertionsCreated++

    return nil
}
```

---

# Conclusion

This comprehensive implementation plan provides complete coverage of GEDCOM 5.5.1 and 7.0 specifications for import into GLX format. The plan includes:

1. **Complete tag inventory** - All 95+ GEDCOM 5.5.1 tags and 110+ GEDCOM 7.0 tags mapped
2. **Vocabulary additions** - 60+ new vocabulary entries needed across all types
3. **Property additions** - 40+ new properties needed for persons, events, relationships, etc.
4. **Full conversion strategies** - Detailed algorithms for every conversion scenario
5. **Complete examples** - Working examples of complex conversions
6. **Edge case handling** - Strategies for all known edge cases
7. **Implementation patterns** - Reusable patterns for efficient implementation
8. **Comprehensive checklist** - 250+ checkboxes covering every aspect
9. **Reference tables** - Complete references for dates, names, places, etc.
10. **Test strategy** - Coverage of all 12 test files

The implementation is designed to be:
- **Complete** - 100% coverage of both specifications
- **Robust** - Handles all edge cases and errors gracefully
- **Efficient** - Optimized for large files (17K+ lines)
- **Standard** - Uses only standard GLX vocabularies (no custom needed)
- **Evidence-rich** - Creates proper evidence chains from GEDCOM sources

Total estimated implementation: **1500-2000 lines of Go code** (excluding tests)
Total estimated tests: **500-800 lines of test code**
Total vocabulary additions: **~60 entries across 10+ files**

This plan is ready for immediate implementation without need for phasing or incremental planning.

---

**Document Version**: 2.0 - Complete Unified Plan
**Last Updated**: 2025-11-18
**Status**: Ready for Implementation

