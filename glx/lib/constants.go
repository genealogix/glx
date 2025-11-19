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
	RelationshipTypeMarriage    = "marriage"
	RelationshipTypeParentChild = "parent-child"
	RelationshipTypeSibling     = "sibling"
	RelationshipTypeAdoption    = "adoption"
	RelationshipTypeStepParent  = "step-parent"
	RelationshipTypeGodparent   = "godparent"
	RelationshipTypeGuardian    = "guardian"
	RelationshipTypePartner     = "partner"
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
