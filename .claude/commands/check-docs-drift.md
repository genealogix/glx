---
description: Identify drift between docs/ folder and specification/schemas/code
---

You are tasked with identifying any drift between the GLX documentation and the source of truth (specification, schemas, and code).

## Source of Truth Flow

**IMPORTANT**: Documentation is derived from the source of truth:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
         ↓                      ↓                      ↓
    User Docs              Examples              Dev Docs
  (quickstart,           (docs/examples/      (architecture,
   guides/)               *.glx files)      schema-development)
```

**This means:**
- **Specification/Schema/Code are the source of truth**
- Documentation is **derived from** these sources
- Any drift detected means the **documentation needs to be updated**
- When reporting drift, frame it as "Documentation X needs to be updated because source says Y"

## Task

Analyze the documentation in **docs/** and compare it with the source of truth.

## Areas to Check

### 1. User Documentation (quickstart.md, guides/)

Compare with **specification/4-entity-types/*.md**:

- **Field names and types** - Are entity fields documented correctly?
- **Required vs optional** - Are requirements accurate?
- **Examples in docs** - Do code examples match the specification?
- **Vocabulary references** - Are vocabulary types correct?
- **File format** - Is the YAML structure accurate?
- **CLI commands** - Do documented commands exist in the actual CLI?

**Files to check:**
- `docs/quickstart.md`
- `docs/guides/best-practices.md`
- `docs/guides/glossary.md`
- `docs/guides/migration-from-gedcom.md`
- `docs/use-cases.md`

### 2. Example Archives (docs/examples/*.glx)

Compare with **specification/schema/v1/*.schema.json**:

- **Schema validity** - Are example files valid according to schemas?
- **Required fields** - Do examples include all required fields?
- **Field types** - Are field values the correct type?
- **References** - Do entity references point to existing entities?
- **Vocabulary usage** - Do examples use valid vocabulary values?
- **Best practices** - Do examples follow patterns from specification?

**Example directories to check:**
- `docs/examples/single-file/`
- `docs/examples/minimal/`
- `docs/examples/complete-family/`
- `docs/examples/basic-family/`
- `docs/examples/temporal-properties/`
- `docs/examples/participant-assertions/`

### 3. Development Documentation (docs/development/)

Compare with **glx/lib/types.go** and **specification/schema/v1/**:

- **Go type references** - Are Go struct names and fields accurate?
- **Schema references** - Are schema descriptions accurate?
- **Architecture** - Does it match actual code structure?
- **Implementation details** - Are technical details correct?

**Files to check:**
- `docs/development/architecture.md`
- `docs/development/schema-development.md`
- `docs/development/testing-guide.md`
- `docs/development/gedcom-import.md`

## What to Check

### For User Documentation:

1. **Entity Field Documentation**
   - Compare documented fields with specification/4-entity-types/*.md
   - Check that field names match exactly (e.g., `state_province` not `state`)
   - Verify required vs optional matches specification

2. **Example Code Blocks**
   - Extract YAML examples from markdown
   - Verify they would pass schema validation
   - Check for outdated syntax or deprecated fields

3. **CLI Command Examples**
   - Verify documented commands exist in glx/cmd_*.go
   - Check command flags and arguments are accurate
   - Verify output examples match actual behavior

4. **Vocabulary References**
   - Check that referenced vocabulary types exist
   - Verify example values are in standard vocabularies

### For Example Archives:

1. **Schema Compliance**
   - Each .glx file should be valid against its schema
   - Required fields must be present
   - Field types must match schema

2. **Cross-References**
   - Entity references should point to entities that exist in the example
   - No dangling references

3. **Completeness**
   - Examples should demonstrate documented features
   - README files should accurately describe what's in the example

### For Development Documentation:

1. **Code Accuracy**
   - Go struct field names and types match lib/types.go
   - File paths and module names are correct
   - Function signatures match actual code

2. **Schema Accuracy**
   - Schema field names and types match schema files
   - Validation rules described match schema constraints

3. **Architecture Accuracy**
   - Package structure matches actual codebase
   - Described patterns exist in the code

## Output Format

```
# Documentation Drift Report

## User Documentation

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

## Example Archives

### docs/examples/complete-family/persons/person-john.glx
⚠️ Drift detected - Example needs updates:

- Missing required field `properties` (schema requires it)
- Field `birthdate` doesn't exist in Person schema (should be in properties)
- References `event-birth-123` but that event doesn't exist in the example

### docs/examples/single-file/archive.glx
✅ No drift detected - Valid according to schema

## Development Documentation

### docs/development/architecture.md
⚠️ Drift detected - Documentation needs updates:

- Line 25: References struct `GLXArchive` but code uses `GLXFile`
- Line 45: Shows field `Relationships map[string]Relationship` but code uses `*Relationship`
- Line 78: Describes package `glx/validator` but code uses `glx/lib` with validation methods

### docs/development/schema-development.md
✅ No drift detected - Matches schemas
```

## Summary

At the end, provide:
- Total documentation files checked
- Count of files with drift
- List of files that need updates
- Severity assessment per file (critical/major/minor)
- Recommended actions: "Update [doc files] to match [source of truth]"

## Common Issues to Look For

### In User Documentation:
- Outdated CLI commands or flags
- Incorrect required/optional field documentation
- Examples with invalid YAML syntax
- References to deprecated fields
- Wrong vocabulary type names

### In Examples:
- Missing required fields
- Wrong field types (string vs array)
- Dangling references to non-existent entities
- Invalid vocabulary values
- Malformed dates

### In Development Documentation:
- Outdated Go struct names or fields
- Wrong file paths or module names
- Incorrect schema field names
- Mismatched types (e.g., `string` vs `[]string`)
- References to removed code

## Validation Commands

You can use these commands to help:

```bash
# Validate all example archives
glx validate docs/examples/

# Check if CLI commands exist
glx --help
glx validate --help
glx import --help
```

## Notes

- **Source of truth hierarchy**: Specification → Schema → Code → Documentation
- Examples must be **schema-valid** - this is critical for user trust
- User documentation errors are **high severity** - users rely on these
- Minor wording differences are acceptable if meaning is preserved
- Focus on technical accuracy, not writing style
- Development docs can reference internal implementation details
- CLI examples should be copy-paste ready
- **Required field missing in example is CRITICAL** - breaks user trust
