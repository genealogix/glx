---
title: ResearchLog Entity
description: Documents a research investigation (the searches performed, including negative results, and the competing hypotheses under evaluation) to support a "reasonably exhaustive search"
layout: doc
---

# ResearchLog Entity

[← Back to Entity Types](README.md)

## Overview

A ResearchLog entity records what a researcher has searched for, where, when, and what was (or was not) found, together with the **leads** (competing hypotheses) the research is trying to settle. It treats **negative evidence** — "the source was searched and the target was not present" — as a first-class outcome, supporting the [Genealogical Proof Standard](https://bcgcertification.org/ethics-standards/) requirement for a "reasonably exhaustive search."

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
- A log embeds two component types, each specified in its own section below: [Search](#search), one search attempt, and [ResearchLead](#researchlead), one competing hypothesis. Neither is a standalone entity: they have no IDs and cannot be referenced from elsewhere in the archive

## Core Concepts

### Why a structured log

A ResearchLog answers four questions that a `notes` field cannot:

1. **What did we already search?** Avoid repeating a fruitless query in the next session.
2. **Where are the gaps?** A list of `not_searched` and `requires_visit` plans surfaces the next concrete steps, while `unavailable` records the gaps that cannot be closed.
3. **What is the negative evidence?** "Source X searched, target absent" is a genealogical conclusion, not a non-result.
4. **How exhaustive was the search?** A reviewer can audit how many distinct sources were checked.

### Status lifecycle

`status` tracks where the investigation stands:

- `open` — defined but not started
- `in_progress` — actively being worked
- `complete` — objective met, conclusions documented
- `blocked` — cannot proceed (waiting on access, missing records, etc.)

**See [Vocabularies - Research Log Status Types](vocabularies.md#research-log-status-types-vocabulary).**

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |

A log with only an entity ID is a valid, minimal placeholder; `objective` and a populated `searches` list typically follow.

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Optional human-readable title for the log |
| `subject` | object | Typed reference (`person` / `event` / `relationship` / `place`) to what is being investigated |
| `date` | date | Date the research was performed (or session start) |
| `researcher` | string | Name or identifier of the researcher |
| `objective` | string | Research question or goal |
| `status` | string | Lifecycle status (validated against `research_log_status_types` vocabulary) |
| `searches` | array | List of [Search](#search) entries, one per search attempt |
| `leads` | array | List of [ResearchLead](#researchlead) entries, the research alternatives under investigation |
| `citations` | array | Citations produced by this log (denormalized from per-search refs) |
| `conclusions` | string | Summary of findings |
| `properties` | object | Vocabulary-extensible metadata. There is no research-log property vocabulary in this revision, so keys are not validated and unknown keys do not produce warnings |
| `notes` | string \| string[] | Free-form notes about the log |

## Search

A **Search** is one entry in a log's `searches` list. It records *what was searched*: one query against one repository, source, or collection, when it was run, and what came of it, including nothing. A search that found nothing is negative evidence, and recording it is what lets the log show a reasonably exhaustive search.

### Search results

A search's `result` records its outcome. The standard outcomes are:

- `found` — the target was located in the source
- `not_found` — the source was searched and the target was not present (negative evidence)
- `inconclusive` — a candidate record was located but cannot be confirmed
- `partial` — some relevant information was located, but the objective was not fully met
- `not_searched` — search is planned but has not yet been performed (use to record outstanding work alongside completed searches)
- `requires_visit` — the source exists but can only be searched on site or by request; outstanding work, but not something that can be done from a desk
- `unavailable` — the source was sought but cannot be searched because it does not survive, is restricted, or is not accessible; say why in `notes`

`unavailable` is the opposite of `not_searched`: the search was attempted and the gap is permanent (a burned courthouse, a lost volume, a closed record series). Tools that build a to-do list from a log treat `not_searched` and `requires_visit` as outstanding work and `unavailable` as a documented gap; `glx proof` labels each accordingly in its reasonably-exhaustive-search section.

**See [Vocabularies - Search Result Types](vocabularies.md#search-result-types-vocabulary)** for the full vocabulary.

### Search fields

All fields are optional.

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

### Search validation

- `repository`, `source`, and `citation` if present must reference existing Repository, Source, and Citation entities
- `result` if present must be from the [search result types vocabulary](vocabularies.md#search-result-types-vocabulary)

## ResearchLead

A **ResearchLead** is one entry in a log's `leads` list. Where searches record *what was searched*, leads record *what we are trying to figure out*. A brick-wall question such as "who were Mary Green's parents?" usually has several candidate answers at once, and each candidate is a lead: a description, optional references to the candidate persons, the evidence for and against it, and the next steps for pursuing it. Researchers eliminate leads one by one until one is confirmed.

Leads track **research alternatives**. They are workflow state that says which possibilities are still open, not conclusions about the people in the archive.

### Lead statuses

A lead's `status` is one of:

- `active`: still being pursued; the evidence neither rules it in nor out yet
- `eliminated`: ruled out by contradicting evidence (keep it, with `evidence_against`, so the elimination is documented and nobody re-investigates it)
- `confirmed`: the research has settled on this alternative

**See [Vocabularies - Lead Statuses](vocabularies.md#lead-statuses-vocabulary).** Archives can add their own values, such as `parked`.

### Leads and assertions

Conclusions live in [Assertions](assertion.md), not in leads. The two are kept deliberately separate:

- **Resolving a lead does not change any assertion.** Setting a lead's `status` to `confirmed`, `eliminated`, or any other value leaves every assertion in its `assertions` list exactly as it was, and editing an assertion leaves the lead unchanged. Tools do not update one from the other.
- **The `assertions` list is a cross-reference, not a binding.** It records which assertions model the hypothesis, so a reader can move between the research question and the claims it bears on.
- **Recording the outcome is a separate step.** When a lead is confirmed, write the conclusion where it belongs in the archive: a [Relationship](relationship.md) or property value backed by an Assertion with citations. When a lead is eliminated, a speculative assertion that modelled it (for example a low-confidence `parent_child` claim for a candidate father) is updated on its own, usually to `status: disproven`.
- **Leads carry no confidence of their own.** How strongly the evidence supports a candidate is the `confidence` of the assertion that models it; where the research on that candidate stands is the lead's `status`. A second confidence on the lead would be a second answer to the assertion's question, free to drift from it.

### Evidence for and against

`evidence_for` and `evidence_against` are free-text summaries, since much brick-wall reasoning ("daughter of the right age in the 1840 census", "father born in England") is an inference over several records rather than a single citation. Use the lead's `citations` and `assertions` lists to point at the structured evidence those summaries rest on.

### Lead fields

All fields are optional, though `description` is strongly recommended.

| Field | Type | Description |
|-------|------|-------------|
| `description` | string | Brief description of the hypothesis (e.g., "John H. Green of Wheeling, VA") |
| `persons` | string[] | References to candidate Person entities the lead is about |
| `status` | string | Where the research on this alternative stands: `active`, `eliminated`, or `confirmed` (validated against `lead_statuses` vocabulary) |
| `evidence_for` | string[] | Free-text summaries of evidence supporting the hypothesis |
| `evidence_against` | string[] | Free-text summaries of evidence contradicting the hypothesis |
| `citations` | string[] | References to Citations that bear on the lead |
| `assertions` | string[] | Cross-references to Assertions that model the hypothesis (e.g., a speculative `parent_child` claim). Informational only; see [Leads and assertions](#leads-and-assertions) |
| `next_steps` | string[] | Actionable next steps for pursuing the lead |
| `notes` | string \| string[] | Free-form notes about the lead |

### Lead validation

- `persons`, `citations`, and `assertions` entries must reference existing Person, Citation, and Assertion entities
- `status` if present must be from the [lead statuses vocabulary](vocabularies.md#lead-statuses-vocabulary)

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

### Tracking competing candidates

When several candidate answers are in play, record each as a lead and eliminate them as evidence comes in:

```yaml
research_logs:
  research-log-mary-green-parents:
    subject:
      person: person-mary-green
    objective: "Identify Mary Green's parents"
    status: in_progress
    searches:
      - collection: "United States, Census, 1850"
        query: "Green households, Ohio County, Virginia"
        result: not_searched
    leads:
      - description: "John H. Green of Wheeling, VA"
        persons: [person-john-h-green]
        status: active
        evidence_for:
          - "Daughter aged 5-10 in the 1840 census"
          - "Wheeling is Daniel Lane's birthplace"
        evidence_against:
          - "Father born in England, not connected to the NY Greens"
        next_steps:
          - "Search 1850 census, Ohio County, VA"
      - description: "Luther Green of Springwater, NY"
        persons: [person-luther-green]
        status: eliminated
        evidence_against:
          - "No daughter of the right age in the 1840 or 1850 census"
```

`glx query research_logs --subject person-mary-green` lists the logs about Mary (as subject) and, because leads name candidate persons, `--subject person-john-h-green` finds this log too. `glx summary` shows a Research section for a person with each log's objective, status, outstanding `not_searched` searches, and open leads. Lead counts retain each recorded status, including archive-defined values; a lead without a status is counted as `unspecified`.

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
- `status` if present must be from the [research log status types vocabulary](vocabularies.md#research-log-status-types-vocabulary)
- Each entry in `searches` must satisfy [Search validation](#search-validation)
- Each entry in `leads` must satisfy [Lead validation](#lead-validation)
- `citations` entries must reference existing Citation entities

## Related Issues

ResearchLog records both *what was searched* (`searches`) and *what we are trying to figure out* (`leads`):

- **Research investigation** ([#660](https://github.com/genealogix/glx/issues/660)) originally proposed a separate Research entity for leads, hypotheses, and next steps. Because ResearchLog already carried the subject, objective, status, conclusions, and searches that entity needed, the proposal was folded into ResearchLog as `leads` instead of adding a new entity type. Candidate matching for unknown parentage ([#183](https://github.com/genealogix/glx/issues/183)) is modelled the same way: one lead per candidate, with evidence for and against.
- **[Study](study.md)** (issue #226, shipped in beta.11): defines the scope of a research project (e.g., a One Place Study). Logs performed within a study are associated by convention (shared `subject`, places, or sources); there is no linking field between the two entities in this revision.

The `glx` CLI creates logs with [`glx add research-log`](../../docs/cli/glx_add_research-log.md) and appends one search at a time with [`glx add search`](../../docs/cli/glx_add_search.md); studies are created with [`glx add study`](../../docs/cli/glx_add_study.md), and both are listed with [`glx query research_logs`](../../docs/cli/glx_query.md) and `glx query studies`. These commands are tooling, not part of this specification.

`glx query research_logs --subject` finds a person's logs as subject or as a lead candidate, and [`glx summary`](../../docs/cli/glx_summary.md) displays their Research section.

## GEDCOM Mapping

ResearchLog has no direct GEDCOM equivalent. GEDCOM treats research process as freeform NOTE content; GLX promotes it to a structured, queryable entity.

## Schema Reference

See [research-log.schema.json](../schema/v1/research-log.schema.json) for the complete JSON Schema definition.

## See Also

- [Citation Entity](citation.md) — the per-record evidence produced by a successful search
- [Repository Entity](repository.md) — the institution searched
- [Source Entity](source.md) — the bibliographic resource searched
