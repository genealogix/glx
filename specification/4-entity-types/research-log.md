---
title: Research Log Entity
description: Structured records of research process — what was searched, where, with what outcome — supporting the Genealogical Proof Standard's reasonably exhaustive search
layout: doc
---

# Research Log Entity

[← Back to Entity Types](README)

## Overview

A Research Log entity documents the *process* of research against a specific objective: which repositories and collections were searched, what terms were used, and what was or was not found. It is the structured home for **negative evidence** — the documented fact that a person does *not* appear in a particular record set, which is critical to genealogical proof but has no first-class slot in the rest of the spec.

Research logs are distinct from assertions:

- **Assertions** record *conclusions* drawn from evidence ("Jane Webb was born in Florida in 1832").
- **Research logs** record the *process* that produced — or failed to produce — that evidence ("On 2026-03-09 I searched the FamilySearch 1850 census for Jane Webb in Hartford County and found no match").

A single research log is scoped to one **objective** (a specific question being investigated). One session may produce several logs if it pursues several questions.

## File Format

```yaml
# Any .glx file (commonly in research_logs/ directory)
research_logs:
  log-1860-census-search-jane-webb:
    objective: "Locate Jane Webb in the 1860 U.S. Census to confirm birthplace"
    date: "2026-03-09"
    researcher: "Isaac Schepp"
    status: complete
    searches:
      - repository: repo-familysearch
        collection: "United States, Census, 1860"
        search_terms: "Jane Webb, Wisconsin"
        result: found
        citation: citation-1860-census-webb-household
        notes: "Found as Jane, age 28, born Florida, in R Webb household"
      - repository: repo-familysearch
        collection: "United States, Census, 1860"
        search_terms: "Jane Miller, Wisconsin"
        result: not_found
        notes: "No results under maiden name for 1860"
    conclusions: "Jane Webb located in 1860 Hartford County. Born Florida confirmed. No record found under maiden name Miller for this census year."
    related_persons:
      - person-jane-webb
```

**Key Points:**

- Entity ID is the map key (`log-1860-census-search-jane-webb`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `objective` | string | The specific research question this log is investigating |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `date` | string | Date or date range when the log entry was created or last updated (GLX date format) |
| `researcher` | string | Name of the researcher who created or owns this log |
| `status` | string | Current research status (validated against `research_status_types` vocabulary) |
| `searches` | array | Individual searches performed against the objective — see [Search Entries](#search-entries) |
| `conclusions` | string | Conclusions or summary the researcher drew from the searches |
| `related_persons` | string[] | References to Person entities this research is investigating |
| `related_events` | string[] | References to Event entities this research is investigating |
| `related_relationships` | string[] | References to Relationship entities this research is investigating |
| `related_places` | string[] | References to Place entities this research is investigating |
| `notes` | string \| string[] | General notes about the research log entry |

### Search Entries

Each item in `searches` records one search action:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `result` | string | Yes | Outcome — validated against `search_result_types` vocabulary (`found`, `not_found`, `inconclusive`, `not_searched`) |
| `repository` | string | No | Reference to a Repository entity where the search was performed |
| `collection` | string | No | Free-form name of the collection or database (e.g., "United States, Census, 1860") |
| `search_terms` | string | No | Free-form record of the search terms or strategy used |
| `date` | string | No | Date the search was executed |
| `citation` | string | No | Reference to a Citation entity recording the located evidence (typical for `found` results) |
| `media` | string[] | No | References to Media entities — e.g. screenshots of result pages |
| `notes` | string \| string[] | No | Notes about this individual search |

## Status Vocabulary

The `status` field is validated against the `research_status_types` vocabulary. The standard vocabulary provides:

- `complete` — Research objective has been answered or definitively closed
- `in_progress` — Research is actively underway with searches still planned or pending
- `blocked` — Research is paused waiting on access, a record release, or external input

Archives can extend the vocabulary with custom statuses (e.g., `paused`, `archived`).

## Search Result Vocabulary

The `result` field on each search entry is validated against the `search_result_types` vocabulary. The standard vocabulary provides:

- `found` — Record located, target identified
- `not_found` — Collection searched, target not present (this is the key entry for **negative evidence**)
- `inconclusive` — A candidate record was located but identity could not be confirmed
- `not_searched` — Search planned but not yet executed (use to record the research roadmap)

Archives can extend the vocabulary with custom outcomes if their methodology requires finer distinctions.

## Negative Evidence

A `result: not_found` search entry is itself a piece of evidence: "this collection was searched, this person is not in it." This is a first-class outcome under the Genealogical Proof Standard.

A research log can be paired with a regular `Assertion` to record the negative conclusion explicitly:

```yaml
research_logs:
  log-1850-census-search-jane:
    objective: "Confirm Jane Webb's absence from 1850 Hartford County census"
    searches:
      - repository: repo-familysearch
        collection: "United States, Census, 1850"
        search_terms: "Jane Webb, Hartford County, Wisconsin"
        result: not_found
        citation: citation-1850-hartford-co-census-no-jane
    conclusions: "Jane Webb does not appear in the 1850 Hartford County census, consistent with the family's later-recorded migration date."
    related_persons:
      - person-jane-webb

assertions:
  assertion-jane-not-in-1850-hartford:
    subject:
      person: person-jane-webb
    notes: "Jane Webb does not appear in the 1850 Hartford County census; see log-1850-census-search-jane for the searched collections and terms."
    citations:
      - citation-1850-hartford-co-census-no-jane
    confidence: medium
    status: proven
```

The citation referenced by both the search entry and the assertion captures the searched collection and the date of access.

## Usage Patterns

### Tracking Research Roadmaps

`result: not_searched` lets you record planned searches without executing them — the log becomes both an audit trail of completed work *and* a checklist of what remains:

```yaml
research_logs:
  log-find-jane-marriage:
    objective: "Locate Jane Webb's marriage record (estimated 1853–1855)"
    status: in_progress
    searches:
      - repository: repo-familysearch
        collection: "Wisconsin, Marriage Index, 1820–1907"
        result: not_found
        date: "2026-03-09"
      - repository: repo-ancestry
        collection: "U.S. Newspapers, 1850–1860"
        search_terms: "Webb marriage"
        result: not_searched
        notes: "Try after Wisconsin index search is exhausted"
      - repository: repo-wisconsin-historical-society
        collection: "Hartford Co. court records"
        result: not_searched
        notes: "Requires on-site visit"
```

### Linking to Multiple Subjects

A single research session can target many entities — a couple's marriage involves both spouses; a parentage search ranges over both parents and the child:

```yaml
research_logs:
  log-find-parentage-thomas:
    objective: "Identify Thomas Brown's biological parents"
    related_persons:
      - person-thomas-brown
      - person-john-brown
      - person-mary-smith
    related_relationships:
      - rel-thomas-parents
    status: blocked
    conclusions: "Suspect John Brown / Mary Smith based on naming pattern. Awaiting access to St. Cuthbert's parish register 1820–1830."
```

## Validation Rules

- `objective` must be present and non-empty
- `status`, when present, must match a value in the `research_status_types` vocabulary (unknown values are validation errors, mirroring how `assertion.confidence` is checked against `confidence_levels`)
- Each `searches` entry must have a `result` field
- Each `searches[].result` must match a value in the `search_result_types` vocabulary (unknown values are validation errors)
- All entity references (`repository`, `citation`, `media`, `related_*`) must point to existing entities of the corresponding type, otherwise validation fails

## File Organization

Research log files are typically organized under `research_logs/`, one entity per file. As with all GLX entities, file paths are flexible — the parser discovers entities by their YAML top-level keys.

```text
research_logs/
├── log-1860-census-search-jane-webb.glx
├── log-find-jane-marriage.glx
└── log-find-parentage-thomas.glx
```

## Relationship to Other Entities

```text
ResearchLog
    ├── searches[].repository → references Repository (where the search was run)
    ├── searches[].citation   → references Citation (the located evidence)
    ├── searches[].media      → references Media (e.g., screenshot of result page)
    ├── related_persons       → references Person entities
    ├── related_events        → references Event entities
    ├── related_relationships → references Relationship entities
    └── related_places        → references Place entities

Citation
    └── may be linked from a ResearchLog search entry as the located evidence
```

## GEDCOM Mapping

Research logs are GLX-specific — there is no GEDCOM equivalent. GEDCOM 7's `NO` (Negative Assertion) is closer to a negative `Assertion` than to a process log. Research logs are exported only in GLX-native form.

## Schema Reference

See [research-log.schema.json](../schema/v1/research-log.schema.json) for the complete JSON Schema definition.

## See Also

- [Assertion Entity](assertion) — record conclusions drawn from research, including negative conclusions
- [Citation Entity](citation) — capture the specific evidence referenced from a search entry
- [Repository Entity](repository) — represents the institution or platform that holds a searched collection
- [Vocabularies](vocabularies) — reference for `search_result_types` and `research_status_types`
