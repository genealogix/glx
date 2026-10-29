---
title: Event Entity
description: Occurrences in time and place - lifecycle events and significant happenings
layout: doc
---

# Event Entity

[← Back to Entity Types](README.md)

## Overview

An Event entity represents a single occurrence in time, place, and context that is relevant to the family archive. Events are discrete happenings like birth, marriage, death, baptism, etc.

**Note:** Facts and attributes such as occupation, nationality, and religion are typically represented as temporal properties on Person entities rather than events. Residence is an exception: it can be modeled as either a person property (for simple place references) or a `residence` event (when participants, dates, or detailed evidence are needed). See [Person Entity](person.md) for details on temporal properties.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in events/ directory)
events:
  event-birth-john-1850:
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: subject
```

**Key Points:**

- Entity ID is the map key (`event-birth-john-1850`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Core Concepts

### Lifecycle Events

Standard events that occur in a person's life:

- **Birth**, **Death**, **Marriage**, **Divorce**, **Engagement**, **Adoption**
- **Baptism**, **Confirmation**, **Bar/Bat Mitzvah**, **Burial**, **Cremation**

### Custom Events

Domain-specific events can be added via vocabularies:

- Military service, Migration/Immigration, Land transactions, Legal proceedings

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `type` | string | Event type (birth, death, marriage, etc.) |
| `participants` | array | People involved in the event (at least one required) |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `title` | string | Short label for the event (e.g., "1860 Census — Webb Household"). Distinct from `properties.description`, which is a longer narrative |
| `date` | string | Date or date range (see [Date Format](../2-core-concepts.md#date-format-standard)) |
| `place` | string | Reference to Place entity |
| `household` | object | Census events only: the household's composition as enumerated (tick-mark tally rows). See [Census Households](#census-households) |
| `neighbors` | array | Census events only: households enumerated nearby on the page. See [Census Page Neighbors](#census-page-neighbors) |
| `properties` | object | Vocabulary-defined properties |
| `notes` | string \| string[] | Free-form notes |

**GEDCOM import note:** Titles are auto-generated from the event type and participant names (for example, "Birth of Robert Webb"). The date is not included because it is already represented by the `date` field. Archives imported before this change may still contain auto-generated titles ending in a parenthetical value derived from date parsing (for example, "Birth of John Smith (1850)"); that value is not guaranteed to be a correct year and may be a day-of-month for non-ISO dates. Run `glx migrate --strip-event-title-year` to remove the stale suffix from such archives.

### Participant Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `person` | string | Yes | Reference to Person entity |
| `role` | string | No | Role of participant |
| `properties` | object | No | Per-participant properties (e.g., `age_at_event`) |
| `notes` | string \| string[] | No | Notes about participant's involvement |

### Participant Structure

```yaml
participants:
  - person: "person-abc123de"
    role: "subject"
    notes: "The bride"
  - person: "person-def456gh"
    role: "officiant"
    notes: "The minister"
  - person: "person-ghi789jk"
    role: "witness"
```

### Properties

Event properties are defined in the archive's `vocabularies/event-properties.glx` file. Standard properties include:

- `description` - Narrative description of the event

**Note:** Event timing and location are handled by the `date` and `place` fields, not properties.

**See [Vocabularies - Event Properties](vocabularies.md#event-properties-vocabulary) for:**

- Complete list of standard event properties
- How to add custom event properties

## Event Types

Event types are defined in the archive's `vocabularies/event-types.glx` file. Each archive includes standard types and can define custom types as needed.

**See [Vocabularies - Event Types](vocabularies.md#event-types-vocabulary) for:**

- Complete list of standard event types
- How to add custom event types
- Vocabulary file structure and examples
- Validation requirements

### Property Records: Probate, Will and Land Transaction

Property records fall under three standard `legal` event types. `probate` and `will` cover inheritance, and `land_transaction` covers the transfer, entry or encumbrance of real property: deeds, land entries, patents, grants and mortgages. For pre-1850 research, land records are often the largest body of evidence for a person.

A `land_transaction` records its kind in the standard `event_subtype` property. The documented subtypes are:

| `event_subtype` | Meaning |
|-----------------|---------|
| `deed` | Deed of sale or gift between private parties |
| `patent` | Patent conveying title from a government to the first private owner |
| `grant` | Grant of land by a government or proprietor (for example a colonial, headright or bounty-land grant) |
| `entry` | Land entry or warrant: the claim or purchase that precedes a patent, including instalment payments |
| `mortgage` | Mortgage or deed of trust encumbering the land |
| `partition` | Division of land among co-owners or heirs |
| `quitclaim` | Release of a claim or interest without warranty of title |

The list is a convention, not a closed set: other values are accepted. The participant roles that go with land records are `grantor`, `grantee`, `witness` and `adjoining_owner` (a neighbour named only in the boundary description).

```yaml
events:
  event-lewis-little-patent-1818:
    type: land_transaction
    date: "1818-06-10"
    place: place-crawford-county-il
    participants:
      - person: person-lewis-little
        role: grantee
    properties:
      event_subtype: patent
```

GEDCOM has no dedicated land-transaction tag (`PROP` is a possessions attribute, not a transaction). Its intended representation is `EVEN` with a `TYPE`, but the current exporter omits `land_transaction` events. Support depends on the unmapped-event fallback ([#1320](https://github.com/genealogix/glx/issues/1320)) and export of participants such as `grantor` and `grantee` ([#1321](https://github.com/genealogix/glx/issues/1321)); neither export change is included in this vocabulary addition.

## Usage Patterns

### Birth Event Example

```yaml
events:
  event-birth-john:
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: subject
```

### Complex Event with Multiple Participants

```yaml
events:
  event-marriage-john-mary:
    type: marriage
    date: "1875-05-10"
    place: place-stpauls
    participants:
      - person: person-john-smith
        role: groom
      - person: person-mary-jones
        role: bride
      - person: person-thomas-brown
        role: witness
        notes: "First witness"
      - person: person-sarah-white
        role: witness
        notes: "Second witness"
      - person: person-reverend-black
        role: officiant
    properties:
      description: "Marriage celebrated at St Paul's Cathedral"
```

### Census Event with Per-Participant Properties

Census events and similar multi-person events can record per-participant data using the `properties` field on each participant entry:

```yaml
events:
  event-1860-census-webb:
    title: "1860 Census — Webb Household"
    type: census
    date: "1860"
    place: place-oakdale-hartford-wi
    participants:
      - person: person-robert-webb
        role: subject
        properties:
          age_at_event: "45"
      - person: person-jane-webb
        role: subject
        properties:
          age_at_event: "28"
      - person: person-harriett-webb
        role: subject
        properties:
          age_at_event: "8"
```

Participant-level properties use the same vocabulary as event properties (`event-properties.glx`) and are validated against it. This avoids needing separate events for each person in a shared event like a census enumeration.

A participant may also carry `relationship_to_head` (head, wife, son, boarder...), the column recorded by the US census from 1880 and by many other national censuses. Tools such as `glx households` list the participant whose `relationship_to_head` is `head` first; without it, the first named participant in a `principal`/`subject` role is taken as the head.

### Census Households

Before 1850 the US federal census, like many state and colonial censuses, named only the head of household. Everyone else is a tick mark in an age-and-sex column. Two things record such a schedule without overstating it:

1. The event's `household.tally` holds the tick-mark columns as enumerated, one row per column.
2. A person the researcher identifies with one of the tick marks is attached as a participant with role `household_member` and the participant property `named: false`. The record does not name them; the identification is the researcher's inference, and its evidence belongs in assertions like any other conclusion.

```yaml
events:
  event-1820-census-little:
    title: "1820 Census — Little Household"
    type: census
    date: "1820"
    place: place-wythe-county-va
    participants:
      - person: person-james-little
        role: principal
      - person: person-elizabeth-starr
        role: household_member
        properties:
          named: false
      - person: person-mary-little
        role: household_member
        properties:
          named: false
    household:
      tally:
        - sex: male
          age_from: 45
          count: 1
          status: free white
        - sex: female
          age_from: 45
          count: 1
          status: free white
        - sex: female
          age_from: 10
          age_to: 15
          count: 2
          status: free white
        - count: 1
          status: engaged in agriculture
```

#### Tally Row Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `count` | integer | Yes | Number of persons tallied in the column (at least 1) |
| `sex` | string | No | Sex of the persons counted, a `sex_types` vocabulary key (unknown keys are a warning) |
| `age_from` | integer | No | Lower bound of the age bracket, inclusive. Omit for "under N" |
| `age_to` | integer | No | Upper bound of the age bracket, inclusive. Omit for "N and upwards". The 1800 column "of 10 and under 16" is `age_from: 10`, `age_to: 15` |
| `status` | string | No | Free-text column status: `free white`, `free colored`, `enslaved`, `foreigner not naturalized`, `engaged in agriculture`... A row with only a status counts persons in that column regardless of age and sex |
| `notes` | string \| string[] | No | Notes about the row |

The head is normally counted in the population tally too. Only mutually exclusive population columns that together cover the whole household can be summed to obtain household size. Other columns can overlap those counts: in the example above, the demographic rows count four people, while the `engaged in agriculture` row counts one person who can also be included in those demographic rows. Summing all five marks would overcount the household. Brackets are free integers: GLX does not yet validate them against each census's column headings.

Tooling treats a `named: false` participant as present in the census: `glx analyze` does not suggest searching that census for them, `glx coverage` ticks the census row, and `glx households` lists them as "counted, not named".

### Census Page Neighbors

FAN (Friends, Associates, Neighbors) research records the households enumerated next to the person of interest. `neighbors` lists them without requiring a Person entity for each:

```yaml
events:
  event-1850-census-baker:
    type: census
    date: "1850"
    place: place-brooklyn-sauk-wi
    participants:
      - person: person-abram-baker
        role: principal
    neighbors:
      - name: "Henry Jeffries"
        person: person-henry-jeffries
        position: previous_household
        page: "12"
        line: "3"
      - name: "James M Clark"
        position: next_household
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | One of `name`/`person` | Head of the neighboring household as written on the page |
| `person` | string | One of `name`/`person` | Reference to a Person entity, when the neighbor is in the archive |
| `position` | string | No | Position relative to this household. Suggested values: `previous_household`, `next_household`, `same_page`, `previous_page`, `next_page` |
| `page` | string | No | Census page or sheet of the neighbor |
| `line` | string | No | Line number on the page |
| `notes` | string \| string[] | No | Notes about the neighbor |

`glx households --neighbors` shows them, and `glx cluster` counts a neighbor linked to a person as an associate.

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Event files are typically stored in an `events/` directory:

```text
events/
├── event-birth-001.glx
├── event-marriage-001.glx
├── event-death-001.glx
├── event-baptism-001.glx
└── event-adoption-001.glx
```

## GEDCOM Mapping

Most events map directly to GEDCOM tags:

| GLX Event Type | GEDCOM Tag | Notes |
|-----------|------------|-------|
| `birth` | INDI.BIRT | Individual birth |
| `death` | INDI.DEAT | Individual death |
| `marriage` | FAM.MARR | Family marriage |
| `divorce` | FAM.DIV | Family divorce |
| `baptism` | INDI.BAPM/CHR | Baptism or christening |
| `burial` | INDI.BURI | Burial |

An event type whose vocabulary entry has no `gedcom` tag (for example `taxation`, `voter_registration`, or an archive-defined type) exports as a generic `EVEN` whose `TYPE` is the type's label, followed by `: <subtype>` when the event has an `event_subtype`. Import reads that `TYPE` back into the original event type.

**Note:** GEDCOM attributes like OCCU (occupation) and RELI (religion) are imported as temporal properties on Person entities, not events. RESI is imported either as a temporal residence property on the Person entity (when a `PLAC` is present) or, when no `PLAC` is present (e.g., bare `RESI Y` or `RESI` with only `DATE`/`TYPE`), as a separate Event entity whose `type` field is `residence`.

### Multi-Participant Events

An individual event is written under the record of each `principal` (or `subject`) participant. An event with no principal is written under its household participants (`household_head`, `boarder`), such as a census entered with only the head of household. A couple's event is written under their `FAM`.

In GEDCOM 7.0, every other participant becomes an `ASSO` under the event with a `ROLE`: `WITN`, `GODP`, `OFFICIATOR`, `CHIL`, `FATH`/`MOTH`/`PARENT`, `HUSB`/`WIFE`/`SPOU`, or `OTHER` with a `PHRASE` naming the role. A `gedcom` value on the role's vocabulary entry that is a ROLE enumeration value takes precedence:

```text
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 10 MAY 1875
2 PLAC Leeds, Yorkshire, England
2 ASSO @I3@
3 ROLE WITN
2 ASSO @I4@
3 ROLE OTHER
4 PHRASE Informant
```

GEDCOM 5.5.1 permits `ASSO` only directly under `INDI`, not under an event or `FAM`. Export therefore writes each association on the event's principal or household host; for a family event it writes it on each known spouse. `RELA` carries the participant's role label, and `NOTE` carries participant notes and descriptive event context (type, GEDCOM tag, archive event ID, date, place and subjects). A role label longer than 5.5.1's 25-character `RELA` limit, or containing a line break, is preserved in a note with `RELA Participant`.

This 5.5.1 representation uses standard structures for interoperability. Its event link is descriptive text, and importing it does not reconstruct the event's participants. Use GEDCOM 7.0 when structured event associations and participant-role round trips are needed.

An event that no record can carry, such as one whose only participant is a witness, is named in the export warnings.

## Participant Roles

Participant roles (principal, witness, officiant, etc.) are defined in the archive's `vocabularies/participant-roles.glx` file.

**See [Vocabularies - Participant Roles](vocabularies.md#participant-roles-vocabulary) for:**

- Complete list of standard participant roles
- How to add custom roles
- Vocabulary file structure and examples
- Which roles apply to events vs. relationships

## Design Note: Participant Requirement

Events require at least one participant. GLX is a genealogy format, and every event should connect to at least one person in the archive. Historical context (wars, famines, plagues) that affected a person can be recorded in event `notes` or `properties`, with the affected person as a participant. Events with no connection to any person in the archive are outside the scope of a genealogy file.

## Validation Rules

- Event type must be from the [event types vocabulary](vocabularies.md#event-types-vocabulary)
- At least one participant is required
- Place, if referenced, must exist in the archive
- All person references must point to existing Person entities
- Date formats must follow the [date format standard](../2-core-concepts.md#date-format-standard) (invalid formats generate warnings)
- Participant roles must be from the [participant roles vocabulary](vocabularies.md#participant-roles-vocabulary); an unknown role is an error, like any other structural type field (see [Vocabulary Validation](vocabularies.md#validation-errors-hard-failures))
- Household tally rows need a `count` of at least 1, non-negative ages, and `age_from` no greater than `age_to` (errors). A row with no sex, age bracket, or status, an unknown `sex`, more `named: false` participants than the tally counts, and `household`/`neighbors` on a non-census event are warnings
- Each neighbor needs a `name` or a `person`, and a `person` must reference an existing Person entity
- A participant role whose `applies_to` excludes `event` generates a warning (see [Participant Roles - applies_to](vocabularies.md#applies_to-semantics))

## Confidence and Provenance

All supporting evidence for an event is stored in [Assertion Entities](assertion.md) that reference the event in their `subject` field. This keeps the event record clean while allowing for a rich, explicit evidence trail.

## Schema Reference

See [event.schema.json](../schema/v1/event.schema.json) for the complete JSON Schema definition.

## See Also

- [Person Entity](person.md) - Participates in events
- [Assertion Entity](assertion.md) - Provides evidence for events
- [Place Entity](place.md) - Geographic context for events
- [Relationship Entity](relationship.md) - Multi-person events
