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
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Documentation Standards](#documentation-standards)
- [Submitting Changes](#submitting-changes)
- [Developer Certificate of Origin (DCO)](#developer-certificate-of-origin-dco)
- [Proposing Major Changes](#proposing-major-changes)
- [AI-Generated Contributions](#ai-generated-contributions)
- [Code of Conduct](#code-of-conduct)
- [Security](#security)

## How Can I Contribute?

### For Genealogists

- **Improve Examples**: Add real-world genealogy scenarios
- **Documentation**: Clarify genealogical concepts and best practices
- **Test Cases**: Contribute edge cases from your research experience
- **Use Case Reports**: Share how GENEALOGIX works (or doesn't) for your needs

### For Developers

- **Bug Fixes**: Fix issues in the CLI tool or validation logic
- **New Features**: Implement accepted proposals from GitHub issues
- **Performance**: Optimize validation speed and memory usage
- **Tooling**: Build integrations, converters, or utilities

### For Everyone

- **Bug Reports**: Use our [bug report template](https://github.com/genealogix/glx/issues/new?template=bug_report.yml)
- **Feature Requests**: Use our [feature request template](https://github.com/genealogix/glx/issues/new?template=feature_request.yml)
- **Documentation**: Fix typos, improve clarity, add examples
- **Community Support**: Help others in [Discussions](https://github.com/genealogix/glx/discussions)

Looking for where to start? Check issues labeled [`good first issue`](https://github.com/genealogix/glx/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22).

## Development Environment Setup

### Prerequisites

- **Go 1.26+** ([install](https://golang.org/doc/install)) — the project uses Go 1.26 in go.mod
- **Git** ([install](https://git-scm.com/downloads))
- **Node.js** — for website builds (`npm install` in `website/`) and schema validation (`npm ci --prefix specification`)

### Dev Container

The easiest way to get started is with the included [Dev Container](https://containers.dev/):

```bash
# VS Code: Install the "Dev Containers" extension, then:
# Ctrl+Shift+P → "Dev Containers: Reopen in Container"

# GitHub Codespaces: Click "Code" → "Codespaces" → "Create codespace on main"
```

The container includes Go, Node.js, golangci-lint, and all other tooling pre-configured.

### Manual Setup

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/glx.git
cd glx
git remote add upstream https://github.com/genealogix/glx.git

# Install dependencies
make install-deps

# Build and verify
make build
./bin/glx --help

# Run tests
make test

# (Optional) Install schema validation tooling
npm ci --prefix specification
make check-schemas
```

### Makefile Reference

| Target | Description |
|--------|-------------|
| `make install-deps` | Install Go modules and npm packages |
| `make test` | Run all tests |
| `make test-verbose` | Run tests with verbose output |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run Go and website linters |
| `make lint-fix` | Run linters with auto-fix |
| `make fmt` | Format Go and website code |
| `make build` | Build CLI and website |
| `make build-cli` | Build just the `glx` binary to `bin/` |
| `make check-schemas` | Validate JSON schema files |
| `make check-links` | Validate internal markdown links |
| `make release-snapshot` | Build cross-platform binaries locally |
| `make clean` | Remove build artifacts |

## Development Workflow

### Fork and Direct-Push

**External contributors** use the fork workflow:

```bash
git fetch upstream
git checkout main
git merge upstream/main
git checkout -b feat/my-feature
# ... make changes ...
git push origin feat/my-feature
# Open PR from your fork
```

**Org members** can push branches directly:

```bash
git checkout main
git pull
git checkout -b feat/my-feature
# ... make changes ...
git push -u origin feat/my-feature
# Open PR
```

### Branch Naming

Use conventional prefixes:

```
feat/short-description
fix/short-description
docs/short-description
chore/short-description
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/). Valid types: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `perf`, `ci`.

```
feat: Add GEDCOM 7.0 EXID support
fix: Handle nil map in merge
docs: Update quickstart guide
```

Every commit must also carry a `Signed-off-by` trailer — see [Developer Certificate of Origin (DCO)](#developer-certificate-of-origin-dco) for how to add one and what you're attesting to.

## Testing

Prefer using the Makefile to run tests for consistency. `go test` directly is fine for targeted runs.

```bash
make test              # Run all tests
make test-verbose      # Verbose output
make test-coverage     # Coverage report
make check-schemas     # Validate JSON schemas
```

### Writing Tests

- Unit tests for all new functions
- Integration tests for full conversion paths
- E2E tests for CLI commands
- Test files live alongside source: `foo.go` → `foo_test.go`
- GEDCOM test files: `glx/testdata/gedcom/`
- Validation test fixtures: `glx/testdata/valid/` and `glx/testdata/invalid/`

### CI Checks

Every PR runs these checks automatically:

| Check | What it does |
|-------|--------------|
| **Validate Specification / test-conformance** | Go tests for `glx/` and `go-glx/` packages |
| **Validate Specification / validate-schemas** | JSON schema validation |
| **Validate Specification / validate-examples** | All example archives pass `glx validate` |
| **Security** | gosec, govulncheck, and npm audit |
| **lint-pr-title** | PR title follows conventional commits format |
| **dependency-review** | Blocks PRs introducing vulnerable dependencies |

All checks must pass before merge.

## Documentation Standards

### Writing Style

- **Specification docs**: Clear, precise, professional language. Define technical terms on first use.
- **User docs**: Friendly, step-by-step instructions with real-world examples.

### Internal Links

Specification documents omit the `.md` file extension for VitePress compatibility:
- Good: `[Person Entity](4-entity-types/person)`
- Bad: `[Person Entity](4-entity-types/person.md)`

### Genealogical Standards

- Follow [Genealogical Proof Standard](https://www.bcgcertification.org/resources/standard.html) terminology
- Citation standards per [Evidence Explained](https://www.evidenceexplained.com/)
- Dates in ISO 8601 (YYYY-MM-DD) or documented historical variations
- Use historical place names with modern equivalents

## Submitting Changes

### Pull Request Process

1. Create an issue first for non-trivial changes
2. Follow the [PR template](https://github.com/genealogix/glx/blob/main/.github/PULL_REQUEST_TEMPLATE.md)
3. Ensure all CI checks pass
4. Add tests for new features and bug fixes
5. Update documentation if behavior changes
6. Update `CHANGELOG.md` for user-facing changes (add to the unreleased section). Every entry must include an issue or PR reference — e.g. `(#123)`, `Fixes #123`, `Closes #123`, `(PR #456)`

### Review Process

- Maintainers will review PRs within 3-5 business days
- Address review comments promptly
- Be open to feedback and iteration

## Developer Certificate of Origin (DCO)

GENEALOGIX uses the [Developer Certificate of Origin (DCO) 1.1](https://developercertificate.org/) — the same lightweight attestation used by the Linux kernel, CNCF projects, and many other open-source projects. Rather than requiring a heavier Contributor License Agreement, there is no separate document to sign and no CLA bot to negotiate with. By signing off each commit, you attest that you wrote the change (or have the right to submit it) and that it can be contributed under the project's Apache 2.0 license. The DCO is an attestation of your right to contribute the code — it is **not** a copyright assignment. You retain copyright in your contribution. There is no automated DCO check today; sign-offs are trusted and verified manually during review.

### Signing off

Add a `Signed-off-by` trailer to every commit using the `-s` (or `--signoff`) flag:

```bash
git commit -s -m "feat: Add GEDCOM 7.0 EXID support"
```

This appends a line to the commit message using the identity from `git config user.name` and `git config user.email`:

```
Signed-off-by: Your Name <you@example.com>
```

Use the same name and email you use for other contributions so maintainers can reach you about your change. If `user.name` or `user.email` are wrong for this repository, set them locally before committing:

```bash
git config user.name "Your Name"
git config user.email "you@example.com"
```

### Fixing a missing sign-off

If you forgot `-s` on your most recent commit:

```bash
git commit --amend --signoff --no-edit
git push --force-with-lease
```

For a range of commits (e.g., the last 3):

```bash
git rebase --signoff HEAD~3
git push --force-with-lease
```

Only force-push branches you own. If others are collaborating on the branch, coordinate with them before rewriting history to avoid disrupting their work.

### The DCO text

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

## Proposing Major Changes

**Proposal required** (via GitHub Issue) for:
- Changes to core data model or entity types
- New required fields or breaking changes
- Changes to validation rules or file format

**No proposal needed** for:
- Bug fixes, documentation improvements, new examples, minor clarifications

### Proposal Workflow

1. **Create Issue**: Describe the proposed change
2. **Discussion**: Community reviews and comments (minimum 7 days for spec changes)
3. **Decision**: Maintainers accept, reject, or request changes
4. **Implementation**: After acceptance, submit a PR

## AI-Generated Contributions

**AI is welcome. Humans are accountable.**

GLX is a small project with limited maintainer capacity. Every issue, PR, and comment costs human time to triage. These guidelines exist to protect that time.

All contributions must reflect genuine understanding of the GLX spec and codebase. Contributors are fully responsible for everything they submit, regardless of how it was produced. If you cannot explain your change and respond to feedback about it, do not submit it.

### Autonomous Agents and Bots

Autonomous AI agents (OpenClaw, bounty bots, or similar) are **not permitted** to open issues, submit PRs, or post comments on any repository in the `genealogix` org. This includes agents acting on behalf of a human who is not actively supervising and reviewing each interaction.

Contributions that appear to be bot-generated will be closed without review and the account may be blocked.

### PRs and Issues

- Contributors are limited to **3 open PRs** at a time across all `genealogix` repositories
- Address all review comments on existing PRs before opening new ones

### AI-Assisted Development

Using AI tools (Copilot, Claude, ChatGPT, etc.) as part of your workflow is fine. The bar is the same as any contribution: you understand the problem, you've tested the change, and you can engage with review feedback.

Contributors SHOULD disclose substantial AI assistance via a commit trailer:

```
Assisted-by: Claude <noreply@anthropic.com>
```

`Co-authored-by` trailers added automatically by AI coding tools are also acceptable.

### Enforcement

Maintainers will close low-quality or bot-generated contributions without detailed explanation. Repeated violations will result in the account being blocked from the org.

We follow the [Linux Foundation Generative AI Policy](https://www.linuxfoundation.org/legal/generative-ai). Contributors must ensure AI tool terms do not conflict with the project's license.

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please read and follow our [Code of Conduct](https://github.com/genealogix/glx/blob/main/CODE_OF_CONDUCT.md).

## Security

To report a vulnerability, see our [Security Policy](https://github.com/genealogix/glx/blob/main/SECURITY.md).

## Building for Release

Releases use [GoReleaser](https://goreleaser.com/install/) (automated in CI on tag push):

```bash
# Test release build locally (requires goreleaser CLI)
make release-snapshot
```

## Questions?

- **Technical questions**: [GitHub Discussions](https://github.com/genealogix/glx/discussions)
- **Private concerns**: Contact maintainers at conduct@genealogix.io

---

Thank you for contributing to GENEALOGIX!
