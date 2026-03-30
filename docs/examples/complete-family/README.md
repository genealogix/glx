---
title: Complete Family Example
description: Demonstrates GENEALOGIX entity types with evidence chains and cross-references
layout: doc
---

# Complete Family Example

This example demonstrates 8 GENEALOGIX entity types with proper cross-references, evidence chains, and descriptive entity IDs.

## Family Structure

**John Smith (person-john-smith-1850)**
- Born: January 15, 1850 in Leeds
- Died: June 20, 1920
- Occupation: Blacksmith
- Married: Mary Brown (May 10, 1875)

**Mary Brown (person-mary-brown-1852)**
- Born: March 20, 1852 in Leeds
- Died: August 15, 1930
- Occupation: Dressmaker
- Married: John Smith (May 10, 1875)

**Jane Smith (person-jane-smith-1876)**
- Born: September 5, 1876 in Leeds
- Died: December 10, 1955
- Parents: John Smith and Mary Brown
- Occupation: Dressmaker

## Entity IDs

This example uses **descriptive IDs** to make the archive more human-readable:
- `person-john-smith-1850` (person with birth year)
- `place-leeds` (descriptive place name)
- `event-birth-john` (event type + person)
- `citation-john-birth` (what it cites)

**Note:** You can use any ID format you prefer. For collaborative projects, random hex IDs (like `person-a1b2c3d4`) are recommended to avoid conflicts.

## File Organization

### Persons (3 files)
- `persons/person-john-smith.glx` - John Smith (b. 1850)
- `persons/person-mary-brown.glx` - Mary Brown (b. 1852)
- `persons/person-jane-smith.glx` - Jane Smith (b. 1876)

### Relationships (2 files)
- `relationships/rel-marriage.glx` - Marriage relationship
- `relationships/rel-parent-child.glx` - Parent-child relationships (2 relationships in one file)

### Events (2 files)
- `events/event-births.glx` - 3 birth events (all in one file)
- `events/event-marriage.glx` - 1 marriage event

### Places (3 files)
- `places/place-england.glx` - England (country)
- `places/place-yorkshire.glx` - Yorkshire (county, parent: England)
- `places/place-leeds.glx` - Leeds (city, parent: Yorkshire)

### Sources (2 files)
- `sources/source-parish-register.glx` - St. Paul's Parish Register
- `sources/source-census.glx` - 1851 Census

### Citations (3 files)
- `citations/citation-john-birth.glx` - Birth citation with transcription
- `citations/citation-marriage.glx` - Marriage citation with transcription
- `citations/citation-census.glx` - Census citation with locator

### Repositories (2 files)
- `repositories/repository-leeds-library.glx` - Leeds Library
- `repositories/repository-national-archives.glx` - The National Archives

### Assertions (3 files)
- `assertions/assertion-john-birth.glx` - Birth date with multiple citations
- `assertions/assertion-john-birthplace.glx` - Birth place
- `assertions/assertion-marriage.glx` - Marriage date

## Evidence Chain in This Example

This example demonstrates a complete evidence chain for John Smith's birth:

Repository → Source → Citation → Assertion

- `repository-leeds-library` → `source-parish-leeds` → `citation-john-birth` → `assertion-john-birth-date`

> **Learn More:** See [Core Concepts: Evidence Hierarchy](/specification/2-core-concepts#evidence-chain) for detailed explanation.

## File Format

All files use the unified GENEALOGIX format:

```yaml
# persons/person-john-smith.glx
persons:
  person-john-smith-1850:
    properties:
      name:
        value: "John Smith"
        fields:
          given: "John"
          surname: "Smith"
      gender: "male"
```

**Key Points:**
- Entity ID is the map key (`person-john-smith-1850`)
- Entity type plural at top level (`persons`)
- Files can contain multiple entities of the same type (see `events/event-births.glx`)

## Validation

```bash
cd examples/complete-family
glx validate
```

Should validate successfully with no errors.

## Key Features Demonstrated

- **8 Entity Types**: Persons, relationships, events, places, sources, citations, repositories, assertions
- **Descriptive IDs**: Human-readable entity identifiers
- **Evidence Chains**: Complete provenance from repository to conclusion
- **Confidence Levels**: Assertions express certainty
- **Hierarchical Places**: England → Yorkshire → Leeds
- **Cross-References**: All entities properly linked and validated
- **Multi-Generation Family**: Parents and children with relationships
- **Flexible Files**: Some files have one entity, some have multiple
- **Unified Format**: All files use entity type keys at top level
