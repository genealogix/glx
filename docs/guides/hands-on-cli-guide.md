---
title: Hands-On CLI Guide
description: A practical walkthrough of every glx command using the Westeros demo archive
layout: doc
---

# Hands-On CLI Guide

This guide walks through every `glx` command using the [Westeros demo archive](/examples/westeros/) — a large-scale genealogy of 790+ characters from *A Song of Ice and Fire*. By the end, you'll know how to validate, query, explore, and export any GLX archive.

## Setup

Clone the demo archive and install the CLI:

```bash
git clone https://github.com/genealogix/glx-archive-westeros.git
cd glx-archive-westeros
```

If you haven't installed the CLI yet, see the [installation instructions](/cli#installation).

## Archive Health

### `glx validate` — Check data integrity

Start by validating the archive. This checks YAML syntax, required fields, cross-references between entities, and vocabulary types.

```bash
glx validate
```

The validator reports errors (hard failures) and warnings (soft failures). For a large archive, you'll typically see a few warnings — these flag potential issues without blocking:

```
Validated 4,841 files.
Found 4 errors:
- ❌ duplicate assertions ID: assertion-lysa-death-littlefinger
- ❌ events[event-duel-bran-assassin].Role references non-existent
      participant_roles: target
...
validation failed
```

::: tip
Use `glx analyze` to check for temporal consistency issues (death before birth, parent younger than child) and other research gaps. Validation focuses on structural correctness.
:::

### `glx stats` — Dashboard overview

Get a quick overview of what's in the archive:

```bash
glx stats
```

```
Entity counts:
  Persons:       791
  Events:        961
  Relationships: 822
  Places:        108
  Sources:       9
  Citations:     247
  Repositories:  4
  Media:         0
  Assertions:    1808

Assertion confidence:
  high         1406  ( 77.8%)
  medium        365  ( 20.2%)
  low            37  (  2.0%)

Entity coverage (referenced by assertions):
  Persons         623/791  (78.8%)
  Events          55/961  (5.7%)
  Relationships   6/822  (0.7%)
  Places          0/108  (0.0%)
```

This tells you the archive has nearly 800 persons with 1,800+ assertions, and that 78% of persons are backed by at least one assertion. The coverage gaps in events and places suggest areas for future research.

### `glx places` — Analyze place data quality

Check places for data quality issues — duplicates, missing coordinates, hierarchy gaps, and unreferenced places:

```bash
glx places
```

```
Place analysis: 108 places

Duplicate names (ambiguous):
  "The Red Keep" appears 2 times:
    place-red-keep     The Red Keep, King's Landing, The Crownlands, Westeros
    place-the-red-keep The Red Keep, King's Landing, The Crownlands, Westeros

Missing coordinates (108 of 108):
  place-acorn-hall  Acorn Hall, The Riverlands, Westeros
  place-astapor     Astapor, Essos
  ...
```

Each place is shown with its canonical hierarchy path (e.g., "Winterfell, The North, Westeros"). The duplicate "Red Keep" flags an ID inconsistency to clean up.

### `glx analyze` — Research gap analysis

Run automated analysis to surface evidence gaps, quality issues, chronological inconsistencies, and research suggestions:

```bash
glx analyze
```

```
=== Research Gap Analysis: 847 issues found ===

EVIDENCE GAPS (312)
  HIGH person-aegon-i-targaryen       No birth date
  HIGH person-aegon-ii-targaryen      No birth date
  HIGH person-aegon-iii-targaryen     No birth date
  ...

EVIDENCE QUALITY (198)
  HIGH person-aegon-i-targaryen       No assertions backed by citations
  HIGH person-aegon-ii-targaryen      No assertions backed by citations
  ...

CONSISTENCY (5)
  HIGH person-aegon-v-targaryen       Death year (259) before birth year (c. 200)
  ...

SUGGESTIONS (332)
  →   person-arya-stark               Search for vital records
  →   person-benjen-stark             Search for vital records
  ...
```

Focus on a single person or category:

```bash
# Analyze one person
glx analyze "Eddard Stark"

# Run only consistency checks
glx analyze --check consistency

# JSON output for tooling
glx analyze --format json
```

## Querying Entities

### `glx query persons` — Find people

List all persons, or filter by name and birth year:

```bash
# Find all Starks
glx query persons --name "Stark"
```

```
  person-arya-stark       Arya Stark  (b. 289)
  person-benjen-stark     Benjen Stark
  person-brandon-stark-bran  Brandon Stark  (b. 290)
  person-eddard-stark     Eddard Stark  (263 – 299)
  person-lyanna-stark     Lyanna Stark  (266 – 283)
  person-robb-stark       Robb Stark  (283 – 299)
  person-sansa-stark      Sansa Stark  (b. 286)
  ...

17 person(s) found
```

The `--name` flag searches all name variants — birth names, married names, aliases, and as-recorded forms. You can also filter by birth year:

```bash
# Find persons born in the 280s
glx query persons --born-after 280 --born-before 290
```

### `glx query events` — Find events by type

```bash
# List all battles
glx query events --type battle
```

```
  event-battle-blackwater          battle  299
  event-battle-green-fork          battle  298
  event-battle-trident             battle  283
  event-battle-whispering-wood     battle  298
  event-tower-of-joy               battle  283
  ...

25 event(s) found
```

```bash
# All executions
glx query events --type execution
```

```
  event-execution-brandon-rickard-stark  execution  282
  event-execution-ned-stark              execution  299
  event-execution-janos-slynt            execution  300
  ...

7 event(s) found
```

### `glx query relationships` — Find connections

```bash
# Find ward relationships
glx query relationships --type ward
```

```
  rel-ward-theon-greyjoy-ned-stark  ward
    [person-theon-greyjoy, person-eddard-stark]
  rel-ward-littlefinger-tully  ward
    [person-petyr-baelish, person-hoster-tully]
  rel-ward-frey-grandsons-winterfell  ward
    [person-walder-frey-big-walder, person-walder-frey-little-walder, person-robb-stark]

3 relationship(s) found
```

### `glx query assertions` — Trace evidence

Find assertions by confidence level, source, or citation:

```bash
# Low-confidence assertions (legendary figures, disputed identities)
glx query assertions --confidence low
```

```
  assertion-aegon-vi-blackfyre-theory  persons:person-young-griff
    house=Blackfyre (theoretical)  [low]
  assertion-alleras-sphinx-identity  persons:person-alleras-the-sphinx
    alias=Possibly Sarella Sand  [low]
  assertion-bran-built-wall  persons:person-bran-the-builder
    role=Builder of the Wall  [low]
  ...

37 assertion(s) found
```

```bash
# All assertions from A Game of Thrones
glx query assertions --source source-agot
```

This returns every fact that traces back to the first novel — useful for auditing which sources support which claims.

## Person Exploration

### `glx vitals` — Quick vital records

Look up a person's key life data:

```bash
glx vitals "Eddard Stark"
```

```
Vitals for person-eddard-stark:

  Name         Eddard Stark
  Sex          male
  Birth        263, Winterfell
  Christening  —
  Death        299, Great Sept of Baelor
  Burial       —
  Battle       283, Stoney Sept
  Battle       283, The Trident
  Coronation   283, King's Landing
  Execution    299, Great Sept of Baelor
  Rebellion    289, Pyke
  Tournament   298, King's Landing
  Trial        298, The Red Keep
  Rebellion    BET 282 AND 283, Westeros
  Tournament   281, Harrenhal
  Battle       283, Tower of Joy
```

You can use a person ID or a name substring — if multiple persons match, all matches are listed for disambiguation.

### `glx timeline` — Chronological events

See every event in a person's life in chronological order, including family events discovered through relationships:

```bash
glx timeline "Eddard Stark"
```

The timeline includes direct events (battles, coronations, trials) and family events like children's births and deaths, discovered by traversing relationships. Use `--no-family` to show only direct events:

```bash
glx timeline "Eddard Stark" --no-family
```

### `glx summary` — Full person profile

Get a comprehensive profile with identity, vital events, life events, family, relationships, and an auto-generated life history:

```bash
glx summary "Eddard Stark"
```

```
=== person-eddard-stark ===

  Name:             Eddard Stark
  Sex:              male

── Vital Events ──────────────────────────────────
  Birth:            263, Winterfell
  Death:            299, Great Sept of Baelor

── Life Events ───────────────────────────────────
  Battle:           283, Stoney Sept
  Battle:           283, The Trident
  Coronation:       283, King's Landing
  ...
  Alias:            Ned
  Alias:            The Quiet Wolf
  House:            Stark
  Title:            Lord of Winterfell (FROM 280 TO 299)
  Title:            Warden of the North (FROM 280 TO 299)
  Title:            Hand of the King (FROM 298 TO 298)

── Family ────────────────────────────────────────
  Spouse:           Catelyn Tully
  Mother:           Flint
  Father:           Rickard Stark
  Siblings:         Benjen Stark, Brandon Stark, Lyanna Stark

── Relationships ─────────────────────────────────
  Hand Of The King: Robert Baratheon I
  Ward:             Theon Greyjoy

── Life History ──────────────────────────────────
  Eddard Stark was born in 263 in Winterfell. He was the child of
  Flint and Rickard Stark. He married Catelyn Tully. He died in 299
  in Great Sept of Baelor.
```

The life history narrative is auto-generated from the structured data — a starting point that you can replace with a hand-written narrative in the person's `notes` field.

## Family Trees

### `glx ancestors` — Ancestor tree

Display the ancestor tree for a person using box-drawing characters:

```bash
glx ancestors person-robb-stark --generations 3
```

```
Robb Stark  (283 – 299)  person-robb-stark
├── Catelyn Tully  (264 – 299)  person-catelyn-tully
│   ├── Hoster Tully  (d. 299)  person-hoster-tully
│   │   └── Tully  person-tully
│   └── Minisa Whent  (d. BEF 300)  person-minisa-whent
└── Eddard Stark  (263 – 299)  person-eddard-stark
    ├── Flint  person-flint
    └── Rickard Stark  person-rickard-stark
        └── Edwyle Stark  person-edwyle-stark
```

Use `--generations` to limit depth (0 for unlimited). The tree traverses all parent-child relationship variants — biological, adoptive, foster, and step-parent.

### `glx descendants` — Descendant tree

```bash
glx descendants person-rickard-stark --generations 3
```

```
Rickard Stark  person-rickard-stark
├── Benjen Stark  person-benjen-stark
├── Brandon Stark  (d. BEF 300)  person-brandon-stark-213
├── Eddard Stark  (263 – 299)  person-eddard-stark
│   ├── Arya Stark  (b. 289)  person-arya-stark
│   ├── Brandon Stark  (b. 290)  person-brandon-stark-bran
│   ├── Jon Snow  (b. 283)  person-jon-snow
│   ├── Rickon Stark  (b. 295)  person-rickon-stark
│   ├── Robb Stark  (283 – 299)  person-robb-stark
│   └── Sansa Stark  (b. 286)  person-sansa-stark
└── Lyanna Stark  (266 – 283)  person-lyanna-stark
    └── Jon Snow  (b. 283)  person-jon-snow
```

Notice Jon Snow appears under both Eddard Stark and Lyanna Stark — the tree correctly traverses all parent-child relationships, revealing R+L=J through the data structure itself.

## Citations

### `glx cite` — Generate citation text

Generate formatted citation text from structured citation data:

```bash
# Format a specific citation
glx cite citation-agot-eddard-xv
```

```
"A Game of Thrones", novel, A Song of Ice and Fire (Novels) (2026-03-06),
  Eddard XV.
```

```bash
# Format all citations in the archive
glx cite
```

This assembles each citation from its source title, source type, repository name, access date, and locator — saving you from writing `citation_text` by hand for each of 247 citations.

## Format Conversion

### `glx split` and `glx join` — Convert between formats

Convert a single-file archive to multi-file or vice versa:

```bash
# Join a multi-file archive into a single file
glx join . westeros-combined.glx

# Split it back out
glx split westeros-combined.glx westeros-expanded/
```

### `glx export` — Export to GEDCOM

Export the archive to GEDCOM format for use with traditional genealogy software:

```bash
# Export to GEDCOM 5.5.1 (most compatible)
glx export . -o westeros.ged

# Export to GEDCOM 7.0
glx export . -o westeros.ged --format 70
```

The exporter reconstructs GEDCOM FAM records from GLX relationships, converts dates and places back to GEDCOM format, and preserves sources, citations, and notes.

::: warning
GEDCOM is a simpler format than GLX. Custom vocabularies (like `battle`, `coronation`, `ward`) will be exported as generic event/relationship types. Evidence chains are preserved as SOUR citations but lose the structured assertion model.
:::

## What to Try Next

Now that you've explored the Westeros archive, try these on your own data:

1. **Import a GEDCOM file**: `glx import family.ged -o family-archive` — see the [Migration Guide](/guides/migration-from-gedcom)
2. **Create an archive from scratch**: `glx init my-archive` — see the [Quickstart](/quickstart)
3. **Add custom vocabularies**: Define domain-specific event types and relationship types for your research
4. **Track evidence**: Build assertion chains from sources through citations to conclusions

## See Also

- [Westeros Example Archive](/examples/westeros/) — Details on the archive structure and contents
- [CLI Reference](/cli) — Full documentation for every command and flag
- [Quickstart Guide](/quickstart) — Create your first archive from scratch
- [Migration from GEDCOM](/guides/migration-from-gedcom) — Import existing GEDCOM files
