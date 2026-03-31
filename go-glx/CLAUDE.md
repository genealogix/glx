# go-glx Library — Claude Guide

This is the core GLX library (`package glx`). It is a **pure library** — all I/O happens in the `glx/` CLI package.

## Critical Rule: No Filesystem I/O

The go-glx package must NEVER perform filesystem I/O:

- NO `os.ReadFile`, `os.WriteFile`, `os.Open`, `os.Create`
- NO `os.MkdirAll`, `os.Stat`, `os.ReadDir`
- NO `filepath.Join` with file operations
- YES to `io.Reader`, `io.Writer`, `[]byte` parameters

```go
// WRONG — library doing I/O
func SerializeSingleFile(glx *GLXFile, outputPath string) error {
    yamlBytes, _ := yaml.Marshal(glx)
    return os.WriteFile(outputPath, yamlBytes, 0o644)
}

// CORRECT — library returns bytes, CLI does I/O
func SerializeToBytes(glx *GLXFile) ([]byte, error) {
    return yaml.Marshal(glx)
}
```

Rationale: testability without filesystem, usable in non-CLI contexts (web servers, embedded), clean separation of concerns.

## Key Files

- `types.go` — core GLX entity type definitions (read first)
- `gedcom_converter.go` — main GEDCOM conversion orchestrator
- `serializer.go` — single/multi-file serialization
- `vocabularies.go` — vocabulary embedding via `go:embed` + `sync.Once` cache

## Performance Profiling

```bash
go test -bench='BenchmarkName' -benchtime=1x -memprofile=/tmp/prof.out ./go-glx/
go tool pprof -top -flat /tmp/prof.out           # direct allocators
go tool pprof -top -cum /tmp/prof.out            # call tree
go tool pprof -top -flat -focus='glx' /tmp/prof.out  # package only
```

Common pitfalls:
- Map literals in function bodies allocate every call — use package-level vars
- `LogInfo(fmt.Sprintf(...))` allocates even when disabled — use `LogInfof`
- `append` doubling — use `make([]T, exactSize)` when size is known
- Vocabularies cached via `sync.Once` — don't add a second load path

## Serializer Architecture

- Vocabulary embedding: `go:embed` in binary
- Entity filenames: random 8-char hex (e.g., `person-a3f8d2c1.glx`), mapped from entity IDs via the serializer
- Write strategy: sequential (no parallelization)
- Validation: default on via `SerializerOptions.Validate` (CLI exposes `--no-validate` flag)

## GEDCOM Import

- GEDCOM 5.5.1: `@REF@` references, NOTE records
- GEDCOM 7.0: `@VOID@` references, SNOTE records
- Features without GLX equivalent go in `Properties` field

### Adding GEDCOM Tag Support

1. Find converter file (e.g., `gedcom_individual.go`)
2. Add tag handling in `switch` statement
3. Map to GLX entity
4. Add test in `gedcom_test.go`
5. Update gap analysis

### Debugging Import

1. Enable verbose logging: `conv.Logger.LogInfo(...)`
2. Check `ConversionContext` entity maps
3. Run `make test`
4. Check `conv.Errors` accumulation

## GEDCOM Specification PDFs

Do NOT read full PDFs — they are too large. Use the split versions:
- 5.5.1: `docs/gedcom-spec/GEDCOM_5.5.1_Specification/part_*.pdf` (6 parts)
- 7.0: `docs/gedcom-spec/GEDCOM_7.0_Specification/part_*.pdf` (6 parts)
- Also available: 3.0, 4.0, 5.0, 5.3, 5.4, 5.5, 5.5.5, 5.6, GEDZIP 0.1/0.2
