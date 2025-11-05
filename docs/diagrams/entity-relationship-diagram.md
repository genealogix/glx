# Entity Relationship Diagram

This diagram shows how the 9 GENEALOGIX entity types connect to form a complete family history archive.

> **See Also:** For the canonical specification of entity types and their relationships, see [Entity Types](../../specification/4-entity-types/README.md#entity-relationships)

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    GENEALOGIX Entity Relationships              │
└─────────────────────────────────────────────────────────────────┘

                           ┌─────────────┐
                           │ Repository  │
                           │ (Archive)   │
                           └─────┬───────┘
                                 │ holds
                                 ▼
                           ┌─────────────┐
                           │   Source    │◄────────────┐
                           │ (Document)  │             │
                           └─────┬───────┘             │
                                 │ cites               │
                                 ▼                     │
                           ┌─────────────┐             │
                           │  Citation   │             │
                           │ (Reference) │             │
                           └─────┬───────┘             │
                                 │ supports            │
                                 ▼                     │
                           ┌─────────────┐             │
                           │ Assertion   │             │
                           │ (Claim)     │             │
                           └─────┬───────┘             │
                                 │ about               │
                                 ▼                     │
                    ┌───────────────────┐              │
                    │                   │              │
           ┌────────┤      Person       │◄─────────────┘
           │        │   (Individual)    │
           │        └─────────┬─────────┘
           │                  │ is participant in
           │                  ▼
           │        ┌───────────────────┐
           │        │                   │
           │        │      Event        │
           │        │   (Life Event)    │
           │        └─────────┬─────────┘
           │                  │ at place
           │                  ▼
           │        ┌─────────────────────┐
           │        │                     │
           └────────┤      Place          │
                    │  (Location)         │
                    └─────────────────────┘
```

## Entity Relationships

### Core Relationships

1. **Person ↔ Event**: Many-to-many through participants
   - A person can participate in multiple events (birth, marriage, death, occupation, etc.)
   - An event can have multiple participants (wedding guests, witnesses, etc.)

2. **Event → Place**: Many-to-one
   - Events occur at specific places
   - Places can host many events

3. **Person ↔ Person**: Many-to-many through relationships
   - Family connections (parent-child, marriage, etc.)
   - Social connections (witness, friend, colleague, etc.)

### Evidence Chain

4. **Repository → Source**: One-to-many
   - A repository (archive, library) holds multiple sources
   - A source is located in one repository

5. **Source → Citation**: One-to-many
   - A source contains many specific citations
   - A citation references one source

6. **Citation → Assertion**: One-to-many
   - A citation can support multiple assertions
   - An assertion is supported by one or more citations

7. **Assertion → Entity**: Many-to-many
   - Assertions make claims about persons, events, places, or relationships
   - Any entity can have multiple assertions made about it

## Cardinality Examples

### Repository Cardinality
```
Leeds Library (1 repository)
├── Parish Registers (source)
├── Census Records (source)
├── City Directories (source)
└── Newspapers (source)
```

### Person-Event Cardinality
```
John Smith (1 person)
├── Birth Event (participant)
├── Marriage Event (participant)
├── Occupation Events (participant)
├── Residence Events (participant)
└── Death Event (participant)

Marriage Event (1 event)
├── John Smith (groom)
├── Mary Brown (bride)
├── Thomas Smith (witness)
├── Sarah Jones (witness)
└── Rev. Johnson (officiant)
```

### Evidence Cardinality
```
Birth Assertion: "John born Jan 15, 1850"
├── Citation: Parish Register Entry 145
│   └── Source: St. Paul's Parish Register
│       └── Repository: Leeds Library
└── Citation: Family Bible Entry
    └── Source: Smith Family Bible
        └── Repository: Private Collection
```

## File Structure Mapping

This diagram shows entity relationships, not file organization. Files can be organized in any structure. Below is one common example structure (one-entity-per-file with dedicated directories), but many other approaches are equally valid:

```
family-archive/
├── persons/           # Person entities (example)
├── relationships/     # Relationship entities (example)
├── events/           # Event entities (example)
├── places/           # Place entities (example)
├── sources/          # Source entities (example)
├── citations/        # Citation entities (example)
├── repositories/     # Repository entities (example)
├── assertions/       # Assertion entities (example)
└── media/           # Media entities (example)
```

You could equally use a single file, hybrid structure, or any other organization that makes sense for your research.

## Implementation Notes

### Required Relationships
- Every citation must reference an existing source
- Every assertion must be supported by at least one citation
- Every event participant must reference an existing person
- Every event must reference an existing place (if place is specified)

### Optional Relationships
- Sources may reference repositories (if known)
- Places may reference parent places (for hierarchy)
- Assertions may reference media files (for visual evidence)

### Validation Rules
- All referenced IDs must exist in their respective directories
- Relationship cardinality must be valid (e.g., marriage has exactly 2 partners)
- Date ranges must be logical (birth before death, etc.)
