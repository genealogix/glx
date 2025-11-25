---
title: Testing Guide
description: Testing framework and practices for GENEALOGIX implementations
layout: doc
---

# Testing Guide

This guide explains the testing framework for GENEALOGIX implementations.

## Test Suite Overview

Tests are written in Go and located in the `glx/` directory:

```
glx/
├── main_test.go          # CLI command tests
├── validate_test.go      # Validation logic tests
└── testdata/
    ├── valid/            # Files that must pass validation
    └── invalid/          # Files that must fail validation
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run glx package tests
go test ./glx/...

# Run lib package tests
go test ./lib/...

# Verbose output
go test -v ./glx/...

# Run specific test
go test ./glx/... -run TestValidateGLXFile

# With coverage
go test ./... -cover
```

### CI Testing

Tests run automatically on every commit via GitHub Actions:
- Go tests for all packages
- Example validation
- Schema metadata validation

See `.github/workflows/test.yml` for CI configuration.

## Test Categories

### Valid Test Files

**Location**: `glx/testdata/valid/`

**Purpose**: Files that must pass validation

### Invalid Test Files

**Location**: `glx/testdata/invalid/`

**Purpose**: Files that must fail validation

## Writing Tests

### Unit Tests

Located in `*_test.go` files:

### Test Data Files

Test files are standard GLX files:

```yaml
# glx/testdata/valid/person-minimal.glx
persons:
  person-abc12345:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
```

**Guidelines**:
- Use realistic but minimal data
- Include comments explaining what's being tested
- Use consistent ID formats
- Keep files focused on one test case

## Test Coverage

### Entity Types

All entity types have test coverage:
- Person
- Relationship
- Event
- Place
- Source
- Citation
- Repository
- Assertion
- Media

### Validation Rules

Tests cover:
- **Required fields**: Missing required properties
- **Field formats**: Invalid ID formats, invalid patterns
- **References**: Broken entity references, broken vocabulary references
- **Duplicates**: Duplicate entity IDs across files
- **Schema compliance**: All JSON schema rules

### Integration Tests

- **Example validation**: All examples in `docs/examples/` must validate
- **Cross-file references**: Multi-file archives with references
- **Vocabulary loading**: Custom vocabularies from any file
- **Merge logic**: Combining multiple files with duplicate detection

## Adding New Tests

### Process

1. **Identify what to test**: New feature, bug fix, or edge case
2. **Choose location**: `testdata/valid/` or `testdata/invalid/`
3. **Create test file**: Follow naming conventions
4. **Add to test suite**: File is automatically discovered
5. **Run tests**: `go test ./glx/...`

### Test File Naming

Use descriptive names that indicate what is being tested:
- Format: `entity-scenario.glx`
- Be specific about what the test validates

## Debugging Tests

### Test Failures

```bash
# Run failing test with verbose output
go test ./glx/... -run TestName -v

# Run single test file
cd glx
./glx validate testdata/valid/person-minimal.glx

# Check test file syntax
cat testdata/valid/person-minimal.glx
```

### Common Issues

**Test file not found:**
- Check file path
- Ensure file has `.glx` extension
- Verify file is in `testdata/` directory

**Unexpected validation errors:**
- Check JSON schema for recent changes
- Verify field names match schema
- Ensure references use singular names

**Test passes when it should fail:**
- Check test expectations
- Verify validation logic
- Review error handling

## Performance Testing

### Benchmarking

```bash
# Run benchmarks
go test -bench=. ./glx/...

# With memory profiling
go test -bench=. -benchmem ./glx/...

# Profile specific function
go test -bench=BenchmarkValidate -cpuprofile=cpu.prof ./glx/...
go tool pprof cpu.prof
```

### Large Archive Testing

```bash
# Generate test data
cd glx
./glx init test-archive --create-test-data 100

# Validate performance
time ./glx validate test-archive/

# Clean up
rm -rf test-archive/
```

## CI/CD Integration

### GitHub Actions

Tests run automatically on:
- Push to any branch
- Pull request creation
- Tag creation (triggers release)

See `.github/workflows/test.yml` for configuration.

### Local CI Simulation

```bash
# Run same tests as CI
go test ./...

# Validate examples
cd glx
./glx validate ../docs/examples/

# Check schemas
./glx check-schemas
```

## Test Maintenance

### Updating Tests

When specification changes:
1. Update affected test files
2. Add new test cases for new features
3. Update expected error messages
4. Remove obsolete tests

### Test Review

Before committing tests:
- [ ] Tests pass locally
- [ ] Test names are descriptive
- [ ] Test files are minimal
- [ ] Documentation is updated if needed

## See Also

- [Architecture Guide](architecture.md) - System architecture
- [Setup Guide](setup.md) - Development environment
- [CLI README](../../glx/README.md) - CLI tool documentation
- [Specification](../../specification/README.md) - Formal specification
