---
title: Introduction to GENEALOGIX
description: Overview, purpose, and key concepts of the GENEALOGIX specification.
layout: doc
---

# Introduction

## What is GENEALOGIX?

GENEALOGIX is a **permanent, human-readable archive format** for genealogical research and related domains. Unlike traditional formats, GLX is designed as a **flexible foundation you own and control**, not just an exchange format.

Built on YAML and Git, GENEALOGIX gives you a research system that will outlast any software application. Define your own event types and vocabularies. Document evidence with source citations. Track every change through version control. Whether you're researching traditional family history, colonial records, maritime history, or biographical prosopography, GLX adapts to your research domain.

### What does it look like?

A GLX archive is a folder of plain text files. Here's a person record:

```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"
```

That's it — no special software needed to read or write it. If you can open a text file, you can work with GLX.

### Why YAML?

YAML is a standard text format used across the software industry. GLX chose it for a simple reason: **you can read it**. Indentation shows structure, colons separate labels from values, and there are no cryptic tags or angle brackets to learn. If you've ever edited a configuration file or written a bulleted list, you already understand the basics.

Compare the same birth record in GEDCOM and GLX:

::: code-group

```yaml [GLX]
events:
  event-john-birth:
    type: birth
    date: "1850-03-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: subject
```

``` [GEDCOM]
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 MAR 1850
2 PLAC Leeds, Yorkshire, England
```

:::

Both represent the same information, but the GLX version is self-explanatory — you don't need a manual to understand what each line means.

### Why Git?

Git is the version control system used by millions of developers to track changes, collaborate, and maintain history. GLX is designed to work naturally with Git because genealogical research shares the same needs:

- **History**: Every edit you make is recorded. You can see exactly what changed, when, and why — no more wondering which copy of your file is the latest.
- **Collaboration**: Multiple researchers can work on the same archive simultaneously and merge their changes together, rather than emailing files back and forth.
- **Backup**: Your archive lives in a Git repository that can be mirrored anywhere — GitHub, a USB drive, your own server. If one copy is lost, the others have the full history.

You don't need to be a developer to use Git. Tools like [GitHub Desktop](https://desktop.github.com/) provide a visual interface, and the `glx` CLI handles the genealogy-specific parts for you.

## Why GENEALOGIX?

Traditional genealogy formats are limiting:
- **GEDCOM** is a decades-old format of cryptic tags that's hard to read and hard to extend
- **Proprietary formats** lock your data inside specific applications — if the software disappears, accessing your research becomes difficult
- **Fixed schemas** force every project into the same mold, with no room for specialized research domains

GENEALOGIX was designed around a different set of principles:

1. **You own your data.** Your archive is plain text files on your computer. No account required, no cloud dependency, no vendor lock-in. If GENEALOGIX itself disappeared tomorrow, your files would still be perfectly readable.

2. **Your research domain sets the rules.** Each archive defines its own controlled vocabularies — the event types, relationship types, and other categories that matter to *your* project. Studying maritime history? Define `ship_departure` and `port_arrival` events. Building an academic prosopography? Add `doctoral_advisor` relationships. The standard vocabularies give you a starting point, but you're never limited by them.

3. **Evidence should be traceable.** GLX has a built-in evidence model: Sources describe where information comes from, Citations point to specific details within a source, and Assertions record your conclusions with confidence levels. This chain from source to conclusion is how professional researchers work — GLX makes it a first-class part of the format.

4. **Every change should be tracked.** Because GLX archives are plain text in a folder structure, they work naturally with Git. Every edit is recorded, every version is preserved, and multiple researchers can collaborate without overwriting each other's work.

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
3. **Quick Start**: Follow the [5-minute tutorial](/quickstart)
4. **Examples**: Explore the [complete family example](/examples/complete-family/)
5. **Entity Specifications**: See detailed entity documentation in [Entity Types](4-entity-types/)
6. **CLI Tool**: Use the [glx command](/cli) for validation and management

## Community and Support

GENEALOGIX is open source and welcomes contributions:

- **Issues**: [Bug reports and feature requests](https://github.com/genealogix/glx/issues)
- **Discussions**: [Community Q&A and collaboration](https://github.com/genealogix/glx/discussions)
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines

## Version History

This specification follows semantic versioning. See [CHANGELOG.md](../CHANGELOG.md) for detailed version history.
