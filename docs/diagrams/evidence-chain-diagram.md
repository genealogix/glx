# Evidence Chain Diagram

This diagram illustrates how evidence flows from physical repositories to specific genealogical claims in GENEALOGIX.

> **See Also:** For the canonical specification of evidence hierarchy, see [Core Concepts - Evidence Hierarchy](../../specification/2-core-concepts.md#evidence-hierarchy)

## Evidence Hierarchy

```
┌─────────────────────────────────────────────────────────────────┐
│                     Evidence Chain Flow                        │
└─────────────────────────────────────────────────────────────────┘

   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │ Repository  │    │   Source    │    │  Citation   │
   │ (Physical   │───▶│ (Document   │───▶│ (Reference  │
   │  Archive)   │    │  Record)    │    │  in Source) │
   └─────────────┘    └─────────────┘    └─────────────┘
         │                   │                   │
         │                   │                   │
         ▼                   ▼                   ▼
   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
   │ Institution │    │ Publication │    │ Specific    │
   │ Details     │    │ Information │    │ Location    │
   │ Contact     │    │ Author      │    │ Page/Entry  │
   │ Hours       │    │ Date        │    │ Quality     │
   └─────────────┘    └─────────────┘    └─────────────┘
```

## Evidence Chain Components

> **See Also:** For complete entity specifications, see:
> - [Repository Entity](../../specification/4-entity-types/repository.md)
> - [Source Entity](../../specification/4-entity-types/source.md)
> - [Citation Entity](../../specification/4-entity-types/citation.md)
> - [Assertion Entity](../../specification/4-entity-types/assertion.md)

### 1. Repository (Archive/Institution)
**Physical location of sources**
- Libraries, archives, churches, government offices
- Contact information, access policies, hours
- Material types held, catalog systems

**Example:**
```yaml
# repositories/repository-leeds-library.glx
id: repository-leeds-library
type: repository
name: Leeds Library Local Studies
type: public_library
address: 18 Commercial Street, Leeds LS1 6AL
contact:
  phone: "+44 113 245 3071"
  email: local.studies@leeds.gov.uk
  website: https://www.leeds.gov.uk/libraries
hours: "Mon-Fri 9:00-17:00, Sat 9:00-16:00"
materials_held:
  - parish_registers
  - census_records
  - city_directories
```

### 2. Source (Document/Record)
**Original material containing information**
- Books, registers, certificates, newspapers
- Author, publisher, publication details
- Physical or digital format

**Example:**
```yaml
# sources/source-parish-register.glx
id: source-parish-register
type: source
title: St. Paul's Parish Register of Baptisms
type: church_register
creator: Church of England, St. Paul's Parish
date: "1849-1855"
repository: repository-leeds-library
publication_info:
  publisher: St. Paul's Church
  publication_date: "1850"
  format: bound_volume
  call_number: PR/LEE/45
```

### 3. Citation (Specific Reference)
**Exact location within a source**
- Page numbers, entry numbers, timestamps
- Optional quality field (for GEDCOM compatibility)
- Transcription of relevant text

**Example:**
```yaml
# citations/citation-birth-entry.glx
id: citation-birth-entry
type: citation
source: source-parish-register
locator: "Entry 145, page 23, January 1850"
transcription: |
  "January 15th, 1850. John, son of Thomas Smith, blacksmith,
  and Mary Smith, of 23 Wellington Street. Baptized January 20th."
```

### 4. Assertion (Evidence-Based Conclusion)
**Claims about entities backed by citations**
- Subject (what entity)
- Claim (what property)
- Value (the conclusion)
- Confidence level (certainty)
- Citations (supporting evidence)

**Example:**
```yaml
# assertions/assertion-birth-date.glx
id: assertion-birth-date
type: assertion
subject: person-john-smith
claim: birth_date
value: "1850-01-15"
confidence: high  # Multiple corroborating sources
citations: [citation-birth-entry, citation-census]
```

The idiomatic GLX approach is to express certainty using assertion confidence levels rather than citation quality ratings. See [Confidence Levels Vocabulary](../../specification/5-standard-vocabularies/confidence-levels.glx).

## Evidence Flow Examples

### Example 1: Birth Certificate Chain
```
Repository: General Register Office (London)
Source: Birth Certificate Register, 1850
Citation: Certificate #BIRTH-1850-LEEDS-00145
↓
Assertion: "John Smith was born on January 15, 1850"
```

### Example 2: Census Evidence Chain
```
Repository: The National Archives (Kew)
Source: 1851 England Census, Yorkshire
Citation: Piece 2319, Folio 234, Page 23, Schedule 145
↓
Assertion: "John Smith lived at 23 Wellington Street, Leeds in 1851"
```

### Example 3: Church Record Chain
```
Repository: Leeds Library Local Studies
Source: St. Paul's Parish Register
Citation: Baptism Entry 145, January 20, 1850
↓
Assertion: "John Smith was baptized at St. Paul's Church, Leeds"
```

## Multiple Evidence Support

A single assertion can be supported by multiple evidence chains:

```yaml
# assertions/assertion-birth-date.glx
id: assertion-birth-date
type: assertion
subject: person-john-smith
claim: born_on
value: "1850-01-15"

# Multiple supporting citations
citations:
  - citation-birth-certificate
  - citation-baptism-record
  - citation-census-1851
  - citation-family-bible

confidence: high  # Multiple corroborating sources
```

## Validation Rules

### Repository Validation
- Contact information should be current
- Repository types must be valid (library, archive, church, etc.)
- URLs should be accessible (when provided)

### Source Validation
- Must reference existing repository (if repository specified)
- Publication dates should be reasonable
- Source types must be valid

### Citation Validation
- Must reference existing source
- Locator should be specific and verifiable
- Transcription should match claim when provided

### Assertion Validation
- Must reference existing entity as subject
- Must include at least one citation
- Confidence level should reflect strength of evidence

## Implementation in GENEALOGIX

The evidence chain is enforced through validation:

1. **Required References**: Citations must reference existing sources
2. **Confidence Levels**: Assertions express certainty (high, medium, low, disputed)
3. **Multiple Support**: Assertions can reference multiple citations
4. **Validation**: CLI validates entire evidence chains

This creates a complete audit trail from physical repository to genealogical conclusion.
