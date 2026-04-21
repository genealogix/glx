---
title: "ADR-0002: Separate Repository, Source, Citation, and Assertion entities"
description: Why GLX splits evidence into four entity types that form a chain from archive to conclusion.
layout: doc
---

# ADR-0002: Separate Repository, Source, Citation, and Assertion entities

## Status

Accepted

## Context

Traditional genealogical formats — most notably GEDCOM — conflate evidence and conclusion. A `BIRT` record simply carries a date, a place, and maybe a free-text source note. There is no structural distinction between "the register says this" and "I concluded this". That makes it hard to:

- represent multiple sources supporting the same fact,
- represent conflicting sources supporting different facts,
- attach confidence levels to conclusions,
- audit where any given fact came from.

Professional genealogy practice has long solved this problem on paper. The [Genealogical Proof Standard](https://www.bcgcertification.org/resources/standard.html) and Elizabeth Shown Mills' *Evidence Explained* both centre on a **source-to-citation-to-conclusion chain** — what the archive holds, exactly what part of it you looked at, and what you inferred from it. GLX aims to encode that chain as first-class data rather than prose.

## Decision

Model evidence as a four-entity chain:

1. **Repository** — an institution, archive, or publisher that holds sources (e.g., "Yorkshire Archives").
2. **Source** — a specific item within a repository (e.g., "Leeds Parish Register, 1850–1855").
3. **Citation** — a reference to a particular part of a source (e.g., "Leeds Parish Register, 1850–1855, entry #42, Baptism of John Smith, 15 March 1850").
4. **Assertion** — a source-backed conclusion about a fact, with a confidence level, referencing one or more citations.

From the [Assertion Entity specification](/specification/4-entity-types/assertion):

> An Assertion entity represents a source-backed conclusion about a specific genealogical fact. Assertions form the core of the GENEALOGIX evidence model, separating *what sources say* (citations) from *what we conclude* (assertions).

## Consequences

**Positive**

- Multiple citations can support a single assertion, and multiple assertions can disagree about the same fact. The data model makes both expressible without hacks.
- Confidence can be recorded on each assertion rather than buried in prose, so downstream tools (proof-argument generators, visualisations) have something structured to work with.
- The chain aligns with established genealogical standards, so outputs can be formatted as proper citations without re-reasoning about what each field meant.
- A full audit trail from archive box to published family tree is native to the format.

**Negative**

- More entity types than a flatter format. A minimal archive with one person and one birth event still benefits from a Source + Citation + Assertion trio if the birth date is to be properly sourced — which means more files and more YAML to write.
- Tools and UIs must handle the indirection. A genealogist looking at a fact in a viewer expects to see the underlying citation in one or two clicks; the format encourages tools to invest in that UX.
- Researchers coming from GEDCOM or proprietary trees often need to be taught the distinction between a citation and an assertion. The `docs/guides/best-practices.md` and `docs/quickstart.md` pages exist in part to ease that on-ramp.
