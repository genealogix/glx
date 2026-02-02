# GLX TODO

> **Note**: Todos are organized by category and priority. Items marked with 🔴 are high priority, 🟡 are medium priority, and 🟢 are low priority or nice-to-have.

---

## 🚀 Infrastructure & Deployment

- 🟢 Add make command for goreleaser

---

## 📖 Documentation

- 🟡 Remove glx archive folder references from all examples and documentation
- 🟡 Add comprehensive example showing assertion-to-entity property workflow (setting properties directly vs creating assertions with evidence)
- 🟡 Add more temporal property examples throughout entity documentation (residence, occupation, name changes over time)
- 🟡 Add `multi_value` property examples to vocabularies.md - show how multi-value properties are used in entity data (array values) and validated

### Specification TODOs

- 🟡 **Rename `claim` to `property`**: Assertion entities use `claim` to reference properties. For clarity, consider renaming `claim` field to `property` in a future version. Currently documented in [assertion.md](specification/4-entity-types/assertion.md) with clarification note.
- 🟡 **Review standard vocabularies**: Audit all standard vocabulary files (.glx) in [5-standard-vocabularies/](specification/5-standard-vocabularies/) to ensure consistency and completeness.
- 🟡 **Rewrite introduction terminology**: The [introduction](specification/1-introduction.md) needs a clearer glossary/terminology section defining key concepts like archive, entity, assertion, evidence chain, claim, property.
- 🟢 **Add validation rule sections**: Each entity type documentation should include a consolidated "Validation Rules" section summarizing all validation requirements for that entity.

---

## 🏗️ Type System & Data Model

### Schema Improvements

- 🟡 **Add `multi_value` to all property schemas**: The `multi_value` field is documented in the spec and used in `source-properties.glx` and `repository-properties.glx`, but missing from all property JSON schemas except `source-properties.schema.json`. Need to add `multi_value` field to: person-properties, event-properties, place-properties, relationship-properties, media-properties, repository-properties, citation-properties schemas.
- 🟢 Should property fields have data types?

### Entity Properties

- ✅ ~~**Source properties**: Create `source-properties.glx` vocabulary~~ (completed - added `abbreviation`, `call_number`, `events_recorded`, `agency`, `coverage`, `external_ids` properties)

### Participant Unification

- 🟢 Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete

### Vocabulary & Type Clarifications

- 🟡 **GEDCOM tag mapping in vocabularies**: Add GEDCOM tag mappings to vocabulary definitions to replace hardcoded switch statements in importer. Should cover: event types (`BIRT`→`birth`, `MARR`→`marriage`), relationship types, place types, source types, repository types, citation properties (`PAGE`→`page`), and all other tags currently mapped via switch statements in `gedcom_*.go` files. This would enable data-driven conversion and round-tripping between GEDCOM and GLX formats.
- 🟡 **Adoption semantics**: Clarify `adoption` event type vs `adoption` relationship type vs `adoptive-parent-child` relationship type. Consider consolidating or documenting distinct use cases.
- 🟢 **Bar/Bat Mitzvah**: Consider consolidating `bat_mitzvah` (BATM) and `bas_mitzvah` (BASM) into a single event type - they represent the same ceremony with alternate spellings
- 🟢 **Godparent duality**: `godparent` exists as both a relationship type and participant role (applies_to: event, relationship). Consider documenting the distinction: relationship type represents ongoing godparent-godchild bond, participant role represents specific event participation (e.g., baptism ceremony).
- 🟢 Gender/sex controlled vocabularies?
- 🟢 Should sex be a temporal property instead of a top-level field?

### Evidence & Assertions

- 🟡 We shouldn't create assertions from imports without citations
- 🟡 Decide what to do with QUAY ratings (removed in beta.2)
- 🟢 Consider adding `media` as a third evidence option for assertions (alongside `citations` and `sources`) - useful for direct visual evidence like gravestone photos
- 🟢 Consider relaxing event participant requirement - the spec says "At least one participant is required (events without participants are not meaningful)" but historical events (wars, famines, natural disasters) may be relevant to genealogy without specific participants

---

## 📥 GEDCOM Import

### Critical Import Issues

- 🔴 **NOTE References Not Resolved**: Linked NOTE records (e.g., `@N15@`) are stored as literal strings instead of resolving to actual note text. Should import NOTE records and replace references with content.
- 🔴 **Media/OBJE Import**: Only 2 of 32 multimedia references imported in torture test (94% loss). Inline OBJE tags in events, URL-type multimedia, and BLOB data are not imported.
- 🔴 **CENS (Census) Tag Handling**: Currently creates census events. Census is not an event - it's a source/citation that supports assertions about a person's attributes (residence, occupation, etc.). Should create citations from census records that can be attached to property assertions.
- 🟡 **GEDCOM Record Ordering**: Records are processed in file order, but sources may reference repositories that appear later in the file. When a source references a repository that hasn't been converted yet, the repository link and call_number are lost. Fix: process repositories in a first pass before sources, or use a two-pass conversion.
- 🟡 **Embedded Citations** ([gedcom_evidence.go:38](glx/lib/gedcom_evidence.go#L38)): Implement support for embedded citations (citation details without full source reference)

### Missing Data Storage

**Issue**: Data is being processed but not stored/exposed after import

- 🟡 **Extension Tags** ([gedcom_converter.go:102-103](glx/lib/gedcom_converter.go#L102-L103)): Store extension tag data (tags starting with `_`) - vendor-specific metadata like _MSTAT, _UID, _NSTY
- 🟡 **HEAD Metadata** ([gedcom_converter.go:220-221](glx/lib/gedcom_converter.go#L220-L221)): Store HEAD metadata (export_date, source_file, copyright, language, source_system)
- 🟡 **SUBM Metadata** ([gedcom_converter.go:246-247](glx/lib/gedcom_converter.go#L246-L247)): Store SUBM (submitter) metadata
- 🟢 **NCHI Tag** ([gedcom_family.go](glx/lib/gedcom_family.go)): Store NCHI (number of children) - can differ from actual CHIL count
- 🟢 **NAME TYPE** ([gedcom_name.go](glx/lib/gedcom_name.go)): Store NAME TYPE subfield (birth, married, aka)

### Notes Anti-Pattern Audit

**Anti-pattern**: Dumping structured data into Notes fields instead of proper typed fields/properties

#### Source ([gedcom_source.go](glx/lib/gedcom_source.go))
- ✅ ~~ABBR, CALN, EVEN, AGNC now stored in `properties` (abbreviation, call_number, events_recorded, agency)~~

#### Citation ([gedcom_evidence.go](glx/lib/gedcom_evidence.go))
- 🟡 **Line 63**: Source date dumped in notes → Add `SourceDate` field to Citation struct

### Data Quality & Validation

- 🟢 **LANG Tag Normalization**: Normalize language tags on import. GEDCOM 7.0 uses ISO format (e.g., `en-US`), but GEDCOM 5.5.x uses free-form text (e.g., `English`, `French`). Should convert 5.5.x values to ISO codes for consistency.
- 🟢 **PLAC Validation** ([gedcom_place.go](glx/lib/gedcom_place.go)): Validate PLAC fields - reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown"
- 🟢 **Relationship Roles**: Verify that the gedcom import correctly assigns parent/child roles to relationship participants
- 🟢 **Place Properties**: Move some place fields to properties?
- 🟢 **Place Type Vocab**: Fix places[place-31].Type references non-existent place_types: locality
- 🟢 **Duplicate Prevention**: Prevent duplicate repository creation

---

## ✅ GLX Validation

- 🟡 **Place hierarchy cycle detection**: Validate that place parent references don't form cycles (circular parent references). Specification requires acyclic hierarchy but validation is not implemented in [lib/validation.go](glx/lib/validation.go).
- 🟡 Require participant roles in events, relationships, assertions?
- 🟡 Add validator tags to GLX structs

---

## 💻 CLI & User Experience

(No current items)

---

## 🧹 Code Organization & Quality

### Architectural Issues

- 🔴 **lib I/O violation** ([gedcom_logging.go:36](glx/lib/gedcom_logging.go#L36)): `NewImportLogger` does direct file I/O with `os.Create`, violating CLAUDE.md rule that lib package must never do I/O. Should accept `io.Writer` instead of path.

### Bugs

- 🔴 **Unreachable code path** ([gedcom_evidence.go:206](glx/lib/gedcom_evidence.go#L206)): The shared note lookup for GEDCOM 7.0 is unreachable because the function already returns if `noteRecord.Value != ""`. Dead code should be removed or logic fixed.
- 🔴 **Missing bounds check** ([gedcom_place.go:219-220](glx/lib/gedcom_place.go#L219-L220)): `parseCoordinate` can panic if coordinate value has length < 2 (e.g., just "N" without a number). Should add length validation.

### Performance

- 🟡 **Regex compilation in hot path** ([gedcom_name.go:52-53](glx/lib/gedcom_name.go#L52-L53)): `surnameRegex` and `nicknameRegex` are compiled on every call to `parseGEDCOMName`. Should be package-level compiled regexes.
- 🟡 **Map allocation in hot paths**: Type mapping functions allocate maps on every call instead of using package-level maps: `mapGEDCOMEventType` ([gedcom_individual.go:411](glx/lib/gedcom_individual.go#L411)), `mapFamilyEventType` ([gedcom_family.go:437](glx/lib/gedcom_family.go#L437)), `mapSourceType` ([gedcom_source.go:167](glx/lib/gedcom_source.go#L167)), `mapRepositoryType` ([gedcom_repository.go:129](glx/lib/gedcom_repository.go#L129)), `inferMimeType` ([gedcom_media.go:242](glx/lib/gedcom_media.go#L242)), `mapFormatToMimeType` ([gedcom_media.go:295](glx/lib/gedcom_media.go#L295)).

### DRY Violations

- 🟡 **Duplicated event extraction logic**: Same date/place/note/citation extraction pattern duplicated across `convertIndividualEvent` ([gedcom_individual.go:290](glx/lib/gedcom_individual.go#L290)), `convertMarriageEvent` ([gedcom_family.go:135](glx/lib/gedcom_family.go#L135)), `convertDivorceEvent` ([gedcom_family.go:259](glx/lib/gedcom_family.go#L259)), `convertFamilyEvent` ([gedcom_family.go:357](glx/lib/gedcom_family.go#L357)). Should extract common `extractEventDetails` helper.
- 🟡 **Duplicated media conversion logic** ([gedcom_media.go](glx/lib/gedcom_media.go)): `convertMedia` and `convertEmbeddedMedia` share ~90% identical code. Should refactor to common `convertMediaCommon` function.

### Code Style

- 🟢 **ConversionContext too large** ([gedcom_import.go:134-173](glx/lib/gedcom_import.go#L134-L173)): Has 20+ fields. Consider breaking into focused sub-structs (IDMappings, IDCounters, GEDCOM70State).
- 🟢 **Magic string "REPO"** ([gedcom_repository.go:24](glx/lib/gedcom_repository.go#L24)): Uses literal `"REPO"` instead of `GedcomTagRepo` constant.
- 🟢 **Inconsistent error wrapping**: Some errors use `fmt.Errorf` with `%w` for wrapping, others use different patterns. Should standardize error wrapping approach.
- 🟢 **String building inefficiency** ([gedcom_converter.go:288-298](glx/lib/gedcom_converter.go#L288-L298)): Uses unnecessary intermediate `result` variable with `strings.Builder`. Should simplify.
- 🟢 **Silent error swallowing** ([gedcom_family.go:85](glx/lib/gedcom_family.go#L85)): `extractCitations` errors are silently ignored. Should log warnings for failed citation extractions.
- 🟢 Move Loggers to their own package?

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker