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
---

## Why GENEALOGIX?

Traditional genealogy software traps your research in proprietary databases and limited file formats. GENEALOGIX gives you **true data ownership** with human-readable files you can edit in any text editor, store anywhere, and collaborate on using Git. Whether you're documenting traditional family trees, researching local history, or building biographical databases, GLX adapts to **your research needs** - not the other way around. It's a **permanent foundation** for your work that will outlast any single software tool.

### Quick Comparison

| Feature               | GEDCOM                       | GENEALOGIX                    |
| --------------------- | ---------------------------- | ----------------------------- |
| **Collaboration**     | File sharing only            | Git-native workflows          |
| **Evidence Tracking** | Basic source records         | Complete evidence chains      |
| **Version Control**   | Manual or difficult          | Built-in Git integration      |
| **Human Readability** | Binary-like format           | Clear YAML structure          |
| **Validation**        | Inconsistent implementations | Schema-based validation       |
| **Extensibility**     | Limited                      | JSON Schema based             |
| **Data Portability**  | Vendor lock-in               | Open format you own           |
| **Interoperability**  | GEDCOM export only           | Import/export + Git workflows |
| **Custom Types**      | Fixed schema                 | Archive-defined vocabularies  |

## What is a GLX Archive?

A GENEALOGIX archive is a collection of plain YAML files — one per person, event, place, source, and so on — organized in a simple folder structure. Each archive also includes vocabulary files that define the types your research uses (event types, relationship types, etc.), so the archive is completely self-describing.

Because it's just files and folders, you can edit your archive in any text editor, store it anywhere, back it up however you like, and track every change with Git. There's no database, no proprietary binary format, and no software required to read your data.

::: tip Ready to dive in?
Follow the [Quickstart Guide](/quickstart) to create your first archive in 5 minutes, or read the [Core Concepts](/specification/2-core-concepts) to understand the architecture.
:::

## Migrating from GEDCOM?

If you already have a GEDCOM file, the `glx` CLI can import it automatically:

```bash
glx import family.ged -o family-archive
```

See the full [Migration from GEDCOM](/guides/migration-from-gedcom) guide for field mapping details, troubleshooting, and post-migration workflow.

## Community

- 🐛 [Report Issues](https://github.com/genealogix/glx/issues)
- 💬 [Discussions](https://github.com/genealogix/glx/discussions)
- 🤝 [Contributing Guide](https://github.com/genealogix/glx/blob/main/CONTRIBUTING.md)
- 📖 [Full Specification](/specification/)

---

**Made with ❤️ for genealogists, by genealogists**

Licensed under [Apache License 2.0](https://github.com/genealogix/glx/blob/main/LICENSE) • Copyright © 2025-2026 Oracynth, Inc.
