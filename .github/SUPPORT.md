# Getting Help with GLX

Thanks for using GENEALOGIX (GLX). The GitHub issue tracker is reserved for verified bugs and accepted feature work. For everything else, use one of the channels below — it keeps the tracker focused and gets you a faster answer.

## Ask a question

- **[GitHub Discussions](https://github.com/genealogix/glx/discussions)** — the primary place for usage questions, design ideas, and troubleshooting. Search first — your question may already be answered.
- **[Discord](https://genealogix.io/discord)** — real-time chat with maintainers and other users.
- **[Mailing list](https://groups.google.com/g/genealogix)** — lower-traffic, email-based discussion.

GLX is maintained by a small volunteer team. We answer Discussions on a best-effort basis as bandwidth allows — well-formed, reproducible reports get answered fastest.

## Before you open an issue

Most questions are already answered in the docs:

- [README](/README.md) — overview, installation, quick start
- [Specification](/specification) — format reference
- [Contributing Guide](/CONTRIBUTING.md) — development setup, testing, submission process
- [Changelog](/CHANGELOG.md) — recent changes; confirm the bug isn't already fixed on `main`

GLX welcomes AI-assisted work but holds humans accountable, bans autonomous/agent-filed issues and PRs, and caps open PRs at 3 — see [AI-Generated Contributions](/CONTRIBUTING.md#ai-generated-contributions) before filing.

## Report a bug

Use the [bug report template](https://github.com/genealogix/glx/issues/new?template=bug_report.yml). Include:

- GLX version (`glx --version`) and Go version (`go version`)
- Operating system
- A minimal reproduction: commands, input file snippet, expected vs. actual output

Reports without a reproduction won't be triaged until one is provided.

## Request a feature

Use the [feature request template](https://github.com/genealogix/glx/issues/new?template=feature_request.yml) to suggest a brand-new capability.

To suggest an improvement to an existing capability, use the [enhancement template](https://github.com/genealogix/glx/issues/new?template=enhancement.yml) instead.

Changes to the core data model, entity types, validation rules, or file format require a written proposal and community discussion period — see [Proposing Major Changes](/CONTRIBUTING.md#proposing-major-changes) in the Contributing Guide.

## Report a security vulnerability

**Do not** open a public issue. Follow the [Security Policy](/SECURITY.md) — reports go through [GitHub Security Advisories](https://github.com/genealogix/glx/security/advisories/new).

## Code of Conduct and private concerns

For Code of Conduct violations or other private concerns, email <conduct@genealogix.io>. See the [Code of Conduct](/CODE_OF_CONDUCT.md) for the full policy.
