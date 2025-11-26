# GLX TODO

## Infrastructure

- Deploy JSON schemas to `https://schema.genealogix.io/v1/*` URLs referenced in specification/schema/README.md

## Documentation

- Remove glx archive folder references from all examples and documentation
- Add comprehensive example showing assertion-to-entity property workflow (setting properties directly vs creating assertions with evidence)
- Add more temporal property examples throughout entity documentation (residence, occupation, name changes over time)

## Type System & Schema

- Clarify adoption semantics: `adoption` event type vs `adoption` relationship type vs `adoptive-parent-child` relationship type. Consider consolidating or documenting distinct use cases.
- Consider consolidating `bat_mitzvah` (BATM) and `bas_mitzvah` (BASM) into a single event type - they represent the same ceremony with alternate spellings
- Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete
- we shouldn't create assertions from imports without citations
- decide what to do with QUAY ratings... (removed in beta.2)
- Consider adding `media` as a third evidence option for assertions (alongside `citations` and `sources`) - useful for direct visual evidence like gravestone photos
- Consider relaxing event participant requirement - the spec says "At least one participant is required (events without participants are not meaningful)" but historical events (wars, famines, natural disasters) may be relevant to genealogy without specific participants
- JSON schemas don't validate entity `properties` structure (e.g., person name with fields). Properties are vocabulary-controlled and dynamic, so schema validation uses `additionalProperties: true`. Consider documenting this as intentional or adding runtime property validation in the CLI.
- `godparent` exists as both a relationship type and participant role (applies_to: event, relationship). Consider documenting the distinction: relationship type represents ongoing godparent-godchild bond, participant role represents specific event participation (e.g., baptism ceremony).
- Media schema is missing several fields documented in media.md (`type`, `date`, `subjects`, `source`, `citation`, `width`, `height`, `duration`, `file_size`). Many of these should move to vocabulary-controlled `properties` rather than top-level fields. Update schema and documentation together when refactoring media entity structure.

## GEDCOM Import: Census Tag Handling

- **CENS (Census)**: Currently creates census events. Census is not an event - it's a source/citation that supports assertions about a person's attributes (residence, occupation, etc.). Should create citations from census records that can be attached to property assertions.

## GEDCOM Import: Missing Data Storage

**Issue**: Data is being processed but not stored/exposed after import

- **gedcom_converter.go:102-103**: Store extension tag data (tags starting with `_`) - vendor-specific metadata like _MSTAT, _UID, _NSTY
- **gedcom_converter.go:220-221**: Store HEAD metadata (export_date, source_file, copyright, language, source_system)
- **gedcom_converter.go:246-247**: Store SUBM (submitter) metadata
- **gedcom_family.go**: Store NCHI (number of children) - can differ from actual CHIL count
- **gedcom_name.go**: Store NAME TYPE subfield (birth, married, aka)
- **gedcom_place.go**: Validate PLAC fields - reject non-geographic text like "Died in childbirth", "Unmarried", "Unknown"

## GEDCOM Import: Notes Anti-Pattern Audit

**Anti-pattern**: Dumping structured data into Notes fields instead of proper typed fields/properties

### Repository (gedcom_repository.go)
- **Line 104**: Additional phones concatenated into notes → Change `Phone string` to `Phones []string`
- **Line 110**: Additional emails concatenated into notes → Change `Email string` to `Emails []string`

### Source (gedcom_source.go)
- **Line 65**: ABBR (abbreviation) dumped in notes → Add `Abbreviation` field to Source struct
- **Line 76**: CALN (call number) dumped in notes → Add to Citation or create RepositoryHolding struct
- **Line 98**: EVEN (events recorded) dumped in notes → Add `EventsRecorded []string` field to Source
- **Line 101**: AGNC (agency) dumped in notes → Add `Agency` field to Source struct

### Media (gedcom_media.go)
- **Line 82, 180**: MEDI (medium type) dumped in notes → Add `Medium` or `MediaType` field to Media struct
- **Line 96, 192**: CROP coordinates dumped in notes → Add structured `Crop *CropCoordinates` field
- **Line 110**: Citation IDs dumped as strings in notes → Add `Citations []string` field to Media struct

### Citation (gedcom_evidence.go)
- **Line 63**: Source date dumped in notes → Add `SourceDate` field to Citation struct
- **Line 38**: Embedded citations skipped entirely → Implement embedded citation support (no source reference)