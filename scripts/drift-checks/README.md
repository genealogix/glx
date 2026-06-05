# Deterministic drift checks (CI, no LLM)

Implementation spec for **PR 1 of epic #1016** — the deterministic half of the drift strategy.

**No LLMs run in CI.** CI catches the mechanical, decidable drift classes here; the on-demand `check-*` skills (PR 2) handle the semantic remainder locally.

## Pieces

### 1. `glx validate --stdin --entity-type` (#910) — Go
Enabler for the skills' "defer to deterministic tooling" pre-flight. Adds two flags to the `validate` command so a single entity snippet can be validated from stdin without writing a temp file.
- **Where:** `glx/validation_runner.go` (thin cobra wrapper) + `go-glx` validation entry (pure, no I/O).
- **Done when:** `echo '<yaml>' | glx validate --stdin --entity-type person` validates one entity and exits non-zero on failure. Unit + E2E test.

### 2. AST field extractor (#795) — Go
Deterministic extraction of `(field name, yaml tag, line)` from `go-glx/types.go` entity structs, via `go/ast` + `go/parser`. Feeds (a) the spec↔schema parser and (b) the skills' mandatory `file:line` (#676 item 11).
- **Where:** `go-glx/internal/structdump/` (library package, importable by tests and the CLI).
- **Done when:** extractor returns every `map[string]*X` entity/vocabulary field of `GLXFile` with struct field names, yaml tags, and source lines. Table test.

### 3. Spec ↔ schema parity (#309) — Node, **warn-first**
Parses the `| Field | Type | ... |` tables in `specification/4-entity-types/*.md` and compares field names / required-ness against `specification/schema/v1/*.schema.json`. Flags fields present in spec but missing from schema (and vice versa) — the dangerous case under `additionalProperties: false`.
- **Where:** `scripts/drift-checks/spec-schema-drift.mjs`, wired to `.github/workflows/drift-checks.yml`.
- **Tooling:** Node, custom markdown-table parser (no off-the-shelf tool exists for prose→schema). Runs alongside the existing `validate-schemas.mjs`.
- **Policy:** **warn** (non-blocking) until the table parser is proven against the full entity set, then flip to blocking.

### 4. Schema ↔ schema backward-compat (#311) — Node, **hard-fail**
On a PR that edits a `specification/schema/v1/*.schema.json`, diffs it against the base branch and fails if a change is backward-incompatible (removed property under `additionalProperties:false`, tightened pattern, new `required`).
- **Where:** `scripts/drift-checks/schema-compat.mjs` + workflow job.
- **Tooling:** Node `json-schema-diff-validator` (Atlassian origin), **not** getsentry/json-schema-diff. Before wiring, confirm the chosen package has a pinnable release (repo policy on pinning third-party actions/deps).
- **Policy:** **hard-fail** — its whole job is blocking data-breaking merges.

### 5. `validate-schemas.mjs` Step 4 (#839) — Node
Extends the existing `specification/validate-schemas.mjs` to also validate every vocabulary `.glx` file in `specification/5-standard-vocabularies/` against its schema, and to resolve/validate `$ref`s. Deterministic; replaces the equivalent LLM-simulated steps in the skills.
- **Done when:** `make check-schemas` (or the script directly) fails on a malformed vocabulary `.glx`. No new heavy deps (reuse existing AJV; add `eemeli/yaml` only if no YAML parser is already present — see #839/#849 judgment note).

## CI wiring

New workflow `.github/workflows/drift-checks.yml` with jobs `spec-schema-parity` (#309, warn) and `schema-compat` (#311, fail), path-filtered to `specification/**` and `go-glx/*.go`. The `validate-schemas.mjs` Step 4 rides the existing schema-validation workflow.

## Closes
#910, #795, #309, #311, #839.
