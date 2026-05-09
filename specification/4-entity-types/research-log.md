---
title: ResearchLog Entity
description: Documents searches performed during a research investigation, including negative results, to support a "reasonably exhaustive search"
layout: doc
---

# ResearchLog Entity

[← Back to Entity Types](README)

## Overview

A ResearchLog entity records what a researcher has searched for, where, when, and what was (or was not) found. It treats **negative evidence** — "the source was searched and the target was not present" — as a first-class outcome, supporting the [Genealogical Proof Standard](https://bcgcertification.org/ethics-standards/) requirement for a "reasonably exhaustive search."

Without a ResearchLog, this information lives in freeform notes or external spreadsheets and is not queryable. Researchers end up duplicating searches across sessions because there is no structured record that a given collection has already been checked.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in research_logs/ directory)
research_logs:
  research-log-1860-census-jane-webb:
    title: "1860 census — locate Jane Webb"
    subject:
      person: person-jane-webb
    objective: "Confirm Jane Webb's birthplace via 1860 federal census"
    status: complete
    searches:
      - repository: repository-familysearch
        collection: "United States, Census, 1860"
        query: "Jane Webb, Hartford County, Wisconsin"
        result: found
        citation: citation-1860-census-webb-household
      - repository: repository-familysearch
        collection: "United States, Census, 1860"
        query: "Jane Miller, Hartford County, Wisconsin"
        result: not_found
        notes: "No matches under maiden name"
    conclusions: "Jane Webb located in 1860 Hartford County household; born Florida."
```

**Key Points:**

- Entity ID is the map key (`research-log-1860-census-jane-webb`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens
- Searches are embedded inside the log; they are not standalone entities

## Core Concepts

### Why a structured log

A ResearchLog answers four questions that a `notes` field cannot:

1. **What did we already search?** Avoid repeating a fruitless query in the next session.
2. **Where are the gaps?** A list of `not_searched` plans surfaces the next concrete steps.
3. **What is the negative evidence?** "Source X searched, target absent" is a genealogical conclusion, not a non-result.
4. **How exhaustive was the search?** A reviewer can audit how many distinct sources were checked.

### Search results

Each `Search` records one query and its outcome. The standard outcomes are:

- `found` — the target was located in the source
- `not_found` — the source was searched and the target was not present (negative evidence)
- `inconclusive` — a candidate record was located but cannot be confirmed
- `partial` — some relevant information was located, but the objective was not fully met

**See [Vocabularies - Search Result Types](vocabularies#search-result-types-vocabulary)** for the full vocabulary.

### Status lifecycle

`status` tracks where the investigation stands:

- `open` — defined but not started
- `in_progress` — actively being worked
- `complete` — objective met, conclusions documented
- `blocked` — cannot proceed (waiting on access, missing records, etc.)

**See [Vocabularies - Research Log Status Types](vocabularies#research-log-status-types-vocabulary).**

## Fields

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `title` | string | Optional human-readable title for the log |
| `subject` | object | Typed reference (`person` / `event` / `relationship` / `place`) to what is being investigated |
| `date` | date | Date the research was performed (or session start) |
| `researcher` | string | Name or identifier of the researcher |
| `objective` | string | Research question or goal |
| `status` | string | Lifecycle status (validated against `research_log_status_types` vocabulary) |
| `searches` | array | List of `Search` entries — one per search attempt |
| `citations` | array | Citations produced by this log (denormalized from per-search refs) |
| `conclusions` | string | Summary of findings |
| `properties` | object | Vocabulary-defined properties |
| `notes` | string \| string[] | Free-form notes about the log |

No fields are required. A log with only an `objective` and an empty `searches` list represents a planned investigation.

### Search entry fields

Each entry in `searches` is a structured record of one search attempt:

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | Reference to the Repository searched |
| `source` | string | Reference to the Source searched (when known to the archive) |
| `collection` | string | Free-form collection name (used when no Source entity exists yet, e.g., "United States, Census, 1860") |
| `query` | string | Search terms used |
| `date` | date | Date this specific search was performed |
| `result` | string | Outcome (validated against `search_result_types` vocabulary) |
| `citation` | string | Reference to the Citation produced when result is `found` |
| `notes` | string \| string[] | Free-form notes about this attempt |

## Usage Patterns

### Tracking negative evidence

```yaml
research_logs:
  research-log-1850-census-john-doe:
    objective: "Establish whether John Doe was in Hartford County in 1850"
    status: complete
    searches:
      - repository: repository-familysearch
        collection: "United States, Census, 1850"
        query: "John Doe, Hartford County, Wisconsin"
        result: not_found
      - repository: repository-ancestry
        collection: "1850 U.S. Federal Census"
        query: "Doe, Hartford County"
        result: not_found
    conclusions: "John Doe is absent from the 1850 Hartford County census in two independent indexes; arrival post-1850 is consistent with the 1854 marriage record."
```

This documents a negative finding that is part of the evidence chain.

### Brick-wall investigation in progress

```yaml
research_logs:
  research-log-lucinda-norton-parents:
    title: "Lucinda Norton — parentage"
    subject:
      person: person-lucinda-norton
    objective: "Identify Lucinda Norton's parents (born ~1808 Connecticut)"
    status: in_progress
    researcher: "Isaac Schepp"
    searches:
      - repository: repository-familysearch
        collection: "Connecticut, Hartford, Vital Records"
        query: "Norton, Hartford County 1805-1815"
        result: inconclusive
        notes: "Multiple Norton families; cannot match to Lucinda without parent name"
    conclusions: "Strongest hypothesis: née Munson. Online sources exhausted; need on-site Litchfield County land records."
    notes:
      - "Continue with Litchfield County land records 1785-1810 next session"
```

### Linking to citations

When a search produces a citable record, link the citation both inline (on the search entry) and at the log level for easy roll-up:

```yaml
research_logs:
  research-log-1860-census-jane-webb:
    objective: "Locate Jane Webb in 1860 census"
    status: complete
    searches:
      - repository: repository-familysearch
        source: source-1860-census
        result: found
        citation: citation-1860-census-webb-household
    citations:
      - citation-1860-census-webb-household
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. Per-entity files are recommended for collaborative projects.

```text
research_logs/
├── research-log-1860-census-jane-webb.glx
├── research-log-lucinda-norton-parents.glx
└── research-log-pohl-gons-parish-1610-1620.glx
```

## Validation Rules

- `subject` if present must reference an existing entity of the type indicated (Person, Event, Relationship, or Place)
- `status` if present must be from the [research log status types vocabulary](vocabularies#research-log-status-types-vocabulary)
- Each search's `repository`, `source`, and `citation` if present must reference existing entities
- Each search's `result` if present must be from the [search result types vocabulary](vocabularies#search-result-types-vocabulary)
- `citations` entries must reference existing Citation entities

## Related Issues

ResearchLog tracks individual searches and their outcomes. It is intentionally distinct from related concepts being designed in parallel:

- **Research investigation** ([#660](https://github.com/genealogix/glx/issues/660)): a higher-level workflow tracker for a research question — leads, hypotheses, next steps. ResearchLog records *what was searched*; Research records *what we are trying to figure out*.
- **Study/Project** ([#226](https://github.com/genealogix/glx/issues/226)): defines the scope of a research project (e.g., a One Place Study). A Study contains many ResearchLogs.

CLI commands for adding and querying logs (`glx log add`, `glx log list`, `glx log report`) are out of scope for the entity-spec PR and tracked separately.

## GEDCOM Mapping

ResearchLog has no direct GEDCOM equivalent. GEDCOM treats research process as freeform NOTE content; GLX promotes it to a structured, queryable entity.

## Schema Reference

See [research-log.schema.json](../schema/v1/research-log.schema.json) for the complete JSON Schema definition.

## See Also

- [Citation Entity](citation) — the per-record evidence produced by a successful search
- [Repository Entity](repository) — the institution searched
- [Source Entity](source) — the bibliographic resource searched
