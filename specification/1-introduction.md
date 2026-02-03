---
title: Introduction to GENEALOGIX
description: Overview, purpose, and key concepts of the GENEALOGIX specification.
layout: doc
---

# Introduction

## Purpose and Scope

GENEALOGIX is a **portable, extensible archive format** for genealogical research and related domains. Unlike traditional formats, GLX provides a **permanent, human-readable foundation** you can own, modify, and preserve indefinitely. It's designed as a **source of truth** for collaborative research, not just an exchange format.

GENEALOGIX addresses the limitations of existing genealogy formats by providing:

- **Human-readable YAML files** instead of the GEDCOM wall of text or xml-tagged data
- **Archive autonomy** - Each archive can define its own custom events and relationships - or stick with the defaults
- **Assertion model** where all claims made by sources/citations can be explicitly defined
- **Flexible organization** - archives can be a single file, many files, or any combination
- **Domain flexibility** - Genealogy, biography, local history, prosopography, and more
- **Permanent foundation** - Open source data format that will outlast any software application
- **Interoperability by design** - Import from GEDCOM, collaborate with Git workflows

The specification covers:
- Core entity types
- Universal file format with entity type keys
- Validation tools and conformance testing
- Archive-owned "Vocabularies" to define ontologies

## GLX as a Foundation, Not Just an Exchange Format

### More Than Interoperability

While GENEALOGIX provides excellent **interoperability**, its primary purpose is to be a **permanent research foundation**:

#### Your Research System
- **Long-term storage**: Human-readable files you can read in 50 years
- **Version-controlled**: Complete research history in Git
- **Software-independent**: Not tied to any specific application
- **Self-documenting**: Evidence chains explain all conclusions

#### Customizable Research Framework
- **Domain-specific vocabularies**: Define types that match your research
- **Extensible properties**: Add custom fields without major changes
- **Flexible organization**: Single file, multi-file, or hybrid approaches
- **Research-driven**: The format adapts to your methodology

#### Collaborative Knowledge Base
- **Git workflows**: Industry-standard collaboration patterns
- **Distributed research**: Multiple teams, one coherent archive
- **Quality assurance**: Peer review through pull requests
- **Conflict resolution**: Systematic handling of competing evidence

### When to Use GLX

**GLX is ideal when you need:**
- Research that extends beyond traditional genealogy
- A permanent, versionable home for research data (not just export files)
- Collaboration with other researchers using Git workflows
- Custom types and vocabularies for specialized research
- Complete provenance and evidence documentation

## Terminology

### Core Concepts
- **Archive**: A complete GENEALOGIX data collection containing all family history data (typically stored in a Git repository)
- **Entity**: A typed record representing a person, event, place, or other genealogical concept
- **Assertion**: A discrete, evidence-backed claim about a person, event, or relationship
- **Evidence Chain**: The complete path from physical repository through source and citation to conclusion

### Entity Types
- **Person**: Individual human being with biographical information
- **Relationship**: Connection between people (parent-child, marriage, etc.)
- **Event**: Life events and facts (birth, marriage, occupation, residence, death)
- **Place**: Geographic locations with hierarchical organization
- **Media**: Photos, documents, and multimedia files

### Evidence Hierarchy
- **Repository**: Physical location (archive, library, church, government office)
- **Source**: Document or record (parish register, census, certificate)
- **Citation**: Specific reference (page number, entry number, URL)
- **Assertion**: Evidence-based conclusion or claim supported by citations (ie. person born on specific date)

## Use Cases

### Individual Research
Family historians maintaining personal archives with:
- Complete family trees with source documentation
- Research notes and evidence evaluation
- Photo and document organization
- Version-controlled research progress

### Collaborative Projects
Research teams working together on:
- Extended family documentation across multiple branches
- Surname studies and one-name studies
- Local history projects
- Genealogical society publications

### Institutional Archives
Libraries and archives preserving:
- Community genealogy collections
- Historical society records
- Government genealogy databases
- Academic research projects

## Comparison with Existing Formats

| Feature             | GENEALOGIX             | GEDCOM                  | Gramps XML   |
|---------------------|------------------------|-------------------------|--------------|
| **Format**          | YAML (human-readable)  | Custom text (tag-based) | XML          |
| **Version Control** | Git-native             | Difficult               | Manual       |
| **Evidence Model**  | Built-in citations     | Basic sources           | Complex      |
| **Collaboration**   | Git workflows          | File sharing            | Database     |
| **Validation**      | JSON Schema            | Syntax only             | Partial      |
| **Extensibility**   | Schema-based           | Limited                 | Plugin-based |

## Getting Started

The quickest way to understand GENEALOGIX is through examples:

1. **Quick Start**: Follow the [5-minute tutorial](../../docs/quickstart.md)
2. **Complete Examples**: Explore the [complete family example](../docs/examples/complete-family/)
3. **Specification Details**: Read the detailed entity specifications in sections 4-8
4. **Implementation**: Use the [CLI tool](../../glx/) for validation and management

## Community and Support

GENEALOGIX is an open-source project welcoming contributions:

- **Issues**: [Bug reports and feature requests](https://github.com/genealogix/glx/issues)
- **Discussions**: [Community Q&A and collaboration](https://github.com/genealogix/glx/discussions)
- **Contributing**: See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines

## Version History

This specification follows semantic versioning

See [CHANGELOG.md](../../CHANGELOG.md) for detailed version history.


