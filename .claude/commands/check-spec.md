---
description: Review the GLX specification for issues, contradictions, and ambiguities
---

You are tasked with conducting a comprehensive audit of the GLX specification to identify issues, contradictions, ambiguities, and areas for improvement.

## Scope

Analyze all specification files in the `specification/` directory:

### Top-Level Specification Files
- `1-introduction.md` - Project overview and purpose
- `2-core-concepts.md` - Core GLX concepts
- `3-archive-organization.md` - How GLX archives are structured
- `README.md` - Specification index

### Entity Type Specifications
- `4-entity-types/` - Individual entity type definitions
  - assertion.md, citation.md, event.md, media.md, person.md, place.md, relationship.md, repository.md, source.md, vocabularies.md

### Standard Vocabularies
- `5-standard-vocabularies/` - Controlled vocabulary definitions (.glx files)
  - confidence-levels.glx, event-types.glx, event-properties.glx, media-types.glx, participant-roles.glx, person-properties.glx, place-types.glx, place-properties.glx, relationship-types.glx, relationship-properties.glx, repository-types.glx, source-types.glx

**Note**: Schema validation is handled by `/check-schema-drift` - this command focuses on internal specification consistency only.

## What to Check

### 1. Internal Contradictions

Identify statements that contradict each other:
- Same field described with different types in different places
- Conflicting requirements (e.g., "field is required" vs "field is optional")
- Contradictory descriptions of behavior
- Incompatible examples

### 2. Terminology Consistency

Check for inconsistent use of terms:
- Same concept referred to by different names
- Same term used to mean different things
- Inconsistent capitalization (e.g., "Event" vs "event")
- Mixing of synonyms (e.g., "archive" vs "file" vs "document")

### 3. Broken or Invalid References

Verify all cross-references work:
- Links to other specification sections that don't exist
- References to entity types that aren't defined
- Citations of vocabulary terms not in the vocabulary files
- Examples referencing undefined fields

### 4. Completeness Issues

Check for missing or incomplete content:
- Entity types mentioned but not fully documented
- Fields listed in examples but not in field tables
- Vocabularies referenced but not defined
- Sections marked as "TODO" or "Coming soon"
- Missing examples for complex features

### 5. Ambiguous Language

Flag unclear or ambiguous specifications:
- Vague requirements using "should", "may", "can" without clear meaning
- Ambiguous field descriptions that could be interpreted multiple ways
- Unclear validation rules or constraints
- Missing details on edge cases or error handling

### 6. Example Validation

Verify all examples are correct:
- YAML examples are valid YAML syntax
- Field values match field types documented in the specification
- Required fields (as documented) are present in examples
- Examples demonstrate the features they claim to
- Examples are consistent with surrounding prose

### 7. Logical Inconsistencies

Check for logical problems:
- Circular dependencies between entity types
- Impossible constraints (e.g., mutually exclusive required fields)
- Missing relationship definitions (e.g., entity A references B, but B doesn't define the relationship)
- Validation rules that conflict with examples

### 8. Vocabulary Issues

Review standard vocabularies for problems:
- Terms defined multiple times with different meanings
- Missing standard vocabulary files referenced in docs
- Vocabulary structure inconsistent with documented format
- Terms in examples not in standard vocabularies

### 9. Version Consistency

Check version-related issues:
- Version numbers inconsistent across files
- Changelog doesn't match actual changes
- Breaking changes not clearly marked
- Migration guidance missing or incomplete

## Output Format

Organize findings by category and severity:

### Critical Issues 🔴
Issues that make the specification unusable or dangerously misleading:
- `[Location]` - Brief description
- **Problem**: Detailed explanation
- **Impact**: Why this is critical
- **Recommendation**: How to fix

### Major Issues 🟡
Issues that significantly impact usability or clarity:
- `[Location]` - Brief description
- **Problem**: Detailed explanation
- **Impact**: Why this matters
- **Recommendation**: How to fix

### Minor Issues 🔵
Issues that could be improved but don't block usage:
- `[Location]` - Brief description
- **Problem**: Detailed explanation
- **Recommendation**: How to fix

### Positive Findings ✅
Things done well that should be maintained:
- `[Aspect]` - What works well and why

## Summary Report

At the end, provide:

1. **Statistics**:
   - Total specification files reviewed
   - Critical issues found
   - Major issues found
   - Minor issues found

2. **Top Priority Fixes**:
   - List the 3-5 most important issues to address first

3. **Overall Assessment**:
   - Overall specification quality rating (needs work / good / excellent)
   - Key strengths
   - Key areas for improvement

4. **Recommendations**:
   - Concrete next steps to improve specification quality
   - Process improvements to prevent future issues

## Methodology

- Be thorough but practical - focus on real issues that impact users
- Provide specific file paths and line numbers when possible
- Include quotes from the specification to support findings
- Suggest concrete improvements, not just criticism
- Consider the specification from a user's perspective (someone implementing GLX)
- Cross-reference between different specification sections
- Validate examples are internally consistent with the specification prose

## Important Notes

- This is a quality audit, not a style critique - focus on correctness and clarity
- Prioritize issues that would confuse implementers or cause incompatible implementations
- Consider the specification from both a human reader's perspective and as documentation that must be clear and unambiguous
- Flag issues even if you're not 100% certain - better to investigate than miss problems
- If examples use vocabulary terms, verify those terms exist in vocabulary files
- Schema-related issues should be reported via `/check-schema-drift` instead

## Cross-Reference with todo.md

**IMPORTANT**: Before finalizing your report, read `todo.md` and exclude any issues that are already tracked there. The todo.md file is the canonical list of known issues.

When you find an issue:
1. Check if it's already in todo.md under any section (Documentation, Type System, GEDCOM Import, Validation, etc.)
2. If already tracked: Do NOT include it in your report
3. If NOT tracked: Include it in your report with full details

This prevents duplicate tracking and keeps the audit focused on newly discovered issues.
