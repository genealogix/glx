---
layout: home
title: GENEALOGIX
description: A modern, evidence-first, Git-native genealogy data standard

hero:
  name: GENEALOGIX
  text: Git-Native Genealogy
  tagline: A modern, evidence-first family archive format built for collaboration
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
  - icon: 📚
    title: Evidence-First Model
    details: Every genealogical claim is backed by documented sources, creating a complete audit trail from repository to conclusion.

  - icon: 🔍
    title: Quality Assessment
    details: Structured evaluation of evidence reliability with a 0-3 quality scale and confidence levels for all assertions.

  - icon: 🌳
    title: Git-Native Architecture
    details: Full version control and collaboration support using Git workflows that genealogists already know and trust.

  - icon: 📋
    title: Human-Readable Format
    details: Clear YAML files instead of binary formats make your genealogy data easy to read, edit, and review.

  - icon: ✅
    title: Schema Validation
    details: JSON Schema-based validation ensures data integrity and catches errors before they propagate.

  - icon: 🔗
    title: Complete Provenance
    details: Full audit trail from original source documents through citations to final genealogical conclusions.

  - icon: 🎯
    title: Repository-Owned Vocabularies
    details: Define custom relationship types, event types, and other controlled vocabularies within each archive.

  - icon: 🚀
    title: Modern Tooling
    details: CLI tool with GEDCOM import, validation, split/join operations, and cross-reference checking.

  - icon: 🤝
    title: Open Standard
    details: Apache 2.0 licensed specification with an active community and open governance.
---

## Why GENEALOGIX?

Traditional genealogy formats like GEDCOM have served researchers well, but they have limitations in the modern collaborative research environment. GENEALOGIX addresses these challenges with a fresh approach designed for Git workflows, evidence-based research, and community collaboration.

### Quick Comparison

| Feature               | GEDCOM               | GENEALOGIX               |
| --------------------- | -------------------- | ------------------------ |
| **Collaboration**     | File sharing only    | Git-native workflows     |
| **Evidence Tracking** | Basic source records | Complete evidence chains |
| **Version Control**   | Manual or difficult  | Built-in Git integration |
| **Human Readability** | Binary-like format   | Clear YAML structure     |
| **Validation**        | Syntax only          | Schema-based validation  |
| **Extensibility**     | Limited              | JSON Schema based        |

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
