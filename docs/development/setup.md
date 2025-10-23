# Development Setup Guide

This guide helps you set up a development environment for working on the GENEALOGIX specification and tools.

## Prerequisites

### Required Software

**Core Requirements:**
- **Go 1.19+**: For CLI tool development
- **Git**: For version control
- **Node.js 18+**: For schema validation and testing
- **Python 3.8+**: For test scripts and validation tools

**Optional but Recommended:**
- **Docker**: For containerized development
- **Visual Studio Code**: With YAML and Go extensions
- **GitHub CLI**: For repository management

### Install Required Tools

#### Go Installation
```bash
# macOS
brew install go

# Ubuntu/Debian
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc

# Verify
go version
```

#### Node.js Installation
```bash
# Using Node Version Manager (recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
nvm install 18
nvm use 18

# Verify
node --version
npm --version
```

#### Python Installation
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install python3 python3-pip python3-venv

# macOS
brew install python3

# Verify
python3 --version
pip3 --version
```

## Repository Setup

### 1. Clone the Repository
```bash
git clone https://github.com/genealogix/spec.git
cd spec
```

### 2. Set Up Go Module
```bash
# Download dependencies
go mod download

# Verify module setup
go mod verify
```

### 3. Install Development Tools
```bash
# Install glx CLI tool
go install ./glx

# Install schema validation tools
npm install -g ajv-cli

# Set up Python virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements-dev.txt  # If available
```

## Development Environment

### 1. IDE Configuration

**Visual Studio Code:**
```json
// .vscode/settings.json
{
  "go.toolsManagement.checkForUpdates": "local",
  "go.useLanguageServer": true,
  "yaml.schemas": {
    "schema/v1/person.schema.json": ["persons/*.glx"],
    "schema/v1/event.schema.json": ["events/*.glx"],
    "schema/v1/place.schema.json": ["places/*.glx"]
  },
  "yaml.validate": true,
  "yaml.format.enable": true
}
```

**Recommended Extensions:**
- Go (Google)
- YAML (Red Hat)
- GitLens
- Prettier
- JSON Schema Validator

### 2. Git Configuration
```bash
# Set up Git for development
git config user.name "Your Name"
git config user.email "your.email@example.com"

# Enable useful features
git config pull.rebase true
git config rebase.autostash true

# For Windows developers
git config core.autocrlf false
```

### 3. Pre-commit Hooks (Future)
```bash
# Install pre-commit hooks when available
pip install pre-commit
pre-commit install

# This will run validation before each commit
# - YAML syntax checking
# - Schema validation
# - File naming validation
```

## Development Workflows

### 1. Running Tests

**Full Test Suite:**
```bash
# Run all validation tests
cd test-suite
./run-tests.sh

# Run specific test categories
./run-tests.sh valid
./run-tests.sh invalid

# Run with specific validator
./run-tests.sh --validator /path/to/glx
```

**Schema Validation:**
```bash
# Validate all JSON schemas
npm install -g ajv-cli
find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;

# Test schema compilation
ajv compile -s schema/meta/schema.schema.json
```

**Example Validation:**
```bash
# Validate example archives
glx validate examples/complete-family/
glx validate examples/minimal/
glx validate examples/basic-family/
```

### 2. CLI Development

**Build and test the CLI:**
```bash
# Build the CLI
cd glx
go build -o ../bin/glx .

# Test CLI functionality
../bin/glx --help
../bin/glx init test-repo
../bin/glx validate test-repo/
```

**Development cycle:**
```bash
# Edit source code
vim glx/main.go

# Build and test
go build -o bin/glx glx/main.go
bin/glx validate examples/complete-family/

# Run tests
go test ./...
```

### 3. Schema Development

**Validate schema changes:**
```bash
# Check schema syntax
ajv compile -s schema/v1/person.schema.json

# Test against examples
glx validate examples/complete-family/persons/

# Validate schema references
go run glx/main.go check-schemas
```

**Schema testing:**
```bash
# Test schema compilation
find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;

# Test with test suite
cd test-suite
./run-tests.sh --validator ../bin/glx
```

## Testing Framework

### 1. Test Suite Structure

**Valid Tests:**
```bash
test-suite/valid/
├── person-minimal.glx        # Minimal valid person
├── person-complete.glx       # Person with all optional fields
├── event-birth.glx          # Standard birth event
├── place-hierarchy.glx      # Place with parent
├── citation-quality3.glx    # Primary evidence citation
└── assertion-evidence.glx   # Assertion with evidence chain
```

**Invalid Tests:**
```bash
test-suite/invalid/
├── person-missing-id.glx     # Missing required field
├── person-bad-format.glx     # Invalid ID format
├── event-no-place.glx       # Event without required place
├── citation-no-source.glx    # Citation without source
└── place-circular-ref.glx    # Circular place reference
```

### 2. Adding New Tests

**Create valid test:**
```yaml
# test-suite/valid/person-with-nickname.glx
# TEST: person-with-nickname
# EXPECT: valid
# DESCRIPTION: Person with nickname and alternative names

id: person-a1b2c3d4
version: "1.0"
type: person
name:
  given: John
  surname: Smith
  display: John Smith
  nickname: Johnny
alternative_names:
  - John Henry Smith
  - J.H. Smith
```

**Create invalid test:**
```yaml
# test-suite/invalid/person-missing-name.glx
# TEST: person-missing-name
# EXPECT: invalid
# ERROR: "name field is required"
# DESCRIPTION: Person entity must have a name field

id: person-b2c3d4e5
version: "1.0"
type: person
# Missing required name field
```

**Test file format:**
```yaml
# Comments at top of file
# TEST: descriptive-name
# EXPECT: valid|invalid
# ERROR: "expected error message"
# DESCRIPTION: What this test validates
```

### 3. Running Test Categories

**Comprehensive testing:**
```bash
# All tests
./run-tests.sh

# Valid tests only
./run-tests.sh valid

# Invalid tests only
./run-tests.sh invalid

# With custom validator
./run-tests.sh --validator /path/to/your-glx-implementation

# Verbose output
./run-tests.sh --verbose

# Stop on first failure
./run-tests.sh --fail-fast
```

## Documentation Development

### 1. Building Documentation

**Markdown validation:**
```bash
# Check markdown links
npm install -g markdown-link-check
find . -name "*.md" -exec markdown-link-check {} \;

# Check with project config
markdown-link-check .github/markdown-link-check-config.json
```

**Specification validation:**
```bash
# Check specification completeness
# Look for TODO comments
grep -r "TODO" specification/

# Check for broken internal links
grep -r "\.\./\.\./" specification/ | grep -v "^Binary"
```

### 2. Example Validation

**Validate all examples:**
```bash
# Check examples directory
glx validate examples/

# Test specific examples
glx validate examples/complete-family/
glx validate examples/minimal/
glx validate examples/basic-family/

# Check example completeness
# Verify all entity types are demonstrated
# Check evidence chains are complete
```

## Continuous Integration

### 1. Local CI Testing

**Run full CI pipeline locally:**
```bash
# Schema validation
npm install -g ajv-cli
ajv compile -s schema/meta/schema.schema.json
find schema/v1 -name "*.schema.json" -exec ajv compile -s {} \;

# Example validation
glx validate examples/

# Test suite
cd test-suite
./run-tests.sh

# Link checking
markdown-link-check .github/markdown-link-check-config.json
```

### 2. GitHub Actions

**Understand CI workflow:**
```yaml
# .github/workflows/validate-spec.yml
jobs:
  validate-schemas:     # JSON Schema validation
  validate-examples:    # Example archive validation
  test-conformance:     # Test suite execution
  check-links:         # Markdown link checking
```

**Test CI locally:**
```bash
# Simulate CI environment
# Run all validation steps
# Check for common CI failures
```

## Debugging Tools

### 1. Validation Debugging

**Verbose validation:**
```bash
# Get detailed error messages
glx validate --verbose examples/complete-family/

# Check specific file
glx validate --debug persons/person-example.glx

# Schema validation details
ajv validate -s schema/v1/person.schema.json persons/person-example.glx
```

**Debug YAML issues:**
```bash
# Check YAML syntax
python3 -c "import yaml; yaml.safe_load(open('file.glx'))"

# Validate with online tools
# Use YAML validators for complex structures

# Check indentation
cat -A file.glx  # Show tabs vs spaces
```

### 2. Git Debugging

**Find when issues were introduced:**
```bash
# Bisect for regressions
git bisect start
git bisect bad HEAD
git bisect good v1.0.0
git bisect run ./run-tests.sh

# Find who changed what
git blame specification/1-introduction.md

# See file history
git log --oneline --follow persons/person-example.glx
```

## Performance Optimization

### 1. Large Archive Testing

**Test with large datasets:**
```bash
# Generate test data
# python3 scripts/generate-large-test.py

# Validate performance
time glx validate large-test-archive/

# Profile validation
go tool pprof bin/glx cpu.prof
```

### 2. Memory and Speed

**Monitor resource usage:**
```bash
# Memory profiling
go build -o bin/glx-profile glx/main.go
./bin/glx-profile validate large-archive/

# CPU profiling
go tool pprof bin/glx-profile cpu.prof
```

## Contributing Workflow

### 1. Development Process

**Standard contribution workflow:**
```bash
# 1. Create issue or find existing issue
# 2. Create feature branch
git checkout -b feature/new-validation-rule

# 3. Implement changes
# Edit code, tests, documentation

# 4. Validate changes
glx validate
./run-tests.sh

# 5. Commit with good message
git add .
git commit -m "Add new validation rule for place coordinates

- Validate WGS84 coordinate format
- Add tests for valid/invalid coordinates
- Update documentation with examples
- Fixes issue #123"

# 6. Push and create PR
git push origin feature/new-validation-rule
```

### 2. Code Review Process

**Before submitting PR:**
- [ ] All tests pass
- [ ] Examples validate
- [ ] Documentation updated
- [ ] Schema changes validated
- [ ] Commit messages are descriptive

**PR template includes:**
- Description of changes
- Testing performed
- Breaking changes (if any)
- Documentation updates

## Troubleshooting

### Common Setup Issues

**Go module problems:**
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download

# Verify module
go mod verify
```

**Schema validation issues:**
```bash
# Check schema syntax
ajv compile -s schema/v1/person.schema.json

# Test with simple example
glx validate examples/minimal/persons/

# Check file permissions
ls -la schema/v1/
```

**Test suite issues:**
```bash
# Check test file permissions
chmod +x test-suite/run-tests.sh

# Verify test file format
head -5 test-suite/valid/person-minimal.glx

# Run with debug output
./run-tests.sh --verbose
```

### Getting Help

**Development community:**
- [GitHub Issues](https://github.com/genealogix/spec/issues) - Bug reports
- [GitHub Discussions](https://github.com/genealogix/spec/discussions) - Q&A
- [Contributing Guide](../../CONTRIBUTING.md) - Guidelines

**Debug information:**
```bash
# Collect system info
echo "Go version: $(go version)"
echo "Node version: $(node --version)"
echo "Python version: $(python3 --version)"
echo "Git version: $(git --version)"

# Repository status
git status
git log --oneline -5
```

This development setup provides everything needed to contribute to the GENEALOGIX specification and tools effectively.
