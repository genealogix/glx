# GEDCOM Import - Required Schema Additions

**Date**: 2025-11-18
**Status**: Proposed schema changes for GEDCOM import support

---

## Executive Summary

During GEDCOM import implementation, we discovered that several GLX entity types are **missing Properties fields**, which prevents us from properly storing GEDCOM data that doesn't map to specific GLX fields. This document proposes adding Properties fields to 5 entity types to enable full GEDCOM support.

---

## Current Properties Field Coverage

### Entity Types WITH Properties Field ✅

1. **Person** - Has `Properties map[string]interface{}`
   - Used for: gender, occupation, religion, education, nationality, caste, etc.
   - GEDCOM usage: Stores EXID (external IDs) properly

2. **Relationship** - Has `Properties map[string]interface{}`
   - Used for: Custom relationship attributes
   - GEDCOM usage: Marriage type, citations, etc.

3. **Event** - Has `Properties map[string]interface{}`
   - Used for: Age at event, cause, agency, etc.
   - GEDCOM usage: Event-specific attributes

4. **Place** - Has `Properties map[string]interface{}`
   - Used for: Place-specific attributes
   - GEDCOM usage: Ready for future GEDCOM place properties

### Entity Types MISSING Properties Field ❌

5. **Source** - MISSING
   - Current fields: Title, Type, Authors, Date, Description, RepositoryID, PublicationInfo, Notes, Media, Tags
   - **Problem**: Can't store GEDCOM ABBR (abbreviation), EXID (external IDs), custom SOURCE tags
   - **Workaround**: Currently storing EXIDs in Notes field with text format

6. **Citation** - MISSING
   - Current fields: SourceID, Page, TextFromSource, Transcription, Quality, Locator, RepositoryID, Media, Notes, Tags
   - **Problem**: Can't store GEDCOM EVEN (event type cited), ROLE (role in event), DATA.DATE (entry date)
   - **Workaround**: Not currently extracted from GEDCOM

7. **Repository** - MISSING
   - Current fields: Name, Type, Address, City, State, PostalCode, Country, Phone, Email, Website, Notes, Tags
   - **Problem**: Can't store GEDCOM FAX, multiple phones/emails, EXID (external IDs)
   - **Workaround**: Currently storing additional phones/emails and EXIDs in Notes field

8. **Media** - MISSING
   - Current fields: URI, MimeType, Hash, Title, Notes, Tags
   - **Problem**: Can't store GEDCOM CROP (image crop coordinates), multiple TITLs (GEDCOM 7.0), EXID
   - **Workaround**: Currently storing CROP coords in Notes field as text

9. **Assertion** - MISSING
   - Current fields: Subject, Claim, Value, Participant, Confidence, Sources, Citations, Notes, Tags
   - **Problem**: Limited ability to store assertion metadata
   - **Impact**: Low (assertions are derived, not source data)

---

## Proposed Schema Changes

### Add Properties Field to 5 Entity Types

```go
// Source represents a source of information.
type Source struct {
	Title           string                 `yaml:"title"`
	Type            string                 `yaml:"type,omitempty" refType:"source_types"`
	Authors         []string               `yaml:"authors,omitempty"`
	Date            string                 `yaml:"date,omitempty"`
	Description     string                 `yaml:"description,omitempty"`
	RepositoryID    string                 `yaml:"repository,omitempty" refType:"repositories"`
	PublicationInfo string                 `yaml:"publication_info,omitempty"`
	Properties      map[string]interface{} `yaml:"properties,omitempty"` // NEW
	Notes           string                 `yaml:"notes,omitempty"`
	Media           []string               `yaml:"media,omitempty" refType:"media"`
	Tags            []string               `yaml:"tags,omitempty"`
}

// Citation represents a citation of a source.
type Citation struct {
	SourceID       string                 `yaml:"source,omitempty" refType:"sources"`
	Page           string                 `yaml:"page,omitempty"`
	TextFromSource string                 `yaml:"text_from_source,omitempty"`
	Transcription  string                 `yaml:"transcription,omitempty"`
	Quality        *int                   `yaml:"quality,omitempty" refType:"quality_ratings"`
	Locator        interface{}            `yaml:"locator,omitempty"`
	RepositoryID   string                 `yaml:"repository,omitempty" refType:"repositories"`
	Properties     map[string]interface{} `yaml:"properties,omitempty"` // NEW
	Media          []string               `yaml:"media,omitempty" refType:"media"`
	Notes          string                 `yaml:"notes,omitempty"`
	Tags           []string               `yaml:"tags,omitempty"`
}

// Repository represents a repository where sources are held.
type Repository struct {
	Name       string                 `yaml:"name"`
	Type       string                 `yaml:"type,omitempty" refType:"repository_types"`
	Address    string                 `yaml:"address,omitempty"`
	City       string                 `yaml:"city,omitempty"`
	State      string                 `yaml:"state_province,omitempty"`
	PostalCode string                 `yaml:"postal_code,omitempty"`
	Country    string                 `yaml:"country,omitempty"`
	Phone      string                 `yaml:"phone,omitempty"`
	Email      string                 `yaml:"email,omitempty"`
	Website    string                 `yaml:"website,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"` // NEW
	Notes      string                 `yaml:"notes,omitempty"`
	Tags       []string               `yaml:"tags,omitempty"`
}

// Media struct
type Media struct {
	URI        string                 `yaml:"uri"`
	MimeType   string                 `yaml:"mime_type,omitempty"`
	Hash       string                 `yaml:"hash,omitempty"`
	Title      string                 `yaml:"title,omitempty"`
	Properties map[string]interface{} `yaml:"properties,omitempty"` // NEW
	Notes      string                 `yaml:"notes,omitempty"`
	Tags       []string               `yaml:"tags,omitempty"`
}

// Assertion represents a conclusion made by a researcher.
type Assertion struct {
	Subject     string                 `yaml:"subject" refType:"persons,events,relationships,places"`
	Claim       string                 `yaml:"claim,omitempty"`
	Value       string                 `yaml:"value,omitempty"`
	Participant *AssertionParticipant  `yaml:"participant,omitempty"`
	Confidence  string                 `yaml:"confidence,omitempty" refType:"confidence_levels"`
	Sources     []string               `yaml:"sources,omitempty" refType:"sources"`
	Citations   []string               `yaml:"citations,omitempty" refType:"citations"`
	Properties  map[string]interface{} `yaml:"properties,omitempty"` // NEW
	Notes       string                 `yaml:"notes,omitempty"`
	Tags        []string               `yaml:"tags,omitempty"`
}
```

---

## Impact Analysis

### Benefits

1. **Proper GEDCOM Storage**
   - External IDs (EXID) stored as `properties.external_ids` arrays
   - Custom GEDCOM tags preserved in properties
   - GEDCOM 7.0 features properly stored

2. **Consistency**
   - All major entity types have Properties fields
   - Uniform approach to vocabulary-defined properties
   - Better alignment with GLX philosophy

3. **Future-Proofing**
   - Room for custom properties without schema changes
   - Vocabulary definitions can add new properties
   - Compatible with extensions

4. **Clean Code**
   - No more hacky workarounds storing structured data in Notes
   - Type-safe access via properties vocabulary
   - Clear separation of freeform notes vs. structured data

### Breaking Changes

**None** - Adding optional fields is backward compatible:
- Existing GLX files without Properties will validate fine
- New imports will use Properties where appropriate
- Old importers will ignore Properties (YAML omitempty)

### Migration Path

1. **Phase 1** - Add Properties fields to schema (this PR)
2. **Phase 2** - Update GEDCOM importers to use Properties
3. **Phase 3** - Add property vocabularies for GEDCOM-specific properties
4. **Phase 4** - Update validation to check property vocabularies

No migration of existing archives needed.

---

## GEDCOM Properties to Add

Once Properties fields are in place, we should add these to properties vocabularies:

### Source Properties

```yaml
# vocabularies/source-properties.glx
properties:
  abbreviation:
    label: "Abbreviation"
    description: "Short form of source title (GEDCOM ABBR)"
    type: "string"

  external_ids:
    label: "External Identifiers"
    description: "Links to external databases (GEDCOM 7.0 EXID)"
    type: "array"
    item_type: "external_id"

  record_id:
    label: "Record ID"
    description: "Automated record ID (GEDCOM RIN)"
    type: "string"
```

### Repository Properties

```yaml
# vocabularies/repository-properties.glx
properties:
  fax:
    label: "Fax Number"
    description: "Repository fax number (GEDCOM FAX)"
    type: "string"

  additional_phones:
    label: "Additional Phone Numbers"
    description: "Secondary phone numbers"
    type: "array"
    item_type: "string"

  additional_emails:
    label: "Additional Email Addresses"
    description: "Secondary email addresses"
    type: "array"
    item_type: "string"

  external_ids:
    label: "External Identifiers"
    description: "Links to external databases (GEDCOM 7.0 EXID)"
    type: "array"
    item_type: "external_id"
```

### Citation Properties

```yaml
# vocabularies/citation-properties.glx
properties:
  event_type_cited:
    label: "Event Type Cited"
    description: "Type of event this citation supports (GEDCOM SOUR.EVEN)"
    type: "string"
    ref_type: "event_types"

  role_in_event:
    label: "Role in Event"
    description: "Person's role in cited event (GEDCOM SOUR.ROLE)"
    type: "string"
    ref_type: "participant_roles"

  entry_recording_date:
    label: "Entry Recording Date"
    description: "When citation was recorded (GEDCOM SOUR.DATA.DATE)"
    type: "date"
```

### Media Properties

```yaml
# vocabularies/media-properties.glx
properties:
  crop:
    label: "Crop Coordinates"
    description: "Image crop region (GEDCOM 7.0 CROP)"
    type: "object"
    fields:
      top: "number"
      left: "number"
      height: "number"
      width: "number"

  alternative_titles:
    label: "Alternative Titles"
    description: "Additional titles (GEDCOM 7.0 multiple TITL)"
    type: "array"
    item_type: "string"

  external_ids:
    label: "External Identifiers"
    description: "Links to external databases (GEDCOM 7.0 EXID)"
    type: "array"
    item_type: "external_id"
```

### External ID Type

```yaml
# vocabularies/types/external-id.glx
external_id:
  type: "object"
  fields:
    id:
      type: "string"
      required: true
      description: "The external identifier value"
    type:
      type: "string"
      description: "Type of external system (wikitree, familysearch, findagrave, etc.)"
    url:
      type: "string"
      description: "Direct URL to external record (optional)"
```

---

## Recommendation

**Approve and implement immediately** - This is a critical schema gap that affects:
1. GEDCOM import quality
2. Schema consistency
3. Future extensibility

The changes are backward compatible and will significantly improve GEDCOM import fidelity.

---

## Implementation Checklist

- [ ] Add Properties field to Source type
- [ ] Add Properties field to Citation type
- [ ] Add Properties field to Repository type
- [ ] Add Properties field to Media type
- [ ] Add Properties field to Assertion type
- [ ] Update GEDCOM importers to use Properties instead of Notes
- [ ] Create source-properties vocabulary
- [ ] Create repository-properties vocabulary
- [ ] Create citation-properties vocabulary
- [ ] Create media-properties vocabulary
- [ ] Create external-id type definition
- [ ] Update validation to check property vocabularies
- [ ] Update documentation
- [ ] Update website examples
- [ ] Add tests for property validation

---

## Next Steps

1. Get approval for schema changes
2. Implement Properties fields in lib/types.go
3. Update GEDCOM importers to use Properties
4. Create property vocabularies
5. Update documentation and examples
