---
name: check-examples
description: Review example GLX archives for compliance with specification and code
allowed-tools:
  - Read
  - Grep
  - Glob
  - Bash(git rev-parse:*)
  - Bash(gh issue list:*)
  - Bash(gh run list:*)
  - Bash(./bin/glx validate:*)
  - Bash(./bin/glx --version:*)
  - Bash(date:*)
model: claude-opus-4-8
---

# check-examples

Review example GLX archives in `docs/examples/` for compliance with the specification and implementation. This skill separates deterministic checks (run by `glx validate` and CI tests) from the semantic checks only an LLM can make, and emits a machine-readable `findings-json` block at the end of every run.

Reference materials that govern this skill:
- Severity rubric: `.claude/skills/check-suite/severity-rubric.md`
- Output schema: `.claude/skills/check-suite/findings.schema.json`

## Source of Truth Flow

```
Specification (*.md) → Schema (*.schema.json) → Go code (types.go)
                                    ↓
                              Example archives
                            (docs/examples/*.glx)
```

Examples must be valid YAML that the GLX CLI accepts, follow schema definitions in `specification/schema/v1/`, use valid vocabulary terms, and demonstrate best practices from the specification.

## Which directories to review

```
docs/examples/
├── minimal/                # Bare minimum valid archive
├── basic-family/           # Simple family relationships
├── complete-family/        # Full-featured example with all entity types
├── single-file/            # Single-file archive format
├── assertion-workflow/     # Assertion workflow demonstration
├── temporal-properties/    # Temporal property demonstrations
└── participant-assertions/ # Assertion participant patterns
```

**`docs/examples/westeros/` is an external-archive pointer, not a reviewable example archive.** It contains only a `README.md` documenting that the live archive lives in a separate repository (`github.com/genealogix/glx-archive-westeros`). Do not pass it to `glx validate` and do not loop it into the per-directory checks below. The only westeros task is described in step 5.

## Step 1: Capture deterministic validator output

Before any semantic review, capture the authoritative, machine-verified results of the GLX CLI validator. These results are ground truth — do not re-derive them by hand in later steps.

```bash
git rev-parse HEAD
./bin/glx --version
./bin/glx validate docs/examples/minimal/
./bin/glx validate docs/examples/basic-family/
./bin/glx validate docs/examples/complete-family/
./bin/glx validate docs/examples/single-file/
./bin/glx validate docs/examples/assertion-workflow/
./bin/glx validate docs/examples/temporal-properties/
./bin/glx validate docs/examples/participant-assertions/
```

For each directory, record:
- **Exit code** (0 = clean, non-zero = failed)
- **Full stdout** verbatim
- **Full stderr** verbatim

Do not paraphrase the validator output — quote it so reviewers can audit. Any directory where the validator exits non-zero produces a finding with `validator_caught: true, llm_only: false`.

**The validator is authoritative for entity structure, vocabulary terms, cross-references, and date formats.** The CLI runs two validation passes: (1) JSON Schema validation — required fields, types, patterns, mutual exclusivity; (2) Go cross-reference validation — entity reference existence, cycle detection, vocabulary validation, date format validation. Do not re-verify these items by hand.

## Step 2: LLM semantic review (validator-blind spots only)

Only review items that `glx validate` cannot judge. Findings here get `validator_caught: false, llm_only: true`.

For each of the seven reviewable example directories:

### 2.1 Pedagogical clarity

Does the example actually demonstrate what its directory name and README claim? Specific questions:
- Does `assertion-workflow/` show the full assertion lifecycle (created → contested → resolved), or just static assertion entities?
- Does `temporal-properties/` exercise each temporal property variant documented in the spec?
- Does `participant-assertions/` cover multiple participant roles?
- Does `complete-family/` genuinely use all major entity types (Person, Event, Relationship, Place, Source, Citation, Repository, Media, Assertion)?
- Does `minimal/` remain genuinely minimal — i.e., no entities beyond what is required to be a valid archive?

If an example's content does not match its stated purpose, that is a `major` finding.

### 2.2 Best practices

- **Meaningful IDs**: IDs should be descriptive (e.g., `person-shakespeare-william-1564`) not opaque (e.g., `p1`, `e3`).
- **Realistic data**: Names, dates, and places should be plausible historical or fictional examples — not placeholder text.
- **Notes usage**: Are `notes` fields used appropriately for context that does not belong in structured fields?
- **Properties field**: Is the `properties` field used only for genuinely custom data, not as a workaround for standard fields?
- **Completeness**: Does each example demonstrate its intended feature fully, or is it truncated in a way that would confuse a user learning GLX from it?

### 2.3 README accuracy

For each example directory, read its `README.md` and verify:
- It accurately describes what entities are in the archive (no stale entity counts or missing entity types).
- All feature highlights it claims the example demonstrates are actually present in the `.glx` files.
- There is no documentation for features or files that no longer exist in the directory.

### 2.4 Feature coverage (newer CLI commands)

Check whether any example demonstrates recently-added CLI commands (`glx add`, `glx search`, `glx merge`). Note gaps as `info`-level findings — not all examples need to cover every command, but it is useful to flag where new feature coverage could be added.

## Step 3: Specification pattern alignment (LLM-checkable items only)

Do not re-run field-level or vocabulary-level checks; the validator handles those. Limit this step to structural patterns that require reading the specification prose.

### 3.1 Archive organization patterns

Verify that examples using the multi-file format follow the directory structure described in `specification/3-archive-organization.md`, and that `single-file/` matches the single-file format specification. This is a layout-level check, not a field-level check.

### 3.2 Vocabulary embedding

Verify that any custom vocabularies referenced within the examples are properly defined or linked. Standard vocabulary terms are already checked by the validator; this step covers only custom-vocabulary embedding patterns described in the specification.

## Step 4: Roundtrip validation (CI-verified)

Roundtrip serialization (YAML deserialize → Go struct → re-serialize → re-validate against per-entity JSON schemas) is checked deterministically by `go-glx/example_archives_roundtrip_test.go` (`TestExampleArchivesRoundTrip`), which runs in CI as the `test-conformance` job in `.github/workflows/validate-spec.yml`.

Check the latest CI run on HEAD:

```bash
gh run list --workflow validate-spec.yml --limit 1 \
   --json conclusion,headSha,databaseId \
   --jq '.[0]'
```

If `conclusion` is `"success"`, roundtrip alignment is verified — record this as a positive note.

If `conclusion` is `"failure"` or `"cancelled"`, fetch the failing test output from the `test-conformance` job and include it verbatim in the report. The per-entity diff produced by the test is more actionable than any LLM-simulated drift claim. File each failing test case as a `critical` finding with `validator_caught: true, llm_only: false`.

Do not re-verify type definitions or YAML marshaling by hand. The test exercises every entity in every example against its real schema — reading `types.go` adds no information the test does not have, and the LLM cannot detect omitempty drops at the YAML level in any case.

## Step 5: Westeros external pointer

`docs/examples/westeros/` does not contain `.glx` files. It is a documentation-only pointer to `github.com/genealogix/glx-archive-westeros`.

Read `docs/examples/westeros/README.md` and verify:
- The external repository link is still valid and points to the correct location.
- The description accurately represents what the Westeros archive demonstrates.
- There is no implication that `docs/examples/westeros/` itself is a complete archive (which would mislead users running `glx validate docs/examples/westeros/` and seeing no output).

File findings here as `minor` if the README is misleading; `info` if everything is clear.

## Step 6: Cross-reference with known issues

Before writing findings, run:

```bash
gh issue list -R genealogix/glx --state open --limit 50 --json number,title --jq '.[] | "#\(.number) \(.title)"'
```

For each finding you would emit, check whether it matches an already-open GitHub issue. If it does, suppress the finding from the `findings` array and increment `suppressed_as_duplicate_of_known_issue` in the summary. Record what you suppressed and why in the prose report, so reviewers can see what was filtered.

## Output format

### Prose report

Write a human-readable summary covering each example directory. Use one consistent severity notation (severity words: critical, major, minor, info). Include:

1. Validator output per directory (verbatim exit code and any error/warning lines)
2. LLM semantic findings per directory
3. CI roundtrip status
4. Westeros pointer status
5. Summary counts

### Machine-readable findings block

After the prose report, emit exactly one fenced `findings-json` block. The block must conform to `.claude/skills/check-suite/findings.schema.json`. Required fields per finding: `file`, `severity`, `category`, `message`, `validator_caught`, `llm_only`.

**Category enum for this skill:**

| Category | Default severity | Description |
|----------|-----------------|-------------|
| `example_validation` | critical | `glx validate` exits non-zero on this archive |
| `roundtrip_failure` | critical | `TestExampleArchivesRoundTrip` fails for this archive |
| `pedagogical` | major | Example does not demonstrate what its name/README claims |
| `best_practices` | minor | ID naming, realistic data, notes/properties misuse |
| `readme_accuracy` | minor | README does not match archive contents |
| `feature_coverage` | info | Newer CLI feature not demonstrated by any example |
| `external_pointer` | minor/info | Westeros README misleading or stale |

Severity follows the shared rubric in `.claude/skills/check-suite/severity-rubric.md`. When uncertain, assign `info`.

Example output structure:

````
```findings-json
{
  "command": "check-examples",
  "commit": "<output of git rev-parse HEAD>",
  "checked_files": [
    "docs/examples/minimal",
    "docs/examples/basic-family",
    "docs/examples/complete-family",
    "docs/examples/single-file",
    "docs/examples/assertion-workflow",
    "docs/examples/temporal-properties",
    "docs/examples/participant-assertions",
    "docs/examples/westeros"
  ],
  "findings": [
    {
      "file": "docs/examples/complete-family/persons/person-john-smith.glx",
      "line": null,
      "severity": "minor",
      "category": "best_practices",
      "message": "Entity ID 'person-john-smith' is ambiguous when multiple persons share the name; consider including birth year.",
      "fix": "Rename to 'person-john-smith-1842' or similar disambiguator.",
      "validator_caught": false,
      "llm_only": true
    }
  ],
  "positive_notes": [
    "glx validate exits 0 on all seven reviewable example directories",
    "TestExampleArchivesRoundTrip passes on HEAD (validate-spec.yml conclusion: success)"
  ],
  "summary": {
    "critical": 0,
    "major": 0,
    "minor": 1,
    "info": 0,
    "suppressed_as_duplicate_of_known_issue": 0
  }
}
```
````

The `telemetry` field is injected by the eval harness runner, not self-reported — omit it.

## Important notes

- **Validator exit code is the highest-priority signal.** Any non-zero exit from `glx validate` on an example directory means the example is broken and users cannot use it. These are always `critical`.
- **Do not re-verify what the validator already checked.** Entity structure, vocabulary terms, cross-references, and date formats are all deterministically checked by `glx validate`. Re-verifying them by hand is the leading source of false positives in this skill.
- **`westeros/` is never passed to `glx validate`.** It contains no `.glx` files; the validator would silently succeed and the result would be meaningless.
- **`validator_caught` and `llm_only` must be logical inverses.** If `validator_caught` is `true`, `llm_only` must be `false`, and vice versa.
- **Default to `info` when uncertain about severity.** A false `critical` costs more reviewer trust than a missed `minor`.
