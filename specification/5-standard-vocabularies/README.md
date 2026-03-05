---
title: Standard Vocabularies
description: Standard vocabulary templates for GENEALOGIX archives
---

<script setup>
import YamlFile from '../../website/.vitepress/components/YamlFile.vue'
import { data as vocabularies } from '../../website/.vitepress/data/vocabularies.data.js'
</script>

# Standard Vocabularies

::: warning Viewing Outside the Website
This page uses VitePress components to render vocabulary content inline. If you're viewing the raw markdown on GitHub or another platform, the vocabulary content won't display. Use the **View Source** links below each vocabulary to see the raw `.glx` files directly.
:::

GENEALOGIX includes a comprehensive set of standard vocabularies that define controlled types for events, relationships, places, sources, media, and more. These vocabularies are automatically copied to new archives by `glx init` and `glx import`.

## Overview

Standard vocabularies provide:

- **Consistency** - Ensures all researchers use the same terminology
- **Validation** - The `glx validate` command checks all types exist
- **Customization** - Archives can extend with custom definitions
- **Interoperability** - Maps to GEDCOM and other formats

## Vocabulary Files

### Event Types

Defines lifecycle events (birth, death, marriage, adoption), religious events (baptism, confirmation, bar/bat mitzvah), legal events (annulment, probate, will), and migration events (immigration, emigration, naturalization).

<YamlFile
  :content="vocabularies['event-types']"
  title="vocabularies/event-types.glx"
/>

**View Source:** [event-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/event-types.glx) | **See Also:** [Event Entity Documentation](../4-entity-types/event) | [Vocabularies Specification](../4-entity-types/vocabularies#event-types-vocabulary)

---

### Relationship Types

Defines relationships between people including marriage, parent-child (biological, adoptive, foster), sibling, and other family connections.

<YamlFile
  :content="vocabularies['relationship-types']"
  title="vocabularies/relationship-types.glx"
/>

**View Source:** [relationship-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/relationship-types.glx) | **See Also:** [Relationship Entity Documentation](../4-entity-types/relationship) | [Vocabularies Specification](../4-entity-types/vocabularies#relationship-types-vocabulary)

---

### Place Types

Defines geographic and administrative place classifications from countries down to buildings.

<YamlFile
  :content="vocabularies['place-types']"
  title="vocabularies/place-types.glx"
/>

**View Source:** [place-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/place-types.glx) | **See Also:** [Place Entity Documentation](../4-entity-types/place) | [Vocabularies Specification](../4-entity-types/vocabularies#place-types-vocabulary)

---

### Source Types

Defines categories of genealogical sources including vital records, census, church registers, military records, newspapers, and more.

<YamlFile
  :content="vocabularies['source-types']"
  title="vocabularies/source-types.glx"
/>

**View Source:** [source-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/source-types.glx) | **See Also:** [Source Entity Documentation](../4-entity-types/source) | [Vocabularies Specification](../4-entity-types/vocabularies#source-types-vocabulary)

---

### Media Types

Defines categories of media objects including photographs, documents, audio recordings, and video.

<YamlFile
  :content="vocabularies['media-types']"
  title="vocabularies/media-types.glx"
/>

**View Source:** [media-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/media-types.glx) | **See Also:** [Media Entity Documentation](../4-entity-types/media) | [Vocabularies Specification](../4-entity-types/vocabularies#media-types-vocabulary)

---

### Confidence Levels

Defines confidence levels for assertions, representing researcher certainty in conclusions.

<YamlFile
  :content="vocabularies['confidence-levels']"
  title="vocabularies/confidence-levels.glx"
/>

**View Source:** [confidence-levels.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/confidence-levels.glx) | **See Also:** [Assertion Entity Documentation](../4-entity-types/assertion) | [Vocabularies Specification](../4-entity-types/vocabularies#confidence-levels-vocabulary)

---

### Participant Roles

Defines roles that people play in events and relationships (principal, witness, officiant, spouse, parent, child).

<YamlFile
  :content="vocabularies['participant-roles']"
  title="vocabularies/participant-roles.glx"
/>

**View Source:** [participant-roles.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/participant-roles.glx) | **See Also:** [Event Entity Documentation](../4-entity-types/event) | [Relationship Entity Documentation](../4-entity-types/relationship) | [Vocabularies Specification](../4-entity-types/vocabularies#participant-roles-vocabulary)

---

### Repository Types

Defines categories of institutions that hold genealogical sources (archives, libraries, churches, online databases).

<YamlFile
  :content="vocabularies['repository-types']"
  title="vocabularies/repository-types.glx"
/>

**View Source:** [repository-types.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/repository-types.glx) | **See Also:** [Repository Entity Documentation](../4-entity-types/repository) | [Vocabularies Specification](../4-entity-types/vocabularies#repository-types-vocabulary)

---

## Property Vocabularies

Property vocabularies define the custom properties available for each entity type. These enable flexible, extensible data modeling for person, event, relationship, and place entities.

### Person Properties

Defines standard and custom properties for person entities (birth date, occupation, residence, etc.). Supports temporal properties that change over time.

**View Source:** [person-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/person-properties.glx)

**Standard Properties Include:**
- `name` - Unified name property with optional structured fields (type, given, surname, prefix, suffix, etc.) (temporal)
- `gender` - Gender identity (temporal)
- `born_on` - Date of birth
- `born_at` - Place of birth (reference)
- `died_on` - Date of death
- `died_at` - Place of death (reference)
- `occupation` - Profession (temporal, GEDCOM: OCCU)
- `title` - Nobility or honorific title (temporal, GEDCOM: TITL)
- `residence` - Place of residence (temporal, reference)
- `religion` - Religious affiliation (temporal, GEDCOM: RELI)
- `education` - Educational attainment (temporal, GEDCOM: EDUC)
- `ethnicity` - Ethnic background (temporal)
- `nationality` - National citizenship (temporal, GEDCOM: NATI)
- `caste` - Caste, tribe, or social group (temporal, GEDCOM: CAST)
- `ssn` - Social Security Number (GEDCOM: SSN)
- `external_ids` - External identifiers from other systems (multi-value, GEDCOM: EXID)

**See Also:** [Person Entity Documentation](../4-entity-types/person#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#person-properties-vocabulary)

---

### Event Properties

Defines standard and custom properties for event entities.

**View Source:** [event-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/event-properties.glx)

**Standard Properties Include:**
- `age_at_event` - Age of the person at the time of the event (GEDCOM: AGE)
- `cause` - Cause of the event, e.g., cause of death (GEDCOM: CAUS)
- `event_subtype` - Further classification of the event type (GEDCOM: TYPE)
- `description` - Event description

**Note:** Event timing and location are handled by the `date` and `place` fields directly on events, not as properties. The `notes` field is a common entity field, not a property.

**See Also:** [Event Entity Documentation](../4-entity-types/event#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#event-properties-vocabulary)

---

### Relationship Properties

Defines standard and custom properties for relationship entities.

**View Source:** [relationship-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/relationship-properties.glx)

**Standard Properties Include:**
- `started_on` - When the relationship began
- `ended_on` - When the relationship ended
- `location` - Location of the relationship (reference)
- `description` - Relationship description

**See Also:** [Relationship Entity Documentation](../4-entity-types/relationship#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#relationship-properties-vocabulary)

---

### Place Properties

Defines standard and custom properties for place entities.

**View Source:** [place-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/place-properties.glx)

**Standard Properties Include:**
- `existed_from` - When the place came into existence
- `existed_to` - When the place ceased to exist
- `population` - Population count (temporal)
- `description` - Place description
- `jurisdiction` - Formal jurisdiction identifier or code
- `place_format` - Standard format string for place hierarchy
- `alternative_names` - Historical or alternate names (temporal, multi-value)

**See Also:** [Place Entity Documentation](../4-entity-types/place#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#place-properties-vocabulary)

---

### Media Properties

Defines standard and custom properties for media entities.

**View Source:** [media-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/media-properties.glx)

**Standard Properties Include:**
- `subjects` - People or entities depicted/recorded (multi-value, reference)
- `width`, `height` - Dimensions in pixels (for images/video)
- `duration` - Duration in seconds (for audio/video)
- `file_size` - File size in bytes
- `crop` - Crop coordinates for images
- `medium` - Physical medium type
- `original_filename` - Original filename when imported
- `photographer` - Person who captured the media (reference)
- `location` - Location where media was captured (reference)

**See Also:** [Media Entity Documentation](../4-entity-types/media#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#media-properties-vocabulary)

---

### Repository Properties

Defines standard and custom properties for repository entities including contact information, access details, and holdings.

**View Source:** [repository-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/repository-properties.glx)

**Standard Properties Include:**
- `phones` - Phone numbers for the repository (multi-value)
- `emails` - Email addresses for the repository (multi-value)
- `fax` - Fax number
- `access_hours` - Hours of operation or access availability
- `access_restrictions` - Any restrictions on access (appointment required, subscription, etc.)
- `holding_types` - Types of materials held (microfilm, digital, books, etc.) (multi-value)
- `external_ids` - External identifiers from other systems like FamilySearch, WikiTree (multi-value)

**See Also:** [Repository Entity Documentation](../4-entity-types/repository#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#repository-properties-vocabulary)

---

### Source Properties

Defines standard and custom properties for source entities including bibliographic metadata from GEDCOM imports.

**View Source:** [source-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/source-properties.glx)

**Standard Properties Include:**
- `abbreviation` - Short reference name or title (from GEDCOM ABBR)
- `call_number` - Repository catalog or call number (from GEDCOM CALN)
- `events_recorded` - Types of events documented (multi-value, from GEDCOM EVEN)
- `agency` - Responsible agency (from GEDCOM AGNC)
- `coverage` - Geographic or temporal scope
- `external_ids` - External identifiers (multi-value)
- `publication_info` - Publication details: publisher, place, edition (from GEDCOM PUBL)
- `url` - Web address where the source can be accessed online

**See Also:** [Source Entity Documentation](../4-entity-types/source#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#source-properties-vocabulary)

---

### Citation Properties

Defines standard and custom properties for citation entities including location and transcribed text.

**View Source:** [citation-properties.glx](https://github.com/genealogix/glx/blob/main/specification/5-standard-vocabularies/citation-properties.glx)

**Standard Properties Include:**
- `locator` - Location within source where cited material can be found (page number, film number, image number, entry reference, etc.)
- `text_from_source` - Transcription or excerpt of relevant text from the source
- `source_date` - Date when the source recorded the information
- `accessed` - Date when an online source or digital record was last accessed or retrieved

**See Also:** [Citation Entity Documentation](../4-entity-types/citation#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#citation-properties-vocabulary)

---

## Customizing Vocabularies

::: tip Complete Syntax Reference
For detailed field requirements, validation rules, and exact syntax for each vocabulary type, see the [Vocabularies Specification](../4-entity-types/vocabularies).
:::

### Adding Custom Types

Extend standard vocabularies by adding custom entries:

```yaml
# vocabularies/event-types.glx
event_types:
  # ... standard types ...

  # Custom types
  apprenticeship:
    label: "Apprenticeship"
    description: "Beginning of apprenticeship training"
    category: "occupation"
```

### Using Custom Types

Once defined, use custom types in your entities:

```yaml
# events/event-apprenticeship.glx
events:
  event-john-apprentice:
    type: apprenticeship  # Custom type from vocabulary
    date: "1845-03-10"
    place: place-leeds
    participants:
      - person: person-john
        role: subject
    notes: "Apprenticed to blacksmith"
```

### Validation

The `glx validate` command ensures all types are properly defined:

```bash
$ glx validate
✓ vocabularies/event-types.glx
✓ events/event-apprenticeship.glx
  - event type 'apprenticeship' found in vocabulary (custom)
```

## Best Practices

- **Use Standard Types First** - Standard types ensure GEDCOM compatibility and interoperability
- **Document Custom Types** - Provide clear labels and descriptions for custom types
- **Map to GEDCOM** - Include GEDCOM mappings when possible (use `_TAG` format for custom tags)
- **Keep Consistent** - Use consistent naming conventions (lowercase with underscores)

## See Also

- [Vocabularies Documentation](../4-entity-types/vocabularies) - Complete vocabulary reference
- [Core Concepts - Archive-Owned Vocabularies](../2-core-concepts#archive-owned-vocabularies)
- [Archive Organization](../3-archive-organization#vocabulary-files)
