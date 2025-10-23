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
version: 1.0
concluded_identity:
  primary_name: "Mary Thompson"
  gender: female
  living: false
  
relationships:
  - relationships/rel-marriage.glx
  - relationships/rel-parent-alice.glx
  - relationships/rel-parent-bob.glx

created_at: "2024-01-20T09:00:00Z"
created_by: demo-user
```

### persons/person-father.glx
```yaml
id: person-22222222
version: 1.0
concluded_identity:
  primary_name: "Robert Thompson"
  gender: male
  living: false

relationships:
  - relationships/rel-marriage.glx
  - relationships/rel-parent-alice.glx
  - relationships/rel-parent-bob.glx

created_at: "2024-01-20T09:05:00Z"
created_by: demo-user
```

### persons/person-child-alice.glx
```yaml
id: person-33333333
version: 1.0
concluded_identity:
  primary_name: "Alice Thompson"
  gender: female
  living: true

relationships:
  - relationships/rel-parent-alice.glx

created_at: "2024-01-20T09:10:00Z"
created_by: demo-user
```

### persons/person-child-bob.glx
```yaml
id: person-44444444
version: 1.0
concluded_identity:
  primary_name: "Robert Thompson Jr."
  gender: male
  living: true

relationships:
  - relationships/rel-parent-bob.glx

created_at: "2024-01-20T09:15:00Z"
created_by: demo-user
```

### relationships/rel-marriage.glx
```yaml
id: rel-aaaa1111
version: 1.0
type: marriage
persons:
  - person-11111111
  - person-22222222

created_at: "2024-01-20T09:20:00Z"
created_by: demo-user
```

### relationships/rel-parent-alice.glx
```yaml
id: rel-bbbb2222
version: 1.0
type: parent-child
persons:
  - person-11111111
  - person-22222222
  - person-33333333

created_at: "2024-01-20T09:25:00Z"
created_by: demo-user
```

### relationships/rel-parent-bob.glx
```yaml
id: rel-cccc3333
version: 1.0
type: parent-child
persons:
  - person-11111111
  - person-22222222
  - person-44444444

created_at: "2024-01-20T09:30:00Z"
created_by: demo-user
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
