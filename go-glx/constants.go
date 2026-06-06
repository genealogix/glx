// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package glx

// Standard Event Types - from event-types.glx vocabulary
const (
	EventTypeBirth              = "birth"
	EventTypeDeath              = "death"
	EventTypeMarriage           = "marriage"
	EventTypeDivorce            = "divorce"
	EventTypeBaptism            = "baptism"
	EventTypeBurial             = "burial"
	EventTypeCremation          = "cremation"
	EventTypeEngagement         = "engagement"
	EventTypeAnnulment          = "annulment"
	EventTypeMarriageBanns      = "marriage_banns"
	EventTypeMarriageContract   = "marriage_contract"
	EventTypeMarriageLicense    = "marriage_license"
	EventTypeMarriageSettlement = "marriage_settlement"
	EventTypeDivorceFiled       = "divorce_filed"
	EventTypeImmigration        = "immigration"
	EventTypeEmigration         = "emigration"
	EventTypeNaturalization     = "naturalization"
	EventTypeResidence          = "residence"
	EventTypeOccupation         = "occupation"
	EventTypeEducation          = "education"
	EventTypeGraduation         = "graduation"
	EventTypeRetirement         = "retirement"
	EventTypeChristening        = "christening"
	EventTypeAdultChristening   = "adult_christening"
	EventTypeAdoption           = "adoption"
	EventTypeBarMitzvah         = "bar_mitzvah"
	EventTypeBatMitzvah         = "bat_mitzvah"
	EventTypeBlessing           = "blessing"
	EventTypeConfirmation       = "confirmation"
	EventTypeFirstCommunion     = "first_communion"
	EventTypeOrdination         = "ordination"
	EventTypeProbate            = "probate"
	EventTypeWill               = "will"
	EventTypeLegalSeparation    = "legal_separation"
	EventTypeTaxation           = "taxation"
	EventTypeVoterRegistration  = "voter_registration"
	EventTypeCensus             = "census"
	EventTypeGeneric            = "event"
)

// Standard Relationship Types - from relationship-types.glx vocabulary
const (
	RelationshipTypeMarriage              = "marriage"
	RelationshipTypeParentChild           = "parent_child"
	RelationshipTypeBiologicalParentChild = "biological_parent_child"
	RelationshipTypeAdoptiveParentChild   = "adoptive_parent_child"
	RelationshipTypeFosterParentChild     = "foster_parent_child"
	RelationshipTypeSibling               = "sibling"
	RelationshipTypeStepParent            = "step_parent"
	RelationshipTypeGodparent             = "godparent"
	RelationshipTypeGuardian              = "guardian"
	RelationshipTypePartner               = "partner"
	RelationshipTypeNeighbor              = "neighbor"
	RelationshipTypeCoworker              = "coworker"
	RelationshipTypeHousemate             = "housemate"
	RelationshipTypeApprenticeship        = "apprenticeship"
	RelationshipTypeEmployment            = "employment"
	RelationshipTypeEnslavement           = "enslavement"
	RelationshipTypePossiblySamePerson    = "possibly_same_person"
	RelationshipTypeRelative              = "relative"
	RelationshipTypeAssociate             = "associate"
	RelationshipTypeBoarder               = "boarder"
)

// Standard Participant Roles - from participant-roles.glx vocabulary
const (
	ParticipantRolePrincipal      = "principal"
	ParticipantRoleSubject        = "subject"
	ParticipantRoleWitness        = "witness"
	ParticipantRoleOfficiant      = "officiant"
	ParticipantRoleInformant      = "informant"
	ParticipantRoleGroom          = "groom"
	ParticipantRoleBride          = "bride"
	ParticipantRoleSpouse         = "spouse"
	ParticipantRoleParent         = "parent"
	ParticipantRoleChild          = "child"
	ParticipantRoleAdoptiveParent = "adoptive_parent"
	ParticipantRoleAdoptedChild   = "adopted_child"
	ParticipantRoleSibling        = "sibling"
	ParticipantRoleGodparent      = "godparent"
	ParticipantRoleGodchild       = "godchild"
	ParticipantRoleEnslaver       = "enslaver"
	ParticipantRoleEnslavedPerson = "enslaved_person"
	ParticipantRoleAssociate      = "associate"
	ParticipantRoleHouseholdHead  = "household_head"
	ParticipantRoleBoarder        = "boarder"
)

// Standard Person Property Names - commonly used properties on Person entities
const (
	PersonPropertyName       = "name"
	PersonPropertySex        = "sex"    // Sex as recorded in source documents; maps to GEDCOM SEX
	PersonPropertyGender     = "gender" // Self-identified gender identity; no direct GEDCOM mapping
	PersonPropertyResidence  = "residence"
	PersonPropertyOccupation = "occupation"
	PersonPropertyLiving     = "living" // Boolean; opt-in marker honored by export privacy filters
)

// Deprecated property constants - these properties have been removed from the spec.
// Use birth/death/burial events instead. Kept for validation error messages and migration tooling.
const (
	DeprecatedPropertyBornOn   = "born_on"
	DeprecatedPropertyBornAt   = "born_at"
	DeprecatedPropertyDiedOn   = "died_on"
	DeprecatedPropertyDiedAt   = "died_at"
	DeprecatedPropertyBuriedOn = "buried_on"
	DeprecatedPropertyBuriedAt = "buried_at"
)

// Name Field Constants - used in the name property's fields structure
const (
	NameFieldType          = "type"
	NameFieldPrefix        = "prefix"
	NameFieldGiven         = "given"
	NameFieldNickname      = "nickname"
	NameFieldSurnamePrefix = "surname_prefix"
	NameFieldSurname       = "surname"
	NameFieldSuffix        = "suffix"
)

// Common Property Names - used across multiple entity types
const (
	PropertySources          = "sources"
	PropertyCitations        = "citations"
	PropertyMedia            = "media"
	PropertyNotes            = "notes"
	PropertyAddress          = "address"
	PropertyMarriageType     = "marriage_type"
	PropertyGEDCOMExtensions = "gedcom_extensions" // Preserved GEDCOM 7.0 vendor and SCHMA-defined extension tags
)

// Extension Entry Keys - used inside the gedcom_extensions list.
// These are the import↔export contract: import writes them in
// recordToExtensionEntry, export reads them in extensionEntryToRecord, and
// any drift breaks round-trip silently.
const (
	ExtensionEntryTag        = "tag"
	ExtensionEntryValue      = "value"
	ExtensionEntrySubrecords = "subrecords"
)

// MediaFilesDir is the directory within an archive where media files are stored.
const MediaFilesDir = "media/files"

// GEDCOM Tags - Top-Level Records
const (
	GedcomTagHead  = "HEAD"  // File header
	GedcomTagTrlr  = "TRLR"  // Trailer (end of file)
	GedcomTagIndi  = "INDI"  // Individual person record
	GedcomTagFam   = "FAM"   // Family record
	GedcomTagNote  = "NOTE"  // Shared note record (GEDCOM 5.5.1)
	GedcomTagSnote = "SNOTE" // Shared note record (GEDCOM 7.0)
	GedcomTagSour  = "SOUR"  // Source record
	GedcomTagRepo  = "REPO"  // Repository record
	GedcomTagObje  = "OBJE"  // Media object record
	GedcomTagSubm  = "SUBM"  // Submitter record
	GedcomTagSchma = "SCHMA" // Extension schema record (GEDCOM 7.0)
)

// GEDCOM Tags - Individual Personal Information
const (
	GedcomTagName = "NAME" // Person's name
	GedcomTagSex  = "SEX"  // Recorded sex; maps to GLX sex property
	GedcomTagOccu = "OCCU" // Occupation
	GedcomTagReli = "RELI" // Religion
	GedcomTagEduc = "EDUC" // Education
	GedcomTagNati = "NATI" // Nationality
	GedcomTagCast = "CAST" // Caste/tribe
	GedcomTagSsn  = "SSN"  // Social security number
	GedcomTagTitl = "TITL" // Title (nobility, rank, honor)
	GedcomTagFact = "FACT" // Generic fact
	GedcomTagResi = "RESI" // Residence
)

// GEDCOM Tags - Life Events
const (
	GedcomTagBirt = "BIRT" // Birth
	GedcomTagChr  = "CHR"  // Christening
	GedcomTagDeat = "DEAT" // Death
	GedcomTagBuri = "BURI" // Burial
	GedcomTagCrem = "CREM" // Cremation
	GedcomTagAdop = "ADOP" // Adoption
	GedcomTagBapm = "BAPM" // Baptism
	GedcomTagBarm = "BARM" // Bar Mitzvah
	GedcomTagBatm = "BATM" // Bat Mitzvah
	GedcomTagBasm = "BASM" // Bas Mitzvah (alternate spelling, maps to bat_mitzvah)
	GedcomTagBles = "BLES" // Blessing
	GedcomTagChra = "CHRA" // Adult Christening
	GedcomTagConf = "CONF" // Confirmation
	GedcomTagFcom = "FCOM" // First Communion
	GedcomTagOrdn = "ORDN" // Ordination
	GedcomTagNatu = "NATU" // Naturalization
	GedcomTagEmig = "EMIG" // Emigration
	GedcomTagImmi = "IMMI" // Immigration
	GedcomTagCens = "CENS" // Census
	GedcomTagProb = "PROB" // Probate
	GedcomTagWill = "WILL" // Will
	GedcomTagGrad = "GRAD" // Graduation
	GedcomTagReti = "RETI" // Retirement
)

// GEDCOM Tags - Family Relationships
const (
	GedcomTagFamc = "FAMC" // Family as child
	GedcomTagFams = "FAMS" // Family as spouse
	GedcomTagHusb = "HUSB" // Husband reference
	GedcomTagWife = "WIFE" // Wife reference
	GedcomTagChil = "CHIL" // Child reference
)

// GEDCOM Tags - Associations
const (
	GedcomTagAsso = "ASSO" // Association (links person to event/individual with role)
	GedcomTagRole = "ROLE" // Role in association
)

// gedcomRoleToGLX maps GEDCOM ROLE enumeration values to GLX participant roles.
// Only maps to roles that exist in the standard participant-roles vocabulary.
// Roles without a vocabulary match (NGHBR, FRIEND, MULTIPLE) are stored in
// participant notes instead of the role field to avoid validation errors.
var gedcomRoleToGLX = map[string]string{
	"WITN":       ParticipantRoleWitness,
	"OFFICIATOR": ParticipantRoleOfficiant,
	"CLERGY":     ParticipantRoleOfficiant,
	"GODP":       ParticipantRoleGodparent,
	"CHIL":       ParticipantRoleChild,
	"FATH":       ParticipantRoleParent,
	"MOTH":       ParticipantRoleParent,
	"HUSB":       ParticipantRoleSpouse,
	"WIFE":       ParticipantRoleSpouse,
	"PARENT":     ParticipantRoleParent,
	"SPOU":       ParticipantRoleSpouse,
}

// GEDCOM Tags - Family Events
const (
	GedcomTagMarr = "MARR" // Marriage event
	GedcomTagDiv  = "DIV"  // Divorce event
	GedcomTagEnga = "ENGA" // Engagement
	GedcomTagMarb = "MARB" // Marriage banns
	GedcomTagMarc = "MARC" // Marriage contract
	GedcomTagMarl = "MARL" // Marriage license
	GedcomTagMars = "MARS" // Marriage settlement
	GedcomTagAnul = "ANUL" // Annulment
	GedcomTagDivf = "DIVF" // Divorce filed
	GedcomTagEven = "EVEN" // Generic event
)

// GEDCOM Tags - Name Substructure
const (
	GedcomTagNpfx = "NPFX" // Name prefix
	GedcomTagGivn = "GIVN" // Given name
	GedcomTagNick = "NICK" // Nickname
	GedcomTagSpfx = "SPFX" // Surname prefix
	GedcomTagSurn = "SURN" // Surname
	GedcomTagNsfx = "NSFX" // Name suffix
)

// GEDCOM Tags - Event Substructure
const (
	GedcomTagDate = "DATE" // Date of event
	GedcomTagPlac = "PLAC" // Place of event
	GedcomTagAge  = "AGE"  // Age at event
	GedcomTagCaus = "CAUS" // Cause (of death, etc.)
	GedcomTagType = "TYPE" // Event type/subtype
	GedcomTagAddr = "ADDR" // Address
)

// GEDCOM Tags - Address Substructure
const (
	GedcomTagAdr1 = "ADR1" // Address line 1
	GedcomTagAdr2 = "ADR2" // Address line 2
	GedcomTagAdr3 = "ADR3" // Address line 3
	GedcomTagCity = "CITY" // City
	GedcomTagStae = "STAE" // State
	GedcomTagPost = "POST" // Postal code
	GedcomTagCtry = "CTRY" // Country
)

// GEDCOM Tags - Place Substructure
const (
	GedcomTagMap  = "MAP"  // Map coordinates
	GedcomTagLati = "LATI" // Latitude
	GedcomTagLong = "LONG" // Longitude
)

// GEDCOM Tags - Source Information
const (
	GedcomTagAuth = "AUTH" // Author
	GedcomTagPubl = "PUBL" // Publication information
	GedcomTagAbbr = "ABBR" // Abbreviation
	GedcomTagText = "TEXT" // Full source text
	GedcomTagData = "DATA" // Source data
	GedcomTagAgnc = "AGNC" // Agency
)

// GEDCOM Tags - Repository Information
const (
	GedcomTagPhon  = "PHON"  // Phone
	GedcomTagEmail = "EMAIL" // Email
	GedcomTagWww   = "WWW"   // Website
)

// GEDCOM Tags - Media/File Handling
const (
	GedcomTagFile   = "FILE"   // File reference
	GedcomTagForm   = "FORM"   // Format
	GedcomTagCrop   = "CROP"   // Crop coordinates (GEDCOM 7.0)
	GedcomTagMime   = "MIME"   // MIME type (GEDCOM 7.0)
	GedcomTagMedi   = "MEDI"   // Media type (GEDCOM 5.5.1)
	GedcomTagTop    = "TOP"    // Top coordinate
	GedcomTagLeft   = "LEFT"   // Left coordinate
	GedcomTagHeight = "HEIGHT" // Height
	GedcomTagWidth  = "WIDTH"  // Width
	GedcomTagBlob   = "BLOB"   // Binary large object (GEDCOM 5.5.1, deprecated)
)

// GEDCOM Tags - Citation/Evidence
const (
	GedcomTagPage = "PAGE" // Page/location within source
	GedcomTagQuay = "QUAY" // Quality assessment (preserved in citation notes)
	GedcomTagCont = "CONT" // Continuation on new line
	GedcomTagConc = "CONC" // Concatenation (same line)
)

// GEDCOM Tags - Header Metadata
const (
	GedcomTagCopr = "COPR" // Copyright
	GedcomTagLang = "LANG" // Language
	GedcomTagGedc = "GEDC" // GEDCOM version info
	GedcomTagChar = "CHAR" // Character set
	GedcomTagVers = "VERS" // Version
	GedcomTagCorp = "CORP" // Corporation
)

// GEDCOM Tags - GEDCOM 7.0 Specific
const (
	GedcomTagNo   = "NO"   // Negative assertion
	GedcomTagExid = "EXID" // External ID
	GedcomTagPedi = "PEDI" // Pedigree linkage type
	GedcomTagCaln = "CALN" // Call number
	GedcomTagTag  = "TAG"  // Extension tag name
	GedcomTagURI  = "URI"  // Schema URI
)

// Standard Confidence Levels - from confidence-levels.glx vocabulary
const (
	ConfidenceLevelHigh   = "high"   // Multiple high-quality sources agree, minimal uncertainty
	ConfidenceLevelMedium = "medium" // Some evidence supports conclusion, but conflicts or gaps exist
	ConfidenceLevelLow    = "low"    // Limited evidence, significant uncertainty
)

// Standard Search Result Types - from search-result-types.glx vocabulary.
// Used by Search.Result inside a ResearchLog to record whether a search located
// the target, found nothing (negative evidence), returned an unclear result, or
// is still queued.
const (
	SearchResultFound        = "found"
	SearchResultNotFound     = "not_found"
	SearchResultInconclusive = "inconclusive"
	SearchResultPartial      = "partial"
	SearchResultNotSearched  = "not_searched"
)

// Standard Research Log Status Types - from research-log-status-types.glx vocabulary.
// Used by ResearchLog.Status to record the lifecycle state of a research investigation.
const (
	ResearchLogStatusOpen       = "open"
	ResearchLogStatusInProgress = "in_progress"
	ResearchLogStatusComplete   = "complete"
	ResearchLogStatusBlocked    = "blocked"
)

// Standard Source Types - from source-types.glx vocabulary
const (
	SourceTypeVitalRecord        = "vital_record"        // Birth, marriage, death certificates
	SourceTypeCensus             = "census"              // Census records and population enumerations
	SourceTypeChurchRegister     = "church_register"     // Parish registers of baptisms, marriages, burials
	SourceTypeMilitary           = "military"            // Military service records, pension files
	SourceTypeNewspaper          = "newspaper"           // Newspapers, periodicals, gazettes
	SourceTypeProbate            = "probate"             // Wills, probate records, estate files
	SourceTypeLand               = "land"                // Deeds, land grants, property records
	SourceTypeCourt              = "court"               // Court records, legal proceedings
	SourceTypeImmigration        = "immigration"         // Passenger lists, naturalization records
	SourceTypeDirectory          = "directory"           // City directories, telephone books
	SourceTypeBook               = "book"                // Published genealogies, family histories
	SourceTypeDatabase           = "database"            // Online databases, compiled records
	SourceTypeOralHistory        = "oral_history"        // Interviews, recorded memories
	SourceTypeCorrespondence     = "correspondence"      // Letters, emails, personal papers
	SourceTypePhotograph         = "photograph"          // Photograph collections
	SourceTypePopulationRegister = "population_register" // Civil population registers
	SourceTypeTaxRecord          = "tax_record"          // Tax rolls, assessments, tithes
	SourceTypeNotarialRecord     = "notarial_record"     // Notarial acts and contracts
	SourceTypeFamilyBible        = "family_bible"        // Family Bible births, marriages, deaths
	SourceTypeGravestone         = "gravestone"          // Tombstone and headstone inscriptions
	SourceTypeDNATest            = "dna_test"            // Genetic test results (autosomal, Y-DNA, mtDNA)
	SourceTypeMemoir             = "memoir"              // Autobiographies, diaries, personal journals
	SourceTypeManuscript         = "manuscript"          // Unpublished manuscripts or typescripts
	SourceTypeMap                = "map"                 // Historical maps, atlases, survey plats
	SourceTypeSocialMedia        = "social_media"        // Social media posts and profiles
	SourceTypeOther              = "other"               // Other source types
)

// gedcomSourceTypeMapping maps GEDCOM source type values to GLX source types.
// Package-level to avoid allocation on every call.
var gedcomSourceTypeMapping = map[string]string{
	"book":       SourceTypeBook,
	"article":    SourceTypeBook,
	"website":    SourceTypeDatabase,
	"database":   SourceTypeDatabase,
	"census":     SourceTypeCensus,
	"vital":      SourceTypeVitalRecord,
	"church":     SourceTypeChurchRegister,
	"military":   SourceTypeMilitary,
	"newspaper":  SourceTypeNewspaper,
	"probate":    SourceTypeProbate,
	"land":       SourceTypeLand,
	"court":      SourceTypeCourt,
	"photo":      SourceTypePhotograph,
	"photograph": SourceTypePhotograph,
	"tax":        SourceTypeTaxRecord,
	"notarial":   SourceTypeNotarialRecord,
	"population": SourceTypePopulationRegister,
	"bible":      SourceTypeFamilyBible,
	"gravestone": SourceTypeGravestone,
	"tombstone":  SourceTypeGravestone,
	"headstone":  SourceTypeGravestone,
	"dna":        SourceTypeDNATest,
	"memoir":     SourceTypeMemoir,
	"diary":      SourceTypeMemoir,
	"manuscript": SourceTypeManuscript,
	"map":        SourceTypeMap,
	"social":     SourceTypeSocialMedia,
	// Identity mappings for canonical GLX keys whose GEDCOM alias differs from
	// the key itself. Our GEDCOM 7.0 exporter writes Source.Type verbatim as the
	// TYPE value, so without these a GLX->GEDCOM->GLX round-trip would downgrade
	// these types to "other" on re-import (#563). Identity mappings carry no
	// false-positive risk. (gravestone/memoir/manuscript/map already round-trip
	// because their GEDCOM alias equals the canonical key.)
	SourceTypeFamilyBible: SourceTypeFamilyBible,
	SourceTypeDNATest:     SourceTypeDNATest,
	SourceTypeSocialMedia: SourceTypeSocialMedia,
}

// EntityType identifies a kind of GLX entity. The string value is the canonical
// PLURAL form, which doubles as the top-level YAML key, the multi-file archive
// directory name, and the map key in GLXFile (e.g., `persons:`, `persons/`).
// Use the Singular method for the singular form (entity ID prefixes, per-entity
// YAML wrappers, display labels).
type EntityType string

// Plural entity-type values. The string value is the canonical plural form.
const (
	EntityTypePersons       EntityType = "persons"
	EntityTypeRelationships EntityType = "relationships"
	EntityTypeEvents        EntityType = "events"
	EntityTypePlaces        EntityType = "places"
	EntityTypeSources       EntityType = "sources"
	EntityTypeCitations     EntityType = "citations"
	EntityTypeRepositories  EntityType = "repositories"
	EntityTypeAssertions    EntityType = "assertions"
	EntityTypeMedia         EntityType = "media"
	EntityTypeResearchLogs  EntityType = "research_logs"
	EntityTypeStudies       EntityType = "studies"
)

// ArchiveDirVocabularies is the top-level directory name (and YAML key) for
// archive-owned vocabulary files in a multi-file GLX archive. (Vocabularies
// are not entities, so this is intentionally not an EntityType.)
const ArchiveDirVocabularies = "vocabularies"

// AllEntityTypes lists every GLX entity type in canonical order. Consumers that
// need to enumerate entity directories (init scaffolding, archive-managed
// top-level set, etc.) MUST derive their list from this slice so that adding
// a new entity type updates every consumer in one place.
var AllEntityTypes = []EntityType{
	EntityTypePersons,
	EntityTypeRelationships,
	EntityTypeEvents,
	EntityTypePlaces,
	EntityTypeSources,
	EntityTypeCitations,
	EntityTypeRepositories,
	EntityTypeAssertions,
	EntityTypeMedia,
	EntityTypeResearchLogs,
	EntityTypeStudies,
}

// entityTypeSingular maps each EntityType to its singular form. Singulars use
// hyphens (`research-log`) rather than underscores (`research_logs`) because
// they participate in entity IDs (e.g., `research-log-john-smith`), which are
// constrained to `[a-zA-Z0-9-]{1,64}`. This map IS the source of truth — the
// values are inherently string literals tied to the entity-type vocabulary,
// so goconst's "extract a constant" suggestion is misplaced here.
//
//nolint:goconst // the literal values are the source of truth for singular names
var entityTypeSingular = map[EntityType]string{
	EntityTypePersons:       "person",
	EntityTypeRelationships: "relationship",
	EntityTypeEvents:        "event",
	EntityTypePlaces:        "place",
	EntityTypeSources:       "source",
	EntityTypeCitations:     "citation",
	EntityTypeRepositories:  "repository",
	EntityTypeAssertions:    "assertion",
	EntityTypeMedia:         "media",
	EntityTypeResearchLogs:  "research-log",
	EntityTypeStudies:       "study",
}

// String returns the plural form (the canonical wire-format value).
func (t EntityType) String() string { return string(t) }

// Plural returns the plural form (the canonical YAML key / directory name).
func (t EntityType) Plural() string { return string(t) }

// Singular returns the singular form used in entity ID prefixes (e.g.,
// `person`) and per-entity YAML wrappers (`{ person: { ... } }`).
func (t EntityType) Singular() string { return entityTypeSingular[t] }

// IDPrefix returns `Singular() + "-"` — the prefix used when minting
// deterministic entity IDs (e.g., `person-john-smith`).
func (t EntityType) IDPrefix() string { return t.Singular() + "-" }

// IsValidEntityType reports whether s is one of the recognized plural-form
// entity-type values.
func IsValidEntityType(s string) bool {
	_, ok := entityTypeSingular[EntityType(s)]

	return ok
}

// Entity ID prefixes - prepended to slugs when minting deterministic IDs.
// Always reference these constants instead of rebuilding the "<type>-" literal
// at the call site.
const (
	EntityIDPrefixPerson       = "person-"
	EntityIDPrefixRelationship = "relationship-"
	EntityIDPrefixEvent        = "event-"
	EntityIDPrefixPlace        = "place-"
	EntityIDPrefixSource       = "source-"
	EntityIDPrefixCitation     = "citation-"
	EntityIDPrefixRepository   = "repository-"
	EntityIDPrefixAssertion    = "assertion-"
	EntityIDPrefixMedia        = "media-"
)

// Entity ID length bounds, per the GLX entity ID pattern `[a-zA-Z0-9-]{1,64}`.
const (
	MinEntityIDLength = 1
	MaxEntityIDLength = 64
)

// Vocabulary type constants - used as map keys in GLXFile
const (
	VocabRelationshipTypes      = "relationship_types"
	VocabEventTypes             = "event_types"
	VocabPlaceTypes             = "place_types"
	VocabRepositoryTypes        = "repository_types"
	VocabParticipantRoles       = "participant_roles"
	VocabMediaTypes             = "media_types"
	VocabConfidenceLevels       = "confidence_levels"
	VocabSourceTypes            = "source_types"
	VocabSexTypes               = "sex_types"
	VocabGenderTypes            = "gender_types"
	VocabSearchResultTypes      = "search_result_types"
	VocabResearchLogStatusTypes = "research_log_status_types"
	VocabStudyTypes             = "study_types"
	VocabStudyStatuses          = "study_statuses"
	VocabLegalStatuses          = "legal_statuses"
	VocabSourceNatures          = "source_natures"
	VocabInformationTypes       = "information_types"
)

// Standard Study Types - from study-types.glx vocabulary
const (
	StudyTypeOnePlaceStudy        = "one_place_study"
	StudyTypeOneNameStudy         = "one_name_study"
	StudyTypeFamilyReconstruction = "family_reconstruction"
	StudyTypeDescendancyStudy     = "descendancy_study"
	StudyTypeAncestryStudy        = "ancestry_study"
	StudyTypeBrickWall            = "brick_wall"
	StudyTypeOther                = "other"
)

// Standard Study Statuses - from study-statuses.glx vocabulary
const (
	StudyStatusActive    = "active"
	StudyStatusPaused    = "paused"
	StudyStatusCompleted = "completed"
	StudyStatusAbandoned = "abandoned"
)

// Property vocabulary constants - used as map keys in GLXFile
const (
	PropPersonProperties       = "person_properties"
	PropEventProperties        = "event_properties"
	PropRelationshipProperties = "relationship_properties"
	PropPlaceProperties        = "place_properties"
	PropMediaProperties        = "media_properties"
	PropRepositoryProperties   = "repository_properties"
	PropCitationProperties     = "citation_properties"
	PropSourceProperties       = "source_properties"
)

// Place Types - used by inferPlaceType function
const (
	PlaceTypeCemetery = "cemetery"
	PlaceTypeChurch   = "church"
	PlaceTypeHospital = "hospital"
	PlaceTypeCounty   = "county"
	PlaceTypeState    = "state"
	PlaceTypeCity     = "city"
	PlaceTypeCountry  = "country"
	PlaceTypeLocality = "locality"
)

// Repository Types - used by inferRepositoryType and mapRepositoryType. The
// GEDCOM REPO.TYPE → GLX-key mapping is no longer hardcoded here; it is read
// from the gedcom: fields on entries in
// specification/5-standard-vocabularies/repository-types.glx.
const (
	RepositoryTypeArchive           = "archive"
	RepositoryTypeLibrary           = "library"
	RepositoryTypeChurch            = "church"
	RepositoryTypeMuseum            = "museum"
	RepositoryTypeUniversity        = "university"
	RepositoryTypeHistoricalSociety = "historical_society"
	RepositoryTypeDatabase          = "database"
	RepositoryTypeRegistry          = "registry"
	RepositoryTypeGovernmentAgency  = "government_agency"
	RepositoryTypeOther             = "other"
)

// MIME Types - Common media types
const (
	MimeTypeJPEG        = "image/jpeg"
	MimeTypePNG         = "image/png"
	MimeTypeGIF         = "image/gif"
	MimeTypeBMP         = "image/bmp"
	MimeTypeTIFF        = "image/tiff"
	MimeTypeWEBP        = "image/webp"
	MimeTypePCX         = "image/x-pcx"
	MimeTypeMP3         = "audio/mpeg"
	MimeTypeWAV         = "audio/wav"
	MimeTypeOGG         = "audio/ogg"
	MimeTypeM4A         = "audio/mp4"
	MimeTypeFLAC        = "audio/flac"
	MimeTypeMP4         = "video/mp4"
	MimeTypeAVI         = "video/x-msvideo"
	MimeTypeMOV         = "video/quicktime"
	MimeTypeWMV         = "video/x-ms-wmv"
	MimeTypeFLV         = "video/x-flv"
	MimeTypeWEBM        = "video/webm"
	MimeTypePDF         = "application/pdf"
	MimeTypeDOC         = "application/msword"
	MimeTypeDOCX        = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeTypeTXT         = "text/plain"
	MimeTypeRTF         = "application/rtf"
	MimeTypeZIP         = "application/zip"
	MimeTypeRAR         = "application/x-rar-compressed"
	MimeType7Z          = "application/x-7z-compressed"
	MimeTypeTAR         = "application/x-tar"
	MimeTypeGZIP        = "application/gzip"
	MimeTypeOctetStream = "application/octet-stream"
)

// mimeTypeByExtension maps file extensions (with dot) to MIME types.
// Package-level to avoid allocation on every call.
var mimeTypeByExtension = map[string]string{
	// Images
	".jpg":  MimeTypeJPEG,
	".jpeg": MimeTypeJPEG,
	".png":  MimeTypePNG,
	".gif":  MimeTypeGIF,
	".bmp":  MimeTypeBMP,
	".tif":  MimeTypeTIFF,
	".tiff": MimeTypeTIFF,
	".webp": MimeTypeWEBP,
	".svg":  "image/svg+xml",
	// Audio
	".mp3":  MimeTypeMP3,
	".wav":  MimeTypeWAV,
	".ogg":  MimeTypeOGG,
	".m4a":  MimeTypeM4A,
	".flac": MimeTypeFLAC,
	// Video
	".mp4":  MimeTypeMP4,
	".avi":  MimeTypeAVI,
	".mov":  MimeTypeMOV,
	".wmv":  MimeTypeWMV,
	".flv":  MimeTypeFLV,
	".webm": MimeTypeWEBM,
	// Documents
	".pdf":  MimeTypePDF,
	".doc":  MimeTypeDOC,
	".docx": MimeTypeDOCX,
	".txt":  MimeTypeTXT,
	".rtf":  MimeTypeRTF,
	// Archives
	".zip": MimeTypeZIP,
	".rar": MimeTypeRAR,
	".7z":  MimeType7Z,
	".tar": MimeTypeTAR,
	".gz":  MimeTypeGZIP,
}

// mimeTypeByFormat maps GEDCOM 5.5.1 FORM values (without dot) to MIME types.
// Package-level to avoid allocation on every call.
var mimeTypeByFormat = map[string]string{
	// Images
	"jpg":  MimeTypeJPEG,
	"jpeg": MimeTypeJPEG,
	"png":  MimeTypePNG,
	"gif":  MimeTypeGIF,
	"bmp":  MimeTypeBMP,
	"tif":  MimeTypeTIFF,
	"tiff": MimeTypeTIFF,
	"pcx":  MimeTypePCX,
	// Audio
	"wav": MimeTypeWAV,
	"mp3": MimeTypeMP3,
	// Video
	"avi": MimeTypeAVI,
	"mpg": "video/mpeg",
	"mp4": MimeTypeMP4,
	// Documents
	"pdf": MimeTypePDF,
	"txt": MimeTypeTXT,
}

// Crop Coordinate Keys - used for GEDCOM CROP tag
const (
	CropKeyTop    = "top"
	CropKeyLeft   = "left"
	CropKeyHeight = "height"
	CropKeyWidth  = "width"
)

// GEDCOM Version Strings
const (
	GEDCOMVersion551 = "5.5.1"
	GEDCOMVersion70  = "7.0"
)

// Sex values for GEDCOM import/export mapping.
// GEDCOM SEX tag: M→male, F→female, U→unknown, X→other.
// Sex is vocabulary-constrained via vocabulary_type: sex_types.
const (
	SexMale        = "male"
	SexFemale      = "female"
	SexUnknown     = "unknown"
	SexNotRecorded = "not_recorded"
	SexOther       = "other"
)

// Gender identity values. Vocabulary-constrained via vocabulary_type: gender_types.
// No direct GEDCOM mapping (GEDCOM defers identity to FACT).
const (
	GenderMale      = "male"
	GenderFemale    = "female"
	GenderNonbinary = "nonbinary"
	GenderOther     = "other"
)

// File Extensions
const (
	FileExtGLX = ".glx"
)
