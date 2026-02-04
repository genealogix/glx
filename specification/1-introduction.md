---
title: Introduction to GENEALOGIX
description: Overview, purpose, and key concepts of the GENEALOGIX specification.
layout: doc
---

# Introduction

## What is GENEALOGIX?

GENEALOGIX is a **permanent, human-readable archive format** for genealogical research and related domains. Unlike traditional formats, GLX is designed as a **flexible foundation you own and control**, not just an exchange format.

Built on YAML and Git, GENEALOGIX gives you a research system that will outlast any software application. Define your own event types and vocabularies. Document evidence with source citations. Track every change through version control. Whether you're researching traditional family history, colonial records, maritime history, or biographical prosopography, GLX adapts to your research domain.

## Why GENEALOGIX?

Traditional genealogy formats are limiting:
- **GEDCOM** is a 30-year-old wall of tags that's hard to read and hard to extend
- **Proprietary formats** lock your data in specific applications
- **Fixed schemas** can't accommodate specialized research domains

GENEALOGIX gives you control: **flexible vocabularies** for domain-specific types, **human-readable files** you can edit in any text editor, **Git-native design** for version control and collaboration, and a **permanent foundation** that will outlast any software.

## Who is it for?

- **Researchers** who need permanent, versionable archives independent of any software
- **Collaborative teams** using Git workflows for distributed research
- **Specialized projects** requiring custom types (colonial history, religious studies, maritime research, biography)
- **Anyone** wanting rigorous evidence documentation with source citations and confidence levels

## Comparison with Existing Formats

| Feature             | GENEALOGIX             | GEDCOM                  | Gramps XML   |
|---------------------|------------------------|-------------------------|--------------|
| **Format**          | YAML (human-readable)  | Custom text (tag-based) | XML          |
| **Version Control** | Git-native             | Difficult               | Manual       |
| **Evidence Model**  | Built-in assertions    | Basic sources           | Complex      |
| **Collaboration**   | Git workflows          | File sharing            | Database     |
| **Validation**      | JSON Schema            | Syntax only             | Partial      |
| **Extensibility**   | Archive-owned types    | Limited                 | Plugin-based |

## Getting Started

The quickest way to understand GENEALOGIX:

1. **Glossary**: Review key terms in the [Glossary](6-glossary)
2. **Core Concepts**: Read [Core Concepts](2-core-concepts) to understand the architecture
3. **Quick Start**: Follow the [5-minute tutorial](../../docs/quickstart.md)
4. **Examples**: Explore the [complete family example](../docs/examples/complete-family/)
5. **Entity Specifications**: See detailed entity documentation in [Entity Types](4-entity-types)
6. **CLI Tool**: Use the [glx command](../../glx/) for validation and management

## Community and Support

GENEALOGIX is open source and welcomes contributions:

- **Issues**: [Bug reports and feature requests](https://github.com/genealogix/glx/issues)
- **Discussions**: [Community Q&A and collaboration](https://github.com/genealogix/glx/discussions)
- **Contributing**: See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines

## Version History

This specification follows semantic versioning. See [CHANGELOG.md](../../CHANGELOG.md) for detailed version history.
