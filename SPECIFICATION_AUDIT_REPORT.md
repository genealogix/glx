# GLX Specification Audit Report

**Specification Version:** 0.0.0-beta.3
**Audit Date:** 2026-02-04
**Latest Audit:** 2026-02-05
**Files Reviewed:** 17 markdown files, 16 vocabulary files, 27 schema files

---

## Executive Summary

This audit reviewed the GLX specification for internal contradictions, terminology inconsistencies, broken references, completeness issues, and ambiguous language. The specification is generally well-structured and comprehensive.

**Overall Rating:** Good

**Issues Remaining:**
- Critical: 0
- Major: 1
- Minor: 1

---

## Major Issues

### 1. VitePress Components Make Standard Vocabularies README Non-Portable

**Priority:** Medium

**Location:** `specification/5-standard-vocabularies/README.md:6-9`

**Problem:** VitePress-specific Vue components (`<script setup>`, `<YamlFile>`) make the specification README unusable outside the website context. Viewing the raw markdown shows import statements and component tags instead of actual vocabulary content.

**Impact:** Developers or researchers wanting to understand vocabularies without the VitePress site cannot see the content. GitHub preview renders the Vue code as plain text.

**Fix:** One of:
- Generate a static fallback README from vocabulary files
- Add `::: details` collapsible sections with raw vocabulary content
- Accept that this file is website-only and document this limitation

---

## Minor Issues

### 1. Bat/Bas Mitzvah Duplication

**Location:** `specification/5-standard-vocabularies/event-types.glx:101-111`

**Problem:** Both `bat_mitzvah` and `bas_mitzvah` are defined as separate event types, representing the same ceremony with alternate spellings.

**Tracked:** `todo.md` line 40

**Fix:** Either:
- Consolidate into one entry with both spellings noted in description
- Add a note explaining these are alternate spellings for cultural/regional preferences

---

## Resolved Issues (Since Last Audit)

The following issues from the previous audit have been resolved:

- ✅ **Alternative Names Field Removal** - The `alternative_names` field was removed from Place entity documentation per specification simplification
- ✅ **Entity ID Prefix Clarification** - Documentation now states prefixes are optional, not required
- ✅ **Tags Field Removal** - The `tags` field was removed from entity specifications
- ✅ **Common Fields Anchor Reference** - Removed broken link in `vocabularies.md:671`, simplified text to describe `notes` as a standard entity field
- ✅ **Glossary Schema Reference Link** - Fixed link in `6-glossary.md:242` to point to correct schema directory

---

## Recommendations

### Quick Wins (Low Effort)

1. **Document Bat/Bas Mitzvah distinction** (Minor #1) - Add note in vocabulary file explaining these are cultural/regional spelling variants of the same ceremony

### Deferred (Architectural Decisions Needed)

2. **VitePress portability** (Major #1) - Decide: generate static README from vocab files, or accept website-only rendering

### Process Improvements

3. **Add link validation to CI** - Automated checking of internal links and anchors

---

## Items Tracked in todo.md

The following items are already tracked in `todo.md` and excluded from this audit's issue counts:

| Line | Item | Related Finding |
|------|------|-----------------|
| 22 | Review standard vocabularies | Related to vocabulary audit |
| 23 | Add validation rule sections | Documentation enhancement |
| 39 | GEDCOM tag mapping in vocabularies | Enhancement request |
| 40 | Bar/Bat Mitzvah consolidation | Minor #1 (tracked) |
| 41 | Gender/sex controlled vocabularies | Enhancement request |
| 42 | Sex as temporal property | Enhancement request |

---

## Positive Findings

- **Well-Structured Documentation** - Logical organization from intro → core concepts → entity types → vocabularies
- **Comprehensive Vocabulary System** - 16 vocabulary files all present and properly formatted with consistent structure
- **Consistent YAML Examples** - Examples throughout the specification are valid YAML and follow documented structure
- **Clear Assertion Model** - The separation of properties from assertions is clearly explained with good examples
- **Thorough Date Format Documentation** - ISO 8601 + FamilySearch hybrid format is well-documented in Core Concepts
- **Complete Schema Coverage** - All 27 JSON schema files are present for entities and vocabularies
- **Excellent Cross-References** - Entity type documents consistently link to related entities and vocabulary documentation
- **Properties/Vocabulary Integration** - Property vocabularies are well-documented with clear examples of temporal and multi-value properties

---

## Audit Methodology

### Files Reviewed

**Specification Documents (17):**
- `README.md`, `1-introduction.md`, `2-core-concepts.md`, `3-archive-organization.md`, `6-glossary.md`
- `4-entity-types/`: README.md, person.md, event.md, relationship.md, place.md, assertion.md, citation.md, source.md, repository.md, media.md, vocabularies.md
- `5-standard-vocabularies/README.md`

**Vocabulary Files (16):**
- Type vocabularies: event-types.glx, relationship-types.glx, place-types.glx, source-types.glx, media-types.glx, repository-types.glx, confidence-levels.glx, participant-roles.glx
- Property vocabularies: person-properties.glx, event-properties.glx, relationship-properties.glx, place-properties.glx, media-properties.glx, repository-properties.glx, source-properties.glx, citation-properties.glx

**Schema Files (27):**
- Entity schemas (9): person, event, relationship, place, assertion, citation, source, repository, media, glx-file
- Vocabulary schemas (16): All type and property vocabulary schemas
- Meta schema (1): schema.schema.json

### Checks Performed

1. ✅ Internal contradictions - No contradictions found
2. ✅ Terminology consistency - Terms used consistently throughout
3. ✅ Broken references - All anchor references now valid
4. ✅ Completeness - All entity types fully documented
5. ✅ Example validation - Examples syntactically correct and consistent with prose
6. ✅ Vocabulary coverage - All 16 vocabulary files present and complete
7. ⚠️ Link validation - Some links rely on VitePress rendering (Major #1)

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Specification files reviewed | 17 |
| Vocabulary files reviewed | 16 |
| Schema files checked | 27 |
| **Major issues remaining** | **1** |
| **Minor issues remaining** | **1** |

---

*Report generated by GLX Specification Audit*
*Initial Audit: 2026-02-04*
*Latest Audit: 2026-02-05*
