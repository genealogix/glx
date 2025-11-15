# Event/Fact Entity

[← Back to Entity Types](README.md)

## Overview

An Event (also called a Fact) entity represents a single occurrence in time, place, and context that is relevant to the family archive. Events can be lifecycle events (birth, marriage, death), attribute facts (occupation, residence), or custom events.

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
- **Baptism**, **Burial**, **Cremation**, **Christening**

### Attributes

Characteristic facts about a person:
- **Occupation**, **Residence**, **Education**, **Religion**, **Title**
- **Nationality**, **Ethnicity**

### Custom Events

Domain-specific events:
- Military service, Migration/Immigration, Land transactions, Legal proceedings

## Properties

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `version` | string | Schema version (e.g., "1.0") |
| `type` | string | Event type (birth, death, marriage, occupation, etc.) |

### Optional Properties

| Property | Type | Description |
|----------|------|-------------|
| `date` | string or object | Date as string or object with fuzzy support |
| `date.value` | string | Main date expression (if object) |
| `date.range_start` | string | Fuzzy date range start (if object) |
| `date.range_end` | string | Fuzzy date range end (if object) |
| `place` | string | Reference to Place entity |
| `participants` | array | People involved in the event |
| `participant.person` | string | Reference to Person entity |
| `participant.role` | string | Role of participant |
| `participant.notes` | string | Notes about participant |
| `value` | string | Value for attribute facts (e.g., "blacksmith" for occupation) |
| `properties` | object | Vocabulary-defined properties for the event |
| `age_text` | string | Age at time of event |
| `cause_text` | string | Cause for event |
| `description` | string | Narrative description |
| `notes` | string | Free-form notes |

### Date Structure

Events support fuzzy dates using multiple formats:

```yaml
date:
  value: "about 1850"
  range_start: "1845"
  range_end: "1855"
```

### Participant Structure

```yaml
participants:
  - person: "person-abc123de"
    role: "principal"
    notes: "The bride"
  - person: "person-def456gh"
    role: "officiant"
    notes: "The minister"
  - person: "person-ghi789jk"
    role: "witness"
```

## Event Types

Event types are defined in the archive's `vocabularies/event-types.glx` file. Each archive includes standard types and can define custom types as needed.

**See [Vocabularies - Event Types](vocabularies.md#event-types-vocabulary) for:**
- Complete list of standard event types
- How to add custom event types
- Vocabulary file structure and examples
- Validation requirements

## Usage Patterns

### Birth Event Example

```yaml
# events/event-birth.glx
events:
  event-birth-john:
    type: birth
    date: "1850-01-15"
    place: place-leeds
    participants:
      - person: person-john-smith
        role: principal
```

### Complex Event with Multiple Participants

```yaml
# events/event-marriage.glx
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
    description: "Marriage celebrated at St Paul's Cathedral"
```

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Event files are typically stored in a `events/` directory organized by type:

```
events/
├── lifecycle/
│   ├── event-birth-001.glx
│   ├── event-marriage-001.glx
│   ├── event-death-001.glx
│   └── event-adoption-001.glx
├── attributes/
│   ├── event-occupation-001.glx
│   ├── event-residence-001.glx
│   └── event-religion-001.glx
└── custom/
    ├── event-military-001.glx
    └── event-migration-001.glx
```

Or embedded directly in person records as nested structures.

## GEDCOM Mapping

Most events map directly to GEDCOM tags:

| GLX Type | GEDCOM Tag | Notes |
|----------|-----------|-------|
| `birth` | INDI.BIRT | Individual birth |
| `death` | INDI.DEAT | Individual death |
| `marriage` | FAM.MARR | Family marriage |
| `divorce` | FAM.DIV | Family divorce |
| `residence` | INDI.RESI | Residence attribute |
| `occupation` | INDI.OCCU | Occupation attribute |

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

**See [Vocabularies - Participant Roles](vocabularies.md#participant-roles-vocabulary) for:**
- Complete list of standard participant roles
- How to add custom roles
- Vocabulary file structure and examples
- Which roles apply to events vs. relationships

## Validation Rules

- Event type must be defined in `vocabularies/event-types.glx`
- At least one participant must be present (except for attribute-type events)
- Place, if referenced, must exist in the archive
- All person references must point to existing Person entities
- Date formats must follow genealogical date conventions
- Participant roles must be defined in `vocabularies/participant-roles.glx`

## Confidence and Provenance

All supporting evidence for an event is stored in [Assertion Entities](assertion.md) that reference the event in their `subject` field. This keeps the event record clean while allowing for a rich, explicit evidence trail.

## See Also

- [Person Entity](person.md) - Contains event references
- [Assertion Entity](assertion.md) - Provides evidence for events
- [Place Entity](place.md) - Geographic context for events
- [Relationship Entity](relationship.md) - Multi-person events




