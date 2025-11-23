# GLX TODO

- Clean up `specification/4-entity-types/source.md`
- Ensure all entity type files have a table with their normal fields, standard properties and their types
- Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete.
- remove the custom flag from all vocabularies and property definitions
- remove glx archive folder references from all examples and documentation
- explore making name property an object?
- Store GEDCOM extension tag data (tags starting with `_`) in ImportResult or a dedicated structure so it can be accessed after import
- Store GEDCOM HEAD metadata (export_date, source_file, copyright, language, source_system, etc.) in ImportResult or GLXFile properties
- Store GEDCOM SUBM (submitter) metadata in ImportResult or GLXFile properties