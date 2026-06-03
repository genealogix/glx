# GENEALOGIX Makefile
.PHONY: help check build build-cli build-website install-deps install-hooks lint lint-fix fix fix-diff test test-verbose test-race test-coverage bench mod-tidy mod-verify tidy-check tools-tidy-check clean fmt check-schemas check-drift-allowlist check-links validate-examples docs-cli release-snapshot vulncheck

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
check: tidy-check tools-tidy-check lint test check-schemas check-drift-allowlist check-links validate-examples ## Run all checks (mirrors CI)
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

tools-tidy-check: ## Verify tools/go.mod and tools/go.sum are tidy
	go -C tools mod tidy -diff

## Security
# govulncheck is pinned via the tool directive in tools/go.mod (single source of
# truth, shared with .github/workflows/security.yml); -modfile keeps its graph out
# of the main module. gosec is intentionally NOT here — its autofix package drags in
# a heavy Cloud-SDK/grpc/otel tree, so CI keeps it on a version-pinned `go install`
# (see tools/README.md). Run gosec ad hoc with:
#   go run github.com/securego/gosec/v2/cmd/gosec@v2.22.4 -quiet ./...
vulncheck: ## Run govulncheck against the Go vulnerability DB (pinned via tools/go.mod)
	go tool -modfile=tools/go.mod govulncheck ./...

## Specification
check-schemas: ## Validate JSON schema files
	@node specification/validate-schemas.mjs

check-drift-allowlist: ## Validate .claude/drift-allowlist.yaml against its schema
	@node specification/validate-drift-allowlist.mjs

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
