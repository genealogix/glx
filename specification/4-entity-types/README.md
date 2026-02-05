---
title: Entity Types
description: Core entity types in GENEALOGIX - person, relationship, event, place, source, citation, assertion, media, repository
layout: doc
---

# Entity Types

This section defines the core entity types in GENEALOGIX. Each entity represents a distinct concept in genealogical research and can be referenced, extended, and related to other entities.

## Core Entities

### [Person](person)
Represents an individual in the family archive. Contains personal identity information, names, events, and relationships.

- **Key Properties**: Names, gender, living status, events, relationships
- **File Format**: `.glx`
- **ID Format**: `person-{id}` (see below)
- **GEDCOM Equivalent**: INDI (Individual Record)

### [Relationship](relationship)
Represents connections between people such as spouse, parent-child, and other family relationships.

- **Key Properties**: Relationship type, participants, start/end events
- **File Format**: `.glx`
- **ID Format**: `rel-{id}` (see below)
- **GEDCOM Equivalent**: FAM (Family Record)

### [Event](event)
Represents occurrences in time and place: births, marriages, deaths, baptisms, etc.

- **Key Properties**: Type, date, place, participants, description
- **File Format**: `.glx`
- **ID Format**: `event-{id}` (see below)
- **GEDCOM Equivalent**: BIRT, DEAT, MARR, BAPM, etc.

### [Place](place)
Represents geographic locations forming a hierarchical structure. Supports multiple names and historical variations.

- **Key Properties**: Name, type, hierarchy, coordinates, alternative names
- **File Format**: `.glx`
- **ID Format**: `place-{id}` (see below)
- **GEDCOM Equivalent**: PLAC (Place structures)

### [Assertion](assertion)
Represents an evidence-based conclusion about a specific genealogical fact. Forms the core of the GENEALOGIX assertion model.

- **Key Properties**: Subject, property, value, citations, confidence
- **File Format**: `.glx`
- **ID Format**: `assertion-{id}` (see below)
- **GEDCOM Equivalent**: Implicit (derived from GEDCOM structure and SOUR references)

### [Source](source)
Represents a bibliographic resource or information source. Can be books, documents, databases, websites, etc.

- **Key Properties**: Title, author, publication info, repository
- **File Format**: `.glx`
- **ID Format**: `source-{id}` (see below)
- **GEDCOM Equivalent**: SOUR (Source Record)

### [Citation](citation)
Represents a specific reference to evidence within a source. Links sources to specific pages, records, or items.

- **Key Properties**: Source reference, page, data date, locator
- **File Format**: `.glx`
- **ID Format**: `citation-{id}` (see below)
- **GEDCOM Equivalent**: SOUR.PAGE, SOUR.QUAY

### [Repository](repository)
Represents an institution or organization that holds genealogical sources (archives, libraries, databases, etc.).

- **Key Properties**: Name, type, address, contact info, access restrictions
- **File Format**: `.glx`
- **ID Format**: `repository-{id}` (see below)
- **GEDCOM Equivalent**: REPO (Repository Record)

### [Media](media)
Represents digital or physical media objects associated with genealogical entities (photographs, documents, audio, etc.).

- **Key Properties**: Title, file path, MIME type, description
- **File Format**: `.glx`
- **ID Format**: `media-{id}` (see below)
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

| Entity | Required Fields | Unique ID | Hierarchical |
|--------|-----------------|-----------|--------------|
| Person | id | ✓ | - |
| Relationship | id, type, participants | ✓ | - |
| Event | id, type, participants | ✓ | - |
| Place | id, name | ✓ | ✓ (parent) |
| Assertion | id, subject, claim | ✓ | - |
| Source | id, title | ✓ | - |
| Citation | id, source | ✓ | - |
| Repository | id, name | ✓ | - |
| Media | id, uri | ✓ | - |

### ID Format Conventions

Entity IDs use `{id}` as placeholder above. IDs can be any unique alphanumeric identifier with hyphens (1-64 chars).

See [Archive Organization - ID Format Standards](../3-archive-organization#id-format-standards) for complete details and examples.

## ID Scheme

Documentation examples use prefixes for readability:

- `person-XXXXXXXX`: Person entities
- `rel-XXXXXXXX`: Relationship entities
- `event-XXXXXXXX`: Event entities
- `place-XXXXXXXX`: Place entities
- `assertion-XXXXXXXX`: Assertion entities
- `source-XXXXXXXX`: Source entities
- `citation-XXXXXXXX`: Citation entities
- `repository-XXXXXXXX`: Repository entities
- `media-XXXXXXXX`: Media entities

> **Note:** Prefixes are not required. Any unique identifier meeting format requirements (1-64 alphanumeric characters and hyphens) is valid.

## Entity Lifecycle

### Creation
- Each entity is created with a unique ID
- Initial state is valid and complete

### Modification
- All modifications are tracked in Git history

### Deletion
- Entities are typically not deleted but rather marked as inactive
- Soft deletion patterns are preferred to preserve audit trails
- Hard deletion may break referential integrity

## Validation

The `glx validate` command performs comprehensive validation at multiple levels:

### Structural Validation
- **Entity IDs**: Must be unique, alphanumeric with hyphens, 1-64 characters
- **File structure**: Must follow proper YAML/JSON format
- **Schema compliance**: All entities validated against JSON schemas in `specification/schema/v1/`
- **Required fields**: Entity-specific required fields per schema

### Referential Integrity (Errors)
All references must point to existing entities:
- Entity references (persons, events, places, sources, citations, repositories, media, relationships)
- Vocabulary type references (event_types, relationship_types, place_types, etc.)
- Property `reference_type` values (when properties are defined as referential)

### Property Validation (Warnings)
- Unknown properties (not defined in property vocabularies) generate warnings
- Unknown assertion claims (not defined in property vocabularies) generate warnings
- Warnings allow flexibility for rapid data entry and emerging properties

See [Vocabularies - Vocabulary Validation](vocabularies#vocabulary-validation) for complete validation policy details.

## Extension Points

The GENEALOGIX specification allows extension through:
- Custom properties in entities (via `additionalProperties`)
- Custom event types
- Custom relationship types
- Custom tags and notes

See [Core Concepts](../2-core-concepts#archive-owned-vocabularies) for vocabulary and extension guidelines.

## See Also

- [Archive Organization](../3-archive-organization) - How entities are organized in files
- [Core Concepts](../2-core-concepts#evidence-chain) - How entities relate to evidence and provenance
- [Vocabularies](vocabularies) - Complete reference for all vocabulary types
- Entity type documentation includes GEDCOM mapping information

## Common Fields

| Field       | Type   | Description                               |
|-------------|--------|-------------------------------------------|
| `tags`      | array  | User-defined tags for organization        |
| `notes`     | string | Free-form notes about the entity          |

- **IDs are map keys**: The unique ID for each entity is the key in the map (`person-a1b2c3d4`).
- **Git Tracks Provenance**: Change history is handled by Git.


