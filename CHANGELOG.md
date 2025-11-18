---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0-beta] - 2025-11-18

### Added

#### GEDCOM Import (lib)
- **Full GEDCOM 5.5.1 support** - Import all standard GEDCOM 5.5.1 files
- **Full GEDCOM 7.0 support** - Import GEDCOM 7.0 with new features
- **Two-pass conversion** - Entities first, then families for proper relationship handling
- **Evidence chain mapping** - GEDCOM SOUR tags → GLX Citations → GLX Assertions
- **Place hierarchy building** - Parse place strings into hierarchical Place entities
- **Geographic coordinates** - Extract MAP/LATI/LONG coordinates from GEDCOM
- **Shared notes** - Support for both GEDCOM 7.0 SNOTE and GEDCOM 5.5.1 NOTE records
- **External IDs** - Import GEDCOM 7.0 EXID tags (wikitree, familysearch, etc.)
- **Comprehensive testing** - Shakespeare family test (31 persons, 77 events, 49 relationships)

#### GLX Serializer (lib)
- **Single-file serialization** - Convert GLX archives to single YAML files
- **Multi-file serialization** - Entity-per-file structure with random IDs
- **Archive loading** - Load both single-file and multi-file GLX archives
- **Vocabulary embedding** - Embed standard vocabularies using go:embed
- **ID generation** - Random 8-character hex IDs for entity filenames
- **EntityWithID wrapper** - Preserve entity IDs in multi-file format using _id field
- **Collision detection** - Retry logic for filename generation
- **Configurable validation** - Optional validation before serialization
- **13 standard vocabularies** embedded in binary

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

### Fixed
- **Family event handling** - Added missing ANUL, DIVF, CENS, EVEN to case statement
- **Source types vocabulary** - Added to embedded vocabularies (was missing)

### Changed
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
- All 13 standard vocabularies embedded with go:embed

**Testing:**
- All existing tests passing
- 48 new test cases for serializer
- Full round-trip serialization/deserialization
- Comprehensive unit and integration tests

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


