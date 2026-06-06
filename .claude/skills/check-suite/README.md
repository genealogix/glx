# `check-*` drift skill suite

Shared home for the five drift-detection skills migrated from `.claude/commands/check-*.md` (epic #1016, decision: skills migration per #833).

## Shared assets (this directory)

- **`findings.schema.json`** — the one machine-readable output contract every skill emits (one object-wrapper report per run). Eval harness #796 consumes it.
- **`severity-rubric.md`** — the one shared `critical|major|minor|info` rubric.

Each skill lives in its own sibling directory as `check-<name>/SKILL.md` and references these two files rather than redefining the schema or the scale.

## Migration status (PR 2 of epic #1016)

| Skill | From | Folds in | Status |
|-------|------|----------|--------|
| check-docs-drift | `commands/check-docs-drift.md` | #829 (dynamic `docs/**` glob, ADRs excluded), #800 (glob fix), #831 (cross-ref + findings-json); closes #298 | ✅ |
| check-schema-drift | `commands/check-schema-drift.md` | #838 (shared rubric), #986 (dynamic enumeration) | ✅ |
| check-code-drift | `commands/check-code-drift.md` | all of #676 (1–12), #771 (six bugs) | ✅ |
| check-examples | `commands/check-examples.md` | #834 (keep Step 1), #835 (Step 4→CI), #303 (westeros pointer), #836 (findings-json) | ✅ |
| check-spec | `commands/check-spec.md` | #314 (RFC 2119 tooling-only), #315, #847 (findings-json), #849 (delegate + mktemp-safe cleanup) | ✅ |

Shared assets (contract + rubric): ☑ landed in this commit.

## Division of labour (no LLM in CI)

- **Deterministic checks run in CI** (PR 1): spec↔schema parity (#309), schema↔schema breaking-change (#311), `validate-schemas.mjs` Step 4 (#839), AST extraction (#795), `glx validate --stdin` (#910).
- **These skills run on-demand locally** for the *semantic* drift the deterministic layer can't see, emitting `findings-json`.
- **Eval harness (#796)** runs locally/nightly, never in CI.
