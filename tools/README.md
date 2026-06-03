# `tools/` — pinned development tools

This is a **separate Go module** (`github.com/genealogix/glx/tools`) that exists
only to pin the versions of CI/development tools via the Go 1.24+
[`tool` directive](https://go.dev/doc/go1.24#tool-tracking). It is **not**
imported by, and shares no dependency graph with, the main module — so the heavy
transitive dependencies these tools drag in (gRPC, OpenTelemetry, the Google
Cloud SDK, …) never pollute the root `go.mod`/`go.sum` and are never inherited by
anyone importing `github.com/genealogix/glx/go-glx` as a library.

## Tools pinned here

| Tool | Package | Used by |
|---|---|---|
| gosec | `github.com/securego/gosec/v2/cmd/gosec` | `.github/workflows/security.yml`, `make gosec` |
| govulncheck | `golang.org/x/vuln/cmd/govulncheck` | `.github/workflows/security.yml`, `make vulncheck` |

> **Not here:** `golangci-lint` is pinned via `.golangci-lint-version` and run
> through `golangci-lint-action` (for `only-new-issues` support — see #272), and
> `goreleaser` runs through `goreleaser-action`. Both are intentionally kept on
> their actions rather than the `tool` directive.

## Running a tool

Invoke from the **repository root** with `-modfile` so the tool builds from this
module's pinned versions while analyzing the main module's packages:

```bash
go tool -modfile=tools/go.mod gosec -quiet ./...
go tool -modfile=tools/go.mod govulncheck ./...
```

Or use the Makefile wrappers: `make gosec`, `make vulncheck`, `make security`.

## Bumping or adding a tool

```bash
cd tools
go get -tool github.com/securego/gosec/v2/cmd/gosec@vX.Y.Z   # bump
go get -tool example.com/some/new/tool@vX.Y.Z                # add
go mod tidy
```

Commit the resulting `tools/go.mod` and `tools/go.sum`. The pinned version is the
single source of truth for both CI and local runs.
