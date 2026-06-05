---
description: Identify drift between specification markdown files and JSON schemas
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash(git rev-parse:*)
  - Bash(date -u:*)
  - Bash(make check-schemas:*)
  - Bash(gh issue list:*)
model: claude-opus-4-7
---

You are tasked with identifying any drift between the GLX specification markdown files and the JSON schemas.

## Allowlisted Drift (read this first)

Before analyzing, read **`.claude/drift-allowlist.yaml`** (validated by
`.claude/drift-allowlist.schema.json`) — the single source of truth for known,
triaged drift, shared with `/check-code-drift`. If a finding concerns the same
`file` and symbol as an entry (match on symbol identity, not exact string),
suppress it: `permanent: true` is by-design (not drift); an
entry with a `tracking_issue` is a temporary deferral (refer to the issue rather
than re-reporting). Everything not in the allowlist is reported normally. The
allowlist holds only per-symbol exceptions — not class-level methodology — and is
human-curated, so don't invent entries.

## Source of Truth Flow

**IMPORTANT**: The source of truth hierarchy is:

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
     SOURCE OF TRUTH         DERIVED FROM SPEC      DERIVED FROM SCHEMA
```

**This means:**
- The **specification markdown files are the source of truth**
- JSON schemas are **derived from** the specification
- Any drift detected means the **schema needs to be updated** to match the specification
- When reporting drift, frame it as "Schema X needs to be updated because specification says Y"

## Task

Analyze all entity types and compare:

1. **specification/4-entity-types/*.md** - The source of truth (human-readable specification)
2. **specification/schema/v1/*.schema.json** - Derived schemas (machine-readable)

## Schemas to Check

### Entity Schemas
- assertion
- citation
- event
- media
- person
- place
- relationship
- repository
- source

### Top-Level Archive Schema
- `glx-file.schema.json` — the root schema defining overall GLX file structure, compare against `specification/3-archive-organization.md`:
  - `metadata` object fields match spec
  - All 9 entity map sections are present with correct `patternProperties` references
  - All vocabulary map sections are present
  - Entity ID pattern (`^[a-zA-Z0-9-]{1,64}$`) matches spec
  - Vocabulary key pattern matches spec

### Vocabulary Schemas
Compare `specification/4-entity-types/vocabularies.md` and `specification/5-standard-vocabularies/*.glx` against `specification/schema/v1/vocabularies/*.schema.json`:

**Type vocabularies** (10 schemas):
- event-types, relationship-types, place-types, source-types, media-types, repository-types, confidence-levels, participant-roles, sex-types, gender-types

**Property vocabularies** (8 schemas):
- person-properties, event-properties, relationship-properties, place-properties, media-properties, repository-properties, source-properties, citation-properties

For each vocabulary schema, verify:
- Schema structure matches the vocabulary format documented in `vocabularies.md` (label, description, gedcom, fields, etc.)
- Required fields in the schema match what the spec says is required for vocabulary entries
- For property vocabulary schemas specifically, property definition schemas enforce `value_type`/`reference_type`/`vocabulary_type` mutual exclusivity per spec
- All vocabulary `.glx` template files validate against their corresponding schema

### Files to ignore

These files live in the schema directories but are **not** subject to drift analysis. Always enumerate schemas with the glob `specification/schema/v1/*.schema.json` (and `specification/schema/v1/vocabularies/*.schema.json`) rather than a bare `ls`/`Glob specification/schema/v1/*` — the glob skips these automatically, but if you list the directory directly, exclude them explicitly:

- `specification/schema/v1/embed.go` — Go embed file for binary distribution (also present at `specification/5-standard-vocabularies/embed.go`)
- Any `README.md` that may live in a schema directory — directory readme, no spec counterpart
- Any file not ending in `.schema.json`

An agent that compares one of these (e.g., `embed.go`) against a non-existent spec section will emit a pure false positive. Do not.

## What to Check

For each entity type, verify:

### 1. Required Fields Alignment
- Compare "Required Fields" table in markdown with `required` array in JSON schema
- **EXCLUDE rows whose field name reads "Entity ID (map key)"** — that row documents the parent-map keying constraint (the entity is keyed by its ID in the parent map, enforced via `patternProperties` in `glx-file.schema.json`), not a property on the entity itself. The entity schema's `required` array MUST NOT contain it, so a naive "schema missing required field 'Entity ID (map key)'" comparison is a pure false positive on every entity. Do not flag it.
- Check that all remaining markdown-listed required fields appear in schema
- Check that schema doesn't have additional required fields not documented

### 2. Optional Fields Alignment
- Compare "Optional Fields" table in markdown with `properties` in JSON schema
- Verify all markdown-listed optional fields are in schema
- Verify schema doesn't have properties missing from markdown

### 3. Field Types
- Compare field types in markdown tables with JSON schema types
- Check for type mismatches (e.g., markdown says "string", schema says "array")

### 4. Field Descriptions

Use a **two-tier rule** rather than a free-form "do these roughly match?" judgment — paraphrase scored as drift is the dominant false-positive source in description comparison.

1. **Structural mismatch (flag, severity `major`)** — the schema description states a constraint **not** in the spec (e.g., "must be unique" appears only in the schema), OR **omits** a constraint the spec documents (e.g., spec says "ISO 8601 date", schema says only "date"). A constraint is anything that changes what values validate or how a consumer must treat the field.
2. **Wording-only (do NOT flag)** — pure rewording, expanded examples, or added length when no constraint changes. Natural-language paraphrase is not drift.

If unsure which tier applies, treat it as wording-only — false positives in description comparison degrade the entire report's signal-to-noise.

- **IGNORE** inline lists of vocabulary-defined values in description strings (e.g., a `properties` field description listing common property names like "locator, text_from_source, ..."). These are informational hints, not normative. They do not need to be updated every time a vocabulary entry is added.

### 5. Special Constraints
- Check for complex validation rules (patterns, minItems, anyOf, allOf, not)
- Verify these are documented in the markdown
- Look for undocumented constraints in schemas

### 6. Entity ID Patterns
- Verify entity ID pattern constraints match between docs and schemas
- Check that the pattern `^[a-zA-Z0-9-]{1,64}$` is consistently applied where needed

### 7. additionalProperties Severity
All top-level entity schemas and the archive root in `glx-file.schema.json` set `additionalProperties: false` on the entity objects. Some nested map fields (e.g., properties maps) intentionally use `additionalProperties: true` to allow arbitrary keys. For fields covered by `additionalProperties: false`, drift direction is critical:
- **Spec documents a field, schema missing it** → **CRITICAL** — `glx validate` will reject valid archives using that field (data loss risk)
- **Schema has a field, spec doesn't document it** → **MAJOR** — undocumented but functional, no data loss

### 8. `$ref` Resolution
- Verify all `$ref` values in schemas point to files that actually exist (e.g., `"$ref": "person.schema.json"`, `"$ref": "vocabularies/event-types.schema.json"`)
- Check for orphaned schemas — schema files with no `$ref` pointing to them and no corresponding spec section

### 9. File-Level Existence
- Every entity type documented in the spec should have a corresponding schema file
- Every schema file should have a corresponding spec section
- Every vocabulary `.glx` file should have a corresponding vocabulary schema
- Flag any mismatches in either direction

### 10. Special Focus Areas
These patterns are known to be complex and drift-prone:
- **Assertion mutual exclusivity** — `allOf`/`anyOf`/`not` constraints enforcing that `property`/`value` and `participant` are mutually exclusive
- **Participant object** — shared nested structure used by Event, Relationship, and Assertion; must stay consistent across all three schemas
- **Temporal property structure** — `value`/`date` object pattern for properties that change over time
- **Evidence requirement** — at least one of `citations`, `sources`, or `media` required on assertions
- **Generic `properties` field** — `additionalProperties: true` on entity properties maps (intentional, not drift)

## Cross-References

- If drift is found in a schema, it likely also affects Go code — see `/check-code-drift` for downstream impact
- For specification-internal issues (contradictions, ambiguity), use `/check-spec` instead
- Schema-related issues found by `/check-spec` should be redirected here

## Provenance

Before the findings, record a provenance header at the top of the report so a run is reproducible and silent coverage gaps are detectable:

- **Commit SHA** being checked — `git rev-parse HEAD`
- **Run timestamp** — `date -u +%Y-%m-%dT%H:%M:%SZ`
- **Schema files actually visited** — the concrete list of `*.schema.json` paths you compared, not just the file-list spec above. Listing what you visited (vs. what you were told to visit) catches the case where a file was silently dropped.
- **`make check-schemas` exit status** — run it and record the exit code. This target (`node specification/validate-schemas.mjs`) validates every entity and vocabulary `*.schema.json` against the JSON-Schema meta-schema, compiles each under ajv strict mode, and compiles `glx-file.schema.json` with all entity/vocabulary schemas registered as `$ref` targets (so a broken cross-schema reference fails here). It does **not** read or validate the `specification/5-standard-vocabularies/*.glx` template files — checking those `.glx` templates against their schemas is a manual step (see "Vocabulary Schemas" above) and the proposed delegation is the separate scope of #839. A **non-zero** status means schema-level validity is broken, so spec↔schema comparison on top of malformed schemas is unreliable — note this prominently and treat findings with lower confidence.

These values also populate the `commit`, `timestamp`, and `checked_schemas` fields of the machine-readable block below.

## Output Format

Produce the report in two parts, in this order: (1) a human-readable prose report grouped by scope, then (2) a single machine-readable `findings-json` block.

### Part 1 — Human-readable report

Group findings by scope using the **exact** heading for each scope, so every finding is addressable by `scope` + `target`. Emit one section per schema you actually visited (see "Schemas to Check" for the scope inventory). Do NOT invent section structure for the non-entity schemas — these four headings cover every schema in scope:

| Scope (`scope` enum value)                    | Heading to use                    |
|-----------------------------------------------|-----------------------------------|
| Entities (`entity`)                           | `## Entity: [name]`               |
| Archive root (`archive_root`)                 | `## Archive Root: glx-file`       |
| Type vocabularies (`vocabulary_type`)         | `## Vocabulary (type): [name]`    |
| Property vocabularies (`vocabulary_property`) | `## Vocabulary (property): [name]` |

Under each heading, emit either a no-drift line or a drift block:

```
## Entity: [name]

✅ No drift detected - Schema matches specification

OR

⚠️ Drift detected - Schema needs updates:

### Required Fields
- Schema missing required field `field_name` documented in specification
- Schema has undocumented required field `field_name` not in specification

### Optional Fields
- Schema missing optional field `field_name` documented in specification
- Schema has undocumented field `field_name` not in specification

### Field Types
- Schema has `field_name` as type X but specification documents it as type Y

### Descriptions
- Schema description for `field_name` adds or drops a constraint vs. the spec (per the two-tier rule)

### Constraints
- Specification documents constraint X but schema doesn't enforce it
- Schema enforces undocumented constraint X
```

**Remember**: Frame all drift as "what the schema needs to change" to match the specification.

### Part 2 — Machine-readable findings block

After the prose, append **exactly one** fenced block with the info-string `findings-json` so the eval harness (#796) can grep for it deterministically and compute precision/recall. It MUST be **valid JSON** — not pseudo-YAML, no comments, no trailing commas. Always emit it, even on a fully clean run (`"findings": []`).

Field contract:

- `command` — always `"check-schema-drift"`.
- `commit` / `timestamp` — copied from the Provenance header.
- `checked_schemas` — the schemas you **actually visited** this run, grouped by scope key (`entity`, `archive_root`, `vocabulary_type`, `vocabulary_property`). Populate it from the `*.schema.json` glob, not a memorized list — the schema set grows over time.
- `findings[]` — one object per drift:
  - `scope` — enum: `entity | archive_root | vocabulary_type | vocabulary_property | meta`. (`meta` is for file-level / cross-cutting findings such as an orphaned schema.)
  - `target` — schema name without extension (e.g., `person`, `glx-file`, `event-types`).
  - `schema_path` — path to the `.schema.json`.
  - `spec_path` — path to the governing spec markdown, or `null` for an orphaned schema with no spec.
  - `json_pointer` — JSON Pointer into the schema (e.g., `#/properties/notes`), or `null` if not field-scoped.
  - `category` — enum: `required_field | optional_field | field_type | description | constraint | entity_id_pattern | additional_properties | ref_resolution | file_existence | special_focus`.
  - `severity` — enum: `critical | major | minor | info` (the precise rubric is defined in #838).
  - `drift_direction` — enum: `spec_to_schema` (spec documents it, schema lacks it) | `schema_to_spec` (schema has it, spec doesn't). This is the same split section 7 uses to assign `additionalProperties: false` severity; surfacing it here makes the distinction queryable.
  - `message` — one sentence, framed as "what the schema needs to change".
- `summary` — `total_findings`, the four per-severity counts, and `suppressed_as_duplicate_of_known_issue` (incremented per the Cross-Reference section below).

Example (the `checked_schemas` lists and the single finding are **illustrative** — populate them from the schemas you actually visited; emit `"findings": []` when clean):

```findings-json
{
  "command": "check-schema-drift",
  "commit": "<HEAD SHA>",
  "timestamp": "<ISO 8601>",
  "checked_schemas": {
    "entity": ["assertion", "citation", "event", "media", "person", "place", "relationship", "repository", "source"],
    "archive_root": ["glx-file"],
    "vocabulary_type": ["event-types", "relationship-types", "place-types", "source-types", "media-types", "repository-types", "confidence-levels", "participant-roles", "sex-types", "gender-types"],
    "vocabulary_property": ["person-properties", "event-properties", "relationship-properties", "place-properties", "media-properties", "repository-properties", "source-properties", "citation-properties"]
  },
  "findings": [
    {
      "scope": "entity",
      "target": "person",
      "schema_path": "specification/schema/v1/person.schema.json",
      "spec_path": "specification/4-entity-types/person.md",
      "json_pointer": "#/properties/notes",
      "category": "field_type",
      "severity": "major",
      "drift_direction": "spec_to_schema",
      "message": "Spec marks `notes` as `string | string[]`; schema needs a `oneOf` to accept the array form."
    }
  ],
  "summary": {
    "total_findings": 1,
    "critical": 0,
    "major": 1,
    "minor": 0,
    "info": 0,
    "suppressed_as_duplicate_of_known_issue": 0
  }
}
```

## Summary

At the end, provide:
- Total schemas checked, broken down by scope (entities, archive root, type vocabularies, property vocabularies)
- Count of schemas with drift
- List of schemas that need updates to match specification
- Severity assessment (`critical | major | minor | info`)
- Recommended actions: "Update [schema files] to match specification"

These counts must agree with the `summary` block in the `findings-json` output.

## Notes

- **Specification is the source of truth** - schemas should be updated to match it
- Be thorough but practical - focus on structural and semantic differences that could cause confusion or validation issues
- If a field is marked as "required" but has a complex `anyOf` constraint, document this clearly
- Check both directions: specification → schema (missing in schema) AND schema → specification (undocumented in spec)
- **Description comparison**: apply the two-tier rule in "What to Check → 4. Field Descriptions" — flag only structural mismatches (a constraint added or omitted), never pure rewording. When unsure, treat as wording-only.

## Cross-Reference with Known Issues

Two layers suppress already-known drift; apply both before finalizing:

1. **File-backed allowlist (per-symbol).** The "Allowlisted Drift" section at the top of this command already consults `.claude/drift-allowlist.yaml` for known, triaged drift on a specific `file` + symbol. Those are suppressed there and need no further action here.
2. **Open GitHub issues (this section).** For drift that is *not* yet in the allowlist but may already be tracked as an open issue:
   1. Run `gh issue list -R genealogix/glx --state open --limit 100 --json number,title --jq '.[] | "#\(.number) \(.title)"'`.
   2. For each finding, scan titles for overlap on the schema file path or the field/entity name.
   3. If already tracked: omit it from the prose report and increment `suppressed_as_duplicate_of_known_issue` in the `findings-json` summary (do not also add it to `findings[]`).
   4. If NOT tracked: include it in full, in both the prose and `findings[]`.

This keeps the report focused on newly discovered drift. As `.claude/drift-allowlist.yaml` accumulates entries (#797), more drift is caught by layer 1, reducing the `gh issue list` scans in layer 2.
