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

3. [File Structure](3-file-structure.md)
   - Repository Layout
   - Naming Conventions
   - File Organization Patterns

4. [Entity Types](4-entity-types/)
   - [Person](4-entity-types/person.md)
   - [Relationship](4-entity-types/relationship.md)
   - [Assertion](4-entity-types/assertion.md)
   - [Source](4-entity-types/source.md)
   - [Media](4-entity-types/media.md)

5. [Data Model](5-data-model/)
   - [Assertion Model](5-data-model/assertion-model.md)
   - [Evidence Hierarchy](5-data-model/evidence-hierarchy.md)
   - [Provenance Tracking](5-data-model/provenance-tracking.md)
   - [Confidence Scales](5-data-model/confidence-scales.md)

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


