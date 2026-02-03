---
title: Contributing Guide
description: How to contribute to GENEALOGIX - for genealogists, developers, and everyone
layout: doc
---

# Contributing to GENEALOGIX

Thank you for your interest in contributing to GENEALOGIX! Whether you're a genealogist, developer, or both, we welcome contributions of all kinds.

## Table of Contents

- [How Can I Contribute?](#how-can-i-contribute)
- [Development Environment Setup](#development-environment-setup)
- [Code Organization](#code-organization)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)
- [Submitting Changes](#submitting-changes)
- [Good First Issues](#good-first-issues)
- [Code of Conduct](#code-of-conduct)

## How Can I Contribute?

### For Genealogists

- **Improve Examples**: Add real-world genealogy scenarios
- **Documentation**: Clarify genealogical concepts and best practices
- **Test Cases**: Contribute edge cases from your research experience
- **Use Case Reports**: Share how GENEALOGIX works (or doesn't) for your needs

### For Developers

- **Bug Fixes**: Fix issues in the CLI tool or validation logic
- **New Features**: Implement accepted proposals from GitHub issues/discussions
- **Performance**: Optimize validation speed and memory usage
- **Tooling**: Build integrations, converters, or utilities

### For Everyone

- **Bug Reports**: Use our [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml)
- **Feature Requests**: Use our [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml)
- **Documentation**: Fix typos, improve clarity, add examples
- **Community Support**: Help others in [Discussions](https://github.com/genealogix/glx/discussions)

## Development Environment Setup

### Prerequisites

**Required:**
- **Go 1.25+** - [Install Go](https://golang.org/doc/install)
- **Git** - [Install Git](https://git-scm.com/downloads)

**Optional but Recommended:**
- **VS Code** with Go extension
- **ajv-cli** for schema validation: `npm install -g ajv-cli`
- **yamllint** for YAML validation: `pip install yamllint`

### Initial Setup

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/YOUR_USERNAME/spec.git
cd spec

# 3. Add upstream remote
git remote add upstream https://github.com/genealogix/glx.git

# 4. Install CLI tool for testing
go install ./glx

# 5. Verify installation
glx --help

# 6. Run validation tests
glx validate glx/tests/valid/
```

### Development Workflow

```bash
# 1. Sync with upstream
git fetch upstream
git checkout main
git merge upstream/main

# 2. Create feature branch
git checkout -b feature/my-contribution

# 3. Make changes and test
# ... edit files ...
glx validate
go test ./...

# 4. Commit changes
git add .
git commit -m "Add feature X

Detailed explanation of changes and why they're needed."

# 5. Push and create PR
git push origin feature/my-contribution
# Open pull request on GitHub
```

## Code Organization

### Repository Structure

```
genealogix/glx/
├── specification/       # Specification documents (markdown)
│   ├── 1-introduction.md
│   ├── 2-core-concepts.md
│   ├── 3-archive-organization.md
│   ├── 4-entity-types/  # Per-entity specifications
│   ├── schema/          # JSON Schema definitions
│   │   ├── v1/          # Version 1 schemas
│   │   └── meta/        # Schema validation schemas
├── docs/                # User and developer documentation
│   ├── quickstart.md
│   ├── guides/          # User guides
│   ├── development/     # Developer guides
│   ├── diagrams/        # Architecture diagrams
│   └── examples/        # Working example archives
│       └── complete-family/ # Best starting point
└── glx/                 # CLI tool source code
    ├── main.go
    └── tests/           # Validation test cases
        ├── valid/       # Should pass validation
        └── invalid/     # Should fail validation
```

### Understanding the Architecture

**Key Concepts:**
1. **Specification Documents** (`specification/`) define behavior and requirements
2. **JSON Schemas** (`specification/schema/`) enforce structure and validation rules
3. **Examples** (`docs/examples/`) demonstrate practical usage
4. **Test Suite** (`glx/tests/`) ensures conformance

**Entity Types:**
- 9 core entity types: Person, Relationship, Event, Place, Source, Citation, Repository, Assertion, Media
- Each has: specification doc, JSON schema, examples, tests

**Evidence Hierarchy:**
- Repository → Source → Citation → Assertion
- All genealogical claims must be backed by evidence

## Testing Requirements

### Running Tests

```bash
# Run all tests (preferred - use Makefile)
make test

# Run Go tests directly
go test ./...

# Run benchmarks
go test -bench=. -benchmem ./glx/...

# Validate examples
cd glx && ./glx validate ../docs/examples/
```

### Writing New Tests

**Valid Test Cases** (`glx/tests/valid/`):
- Should pass validation
- Test happy paths and optional fields
- Include comments explaining what's being tested

```yaml
# glx/tests/valid/person-with-multiple-names.glx
persons:
  person-12345678:
    properties:
      name:
        - value: "John Smith"
          date: "1850"
          fields:
            given: "John"
            surname: "Smith"
        - value: "Juan Hernandez"  # Testing multilingual names
          date: "1880"
          fields:
            given: "Juan"
            surname: "Hernandez"
```

**Invalid Test Cases** (`glx/tests/invalid/`):
- Should fail validation with specific error
- Test boundary conditions and error handling
- Document expected error message

```yaml
# glx/tests/invalid/person-invalid-id-format.glx
# Expected error: ID must match pattern person-[a-f0-9]{8}
persons:
  invalid-id:
    properties:
      name:
        value: "Test Person"
```

### Test Coverage Requirements

- **Minimum**: 3 test cases per entity type (27 total for 9 entities)
- **Recommended**: Cover all optional fields, edge cases, and error conditions
- **Invalid tests**: At least 1 per common error type

## Documentation Standards

### Writing Style

**For Specification Documents:**
- Clear, precise, professional language
- Define technical terms on first use
- Include examples for complex concepts
- Use consistent terminology throughout

**For User Documentation:**
- Friendly, helpful tone
- Step-by-step instructions
- Real-world examples
- Anticipate common questions

### Markdown Conventions

```markdown
# Top-level title (only one per document)

## Major sections

### Subsections

**Bold** for emphasis and UI elements
*Italic* for terms and file names
`code` for commands, file names, and code elements

Code blocks with language:
\`\`\`bash
command here
\`\`\`

Tables for structured comparisons
Links with descriptive text: [see the guide](link)
```

**Internal Links (Specification Documents):**

Internal links in specification documents omit the `.md` file extension for VitePress compatibility:
- ✓ Good: `[Person Entity](4-entity-types/person)`
- ✗ Bad: `[Person Entity](4-entity-types/person.md)`

This works correctly in both VitePress-generated site and raw markdown viewers.

### Genealogical Standards

When documenting genealogical concepts:

- **Use Standard Terminology**: Follow [Genealogical Proof Standard](https://www.bcgcertification.org/resources/standard.html) where applicable
- **Evidence Quality**: Explain primary/secondary and direct/indirect evidence
- **Citation Standards**: Follow [Evidence Explained](https://www.evidenceexplained.com/) principles
- **Date Formats**: Use ISO 8601 (YYYY-MM-DD) or historical variations documented
- **Place Names**: Use historical names with modern equivalents

## Proposing Major Changes

Major changes to the specification are discussed through GitHub issues and discussions.

### When to Create a Proposal

**Proposal Required (via GitHub Issue):**
- Changes to core data model or entity types
- New required fields or breaking changes
- Changes to validation rules or file format
- Git workflow convention changes

**Proposal Not Required:**
- Bug fixes
- Documentation improvements
- New examples
- Minor clarifications

### Proposal Workflow

1. **Create Issue**: Open a GitHub issue describing your proposed change
2. **Discussion**: Use GitHub Discussions for extended community discussion
3. **Community Review**: Community reviews and comments (minimum 7 days)
4. **Decision**: Maintainers accept, reject, or request changes
5. **Implementation**: After acceptance, implement the change via pull request
6. **Documentation**: Update relevant documentation as part of the implementation

For questions or to start a discussion, use [GitHub Discussions](https://github.com/genealogix/glx/discussions).

## Submitting Changes

### Pull Request Process

1. **Create Issue First**: For non-trivial changes, create an issue to discuss
2. **Branch Naming**: Use descriptive names like `feature/add-xyz` or `fix/issue-123`
3. **Commit Messages**: Follow [Conventional Commits](https://www.conventionalcommits.org/)
   ```
   type(scope): brief description
   
   Longer explanation if needed.
   
   Fixes #123
   ```
4. **PR Description**: Use the [PR template](.github/PULL_REQUEST_TEMPLATE.md)
5. **Testing**: Ensure all tests pass and add new tests for new features
6. **Documentation**: Update relevant documentation

### Review Process

- Maintainers will review PRs within 3-5 business days
- Address review comments promptly
- Be open to feedback and iteration
- Squash commits if requested before merge

### What Makes a Good PR

✅ **Good PR:**
- Focused on a single issue or feature
- Includes tests and documentation
- Passes all CI checks
- Has clear commit messages
- References related issues

❌ **Needs Improvement:**
- Multiple unrelated changes
- No tests or documentation updates
- Fails validation or tests
- Unclear purpose or description

## Good First Issues

Looking for where to start? Check issues labeled `good-first-issue`:

**Documentation:**
- Fix typos or improve clarity
- Add examples to specification docs
- Write migration guides from other formats

**Testing:**
- Add test cases for edge cases
- Improve test coverage for entity types
- Document expected validation errors

**Examples:**
- Create examples for specific scenarios
- Add README files to examples
- Improve example documentation

**Tooling:**
- Add helpful error messages to CLI
- Improve validation output formatting
- Add progress indicators for large archives

## Community Guidelines

### Communication Channels

- **GitHub Issues**: Bug reports, feature requests
- **GitHub Discussions**: Questions, ideas, show-and-tell
- **Pull Requests**: Code and documentation contributions

### Getting Help

- Check existing issues and discussions first
- Provide complete information when asking questions
- Be patient and respectful
- Help others when you can

### Recognition

Contributors are recognized in:
- GitHub contributor graph
- Release notes
- Acknowledgments in major documentation updates

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please read and follow our [Code of Conduct](CODE_OF_CONDUCT.md).

### Expected Behavior

- Be respectful and constructive
- Welcome newcomers
- Focus on what's best for the community
- Show empathy and kindness

### Unacceptable Behavior

- Harassment or discrimination
- Trolling or insulting comments
- Publishing others' private information
- Any conduct inappropriate in a professional setting

## Building for Release

Releases use GoReleaser (automated in CI):

```bash
# Install GoReleaser
go install github.com/goreleaser/goreleaser@latest

# Test release build locally
goreleaser build --snapshot --clean
```

Releases are automated via GitHub Actions on tag push.

## Troubleshooting

### Go Module Issues

```bash
go clean -modcache    # Clear cache
go mod download       # Re-download dependencies
go mod verify         # Verify module
```

### Build Issues

```bash
go clean              # Clean build artifacts
go build -v           # Rebuild with verbose output
go mod tidy           # Check for missing dependencies
```

## Questions?

- **Technical questions**: [GitHub Discussions](https://github.com/genealogix/glx/discussions)
- **Private concerns**: Contact maintainers at conduct@genealogix.io

---

Thank you for contributing to GENEALOGIX! Together we're building a better future for genealogical research.

