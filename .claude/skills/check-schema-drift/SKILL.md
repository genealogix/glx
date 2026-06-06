---
name: check-schema-drift
description: Identify drift between specification markdown files and JSON schemas
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash(git rev-parse:*)
  - Bash(date -u:*)
  - Bash(make check-schemas:*)
  - Bash(gh issue list:*)
model: claude-opus-4-8
---

You are tasked with identifying any drift between the GLX specification markdown files and the JSON schemas.

## Allowlisted Drift (read this first)

Before analyzing, read **`.claude/drift-allowlist.yaml`** (validated by `.claude/drift-allowlist.schema.json`) — the single source of truth for known, triaged drift, shared with `/check-code-drift`. If a finding concerns the same `file` and symbol as an entry (match on symbol identity, not exact string), suppress it: `permanent: true` is by-design (not drift); an entry with a `tracking_issue` is a temporary deferral (refer to the issue rather than re-reporting). Everything not in the allowlist is reported normally. The allowlist holds only per-symbol exceptions — not class-level methodology — and is human-curated, so don't invent entries.

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

1. **specification/4-entity-types/*.md** — The source of truth (human-readable specification)
2. **specification/schema/v1/*.schema.json** — Derived schemas (machine-readable)

## Schemas to Enumerate (Dynamic — Do Not Hardcode)

**Do not use a static list of schema names.** Instead, enumerate at scan time:

- **Entity schemas**: glob `specification/schema/v1/*.schema.json`, then exclude `glx-file.schema.json` and any file that does not end in `.schema.json`. Each remaining file is an entity schema.
- **Archive root**: `specification/schema/v1/glx-file.schema.json` is treated separately as the archive root.
- **Vocabulary schemas**: glob `specification/schema/v1/vocabularies/*.schema.json`. Classify each by naming convention:
  - Files ending in `-properties.schema.json` → property vocabulary
  - All others → type vocabulary
- **Match schema to spec by naming convention**: `person.schema.json` ↔ `specification/4-entity-types/person.md`; `event-types.schema.json` ↔ `specification/5-standard-vocabularies/event-types.glx`; etc.

This approach automatically covers schemas added in the future and eliminates the stale-list problem documented in issue #986 (where `study`, `research-log`, and five type vocabularies were silently never checked because they were missing from a static list).

### Files to Ignore

Always use the glob patterns above rather than bare directory listing. If you do list the directory, exclude these explicitly — they are not subject to drift analysis:

- `specification/schema/v1/embed.go` — Go embed file for binary distribution
- `specification/5-standard-vocabularies/embed.go`
- Any `README.md` in a schema directory
- Any file not ending in `.schema.json`

Comparing `embed.go` or a README against a non-existent spec section produces pure false positives. Do not flag them.

### Archive Root Check

For `glx-file.schema.json`, compare against `specification/3-archive-organization.md`. Verify:
- `metadata` object fields match spec
- Every entity map section in the schema is present in the spec, and vice versa — do not assert a fixed count, derive it from the glob
- All vocabulary map sections are present
- Entity ID pattern (`^[a-zA-Z0-9-]{1,64}$`) matches spec
- Vocabulary key pattern matches spec

### Vocabulary Check

Compare `specification/4-entity-types/vocabularies.md` and `specification/5-standard-vocabularies/*.glx` against `specification/schema/v1/vocabularies/*.schema.json`. For each vocabulary schema, verify:
- Schema structure matches the vocabulary format documented in `vocabularies.md` (label, description, gedcom, fields, etc.)
- Required fields in the schema match what the spec says is required for vocabulary entries
- For property vocabulary schemas specifically, property definition schemas enforce `value_type`/`reference_type`/`vocabulary_type` mutual exclusivity per spec
- All vocabulary `.glx` template files conform to their corresponding schema. **On this branch** `make check-schemas` validates the JSON *schemas* only — `.glx` template validation lands with #839 (PR #1018). Until it does, treat this bullet as an LLM-simulated structural check (`llm_only: true`, conservative severity), not a tool-backed one

## What to Check

For each entity type, verify:

### 1. Field Presence

Compare fields documented in the spec markdown against `properties` in the JSON schema. Do this in both directions: spec→schema (missing in schema) and schema→spec (undocumented in spec).

When a schema has `additionalProperties: false` (all top-level entity schemas and the archive root), drift direction determines severity — see the Severity Rubric section below.

**EXCLUDE rows whose field name reads "Entity ID (map key)"** — that row documents the parent-map keying constraint (the entity is keyed by its ID in the parent map, enforced via `patternProperties` in `glx-file.schema.json`), not a property on the entity itself. The entity schema's `required` array must not contain it, so flagging a missing "Entity ID (map key)" is a pure false positive on every entity. Do not flag it.

Category: `field_presence`

### 2. Required vs Optional

Compare "Required Fields" tables in markdown with `required` arrays in JSON schema. A field being required vs optional has different failure modes in each direction.

Category: `required_optional`

### 3. Field Types

Compare field types in markdown tables with JSON schema types. Check for type mismatches (e.g., markdown says "string", schema says "array").

Note: `oneOf[string, array]` in the schema where the spec says `string | string[]` is a known idiom and is not drift — treat as `info`.

Category: `field_type`

### 4. Field Descriptions

Use a **two-tier rule** — paraphrase scored as drift is the dominant false-positive source in description comparison.

1. **Structural mismatch (flag)** — the schema description states a constraint **not** in the spec (e.g., "must be unique" appears only in the schema), OR **omits** a constraint the spec documents (e.g., spec says "ISO 8601 date", schema says only "date"). A constraint is anything that changes what values validate or how a consumer must treat the field.
2. **Wording-only (do NOT flag)** — pure rewording, expanded examples, or added length when no constraint changes. Natural-language paraphrase is not drift.

If unsure which tier applies, treat it as wording-only.

**IGNORE** inline lists of vocabulary-defined values in description strings (e.g., a `properties` field description listing common property names like "locator, text_from_source, ..."). These are informational hints, not normative.

Category: `field_type` (constraint differences surfaced through descriptions)

### 5. Special Constraints

Check for complex validation rules (`patterns`, `minItems`, `anyOf`, `allOf`, `not`). Verify these are documented in the markdown. Look for undocumented constraints in schemas.

Category: `constraint`

### 6. Entity ID Patterns

Verify entity ID pattern constraints match between docs and schemas. Check that `^[a-zA-Z0-9-]{1,64}$` is consistently applied where needed.

Category: `entity_id_pattern`

### 7. `$ref` Resolution

Verify all `$ref` values in schemas point to files that actually exist (e.g., `"$ref": "person.schema.json"`, `"$ref": "vocabularies/event-types.schema.json"`). Check for orphaned schemas — schema files with no `$ref` pointing to them and no corresponding spec section.

Category: `ref_type`

### 8. File-Level Existence

Every entity type documented in the spec should have a corresponding schema file. Every schema file should have a corresponding spec section. Every vocabulary `.glx` file should have a corresponding vocabulary schema. Flag any mismatches in either direction.

Category: `file_existence`

### 9. Special Focus Areas

These patterns are known to be complex and drift-prone:

- **Assertion mutual exclusivity** — `allOf`/`anyOf`/`not` constraints enforcing that `property`/`value` and `participant` are mutually exclusive
- **Participant object** — shared nested structure used by Event, Relationship, and Assertion; must stay consistent across all three schemas
- **Temporal property structure** — `value`/`date` object pattern for properties that change over time
- **Evidence requirement** — at least one of `citations`, `sources`, or `media` required on assertions
- **Generic `properties` field** — `additionalProperties: true` on entity properties maps (intentional, not drift)

Category: `special_focus`

## Severity Rubric

Severity levels (`critical | major | minor | info`) are defined in the shared rubric at `.claude/skills/check-suite/severity-rubric.md`. Reference it — do not redefine the four levels here. The table below maps each finding category for this skill to its default severity, implementing the proposal in GitHub issue #838.

| Category | Condition | Severity |
|---|---|---|
| `field_presence` | Spec documents a field; schema missing it; entity has `additionalProperties: false` | **critical** — `glx validate` rejects spec-valid archives |
| `field_presence` | Spec documents a field; schema missing it; entity uses `additionalProperties: true` (e.g., a properties map) | **info** — vocab-defined, no data loss |
| `field_presence` | Schema has a field; spec does not document it | **major** — functional but undocumented |
| `required_optional` | Schema marks a field required; spec marks it optional | **critical** — `glx validate` rejects spec-valid archives |
| `required_optional` | Schema marks a field optional; spec marks it required | **major** — spec contract violated, but `glx validate` permits it |
| `field_type` | Type mismatch that affects validation (e.g., spec says `string`, schema says `array`) | **critical** |
| `field_type` | Compatible widening — `oneOf[string, array]` in schema where spec says `string \| string[]` | **info** — known idiom, not drift |
| `field_type` | Schema description adds or removes a constraint vs. spec | **major** |
| `field_type` | Pure rewording or added example, no constraint change | **info** — do NOT flag |
| `entity_id_pattern` | Pattern in schema (`^[a-zA-Z0-9-]{1,64}$`) differs from spec | **critical** — affects identifier validity |
| `entity_id_pattern` | Pattern constraint present in schema, no spec mention | **major** — undocumented constraint |
| `ref_type` | `$ref` points to a non-existent file | **critical** — schema compilation fails |
| `ref_type` | Orphaned schema file: no `$ref` points to it, no spec section | **major** — confirm intentional |
| `constraint` | `allOf`/`anyOf`/`not` in schema; not documented in spec | **major** |
| `constraint` | Spec documents a constraint; schema does not enforce it | **critical** — silent data acceptance |
| `constraint` | Assertion mutual exclusivity (`property`/`value` ↔ `participant`) missing or mis-shaped | **critical** — semantic data corruption |
| `constraint` | Evidence requirement on assertion (`anyOf [citations, sources, media]`) missing | **critical** |
| `constraint` | Entity properties map uses `additionalProperties: true` | **info** — intentional idiom |
| `constraint` | Entity object uses `additionalProperties: true` where spec implies `false` | **major** |
| `special_focus` | Participant object inconsistent across Event/Relationship/Assertion schemas | **critical** — cross-entity inconsistency |
| `special_focus` | Temporal property structure (`value`/`date` object) missing or mismatched | **major** |
| `file_existence` | Spec-documented entity has no schema file | **critical** |
| `file_existence` | Schema file has no corresponding spec section | **major** |
| `file_existence` | Vocabulary `.glx` template fails validation against its corresponding schema | **critical** — shipped example archives would fail validation |
| `file_existence` | Vocabulary schema file has no corresponding `.glx` template | **minor** |
| `vocabulary` | Vocabulary schema structure deviates from the format in `vocabularies.md` | **major** |
| `vocabulary` | Property vocabulary schema does not enforce `value_type`/`reference_type`/`vocabulary_type` mutual exclusivity | **critical** |

When uncertain, assign `info`. A false `critical` costs more reviewer trust than a missed `minor`.

## `validator_caught` / `llm_only` Assignment

These two fields are required on every finding and must be logical inverses of each other.

- `validator_caught: true, llm_only: false` — the finding is already surfaced by a deterministic tool: `make check-schemas` (validate-schemas.mjs), `glx validate`, or a CI check. Broken `$ref` values and schema-invalidity-against-meta-schema fall here.
- `validator_caught: false, llm_only: true` — the finding comes from semantic comparison of spec prose vs. schema structure. All spec↔schema field/description/constraint comparisons fall here.

Wire this to the provenance `make check-schemas` run: if that run exits non-zero, any finding attributable to a broken `$ref` or invalid schema compilation should be marked `validator_caught: true`.

## Provenance

Before the findings, record a provenance header at the top of the report so a run is reproducible and silent coverage gaps are detectable:

- **Commit SHA** being checked — `git rev-parse HEAD`
- **Run timestamp** — `date -u +%Y-%m-%dT%H:%M:%SZ`
- **Schema files actually visited** — the concrete list of `*.schema.json` paths you compared (derived from the glob at scan time, not a memorized list). Listing what you visited versus what you were told to visit catches the case where a file was silently dropped.
- **`make check-schemas` exit status** — run it and record the exit code. This target (`node specification/validate-schemas.mjs`) validates every entity and vocabulary `*.schema.json` against the JSON Schema meta-schema, compiles each under ajv strict mode, and compiles `glx-file.schema.json` with all entity/vocabulary schemas registered as `$ref` targets (so a broken cross-schema reference fails here). It does **not** read or validate the `specification/5-standard-vocabularies/*.glx` template files — checking those `.glx` templates against their schemas is a separate manual step. A **non-zero** status means schema-level validity is broken; note this prominently and treat findings with lower confidence.

The commit SHA and schema file list also populate the `commit` and `checked_files` fields of the machine-readable block below.

## Output Format

Produce the report in two parts, in this order: (1) a human-readable prose report grouped by scope, then (2) a single machine-readable `findings-json` block.

### Part 1 — Human-readable report

Group findings using the exact headings below, so every finding is addressable by scope and target. Emit one section per schema you actually visited (derived from the glob).

| Scope | Heading to use |
|---|---|
| Entity schemas | `## Entity: [name]` |
| Archive root | `## Archive Root: glx-file` |
| Type vocabularies | `## Vocabulary (type): [name]` |
| Property vocabularies | `## Vocabulary (property): [name]` |

Under each heading, emit either a no-drift line or a drift block:

```
## Entity: [name]

No drift detected — schema matches specification.

OR

Drift detected — schema needs updates:

### Field Presence
- Schema is missing field `field_name` documented in specification (spec→schema).
- Schema has field `field_name` not documented in specification (schema→spec).

### Required vs Optional
- Schema marks `field_name` as required; specification marks it optional.

### Field Types
- Schema has `field_name` typed as X; specification documents it as type Y.

### Constraints
- Specification documents constraint X but schema does not enforce it (spec→schema).
- Schema enforces undocumented constraint X (schema→spec).
```

Frame all drift as "what the schema needs to change" to match the specification.

### Part 2 — Machine-readable findings block

After the prose, append **exactly one** fenced code block with the info-string `findings-json` so the eval harness (#796) can grep for it deterministically. The block MUST be **valid JSON** — no comments, no trailing commas. Always emit it, even on a fully clean run (`"findings": []`).

The block MUST conform to `.claude/skills/check-suite/findings.schema.json`. The schema uses `additionalProperties: false` at both the top level and per finding, so any unrecognized field is a validation failure.

**Top-level fields:**
- `command` — always `"check-schema-drift"`.
- `commit` — HEAD SHA from provenance (7–40 hex chars).
- `checked_files` — flat array of repo-relative paths actually examined this run: all `.schema.json` paths opened plus the spec `.md` files and any `.glx` templates you read. Derived from the glob, not a memorized list.
- `findings` — array of finding objects (empty array on a clean run).
- `positive_notes` — optional array of strings for things verified as correctly aligned. Route no-drift confirmations here rather than as `info` findings.
- `summary` — `critical`, `major`, `minor`, `info`, and `suppressed_as_duplicate_of_known_issue` (integer counts; all five are required by the shared schema).

**Per-finding required fields** — these are the only fields the schema accepts on a finding:
- `file` — repo-relative path of the drifting artifact (the `.schema.json`).
- `line` — 1-based line in `file`, or `null` when not line-addressable.
- `severity` — one of `critical | major | minor | info` per the rubric table above.
- `category` — one of: `field_presence | required_optional | field_type | entity_id_pattern | ref_type | constraint | special_focus | file_existence | vocabulary`. Must match the category used to look up severity in the rubric table.
- `message` — one sentence framed as "what the schema needs to change." Include drift direction in the message text (e.g., "Spec documents field X; schema is missing it" vs. "Schema has field X not documented in spec").
- `fix` — optional concrete remediation.
- `validator_caught` — `true` if `make check-schemas` or another deterministic tool already flags this; `false` otherwise.
- `llm_only` — logical inverse of `validator_caught`.

Note: the shared schema has no `scope`, `target`, `drift_direction`, `spec_path`, or `json_pointer` fields. Convey that information in `message` and `fix` instead.

Example (illustrative — populate from actual glob and analysis; emit `"findings": []` on a clean run):

```findings-json
{
  "command": "check-schema-drift",
  "commit": "8709e09",
  "checked_files": [
    "specification/schema/v1/person.schema.json",
    "specification/4-entity-types/person.md",
    "specification/schema/v1/citation.schema.json",
    "specification/4-entity-types/citation.md"
  ],
  "findings": [
    {
      "file": "specification/schema/v1/person.schema.json",
      "line": null,
      "severity": "critical",
      "category": "field_presence",
      "message": "Spec documents field `aliases` on person (spec→schema); schema is missing it and sets additionalProperties: false, so glx validate rejects archives using this field.",
      "fix": "Add `aliases` to the properties object in person.schema.json.",
      "validator_caught": false,
      "llm_only": true
    }
  ],
  "positive_notes": [
    "citation.schema.json: all required and optional fields match specification."
  ],
  "summary": {
    "critical": 1,
    "major": 0,
    "minor": 0,
    "info": 0,
    "suppressed_as_duplicate_of_known_issue": 0
  }
}
```

## Cross-References

- If drift is found in a schema, it likely also affects Go code — see `/check-code-drift` for downstream impact.
- For specification-internal issues (contradictions, ambiguity), use `/check-spec` instead.
- Schema-related issues found by `/check-spec` should be redirected here.

## Cross-Reference with Known Issues

Two layers suppress already-known drift; apply both before finalizing:

1. **File-backed allowlist (per-symbol).** The "Allowlisted Drift" section at the top of this skill already consults `.claude/drift-allowlist.yaml` for known, triaged drift on a specific `file` + symbol. Those are suppressed there and need no further action here.
2. **Open GitHub issues (this section).** For drift that is not yet in the allowlist but may already be tracked as an open issue:
   1. Run `gh issue list -R genealogix/glx --state open --limit 100 --json number,title --jq '.[] | "#\(.number) \(.title)"'`.
   2. For each finding, scan titles for overlap on the schema file path or the field/entity name.
   3. If already tracked: omit it from the prose report and increment `suppressed_as_duplicate_of_known_issue` in the `findings-json` summary (do not also add it to `findings[]`).
   4. If NOT tracked: include it in full, in both the prose and `findings[]`.

This keeps the report focused on newly discovered drift. As `.claude/drift-allowlist.yaml` accumulates entries (#797), more drift is caught by layer 1, reducing the `gh issue list` scans in layer 2.

## Summary Section

At the end of the prose report, provide:
- Total schemas checked, broken down by scope (entity schemas, archive root, type vocabularies, property vocabularies), with counts derived from what the glob returned at scan time
- Count of schemas with drift
- List of schemas that need updates to match specification
- Highest severity present
- Recommended actions: "Update [schema files] to match specification"

These counts must agree with the `summary` block in the `findings-json` output.

## Notes

- **Specification is the source of truth** — schemas should be updated to match it, not the other way around
- Be thorough but practical — focus on structural and semantic differences that could cause confusion or validation issues
- **Description comparison**: apply the two-tier rule — flag only structural mismatches (a constraint added or omitted), never pure rewording. When unsure, treat as wording-only
- Check both directions: specification → schema (missing in schema) AND schema → specification (undocumented in spec)
- If a field is marked as "required" but has a complex `anyOf` constraint, document this clearly
- **Generic `properties` fields** on entities use `additionalProperties: true` intentionally — this is not drift
