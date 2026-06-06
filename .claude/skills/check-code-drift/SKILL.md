---
name: check-code-drift
description: Detect drift between go-glx Go types and JSON schemas
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash(git rev-parse:*)
  - Bash(git diff:*)
  - Bash(grep:*)
  - Bash(rg:*)
  - Bash(make check-code-drift:*)
  - Bash(make check-drift-allowlist:*)
model: claude-opus-4-8
---

<!-- last-verified: 8709e09 -->

# check-code-drift

Identify drift between the GLX Go implementation and JSON schemas/specification. Emit one `findings-json` block per run, conforming to `.claude/skills/check-suite/findings.schema.json`. Severity levels follow `.claude/skills/check-suite/severity-rubric.md` — default to `info` when uncertain.

## Relationship to the deterministic checker

`make check-code-drift` (`tools/driftcheck`, #673) runs in CI and mechanically catches structural drift: missing/extra fields, yaml-tag mismatches, `omitempty`-vs-`required` mismatches, and type-family mismatches. Those are already covered. Focus on what the deterministic tool cannot decide: semantic/validation drift (section 9), GEDCOM converter round-trip drift (section 11), documentation/comment mismatches, and structural cases the tool deliberately skips (value constraints like `minLength`/`pattern`/`enum`). Both tools read the same allowlist.

## Step 0 — Capture run context

Before anything else, run:

```bash
git rev-parse HEAD
```

Record the short SHA. It goes in the `commit` field of the `findings-json` block.

## Step 1 — Scope (incremental by default, --full override)

Unless the user passes `--full`:

```bash
git diff --name-only origin/main...HEAD -- 'go-glx/*.go' 'specification/schema/v1/**/*.json'
```

Check only the entities/vocabularies touched by the changed files. If nothing is returned (clean branch or no relevant changes), fall back to full scope. With `--full`, always check every entity and vocabulary.

## Step 2 — Read allowlist

Read `.claude/drift-allowlist.yaml` (validated by `.claude/drift-allowlist.schema.json`). It is the single source of truth for known, per-symbol drift already triaged.

- If a finding's `file` and Go symbol match an allowlist entry, suppress it. Match on symbol identity, not exact string: an entry written as `GLXFile.ImportMetadata` matches a finding naming the field bare (`ImportMetadata`) or by yaml tag (`metadata`).
- `permanent: true` entries are by-design — treat them as not drift.
- Entries with `tracking_issue` are temporary deferrals — don't re-report; note the tracking issue.
- Do not invent allowlist entries. If a new exception seems warranted, report the drift and recommend adding an entry.

## Step 3 — Discover entity and vocabulary types dynamically

Read `go-glx/types.go`. Find the `GLXFile` struct. For every field of type `map[string]*X` where X is any struct, treat that as one type-to-check. The yaml tag on the `GLXFile` field is the schema property name; the pointee struct is the Go side. Do not use a hard-coded entity list — this auto-tracks every type added in the future.

As of the last-verified SHA above, `GLXFile` contains:
- Entity maps: `persons`, `relationships`, `events`, `places`, `sources`, `citations`, `repositories`, `assertions`, `media`, `research_logs`, `studies`
- VocabularyEntry maps: `event_types`, `participant_roles`, `confidence_levels`, `relationship_types`, `place_types`, `source_types`, `repository_types`, `media_types`, `sex_types`, `gender_types`, `search_result_types`, `research_log_status_types`, `study_types`, `study_statuses`, `legal_statuses`
- PropertyDefinition maps: `person_properties`, `event_properties`, `relationship_properties`, `place_properties`, `media_properties`, `repository_properties`, `citation_properties`, `source_properties`

This list is illustrative. Always derive it from the live file; the definitive list is whatever `GLXFile` contains at HEAD.

## Step 4 — Deterministic pre-check per entity

For each entity in scope, before invoking LLM reasoning:

1. Count the fields in the Go struct.
2. Count the properties in the corresponding JSON schema.
3. Check that every yaml tag in the Go struct appears as a property name in the schema.

If field counts match AND all yaml tags match property names, record the entity as "no structural drift detected" and skip deep LLM analysis for it. Only apply full LLM reasoning to entities where a count or tag mismatch is found, or where the entity type is `Assertion` or involves custom marshaling.

## Step 5 — Per-entity analysis (fan-out)

For each entity that requires full analysis, spawn one read-only subagent. Each subagent receives:
- The relevant Go struct excerpt from `go-glx/types.go`
- The corresponding JSON schema (`specification/schema/v1/<entity>.schema.json`)
- The relevant section from `specification/4-entity-types/<entity>.md`
- The relevant section from `go-glx/validation.go` (and `go-glx/validation_temporal.go` if needed)
- The allowlist (from step 2)
- These instructions for what to check

The parent agent merges all subagent findings and deduplicates.

## Source of truth flow

```
Specification (*.md) → Schema (*.schema.json) → Go Code (types.go)
     SOURCE OF TRUTH         DERIVED FROM SPEC      DERIVED FROM SCHEMA
```

Frame struct/field drift as "the Go code needs to change to match the schema." Frame validation drift bidirectionally — both code and spec may need updates.

## What to check per entity

### 1. Field presence

- All fields in JSON schema `properties` exist in the Go struct.
- All fields in the Go struct exist in the JSON schema (except internal fields like `validation *ValidationResult`).

### 2. Field types

Map JSON schema types to Go types:

| Schema type | Expected Go type |
|---|---|
| `string` | `string` |
| `array` of strings | `[]string` |
| `array` of objects | `[]StructType` |
| `object` | struct or `map[string]any` |
| `number` | `float64` or `*float64` |
| `boolean` (required) | `bool` |
| `boolean` (optional, tri-state) | `*bool` — `nil` means omitted, which is distinct from `false`; `PropertyDefinition.Temporal` and `PropertyDefinition.MultiValue` use this pattern intentionally |
| `oneOf: [string, array of strings]` | `NoteList` with custom YAML marshal/unmarshal |
| `string` for dates | `DateString` (type alias) |
| nested `object` (optional) | `*StructType` pointer |

### 3. Required vs optional

- Schema-required fields must NOT have `omitempty`.
- Schema-optional fields should have `omitempty`.
- Pointer types signal truly optional fields (e.g., `*float64` for latitude/longitude).

### 4. YAML tag names

- Go `yaml:"field_name"` must match the JSON schema property name exactly.
- Check for snake_case vs camelCase mismatches.

### 5. Reference types

- Reference fields must carry a `refType:"<target>"` struct tag.
- Two distinct reference patterns exist — apply the correct one:
  - **Entity ID pattern** (for `persons`, `events`, `places`, etc.): `^[a-zA-Z0-9-]{1,64}$` — hyphens allowed, no underscores, max 64 chars.
  - **Vocabulary key pattern** (for `event_types`, `participant_roles`, etc.): `^[a-zA-Z0-9_-]+$` — underscores also allowed, no length cap.
  Do not apply the entity-ID pattern to vocabulary `refType` targets, or vice versa.

### 6. Nested types

- Check `Participant` struct (used by Event, Relationship, and Assertion). Verify it matches the schema's object definition and its required fields.
- Check `EntityRef` struct (used by Assertion.Subject and ResearchLog.Subject).

### 7. Special cases

#### Assertion entity

- Verify mutually exclusive fields: `property`/`participant`, `value`/`participant`.
- Check that the `anyOf: [sources, citations, media]` constraint is handled.
- Verify `subject` allows multiple entity types via `oneOf`.

#### EntityRef

- Mutually exclusive fields: Person, Event, Relationship, Place.
- Schema enforces via `oneOf` — exactly one field must be set.
- All fields have `omitempty` — Go serialization produces correct YAML.

#### NoteList

- Go type: `NoteList` (alias for `[]string`) with custom YAML marshal/unmarshal.
- Schema: `oneOf: [{type: string}, {type: array, items: {type: string}}]`
- Single note marshals as plain string; multiple as array.
- Present in **11** locations in `types.go`: 9 entities (Person, Relationship, Event, Place, Source, Citation, Repository, Assertion, Media) + Participant + Metadata.

#### Properties field

- All entity types **except Assertion** have `properties map[string]any` with `omitempty`, documented as "Vocabulary-defined properties."
- Assertion carries `property`/`value`/`participant` (mutually exclusive) instead of a `properties` map.

#### Metadata and Submitter

- `Metadata` struct — check against `glx-file.schema.json` property `metadata`.
- `Submitter` is nested via `*Submitter` pointer — check against the schema's submitter object definition.

#### GLXFile top-level

- All entity type maps and vocabulary maps must be present.
- yaml tags must match schema (e.g., `persons`, `events`).
- `ImportMetadata *Metadata` field has yaml tag `metadata` — the Go field name vs yaml-tag difference is allowlisted.

### 8. Vocabulary struct types

All type-vocabulary schemas share `VocabularyEntry`. Its fields are the superset across all vocabularies; unused fields are elided via `omitempty`. The YAML field order `label, description, category, applies_to, mime_type, gedcom` is load-bearing — guarded by `TestVocabularyEntryYAMLFieldOrder`. Each `GLXFile` vocabulary map must use `map[string]*VocabularyEntry`.

Also check:
- `FieldDefinition` struct (Label, Description, ValueType) against property vocabulary schemas.
- `PropertyDefinition` struct against all property vocabulary schemas. Note its `Temporal` and `MultiValue` fields are `*bool` by design (tri-state: true, false, nil-omitted).
- `go-glx/constants.go` for vocabulary constant coverage.

### 9. Validation logic and constraints

**Two-layer validation architecture:**

Pass 1 — JSON schema validation (`glx/validator.go` → `ValidateGLXFileStructure`): enforces `required`, `minLength`/`minItems`/`minimum`/`maximum`, `allOf`/`anyOf`/`not`, `additionalProperties: false`, `pattern`, `format`.

Pass 2 — Go cross-reference validation (`go-glx/validation.go` → `archive.Validate()`): handles what JSON schema cannot:
1. Entity and vocabulary reference existence (does the referenced ID actually exist?)
2. Place hierarchy cycle detection
3. Property vocabulary and value type validation (one phase: `validateAllProperties` handles both)
4. Entity field format validation — dates, URIs, and other formats (broader than dates alone; handled by `validateEntityFieldFormats`)
5. Temporal property structure validation

Do NOT flag constraints already enforced by the JSON schema as missing from Go code. Only flag validation gaps where NEITHER layer covers a specification requirement.

Also check the reverse: validation logic in Go code not documented in the specification.

### 10. GEDCOM converter drift

The GEDCOM converter is the I/O boundary to external genealogy software — field drift causes silent round-trip data loss. Importer: `gedcom_<entity>.go`. Exporter: `gedcom_export_<entity>.go`. Exceptions:
- Person: `gedcom_individual.go` / `gedcom_export_person.go`
- Relationship: `gedcom_family.go` / `gedcom_export_family.go`
- Citation, Assertion, source-citation chains: `gedcom_evidence.go`
- Event: nested in Person/Relationship importers (no dedicated file)

Search all `go-glx/gedcom_*.go` (including helpers and `_test.go` goldens), case-insensitive:

- Renamed field: search for old name — any active read/write is drift.
- Removed field: search for field name — active read/write is drift; an `assert.NotEqual` or "should be skipped" guard assertion is the post-cleanup state, not drift.
- Added field: zero matches means the GEDCOM layer ignores the field. Some GLX fields legitimately have no GEDCOM tag — ask "is this intentionally not exported?"

## Calibration examples (true positive / true negative)

**True positive — critical:** Go struct has `State string \`yaml:"state"\`` but schema property is `state_province`. Tag mismatch that breaks marshaling. Report it.

**True positive — major:** Schema property `language` is optional (not in `required` array) but Go field has no `omitempty`. Serialization produces the field even when empty. Report it.

**True negative — suppress:** `GLXFile.ImportMetadata` has Go name `ImportMetadata` but yaml tag `metadata`. This is in the allowlist as a permanent by-design difference. Do not report it.

**True negative — suppress:** `PropertyDefinition.Temporal` is `*bool` while the schema type is `boolean`. This is intentional tri-state semantics (see section 2 above). Do not report it as a type mismatch.

**Uncertain finding — default to info:** Go comment says "maximum 64 characters" but schema has no `maxLength` constraint visible in the excerpt you have. Without seeing the full schema, assign `info` rather than guessing `major`.

## Severity category table

| Category | Condition | Severity |
|---|---|---|
| `field_presence` | Required schema field has no Go counterpart | critical |
| `field_presence` | Go field has no schema counterpart; schema has `additionalProperties: false` | critical |
| `field_presence` | Optional schema field has no Go counterpart | major |
| `field_presence` | Internal Go field not in schema (e.g., `validation *ValidationResult`) | info |
| `required_optional` | Schema-required field has `omitempty` in Go | critical |
| `required_optional` | Schema-optional field missing `omitempty` in Go | major |
| `field_type` | Type mismatch that breaks marshaling (`string` vs `[]string`) | critical |
| `field_type` | Type widening covered by custom marshaler (`NoteList` vs `oneOf[string,array]`) | info |
| `yaml_tag` | Tag name mismatches schema property name | critical |
| `ref_type` | Missing `refType` tag on a reference field | major |
| `ref_type` | Wrong `refType` target (e.g., `persons` vs `events`) | critical |
| `gedcom_converter` | Removed schema field still read/written by `gedcom_*.go` — silent data loss | critical |
| `gedcom_converter` | Renamed schema field still read/written under old name | critical |
| `gedcom_converter` | Added schema field with zero `gedcom_*.go` references — verify intentionally not exported | info |
| `validation` | Spec requires constraint neither JSON schema nor Go enforces | major |
| `validation` | Go enforces constraint not documented in spec | minor |
| `documentation` | Go comment doesn't match schema description | info |

When uncertain about category or severity, use `info`. A false `critical` costs more reviewer trust than a missed `minor`.

## Output format

### Human-readable section

For each entity type, emit a brief section:

```
## [Entity Type]

No structural drift detected — counts match, all yaml tags align.

OR

Drift detected — see findings-json below.
```

Findings in the human-readable section should include severity, category, a one-sentence description, and a Fix line.

### Machine-readable findings-json block

At the end of your report, emit exactly one fenced code block with info-string `findings-json`. The block must be valid JSON conforming to `.claude/skills/check-suite/findings.schema.json`.

Every finding must have `file`, `line`, `severity`, `category`, `message`, `validator_caught`, `llm_only`, and `location`. The `location` object must include `go_line` and `schema_line` (integers or null). Drop any finding you cannot assign a line number to rather than emitting it as low-confidence noise — uncertain findings without coordinates cost more reviewer trust than they pay.

The `telemetry` block is injected by the runner (Claude Code harness), not self-reported. Leave it absent; the runner will add it.

```findings-json
{
  "command": "check-code-drift",
  "commit": "<7-char SHA from git rev-parse HEAD>",
  "checked_files": [
    "go-glx/types.go",
    "specification/schema/v1/person.schema.json",
    "..."
  ],
  "findings": [
    {
      "file": "go-glx/types.go",
      "line": 152,
      "severity": "critical",
      "category": "yaml_tag",
      "message": "Go field State has yaml tag `state` but schema property is `state_province`.",
      "fix": "Change yaml tag to `yaml:\"state_province,omitempty\"`.",
      "validator_caught": false,
      "llm_only": true,
      "location": {
        "go_line": 152,
        "schema_line": 34
      }
    }
  ],
  "positive_notes": [
    "Person struct: all 4 fields present, yaml tags match, required/optional alignment correct."
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

### Output discipline (terse instructions, native structured outputs)

Keep reasoning terse and deterministic: when a comparison is ambiguous, prefer `info` with a short factual `message` over speculative prose — overlong chain-of-thought increases false positives here.

When this skill runs through the eval harness or any runner that supports it, emit `findings-json` via the model's **native structured-output mode** (`response_format` with a `json_schema` compiled from `.claude/skills/check-suite/findings.schema.json`) rather than free-form "return JSON" generation — the schema exists for exactly this purpose and makes schema-violating output impossible. If native structured output is unavailable, fall back to emitting the fenced block by hand and validate it against the schema before finishing.

### Closing sentinel

End your report with one of these two lines — nothing after it:

```
END_OF_DRIFT_REPORT (entities checked: N, findings: M)
```

or, if you hit a context limit before completing all entities:

```
REPORT_TRUNCATED (last entity completed: <EntityName>)
```

The runner uses the sentinel to detect truncation and restart on the remaining scope.

## Notes

- Schema is the source of truth. Go code is updated to match it, not the reverse.
- Two-layer validation: JSON schema (pass 1) handles structural constraints; Go validator (pass 2) handles cross-references and semantic checks. Do not flag schema-covered constraints as missing from Go code.
- Internal fields (`validation *ValidationResult` in `GLXFile`) are expected to be absent from schemas.
- Comment differences are informational only.
- Check both directions: schema → Go (missing in Go) and Go → schema (not in schema, may need removal).
