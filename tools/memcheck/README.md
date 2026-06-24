# memcheck — deterministic memory-file drift detector

`memcheck` checks the repository's agent-memory files — every `CLAUDE.md`, and
`AGENTS.md` if adopted — against reality and fails when an instruction has
rotted. It is the offline, API-key-free counterpart to the LLM-based
[`check-*` drift suite](../../.claude/skills/check-suite/README.md)
(genealogix/glx#1053): stale instructions are worse than missing ones, because an
agent confidently applies dead patterns.

```bash
make check-memory-drift      # or: go run ./tools/memcheck
```

Exit `0` = no drift, `1` = gating drift found, `2` = the tool itself failed
(e.g. `go.mod` not found). The report is written to stdout.

## What it checks

For every discovered memory file it asserts three cheap, deterministic
invariants:

1. **Make targets exist** — every `make <target>` referenced (in an inline code
   span, or in command position inside a fenced block) is a real target in the
   `Makefile`.
2. **Paths exist** — every concrete file or directory path referenced resolves
   on disk, checked relative to the repo root **and** to the memory file's own
   directory (so `go-glx/CLAUDE.md` may write either `serializer.go`-style or
   repo-root-relative paths).
3. **Import path matches go.mod** — any `github.com/<org>/<repo>` path under this
   module's own org is prefixed by the `go.mod` module line, so a module rename
   that a memory file failed to track is caught.

It deliberately does **not** verify issue/PR references (that needs network and
the GitHub API — out of scope for an offline check), and it does not judge
*semantic* correctness of instructions. Those are left to the LLM-based skills.

## What counts as a "path reference"

The hard part is separating a genuine repo path from the many path-shaped tokens
that are not (GitHub actions, stdlib imports, routes, branch prefixes, globs).
The rule is syntactic and intentionally conservative — it would rather miss an
obscure reference than flag a false one, because this check gates CI:

- A **file claim** is an inline-code token that contains a `/` **and** ends in a
  recognized file extension (`go`, `md`, `json`, `yaml`, `ged`, …). This catches
  `testdata/gedcom/shakespeare.ged` while ignoring `actions/checkout`, `os/exec`,
  `bin/glx`, and `release.yml`.
- A **directory claim** is an inline-code token ending in `/` with an internal
  slash (multi-segment), e.g. `docs/cli/`. Single-word trailing-slash tokens
  like `claude/` (a branch prefix) or `go-glx/` (a top-level dir that never
  drifts) are not verified.
- Tokens with a leading `/`, a `://`, an `@`, a `:` / `=`, glob/shell
  metacharacters, or whitespace are never treated as paths.
- Only inline code spans are scanned for paths; fenced code blocks (command/code
  examples) are scanned only for `make` invocations.

## Allowlist

A deliberately-documented reference that should *not* resolve (an illustrative
example path, or a `make` target inside a fenced shell comment) can be suppressed
by adding an entry to `.claude/memory-drift-allowlist.yaml`:

```yaml
- token: path/to/example.go      # the reference, verbatim
  kind: stale-path               # optional: stale-path | unknown-make-target | import-path-drift
  file: docs/CLAUDE.md           # optional: restrict to one memory file
  reason: "Illustrative example, not a real file (#1234)"
```

A missing allowlist file is treated as empty. Run
`go run ./tools/memcheck -v` to list which findings the allowlist suppressed.

## Layout

| File | Responsibility |
|------|----------------|
| `main.go` | I/O: locate repo root, discover memory files, read Makefile + go.mod, write report, set exit code |
| `scan.go` | Pure scan engine: reference extraction and the three checks (no I/O) |
| `allowlist.go` | Parse and match `.claude/memory-drift-allowlist.yaml` |
| `finding.go` | Finding model, severity, sorting |
| `report.go` | Render the human-readable report |

`TestRealTreeHasNoDrift` runs the full pipeline against the committed memory
files under `make test`, so drift is caught even without the dedicated
`make check-memory-drift` target.
