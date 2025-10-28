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

Defines lifecycle events (birth, death, marriage), religious events (baptism, confirmation), and attribute facts (occupation, residence, education).

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

### Quality Ratings

Defines the 0-3 quality rating scale for citation evidence quality, compatible with GEDCOM 5.5.1 QUAY values.

<YamlFile 
  :content="vocabularies['quality-ratings']"
  title="vocabularies/quality-ratings.glx"
/>

**See Also:** [Citation Entity Documentation](../4-entity-types/citation) | [Vocabularies Specification](../4-entity-types/vocabularies#quality-ratings-vocabulary)

---

### Confidence Levels

Defines confidence levels for assertions, providing an alternative to citation quality ratings.

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

## Customizing Vocabularies

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
    custom: true
```

### Using Custom Types

Once defined, use custom types in your entities:

```yaml
# events/event-apprenticeship.glx
events:
  event-john-apprentice:
    version: "1.0"
    type: apprenticeship  # Custom type from vocabulary
    date: "1845-03-10"
    place: place-leeds
    value: "Apprenticed to blacksmith"
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
- **Mark Custom Types** - Always include `custom: true` for non-standard types
- **Map to GEDCOM** - Include GEDCOM mappings when possible (use `_TAG` format for custom tags)
- **Keep Consistent** - Use consistent naming conventions (lowercase with hyphens)

## See Also

- [Vocabularies Documentation](../4-entity-types/vocabularies) - Complete vocabulary reference
- [Core Concepts - Repository-Owned Vocabularies](../2-core-concepts#repository-owned-vocabularies)
- [Archive Organization](../3-archive-organization#vocabularies-directory)
