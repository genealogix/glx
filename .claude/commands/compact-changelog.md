---
description: Compact the latest changelog entry by merging duplicates, removing self-cancelling changes, and consolidating follow-ups
---

Compact the latest version entry in `CHANGELOG.md`. Read the file, identify the latest `## [version]` section (everything from the first `## [` to the next `## [` or end of file), and apply the following compaction rules:

## Rules

### 1. Merge Duplicate Sections

Agentic editing often creates duplicate top-level sections (e.g., two `### Added` or two `### Changed` blocks). Merge them into a single section per change type. Preserve all unique `####` subsections and bullet points from both duplicates. Maintain the standard section order: Added, Changed, Fixed, Removed.

### 2. Remove Self-Cancelling Changes

If a feature was added and then removed in the same version, or a bug was introduced and then fixed, remove both entries. Look for patterns like:
- An "Added" entry paired with a "Removed" entry for the same feature
- A "Fixed" entry that fixes something introduced in an "Added" or "Changed" entry in this same version
- A "Changed" entry that reverts a previous "Changed" entry

Only remove entries that fully cancel out. If a feature was added and then modified, keep the final state as a single "Added" entry.

### 3. Consolidate Follow-Up Entries

When a feature is added and then enhanced/refined in the same version, combine them into a single entry reflecting the final state. Look for:
- Multiple bullet points about the same feature across different sections
- An "Added" entry followed by "Changed" entries that refine it
- "Fixed" entries that fix issues in features added in this same version (fold the fix into the feature description)

The result should read as if the feature was implemented correctly the first time.

### 4. Preserve Structure

- Keep the VitePress frontmatter and preamble unchanged
- Keep all other version sections (`## [older-version]`) unchanged
- Maintain `#### Subsection` groupings where they add clarity
- Remove empty sections (a `### Added` with no bullets under it)
- Keep the "Keep a Changelog" format and conventions

### 5. Remove any specific audit references

- We don't need to keep the context of how specific audits identified changes to make.
- Items should be regrouped in normal sections

## Process

1. Read `CHANGELOG.md`
2. Extract ONLY the latest version section
3. Apply rules 1-4 to that section
4. Write the compacted changelog back
5. Show a summary of what changed:
   - Sections merged
   - Entries removed (self-cancelling)
   - Entries consolidated
   - Net reduction in line count
