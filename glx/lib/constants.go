// Package lib provides core GENEALOGIX data types and GEDCOM conversion functionality.
package lib

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
	EventTypeMarriageBan        = "marriage-ban"
	EventTypeMarriageBanns      = "marriage_banns"
	EventTypeMarriageContract   = "marriage_contract"
	EventTypeMarriageLicense    = "marriage_license"
	EventTypeMarriageSettlement = "marriage_settlement"
	EventTypeDivorceFiled       = "divorce_filed"
	EventTypeCensus             = "census"
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
	EventTypeBasMitzvah         = "bas_mitzvah"
	EventTypeBlessing           = "blessing"
	EventTypeConfirmation       = "confirmation"
	EventTypeFirstCommunion     = "first_communion"
	EventTypeOrdination         = "ordination"
	EventTypeProbate            = "probate"
	EventTypeWill               = "will"
	EventTypeGeneric            = "event"
)

// Standard Relationship Types - from relationship-types.glx vocabulary
const (
	RelationshipTypeMarriage              = "marriage"
	RelationshipTypeParentChild           = "parent-child"
	RelationshipTypeBiologicalParentChild = "biological-parent-child"
	RelationshipTypeAdoptiveParentChild   = "adoptive-parent-child"
	RelationshipTypeFosterParentChild     = "foster-parent-child"
	RelationshipTypeSibling               = "sibling"
	RelationshipTypeAdoption              = "adoption"
	RelationshipTypeStepParent            = "step-parent"
	RelationshipTypeGodparent             = "godparent"
	RelationshipTypeGuardian              = "guardian"
	RelationshipTypePartner               = "partner"
)

// Standard Participant Roles - from participant-roles.glx vocabulary
const (
	ParticipantRolePrincipal = "principal"
	ParticipantRoleSpouse    = "spouse"
	ParticipantRoleParent    = "parent"
	ParticipantRoleChild     = "child"
	ParticipantRoleWitness   = "witness"
	ParticipantRoleOfficiant = "officiant"
	ParticipantRoleInformant = "informant"
)

// Standard Person Property Names - commonly used properties on Person entities
const (
	PersonPropertyName        = "name"
	PersonPropertyGender      = "gender"
	PersonPropertyBirthPlace  = "birth_place"
	PersonPropertyDeathPlace  = "death_place"
	PersonPropertyBirthDate   = "birth_date"
	PersonPropertyDeathDate   = "death_date"
	PersonPropertyOccupation  = "occupation"
	PersonPropertyTitle       = "title"
	PersonPropertyReligion    = "religion"
	PersonPropertyEducation   = "education"
	PersonPropertyNationality = "nationality"
	PersonPropertyCaste       = "caste"
	PersonPropertySSN         = "ssn"
	PersonPropertyExternalIDs = "external_ids"
	PersonPropertyBornOn      = "born_on"
	PersonPropertyBornAt      = "born_at"
	PersonPropertyDiedOn      = "died_on"
	PersonPropertyDiedAt      = "died_at"
	PersonPropertyResidence   = "residence"
)

// Name Field Constants - used in the name property's fields structure
const (
	NameFieldPrefix        = "prefix"
	NameFieldGiven         = "given"
	NameFieldNickname      = "nickname"
	NameFieldSurnamePrefix = "surname_prefix"
	NameFieldSurname       = "surname"
	NameFieldSuffix        = "suffix"
)

// Common Property Names - used across multiple entity types
const (
	PropertyCitations     = "citations"
	PropertyMedia         = "media"
	PropertyNotes         = "notes"
	PropertyAddress       = "address"
	PropertyMarriageEvent = "marriage_event"
	PropertyDivorceEvent  = "divorce_event"
	PropertyMarriageType  = "marriage_type"
	PropertyAgeAtEvent    = "age_at_event"
	PropertyCause         = "cause"
	PropertyEventSubtype  = "event_subtype"
)

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
	GedcomTagSex  = "SEX"  // Gender/sex
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
	GedcomTagBasm = "BASM" // Bas Mitzvah
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
	ConfidenceLevelHigh     = "high"     // Multiple high-quality sources agree, minimal uncertainty
	ConfidenceLevelMedium   = "medium"   // Some evidence supports conclusion, but conflicts or gaps exist
	ConfidenceLevelLow      = "low"      // Limited evidence, significant uncertainty
	ConfidenceLevelDisputed = "disputed" // Multiple sources conflict, resolution unclear
)

// Standard Source Types - from source-types.glx vocabulary
const (
	SourceTypeVitalRecord    = "vital_record"    // Birth, marriage, death certificates
	SourceTypeCensus         = "census"          // Census records and population enumerations
	SourceTypeChurchRegister = "church_register" // Parish registers of baptisms, marriages, burials
	SourceTypeMilitary       = "military"        // Military service records, pension files
	SourceTypeNewspaper      = "newspaper"       // Newspapers, periodicals, gazettes
	SourceTypeProbate        = "probate"         // Wills, probate records, estate files
	SourceTypeLand           = "land"            // Deeds, land grants, property records
	SourceTypeCourt          = "court"           // Court records, legal proceedings
	SourceTypeImmigration    = "immigration"     // Passenger lists, naturalization records
	SourceTypeDirectory      = "directory"       // City directories, telephone books
	SourceTypeBook           = "book"            // Published genealogies, family histories
	SourceTypeDatabase       = "database"        // Online databases, compiled records
	SourceTypeOralHistory    = "oral_history"    // Interviews, recorded memories
	SourceTypeCorrespondence = "correspondence"  // Letters, emails, personal papers
	SourceTypePhotograph     = "photograph"      // Photograph collections
	SourceTypeOther          = "other"           // Other source types
)

// Entity type constants - plural form used as map keys in GLXFile
const (
	EntityTypePersons       = "persons"
	EntityTypeRelationships = "relationships"
	EntityTypeEvents        = "events"
	EntityTypePlaces        = "places"
	EntityTypeSources       = "sources"
	EntityTypeCitations     = "citations"
	EntityTypeRepositories  = "repositories"
	EntityTypeAssertions    = "assertions"
	EntityTypeMedia         = "media"
)

// Vocabulary type constants - used as map keys in GLXFile
const (
	VocabRelationshipTypes = "relationship_types"
	VocabEventTypes        = "event_types"
	VocabPlaceTypes        = "place_types"
	VocabRepositoryTypes   = "repository_types"
	VocabParticipantRoles  = "participant_roles"
	VocabMediaTypes       = "media_types"
	VocabConfidenceLevels = "confidence_levels"
	VocabSourceTypes      = "source_types"
)

// Property vocabulary constants - used as map keys in GLXFile
const (
	PropPersonProperties       = "person_properties"
	PropEventProperties        = "event_properties"
	PropRelationshipProperties = "relationship_properties"
	PropPlaceProperties        = "place_properties"
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

// Repository Types - used by inferRepositoryType function
const (
	RepositoryTypeArchive           = "archive"
	RepositoryTypeLibrary           = "library"
	RepositoryTypeChurch            = "church"
	RepositoryTypeMuseum            = "museum"
	RepositoryTypeUniversity        = "university"
	RepositoryTypeHistoricalSociety = "historical_society"
	RepositoryTypeDatabase          = "database"
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

// Gender Values
const (
	GenderMale    = "male"
	GenderFemale  = "female"
	GenderUnknown = "unknown"
)

// File Extensions
const (
	FileExtGLX = ".glx"
)
