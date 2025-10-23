# Complex Relationships Example

A GENEALOGIX archive demonstrating adoption, chosen family,
and complex relationship structures beyond traditional nuclear families.

## Structure

```
complex-relationships/
├── .glx-archive/
│   ├── config.glx
│   └── schema-version.glx
├── persons/
│   ├── person-biological-mother.glx
│   ├── person-adoptive-mother.glx
│   ├── person-adoptive-father.glx
│   ├── person-adopted-child.glx
│   └── person-chosen-sister.glx
├── relationships/
│   ├── rel-birth-sarah.glx
│   ├── rel-adoption-sarah.glx
│   ├── rel-marriage-maria.glx
│   └── rel-chosen-sister.glx
├── sources/
│   └── README.md
├── media/
│   └── README.md
└── README.md
```

## Family Overview

- **Sarah Johnson** (biological mother) gave birth to **Sarah Rodriguez**
- **Maria and Carlos Rodriguez** (adoptive parents) adopted **Sarah Rodriguez**
- **Sarah Rodriguez** has a chosen sister relationship with **Emma Thompson**
- **Maria and Carlos Rodriguez** are married

## Key Concepts Demonstrated

### Adoption Relationships
- Biological parent-child relationship (birth)
- Adoptive parent-child relationship (legal/social)
- Both relationships can coexist for the same person

### Chosen Family
- Non-biological family relationships
- Self-defined family connections
- Important for LGBTQ+ communities and others

### Multiple Parent Relationships
- A person can have both biological and adoptive parents
- Different relationship types serve different purposes
- Historical vs. legal vs. social relationships

## Files

### persons/person-biological-mother.glx
```yaml
id: person-bio-mom
version: "1.0"
concluded_identity:
  primary_name: "Sarah Johnson"
  gender: female
  living: false

relationships:
  - relationships/rel-birth-sarah.glx
  - relationships/rel-adoption-sarah.glx
```

### persons/person-adopted-child.glx
```yaml
id: person-adopted
version: "1.0"
concluded_identity:
  primary_name: "Sarah Rodriguez"
  gender: female
  living: true

relationships:
  - relationships/rel-birth-sarah.glx
  - relationships/rel-adoption-sarah.glx
  - relationships/rel-chosen-sister.glx
```

### relationships/rel-birth-sarah.glx
```yaml
id: rel-birth-001
version: "1.0"
type: parent-child
persons:
  - person-bio-mom
  - person-adopted
```

### relationships/rel-adoption-sarah.glx
```yaml
id: rel-adopt-001
version: "1.0"
type: parent-child
persons:
  - person-adopt-mom
  - person-adopt-dad
  - person-adopted
```

### relationships/rel-chosen-sister.glx
```yaml
id: rel-chosen-001
version: "1.0"
type: chosen-family
persons:
  - person-adopted
  - person-chosen
```

## Validation

```bash
glx validate
# ✓ All files valid

glx check-schemas
# ✓ schemas valid
```

## What This Demonstrates

- **Adoption relationships** with both biological and adoptive parents
- **Chosen family** relationships beyond blood ties
- **Multiple relationship types** for the same person
- **Complex family structures** common in modern families
- **Relationship type diversity** (parent-child, marriage, chosen-family)

## Next Steps

Add supporting documentation:
- Adoption certificates under `sources/`
- Photos of family gatherings under `media/`
- Assertions documenting the adoption process
- Evidence for chosen family relationships
