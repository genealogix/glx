---
title: Glossary
description: Key terms and concepts in GENEALOGIX
layout: doc
---

# GENEALOGIX Glossary

This glossary defines key terms used in the GENEALOGIX specification and documentation.

## A

### Archive
A complete GENEALOGIX repository containing family history data organized in a Git repository with standardized directory structure and validation.

### Assertion
A discrete, evidence-backed claim about a person, event, place, or relationship. Assertions separate conclusions from evidence, allowing multiple claims about the same fact with different supporting evidence.

**Example:**
```yaml
assertions/assertion-john-birth.glx:
  subject: person-john-smith
  claim: born_on
  value: "1850-01-15"
  citations: [citation-birth-cert]
  confidence: high
```

## C

### Citation
A specific reference to a location within a source document, including quality assessment and optional transcription. Citations link evidence to assertions.

**Example:**
```yaml
citations/citation-birth-entry.glx:
  source: source-parish-register
  locator: "Entry 145, page 23"
  quality: 3
  transcription: "John, son of Thomas Smith, born January 15, 1850"
```

### Confidence Level
An assessment of how certain a conclusion is based on available evidence. Common levels include: high, medium, low, disputed.

## E

### Entity
A typed record in a GENEALOGIX archive representing a person, event, place, relationship, source, citation, repository, assertion, or media file.

### Event
A life event or biographical fact such as birth, marriage, death, occupation, or residence. Events have participants, dates, places, and descriptions.

**Example:**
```yaml
events/event-john-birth.glx:
  event_type: birth
  date: "1850-01-15"
  place: place-leeds
  participants:
    - person: person-john-smith
      role: subject
```

### Evidence Chain
The complete path from physical repository through source and citation to genealogical assertion. A complete chain includes repository → source → citation → assertion.

### Evidence Hierarchy
The four dimensions used to evaluate evidence quality: primary vs secondary, direct vs indirect, original vs derivative, and information vs evidence.

## G

### GENEALOGIX (GLX)
An open standard for version-controlled family archives using Git-native workflows, human-readable YAML files, and evidence-first data modeling.

### Git Workflow
The process of using Git version control for collaborative genealogy research, including branching strategies, merge conflict resolution, and evidence integration.

## I

### ID (Identifier)
A unique identifier for each entity in the format `{type}-{8hex}` where type is the entity type (person, event, etc.) and 8hex is 8 lowercase hexadecimal characters.

**Examples:**
- `person-a1b2c3d4`
- `event-b2c3d4e5`
- `place-c3d4e5f6`

## M

### Media
Supporting files such as photos, documents, audio recordings, or videos that provide evidence or context for genealogical assertions.

## P

### Participant
A person involved in an event with a specific role such as subject, witness, officiant, parent, or spouse.

**Example:**
```yaml
participants:
  - person: person-john-smith
    role: groom
  - person: person-mary-brown
    role: bride
  - person: person-thomas-smith
    role: witness
```

### Person
An individual human being with biographical information including name, dates, places, and relationships.

**Example:**
```yaml
persons/person-john-smith.glx:
  id: person-john-smith
  name:
    given: John
    surname: Smith
    display: John Smith
  birth: "1850-01-15"
  death: "1920-03-10"
```

### Place
A geographic location with hierarchical organization, including coordinates, alternative names, and type classification.

**Example:**
```yaml
places/place-leeds.glx:
  name: Leeds
  type: city
  parent: place-yorkshire
  coordinates:
    latitude: 53.7960
    longitude: -1.5479
```

### Provenance
The complete history of how information came to be known, including source attribution, chain of custody, author identification, and research context.

## Q

### Quality Rating
A 0-3 scale indicating the reliability of evidence:
- **3**: Primary, direct evidence (birth certificates, contemporary records)
- **2**: Secondary, direct evidence (census records, indexes)
- **1**: Primary, indirect evidence (family Bibles, letters)
- **0**: Secondary, indirect evidence (published genealogies, oral history)

## R

### Relationship
A connection between people such as parent-child, marriage, adoption, or other family/social connections.

**Example:**
```yaml
relationships/rel-john-mary.glx:
  participants:
    - person: person-john-smith
      role: husband
    - person: person-mary-brown
      role: wife
  relationship_type: marriage
```

### Repository
A physical or digital archive, library, church, or institution that holds genealogical sources.

**Example:**
```yaml
repositories/repository-leeds-library.glx:
  name: Leeds Library Local Studies
  type: public_library
  address: "18 Commercial Street, Leeds LS1 6AL"
  contact: "local.studies@leeds.gov.uk"
```

### Research Notes
Documented analysis and decision-making process for genealogical conclusions, including conflicting evidence resolution and future research plans.

## S

### Schema
JSON Schema definitions that specify the structure, validation rules, and data types for each GENEALOGIX entity type.

### Source
An original document, record, publication, or material containing genealogical information.

**Example:**
```yaml
sources/source-parish-register.glx:
  title: St. Paul's Parish Register
  type: church_register
  creator: "Church of England"
  date: "1849-1855"
  repository: repository-leeds-library
```

## T

### Transcription
The text content of a source document, especially when the original is not directly accessible or when specific text is relevant to an assertion.

## V

### Validation
The process of checking GENEALOGIX files for syntax correctness, schema compliance, reference integrity, and structural consistency using the `glx validate` command.

## Y

### YAML
YAML Ain't Markup Language - the human-readable data serialization format used for all GENEALOGIX entity files. Features include indentation-based structure, support for complex data types, and comments.

**Example YAML structure:**
```yaml
# This is a comment
persons/person-john.glx:
  id: person-john-smith
  version: "1.0"
  type: person
  name:
    given: John
    surname: Smith
  birth: "1850-01-15"
  notes: |
    John was a blacksmith in Leeds.
    He worked at the ironworks on Wellington Street.
```

## File Structure Terms

### .glx Extension
The file extension used for all GENEALOGIX entity files. Pronounced "gl-ex" (genealogical exchange).

### Directory Structure
The standardized organization of GENEALOGIX files:
- `persons/` - Individual people
- `relationships/` - Family connections
- `events/` - Life events
- `places/` - Geographic locations
- `sources/` - Original materials
- `citations/` - Evidence references
- `repositories/` - Archives and libraries
- `assertions/` - Evidence-based conclusions
- `media/` - Supporting files

## Evidence Terms

### Primary Evidence
Information created at the time of the event by someone with direct knowledge (birth certificates, contemporary letters).

### Secondary Evidence
Information created later, often compiled from primary sources (census records, published indexes).

### Direct Evidence
Evidence that explicitly states the fact you're trying to prove without requiring inference.

### Indirect Evidence
Evidence that requires interpretation or additional information to support a conclusion.

### Original Evidence
First-hand, eyewitness accounts or documents created at the time of the event.

### Derivative Evidence
Copies, transcriptions, or compilations of original evidence.

## Quality Assessment Terms

### Chain of Custody
The documented path that evidence has taken from creation to current location, ensuring authenticity and reliability.

### Corroboration
Supporting evidence from multiple independent sources that agree on a conclusion.

### Preponderance of Evidence
When conflicting evidence exists, the conclusion supported by the majority of higher-quality sources.

## Git and Collaboration Terms

### Feature Branch
A Git branch used for developing new features or researching specific topics in isolation.

**Example workflow:**
```bash
git checkout -b research/1851-census
# ... research work ...
git checkout main
git merge research/1851-census
```

### Evidence Integration
The process of combining evidence from multiple sources and resolving conflicts through Git merge operations.

### Research Branch
A Git branch dedicated to investigating a specific research question or time period.

## Entity Type Terms

### Event Type
Classification of life events including birth, marriage, death, occupation, residence, education, military service, etc.

### Place Type
Classification of geographic locations including country, county, city, address, cemetery, church, etc.

### Relationship Type
Classification of connections between people including parent-child, marriage, adoption, guardianship, etc.

### Source Type
Classification of original materials including vital_record, census, church_register, newspaper, letter, etc.

## Validation Terms

### Reference Integrity
The requirement that all entity references (person IDs, place IDs, etc.) must point to existing entities.

### Schema Compliance
Conformance to JSON Schema definitions that specify valid structure and data types for each entity.

### Structural Validation
Checking that files are in correct directories, have proper extensions, and follow naming conventions.

## Research Process Terms

### Evidence Evaluation
The process of assessing source quality, analyzing content, and determining reliability for genealogical conclusions.

### Source Analysis
Examining original documents for content, context, and credibility to extract genealogical information.

### Conflicting Evidence Resolution
The process of evaluating multiple sources with different conclusions and determining which to accept based on quality and corroboration.

## Technical Terms

### ID Pattern
The regular expression pattern `{type}-{8hex}` used for all entity identifiers.

### Version Field
The schema version field (`version: "1.0"`) that indicates which specification version the file conforms to.

### Required Fields
Fields that must be present in every entity file: `id`, `version`, and `type`.

## Common Abbreviations

### GLX
Abbreviation for GENEALOGIX format files and the specification itself.

### GRO
General Register Office (UK birth, marriage, death records).

### HO107
UK Census series identifier (1841-1911 census returns).

### QUAY
GEDCOM quality indicator (0-3 scale), replaced by structured quality ratings in GENEALOGIX.

### WGS84
World Geodetic System 1984 - standard coordinate system used for geographic coordinates in GENEALOGIX.

This glossary provides a comprehensive reference for understanding GENEALOGIX terminology and concepts. Terms are organized alphabetically and include practical examples where helpful.
