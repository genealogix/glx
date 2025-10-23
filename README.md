# GENEALOGIX Specification

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/genealogix/spec/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![CI](https://github.com/genealogix/spec/workflows/Validate%20Specification/badge.svg)](https://github.com/genealogix/spec/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/genealogix/spec)](https://goreportcard.com/report/github.com/genealogix/spec)
[![Contributors](https://img.shields.io/github/contributors/genealogix/spec.svg)](https://github.com/genealogix/spec/graphs/contributors)

The official specification for the **GENEALOGIX (GLX)** family archive format - a modern, evidence-first, Git-native genealogy data standard.

## Quick Links

- [📖 Read the Specification](specification/)
- [📋 JSON Schemas](schema/)
- [💡 Examples](examples/)
- [🧪 Test Suite](test-suite/)
- [🛠 CLI](glx/)
- [🧱 Dev Container](.devcontainer/)

## Current Version

**Version:** 1.0.0  
**Status:** Draft  
**Last Updated:** 2025-10-15

## Why GENEALOGIX?

Traditional genealogy formats like GEDCOM have served researchers well, but they have limitations in the modern collaborative research environment. GENEALOGIX addresses these challenges with a fresh approach.

### The Problem with Traditional Formats

| Challenge | GEDCOM | GENEALOGIX |
|-----------|--------|------------|
| **Collaboration** | File sharing only | Git-native workflows |
| **Evidence Tracking** | Basic source records | Complete evidence chains |
| **Version Control** | Manual or difficult | Built-in Git integration |
| **Human Readability** | Binary-like format | Clear YAML structure |
| **Validation** | Syntax only | Schema-based validation |
| **Extensibility** | Limited | JSON Schema based |

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
# Evidence chain
sources/source-birth-cert.glx:
  title: Birth Certificate
  type: vital_record

citations/citation-birth.glx:
  source: source-birth-cert
  locator: "Certificate 1850-LEEDS-00145"
  quality: 3
  transcription: "John Smith, born January 15, 1850"

# Evidence-based conclusion
assertions/assertion-john-birth.glx:
  subject: person-john-smith
  claim: born_on
  value: "1850-01-15"
  citations: [citation-birth]
  confidence: high
```

## What is GENEALOGIX?

GENEALOGIX is an open standard for version-controlled family archives that provides:

- **📚 Evidence-First Model**: Every claim backed by documented sources
- **🔍 Quality Assessment**: Structured evaluation of evidence reliability (0-3 scale)
- **🌳 Git-Native Architecture**: Full version control and collaboration support
- **📋 Human-Readable Format**: Clear YAML files instead of binary formats
- **✅ Schema Validation**: JSON Schema-based validation and error checking
- **🔗 Complete Provenance**: Audit trail from repository to conclusion

## Quick Start

```bash
# Install the glx CLI tool
go install github.com/genealogix/spec/glx@latest

# Create a new genealogix repository
glx init

# Validate .glx files
glx validate

# Validate schema files
glx check-schemas
```

## Specification Status

This specification follows [Semantic Versioning](https://semver.org/).

- **Draft**: Under active development, may change significantly
- **Release Candidate**: Stable, final review before release
- **Released**: Production-ready, changes follow RFC process

## Community & Support

GENEALOGIX is an open-source project that thrives on community participation:

### 🐛 Issues & Bug Reports
- [GitHub Issues](https://github.com/genealogix/spec/issues) - Report bugs and request features
- [Security Issues](SECURITY.md) - Responsible disclosure for security vulnerabilities

### 💬 Discussion & Q&A
- [GitHub Discussions](https://github.com/genealogix/spec/discussions) - Community conversations
- [Discord Community](https://discord.gg/genealogix) - Real-time chat and support
- [Mailing List](https://groups.google.com/g/genealogix) - Email discussions

### 📚 Documentation & Learning
- [Quickstart Guide](docs/quickstart.md) - 5-minute getting started
- [Best Practices](docs/guides/best-practices.md) - Recommended workflows
- [Migration Guide](docs/guides/migration-from-gedcom.md) - Convert from GEDCOM
- [Glossary](docs/guides/glossary.md) - Key terms and concepts

### 🤝 Contributing
- [Contributing Guide](CONTRIBUTING.md) - How to contribute to the project
- [Code of Conduct](CODE_OF_CONDUCT.md) - Community standards
- [RFC Process](rfcs/) - Propose major changes through RFCs
- [Development Setup](docs/development/setup.md) - Set up development environment

### 🎯 Getting Help

**For Users:**
1. Start with the [Quickstart Guide](docs/quickstart.md)
2. Explore [Complete Examples](examples/complete-family/)
3. Check [Common Pitfalls](docs/guides/common-pitfalls.md)
4. Ask questions in [GitHub Discussions](https://github.com/genealogix/spec/discussions)

**For Developers:**
1. Read the [Architecture Guide](docs/development/architecture.md)
2. Set up [Development Environment](docs/development/setup.md)
3. Review [Testing Framework](docs/development/testing-guide.md)
4. Join the [Discord Community](https://discord.gg/genealogix)

**For Contributors:**
1. Review [Contributing Guidelines](CONTRIBUTING.md)
2. Understand the [RFC Process](rfcs/)
3. Check [Schema Development](docs/development/schema-development.md)
4. Follow [Best Practices](docs/guides/best-practices.md)

### 📊 Project Status

**Current Release:** v1.0.0 (Draft)
- ✅ 9 core entity types defined
- ✅ JSON Schema validation
- ✅ CLI tool with basic validation
- ✅ Complete test suite
- ✅ Comprehensive examples
- 🔄 Community building and feedback
- 📋 RFC process for major changes

**Roadmap:**
- v1.1: Enhanced validation and performance
- v1.2: Advanced relationship types
- v2.0: Breaking changes with migration tools

### 🙏 Acknowledgments

GENEALOGIX builds on decades of genealogy research and the contributions of:
- The genealogy community for identifying core requirements
- Open-source projects like GEDCOM, Gramps, and FamilySearch
- Contributors who help improve the specification
- Researchers who provide real-world use cases

---

**Made with ❤️ for genealogists, by genealogists**

[⭐ Star us on GitHub](https://github.com/genealogix/spec) • [🐛 Report Issues](https://github.com/genealogix/spec/issues) • [💬 Join Discussions](https://github.com/genealogix/spec/discussions)

## License

Copyright 2025 Oracynth, Inc.

Licensed under the [Apache License, Version 2.0](LICENSE) (the "License");
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
genealogix/spec/
├── README.md
├── LICENSE
├── CONTRIBUTING.md
├── CHANGELOG.md
├── specification/
│   ├── README.md
│   ├── 1-introduction.md
│   ├── 2-core-concepts.md
│   ├── 3-file-structure.md
│   ├── 4-entity-types/
│   ├── 5-data-model/
│   ├── 6-extensibility/
│   ├── 7-git-integration/
│   └── 8-interoperability/
├── schema/
│   ├── README.md
│   ├── v1/
│   └── meta/
├── examples/
│   ├── README.md
│   └── minimal/
├── test-suite/
│   ├── README.md
│   ├── run-tests.sh
│   ├── valid/
│   └── invalid/
├── rfcs/
│   ├── README.md
│   ├── 0000-template.md
│   ├── 0001-initial-spec.md
│   └── 0002-custom-relationship-types.md
├── glx/
│   └── main.go
└── .devcontainer/
    └── devcontainer.json
```


