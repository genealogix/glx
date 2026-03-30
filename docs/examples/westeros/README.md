---
title: "Westeros: A Song of Ice and Fire"
description: A large-scale GLX archive covering 790+ characters from the A Song of Ice and Fire universe with full evidence chains, custom vocabularies, and complex relationships
layout: doc
---

# Westeros: A Song of Ice and Fire

A comprehensive genealogical archive of the known world from George R.R. Martin's *A Song of Ice and Fire* novels. This archive demonstrates GLX at scale — 790 persons across 70+ houses, with full evidence chains traced back to canonical sources.

::: tip Full Archive
This example is hosted as its own repository. Clone it to follow along:
```bash
git clone https://github.com/genealogix/glx-archive-westeros.git
cd glx-archive-westeros
```
:::

## Archive Statistics

| Entity Type | Count | Notes |
|-------------|-------|-------|
| Persons | 791 | Characters from 73+ houses |
| Events | 961 | Births, deaths, battles, coronations, trials, and more |
| Relationships | 822 | Marriages, parent-child, wards, sworn swords, political bonds |
| Places | 108 | Castles, cities, regions, landmarks across Westeros and Essos |
| Sources | 9 | Five novels, two companion books, HBO series, MyHeritage import |
| Citations | 247 | Chapter-level references with direct text quotes |
| Assertions | 1,808 | Source-backed conclusions with confidence levels |
| Repositories | 4 | Organized by source type |
| Vocabularies | 16 | Extensively customized for the ASOIAF domain |

## Key Features Demonstrated

### Evidence Chains at Scale

Every fact traces back to a canonical source through the full evidence chain:

```
Repository → Source → Citation → Assertion → Entity Property
```

For example, Robb Stark's death is documented as:
- **Repository**: "A Song of Ice and Fire (Novels)"
- **Source**: *A Storm of Swords*
- **Citation**: Catelyn VII chapter, with direct text quotes from the Red Wedding
- **Assertion**: `status: proven`, `confidence: high`
- **Death event**: date `"299"`, with person property `cause_of_death: "Murdered at the Red Wedding"`

### Custom Vocabularies

The archive extends standard GLX vocabularies with 200+ domain-specific types:

**Event types** (58 custom): `battle`, `coronation`, `execution`, `tournament`, `trial`, `siege`, `rebellion`, `dragon_hatching`, `guest_right_violation`, `kingsmoot`, `trial_by_combat`, and more.

**Relationship types** (40+ custom): `betrothal`, `ward`, `sworn_sword`, `liege_vassal`, `kingsguard`, `hand_of_the_king`, `master_of_whisperers`, `blood_rider`, `faceless_man`, and more.

**Participant roles** (80+ custom): `commander`, `champion`, `condemned`, `conspirator`, `dragon_rider`, `betrayer`, `presiding_judge`, and more.

**Person properties** (50+ custom): `house`, `alias`, `cause_of_death`, `dragon`, `direwolf`, `valyrian_steel`, `claim_to_throne`, `warging_ability`, and more.

### Temporal Properties

Titles, allegiances, and positions change over time:

```yaml
# From person-eddard-stark.glx
title:
  - value: "Lord of Winterfell"
    date: "FROM 280 TO 299"
  - value: "Warden of the North"
    date: "FROM 280 TO 299"
  - value: "Hand of the King"
    date: "FROM 298 TO 298"
```

### Complex Events with Multiple Participants

Events like the Red Wedding record 15+ participants with distinct roles:

```yaml
# From event-red-wedding.glx
participants:
  - person: person-robb-stark
    role: victim
  - person: person-catelyn-tully
    role: victim
  - person: person-walder-frey
    role: host
  - person: person-roose-bolton
    role: perpetrator
  # ... and more
```

### Hierarchical Place Model

Places form a geographic hierarchy from continents down to individual buildings:

```
Westeros
├── The North
│   ├── Winterfell (castle)
│   ├── Castle Black (castle)
│   └── Bear Island (island)
├── The Crownlands
│   ├── King's Landing (city)
│   │   └── The Red Keep (castle)
│   └── Dragonstone (castle)
└── ...
Essos
├── Braavos (free_city)
├── Astapor (slave_city)
└── ...
```

### Multi-Canon Support

Separate assertions handle novel vs. TV show divergences. For example, Robb Stark's wife:
- **Novel canon**: Jeyne Westerling (confidence: high, source: *A Storm of Swords*)
- **Show canon**: Talisa Maegyr (documented as a separate relationship)

### Confidence and Status Tracking

Assertions reflect the nature of ASOIAF's unreliable narration:

| Confidence | Count | Example |
|-----------|-------|---------|
| High (78%) | 1,406 | Named characters with on-page appearances |
| Medium (20%) | 365 | Characters mentioned but not directly seen |
| Low (2%) | 37 | Legendary figures, disputed identities (e.g., Young Griff's true parentage) |

## File Organization

The archive uses the standard multi-file format:

```
glx-archive-westeros/
├── persons/           # 791 files — one per character
├── events/            # 961 files — births, deaths, battles, etc.
├── relationships/     # 822 files — marriages, parent-child, political bonds
├── places/            # 108 files — castles, cities, regions
├── sources/           # 9 files — novels, companion books
├── citations/         # 247 files — chapter-level references
├── assertions/        # 1,808 files — source-backed conclusions
├── repositories/      # 4 files — source type groupings
└── vocabularies/      # 16 files — custom type definitions
```

## Using with the CLI

See the [Hands-On CLI Guide](/guides/hands-on-cli-guide) for a walkthrough of every `glx` command using this archive.

```bash
# Clone and explore
git clone https://github.com/genealogix/glx-archive-westeros.git
cd glx-archive-westeros

# Stats dashboard
glx stats

# Look up a character
glx vitals "Eddard Stark"
glx summary "Eddard Stark"

# Explore family trees
glx ancestors person-robb-stark --generations 3
glx descendants person-eddard-stark
```

## See Also

- [Hands-On CLI Guide](/guides/hands-on-cli-guide) — Step-by-step walkthrough using this archive
- [Complete Family](/examples/complete-family/) — Smaller example showing all entity types
- [Temporal Properties](/examples/temporal-properties/) — Detailed patterns for time-changing values
- [CLI Reference](/cli) — Full command documentation
