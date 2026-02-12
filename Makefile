# GENEALOGIX Makefile
.PHONY: help build build-cli build-website install-deps lint lint-fix test test-verbose test-coverage clean fmt check-schemas

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

## Build
build-cli: ## Build the glx binary to bin/
	@mkdir -p bin
	go build -o bin/glx ./glx

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

## Testing
test: ## Run all tests
	go test ./...

test-verbose: ## Run all tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"
	@echo "Opening coverage report in browser..."
	@go tool cover -func=coverage/coverage.out | tail -n 1

## Specification
check-schemas: ## Validate JSON schema files have required metadata
	@fail=0; \
	for f in $$(find specification/schema -name '*.json'); do \
		for field in '"\$$schema"' '"\$$id"' '"title"' '"description"'; do \
			if ! grep -q $$field "$$f"; then \
				echo "missing $$field in $$f"; fail=1; \
			fi; \
		done; \
	done; \
	if [ "$$fail" -eq 1 ]; then \
		echo "schema validation failed"; exit 1; \
	else \
		echo "All schema files contain required metadata"; \
	fi

## Cleanup
clean: ## Remove build artifacts
	rm -rf bin
	rm -rf coverage
	rm -rf website/.vitepress/dist
	rm -rf website/.vitepress/cache
