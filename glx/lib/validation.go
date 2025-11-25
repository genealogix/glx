package lib

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
}

// buildPropertyVocabMaps builds maps of property vocabularies.
func (glx *GLXFile) buildPropertyVocabMaps(result *ValidationResult) {
	result.PropertyVocabs[PropPersonProperties] = glx.PersonProperties
	result.PropertyVocabs[PropEventProperties] = glx.EventProperties
	result.PropertyVocabs[PropRelationshipProperties] = glx.RelationshipProperties
	result.PropertyVocabs[PropPlaceProperties] = glx.PlaceProperties
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
			glx.validateProperties(entityType, entityID, properties, propVocab, result)
		}
	}
}

// validateProperties validates a single `properties` map against its vocabulary.
func (glx *GLXFile) validateProperties(
	entityType, entityID string,
	properties map[string]any,
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
				Field:      "properties." + propName,
				Message:    fmt.Sprintf("%s[%s]: unknown property '%s'", entityType, entityID, propName),
			})

			continue
		}
		if propDef.ReferenceType != "" {
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
	if refID, ok := propValue.(string); ok {
		if _, exists := result.Entities[referenceType][refID]; !exists {
			result.Errors = append(result.Errors, ValidationError{
				SourceType:  entityType,
				SourceID:    entityID,
				SourceField: "properties." + propName,
				TargetType:  referenceType,
				TargetID:    refID,
				Message: fmt.Sprintf("%s[%s].properties.%s references non-existent %s: %s",
					entityType, entityID, propName, referenceType, refID),
			})
		}

		return
	}
	if valueList, ok := propValue.([]any); ok {
		for i, item := range valueList {
			if itemMap, ok := item.(map[string]any); ok {
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

// validatePropertyValue validates a property value against its value_type definition.
// For temporal properties, it accepts either a simple value OR a list of {value, date} objects.
func (glx *GLXFile) validatePropertyValue(
	entityType, entityID, propName string,
	propValue any,
	propDef *PropertyDefinition,
	result *ValidationResult,
) {
	isTemporal := propDef.Temporal != nil && *propDef.Temporal

	// Handle non-temporal properties: must be simple value
	if !isTemporal {
		// For non-temporal properties, value should NOT be a list
		if _, isList := propValue.([]any); isList {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: entityType,
				SourceID:   entityID,
				Field:      "properties." + propName,
				Message: fmt.Sprintf("%s[%s].properties.%s: non-temporal property has list value (expected simple value)",
					entityType, entityID, propName),
			})
		} else {
			// Validate the simple value against its value_type
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

	// Optional: validate 'date' field exists (temporal items should have dates, but single objects may omit it)
	if dateVal, hasDate := itemMap["date"]; hasDate {
		// Validate date format
		if dateStr, ok := dateVal.(string); ok {
			glx.validateDateFormat(entityType, entityID, fieldPath+".date", dateStr, result)
		}
	} else if index >= 0 {
		// Only warn about missing date for list items, not single objects
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: entityType,
			SourceID:   entityID,
			Field:      fieldPath + ".date",
			Message:    msgPath + ": temporal list item missing 'date' field",
		})
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

// isValidSimpleDate checks if a string matches YYYY, YYYY-MM, or YYYY-MM-DD format
func isValidSimpleDate(s string) bool {
	// YYYY (4 digits)
	if len(s) == 4 {
		for _, c := range s {
			if c < '0' || c > '9' {
				return false
			}
		}

		return true
	}

	// YYYY-MM (7 characters)
	if len(s) == 7 {
		return s[4] == '-' && isDigits(s[0:4]) && isDigits(s[5:7])
	}

	// YYYY-MM-DD (10 characters)
	if len(s) == 10 {
		return s[4] == '-' && s[7] == '-' && isDigits(s[0:4]) && isDigits(s[5:7]) && isDigits(s[8:10])
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
