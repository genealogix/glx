---
title: Basic Family Example
description: Foundational GENEALOGIX archive with two-parent household and basic relationships
layout: doc
---

# Basic Family Example

A foundational GENEALOGIX archive demonstrating a two-parent household
with two children and basic relationship entries.

## Structure

```
basic-family/
├── .glx-archive/
│   ├── config.glx
│   └── schema-version.glx
├── persons/
│   ├── person-mother.glx
│   ├── person-father.glx
│   ├── person-child-alice.glx
│   └── person-child-bob.glx
├── relationships/
│   ├── rel-marriage.glx
│   ├── rel-parent-alice.glx
│   └── rel-parent-bob.glx
├── sources/
│   └── README.md
├── media/
│   └── README.md
└── README.md
```

## Family Overview

- Mary and Robert Thompson are married.
- They have two children: Alice and Robert Jr.
- Relationships demonstrate marriage and parent-child connections.

## Files

### persons/person-mother.glx
```yaml
persons:
  person-mother:
    properties:
      name:
        value: "Mary Thompson"
        fields:
          given: "Mary"
          surname: "Thompson"
      gender: female
```

### persons/person-father.glx
```yaml
persons:
  person-father:
    properties:
      name:
        value: "Robert Thompson"
        fields:
          given: "Robert"
          surname: "Thompson"
      gender: male
```

### persons/person-child-alice.glx
```yaml
persons:
  person-child-alice:
    properties:
      name:
        value: "Alice Thompson"
        fields:
          given: "Alice"
          surname: "Thompson"
      gender: female
```

### persons/person-child-bob.glx
```yaml
persons:
  person-child-bob:
    properties:
      name:
        value: "Robert Thompson"
        fields:
          given: "Robert"
          surname: "Thompson"
      gender: male
```

### relationships/rel-marriage.glx
```yaml
relationships:
  rel-marriage:
    type: marriage
    participants:
      - person: person-mother
        role: spouse
      - person: person-father
        role: spouse
```

### relationships/rel-parent-alice.glx
```yaml
relationships:
  rel-parent-alice:
    type: parent_child
    participants:
      - person: person-mother
        role: parent
      - person: person-father
        role: parent
      - person: person-child-alice
        role: child
```

### relationships/rel-parent-bob.glx
```yaml
relationships:
  rel-parent-bob:
    type: parent_child
    participants:
      - person: person-mother
        role: parent
      - person: person-father
        role: parent
      - person: person-child-bob
        role: child
```

## Validation

```bash
glx validate
# ✓ All files valid

glx check-schemas
# ✓ schemas valid
```

## What This Demonstrates

- Marriage and parent-child relationship entries
- Multiple persons with cross-referenced relationships
- Config and schema version files
- Layout ready for adding sources, media, and assertions

## Next Steps

Add supporting sources (certificates, census records) under `sources/`
and attach them to relationship or person assertion files.
