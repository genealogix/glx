---
title: Vocabularies
description: Archive-owned controlled lists for types, roles, and classifications
layout: doc
---

# Vocabularies

[← Back to Entity Types](README)

## Overview

GENEALOGIX uses **archive-owned vocabularies** to define controlled lists of types, roles, and classifications used throughout the archive. Vocabulary files are ordinary `.glx` files that can live anywhere in the archive — the parser scans all `.glx` and `.yaml` files regardless of directory. By convention, the CLI places them in a `vocabularies/` directory (via `glx init` and `glx import`), but this is not a requirement.

## Benefits of Vocabularies

- **Consistency**: Ensures all researchers use the same terminology
- **Validation**: The `glx validate` command checks that all types exist in vocabularies
- **Customization**: Archives can extend standard types with custom definitions
- **Documentation**: Each type can include labels, descriptions, and metadata
- **Interoperability**: Standard types map to GEDCOM and other formats

## Vocabulary Files

The standard vocabulary files are:

- `event-types.glx`
- `relationship-types.glx`
- `place-types.glx`
- `source-types.glx`
- `media-types.glx`
- `confidence-levels.glx`
- `participant-roles.glx`
- `repository-types.glx`
- `person-properties.glx`
- `event-properties.glx`
- `relationship-properties.glx`
- `place-properties.glx`
- `media-properties.glx`
- `repository-properties.glx`
- `source-properties.glx`
- `citation-properties.glx`

When creating an archive with `glx init` or `glx import`, these files are automatically copied from the [Standard Vocabularies](../5-standard-vocabularies/) templates into a `vocabularies/` directory. You can reorganize or relocate them as you see fit — the parser discovers vocabulary definitions by their top-level keys, not by file path.

---

## Event Types Vocabulary

**Default file**: `vocabularies/event-types.glx`

**Used By**: [Event Entity](event#event-types)

**Purpose**: Defines all event and fact types used in the archive (birth, marriage, death, immigration, etc.)

**Standard Templates**: See [Standard Vocabularies - Event Types](../5-standard-vocabularies/#event-types) for the complete default vocabulary with all standard event types.

### Structure

```yaml
event_types:
  birth:
    label: "Birth"
    description: "Person's birth"
    category: "lifecycle"
    gedcom: "BIRT"
  
  marriage:
    label: "Marriage"
    description: "Marriage ceremony"
    category: "lifecycle"
    gedcom: "MARR"

  # Additional event types
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "lifecycle"
```

**Note:** Attributes like occupation, residence, religion, and nationality are represented as temporal properties on Person entities, not as events. See [Person Entity](person) for details.

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `category` | No | Category (lifecycle, religious, legal, migration, other) |
| `gedcom` | No | GEDCOM tag mapping |

### Standard Event Types

**Standard Event Types**: GENEALOGIX provides standardized event type codes including lifecycle events (birth, death, marriage, adoption), religious events (baptism, confirmation, bar/bat mitzvah), legal events (annulment, probate, will), and migration events (immigration, emigration, naturalization).

**Complete List**: See [Standard Vocabularies - Event Types](../5-standard-vocabularies/#event-types) for the complete default vocabulary file with all standard types.

### Adding Additional Event Types

Add additional event types for specialized research:

```yaml
event_types:
  # ... standard types ...

  # Additional types
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"

  land-grant:
    label: "Land Grant"
    description: "Receipt of land grant or patent"
    category: "property"
    gedcom: "_LAND"
```

---

## Relationship Types Vocabulary

**Default file**: `vocabularies/relationship-types.glx`

**Used By**: [Relationship Entity](relationship#relationship-types)

**Purpose**: Defines all relationship types between persons (marriage, parent-child, sibling, etc.)

**Standard Templates**: See [Standard Vocabularies - Relationship Types](../5-standard-vocabularies/#relationship-types) for the complete default vocabulary with all standard relationship types.

### Structure

```yaml
relationship_types:
  marriage:
    label: "Marriage"
    description: "Legal or religious union of two people"
    gedcom: "MARR"
  
  parent_child:
    label: "Parent-Child"
    description: "Biological, adoptive, or legal parent-child relationship"
    gedcom: "CHIL/FAMC"
  
  sibling:
    label: "Sibling"
    description: "Brother or sister relationship"
    gedcom: "SIBL"
  
  # Additional relationship types
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor relationship"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `gedcom` | No | GEDCOM tag mapping |

### Standard Relationship Types

**Standard Relationship Types**: GENEALOGIX provides 10 standardized relationship type codes including marriage, parent-child (plus biological, adoptive, and foster variants), sibling, step-parent, godparent, guardian, and partner relationships.

**Complete List**: See [Standard Vocabularies - Relationship Types](../5-standard-vocabularies/#relationship-types) for the complete default vocabulary file with all standard types.

### Adding Additional Relationship Types

Add additional relationship types for specialized research:

```yaml
relationship_types:
  # ... standard types ...

  # Additional types
  blood-brother:
    label: "Blood Brother"
    description: "Non-biological brotherhood bond through ceremony"

  chosen-family:
    label: "Chosen Family"
    description: "Close familial bond without biological or legal tie"
```

---

## Place Types Vocabulary

**Default file**: `vocabularies/place-types.glx`

**Used By**: [Place Entity](place#place-types)

**Purpose**: Defines geographic and administrative place classifications (country, state, city, parish, etc.)

**Standard Templates**: See [Standard Vocabularies - Place Types](../5-standard-vocabularies/#place-types) for the complete default vocabulary with all standard place types.

### Structure

```yaml
place_types:
  country:
    label: "Country"
    description: "Nation state or country"
    category: "administrative"
  
  county:
    label: "County"
    description: "County or similar administrative division"
    category: "administrative"
  
  city:
    label: "City"
    description: "City or town"
    category: "geographic"
  
  parish:
    label: "Parish"
    description: "Church parish or ecclesiastical division"
    category: "religious"
  
  # Additional place types
  plantation:
    label: "Plantation"
    description: "Agricultural estate or plantation"
    category: "geographic"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `category` | No | Category (administrative, geographic, religious, institution, other) |

### Standard Place Types

**Standard Place Types**: GENEALOGIX provides 15 standardized place type codes including administrative divisions (country, state, county, district), geographic features (city, town, locality, region, neighborhood, street, building), religious divisions (parish, church), and institutions (hospital, cemetery).

**Complete List**: See [Standard Vocabularies - Place Types](../5-standard-vocabularies/#place-types) for the complete default vocabulary file with all standard types.

### Adding Additional Place Types

Add additional place types for specialized research:

```yaml
place_types:
  # ... standard types ...

  # Additional types
  plantation:
    label: "Plantation"
    description: "Agricultural estate or plantation"
    category: "geographic"

  mission:
    label: "Mission"
    description: "Religious mission station"
    category: "religious"
```

---

## Source Types Vocabulary

**Default file**: `vocabularies/source-types.glx`

**Used By**: [Source Entity](source#source-types)

**Purpose**: Defines categories of sources (vital records, census, church registers, newspapers, etc.)

**Standard Templates**: See [Standard Vocabularies - Source Types](../5-standard-vocabularies/#source-types) for the complete default vocabulary with all standard source types.

### Structure

```yaml
source_types:
  vital_record:
    label: "Vital Record"
    description: "Birth, marriage, death certificates"
    
  census:
    label: "Census Record"
    description: "Population census enumerations"
    
  church_register:
    label: "Church Register"
    description: "Parish registers of baptisms, marriages, burials"
    
  newspaper:
    label: "Newspaper"
    description: "Newspapers, periodicals, gazettes"
  
  oral_history:
    label: "Oral History"
    description: "Interviews, recorded memories"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |

### Standard Source Types

**Standard Source Types**: GENEALOGIX provides 16 standardized source type codes including vital records, census, church registers, military records, newspapers, probate, land records, court records, immigration records, directories, books, databases, oral history, correspondence, photograph collections, and other.

**Complete List**: See [Standard Vocabularies - Source Types](../5-standard-vocabularies/#source-types) for the complete default vocabulary file with all standard types.

### Adding Additional Source Types

Add additional source types for specialized research:

```yaml
source_types:
  # ... standard types ...

  # Additional types
  oral_history:
    label: "Oral History"
    description: "Interviews, recorded memories"
```

---

## Media Types Vocabulary

**Default file**: `vocabularies/media-types.glx`

**Used By**: [Media Entity](media#media-types)

**Purpose**: Defines categories of media objects (photographs, documents, audio, video, etc.)

**Standard Templates**: See [Standard Vocabularies - Media Types](../5-standard-vocabularies/#media-types) for the complete default vocabulary with all standard media types.

### Structure

```yaml
media_types:
  photograph:
    label: "Photograph"
    description: "Photographic image"
    mime_type: "image/jpeg"
  
  document:
    label: "Document"
    description: "Scanned or digital document"
    mime_type: "application/pdf"
  
  audio:
    label: "Audio Recording"
    description: "Audio interview or recording"
    mime_type: "audio/mpeg"
  
  video:
    label: "Video Recording"
    description: "Video recording or footage"
    mime_type: "video/mp4"
  
  certificate:
    label: "Certificate"
    description: "Official certificate or license"
    mime_type: "image/tiff"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `mime_type` | No | Default MIME type for this media type |

### Standard Media Types

**Standard Media Types**: GENEALOGIX provides 7 standardized media type codes including photograph, document, audio, video, scan, image, and certificate, each with default MIME types.

**Complete List**: See [Standard Vocabularies - Media Types](../5-standard-vocabularies/#media-types) for the complete default vocabulary file with all standard types.

### Adding Additional Media Types

Add additional media types for specialized collections:

```yaml
media_types:
  # ... standard types ...

  # Additional types
  certificate:
    label: "Certificate"
    description: "Official certificate or license"
    mime_type: "image/tiff"

  artifact:
    label: "Artifact"
    description: "3D scan or photo of physical artifact"
    mime_type: "model/gltf"
```

---

## Confidence Levels Vocabulary

**Default file**: `vocabularies/confidence-levels.glx`

**Used By**: [Assertion Entity](assertion#confidence)

**Purpose**: Defines confidence levels for assertions

**Standard Templates**: See [Standard Vocabularies - Confidence Levels](../5-standard-vocabularies/#confidence-levels) for the complete default vocabulary with all standard confidence levels.

### Structure

```yaml
confidence_levels:
  high:
    label: "High Confidence"
    description: "Multiple high-quality sources agree, minimal uncertainty"
  
  medium:
    label: "Medium Confidence"
    description: "Some evidence supports conclusion, but conflicts or gaps exist"
  
  low:
    label: "Low Confidence"
    description: "Limited evidence, significant uncertainty"
  
  disputed:
    label: "Disputed"
    description: "Multiple sources conflict, resolution unclear"
  
  # Additional confidence levels
  tentative:
    label: "Tentative"
    description: "Working hypothesis pending additional research"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |

### Important Notes

- **Researcher's judgment**: Reflects overall confidence in conclusion
- **Archive-defined**: Each archive can customize the meaning of confidence levels

See [Assertion Entity - Confidence](assertion#confidence) for usage details.

---

## Repository Types Vocabulary

**Default file**: `vocabularies/repository-types.glx`

**Used By**: [Repository Entity](repository#repository-types)

**Purpose**: Defines categories of repositories (archives, libraries, churches, online databases, etc.)

**Standard Templates**: See [Standard Vocabularies - Repository Types](../5-standard-vocabularies/#repository-types) for the complete default vocabulary with all standard repository types.

### Structure

```yaml
repository_types:
  archive:
    label: "Archive"
    description: "Government or historical archive"
  
  library:
    label: "Library"
    description: "Public, university, or specialty library"
  
  church:
    label: "Church"
    description: "Church or religious organization archives"
  
  database:
    label: "Online Database"
    description: "Online genealogical database service"
  
  museum:
    label: "Museum"
    description: "Museum with genealogical collections"
  
  # Additional repository types
  historical_society:
    label: "Historical Society"
    description: "Local historical society"
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |

### Standard Repository Types

See [Repository Entity](repository#repository-types) for the complete list of standard repository types.

---

## Participant Roles Vocabulary

**Default file**: `vocabularies/participant-roles.glx`

**Used By**: [Event Entity](event#participant-roles), [Relationship Entity](relationship#participant-roles)

**Purpose**: Defines roles that people play in events and relationships (principal, witness, officiant, etc.)

**Standard Templates**: See [Standard Vocabularies - Participant Roles](../5-standard-vocabularies/#participant-roles) for the complete default vocabulary with all standard participant roles.

### Structure

```yaml
participant_roles:
  # Event roles
  principal:
    label: "Principal"
    description: "Primary person in the event"
    applies_to:
      - event
  
  witness:
    label: "Witness"
    description: "Person who witnessed the event"
    applies_to:
      - event
  
  officiant:
    label: "Officiant"
    description: "Person who officiated the ceremony"
    applies_to:
      - event
  
  # Relationship roles
  spouse:
    label: "Spouse"
    description: "Marriage partner"
    applies_to:
      - relationship
  
  parent:
    label: "Parent"
    description: "Parent in parent-child relationship"
    applies_to:
      - relationship
  
  child:
    label: "Child"
    description: "Child in parent-child relationship"
    applies_to:
      - relationship
  
  # Additional roles
  godparent:
    label: "Godparent"
    description: "Spiritual sponsor at baptism"
    applies_to:
      - event
      - relationship
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label |
| `description` | No | Detailed description |
| `applies_to` | No | Array of entity types (event, relationship) |

### Standard Participant Roles

Common event roles:
- `principal` - Primary person in the event
- `subject` - Subject of the event (alias for principal)
- `groom`, `bride` - Marriage participants
- `witness` - Event witness
- `officiant` - Ceremony officiant
- `informant` - Person providing information

Common relationship roles:
- `spouse` - Marriage partner
- `parent` - Parent in parent-child relationship
- `child` - Child in parent-child relationship
- `adoptive_parent`, `adopted_child` - Adoption roles
- `sibling` - Brother or sister

---

## Property Vocabularies

Property vocabularies define the custom properties available for each entity type. These properties represent "concluded" or "accepted" values and support flexible, extensible data modeling beyond the standard entity fields.

### Overview

Property vocabularies enable archives to:

- Define custom properties for any entity type
- Control property value types (string, date, integer, boolean)
- Validate entity references for reference-type properties
- Support temporal properties that change over time
- Extend GENEALOGIX for domain-specific needs

### Files

Property vocabulary files are included in the `vocabularies/` directory:

```
vocabularies/
├── person-properties.glx
├── event-properties.glx
├── relationship-properties.glx
├── place-properties.glx
├── media-properties.glx
├── repository-properties.glx
├── source-properties.glx
└── citation-properties.glx
```

### Person Properties Vocabulary

**Default file**: `vocabularies/person-properties.glx`

**Used By**: [Person Entity](person#properties)

**Purpose**: Defines properties that can be set on person entities (birth date, occupation, residence, etc.)

### Standard Properties

GENEALOGIX provides standard person properties:

| Property | Type | Temporal | GEDCOM | Description |
|----------|------|----------|--------|-------------|
| `name` | string (with fields) | Yes | | Person's name as recorded, with optional structured fields (given, surname, prefix, suffix, etc.) |
| `gender` | string | Yes | | Gender identity |
| `born_on` | date | No | | Date of birth |
| `born_at` | places | No | | Place of birth |
| `died_on` | date | No | | Date of death |
| `died_at` | places | No | | Place of death |
| `occupation` | string | Yes | OCCU | Profession or trade |
| `title` | string | Yes | TITL | Nobility or honorific title |
| `residence` | places | Yes | | Place of residence |
| `religion` | string | Yes | RELI | Religious affiliation |
| `education` | string | Yes | EDUC | Educational attainment |
| `ethnicity` | string | Yes | | Ethnic background |
| `nationality` | string | Yes | NATI | National citizenship |
| `caste` | string | Yes | CAST | Caste, tribe, or social group |
| `ssn` | string | No | SSN | Social Security Number |
| `external_ids` | string (multi) | No | EXID | External identifiers from other systems |

### Event Properties Vocabulary

**Default file**: `vocabularies/event-properties.glx`

**Used By**: [Event Entity](event#properties)

**Purpose**: Defines properties that can be set on event entities

Event properties are generally less common than person properties, since most event data is structural (type, date, place, participants). Standard properties include:

- `age_at_event` - Age of the person at the time of the event (GEDCOM: AGE)
- `cause` - Cause of the event, e.g., cause of death (GEDCOM: CAUS)
- `event_subtype` - Further classification of the event type (GEDCOM: TYPE)
- `description` - Event description

**Note:** Event timing and location are handled by the `date` and `place` fields directly on the event, not as properties. The `notes` field is a standard entity field available on all entity types, not a property.

### Relationship Properties Vocabulary

**Default file**: `vocabularies/relationship-properties.glx`

**Used By**: [Relationship Entity](relationship#properties)

**Purpose**: Defines properties that can be set on relationship entities

Standard properties include:

- `started_on` - When the relationship began
- `ended_on` - When the relationship ended
- `location` - Location of the relationship
- `description` - Relationship description

### Place Properties Vocabulary

**Default file**: `vocabularies/place-properties.glx`

**Used By**: [Place Entity](place#properties)

**Purpose**: Defines properties that can be set on place entities

Standard properties include:

- `existed_from` - When the place came into existence
- `existed_to` - When the place ceased to exist
- `population` - Population count (temporal)
- `description` - Place description
- `jurisdiction` - Formal jurisdiction identifier or code
- `place_format` - Standard format string for place hierarchy
- `alternative_names` - Historical or alternate names for a place (temporal, multi-value)

### Media Properties Vocabulary

**Default file**: `vocabularies/media-properties.glx`

**Used By**: [Media Entity](media#properties)

**Purpose**: Defines properties that can be set on media entities

Standard properties include:

- `subjects` - People or entities depicted/recorded
- `width` - Width in pixels (for images/video)
- `height` - Height in pixels (for images/video)
- `duration` - Duration in seconds (for audio/video)
- `file_size` - File size in bytes
- `crop` - Crop coordinates for images
- `medium` - Physical medium type
- `original_filename` - Original filename when imported
- `photographer` - Person who captured the media
- `location` - Location where media was captured

### Repository Properties Vocabulary

**Default file**: `vocabularies/repository-properties.glx`

**Used By**: [Repository Entity](repository#properties)

**Purpose**: Defines properties that can be set on repository entities for contact information, access details, and holdings

Standard properties include:

- `phones` - Phone number(s) for the repository
- `emails` - Email address(es) for the repository
- `fax` - Fax number
- `access_hours` - Hours of operation/access
- `access_restrictions` - Any restrictions on access (appointment required, subscription, etc.)
- `holding_types` - Types of materials held (microfilm, digital, books, etc.)
- `external_ids` - External identifiers from other systems (FamilySearch, WikiTree, etc.)

### Source Properties Vocabulary

**Default file**: `vocabularies/source-properties.glx`

**Used By**: [Source Entity](source#properties)

**Purpose**: Defines properties that can be set on source entities for bibliographic metadata

Standard properties include:

- `abbreviation` - Short reference name or title for the source (from GEDCOM ABBR)
- `call_number` - Repository catalog or call number (from GEDCOM CALN)
- `events_recorded` - Types of events this source documents (from GEDCOM EVEN)
- `agency` - Agency responsible for creating/maintaining this source (from GEDCOM AGNC)
- `coverage` - Geographic or temporal scope of source content
- `external_ids` - External identifiers from other systems
- `publication_info` - Publication details: publisher, place, edition (from GEDCOM PUBL)
- `url` - Web address where the source can be accessed online (e.g., FamilySearch, Ancestry)

### Citation Properties Vocabulary

**Default file**: `vocabularies/citation-properties.glx`

**Used By**: [Citation Entity](citation#properties)

**Purpose**: Defines properties that can be set on citation entities for locator and transcription details

Standard properties include:

- `locator` - Location within source where cited material can be found (page number, film number, image number, entry reference, etc.; from GEDCOM PAGE)
- `text_from_source` - Transcription or excerpt of relevant text from the source (from GEDCOM TEXT)
- `source_date` - Date when the source recorded the information (from GEDCOM DATE)
- `accessed` - Date when an online source or digital record was last accessed or retrieved

### Property Definition Structure

Each property in a property vocabulary is defined with the following fields:

```yaml
person_properties:
  born_on:
    label: "Birth Date"
    description: "Date of birth"
    value_type: date
    temporal: false

  occupation:
    label: "Occupation"
    description: "Profession or trade"
    value_type: string
    temporal: true
    gedcom: "OCCU"

  residence:
    label: "Residence"
    description: "Place where person lived"
    reference_type: places
    temporal: true
```

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label for the property |
| `description` | No | Detailed description of the property |
| `value_type` | No* | Data type: `string`, `date`, `integer`, or `boolean` |
| `reference_type` | No* | Entity type for references: `persons`, `places`, `events`, `relationships`, `sources`, `citations`, `repositories`, `media` |
| `temporal` | No | Whether property can change over time (default: false) |
| `multi_value` | No | Whether property can have multiple values as an array (default: false) |
| `gedcom` | No | Corresponding GEDCOM tag for import/export mapping (e.g., `OCCU`, `PAGE`) |
| `fields` | No | Sub-schema for structured property components (see below) |

***Exactly one of `value_type` or `reference_type` must be specified** (there is no implicit default)

### Multi-Value Properties

Some properties naturally have multiple values. For example, a repository may have several phone numbers, or a media item may depict multiple people. The `multi_value` attribute indicates that a property accepts an array of values.

#### Defining Multi-Value Properties

```yaml
repository_properties:
  phones:
    label: "Phone Numbers"
    description: "Telephone numbers for the repository"
    value_type: string
    multi_value: true

  emails:
    label: "Email Addresses"
    description: "Email addresses for the repository"
    value_type: string
    multi_value: true

media_properties:
  subjects:
    label: "Subjects"
    description: "People depicted in the media"
    reference_type: persons
    multi_value: true
```

#### Using Multi-Value Properties in Data

When a property has `multi_value: true`, provide values as a YAML array:

```yaml
repositories:
  repository-family-history-library:
    name: "Family History Library"
    properties:
      phones:
        - "+1 801-240-2584"
        - "+1 801-240-2331"
      emails:
        - "fhl@familysearch.org"
        - "research@familysearch.org"
      holding_types:
        - "microfilm"
        - "digital"
        - "books"
```

For reference properties with `multi_value: true`:

```yaml
media:
  media-family-photo:
    uri: "media/files/family-reunion-1985.jpg"
    properties:
      subjects:
        - person-john
        - person-mary
        - person-sarah
```

#### Multi-Value with Temporal Properties

A property can be both `multi_value: true` and `temporal: true`. In this case, each temporal entry contains an array:

```yaml
person_properties:
  nicknames:
    label: "Nicknames"
    description: "Informal names used for the person"
    value_type: string
    multi_value: true
    temporal: true
```

```yaml
persons:
  person-john:
    properties:
      nicknames:
        - value:
            - "Johnny"
            - "Jack"
          date: "FROM 1950 TO 1970"
        - value:
            - "Big John"
            - "J.D."
          date: "FROM 1970"
```

#### Validation Behavior

- **Single value for multi_value property**: Validators should accept a single value (not in array) and treat it as a one-element array
- **Array for non-multi_value property**: Validators should issue an error if an array is provided for a property without `multi_value: true`

### Structured Properties with Fields

Some properties benefit from having structured sub-components. For example, a person's name can be stored as a simple string but may also include parsed components like given name, surname, prefix, etc. The `fields` attribute allows you to define this structured breakdown in the vocabulary.

#### Defining Fields

```yaml
person_properties:
  name:
    label: "Name"
    description: "Person's name as recorded, with optional structured breakdown"
    value_type: string
    temporal: true
    fields:
      prefix:
        label: "Prefix"
        description: "Honorific prefix (Dr., Rev., Hon.)"
      given:
        label: "Given Name"
        description: "Given/first name(s)"
      nickname:
        label: "Nickname"
        description: "Familiar or descriptive name"
      surname_prefix:
        label: "Surname Prefix"
        description: "Article or prefix (von, van, de)"
      surname:
        label: "Surname"
        description: "Family name"
      suffix:
        label: "Suffix"
        description: "Generational suffix (Jr., Sr., III)"
```

#### Field Definition Structure

Each field in the `fields` map is defined with:

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Human-readable label for the field |
| `description` | No | Detailed description of the field |

#### Using Structured Properties in Data

When a property has `fields` defined, the property value can be either:

1. **Simple value** - Just a string:
   ```yaml
   properties:
     name: "John Smith"
   ```

2. **Structured value** - Object with `value` and optional `fields`:
   ```yaml
   properties:
     name:
       value: "John Smith"
       fields:
         given: "John"
         surname: "Smith"
   ```

3. **Temporal list** - For properties with `temporal: true`:
   ```yaml
   properties:
     name:
       - value: "Mary Johnson"
         date: "1850"
         fields:
           given: "Mary"
           surname: "Johnson"
       - value: "Mary Smith"
         date: "FROM 1875"
         fields:
           given: "Mary"
           surname: "Smith"
   ```

#### When to Use Fields

Use `fields` when:

- A property has well-known components (name → given, surname, etc.)
- You want to preserve both the original recorded value and parsed components
- Different sources may record different components
- You need to support searching or sorting by component

The `value` field should always contain the complete value as recorded, while `fields` provides the optional parsed breakdown. This design:

- Preserves the original source data
- Allows flexible parsing (not all sources have all components)
- Supports temporal changes (name changes over time)
- Enables rich querying and display

#### Field Validation

When validating structured properties with `fields`, the following rules apply:

**All Fields Are Optional**

Fields defined in a vocabulary are never required. You can provide any subset of the defined fields:

```yaml
properties:
  name:
    value: "Dr. John Smith Jr."
    fields:
      given: "John"
      surname: "Smith"
      # prefix and suffix omitted - this is valid
```

**Unknown Fields Generate Warnings**

If a property value includes fields not defined in the vocabulary, validators should issue a warning (not an error). This allows archives to capture additional data while encouraging consistency:

```yaml
properties:
  name:
    value: "John Smith"
    fields:
      given: "John"
      surname: "Smith"
      clan: "MacLeod"  # WARNING: unknown field if not defined in vocabulary
```

**The `value` Field**

When a property has a natural single-value representation, include `value` alongside `fields` to preserve the original recorded form:

```yaml
# Value + fields: preserves original while providing structure
properties:
  name:
    value: "John Smith"
    fields:
      given: "John"
      surname: "Smith"
```

When there is no natural single-value representation, fields-only is valid:

```yaml
# Fields only: appropriate for structured data like coordinates
properties:
  crop:
    fields:
      top: 450
      left: 100
      width: 800
      height: 200
```

#### Custom Structured Properties

You can define `fields` for any custom property:

```yaml
person_properties:
  # Custom structured property for address
  mailing_address:
    label: "Mailing Address"
    description: "Postal address with structured components"
    value_type: string
    temporal: true
    fields:
      street:
        label: "Street Address"
        description: "House number and street name"
      city:
        label: "City"
        description: "City or town"
      state:
        label: "State/Province"
        description: "State, province, or region"
      postal_code:
        label: "Postal Code"
        description: "ZIP or postal code"
      country:
        label: "Country"
        description: "Country name"
```

Usage:
```yaml
properties:
  mailing_address:
    value: "123 Main St, Springfield, IL 62701, USA"
    fields:
      street: "123 Main St"
      city: "Springfield"
      state: "IL"
      postal_code: "62701"
      country: "USA"
```

### Temporal Properties

Properties marked with `temporal: true` support capturing how values change over time:

```yaml
properties:
  occupation:
    - value: "blacksmith"
      date: "1880"
    - value: "farmer"
      date: "FROM 1885 TO 1920"
  residence:
    - value: "place-leeds"
      date: "1900"
    - value: "place-london"
      date: "FROM 1920 TO 1950"
```

See [Core Concepts - Data Types - Temporal Properties](../2-core-concepts#temporal-properties) for complete documentation.

### Adding Additional Properties

Add additional properties for archive-specific needs:

```yaml
person_properties:
  # Standard properties...

  # Additional properties
  militia_service:
    label: "Militia Service"
    description: "Service in local militia"
    value_type: string
    temporal: true

  land_holdings:
    label: "Land Holdings"
    description: "Land owned by person"
    reference_type: places
    temporal: true
```

### Context-Aware Validation

Assertions use property vocabularies for context-aware property validation:

```yaml
assertions:
  assertion-john-birth:
    subject:
      person: person-john
    property: born_on  # Validated against person_properties
    value: "1850-01-15"
    citations: [citation-birth]
    confidence: high
```

The validator:
1. Determines the subject's entity type from the typed reference (person, event, relationship, or place)
2. Looks up the appropriate property vocabulary for that type
3. Validates the `property` against the vocabulary
4. **Emits warnings for unknown properties** (allows flexibility for emerging properties)
5. Validates the value according to the property's `value_type` or `reference_type`
6. **Emits errors for broken references** when a property is defined with `reference_type` but the referenced entity doesn't exist

---

## Vocabulary Validation

The `glx validate` command performs comprehensive validation with different severity levels:

### Validation Errors (Hard Failures)

The following issues cause validation to fail:

1. **Missing vocabulary types**: All types used in entities must be defined in vocabularies
   - Event types (`event_types`)
   - Relationship types (`relationship_types`)
   - Place types (`place_types`)
   - Source types (`source_types`)
   - Repository types (`repository_types`)
   - Media types (`media_types`)
   - Participant roles (`participant_roles`)
   - Confidence levels (`confidence_levels`)

2. **Broken entity references**: All entity references must point to existing entities
   - Person references
   - Event references
   - Place references
   - Source references
   - Citation references
   - Repository references
   - Media references
   - Relationship references

3. **Broken property references**: Properties defined with `reference_type` must reference existing entities
   ```yaml
   # Error: place-nonexistent doesn't exist
   persons:
     person-john:
       properties:
         born_at: place-nonexistent  # ERROR if born_at has reference_type: places
   ```

4. **Structural validation**: Files must follow proper YAML/JSON structure and schema

### Validation Warnings (Soft Failures)

The following issues generate warnings but don't fail validation:

1. **Unknown property definitions**: Properties used but not defined in property vocabularies
   ```yaml
   # Warning: custom_field not in person_properties vocabulary
   persons:
     person-john:
       properties:
         custom_field: "some value"  # WARNING: unknown property
   ```

2. **Unknown assertion properties**: Properties used but not defined in property vocabularies
   ```yaml
   # Warning: custom_property not in person_properties vocabulary
   assertions:
     assertion-custom:
       subject:
         person: person-john
       property: custom_property  # WARNING: unknown property
       value: "some value"
   ```

Warnings allow flexibility for emerging properties and rapid data entry while still notifying researchers of potential issues.

### Example Validation Output

```bash
$ glx validate

✓ vocabularies/event-types.glx
✓ vocabularies/relationship-types.glx
✓ events/event-birth.glx
  - event type 'birth' found in vocabulary
✓ relationships/rel-marriage.glx
  - relationship type 'marriage' found in vocabulary

⚠ persons/person-john.glx
  - WARNING: property 'custom_field' not defined in person_properties vocabulary

❌ events/event-custom.glx
  - ERROR: event type 'unknown-type' not found in vocabularies/event-types.glx
  - ERROR: place reference 'place-nonexistent' not found
```

---

## Creating Additional Types

### Step 1: Add to Vocabulary File

```yaml
event_types:
  # ... standard types ...

  # Additional types
  land-grant:
    label: "Land Grant"
    description: "Receipt of land grant or patent"
    category: "property"
    gedcom: "_LAND"  # Non-standard GEDCOM tag
```

### Step 2: Use in Entity

```yaml
events:
  event-john-land-grant:
    type: land-grant  # Custom type from vocabulary
    date: "1850-03-10"
    place: place-indiana
    participants:
      - person: person-john
        role: subject
    notes: "160 acres in Howard County"
```

### Step 3: Validate

```bash
$ glx validate events/event-land-grant.glx
✓ events/event-land-grant.glx
  - event type 'land-grant' found in vocabulary
```

---

## Best Practices

### Use Standard Types First

Before creating custom types, check if standard types meet your needs. Standard types:
- Ensure GEDCOM compatibility
- Work with most genealogy software
- Are understood by other researchers

### Document Additional Types

When adding additional types:
- Provide clear `label` and `description`
- Document why the additional type is needed
- Include GEDCOM mapping if possible (use `_TAG` format for custom GEDCOM tags)

### Keep Vocabularies Consistent

- Use consistent naming conventions (lowercase with underscores)
- Group related types together
- Add comments to explain complex types
- Version vocabulary files alongside schema updates

### Share Vocabularies

- Additional vocabularies can be shared between archives
- Archives working on similar research can standardize additional types
- Consider submitting useful types as proposals for standard types

---

## Vocabulary Extensibility

Vocabularies support various extension mechanisms:

### Additional Fields

Add custom fields to vocabulary entries:

```yaml
event_types:
  baptism:
    label: "Baptism"
    description: "Religious baptism ceremony"
    category: "religious"
    gedcom: "BAPM"
    # Custom fields
    icon: "water-drop"
    color: "#4A90E2"
    requires_place: true
    requires_participants: true
```

### Localization

Add translations for labels:

```yaml
place_types:
  county:
    label: "County"
    label_es: "Condado"
    label_fr: "Comté"
    label_de: "Landkreis"
    description: "County or similar administrative division"
```

---

## Schema Reference

Each vocabulary type has a corresponding JSON Schema for validation:

| Vocabulary | Schema File |
|------------|-------------|
| Event Types | [event-types.schema.json](../schema/v1/vocabularies/event-types.schema.json) |
| Relationship Types | [relationship-types.schema.json](../schema/v1/vocabularies/relationship-types.schema.json) |
| Place Types | [place-types.schema.json](../schema/v1/vocabularies/place-types.schema.json) |
| Source Types | [source-types.schema.json](../schema/v1/vocabularies/source-types.schema.json) |
| Media Types | [media-types.schema.json](../schema/v1/vocabularies/media-types.schema.json) |
| Participant Roles | [participant-roles.schema.json](../schema/v1/vocabularies/participant-roles.schema.json) |
| Repository Types | [repository-types.schema.json](../schema/v1/vocabularies/repository-types.schema.json) |
| Confidence Levels | [confidence-levels.schema.json](../schema/v1/vocabularies/confidence-levels.schema.json) |
| Person Properties | [person-properties.schema.json](../schema/v1/vocabularies/person-properties.schema.json) |
| Event Properties | [event-properties.schema.json](../schema/v1/vocabularies/event-properties.schema.json) |
| Relationship Properties | [relationship-properties.schema.json](../schema/v1/vocabularies/relationship-properties.schema.json) |
| Place Properties | [place-properties.schema.json](../schema/v1/vocabularies/place-properties.schema.json) |
| Media Properties | [media-properties.schema.json](../schema/v1/vocabularies/media-properties.schema.json) |
| Repository Properties | [repository-properties.schema.json](../schema/v1/vocabularies/repository-properties.schema.json) |
| Source Properties | [source-properties.schema.json](../schema/v1/vocabularies/source-properties.schema.json) |
| Citation Properties | [citation-properties.schema.json](../schema/v1/vocabularies/citation-properties.schema.json) |

All vocabulary schemas are located in `specification/schema/v1/vocabularies/` and define:
- Required top-level key (e.g., `event_types`, `relationship_types`)
- Required fields for each entry (typically `label`)
- Optional fields (e.g., `description`, `gedcom`)
- Pattern properties for vocabulary keys (alphanumeric with hyphens, 1-64 characters)

Vocabulary files are validated by the `glx validate` command using these schemas.

## See Also

- **[Standard Vocabularies](../5-standard-vocabularies/)** - Complete default vocabulary files with all standard types
- [Core Concepts - Archive-Owned Vocabularies](../2-core-concepts#archive-owned-vocabularies)
- [Archive Organization](../3-archive-organization) - Where vocabulary files are stored
- [Event Entity](event) - Event types vocabulary
- [Relationship Entity](relationship) - Relationship types vocabulary
- [Place Entity](place) - Place types vocabulary
- [Source Entity](source) - Source types vocabulary
- [Media Entity](media) - Media types vocabulary
- [Citation Entity](citation) - Citation documentation

---

