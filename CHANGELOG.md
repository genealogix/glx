---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.0-beta.10]

### Added

#### Date Handling
- **Non-Gregorian calendar support** — GEDCOM calendar escape sequences (`@#DJULIAN@`, `@#DHEBREW@`, `@#DFRENCH R@`) are now preserved as calendar prefixes on DateString values (e.g., `JULIAN 1731-03-15`). Previously, calendar designations were silently discarded. Gregorian remains the default (no prefix). Includes full roundtrip support on GEDCOM export.

#### CLI
- **Added `glx merge` command** — Combine two GLX archives by merging all content from a source into a destination. Duplicate entities are reported and skipped (destination version kept). Supports both single-file and multi-file archives, with `--dry-run` for preview
- **Added `glx migrate` command** - Converts deprecated person properties (`born_on`, `born_at`, `died_on`, `died_at`) to birth/death Event entities. Creates new events when none exist, merges date/place into existing events when they do, converts property assertions to event assertions, and removes the deprecated properties

### Removed

#### Person Properties
- **BREAKING**: Removed `born_on`, `born_at`, `died_on`, `died_at` person properties. Birth and death information now lives exclusively on Event entities of type `birth`/`death`. Use `glx migrate` to convert existing archives

---

## [0.0.0-beta.9] - 2026-03-29

### Added

#### Supply Chain Security
- **Dependency review on PRs** — `dependency-review-action` blocks PRs that introduce dependencies with moderate+ vulnerabilities
- **Renovate lockfile maintenance** — weekly lockfile refresh keeps transitive dependencies at latest allowed versions
- **govulncheck SARIF integration** — vulnerability results now upload to GitHub Code Scanning for richer triage
- **npm audit in CI** — website dependencies are audited for known vulnerabilities on every push and PR

#### CLI
- **Added `glx path` command** - Find the shortest relationship path between two people using breadth-first search. Traverses all relationship types (parent-child, marriage, sibling, godparent, etc.). Supports `--max-hops` to limit search depth and `--json` for machine-readable output
- **Added `glx cluster` command** - FAN (Friends, Associates, Neighbors) club analysis for brickwall research. Cross-references census households, shared events, and place overlap to identify associates of a target person. Ranks associates by connection strength with compound scoring. Supports `--place`, `--before`, `--after` filters and `--json` output
- **Added `glx census add` command** - Bulk census import helper that generates GLX entities from a structured YAML template. Reads census year, location, household members, and citation details to produce person records, a census event with participants, source/citation entities, and evidence-based assertions. Supports matching members to existing archive persons by ID or name, `--dry-run` preview, and FAN notes
- **Added `conflicts` analysis category to `glx analyze`** - Detects assertions with conflicting values for the same person/property combination (e.g., multiple conflicting birthplaces). Reports the number of distinct values and their confidence levels. Use `--check conflicts` to run independently. Fixes #156
- **Analyze flags duplicate given names among siblings** - `glx analyze` now detects when a parent's children share the same given name, which may indicate incorrect family reconstruction, a "replacement child" pattern, or a middle-name situation. Skips the pattern when earlier child died before the later was born. Fixes #164
- **Added `--subject` filter to `glx query assertions`** - Filter assertions by subject entity ID or person name substring. Matches any subject type by exact ID; for person subjects, also matches by case-insensitive name search. Fixes #150
- **Added `--birthplace` filter to `glx query persons`** - Filter persons by birthplace using place ID or name substring (case-insensitive). Matches against both `born_at` value and resolved place name. Fixes #141
- **Analyze flags uncited claims in notes** - `glx analyze` evidence checks now detect assertion notes that reference sources (e.g., "per county history," "census shows") without a corresponding citation. Fixes #162

### Changed
- **Life History narrative mentions children** - `glx summary` now includes children in the biographical narrative, listed by given name in birth order (e.g., "She had three children: Harriett, Elijah, and Mary."). Fixes #153

### Fixed
- **Validate catches dangling property references** - `glx validate` now detects when property values like `born_at`, `died_at`, or `residence` reference non-existent entities. Previously, standard vocabularies were not loaded during validation, so property reference checks were silently skipped. Fixes #147
- **Consistent date display across timeline and summary** - ISO dates like `1860-07-17` now render as `July 17, 1860` in timeline tabular output, summary vital events, and life events. Previously, dates appeared in whichever format they were stored (GEDCOM or ISO), creating inconsistent mixed output. Fixes #139
- **Stats lists duplicate entity IDs** - `glx stats` now lists the specific duplicate IDs in its warning, consistent with `glx analyze`. Fixes #177
- **Validate and archive loading skip non-.glx files** - `glx validate` and archive loading now only process files with the `.glx` extension. Previously, `.yaml` and `.yml` files in the archive directory were also parsed, causing spurious validation errors on non-GLX files like `.wikitree.yml`. Fixes #178
- **Windows compatibility for symlinked vocabulary files** - Archive loading now resolves Git symlink placeholders on Windows, where symlinks are stored as text files containing the target path. Previously, ~35 tests failed on Windows because example archives contain symlinked vocabulary files. Fixes #206
- **Analyze flags missing marriage events per spouse** - `glx analyze` now checks each spouse relationship independently instead of checking for any marriage event. Persons with multiple spouses where one has an event and another doesn't are now correctly flagged with the specific spouse name. Fixes #166
- **Places command detects person property references** - `glx places` no longer reports places as "Unreferenced" when they are used in person properties (`born_at`, `died_at`, `buried_at`, `residence`). Also checks assertion values for place-reference properties. Handles string, structured map, and temporal list property shapes. Fixes #145
- **Analyze checks citations for census coverage** - `glx analyze` now checks assertions' citations and sources (not just census event entities) when determining whether a census year is covered. Previously, census records documented only via citations were still suggested as missing, contradicting `glx coverage` output. Fixes #140
- **BEF date prefix respected in census suggestions** - `glx analyze` and `glx coverage` now treat `BEF <year>` death dates as exclusive upper bounds. A person with `died_on: "BEF 1870"` no longer gets 1870 census suggestions. Fixes #165
- **Summary shows marriages in chronological order** - `glx summary` now sorts spouses by full marriage date (earliest first, using the same date sort key as `glx timeline`) instead of relationship ID order. Correctly orders marriages within the same year. Undated marriages sort after dated ones. The Life History narrative also reflects the correct order. Fixes #136
- **Life History narrative formats ISO dates as readable text** - Dates like `1863-06-18` now render as "on June 18, 1863" instead of "in 1863-06-18". Handles full dates, year-month, and prefixed dates (ABT, BEF, AFT)
- **Census suggestions capped at plausible lifespan** - `glx analyze` and `glx coverage` no longer suggest census years beyond `birth_year + 100` when no death date is known. Previously, a person born ~1832 would get suggestions for 1940 and 1950 censuses. Fixes #130
- **Burial events infer death for census suggestions** - When `died_on` is not set but a burial event exists, the burial date is used as the death upper bound for census suggestions. Prevents suggesting post-death censuses for persons with burial records but no explicit death date. Fixes #134
- **1890 census annotated as mostly destroyed** - `glx coverage` and `glx analyze` now note that the 1890 US Census was mostly destroyed in a 1921 fire, so researchers don't waste time searching for non-existent records. Fixes #131
- **Timeline includes person's own birth and death** - `glx timeline` now synthesizes birth/death entries from `born_on`/`died_on` person properties when no corresponding event entity exists. Previously these were omitted, making the person's own vital events the only events missing from their timeline. Fixes #142

---

## [0.0.0-beta.8] - 2026-03-15

### Added

#### CLI
- **Added `glx analyze` command** - Automated research gap analysis engine that cross-references all entities in a GLX archive to surface evidence gaps (missing dates, no parents, no events), evidence quality issues (unsupported assertions, single-source persons, orphaned citations/sources), chronological inconsistencies (death before birth, parent younger than child, implausible lifespan), and research suggestions (census years to search, vital records to locate). Supports `--check` to run a single category, `--format json` for machine-readable output, and person filtering by ID or name
- **Added `glx diff` command** - Compare two GLX archive states with genealogy-aware diffing. Shows added, modified, and removed entities with field-level detail, confidence upgrade/downgrade tracking, and new evidence metrics. Supports summary, verbose, short, and JSON output modes. Use `--person` to filter changes for a specific person
- **Added `glx coverage` command** - Show source coverage matrix for a person, listing expected records (US census, vital, probate, land, military, church) and which are present vs missing. Flags high-priority gaps like the 1880 census. Supports `--json` output
- **Added `glx duplicates` command** - Detect potential duplicate person records using a weighted scoring model (name similarity with Levenshtein distance and nickname matching, birth/death year proximity, place match, shared relationships and events). Supports person-specific filtering and JSON output. Automatically skips persons already linked by relationships

#### Library
- **Exported `ExtractFirstYear` and `ExtractPropertyYear`** - Year-extraction utilities are now public API for use by CLI commands and external consumers

#### Validation
- **Moved temporal consistency checks to `glx analyze`** - Death before birth, parent younger than child, and marriage before birth checks are now part of the analyze command's consistency category instead of the validator, keeping `glx validate` focused on structural and referential integrity

#### Standard Vocabularies
- **Added `vocabulary_type` to property definitions** - Properties can now reference a controlled vocabulary (e.g., `vocabulary_type: gender_types`) instead of a free-form `value_type`. Validation warns on out-of-vocabulary values. Mutually exclusive with `value_type` and `reference_type`
- **Added `gender_types` vocabulary** - First vocabulary-constrained property type. Standard entries: male, female, unknown, other — with GEDCOM SEX mappings. GEDCOM export now looks up gender→SEX via the vocabulary before falling back to hardcoded mappings
- **Added `marriage_type` event property** - Classification of marriage (civil, religious, common-law). Was used in GEDCOM import/export but missing from standard vocabulary
- **Added `primary_name` person property** - Simple display name fallback when structured name property is not available. Was used in event titles and data generation but missing from standard vocabulary
- **Added `blob_size` media property** - Size in bytes of inline binary data from GEDCOM 5.5.1 BLOB records. Was used in GEDCOM media import but missing from standard vocabulary

### Changed
- **GEDCOM encoding conversion now streams for charmap encodings** - CP1252/ISO-8859-1 decoding uses `transform.NewReader` instead of reading the entire file into memory. Only ANSEL (which requires combining-mark reordering) buffers the full file. UTF-8 files pass through with near-zero overhead
- **ANSEL converter handles multiple combining diacriticals** - Consecutive combining marks preceding a base letter are now all buffered and emitted after the base letter in Unicode order, instead of only handling a single combining mark

### Fixed
- **GEDCOM import now converts non-UTF-8 encodings** - Files with `CHAR ANSI` (Windows-1252), `CHAR cp1252`, `CHAR ANSEL`, or `CHAR ISO-8859-1` are now automatically converted to UTF-8 during import. Previously, non-ASCII characters (German umlauts, accented letters, copyright symbols) were stored as raw bytes, producing `!!binary` YAML tags, garbled event titles, and `{"type":"Buffer"}` place names in the web UI
- **GEDCOM date import mangled when day-of-month matches level number** - Dates like `2 AUG 1944` (day 2) were imported as `2 DATE 2 AUG 1944` because the parser's value extraction matched the level number instead of the actual value. Fixed by walking past tokens positionally instead of using string search
- **Date year extraction now handles 1–3 digit years** - Year extraction previously hardcoded a 4-digit assumption (`\d{4}`), silently ignoring dates like `800`, `476`, or `ABT 476`. All four extraction sites (query filtering, timeline sorting, temporal validation, event titles) now support 1–4 digit years. Day-of-month values (e.g., `15` in `15 MAR 1850`) are correctly disambiguated. Timeline sort keys are zero-padded to 4 digits for proper chronological ordering. Fixes #108

---

## [0.0.0-beta.7] - 2026-03-10

### Added

#### CLI
- **Added `glx export` command** - Export GLX archives to GEDCOM 5.5.1 or 7.0 format. Supports both single-file and multi-file archives as input. Reconstructs GEDCOM FAM records from GLX relationships, converts dates/places/names back to GEDCOM format, and preserves sources, repositories, media, citations, and notes. Use `--format 70` for GEDCOM 7.0 output
- **Added `glx timeline` command** - Display chronological events for a person, including direct events and family events (spouse/child births, parent deaths) via relationship traversal. Supports `--no-family` flag to exclude family events; undated events shown in a separate section
- **Added `glx summary` command** - Comprehensive person profile showing identity, vital events, life events, family (spouses, parents, siblings), other relationships, and an auto-generated life history narrative
- **Added `glx ancestors` and `glx descendants` commands** - Display ancestor/descendant trees using box-drawing characters. Traverses parent-child relationships with `--generations` flag to limit depth. Handles biological, adoptive, foster, and step-parent types with cycle detection
- **Added `glx vitals` command** - Display vital records (name, sex, birth, christening, death, burial) for a person by ID or name search, plus any other life events they participated in
- **Added `glx cite` command** - Generate formatted citation text from structured fields (source title, type, repository, URL, accessed date, locator), eliminating repetitive manual `citation_text` writing
- **Added `--source` and `--citation` filters to `glx query assertions`** - Filter assertions by source or citation ID to find all claims derived from a specific source
- **Improved `glx query persons --name` to search all name variants** - Now matches across birth names, married names, maiden names, and as-recorded variants (temporal name lists), not just the primary name. Results show alternate names with "aka:" suffix

#### Event Entity
- **Added optional `title` field** - Human-readable label for events (e.g., "1860 Census — Webb Household"). Auto-generated on GEDCOM import (e.g., "Birth of Robert Webb (1815)", "Marriage of John Smith and Jane Doe (1850)")

#### GEDCOM Import
- **Non-standard date preservation** - BCE dates, Julian/Hebrew/French Republican calendar dates, and dual-year dates are preserved as raw strings instead of being dropped
- **TITL with DATE/PLAC sub-records** - Title properties with dates and places are stored as temporal list items and roundtrip correctly
- **Empty OCCU with PLAC fallback** - OCCU records with empty values but PLAC sub-records now extract the place text as the occupation value
- **HEAD-level NOTE preservation** - Notes on GEDCOM HEAD records are now imported and exported
- **Family-level RESI import** - RESI records under FAM are now distributed to both spouses as residence properties
- **Family-level NOTE import/export** - NOTE records on FAM are now stored on the relationship's Notes field and roundtrip correctly

#### GEDCOM Export
- **Inline SOUR citations on individual events** - Birth, death, burial, and other individual events now preserve SOUR citations during export
- **Single-spouse family marriages** - FAM records with only HUSB or WIFE now export marriage relationships and events instead of being silently dropped
- **Multiple MARR events per family** - Families with multiple MARR records now preserve all marriage events
- **Marriage TYPE export** - Marriage `marriage_type` property now exported as TYPE sub-record on MARR
- **Family event TYPE/properties export** - Family events (EVEN, ENGA, etc.) now export event_subtype and other event properties (TYPE, CAUS, AGE) that were previously lost
- **HEAD metadata roundtrip** - LANG, FILE, COPR sub-records from the original GEDCOM HEAD are now preserved through import/export
- **Single-value RESI export** - RESI stored as scalar (not list) now exports correctly instead of being silently dropped
- **Multi-family children placed in all matching families** - Children belonging to multiple FAM records (e.g., birth family + step-family) are now placed in all matching families instead of only the first match

#### Validation
- **Added temporal consistency checks** - Validator now warns on: death year before birth year, parent born after child, marriage event before participant's birth. Reported as warnings since dates are often estimates

#### Documentation
- **Added [Westeros example archive](/examples/westeros/)** - Large-scale example featuring 790+ characters from *A Song of Ice and Fire* with full evidence chains, 200+ custom vocabulary types, and temporal properties. Hosted at [github.com/genealogix/glx-archive-westeros](https://github.com/genealogix/glx-archive-westeros)
- **Added [Hands-On CLI Guide](/guides/hands-on-cli-guide)** - Step-by-step walkthrough of every `glx` command using the Westeros demo archive, with real output examples

### Fixed
- **SOUR citation duplication on multi-value properties** - Assertion-based SOUR references now filter by matching value, preventing N×N duplication when a person has multiple values for TITL, OCCU, etc.

---

## [0.0.0-beta.6] - 2026-03-08

### Added

#### CLI
- **Added `glx places` command** - Analyze places for ambiguity and completeness: flags duplicate names, missing coordinates, missing types, hierarchy gaps, and unreferenced places with canonical hierarchy paths
- **Added `glx query` command** - Filter and list entities from a GLX archive with type-specific flags: `--name`, `--born-before`, `--born-after` for persons; `--type`, `--before`, `--after` for events; `--confidence`, `--status` for assertions
- **Added `glx stats` command** - Summary dashboard showing entity counts, assertion confidence distribution, and entity coverage for quick feedback on archive health

#### Build & Release
- **Added `make release-snapshot` target** - Build cross-platform binaries locally without publishing, using GoReleaser snapshot mode
- **Updated release workflow to latest action versions** - `actions/checkout@v4` (with `fetch-depth: 0` for proper changelog), `actions/setup-go@v5`, `goreleaser/goreleaser-action@v6`

#### Person Entity
- **Added name variation tracking** - Expanded the `name.fields.type` classification field with standard values for alternate spellings, abbreviations, and as-recorded forms (`aka`, `maiden`, `anglicized`, `professional`, `as_recorded`). Added documentation and examples for representing name variations like "R. Webb" vs. "Robert Webb"

#### Standard Vocabularies
- **Added `original_place_name` citation property** - Records the verbatim place name from a source before normalization to a place entity (e.g., "The Town Of Oakdale" vs the normalized place reference)
- **Added relationship types** - `neighbor`, `coworker`, `housemate` for census/social records; `apprenticeship`, `employment`, `enslavement`, `relative` for occupational and generic kinship relationships
- **Added event types** - `legal_separation`, `taxation`, `voter_registration` for legal/administrative events; `military_service`, `stillborn`, `affiliation` for service periods, stillbirths, and memberships
- **Added source types `population_register`, `tax_record`, `notarial_record`** - Common European and colonial record types
- **Expanded `military` source type description** - Now includes draft registrations and muster rolls

#### Participant Object
- **Added `properties` to participants** - Participants across events, relationships, and assertions can now carry per-participant properties like `age_at_event`, enabling shared events (census, passenger lists) to record individual data without creating separate events per person
- **Participant properties validated against parent entity vocabulary** - Event participant properties validated against event_properties, relationship participant properties against relationship_properties, assertion participant properties against event_properties

#### Assertion Entity
- **Added existential assertions** - Assertions no longer require `property` or `participant`; an assertion with only `subject` and evidence asserts the entity's existence, optionally at a specific `date` (#26)

#### GEDCOM Import
- **Import HEAD metadata** - GEDCOM HEAD record fields (export date, source file, copyright, language, source system/version/corporation, GEDCOM version, character set, notes) are now stored in a `metadata` section on the GLX archive instead of being discarded after logging
- **Import SUBM metadata** - GEDCOM SUBM submitter information (name, address, phone, email, website) is now stored in `metadata.submitter` on the GLX archive

#### Data Model
- **Added `Metadata` type** - New top-level `metadata` field on GLX archives for storing import provenance information
- **Added `Submitter` type** - Nested within metadata to hold submitter contact details

### Changed

#### Specification
- **Removed hard-coded vocabulary counts** - Replaced "N standardized type codes" with descriptive text to prevent stale counts as vocabularies grow
- **Improved custom type example** - Custom event type example now shows defining custom participant roles (`apprentice`, `master`) alongside the custom event type
- **Clarified `subject` participant role** - Documented as preferred over `principal`

### Fixed

#### Specification
- **Fixed confidence levels example format** - Core concepts example now uses the correct `label`/`description` structure instead of simple key-value strings
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

[0.0.0-beta.10]: https://github.com/genealogix/glx/compare/v0.0.0-beta.9...HEAD
[0.0.0-beta.9]: https://github.com/genealogix/glx/compare/v0.0.0-beta.8...v0.0.0-beta.9
[0.0.0-beta.8]: https://github.com/genealogix/glx/compare/v0.0.0-beta.7...v0.0.0-beta.8
[0.0.0-beta.7]: https://github.com/genealogix/glx/compare/v0.0.0-beta.6...v0.0.0-beta.7
[0.0.0-beta.6]: https://github.com/genealogix/glx/compare/v0.0.0-beta.5...v0.0.0-beta.6
[0.0.0-beta.5]: https://github.com/genealogix/glx/compare/v0.0.0-beta.4...v0.0.0-beta.5
[0.0.0-beta.4]: https://github.com/genealogix/glx/compare/v0.0.0-beta.3...v0.0.0-beta.4
[0.0.0-beta.3]: https://github.com/genealogix/glx/compare/v0.0.0-beta.2...v0.0.0-beta.3
[0.0.0-beta.2]: https://github.com/genealogix/glx/compare/v0.0.0-beta.1...v0.0.0-beta.2
[0.0.0-beta.1]: https://github.com/genealogix/glx/compare/v0.0.0-beta.0...v0.0.0-beta.1
[0.0.0-beta.0]: https://github.com/genealogix/glx/releases/tag/v0.0.0-beta.0
