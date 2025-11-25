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

// E2E tests for GEDCOM import - test full import pipeline without serialization

func TestE2E_GEDCOM551_Shakespeare(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "shakespeare-family", "shakespeare.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion551 {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	// Entity counts
	if len(glx.Persons) != 31 {
		t.Errorf("Expected 31 persons, got %d", len(glx.Persons))
	}
	if len(glx.Events) != 77 {
		t.Errorf("Expected 77 events, got %d", len(glx.Events))
	}
	if len(glx.Relationships) != 49 {
		t.Errorf("Expected 49 relationships, got %d", len(glx.Relationships))
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}
	if len(glx.RelationshipTypes) == 0 {
		t.Error("Relationship types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 10 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	// Spot check: William Shakespeare
	foundWilliam := false
	for _, person := range glx.Persons {
		given, family := ExtractNameFields(person.Properties[PersonPropertyName])
		if given == "William" && family == "Shakespeare" {
			foundWilliam = true
			if gender, ok := person.Properties[PersonPropertyGender].(string); !ok || gender != "male" {
				t.Error("William Shakespeare should be male")
			}
		}
	}
	if !foundWilliam {
		t.Error("Did not find William Shakespeare in persons - person data not persisted")
	}

	// Verify birth events have dates
	birthEventsWithDates := 0
	for _, event := range glx.Events {
		if event.Type == EventTypeBirth && event.Date != "" {
			birthEventsWithDates++
		}
	}
	if birthEventsWithDates == 0 {
		t.Error("No birth events have dates - event date data not persisted")
	} else {
		t.Logf("✓ Found %d birth events with dates", birthEventsWithDates)
	}

	// Verify parent-child relationships exist
	parentChildRels := 0
	for _, rel := range glx.Relationships {
		if rel.Type == RelationshipTypeParentChild {
			parentChildRels++
			// Verify participants are linked
			if len(rel.Participants) < 2 {
				t.Errorf("Parent-child relationship has %d participants, expected at least 2", len(rel.Participants))
			}
		}
	}
	if parentChildRels == 0 {
		t.Error("No parent-child relationships found - relationship data not persisted")
	} else {
		t.Logf("✓ Found %d parent-child relationships with proper participant linkage", parentChildRels)
	}

	t.Logf("✓ Shakespeare: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM551_Kennedy(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "kennedy-family", "kennedy.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion551 {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}
	if glx.Events == nil {
		t.Fatal("Events map is nil")
	}
	if glx.Relationships == nil {
		t.Fatal("Relationships map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ Kennedy: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM551_BritishRoyalty(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "british-royalty", "british-royalty.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion551 {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ British Royalty: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM551_Bullinger(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "bullinger-family", "bullinger.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion551 {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ Bullinger: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM551_TortureTest(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "5.5.1", "torture-test-551", "torture-test.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion551 {
		t.Errorf("Expected version 5.5.1, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ Torture Test: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM70_Minimal(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "minimal-valid", "minimal70.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ GEDCOM 7.0 Minimal: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM70_Maximal(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "comprehensive-spec", "maximal70.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ GEDCOM 7.0 Maximal: %d persons, %d events, %d relationships",
		len(glx.Persons), len(glx.Events), len(glx.Relationships))
}

func TestE2E_GEDCOM70_DateFormats(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "date-formats", "date-all.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	// Events should exist to test date parsing
	if len(glx.Events) == 0 {
		t.Error("Expected events to test date formats")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ GEDCOM 7.0 Date Formats: %d persons, %d events",
		len(glx.Persons), len(glx.Events))
}

func TestE2E_GEDCOM70_AgeValues(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "age-values", "age-all.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	// Basic structure
	if glx.Persons == nil {
		t.Fatal("Persons map is nil")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ GEDCOM 7.0 Age Values: %d persons, %d events",
		len(glx.Persons), len(glx.Events))
}

func TestE2E_GEDCOM70_SameSexMarriage(t *testing.T) {
	gedcomPath := filepath.Join("..", "testdata", "gedcom", "7.0", "same-sex-marriage", "same-sex-marriage.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	// Version check
	if result.Version != GEDCOMVersion70 {
		t.Errorf("Expected version 7.0, got %s", result.Version)
	}

	// Should have at least 2 persons and 1 marriage relationship
	if len(glx.Persons) < 2 {
		t.Error("Expected at least 2 persons for same-sex marriage")
	}

	foundMarriage := false
	for _, rel := range glx.Relationships {
		if rel.Type == RelationshipTypeMarriage {
			foundMarriage = true

			break
		}
	}
	if !foundMarriage {
		t.Error("Did not find marriage relationship")
	}

	// Vocabularies loaded
	if len(glx.EventTypes) == 0 {
		t.Error("Event types vocabulary not loaded")
	}

	// Validation should pass
	validationResult := glx.Validate()
	if len(validationResult.Errors) > 0 {
		t.Errorf("Validation failed with %d errors:", len(validationResult.Errors))
		for i, e := range validationResult.Errors {
			if i < 5 {
				t.Logf("  - %s", e.Message)
			}
		}
	}

	t.Logf("✓ GEDCOM 7.0 Same-Sex Marriage: %d persons, %d relationships",
		len(glx.Persons), len(glx.Relationships))
}

// Test that all imported GLX files have required structure
func TestE2E_AllFilesHaveRequiredStructure(t *testing.T) {
	testFiles := []struct {
		name    string
		path    string
		version string
	}{
		{"Shakespeare 5.5.1", "5.5.1/shakespeare-family/shakespeare.ged", "5.5.1"},
		{"Kennedy 5.5.1", "5.5.1/kennedy-family/kennedy.ged", "5.5.1"},
		{"British Royalty 5.5.1", "5.5.1/british-royalty/british-royalty.ged", "5.5.1"},
		{"Bullinger 5.5.1", "5.5.1/bullinger-family/bullinger.ged", "5.5.1"},
		{"Torture Test 5.5.1", "5.5.1/torture-test-551/torture-test.ged", "5.5.1"},
		{"Minimal 7.0", "7.0/minimal-valid/minimal70.ged", "7.0"},
		{"Maximal 7.0", "7.0/comprehensive-spec/maximal70.ged", "7.0"},
		{"Date Formats 7.0", "7.0/date-formats/date-all.ged", "7.0"},
		{"Age Values 7.0", "7.0/age-values/age-all.ged", "7.0"},
		{"Same-Sex Marriage 7.0", "7.0/same-sex-marriage/same-sex-marriage.ged", "7.0"},
	}

	for _, tc := range testFiles {
		t.Run(tc.name, func(t *testing.T) {
			gedcomPath := filepath.Join("..", "testdata", "gedcom", tc.path)
			logPath := filepath.Join(t.TempDir(), "import.log")

			glx, result, err := importGEDCOMFromFile(gedcomPath, logPath)
			if err != nil {
				t.Fatalf("Import failed: %v", err)
			}

			// Version
			if result.Version != tc.version {
				t.Errorf("Expected version %s, got %s", tc.version, result.Version)
			}

			// Required maps initialized
			if glx.Persons == nil {
				t.Error("Persons map is nil")
			}
			if glx.Events == nil {
				t.Error("Events map is nil")
			}
			if glx.Relationships == nil {
				t.Error("Relationships map is nil")
			}
			if glx.Places == nil {
				t.Error("Places map is nil")
			}
			if glx.Sources == nil {
				t.Error("Sources map is nil")
			}
			if glx.Citations == nil {
				t.Error("Citations map is nil")
			}
			if glx.Repositories == nil {
				t.Error("Repositories map is nil")
			}
			if glx.Media == nil {
				t.Error("Media map is nil")
			}
			if glx.Assertions == nil {
				t.Error("Assertions map is nil")
			}

			// Vocabularies loaded
			if len(glx.EventTypes) == 0 {
				t.Error("Event types vocabulary not loaded")
			}
			if len(glx.RelationshipTypes) == 0 {
				t.Error("Relationship types vocabulary not loaded")
			}
			if len(glx.ParticipantRoles) == 0 {
				t.Error("Participant roles vocabulary not loaded")
			}
			if len(glx.ConfidenceLevels) == 0 {
				t.Error("Confidence levels vocabulary not loaded")
			}

			// Validation passes
			validationResult := glx.Validate()
			if len(validationResult.Errors) > 0 {
				t.Errorf("Validation failed with %d errors (showing first 3):", len(validationResult.Errors))
				for i, e := range validationResult.Errors {
					if i < 3 {
						t.Logf("  - %s", e.Message)
					}
				}
			}
		})
	}
}
