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

- **Key Properties**: Names, gender, birth/death dates, events, relationships
- **GEDCOM Equivalent**: INDI (Individual Record)

### [Relationship](relationship)
Represents connections between people such as spouse, parent-child, and other family relationships.

- **Key Properties**: Relationship type, participants, start/end events
- **GEDCOM Equivalent**: FAM (Family Record)

### [Event](event)
Represents occurrences in time and place: births, marriages, deaths, baptisms, etc.

- **Key Properties**: Type, date, place, participants, notes
- **GEDCOM Equivalent**: BIRT, DEAT, MARR, BAPM, etc.

### [Place](place)
Represents geographic locations forming a hierarchical structure. Supports multiple names and historical variations.

- **Key Properties**: Name, type, hierarchy, coordinates, alternative names (via properties)
- **GEDCOM Equivalent**: PLAC (Place structures)

### [Assertion](assertion)
Represents an evidence-based conclusion about a specific genealogical fact. Forms the core of the GENEALOGIX assertion model.

- **Key Properties**: Subject, property, value, citations, confidence
- **GEDCOM Equivalent**: Implicit (derived from GEDCOM structure and SOUR references)

### [Source](source)
Represents a bibliographic resource or information source. Can be books, documents, databases, websites, etc.

- **Key Properties**: Title, author, publication info, repository
- **GEDCOM Equivalent**: SOUR (Source Record)

### [Citation](citation)
Represents a specific reference to evidence within a source. Links sources to specific pages, records, or items.

- **Key Properties**: Source reference, page, data date, locator
- **GEDCOM Equivalent**: SOUR.PAGE, SOUR.QUAY

### [Repository](repository)
Represents an institution or organization that holds genealogical sources (archives, libraries, databases, etc.).

- **Key Properties**: Name, type, address, contact info, access restrictions
- **GEDCOM Equivalent**: REPO (Repository Record)

### [Media](media)
Represents digital or physical media objects associated with genealogical entities (photographs, documents, audio, etc.).

- **Key Properties**: Title, URI, MIME type, description
- **GEDCOM Equivalent**: OBJE (Object/Media Record)

## Entity Relationships

```
Person
  ├── participates in Events (birth, marriage, immigration, etc.)
  ├── has many Properties
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
  ├── has alternative names (via properties)
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

## See Also

- [Archive Organization](../3-archive-organization) - How entities are organized in files
- [Core Concepts](../2-core-concepts#evidence-chain) - How entities relate to evidence and provenance
- [Vocabularies](vocabularies) - Complete reference for all vocabulary types
- Entity type documentation includes GEDCOM mapping information
