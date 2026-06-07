# `check-*` drift skill suite

Shared home for the five drift-detection skills migrated from `.claude/commands/check-*.md` (epic #1016, decision: skills migration per #833).

## Shared assets (this directory)

- **`findings.schema.json`** — the one machine-readable output contract every skill emits (one object-wrapper `findings-json` report per run). Eval harness #796 consumes it.
- **`severity-rubric.md`** — the one shared `critical|major|minor|info` rubric.

Each skill lives in its own sibling directory as `check-<name>/SKILL.md` and references these two files rather than redefining the schema or the scale.

**What "one contract" means.** Every skill emits a `findings-json` block that *validates against* `findings.schema.json`, and every finding's `severity` comes from `severity-rubric.md`. The schema is intentionally permissive in two ways, so "shared contract" does not mean "byte-identical output":

- The per-finding `category` is a free-form string, not an enum — each skill defines its own category vocabulary in its `SKILL.md` and is responsible for keeping it stable. Only the top-level `command` field is enum-constrained.
- Some fields are optional and only some skills use them: `check-code-drift` additionally emits the `location` object (`go_line`/`schema_line`) and wraps its prose report with a `<!-- last-verified: SHA -->` marker and an `END_OF_DRIFT_REPORT` / `REPORT_TRUNCATED` sentinel for the runner's truncation detection. These are code-drift-specific extras layered on top of the shared schema, not suite-wide requirements.

## Cross-finding conventions

These hold across all five skills (the shared schema can't express them all, so they live here):

- **`validator_caught` and `llm_only` are logical inverses** — exactly one is `true` (enforced by the schema's `oneOf`).
- **`validator_unavailable` findings use `validator_caught: false, llm_only: true`.** When a deterministic tool can't run at all (`./bin/glx` missing, `make check-schemas` can't run), the finding is genuinely neither validator-caught nor a semantic finding, but the schema forces a choice — so the suite convention is to record it as `llm_only: true` (it is the LLM run, not the validator, that surfaced the gap). Never emit `validator_caught: false, llm_only: false`; it fails schema validation.
- **Clean validations are `positive_notes`, not `info` findings** — a passing `glx validate` is not drift.

## Invocation & frontmatter

Invocation moves from `/check-*` slash commands to the skills surface (skills are model-invoked by description, or run via the Skill tool).

Each `SKILL.md` carries `model: claude-opus-4-8` and an `allowed-tools:` list. **For a skill that runs inline in the active conversation (none here set `context: fork`), these are intent/forward-compat, not enforced at runtime:** the skill executes under whatever model and tool-permission set the current session already has, so `model:` does not switch models mid-session the way a slash command's `model:` does. They document the top-tier model and minimal tool surface these accuracy-sensitive checks are meant to run under, and become load-bearing if a skill later adopts `context: fork`. To actually run a check on Opus today, invoke it from an Opus session.

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
