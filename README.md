---
title: GENEALOGIX Specification
description: Modern, evidence-first, Git-native genealogy data standard
layout: doc
---

# GENEALOGIX Specification

[![Version](https://img.shields.io/badge/version-0.0.0--beta.6-blue.svg)](https://github.com/genealogix/glx/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](https://github.com/genealogix/glx/blob/main/LICENSE)
[![CI](https://github.com/genealogix/glx/workflows/Validate%20Specification/badge.svg)](https://github.com/genealogix/glx/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/glx)](https://goreportcard.com/report/github.com/genealogix/glx)
[![Contributors](https://img.shields.io/github/contributors/genealogix/glx.svg)](https://github.com/genealogix/glx/graphs/contributors)

The official specification for **GENEALOGIX (GLX)** - a portable, extensible archive format for genealogical research and beyond. Built on Git, designed for collaboration, and customizable through archive-owned vocabularies. Your data, your way, forever.

## Quick Links

- [📖 Read the Specification](/specification/)
- [📋 JSON Schemas](/specification/schema/)
- [💡 Examples](/examples/)
- [🧪 Test Suite](https://github.com/genealogix/glx/tree/main/glx/tests)
- [🛠 CLI](/cli)
- [🧱 Dev Container](https://github.com/genealogix/glx/tree/main/.devcontainer)

## Why GENEALOGIX?

Traditional genealogy formats like GEDCOM have served researchers well, but they have limitations in the modern collaborative research environment. GENEALOGIX addresses these challenges with a fresh approach.

### The Problem with Traditional Formats

| Challenge | GEDCOM | GENEALOGIX |
|-----------|--------|------------|
| **Collaboration** | File sharing only | Git-native workflows |
| **Evidence Tracking** | Basic source records | Complete evidence chains |
| **Version Control** | Manual or difficult | Built-in Git integration |
| **Human Readability** | Don't even try | Clear YAML structure |
| **Validation** | Syntax only | Schema-based validation |
| **Extensibility** | Limited | JSON Schema based |
| **Data Portability** | Vendor lock-in | Open format you own |
| **Interoperability** | GEDCOM export only | Import/export + Git workflows |
| **Custom Types** | Fixed schema | Archive-defined vocabularies |

### Visual Comparison

**GEDCOM Format:**
```
0 @I1@ INDI
1 NAME John /Smith/
1 BIRT
2 DATE 15 JAN 1850
2 PLAC Leeds, Yorkshire, England
2 SOUR @S1@
3 QUAY 2
```

**GENEALOGIX Format:**
```yaml
persons:
  person-john-smith:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      born_on: "1850-01-15"

assertions:
  assertion-john-birth:
    subject: person-john-smith
    claim: born_on
    value: "1850-01-15"
    citations: [citation-birth-cert]
    confidence: high
```

> **Learn More:** See [Core Concepts](/specification/2-core-concepts) for the complete assertion model and evidence chain explanation.

### Beyond Exchange: A True Research Foundation

GENEALOGIX is more than an import/export format. It's a **permanent foundation** for your research:

#### Data You Own and Control
- **Human-readable YAML** - Edit in any text editor, no special software required
- **Open format** - No proprietary database, no vendor lock-in
- **Portable forever** - Your data outlasts any software application
- **Git-native** - Industry-standard version control built in

#### Flexible to Your Research Domain
- **Custom vocabularies** - Define your own event types, relationship types, properties
- **Beyond genealogy** - Biography projects, local history, prosopography, and more
- **Extensible schema** - Add custom fields without breaking compatibility
- **No central registry** - Each archive is autonomous and self-contained

#### Built for Collaboration
- **Git workflows** - Branch, merge, and collaborate like software developers
- **Evidence-first** - Every claim backed by documented sources
- **Conflict resolution** - Handle competing evidence systematically
- **Pull request reviews** - Quality assurance through peer review

## What is GENEALOGIX?

GENEALOGIX is an open standard for version-controlled family archives that provides:

- **📚 Evidence-First Model**: Every claim backed by documented sources
- **🔍 Quality Assessment**: Structured evaluation of evidence reliability (0-3 scale)
- **🌳 Git-Native Architecture**: Full version control and collaboration support
- **📋 Human-Readable Format**: Clear YAML files instead of binary formats
- **✅ Schema Validation**: JSON Schema-based validation and error checking
- **🔗 Complete Provenance**: Audit trail from repository to conclusion
- **🎯 Repository-Owned Vocabularies**: Define custom types within each archive

## Installation

The recommended way to install the `glx` CLI is to download the latest pre-compiled binary for your operating system from the [GitHub Releases](https://github.com/genealogix/glx/releases) page.

Alternatively, developers can install from source:

```bash
# Install the glx CLI tool
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

# Validate schema files
glx check-schemas
```

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
      born_on: "1850-01-15"

sources:
  source-12345678:
    title: "Birth Certificate"
```

**Key Points:**
- Entity IDs are map keys: `person-john-smith` or `person-a1b2c3d4`
- IDs can be descriptive or random (1-64 alphanumeric/hyphens)
- Files can contain any combination of entity types
- Parser collates all entities across all .glx files in repository
- Controlled vocabularies define valid types in `vocabularies/` directory

## Specification Status

This specification follows [Semantic Versioning](https://semver.org/).

- **Draft**: Under active development, may change significantly
- **Release Candidate**: Stable, final review before release
- **Released**: Production-ready, changes discussed via GitHub issues and discussions

## Community & Support

GENEALOGIX is an open-source project that thrives on community participation:

### 🐛 Issues & Bug Reports
- [GitHub Issues](https://github.com/genealogix/glx/issues) - Report bugs and request features

### 💬 Discussion & Q&A
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Community conversations
- [Discord Community](https://discord.gg/genealogix) - Real-time chat and support
- [Mailing List](https://groups.google.com/g/genealogix) - Email discussions

### 📚 Documentation & Learning
- [Quickstart Guide](/quickstart) - 5-minute getting started
- [Best Practices](/guides/best-practices) - Recommended workflows
- [Migration Guide](/guides/migration-from-gedcom) - Manual conversion guidance from GEDCOM
- [Glossary](/specification/6-glossary) - Key terms and concepts

### 🤝 Contributing
- [Contributing Guide](/development/contributing) - How to contribute to the project
- [Code of Conduct](/development/code-of-conduct) - Community standards

### 🎯 Getting Help

**For Users:**
1. Start with the [Quickstart Guide](/quickstart)
2. Explore [Complete Examples](/examples/complete-family/)
3. Ask questions in [GitHub Discussions](https://github.com/genealogix/glx/discussions)

**For Developers:**
1. Read [CLAUDE.md](CLAUDE.md) for project context
2. Review the [Contributing Guide](/development/contributing)
3. Check [GitHub Issues](https://github.com/genealogix/glx/issues) and [Discussions](https://github.com/genealogix/glx/discussions)
4. Follow [Best Practices](/guides/best-practices)

### 📊 Project Status

**Current Release:** v0.0.0-beta.9 (Beta)
- ✅ 9 core entity types defined
- ✅ JSON Schema validation
- ✅ CLI tool with vocabulary-based validation
- ✅ Repository-owned controlled vocabularies
- ✅ Complete test suite
- ✅ Comprehensive examples
- 🔄 Community building and feedback
- 📋 GitHub issues and discussions for major changes

### 🙏 Acknowledgments

GENEALOGIX builds on decades of genealogy research and the contributions of:
- The genealogy community for identifying core requirements
- Open-source projects like GEDCOM, Gramps, and FamilySearch
- Contributors who help improve the specification
- Researchers who provide real-world use cases

---

**Made with ❤️ for genealogists, by genealogists**

[⭐ Star us on GitHub](https://github.com/genealogix/glx) • [🐛 Report Issues](https://github.com/genealogix/glx/issues) • [💬 Join Discussions](https://github.com/genealogix/glx/discussions)

## License

Copyright 2025 Oracynth, Inc.

Licensed under the [Apache License, Version 2.0](https://github.com/genealogix/glx/blob/main/LICENSE) (the "License");
you may not use this project except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

## Repository Structure

```
genealogix/glx/
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── CHANGELOG.md
├── specification/
│   ├── README.md
│   ├── 1-introduction.md
│   ├── 2-core-concepts.md
│   ├── 3-archive-organization.md
│   ├── 4-entity-types/
│   ├── schema/                     # JSON Schemas
│   │   ├── README.md
│   │   ├── v1/
│   │   │   ├── person.schema.json
│   │   │   ├── relationship.schema.json
│   │   │   ├── event.schema.json
│   │   │   └── vocabularies/      # Vocabulary schemas
│   │   │       ├── relationship-types.schema.json
│   │   │       ├── event-types.schema.json
│   │   │       └── ...
│   │   └── meta/
├── docs/
│   ├── quickstart.md
│   ├── guides/
│   │   ├── best-practices.md
│   │   ├── glossary.md
│   │   └── migration-from-gedcom.md
│   ├── development/
│   │   ├── architecture.md
│   │   ├── setup.md
│   │   ├── testing-guide.md
│   │   └── schema-development.md
│   └── examples/                   # Example archives
│       ├── README.md
│       ├── basic-family/
│       │   ├── persons/
│       │   ├── relationships/
│       │   └── vocabularies/      # Example vocabularies
│       └── complete-family/
│           └── vocabularies/
├── glx/                            # Go CLI implementation
│   ├── main.go
│   ├── validator.go
│   ├── validate.go
│   └── tests/                      # Test fixtures
│       ├── README.md
│       ├── run-tests.sh
│       ├── valid/
│       └── invalid/
├── website/                        # VitePress documentation site
│   ├── package.json
│   └── .vitepress/
│       └── config.js
└── .devcontainer/
    └── devcontainer.json
```


