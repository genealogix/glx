---
description: Identify drift between docs/ folder and specification/schemas/code
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
- Extract YAML examples from markdown prose
- Verify they would pass schema validation
- Check for outdated syntax or deprecated fields
- Verify field names, types, and structure match specification

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
