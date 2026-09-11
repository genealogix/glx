---
title: GENEALOGIX Specification
description: Complete technical specification for the GENEALOGIX archive format.
layout: doc
---

# GENEALOGIX Specification

Version 0.0.0-beta.12

## Table of Contents

1. [Introduction](1-introduction.md)
   - What is GENEALOGIX?
   - Why GENEALOGIX?
   - Comparison with Existing Formats

2. [Core Concepts](2-core-concepts.md)
   - Archive-Owned Vocabularies
   - Entity Relationships
   - Data Types (Primitive, Temporal, Reference)
   - Properties and Assertions
   - Evidence Chain
   - Collaboration with Git

3. [Archive Organization](3-archive-organization.md)
   - GLX File Format and Archive Metadata
   - Validation Levels
   - Organization Strategies, Media File Storage, and ID Format Standards

4. [Entity Types](4-entity-types/)
   - [Person](4-entity-types/person.md) - Individual records
   - [Relationship](4-entity-types/relationship.md) - Connections between people
   - [Event](4-entity-types/event.md) - Occurrences in time and place
   - [Place](4-entity-types/place.md) - Geographic locations with hierarchy
   - [Assertion](4-entity-types/assertion.md) - Evidence-based conclusions
   - [Source](4-entity-types/source.md) - Bibliographic resources
   - [Citation](4-entity-types/citation.md) - References to specific evidence
   - [Repository](4-entity-types/repository.md) - Institutions holding sources
   - [Media](4-entity-types/media.md) - Photographs, documents, etc.
   - [ResearchLog](4-entity-types/research-log.md) - Research investigations and search history
   - [Study](4-entity-types/study.md) - Research-project scope (One Place / One Name / family reconstruction)
   - [Vocabularies](4-entity-types/vocabularies.md) - Controlled type definitions (not an entity type)

5. [Standard Vocabularies](5-standard-vocabularies/)
   - Standard vocabulary templates for archive initialization

6. [Glossary](6-glossary.md)
   - Key terms and definitions

## Specification Status

This specification is under active development.

- **Version**: 0.0.0-beta.12
- **Status**: Beta
- **Stability**: Unstable API (breaking changes possible)

## Key Features

- **Assertion-Based Model**: Every genealogical fact can be supported by explicitly tracked evidence
- **Git-Native**: Built from the ground up for version control
- **Hierarchical Places**: Supports complex place hierarchies with historical variations
- **Extensible**: Custom entity types and properties supported via archive-owned vocabularies

## Quick Example

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith-1850:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      sex: "male"
      occupation: "blacksmith"
    notes: "Blacksmith in Leeds, Yorkshire"

# events/event-birth.glx
events:
  event-birth-john:
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith-1850
        role: subject

# relationships/rel-marriage.glx
relationships:
  rel-marriage-john-mary:
    type: marriage
    participants:
      - person: person-john-smith-1850
        role: spouse
      - person: person-mary-brown-1852
        role: spouse
```

## Getting Started

1. Read [Introduction](1-introduction.md) for overview
2. Review [Glossary](6-glossary.md) for key terms and definitions
3. Read [Core Concepts](2-core-concepts.md) to understand the architecture
4. Review [Entity Types](4-entity-types/) to understand data structure
5. Check [Archive Organization](3-archive-organization.md) for organization patterns
6. Review [Standard Vocabularies](5-standard-vocabularies/) for controlled type definitions
7. See [examples/](../docs/examples/README.md) for working examples
8. Use [glx CLI](../docs/cli/index.md) for validation

## Contributing

Major changes are discussed via GitHub issues and discussions. See [Contributing](../CONTRIBUTING.md)

## License

This specification is licensed under the Apache License 2.0
