---
description: Identify drift between docs/ folder and specification/schemas/code
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash(mktemp:*)
  - Bash(./bin/glx validate:*)
  - Bash(rm -rf /tmp/glx-drift-*:*)
model: claude-opus-4-7
---

You are tasked with identifying any drift between the GLX user documentation and the source of truth (specification, schemas, and code).

**Note:** Example archives (`docs/examples/`) are checked separately by the `check-examples` command. This command focuses only on user-facing documentation files.

## Source of Truth Flow

**IMPORTANT**: Documentation is derived from the source of truth:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
         ↓
    User Docs
  (quickstart,
   guides/)
```

**This means:**
- **Specification/Schema/Code are the source of truth**
- Documentation is **derived from** these sources
- Any drift detected means the **documentation needs to be updated**
- When reporting drift, frame it as "Documentation X needs to be updated because source says Y"

## Task

Analyze the user documentation in **docs/** (excluding `docs/examples/`) and compare it with the source of truth.

## Files to Check

- `docs/quickstart.md`
- `docs/guides/best-practices.md`
- `docs/guides/migration-from-gedcom.md`
- `docs/use-cases.md`
- `specification/6-glossary.md` (glossary is part of specification)

Compare with **specification/4-entity-types/*.md** and **glx/cmd_*.go** (CLI commands).

## What to Check

### 1. Entity Field Documentation
- Compare documented fields with specification/4-entity-types/*.md
- Check that field names match exactly (e.g., `state_province` not `state`)
- Verify required vs optional matches specification
- Check field types are accurately described

### 2. Example Code Blocks in Documentation

For each YAML-tagged fenced code block in a documentation markdown file:

1. Extract the YAML body.
2. Create a unique scratch directory, write the snippet to a file inside it, then invoke the real validator — do NOT mentally simulate schema validation:

```bash
tmpdir=$(mktemp -d /tmp/glx-drift-XXXXXX)
printf '%s\n' "$snippet" > "$tmpdir/snippet.glx"
./bin/glx validate "$tmpdir/snippet.glx"
```

3. Record the validator's exit code and stderr verbatim in the findings.
4. After all snippets in the run have been validated, clean up once:

```bash
rm -rf /tmp/glx-drift-*
```

If `./bin/glx` is unavailable in the session, surface a finding with `category: validator_unavailable` rather than falling back to a guess (build with `make build-cli` if needed).

**Phase-1 limitation:** `glx validate` currently requires the snippet to be a full archive-shape file (top-level `persons:` / `events:` / `places:` / etc.). If the doc block is a partial snippet (bare entity, comment-only header, or properties fragment), the validator will report a structural error at root. In that case, classify the finding as `category: snippet_not_archive_shape` (informational, not CRITICAL) and fall through to the narrative checks below. A follow-up issue tracks adding `--stdin --entity-type` to `glx validate` so partial snippets can be validated in-place.

**Findings semantics:**
- Validator exits non-zero on an archive-shaped snippet → **CRITICAL** (a documented example does not validate).
- Validator exits non-zero on a partial snippet (root-level "additional properties not allowed") → **INFORMATIONAL** (`snippet_not_archive_shape`).
- Validator exits zero → still apply the narrative checks below, since `glx validate` enforces schema conformance but not specification-prose accuracy.

**Beyond the structural check**, compare each snippet against `specification/4-entity-types/*.md`:
- Check for outdated syntax or deprecated fields
- Verify field names, types, and structure match what the specification documents

### 3. CLI Command Examples
- Verify documented commands exist in glx/cmd_*.go
- Check command flags and arguments are accurate
- Verify output examples match actual behavior

### 4. Vocabulary References
- Check that referenced vocabulary types exist
- Verify example values are in standard vocabularies (specification/5-standard-vocabularies/)

### 5. Internal Links
- Verify links between documentation files are correct
- Check that specification links omit `.md` extension (VitePress compatibility)

## Output Format

```
# Documentation Drift Report

### docs/quickstart.md
✅ No drift detected - Matches specification

OR

⚠️ Drift detected - Documentation needs updates:

- Line 42: Example shows field `name` as required, but specification says it's optional
- Line 78: CLI command `glx init` shown with flag `--format` that doesn't exist in code
- Line 105: Example uses deprecated field `description`, should use `notes`
- Line 120: References vocabulary type `birth` but standard vocabulary uses `natural-birth`

### docs/guides/best-practices.md
[Similar format]
```

## Summary

At the end, provide:
- Total documentation files checked
- Count of files with drift
- List of files that need updates
- Severity assessment per file (critical/major/minor)
- Recommended actions: "Update [doc files] to match [source of truth]"

## Common Issues to Look For

- Outdated CLI commands or flags
- Incorrect required/optional field documentation
- Examples in prose with invalid YAML syntax
- References to deprecated or renamed fields (e.g., `state` vs `state_province`)
- Wrong vocabulary type names
- Internal links with `.md` extension (should be omitted for VitePress)

## Notes

- **Source of truth hierarchy**: Specification → Schema → Code → Documentation
- User documentation errors are **high severity** - users rely on these
- Minor wording differences are acceptable if meaning is preserved
- Focus on technical accuracy, not writing style
- CLI examples should be copy-paste ready
