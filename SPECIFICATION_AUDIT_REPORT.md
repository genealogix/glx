# GLX Specification Audit Report

**Specification Version:** 0.0.0-beta.3
**Audit Date:** 2026-02-04
**Files Reviewed:** 17 markdown files, 16 vocabulary files, 31 schema files

---

## Executive Summary

This audit reviewed the GLX specification for internal contradictions, terminology inconsistencies, broken references, completeness issues, and ambiguous language. The specification is generally well-structured and comprehensive.

**Overall Rating:** Good

**Issues Remaining:**
- Critical: 0 (3 resolved)
- Major: 8 (4 resolved)
- Minor: 4 (4 resolved)

---

## Major Issues

### 3. `claim` vs `property` Terminology Confusion

**Priority:** High

**Location:** `specification/4-entity-types/assertion.md`

**Problem:** The `claim` field references property names, but terminology is confusing. Users must understand claims are *assertions about* properties.

**Tracked:** `todo.md` line 23

**Fix:** Add clarifying note. Consider renaming `claim` to `property` in future version.

---

### 4. VitePress Components Make Standard Vocabularies README Non-Portable

**Priority:** Medium

**Location:** `specification/5-standard-vocabularies/README.md:6-9`

**Problem:** VitePress-specific Vue components make the spec unusable outside the website context.

**Fix:** Add fallback for non-VitePress contexts or generate README from vocabulary files.

---

### 5. Redundant `description` Field in Events

**Priority:** Medium

**Locations:**
- `specification/4-entity-types/event.md:67-68` (top-level field)
- `specification/5-standard-vocabularies/event-properties.glx` (vocabulary property)

**Problem:** Events have both a top-level `description` and a `properties.description` with no guidance on when to use which.

**Fix:** Clarify distinction or consolidate.

---

### 6. Undocumented Schema Files

**Priority:** Medium

**Locations:**
- `specification/schema/v1/config/assertion-types.schema.json`
- `specification/schema/v1/config/confidence-scales.schema.json`
- `specification/schema/v1/config/relationship-types.schema.json`
- `specification/schema/v1/archive-metadata.schema.json`

**Problem:** These schema files exist but are not documented.

**Fix:** Document these schemas or remove if deprecated.

---

### 7. Ambiguous Terminology: "Archive" Used with Three Meanings

**Priority:** Medium

**Locations:** Throughout specification

**Problem:** "Archive" means: GLX data structure, physical repositories/institutions, and Git repository structure.

**Fix:** Use specific terminology: "GLX archive", "repository/institution", "Git repository".

---

### 8. Ambiguous ID Prefix Requirement

**Priority:** Medium

**Locations:**
- `specification/3-archive-organization.md:90`
- `specification/4-entity-types/README.md:153-156`
- `specification/4-entity-types/person.md:56-58`

**Problem:** Inconsistent about whether entity ID prefixes are required or recommended.

**Fix:** Clarify with RFC 2119 language (MUST/SHOULD/MAY).

---

### 9. ID Format Documentation Scattered

**Priority:** Low

**Locations:**
- `specification/3-archive-organization.md:258`
- `specification/4-entity-types/README.md:151`
- `specification/4-entity-types/README.md:160`

**Problem:** ID format information scattered across three sections with overlapping content.

**Fix:** Consolidate to single authoritative source with cross-references.

---

### 10. Ambiguous Property Fields Validation

**Priority:** Low

**Location:** `specification/4-entity-types/vocabularies.md:792-884`

**Problem:** The `fields` feature for structured properties doesn't explain validation behavior.

**Fix:** Add "Field Validation" subsection explaining optional fields and partial sets.

---

## Minor Issues

### 1. Bat/Bas Mitzvah Duplication

**Location:** `specification/5-standard-vocabularies/event-types.glx:101-111`

**Problem:** Both spellings defined as separate types.

**Tracked:** `todo.md` line 44

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

### Immediate Actions

1. **Add Terminology Clarification** (#3)
   - Add note explaining `claim` references property names

### Short-Term

3. **Consolidate ID Documentation** (#8, #9)
   - Single authoritative source
   - Clear cross-references
   - RFC 2119 language (MUST/SHOULD/MAY)

4. **Clarify Event Description Fields** (#5)
   - Document when to use top-level vs property

5. **Document or Remove Schema Files** (#6)
   - Review config/ schemas for current relevance

### Medium-Term

6. **Improve Entity Documentation Consistency** (Minor #2, #4)
   - Add property tables to Place, Repository, Relationship
   - Add common fields note to each entity page

7. **Address VitePress Dependency** (#4)
   - Make Standard Vocabularies README portable

8. **Document Validation Behavior** (#10)
   - Field validation subsection

9. **Address Terminology Consistency** (#7)
   - Audit "archive" usage throughout spec
   - Use specific terms: "GLX archive", "repository/institution"

### Process Improvements

10. **Add Link Validation to CI**
    - Check internal links exist
    - Validate anchors match headers

---

## Items Tracked in todo.md

| Line | Item | Audit Issue |
|------|------|-------------|
| 23 | Rename `claim` to `property` | #3 |
| 43 | Bar/Bat Mitzvah consolidation | Minor #1 |
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

## Completed Issues

The following issues were resolved during this audit:

### CRIT-1. Broken `#evidence-hierarchy` Links ✅
Updated 4 files to use correct `#evidence-chain` anchor.

### CRIT-2. Wrong Property Name `birth_date` ✅
Changed to `born_on` in 2-core-concepts.md and vocabularies.md.

### CRIT-3. Glossary Not Part of Specification ✅
Moved to `specification/6-glossary.md` with full integration.

### MAJ-5. Inconsistent `.md` Extension in Internal Links ✅
Removed `.md` from ~40 internal links across 12 specification files.

### MAJ-10. Misleading Vocabulary Directory Structure Example ✅
Removed `property vocabularies/` subdirectory from 2-core-concepts.md tree example.

### MIN-3. Inconsistent GEDCOM Mapping Table Headers ✅
Standardized all 8 entity type files to use "GLX Field | GEDCOM Tag | Notes".

### MIN-4. Person `name` Should Be Documented as Recommended ✅
Added note that `name` is recommended for most records.

### MIN-6. Schema README Lists Only 3 Example URLs ✅
Added clarifying text indicating these are examples and more schemas exist.

### MIN-7. Terminology: "Event/Fact" vs "Event" ✅
Changed to just "Event" in entity types README for consistency.

### MAJ-1. Adoption Semantics: Three Overlapping Definitions ✅
Removed redundant `adoption` relationship type from vocabulary. The `adoption` event type (for the legal proceeding) and `adoptive-parent-child` relationship type (for the ongoing parent-child connection) now have clear, distinct purposes. Updated relationship.md with a comprehensive example showing how to use the adoption event as the `start_event` for an adoptive-parent-child relationship.

### MAJ-2. Godparent Defined in Both Roles and Relationship Types ✅
Documented the intentional distinction: participant role `godparent` is for event participation (e.g., baptism sponsor), while relationship type `godparent` represents the ongoing godparent-godchild bond. Added `godchild` participant role. Updated relationship.md with comprehensive example showing both usages.

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Specification files reviewed | 17 |
| Vocabulary files reviewed | 16 |
| Schema files checked | 31 |
| Critical issues resolved | 3 |
| Major issues resolved | 4 |
| Minor issues resolved | 4 |
| **Major issues remaining** | **8** |
| **Minor issues remaining** | **4** |

---

*Report generated by GLX Specification Audit*
*Audit Date: 2026-02-04*
*Last Updated: 2026-02-04*
