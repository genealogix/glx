package lib

import (
	"fmt"
	"reflect"
	"strings"
)

// Validate performs a comprehensive validation of the archive's cross-references.
// The result is cached; subsequent calls will return the same result unless
// InvalidateCache is called.
func (glx *GLXFile) Validate() *ValidationResult {
	if glx.validation != nil && glx.validation.validated {
		return glx.validation // Return cached result
	}

	result := &ValidationResult{
		Entities:       make(map[string]map[string]struct{}),
		Vocabularies:   make(map[string]map[string]struct{}),
		PropertyVocabs: make(map[string]map[string]*PropertyDefinition),
	}

	// Phase 1: Build existence maps
	glx.buildEntityMaps(result)
	glx.buildVocabularyMaps(result)
	glx.buildPropertyVocabMaps(result)

	// Phase 2: Validate all references using reflection
	glx.validateAllReferences(result)

	// Phase 3: Validate entity properties
	glx.validateAllProperties(result)

	result.validated = true
	glx.validation = result
	return result
}

// InvalidateCache clears the cached validation results.
// This should be called after any modification to the GLXFile struct.
func (glx *GLXFile) InvalidateCache() {
	glx.validation = nil
}

// buildEntityMaps builds maps of all entity IDs for quick lookup.
func (glx *GLXFile) buildEntityMaps(result *ValidationResult) {
	result.Entities[EntityTypePersons] = buildIDSet(glx.Persons)
	result.Entities[EntityTypeEvents] = buildIDSet(glx.Events)
	result.Entities[EntityTypeRelationships] = buildIDSet(glx.Relationships)
	result.Entities[EntityTypePlaces] = buildIDSet(glx.Places)
	result.Entities[EntityTypeSources] = buildIDSet(glx.Sources)
	result.Entities[EntityTypeCitations] = buildIDSet(glx.Citations)
	result.Entities[EntityTypeRepositories] = buildIDSet(glx.Repositories)
	result.Entities[EntityTypeAssertions] = buildIDSet(glx.Assertions)
	result.Entities[EntityTypeMedia] = buildIDSet(glx.Media)
}

// buildVocabularyMaps builds maps of all vocabulary values for quick lookup.
func (glx *GLXFile) buildVocabularyMaps(result *ValidationResult) {
	result.Vocabularies[VocabEventTypes] = buildIDSet(glx.EventTypes)
	result.Vocabularies[VocabRelationshipTypes] = buildIDSet(glx.RelationshipTypes)
	result.Vocabularies[VocabPlaceTypes] = buildIDSet(glx.PlaceTypes)
	result.Vocabularies[VocabRepositoryTypes] = buildIDSet(glx.RepositoryTypes)
	result.Vocabularies[VocabParticipantRoles] = buildIDSet(glx.ParticipantRoles)
	result.Vocabularies[VocabMediaTypes] = buildIDSet(glx.MediaTypes)
	result.Vocabularies[VocabConfidenceLevels] = buildIDSet(glx.ConfidenceLevels)
	result.Vocabularies[VocabQualityRatings] = buildIDSet(glx.QualityRatings)
	result.Vocabularies[VocabSourceTypes] = buildIDSet(glx.SourceTypes)
}

// buildPropertyVocabMaps builds maps of property vocabularies.
func (glx *GLXFile) buildPropertyVocabMaps(result *ValidationResult) {
	result.PropertyVocabs[PropPersonProperties] = glx.PersonProperties
	result.PropertyVocabs[PropEventProperties] = glx.EventProperties
	result.PropertyVocabs[PropRelationshipProperties] = glx.RelationshipProperties
	result.PropertyVocabs[PropPlaceProperties] = glx.PlaceProperties
}

// buildIDSet is a helper function that creates a set of IDs from a map[string]interface{}.
func buildIDSet(m interface{}) map[string]struct{} {
	set := make(map[string]struct{})
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		return set
	}
	for _, key := range val.MapKeys() {
		set[key.String()] = struct{}{}
	}
	return set
}

// validateAllReferences uses reflection to validate all refType tags.
func (glx *GLXFile) validateAllReferences(result *ValidationResult) {
	glx.validateEntityTypeReferences(EntityTypePersons, glx.Persons, result)
	glx.validateEntityTypeReferences(EntityTypeEvents, glx.Events, result)
	glx.validateEntityTypeReferences(EntityTypeRelationships, glx.Relationships, result)
	glx.validateEntityTypeReferences(EntityTypePlaces, glx.Places, result)
	glx.validateEntityTypeReferences(EntityTypeSources, glx.Sources, result)
	glx.validateEntityTypeReferences(EntityTypeCitations, glx.Citations, result)
	glx.validateEntityTypeReferences(EntityTypeRepositories, glx.Repositories, result)
	glx.validateEntityTypeReferences(EntityTypeAssertions, glx.Assertions, result)
	glx.validateEntityTypeReferences(EntityTypeMedia, glx.Media, result)
}

// validateEntityTypeReferences validates all entities of a given type.
func (glx *GLXFile) validateEntityTypeReferences(
	entityType string,
	entities interface{},
	result *ValidationResult,
) {
	entitiesVal := reflect.ValueOf(entities)
	if entitiesVal.Kind() != reflect.Map {
		return
	}
	for _, key := range entitiesVal.MapKeys() {
		entityID := key.String()
		entity := entitiesVal.MapIndex(key)
		glx.validateStructReferences(entityType, entityID, entity.Elem(), result)
	}
}

// validateStructReferences recursively validates all refType tags in a struct.
func (glx *GLXFile) validateStructReferences(
	entityType, entityID string,
	entityVal reflect.Value,
	result *ValidationResult,
) {
	if entityVal.Kind() != reflect.Struct {
		return
	}
	entityTypeVal := entityVal.Type()
	for i := 0; i < entityVal.NumField(); i++ {
		field := entityTypeVal.Field(i)
		fieldVal := entityVal.Field(i)
		refType := field.Tag.Get("refType")
		if refType == "" {
			glx.validateNestedStructs(entityType, entityID, field.Name, fieldVal, result)
			continue
		}
		glx.validateFieldReference(entityType, entityID, field.Name, fieldVal, refType, result)
	}
}

// validateNestedStructs handles recursion into nested structs and slices of structs.
func (glx *GLXFile) validateNestedStructs(entityType, entityID, fieldName string, fieldVal reflect.Value, result *ValidationResult) {
	switch fieldVal.Kind() {
	case reflect.Ptr:
		if !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			glx.validateStructReferences(entityType, entityID, fieldVal.Elem(), result)
		}
	case reflect.Struct:
		glx.validateStructReferences(entityType, entityID, fieldVal, result)
	case reflect.Slice:
		for i := 0; i < fieldVal.Len(); i++ {
			itemVal := fieldVal.Index(i)
			if itemVal.Kind() == reflect.Struct {
				glx.validateStructReferences(entityType, entityID, itemVal, result)
			}
		}
	}
}

// validateFieldReference validates a field with a refType tag.
func (glx *GLXFile) validateFieldReference(
	entityType, entityID, fieldName string,
	fieldVal reflect.Value,
	refType string,
	result *ValidationResult,
) {
	switch fieldVal.Kind() {
	case reflect.String:
		refID := fieldVal.String()
		if refID != "" {
			glx.checkReference(entityType, entityID, fieldName, refType, refID, result)
		}
	case reflect.Slice:
		for i := 0; i < fieldVal.Len(); i++ {
			if itemVal := fieldVal.Index(i); itemVal.Kind() == reflect.String {
				refID := itemVal.String()
				fieldPath := fmt.Sprintf("%s[%d]", fieldName, i)
				glx.checkReference(entityType, entityID, fieldPath, refType, refID, result)
			}
		}
	case reflect.Ptr:
		if !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Int {
			refID := fmt.Sprintf("%d", fieldVal.Elem().Int())
			glx.checkReference(entityType, entityID, fieldName, refType, refID, result)
		}
	}
}

// checkReference validates that a single referenced ID exists.
func (glx *GLXFile) checkReference(
	entityType, entityID, fieldName, refType, refID string,
	result *ValidationResult,
) {
	targetTypes := strings.Split(refType, ",")
	found := false
	for _, targetType := range targetTypes {
		targetType = strings.TrimSpace(targetType)
		if isVocabularyType(targetType) {
			if _, exists := result.Vocabularies[targetType][refID]; exists {
				found = true
				break
			}
		} else {
			if _, exists := result.Entities[targetType][refID]; exists {
				found = true
				break
			}
		}
	}
	if !found {
		result.Errors = append(result.Errors, ValidationError{
			SourceType:  entityType,
			SourceID:    entityID,
			SourceField: fieldName,
			TargetType:  refType,
			TargetID:    refID,
			Message: fmt.Sprintf("%s[%s].%s references non-existent %s: %s",
				entityType, entityID, fieldName, refType, refID),
		})
	}
}

// isVocabularyType is a helper to distinguish vocabulary types from entity types.
func isVocabularyType(typeName string) bool {
	return strings.HasSuffix(typeName, "_types") ||
		strings.HasSuffix(typeName, "_levels") ||
		strings.HasSuffix(typeName, "_roles") ||
		strings.HasSuffix(typeName, "_ratings") ||
		strings.HasSuffix(typeName, "_properties")
}

// validateAllProperties validates the `properties` field on all relevant entities.
func (glx *GLXFile) validateAllProperties(result *ValidationResult) {
	glx.validateEntityProperties(EntityTypePersons, glx.Persons, result.PropertyVocabs[PropPersonProperties], result)
	glx.validateEntityProperties(EntityTypeEvents, glx.Events, result.PropertyVocabs[PropEventProperties], result)
	glx.validateEntityProperties(EntityTypeRelationships, glx.Relationships, result.PropertyVocabs[PropRelationshipProperties], result)
	glx.validateEntityProperties(EntityTypePlaces, glx.Places, result.PropertyVocabs[PropPlaceProperties], result)
}

// validateEntityProperties iterates over entities and validates their properties.
func (glx *GLXFile) validateEntityProperties(
	entityType string,
	entities interface{},
	propVocab map[string]*PropertyDefinition,
	result *ValidationResult,
) {
	entitiesVal := reflect.ValueOf(entities)
	if entitiesVal.Kind() != reflect.Map {
		return
	}
	for _, key := range entitiesVal.MapKeys() {
		entityID := key.String()
		entity := entitiesVal.MapIndex(key).Elem()
		propsField := entity.FieldByName("Properties")
		if !propsField.IsValid() || propsField.IsNil() {
			continue
		}
		if properties, ok := propsField.Interface().(map[string]interface{}); ok {
			glx.validateProperties(entityType, entityID, properties, propVocab, result)
		}
	}
}

// validateProperties validates a single `properties` map against its vocabulary.
func (glx *GLXFile) validateProperties(
	entityType, entityID string,
	properties map[string]interface{},
	propVocab map[string]*PropertyDefinition,
	result *ValidationResult,
) {
	if len(properties) > 0 && len(propVocab) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      "properties",
			Message:    fmt.Sprintf("%s[%s]: has properties but no %s_properties vocabulary was found", entityType, entityID, entityType),
		})
		return
	}
	for propName, propValue := range properties {
		propDef, exists := propVocab[propName]
		if !exists {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      fmt.Sprintf("properties.%s", propName),
				Message:    fmt.Sprintf("%s[%s]: unknown property '%s'", entityType, entityID, propName),
			})
			continue
		}
		if propDef.ReferenceType != "" {
			glx.validatePropertyReference(entityType, entityID, propName, propValue, propDef.ReferenceType, result)
		}
	}
}

// validatePropertyReference validates a property value that is an entity reference.
func (glx *GLXFile) validatePropertyReference(
	entityType, entityID, propName string,
	propValue interface{},
	referenceType string,
	result *ValidationResult,
) {
	if refID, ok := propValue.(string); ok {
		if _, exists := result.Entities[referenceType][refID]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				SourceType:  entityType,
				SourceID:    entityID,
				SourceField: fmt.Sprintf("properties.%s", propName),
				TargetType:  referenceType,
				TargetID:    refID,
				Message: fmt.Sprintf("%s[%s].properties.%s references non-existent %s: %s",
					entityType, entityID, propName, referenceType, refID),
			})
		}
		return
	}
	if valueList, ok := propValue.([]interface{}); ok {
		for i, item := range valueList {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if refID, ok := itemMap["value"].(string); ok {
					if _, exists := result.Entities[referenceType][refID]; !exists {
						result.Errors = append(result.Errors, ValidationError{
							SourceType:  entityType,
							SourceID:    entityID,
							SourceField: fmt.Sprintf("properties.%s[%d].value", propName, i),
							TargetType:  referenceType,
							TargetID:    refID,
							Message: fmt.Sprintf("%s[%s].properties.%s[%d].value references non-existent %s: %s",
								entityType, entityID, propName, i, referenceType, refID),
						})
					}
				}
			}
		}
	}
}
