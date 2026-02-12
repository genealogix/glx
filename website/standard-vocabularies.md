---
title: Standard Vocabularies
description: Standard vocabulary templates for GENEALOGIX archives
---

<script setup>
import YamlFile from './.vitepress/components/YamlFile.vue'
import { data as vocabularies } from './.vitepress/data/vocabularies.data.js'
</script>

# Standard Vocabularies

GENEALOGIX includes a comprehensive set of standard vocabularies that define controlled types for events, relationships, places, sources, media, and more. These vocabularies are automatically copied to new archives during initialization with `glx init`.

::: tip Archive Initialization
When you run `glx init`, these standard vocabulary files are copied to your archive's `vocabularies/` directory. You can then customize them by:

- Editing descriptions and labels
- Adding additional types
- Adjusting to match your research focus
  :::

## Overview

Standard vocabularies provide:

- **Consistency** - Ensures all researchers use the same terminology
- **Validation** - The `glx validate` command checks all types exist
- **Customization** - Archives can extend with custom definitions
- **Interoperability** - Maps to GEDCOM and other formats

## Vocabulary Files

### Event Types

Defines lifecycle events (birth, death, marriage), religious events (baptism, confirmation), and attribute facts (occupation, residence, education).

<YamlFile 
  :content="vocabularies['event-types']"
  title="vocabularies/event-types.glx"
/>

**See Also:** [Event Entity Documentation](/specification/4-entity-types/event) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#event-types-vocabulary)

---

### Event Properties

Defines additional properties that can be associated with events (like certainty, privacy, historical context).

<YamlFile
  :content="vocabularies['event-properties']"
  title="vocabularies/event-properties.glx"
/>

**See Also:** [Event Entity Documentation](/specification/4-entity-types/event) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#event-properties-vocabulary)

---

### Relationship Types

Defines relationships between people including marriage, parent-child (biological, adoptive, foster), sibling, and other family connections.

<YamlFile 
  :content="vocabularies['relationship-types']"
  title="vocabularies/relationship-types.glx"
/>

**See Also:** [Relationship Entity Documentation](/specification/4-entity-types/relationship) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#relationship-types-vocabulary)

---

### Relationship Properties

Defines additional properties that can be associated with relationships (like custody, legality, adoption type).

<YamlFile
  :content="vocabularies['relationship-properties']"
  title="vocabularies/relationship-properties.glx"
/>

**See Also:** [Relationship Entity Documentation](/specification/4-entity-types/relationship) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#relationship-properties-vocabulary)

---

### Person Properties

Defines properties that can be associated with people (such as gender, physical characteristics, titles).

<YamlFile
  :content="vocabularies['person-properties']"
  title="vocabularies/person-properties.glx"
/>

**See Also:** [Person Entity Documentation](/specification/4-entity-types/person) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#person-properties-vocabulary)

---

### Place Types

Defines geographic and administrative place classifications from countries down to buildings.

<YamlFile 
  :content="vocabularies['place-types']"
  title="vocabularies/place-types.glx"
/>

**See Also:** [Place Entity Documentation](/specification/4-entity-types/place) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#place-types-vocabulary)

---

### Place Properties

Defines additional properties that can be associated with places (such as historical names, coordinates, administrative divisions).

<YamlFile
  :content="vocabularies['place-properties']"
  title="vocabularies/place-properties.glx"
/>

**See Also:** [Place Entity Documentation](/specification/4-entity-types/place) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#place-properties-vocabulary)

---

### Source Types

Defines categories of genealogical sources including vital records, census, church registers, military records, newspapers, and more.

<YamlFile 
  :content="vocabularies['source-types']"
  title="vocabularies/source-types.glx"
/>

**See Also:** [Source Entity Documentation](/specification/4-entity-types/source) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#source-types-vocabulary)

---

### Source Properties

Defines additional properties that can be associated with sources (such as abbreviation, call number, publication information).

<YamlFile
  :content="vocabularies['source-properties']"
  title="vocabularies/source-properties.glx"
/>

**See Also:** [Source Entity Documentation](/specification/4-entity-types/source) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#source-properties-vocabulary)

---

### Citation Properties

Defines properties that can be associated with citations (such as locator, text from source, source date).

<YamlFile
  :content="vocabularies['citation-properties']"
  title="vocabularies/citation-properties.glx"
/>

**See Also:** [Citation Entity Documentation](/specification/4-entity-types/citation) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#citation-properties-vocabulary)

---

### Media Types

Defines categories of media objects including photographs, documents, audio recordings, and video.

<YamlFile 
  :content="vocabularies['media-types']"
  title="vocabularies/media-types.glx"
/>

**See Also:** [Media Entity Documentation](/specification/4-entity-types/media) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#media-types-vocabulary)

---

### Media Properties

Defines additional properties that can be associated with media objects (such as dimensions, duration, subjects, crop coordinates).

<YamlFile
  :content="vocabularies['media-properties']"
  title="vocabularies/media-properties.glx"
/>

**See Also:** [Media Entity Documentation](/specification/4-entity-types/media) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#media-properties-vocabulary)

---

### Confidence Levels

Defines confidence levels for assertions, representing researcher certainty in conclusions.

<YamlFile 
  :content="vocabularies['confidence-levels']"
  title="vocabularies/confidence-levels.glx"
/>

**See Also:** [Assertion Entity Documentation](/specification/4-entity-types/assertion) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#confidence-levels-vocabulary)

---

### Participant Roles

Defines roles that people play in events and relationships (principal, witness, officiant, spouse, parent, child).

<YamlFile 
  :content="vocabularies['participant-roles']"
  title="vocabularies/participant-roles.glx"
/>

**See Also:** [Event Entity Documentation](/specification/4-entity-types/event) | [Relationship Entity Documentation](/specification/4-entity-types/relationship) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#participant-roles-vocabulary)

---

### Repository Types

Defines categories of institutions that hold genealogical sources (archives, libraries, churches, online databases).

<YamlFile 
  :content="vocabularies['repository-types']"
  title="vocabularies/repository-types.glx"
/>

**See Also:** [Repository Entity Documentation](/specification/4-entity-types/repository) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#repository-types-vocabulary)

---

### Repository Properties

Defines additional properties that can be associated with repositories (such as phone numbers, email, access hours, holdings).

<YamlFile
  :content="vocabularies['repository-properties']"
  title="vocabularies/repository-properties.glx"
/>

**See Also:** [Repository Entity Documentation](/specification/4-entity-types/repository) | [Vocabularies Specification](/specification/4-entity-types/vocabularies#repository-properties-vocabulary)

---

## Customizing Vocabularies

### Adding Custom Types

Extend standard vocabularies by adding custom entries:

```yaml
# vocabularies/event-types.glx
event_types:
  # ... standard types ...

  # Custom types
  apprenticeship:
    label: 'Apprenticeship'
    description: 'Beginning of apprenticeship training'
    category: 'occupation'
```

### Using Custom Types

Once defined, use custom types in your entities:

```yaml
# events/event-apprenticeship.glx
events:
  event-john-apprentice:
    type: apprenticeship # Custom type from vocabulary
    date: '1845-03-10'
    place: place-leeds
    value: 'Apprenticed to blacksmith'
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

1. **Use Standard Types First** - Standard types ensure GEDCOM compatibility and interoperability
2. **Document Custom Types** - Provide clear labels and descriptions for custom types
3. **Map to GEDCOM** - Include GEDCOM mappings when possible (use `_TAG` format for custom tags)
5. **Keep Consistent** - Use consistent naming conventions (lowercase with hyphens)

## See Also

- [Vocabularies Documentation](/specification/4-entity-types/vocabularies) - Complete vocabulary reference
- [Core Concepts - Archive-Owned Vocabularies](/specification/2-core-concepts#archive-owned-vocabularies)
- [Archive Organization](/specification/3-archive-organization#vocabularies-directory)
