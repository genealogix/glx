# GEDCOM to GLX Mapping

This document describes how GEDCOM (versions 5.5.1 and 7.0) structures map to GLX (Genealogix Archive) format.

## Status Legend

- ✅ **Fully Supported** - Direct mapping exists with full coverage
- 🟡 **Partially Supported** - Mapping exists but some data may be lost or transformed
- 🔴 **Not Supported** - No mapping exists yet
- 📝 **Notes Field** - Data stored in notes or properties field

## Entity Mappings

### INDI (Individual) → Person

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `@ID@` | ✓ | ✓ | Map key (person ID) | ✅ | Direct mapping |
| `NAME` | ✓ | ✓ | `properties.name` | 🟡 | Needs property definition for `name` |
| `GIVN` | ✓ | ✓ | `properties.given_name` | 🟡 | Needs property definition |
| `SURN` | ✓ | ✓ | `properties.surname` | 🟡 | Needs property definition |
| `SEX` | ✓ | ✓ | `properties.sex` | 🟡 | Needs property definition |
| `BIRT` | ✓ | ✓ | Event (type: `birth`) | ✅ | Create event with participant |
| `DEAT` | ✓ | ✓ | Event (type: `death`) | ✅ | Create event with participant |
| `BAPM/CHR` | ✓ | ✓ | Event (type: `baptism`/`christening`) | ✅ | Create event with participant |
| `BURI` | ✓ | ✓ | Event (type: `burial`) | ✅ | Create event with participant |
| `CENS` | ✓ | ✓ | Event (type: `census`) | ✅ | Create event with participant |
| `GRAD` | ✓ | ✓ | Event (type: `graduation`) | 🟡 | May need custom event type |
| `RESI` | ✓ | ✓ | Event (type: `residence`) | 🟡 | May need custom event type |
| `OCCU` | ✓ | ✓ | `properties.occupation` (temporal) | 🟡 | Temporal property with date |
| `EDUC` | ✓ | ✓ | `properties.education` (temporal) | 🟡 | Temporal property |
| `RELI` | ✓ | ✓ | `properties.religion` (temporal) | 🟡 | Temporal property |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |
| `OBJE` | ✓ | ✓ | Media references | 🔴 | Need to link to media entities |
| `SOUR` | ✓ | ✓ | Create citations | 🔴 | Need to create citation entities |
| `FAMC` | ✓ | ✓ | Relationship (child) | ✅ | Create family relationships |
| `FAMS` | ✓ | ✓ | Relationship (spouse) | ✅ | Create family relationships |
| `ASSO` | ✓ | ✓ | `notes` or custom relationship | 🔴 | Needs design decision |
| `REFN` | ✓ | ✓ | `properties.reference_number` | 🔴 | User reference numbers |
| `RIN` | ✓ | ✓ | `properties.record_id` | 🔴 | Automated record ID |
| `CHAN` | ✓ | ✓ | `notes` | 📝 | Change metadata |
| `_CUSTOM` | ✓ | ✓ | `properties._custom` | 🔴 | Custom tags |

### FAM (Family) → Relationship

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `@ID@` | ✓ | ✓ | Map key (relationship ID) | ✅ | Direct mapping |
| `HUSB` | ✓ | ✓ | `participants[0]` (role: `partner`) | ✅ | First partner |
| `WIFE` | ✓ | ✓ | `participants[1]` (role: `partner`) | ✅ | Second partner |
| `CHIL` | ✓ | ✓ | Create parent-child relationships | ✅ | Separate relationship per child |
| `MARR` | ✓ | ✓ | `start_event` (type: `marriage`) | ✅ | Create marriage event |
| `DIV` | ✓ | ✓ | `end_event` (type: `divorce`) | 🟡 | May need custom event type |
| `ANUL` | ✓ | ✓ | `end_event` (type: `annulment`) | 🟡 | May need custom event type |
| `MARB` | ✓ | ✓ | Event (type: `marriage_banns`) | 🔴 | Marriage banns event |
| `MARC` | ✓ | ✓ | Event (type: `marriage_contract`) | 🔴 | Marriage contract |
| `MARL` | ✓ | ✓ | Event (type: `marriage_license`) | 🔴 | Marriage license |
| `MARS` | ✓ | ✓ | Event (type: `marriage_settlement`) | 🔴 | Marriage settlement |
| `ENGA` | ✓ | ✓ | Event (type: `engagement`) | 🔴 | Engagement event |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |
| `SOUR` | ✓ | ✓ | Create citations | 🔴 | Need citation linking |

### SOUR (Source) → Source

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `@ID@` | ✓ | ✓ | Map key (source ID) | ✅ | Direct mapping |
| `TITL` | ✓ | ✓ | `title` | ✅ | Direct mapping |
| `AUTH` | ✓ | ✓ | `authors` | ✅ | Direct mapping (array) |
| `PUBL` | ✓ | ✓ | `publication_info` | ✅ | Direct mapping |
| `DATE` | ✓ | ✓ | `date` | ✅ | Direct mapping |
| `TEXT` | ✓ | ✓ | `description` | 🟡 | Source text → description |
| `REPO` | ✓ | ✓ | `repository` | ✅ | Repository reference |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |
| `OBJE` | ✓ | ✓ | `media` | 🔴 | Media references |
| `ABBR` | ✓ | ✓ | `properties.abbreviation` | 🔴 | Custom property |

### REPO (Repository) → Repository

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `@ID@` | ✓ | ✓ | Map key (repository ID) | ✅ | Direct mapping |
| `NAME` | ✓ | ✓ | `name` | ✅ | Direct mapping |
| `ADDR` | ✓ | ✓ | `address` | ✅ | Direct mapping |
| `CITY` | ✓ | ✓ | `city` | ✅ | Direct mapping |
| `STAE` | ✓ | ✓ | `state_province` | ✅ | Direct mapping |
| `POST` | ✓ | ✓ | `postal_code` | ✅ | Direct mapping |
| `CTRY` | ✓ | ✓ | `country` | ✅ | Direct mapping |
| `PHON` | ✓ | ✓ | `phone` | ✅ | Direct mapping |
| `EMAIL` | ✓ | ✓ | `email` | ✅ | Direct mapping |
| `WWW` | ✓ | ✓ | `website` | ✅ | Direct mapping |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |

### OBJE (Multimedia) → Media

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `@ID@` | ✓ | ✓ | Map key (media ID) | ✅ | Direct mapping |
| `FILE` | ✓ | ✓ | `uri` | ✅ | File path/URI |
| `FORM` | ✓ | ✓ | `mime_type` | 🟡 | Convert format to MIME type |
| `TITL` | ✓ | ✓ | `title` | ✅ | Direct mapping |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |

### PLAC (Place) → Place

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `PLAC` | ✓ | ✓ | Create place entity | ✅ | Parse hierarchical place names |
| `FORM` | ✓ | ✓ | Parse structure | 🟡 | Use to understand hierarchy |
| `MAP.LATI` | ✓ | ✓ | `latitude` | ✅ | Direct mapping |
| `MAP.LONG` | ✓ | ✓ | `longitude` | ✅ | Direct mapping |
| `NOTE` | ✓ | ✓ | `notes` | ✅ | Direct mapping |

### SNOTE (Shared Note) → Notes

| GEDCOM Tag | GEDCOM 5.5.1 | GEDCOM 7.0 | GLX Field | Status | Notes |
|------------|--------------|------------|-----------|--------|-------|
| `NOTE` (inline) | ✓ | ✓ | Inline notes | ✅ | Direct text to notes field |
| `@NOTE@` (shared) | - | ✓ | 🔴 | Need shared note resolution |

## Event Mappings

Events in GEDCOM are typically subordinate to INDI or FAM records. In GLX, they become first-class entities.

### Individual Events

| GEDCOM Tag | Event Type | Status | Notes |
|------------|------------|--------|-------|
| `BIRT` | `birth` | ✅ | Standard event type |
| `CHR/BAPM` | `christening`/`baptism` | ✅ | Standard event type |
| `DEAT` | `death` | ✅ | Standard event type |
| `BURI` | `burial` | ✅ | Standard event type |
| `CREM` | `cremation` | 🔴 | Need event type |
| `ADOP` | `adoption` | 🔴 | Need event type |
| `BAPM` | `baptism` | ✅ | Standard event type |
| `BARM` | `bar_mitzvah` | 🔴 | Need event type |
| `BASM` | `bas_mitzvah` | 🔴 | Need event type |
| `BLES` | `blessing` | 🔴 | Need event type |
| `CHRA` | `adult_christening` | 🔴 | Need event type |
| `CONF` | `confirmation` | 🔴 | Need event type |
| `FCOM` | `first_communion` | 🔴 | Need event type |
| `ORDN` | `ordination` | 🔴 | Need event type |
| `NATU` | `naturalization` | 🔴 | Need event type |
| `EMIG` | `emigration` | 🔴 | Need event type |
| `IMMI` | `immigration` | 🔴 | Need event type |
| `CENS` | `census` | ✅ | Standard event type |
| `PROB` | `probate` | 🔴 | Need event type |
| `WILL` | `will` | 🔴 | Need event type |
| `GRAD` | `graduation` | 🟡 | May need custom event type |
| `RETI` | `retirement` | 🔴 | Need event type |

### Family Events

| GEDCOM Tag | Event Type | Status | Notes |
|------------|------------|--------|-------|
| `MARR` | `marriage` | ✅ | Standard event type |
| `MARB` | `marriage_banns` | 🔴 | Need event type |
| `MARC` | `marriage_contract` | 🔴 | Need event type |
| `MARL` | `marriage_license` | 🔴 | Need event type |
| `MARS` | `marriage_settlement` | 🔴 | Need event type |
| `DIV` | `divorce` | 🔴 | Need event type |
| `DIVF` | `divorce_filed` | 🔴 | Need event type |
| `ANUL` | `annulment` | 🔴 | Need event type |
| `ENGA` | `engagement` | 🔴 | Need event type |

## Event Substructures

| GEDCOM Tag | GLX Field | Status | Notes |
|------------|-----------|--------|-------|
| `DATE` | `event.date` | ✅ | Parse GEDCOM date format |
| `PLAC` | `event.place` | ✅ | Create/reference place entity |
| `TYPE` | `event.type` or `event.value` | 🟡 | Context-dependent |
| `AGE` | `participant.notes` | 📝 | Store in participant notes |
| `CAUS` | `event.description` or `event.properties.cause` | 🟡 | Cause of event |
| `NOTE` | `event.notes` | ✅ | Direct mapping |
| `SOUR` | Create citation | 🔴 | Link citations to events |
| `OBJE` | Media reference | 🔴 | Link media to events |

## Citation Mapping

GEDCOM `SOUR` citations (subordinate to other records) map to GLX `Citation` entities:

| GEDCOM Tag | GLX Field | Status | Notes |
|------------|-----------|--------|-------|
| `SOUR @ID@` | `citation.source` | 🔴 | Reference to source |
| `PAGE` | `citation.page` | ✅ | Direct mapping |
| `TEXT` | `citation.text_from_source` | ✅ | Direct mapping |
| `QUAY` | `citation.quality` | ✅ | Quality rating (0-3) |
| `DATA.DATE` | `citation.locator` or `notes` | 🔴 | Entry date in source |
| `DATA.TEXT` | `citation.transcription` | 🔴 | Text from source |
| `OBJE` | `citation.media` | 🔴 | Media references |

## Header Information

GEDCOM header (HEAD) contains metadata that doesn't directly map to GLX entities but could be stored:

| GEDCOM Tag | GLX Storage | Status | Notes |
|------------|-------------|--------|-------|
| `SOUR` | `properties.gedcom_source` | 🔴 | Source system |
| `VERS` | `properties.gedcom_version` | 🔴 | Source system version |
| `NAME` | `properties.gedcom_source_name` | 🔴 | Source system name |
| `CORP` | `properties.gedcom_corporation` | 🔴 | Corporation |
| `DATE` | `properties.gedcom_date` | 🔴 | Transmission date |
| `SUBM` | Reference to submitter | 🔴 | Submitter reference |
| `FILE` | `properties.gedcom_filename` | 🔴 | Original filename |
| `COPR` | `properties.gedcom_copyright` | 🔴 | Copyright notice |
| `GEDC.VERS` | Used to determine parser version | ✅ | GEDCOM version (5.5.1 or 7.0) |
| `CHAR` | Used for encoding | ✅ | Character encoding |
| `LANG` | `properties.gedcom_language` | 🔴 | Language of data |
| `PLAC.FORM` | Used for place parsing | 🟡 | Place hierarchy format |

## Data Type Conversions

### Dates

GEDCOM uses various date formats that need parsing:

| GEDCOM Date Format | GLX Date Format | Status | Notes |
|-------------------|-----------------|--------|-------|
| `25 DEC 1800` | `1800-12-25` | 🔴 | Parse to ISO 8601 |
| `DEC 1800` | `1800-12` | 🔴 | Partial date |
| `1800` | `1800` | ✅ | Year only |
| `BET 1800 AND 1805` | Range: `1800` to `1805` | 🔴 | Date range |
| `AFT 1800` | `>1800` or notes | 🔴 | Qualified date |
| `BEF 1800` | `<1800` or notes | 🔴 | Qualified date |
| `ABT 1800` | `~1800` or notes | 🔴 | Approximate date |
| `CAL 1800` | `1800` + note | 🔴 | Calculated date |
| `EST 1800` | `1800` + note | 🔴 | Estimated date |
| `FROM 1800 TO 1805` | Range: `1800` to `1805` | 🔴 | Date range |

### Names

GEDCOM names use `/surname/` notation:

| GEDCOM Name | GLX Properties | Status | Notes |
|-------------|----------------|--------|-------|
| `John /Smith/` | `given_name: John`, `surname: Smith` | 🔴 | Parse name parts |
| `John Q. /Public/` | `given_name: John Q.`, `surname: Public` | 🔴 | Parse name parts |
| `John /Smith/ Jr.` | Parse suffix | 🔴 | Handle suffix |

### Places

GEDCOM places are comma-separated hierarchical:

| GEDCOM Place | GLX Place | Status | Notes |
|--------------|-----------|--------|-------|
| `City, County, State, Country` | Create hierarchy | 🔴 | Create parent-child places |
| Single place | Single place entity | ✅ | Direct mapping |

## Version-Specific Differences

### GEDCOM 7.0 New Features

| Feature | Status | Notes |
|---------|--------|-------|
| SNOTE (shared notes) | 🔴 | Need to implement shared note resolution |
| EXID (external IDs) | 🔴 | Could map to properties |
| NO tag (explicitly no data) | 🔴 | Handle negative assertions |
| PHRASE clarifications | 🔴 | Additional context for values |
| Extension URIs | 🔴 | Custom extension handling |
| Enhanced media handling | 🔴 | Multiple CROP, TITL on OBJE |

### GEDCOM 5.5.1 Legacy Features

| Feature | Status | Notes |
|---------|--------|-------|
| ANCI/DECI (ancestor/descendant interest) | 🔴 | Low priority, store in notes |
| ALIA (alias) | 🔴 | Could create name property |
| _CUSTOM tags | 🔴 | Generic custom property handling |

## Implementation Priority

### Phase 1: Core Entities (MVP)
- ✅ Person (INDI) basic fields
- ✅ Basic name parsing
- ✅ Birth/Death events (BIRT/DEAT)
- ✅ Family relationships (FAM → Relationship)
- ✅ Basic places (PLAC)
- ✅ Simple date parsing (year, full dates)

### Phase 2: Extended Entities
- 🔴 Source (SOUR) entities
- 🔴 Repository (REPO) entities
- 🔴 Media (OBJE) entities
- 🔴 Citation linking
- 🔴 Additional events (baptism, burial, marriage, etc.)
- 🔴 Advanced date parsing (ranges, qualifiers)

### Phase 3: Advanced Features
- 🔴 Shared notes (GEDCOM 7.0)
- 🔴 Place hierarchies
- 🔴 Name variants and alternative names
- 🔴 Temporal properties (occupation, residence)
- 🔴 Custom tags and extensions
- 🔴 Assertion entities for research conclusions

## Coverage Summary

| Category | Fully Supported | Partially Supported | Not Yet Supported |
|----------|----------------|---------------------|-------------------|
| **Core Entities** | 3/7 | 3/7 | 1/7 |
| **Individual Tags** | 3/20 | 8/20 | 9/20 |
| **Family Tags** | 2/10 | 3/10 | 5/10 |
| **Source Tags** | 5/10 | 1/10 | 4/10 |
| **Repository Tags** | 11/11 | 0/11 | 0/11 |
| **Media Tags** | 3/5 | 1/5 | 1/5 |
| **Events** | 5/35 | 1/35 | 29/35 |
| **Date Formats** | 1/10 | 0/10 | 9/10 |

**Overall Coverage**: ~25% complete for MVP, ~50% with Phase 2, ~85% with Phase 3

---

*Last Updated: 2025-11-18*
*This document will be updated as implementation progresses.*
