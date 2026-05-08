---
title: Basic Family Example
description: Foundational GENEALOGIX archive with two-parent household and basic relationships
layout: doc
---

# Basic Family Example

A foundational GENEALOGIX archive demonstrating a two-parent household
with two children and basic relationship entries.

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
├── vocabularies/           # Symlinks to standard vocabularies
└── README.md
```

## Family Overview

- Mary and Robert Thompson are married.
- They have two children: Alice and Robert Jr.
- Relationships demonstrate marriage and parent-child connections.

## Files

### persons/person-mary-thompson.glx

```yaml
persons:
  person-mary-thompson:
    properties:
      name:
        value: "Mary Thompson"
        fields:
          given: "Mary"
          surname: "Thompson"
      sex: female
```

### persons/person-robert-thompson.glx

```yaml
persons:
  person-robert-thompson:
    properties:
      name:
        value: "Robert Thompson"
        fields:
          given: "Robert"
          surname: "Thompson"
      sex: male
```

### persons/person-alice-thompson.glx

```yaml
persons:
  person-alice-thompson:
    properties:
      name:
        value: "Alice Thompson"
        fields:
          given: "Alice"
          surname: "Thompson"
      sex: female
```

### persons/person-robert-thompson-jr.glx

```yaml
persons:
  person-robert-thompson-jr:
    properties:
      name:
        value: "Robert Thompson Jr."
        fields:
          given: "Robert"
          surname: "Thompson"
          suffix: "Jr."
      sex: male
```

### relationships/rel-marriage.glx

```yaml
relationships:
  rel-marriage:
    type: marriage
    participants:
      - person: person-mary-thompson
        role: spouse
      - person: person-robert-thompson
        role: spouse
```

### relationships/rel-parent-alice.glx

```yaml
relationships:
  rel-parent-alice:
    type: parent_child
    participants:
      - person: person-mary-thompson
        role: parent
      - person: person-robert-thompson
        role: parent
      - person: person-alice-thompson
        role: child
```

### relationships/rel-parent-robert-jr.glx

```yaml
relationships:
  rel-parent-robert-jr:
    type: parent_child
    participants:
      - person: person-mary-thompson
        role: parent
      - person: person-robert-thompson
        role: parent
      - person: person-robert-thompson-jr
        role: child
```

## Validation

```bash
glx validate
# ✓ All files valid
```

## What This Demonstrates

- Marriage and parent-child relationship entries
- Multiple persons with cross-referenced relationships
- Layout ready for adding sources, media, and assertions

## Next Steps

Add supporting sources (certificates, census records) under `sources/`
and attach them to relationship or person assertion files.
