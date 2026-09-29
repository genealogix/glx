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

- **`go-glx/`** — the core library. Types, serialization, validation, GEDCOM conversion, vocabulary handling.
- **`glx/`** — the CLI application. Cobra commands, flag parsing, user-facing messages, and — critically — every interaction with the filesystem.

Without an explicit rule, library functions would naturally reach for `os.ReadFile`, `os.WriteFile`, and `os.MkdirAll` — that pattern causes three recurring problems:

- **Tests needed the filesystem.** A unit test for "does this serialize correctly?" had to create temp directories, write bytes, and clean up. It made tests slower, flakier (especially on Windows with antivirus or indexer locks — see #701), and harder to run in parallel.
- **The library could not be embedded anywhere else.** A web server wanting to convert an uploaded GEDCOM file to GLX bytes had no way to do so without first writing the input to disk and then reading output from disk.
- **Separation of concerns was muddy.** "What does this function do?" had no clean answer — the CLI layer, the library layer, and the filesystem were all entangled.

## Decision

The `go-glx` package MUST NOT perform filesystem I/O. This ADR is the canonical
statement of that rule. Anything else in the repository that states it — agent
guides included — may only summarize it and must point back here; none of it is
authoritative on its own.

Prohibited in `go-glx/` production code:

- `os.ReadFile`, `os.WriteFile`, `os.Open`, `os.Create`
- `os.MkdirAll`, `os.Stat`, `os.ReadDir`
- `filepath.Join` (or any other path construction) combined with file operations
- walking the real filesystem — `filepath.WalkDir`, `filepath.Walk`, `os.ReadDir`
  recursion
- any other direct access to the operating system's filesystem

Allowed instead:

- `io.Reader`, `io.Writer`, and `[]byte` parameters and return values
- `io/fs.FS` (`fs.FS`) when a function genuinely needs a tree of files, so the
  caller chooses the backing store — a real directory, an embedded filesystem,
  or an in-memory one. Walking a caller-supplied `fs.FS` with `fs.WalkDir` is
  fine: the library never decides what the tree is backed by. `ImportGEDZIP`
  in `go-glx/gedzip_import.go` works this way.
- `go:embed` for data the library itself owns, such as the standard vocabularies

Concretely:

```go
// Wrong — library doing I/O
func (s *DefaultSerializer) SerializeSingleFile(glx *GLXFile, outputPath string) error {
    yamlBytes, err := yaml.Marshal(glx)
    if err != nil {
        return err
    }
    return os.WriteFile(outputPath, yamlBytes, 0o644)
}

// Correct — library returns bytes, CLI does I/O.
// This is the real signature; see go-glx/serializer.go.
func (s *DefaultSerializer) SerializeSingleFileBytes(glx *GLXFile) ([]byte, error) {
    return yaml.Marshal(glx)
}
```

Everything on the prohibited list lives in the `glx/` CLI package. The one carve-out
is unexported test helpers that exist purely to feed fixtures to the library's own
tests — `go-glx/gedcom_test_helpers.go` opens files for that reason. They are not part
of the library's API and no production path reaches them; nothing exported may rely on
them.

## Consequences

**Positive**

- Library APIs let tests avoid disk I/O — faster, more deterministic, parallel-safe on every OS. Some existing serializer and roundtrip tests still use temp dirs as a convenience, but nothing in the library forces them to.
- The library is embeddable. A hypothetical web service, bulk-processing pipeline, or another CLI can use `go-glx` by piping bytes in and out.
- Responsibility is obvious. A developer reading a function signature sees `([]byte) → ([]byte, error)` and knows immediately whether it touches the filesystem.
- Windows-specific filesystem quirks (AV locks, case-insensitive collisions, path length limits) are confined to the CLI layer, where they can be handled once.

**Negative**

- The CLI layer has more plumbing code — it has to read the file, hand the bytes to the library, take the bytes back, and write the file. That is the right place for that code, but it is not free.
- Contributors touching `go-glx/` have to resist the reflex of calling `os.ReadFile` in non-test code to load a fixture or vocabulary. Where fixture data is needed inside the library itself (e.g., the standard vocabularies), the project uses `go:embed` — see `specification/5-standard-vocabularies/embed.go`, which `go-glx/vocabularies.go` imports.
- Functions that would naturally take a path (e.g., loading vocabularies) have to take a reader or embed the data. The library uses `go:embed` for the standard vocabularies precisely because of this rule.
