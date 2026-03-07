---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.0-beta.6] - Unreleased

### Added

#### Build & Release
- **Added `make release-snapshot` target** - Build cross-platform binaries locally without publishing, using GoReleaser snapshot mode
- **Updated release workflow to latest action versions** - `actions/checkout@v4` (with `fetch-depth: 0` for proper changelog), `actions/setup-go@v5`, `goreleaser/goreleaser-action@v6`

#### Person Entity
- **Added name variation tracking** - Expanded the `name.fields.type` classification field with standard values for alternate spellings, abbreviations, and as-recorded forms (`aka`, `maiden`, `anglicized`, `professional`, `as_recorded`). Added documentation and examples for representing name variations like "D. Lane" vs. "Daniel Lane"

#### Standard Vocabularies
- **Added `original_place_name` citation property** - Records the verbatim place name from a source before normalization to a place entity (e.g., "The Town Of Marston" vs the normalized place reference)
- **Added relationship types `neighbor`, `coworker`, `housemate`** - Non-familial relationships commonly found in census, tax, and social records
- **Added event types `legal_separation`, `taxation`, `voter_registration`** - Legal/administrative events for separations, tax rolls, and voter rolls
- **Added source types `population_register`, `tax_record`, `notarial_record`** - Common European and colonial record types
- **Expanded `military` source type description** - Now includes draft registrations and muster rolls

#### Assertion Entity
- **Added existential assertions** - Assertions no longer require `property` or `participant`; an assertion with only `subject` and evidence asserts the entity's existence, optionally at a specific `date` (#26)

### Changed

#### Specification
- **Clarified validation wording** - Person properties and event/relationship participant roles generate warnings (not errors) for unknown values, matching the validation policy in core concepts
- **Clarified `subject` participant role** - Documented as preferred over `principal`

### Fixed

#### Specification
- **Fixed citation GEDCOM mapping** - Corrected invalid `SOUR.CITN.EXID` tag to `SOUR.EXID`
- **Fixed core-concepts.md formatting** - Property Vocabularies heading was merging with preceding table
- **Fixed glossary Secondary Evidence example** - Replaced "census records" (primary evidence) with "published indexes, compiled genealogies"

---

## [0.0.0-beta.5] - 2026-03-06

### Added

#### Standard Vocabularies
- **Added `url` and `accessed` properties for digital sources** - Sources can now record a `url` property, and citations can record an `accessed` date for when an online source was last verified (#21)
- **Added `race` person property** - Temporal string property for recording racial classifications as they appear in historical documents such as census records (#24)
- **Added `url` and `external_ids` citation properties** - Citations can now record a direct URL to cited material and external identifiers (e.g., FamilySearch ARK) for record-level specificity (#23)
- **Added `type` field to `external_ids` property** - All `external_ids` properties (person, source, citation, repository) now support a structured `fields.type` to record the issuing authority (e.g., FamilySearch URI from GEDCOM EXID.TYPE) (#32)
- **Added `type` field to `name` property** - Name property now supports a `fields.type` to classify name usage (e.g., birth, married, alias) (#25)

#### Assertion Entity
- **Added `status` field to assertion entity** — Assertions can now record a research status (e.g., `proven`, `disproven`, `speculative`) independently of `confidence`, allowing researchers to distinguish between certainty and verification state (#27)

#### GEDCOM Import
- **Import NAME.TYPE subfield** - GEDCOM `NAME.TYPE` values (BIRTH, MARRIED, AKA, etc.) are now lowercased and stored in the name property's `type` field (#25)
- **Import EXID on citations** - GEDCOM 7.0 `EXID` tags on source citations are now imported as `external_ids` citation properties (#32)
- **Structured EXID import** - GEDCOM EXID.TYPE is now stored in `fields.type` instead of being concatenated into the ID string; applies to all entity types (#32)

### Fixed

#### GEDCOM Import
- **Multiple GEDCOM NAME records no longer silently dropped** (#29) - When a person has multiple NAME records (birth name, married name, etc.), all names are now stored as a temporal list instead of only keeping the last one
- **FAM event processing no longer depends on HUSB/WIFE tag order** (#15) - Family events (CENS, ENGA, MARB, etc.) are now collected in a first pass and processed after spouse IDs are extracted, so GEDCOM tag order no longer matters
- **Census NOTE no longer discarded when SOUR exists** (#30) - NOTE text on CENS records is now appended to existing citation notes when SOUR sub-records are present, instead of being silently lost
- **Marriage/divorce events use `start_event`/`end_event` instead of properties** - GEDCOM MARR and DIV events are now correctly linked to relationships via the top-level `start_event` and `end_event` fields, eliminating non-vocabulary `marriage_event`/`divorce_event` property warnings
- **Append residence on PLAC-without-DATE instead of overwriting** - When residence came from a GEDCOM RESI tag or census-derived CENS data with a PLAC but no DATE, the residence property was overwritten instead of appended (#22)

---

## [0.0.0-beta.4] - 2026-03-04

### Added

#### Standard Vocabularies
- **Added `township` place type** - Township is a common administrative division in U.S. census and land records, distinct from `town` (a geographic settlement vs. a civil subdivision of a county) (#16)

### Fixed

#### Validation
- **Suggest correct vocabulary key on hyphen/underscore mismatch** - When a reference fails validation due to a hyphen/underscore swap (e.g., `birth_date` vs `birth-date`), the error message now suggests the correct key (#19)

#### CLI
- **Show directory contents in `glx init` non-empty error** - When `glx init` fails because the target directory is not empty, the error message now lists up to 5 files found (e.g., `.DS_Store`, `.git`), helping users diagnose unexpected blockers like hidden files or sync artifacts (#18)
- **Remove self-referencing `replace` directive that blocks `go install`** - The `go.mod` contained a no-op self-referencing replace directive that prevented `go install github.com/genealogix/glx/glx@latest` from working (#17)

#### GEDCOM Import
- **Deduplicate evidence references** - When a GEDCOM record references the same source multiple times, `extractEvidence()` and `extractEventDetails()` now skip IDs already seen, preventing duplicate entries that violate unique constraints in downstream consumers (#13)

#### Documentation & Website
- **Fix dead links and website issues** - Rewrote 83 dead links across the site to point to GitHub URLs and VitePress paths, added solid background to navbar on home page, and fixed module path resolution (#10)
- **Fix Go Report Card link** - Corrected badge link in CLI README to point to the repository root (#11)

## [0.0.0-beta.3] - 2026-02-10

### Added

#### Census Event Type
- **Added `census` event type to standard vocabulary** - Census enumeration events (`CENS` GEDCOM tag) now included in `event-types.glx`

#### Schema Embeds
- **`CitationPropertiesSchema` and `SourcePropertiesSchema` embed variables** - Completes the pattern established by all other vocabulary schema embeds in `embed.go`

#### GEDCOM Import: Eliminate Meaningless Citations
- **Bare source references no longer create empty citation entities** - When a GEDCOM SOUR tag references a source without any citation-level detail (no PAGE, DATA, TEXT, QUAY, NOTE, or OBJE subrecords), the assertion or event now references the source directly via the `sources` field instead of creating a citation that only contains a source reference
- Added `PropertySources` constant for event/relationship properties

### Changed

#### Assertion Entity Improvements

##### Renamed `claim` to `property`
- **Renamed `claim` field to `property`** - The field name now matches the vocabulary terminology (property vocabularies)
- Updated JSON schema, Go types (`Assertion.Claim` → `Assertion.Property`), all specification examples, example archives, test data, and terminology throughout docs
- Renamed test directories: `assertion-unknown-claim` → `assertion-unknown-property`, `assertion-participant-and-claim` → `assertion-participant-and-property`, `invalid-assertion-claims` → `invalid-assertion-properties`

##### Typed Subject Reference
- **Changed `subject` from string to typed reference object** - Prevents entity ID collisions in large archives
- Must specify exactly one of: `person`, `event`, `relationship`, or `place`
- **Before**: `subject: person-john-smith` → **After**: `subject: { person: person-john-smith }`
- Added `EntityRef` Go type with `Type()` and `ID()` helper methods
- Updated validation to ensure exactly one field is set and referenced entity exists

##### Media as Assertion Evidence
- **Added `media` as a third evidence option for assertions** - Assertions can now reference media entities directly as evidence, alongside citations and sources
- Useful for direct visual evidence like gravestone photos, handwritten documents, or family photographs
- JSON schema `anyOf` evidence constraint updated to include `media`

##### Temporal `date` Field
- **Added `date` field to assertions** - Assertions can now specify a date or date range indicating when the asserted property value applies, enabling precise temporal targeting for properties like occupation, residence, and religion that change over time
- Added `Date` field to `Assertion` Go struct and `date` property to assertion JSON schema
- Assertion `value` field is now required when `property` is present

#### Vocabulary Consolidation

##### Adoption Modeling
- **Removed redundant `adoption` relationship type** - Use `adoptive-parent-child` relationship type instead
- Clarified adoption semantics: `adoption` event type records the legal proceeding; `adoptive-parent-child` relationship type models the ongoing bond
- Removed `RelationshipTypeAdoption` constant from Go code

##### Godparent Modeling
- **Clarified godparent dual usage** - Participant role `godparent` for event participation (baptism sponsor); relationship type `godparent` for the ongoing bond
- Added `godchild` participant role for use in godparent relationships

#### Type System

##### Unified Participant Type
- **Unified participant types** - Consolidated `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into single `Participant` struct
  - All three had identical structure: `person`, `role`, `notes` fields
  - `Event.Participants`, `Relationship.Participants`, and `Assertion.Participant` now all use the unified type

#### Property Vocabularies

##### Media Properties
- **New `media-properties.glx` vocabulary** - Standard properties for media entities:
  - `subjects` - People depicted or referenced in the media (multi-value)
  - `width`, `height` - Dimensions in pixels for images/video
  - `duration` - Duration in seconds for audio/video
  - `file_size` - File size in bytes
  - `crop` - Crop coordinates as integers (top, left, width, height)
  - `medium` - Physical medium type (photograph, document, film)
  - `original_filename` - Original filename before import
  - `photographer` - Person who created the media
  - `location` - Place where the media was created
- Added `Properties` field to Media struct and `MediaProperties` to GLXFile

##### Repository Properties
- **New `repository-properties.glx` vocabulary** - Standard properties for repository entities:
  - `phones` - Phone numbers for the repository (multi-value)
  - `emails` - Email addresses for the repository (multi-value)
  - `fax` - Fax number
  - `access_hours` - Hours of operation or access availability
  - `access_restrictions` - Any restrictions on access (appointment required, subscription, etc.)
  - `holding_types` - Types of materials held as YAML arrays (multi-value)
  - `external_ids` - External identifiers from other systems like FamilySearch, WikiTree (multi-value)
- Added `RepositoryProperties` to GLXFile
- Moved contact fields (phone, email) from direct entity fields to `properties`

##### Citation Properties
- **New `citation-properties.glx` vocabulary** - Standard properties for citation entities:
  - `locator` - Location within source (consolidates former `page` and `locator` direct fields; GEDCOM PAGE)
  - `text_from_source` - Transcription or excerpt of relevant text (moved from direct entity field)
  - `source_date` - Date when the source recorded the information (from GEDCOM DATA.DATE)
- Added `Properties` field to Citation struct, `CitationProperties` to GLXFile, and vocabulary specification section

##### Source Properties
- **New `source-properties.glx` vocabulary** - Standard properties for source entities:
  - `abbreviation` - Short reference name (from GEDCOM ABBR)
  - `call_number` - Repository catalog number (from GEDCOM CALN)
  - `events_recorded` - Types of events documented by this source (multi-value, from GEDCOM EVEN)
  - `agency` - Responsible agency (from GEDCOM AGNC)
  - `coverage` - Geographic/temporal scope of source content
  - `external_ids` - External system identifiers (multi-value)
- Added `Properties` field to Source struct, `SourceProperties` to GLXFile, and `source-properties.schema.json`

##### Multi-Value Property Support
- **Added `multi_value` field to PropertyDefinition** - Properties can now be marked as supporting multiple values
- Validation correctly handles array values for multi-value properties

#### GEDCOM Import

##### Media/OBJE Import
- **Implemented inline OBJE handling for all record types** - Media references and embedded OBJE records on individuals, events, sources, families, submitters, census records, and person property tags are now imported (previously only marriage events and top-level OBJE were handled)
- Added `handleOBJE` shared helper for XRef references, GEDCOM 7.0 `@VOID@` pointers, and embedded OBJE
- Added BLOB data handling, URL-type multimedia import, and OBJE processing in `extractEventDetails`
- Torture test media import improved from 2 to 32 entities (100% coverage)

##### Media File Import
- **Media files are now copied into the archive during GEDCOM import** - Relative FILE paths copied to `media/files/`; BLOB data decoded and written to files
- Media URIs rewritten to archive-relative paths; URL and absolute path references left as-is
- Filename deduplication with counter suffixes; missing source files produce warnings, not errors

##### Census (CENS) Support
- **Implemented CENS tag handling for individual and family records** - Census records treated as evidence sources, not events
- Each CENS creates a Source (type: `census`) and Citation; extracts PLAC for temporal `residence` property
- Family-level CENS applies census data to both husband and wife
- Added `createPropertyAssertionWithCitations()` helper

##### Vocabulary-Driven Tag Resolution
- **Added `gedcom` field to `PropertyDefinition` struct** - Property vocabulary entries can now declare their corresponding GEDCOM tag
- Added GEDCOM tag mappings to all 6 property vocabularies (person, event, citation, source, repository, media)
- Added `external_ids` to person-properties.glx and event detail properties (`age_at_event`, `cause`, `event_subtype`) to event-properties.glx
- Added `GEDCOMIndex` reverse lookup infrastructure; replaced hardcoded mappings with vocabulary-driven lookups
- Added `gedcom` field and `fields`/`FieldDefinition` to all 8 property vocabulary JSON schemas
- Updated vocabulary specification documentation with `gedcom` field and GEDCOM column

##### Evidence and Citation Handling
- **Assertions require citations** - Assertions are now only created when SOUR tags are present
- **Embedded citation support** - SOURCE_CITATION without pointer creates synthetic Source entity
- **Properties-based storage** - Source, media, and citation tags now stored in vocabulary-defined `properties` instead of notes
- **Citation linkage on media** - SOUR on OBJE now properly links via `citation.Media`

#### Validation
- **Place hierarchy cycle detection** - Validates that place parent references don't form cycles (e.g., A -> B -> C -> A). Reports exactly one error per cycle with the full cycle path in the error message.

#### Place Entity
- **Moved `jurisdiction`, `place_format`, and `alternative_names` to properties** - Now stored as vocabulary-defined properties instead of dedicated entity fields. `alternative_names` simplified from `AlternativeName`/`DateRange` types to a temporal, multi-value string property.

#### Relationship Entity
- **Consolidated `description` into `properties.description`** - Removed as a top-level field

#### Source Entity
- **Consolidated `creator` field into `authors`** - Removed `creator` from spec, schema, and Go types

#### Library Package Restructuring
- **Moved core library from `glx/lib/` to `go-glx/`** - The library is now at the repository root for clean external imports
- **Renamed package from `lib` to `glx`** - External consumers import as `glxlib "github.com/genealogix/glx/go-glx"` and use `glxlib.GLXFile`, `glxlib.NewSerializer()`, etc.
- Updated all CLI files to use new import path and `glxlib.` qualifier

#### CLI
- **Changed `glx import` default format** - Now defaults to multi-file (`-f multi`) instead of single-file

#### JSON Schema URLs
- **Standardized schema `$id` URLs** - All JSON schemas now use consistent GitHub raw content URLs; removed references to `schema.genealogix.io` and `genealogix.org` domains

#### Documentation
- **Rewrote Migration from GEDCOM guide** - Expanded from a skeleton to a comprehensive guide covering all supported GEDCOM tags, CLI flags, field mapping tables, common challenges, troubleshooting, and GEDCOM 5.5.1 vs 7.0 differences
- **Clarified vocabulary file location is flexible** - Spec, quickstart, and vocabulary docs now emphasize that vocabulary files can live anywhere in the archive, not only in `vocabularies/`
- **Streamlined Introduction** - Simplified [1-introduction.md](specification/1-introduction.md) from 120 to 63 lines
- **Restructured Core Concepts** - Reorganized [2-core-concepts.md](specification/2-core-concepts.md) to emphasize flexibility; new section order: Archive-Owned Vocabularies → Entity Relationships → Data Types → Properties → Assertions → Evidence Chain → Collaboration
- **Merged Data Types into Core Concepts** - Integrated `6-data-types.md` as section 3; deleted standalone file
- **Added Glossary to specification** - Moved from `docs/guides/glossary.md` to [specification/6-glossary.md](specification/6-glossary.md) with "Property" and "Temporal Property" definitions
- Updated table of contents and fixed broken links after restructuring
- Removed `.md` extensions from ~40 internal links for VitePress compatibility
- Standardized GEDCOM mapping table headers across all 8 entity type files
- Added Properties sections to [place.md](specification/4-entity-types/place.md) and [relationship.md](specification/4-entity-types/relationship.md)
- Standardized entity file structure across all entity type docs
- Added Schema Reference sections to event, relationship, place, citation, and repository entity docs
- Added naming convention note (hyphens for file/entry names, underscores for YAML section keys) to core concepts
- Moved "Change Tracking with Git" section before "Next Steps" in core-concepts
- Removed 59 file path comments from YAML code blocks
- Standardized validation rules to reference vocabularies with links
- Added `participants` to all event examples that were missing the required field
- **Enhanced VitePress sidebar** - Core Concepts promoted to its own collapsible sidebar section with 8 direct anchor links
- **Updated quickstart.md** - Examples updated to reflect schema changes
- **Updated best-practices.md** - Assertion examples updated to use typed `subject` reference and `property` field

### Fixed

#### Specification
- Fixed Place hierarchy example that used duplicate YAML top-level keys
- Fixed examples using incorrect field names throughout specification (`description` → `notes`, `value` → `notes`, `file:` → `uri:`, `death_year` → `died_on`, `married_on` → `born_on`, `residence_dates` → `residence`, `registration_district` → `district`)
- Fixed assertion example using invalid date format (`circa 1825` → `ABT 1825`)
- Removed undocumented `birth_surname` from person name example
- Fixed broken anchor link in repository.md (`#repository-properties` → `#repository-properties-vocabulary`)
- Standardized all event examples to use `subject` role consistently (replaced remaining `principal` usages)
- Fixed Event `date` field type from `string/object` to `string` (object form was never documented)
- Fixed Event See Also to say Person "participates in events" instead of "contains event references"
- Fixed broken relative links in `1-introduction.md` and `specification/README.md`
- Fixed `residence` reference type example in `2-core-concepts.md` to use temporal format
- Added minimum participant count (at least 2) to relationship fields table
- Removed stale `Created At` and `Created By` glossary entries
- Fixed glossary Event and Event Type definitions that incorrectly included occupation and residence
- Fixed labels: "Event/Fact" → "Event", "living status" → "birth/death dates"
- Replaced `living: true` boolean example with non-misleading property names
- Replaced "occupation" with "immigration" as event type example in 3 locations
- Fixed Event key properties ("description" → "notes") and Media key properties ("file path" → "URI") in entity-types README
- Fixed place types count from 14 to 15; added missing `locality` to place-types.glx standard vocabulary
- Fixed vocabulary directory structure example in core-concepts

#### GEDCOM Import
- **Repository deduplication** - Repositories with the same name and location are now deduplicated during import
- **Dependency-ordered record processing** - Records now grouped by type and processed in dependency order
- **Repository-to-source linking** - Sources now correctly link to their repository even when REPO records appear after SOUR records in the file
- **NOTE reference resolution** - Shared NOTE records now resolved to actual text content during import
- **CONT/CONC text continuation** - Long text fields spanning multiple lines now properly combined
- **CR line ending support** - GEDCOM files using CR-only line endings (old Mac Classic format) now import correctly

#### Code Quality & Robustness
- **`unmarshalVocab` now returns error on missing YAML key** - Previously silently returned nil when the expected top-level key was absent, causing downstream validation to think no vocabulary entries exist
- **`appendMediaID` safe type assertion** - Now handles `[]any` (from YAML deserialization) instead of panicking on a bare type assertion to `[]string`
- **`extensionFromMimeType` deterministic output** - MIME types with multiple extensions (`.jpg`/`.jpeg`, `.tif`/`.tiff`) now return a consistent preferred extension instead of random map iteration order
- **Directory emptiness check error handling** - `isDirectoryEmpty` now only treats `io.EOF` as "empty", not all errors (permissions, I/O failures now properly reported)
- **Media file copy error handling** - `copyMediaFile` now checks `os.IsNotExist` before fallback to URL-decoded paths, preserving original errors for permissions/disk issues
- **BLOB character validation** - `decodeGEDCOMBlob` now validates characters are in valid GEDCOM BLOB range ('.' to 'm') before decoding, preventing silent corruption
- **EXID ID validation** - GEDCOM external ID extraction now validates `id` field exists before use, skipping entries without usable IDs
- **Event Properties initialization** - `extractEventDetails` now ensures `event.Properties` map is initialized before writing, preventing panics
- **Archive validation wiring** - `LoadArchiveWithOptions` now correctly passes `schemaValidate` flag to serializer for referential integrity validation
- **Property vocabulary documentation** - Fixed `value_type` and `reference_type` field requirements (marked "No*" instead of "Yes*" to match "exactly one required" constraint)
- **Test assertion completeness** - `TestRunValidate_MediaFileMissing` now captures stdout and verifies warning is actually produced

#### CLI
- **`glx validate` single file behavior** - Validating a single file now only validates that file's structure instead of loading the entire current directory. Cross-reference validation is skipped for single files with a warning message. Directory validation still performs full cross-reference checks.

### Removed

- **Removed `glx check-schemas` CLI command** - Moved to `make check-schemas` Makefile target; this is a repo-internal dev tool, not a user-facing command

#### Citation Entity
- Removed `data_date`, `page`, `locator`, and `text_from_source` direct fields — consolidated into `properties`

#### Source Entity
- Removed `citation`, `coverage`, and `creator` direct fields (`creator` consolidated into `authors`)

#### Event Entity
- Removed `description` field (use `properties.description`) and `tags` field

## [0.0.0-beta.2] - 2025-11-25

### Added

#### GEDCOM Import (lib)
- **GEDCOM 5.5.1 support** - Import standard GEDCOM 5.5.1 files
- **GEDCOM 7.0 support** - Import GEDCOM 7.0 with new features
- **GEDCOM 5.5.5 support** - Import GEDCOM 5.5.5 specification samples
- **Two-pass conversion** - Entities first, then families for proper relationship handling
- **Evidence chain mapping** - GEDCOM SOUR tags → GLX Citations → GLX Assertions
- **Place hierarchy building** - Parse place strings into hierarchical Place entities
- **Geographic coordinates** - Extract MAP/LATI/LONG coordinates from GEDCOM
- **Shared notes** - Support for both GEDCOM 7.0 SNOTE and GEDCOM 5.5.1 NOTE records
- **External IDs** - Import GEDCOM 7.0 EXID tags (wikitree, familysearch, etc.)
- **Comprehensive test coverage** - 33 GEDCOM test files (5.5.1, 5.5.5, 7.0) successfully imported
- **Large file support** - Tested with files containing thousands of persons and events
- **Edge case handling** - Empty families, self-marriages, same-sex marriages, unknown genders
- **Character encoding support** - ASCII, UTF-8, Windows CP1252 (CRLF and LF)

#### GLX Serializer (lib)
- **Single-file serialization** - Convert GLX archives to single YAML files
- **Multi-file serialization** - Entity-per-file structure with random IDs
- **Archive loading** - Load both single-file and multi-file GLX archives
- **Vocabulary embedding** - Embed standard vocabularies using go:embed
- **Vocabulary loading from directory** - Load vocabularies from multi-file archives
- **ID generation** - Random 8-character hex IDs for entity filenames
- **EntityWithID wrapper** - Preserve entity IDs in multi-file format using _id field
- **Collision detection** - Retry logic for filename generation
- **Configurable validation** - Optional validation before serialization
- **12 standard vocabularies** embedded in binary
- **Round-trip preservation** - Single→Multi→Single conversions preserve all data

#### CLI Commands (glx)
- **`glx import`** - Import GEDCOM files to GLX format
  - Single-file and multi-file output formats
  - Optional vocabulary inclusion (default: true)
  - Optional validation (default: true)
  - Verbose mode with import statistics
  - Supports both GEDCOM 5.5.1 and 7.0
- **`glx split`** - Convert single-file GLX to multi-file format
  - Splits archive into entity-per-file structure
  - Includes standard vocabularies
  - Preserves entity IDs
- **`glx join`** - Convert multi-file GLX to single-file format
  - Combines multi-file archive into single YAML
  - Restores entity IDs from _id fields

#### Schema Enhancements
- **Properties field added** to 5 entity types for extensibility:
  - Source - Store GEDCOM ABBR, EXID, custom tags
  - Citation - Store event type cited, role, entry date
  - Repository - Store FAX, additional contacts, EXID
  - Media - Store crop coordinates, alternative titles, EXID
  - Assertion - Store assertion metadata
- **Backward compatible** - Properties fields are optional with omitempty

#### Project Organization
- **`.claude/plans/`** directory for all planning documents
- **`CLAUDE.md`** project context guide for AI assistants
- **Plans README** documenting all planning files and current status
- Moved all planning docs from `docs/` to `.claude/plans/`

#### Vocabularies & Standards
- **Developer documentation** - GEDCOM import docs in `glx/lib/doc.go`
- **User documentation** - Updated [Migration from GEDCOM Guide](docs/guides/migration-from-gedcom.md)
  - Automated import instructions
  - Testing and validation procedures
  - Import result expectations

### Fixed

#### GEDCOM Import
- **Malformed line recovery** - Parser now handles MyHeritage export bug
  - Recovers from NOTE fields with missing CONT/CONC prefixes
  - Gracefully imports files with HTML-formatted notes
  - Test case: queen.ged (4,683 persons, line 15903 missing CONT prefix)
- **Family event handling** - Added missing ANUL, DIVF, EVEN to case statement
- **Place type references** - Fixed gedcom_place.go to use "state" instead of "state_province"

#### Vocabularies
- **Event types vocabulary** - Fixed probate description ("Probate of estate" not "of will")
- **Place types vocabulary** - Removed duplicate state_province alias (use "state" instead)
- **Schema categories** - Updated allowed categories in vocabulary schemas
  - Event types: Added "legal", "migration"; changed "custom" → "other"
  - Place types: Added "institution"; changed "custom" → "other"
- **Source types vocabulary** - Added to embedded vocabularies (was missing)

#### Code Quality
- **Clean architecture** - Removed file I/O from library layer
  - Moved importGEDCOMFromFile to test helpers (gedcom_test_helpers.go)
  - CLI handles file operations, lib works with io.Reader
  - Better separation of concerns
- **File organization** - Renamed gedcom_7_0.go → gedcom_shared.go (more accurate)

#### Testing & CI
- **Multi-file vocabulary loading** - Fixed LoadMultiFile to properly load vocabularies from directory
- **Vocabulary preservation** - Vocabularies now correctly preserved in round-trip conversions
- **CI test coverage** - Updated GitHub Actions to explicitly run all tests
  - Large file tests (habsburg.ged: 34,020 persons)
  - Added 15-minute timeout for comprehensive test runs
  - No tests skipped in CI (no -short flag)
- **Test documentation** - Fixed queen.ged README with correct software attribution
- **GEDCOM TITL handling** - Now uses proper `PersonPropertyTitle` constant instead of hardcoded string
- **GEDCOM name fields** - Only populate `name.fields` from explicit GEDCOM substructure tags (GIVN, SURN, etc.), not inferred from parsing the name string
- **Test data consistency** - All testdata files updated to use unified name format

### Removed

#### Attribute Event Types
- **Removed attribute-type events from schema** - Events are now strictly discrete occurrences with participants
  - Removed from event.schema.json enum: `residence`, `occupation`, `title`, `nationality`, `religion`, `education`
  - Removed `census` from event-types.glx vocabulary
  - These attributes are now represented as temporal properties on Person entities
- **Removed CENS (Census) event handling** - Census records are skipped during GEDCOM import (TODO: re-implement as citations supporting property assertions)
- **Converted RESI (Residence) to temporal property** - GEDCOM RESI tags now create temporal `residence` properties on Person entities instead of events

#### Quality Ratings Support
- **Removed `quality_ratings` vocabulary** - The GEDCOM 0-3 Quality Assessment scale was removed from the GLX specification
  - Deleted `quality-ratings.glx` vocabulary file
  - Deleted `quality-ratings.schema.json` schema file
  - Removed `quality` field from Citation entity
  - Removed `QualityRating` type from Go code
- **Removed auto-generated assertion confidence** - GEDCOM imports no longer auto-populate assertion confidence levels
  - Confidence levels should reflect researcher judgment, not be inferred from QUAY values
  - GEDCOM QUAY tags are now preserved in citation notes (e.g., `GEDCOM QUAY: 2`)

#### Assertion Entity Fields
- **Removed `evidence_type` field** - Evidence quality classification belongs on citations, not assertions
- **Removed `type` field** - Redundant with `claim` field and `tags` for categorization
- **Removed `research_notes` field** - Consolidated into single `notes` field

#### Provenance Fields (All Entities)
- **Removed `modified_at`, `modified_by`, `created_at`, `created_by` fields** - Redundant with git history; use `git log` and `git blame` instead

### Changed

#### Person Properties Schema
- **Unified `name` property** - Replaced fragmented name properties with single unified property
  - Old: Separate `given_name`, `family_name` properties
  - New: Single `name` property with `value` and optional `fields` breakdown
  - Format: `name: { value: "John Smith", fields: { given: "John", surname: "Smith" } }`
  - Supports temporal lists for name changes over time
  - Fields include: `prefix`, `given`, `nickname`, `surname_prefix`, `surname`, `suffix`
- **Added `title` property** - Nobility or honorific titles (temporal, like occupation)
  - Properly handles GEDCOM TITL tag imports
  - Added `PersonPropertyTitle` constant

#### Vocabulary Updates
- **person_properties vocabulary** - Updated to reflect unified name structure
  - `name` property now includes `fields` sub-schema for structured breakdown
  - Added `title` property definition

#### Other
- **Documentation structure** - Separated user docs (docs/) from planning docs (.claude/plans/)

### Technical Details

**GEDCOM Import Coverage:**
- 100% critical features implemented
- 94% high-priority features implemented
- PRODUCTION-READY status
- Comprehensive gap analysis completed

**Serializer Features:**
- Uses crypto/rand for ID generation
- 32 bits of randomness per ID (4.3 billion possible values)
- Collision probability: ~1 in 400,000 with 10,000 entities
- EntityWithID wrapper pattern for multi-file format
- All 12 standard vocabularies embedded with go:embed

**Testing:**
- All existing tests passing
- 48 new test cases for serializer
- 33 GEDCOM files tested for import (100% coverage of test files)
- Full round-trip serialization/deserialization tests
- Vocabulary preservation tests for both single-file and multi-file formats
- Comprehensive unit and integration tests
- Large file stress tests (3000+ persons, 4000+ events)

## [0.0.0-beta.1] - 2025-11-18

### Fixed
- Fixed GitHub release workflow to build on beta tags (`v*.*.*-beta*` pattern)
- Fixed VitePress build by adding `shiki` dependency to `website/package.json`

### Changed
- Removed roadmap section from README (no longer maintaining public roadmap)

### Removed
- Removed archive folder containing old planning documents

## [0.0.0-beta.0] - 2025-11-14

### Added

#### Specification & Standards
- Complete GENEALOGIX specification defining modern, evidence-first genealogy data standard
- 9 core entity types with full JSON Schema definitions:
  - Person (individuals with biographical properties)
  - Relationship (family connections with types and dates)
  - Event (life events with sources and locations)
  - Assertion (evidence-backed claims with quality assessment)
  - Citation (evidence references with source quotations)
  - Source (primary/secondary evidence documentation)
  - Repository (physical storage information)
  - Place (geographic locations with coordinate data)
  - Participant (individuals involved in events)
- Repository-owned controlled vocabularies for extensibility
- Git-native architecture for version control and collaboration
- YAML-based human-readable format with schema validation

#### CLI Tool (`glx`)
- `glx init`: Initialize new GLX repositories with optional single-file mode
- `glx validate`: Comprehensive validation with:
  - Schema compliance checking against JSON Schemas
  - Cross-reference integrity verification across all files
  - Vocabulary constraint validation
  - Detailed error reporting with file/line locations
- `glx check-schemas`: Utility for verifying schema metadata and structure
- Support for both directory-based and single-file archives
- Cross-file entity resolution and validation

#### Documentation & Examples
- Comprehensive specification documentation (6 core documents)
- Complete examples demonstrating various use cases:
  - Minimal single-file archive
  - Basic family structure with multiple generations
  - Complete family with all entity types
  - Participant assertions workflow
  - Temporal properties and date ranges
- Development guides covering:
  - Architecture and design decisions
  - Schema development practices
  - Testing framework and test suite structure
  - Local development environment setup
- User guides including:
  - Quick-start guide for new users
  - Best practices and recommendations
  - Common pitfalls and troubleshooting
  - Manual migration guide for converting from GEDCOM format
  - Glossary of key terms and concepts

#### Testing & Quality Assurance
- Comprehensive test suite with:
  - Valid example fixtures demonstrating correct usage
  - Invalid example fixtures testing error handling
  - Cross-reference validation tests
  - Vocabulary constraint tests
  - Schema compliance validation tests
- Automated CI/CD pipeline using GitHub Actions
- Full code coverage reporting

#### Project Infrastructure
- Apache 2.0 open-source license
- Community guidelines and code of conduct
- Contributing guidelines for developers
- GitHub issue and discussion templates
- Development container configuration for consistent environments
- Pre-configured VitePress documentation site


