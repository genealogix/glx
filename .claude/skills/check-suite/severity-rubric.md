# Shared severity rubric — `check-*` drift suite

Single source of truth for severity across **all five** `check-*` skills (epic #1016). Each skill references this file instead of defining its own scale, so `critical | major | minor | info` mean the same thing everywhere and the `findings-json` `severity` enum is consistent for the eval harness (#796).

Generalised from the `check-code-drift` rubric established in #827; widened to the whole suite per the maintainer note on that issue.

## Levels

| Level | Meaning | Rule of thumb |
|-------|---------|---------------|
| **critical** | Drift that makes previously-valid data invalid, or makes the validator reject valid data. Silent data loss / corruption. | A field exists in spec but is missing from a schema with `additionalProperties: false` → the validator rejects archives the spec says are valid. Ship-blocker. |
| **major** | Drift that misleads users or tools, or breaks a documented contract, but does not corrupt data. | Documentation describes a removed/renamed command; a schema requires a field the spec marks optional; a CLI example that no longer runs. |
| **minor** | Cosmetic or low-impact inconsistency. Correct to fix, safe to defer. | Description wording mismatch; an omitted `omitempty`; stale-but-harmless cross-reference. |
| **info** | Not drift. An idiomatic/intentional difference, or a positive confirmation. Also the **default when uncertain**. | A field deliberately absent from a derived doc; a known, allow-listed difference. Prefer `info` over guessing a higher level. |

## Assignment principles

1. **Severity is per finding *category*, not global.** Each skill's `SKILL.md` carries a small table mapping its categories → default severity, using only the four levels above.
2. **Default to `info` when uncertain.** A false `critical` costs more reviewer trust than a missed `minor`. (#676 false-positive-reduction.)
3. **`additionalProperties: false` raises the stakes.** Any field-presence drift on a schema with `additionalProperties: false` is at least **major**, and **critical** if it means valid data is rejected.
4. **Deterministic-caught ≠ free.** If a deterministic check (`glx validate`, `validate-schemas.mjs`, CI) already flags it, set `validator_caught: true` — but keep the finding so the report is complete; severity is unchanged by who caught it.
5. **Allow-listed differences are `info`.** Entries in `.claude/drift-allowlist.yaml` are already triaged: `permanent: true` entries are by-design (treat as `info` / not drift), while entries with a `tracking_issue` are *temporary deferrals* — don't re-report them as new findings, refer to the tracking issue. Never escalate an allow-listed item.

## Per-skill category tables

Each skill carries its own category→severity table in its `SKILL.md` (added during the migration in this PR); this file defines the shared four-level scale they all reference:

- `check-schema-drift` — see #838 (rubric rows for field_presence / field_type / required_optional / ref_type / vocabulary).
- `check-code-drift` — established in #827.
- `check-docs-drift`, `check-examples`, `check-spec` — adopt this file during their migration.
