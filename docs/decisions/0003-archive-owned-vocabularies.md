---
title: "ADR-0003: Each archive owns its controlled vocabularies"
description: Why vocabularies (event types, relationship types, etc.) live inside each archive rather than in a central registry.
layout: doc
---

# ADR-0003: Each archive owns its controlled vocabularies

## Status

Accepted

## Context

Genealogical data needs controlled vocabularies for things like event types (`birth`, `marriage`, `arrival`), relationship types (`parent`, `godparent`, `employer`), place types, source types, and so on. A free-text field is too loose (spelling drift, untranslatable terms); a hard-coded enum is too rigid (different research domains need different types — colonial history, maritime research, and religious studies each have domain-specific event types).

Two structural choices were on the table:

- **Central registry** — a project-maintained list of valid types. Archives reference them. New types require a PR to the central list.
- **Archive-owned vocabularies** — each archive ships its own vocabulary files. A standard starter set is embedded in the tooling, and archives extend it as needed.

A central registry gives better cross-archive comparability but creates a governance bottleneck: every new use case has to negotiate with a committee. An archive-owned model favors autonomy at the cost of some interoperability for extension types.

## Decision

Vocabularies live inside each archive as YAML files (typically in a `vocabularies/` directory). The project ships a **standard starter set** — embedded in the `go-glx` library via `go:embed` — that archives can use directly or extend.

From [Core Concepts](/specification/2-core-concepts):

> Unlike traditional genealogy formats with fixed type systems, GENEALOGIX uses *archive-owned controlled vocabularies*. Each archive defines its own valid types in vocabulary files, combining standardization with flexibility.

The `glx validate` command ensures every type used in an archive is defined in that archive's vocabularies — so the archive is internally consistent even if it uses terms no other archive has heard of.

## Consequences

**Positive**

- Domain specialization (colonial-history-specific event types, religious-studies-specific relationship types) does not require project approval. Researchers ship types they need.
- Vocabulary changes are versioned alongside the data they describe. If you add a new event type in April 2026, that change is in the same commit history as the events that used it.
- The standard starter set gives new archives a sensible default so they do not have to define vocabularies from scratch.
- No central governance body has to triage vocabulary additions, which scales.

**Negative**

- Cross-archive interoperability for extension types is weaker. Two archives that independently coin a `maritime-landing` event type may use incompatible semantics.
- Tools that consume multiple archives must either ignore unknown types or provide a mapping layer. This is by design — GLX is not an exchange format first, it is an archive format first.
- The cost of interoperability falls on downstream tools rather than on vocabulary authors. That is an explicit trade-off.
