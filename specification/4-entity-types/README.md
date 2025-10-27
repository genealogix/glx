# Entity Types

This section defines the core entity types in GENEALOGIX. Each entity represents a distinct concept in genealogical research and can be referenced, extended, and related to other entities.

## Core Entities

### [Person](person.md)
Represents an individual in the family archive. Contains personal identity information, names, events, and relationships.

- **Key Properties**: Names, gender, living status, events, relationships
- **File Format**: `.glx`
- **ID Format**: `person-[hex8]`
- **GEDCOM Equivalent**: INDI (Individual Record)

### [Relationship](relationship.md)
Represents connections between people such as spouse, parent-child, and other family relationships.

- **Key Properties**: Relationship type, participants, start/end events
- **File Format**: `.glx`
- **ID Format**: `rel-[hex8]`
- **GEDCOM Equivalent**: FAM (Family Record)

### [Event/Fact](event.md)
Represents occurrences in time and place: births, marriages, occupations, residences, etc.

- **Key Properties**: Type, date, place, participants, description
- **File Format**: `.glx`
- **ID Format**: `event-[hex8]`
- **GEDCOM Equivalent**: BIRT, DEAT, MARR, OCCU, etc.

### [Place](place.md)
Represents geographic locations forming a hierarchical structure. Supports multiple names and historical variations.

- **Key Properties**: Name, type, hierarchy, coordinates, alternative names
- **File Format**: `.glx`
- **ID Format**: `place-[hex8]`
- **GEDCOM Equivalent**: PLAC (Place structures)

### [Assertion](assertion.md)
Represents an evidence-based conclusion about a specific genealogical fact. Forms the core of the GENEALOGIX assertion model.

- **Key Properties**: Subject, property, value, citations, confidence
- **File Format**: `.glx`
- **ID Format**: `assertion-[hex8]`
- **GEDCOM Equivalent**: Implicit (derived from GEDCOM structure and SOUR references)

### [Source](source.md)
Represents a bibliographic resource or information source. Can be books, documents, databases, websites, etc.

- **Key Properties**: Title, author, publication info, repository
- **File Format**: `.glx`
- **ID Format**: `source-[hex8]`
- **GEDCOM Equivalent**: SOUR (Source Record)

### [Citation](citation.md)
Represents a specific reference to evidence within a source. Links sources to specific pages, records, or items.

- **Key Properties**: Source reference, page, data date, quality, locator
- **File Format**: `.glx`
- **ID Format**: `citation-[hex8]`
- **GEDCOM Equivalent**: SOUR.PAGE, SOUR.QUAY

### [Repository](repository.md)
Represents an institution or organization that holds genealogical sources (archives, libraries, databases, etc.).

- **Key Properties**: Name, type, address, contact info, access restrictions
- **File Format**: `.glx`
- **ID Format**: `repository-[hex8]`
- **GEDCOM Equivalent**: REPO (Repository Record)

### [Media](media.md)
Represents digital or physical media objects associated with genealogical entities (photographs, documents, audio, etc.).

- **Key Properties**: Title, file path, MIME type, description
- **File Format**: `.glx`
- **ID Format**: `media-[hex8]`
- **GEDCOM Equivalent**: OBJE (Object/Media Record)

## Entity Relationships

```
Person
  ├── has many Events (birth, marriage, occupation, etc.)
  ├── has many Names
  ├── has many Assertions (about properties)
  ├── links to media (via Media entity)
  └── participates in Relationships

Relationship
  ├── connects multiple Persons
  ├── has start/end Events
  ├── has Assertions (about relationship properties)
  └── links to media

Event
  ├── occurs at a Place
  ├── involves multiple Persons (via participants)
  ├── supported by Assertions
  └── referenced by Assertions

Place
  ├── has parent Place (hierarchy)
  ├── has alternative Names
  └── referenced by Events and Assertions

Assertion
  ├── references Person, Event, Relationship, or other subject
  ├── supported by Citations
  └── may reference Places

Source
  ├── held in Repository
  ├── referenced by Citations
  └── may have media

Citation
  ├── references Source
  ├── may reference Repository
  ├── supports Assertions
  └── references media

Repository
  ├── holds Sources
  └── referenced by Citations

Media
  ├── associated with any entity
  └── referenced by assertions/evidence
```

## Entity Properties Summary

| Entity | Required Fields | Unique ID | Versioned | Hierarchical |
|--------|-----------------|-----------|-----------|--------------|
| Person | id, version | ✓ | ✓ | - |
| Relationship | id, version, type, people | ✓ | ✓ | - |
| Event | id, version, type | ✓ | ✓ | - |
| Place | id, version, name | ✓ | ✓ | ✓ (parent) |
| Assertion | id, version, subject, property | ✓ | ✓ | - |
| Source | id, version, title | ✓ | ✓ | - |
| Citation | id, version, source_id | ✓ | ✓ | - |
| Repository | id, version, name | ✓ | ✓ | - |
| Media | id, version | ✓ | ✓ | - |

## ID Scheme

Each entity type has a distinct ID prefix enabling quick entity type identification:

- `person-XXXXXXXX`: Person entities
- `rel-XXXXXXXX`: Relationship entities
- `event-XXXXXXXX`: Event entities
- `place-XXXXXXXX`: Place entities
- `assertion-XXXXXXXX`: Assertion entities
- `source-XXXXXXXX`: Source entities
- `citation-XXXXXXXX`: Citation entities
- `repository-XXXXXXXX`: Repository entities
- `media-XXXXXXXX`: Media entities

Where `XXXXXXXX` is an 8-character hexadecimal string.

## Entity Lifecycle

### Creation
- Each entity is created with a unique ID and version "1.0"
- `created_at` and `created_by` timestamps are recorded
- Initial state is valid and complete

### Modification
- Entities can be modified; only `modified_at` and `modified_by` change
- Version number may increment for schema-breaking changes
- All modifications are tracked in Git history

### Deletion
- Entities are typically not deleted but rather marked as inactive
- Soft deletion patterns are preferred to preserve audit trails
- Hard deletion may break referential integrity

## Validation

All entities must:
- Have valid, unique IDs in the correct format
- Have version numbers (semantic versioning)
- Have timestamps and creator information
- Pass schema validation against JSON schemas in `specification/schema/v1/`
- Maintain referential integrity (references to other entities must exist)

## Extension Points

The GENEALOGIX specification allows extension through:
- Custom properties in entities (via `additionalProperties`)
- Custom event types
- Custom relationship types
- Custom tags and notes

See [Core Concepts](../2-core-concepts.md#repository-owned-vocabularies) for vocabulary and extension guidelines.

## See Also

- [File Structure](../3-file-structure.md) - How entities are organized in files
- [Data Model](../5-data-model/) - How entities relate to evidence and provenance
- Entity type documentation includes GEDCOM mapping information


