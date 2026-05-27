---
title: Basic Family Example
description: Foundational GENEALOGIX archive with two-parent household, events, places, and a complete evidence chain
layout: doc
---

# Basic Family Example

A foundational GENEALOGIX archive demonstrating a two-parent household with
two children, life events tied to dated places, and a minimal evidence chain
from repository to assertion.

This is the second example in the learning path. It builds on
[Minimal](../minimal/) by adding events, dates, places, and a single
source-citation-assertion chain. [Complete Family](../complete-family/) takes
the same pattern further with multiple sources, repositories, and assertions.

## Structure

```text
basic-family/
├── persons/
│   ├── person-mary-thompson.glx
│   ├── person-robert-thompson.glx
│   ├── person-alice-thompson.glx
│   └── person-robert-thompson-jr.glx
├── relationships/
│   ├── rel-marriage.glx
│   ├── rel-parent-alice.glx
│   └── rel-parent-robert-jr.glx
├── events/
│   ├── event-births.glx
│   └── event-marriage.glx
├── places/
│   ├── place-united-states.glx
│   ├── place-illinois.glx
│   └── place-springfield.glx
├── repositories/
│   └── repository-sangamon-county-courthouse.glx
├── sources/
│   └── source-sangamon-births.glx
├── citations/
│   └── citation-robert-birth.glx
├── assertions/
│   └── assertion-robert-birth-date.glx
├── vocabularies/           # Symlinks to standard vocabularies
└── README.md
```

## Family Overview

- **Robert Thompson** (born April 12, 1850, Springfield, IL)
- **Mary Thompson** (born August 30, 1852, Springfield, IL)
- Robert and Mary marry on June 15, 1875, in Springfield.
- **Alice Thompson** (born March 22, 1878, Springfield, IL)
- **Robert Thompson Jr.** (born November 5, 1881, Springfield, IL)

Robert's birth date is backed by a citation to the Sangamon County civil
birth register, demonstrating the full evidence chain.

## Persons

```yaml
# persons/person-robert-thompson.glx
persons:
  person-robert-thompson:
    properties:
      name:
        value: "Robert Thompson"
        fields:
          given: "Robert"
          surname: "Thompson"
      sex: "male"
```

The other three person files follow the same pattern. Dates and places are
recorded on events, not directly on persons.

## Relationships

```yaml
# relationships/rel-marriage.glx
relationships:
  rel-marriage:
    type: marriage
    participants:
      - person: person-mary-thompson
        role: spouse
      - person: person-robert-thompson
        role: spouse
    start_event: event-marriage-1875
```

The two parent-child relationships connect each child to both parents.

## Events

```yaml
# events/event-marriage.glx
events:
  event-marriage-1875:
    title: "Thompson Wedding"
    type: marriage
    date: "1875-06-15"
    place: place-springfield
    participants:
      - person: person-robert-thompson
        role: groom
      - person: person-mary-thompson
        role: bride
    properties:
      description: "Marriage of Robert Thompson and Mary Thompson in Springfield, Illinois"
```

`events/event-births.glx` contains four `type: birth` events — one for each
family member — each referencing `place-springfield`.

## Places

Places nest hierarchically via the `parent` field, so a single change to
`place-illinois` propagates to every entity that references it.

```yaml
# places/place-springfield.glx
places:
  place-springfield:
    name: "Springfield"
    type: city
    parent: place-illinois
    latitude: 39.7817
    longitude: -89.6501
    notes: "Capital of Illinois, seat of Sangamon County"
```

`place-illinois` has `parent: place-united-states`. `place-united-states`
has no parent — it sits at the top of the hierarchy.

## Evidence Chain

This example demonstrates the four-entity evidence chain for a single fact —
Robert Thompson's birth date:

**Repository → Source → Citation → Assertion**

```yaml
# repositories/repository-sangamon-county-courthouse.glx
repositories:
  repository-sangamon-county-courthouse:
    name: "Sangamon County Courthouse - County Clerk"
    type: registry
    address: "200 South Ninth Street, Springfield, IL 62701"
    website: "https://www.sangamonil.gov/Departments/County-Clerk"
    notes: "Civil records repository for Sangamon County, Illinois, including birth and marriage registers"
```

```yaml
# sources/source-sangamon-births.glx
sources:
  source-sangamon-births:
    title: "Sangamon County, Illinois - Register of Births"
    type: vital_record
    repository: repository-sangamon-county-courthouse
    date: "FROM 1850 TO 1900"
    properties:
      description: "Civil birth register maintained by the Sangamon County Clerk"
```

```yaml
# citations/citation-robert-birth.glx
citations:
  citation-robert-birth:
    source: source-sangamon-births
    properties:
      locator: "Volume 1, Page 47, Entry 312"
      text_from_source: "Robert Thompson, male, born April 12, 1850, Springfield"
    notes: "Primary source - civil birth register"
```

```yaml
# assertions/assertion-robert-birth-date.glx
assertions:
  assertion-robert-birth-date:
    subject:
      event: event-birth-robert
    property: date
    value: "1850-04-12"
    citations:
      - citation-robert-birth
    confidence: high
    status: proven
    notes: "Birth date confirmed by the Sangamon County civil birth register"
```

The other birth dates and the marriage date are recorded directly on their
events without supporting assertions — a common pattern when only one fact in
an archive has been formally evidenced yet. See
[Complete Family](../complete-family/) for an example with multiple
assertions, multiple sources, and conflicting evidence.

## Validation

```bash
glx validate
# ✓ All files valid
```

## What This Demonstrates

- **Persons and relationships** in a nuclear family
- **Events** with dates, places, and typed participant roles
- **Hierarchical places** via `parent` references (city → state → country)
- **The full evidence chain** — repository, source, citation, assertion — for one fact
- **Direct properties vs. assertions** — most dates here are direct on events; one is backed by an assertion
- **Cross-references** between entities resolved at validation time
- **Standard vocabularies** for event types, place types, source types, etc.

## Next Steps

- See [Complete Family](../complete-family/) for multiple assertions per
  subject, multiple sources, and a richer evidence web.
- See [Assertion Workflow](../assertion-workflow/) for direct properties vs.
  assertion-backed properties and how to iteratively raise confidence.
