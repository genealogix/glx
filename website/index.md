---
layout: home
title: GENEALOGIX
description: A modern, evidence-first, Git-native genealogy data standard

hero:
  name: GENEALOGIX
  text: Git-Native Genealogy
  tagline: A portable, extensible family archive format that's yours to keep forever
  image:
    src: /logo.svg
    alt: GENEALOGIX
  actions:
    - theme: brand
      text: Get Started
      link: /quickstart
    - theme: alt
      text: View Specification
      link: /specification/
    - theme: alt
      text: Examples
      link: /examples/

features:
  - icon: 🔄
    title: True Data Portability
    details: Your data in human-readable YAML files you can read, edit, and migrate anywhere - no vendor lock-in, no proprietary formats.

  - icon: ⚡
    title: Infinitely Extensible
    details: Define custom event types, relationship types, and properties for ANY domain - genealogy, biography, local history, and beyond.

  - icon: 🎯
    title: Your Types, Your Rules
    details: Each archive defines its own controlled vocabularies - from traditional genealogy to custom research domains. No central registry, no dependencies.

  - icon: 🚀
    title: Modern Tooling
    details: CLI tool with GEDCOM import, validation, split/join operations, and cross-reference checking.

  - icon: 🤝
    title: Open Standard
    details: Apache 2.0 licensed specification with an active community and open governance.

  - icon: 🌳
    title: Git-Native Architecture
    details: Full version control and collaboration support using Git workflows that genealogists already know and trust.

  - icon: 📋
    title: Human-Readable Format
    details: Clear YAML files instead of binary formats make your genealogy data easy to read, edit, and review.

  - icon: 📚
    title: Evidence-First Model
    details: Every genealogical claim is backed by documented sources, creating a complete audit trail from repository to conclusion.

  - icon: 🔍
    title: Confidence Levels
    details: Express researcher certainty in conclusions with structured confidence levels for all assertions.

  - icon: ✅
    title: Schema Validation
    details: JSON Schema-based validation ensures data integrity and catches errors before they propagate.

  - icon: 🔗
    title: Complete Provenance
    details: Full audit trail from original source documents through citations to final genealogical conclusions.
---

## Why GENEALOGIX?

Traditional genealogy software traps your research in proprietary databases and limited file formats. GENEALOGIX gives you **true data ownership** with human-readable files you can edit in any text editor, store anywhere, and collaborate on using Git. Whether you're documenting traditional family trees, researching local history, or building biographical databases, GLX adapts to **your research needs** - not the other way around. It's a **permanent foundation** for your work that will outlast any single software tool.

### Quick Comparison

| Feature               | GEDCOM               | GENEALOGIX               |
| --------------------- | -------------------- | ------------------------ |
| **Collaboration**     | File sharing only    | Git-native workflows     |
| **Evidence Tracking** | Basic source records | Complete evidence chains |
| **Version Control**   | Manual or difficult  | Built-in Git integration |
| **Human Readability** | Binary-like format   | Clear YAML structure     |
| **Validation**        | Syntax only          | Schema-based validation  |
| **Extensibility**     | Limited              | JSON Schema based        |
| **Data Portability**  | Vendor lock-in       | Open format you own      |
| **Interoperability**  | GEDCOM export only   | Import/export + Git workflows |
| **Custom Types**      | Fixed schema         | Archive-defined vocabularies |

## Quick Start

```bash
# Install the glx CLI tool
go install github.com/genealogix/glx/glx@latest

# Import from GEDCOM
glx import family.ged -o family.glx

# Or create a new genealogix repository
glx init

# Validate your archive
glx validate
```

## Community

- 🐛 [Report Issues](https://github.com/genealogix/glx/issues)
- 💬 [Discussions](https://github.com/genealogix/glx/discussions)
- 🤝 [Contributing Guide](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md)
- 📖 [Full Specification](/specification/)

---

**Made with ❤️ for genealogists, by genealogists**

Licensed under [Apache License 2.0](https://github.com/genealogix/glx/blob/main/LICENSE) • Copyright © 2025 Oracynth, Inc.
