# driftcheck — deterministic schema/code drift detector

`driftcheck` compares the `go-glx` Go type definitions against the GLX JSON
Schemas and fails when they have structurally drifted. It is the deterministic
counterpart to the LLM-based [`/check-code-drift`](../../.claude/commands/check-code-drift.md)
slash command (genealogix/glx#673).

```bash
make check-code-drift        # or: go run ./tools/driftcheck
```

Exit `0` = no drift, `1` = gating drift found, `2` = the tool itself failed
(e.g. schemas not found). The report is written to stdout.

## What it checks

Source-of-truth flow: **specification → JSON Schema → Go code**. A finding means
the Go code needs to change to match the schema. For every entity, nested type,
and vocabulary type, it compares:

- **Field presence**, both directions — every schema property has a Go field
  (keyed by yaml tag), and every Go field has a schema property when the schema
  is closed (`additionalProperties: false`).
- **Required vs optional** — a schema-required field must not carry `omitempty`;
  a schema-optional field must.
- **Type families** — `string`/`number`/`integer`/`boolean`/`array`/`object`,
  with the GLX-specific shapes handled explicitly:
  - `NoteList` ↔ `oneOf[string, array]`
  - `DateString` (and other named string types) ↔ `string`
  - `EntityRef` (and `*EntityRef`) ↔ a `oneOf` of single-field objects (the
    "typed reference" pattern), compared against the union of branches
  - the unified `VocabularyEntry` ↔ the *union* of every type-vocabulary
    `definition` (one Go type backs all of them since #727)
  - pointers, slices, and `map[string]*T` are followed recursively;
    `map[string]any` (free-form `properties`) is a leaf

It deliberately does **not** check value constraints (`minLength`, `pattern`,
`enum`, cross-reference existence, …). Those are enforced at runtime by the
two-layer validator, and asserting on them here would produce false positives.
Anything ambiguous is left to the LLM `/check-code-drift` command.

## Allowlist

Known, by-design drift is suppressed by reading
[`.claude/drift-allowlist.yaml`](../../.claude/drift-allowlist.yaml). A finding
is suppressed when its `file` and per-symbol identity match an entry — the same
matching the slash commands use, so an entry naming a field by its Go name also
matches a finding that names it by yaml tag (within the same owning type).

Run `go run ./tools/driftcheck -v` to list which findings the allowlist
suppressed.

## Layout

| File | Responsibility |
|------|----------------|
| `main.go` | I/O: locate repo root, read schemas + allowlist, write report, set exit code |
| `schema.go` | JSON Schema model + cross-file / JSON-pointer `$ref` resolution |
| `compare.go` | Pure reflection-based comparison engine (no I/O) |
| `bindings.go` | Maps each `go-glx` type to its schema location; the `VocabularyEntry` union |
| `allowlist.go` | Parse and match `.claude/drift-allowlist.yaml` |
| `report.go` | Render the human-readable report |

`TestRealTreeHasNoDrift` runs the full pipeline against the committed schemas
and types under `make test`, so drift is caught even without the dedicated
`make check-code-drift` target.
