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
AJV := npx --yes --package=ajv-cli --package=ajv-formats ajv

check-schemas: ## Validate JSON schema files
	@echo "Validating schemas against meta-schema..."
	@$(AJV) validate -s specification/schema/meta/schema.schema.json -d "specification/schema/v1/*.schema.json" -c ajv-formats
	@$(AJV) validate -s specification/schema/meta/schema.schema.json -d "specification/schema/v1/vocabularies/*.schema.json" -c ajv-formats
	@echo "Compiling schemas..."
	@$(AJV) compile -s specification/schema/meta/schema.schema.json -c ajv-formats
	@find specification/schema/v1 -name "*.schema.json" ! -name "glx-file.schema.json" -exec $(AJV) compile -s {} -c ajv-formats \;
	@$(AJV) compile -s specification/schema/v1/glx-file.schema.json \
		-r "specification/schema/v1/person.schema.json" \
		-r "specification/schema/v1/event.schema.json" \
		-r "specification/schema/v1/relationship.schema.json" \
		-r "specification/schema/v1/place.schema.json" \
		-r "specification/schema/v1/source.schema.json" \
		-r "specification/schema/v1/citation.schema.json" \
		-r "specification/schema/v1/repository.schema.json" \
		-r "specification/schema/v1/assertion.schema.json" \
		-r "specification/schema/v1/media.schema.json" \
		-r "specification/schema/v1/vocabularies/*.schema.json" \
		-c ajv-formats

## Cleanup
clean: ## Remove build artifacts
	rm -rf bin
	rm -rf coverage
	rm -rf website/.vitepress/dist
	rm -rf website/.vitepress/cache
