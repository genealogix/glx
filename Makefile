# GENEALOGIX Makefile
.PHONY: build lint lint-fix test test-verbose clean

# Build target - builds the glx binary to bin directory
build:
	@mkdir -p bin
	go build -o bin/glx ./glx

# Lint target - runs golangci-lint
lint:
	golangci-lint run ./...

# Lint-fix target - runs golangci-lint with automatic fixes
lint-fix:
	golangci-lint run --fix ./...

# Test target - runs all tests
test:
	go test ./...

# Test-verbose target - runs all tests with verbose output
test-verbose:
	go test -v ./...

# Clean target - removes build artifacts
clean:
	rm -rf bin
