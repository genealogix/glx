---
title: "ADR-0006: The go-glx library never performs filesystem I/O"
description: Why the core go-glx library exposes bytes/readers/writers and leaves all filesystem operations to callers.
layout: doc
---

# ADR-0006: The go-glx library never performs filesystem I/O

## Status

Accepted

## Context

The GLX codebase is split into two Go packages:

- **`go-glx/`** — the core library. Types, serialisation, validation, GEDCOM conversion, vocabulary handling.
- **`glx/`** — the CLI application. Cobra commands, flag parsing, user-facing messages, and — critically — every interaction with the filesystem.

Without an explicit rule, library functions would naturally reach for `os.ReadFile`, `os.WriteFile`, and `os.MkdirAll` — that pattern causes three recurring problems:

- **Tests needed the filesystem.** A unit test for "does this serialize correctly?" had to create temp directories, write bytes, and clean up. It made tests slower, flakier (especially on Windows with antivirus or indexer locks — see #701), and harder to run in parallel.
- **The library could not be embedded anywhere else.** A web server wanting to convert an uploaded GEDCOM file to GLX bytes had no way to do so without first writing the input to disk and then reading output from disk.
- **Separation of concerns was muddy.** "What does this function do?" had no clean answer — the CLI layer, the library layer, and the filesystem were all entangled.

## Decision

The `go-glx` package MUST NOT perform filesystem I/O. From [`go-glx/CLAUDE.md`](https://github.com/genealogix/glx/blob/main/go-glx/CLAUDE.md):

> The go-glx package must NEVER perform filesystem I/O: NO `os.ReadFile`, `os.WriteFile`, `os.Open`, `os.Create`. [...] YES to `io.Reader`, `io.Writer`, `[]byte` parameters.

Concretely:

```go
// Wrong — library doing I/O
func SerializeSingleFile(glx *GLXFile, outputPath string) error {
    yamlBytes, _ := yaml.Marshal(glx)
    return os.WriteFile(outputPath, yamlBytes, 0o644)
}

// Correct — library returns bytes, CLI does I/O
func SerializeToBytes(glx *GLXFile) ([]byte, error) {
    return yaml.Marshal(glx)
}
```

All `os.*` calls, `filepath.Join` with file operations, and directory walking live in the `glx/` package or in test helpers outside `go-glx/`.

## Consequences

**Positive**

- Library tests run without touching the disk — faster, more deterministic, parallel-safe on every OS.
- The library is embeddable. A hypothetical web service, bulk-processing pipeline, or another CLI can use `go-glx` by piping bytes in and out.
- Responsibility is obvious. A developer reading a function signature sees `([]byte) → ([]byte, error)` and knows immediately whether it touches the filesystem.
- Windows-specific filesystem quirks (AV locks, case-insensitive collisions, path length limits) are confined to the CLI layer, where they can be handled once.

**Negative**

- The CLI layer has more plumbing code — it has to read the file, hand the bytes to the library, take the bytes back, and write the file. That is the right place for that code, but it is not free.
- Contributors touching `go-glx/` have to resist the reflex of calling `os.ReadFile` to load a test fixture. Test helpers use `go:embed` or explicit byte slices instead.
- Functions that would naturally take a path (e.g., loading vocabularies) have to take a reader or embed the data. The library uses `go:embed` for the standard vocabularies precisely because of this rule.
