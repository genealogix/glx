---
title: GENEALOGIX Specification
description: Modern, evidence-first, Git-native genealogy data standard
layout: doc
---

# GENEALOGIX Specification

[![Version](https://img.shields.io/github/v/release/genealogix/glx?include_prereleases&label=version)](https://github.com/genealogix/glx/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/genealogix/glx.svg)](https://pkg.go.dev/github.com/genealogix/glx)
[![Go Version](https://img.shields.io/github/go-mod/go-version/genealogix/glx)](https://github.com/genealogix/glx/blob/main/go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](https://github.com/genealogix/glx/blob/main/LICENSE)
[![CI](https://github.com/genealogix/glx/workflows/Validate%20Specification/badge.svg)](https://github.com/genealogix/glx/actions)
[![codecov](https://codecov.io/gh/genealogix/glx/branch/main/graph/badge.svg)](https://codecov.io/gh/genealogix/glx)
[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx)](https://goreportcard.com/report/github.com/genealogix/glx)
[![Contributors](https://img.shields.io/github/contributors/genealogix/glx.svg)](https://github.com/genealogix/glx/graphs/contributors)

The official specification for **GENEALOGIX (GLX)** — a portable, extensible archive format for genealogical research and beyond. Built on Git, designed for collaboration, and customizable through archive-owned vocabularies. Your data, your way, forever.

## Installation

Download the latest pre-compiled binary for your operating system from the [GitHub Releases](https://github.com/genealogix/glx/releases) page.

Developers can install from source:

```bash
go install github.com/genealogix/glx/glx@latest
```

## Quick Start

```bash
# Create a new genealogix repository in a new directory
glx init my-family-archive

# Or create a single-file archive
glx init my-family-archive --single-file

# Validate .glx files (checks cross-references and vocabularies)
cd my-family-archive
glx validate
```

## Why GENEALOGIX?

Traditional formats like GEDCOM solve file exchange but stop short of modern collaborative research. GENEALOGIX is a Git-native, evidence-first archive format that aims to be a permanent foundation, not just an export target.

| Challenge | GEDCOM | GENEALOGIX |
|-----------|--------|------------|
| **Collaboration** | File sharing only | Git-native workflows |
| **Evidence Tracking** | Basic source records | Complete evidence chains |
| **Version Control** | Manual or difficult | Built-in Git integration |
| **Human Readability** | Don't even try | Clear YAML structure |
| **Validation** | Syntax only | Schema-based validation |
| **Extensibility** | Limited | JSON Schema-based |
| **Data Portability** | Vendor lock-in | Open format you own |
| **Interoperability** | GEDCOM export only | Import/export + Git workflows |
| **Custom Types** | Fixed schema | Archive-defined vocabularies |

For a side-by-side look at the GEDCOM-vs-GLX wire formats and the assertion model that backs every claim with evidence, see [Core Concepts](/specification/2-core-concepts).

## Features

- **📚 Evidence-First Model** — every claim backed by documented sources
- **🔍 Quality Assessment** — structured evaluation of evidence reliability (0–3 scale)
- **🌳 Git-Native Architecture** — full version control and collaboration support
- **📋 Human-Readable Format** — clear YAML files instead of binary formats
- **✅ Schema Validation** — JSON Schema-based validation and error checking
- **🔗 Complete Provenance** — audit trail from repository to conclusion
- **🎯 Repository-Owned Vocabularies** — define custom types within each archive

## CLI Commands

The `glx` CLI groups its commands into archive management, import/export, exploration, data entry, analysis, and shell completion. See the [full CLI reference](https://genealogix.dev/cli/) for flags, examples, and per-command details.

### Archive Management

- `glx init` — initialize a new archive
- `glx validate` — validate files and cross-references
- `glx split` — convert a single-file archive to multi-file
- `glx join` — convert a multi-file archive to single-file
- `glx merge` — combine two archives with duplicate detection
- `glx migrate` — migrate an archive to the current format
- `glx rename` — rename an entity by ID

### Import & Export

- `glx import` — import a GEDCOM file
- `glx export` — export to GEDCOM or Schema.org-aligned JSON-LD

### Exploration

- `glx search` — full-text search across entities
- `glx query` — filter and list entities
- `glx vitals` — show birth, death, burial for a person
- `glx timeline` — chronological events for a person
- `glx summary` — full person profile with narrative
- `glx ancestors` — ancestor tree
- `glx descendants` — descendant tree
- `glx cite` — formatted citation text
- `glx path` — shortest relationship path between two people

### Data Entry

- `glx census` — census tooling (see subcommands)
- `glx census add` — generate entities from a census template
- `glx link` — create a FamilySearch citation from an ARK

### Analysis

- `glx stats` — entity-count and confidence dashboard
- `glx places` — place data quality issues
- `glx cluster` — FAN-club analysis
- `glx analyze` — gap, conflict, and suggestion analysis
- `glx duplicates` — detect duplicate entities
- `glx coverage` — research coverage report
- `glx diff` — diff two archives

### Shell completion

- `glx completion` — generate shell completion scripts (bash, zsh, fish, powershell)

## File Format

All GENEALOGIX files use the same structure:

```yaml
# Any .glx file
persons:
  person-a1b2c3d4:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      sex: "male"

sources:
  source-12345678:
    title: "Birth Certificate"
```

**Key Points:**

- Entity IDs are map keys: `person-john-smith` or `person-a1b2c3d4`
- IDs can be descriptive or random (1–64 alphanumeric/hyphens)
- Files can contain any combination of entity types
- Parser collates all entities across all .glx files in repository
- Controlled vocabularies define valid types in `vocabularies/` directory

## Documentation

- [🚀 Quickstart](/quickstart) — 5-minute getting started
- [💡 Examples](/examples/) — runnable sample archives
- [🛠 CLI Reference](https://genealogix.dev/cli/) — every command and flag
- [📐 Best Practices](/guides/best-practices) — recommended workflows
- [🔁 Migration from GEDCOM](/guides/migration-from-gedcom) — manual conversion guidance
- [🔀 GLX-aware Git merge driver](/docs/merge-driver) — genealogy-aware conflict resolution for .glx files
- [📖 Specification](/specification/) — full spec
- [📋 JSON Schemas](/specification/schema/) — machine-readable schemas
- [📚 Glossary](/specification/6-glossary) — key terms and concepts
- [🧱 Dev Container](https://github.com/genealogix/glx/tree/main/.devcontainer) — preconfigured dev environment

## Specification Status

This specification follows [Semantic Versioning](https://semver.org/). Current release: **v0.0.0-beta.10** (Beta).

- **Draft** — under active development, may change significantly
- **Release Candidate** — stable, final review before release
- **Released** — production-ready, changes discussed via GitHub issues and discussions

## Community

| Topic | Where |
|---|---|
| **Issues & bug reports** | [github.com/genealogix/glx/issues](https://github.com/genealogix/glx/issues) |
| **Discussions & Q&A** | [github.com/genealogix/glx/discussions](https://github.com/genealogix/glx/discussions) |
| **Chat** | [Discord](https://genealogix.io/discord) |
| **Mailing list** | [groups.google.com/g/genealogix](https://groups.google.com/g/genealogix) |
| **Contributing** | [CONTRIBUTING.md](CONTRIBUTING.md) · [website guide](/development/contributing) |
| **Code of Conduct** | [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) |
| **Security** | [SECURITY.md](SECURITY.md) · [SECURITY-POSTURE.md](SECURITY-POSTURE.md) (OSPS Baseline, EU CRA readiness) |
| **Releases** | [GitHub Releases](https://github.com/genealogix/glx/releases) |

## License

Copyright 2025 Oracynth, Inc.

Licensed under the [Apache License, Version 2.0](https://github.com/genealogix/glx/blob/main/LICENSE) (the "License");
you may not use this project except in compliance with the License.
You may obtain a copy of the License at

```text
http://www.apache.org/licenses/LICENSE-2.0
```

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Attribution for the third-party components bundled in GLX is listed in the
[NOTICE](https://github.com/genealogix/glx/blob/main/NOTICE) file.
