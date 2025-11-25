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

> **See Also:** For archive organization details, see [Archive Organization](../../specification/3-archive-organization.md)

### Assertion
A discrete, evidence-backed claim about a person, event, place, or relationship. Assertions separate conclusions from evidence, allowing multiple claims about the same fact with different supporting evidence.

> **See Also:** For complete specification, see [Assertion Entity](../../specification/4-entity-types/assertion.md)

## C

### Chain of Custody
The documented path that evidence has taken from creation to current location, ensuring authenticity and reliability.

### Claim
A specific property or attribute being asserted about a subject in an assertion (e.g., "born_on", "occupation", "died_in").

### Citation
A specific reference to a location within a source document, including locator information and optional transcription. Citations link evidence to assertions.

> **See Also:** For complete specification, see [Citation Entity](../../specification/4-entity-types/citation.md)

### Created At
A timestamp field indicating when an entity record was originally created.

### Created By
A field indicating which researcher or system originally created an entity record.

### Conflicting Evidence Resolution
The process of evaluating multiple sources with different conclusions and determining which to accept based on quality and corroboration.

### Confidence Level
An assessment of how certain a conclusion is based on available evidence. Common levels include: high, medium, low, disputed.

> **See Also:** For controlled vocabularies, see [Standard Vocabularies](../../specification/5-standard-vocabularies/)

### Corroboration
Supporting evidence from multiple independent sources that agree on a conclusion.

## D

### Derivative Evidence
Copies, transcriptions, or compilations of original evidence.

### Direct Evidence
Evidence that explicitly states the fact you're trying to prove without requiring inference.

### Directory Structure
Common organizational pattern for GENEALOGIX files (not required). Files can be organized in any structure. Common patterns include:
- `persons/`, `relationships/`, `events/`, etc. - Dedicated directories per entity type (recommended for collaboration)
- Single file - All entities in one `.glx` file (good for small archives)
- Hybrid - Mix of directories and multi-entity files based on logical groupings

What matters: entity type ID prefixes must be correct (`person-`, `event-`, etc.)

## E

### Entity
A typed record in a GENEALOGIX archive representing a person, event, place, relationship, source, citation, repository, assertion, or media file.

### Event
A life event or biographical fact such as birth, marriage, death, occupation, or residence. Events have participants, dates, places, and descriptions.

> **See Also:** For complete specification, see [Event/Fact Entity](../../specification/4-entity-types/event.md)

### Event Type
Classification of life events including birth, marriage, death, occupation, residence, education, military service, etc.

### Evidence Chain
The complete path from physical repository through source and citation to genealogical assertion. A complete chain includes repository → source → citation → assertion.

### Evidence Evaluation
The process of assessing source quality, analyzing content, and determining reliability for genealogical conclusions.

### Evidence Hierarchy
The four dimensions used to evaluate evidence quality: primary vs secondary, direct vs indirect, original vs derivative, and information vs evidence.

### Evidence Integration
The process of combining evidence from multiple sources and resolving conflicts through Git merge operations.

## F

### Feature Branch
A Git branch used for developing new features or researching specific topics in isolation.

## G

### GENEALOGIX (GLX)
An open standard for version-controlled family archives using Git-native workflows, human-readable YAML files, and evidence-first data modeling.

### Git Workflow
The process of using Git version control for collaborative genealogy research, including branching strategies, merge conflict resolution, and evidence integration.

### .glx Extension
The file extension used for all GENEALOGIX entity files. Pronounced "gl-ex" (genealogical exchange).

### GRO
General Register Office (UK birth, marriage, death records).

## H

### HO107
UK Census series identifier (1841-1911 census returns).

## I

### ID (Identifier)
A unique identifier for each entity in the format `{type}-{8hex}` where type is the entity type (person, event, etc.) and 8hex is 8 lowercase hexadecimal characters.

**Examples:**
- `person-a1b2c3d4`
- `event-b2c3d4e5`
- `place-c3d4e5f6`

### ID Pattern
The regular expression pattern `{type}-{8hex}` used for all entity identifiers.

### Indirect Evidence
Evidence that requires interpretation or additional information to support a conclusion.

## L

### Living Status
A field indicating whether a person is alive or deceased, used for privacy and research considerations.

### Locator
A specific reference to a location within a source document, such as page number, entry number, film number, or URL.

## M

### Media
Supporting files such as photos, documents, audio recordings, or videos that provide evidence or context for genealogical assertions.

> **See Also:** For complete specification, see [Media Entity](../../specification/4-entity-types/media.md)

### MIME Type
Media type identifier (e.g., "image/jpeg", "application/pdf") that specifies the format of a media file.

## O

### Original Evidence
First-hand, eyewitness accounts or documents created at the time of the event.

## P

### Participant
A person involved in an event with a specific role such as subject, witness, officiant, parent, or spouse.

### Participant Role
The specific function or relationship a person has in an event (e.g., bride, groom, witness, officiant).

### Person
An individual human being with biographical information including name, dates, places, and relationships.

> **See Also:** For complete specification, see [Person Entity](../../specification/4-entity-types/person.md)

### Place
A geographic location with hierarchical organization, including coordinates, alternative names, and type classification.

> **See Also:** For complete specification, see [Place Entity](../../specification/4-entity-types/place.md)

### Place Type
Classification of geographic locations including country, county, city, address, cemetery, church, etc.

### Preponderance of Evidence
When conflicting evidence exists, the conclusion supported by the majority of higher-quality sources.

### Primary Evidence
Information created at the time of the event by someone with direct knowledge (birth certificates, contemporary letters).

### Primary Name
The preferred or most commonly used name for a person in genealogical records.

### Provenance
The complete history of how information came to be known, including source attribution, chain of custody, author identification, and research context.

## Q

### QUAY
GEDCOM quality indicator (0-3 scale). When importing GEDCOM files, QUAY values are preserved in citation notes for reference.

## R

### Reference Integrity
The requirement that all entity references (person IDs, place IDs, etc.) must point to existing entities.

### Relationship
A connection between people such as parent-child, marriage, adoption, or other family/social connections.

> **See Also:** For complete specification, see [Relationship Entity](../../specification/4-entity-types/relationship.md)

### Relationship Type
Classification of connections between people including parent-child, marriage, adoption, guardianship, etc.

### Repository
A physical or digital archive, library, church, or institution that holds genealogical sources.

> **See Also:** For complete specification, see [Repository Entity](../../specification/4-entity-types/repository.md)

### Required Fields
Fields that must be present in every entity file: `id`, `version`, and `type`.

### Research Branch
A Git branch dedicated to investigating a specific research question or time period.

### Research Notes
Documented analysis and decision-making process for genealogical conclusions, including conflicting evidence resolution and future research plans.

## S

### Schema
JSON Schema definitions that specify the structure, validation rules, and data types for each GENEALOGIX entity type.

### Schema Compliance
Conformance to JSON Schema definitions that specify valid structure and data types for each entity.

### Subject
The entity (person, event, place, etc.) that an assertion makes a claim about.

### Secondary Evidence
Information created later, often compiled from primary sources (census records, published indexes).

### Source
An original document, record, publication, or material containing genealogical information.

> **See Also:** For complete specification, see [Source Entity](../../specification/4-entity-types/source.md)

### Source Analysis
Examining original documents for content, context, and credibility to extract genealogical information.

### Source Type
Classification of original materials including vital_record, census, church_register, newspaper, letter, etc.

### Structural Validation
Checking that files are in correct directories, have proper extensions, and follow naming conventions.

## T

### Transcription
The text content of a source document, especially when the original is not directly accessible or when specific text is relevant to an assertion.

## V

### Validation
The process of checking GENEALOGIX files for syntax correctness, schema compliance, reference integrity, and structural consistency using the `glx validate` command.

### Value
The specific data or content of a claim in an assertion (e.g., "1850-01-15" for a birth date, "blacksmith" for an occupation).

### Vocabularies
Controlled lists of valid types and categories used throughout a GENEALOGIX archive, stored in the `vocabularies/` directory.

> **See Also:** For standard vocabularies, see [Standard Vocabularies](../../specification/5-standard-vocabularies/)

## W

### WGS84
World Geodetic System 1984 - standard coordinate system used for geographic coordinates in GENEALOGIX.

## Y

### YAML
YAML Ain't Markup Language - the human-readable data serialization format used for all GENEALOGIX entity files. Features include indentation-based structure, support for complex data types, and comments.

---

This glossary provides a comprehensive reference for understanding GENEALOGIX terminology and concepts. Terms are organized alphabetically.
