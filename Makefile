# GENEALOGIX Makefile
.PHONY: help check build build-cli build-website install-deps install-hooks lint lint-fix lint-codeowners fix fix-diff test test-verbose test-race test-coverage bench mod-tidy mod-verify tidy-check ci-tools-tidy-check clean fmt check-schemas check-drift-allowlist check-code-drift check-memory-drift test-scripts check-links validate-examples docs-cli release-snapshot vulncheck gosec license-check changelog changelog-check

.DEFAULT_GOAL := help

## Help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## Dependencies
install-deps: ## Install Go modules and npm packages
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing website dependencies..."
	cd website && npm install

# lefthook version pin — bump alongside any hook-compatibility changes
LEFTHOOK_VERSION ?= v2.1.8

install-hooks: ## Install lefthook git pre-commit hooks (run once per clone)
	@if ! command -v lefthook >/dev/null 2>&1; then \
		echo "Installing lefthook $(LEFTHOOK_VERSION) via 'go install'..."; \
		go install github.com/evilmartians/lefthook@$(LEFTHOOK_VERSION); \
		GO_BIN_DIR="$$(go env GOBIN)"; \
		if [ -z "$$GO_BIN_DIR" ]; then GO_BIN_DIR="$$(go env GOPATH)/bin"; fi; \
		export PATH="$$GO_BIN_DIR:$$PATH"; \
	fi; \
	if ! command -v lefthook >/dev/null 2>&1; then \
		echo "ERROR: lefthook is installed but not on PATH. Ensure your Go bin directory is on PATH and re-run 'make install-hooks'."; \
		exit 1; \
	fi; \
	lefthook install

## Verification
check: tidy-check ci-tools-tidy-check lint lint-codeowners test check-schemas check-drift-allowlist check-code-drift check-memory-drift test-scripts check-links validate-examples ## Run all checks (mirrors CI)
	@echo "All checks passed."

## Build
# Version injected into the binary via ldflags, mirroring GoReleaser's -trimpath and
# stripped ldflags. Override for a local release-like build:
# `make build-cli VERSION=0.1.0-local`. Defaults to "dev" (matches the fallback in
# glx/cli_commands.go); GoReleaser sets it from the git tag. GoReleaser also injects
# main.commit/main.date (#384) -- the Makefile keeps to version-only so it stays
# git-independent.
VERSION ?= dev

build-cli: ## Build the glx binary to bin/ (override version with VERSION=...)
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/glx ./glx

build-website: ## Build the documentation site
	@echo "Building website..."
	cd website && npm run build

build: build-cli build-website ## Build CLI and website

## Code Quality
fmt: ## Format Go and website code
	@echo "Formatting Go code..."
	golangci-lint fmt
	@echo "Formatting website..."
	cd website && npm run format

lint: ## Run linters (Go + website)
	@echo "Linting Go code..."
	golangci-lint run ./...
	@echo "Linting website..."
	cd website && npm run lint

lint-fix: ## Run linters with automatic fixes
	@echo "Fixing Go code..."
	golangci-lint run --fix ./...
	@echo "Fixing website..."
	cd website && npm run lint:fix

# hmarr/codeowners is pinned via the tool directive in ci-tools/go.mod (same
# pattern as govulncheck). The CLI walks the working tree, not the git index,
# so gitignored artifacts (bin/, website/node_modules/, ...) are scanned too --
# harmless while the catch-all rule owns everything, just slower locally than
# in CI. It always exits 0, hence the explicit empty-output check.
lint-codeowners: ## Verify every file is matched by a .github/CODEOWNERS rule
	@unowned="$$(go tool -modfile=ci-tools/go.mod codeowners --unowned)"; \
	if [ -n "$$unowned" ]; then \
		echo "ERROR: files with no CODEOWNERS owner:"; \
		echo "$$unowned"; \
		exit 1; \
	fi; \
	echo "CODEOWNERS coverage OK"

fix: ## Run Go 1.26 modernizers on codebase
	go fix ./...

fix-diff: ## Preview Go 1.26 modernizer changes without applying
	go fix -diff ./...

## Testing
test: ## Run all tests
	go test -timeout 10m ./...

test-verbose: ## Run all tests with verbose output
	go test -v -timeout 10m ./...

bench: ## Run benchmarks
	go test -bench=. -benchmem -count=6 -run='^$$' -timeout 10m ./glx/... ./go-glx/... > bench.txt
	@cat bench.txt

test-race: ## Run tests with race detector
	CGO_ENABLED=1 go test -race -timeout 15m ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -timeout 10m -coverprofile=coverage/coverage.out ./...
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"
	@echo "Opening coverage report in browser..."
	@go tool cover -func=coverage/coverage.out | tail -n 1

## Module Management
mod-tidy: ## Tidy Go module dependencies
	go mod tidy
	@echo "go.mod and go.sum are tidy"

mod-verify: ## Verify Go module integrity
	go mod verify

tidy-check: ## Verify go.mod and go.sum are tidy
	go mod tidy -diff

ci-tools-tidy-check: ## Verify ci-tools/go.mod and ci-tools/go.sum are tidy
	go -C ci-tools mod tidy -diff

## Security
# govulncheck is pinned via the tool directive in ci-tools/go.mod (single source of
# truth, shared with .github/workflows/security.yml); -modfile keeps its graph out
# of the main module. gosec is intentionally NOT here — its autofix package drags in
# a heavy Cloud-SDK/grpc/otel tree, so CI keeps it on a version-pinned `go install`
# (see ci-tools/README.md). Its pin lives in .gosec-version (single source of truth,
# shared with .github/workflows/security.yml).
vulncheck: ## Run govulncheck against the Go vulnerability DB (pinned via ci-tools/go.mod)
	go tool -modfile=ci-tools/go.mod govulncheck ./...

gosec: ## Run gosec static security analysis (pinned via .gosec-version)
	@v="$$(tr -d '[:space:]' < .gosec-version)"; \
	printf '%s' "$$v" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "invalid .gosec-version: $$v (expected vMAJOR.MINOR.PATCH)" >&2; exit 1; }; \
	go run "github.com/securego/gosec/v2/cmd/gosec@$$v" -quiet ./...

## Licensing
# go-licenses is intentionally NOT in ci-tools/go.mod — its tree pulls
# go.opencensus.io, x/net, and licenseclassifier, which would expose
# dependency-review to advisories in build-time-only code (see ci-tools/README.md).
# Keep the version and allowlist in sync with .github/workflows/license-compliance.yml.
# Source-file SPDX compliance is checked separately in CI via `reuse lint`
# (fsfe/reuse-action); run it locally with: pipx run reuse lint
license-check: ## Check Go dependency licenses against the Apache-2.0-compatible allowlist
	go run github.com/google/go-licenses/v2@v2.0.1 check ./... --allowed_licenses=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC,MPL-2.0

## Specification
check-schemas: ## Validate JSON schema files
	@node specification/validate-schemas.mjs

check-drift-allowlist: ## Validate .claude/drift-allowlist.yaml against its schema
	@node specification/validate-drift-allowlist.mjs

check-code-drift: ## Deterministically detect go-glx type vs JSON-schema drift
	@go run ./tools/driftcheck

check-memory-drift: ## Deterministically detect CLAUDE.md/AGENTS.md drift vs the repo
	@go run ./tools/memcheck

test-scripts: ## Unit-test the Node drift-check scripts (spec-schema parser + schema-compat classifier)
	@node --test scripts/drift-checks/spec-schema-drift.test.mjs specification/schema-compat.test.mjs

## Example Validation
validate-examples: build-cli ## Validate all example archives
	@for dir in docs/examples/*/; do \
	  echo "Validating $$dir..."; \
	  ./bin/glx validate "$$dir" || exit 1; \
	done

## Documentation
docs-cli: build-cli ## Regenerate per-command CLI reference under docs/cli/
	@./bin/glx docs --output ./docs/cli/

## Release
release-snapshot: ## Build cross-platform binaries locally (no publish)
	goreleaser release --snapshot --clean

## Changelog
# changie version pin — bump alongside any .changie.yaml format changes.
CHANGIE_VERSION ?= v1.24.0

# Ensure the *pinned* changie is runnable, installing it on demand, and export
# its absolute path as $CHANGIE for the recipe and for
# scripts/check-changelog-fragments.sh (which falls back to a PATH lookup when
# the variable is unset, e.g. in CI where the workflow installs changie itself).
#
# A changie already on PATH is only used when it IS $(CHANGIE_VERSION);
# otherwise the pinned build is installed and used, so `make changelog` renders
# the same fragments on every machine. The version comes from `go version -m`
# (the module version recorded in the binary), not `changie --version`: a
# `go install` build reports "vdev" for the latter, so comparing against it
# would reinstall on every single run. The first `go version -m` doubles as the
# probe for Windows, where `command -v changie` resolves to an extension-less
# path that `go` cannot stat — an empty result means retry with GOEXE appended.
#
# Exporting a path rather than extending PATH is deliberate, and fixes two
# separate bugs in doing so:
#
#   * `go env GOPATH` is a *list*. "$(go env GOPATH)/bin" builds a bogus
#     directory ("/path/one:/path/two/bin") for anyone with more than one
#     entry, and the separator is ';' on Windows where the entries themselves
#     contain ':' — make has no portable way to split it. Naming the
#     destination via GOBIN on the install sidesteps the question entirely.
#   * A Windows path cannot go on PATH under git-bash at all: PATH is
#     colon-separated there, so "C:/repo/bin" splits into "C" and "/repo/bin".
#
# The install dir is GOBIN when set, else this repo's ./bin; `go env GOEXE`
# supplies the .exe suffix on Windows. `make clean` removes ./bin, so a
# subsequent run reinstalls.
define ensure_changie
	CHANGIE="$$(command -v changie || true)"; \
	if [ -n "$$CHANGIE" ] && [ -z "$$(go version -m "$$CHANGIE" 2>/dev/null)" ]; then \
		CHANGIE="$$CHANGIE$$(go env GOEXE)"; \
	fi; \
	if [ "$$(go version -m "$$CHANGIE" 2>/dev/null | awk '$$1 == "mod" { print $$3; exit }')" != "$(CHANGIE_VERSION)" ]; then \
		GO_BIN_DIR="$$(go env GOBIN)"; \
		if [ -z "$$GO_BIN_DIR" ]; then GO_BIN_DIR="$(CURDIR)/bin"; fi; \
		echo "Installing changie $(CHANGIE_VERSION) into $$GO_BIN_DIR via 'go install'..."; \
		GOBIN="$$GO_BIN_DIR" go install github.com/miniscruff/changie@$(CHANGIE_VERSION); \
		CHANGIE="$$GO_BIN_DIR/changie$$(go env GOEXE)"; \
	fi; \
	export CHANGIE;
endef

changelog: ## Add a changelog fragment for a change (interactive `changie new`)
	@$(ensure_changie) \
	"$$CHANGIE" new

changelog-check: ## Validate changie fragments parse and carry an issue/PR reference
	@$(ensure_changie) \
	bash scripts/check-changelog-fragments.sh

## Link Checking
check-links: ## Validate internal markdown links
	@bash scripts/check-links.sh

## Cleanup
clean: ## Remove build artifacts
	rm -rf bin
	rm -rf coverage
	rm -rf dist
	rm -rf website/.vitepress/dist
	rm -rf website/.vitepress/cache
