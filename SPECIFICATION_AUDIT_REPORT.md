# GLX Specification Audit Report

**Specification Version:** 0.0.0-beta.3
**Audit Date:** 2026-02-04
**Files Reviewed:** 17 markdown files, 16 vocabulary files, 31 schema files

---

## Executive Summary

This audit reviewed the GLX specification for internal contradictions, terminology inconsistencies, broken references, completeness issues, and ambiguous language. The specification is generally well-structured and comprehensive.

**Overall Rating:** Good

**Issues Remaining:**
- Critical: 0
- Major: 6
- Minor: 4

---

## Major Issues

### 1. VitePress Components Make Standard Vocabularies README Non-Portable

**Priority:** Medium

**Location:** `specification/5-standard-vocabularies/README.md:6-9`

**Problem:** VitePress-specific Vue components make the spec unusable outside the website context.

**Fix:** Add fallback for non-VitePress contexts or generate README from vocabulary files.

---

### 3. Ambiguous Terminology: "Archive" Used with Three Meanings

**Priority:** Medium

**Locations:** Throughout specification

**Problem:** "Archive" means: GLX data structure, physical repositories/institutions, and Git repository structure.

**Fix:** Use specific terminology: "GLX archive", "repository/institution", "Git repository".

---

### 4. Ambiguous ID Prefix Requirement

**Priority:** Medium

**Locations:**
- `specification/3-archive-organization.md:90`
- `specification/4-entity-types/README.md:153-156`
- `specification/4-entity-types/person.md:56-58`

**Problem:** Inconsistent about whether entity ID prefixes are required or recommended.

**Fix:** Clarify with RFC 2119 language (MUST/SHOULD/MAY).

---

### 5. ID Format Documentation Scattered

**Priority:** Low

**Locations:**
- `specification/3-archive-organization.md:258`
- `specification/4-entity-types/README.md:151`
- `specification/4-entity-types/README.md:160`

**Problem:** ID format information scattered across three sections with overlapping content.

**Fix:** Consolidate to single authoritative source with cross-references.

---

### 6. Ambiguous Property Fields Validation

**Priority:** Low

**Location:** `specification/4-entity-types/vocabularies.md:792-884`

**Problem:** The `fields` feature for structured properties doesn't explain validation behavior.

**Fix:** Add "Field Validation" subsection explaining optional fields and partial sets.

---

## Minor Issues

### 1. Bat/Bas Mitzvah Duplication

**Location:** `specification/5-standard-vocabularies/event-types.glx:101-111`

**Problem:** Both spellings defined as separate types.

**Tracked:** `todo.md` line 42

**Fix:** Consolidate or document distinction.

---

### 2. Inconsistent Property Documentation Depth

**Problem:** Place, Repository, and Relationship entity docs missing standard property tables.

**Fix:** Add inline property tables to these entity docs.

---

### 3. Missing `multi_value` Usage Documentation

**Location:** `specification/4-entity-types/vocabularies.md`

**Problem:** Complete usage examples missing.

**Tracked:** `todo.md` line 18

**Fix:** Add "Multi-Value Properties" subsection with examples.

---

### 4. Missing `notes` Field Context on Entity Pages

**Location:** `specification/4-entity-types/README.md:230-236`

**Problem:** Entity specs use `notes` without explaining it's available on all entities.

**Fix:** Add common fields note to each entity page.

---

## Recommendations

### Quick Wins (Low Effort)

These can each be completed in a single focused session:

1. **Document Bat/Bas Mitzvah distinction** (Minor #1) - Add note explaining these are alternate spellings of the same ceremony
2. **Add `multi_value` usage examples** (Minor #3) - Add examples subsection to vocabularies.md
3. **Clarify ID prefix requirement** (#4) - Add RFC 2119 language (SHOULD) to one authoritative location

### Medium Effort

These require more coordination but are well-defined:

4. **Consolidate ID documentation** (#5) - Move scattered ID format info to single source with cross-references
5. **Add property tables to entity docs** (Minor #2) - Add inline tables to Place, Repository, Relationship pages
6. **Add common fields note** (Minor #4) - Document that `notes` is available on all entities
7. **Document field validation behavior** (#6) - Add subsection explaining optional fields and partial sets

### Deferred (Architectural Decisions Needed)

These require broader discussion or significant refactoring:

8. **VitePress portability** (#1) - Decide: generate static README from vocab files, or accept website-only rendering
10. **Standardize "archive" terminology** (#3) - Spec-wide audit; decide on "GLX archive" vs "GLX project" vs other terms

### Process Improvements

11. **Add link validation to CI** - Automated checking of internal links and anchors

---

## Items Tracked in todo.md

| Line | Item | Audit Issue |
|------|------|-------------|
| 42 | Bar/Bat Mitzvah consolidation | Minor #1 |
| 18 | Add multi_value examples | Minor #3 |

---

## Positive Findings

- **Well-Structured Documentation** - Logical organization from intro to core concepts to entity types
- **Comprehensive Vocabulary System** - Archive-owned vocabularies well-documented
- **Consistent YAML Examples** - Most examples valid and follow documented structure
- **Clear Assertion Model** - Separation of properties from assertions clearly explained
- **Thorough Date Format Documentation** - ISO 8601 + FamilySearch hybrid well-documented
- **Complete Vocabulary Files** - All 16 files present and well-formatted

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Specification files reviewed | 17 |
| Vocabulary files reviewed | 16 |
| Schema files checked | 31 |
| **Major issues remaining** | **6** |
| **Minor issues remaining** | **4** |

---

*Report generated by GLX Specification Audit*
*Audit Date: 2026-02-04*
*Last Updated: 2026-02-04*
