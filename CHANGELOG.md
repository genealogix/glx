---
title: Changelog
description: Version history and notable changes to the GENEALOGIX specification
layout: doc
---

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.0-beta.0] - 2025-11-14

### Added

#### Specification & Standards
- Complete GENEALOGIX specification defining modern, evidence-first genealogy data standard
- 9 core entity types with full JSON Schema definitions:
  - Person (individuals with biographical properties)
  - Relationship (family connections with types and dates)
  - Event (life events with sources and locations)
  - Assertion (evidence-backed claims with quality assessment)
  - Citation (evidence references with source quotations)
  - Source (primary/secondary evidence documentation)
  - Repository (physical storage information)
  - Place (geographic locations with coordinate data)
  - Participant (individuals involved in events)
- Repository-owned controlled vocabularies for extensibility
- Git-native architecture for version control and collaboration
- YAML-based human-readable format with schema validation

#### CLI Tool (`glx`)
- `glx init`: Initialize new GLX repositories with optional single-file mode
- `glx validate`: Comprehensive validation with:
  - Schema compliance checking against JSON Schemas
  - Cross-reference integrity verification across all files
  - Vocabulary constraint validation
  - Detailed error reporting with file/line locations
- `glx check-schemas`: Utility for verifying schema metadata and structure
- Support for both directory-based and single-file archives
- Cross-file entity resolution and validation

#### Documentation & Examples
- Comprehensive specification documentation (6 core documents)
- Complete examples demonstrating various use cases:
  - Minimal single-file archive
  - Basic family structure with multiple generations
  - Complete family with all entity types
  - Participant assertions workflow
  - Temporal properties and date ranges
- Development guides covering:
  - Architecture and design decisions
  - Schema development practices
  - Testing framework and test suite structure
  - Local development environment setup
- User guides including:
  - Quick-start guide for new users
  - Best practices and recommendations
  - Common pitfalls and troubleshooting
  - Manual migration guide for converting from GEDCOM format
  - Glossary of key terms and concepts

#### Testing & Quality Assurance
- Comprehensive test suite with:
  - Valid example fixtures demonstrating correct usage
  - Invalid example fixtures testing error handling
  - Cross-reference validation tests
  - Vocabulary constraint tests
  - Schema compliance validation tests
- Automated CI/CD pipeline using GitHub Actions
- Full code coverage reporting

#### Project Infrastructure
- Apache 2.0 open-source license
- Community guidelines and code of conduct
- Contributing guidelines for developers
- GitHub issue and discussion templates
- Development container configuration for consistent environments
- Pre-configured VitePress documentation site


