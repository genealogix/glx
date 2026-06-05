---
description: Release-time pass that folds self-cancelling and follow-up changie entries in the freshly-batched version section and verifies reference integrity
---

Consolidate the changelog section that `changie batch` just produced. Since the move to file-per-change fragments (#858), changes are authored as `.changes/unreleased/*.yaml` fragments and assembled by `changie batch <version>` into a single per-version file `.changes/<version>.md` (then `changie merge` regenerates `CHANGELOG.md`). changie concatenates each fragment as its own bullet under its kind — it does **no** semantic merging — so this command is the **once-per-release** pass that does the semantic work changie cannot: removing self-cancelling pairs, folding follow-up entries into the change they refine, and verifying that no issue/PR reference is dropped, swapped, or invented in the process.

This is *not* a continuous after-every-edit command anymore. Run it during release preparation, after `changie batch <version>` and before merging the release PR (the `prepare-release` workflow points here). The duplicate-`###`-section merge that the old command did is gone: changie emits exactly one header per kind in Keep-a-Changelog order, so that class cannot occur (#853 is fixed by construction).

## Target

Operate on the single freshly-batched version file: `.changes/<version>.md` (e.g. `.changes/0.0.0-beta.12.md`). It contains one `## [version] - date` header followed by `### Kind` subsections (Keep a Changelog order: Added, Changed, Deprecated, Removed, Fixed, Security) and `- {body} ({ref})` bullets.

## Pre-Flight

1. Determine the version being prepared. If the user passed one, use `.changes/<version>.md`. Otherwise run `changie latest` to get the most recently batched version and target that file. If the file does not exist, STOP and tell the user to run `changie batch <version>` first.
2. Confirm the target version is **not already released**: run `git fetch --tags --quiet` then `git tag --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+(-beta.*)?$' | head -1`. If a tag already matches the target version, STOP — the section is released history and must not be rewritten. (This is the same POSIX-ERE tag pattern the release workflow uses; `grep -E` avoids fnmatch `*` matching dots.)
3. Save the verbatim original of the target file as the reference baseline:

       cp ".changes/<version>.md" "${TMPDIR:-/tmp}/glx-section.before"

## Rules

### 1. Remove self-cancelling changes

If a feature was added and then removed in the same version, or a bug was introduced and then fixed within this same version, remove both entries. Look for:
- an `Added` bullet paired with a `Removed` bullet for the same feature;
- a `Fixed` bullet that fixes something introduced by an `Added`/`Changed` bullet in this same version;
- a `Changed` bullet that reverts an earlier `Changed` bullet in this same version.

Only remove entries that fully cancel out. If a feature was added and then *modified*, keep the final state as a single `Added` entry (that is Rule 2, not this rule).

### 2. Consolidate follow-up entries

When a feature is added and then enhanced, refined, or fixed within the same version, combine the bullets into a single entry reflecting the final state. The agentic "one fragment per change" workflow makes this common: `add X`, `fix X bug`, and `refine X` arrive as three separate fragments (under three different kinds), so they will not even be adjacent. Fold them into one bullet under the most appropriate kind (usually the original `Added`/`Changed`). The result should read as if the feature were implemented correctly the first time.

**Preserve every reference.** This is the highest-risk part: a careless merge can silently drop a `(#688)`, swap it for `(#687)`, or invent a `(#689)` that no source bullet carried — and `CONTRIBUTING.md` requires every changelog entry to carry an issue/PR reference. When consolidating bullets, the merged bullet MUST carry the **union** of every reference from every source bullet — never drop, renumber, or invent one. "Reference" means any `#NNN` in any form (`(#681)`, `Fixes #687`, `(PR #456)`, bare `#673`) plus any bare commit SHA. For example, an `Added` bullet `(#681)` merged with a `Fixed` bullet `(#688, Fixes #687)` becomes a single bullet carrying `(#681, #687, #688)`.

### 3. Keep the structure changie produced

- Keep the `## [version] - date` header and the `### Kind` headers/order exactly as changie emitted them.
- Remove a `### Kind` header only if consolidation empties it entirely.
- Do not reintroduce audit/provenance prose; bullets stay in their normal kind sections.

## Process

1. Run the pre-flight. Stop if any check fails.
2. Compute the rewritten section **in memory** following Rules 1–3. Save it to a second temp file:

       # write your rewrite to "${TMPDIR:-/tmp}/glx-section.after"

3. **Reference inventory (before).** Extract the reference set from the original:

       grep -oE '#[0-9]+|[0-9a-f]{7,40}' "${TMPDIR:-/tmp}/glx-section.before" | sort -u > "${TMPDIR:-/tmp}/glx-refs.pre"   # R_pre

   (`#[0-9]+` catches every `(#NNN)`/`Fixes #NNN`/`(PR #NNN)` form; `[0-9a-f]{7,40}` catches bare commit SHAs. Bare-hex can rarely match a prose word like `feedbac`; that only ever over-refuses, never misses a real ref, so it fails safe.)
4. **Account for intentional removals.** Build **R_drop** = references that appear *only* inside entries you deliberately removed in full under Rule 1 (they do not also appear in any retained entry). The expected surviving set is **R_expected = R_pre − R_drop**, and every reference in R_drop must be named in the summary alongside the entry it left with.
5. **Reference inventory (after) + forward check.** Extract the reference set from the rewrite:

       grep -oE '#[0-9]+|[0-9a-f]{7,40}' "${TMPDIR:-/tmp}/glx-section.after" | sort -u > "${TMPDIR:-/tmp}/glx-refs.post"  # R_post

   If any reference in R_expected is missing from R_post, **refuse to write the file.** Report each dropped reference, the source entry it came from, and where it should have landed. The user may correct the consolidation and re-run, or explicitly waive the check in natural language (there is no flag), in which case list every waived reference in the summary.
6. **Invented-reference check (reverse direction) + hard error.** Compute references present in the rewrite but absent from the original:

       comm -13 "${TMPDIR:-/tmp}/glx-refs.pre" "${TMPDIR:-/tmp}/glx-refs.post" > "${TMPDIR:-/tmp}/glx-refs.added"   # R_post − R_pre

   By default this must be **empty** (`R_post ⊆ R_pre`). If the user has approved any intentional additions (e.g. backfilling a reference an original entry omitted), list them in `glx-refs.add` and subtract:

       : > "${TMPDIR:-/tmp}/glx-refs.add"                                         # approved additions, default none
       comm -23 "${TMPDIR:-/tmp}/glx-refs.added" "${TMPDIR:-/tmp}/glx-refs.add"   # R_invented

   If R_invented is non-empty, **refuse to write the file.** Report each invented reference and the rewritten entry that introduced it. Like the forward check, this fails safe.
7. Write the consolidated section back to `.changes/<version>.md` **only if both checks passed** (or were explicitly waived/approved).
8. Regenerate the changelog so `CHANGELOG.md` reflects the consolidation:

       changie merge

9. Show a summary: entries removed (self-cancelling, with their R_drop references), entries consolidated, the reference check result (e.g. `12/12 preserved`, any waived, zero invented), and the net reduction in bullet count.
