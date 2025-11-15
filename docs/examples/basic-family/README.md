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
id: person-11111111
concluded_identity:
  primary_name: "Mary Thompson"
  gender: female
  living: false
  
relationships:
  - relationships/rel-marriage.glx
  - relationships/rel-parent-alice.glx
  - relationships/rel-parent-bob.glx
```

### persons/person-father.glx
```yaml
id: person-22222222
concluded_identity:
  primary_name: "Robert Thompson"
  gender: male
  living: false

relationships:
  - relationships/rel-marriage.glx
  - relationships/rel-parent-alice.glx
  - relationships/rel-parent-bob.glx
```

### persons/person-child-alice.glx
```yaml
id: person-33333333
concluded_identity:
  primary_name: "Alice Thompson"
  gender: female
  living: true

relationships:
  - relationships/rel-parent-alice.glx
```

### persons/person-child-bob.glx
```yaml
id: person-44444444
concluded_identity:
  primary_name: "Robert Thompson Jr."
  gender: male
  living: true

relationships:
  - relationships/rel-parent-bob.glx
```

### relationships/rel-marriage.glx
```yaml
id: rel-aaaa1111
type: marriage
persons:
  - person-11111111
  - person-22222222
```

### relationships/rel-parent-alice.glx
```yaml
id: rel-bbbb2222
type: parent-child
persons:
  - person-11111111
  - person-22222222
  - person-33333333
```

### relationships/rel-parent-bob.glx
```yaml
id: rel-cccc3333
type: parent-child
persons:
  - person-11111111
  - person-22222222
  - person-44444444
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
