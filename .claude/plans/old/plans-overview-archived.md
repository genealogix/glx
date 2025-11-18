# GLX Implementation Plans

This directory contains planning and design documents for GLX features. All documents are organized by feature area.

---

## Active Plans

### 🚧 Currently Being Worked On

**[glx-serializer-implementation-steps.md](glx-serializer-implementation-steps.md)**
- **Status**: IN PROGRESS - Task 1.1 (ID Generator)
- **Purpose**: Step-by-step implementation guide for GLX serializer
- **Description**: Detailed breakdown of 13 tasks across 5 phases for implementing single-file and multi-file YAML serialization, including vocabulary embedding and ID generation
- **Estimated**: 6-10 hours total
- **Dependencies**: glx-serializer-plan.md, glx-vocabulary-embedding.md, glx-id-generation.md

---

## Serializer Feature (v0.3.0-beta)

### Architecture & Design

**[glx-serializer-plan.md](glx-serializer-plan.md)**
- **Status**: COMPLETE - Architecture approved
- **Purpose**: Complete architectural design for GLX serialization
- **Description**: Defines single-file and multi-file serialization formats, API design, CLI integration, architectural decisions, and testing strategy
- **Key Decisions**:
  - Embed vocabularies with go:embed
  - Random IDs for entity filenames (person-a3f8d2c1.glx)
  - Sequential writes (no parallelization initially)
  - Default validation with --no-validate flag option

**[glx-vocabulary-embedding.md](glx-vocabulary-embedding.md)**
- **Status**: COMPLETE - Design ready for implementation
- **Purpose**: Strategy for embedding standard vocabularies in binary
- **Description**: Detailed plan using go:embed with embed.FS approach, WriteStandardVocabularies() implementation, and testing strategy
- **Implementation**: lib/vocabularies.go (to be created)

**[glx-id-generation.md](glx-id-generation.md)**
- **Status**: COMPLETE - Design ready for implementation
- **Purpose**: ID generation strategy for multi-file archive filenames
- **Description**: Random 8-character hex IDs using crypto/rand, EntityWithID wrapper for preserving entity IDs, collision detection strategy
- **Format**: `{entity-type}-{random-id}.glx` (e.g., person-a3f8d2c1.glx)
- **Implementation**: lib/id_generator.go (to be created)

---

## GEDCOM Import Feature (v0.2.0-beta)

### Planning & Analysis

**[gedcom-import-complete-plan.md](gedcom-import-complete-plan.md)**
- **Status**: REFERENCE - Original comprehensive plan
- **Purpose**: Complete GEDCOM import implementation plan
- **Description**: 9,065-line comprehensive plan covering GEDCOM 5.5.1 and 7.0 import with entity mapping, evidence chains, two-pass conversion, and all GEDCOM tags
- **Note**: This is the original plan used to implement GEDCOM import

**[gedcom-import-status.md](gedcom-import-status.md)**
- **Status**: COMPLETE - Final status before serializer work
- **Purpose**: Track GEDCOM import implementation progress
- **Description**: Implementation status showing GEDCOM import complete and all tests passing
- **Last Updated**: 2025-11-18

**[gedcom-import-gap-analysis.md](gedcom-import-gap-analysis.md)**
- **Status**: COMPLETE - Analysis finished, recommendations implemented
- **Purpose**: Comprehensive gap analysis of GEDCOM import vs. plan
- **Description**: 715-line analysis comparing implementation against 9,065-line plan. Found 100% critical coverage, 94% high-priority coverage
- **Key Findings**:
  - PRODUCTION-READY status
  - Top 3 recommendations implemented (coordinates, NOTE, EXID)
  - Missing tags documented
- **Recommendations Implemented**:
  1. ✅ Place coordinates (MAP/LATI/LONG)
  2. ✅ GEDCOM 5.5.1 NOTE records
  3. ✅ EXID (External IDs) for GEDCOM 7.0

### Schema Changes

**[gedcom-schema-additions.md](gedcom-schema-additions.md)**
- **Status**: COMPLETE - Schema changes implemented
- **Purpose**: Document required GLX schema additions for GEDCOM support
- **Description**: Analysis of missing Properties fields on 5 entity types (Source, Citation, Repository, Media, Assertion)
- **Changes Made**: Added `Properties map[string]interface{}` to all 5 types
- **Impact**: Backward compatible (optional fields)
- **Future**: Property vocabularies planned for GEDCOM-specific fields

---

## Document Status Legend

- 🚧 **IN PROGRESS**: Currently being implemented
- ✅ **COMPLETE**: Finished and merged
- 📋 **REFERENCE**: Historical reference document
- ⏳ **PLANNED**: Approved but not started

---

## Implementation Workflow

1. **Planning Phase** → Create architecture and design documents
2. **Review Phase** → Get architectural decisions approved
3. **Implementation Phase** → Follow step-by-step guides (e.g., glx-serializer-implementation-steps.md)
4. **Testing Phase** → Unit tests, integration tests, E2E tests
5. **Documentation Phase** → Update changelog, spec, docs, website

---

## Next Steps

### Immediate (Current Sprint)
1. Implement ID generator (lib/id_generator.go) - Task 1.1
2. Implement vocabulary embedder (lib/vocabularies.go) - Task 1.2
3. Implement serializer interface and types - Task 1.3
4. Implement single-file YAML serializer - Task 2.1
5. Implement multi-file serializer - Task 2.2

### Near Future
- Property vocabularies for GEDCOM fields (source-properties.glx, etc.)
- CLI commands: split, join
- Documentation updates
- Version bump to v0.3.0-beta

---

## File Maintenance

**When to archive a plan:**
- Plan is fully implemented and tested
- Plan has been superseded by a newer version
- Plan is no longer relevant to current roadmap

**When to update a plan:**
- Architectural decisions change
- New requirements discovered
- Implementation reveals issues with design

**When to create a new plan:**
- Starting a new major feature
- Significant architectural refactor
- Complex bug fix requiring design work

---

Last Updated: 2025-11-18
