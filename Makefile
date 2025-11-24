# GENEALOGIX Makefile
.PHONY: build build-cli build-website install-deps lint lint-fix test test-verbose test-coverage clean fmt

# Install dependencies - Go modules and npm packages
install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	@echo "Installing website dependencies..."
	cd website && npm install

# Build CLI - builds the glx binary to bin directory
build-cli:
	@mkdir -p bin
	go build -o bin/glx ./glx

# Build website - builds the documentation site
build-website:
	@echo "Building website..."
	cd website && npm run build

# Build - builds both CLI and website
build: build-cli build-website

fmt:
	@echo "Formatting Go code..."
	golangci-lint fmt
	@echo "Formatting website..."
	cd website && npm run format

# Lint target - runs golangci-lint and eslint
lint:
	@echo "Linting Go code..."
	golangci-lint run ./...
	@echo "Linting website..."
	cd website && npm run lint

# Lint-fix target - runs golangci-lint and eslint with automatic fixes
lint-fix:
	@echo "Fixing Go code..."
	golangci-lint run --fix ./...
	@echo "Fixing website..."
	cd website && npm run lint:fix

# Test target - runs all tests
test:
	go test ./...

# Test-verbose target - runs all tests with verbose output
test-verbose:
	go test -v ./...

# Test-coverage target - runs tests with coverage report generation
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -coverprofile=coverage/coverage.out ./...
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated at coverage/coverage.html"
	@echo "Opening coverage report in browser..."
	@go tool cover -func=coverage/coverage.out | tail -n 1

# Clean target - removes build artifacts
clean:
	rm -rf bin
	rm -rf coverage
	rm -rf website/.vitepress/dist
	rm -rf website/.vitepress/cache
