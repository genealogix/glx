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

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/genealogix/spec/lib"
	schema "github.com/genealogix/spec/specification/schema/v1"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

const (
	// Entity type constants
	EntityTypePersons       = "persons"
	EntityTypeRelationships = "relationships"
	EntityTypeEvents        = "events"
	EntityTypePlaces        = "places"
	EntityTypeSources       = "sources"
	EntityTypeCitations     = "citations"
	EntityTypeRepositories  = "repositories"
	EntityTypeAssertions    = "assertions"
	EntityTypeMedia         = "media"

	// File extensions
	FileExtGLX  = ".glx"
	FileExtYAML = ".yaml"
	FileExtYML  = ".yml"

	// ID validation constants
	MinEntityIDLength = 1
	MaxEntityIDLength = 64

	// Vocabulary file keys
	VocabRelationshipTypes = "relationship_types"
	VocabEventTypes        = "event_types"
	VocabPlaceTypes        = "place_types"
	VocabRepositoryTypes   = "repository_types"
	VocabParticipantRoles  = "participant_roles"
	VocabMediaTypes        = "media_types"
	VocabConfidenceLevels  = "confidence_levels"
	VocabQualityRatings    = "quality_ratings"

	// Property vocabulary keys
	PropPersonProperties       = "person_properties"
	PropEventProperties        = "event_properties"
	PropRelationshipProperties = "relationship_properties"
	PropPlaceProperties        = "place_properties"
)

// ParseYAMLFile parses YAML content into a map
func ParseYAMLFile(data []byte) (map[string]interface{}, error) {
	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// ValidateGLXFile validates a single GLX file
func ValidateGLXFile(path string, doc map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Entity type keys and their singular forms for validation
	entityTypes := map[string]string{
		EntityTypePersons:       "person",
		EntityTypeRelationships: "relationship",
		EntityTypeEvents:        "event",
		EntityTypePlaces:        "place",
		EntityTypeSources:       "source",
		EntityTypeCitations:     "citation",
		EntityTypeRepositories:  "repository",
		EntityTypeAssertions:    "assertion",
		EntityTypeMedia:         "media",
	}

	// Validate entity sections
	for pluralKey, singularType := range entityTypes {
		if entities, ok := doc[pluralKey].(map[string]interface{}); ok {
			for entityID, entityData := range entities {
				if entityMap, ok := entityData.(map[string]interface{}); ok {
					// Reject if entity has an "id" field (map key is the ID)
					if _, hasID := entityMap["id"]; hasID {
						issues = append(issues, fmt.Sprintf("%s[%s]: entity must not have 'id' field - the map key is the ID", pluralKey, entityID))
						continue
					}

					// Validate entity ID format (alphanumeric + hyphens, 1-64 chars)
					if !isValidEntityID(entityID) {
						issues = append(issues, fmt.Sprintf("%s[%s]: invalid entity ID (must be alphanumeric/hyphens, 1-64 chars)", pluralKey, entityID))
					}

					// Validate individual entity
					entityIssues := validateEntityByType(singularType, entityMap, vocabs)
					for _, issue := range entityIssues {
						issues = append(issues, fmt.Sprintf("%s[%s]: %s", pluralKey, entityID, issue))
					}
				}
			}
		}
	}

	return issues
}

func isValidEntityID(id string) bool {
	if len(id) < MinEntityIDLength || len(id) > MaxEntityIDLength {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-') {
			return false
		}
	}
	return true
}

func validateEntityByType(entityType string, entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Get schema bytes
	schemaBytes := getSchemaBytes(entityType)
	if schemaBytes == nil {
		// Fallback to basic validation if schema not found
		return basicValidateEntity(entityType, entity, vocabs)
	}

	// Load and validate against JSON schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)

	// Convert entity to JSON for validation
	entityJSON, err := json.Marshal(entity)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal entity: %v", err))
		return issues
	}

	documentLoader := gojsonschema.NewBytesLoader(entityJSON)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		// If schema validation fails, fall back to basic validation
		return basicValidateEntity(entityType, entity, vocabs)
	}

	if !result.Valid() {
		for _, desc := range result.Errors() {
			issues = append(issues, desc.String())
		}
	}

	return issues
}

func getSchemaBytes(entityType string) []byte {
	if schemaBytes, ok := schema.EntitySchemas[entityType]; ok {
		return schemaBytes
	}
	return nil
}

func basicValidateEntity(entityType string, entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	// Delegate to type-specific validation functions
	switch entityType {
	case "person":
		return validatePersonEntity(entity, vocabs)
	case "relationship":
		return validateRelationshipEntity(entity, vocabs)
	case "event":
		return validateEventEntity(entity, vocabs)
	case "place":
		return validatePlaceEntity(entity, vocabs)
	case "source":
		return validateSourceEntity(entity, vocabs)
	case "citation":
		return validateCitationEntity(entity, vocabs)
	case "repository":
		return validateRepositoryEntity(entity, vocabs)
	case "assertion":
		return validateAssertionEntity(entity, vocabs)
	case "media":
		return validateMediaEntity(entity, vocabs)
	default:
		return []string{fmt.Sprintf("unknown entity type: %s", entityType)}
	}
}

// validatePersonEntity validates a person entity
func validatePersonEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	// Persons don't have strict requirements beyond version
	return nil
}

// validateRelationshipEntity validates a relationship entity
func validateRelationshipEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasType := entity["type"]; !hasType {
		issues = append(issues, "type field is required for relationships")
	}
	if _, hasPersons := entity["persons"]; !hasPersons {
		issues = append(issues, "persons field is required for relationships")
	}

	// Validate against vocabularies if available
	if relType, ok := entity["type"].(string); ok && vocabs != nil && len(vocabs.RelationshipTypes) > 0 {
		if _, exists := vocabs.RelationshipTypes[relType]; !exists {
			issues = append(issues, fmt.Sprintf("relationship type '%s' is not defined in vocabulary - add it to relationship_types or use a standard type", relType))
		}
	}

	return issues
}

// validateEventEntity validates an event entity
func validateEventEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasType := entity["type"]; !hasType {
		issues = append(issues, "type field is required for events")
	}

	// Validate against vocabularies if available
	if eventType, ok := entity["type"].(string); ok && vocabs != nil && len(vocabs.EventTypes) > 0 {
		if _, exists := vocabs.EventTypes[eventType]; !exists {
			issues = append(issues, fmt.Sprintf("event type '%s' is not defined in vocabulary - add it to event_types or use a standard type", eventType))
		}
	}

	return issues
}

// validatePlaceEntity validates a place entity
func validatePlaceEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasName := entity["name"]; !hasName {
		issues = append(issues, "name field is required for places")
	}

	// Validate against vocabularies if available
	if placeType, ok := entity["type"].(string); ok && vocabs != nil && len(vocabs.PlaceTypes) > 0 {
		if _, exists := vocabs.PlaceTypes[placeType]; !exists {
			issues = append(issues, fmt.Sprintf("place type '%s' is not defined in vocabulary - add it to place_types or use a standard type", placeType))
		}
	}

	return issues
}

// validateSourceEntity validates a source entity
func validateSourceEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasTitle := entity["title"]; !hasTitle {
		issues = append(issues, "title field is required for sources")
	}

	return issues
}

// validateCitationEntity validates a citation entity
func validateCitationEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasSource := entity["source"]; !hasSource {
		issues = append(issues, "source field is required for citations")
	}

	return issues
}

// validateRepositoryEntity validates a repository entity
func validateRepositoryEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasName := entity["name"]; !hasName {
		issues = append(issues, "name field is required for repositories")
	}

	// Validate against vocabularies if available
	if repoType, ok := entity["type"].(string); ok && vocabs != nil && len(vocabs.RepositoryTypes) > 0 {
		if _, exists := vocabs.RepositoryTypes[repoType]; !exists {
			issues = append(issues, fmt.Sprintf("repository type '%s' is not defined in vocabulary - add it to repository_types or use a standard type", repoType))
		}
	}

	return issues
}

// validateAssertionEntity validates an assertion entity
func validateAssertionEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields
	if _, hasSubject := entity["subject"]; !hasSubject {
		issues = append(issues, "subject field is required for assertions")
	}
	if _, hasClaim := entity["claim"]; !hasClaim {
		issues = append(issues, "claim field is required for assertions")
	}
	if _, hasSources := entity["sources"]; !hasSources {
		if _, hasCitations := entity["citations"]; !hasCitations {
			issues = append(issues, "sources or citations field is required for assertions")
		}
	}

	// Validate against vocabularies if available
	if confidence, ok := entity["confidence"].(string); ok && vocabs != nil && len(vocabs.ConfidenceLevels) > 0 {
		if _, exists := vocabs.ConfidenceLevels[confidence]; !exists {
			issues = append(issues, fmt.Sprintf("confidence level '%s' is not defined in vocabulary - add it to confidence_levels or use a standard level", confidence))
		}
	}

	// Handle quality rating validation (can be float64 or *int)
	if quality, ok := entity["quality"].(float64); ok && vocabs != nil && len(vocabs.QualityRatings) > 0 {
		qualityStr := fmt.Sprintf("%.0f", quality)
		if _, exists := vocabs.QualityRatings[qualityStr]; !exists {
			issues = append(issues, fmt.Sprintf("quality rating '%s' is not defined in vocabulary - add it to quality_ratings or use a standard rating", qualityStr))
		}
	}
	if quality, ok := entity["quality"].(*int); ok && quality != nil && vocabs != nil && len(vocabs.QualityRatings) > 0 {
		qualityStr := fmt.Sprintf("%d", *quality)
		if _, exists := vocabs.QualityRatings[qualityStr]; !exists {
			issues = append(issues, fmt.Sprintf("quality rating '%s' is not defined in vocabulary - add it to quality_ratings or use a standard rating", qualityStr))
		}
	}

	return issues
}

// validateMediaEntity validates a media entity
func validateMediaEntity(entity map[string]interface{}, vocabs *ArchiveVocabularies) []string {
	var issues []string

	// Check required fields - either uri or file_path must be present
	if _, hasURI := entity["uri"]; !hasURI {
		if _, hasFilePath := entity["file_path"]; !hasFilePath {
			issues = append(issues, "uri or file_path field is required for media objects")
		}
	}

	// Validate against vocabularies if available
	if mediaType, ok := entity["media_type"].(string); ok && vocabs != nil && len(vocabs.MediaTypes) > 0 {
		if _, exists := vocabs.MediaTypes[mediaType]; !exists {
			issues = append(issues, fmt.Sprintf("media type '%s' is not defined in vocabulary - add it to media_types or use a standard type", mediaType))
		}
	}

	return issues
}

// ValidateVocabularyFile validates a vocabulary file against its schema
func ValidateVocabularyFile(path string, doc map[string]interface{}) []string {
	var issues []string

	// Check for vocabulary type key
	vocabKeys := []string{VocabRelationshipTypes, VocabEventTypes, VocabPlaceTypes, VocabRepositoryTypes,
		VocabParticipantRoles, VocabMediaTypes, VocabConfidenceLevels, VocabQualityRatings}
	hasVocabKey := false
	var vocabType string
	for _, key := range vocabKeys {
		if _, exists := doc[key]; exists {
			hasVocabKey = true
			vocabType = key
			break
		}
	}

	if !hasVocabKey {
		return []string{"vocabulary file must contain exactly one vocabulary type key"}
	}

	// Get schema bytes for this vocabulary type
	schemaBytes := getVocabSchemaBytes(vocabType)
	if schemaBytes == nil {
		return []string{fmt.Sprintf("no schema found for vocabulary type: %s", vocabType)}
	}

	// Validate against JSON schema
	schemaLoader := gojsonschema.NewBytesLoader(schemaBytes)
	entityJSON, err := json.Marshal(doc)
	if err != nil {
		issues = append(issues, fmt.Sprintf("failed to marshal entity: %v", err))
		return issues
	}
	documentLoader := gojsonschema.NewBytesLoader(entityJSON)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		issues = append(issues, fmt.Sprintf("schema validation failed: %v", err))
	} else if !result.Valid() {
		for _, desc := range result.Errors() {
			issues = append(issues, desc.String())
		}
	}

	return issues
}

func getVocabSchemaBytes(vocabType string) []byte {
	if schemaBytes, ok := schema.VocabularySchemas[vocabType]; ok {
		return schemaBytes
	}
	return nil
}

// ArchiveVocabularies holds all vocabulary definitions from a GLX archive.
//
// Vocabularies provide controlled vocabularies for entity properties and types.
// They are used to validate that entity fields contain valid values and to provide
// metadata about properties like their data types and validation rules.
type ArchiveVocabularies struct {
	// Entity type vocabularies - define valid types for entities
	RelationshipTypes map[string]*lib.RelationshipType
	EventTypes        map[string]*lib.EventType
	PlaceTypes        map[string]*lib.PlaceType
	RepositoryTypes   map[string]*lib.RepositoryType
	ParticipantRoles  map[string]*lib.ParticipantRole
	MediaTypes        map[string]*lib.MediaType

	// Assertion vocabularies - define valid confidence and quality levels
	ConfidenceLevels map[string]*lib.ConfidenceLevel
	QualityRatings   map[string]*lib.QualityRating

	// Property vocabularies - define custom properties for entities
	PersonProperties       map[string]*lib.PropertyDefinition
	EventProperties        map[string]*lib.PropertyDefinition
	RelationshipProperties map[string]*lib.PropertyDefinition
	PlaceProperties        map[string]*lib.PropertyDefinition
}

// buildEntityMaps creates maps of all entity IDs for reference validation.
//
// This creates a lookup structure where each entity type maps to a set of valid IDs.
// Used to quickly check if referenced entities exist during cross-reference validation.
func buildEntityMaps(archive *lib.GLXFile) map[string]map[string]bool {
	allEntities := make(map[string]map[string]bool)
	allEntities[EntityTypePersons] = make(map[string]bool)
	allEntities[EntityTypeRelationships] = make(map[string]bool)
	allEntities[EntityTypeEvents] = make(map[string]bool)
	allEntities[EntityTypePlaces] = make(map[string]bool)
	allEntities[EntityTypeSources] = make(map[string]bool)
	allEntities[EntityTypeCitations] = make(map[string]bool)
	allEntities[EntityTypeRepositories] = make(map[string]bool)
	allEntities[EntityTypeAssertions] = make(map[string]bool)
	allEntities[EntityTypeMedia] = make(map[string]bool)

	for id := range archive.Persons {
		allEntities[EntityTypePersons][id] = true
	}
	for id := range archive.Relationships {
		allEntities[EntityTypeRelationships][id] = true
	}
	for id := range archive.Events {
		allEntities[EntityTypeEvents][id] = true
	}
	for id := range archive.Places {
		allEntities[EntityTypePlaces][id] = true
	}
	for id := range archive.Sources {
		allEntities[EntityTypeSources][id] = true
	}
	for id := range archive.Citations {
		allEntities[EntityTypeCitations][id] = true
	}
	for id := range archive.Repositories {
		allEntities[EntityTypeRepositories][id] = true
	}
	for id := range archive.Assertions {
		allEntities[EntityTypeAssertions][id] = true
	}
	for id := range archive.Media {
		allEntities[EntityTypeMedia][id] = true
	}

	return allEntities
}

// buildVocabularyStruct creates an ArchiveVocabularies struct from the archive.
//
// This extracts all vocabulary definitions from the merged archive into a single
// structure that's easier to work with during validation. Vocabularies provide
// controlled values for entity types, property definitions, and other enumerated fields.
func buildVocabularyStruct(archive *lib.GLXFile) *ArchiveVocabularies {
	return &ArchiveVocabularies{
		RelationshipTypes:      archive.RelationshipTypes,
		EventTypes:             archive.EventTypes,
		PlaceTypes:             archive.PlaceTypes,
		RepositoryTypes:        archive.RepositoryTypes,
		ParticipantRoles:       archive.ParticipantRoles,
		MediaTypes:             archive.MediaTypes,
		ConfidenceLevels:       archive.ConfidenceLevels,
		QualityRatings:         archive.QualityRatings,
		PersonProperties:       archive.PersonProperties,
		EventProperties:        archive.EventProperties,
		RelationshipProperties: archive.RelationshipProperties,
		PlaceProperties:        archive.PlaceProperties,
	}
}

// ValidateArchive validates a merged GLXFile archive
func ValidateArchive(archive *lib.GLXFile, rootPath string) ([]string, []string) {
	var errors, warnings []string

	allEntities := buildEntityMaps(archive)
	vocabs := buildVocabularyStruct(archive)

	// Validate each entity type
	warns, errs := validateRelationships(archive.Relationships, allEntities, vocabs)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validateEvents(archive.Events, allEntities, vocabs)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validatePlaces(archive.Places, allEntities, vocabs)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validatePersons(archive.Persons, allEntities, vocabs)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validateCitations(archive.Citations, allEntities)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validateSources(archive.Sources, allEntities)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	warns, errs = validateAssertions(archive.Assertions, allEntities, vocabs)
	warnings = append(warnings, warns...)
	errors = append(errors, errs...)

	return errors, warnings
}

// validateRelationships validates all relationships in the archive
func validateRelationships(relationships map[string]*lib.Relationship, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string

	for relID, rel := range relationships {
		// Validate properties
		warns, errs := validateEntityPropertiesFromStruct("relationships", relID, rel.Properties, allEntities, vocabs)
		warnings = append(warnings, warns...)
		errors = append(errors, errs...)

		// Check participants reference valid persons
		for i, participant := range rel.Participants {
			if !allEntities[EntityTypePersons][participant.Person] {
				errors = append(errors, fmt.Sprintf("relationships[%s].participants[%d].person references non-existent person: %s", relID, i, participant.Person))
			}
		}
	}

	return warnings, errors
}

// validateEvents validates all events in the archive
func validateEvents(events map[string]*lib.Event, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string

	for eventID, event := range events {
		// Validate properties
		warns, errs := validateEntityPropertiesFromStruct("events", eventID, event.Properties, allEntities, vocabs)
		warnings = append(warnings, warns...)
		errors = append(errors, errs...)

		// Check place references
		if event.PlaceID != "" && !allEntities[EntityTypePlaces][event.PlaceID] {
			errors = append(errors, fmt.Sprintf("events[%s].place references non-existent place: %s", eventID, event.PlaceID))
		}

		// Check participants reference valid persons
		for i, participant := range event.Participants {
			if !allEntities[EntityTypePersons][participant.PersonID] {
				errors = append(errors, fmt.Sprintf("events[%s].participants[%d].person references non-existent person: %s", eventID, i, participant.PersonID))
			}
		}
	}

	return warnings, errors
}

// validatePlaces validates all places in the archive
func validatePlaces(places map[string]*lib.Place, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string

	for placeID, place := range places {
		// Validate properties
		warns, errs := validateEntityPropertiesFromStruct("places", placeID, place.Properties, allEntities, vocabs)
		warnings = append(warnings, warns...)
		errors = append(errors, errs...)

		// Check parent place references
		if place.ParentID != "" && !allEntities[EntityTypePlaces][place.ParentID] {
			errors = append(errors, fmt.Sprintf("places[%s].parent references non-existent place: %s", placeID, place.ParentID))
		}
	}

	return warnings, errors
}

// validatePersons validates all persons in the archive
func validatePersons(persons map[string]*lib.Person, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string

	for personID, person := range persons {
		warns, errs := validateEntityPropertiesFromStruct("persons", personID, person.Properties, allEntities, vocabs)
		warnings = append(warnings, warns...)
		errors = append(errors, errs...)
	}

	return warnings, errors
}

// validateCitations validates all citations in the archive
func validateCitations(citations map[string]*lib.Citation, allEntities map[string]map[string]bool) ([]string, []string) {
	var warnings, errors []string

	for citationID, citation := range citations {
		// Check source references
		if citation.SourceID != "" && !allEntities[EntityTypeSources][citation.SourceID] {
			errors = append(errors, fmt.Sprintf("citations[%s].source references non-existent source: %s", citationID, citation.SourceID))
		}
		// Check repository references
		if citation.RepositoryID != "" && !allEntities[EntityTypeRepositories][citation.RepositoryID] {
			errors = append(errors, fmt.Sprintf("citations[%s].repository references non-existent repository: %s", citationID, citation.RepositoryID))
		}
	}

	return warnings, errors
}

// validateSources validates all sources in the archive
func validateSources(sources map[string]*lib.Source, allEntities map[string]map[string]bool) ([]string, []string) {
	var warnings, errors []string

	for sourceID, source := range sources {
		// Check repository references
		if source.RepositoryID != "" && !allEntities[EntityTypeRepositories][source.RepositoryID] {
			errors = append(errors, fmt.Sprintf("sources[%s].repository references non-existent repository: %s", sourceID, source.RepositoryID))
		}
	}

	return warnings, errors
}

// validateAssertions validates all assertions in the archive
func validateAssertions(assertions map[string]*lib.Assertion, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string

	for assertionID, assertion := range assertions {
		// Semantic validation for assertions
		warns, errs := validateAssertionSemanticsFromStruct(assertionID, assertion, allEntities, vocabs)
		warnings = append(warnings, warns...)
		errors = append(errors, errs...)

		// Check subject references (could be person, event, relationship, place)
		found := false
		for _, entityType := range []string{EntityTypePersons, EntityTypeEvents, EntityTypeRelationships, EntityTypePlaces} {
			if allEntities[entityType][assertion.Subject] {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, fmt.Sprintf("assertions[%s].subject references non-existent entity: %s", assertionID, assertion.Subject))
		}

		// Check citations
		for i, citationID := range assertion.Citations {
			if !allEntities[EntityTypeCitations][citationID] {
				errors = append(errors, fmt.Sprintf("assertions[%s].citations[%d] references non-existent citation: %s", assertionID, i, citationID))
			}
		}

		// Check sources
		for i, sourceID := range assertion.Sources {
			if !allEntities[EntityTypeSources][sourceID] {
				errors = append(errors, fmt.Sprintf("assertions[%s].sources[%d] references non-existent source: %s", assertionID, i, sourceID))
			}
		}
	}

	return warnings, errors
}

// LoadArchive loads and merges all GLX files from a directory into a single GLXFile struct
func LoadArchive(rootPath string) (*lib.GLXFile, []string, error) {
	merged := &lib.GLXFile{
		Persons:       make(map[string]*lib.Person),
		Relationships: make(map[string]*lib.Relationship),
		Events:        make(map[string]*lib.Event),
		Places:        make(map[string]*lib.Place),
		Sources:       make(map[string]*lib.Source),
		Citations:     make(map[string]*lib.Citation),
		Repositories:  make(map[string]*lib.Repository),
		Assertions:    make(map[string]*lib.Assertion),
		Media:         make(map[string]*lib.Media),

		EventTypes:        make(map[string]*lib.EventType),
		ParticipantRoles:  make(map[string]*lib.ParticipantRole),
		ConfidenceLevels:  make(map[string]*lib.ConfidenceLevel),
		RelationshipTypes: make(map[string]*lib.RelationshipType),
		PlaceTypes:        make(map[string]*lib.PlaceType),
		SourceTypes:       make(map[string]*lib.SourceType),
		RepositoryTypes:   make(map[string]*lib.RepositoryType),
		MediaTypes:        make(map[string]*lib.MediaType),
		QualityRatings:    make(map[string]*lib.QualityRating),

		PersonProperties:       make(map[string]*lib.PropertyDefinition),
		EventProperties:        make(map[string]*lib.PropertyDefinition),
		RelationshipProperties: make(map[string]*lib.PropertyDefinition),
		PlaceProperties:        make(map[string]*lib.PropertyDefinition),
	}

	var allDuplicates []string

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(d.Name())
		if ext != FileExtGLX && ext != FileExtYAML && ext != FileExtYML {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var glxFile lib.GLXFile
		if err := yaml.Unmarshal(data, &glxFile); err != nil {
			// Skip files that can't be parsed
			return nil
		}

		// Merge this file into the combined archive
		duplicates := merged.Merge(&glxFile)
		allDuplicates = append(allDuplicates, duplicates...)

		return nil
	})

	return merged, allDuplicates, err
}



// validateEntityPropertiesFromStruct validates properties from a struct
func validateEntityPropertiesFromStruct(entityType, entityID string, properties map[string]interface{}, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string
	if properties == nil {
		return warnings, errors
	}

	var vocab map[string]*lib.PropertyDefinition
	switch entityType {
	case EntityTypePersons:
		vocab = vocabs.PersonProperties
	case EntityTypeEvents:
		vocab = vocabs.EventProperties
	case EntityTypeRelationships:
		vocab = vocabs.RelationshipProperties
	case EntityTypePlaces:
		vocab = vocabs.PlaceProperties
	default:
		return warnings, errors
	}

	if vocab == nil {
		warnings = append(warnings, fmt.Sprintf("%s[%s]: no property vocabulary found for entity type '%s'", entityType, entityID, entityType))
		return warnings, errors
	}

	for key, value := range properties {
		propDef, exists := vocab[key]
		if !exists {
			warnings = append(warnings, fmt.Sprintf("%s[%s]: unknown property '%s'", entityType, entityID, key))
			continue
		}

		// Validate reference_type if it exists
		if propDef.ReferenceType != "" {
			if refID, ok := value.(string); ok {
				if _, refExists := allEntities[propDef.ReferenceType][refID]; !refExists {
					errors = append(errors, fmt.Sprintf("%s[%s]: property '%s' references non-existent %s: %s", entityType, entityID, key, propDef.ReferenceType, refID))
				}
			}
		}
	}
	return warnings, errors
}


// validateAssertionSemanticsFromStruct validates assertion semantics from a struct.
//
// Assertions in GLX come in two forms:
// 1. Participant-based assertions: claim something about a person's role in an event
//    (has participant field, no claim or value fields)
// 2. Claim-based assertions: claim a property value about an entity
//    (has subject and claim fields, may have value field)
//
// This function validates the semantic rules for both types and checks references.
func validateAssertionSemanticsFromStruct(assertionID string, assertion *lib.Assertion, allEntities map[string]map[string]bool, vocabs *ArchiveVocabularies) ([]string, []string) {
	var warnings, errors []string
	hasParticipant := assertion.Participant != nil
	hasClaim := assertion.Claim != ""
	hasValue := assertion.Value != ""

	// Rule: `participant` and `value` are mutually exclusive
	if hasParticipant && hasValue {
		errors = append(errors, fmt.Sprintf("assertions[%s]: participant and value cannot both be present", assertionID))
	}

	// Rule: `participant` and `claim` are mutually exclusive
	if hasParticipant && hasClaim {
		errors = append(errors, fmt.Sprintf("assertions[%s]: participant and claim cannot both be present", assertionID))
	}

	// Validate participant structure if it exists
	if hasParticipant {
		// Rule: participant.person must exist
		if assertion.Participant.Person == "" {
			errors = append(errors, fmt.Sprintf("assertions[%s]: participant.person is required", assertionID))
		} else if !allEntities[EntityTypePersons][assertion.Participant.Person] {
			errors = append(errors, fmt.Sprintf("assertions[%s]: participant.person references non-existent person: %s", assertionID, assertion.Participant.Person))
		}

		// Rule: participant.role must exist in vocabulary if present
		if assertion.Participant.Role != "" {
			if _, roleExists := vocabs.ParticipantRoles[assertion.Participant.Role]; !roleExists {
				errors = append(errors, fmt.Sprintf("assertions[%s]: participant.role references non-existent role: %s", assertionID, assertion.Participant.Role))
			}
		}
	} else { // This is a claim-based assertion
		if assertion.Subject == "" {
			return warnings, errors
		}

		entityType := getEntityType(assertion.Subject, allEntities)
		if entityType == "" {
			return warnings, errors
		}

		if assertion.Claim == "" {
			return warnings, errors
		}

		var vocab map[string]*lib.PropertyDefinition
		switch entityType {
		case "person":
			vocab = vocabs.PersonProperties
		case "event":
			vocab = vocabs.EventProperties
		case "relationship":
			vocab = vocabs.RelationshipProperties
		case "place":
			vocab = vocabs.PlaceProperties
		}

		if vocab != nil {
			if _, exists := vocab[assertion.Claim]; !exists {
				// This is a soft warning
				warnings = append(warnings, fmt.Sprintf("assertions[%s]: unknown claim '%s' for entity type '%s'", assertionID, assertion.Claim, entityType))
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("assertions[%s]: no property vocabulary found for entity type '%s'", assertionID, entityType))
		}
	}
	return warnings, errors
}


// getEntityType determines the type of an entity based on its ID by checking against all known entities.
func getEntityType(subjectID string, allEntities map[string]map[string]bool) string {
	if _, exists := allEntities[EntityTypePersons][subjectID]; exists {
		return "person"
	}
	if _, exists := allEntities[EntityTypeEvents][subjectID]; exists {
		return "event"
	}
	if _, exists := allEntities[EntityTypeRelationships][subjectID]; exists {
		return "relationship"
	}
	if _, exists := allEntities[EntityTypePlaces][subjectID]; exists {
		return "place"
	}
	return ""
}
