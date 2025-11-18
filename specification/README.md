---
title: GENEALOGIX Specification
description: Complete technical specification for the GENEALOGIX family archive format.
layout: doc
---

# GENEALOGIX Specification

Version 0.0.0-beta.1 (Beta)

## Table of Contents

1. [Introduction](1-introduction.md)
   - Purpose and Scope
   - Design Principles
   - Terminology

2. [Core Concepts](2-core-concepts.md)
   - Assertion-Aware Data Model
   - Evidence Hierarchy
   - Provenance Tracking
   - Version Control Integration

3. [Archive Organization](3-archive-organization.md)
   - Repository Layout
   - Naming Conventions
   - File Organization Patterns

4. [Entity Types](4-entity-types/)
   - [Person](4-entity-types/person.md) - Individual records
   - [Relationship](4-entity-types/relationship.md) - Connections between people
   - [Event/Fact](4-entity-types/event.md) - Occurrences in time and place
   - [Place](4-entity-types/place.md) - Geographic locations with hierarchy
   - [Assertion](4-entity-types/assertion.md) - Evidence-based conclusions
   - [Source](4-entity-types/source.md) - Bibliographic resources
   - [Citation](4-entity-types/citation.md) - References to specific evidence
   - [Repository](4-entity-types/repository.md) - Institutions holding sources
   - [Media](4-entity-types/media.md) - Photographs, documents, etc.
   - [Vocabularies](4-entity-types/vocabularies.md) - Controlled type definitions

5. [Standard Vocabularies](5-standard-vocabularies/)
   - Standard vocabulary templates for archive initialization

6. [Data Types](6-data-types.md)
   - Primitive Types (string, date, integer, boolean)
   - Temporal Values
   - Reference Types

## Normative References

This specification uses RFC 2119 keywords (MUST, SHOULD, MAY) for
requirement levels.

## Reading Guide

- **Implementers**: Start with Core Concepts and Entity Types
- **Users**: Start with Introduction and File Structure
- **Contributors**: Read the entire spec plus CONTRIBUTING.md

## Specification Status

This specification is under active development.

- **Version**: 0.0.0-beta.1
- **Status**: Beta
- **Stability**: Stable API (breaking changes unlikely but possible)

## Key Features

- **Assertion-Based Model**: Every genealogical fact is supported by explicitly tracked evidence
- **Multi-Tenant**: Supports family-level isolation within organizations
- **Git-Native**: Built from the ground up for version control
- **Hierarchical Places**: Supports complex place hierarchies with historical variations
- **Evidence Quality**: Supports GEDCOM QUAY standard (0-3 scale) for source reliability assessment; however, assertion confidence levels are the preferred GENEALOGIX approach and quality scales are customizable per archive
- **Extensible**: Custom entity types and properties supported via repository-owned vocabularies

## Quick Example

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith-1850:
    properties:
      given_name: "John"
      family_name: "Smith"
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
    persons:
      - person-john-smith-1850
      - person-mary-brown-1852
```

## Documentation Conventions

### Internal Links
Internal links omit the .md file extension for VitePress compatibility:
- ✓ Good: [Person Entity](4-entity-types/person)
- ✗ Bad: [Person Entity](4-entity-types/person.md)

This works correctly in both VitePress-generated site and raw markdown viewers.

### File Organization
Examples may show entities in single files or multiple files. Both are valid:
- Single file: Simpler for small examples and personal archives
- Multiple files: Better for collaboration (cleaner git diffs)

See [Archive Organization](3-archive-organization) for details.

## Getting Started

1. Read [Introduction](1-introduction.md) for overview
2. Review [Entity Types](4-entity-types/) to understand data structure
3. Check [Archive Organization](3-archive-organization.md) for organization patterns
4. Review [Standard Vocabularies](5-standard-vocabularies/) for controlled type definitions
5. See [examples/](../docs/examples/) for working examples
6. Use [glx CLI](../../glx/) for validation

## Contributing

Major changes are discussed via GitHub issues and discussions. See [CONTRIBUTING.md](../CONTRIBUTING.md)

## License

This specification is licensed under the Apache License 2.0


