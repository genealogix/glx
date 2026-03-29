---
title: GENEALOGIX Specification
description: Complete technical specification for the GENEALOGIX archive format.
layout: doc
---

# GENEALOGIX Specification

Version 0.0.0-beta.9

## Table of Contents

1. [Introduction](1-introduction)
   - Purpose and Scope
   - Design Principles
   - Terminology

2. [Core Concepts](2-core-concepts)
   - Archive-Owned Vocabularies
   - Entity Relationships
   - Data Types (Primitive, Temporal, Reference)
   - Properties and Assertions
   - Evidence Chain
   - Collaboration with Git

3. [Archive Organization](3-archive-organization)
   - Repository Layout
   - Naming Conventions
   - File Organization Patterns

4. [Entity Types](4-entity-types/)
   - [Person](4-entity-types/person) - Individual records
   - [Relationship](4-entity-types/relationship) - Connections between people
   - [Event](4-entity-types/event) - Occurrences in time and place
   - [Place](4-entity-types/place) - Geographic locations with hierarchy
   - [Assertion](4-entity-types/assertion) - Evidence-based conclusions
   - [Source](4-entity-types/source) - Bibliographic resources
   - [Citation](4-entity-types/citation) - References to specific evidence
   - [Repository](4-entity-types/repository) - Institutions holding sources
   - [Media](4-entity-types/media) - Photographs, documents, etc.
   - [Vocabularies](4-entity-types/vocabularies) - Controlled type definitions (not an entity type)

5. [Standard Vocabularies](5-standard-vocabularies/)
   - Standard vocabulary templates for archive initialization

6. [Glossary](6-glossary)
   - Key terms and definitions

## Specification Status

This specification is under active development.

- **Version**: 0.0.0-beta.9
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
      gender: "male"
      born_on: "1850-01-15"
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

1. Read [Introduction](1-introduction) for overview
2. Review [Glossary](6-glossary) for key terms and definitions
3. Read [Core Concepts](2-core-concepts) to understand the architecture
4. Review [Entity Types](4-entity-types/) to understand data structure
5. Check [Archive Organization](3-archive-organization) for organization patterns
6. Review [Standard Vocabularies](5-standard-vocabularies/) for controlled type definitions
7. See [examples/](/examples/) for working examples
8. Use [glx CLI](/cli) for validation

## Contributing

Major changes are discussed via GitHub issues and discussions. See [CONTRIBUTING.md](../CONTRIBUTING.md)

## License

This specification is licensed under the Apache License 2.0
