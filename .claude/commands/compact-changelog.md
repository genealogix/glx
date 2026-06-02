---
description: Compact the latest changelog entry by merging duplicates, removing self-cancelling changes, and consolidating follow-ups
---

Compact the latest version entry in `CHANGELOG.md`. Read the file, identify the latest `## [version]` section (everything from the first `## [` to the next `## [` or end of file), and apply the following compaction rules:

## Pre-Flight Checks

Before compacting, verify changelog integrity:

### 1. Ensure Latest Version Is Unreleased

Run `git fetch --tags --quiet` to refresh local tag refs (prevents stale-local-tag drift from upstream). Then run `git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+(-beta.*)?$' | head -1` to get the most recent release tag. The `grep -E` regex is the POSIX-ERE translation of the GitHub Actions tag filters in `.github/workflows/release.yml` (`v[0-9]+.[0-9]+.[0-9]+` and `v[0-9]+.[0-9]+.[0-9]+-beta*`) — using `grep -E` rather than `git tag --list`'s fnmatch glob avoids `*` matching dots and stray suffixes (so `v1.2.3.4`, `v1.2.3-rc1`, and similar non-release tags are correctly excluded). Compare the tag version against the latest `## [version]` header in the changelog (tags use `v` prefix, changelog doesn't — e.g., tag `v0.0.0-beta.7` matches changelog `0.0.0-beta.7`).

**If the latest changelog version already has a matching tag, STOP.** The latest section has already been released — there should be a newer unreleased section above it.

**STOP semantics.** Print the offending state to the user, print
"Refusing to proceed. Create the next unreleased section first by:

    1. Decide the next version (e.g., 0.0.0-beta.11)
    2. Add a `## [0.0.0-beta.11] - Unreleased` header above the
       most recent dated section
    3. Re-run /compact-changelog

" and return without modifying CHANGELOG.md. Do not attempt to infer
or create the missing section yourself.

### 2. Fix Entries Added to Released Sections

Find the changelog section that matches the latest release tag. Do NOT assume it is the second section — there may be intermediate unreleased versions between the latest section and the tagged one. Search all `## [version]` headers to find the one whose version matches the latest tag.

Run `git show <tag>:CHANGELOG.md` (e.g., `git show v0.0.0-beta.8:CHANGELOG.md`) to get the changelog as it existed at that release. Note: this fails with a non-zero exit and prints `fatal: path 'CHANGELOG.md' does not exist` to stderr if `CHANGELOG.md` did not exist at that tag. In this repo that path cannot trigger today — the project's earliest tag (`v0.0.0-beta.0`) post-dates `CHANGELOG.md` — but if a future rename or removal ever makes it possible, treat the released section as empty for the diff and continue. Extract the matching version section from both the tagged version and the current file. Diff them (ignoring the date-line change from `Unreleased` to a date, which is expected).

**If the released section has new entries that weren't in the tagged release, move them to the latest (unreleased) section.** Then restore the released section to match the tagged version exactly (except the date). This is a common issue when agentic editing adds entries to the wrong section. After moving, merge the relocated entries into the appropriate subsections (Added/Changed/Fixed/Removed) of the latest version, following the same deduplication and ordering rules.

If there are sections between the latest and the tagged section that have no corresponding tag, flag this for the user — they may need to be consolidated or tagged.

### 3. Update the Date on the Latest Version

If the latest section header does not already have a date in `YYYY-MM-DD` format (e.g., `## [0.0.0-beta.9]` or `## [0.0.0-beta.9] - Unreleased`), update it to today's date (e.g., `## [0.0.0-beta.9] - 2026-03-29`). This keeps the date current as work progresses.

**If the latest section header is bare `## [Unreleased]` with no version number**, refuse to proceed. Print the offending header to the user, print "Refusing to proceed. The latest section has no version. Promote `## [Unreleased]` to `## [<next-version>] - YYYY-MM-DD` yourself (e.g., `## [0.0.0-beta.11] - 2026-05-25`), then re-run /compact-changelog." and return without modifying CHANGELOG.md. Do not infer the next version from the latest tag — version bumps are a maintainer decision and out of scope for compaction.

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

**Preserve every reference.** This is the highest-risk part of compaction: a careless merge can silently drop a `(#688)` or swap it for `(#687)`, and `CONTRIBUTING.md` requires every changelog entry to carry an issue or PR reference. When consolidating bullets, the merged bullet MUST carry the **union** of all `(#NNN)`, `Fixes #NNN`, `Closes #NNN`, `(PR #NNN)`, and bare commit-SHA references from every source bullet — never drop, renumber, or invent one. For example, an Added entry `(#681)` merged with a Fixed entry `(#688, Fixes #687)` becomes a single entry carrying `(#681, Fixes #687, fix in #688)`. Reference preservation is verified deterministically after the rewrite is computed and before it is written to disk (see Process steps 4a–4c).

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
3. Extract ONLY the latest version section. Save this verbatim original to a temporary file (e.g. `"${TMPDIR:-/tmp}/glx-section.before"`) — it is the baseline for the reference check below. The extraction is the latest section from its `## [` header to the line before the next `## [` header:

       awk '/^## \[/{n++} n==1' CHANGELOG.md > "${TMPDIR:-/tmp}/glx-section.before"

4. Apply rules 1-5 to that section to produce the rewritten section **in memory** — do not write `CHANGELOG.md` yet. Save the rewritten section to a second temporary file (e.g. `"${TMPDIR:-/tmp}/glx-section.after"`).
4a. **Reference inventory (pre-compaction).** Extract the reference set from the *original* section:

       grep -oE '#[0-9]+|[0-9a-f]{7,40}' "${TMPDIR:-/tmp}/glx-section.before" | sort -u

    The `#[0-9]+` alternative catches `(#NNN)`, `Fixes #NNN`, `Closes #NNN`, and `(PR #NNN)`; the `[0-9a-f]{7,40}` alternative catches bare commit SHAs. (Bare-hex matching can rarely match a prose word like `feedbac`; that only ever causes the check to *over*-refuse, never to miss a real ref, so it fails safe.) Call this set **R_pre**.
4b. **Account for intentional removals (Rule 2).** Self-cancelling pairs removed under Rule 2 legitimately take their references out of the section. Build **R_drop** = the references that appear *only* inside entries you removed as self-cancelling (i.e. they do not also appear in any retained entry). The expected surviving set is **R_expected = R_pre − R_drop**. Removals under Rule 5 (audit context) never carry `#NNN`/SHA references, so they do not affect this set.
4c. **Reference inventory (post-compaction) + hard error.** Extract the reference set from the rewritten section:

       grep -oE '#[0-9]+|[0-9a-f]{7,40}' "${TMPDIR:-/tmp}/glx-section.after" | sort -u

    Call this **R_post**. If any reference in **R_expected** is missing from **R_post**, **refuse to write `CHANGELOG.md`.** Report each dropped reference, the source entry it came from, and the entry it should have landed in. The user can then either (a) fix the consolidation and re-run, or (b) explicitly waive the check by re-running and stating that reference loss is acceptable — the command has no `--allow-ref-loss` flag, so this override is given in natural language and the agent must treat it as a deliberate instruction, proceed, and list every waived reference in the summary.
4d. **Per-entry union rule.** Restating Rule 3 as the operational guard for 4c: whenever two bullets merge, the merged bullet carries the union of both bullets' references. This is what keeps **R_post ⊇ R_expected** by construction; steps 4a–4c are the deterministic backstop for when it silently fails.
5. Write the compacted changelog back **only if step 4c passed** (or was explicitly waived).
6. Show a summary of what changed:
   - Pre-flight check results (tag verified, previous section status, date updated)
   - Sections merged
   - Entries removed (self-cancelling)
   - Entries consolidated
   - **Reference check**: count of references preserved (e.g. `12/12`), plus any in `R_drop` (intentionally removed with self-cancelling entries) or explicitly waived
   - Net reduction in line count
