# GENEALOGIX Specification

The official specification for the GENEALOGIX (GLX) family archive format.

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

## What is GENEALOGIX?

GENEALOGIX is an open standard for version-controlled family archives.
It uses Git-native architecture with human-readable YAML files to ensure
your family history data is portable, transparent, and future-proof.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to propose changes.

Major changes require an RFC (Request for Comments) in the [rfcs/](rfcs/) directory.

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


