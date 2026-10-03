# GENEALOGIX Specification

[![Version](https://img.shields.io/github/v/release/genealogix/glx?include_prereleases&label=version)](https://github.com/genealogix/glx/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/genealogix/glx.svg)](https://pkg.go.dev/github.com/genealogix/glx)
[![Go Version](https://img.shields.io/github/go-mod/go-version/genealogix/glx)](https://github.com/genealogix/glx/blob/main/go.mod)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](https://github.com/genealogix/glx/blob/main/LICENSE)
[![CI](https://github.com/genealogix/glx/actions/workflows/validate-spec.yml/badge.svg?branch=main)](https://github.com/genealogix/glx/actions/workflows/validate-spec.yml)
[![codecov](https://codecov.io/gh/genealogix/glx/branch/main/graph/badge.svg)](https://codecov.io/gh/genealogix/glx)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/genealogix/glx/badge)](https://scorecard.dev/viewer/?uri=github.com/genealogix/glx)
[![Contributors](https://img.shields.io/github/contributors/genealogix/glx.svg)](https://github.com/genealogix/glx/graphs/contributors)

The official specification for **GENEALOGIX (GLX)** — a portable, extensible archive format for genealogical research and beyond. Built on Git, designed for collaboration, and customizable through archive-owned vocabularies. Your data, your way, forever.

## Installation

Download the latest pre-compiled binary for your operating system from the [GitHub Releases](https://github.com/genealogix/glx/releases) page.

Developers can install from source:

```bash
# Pin the release tag for a reproducible install; bump it when a new release ships
go install github.com/genealogix/glx/glx@v0.0.0-beta.12

# Or track main for changes not yet in a release. `glx --version` then
# reports the commit it was built from (a pseudo-version ending in its hash).
go install github.com/genealogix/glx/glx@main
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

For a side-by-side look at the GEDCOM-vs-GLX wire formats and the assertion model that backs every claim with evidence, see [Core Concepts](specification/2-core-concepts.md).

## Features

- **📚 Evidence-First Model** — every claim backed by documented sources
- **🔍 Quality Assessment** — structured confidence on every assertion, with the nature of the source and of the information recorded separately
- **🌳 Git-Native Architecture** — full version control and collaboration support
- **📋 Human-Readable Format** — clear YAML files instead of binary formats
- **✅ Schema Validation** — JSON Schema-based validation and error checking
- **🔗 Complete Provenance** — audit trail from repository to conclusion
- **🎯 Repository-Owned Vocabularies** — define custom types within each archive

## CLI Commands

The `glx` CLI groups its commands into archive management, import/export, exploration, data entry, analysis, and shell completion. Run `glx --help` for the list your build has, `glx <command> --help` for a command's flags, or see the [CLI reference](docs/cli/index.md) for per-command pages with examples.

### Archive Management

- [`glx init`](docs/cli/glx_init.md) — initialize a new archive
- [`glx validate`](docs/cli/glx_validate.md) — validate files and cross-references
- [`glx split`](docs/cli/glx_split.md) — convert a single-file archive to multi-file
- [`glx join`](docs/cli/glx_join.md) — convert a multi-file archive to single-file
- [`glx merge`](docs/cli/glx_merge.md) — combine two archives with duplicate detection
- [`glx merge-persons`](docs/cli/glx_merge-persons.md) — merge two person records into one
- [`glx migrate`](docs/cli/glx_migrate.md) — migrate an archive to the current format
- [`glx rename`](docs/cli/glx_rename.md) — rename an entity by ID
- [`glx cache`](docs/cli/glx_cache.md) — build, inspect, or clear the binary archive cache

### Import & Export

- [`glx import`](docs/cli/glx_import.md) — import a GEDCOM file
- [`glx export`](docs/cli/glx_export.md) — export to GEDCOM or Schema.org-aligned JSON-LD
- [`glx publish`](docs/cli/glx_publish.md) — generate a self-contained static HTML site from an archive

### Exploration

- [`glx search`](docs/cli/glx_search.md) — full-text search across entities
- [`glx query`](docs/cli/glx_query.md) — filter and list entities
- [`glx vitals`](docs/cli/glx_vitals.md) — show birth, death, burial for a person
- [`glx timeline`](docs/cli/glx_timeline.md) — chronological events for a person
- [`glx summary`](docs/cli/glx_summary.md) — full person profile with narrative
- [`glx ancestors`](docs/cli/glx_ancestors.md) — ancestor tree
- [`glx descendants`](docs/cli/glx_descendants.md) — descendant tree
- [`glx cite`](docs/cli/glx_cite.md) — formatted citation text
- [`glx path`](docs/cli/glx_path.md) — shortest relationship path between two people
- [`glx serve`](docs/cli/glx_serve.md) — local browser-based read-only archive viewer

### Data Entry

- [`glx census`](docs/cli/glx_census.md) — census tooling (see subcommands)
- [`glx census add`](docs/cli/glx_census_add.md) — generate entities from a census template
- [`glx add`](docs/cli/glx_add.md) — create entities (person, place, event, source, citation, repository, relationship, assertion, and more) from flags; `glx add --help` lists every entity type
- [`glx link`](docs/cli/glx_link.md) — create a FamilySearch citation (and source/repository scaffolding) from an ARK URL

### Analysis

- [`glx stats`](docs/cli/glx_stats.md) — entity-count and confidence dashboard
- [`glx places`](docs/cli/glx_places.md) — place data quality issues
- [`glx cluster`](docs/cli/glx_cluster.md) — FAN-club analysis
- [`glx analyze`](docs/cli/glx_analyze.md) — gap, conflict, and suggestion analysis
- [`glx duplicates`](docs/cli/glx_duplicates.md) — detect duplicate entities
- [`glx coverage`](docs/cli/glx_coverage.md) — research coverage report
- [`glx diff`](docs/cli/glx_diff.md) — diff two archives
- [`glx evidence`](docs/cli/glx_evidence.md) — lay out every assertion for one subject+property side-by-side
- [`glx proof`](docs/cli/glx_proof.md) — compile a Genealogical Proof Standard summary for a research question
- [`glx migrations`](docs/cli/glx_migrations.md) — trace a person's geographic movement over time

### Shell completion

- [`glx completion`](docs/cli/glx_completion.md) — generate shell completion scripts (bash, zsh, fish, powershell)

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
- Parser collates all entities across all .glx files in repository, except under dot-prefixed directories (`.git`, `.glx`, editor scratch dirs), which are not archive content
- Controlled vocabularies define valid types in `vocabularies/` directory

## Documentation

- [🚀 Quickstart](docs/quickstart.md) — 5-minute getting started
- [💡 Examples](docs/examples/README.md) — runnable sample archives
- [🛠 CLI Reference](https://genealogix.io/cli/commands) — every command and flag
- [📐 Best Practices](docs/guides/best-practices.md) — recommended workflows
- [🔁 Migration from GEDCOM](docs/guides/migration-from-gedcom.md) — manual conversion guidance
- [🔀 GLX-aware Git merge driver](docs/merge-driver.md) — genealogy-aware conflict resolution for .glx files
- [📖 Specification](specification/README.md) — full spec
- [📋 JSON Schemas](specification/schema/README.md) — machine-readable schemas
- [📚 Glossary](specification/6-glossary.md) — key terms and concepts
- [🧱 Dev Container](https://github.com/genealogix/glx/tree/main/.devcontainer) — preconfigured dev environment

## Specification Status

This specification follows [Semantic Versioning](https://semver.org/). Current release: **v0.0.0-beta.12** (Beta).

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
| **Contributing** | [CONTRIBUTING.md](CONTRIBUTING.md) |
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
