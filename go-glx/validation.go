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

import (
	"fmt"
	"reflect"
	"strconv"
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

	// Phase 4: Validate structural constraints
	glx.validatePlaceHierarchyCycles(result)

	// Phase 5: Validate entity-level field formats
	glx.validateEntityFieldFormats(result)

	// Phase 6: Temporal consistency checks
	glx.validateTemporalConsistency(result)

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
	result.Vocabularies[VocabSourceTypes] = buildIDSet(glx.SourceTypes)
	result.Vocabularies[VocabGenderTypes] = buildIDSet(glx.GenderTypes)
}

// buildPropertyVocabMaps builds maps of property vocabularies.
func (glx *GLXFile) buildPropertyVocabMaps(result *ValidationResult) {
	result.PropertyVocabs[PropPersonProperties] = glx.PersonProperties
	result.PropertyVocabs[PropEventProperties] = glx.EventProperties
	result.PropertyVocabs[PropRelationshipProperties] = glx.RelationshipProperties
	result.PropertyVocabs[PropPlaceProperties] = glx.PlaceProperties
	result.PropertyVocabs[PropMediaProperties] = glx.MediaProperties
	result.PropertyVocabs[PropRepositoryProperties] = glx.RepositoryProperties
	result.PropertyVocabs[PropCitationProperties] = glx.CitationProperties
	result.PropertyVocabs[PropSourceProperties] = glx.SourceProperties
}

// buildIDSet is a helper function that creates a set of IDs from a map[string]any.
func buildIDSet(m any) map[string]struct{} {
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
	entities any,
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
	for i := range entityVal.NumField() {
		field := entityTypeVal.Field(i)
		fieldVal := entityVal.Field(i)
		refType := field.Tag.Get("refType")
		if refType == "" {
			glx.validateNestedStructs(entityType, entityID, fieldVal, result)

			continue
		}
		glx.validateFieldReference(entityType, entityID, field.Name, fieldVal, refType, result)
	}
}

// validateNestedStructs handles recursion into nested structs and slices of structs.
func (glx *GLXFile) validateNestedStructs(entityType, entityID string, fieldVal reflect.Value, result *ValidationResult) {
	switch fieldVal.Kind() {
	case reflect.Ptr:
		if !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			glx.validateStructReferences(entityType, entityID, fieldVal.Elem(), result)
		}
	case reflect.Struct:
		// Special handling for EntityRef
		if entityRef, ok := fieldVal.Interface().(EntityRef); ok {
			glx.validateEntityRef(entityType, entityID, "Subject", entityRef, result)

			return
		}
		glx.validateStructReferences(entityType, entityID, fieldVal, result)
	case reflect.Slice:
		for i := range fieldVal.Len() {
			itemVal := fieldVal.Index(i)
			if itemVal.Kind() == reflect.Struct {
				glx.validateStructReferences(entityType, entityID, itemVal, result)
			}
		}
	}
}

// validateEntityRef validates an EntityRef, checking that exactly one field is set
// and that the referenced entity exists.
func (glx *GLXFile) validateEntityRef(entityType, entityID, fieldName string, ref EntityRef, result *ValidationResult) {
	refType := ref.Type()
	refID := ref.ID()

	if refType == "" || refID == "" {
		result.Errors = append(result.Errors, ValidationError{
			SourceType:  entityType,
			SourceID:    entityID,
			SourceField: fieldName,
			Message: fmt.Sprintf("%s[%s].%s: EntityRef must have exactly one field set",
				entityType, entityID, fieldName),
		})

		return
	}

	// Check that the referenced entity exists
	if _, exists := result.Entities[refType][refID]; !exists {
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
		for i := range fieldVal.Len() {
			if itemVal := fieldVal.Index(i); itemVal.Kind() == reflect.String {
				refID := itemVal.String()
				fieldPath := fmt.Sprintf("%s[%d]", fieldName, i)
				glx.checkReference(entityType, entityID, fieldPath, refType, refID, result)
			}
		}
	case reflect.Ptr:
		if !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Int {
			refID := strconv.FormatInt(fieldVal.Elem().Int(), 10)
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
		// Check if a hyphen/underscore swap would match an existing key,
		// and suggest the correct key if so.
		suggestion := glx.suggestReferenceKey(targetTypes, refID, result)
		msg := fmt.Sprintf("%s[%s].%s references non-existent %s: %s",
			entityType, entityID, fieldName, refType, refID)
		if suggestion != "" {
			msg += fmt.Sprintf(" (did you mean '%s'?)", suggestion)
		}
		result.Errors = append(result.Errors, ValidationError{
			SourceType:  entityType,
			SourceID:    entityID,
			SourceField: fieldName,
			TargetType:  refType,
			TargetID:    refID,
			Message:     msg,
		})
	}
}

// suggestReferenceKey checks if swapping hyphens and underscores in refID would
// match an existing vocabulary or entity key. Returns the matching key or "".
func (glx *GLXFile) suggestReferenceKey(targetTypes []string, refID string, result *ValidationResult) string {
	// Try swapping hyphens <-> underscores
	alternatives := []string{
		strings.ReplaceAll(refID, "-", "_"),
		strings.ReplaceAll(refID, "_", "-"),
	}
	for _, alt := range alternatives {
		if alt == refID {
			continue
		}
		for _, targetType := range targetTypes {
			targetType = strings.TrimSpace(targetType)
			if isVocabularyType(targetType) {
				if _, exists := result.Vocabularies[targetType][alt]; exists {
					return alt
				}
			} else {
				if _, exists := result.Entities[targetType][alt]; exists {
					return alt
				}
			}
		}
	}
	return ""
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
	glx.validateEntityProperties(EntityTypePersons, PropPersonProperties, glx.Persons, result.PropertyVocabs[PropPersonProperties], result)
	glx.validateEntityProperties(EntityTypeEvents, PropEventProperties, glx.Events, result.PropertyVocabs[PropEventProperties], result)
	glx.validateEntityProperties(EntityTypeRelationships, PropRelationshipProperties, glx.Relationships, result.PropertyVocabs[PropRelationshipProperties], result)
	glx.validateEntityProperties(EntityTypePlaces, PropPlaceProperties, glx.Places, result.PropertyVocabs[PropPlaceProperties], result)
	glx.validateEntityProperties(EntityTypeMedia, PropMediaProperties, glx.Media, result.PropertyVocabs[PropMediaProperties], result)
	glx.validateEntityProperties(EntityTypeRepositories, PropRepositoryProperties, glx.Repositories, result.PropertyVocabs[PropRepositoryProperties], result)
	glx.validateEntityProperties(EntityTypeCitations, PropCitationProperties, glx.Citations, result.PropertyVocabs[PropCitationProperties], result)
	glx.validateEntityProperties(EntityTypeSources, PropSourceProperties, glx.Sources, result.PropertyVocabs[PropSourceProperties], result)

	// Validate participant properties against the vocabulary matching their parent entity.
	eventPropVocab := result.PropertyVocabs[PropEventProperties]
	relPropVocab := result.PropertyVocabs[PropRelationshipProperties]
	glx.validateParticipantProperties(EntityTypeEvents, PropEventProperties, glx.Events, eventPropVocab, result)
	glx.validateParticipantProperties(EntityTypeRelationships, PropRelationshipProperties, glx.Relationships, relPropVocab, result)
	glx.validateAssertionParticipantProperties(eventPropVocab, result)
}

// validateAssertionParticipantProperties validates properties on assertion participants.
// Unlike events and relationships, assertions have no top-level Properties field, so
// this is the only place where "missing vocab" warnings for assertion participants
// can be emitted.
func (glx *GLXFile) validateAssertionParticipantProperties(
	propVocab map[string]*PropertyDefinition,
	result *ValidationResult,
) {
	for assertionID, assertion := range glx.Assertions {
		if assertion.Participant == nil || len(assertion.Participant.Properties) == 0 {
			continue
		}
		participantEntityID := assertionID + " participant"
		glx.validateProperties(EntityTypeAssertions, participantEntityID, PropEventProperties, assertion.Participant.Properties, propVocab, result)
	}
}

// validateParticipantProperties validates the properties field on participants
// within entities that have a Participants field, using the specified property vocabulary.
// When no vocabulary is loaded, entity-level validation already warns once per entity,
// so we skip participant-level calls to avoid duplicate "missing vocab" warnings.
func (glx *GLXFile) validateParticipantProperties(
	entityType string,
	propVocabKey string,
	entities any,
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
		participantsField := entity.FieldByName("Participants")
		if !participantsField.IsValid() {
			continue
		}
		if len(propVocab) == 0 {
			// Check if any participant has properties — if so, emit one warning per entity.
			hasParticipantProps := false
			for i := range participantsField.Len() {
				p := participantsField.Index(i)
				pf := p.FieldByName("Properties")
				if pf.IsValid() && !pf.IsNil() {
					if props, ok := pf.Interface().(map[string]any); ok && len(props) > 0 {
						hasParticipantProps = true
						break
					}
				}
			}
			// Only warn if the entity-level check won't already warn (i.e., entity has no top-level properties).
			if hasParticipantProps {
				topProps := entity.FieldByName("Properties")
				entityAlreadyWarns := false
				if topProps.IsValid() && !topProps.IsNil() {
					if props, ok := topProps.Interface().(map[string]any); ok && len(props) > 0 {
						entityAlreadyWarns = true
					}
				}
				if !entityAlreadyWarns {
					result.Warnings = append(result.Warnings, ValidationWarning{
						SourceType: entityType,
						SourceID:   entityID,
						Field:      "participants.properties",
						Message:    fmt.Sprintf("%s[%s]: has properties but no %s vocabulary was found", entityType, entityID, propVocabKey),
					})
				}
			}
			continue
		}
		for i := range participantsField.Len() {
			participant := participantsField.Index(i)
			propsField := participant.FieldByName("Properties")
			if !propsField.IsValid() || propsField.IsNil() {
				continue
			}
			if properties, ok := propsField.Interface().(map[string]any); ok {
				participantEntityID := fmt.Sprintf("%s participants[%d]", entityID, i)
				glx.validateProperties(entityType, participantEntityID, propVocabKey, properties, propVocab, result)
			}
		}
	}
}

// validateEntityProperties iterates over entities and validates their properties.
func (glx *GLXFile) validateEntityProperties(
	entityType string,
	propVocabKey string,
	entities any,
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
		if properties, ok := propsField.Interface().(map[string]any); ok {
			glx.validateProperties(entityType, entityID, propVocabKey, properties, propVocab, result)
		}
	}
}

// removedProperties maps property names that have been removed from the spec
// to human-readable migration guidance. These generate validation errors.
var removedProperties = map[string]string{
	DeprecatedPropertyBornOn: "use birth events instead",
	DeprecatedPropertyBornAt: "use birth events instead",
	DeprecatedPropertyDiedOn: "use death events instead",
	DeprecatedPropertyDiedAt: "use death events instead",
}

// validateProperties validates a single `properties` map against its vocabulary.
func (glx *GLXFile) validateProperties(
	entityType, entityID, propVocabKey string,
	properties map[string]any,
	propVocab map[string]*PropertyDefinition,
	result *ValidationResult,
) {
	// Check for removed person properties first, regardless of vocabulary presence.
	// Scoped to person entities — other entity types may legitimately use these names.
	if entityType == EntityTypePersons {
		for propName := range properties {
			if msg, removed := removedProperties[propName]; removed {
				result.Errors = append(result.Errors, ValidationError{
					SourceType:  entityType,
					SourceID:    entityID,
					SourceField: "properties." + propName,
					Message:     fmt.Sprintf("%s[%s]: property '%s' has been removed — %s. Run 'glx migrate' to convert.", entityType, entityID, propName, msg),
				})
			}
		}
	}
	if len(properties) > 0 && len(propVocab) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      "properties",
			Message:    fmt.Sprintf("%s[%s]: has properties but no %s vocabulary was found", entityType, entityID, propVocabKey),
		})

		return
	}
	for propName, propValue := range properties {
		// Skip removed properties — already handled above.
		if _, removed := removedProperties[propName]; removed {
			continue
		}
		propDef, exists := propVocab[propName]
		if !exists {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName,
				Message:    fmt.Sprintf("%s[%s]: unknown property '%s'", entityType, entityID, propName),
			})

			continue
		}
		// Check for conflicting type definitions
		typeCount := 0
		if propDef.VocabularyType != "" {
			typeCount++
		}
		if propDef.ReferenceType != "" {
			typeCount++
		}
		if propDef.ValueType != "" {
			typeCount++
		}
		if typeCount > 1 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName,
				Message: fmt.Sprintf("%s[%s].properties.%s: property definition has conflicting type fields (only one of value_type, reference_type, vocabulary_type should be set)",
					entityType, entityID, propName),
			})
		}

		if propDef.VocabularyType != "" {
			glx.validatePropertyVocabularyValue(entityType, entityID, propName, propValue, propDef, result)
		} else if propDef.ReferenceType != "" {
			glx.validatePropertyReference(entityType, entityID, propName, propValue, propDef.ReferenceType, result)
		} else if propDef.ValueType != "" {
			glx.validatePropertyValue(entityType, entityID, propName, propValue, propDef, result)
		}
	}
}

// validatePropertyReference validates a property value that is an entity reference.
func (glx *GLXFile) validatePropertyReference(
	entityType, entityID, propName string,
	propValue any,
	referenceType string,
	result *ValidationResult,
) {
	// Simple string reference
	if refID, ok := propValue.(string); ok {
		glx.checkPropertyRef(entityType, entityID, "properties."+propName, referenceType, refID, result)

		return
	}

	// Array of references (multi-value or temporal)
	if valueList, ok := propValue.([]any); ok {
		for i, item := range valueList {
			switch v := item.(type) {
			case string:
				// Multi-value: simple string reference IDs
				glx.checkPropertyRef(entityType, entityID,
					fmt.Sprintf("properties.%s[%d]", propName, i), referenceType, v, result)
			case map[string]any:
				// Temporal: {value: refID, date: ...}
				if refID, ok := v["value"].(string); ok {
					glx.checkPropertyRef(entityType, entityID,
						fmt.Sprintf("properties.%s[%d].value", propName, i), referenceType, refID, result)
				}
			}
		}
	}
}

// checkPropertyRef validates that a single property reference ID exists.
func (glx *GLXFile) checkPropertyRef(entityType, entityID, field, referenceType, refID string, result *ValidationResult) {
	if _, exists := result.Entities[referenceType][refID]; !exists {
		result.Errors = append(result.Errors, ValidationError{
			SourceType:  entityType,
			SourceID:    entityID,
			SourceField: field,
			TargetType:  referenceType,
			TargetID:    refID,
			Message: fmt.Sprintf("%s[%s].%s references non-existent %s: %s",
				entityType, entityID, field, referenceType, refID),
		})
	}
}

// validatePropertyVocabularyValue validates that a property value exists in the
// referenced vocabulary. Handles simple strings, temporal objects, and temporal lists.
func (glx *GLXFile) validatePropertyVocabularyValue(
	entityType, entityID, propName string,
	propValue any,
	propDef *PropertyDefinition,
	result *ValidationResult,
) {
	vocabSet, vocabLoaded := result.Vocabularies[propDef.VocabularyType]
	if !vocabLoaded || vocabSet == nil {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      "properties." + propName,
			Message: fmt.Sprintf("%s[%s].properties.%s: vocabulary '%s' not loaded, cannot validate value",
				entityType, entityID, propName, propDef.VocabularyType),
		})
		return
	}

	switch v := propValue.(type) {
	case string:
		glx.checkVocabValue(entityType, entityID, "properties."+propName, propDef.VocabularyType, v, vocabSet, result)
	case map[string]any:
		// Single temporal object: {value: ..., date: ...}
		rawVal, hasValue := v["value"]
		if !hasValue {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName,
				Message: fmt.Sprintf("%s[%s].properties.%s: vocabulary-constrained temporal object missing 'value' field",
					entityType, entityID, propName),
			})
		} else if val, ok := rawVal.(string); ok {
			glx.checkVocabValue(entityType, entityID, "properties."+propName+".value", propDef.VocabularyType, val, vocabSet, result)
		} else {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName + ".value",
				Message: fmt.Sprintf("%s[%s].properties.%s.value: expected string for vocabulary lookup, got %T",
					entityType, entityID, propName, rawVal),
			})
		}
	case []any:
		// List of values: strings, temporal objects, or mixed
		for i, item := range v {
			switch typedItem := item.(type) {
			case string:
				fieldPath := fmt.Sprintf("properties.%s[%d]", propName, i)
				glx.checkVocabValue(entityType, entityID, fieldPath, propDef.VocabularyType, typedItem, vocabSet, result)
			case map[string]any:
				rawVal, hasValue := typedItem["value"]
				if !hasValue {
					result.Warnings = append(result.Warnings, ValidationWarning{
						SourceType: entityType,
						SourceID:   entityID,
						Field:      fmt.Sprintf("properties.%s[%d]", propName, i),
						Message: fmt.Sprintf("%s[%s].properties.%s[%d]: vocabulary-constrained temporal object missing 'value' field",
							entityType, entityID, propName, i),
					})
				} else if val, ok := rawVal.(string); ok {
					fieldPath := fmt.Sprintf("properties.%s[%d].value", propName, i)
					glx.checkVocabValue(entityType, entityID, fieldPath, propDef.VocabularyType, val, vocabSet, result)
				} else {
					result.Warnings = append(result.Warnings, ValidationWarning{
						SourceType: entityType,
						SourceID:   entityID,
						Field:      fmt.Sprintf("properties.%s[%d].value", propName, i),
						Message: fmt.Sprintf("%s[%s].properties.%s[%d].value: expected string for vocabulary lookup, got %T",
							entityType, entityID, propName, i, rawVal),
					})
				}
			default:
				result.Warnings = append(result.Warnings, ValidationWarning{
					SourceType: entityType,
					SourceID:   entityID,
					Field:      fmt.Sprintf("properties.%s[%d]", propName, i),
					Message: fmt.Sprintf("%s[%s].properties.%s[%d]: expected string or temporal object in vocabulary-constrained list, got %T",
						entityType, entityID, propName, i, item),
				})
			}
		}
	default:
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      "properties." + propName,
			Message: fmt.Sprintf("%s[%s].properties.%s: expected string value for vocabulary lookup, got %T",
				entityType, entityID, propName, propValue),
		})
	}
}

// checkVocabValue checks that a single value exists in the given vocabulary.
func (glx *GLXFile) checkVocabValue(
	entityType, entityID, field, vocabType, value string,
	vocabSet map[string]struct{},
	result *ValidationResult,
) {
	if _, exists := vocabSet[value]; !exists {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      field,
			Message: fmt.Sprintf("%s[%s].%s: value '%s' not found in %s vocabulary",
				entityType, entityID, field, value, vocabType),
		})
	}
}

// validatePropertyValue validates a property value against its value_type definition.
// For temporal properties, it accepts either a simple value OR a list of {value, date} objects.
func (glx *GLXFile) validatePropertyValue(
	entityType, entityID, propName string,
	propValue any,
	propDef *PropertyDefinition,
	result *ValidationResult,
) {
	isTemporal := propDef.Temporal != nil && *propDef.Temporal
	isMultiValue := propDef.MultiValue != nil && *propDef.MultiValue

	// Handle multi-value properties: can be array of simple values or structured objects
	if isMultiValue {
		if listVal, isList := propValue.([]any); isList {
			for i, item := range listVal {
				fieldPath := fmt.Sprintf("properties.%s[%d]", propName, i)
				if structuredVal, isMap := item.(map[string]any); isMap {
					glx.validateStructuredValue(entityType, entityID, propName+"["+fmt.Sprint(i)+"]", structuredVal, propDef, result)
				} else {
					glx.validateValueType(entityType, entityID, fieldPath, item, propDef.ValueType, result)
				}
			}
		} else {
			// Single value is also allowed for multi-value properties
			glx.validateValueType(entityType, entityID, "properties."+propName, propValue, propDef.ValueType, result)
		}

		return
	}

	// Handle non-temporal, non-multi-value properties: simple value or structured {value, fields}
	if !isTemporal {
		if _, isList := propValue.([]any); isList {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName,
				Message: fmt.Sprintf("%s[%s].properties.%s: non-temporal property has list value (expected simple value or use multi_value: true)",
					entityType, entityID, propName),
			})
		} else if structuredVal, isMap := propValue.(map[string]any); isMap {
			// Structured value: {value: ..., fields: {...}} — validate inner value and fields
			glx.validateStructuredValue(entityType, entityID, propName, structuredVal, propDef, result)
		} else {
			glx.validateValueType(entityType, entityID, "properties."+propName, propValue, propDef.ValueType, result)
		}

		return
	}

	// Handle temporal properties: accept simple value, single {value, date, fields} object, OR list
	switch v := propValue.(type) {
	case string, float64, int, bool:
		// Simple value is fine for temporal properties - validate it
		glx.validateValueType(entityType, entityID, "properties."+propName, v, propDef.ValueType, result)

		return

	case map[string]any:
		// Single object with value (and optional date/fields) - validate it
		glx.validateTemporalItem(entityType, entityID, propName, -1, v, propDef, result)

		return
	}

	// Must be a list for temporal properties with complex values
	valueList, isList := propValue.([]any)
	if !isList {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      "properties." + propName,
			Message: fmt.Sprintf("%s[%s].properties.%s: temporal property has invalid value type (expected simple value, {value, date} object, or list of such objects)",
				entityType, entityID, propName),
		})

		return
	}

	// Validate each item in the temporal list has {value, date} structure
	for i, item := range valueList {
		itemMap, isMap := item.(map[string]any)
		if !isMap {
			result.Errors = append(result.Errors, ValidationError{
				SourceType:  entityType,
				SourceID:    entityID,
				SourceField: fmt.Sprintf("properties.%s[%d]", propName, i),
				Message: fmt.Sprintf("%s[%s].properties.%s[%d]: temporal list item must be an object with 'value' field",
					entityType, entityID, propName, i),
			})

			continue
		}

		glx.validateTemporalItem(entityType, entityID, propName, i, itemMap, propDef, result)
	}
}

// validateStructuredValue validates a structured property value of the form
// {value: ..., fields: {...}}. This covers non-temporal properties that use the
// structured format (e.g., name with given/surname fields).
func (glx *GLXFile) validateStructuredValue(
	entityType, entityID, propName string,
	structuredVal map[string]any,
	propDef *PropertyDefinition,
	result *ValidationResult,
) {
	fieldPath := "properties." + propName

	// Validate the inner 'value' field if present
	if value, hasValue := structuredVal["value"]; hasValue && propDef.ValueType != "" {
		glx.validateValueType(entityType, entityID, fieldPath+".value", value, propDef.ValueType, result)
	}

	// Validate fields if present and property definition has fields schema
	if fields, hasFields := structuredVal["fields"]; hasFields && propDef.Fields != nil {
		glx.validateTemporalFields(entityType, entityID, fieldPath, fields, propDef.Fields, result)
	}
}

// validateTemporalItem validates a single temporal item (object with value, optional date and fields).
// index is -1 for a single object, or >= 0 for list items.
func (glx *GLXFile) validateTemporalItem(
	entityType, entityID, propName string,
	index int,
	itemMap map[string]any,
	propDef *PropertyDefinition,
	result *ValidationResult,
) {
	// Build field path based on whether this is a list item or single object
	fieldPath := "properties." + propName
	msgPath := fmt.Sprintf("%s[%s].properties.%s", entityType, entityID, propName)
	if index >= 0 {
		fieldPath = fmt.Sprintf("properties.%s[%d]", propName, index)
		msgPath = fmt.Sprintf("%s[%s].properties.%s[%d]", entityType, entityID, propName, index)
	}

	// Check for required 'value' field
	if _, hasValue := itemMap["value"]; !hasValue {
		result.Errors = append(result.Errors, ValidationError{
			SourceType:  entityType,
			SourceID:    entityID,
			SourceField: fieldPath + ".value",
			Message:     msgPath + ": temporal item missing required 'value' field",
		})
	}

	// Validate 'date' field format if present (date is optional on temporal items)
	if dateVal, hasDate := itemMap["date"]; hasDate {
		if dateStr, ok := dateVal.(string); ok {
			glx.validateDateFormat(entityType, entityID, fieldPath+".date", dateStr, result)
		} else {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      fieldPath + ".date",
				Message: fmt.Sprintf("%s: date value should be a quoted string, got %T (use \"1850\" not 1850)",
					msgPath, dateVal),
			})
		}
	}

	// Validate the value field against the property's value_type
	if value, hasValue := itemMap["value"]; hasValue && propDef.ValueType != "" {
		glx.validateValueType(entityType, entityID, fieldPath+".value", value, propDef.ValueType, result)
	}

	// Validate fields if present and property definition has fields schema
	if fields, hasFields := itemMap["fields"]; hasFields && propDef.Fields != nil {
		glx.validateTemporalFields(entityType, entityID, fieldPath, fields, propDef.Fields, result)
	}
}

// validateTemporalFields validates the fields of a structured temporal property.
func (glx *GLXFile) validateTemporalFields(
	entityType, entityID, fieldPath string,
	fields any,
	fieldDefs map[string]*FieldDefinition,
	result *ValidationResult,
) {
	fieldsMap, ok := fields.(map[string]any)
	if !ok {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      fieldPath + ".fields",
			Message:    fmt.Sprintf("%s[%s].%s.fields: expected object, got %T", entityType, entityID, fieldPath, fields),
		})

		return
	}

	// Warn about unknown fields
	for fieldName := range fieldsMap {
		if _, known := fieldDefs[fieldName]; !known {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      fieldPath + ".fields." + fieldName,
				Message:    fmt.Sprintf("%s[%s].%s.fields.%s: unknown field", entityType, entityID, fieldPath, fieldName),
			})
		}
	}
}

// validatePlaceHierarchyCycles detects cycles in place parent references.
// For each place, it walks the parent chain and reports an error if the chain
// loops back to a previously visited place. Each cycle is reported exactly once.
func (glx *GLXFile) validatePlaceHierarchyCycles(result *ValidationResult) {
	if len(glx.Places) == 0 {
		return
	}

	verified := make(map[string]struct{})
	for placeID := range glx.Places {
		if _, done := verified[placeID]; done {
			continue
		}

		var path []string
		visited := make(map[string]int) // placeID -> index in path
		current := placeID
		for {
			if _, done := verified[current]; done {
				break
			}
			if idx, inPath := visited[current]; inPath {
				cycleMembers := path[idx:]
				result.Errors = append(result.Errors, ValidationError{
					SourceType:  EntityTypePlaces,
					SourceID:    cycleMembers[0],
					SourceField: "parent",
					TargetType:  EntityTypePlaces,
					TargetID:    current,
					Message: fmt.Sprintf("places: place hierarchy cycle detected: %s -> %s",
						strings.Join(cycleMembers, " -> "), current),
				})

				break
			}

			place, exists := glx.Places[current]
			if !exists || place.ParentID == "" {
				break
			}

			visited[current] = len(path)
			path = append(path, current)
			current = place.ParentID
		}

		// Mark all path nodes as verified to avoid duplicate reports.
		for _, id := range path {
			verified[id] = struct{}{}
		}
	}
}

// validateDateFormat validates a date string against GENEALOGIX date format.
// GENEALOGIX uses FamilySearch-style keywords (FROM, TO, ABT, BEF, AFT, BET, AND, CAL, INT)
// combined with ISO 8601-style dates (YYYY, YYYY-MM, YYYY-MM-DD).
func (glx *GLXFile) validateDateFormat(entityType, entityID, field, dateStr string, result *ValidationResult) {
	if dateStr == "" {
		return // Empty dates are allowed
	}

	// Check for FamilySearch-style keywords
	dateStr = strings.TrimSpace(dateStr)

	// Valid patterns:
	// - Simple: YYYY, YYYY-MM, YYYY-MM-DD
	// - FROM YYYY TO YYYY
	// - FROM YYYY
	// - ABT YYYY, BEF YYYY, AFT YYYY
	// - BET YYYY AND YYYY
	// - CAL YYYY
	// - INT YYYY (original)

	// Simple validation: check for known keywords or ISO 8601-ish format
	hasKeyword := strings.Contains(dateStr, "FROM") ||
		strings.Contains(dateStr, "TO") ||
		strings.Contains(dateStr, "ABT") ||
		strings.Contains(dateStr, "BEF") ||
		strings.Contains(dateStr, "AFT") ||
		strings.Contains(dateStr, "BET") ||
		strings.Contains(dateStr, "AND") ||
		strings.Contains(dateStr, "CAL") ||
		strings.Contains(dateStr, "INT")

	// If no keywords, validate as simple ISO 8601-style date
	if !hasKeyword {
		if !isValidSimpleDate(dateStr) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: date '%s' should be in format YYYY, YYYY-MM, or YYYY-MM-DD, or use keywords like FROM, TO, ABT, BEF, AFT, BET, CAL, INT",
					entityType, entityID, field, dateStr),
			})
		}
	}
	// If keywords present, assume valid (detailed parsing is complex and deferred)
}

// isValidSimpleDate checks if a string matches a simple date with a 1-4 digit year:
// year only (Y to YYYY), year-month (Y-MM to YYYY-MM), or year-month-day (Y-MM-DD to YYYY-MM-DD).
func isValidSimpleDate(s string) bool {
	// Find the separator position for year-month and year-month-day forms.
	// The year portion is everything before the first '-'.
	dashIdx := strings.Index(s, "-")

	// Year only: 1-4 digits with no dash present.
	if dashIdx == -1 {
		if len(s) >= 1 && len(s) <= 4 { //nolint:mnd // year is 1-4 digits
			return isDigits(s)
		}

		return false
	}

	if dashIdx < 1 || dashIdx > 4 { //nolint:mnd // year portion is 1-4 digits
		return false
	}

	yearPart := s[:dashIdx]
	rest := s[dashIdx+1:]

	if !isDigits(yearPart) {
		return false
	}

	// YYYY-MM: rest is exactly 2 digits
	if len(rest) == 2 { //nolint:mnd // MM is 2 digits
		return isDigits(rest)
	}

	// YYYY-MM-DD: rest is MM-DD (5 characters)
	if len(rest) == 5 { //nolint:mnd // MM-DD is 5 characters
		return rest[2] == '-' && isDigits(rest[0:2]) && isDigits(rest[3:5])
	}

	return false
}

// isDigits checks if all characters in a string are digits
func isDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return len(s) > 0
}

// validateValueType validates a value against its declared value_type
func (glx *GLXFile) validateValueType(entityType, entityID, field string, value any, valueType string, result *ValidationResult) {
	switch valueType {
	case "string":
		if _, ok := value.(string); !ok {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: expected string value, got %T",
					entityType, entityID, field, value),
			})
		}

	case "integer":
		// YAML parses integers as int or float64
		switch value.(type) {
		case int, int64, int32, float64:
			// Valid integer representation
		default:
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: expected integer value, got %T",
					entityType, entityID, field, value),
			})
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: expected boolean value, got %T",
					entityType, entityID, field, value),
			})
		}

	case "date":
		if dateStr, ok := value.(string); ok {
			glx.validateDateFormat(entityType, entityID, field, dateStr, result)
		} else {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: date value should be a string, got %T",
					entityType, entityID, field, value),
			})
		}
	}
}

// validateEntityFieldFormats performs additional lightweight validation of format
// constraints on top-level entity fields beyond what is currently enforced via schema.
func (glx *GLXFile) validateEntityFieldFormats(result *ValidationResult) {
	// Validate date fields on events, sources, media, and assertions
	for id, event := range glx.Events {
		if event.Date != "" {
			glx.validateDateFormat(EntityTypeEvents, id, "date", string(event.Date), result)
		}
	}
	for id, source := range glx.Sources {
		if source.Date != "" {
			glx.validateDateFormat(EntityTypeSources, id, "date", string(source.Date), result)
		}
	}
	for id, media := range glx.Media {
		if media.Date != "" {
			glx.validateDateFormat(EntityTypeMedia, id, "date", string(media.Date), result)
		}
	}
	for id, assertion := range glx.Assertions {
		if assertion.Date != "" {
			glx.validateDateFormat(EntityTypeAssertions, id, "date", string(assertion.Date), result)
		}
	}

	// Validate repository website URLs
	for id, repo := range glx.Repositories {
		if repo.Website != "" && !isValidURL(repo.Website) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeRepositories,
				SourceID:   id,
				Field:      "website",
				Message: fmt.Sprintf("%s[%s].website: '%s' does not appear to be a valid URL",
					EntityTypeRepositories, id, repo.Website),
			})
		}
	}

	// Validate media URI and MIME type formats
	for id, media := range glx.Media {
		if media.URI != "" && !isValidMediaURI(media.URI) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeMedia,
				SourceID:   id,
				Field:      "uri",
				Message: fmt.Sprintf("%s[%s].uri: '%s' does not appear to be a valid URI or relative path",
					EntityTypeMedia, id, media.URI),
			})
		}
		if media.MimeType != "" && !isValidMIMEType(media.MimeType) {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeMedia,
				SourceID:   id,
				Field:      "mime_type",
				Message: fmt.Sprintf("%s[%s].mime_type: '%s' does not follow standard MIME type format (type/subtype)",
					EntityTypeMedia, id, media.MimeType),
			})
		}
	}
}

// isValidURL checks if a string looks like a valid URL (starts with http:// or https://).
func isValidURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// isValidMediaURI checks if a string looks like a valid URI or relative file path.
func isValidMediaURI(s string) bool {
	// Accept URLs
	if strings.Contains(s, "://") {
		return true
	}
	// Accept relative paths (must not be empty, already checked before calling)
	// Reject strings that are clearly not paths (e.g., just whitespace)
	return strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\n\r\t")
}

// isValidMIMEType checks if a string follows the standard type/subtype MIME format.
func isValidMIMEType(s string) bool {
	parts := strings.SplitN(s, "/", 2)
	return len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0
}
