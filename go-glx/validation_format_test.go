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
	"testing"
)

// TestValidateDateFormat tests date format validation
func TestValidateDateFormat(t *testing.T) {
	tests := []struct {
		name          string
		date          string
		expectWarning bool
	}{
		// Valid simple dates
		{"year only", "1850", false},
		{"year-month", "1850-03", false},
		{"year-month-day", "1850-03-15", false},
		{"full date", "2020-12-31", false},

		// Valid keyword dates
		{"FROM TO range", "FROM 1850 TO 1900", false},
		{"FROM only", "FROM 1850", false},
		{"ABT date", "ABT 1850", false},
		{"BEF date", "BEF 1920", false},
		{"AFT date", "AFT 1880", false},
		{"BET AND range", "BET 1880 AND 1890", false},
		{"CAL date", "CAL 1850", false},
		{"INT date", "INT 1999-03-15 (March 15, 1999)", false},

		// Invalid dates
		{"invalid format", "March 15, 1850", true},
		{"partial year", "185", true},
		{"wrong separator", "1850/03/15", true},
		{"wrong order", "15-03-1850", true},
		{"text only", "sometime in 1850", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glx := &GLXFile{}
			result := &ValidationResult{}

			glx.validateDateFormat("persons", "person-1", "properties.born_on", tt.date, result)

			hasWarning := len(result.Warnings) > 0
			if hasWarning != tt.expectWarning {
				t.Errorf("Date '%s': expected warning=%v, got warning=%v (warnings: %v)",
					tt.date, tt.expectWarning, hasWarning, result.Warnings)
			}
		})
	}
}

// TestValidateValueType tests value type validation
func TestValidateValueType(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		valueType     string
		expectWarning bool
	}{
		// String type
		{"valid string", "John Smith", "string", false},
		{"invalid string - int", 123, "string", true},
		{"invalid string - bool", true, "string", true},

		// Integer type
		{"valid integer - int", 1850, "integer", false},
		{"valid integer - float64", float64(1850), "integer", false},
		{"invalid integer - string", "1850", "integer", true},
		{"invalid integer - bool", true, "integer", true},

		// Boolean type
		{"valid boolean - true", true, "boolean", false},
		{"valid boolean - false", false, "boolean", false},
		{"invalid boolean - string", "true", "boolean", true},
		{"invalid boolean - int", 1, "boolean", true},

		// Date type
		{"valid date - simple", "1850-03-15", "date", false},
		{"valid date - year", "1850", "date", false},
		{"valid date - keyword", "FROM 1850 TO 1900", "date", false},
		{"invalid date - wrong type", 1850, "date", true},
		{"invalid date - bad format", "March 1850", "date", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glx := &GLXFile{}
			result := &ValidationResult{}

			glx.validateValueType("persons", "person-1", "properties.test", tt.value, tt.valueType, result)

			hasWarning := len(result.Warnings) > 0
			if hasWarning != tt.expectWarning {
				t.Errorf("Value %v (type %s): expected warning=%v, got warning=%v (warnings: %v)",
					tt.value, tt.valueType, tt.expectWarning, hasWarning, result.Warnings)
			}
		})
	}
}

// TestIsValidSimpleDate tests the simple date validation function
func TestIsValidSimpleDate(t *testing.T) {
	tests := []struct {
		date  string
		valid bool
	}{
		// Valid
		{"1850", true},
		{"2020", true},
		{"1850-03", true},
		{"2020-12", true},
		{"1850-03-15", true},
		{"2020-12-31", true},

		// Invalid
		{"185", false},
		{"18500", false},
		{"1850-3", false},
		{"1850-003", false},
		{"1850-03-5", false},
		{"1850-03-015", false},
		{"1850/03/15", false},
		{"1850.03.15", false},
		{"March 1850", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			result := isValidSimpleDate(tt.date)
			if result != tt.valid {
				t.Errorf("isValidSimpleDate('%s') = %v, want %v", tt.date, result, tt.valid)
			}
		})
	}
}

// TestPropertyValueFormatValidation tests end-to-end format validation
func TestPropertyValueFormatValidation(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {
				Properties: map[string]any{
					"string_prop": "John",       // string - valid
					"born_on":     "1850-03-15", // date - valid
					"age":         30,           // integer - valid
					"living":      true,         // boolean - valid
					"bad_date":    "March 1850", // date - invalid format
					"bad_int":     "thirty",     // integer - invalid type
				},
			},
		},
	}

	propVocab := map[string]*PropertyDefinition{
		"string_prop": {ValueType: "string"},
		"born_on":     {ValueType: "date"},
		"age":         {ValueType: "integer"},
		"living":      {ValueType: "boolean"},
		"bad_date":    {ValueType: "date"},
		"bad_int":     {ValueType: "integer"},
	}

	result := &ValidationResult{
		PropertyVocabs: map[string]map[string]*PropertyDefinition{
			"person_properties": propVocab,
		},
	}

	// Validate all properties
	glx.validateEntityProperties("persons", "person_properties", glx.Persons, propVocab, result)

	// Should have warnings for bad_date and bad_int
	if len(result.Warnings) < 2 {
		t.Errorf("Expected at least 2 warnings, got %d", len(result.Warnings))
		for i, w := range result.Warnings {
			t.Logf("Warning %d: %s", i+1, w.Message)
		}
	}

	// Check that we got the specific warnings
	foundBadDate := false
	foundBadInt := false
	for _, w := range result.Warnings {
		if w.Field == "properties.bad_date" {
			foundBadDate = true
		}
		if w.Field == "properties.bad_int" {
			foundBadInt = true
		}
	}

	if !foundBadDate {
		t.Error("Expected warning for bad_date field")
	}
	if !foundBadInt {
		t.Error("Expected warning for bad_int field")
	}
}
