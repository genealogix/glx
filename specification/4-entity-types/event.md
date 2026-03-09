---
title: Event Entity
description: Occurrences in time and place - lifecycle events and significant happenings
layout: doc
---

# Event Entity

[← Back to Entity Types](README)

## Overview

An Event entity represents a single occurrence in time, place, and context that is relevant to the family archive. Events are discrete happenings like birth, marriage, death, baptism, etc.

**Note:** Facts and attributes (occupation, residence, nationality, religion, etc.) are represented as temporal properties on Person entities, not as events. See [Person Entity](person) for details on temporal properties.

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
| `date` | string | Date or date range (see [Date Format](../2-core-concepts#date-format-standard)) |
| `place` | string | Reference to Place entity |
| `properties` | object | Vocabulary-defined properties |
| `notes` | string | Free-form notes |

### Participant Object Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `person` | string | Yes | Reference to Person entity |
| `role` | string | No | Role of participant |
| `properties` | object | No | Per-participant properties (e.g., `age_at_event`) |
| `notes` | string | No | Notes about participant's involvement |

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

**See [Vocabularies - Event Properties](vocabularies#event-properties-vocabulary) for:**
- Complete list of standard event properties
- How to add custom event properties

## Event Types

Event types are defined in the archive's `vocabularies/event-types.glx` file. Each archive includes standard types and can define custom types as needed.

**See [Vocabularies - Event Types](vocabularies#event-types-vocabulary) for:**
- Complete list of standard event types
- How to add custom event types
- Vocabulary file structure and examples
- Validation requirements

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
  event-1860-census-lane:
    type: census
    date: "1860"
    place: place-marston-sauk-wi
    participants:
      - person: person-d-lane
        role: subject
        properties:
          age_at_event: "45"
      - person: person-mary-lane
        role: subject
        properties:
          age_at_event: "28"
      - person: person-harriett-lane
        role: subject
        properties:
          age_at_event: "8"
```

Participant-level properties use the same vocabulary as event properties (`event-properties.glx`) and are validated against it. This avoids needing separate events for each person in a shared event like a census enumeration.

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Event files are typically stored in an `events/` directory:

```
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

**Note:** GEDCOM attributes like RESI (residence), OCCU (occupation), RELI (religion) are imported as temporal properties on Person entities, not events.

### Multi-Participant Events

For events with multiple participants, GLX uses the ASSO (Associate) tag pattern:

```
0 FAM
1 MARR
2 DATE 10 MAY 1875
2 PLAC Leeds, Yorkshire, England
1 ASSO person-witness1
2 RELA Witness
1 ASSO person-vicar
2 RELA Officiant
```

## Participant Roles

Participant roles (principal, witness, officiant, etc.) are defined in the archive's `vocabularies/participant-roles.glx` file.

**See [Vocabularies - Participant Roles](vocabularies#participant-roles-vocabulary) for:**
- Complete list of standard participant roles
- How to add custom roles
- Vocabulary file structure and examples
- Which roles apply to events vs. relationships

## Design Note: Participant Requirement

Events require at least one participant. GLX is a genealogy format, and every event should connect to at least one person in the archive. Historical context (wars, famines, plagues) that affected a person can be recorded in event `notes` or `properties`, with the affected person as a participant. Events with no connection to any person in the archive are outside the scope of a genealogy file.

## Validation Rules

- Event type must be from the [event types vocabulary](vocabularies#event-types-vocabulary)
- At least one participant is required
- Place, if referenced, must exist in the archive
- All person references must point to existing Person entities
- Date formats must follow genealogical date conventions
- Participant roles should be from the [participant roles vocabulary](vocabularies#participant-roles-vocabulary) (unknown roles generate warnings)

## Confidence and Provenance

All supporting evidence for an event is stored in [Assertion Entities](assertion) that reference the event in their `subject` field. This keeps the event record clean while allowing for a rich, explicit evidence trail.

## Schema Reference

See [event.schema.json](../schema/v1/event.schema.json) for the complete JSON Schema definition.

## See Also

- [Person Entity](person) - Participates in events
- [Assertion Entity](assertion) - Provides evidence for events
- [Place Entity](place) - Geographic context for events
- [Relationship Entity](relationship) - Multi-person events




