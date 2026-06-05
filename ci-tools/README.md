# `ci-tools/` — pinned development tools

This is a **separate Go module** (`github.com/genealogix/glx/ci-tools`) that pins the
versions of *selected* CI/development tools via the Go 1.24+
[`tool` directive](https://go.dev/doc/go1.24#tool-tracking). It does **not** track
every CI tool — only those listed under [Tools pinned here](#tools-pinned-here);
others are intentionally pinned elsewhere (see
[Tools deliberately NOT here](#tools-deliberately-not-here)). It is **not**
imported by, and shares no dependency graph with, the main module — so a tool's
transitive dependencies never pollute the root `go.mod`/`go.sum` and are never
inherited by anyone importing `github.com/genealogix/glx/go-glx` as a library.

> **Why `ci-tools/` and not `tools/`?** `tools/` already holds the first-party
> `tools/driftcheck` package, which *is* part of the main module (it imports
> `go-glx`). Placing this module's `go.mod` at `tools/` would have swallowed
> `driftcheck` into the isolated tool module — breaking `go run ./tools/driftcheck`,
> `go test ./tools/...`, and `make check-code-drift`. Keeping the two concerns in
> separate top-level directories avoids that collision.

## Tools pinned here

| Tool | Package | Used by |
|---|---|---|
| govulncheck | `golang.org/x/vuln/cmd/govulncheck` | `.github/workflows/security.yml`, `make vulncheck` |

## Tools deliberately NOT here

- **gosec** stays on a version-pinned `go install ...@v2.22.4` in
  `.github/workflows/security.yml`. Its `autofix` package imports the Google
  `generative-ai-go` SDK, which transitively pulls the Google Cloud SDK, gRPC,
  and OpenTelemetry. Committing that graph to a scanned `go.mod` would make
  `dependency-review` block PRs whenever any of those (unreachable, build-time-only)
  dependencies picks up a new advisory. `go install` keeps gosec reproducible
  without exposing that tree. Run it ad hoc with
  `go run github.com/securego/gosec/v2/cmd/gosec@v2.22.4 -quiet ./...`.
- **golangci-lint** is pinned via `.golangci-lint-version` and run through
  `golangci-lint-action` (for `only-new-issues` support — see #272).
- **goreleaser** runs through `goreleaser-action`.

## Running a tool

Invoke from the **repository root** with `-modfile` so the tool builds from this
module's pinned versions while analyzing the main module's packages:

```bash
go tool -modfile=ci-tools/go.mod govulncheck ./...
```

Or use the Makefile wrapper: `make vulncheck`.

## Bumping or adding a tool

```bash
cd ci-tools
go get -tool golang.org/x/vuln/cmd/govulncheck@vX.Y.Z   # bump
go get -tool example.com/some/new/tool@vX.Y.Z           # add
go mod tidy
```

Commit the resulting `ci-tools/go.mod` and `ci-tools/go.sum`. The pinned version is the
single source of truth for both CI and local runs. Prefer tools with a lean
dependency graph here; heavyweight trees belong on a pinned `go install` (see
gosec above) to avoid `dependency-review` noise.
