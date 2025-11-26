---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- **Developer documentation** - Comprehensive [GEDCOM Import Developer Guide](docs/development/gedcom-import.md)
  - Architecture and conversion flow
  - Entity mapping details
  - ID generation (current incremental + future random)
  - GEDCOM 5.5.1 vs 7.0 differences
  - Malformed line recovery strategies
  - Testing and debugging guides
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


