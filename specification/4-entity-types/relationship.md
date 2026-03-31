---
title: Relationship Entity
description: Connections between persons in GENEALOGIX - biological, legal, and social relationships
layout: doc
---

# Relationship Entity

[← Back to Entity Types](README)

## Overview

A Relationship entity defines a connection between two or more persons. Relationships can be biological (parent-child), legal (marriage, adoption), social (godparent), or custom types defined by the archive.

## File Format

All GENEALOGIX files use entity type keys at the top level:

```yaml
# Any .glx file (commonly in relationships/ directory)
relationships:
  rel-marriage-john-mary:
    type: marriage
    participants:
      - person: person-john-smith
      - person: person-mary-jones
    start_event: event-marriage-1875
```

**Key Points:**
- Entity ID is the map key (`rel-marriage-john-mary`)
- IDs can be descriptive or random, 1-64 alphanumeric/hyphens

## Fields

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| Entity ID (map key) | string | Unique identifier (alphanumeric/hyphens, 1-64 chars) |
| `type` | string | Relationship type from `vocabularies/relationship-types.glx` |
| `participants` | array | Array of participant objects defining who is in the relationship (at least 2 required) |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `properties` | object | Vocabulary-defined properties for the relationship |
| `start_event` | string | Event that started this relationship |
| `end_event` | string | Event that ended this relationship |
| `notes` | string | Research notes |

### Properties

Relationship properties capture additional details that don't fit into the standard structural fields. The following are standard properties from the default vocabulary; archives can define additional properties by extending the vocabulary.

| Property | Type | Description |
|----------|------|-------------|
| `started_on` | date | When the relationship began |
| `ended_on` | date | When the relationship ended |
| `location` | reference | Location where the relationship occurred (reference to Place) |
| `description` | string | Detailed description of the relationship |

Example:
```yaml
relationships:
  rel-business-partnership:
    type: business-partner
    participants:
      - person: person-john-smith
      - person: person-james-brown
    properties:
      started_on: "1875-03-15"
      ended_on: "1890-06-01"
      location: place-leeds
      description: "Co-owners of Smith & Brown Ironworks"
```

**See [Vocabularies - Relationship Properties](vocabularies#relationship-properties-vocabulary) for the full vocabulary definition.**

## Relationship Types

Relationship types are defined in the archive's `vocabularies/relationship-types.glx` file. Each archive includes standard types and can define custom types as needed.

**See [Vocabularies - Relationship Types](vocabularies#relationship-types-vocabulary) for:**
- Complete list of standard relationship types
- How to add custom relationship types
- Vocabulary file structure and examples
- Validation requirements

## Usage Patterns

### Marriage Relationship

```yaml
relationships:
  rel-marriage-john-mary:
    type: marriage
    participants:
      - person: person-john-smith
        role: spouse
      - person: person-mary-jones
        role: spouse
    start_event: event-marriage-1875
    properties:
      description: "Marriage at St Paul's Cathedral"
```

**Note: Marriage Event vs Marriage Relationship**

Marriage appears in both event types and relationship types by design:

- **Event type `marriage`** ([Event Entity](event)): The wedding ceremony - records the date, place, officiant, witnesses, and other ceremony details
- **Relationship type `marriage`** (this entity): The ongoing marital state - connects two spouses, tracks duration, and can reference when/how it ended

Link them using `start_event` to reference the ceremony. Use both when you have ceremony details; use just the relationship if you only know they were married without specifics about the wedding.

### Parent-Child Relationship

```yaml
relationships:
  rel-parents-alice:
    type: parent_child
    participants:
      - person: person-john-smith
        role: parent
      - person: person-mary-smith
        role: parent
      - person: person-alice-smith
        role: child
```

### Adoptive Parent-Child Relationship

```yaml
# The adoption event records the legal proceeding
events:
  event-adoption-1890:
    type: adoption
    date: "1890-03-15"
    place: place-county-court
    participants:
      - person: person-sarah-jones
        role: subject
    notes: "Legal adoption finalized"

# The relationship connects parent(s) and child
relationships:
  rel-adoptive-family-sarah:
    type: adoptive_parent_child
    participants:
      - person: person-james-smith
        role: adoptive_parent
      - person: person-elizabeth-smith
        role: adoptive_parent
      - person: person-sarah-jones
        role: adopted_child
    start_event: event-adoption-1890
    properties:
      description: "Sarah legally adopted by James and Elizabeth Smith"
```

**Note: Adoption Event vs Adoptive Parent-Child Relationship**

Similar to how marriage works in GLX:

- **Event type `adoption`** ([Event Entity](event)): The legal adoption proceeding - records the date, place, court, and other details of when the adoption was finalized
- **Relationship type `adoptive_parent_child`** (this entity): The ongoing parent-child relationship - connects adoptive parent(s) to the adopted child

Link them using `start_event` to reference the adoption event. Use both when you have details about when the adoption occurred; use just the relationship if you only know the relationship exists without specifics.

### Godparent Relationship

```yaml
# The baptism event records the ceremony where godparents were named
events:
  event-baptism-1885:
    type: baptism
    date: "1885-06-12"
    place: place-st-marys-church
    participants:
      - person: person-baby-william
        role: subject
      - person: person-uncle-james
        role: godparent
      - person: person-aunt-sarah
        role: godparent
    notes: "Baptism at St Mary's Church"

# The relationship represents the ongoing godparent-godchild bond
relationships:
  rel-godparent-james-william:
    type: godparent
    participants:
      - person: person-uncle-james
        role: godparent
      - person: person-baby-william
        role: godchild
    start_event: event-baptism-1885
```

**Note: Godparent Event Role vs Godparent Relationship**

Godparent appears in both participant roles and relationship types by design:

- **Participant role `godparent`** ([Event Entity](event)): A person's role at a baptism or christening ceremony - records who served as spiritual sponsor
- **Relationship type `godparent`** (this entity): The ongoing godparent-godchild bond that may continue throughout their lives

Use the participant role when recording event details; use the relationship type to model the ongoing connection. Link them with `start_event` when you have both.

### Custom Relationship

```yaml
relationships:
  rel-john-james-blood:
    type: blood-brother  # Custom type from vocabulary
    participants:
      - person: person-john-smith
      - person: person-james-brown
    start_event: event-ceremony-1845
    properties:
      description: "Blood brother ceremony witnessed by tribal elders"
```

## Participants Format

The `participants` array defines the people involved in the relationship and their roles:

```yaml
participants:
  - person: person-john-smith
    role: spouse
    notes: "Groom"
  - person: person-mary-jones
    role: spouse
    notes: "Bride"
```

Each participant object supports: `person` (required), `role`, `properties`, and `notes`. The `properties` field allows per-participant data and is validated against the `relationship_properties` vocabulary.

### Per-Participant Properties

When participants in a relationship need individual data (e.g., age at marriage, legal status), use the `properties` field on each participant entry:

```yaml
relationships:
  rel-marriage-john-mary:
    type: marriage
    participants:
      - person: person-john-smith
        role: spouse
        properties:
          description: "Widower at time of marriage"
      - person: person-mary-jones
        role: spouse
        properties:
          description: "First marriage"
    start_event: event-marriage-1875
```

Per-participant properties use the same vocabulary as relationship properties (`relationship-properties.glx`) and are validated against it. This is the same pattern used for [event participant properties](event#census-event-with-per-participant-properties).

## Participant Roles

Participant roles (spouse, parent, child, etc.) are defined in the archive's `vocabularies/participant-roles.glx` file.

**See [Vocabularies - Participant Roles](vocabularies#participant-roles-vocabulary) for:**
- Complete list of standard participant roles
- How to add custom roles
- Vocabulary file structure and examples
- Which roles apply to events vs. relationships

## Validation Rules

- Relationship type must be from the [relationship types vocabulary](vocabularies#relationship-types-vocabulary)
- `participants` array must contain at least 2 participants
- All person references must point to existing Person entities
- Participant roles should be from the [participant roles vocabulary](vocabularies#participant-roles-vocabulary) (unknown roles generate warnings)
- If `start_event` or `end_event` is specified, it must reference an existing Event entity

## File Organization

**Note:** File organization is flexible. Entities can be in any .glx file with any directory structure. The example below shows one-entity-per-file organization, which is recommended for collaborative projects (better git diffs) but not required.

Relationship files are typically stored in a `relationships/` directory:

```
relationships/
├── rel-marriage-001.glx
├── rel-marriage-002.glx
├── rel-parent-child-001.glx
├── rel-parent-child-002.glx
├── rel-adoption-001.glx
└── rel-godparent-001.glx
```


## GEDCOM Mapping

Relationships map to GEDCOM family and individual structures:

| GLX Field | GEDCOM Tag | Notes |
|-----------|------------|-------|
| `marriage` | FAM record | Family with MARR event |
| `parent_child` | FAM.CHIL + INDI.FAMC | Child link to family (no PEDI or PEDI unknown) |
| `biological_parent_child` | FAM.CHIL + INDI.FAMC with PEDI birth | Biological child relationship |
| `adoptive_parent_child` | FAM.CHIL + INDI.FAMC with PEDI adopted | Legally adopted child (use `adoption` event for ADOP tag) |
| `foster_parent_child` | FAM.CHIL + INDI.FAMC with PEDI foster | Foster care relationship |
| `sibling` | Shared FAM.CHIL | Siblings share parents |

**PEDI (Pedigree Linkage)**:

The PEDI tag in GEDCOM 5.5.1 specifies the type of parent-child relationship. During import, PEDI values are mapped to specific relationship types:

- `PEDI birth` → `biological_parent_child`
- `PEDI adopted` → `adoptive_parent_child`
- `PEDI foster` → `foster_parent_child`
- `PEDI unknown` or missing → `parent_child` (generic)
- `PEDI sealed` (LDS) → `parent_child` (not specifically modeled)

This allows GLX to preserve the distinction between biological, adoptive, and foster relationships that is critical for accurate genealogical research.

## Schema Reference

See [relationship.schema.json](../schema/v1/relationship.schema.json) for the complete JSON Schema definition.

## See Also

- [Person Entity](person) - Entities connected by relationships
- [Event Entity](event) - Events that start/end relationships
- [Core Concepts](../2-core-concepts#archive-owned-vocabularies) - Overview of vocabulary system
