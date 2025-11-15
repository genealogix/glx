# GLX TODO

- Clean up `specification/4-entity-types/source.md`
- Ensure all entity type files have a table with the properties and their types
- Remove `version` field from all entities (Person, Event, Relationship, Place, Assertion, etc.) - versioning handled at archive level
- Unify `EventParticipant`, `RelationshipParticipant`, and `AssertionParticipant` into a single `Participant` struct after the current refactor is complete.