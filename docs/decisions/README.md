---
title: Architecture Decision Records
description: Records of the significant architectural and design decisions behind GENEALOGIX.
layout: doc
---

# Architecture Decision Records

This directory contains **Architecture Decision Records (ADRs)** — short documents that capture significant architectural decisions, the context in which they were made, and the consequences that followed.

## Why ADRs?

For a specification project, architectural decisions compound and are expensive to reverse. New contributors need to understand not just *what* GLX does, but *why* it does it that way. Before ADRs, that reasoning lived in scattered GitHub issues and project-internal notes. ADRs move it into the repository, where it is discoverable and versioned alongside the code.

ADRs are **immutable once Accepted**. If a decision changes, we write a new ADR that supersedes the old one, and update the old one's Status to `Superseded by ADR-XXXX`. This preserves the history of reasoning rather than overwriting it.

## Status lifecycle

- **Proposed** — under discussion in an open PR.
- **Accepted** — merged; reflects current practice.
- **Deprecated** — no longer followed, but not replaced.
- **Superseded by ADR-XXXX** — replaced by a later decision.

## How to propose a new ADR

1. Copy [`0000-adr-template`](0000-adr-template) to `docs/decisions/NNNN-kebab-case-title.md`, where `NNNN` is the next unused 4-digit number.
2. Fill in Context, Decision, and Consequences. Keep each section short — one or two paragraphs is usually enough.
3. Open a PR. Initial Status is `Proposed`.
4. On acceptance, flip Status to `Accepted` and add a row to the index below.

See the [Contributing Guide](/development/contributing#architecture-decision-records-adrs) for when an ADR is required versus when a regular issue or PR is sufficient.

## Index

| ADR | Title | Status |
| --- | --- | --- |
| [0001](0001-yaml-file-format) | Use YAML as the archive file format | Accepted |
| [0002](0002-evidence-first-data-model) | Separate Repository, Source, Citation, and Assertion entities | Accepted |
| [0003](0003-archive-owned-vocabularies) | Each archive owns its controlled vocabularies | Accepted |
| [0004](0004-git-native-archives) | Archives are Git repositories of plain-text files | Accepted |
| [0005](0005-flexible-entity-ids) | Flexible entity IDs, with 8-character hex as the recommended default | Accepted |
| [0006](0006-go-glx-library-pure) | The go-glx library never performs filesystem I/O | Accepted |

Numbers are assigned in the order ADRs are proposed, not by topic — this prevents merge conflicts when two ADRs are proposed at the same time.
