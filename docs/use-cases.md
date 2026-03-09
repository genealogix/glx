---
title: GLX Use Cases
description: Real-world applications of GENEALOGIX across diverse research domains
layout: doc
---

# GLX Use Cases

GENEALOGIX is designed to be flexible and extensible beyond traditional genealogy. Here are real-world use cases demonstrating how GLX adapts to different research domains.

## Traditional Genealogy

### Family History Research

**Scenario:** Documenting multi-generational family trees with complete source citations.

**Key Features Used:**
- Standard person, event, and relationship types
- Evidence chains from repositories to assertions
- Git version control for collaborative family research
- GEDCOM import for existing data

**Example Types:**
```yaml
# Standard vocabularies
event_types: birth, death, marriage, baptism, burial
relationship_types: parent_child, marriage, sibling
```

**Perfect For:**
- Personal family history projects
- Multi-generational family trees
- Collaborative family research teams
- Surname studies and one-name studies

---

## Biographical Research

### Academic and Professional Biographies

**Scenario:** Documenting the lives and careers of scholars, artists, or professionals with emphasis on achievements and collaborations.

**Key Features Used:**
- Custom relationship types for professional connections
- Custom event types for career milestones
- Flexible property fields for publications, awards, etc.

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  publication:
    label: "Publication"
    description: "Published work (book, article, etc.)"

  award_received:
    label: "Award Received"
    description: "Academic or professional award"

  appointment:
    label: "Appointment"
    description: "Professional appointment or position"

# vocabularies/relationship-types.glx
relationship_types:
  doctoral_advisor:
    label: "Doctoral Advisor"
    description: "PhD thesis advisor"

  collaborator:
    label: "Research Collaborator"
    description: "Co-author or research partner"

  mentor:
    label: "Mentor"
    description: "Professional or academic mentor"
```

**Perfect For:**
- Academic biography projects
- Artist biographies and career documentation
- Professional networks and collaborations
- Scientific communities and research teams

---

## Local and Community History

### Town and Community Records

**Scenario:** Documenting the people and institutions of a specific geographic community over time.

**Key Features Used:**
- Place hierarchy for community geography
- Custom event types for civic participation
- Custom relationship types for community roles
- Git collaboration for local history societies

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  town_meeting:
    label: "Town Meeting"
    description: "Participation in town meeting"

  election:
    label: "Election"
    description: "Election to public office"

  property_transaction:
    label: "Property Transaction"
    description: "Land purchase or sale"

# vocabularies/relationship-types.glx
relationship_types:
  town_selectman:
    label: "Town Selectman"
    description: "Elected town official"

  church_member:
    label: "Church Member"
    description: "Membership in religious congregation"

  business_partner:
    label: "Business Partner"
    description: "Commercial partnership"
```

**Perfect For:**
- Local history societies
- Town and county histories
- Church and religious community records
- Institutional histories

---

## Maritime History

### Naval Careers and Sea Voyages

**Scenario:** Tracking sailors, naval officers, and maritime events with emphasis on ships and voyages.

**Key Features Used:**
- Custom event types for maritime activities
- Custom properties for ship information
- Place hierarchy for ports and ocean routes
- Evidence from ship manifests and naval records

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  ship_departure:
    label: "Ship Departure"
    description: "Departure on a sea voyage"
    gedcom: "EMIG"

  port_arrival:
    label: "Port Arrival"
    description: "Arrival at a port"
    gedcom: "IMMI"

  shipwreck:
    label: "Shipwreck"
    description: "Vessel lost at sea"

  naval_commission:
    label: "Naval Commission"
    description: "Commission as naval officer"

# vocabularies/participant-roles.glx
participant_roles:
  ship_captain:
    label: "Ship Captain"
    description: "Master of a vessel"

  crew_member:
    label: "Crew Member"
    description: "Member of ship's crew"
```

**Perfect For:**
- Maritime genealogy research
- Naval history projects
- Immigration and emigration studies
- Ship and crew histories

---

## Enslaved Persons Research

### Documenting Enslaved Communities

**Scenario:** Respectfully documenting the lives of enslaved persons with sensitivity to historical trauma.

**Key Features Used:**
- Custom event types for enslavement-specific events
- Properties for ownership documentation (ethical considerations)
- Evidence chains from plantation records, sale documents, etc.
- Git version control for collaborative ethical research

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  manumission:
    label: "Manumission"
    description: "Legal grant of freedom"
    gedcom: "EVEN"

  sale:
    label: "Sale"
    description: "Record of person being sold"
    gedcom: "EVEN"

  escaped:
    label: "Escaped"
    description: "Escape from enslavement"
    gedcom: "EVEN"

# vocabularies/relationship-types.glx
relationship_types:
  enslaved_by:
    label: "Enslaved By"
    description: "Person held in slavery by another"
```

**Ethical Considerations:**
- Use respectful language in all descriptions
- Document sources with complete provenance
- Include agency and resistance when documented
- Collaborate with descendant communities

**Perfect For:**
- Descendant family research
- Historical reconciliation projects
- Academic slavery studies
- Museum and archive documentation

---

## Prosopography

### Collective Biography of Groups

**Scenario:** Systematic study of a defined group of people (e.g., members of parliament, university students, guild members).

**Key Features Used:**
- Standardized property fields across all persons
- Custom event types for group-specific milestones
- Custom relationship types for group membership
- Git version control for team-based data entry

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  matriculation:
    label: "Matriculation"
    description: "University enrollment"

  guild_admission:
    label: "Guild Admission"
    description: "Admission to professional guild"

  elected_to_parliament:
    label: "Elected to Parliament"
    description: "Election to legislative body"

# vocabularies/relationship-types.glx
relationship_types:
  guild_member:
    label: "Guild Member"
    description: "Member of professional guild"

  fellow_student:
    label: "Fellow Student"
    description: "Studied at same institution"
```

**Perfect For:**
- Parliamentary history databases
- University alumni projects
- Professional guild records
- Elite network analysis

---

## Historical Demography

### Population Studies and Census Analysis

**Scenario:** Large-scale population data analysis from census records, vital records, and parish registers.

**Key Features Used:**
- Standard event and person types
- Multi-file organization for large datasets
- Git version control for collaborative data entry
- Validation for data quality assurance

**Event Participant Properties:**
```yaml
# events/event-1851-census.glx
events:
  event-1851-census-district12:
    type: census
    date: "1851-03-30"
    place: place-leeds-yorkshire
    participants:
      - person: person-john-doe
        role: subject
        properties:
          age_at_event: "32"
          occupation: "Agricultural Labourer"
          household_number: "15"
          literacy: "can read and write"
      - person: person-mary-doe
        role: subject
        properties:
          age_at_event: "28"
          occupation: "Domestic Duties"
          household_number: "15"
```

**Perfect For:**
- Census transcription projects
- Vital records databases
- Parish register analysis
- Population migration studies

---

## Religious and Institutional Records

### Church and Monastery Records

**Scenario:** Documenting clergy, religious communities, and church membership with emphasis on institutional relationships.

**Key Features Used:**
- Custom event types for religious milestones
- Custom relationship types for religious roles
- Place hierarchy for church geography
- Evidence from church records and archives

**Custom Vocabularies:**
```yaml
# vocabularies/event-types.glx
event_types:
  ordination:
    label: "Ordination"
    description: "Ordination as clergy"

  investiture:
    label: "Investiture"
    description: "Formal installation in religious office"

  pilgrimage:
    label: "Pilgrimage"
    description: "Religious journey to holy site"

  taking_vows:
    label: "Taking Vows"
    description: "Monastic profession"

# vocabularies/relationship-types.glx
relationship_types:
  clergy:
    label: "Clergy"
    description: "Ordained minister or priest"

  parishioner:
    label: "Parishioner"
    description: "Member of parish"

  monastery_member:
    label: "Monastery Member"
    description: "Member of monastic community"
```

**Perfect For:**
- Church history projects
- Clerical biographies
- Monastery and convent records
- Religious community histories

---

## Getting Started with Your Use Case

### 1. Start with Standard Vocabularies

```bash
glx init my-research-project
# Generates standard genealogy vocabularies
```

### 2. Identify Custom Types Needed

Review your research domain and list:
- Event types not in standard vocabulary
- Relationship types specific to your domain
- Participant roles unique to your research

### 3. Extend Vocabularies Gradually

Add custom types as you encounter them:

```yaml
# vocabularies/event-types.glx
event_types:
  # Keep standard types
  birth:
    label: "Birth"
    gedcom: "BIRT"

  # Add your custom types
  your_custom_event:
    label: "Your Custom Event"
    description: "Clear description for your team"
```

### 4. Validate Regularly

```bash
glx validate
# Ensures all used types are properly defined
```

### 5. Document Your Decisions

Create a `vocabularies/README.md` explaining:
- Why each custom type was added
- When to use each type
- Examples of proper usage

---

## Need Help?

- [Best Practices Guide](guides/best-practices) - Vocabulary design guidelines
- [Core Concepts](../specification/2-core-concepts) - Repository-owned vocabularies
- [GitHub Discussions](https://github.com/genealogix/glx/discussions) - Share your use case

**Have a unique use case?** Share it with the community! GLX is designed to be infinitely extensible.
