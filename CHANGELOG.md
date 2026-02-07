---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.0-beta.3] - 2026-02-06

### Added

#### Media as Assertion Evidence
- **Added `media` as a third evidence option for assertions** - Assertions can now reference media entities directly as evidence, alongside citations and sources
- Useful for direct visual evidence like gravestone photos, handwritten documents, or family photographs
- JSON schema `anyOf` evidence constraint updated to include `media`
- Media entities remain linkable to sources (via `media.source`) and citations (via `citation.media`) as before

#### Media File Import
- **Media files are now copied into the archive during GEDCOM import** - Relative FILE paths from GEDCOM OBJE records are copied from the GEDCOM source directory into `media/files/` in the output archive
- **BLOB data is decoded and written to files** - GEDCOM 5.5.1 deprecated BLOB binary data is decoded using the GEDCOM custom encoding and written to `media/files/`
- **Media URIs rewritten to archive-relative paths** - Relative FILE paths are rewritten from their GEDCOM-relative form (e.g., `photos/portrait.jpg`) to `media/files/portrait.jpg`
- **URL and absolute path references left as-is** - HTTP, FTP, mailto, file://, and absolute paths are preserved unchanged in Media.URI
- **Filename deduplication** - When multiple OBJE records reference files with the same basename, filenames are deduplicated with counter suffixes (e.g., `photo-2.jpg`)
- **Missing source files produce warnings, not errors** - File copy failures are non-fatal; URL-decoded path fallback for GEDCOM 7.0 percent-encoded filenames

#### GEDCOM Media/OBJE Import
- **Implemented inline OBJE handling for persons and events** - Media references (XRef) and embedded OBJE records on individuals, birth/death/marriage events, and other life events are now imported (previously only marriage events and top-level OBJE were handled)
- **Added `handleOBJE` shared helper** - Handles XRef references, GEDCOM 7.0 `@VOID@` pointers with subrecords, and embedded OBJE without XRef
- **Added OBJE handling in `extractEventDetails`** - All event types (individual, family, divorce, etc.) now automatically process OBJE subrecords
- **Added BLOB data handling** - GEDCOM 5.5.1 BLOB binary data on top-level OBJE records is now recognized and blob size recorded in media properties
- **URL-type multimedia imported** - OBJE records with `FORM URL` and http/ftp/mailto FILE references are now correctly imported as media entities
- **OBJE handling on all record types** - Added embedded OBJE support for source records, family records, submitter records, census records, and person property tags (e.g., OCCU). Citation OBJE now handles both references and embedded media.
- Torture test media import improved from 2 to 32 entities (100% coverage, 0% loss)

#### GEDCOM Census (CENS) Support
- **Implemented CENS tag handling for individual and family records** - Census records are no longer silently skipped during GEDCOM import
- Census records are treated as evidence sources, not events: each CENS creates a Source (type: `census`) and Citation
- When CENS has SOUR sub-records, uses existing sources; otherwise creates a synthetic census source (title from TYPE or DATE)
- Extracts PLAC to set temporal `residence` property on persons with date from DATE sub-record
- Creates property assertions for residence backed by census citations
- Family-level CENS applies census data to both husband and wife
- Added `createPropertyAssertionWithCitations()` helper for creating assertions with pre-extracted citation IDs

#### GEDCOM Tag Mapping in Vocabularies
- **Added `gedcom` field to `PropertyDefinition` struct** - Property vocabulary entries can now declare their corresponding GEDCOM tag
- **Added GEDCOM tag mappings to property vocabularies**:
  - `person-properties.glx`: occupation (OCCU), title (TITL), religion (RELI), education (EDUC), nationality (NATI), caste (CAST), ssn (SSN), external_ids (EXID)
  - `event-properties.glx`: age_at_event (AGE), cause (CAUS), event_subtype (TYPE)
  - `citation-properties.glx`: locator (PAGE), text_from_source (TEXT), source_date (DATE)
  - `source-properties.glx`: abbreviation (ABBR), publication_info (PUBL), call_number (CALN), events_recorded (EVEN), agency (AGNC), external_ids (EXID)
  - `repository-properties.glx`: phones (PHON), emails (EMAIL), external_ids (EXID)
  - `media-properties.glx`: medium (MEDI), crop (CROP)
- **Added `external_ids` property to person-properties.glx** - For GEDCOM 7.0 EXID tags on person records
- **Added event detail properties to event-properties.glx** - `age_at_event`, `cause`, `event_subtype` properties now defined in vocabulary
- **Added `GEDCOMIndex` reverse lookup infrastructure** - Builds `map[GEDCOMTag]GLXKey` indices from loaded vocabularies at import initialization, covering event types, person/event/citation/source/repository/media properties
- **Added `gedcom` field to all property vocabulary JSON schemas** - All 8 property vocabulary schemas now include the optional `gedcom` field in their PropertyDefinition
- **Added `fields`/`FieldDefinition` to all property vocabulary JSON schemas** - All 8 property vocabulary schemas now support structured property components with sub-fields (previously only media-properties had this)
- **Updated vocabulary specification documentation** - Property Definition Structure table and examples now document the `gedcom` field; person properties table includes GEDCOM column; event properties listing updated with new standard properties

### Changed

#### Assertion Entity Improvements

##### Renamed `claim` to `property`
- **Renamed `claim` field to `property`** - The field name now matches the vocabulary terminology (property vocabularies)
- **Updated JSON schema** - Changed field name from `claim` to `property`
- **Updated Go types** - `Assertion.Claim` is now `Assertion.Property`
- **Updated all specification examples and example archives** to use new field name

##### Typed Subject Reference
- **Changed `subject` from string to typed reference object** - Prevents entity ID collisions in large archives
- **Subject now uses oneOf pattern** - Must specify exactly one of: `person`, `event`, `relationship`, or `place`
- **Before**: `subject: person-john-smith`
- **After**: `subject: { person: person-john-smith }` or `subject: { event: marriage-1880 }`
- **Added `EntityRef` Go type** - New type with `Type()` and `ID()` helper methods
- **Updated validation** - EntityRef validation ensures exactly one field is set and referenced entity exists

#### Vocabulary Consolidation

##### Adoption Modeling
- **Removed redundant `adoption` relationship type** - Use `adoptive-parent-child` relationship type instead
- **Clarified adoption semantics** - `adoption` event type records the legal proceeding; `adoptive-parent-child` relationship type models the ongoing parent-child bond
- **Updated relationship.md** - Added comprehensive example showing adoption event linked to adoptive-parent-child relationship via `start_event`
- **Removed `RelationshipTypeAdoption` constant** - Code now uses only `RelationshipTypeAdoptiveParentChild`

##### Godparent Modeling
- **Clarified godparent dual usage** - Participant role `godparent` is for event participation (baptism sponsor); relationship type `godparent` is for the ongoing bond
- **Added `godchild` participant role** - For use in godparent relationships
- **Updated relationship.md** - Added comprehensive example showing baptism event with godparent role linked to godparent relationship

#### Type System

##### Unified Participant Type
- **Unified participant types** - Consolidated `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into single `Participant` struct
  - All three types had identical structure: `person`, `role`, `notes` fields
  - Reduces code duplication and simplifies the API
  - `Event.Participants`, `Relationship.Participants`, and `Assertion.Participant` now all use the unified `Participant` type

#### Property Vocabularies

##### Media Properties
- **New `media-properties.glx` vocabulary** - Standard properties for media entities:
  - `subjects` - People depicted or referenced in the media (multi-value)
  - `width`, `height` - Dimensions in pixels for images/video
  - `duration` - Duration in seconds for audio/video
  - `file_size` - File size in bytes
  - `crop` - Crop coordinates (top, left, width, height)
  - `medium` - Physical medium type (photograph, document, film)
  - `original_filename` - Original filename before import
  - `photographer` - Person who created the media
  - `location` - Place where the media was created
- **Added `Properties` field to Media struct and `MediaProperties` to GLXFile**

##### Repository Properties
- **New `repository-properties.glx` vocabulary** - Standard properties for repository entities:
  - `phones` - Phone numbers for the repository (multi-value)
  - `emails` - Email addresses for the repository (multi-value)
  - `fax` - Fax number
  - `access_hours` - Hours of operation or access availability
  - `access_restrictions` - Any restrictions on access (appointment required, subscription, etc.)
  - `holding_types` - Types of materials held (microfilm, digital, books, etc.) (multi-value)
  - `external_ids` - External identifiers from other systems like FamilySearch, WikiTree (multi-value)
- **Added `RepositoryProperties` to GLXFile**
- **Moved contact fields to properties** - Repository contact information (phone, email) now stored in `properties` instead of direct entity fields

##### Citation Properties
- **New `citation-properties.glx` vocabulary** - Standard properties for citation entities:
  - `locator` - Location within source (consolidates former `page` and `locator` direct fields; GEDCOM PAGE)
  - `text_from_source` - Transcription or excerpt of relevant text (moved from direct entity field)
  - `source_date` - Date when the source recorded the information (from GEDCOM DATA.DATE)
- **Added `Properties` field to Citation struct and `CitationProperties` to GLXFile**

##### Source Properties
- **New `source-properties.glx` vocabulary** - Standard properties for source entities:
  - `abbreviation` - Short reference name (from GEDCOM ABBR)
  - `call_number` - Repository catalog number (from GEDCOM CALN)
  - `events_recorded` - Types of events documented by this source (multi-value, from GEDCOM EVEN)
  - `agency` - Responsible agency (from GEDCOM AGNC)
  - `coverage` - Geographic/temporal scope of source content
  - `external_ids` - External system identifiers (multi-value)
- **Added `Properties` field to Source struct, `SourceProperties` to GLXFile, and `source-properties.schema.json`**

##### Multi-Value Property Support
- **Added `multi_value` field to PropertyDefinition** - Properties can now be marked as supporting multiple values
- **Validation support for multi-value properties** - Validator correctly handles array values for multi-value properties

#### GEDCOM Import
- **Vocabulary-driven tag resolution** - Replaced hardcoded event type and property mappings with `GEDCOMIndex` lookups from vocabulary `gedcom` fields. Collapsed individual property switch cases into generic `handlePersonPropertyTag()` handler.
- **Assertions require citations** - Assertions are now only created when SOUR tags are present. Property values are still stored directly on entities; assertions exist to document evidence.
- **Embedded citation support** - SOURCE_CITATION without pointer (`SOUR description text`) now creates synthetic Source entity per GEDCOM spec recommendation
- **Properties-based storage** - Source tags (ABBR, CALN, EVEN, AGNC, EXID), media tags (MEDI, CROP), and citation data now stored in vocabulary-defined `properties` instead of notes
- **Citation linkage on media** - SOUR on OBJE now properly links via `citation.Media` instead of dumping to notes

#### Validation
- **Place hierarchy cycle detection** - Validates that place parent references don't form cycles (e.g., A -> B -> C -> A). Reports exactly one error per cycle with the full cycle path in the error message.

#### Place Entity
- **Moved `jurisdiction` and `place_format` to properties** - These rarely-used fields are now stored in `properties` instead of dedicated entity fields
  - `properties.jurisdiction` - Formal jurisdiction identifier or code (e.g., ISO 3166, FIPS code)
  - `properties.place_format` - Standard format string for place hierarchy (GEDCOM PLAC.FORM style)
- **Updated place-properties vocabulary** - Added `jurisdiction` and `place_format` property definitions

#### CLI
- **Changed `glx import` default format** - Now defaults to multi-file (`-f multi`) instead of single-file

#### JSON Schema URLs
- **Standardized schema `$id` URLs** - All JSON schemas now use consistent GitHub raw content URLs
  - Format: `https://raw.githubusercontent.com/genealogix/glx/main/specification/schema/v1/{name}.schema.json`
  - Removed references to `schema.genealogix.io` and `genealogix.org` domains

#### Documentation

##### Specification Structure
- **Streamlined Introduction** - Simplified [1-introduction.md](specification/1-introduction.md) from 120 to 63 lines
  - New concise structure: What is GENEALOGIX? → Why GENEALOGIX? → Who is it for? → Comparison Table → Getting Started → Community
- **Restructured Core Concepts** - Reorganized [2-core-concepts.md](specification/2-core-concepts.md) to emphasize flexibility as primary differentiator
  - **New section order**: Archive-Owned Vocabularies → Entity Relationships → Data Types → Properties → Assertions → Evidence Chain → Collaboration
  - Removed ~115 lines of Git implementation details
- **Merged Data Types into Core Concepts** - Integrated [6-data-types.md](specification/6-data-types.md) as section 3 of Core Concepts; deleted standalone file
- **Updated specification table of contents** - [specification/README.md](specification/README.md) now accurately reflects 5-section structure
- **Added Glossary to specification** - Moved glossary from `docs/guides/glossary.md` to [specification/6-glossary.md](specification/6-glossary.md)
  - Added "Property" and "Temporal Property" term definitions
- **Fixed broken links** - Updated 7 links across 6 files after merging data types; fixed `#evidence-hierarchy` → `#evidence-chain` anchors in 4 files; fixed `birth_date` → `born_on` in examples

##### Specification Audit Quick Wins
- **Removed `.md` extensions from internal links** - Updated ~40 internal links across 12 specification files for VitePress compatibility
- **Standardized GEDCOM mapping table headers** - All 8 entity type files now use consistent "GLX Field | GEDCOM Tag | Notes" format
- **Fixed vocabulary directory structure example** - Removed misleading `property vocabularies/` subdirectory from [2-core-concepts.md](specification/2-core-concepts.md)
- **Added Properties sections to entity docs** - Inline property tables added to [place.md](specification/4-entity-types/place.md) and [relationship.md](specification/4-entity-types/relationship.md)
- Various small fixes: Person name recommended note, schema README clarification, Event/Fact → Event terminology

##### User Documentation Updates
- **Updated quickstart.md** - Examples updated to reflect schema changes: `description` → `properties.description`, `publication_info` → `properties.publication_info`, `locator`/`text_from_source` → `properties`, typed `subject` reference, `claim` → `property`
- **Updated best-practices.md** - Assertion examples updated to use typed `subject` reference and `property` field; citation example updated to use `properties.text_from_source`

##### Website Navigation
- **Enhanced VitePress sidebar** - Core Concepts promoted to its own collapsible sidebar section with 8 direct anchor links to subsections

#### Entity Type Documentation Structure
- **Standardized entity file structure** - All entity type documentation now follows consistent section order:
  1. Overview → File Format → Core Concepts → Fields → Usage Patterns → File Organization → GEDCOM Mapping → Validation Rules → See Also
- **Added File Format sections** to person.md, place.md, citation.md, repository.md
- **Added missing standard sections** to person.md (File Organization, GEDCOM Mapping, Validation Rules, expanded See Also)
- **Reorganized place.md** - Moved GEDCOM Mapping after File Organization, merged Examples into Usage Patterns
- **Cleaned up YAML code blocks** - Removed 59 file path comments (e.g., `# places/place-england.glx`) from examples

#### Vocabulary References
- **Standardized validation rules** - All entity types now reference vocabularies with links instead of hardcoded values or file paths
- **Added missing type validation rules** to media.md and repository.md

### Fixed

- **Added missing `locality` place type to standard vocabulary** - The GEDCOM importer's `inferPlaceType` function could assign `locality` to deeply-nested place components, but the term was not defined in `place-types.glx`

#### GEDCOM Import
- **Repository deduplication** - Repositories with the same name and location are now deduplicated during import
- **Dependency-ordered record processing** - Records now grouped by type and processed in dependency order: (1) Notes, Repositories, Schemas → (2) Sources, Media → (3) Individuals → (4) Families
- **Repository-to-source linking** - Sources now correctly link to their repository even when REPO records appear after SOUR records in the file
- **NOTE reference resolution** - Shared NOTE records (e.g., `NOTE @N123@`) now resolved to actual text content during import
- **CONT/CONC text continuation** - Long text fields spanning multiple lines using CONT/CONC tags now properly combined
- **CR line ending support** - GEDCOM files using CR-only line endings (old Mac Classic format) now import correctly

### Removed

#### Citation Entity
- **Removed `data_date` field** - Date the data was recorded is now captured in source or assertion context
- **Removed `page`, `locator`, and `text_from_source` direct fields** - Consolidated into `properties` (see Citation Properties under Changed)

#### Source Entity
- **Removed `citation` field** - Formatted citations belong in citation entities, not sources
- **Removed `coverage` field** - Geographic/temporal coverage can be captured in description or properties

#### Place Entity
- **Removed `alternative_names` field and supporting types** (`AlternativeName`, `DateRange`) - Feature removed for simplicity in initial release

#### Event Entity
- **Removed `description` field** - Use `properties.description` instead (eliminates redundancy with event-properties vocabulary)
- **Removed `tags` field** - Tags feature removed for simplicity in initial release

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


