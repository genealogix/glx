---
title: Assertion Workflow Example
description: Demonstrates direct property setting vs assertion-backed properties in GLX
layout: doc
---

# Assertion Workflow Example

This example demonstrates the two approaches to recording genealogical data in GLX and when to use each.

## The Two Approaches

### Approach 1: Direct Property Setting

Set properties directly on entities without creating assertions:

```yaml
persons:
  person-alice-chen:
    properties:
      name:
        value: "Alice Chen"
        fields:
          given: "Alice"
          surname: "Chen"
      born_on: "1985-06-15"
      born_at: "place-boston"
      occupation: "software engineer"
```

**Best for:**
- Initial data entry from family records
- Quick capture (document evidence later)
- Personal research with trusted sources
- Early stages before formal research begins

**Limitations:**
- No documented evidence chain
- Hard to verify conclusions later
- Doesn't capture conflicting evidence

### Approach 2: Assertion-Backed Properties

Create assertions that document evidence for each property value:

```yaml
# First, set the property on the person
persons:
  person-robert-chen:
    properties:
      born_on: "1955-03-22"

# Then, create an assertion documenting the evidence
assertions:
  assertion-robert-birth:
    subject:
      person: person-robert-chen
    property: born_on
    value: "1955-03-22"
    citations:
      - citation-robert-birth-cert
    confidence: high
    notes: "Primary source: original birth certificate"
```

**Best for:**
- Professional genealogy research
- Documenting conflicting evidence
- Collaborative research projects
- Building verifiable research trails

## The Evidence Chain

Complete evidence documentation follows this pattern:

```
Repository → Source → Citation → Assertion → Property
    ↓          ↓          ↓          ↓           ↓
 Archives   Records   Specific   Evidence-   Concluded
            & Docs    Reference  Based Claim   Value
```

**Example chain in this archive:**

1. **Repository**: `repository-nyc-records` (NYC Department of Records)
2. **Source**: `source-nyc-birth-records` (NYC Birth Certificates)
3. **Citation**: `citation-robert-birth-cert` (specific certificate reference)
4. **Assertion**: `assertion-robert-birth` (claim that Robert was born March 22, 1955)
5. **Property**: `person-robert-chen.properties.born_on: "1955-03-22"`

## Recommended Workflow

### For Quick Data Entry

1. Create person with properties directly
2. Add a note indicating source isn't documented
3. Return later to add assertions with evidence

```yaml
persons:
  person-alice-chen:
    properties:
      born_on: "1985-06-15"
    notes: |
      Quick entry from family records.
      TODO: Add source citations when time permits.
```

### For Rigorous Research

1. Create sources and repositories first
2. Add citations referencing specific evidence
3. Create assertions linking citations to claims
4. Set properties based on assertion conclusions

### Iterative Approach (Best Practice)

Start with direct properties, then add evidence chain incrementally:

1. **Day 1**: Quick data entry with properties
2. **Week 2**: Add source for key documents
3. **Month 3**: Create citations for specific references
4. **Ongoing**: Build assertions as you research

## Properties vs Assertions: Key Differences

| Aspect | Properties Only | With Assertions |
|--------|-----------------|-----------------|
| Speed | Fast | Slower |
| Evidence | Implicit | Explicit |
| Verification | Difficult | Easy |
| Conflicts | Hidden | Documented |
| Collaboration | Limited | Excellent |
| Audit Trail | None | Complete |

## Multiple Evidence for Same Property

Assertions can corroborate each other:

```yaml
assertions:
  assertion-robert-financial-advisor:
    subject:
      person: person-robert-chen
    property: occupation
    value: "financial advisor"
    citations:
      - citation-2000-census-chen    # Census record
      - citation-linkedin-career      # LinkedIn profile
    confidence: high
    notes: "Multiple sources confirm career change in 1995"
```

## Temporal Properties with Assertions

For properties that change over time, create separate assertions for each time period:

```yaml
persons:
  person-robert-chen:
    properties:
      occupation:
        - value: "accountant"
          date: "FROM 1978 TO 1995"
        - value: "financial advisor"
          date: "FROM 1995 TO 2020"

assertions:
  assertion-robert-accountant:
    subject:
      person: person-robert-chen
    property: occupation
    value: "accountant"
    citations: [citation-1990-census-chen]

  assertion-robert-financial-advisor:
    subject:
      person: person-robert-chen
    property: occupation
    value: "financial advisor"
    citations: [citation-2000-census-chen, citation-linkedin-career]
```

## Files in This Example

- `archive.glx` - Single-file archive demonstrating both approaches
- `README.md` - This documentation

## See Also

- [Temporal Properties Example](../temporal-properties/) - Detailed temporal value patterns
- [Complete Family Example](../complete-family/) - Full multi-file archive structure
- [Core Concepts - Assertion-Aware Data Model](../../../specification/2-core-concepts#assertion-aware-data-model)
