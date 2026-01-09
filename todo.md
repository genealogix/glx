# GLX TODO

> **Note**: Todos are organized by category and priority. Items marked with 🔴 are high priority, 🟡 are medium priority, and 🟢 are low priority or nice-to-have.

---

## 🚀 Infrastructure & Deployment

- 🔴 Deploy JSON schemas to `https://schema.genealogix.io/v1/*` URLs referenced in [specification/schema/README.md](specification/schema/README.md)
- 🟢 Add make command for goreleaser

---

## 📖 Documentation

- 🟡 Remove glx archive folder references from all examples and documentation
- 🟡 Add comprehensive example showing assertion-to-entity property workflow (setting properties directly vs creating assertions with evidence)
- 🟡 Add more temporal property examples throughout entity documentation (residence, occupation, name changes over time)

### Specification TODOs

- 🔴 **Place alternative_names refactor**: The `alternative_names` structure in [place.md](specification/4-entity-types/place.md) uses a `date_range` sub-object that should be handled via properties instead for consistency with the temporal property system.
- 🟡 **Rename `claim` to `property`**: Assertion entities use `claim` to reference properties. For clarity, consider renaming `claim` field to `property` in a future version. Currently documented in [assertion.md](specification/4-entity-types/assertion.md) with clarification note.
- 🟡 **Review standard vocabularies**: Audit all standard vocabulary files (.glx) in [5-standard-vocabularies/](specification/5-standard-vocabularies/) to ensure consistency and completeness.
- 🟡 **Rewrite introduction terminology**: The [introduction](specification/1-introduction.md) needs a clearer glossary/terminology section defining key concepts like archive, entity, assertion, evidence chain, claim, property.
- 🟢 **Add validation rule sections**: Each entity type documentation should include a consolidated "Validation Rules" section summarizing all validation requirements for that entity.

---

## 🏗️ Type System & Data Model

### Schema Improvements

- 🟡 JSON schemas don't validate entity `properties` structure (e.g., person name with fields). Properties are vocabulary-controlled and dynamic, so schema validation uses `additionalProperties: true`. Consider documenting this as intentional or adding runtime property validation in the CLI.
- 🟢 Should property fields have data types?

### Entity Properties

- 🟡 **Source properties**: Create `source-properties.glx` vocabulary. Consider which fields should move to properties (e.g., `coverage` was removed as a direct field).
- 🟡 Citation properties - should some fields be vocabulary-controlled properties?

### Participant Unification

- 🟢 Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete

### Vocabulary & Type Clarifications

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

- 🔴 **CENS (Census) Tag Handling**: Currently creates census events. Census is not an event - it's a source/citation that supports assertions about a person's attributes (residence, occupation, etc.). Should create citations from census records that can be attached to property assertions.
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
- 🟡 **Line 65**: ABBR (abbreviation) dumped in notes → Add `Abbreviation` field to Source struct
- 🟡 **Line 76**: CALN (call number) dumped in notes → Add to Citation or create RepositoryHolding struct
- 🟡 **Line 98**: EVEN (events recorded) dumped in notes → Add `EventsRecorded []string` field to Source
- 🟡 **Line 101**: AGNC (agency) dumped in notes → Add `Agency` field to Source struct

#### Citation ([gedcom_evidence.go](glx/lib/gedcom_evidence.go))
- 🟡 **Line 63**: Source date dumped in notes → Add `SourceDate` field to Citation struct

### Data Quality & Validation

- 🟢 **PLAC Validation** ([gedcom_place.go](glx/lib/gedcom_place.go)): Validate PLAC fields - reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown"
- 🟢 **Relationship Roles**: Verify that the gedcom import correctly assigns parent/child roles to relationship participants
- 🟢 **Place Properties**: Move some place fields to properties?
- 🟢 **Place Type Vocab**: Fix places[place-31].Type references non-existent place_types: locality
- 🟢 **Duplicate Prevention**: Prevent duplicate repository creation

---

## ✅ GLX Validation

- 🟡 Require participant roles in events, relationships, assertions?
- 🟡 Add validator tags to GLX structs

---

## 💻 CLI & User Experience

(No current items)

---

## 🧹 Code Organization & Quality

- 🟢 Move Loggers to their own package?

---

## 📝 Notes

- All TODO comments in code reference this file
- Keep this file as the single source of truth for project todos
- When adding new todos, place them in the appropriate category with priority marker