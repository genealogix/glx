---
title: Introduction to GENEALOGIX
description: Overview, purpose, and key concepts of the GENEALOGIX specification.
layout: doc
---

# Introduction

## Purpose and Scope

GENEALOGIX defines a **portable, extensible archive format** for genealogical research and related domains. Unlike traditional formats that lock data in proprietary structures, GLX provides a **permanent, human-readable foundation** you can own, modify, and preserve indefinitely. It's designed as a **source of truth** for collaborative research, not just an exchange format.

GENEALOGIX addresses the limitations of existing genealogy formats by providing:

- **Git-native architecture** for reliable collaboration and version control
- **Evidence-first model** where all claims are backed by documented sources
- **Human-readable YAML files** instead of binary or proprietary formats
- **Structured data validation** with JSON Schema compliance
- **Complete provenance tracking** from repository to conclusion
- **Flexible organization** - archives can be a single file, many files, or any combination
- **True data ownership** - Human-readable files you can edit anywhere
- **Archive autonomy** - Each repository defines its own controlled vocabularies
- **Domain flexibility** - Genealogy, biography, local history, prosopography, and more
- **Permanent foundation** - Data format that will outlast any software application
- **Interoperability by design** - Import from GEDCOM, integrate with Git workflows

The specification covers:
- 9 core entity types for comprehensive family history documentation
- Universal file format with entity type keys
- Evidence hierarchy from physical repositories to specific claims
- Git workflow integration for collaborative research
- Extensible schema system for future enhancements
- Validation tools and conformance testing

**File Format**: All GENEALOGIX files use the same structure with top-level entity type keys (persons, sources, etc.) containing maps of entities. Parsers collate all entities of each type across all .glx files in the repository.

## Design Principles

### Clarity and Simplicity
- **YAML-based files** that are readable and editable in any text editor
- **Consistent naming conventions** with structured ID formats
- **Hierarchical organization** following standard archival practices
- **Minimal required fields** with rich optional metadata

### Evidence-First Architecture
- **Source-backed assertions** - every claim must reference evidence
- **Confidence levels** - researcher assessment of conclusion certainty
- **Citation specificity** - exact references to source locations
- **Multiple evidence support** - corroboration from multiple sources

### Provenance and Traceability
- **Complete audit trails** from repository to genealogical conclusion
- **Author attribution** for all changes and contributions
- **Timestamp tracking** for research chronology
- **Change documentation** through Git commit history

### Git-Native Collaboration
- **Branch-based research** - isolate investigations in feature branches
- **Merge conflict resolution** for conflicting evidence
- **Pull request reviews** for quality assurance
- **Tag-based releases** for milestone preservation

## GLX as a Foundation, Not Just an Exchange Format

### More Than Interoperability

While GENEALOGIX provides excellent **interoperability** (importing from GEDCOM, exporting to various formats), its primary purpose is to be a **permanent research foundation**:

#### Your Research System
- **Long-term storage**: Human-readable files you can read in 50 years
- **Version-controlled**: Complete research history in Git
- **Software-independent**: Not tied to any specific application
- **Self-documenting**: Evidence chains explain all conclusions

#### Customizable Research Framework
- **Domain-specific vocabularies**: Define types that match your research
- **Extensible properties**: Add custom fields without schema changes
- **Flexible organization**: Single file, multi-file, or hybrid approaches
- **Research-driven**: The format adapts to your methodology

#### Collaborative Knowledge Base
- **Git workflows**: Industry-standard collaboration patterns
- **Distributed research**: Multiple teams, one coherent archive
- **Quality assurance**: Peer review through pull requests
- **Conflict resolution**: Systematic handling of competing evidence

### When to Use GLX

**GLX is ideal when you need:**
- A permanent home for research data (not just temporary export files)
- Collaboration with other researchers using Git workflows
- Custom types and vocabularies for specialized research
- Complete provenance and evidence documentation
- Research that extends beyond traditional genealogy

**GLX may not be necessary if you:**
- Only need to transfer data between two applications (GEDCOM suffices)
- Have simple trees with no collaborative needs
- Don't require evidence documentation or provenance tracking

## Terminology

### Core Concepts
- **Archive**: A complete GENEALOGIX repository containing all family history data
- **Entity**: A typed record representing a person, event, place, or other genealogical concept
- **Assertion**: A discrete, evidence-backed claim about a person, event, or relationship
- **Evidence Chain**: The complete path from physical repository through source and citation to conclusion

### Entity Types
- **Person**: Individual human being with biographical information
- **Relationship**: Connection between people (parent-child, marriage, etc.)
- **Event**: Life events and facts (birth, marriage, occupation, residence, death)
- **Place**: Geographic locations with hierarchical organization
- **Source**: Original materials (books, records, certificates, websites)
- **Citation**: Specific reference within a source with quality assessment
- **Repository**: Physical or digital archive holding sources
- **Assertion**: Evidence-based conclusion or claim
- **Media**: Supporting photos, documents, and multimedia files

### Evidence Hierarchy
- **Repository**: Physical location (archive, library, church, government office)
- **Source**: Document or record (parish register, census, certificate)
- **Citation**: Specific reference (page number, entry number, URL)
- **Assertion**: Claim supported by citations (person born on specific date)

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

### Migration and Integration
Converting from existing formats:
- GEDCOM format compatibility guidance (manual conversion)
- Legacy database migration patterns
- Paper record digitization guidance
- Integration with existing genealogy software workflows

## Comparison with Existing Formats

| Feature | GENEALOGIX | GEDCOM | Gramps XML |
|---------|------------|--------|------------|
| **Format** | YAML (human-readable) | Custom text (tag-based) | XML |
| **Version Control** | Git-native | Difficult | Manual |
| **Evidence Model** | Built-in citations | Basic sources | Complex |
| **Collaboration** | Git workflows | File sharing | Database |
| **Validation** | JSON Schema | Syntax only | Partial |
| **Extensibility** | Schema-based | Limited | Plugin-based |

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

This specification follows semantic versioning:

- **Version 0.0.0-beta.2**: Beta release
- **Version 1.0**: Initial stable release (future)
- **Version 1.1+**: Backwards-compatible enhancements (future)
- **Version 2.0**: May include breaking changes with migration path (future)

See [CHANGELOG.md](../../CHANGELOG.md) for detailed version history.


