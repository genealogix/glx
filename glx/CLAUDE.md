# glx CLI — Claude Guide

This is the main CLI application. All filesystem I/O happens here — the `go-glx/` library is I/O-free.

## Cobra Command Handler Pattern

Functions with `_` parameters (unused cobra.Command) must be thin wrappers with no logic:

```go
// CORRECT — thin wrapper delegates immediately
func runValidate(_ *cobra.Command, args []string) error {
    return validatePaths(args)
}

func validatePaths(args []string) error {
    // All logic here
}

// INCORRECT — logic in function with _ parameter
func runValidate(_ *cobra.Command, args []string) error {
    paths := args  // NO — move this to a separate function
    _ = paths
    return nil
}
```

## Unused Parameters

`_` parameters are ONLY acceptable when required by an interface (cobra handlers). For regular functions, remove unused parameters entirely and update call sites.

## Key Files

- `cli_commands.go` — all command definitions and `rootCmd.AddCommand()` calls
- `*_runner.go` — one per CLI command (analyze, import, export, merge, etc.)
- `docs_runner.go` — hidden `glx docs` subcommand that regenerates `docs/cli/` from the live Cobra tree (called by `make docs-cli`)
- `archive_io.go` — single/multi-file archive read/write
- `testdata/gedcom/` — 180+ GEDCOM test files

## Git Operations — never shell out to the `git` binary

The CLI must have **no runtime dependency on `git` being on `PATH`**. For any git
operation (reading HEAD, checking whether the work tree is clean, etc.), use the
pure-Go library `github.com/go-git/go-git/v5` instead of `os/exec`-ing `git`.
See `archive_cache.go` (`openArchiveRepo`, `gitHeadSHA`, `gitWorkingTreeClean`)
for the pattern: open with `git.PlainOpenWithOptions(root, &git.PlainOpenOptions{DetectDotGit: true})`
so an archive nested in a larger repo still resolves, and treat any error as
"not a git repo / can't tell" so callers fall back gracefully. Shelling out to
`git` in **test fixtures** (e.g. to construct a real repo) is fine and even
preferable — it verifies go-git interoperates with real-git output.

## Serialization Gotchas

- Multi-file archive filenames are derived deterministically from entity IDs (lowercased, `.glx` suffix) — see `go-glx/id_generator.go::EntityIDToFilename`
- Two entity IDs that differ only by case (e.g., `Person-A` and `person-a`) collide on case-insensitive filesystems and are rejected at serialize time with `ErrCaseInsensitiveCollision`
- Vocabularies are serialized as part of multi-file archives automatically
