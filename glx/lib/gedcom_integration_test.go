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

package lib

import (
	"path/filepath"
	"testing"
)

func TestImportMinimal70(t *testing.T) {
	// Test minimal GEDCOM 7.0 file
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "minimal-valid", "minimal70.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	t.Logf("Import statistics: %+v", result.Statistics)
	t.Logf("Errors: %d, Warnings: %d", len(result.Statistics.Errors), len(result.Statistics.Warnings))

	// Log any errors
	for _, e := range result.Statistics.Errors {
		t.Logf("  Error [Line %d] %s: %s", e.Line, e.Tag, e.Message)
	}

	// Verify GLX structure
	if glx == nil {
		t.Fatal("GLX file is nil")
	}

	// Basic validation
	if glx.Persons == nil {
		t.Error("Persons map is nil")
	}
	if glx.Events == nil {
		t.Error("Events map is nil")
	}
}

func TestImportShakespeare(t *testing.T) {
	// Test GEDCOM 5.5.1 with Shakespeare family
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "shakespeare-family", "shakespeare.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if result.Version != "5.5.1" {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	t.Logf("Imported %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))

	t.Logf("Full statistics: %+v", result.Statistics)
	t.Logf("Errors: %d, Warnings: %d", len(result.Statistics.Errors), len(result.Statistics.Warnings))

	// Log first few warnings (since converters not yet implemented)
	for i, w := range result.Statistics.Warnings {
		if i < 5 {
			t.Logf("  Warning [Line %d] %s: %s", w.Line, w.Tag, w.Message)
		}
	}

	// Verify actual data is persisted correctly
	// Check for William Shakespeare
	foundWilliam := false
	for _, person := range glx.Persons {
		givenName, familyName := ExtractNameFields(person.Properties[PersonPropertyName])

		if givenName == "William" && familyName == "Shakespeare" {
			foundWilliam = true

			// Verify gender
			if gender, ok := person.Properties[PersonPropertyGender].(string); !ok || gender != "male" {
				t.Error("William Shakespeare should have gender 'male'")
			}

			t.Logf("✓ Found William Shakespeare with correct name and gender")

			break
		}
	}
	if !foundWilliam {
		t.Error("Failed to import William Shakespeare - person data not persisted")
	}

	// Verify events are properly linked to persons
	eventCount := 0
	for _, event := range glx.Events {
		if len(event.Participants) > 0 {
			eventCount++
		}
	}
	if eventCount == 0 {
		t.Error("No events have participants - event-person linkage not persisted")
	}

	// Verify relationships are properly linked
	relationshipCount := 0
	for _, rel := range glx.Relationships {
		if len(rel.Participants) >= 2 {
			relationshipCount++
		}
	}
	if relationshipCount == 0 {
		t.Error("No relationships have participants - relationship data not persisted")
	}
}

func TestParseGEDCOMDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"exact date with day", "1 JAN 1900", "1900-01-01"},
		{"exact date month/year", "JAN 1900", "1900-01"},
		{"exact date year only", "1900", "1900"},
		{"about date", "ABT 1900", "ABT 1900"},
		{"before date", "BEF 15 JAN 1900", "BEF 1900-01-15"},
		{"after date", "AFT 1900", "AFT 1900"},
		{"calculated date", "CAL 1900", "CAL 1900"},
		{"between range", "BET 1900 AND 1910", "BET 1900 AND 1910"},
		{"from-to range", "FROM 1900 TO 1910", "FROM 1900 TO 1910"},
		{"open-ended from", "FROM 1900", "FROM 1900"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGEDCOMDate(tt.input)
			if string(result) != tt.expected {
				t.Errorf("parseGEDCOMDate(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGEDCOMName(t *testing.T) {
	tests := []struct {
		input    string
		given    string
		surname  string
		nickname string
	}{
		{"John /Smith/", "John", "Smith", ""},
		{"John \"Jack\" /Smith/", "John", "Smith", "Jack"},
		{"Dr. John /Smith/ Jr.", "John", "Smith", ""},
		{"/von Neumann/", "", "Neumann", ""},
		{"Mary Jane /Smith-Jones/", "Mary Jane", "Smith-Jones", ""},
	}

	for _, tt := range tests {
		result := parseGEDCOMName(tt.input, nil)
		if result.GivenName != tt.given {
			t.Errorf("parseGEDCOMName(%q).GivenName = %q, want %q", tt.input, result.GivenName, tt.given)
		}
		if result.Surname != tt.surname {
			t.Errorf("parseGEDCOMName(%q).Surname = %q, want %q", tt.input, result.Surname, tt.surname)
		}
		if result.Nickname != tt.nickname {
			t.Errorf("parseGEDCOMName(%q).Nickname = %q, want %q", tt.input, result.Nickname, tt.nickname)
		}
	}
}

func TestParseGEDCOMPlace(t *testing.T) {
	tests := []struct {
		input      string
		components int
	}{
		{"New York, New York, USA", 3},
		{"London, England", 2},
		{"Paris", 1},
		{"", 0},
	}

	for _, tt := range tests {
		result := parseGEDCOMPlace(tt.input)
		if result == nil && tt.components > 0 {
			t.Errorf("parseGEDCOMPlace(%q) returned nil, expected %d components", tt.input, tt.components)

			continue
		}
		if result != nil && len(result.Components) != tt.components {
			t.Errorf("parseGEDCOMPlace(%q) has %d components, want %d", tt.input, len(result.Components), tt.components)
		}
	}
}
