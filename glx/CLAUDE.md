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
- `archive_io.go` — single/multi-file archive read/write
- `testdata/gedcom/` — 180+ GEDCOM test files

## Serialization Gotchas

- Multi-file archives use random filenames; entity IDs are preserved as YAML map keys
- Vocabularies are serialized as part of multi-file archives automatically
