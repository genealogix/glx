# Glossary

This glossary defines key terms used in the GENEALOGIX specification.

## A

### Approximate Date
A date with uncertainty or imprecision, expressed using keywords like ABT (about), BEF (before), AFT (after), or BET (between). Part of GENEALOGIX's fuzzy date support.

> **See Also:** [Date Format Standard](2-core-concepts#date-format-standard)

### Archive
A complete GENEALOGIX repository containing family history data organized in a Git repository with standardized directory structure and validation.

> **See Also:** [Archive Organization](3-archive-organization)

### Archive-Level Validation
Validation performed across all files in an archive, checking for duplicate entity IDs, broken cross-references, and undefined vocabulary types. Generates both errors (hard failures) and warnings (soft failures).

> **See Also:** [Validation Levels](3-archive-organization#validation-levels)

### Archive-Owned Vocabularies
Controlled vocabulary definitions stored in each archive's `vocabularies/` directory, allowing archives to customize and extend type systems while maintaining validation and consistency.

> **See Also:** [Archive-Owned Vocabularies](2-core-concepts#archive-owned-vocabularies)

### Assertion
A discrete, evidence-backed claim about a person, event, place, or relationship. Assertions separate conclusions from evidence, allowing multiple claims about the same fact with different supporting evidence.

> **See Also:** [Assertion Entity](4-entity-types/assertion)

## C

### Citation
A specific reference to a location within a source document, including locator information and optional transcription. Citations link evidence to assertions.

> **See Also:** [Citation Entity](4-entity-types/citation)

### Conflicting Evidence Resolution
The process of evaluating multiple sources with different conclusions and determining which to accept based on quality and corroboration.

### Confidence Level
An assessment of how certain a conclusion is based on available evidence. Common levels include: high, medium, low, disputed.

> **See Also:** [Confidence Levels Vocabulary](4-entity-types/vocabularies#confidence-levels-vocabulary)

### Corroboration
Supporting evidence from multiple independent sources that agree on a conclusion.

### Cross-Reference
A reference from one entity to another entity in the archive (e.g., event referencing a place, assertion referencing a citation). Cross-references are validated to ensure they point to existing entities.

> **See Also:** [Reference Integrity](#reference-integrity)

## D

### Date Keywords
Keywords used in GENEALOGIX date formats to express uncertainty or ranges: ABT (about), BEF (before), AFT (after), BET (between), FROM, TO, CAL (calculated), INT (interpreted).

> **See Also:** [Date Format Standard](2-core-concepts#date-format-standard)

### Derivative Evidence
Copies, transcriptions, or compilations of original evidence.

### Direct Evidence
Evidence that explicitly states the fact you're trying to prove without requiring inference.

### Directory Structure
Common organizational pattern for GENEALOGIX files (not required). Files can be organized in any structure. Common patterns include:
- `persons/`, `relationships/`, `events/`, etc. - Dedicated directories per entity type (recommended for collaboration)
- Single file - All entities in one `.glx` file (good for small archives)
- Hybrid - Mix of directories and multi-entity files based on logical groupings

## E

### Entity
A typed record in a GENEALOGIX archive representing a person, event, place, relationship, source, citation, repository, assertion, or media file.

> **See Also:** [Entity Types](4-entity-types/)

### Entity ID
A unique identifier for each entity, used as the YAML map key. Format: 1-64 alphanumeric characters with hyphens. Can be descriptive (john-smith-1850) or random (a1b2c3d4).

> **See Also:** [ID (Identifier)](#id-identifier), [ID Format Standards](3-archive-organization#id-format-standards)

### Event
A discrete occurrence in time and place such as birth, marriage, death, baptism, or burial. Events have participants, dates, places, and descriptions. Note: attributes like occupation and residence are represented as temporal properties on Person entities, not as events.

> **See Also:** [Event Entity](4-entity-types/event)

### Event Type
Classification of life events including birth, marriage, death, baptism, burial, military service, immigration, naturalization, etc. Note: attributes like occupation and residence are temporal properties on Person entities, not events.

> **See Also:** [Event Types Vocabulary](4-entity-types/vocabularies#event-types-vocabulary)

### Evidence Chain
The complete path from physical repository through source and citation to genealogical assertion. A complete chain includes repository → source → citation → assertion.

> **See Also:** [Evidence Chain](2-core-concepts#evidence-chain)

### Evidence Evaluation
The process of assessing source quality, analyzing content, and determining reliability for genealogical conclusions.

### Evidence Hierarchy
Classification system for evaluating evidence quality, including dimensions such as primary vs secondary, direct vs indirect, and original vs derivative evidence.

### Evidence Integration
The process of combining evidence from multiple sources and resolving conflicts through Git merge operations.

## F

### Feature Branch
A Git branch used for developing new features or researching specific topics in isolation.

### Fields
Structured components within properties that break down complex values into named parts (e.g., name property has fields: given, surname, prefix, suffix).

> **See Also:** [Structured Properties](#structured-property)

### File-Level Validation
Validation performed on individual GLX files, checking for valid YAML syntax, proper entity type keys, schema compliance, and properly formatted entity IDs.

> **See Also:** [Validation Levels](3-archive-organization#validation-levels)

### Fuzzy Date
A date with imprecision or uncertainty, such as "about 1850" or "between 1880 and 1890". GENEALOGIX supports fuzzy dates through date keywords.

> **See Also:** [Date Format Standard](2-core-concepts#date-format-standard), [Approximate Date](#approximate-date)

## G

### GENEALOGIX (GLX)
An open standard for version-controlled family archives using Git-native workflows, human-readable YAML files, and evidence-first data modeling.

### Git Workflow
The process of using Git version control for collaborative genealogy research, including branching strategies, merge conflict resolution, and evidence integration.

### .glx Extension
The file extension used for all GENEALOGIX entity files.

## H

### Hard Failure
A validation error that must be fixed before the archive is considered valid. Examples include broken entity references, duplicate IDs, and undefined vocabulary types.

> **See Also:** [Archive-Level Validation](#archive-level-validation)

## I

### ID (Identifier)
A unique identifier for each entity, used as the map key in YAML. Format: 1-64 alphanumeric characters with hyphens.

**Examples:**
- `person-a1b2c3d4` - random hex format
- `person-john-smith` - descriptive format
- `event-birth-1850`
- `place-leeds`
- `abc12345` - simple format (no prefix)

> **Note:** Examples use prefixes (e.g., `person-`) for readability. Prefixes are not required.

> **See Also:** [ID Format Standards](3-archive-organization#id-format-standards), [Entity ID](#entity-id)

### Indirect Evidence
Evidence that requires interpretation or additional information to support a conclusion.

## L

### Locator
A specific reference to a location within a source document, such as page number, entry number, film number, or URL.

## M

### Map Key
The YAML map key used to identify entities. In GENEALOGIX, entity IDs serve as map keys in the entity type collections (persons, events, etc.).

> **See Also:** [Entity ID](#entity-id)

### Media
Supporting files such as photos, documents, audio recordings, or videos that provide evidence or context for genealogical assertions.

> **See Also:** [Media Entity](4-entity-types/media)

### MIME Type
Media type identifier (e.g., "image/jpeg", "application/pdf") that specifies the format of a media file.

### Multi-File Archive
An archive organization strategy where each entity is stored in a separate file, recommended for collaborative projects due to better Git diff granularity and merge conflict resolution.

> **See Also:** [One Entity Per File](3-archive-organization#one-entity-per-file-recommended-for-collaboration)

### Multi-Value Property
A property that can have multiple values, either as a simple list or as temporal values with associated dates (e.g., multiple occupations or residences over time).

> **See Also:** [Temporal Property](#temporal-property)

## O

### Original Evidence
First-hand, eyewitness accounts or documents created at the time of the event.

## P

### Participant
A person involved in an event with a specific role such as subject, witness, officiant, parent, or spouse.

### Participant Role
The specific function or relationship a person has in an event (e.g., bride, groom, witness, officiant).

> **See Also:** [Participant Roles Vocabulary](4-entity-types/vocabularies#participant-roles-vocabulary)

### Person
An individual human being with biographical information including name, dates, places, and relationships.

> **See Also:** [Person Entity](4-entity-types/person)

### Place
A geographic location with hierarchical organization, including coordinates, alternative names, and type classification.

> **See Also:** [Place Entity](4-entity-types/place)

### Place Type
Classification of geographic locations including country, county, city, parish, cemetery, church, etc.

> **See Also:** [Place Types Vocabulary](4-entity-types/vocabularies#place-types-vocabulary)

### Primary Evidence
Information created at the time of the event by someone with direct knowledge (birth certificates, contemporary letters).

### Property
A vocabulary-defined attribute of an entity (e.g., `born_on`, `occupation`, `residence`). Properties are defined in property vocabularies and used in the `properties` field of entities.

> **See Also:** [Property Vocabularies](4-entity-types/vocabularies#property-vocabularies)

### Property Vocabulary
A special vocabulary type that defines available properties for each entity type, including property names, data types, whether they're temporal, and whether they have structured fields.

> **See Also:** [Property Vocabularies](2-core-concepts#property-vocabularies)

### Provenance
The complete history of how information came to be known, including source attribution, chain of custody, author identification, and research context.

## Q

### QUAY
GEDCOM quality indicator (0-3 scale). When importing GEDCOM files, QUAY values are preserved in citation notes for reference.

## R

### Reference Integrity
The requirement that all entity references (person IDs, place IDs, etc.) must point to existing entities.

### Reference Type
A property value type that indicates the value is an entity ID reference rather than a literal value. Reference types specify which entity collection the reference points to (persons, places, events, etc.).

> **See Also:** [Reference Types](2-core-concepts#reference-types)

### Relationship
A connection between people such as parent-child, marriage, adoption, or other family/social connections.

> **See Also:** [Relationship Entity](4-entity-types/relationship)

### Relationship Type
Classification of connections between people including parent-child, marriage, adoption, guardianship, etc.

> **See Also:** [Relationship Types Vocabulary](4-entity-types/vocabularies#relationship-types-vocabulary)

### Repository
A physical or digital archive, library, church, or institution that holds genealogical sources.

> **See Also:** [Repository Entity](4-entity-types/repository)

### Required Fields
Varies by entity type. Common required fields include `title` (sources), `name` (places, repositories), and entity-specific fields. See individual entity specifications for details.

### Research Branch
A Git branch dedicated to investigating a specific research question or time period.

### Research Notes
Documented analysis and decision-making process for genealogical conclusions, including conflicting evidence resolution and future research plans.

## S

### Schema
JSON Schema definitions that specify the structure, validation rules, and data types for each GENEALOGIX entity type.

> **See Also:** [Schema Reference](schema/)

### Schema Compliance
Conformance to JSON Schema definitions that specify valid structure and data types for each entity.

### Secondary Evidence
Information created later, often compiled from primary sources (census records, published indexes).

### Single-File Archive
An archive organization strategy where all entities are stored in a single GLX file, recommended for personal research and small family trees.

> **See Also:** [Single File Archive](3-archive-organization#single-file-archive)

### Soft Failure
A validation warning that indicates a potential issue but doesn't prevent the archive from being valid. Examples include unknown properties not defined in property vocabularies.

> **See Also:** [Archive-Level Validation](#archive-level-validation)

### Source
An original document, record, publication, or material containing genealogical information.

> **See Also:** [Source Entity](4-entity-types/source)

### Source Analysis
Examining original documents for content, context, and credibility to extract genealogical information.

### Source Type
Classification of original materials including vital_record, census, church_register, newspaper, letter, etc.

> **See Also:** [Source Types Vocabulary](4-entity-types/vocabularies#source-types-vocabulary)

### Structural Validation
Checking that files are valid YAML, have proper entity type keys at the top level, and pass JSON Schema validation.

### Structured Property
A property with structured fields that break down complex values into named components. The `value` field preserves the original form while `fields` provide structured access.

> **See Also:** [Structured Properties](2-core-concepts#structured-properties)

### Subject
In assertions, the typed reference to the entity the assertion is about. The subject field specifies both the entity type and entity ID.

> **See Also:** [Assertion Entity](4-entity-types/assertion)

## T

### Typed Reference
A reference that specifies both the entity type and entity ID, used in assertion subjects (e.g., `person: person-john-smith` rather than just `person-john-smith`).

### Temporal Property
A property that can change over time (e.g., residence, occupation, name). Temporal properties support date ranges and multiple values representing changes over a person's life.

### Transcription
The text content of a source document, especially when the original is not directly accessible or when specific text is relevant to an assertion.

## V

### Validation
The process of checking GENEALOGIX files for syntax correctness, schema compliance, reference integrity, and structural consistency using the `glx validate` command.

### Value
The specific data or content of a property in an assertion (e.g., "1850-01-15" for a birth date, "blacksmith" for an occupation).

### Value Type
The data type specification for a property value: string, date, integer, boolean, or a reference type pointing to entities.

> **See Also:** [Data Types](2-core-concepts#data-types)

### Vocabularies
Controlled lists of valid types and categories used throughout a GENEALOGIX archive, stored in the `vocabularies/` directory.

> **See Also:** [Vocabularies](4-entity-types/vocabularies), [Standard Vocabularies](5-standard-vocabularies/)

## W

### WGS84
World Geodetic System 1984 - standard coordinate system used for geographic coordinates in GENEALOGIX.

## Y

### YAML
YAML Ain't Markup Language - the human-readable data serialization format used for all GENEALOGIX entity files. Features include indentation-based structure, support for complex data types, and comments.
