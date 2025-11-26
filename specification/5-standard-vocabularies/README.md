---
title: Standard Vocabularies
description: Standard vocabulary templates for GENEALOGIX archives
---

<script setup>
import YamlFile from '../../website/.vitepress/components/YamlFile.vue'
import { data as vocabularies } from '../../website/.vitepress/data/vocabularies.data.js'
</script>

# Standard Vocabularies

GENEALOGIX includes a comprehensive set of standard vocabularies that define controlled types for events, relationships, places, sources, media, and more. These vocabularies are automatically copied to new archives during initialization with `glx init`.

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

**See Also:** [Event Entity Documentation](../4-entity-types/event) | [Vocabularies Specification](../4-entity-types/vocabularies#event-types-vocabulary)

---

### Relationship Types

Defines relationships between people including marriage, parent-child, sibling, adoption, and other family connections.

<YamlFile 
  :content="vocabularies['relationship-types']"
  title="vocabularies/relationship-types.glx"
/>

**See Also:** [Relationship Entity Documentation](../4-entity-types/relationship) | [Vocabularies Specification](../4-entity-types/vocabularies#relationship-types-vocabulary)

---

### Place Types

Defines geographic and administrative place classifications from countries down to buildings.

<YamlFile 
  :content="vocabularies['place-types']"
  title="vocabularies/place-types.glx"
/>

**See Also:** [Place Entity Documentation](../4-entity-types/place) | [Vocabularies Specification](../4-entity-types/vocabularies#place-types-vocabulary)

---

### Source Types

Defines categories of genealogical sources including vital records, census, church registers, military records, newspapers, and more.

<YamlFile 
  :content="vocabularies['source-types']"
  title="vocabularies/source-types.glx"
/>

**See Also:** [Source Entity Documentation](../4-entity-types/source) | [Vocabularies Specification](../4-entity-types/vocabularies#source-types-vocabulary)

---

### Media Types

Defines categories of media objects including photographs, documents, audio recordings, and video.

<YamlFile 
  :content="vocabularies['media-types']"
  title="vocabularies/media-types.glx"
/>

**See Also:** [Media Entity Documentation](../4-entity-types/media) | [Vocabularies Specification](../4-entity-types/vocabularies#media-types-vocabulary)

---

### Confidence Levels

Defines confidence levels for assertions, representing researcher certainty in conclusions.

<YamlFile 
  :content="vocabularies['confidence-levels']"
  title="vocabularies/confidence-levels.glx"
/>

**See Also:** [Assertion Entity Documentation](../4-entity-types/assertion) | [Vocabularies Specification](../4-entity-types/vocabularies#confidence-levels-vocabulary)

---

### Participant Roles

Defines roles that people play in events and relationships (principal, witness, officiant, spouse, parent, child).

<YamlFile 
  :content="vocabularies['participant-roles']"
  title="vocabularies/participant-roles.glx"
/>

**See Also:** [Event Entity Documentation](../4-entity-types/event) | [Relationship Entity Documentation](../4-entity-types/relationship) | [Vocabularies Specification](../4-entity-types/vocabularies#participant-roles-vocabulary)

---

### Repository Types

Defines categories of institutions that hold genealogical sources (archives, libraries, churches, online databases).

<YamlFile 
  :content="vocabularies['repository-types']"
  title="vocabularies/repository-types.glx"
/>

**See Also:** [Repository Entity Documentation](../4-entity-types/repository) | [Vocabularies Specification](../4-entity-types/vocabularies#repository-types-vocabulary)

---

## Property Vocabularies

Property vocabularies define the custom properties available for each entity type. These enable flexible, extensible data modeling for person, event, relationship, and place entities.

### Person Properties

Defines standard and custom properties for person entities (birth date, occupation, residence, etc.). Supports temporal properties that change over time.

**File:** `vocabularies/person-properties.glx`

**Standard Properties Include:**
- `name` - Unified name property with optional structured fields (given, surname, prefix, suffix, etc.) (temporal)
- `gender` - Gender identity (temporal)
- `born_on` - Date of birth
- `born_at` - Place of birth (reference)
- `died_on` - Date of death
- `died_at` - Place of death (reference)
- `occupation` - Profession (temporal, reference or string)
- `residence` - Place of residence (temporal, reference)
- `religion`, `education`, `ethnicity`, `nationality` - Additional biographical attributes

**See Also:** [Person Entity Documentation](../4-entity-types/person#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#person-properties-vocabulary)

---

### Event Properties

Defines standard and custom properties for event entities.

**File:** `vocabularies/event-properties.glx`

**Standard Properties Include:**
- `description` - Event description
- `notes` - Additional notes

**Note:** Event timing and location are handled by the `date` and `place` fields directly on events, not as properties.

**See Also:** [Event Entity Documentation](../4-entity-types/event#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#event-properties-vocabulary)

---

### Relationship Properties

Defines standard and custom properties for relationship entities.

**File:** `vocabularies/relationship-properties.glx`

**Standard Properties Include:**
- `started_on` - When the relationship began
- `ended_on` - When the relationship ended
- `location` - Location of the relationship (reference)
- `description` - Relationship description

**See Also:** [Relationship Entity Documentation](../4-entity-types/relationship#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#relationship-properties-vocabulary)

---

### Place Properties

Defines standard and custom properties for place entities.

**File:** `vocabularies/place-properties.glx`

**Standard Properties Include:**
- `existed_from` - When the place came into existence
- `existed_to` - When the place ceased to exist
- `population` - Population count (temporal)
- `description` - Place description

**See Also:** [Place Entity Documentation](../4-entity-types/place#properties) | [Vocabularies Specification](../4-entity-types/vocabularies#place-properties-vocabulary)

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
    description: "Apprenticed to blacksmith"
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
- **Keep Consistent** - Use consistent naming conventions (lowercase with hyphens)

## See Also

- [Vocabularies Documentation](../4-entity-types/vocabularies) - Complete vocabulary reference
- [Core Concepts - Repository-Owned Vocabularies](../2-core-concepts#repository-owned-vocabularies)
- [Archive Organization](../3-archive-organization#vocabularies-directory)
