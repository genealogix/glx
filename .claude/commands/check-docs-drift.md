---
description: Identify drift between docs/ folder and specification/schemas/code
allowed-tools:
  - Read
  - Grep
  - Glob
  - Write
  - Bash(mktemp -d /tmp/glx-drift-*:*)
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

1. Extract the YAML body from the markdown source.
2. Create a unique scratch directory and capture its path:

```bash
mktemp -d /tmp/glx-drift-XXXXXX
```

   Read the printed path (e.g. `/tmp/glx-drift-aB3xYz`); substitute it for `<TMPDIR>` in the remaining steps. Bash sessions don't persist between tool calls, so each subsequent command must use the literal path you captured — this is also what makes each command individually match its allow-list entry.

3. Use the **Write** tool to save the extracted YAML body to `<TMPDIR>/snippet.glx`. Using Write rather than shelling out via `cat`/`printf`/heredoc avoids every shell-quoting concern around YAML bodies that contain `$`, backticks, or quotes.

4. Run the validator on the file — do NOT mentally simulate schema validation:

```bash
./bin/glx validate <TMPDIR>/snippet.glx
```

5. Clean up this snippet's scratch directory before moving to the next one:

```bash
rm -rf <TMPDIR>
```

   Per-snippet cleanup keeps concurrent runs of this command from racing on the shared `/tmp/glx-drift-*` namespace. Substituting the literal path also keeps the command unambiguously matched against the `Bash(rm -rf /tmp/glx-drift-*:*)` allow-list entry regardless of how the matcher handles variable expansion.

6. Classify the result deterministically by exit code alone — do NOT inspect the snippet's keys to second-guess the validator (that would re-introduce the LLM-simulation anti-pattern this command was rewritten to remove). Record the exit code and full stderr in all cases.
   - **Exit 0** → snippet passed structural + semantic validation (`glx validate` runs both in single-file mode; only cross-reference checks are skipped). Still apply the narrative checks below since the validator does not catch specification-prose drift.
   - **Any non-zero exit** → **CRITICAL**. Report the validator's stderr verbatim so the human reviewer can decide whether the finding is real drift (typoed wrapper like `people:` instead of `persons:`, deprecated field, malformed structure) or a *Phase-1 limitation* artifact: `glx validate` requires archive-shape input at top level, so doc blocks that demonstrate a bare entity fragment, a vocabulary entry, or a properties excerpt will trip `(root): additional properties '<key>', ... not allowed` even when the doc itself is correct. #910 tracks `--stdin --entity-type` to validate fragments directly; until it lands, treat ambiguous root-level errors as a human-review request rather than auto-downgrading them with an LLM heuristic.

If `./bin/glx` is unavailable in the session, surface `category: validator_unavailable` rather than guessing (build with `make build-cli` — or directly `go build -o bin/glx ./glx` on systems without `make` — if needed).

Beyond the structural check, compare each snippet against `specification/4-entity-types/*.md`:
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
