# Deterministic drift checks (CI, no LLM)

**PR 1 of epic #1016** — the deterministic half of the drift strategy.

**No LLMs run in CI.** CI catches the mechanical, decidable drift classes here; the on-demand `check-*` skills (PR 2) handle the semantic remainder locally.

## Pieces

### 1. `glx validate --stdin --entity-type` (#910) — Go
Enabler for the skills' "defer to deterministic tooling" pre-flight: validate a single entity snippet from stdin without the mktemp/cat/rm temp-file dance.
- **Where:** the command + `--stdin`/`--entity-type` flags are wired in `glx/cli_commands.go` (the cobra command lives there); the logic is in `glx/validation_runner.go` — `validateStdinEntity` reads stdin and delegates to the testable `validateEntitySnippet`, which wraps the snippet under its collection key and reuses `ValidateGLXFileStructure`.
- **Done:** `echo '<yaml>' | glx validate --stdin --entity-type person` validates one entity and exits non-zero on failure. Unit (`validation_stdin_test.go`) + manual E2E.

### 2. AST field extractor (#795) — Go
Deterministic extraction of `(field name, yaml tag, line)` from `go-glx/types.go` entity structs, via `go/ast` + `go/parser`. Reflection (used by `driftcheck`) can't report source positions; the AST can.
- **Where:** `tools/internal/structdump/`. `Extract(filename, src []byte)` is **I/O-free** (the caller reads the file), which keeps it trivially testable.
- **Consumer:** `tools/driftcheck` (#673) calls it to attach `go-glx/types.go:NN` to each finding, so `make check-code-drift` — and the `check-code-drift` skill that defers to it — now emit `file:line` (satisfying #676 item 11).
- **Done:** returns every `map[string]*X` collection of `GLXFile` (yaml key → Go type) and each struct's serialized fields, skipping untagged / `yaml:"-"` fields. Table-tested; line attachment verified end-to-end against injected drift.

### 3. Spec ↔ schema parity (#309) — Node, **hard-fail in CI**
Parses the top-level field tables (under `### Required Fields` / `### Optional Fields`) in `specification/4-entity-types/*.md` and compares them against `specification/schema/v1/*.schema.json` on **both** axes: **field presence** (documented but missing from the schema — dangerous under `additionalProperties: false` — and in-schema-but-undocumented) and **required/optional** (a field under "Required Fields" must be in the schema's `required[]`; one under "Optional Fields" must not be).
- **Where:** `scripts/drift-checks/spec-schema-drift.mjs` (no npm deps — pure parsing). The `parseSpecFields`/`compareEntity` core is exported and I/O-free; `spec-schema-drift.test.mjs` fixtures pin the parser (section scoping, the map-key non-field row, combined-row tokens) and all four drift classes, and drive the exported `scanTree` against temp-dir fixture trees for the spec/schema pairing rules.
- **Policy:** The always-running, required `validate-schemas` job in `validate-spec.yml` runs the script with `DRIFT_STRICT=1`; a drift finding therefore prevents a PR from merging (#309). The pairing rules count a missing, unreadable or malformed schema, a schema without a spec page, and a scan that verified no entity as drift. `vocabularies.md` is exempt only when its schema is absent; `glx-file.schema.json` needs no page under `4-entity-types/`. Directory and page read failures are reported as drift rather than crashing the scan. The exported `scanTree` and `run` functions let fixture tests verify the scan and strict exit policy. A bare local `node scripts/drift-checks/spec-schema-drift.mjs` still reports and exits 0; prefix `DRIFT_STRICT=1` to reproduce CI. The current tree reports 0 findings.

### 4. Schema ↔ schema backward-compat (#311) — Node, **hard-fail**
On a PR that edits a `specification/schema/v1/*.schema.json`, diffs it against the base branch and fails on backward-incompatible changes (removed property under `additionalProperties:false`, tightened pattern, new `required`).
- **Where:** `specification/schema-compat.mjs` — it lives under `specification/` so it resolves `json-schema-diff-validator` from `specification/node_modules` (the script the workflow runs). The verdict engine `classifySchemaChange(base, current)` is exported and git-free; `schema-compat.test.mjs` covers new / deleted / invalid-JSON (both sides) and compatible-vs-breaking (add-optional vs. add-required / remove / tighten).
- **Tooling:** `json-schema-diff-validator` (Atlassian origin, a `devDependency`), **not** getsentry/json-schema-diff. Pre-1.0 (`^0.4.2`), CI-only, exact-pinned in `specification/package-lock.json`; its lone unmaintained transitive (`foreach`) never reaches a shipped artifact. git is invoked via `execFileSync` argument arrays (no shell).
- **Policy:** **hard-fail** — its whole job is blocking data-breaking merges.

### 5. `validate-schemas.mjs` Step 4 (#839) — Node
Extends `specification/validate-schemas.mjs` to validate every vocabulary `.glx` file in `specification/5-standard-vocabularies/` against its `vocabularies/<stem>.schema.json`, reusing the already ref-resolved AJV instance from Step 3 (so `$ref`s resolve). Uses the existing `js-yaml` dependency — no new deps.
- **Done:** `make check-schemas` fails on a malformed/non-conforming vocabulary `.glx`. All 23 vocabularies pass today.

## CI wiring

`.github/workflows/validate-spec.yml` runs on every PR. Its `validate-schemas` job is a required status check and runs spec↔schema parity with `DRIFT_STRICT=1` (#309).

`.github/workflows/drift-checks.yml` is path-filtered to `specification/**`, `scripts/drift-checks/**`, and the workflow file itself. Its `schema-compat` job (#311) fetches the base branch, installs the specification dependencies, and fails on backward-incompatible changes. This context is not required by the Main Protection ruleset, so a red result does not itself hold a merge; check it before merging.

The Step 4 vocabulary validation (#839) rides the existing `make check-schemas` rather than this workflow.

The two Node scripts' unit tests (`*.test.mjs`) run via `make test-scripts` (`node --test`, no test framework) — wired into `make check` and the `validate-schemas` job of `validate-spec.yml`, where the specification deps are already installed.

## Closes
#910, #795, #311, #839, #309.

`#309` shipped warn-only in PR #1018. The required `validate-schemas` job now runs it in strict mode, satisfying the issue's merge-gate criterion.
