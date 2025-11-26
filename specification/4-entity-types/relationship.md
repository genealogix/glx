---
title: Relationship Entity
description: Connections between persons in GENEALOGIX - biological, legal, and social relationships
layout: doc
---

# Relationship Entity

[← Back to Entity Types](README.md)

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
| `participants` | array | Array of participant objects defining who is in the relationship |

### Optional Fields

| Field | Type | Description |
|-------|------|-------------|
| `properties` | object | Vocabulary-defined properties for the relationship |
| `start_event` | string | Event that started this relationship |
| `end_event` | string | Event that ended this relationship |
| `description` | string | Narrative description of the relationship |
| `notes` | string | Research notes |
| `tags` | array | Tags for categorization |

## Relationship Types

Relationship types are defined in the archive's `vocabularies/relationship-types.glx` file. Each archive includes standard types and can define custom types as needed.

**See [Vocabularies - Relationship Types](vocabularies.md#relationship-types-vocabulary) for:**
- Complete list of standard relationship types
- How to add custom relationship types
- Vocabulary file structure and examples
- Validation requirements

## Usage Patterns

### Marriage Relationship

```yaml
# relationships/rel-marriage.glx
relationships:
  rel-marriage-john-mary:
    type: marriage
    participants:
      - person: person-john-smith
        role: spouse
      - person: person-mary-jones
        role: spouse
    start_event: event-marriage-1875
    description: "Marriage at St Paul's Cathedral"
```

**Note: Marriage Event vs Marriage Relationship**

Marriage appears in both event types and relationship types by design:

- **Event type `marriage`** ([Event Entity](event.md)): The wedding ceremony - records the date, place, officiant, witnesses, and other ceremony details
- **Relationship type `marriage`** (this entity): The ongoing marital state - connects two spouses, tracks duration, and can reference when/how it ended

Link them using `start_event` to reference the ceremony. Use both when you have ceremony details; use just the relationship if you only know they were married without specifics about the wedding.

### Parent-Child Relationship

```yaml
# relationships/rel-parent-child.glx
relationships:
  rel-parents-alice:
    type: parent-child
    participants:
      - person: person-john-smith
        role: parent
      - person: person-mary-smith
        role: parent
      - person: person-alice-smith
        role: child
```

### Adoption Relationship

```yaml
# relationships/rel-adoption.glx
relationships:
  rel-adoption-sarah:
    type: adoption
    participants:
      - person: person-adoptive-father
        role: adoptive-parent
      - person: person-adoptive-mother
        role: adoptive-parent
      - person: person-adopted-child
        role: adopted-child
    start_event: event-adoption-1890
```

### Custom Relationship

```yaml
# relationships/rel-blood-brothers.glx
relationships:
  rel-john-james-blood:
    type: blood-brother  # Custom type from vocabulary
    participants:
      - person: person-john-smith
      - person: person-james-brown
    start_event: event-ceremony-1845
    description: "Blood brother ceremony witnessed by tribal elders"
```

## Participants Format

The `participants` array provides an alternative format that includes role information:

```yaml
participants:
  - person: person-john-smith
    role: spouse
    notes: "Groom"
  - person: person-mary-jones
    role: spouse
    notes: "Bride"
```

## Participant Roles

Participant roles (spouse, parent, child, etc.) are defined in the archive's `vocabularies/participant-roles.glx` file.

**See [Vocabularies - Participant Roles](vocabularies.md#participant-roles-vocabulary) for:**
- Complete list of standard participant roles
- How to add custom roles
- Vocabulary file structure and examples
- Which roles apply to events vs. relationships

## Validation Rules

- Relationship type must be defined in `vocabularies/relationship-types.glx`
- `participants` array must contain at least 2 participants
- All person references must point to existing Person entities
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

| GLX Type | GEDCOM Structure | Notes |
|----------|------------------|-------|
| `marriage` | FAM record | Family with MARR event |
| `parent-child` | FAM.CHIL + INDI.FAMC | Child link to family (no PEDI or PEDI unknown) |
| `biological-parent-child` | FAM.CHIL + INDI.FAMC with PEDI birth | Biological child relationship |
| `adoptive-parent-child` | FAM.CHIL + INDI.FAMC with PEDI adopted | Legally adopted child |
| `foster-parent-child` | FAM.CHIL + INDI.FAMC with PEDI foster | Foster care relationship |
| `adoption` | FAM.CHIL + ADOP | Adoption event |
| `sibling` | Shared FAM.CHIL | Siblings share parents |

**PEDI (Pedigree Linkage)**:

The PEDI tag in GEDCOM 5.5.1 specifies the type of parent-child relationship. During import, PEDI values are mapped to specific relationship types:

- `PEDI birth` → `biological-parent-child`
- `PEDI adopted` → `adoptive-parent-child`
- `PEDI foster` → `foster-parent-child`
- `PEDI unknown` or missing → `parent-child` (generic)
- `PEDI sealed` (LDS) → `parent-child` (not specifically modeled)

This allows GLX to preserve the distinction between biological, adoptive, and foster relationships that is critical for accurate genealogical research.

## See Also

- [Person Entity](person.md) - Entities connected by relationships
- [Event Entity](event.md) - Events that start/end relationships
- [Core Concepts](../2-core-concepts.md#repository-owned-vocabularies) - Overview of vocabulary system
