---
title: "ADR-0004: Archives are Git repositories of plain-text files"
description: Why a GLX archive is a Git repository of YAML files, not a database.
layout: doc
---

# ADR-0004: Archives are Git repositories of plain-text files

## Status

Accepted

## Context

Genealogical research needs four things that conventional databases do not offer out of the box:

- **History.** Every edit should be recoverable — who changed what, when, and why.
- **Collaboration.** Multiple researchers should be able to work in parallel and merge their changes, without emailing files back and forth.
- **Backup.** Losing one copy should not lose the archive; mirroring should be trivial.
- **Longevity.** The archive should remain readable even if the GLX project goes away.

A database-backed design — SQLite file, server-backed Postgres, a cloud service — fails one or more of these. SQLite gives history only via explicit application support. A server-backed store creates vendor dependency. Cloud services create both vendor dependency and a lifespan tied to the vendor's business.

Version control systems already solve all four problems, and Git in particular has been solving them at planetary scale for two decades.

## Decision

A GLX archive is a **Git repository** whose contents are YAML files organised by a specified directory layout. The `glx` CLI reads, writes, and validates those files; Git itself provides history, branching, merging, and backup.

From the [Introduction](/specification/1-introduction):

> GLX is designed to work naturally with Git because genealogical research shares the same needs: History (every edit recorded), Collaboration (multiple researchers merge changes), Backup (archive lives in Git repository mirrored anywhere).

An archive is not *required* to be a Git repository — nothing stops a researcher from storing `.glx` files in a plain directory. But every convention in the format, from 8-character hex IDs to deterministic filenames, assumes Git is the likely host and optimises for it.

## Consequences

**Positive**

- A researcher owns their archive outright. No account is required and no cloud service can take it away. If GLX the project disappeared tomorrow, the `.glx` files in a researcher's folder would still open in any text editor.
- Mirroring is a one-line `git remote add`. Collaboration is `git clone`/`git pull`/`git push`.
- The full audit trail — who changed what and when — is native. Blame, bisect, and diff all work on genealogical data without any GLX-specific tooling.
- Releases and public-sharing workflows line up with existing open-source practice. GitHub-style PRs work unmodified for archive collaboration.

**Negative**

- Git has scale limits. Archives with tens of millions of entities will strain Git's object and index models before they strain the YAML format itself. Solutions (partial clones, sub-archive splits) are workable but not free.
- Contributors must learn at least minimal Git. Tools like GitHub Desktop and the `glx` CLI try to hide the rough edges, but a researcher who edits a file and then wants to "undo" still has to learn at least `git restore`.
- There is no server-side transactional semantics — two researchers can genuinely disagree about a fact, and the archive is fine with that. It is the researchers, not the database, who have to reconcile via review.
- Windows line-ending handling and case-insensitive filesystems create occasional friction. The repository pins line endings (`.gitattributes`) and the serializer lowercases filenames to mitigate.
