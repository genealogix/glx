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
		Temporal:    new(false), // Non-temporal
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
		if warning.SourceType != EntityTypePersons {
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
		Temporal:    new(true),
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
		if err.SourceType != EntityTypePersons {
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
		Temporal:    new(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "name", glx.Persons["person-1"].Properties["name"], propDef, result)

	// Should have 2 errors: both items are not objects
	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d: %v", len(result.Errors), result.Errors)
	} else {
		// Verify both errors have correct source info
		for i, err := range result.Errors {
			if err.SourceType != EntityTypePersons {
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
		Temporal:    new(true),
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
		Temporal:    new(true),
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
		Temporal:    new(true),
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

// Pin value_type enforcement on the simple-value temporal path. Regression test for #668.
func TestValidatePropertyValue_TemporalSimpleValueTypeMismatch(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"birth_year": "not-a-number",
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:     "Birth Year",
		ValueType: "integer",
		Temporal:  new(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "birth_year", glx.Persons["person-1"].Properties["birth_year"], propDef, result)

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "expected integer value, got string") {
		t.Errorf("Warning should report integer/string mismatch, got: %s", result.Warnings[0].Message)
	}
	if result.Warnings[0].Field != "properties.birth_year" {
		t.Errorf("Warning field should be 'properties.birth_year', got: %s", result.Warnings[0].Field)
	}
}

// Pin value_type enforcement on the single-object temporal path. Regression test for #668.
func TestValidatePropertyValue_TemporalSingleObjectValueTypeMismatch(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"birth_year": map[string]any{
						"value": "not-a-number",
						"date":  "1850",
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:     "Birth Year",
		ValueType: "integer",
		Temporal:  new(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "birth_year", glx.Persons["person-1"].Properties["birth_year"], propDef, result)

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "expected integer value, got string") {
		t.Errorf("Warning should report integer/string mismatch, got: %s", result.Warnings[0].Message)
	}
	if result.Warnings[0].Field != "properties.birth_year.value" {
		t.Errorf("Warning field should be 'properties.birth_year.value', got: %s", result.Warnings[0].Field)
	}
}

// Pin value_type enforcement on the list-of-objects temporal path. Mirrors the
// exact YAML scenario from #668.
func TestValidatePropertyValue_TemporalListItemValueTypeMismatch(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"birth_year": []any{
						map[string]any{
							"value": "not-a-number",
							"date":  "1850",
						},
					},
				},
			},
		},
	}

	propDef := &PropertyDefinition{
		Label:     "Birth Year",
		ValueType: "integer",
		Temporal:  new(true),
	}

	result := &ValidationResult{}
	glx.validatePropertyValue("persons", "person-1", "birth_year", glx.Persons["person-1"].Properties["birth_year"], propDef, result)

	if len(result.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0].Message, "expected integer value, got string") {
		t.Errorf("Warning should report integer/string mismatch, got: %s", result.Warnings[0].Message)
	}
	if result.Warnings[0].Field != "properties.birth_year[0].value" {
		t.Errorf("Warning field should be 'properties.birth_year[0].value', got: %s", result.Warnings[0].Field)
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

// relationshipEventOrderFixture builds an archive with a marriage relationship
// whose start and end events carry the given dates.
func relationshipEventOrderFixture(startDate, endDate string) *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{}},
			"person-2": {Properties: map[string]any{}},
		},
		Events: map[string]*Event{
			"event-start": {
				Type: EventTypeMarriage, Date: DateString(startDate),
				Participants: []Participant{{Person: "person-1", Role: "spouse"}},
			},
			"event-end": {
				Type: EventTypeDivorce, Date: DateString(endDate),
				Participants: []Participant{{Person: "person-1", Role: "spouse"}},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: RelationshipTypeMarriage,
				Participants: []Participant{
					{Person: "person-1", Role: "spouse"},
					{Person: "person-2", Role: "spouse"},
				},
				StartEvent: "event-start",
				EndEvent:   "event-end",
			},
		},
	}
}

func TestValidateRelationshipEventOrder(t *testing.T) {
	glx := relationshipEventOrderFixture("1880", "1850")
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(result.Warnings))
	}
	warning := result.Warnings[0]
	if !strings.Contains(warning.Message, "end event event-end year (1850) is before start event event-start year (1880)") {
		t.Errorf("Unexpected message: %s", warning.Message)
	}
	if warning.SourceType != EntityTypeRelationships {
		t.Errorf("Warning should have source type 'relationships', got: %s", warning.SourceType)
	}
	if warning.SourceID != "rel-1" {
		t.Errorf("Warning should have source ID 'rel-1', got: %s", warning.SourceID)
	}
	if warning.Field != "end_event" {
		t.Errorf("Warning field should be 'end_event', got: %s", warning.Field)
	}
}

func TestValidateRelationshipEventOrder_NoWarningWhenValid(t *testing.T) {
	glx := relationshipEventOrderFixture("1850", "1880")
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_NoWarningWhenSameYear(t *testing.T) {
	// A relationship that starts and ends in the same year (e.g. shared
	// boundary events) is plausible and must not warn.
	glx := relationshipEventOrderFixture("1850", "1850")
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_SkipsMissingDate(t *testing.T) {
	glx := relationshipEventOrderFixture("1880", "")
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings when a date is missing, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_SkipsUnparseableDate(t *testing.T) {
	glx := relationshipEventOrderFixture("1880", "sometime later")
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for unparseable date, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_SkipsDanglingEventReference(t *testing.T) {
	// Broken event references are reported by reference validation, not here.
	glx := relationshipEventOrderFixture("1880", "1850")
	glx.Relationships["rel-1"].EndEvent = "event-missing"
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for dangling event reference, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_SkipsWhenOnlyStartEvent(t *testing.T) {
	glx := relationshipEventOrderFixture("1880", "1850")
	glx.Relationships["rel-1"].EndEvent = ""
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings when end_event is unset, got %d: %v", len(result.Warnings), result.Warnings)
	}
}

func TestValidateRelationshipEventOrder_NilRelationship(t *testing.T) {
	glx := &GLXFile{
		Relationships: map[string]*Relationship{
			"rel-1": nil,
		},
	}
	result := &ValidationResult{}
	glx.validateRelationshipEventOrder(result)

	if len(result.Warnings) != 0 {
		t.Errorf("Expected 0 warnings for nil relationship, got %d", len(result.Warnings))
	}
}

func TestValidateRelationshipEventOrder_WiredIntoValidate(t *testing.T) {
	glx := relationshipEventOrderFixture("1880", "1850")
	result := glx.Validate()

	hasOrderWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Message, "is before start event") {
			hasOrderWarning = true

			break
		}
	}
	if !hasOrderWarning {
		t.Error("Validate() should warn when a relationship's end event precedes its start event")
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
