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
	"strings"
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
	} else {
		// Verify warning mentions the property and entity
		warning := result.Warnings[0]
		if !strings.Contains(warning.Message, "gender") && !strings.Contains(warning.Field, "gender") {
			t.Errorf("Warning should reference 'gender' property, got: %+v", warning)
		}
		if warning.SourceType != "persons" {
			t.Errorf("Warning should have source type 'persons', got: %s", warning.SourceType)
		}
		if warning.SourceID != "person-1" {
			t.Errorf("Warning should have source ID 'person-1', got: %s", warning.SourceID)
		}
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
	} else {
		// Verify error details
		err := result.Errors[0]
		if err.SourceType != "persons" {
			t.Errorf("Error should have source type 'persons', got: %s", err.SourceType)
		}
		if err.SourceID != "person-1" {
			t.Errorf("Error should have source ID 'person-1', got: %s", err.SourceID)
		}
		if !strings.Contains(err.Message, "value") {
			t.Errorf("Error message should mention missing 'value', got: %s", err.Message)
		}
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
	} else {
		// Verify both errors have correct source info
		for i, err := range result.Errors {
			if err.SourceType != "persons" {
				t.Errorf("Error %d should have source type 'persons', got: %s", i, err.SourceType)
			}
			if err.SourceID != "person-1" {
				t.Errorf("Error %d should have source ID 'person-1', got: %s", i, err.SourceID)
			}
			if !strings.Contains(err.Message, "object") {
				t.Errorf("Error %d message should mention 'object', got: %s", i, err.Message)
			}
		}
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
							// Missing "date" field — this is valid (date is optional)
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

	// Should have 0 errors and 0 warnings — date is optional on temporal list items
	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
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

// --- Temporal consistency tests ---

func TestValidateDeathBeforeBirth(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-birth-1": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
			"event-death-1": {
				Type: EventTypeDeath, Date: "1820",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
		},
	}
	result := &ValidationResult{}
	glx.validateDeathBeforeBirth(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0].Message, "death year (1820) is before birth year (1850)") {
		t.Errorf("Unexpected message: %s", result.Warnings[0].Message)
	}
}

func TestValidateDeathBeforeBirth_NoWarningWhenValid(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-birth-1": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
			"event-death-1": {
				Type: EventTypeDeath, Date: "1920",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
		},
	}
	result := &ValidationResult{}
	glx.validateDeathBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d", len(result.Warnings))
	}
}

func TestValidateDeathBeforeBirth_NilPerson(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": nil,
		},
	}
	result := &ValidationResult{}
	glx.validateDeathBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for nil person, got %d", len(result.Warnings))
	}
}

func TestValidateParentChildAges(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"parent": {Properties: map[string]any{}},
			"child":  {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-birth-parent": {
				Type: EventTypeBirth, Date: "1880",
				Participants: []Participant{{Person: "parent", Role: ParticipantRolePrincipal}},
			},
			"event-birth-child": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "child", Role: ParticipantRolePrincipal}},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: RelationshipTypeParentChild,
				Participants: []Participant{
					{Person: "parent", Role: ParticipantRoleParent},
					{Person: "child", Role: ParticipantRoleChild},
				},
			},
		},
	}
	result := &ValidationResult{}
	glx.validateParentChildAges(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0].Message, "parent parent (born 1880) is born after child child (born 1850)") {
		t.Errorf("Unexpected message: %s", result.Warnings[0].Message)
	}
}

func TestValidateParentChildAges_NilRelationship(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"parent": {Properties: map[string]any{}},
		},
		Relationships: map[string]*Relationship{
			"rel-1": nil,
		},
	}
	result := &ValidationResult{}
	glx.validateParentChildAges(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for nil relationship, got %d", len(result.Warnings))
	}
}

func TestValidateMarriageBeforeBirth(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-birth-1": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
			"event-1": {
				Type: EventTypeMarriage,
				Date: "1840",
				Participants: []Participant{
					{Person: "person-1", Role: "spouse"},
				},
			},
		},
	}
	result := &ValidationResult{}
	glx.validateMarriageBeforeBirth(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}
	if !strings.Contains(result.Warnings[0].Message, "marriage year (1840) is before participant person-1 birth year (1850)") {
		t.Errorf("Unexpected message: %s", result.Warnings[0].Message)
	}
}

func TestValidateMarriageBeforeBirth_NilEvent(t *testing.T) {
	glx := &GLXFile{
		Events: map[string]*Event{
			"event-1": nil,
		},
	}
	result := &ValidationResult{}
	glx.validateMarriageBeforeBirth(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for nil event, got %d", len(result.Warnings))
	}
}

func TestValidateTemporalConsistency_WiredIntoValidate(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-birth-1": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
			"event-death-1": {
				Type: EventTypeDeath, Date: "1820",
				Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
			},
		},
	}
	result := glx.Validate()

	hasTemporalWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "death year") {
			hasTemporalWarning = true
			break
		}
	}
	if !hasTemporalWarning {
		t.Error("Validate() should produce temporal consistency warnings")
	}
}
