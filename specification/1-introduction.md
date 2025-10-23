# Introduction

## Purpose and Scope

GENEALOGIX defines a human-readable, version-controlled archive format for family history data. It addresses the limitations of existing genealogy formats by providing:

- **Git-native architecture** for reliable collaboration and version control
- **Evidence-first model** where all claims are backed by documented sources
- **Human-readable YAML files** instead of binary or proprietary formats
- **Structured data validation** with JSON Schema compliance
- **Complete provenance tracking** from repository to conclusion

The specification covers:
- 9 core entity types for comprehensive family history documentation
- Evidence hierarchy from physical repositories to specific claims
- Git workflow integration for collaborative research
- Extensible schema system for future enhancements
- Validation tools and conformance testing

## Design Principles

### Clarity and Simplicity
- **YAML-based files** that are readable and editable in any text editor
- **Consistent naming conventions** with structured ID formats
- **Hierarchical organization** following standard archival practices
- **Minimal required fields** with rich optional metadata

### Evidence-First Architecture
- **Source-backed assertions** - every claim must reference evidence
- **Quality assessment** - structured evaluation of evidence reliability
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

### Quality Assessment
- **Quality Rating**: 0-3 scale indicating evidence reliability
  - 3 = Primary, direct evidence (birth certificate)
  - 2 = Secondary, direct evidence (census record)
  - 1 = Primary, indirect evidence (family Bible notation)
  - 0 = Secondary, indirect evidence (published biography)

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
- GEDCOM file import and cleanup
- Legacy database migration
- Paper record digitization
- Integration with existing genealogy software

## Comparison with Existing Formats

| Feature | GENEALOGIX | GEDCOM | Gramps XML |
|---------|------------|--------|------------|
| **Format** | YAML (human-readable) | Custom binary | XML |
| **Version Control** | Git-native | Difficult | Manual |
| **Evidence Model** | Built-in citations | Basic sources | Complex |
| **Collaboration** | Git workflows | File sharing | Database |
| **Validation** | JSON Schema | Syntax only | Partial |
| **Extensibility** | Schema-based | Limited | Plugin-based |

## Getting Started

The quickest way to understand GENEALOGIX is through examples:

1. **Quick Start**: Follow the [5-minute tutorial](../../docs/quickstart.md)
2. **Complete Examples**: Explore the [complete family example](../../examples/complete-family/)
3. **Specification Details**: Read the detailed entity specifications in sections 4-8
4. **Implementation**: Use the [CLI tool](../../glx/) for validation and management

## Community and Support

GENEALOGIX is an open-source project welcoming contributions:

- **Issues**: [Bug reports and feature requests](https://github.com/genealogix/spec/issues)
- **Discussions**: [Community Q&A and collaboration](https://github.com/genealogix/spec/discussions)
- **RFCs**: [Major changes through Request for Comments](../../rfcs/)
- **Contributing**: See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines

## Version History

This specification follows semantic versioning:

- **Version 1.0**: Initial release with 9 core entity types
- **Version 1.1+**: Backwards-compatible enhancements
- **Version 2.0**: May include breaking changes with migration path

See [CHANGELOG.md](../../CHANGELOG.md) for detailed version history.


