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
	PersonPropertyGivenName     = "given_name"
	PersonPropertyFamilyName    = "family_name"
	PersonPropertyNickname      = "nickname"
	PersonPropertyNamePrefix    = "name_prefix"
	PersonPropertySurnamePrefix = "surname_prefix"
	PersonPropertyNameSuffix    = "name_suffix"
	PersonPropertyGender        = "gender"
	PersonPropertyBirthPlace    = "birth_place"
	PersonPropertyDeathPlace    = "death_place"
	PersonPropertyBirthDate     = "birth_date"
	PersonPropertyDeathDate     = "death_date"
	PersonPropertyOccupation    = "occupation"
	PersonPropertyReligion      = "religion"
	PersonPropertyEducation     = "education"
	PersonPropertyNationality   = "nationality"
	PersonPropertyCaste         = "caste"
	PersonPropertySSN           = "ssn"
	PersonPropertyTitle         = "title"
	PersonPropertyExternalIDs   = "external_ids"
	PersonPropertyBornOn        = "born_on"
	PersonPropertyBornAt        = "born_at"
	PersonPropertyDiedOn        = "died_on"
	PersonPropertyDiedAt        = "died_at"
	PersonPropertyResidence     = "residence"
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
