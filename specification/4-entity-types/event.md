# Event/Fact Entity

[← Back to Entity Types](README.md)

## Overview

An Event (also called a Fact) entity represents a single occurrence in time, place, and context that is relevant to the family archive. Events can be lifecycle events (birth, marriage, death), attribute facts (occupation, residence), or custom events.

## Core Concepts

### Lifecycle Events

Standard events that occur in a person's life, recognized across genealogical systems:

- **Birth**: Person's birth
- **Death**: Person's death  
- **Marriage**: Union of two people
- **Divorce**: Dissolution of marriage
- **Engagement**: Betrothal
- **Adoption**: Adoption event
- **Baptism**: Religious baptism ceremony
- **Burial**: Interment
- **Cremation**: Cremation of remains
- **Christening**: Religious christening

### Attributes

Characteristic facts about a person or family:

- **Occupation**: Employment or trade
- **Residence**: Place of residence during a period
- **Education**: Educational institution or achievement
- **Religion**: Religious affiliation or practice
- **Title**: Nobility, courtesy, or honorific titles
- **Nationality**: National citizenship
- **Ethnicity**: Ethnic or racial identification

### Custom Events

Researchers can define domain-specific events relevant to their research:

- Military service
- Migration/Immigration
- Land transactions
- Legal proceedings
- Business partnerships
- Community activities

## Properties

### Required Properties

| Property | Type | Description |
|----------|------|-------------|
| `id` | string | Unique identifier (format: `event-[hex8]`) |
| `version` | string | Schema version (e.g., "1.0") |
| `type` | string | Event type (birth, death, marriage, occupation, etc.) |

### Optional Properties

| Property | Type | Description |
|----------|------|-------------|
| `date` | object | Date information with fuzzy support |
| `date.value` | string | Main date expression |
| `date.range_start` | string | Fuzzy date range start |
| `date.range_end` | string | Fuzzy date range end |
| `place_id` | string | Reference to Place entity |
| `participants` | array | People involved in the event |
| `participant.person_id` | string | Reference to Person entity |
| `participant.role` | string | Role of participant (principal, witness, officiant, etc.) |
| `participant.notes` | string | Notes about participant's involvement |
| `attribute_value` | string | For attribute facts: the value (e.g., "blacksmith" for occupation) |
| `age_text` | string | Age at time of event (e.g., "about 25", "infant") |
| `cause_text` | string | Cause for event (e.g., "pneumonia" for death) |
| `description` | string | Narrative description of the event |
| `restrictions` | array | Restriction codes (confidential, locked, privacy) |
| `created_at` | datetime | Creation timestamp |
| `created_by` | string | User who created this record |
| `modified_at` | datetime | Last modification timestamp |
| `modified_by` | string | User who last modified this record |
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
  - person_id: "person-abc123de"
    role: "principal"
    notes: "The bride"
  - person_id: "person-def456gh"
    role: "officiant"
    notes: "The minister"
  - person_id: "person-ghi789jk"
    role: "witness"
```

## Standard Event Types

GENEALOGIX defines standardized event type codes for interoperability:

| Type | Category | GEDCOM Equivalent |
|------|----------|-------------------|
| `birth` | Lifecycle | BIRT |
| `death` | Lifecycle | DEAT |
| `marriage` | Lifecycle | MARR |
| `divorce` | Lifecycle | DIV |
| `engagement` | Lifecycle | ENGA |
| `adoption` | Lifecycle | ADOP |
| `baptism` | Religious | BAPM |
| `confirmation` | Religious | CONF |
| `bar_mitzvah` | Religious | BARM |
| `bat_mitzvah` | Religious | BATM |
| `burial` | Lifecycle | BURI |
| `cremation` | Lifecycle | CREM |
| `residence` | Attribute | RESI |
| `occupation` | Attribute | OCCU |
| `title` | Attribute | TITL |
| `nationality` | Attribute | NATI |
| `religion` | Attribute | RELI |
| `education` | Attribute | EDUC |

## Usage Patterns

### In Person Records

```yaml
events:
  - id: "event-birth001"
    type: "birth"
    date:
      value: "15 JAN 1850"
      range_start: "1850"
      range_end: "1850"
    place_id: "place-leeds123"
    participants:
      - person_id: "person-abc123de"
        role: "principal"
    created_at: "2025-01-15T10:30:00Z"
```

### Complex Event with Multiple Participants

```yaml
id: event-marr001
version: "1.0"
type: "marriage"
date:
  value: "10 MAY 1875"
  range_start: "1875"
  range_end: "1875"
place_id: "place-stpauls"
participants:
  - person_id: "person-groom"
    role: "groom"
  - person_id: "person-bride"
    role: "bride"
  - person_id: "person-witness1"
    role: "witness"
    notes: "First witness"
  - person_id: "person-witness2"
    role: "witness"
    notes: "Second witness"
  - person_id: "person-vicar"
    role: "officiant"
description: "Marriage celebrated at St Paul's Cathedral"
created_at: "2025-01-15T10:30:00Z"
created_by: "researcher@example.com"
```

## File Organization

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

## Validation Rules

- Event type must be from standard or registered custom types
- At least one participant must be present (except for attribute-type events)
- Place, if referenced, must exist in the archive
- All person references must point to existing Person entities
- Date formats must follow genealogical date conventions
- Participant roles must be from registered role types

## Confidence and Provenance

Events inherit assertion model from parent person or relationship:

```yaml
id: event-birth001
version: "1.0"
type: "birth"
date:
  value: "15 JAN 1850"
assertions:
  - assertion-birth001
```

All supporting evidence for the event is stored in referenced Assertion entities.

## See Also

- [Person Entity](person.md) - Contains event references
- [Assertion Entity](assertion.md) - Provides evidence for events
- [Place Entity](place.md) - Geographic context for events
- [Relationship Entity](relationship.md) - Multi-person events




