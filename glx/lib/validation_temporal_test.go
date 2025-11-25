package lib

import (
	"testing"
)

func TestValidatePropertyValue_NonTemporalWithList(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"gender": []any{
						map[string]any{
							"value": "male",
							"date":  "1850",
						},
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Gender",
		Description: "Gender identity",
		ValueType:   "string",
		Temporal:    boolPtr(false), // Non-temporal
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "gender", glx.Persons["person-1"].Properties["gender"], propDef, result)

	// Should have 1 warning: non-temporal property with list value
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(result.Errors))
	}
}

func TestValidatePropertyValue_TemporalListMissingValue(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"name": []any{
						map[string]any{
							"date": "1850",
							// Missing "value" field
						},
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Name",
		Description: "Person's name",
		ValueType:   "string",
		Temporal:    boolPtr(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "name", glx.Persons["person-1"].Properties["name"], propDef, result)

	// Should have 1 error: temporal list item missing 'value' field
	// Should have 0 warnings (date is present)
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidatePropertyValue_TemporalListNotObject(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"name": []any{
						"Smith",
						"Jones",
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Name",
		Description: "Person's name",
		ValueType:   "string",
		Temporal:    boolPtr(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "name", glx.Persons["person-1"].Properties["name"], propDef, result)

	// Should have 2 errors: both items are not objects
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(result.Errors), result.Errors)
	}
}

func TestValidatePropertyValue_TemporalListMissingDate(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"occupation": []any{
						map[string]any{
							"value": "blacksmith",
							// Missing "date" field
						},
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Occupation",
		Description: "Occupation",
		ValueType:   "string",
		Temporal:    boolPtr(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "occupation", glx.Persons["person-1"].Properties["occupation"], propDef, result)

	// Should have 0 errors (value is present)
	// Should have 1 warning: temporal list item missing 'date' field
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidatePropertyValue_TemporalSimpleValue(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"name": "John Smith",
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Name",
		Description: "Person's name",
		ValueType:   "string",
		Temporal:    boolPtr(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "name", glx.Persons["person-1"].Properties["name"], propDef, result)

	// Should have no errors or warnings - simple values are valid for temporal properties
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidatePropertyValue_TemporalValidList(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"occupation": []any{
						map[string]any{
							"value": "blacksmith",
							"date":  "1870",
						},
						map[string]any{
							"value": "farmer",
							"date":  "FROM 1880 TO 1920",
						},
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:       "Occupation",
		Description: "Occupation",
		ValueType:   "string",
		Temporal:    boolPtr(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "occupation", glx.Persons["person-1"].Properties["occupation"], propDef, result)

	// Should have no errors or warnings - this is a valid temporal list
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}
