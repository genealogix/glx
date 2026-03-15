---
description: Compact the latest changelog entry by merging duplicates, removing self-cancelling changes, and consolidating follow-ups
---

Compact the latest version entry in `CHANGELOG.md`. Read the file, identify the latest `## [version]` section (everything from the first `## [` to the next `## [` or end of file), and apply the following compaction rules:

## Pre-Flight Checks

Before compacting, verify changelog integrity:

### 1. Ensure Latest Version Is Unreleased

Run `git tag --sort=-v:refname | head -1` to get the most recent release tag. Compare the tag version against the latest `## [version]` header in the changelog (tags use `v` prefix, changelog doesn't — e.g., tag `v0.0.0-beta.7` matches changelog `0.0.0-beta.7`).

**If the latest changelog version already has a matching tag, STOP.** The latest section has already been released — there should be a newer unreleased section above it. If there isn't, warn the user that a new section needs to be created before adding entries.

### 2. Fix Entries Added to Released Sections

Identify the second `## [version]` section (the one immediately after the latest). This should correspond to the most recent release tag.

Run `git show <tag>:CHANGELOG.md` (e.g., `git show v0.0.0-beta.7:CHANGELOG.md`) to get the changelog as it existed at that release. Extract the matching version section from both the tagged version and the current file. Diff them (ignoring the date-line change from `Unreleased` to a date, which is expected).

**If the previous section has new entries that weren't in the tagged release, move them to the latest (unreleased) section.** Then restore the released section to match the tagged version exactly (except the date). This is a common issue when agentic editing adds entries to the wrong section. After moving, merge the relocated entries into the appropriate subsections (Added/Changed/Fixed/Removed) of the latest version, following the same deduplication and ordering rules.

### 3. Update the Date on the Latest Version

If the latest section header says `- Unreleased` (e.g., `## [0.0.0-beta.8] - Unreleased`), update it to today's date (e.g., `## [0.0.0-beta.8] - 2026-03-15`). This keeps the date current as work progresses.

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

1. Run pre-flight checks (1-3 above). Stop if any check fails.
2. Read `CHANGELOG.md`
3. Extract ONLY the latest version section
4. Apply rules 1-5 to that section
5. Write the compacted changelog back
6. Show a summary of what changed:
   - Pre-flight check results (tag verified, previous section status, date updated)
   - Sections merged
   - Entries removed (self-cancelling)
   - Entries consolidated
   - Net reduction in line count
