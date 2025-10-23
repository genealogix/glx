# Evidence Chain Diagram

This diagram illustrates how evidence flows from physical repositories to specific genealogical claims in GENEALOGIX.

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

### 1. Repository (Archive/Institution)
**Physical location of sources**
- Libraries, archives, churches, government offices
- Contact information, access policies, hours
- Material types held, catalog systems

**Example:**
```yaml
# repositories/repository-leeds-library.glx
id: repository-leeds-library
version: "1.0"
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
version: "1.0"
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
- Quality assessment (0-3 scale)
- Transcription of relevant text

**Example:**
```yaml
# citations/citation-birth-entry.glx
id: citation-birth-entry
version: "1.0"
type: citation
source: source-parish-register
locator: "Entry 145, page 23, January 1850"
quality: 3  # Primary source, direct evidence
transcription: |
  "January 15th, 1850. John, son of Thomas Smith, blacksmith,
  and Mary Smith, of 23 Wellington Street. Baptized January 20th."
```

## Evidence Quality Scale

GENEALOGIX uses a 0-3 quality scale compatible with GEDCOM 5.5.1 QUAY:

| Quality | Description | Example |
|---------|-------------|---------|
| **3** | Direct and primary evidence | Original birth certificate, contemporary baptism record |
| **2** | Secondary evidence, officially recorded | Census record, published vital records |
| **1** | Questionable reliability | Undocumented oral history, conflicting sources |
| **0** | Unreliable or estimated | Unverified family tradition, unsourced trees |

**Note:** This scale maintains 1:1 compatibility with GEDCOM for seamless data exchange.

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
version: "1.0"
type: assertion
subject: person-john-smith
claim: born_on
value: "1850-01-15"

# Multiple supporting citations
citations:
  - citation-birth-certificate    # Quality 3 - birth certificate
  - citation-baptism-record       # Quality 3 - church record
  - citation-census-1851          # Quality 2 - census record
  - citation-family-bible         # Quality 1 - family record

confidence: high  # Multiple primary sources confirm
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
- Quality rating must be 0-3
- Transcription should match claim when provided

## Evidence Best Practices

### 1. Primary vs Secondary Sources
- **Primary**: Created at the time of the event (certificates, contemporary records)
- **Secondary**: Created later (censuses, compiled records, histories)

### 2. Direct vs Indirect Evidence
- **Direct**: Explicitly states the fact (birth certificate shows birth date)
- **Indirect**: Requires inference (census shows age, implying birth year)

### 3. Original vs Derivative
- **Original**: First-hand account
- **Derivative**: Copy or compilation of original

### 4. Information vs Evidence
- **Information**: Data from a source
- **Evidence**: Information + source analysis + correlation

## Implementation in GENEALOGIX

The evidence chain is enforced through:

1. **Required References**: Citations must reference existing sources
2. **Quality Tracking**: Each citation includes quality assessment
3. **Multiple Support**: Assertions can reference multiple citations
4. **Validation**: CLI validates entire evidence chains

This creates a complete audit trail from physical repository to genealogical conclusion.
