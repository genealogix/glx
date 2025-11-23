# GLX TODO

## Documentation

- Clean up `specification/4-entity-types/source.md`
- Ensure all entity type files have a table with their normal fields, standard properties and their types
- Remove glx archive folder references from all examples and documentation

## Type System & Schema

- Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete
- Remove the custom flag from all vocabularies and property definitions
- Explore making name property an object?

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

### Priority
1. **High**: Repository phones/emails, Source abbreviation/agency, Media citations
2. **Medium**: Source events_recorded, Media medium/media_type, Citation source_date, embedded citations
3. **Low**: Media crop coordinates (needs new type), Source call_number (architectural decision)