---
title: Development Setup Guide
description: Set up your development environment for GENEALOGIX specification and tools
layout: doc
---

# Development Setup Guide

This guide helps you set up a development environment for working on the GENEALOGIX specification and tools.

## Prerequisites

### Required Software

- **Go 1.25+**: For CLI tool development
- **Git**: For version control
- **Node.js 18+**: For static site generation (optional)

### Install Go

```bash
# macOS
brew install go

# Ubuntu/Debian
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify
go version
```

## Repository Setup

### Clone and Build

```bash
# Clone repository
git clone https://github.com/genealogix/glx.git
cd spec

# Download dependencies
go mod download

# Build CLI tool
cd glx
go build

# Verify installation
./glx --version
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./glx/...
go test ./lib/...

# Run with verbose output
go test -v ./glx/...
```

## Development Environment

### IDE Configuration

**Visual Studio Code** (recommended):

Install extensions:
- Go (golang.go)
- YAML (redhat.vscode-yaml)

Configure YAML validation in `.vscode/settings.json`:
```json
{
  "yaml.validate": true,
  "yaml.format.enable": true,
  "go.useLanguageServer": true
}
```

### Git Configuration

```bash
# Basic setup
git config user.name "Your Name"
git config user.email "your.email@example.com"

# Recommended settings
git config pull.rebase true
git config core.autocrlf false  # Important for cross-platform
```

## Development Workflows

### CLI Development

```bash
# Make changes to CLI code
vim glx/cmd_validate.go

# Build and test
cd glx
go build
./glx validate ../docs/examples/basic-family/

# Run tests
go test -v
```

### Schema Development

```bash
# Edit schema
vim specification/schema/v1/person.schema.json

# Rebuild CLI (schemas are embedded)
cd glx
go build

# Test against examples
./glx validate ../docs/examples/complete-family/

# Run schema validation tests
go test -run TestRunCheckSchemas
```

### Adding Test Data

```bash
# Add valid test file
vim glx/testdata/valid/new-test.glx

# Add invalid test file
vim glx/testdata/invalid/new-test.glx

# Run tests
go test ./glx/... -v
```

## Testing

### Run Test Suite

```bash
# All tests
go test ./...

# Specific tests
go test ./glx/... -run TestValidateGLXFile
go test ./lib/... -run TestGenerateTestData

# With coverage
go test ./... -cover

# With race detection
go test ./... -race
```

### Validate Examples

```bash
# Validate all examples
cd glx
./glx validate ../docs/examples/

# Validate specific example
./glx validate ../docs/examples/basic-family/
```

### Check Schemas

```bash
# Ensure all schemas have required metadata
cd glx
./glx check-schemas
```

## Building for Release

### Local Build

```bash
cd glx
go build -o glx
```

### Cross-Platform Builds

Uses GoReleaser (automated in CI):

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test release build locally
goreleaser build --snapshot --clean
```

Releases are automated via GitHub Actions on tag push.

## Contributing Workflow

### Standard Process

```bash
# 1. Create feature branch
git checkout -b feature/your-feature

# 2. Make changes
# Edit code, tests, documentation

# 3. Run tests
go test ./...

# 4. Validate examples
cd glx && ./glx validate ../docs/examples/

# 5. Commit changes
git add .
git commit -m "Description of changes"

# 6. Push and create PR
git push origin feature/your-feature
```

### Before Submitting PR

- [ ] All tests pass (`go test ./...`)
- [ ] Examples validate (`glx validate docs/examples/`)
- [ ] Documentation updated
- [ ] Commit messages are clear

## Troubleshooting

### Go Module Issues

```bash
# Clear cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify module
go mod verify
```

### Build Issues

```bash
# Clean build artifacts
go clean

# Rebuild with verbose output
go build -v

# Check for missing dependencies
go mod tidy
```

### Test Failures

```bash
# Run specific failing test
go test ./glx/... -run TestName -v

# Check test data files
ls -la glx/testdata/valid/
ls -la glx/testdata/invalid/

# Validate test file manually
cd glx
./glx validate testdata/valid/person-minimal.glx
```

## Getting Help

- [GitHub Issues](https://github.com/genealogix/glx/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Questions and discussions
- [Specification](../../specification/README.md) - Formal specification
- [CLI README](../../glx/README.md) - CLI tool documentation
