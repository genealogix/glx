---
title: GENEALOGIX Specification
description: Complete technical specification for the GENEALOGIX family archive format.
layout: doc
---

# GENEALOGIX Specification

Version 1.0.0 (Draft)

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

5. [Data Model](5-data-model/)
   - [Assertion Model](5-data-model/assertion-model.md) - Evidence framework
   - [Evidence Hierarchy](5-data-model/evidence-hierarchy.md) - Provenance levels
   - [Provenance Tracking](5-data-model/provenance-tracking.md) - Audit trails
   - [Confidence Scales](5-data-model/confidence-scales.md) - Assessing reliability

6. [Extensibility](6-extensibility/)
   - [Custom Types](6-extensibility/custom-types.md)
   - [Schema Registry](6-extensibility/schema-registry.md)
   - [Versioning Strategy](6-extensibility/versioning.md)

7. [Git Integration](7-git-integration/)
   - [Merge Strategies](7-git-integration/merge-strategies.md)
   - [Conflict Resolution](7-git-integration/conflict-resolution.md)
   - [Branch Workflows](7-git-integration/branch-workflows.md)

8. [Interoperability](8-interoperability/)
   - [GEDCOM Mapping](8-interoperability/gedcom-mapping.md)
   - [Gramps XML Mapping](8-interoperability/gramps-mapping.md)

## Normative References

This specification uses RFC 2119 keywords (MUST, SHOULD, MAY) for
requirement levels.

## Reading Guide

- **Implementers**: Start with Core Concepts and Entity Types
- **Users**: Start with Introduction and File Structure
- **Contributors**: Read the entire spec plus CONTRIBUTING.md

## Specification Status

This specification is under active development.

- **Version**: 1.0.0
- **Status**: Draft
- **Stability**: Experimental (breaking changes possible)

## Key Features

- **Assertion-Based Model**: Every genealogical fact is supported by explicitly tracked evidence
- **Multi-Tenant**: Supports family-level isolation within organizations
- **Git-Native**: Built from the ground up for version control
- **GEDCOM Compatible**: Can be imported from and exported to GEDCOM format
- **Hierarchical Places**: Supports complex place hierarchies with historical variations
- **Evidence Quality**: Uses GEDCOM QUAY standard for assessing source reliability
- **Extensible**: Custom entity types and properties supported via repository-owned vocabularies

## Quick Example

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith-1850:
    version: "1.0"
    concluded_identity:
      primary_name: "John Smith"
      gender: male
      living: false
    notes: "Blacksmith in Leeds, Yorkshire"

# events/event-birth.glx
events:
  event-birth-john:
    version: "1.0"
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith-1850
        role: subject

# relationships/rel-marriage.glx
relationships:
  rel-marriage-john-mary:
    version: "1.0"
    type: marriage
    persons:
      - person-john-smith-1850
      - person-mary-brown-1852
```

## Getting Started

1. Read [Introduction](1-introduction.md) for overview
2. Review [Entity Types](4-entity-types/) to understand data structure
3. Check [Archive Organization](3-archive-organization.md) for organization patterns
4. See [examples/](../docs/examples/) for working examples
5. Use [glx CLI](../../glx/) for validation

## Contributing

Major changes require an RFC. See [CONTRIBUTING.md](../CONTRIBUTING.md)

## License

This specification is licensed under the Apache License 2.0


