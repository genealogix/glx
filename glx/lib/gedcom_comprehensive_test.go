package lib

import (
	"path/filepath"
	"testing"
)

// TestGEDCOM_ImportAllTestFiles tests that all 35 GEDCOM test files import successfully
// This is a comprehensive test to ensure full GEDCOM import coverage for v0.0.0-beta.2
func TestGEDCOM_ImportAllTestFiles(t *testing.T) {
	testFiles := []struct {
		path       string
		minPersons int    // Minimum expected persons (0 = any)
		minEvents  int    // Minimum expected events (0 = any)
		notes      string // Description of test file
	}{
		// GEDCOM 5.5.1 - Edge cases
		{"5.5.1/edge-cases/empty-family.ged", 0, 0, "Empty family record"},
		{"5.5.1/edge-cases/self-marriage.ged", 1, 0, "Person married to self"},
		{"5.5.1/edge-cases/all-genders.ged", 3, 0, "M/F/U genders"},
		{"5.5.1/edge-cases/female-female-marriage.ged", 2, 0, "Same-sex marriage"},
		{"5.5.1/edge-cases/male-male-marriage.ged", 2, 0, "Same-sex marriage"},
		{"5.5.1/edge-cases/unknown-unknown-marriage.ged", 2, 0, "Unknown gender marriage"},

		// GEDCOM 5.5.1 - Encoding tests
		{"5.5.1/character-encoding/simple-ascii.ged", 1, 0, "ASCII only"},
		{"5.5.1/gramps-encoding/cp1252-crlf.ged", 1, 0, "Windows CP1252 CRLF"},
		{"5.5.1/gramps-encoding/cp1252-lf.ged", 1, 0, "Windows CP1252 LF"},
		{"5.5.1/gramps-encoding/utf8-nobom-lf.ged", 1, 0, "UTF-8 no BOM"},

		// GEDCOM 5.5.1 - Famous people
		{"5.5.1/famous-people/bronte.ged", 1, 0, "Brontë family"},
		{"5.5.1/famous-people/royal92.ged", 1, 0, "Royal family"},

		// GEDCOM 5.5.1 - Large files (skip in normal test runs)
		// These are tested separately due to size
		// {"5.5.1/large-files/habsburg.ged", 100, 0, "Habsburg dynasty"},
		// {"5.5.1/large-files/queen.ged", 50, 0, "British monarchy"},

		// GEDCOM 5.5.1 - Assessment
		{"5.5.1/gedcom-assessment/assess.ged", 1, 0, "GEDCOM quality assessment"},

		// GEDCOM 5.5.5
		{"5.5.5/spec-samples/minimal.ged", 0, 0, "Minimal valid GEDCOM - header only"},
		{"5.5.5/spec-samples/remarriage.ged", 2, 0, "Remarriage scenario"},
		{"5.5.5/spec-samples/same-sex-marriage.ged", 2, 0, "Same-sex marriage"},
		{"5.5.5/spec-samples/sample.ged", 3, 7, "Spec sample file"},

		// GEDCOM 7.0 - Additional test coverage
		{"7.0/cross-references/xref.ged", 1, 0, "Cross-reference handling"},
		{"7.0/escaping/escapes.ged", 1, 0, "String escaping"},
		{"7.0/extensions/extensions.ged", 1, 0, "Extension tags"},
		{"7.0/language/lang.ged", 0, 0, "Language support - tests language tags, no persons"},
		{"7.0/notes/notes-1.ged", 0, 0, "Note handling - tests shared notes, no persons"},
		{"7.0/void-pointers/voidptr.ged", 1, 0, "VOID pointer handling"},
	}

	for _, tc := range testFiles {
		t.Run(tc.path, func(t *testing.T) {
			fullPath := filepath.Join("..", "testdata", "gedcom", tc.path)
			logPath := filepath.Join(t.TempDir(), "import.log")

			glx, result, err := importGEDCOMFromFile(fullPath, logPath)
			if err != nil {
				t.Fatalf("Import failed for %s: %v\nNotes: %s", tc.path, err, tc.notes)
			}

			// Verify import succeeded
			if result == nil {
				t.Fatal("Import returned nil result")
			}

			// Basic sanity checks
			if tc.minPersons > 0 && len(glx.Persons) < tc.minPersons {
				t.Errorf("Expected at least %d persons, got %d", tc.minPersons, len(glx.Persons))
			}
			if tc.minEvents > 0 && len(glx.Events) < tc.minEvents {
				t.Errorf("Expected at least %d events, got %d", tc.minEvents, len(glx.Events))
			}

			// Verify vocabularies loaded (all imports should have event types)
			if len(glx.EventTypes) == 0 {
				t.Error("Expected event type vocabularies to be loaded")
			}

			// Data persistence checks - verify person properties are populated
			if len(glx.Persons) > 0 {
				personWithProperties := 0
				for _, person := range glx.Persons {
					if len(person.Properties) > 0 {
						personWithProperties++
					}
				}
				if personWithProperties == 0 {
					t.Error("No persons have properties - person data not persisted")
				}
			}

			// Verify relationships have participants when present
			if len(glx.Relationships) > 0 {
				relsWithParticipants := 0
				for _, rel := range glx.Relationships {
					if len(rel.Participants) >= 2 {
						relsWithParticipants++
					}
				}
				if relsWithParticipants == 0 {
					t.Error("No relationships have participants - relationship linkage not persisted")
				}
			}

			t.Logf("✓ %s: %d persons, %d events, %d relationships",
				tc.notes, len(glx.Persons), len(glx.Events), len(glx.Relationships))
		})
	}
}

// TestGEDCOM_ImportLargeFiles tests large GEDCOM files separately
// These are marked as slow and may be skipped in quick test runs
func TestGEDCOM_ImportLargeFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file tests in short mode")
	}

	largeFiles := []struct {
		path       string
		minPersons int
		notes      string
	}{
		{"5.5.1/large-files/habsburg.ged", 100, "Habsburg dynasty - large file"},
		{"5.5.1/large-files/queen.ged", 50, "British monarchy - large file"},
	}

	for _, tc := range largeFiles {
		t.Run(tc.path, func(t *testing.T) {
			fullPath := filepath.Join("..", "testdata", "gedcom", tc.path)
			logPath := filepath.Join(t.TempDir(), "import.log")

			glx, result, err := importGEDCOMFromFile(fullPath, logPath)
			if err != nil {
				t.Fatalf("Import failed for %s: %v\nNotes: %s", tc.path, err, tc.notes)
			}

			if result == nil {
				t.Fatal("Import returned nil result")
			}

			if tc.minPersons > 0 && len(glx.Persons) < tc.minPersons {
				t.Errorf("Expected at least %d persons, got %d", tc.minPersons, len(glx.Persons))
			}

			if len(glx.EventTypes) == 0 {
				t.Error("Expected event type vocabularies to be loaded")
			}

			t.Logf("✓ %s: %d persons, %d events, %d relationships",
				tc.notes, len(glx.Persons), len(glx.Events), len(glx.Relationships))
		})
	}
}

// TestGEDCOM555_Sample_DataPersistence validates that actual GEDCOM data is correctly imported
// This test verifies specific values from the 5.5.5 sample.ged file
func TestGEDCOM555_Sample_DataPersistence(t *testing.T) {
	fullPath := filepath.Join("..", "testdata", "gedcom", "5.5.5", "spec-samples", "sample.ged")
	logPath := filepath.Join(t.TempDir(), "import.log")

	glx, result, err := importGEDCOMFromFile(fullPath, logPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if result.Version != "5.5.5" {
		t.Errorf("Expected version 5.5.5, got %s", result.Version)
	}

	// Exact counts
	if len(glx.Persons) != 3 {
		t.Errorf("Expected exactly 3 persons, got %d", len(glx.Persons))
	}

	// Test Person Data Persistence: Robert Eugene Williams
	foundRobert := false
	for _, person := range glx.Persons {
		givenName, familyName := ExtractNameFields(person.Properties[PersonPropertyName])

		if givenName == "Robert Eugene" && familyName == "Williams" {
			foundRobert = true

			// Verify gender persisted
			if gender, ok := person.Properties[PersonPropertyGender].(string); !ok || gender != GenderMale {
				t.Error("Robert Eugene Williams should have gender 'male'")
			}

			t.Logf("✓ Robert Eugene Williams: name and gender persisted correctly")

			break
		}
	}
	if !foundRobert {
		t.Error("Failed to find Robert Eugene Williams - person name data not persisted")
	}

	// Test Person Data Persistence: Mary Ann Wilson
	foundMary := false
	for _, person := range glx.Persons {
		givenName, familyName := ExtractNameFields(person.Properties[PersonPropertyName])

		if givenName == "Mary Ann" && familyName == "Wilson" {
			foundMary = true

			// Verify gender persisted
			if gender, ok := person.Properties[PersonPropertyGender].(string); !ok || gender != "female" {
				t.Error("Mary Ann Wilson should have gender 'female'")
			}

			t.Logf("✓ Mary Ann Wilson: name and gender persisted correctly")

			break
		}
	}
	if !foundMary {
		t.Error("Failed to find Mary Ann Wilson - person name data not persisted")
	}

	// Test Event Data Persistence: Birth dates should be imported
	birthEventsWithDates := 0
	for _, event := range glx.Events {
		if event.Type == EventTypeBirth && event.Date != "" {
			birthEventsWithDates++
		}
	}
	if birthEventsWithDates == 0 {
		t.Error("No birth events have dates - event date data not persisted")
	}

	// Test Place Data Persistence
	if len(glx.Places) == 0 {
		t.Error("No places imported - place data not persisted")
	}

	// Test Relationship Data Persistence: Marriage relationships
	marriageRelationships := 0
	for _, rel := range glx.Relationships {
		if rel.Type == RelationshipTypeMarriage {
			marriageRelationships++
			// Verify participants are linked
			if len(rel.Participants) < 2 {
				t.Errorf("Marriage relationship has %d participants, expected at least 2", len(rel.Participants))
			}
		}
	}
	if marriageRelationships == 0 {
		t.Error("No marriage relationships found - relationship data not persisted")
	}

	// Test Source Data Persistence
	if len(glx.Sources) == 0 {
		t.Error("No sources imported - source data not persisted")
	} else {
		foundMadisonSource := false
		for _, source := range glx.Sources {
			if source.Title == "Madison County Birth, Death, and Marriage Records" {
				foundMadisonSource = true
				t.Logf("✓ Source 'Madison County Birth, Death, and Marriage Records' persisted correctly")

				break
			}
		}
		if !foundMadisonSource {
			t.Error("Failed to find expected source - source title data not persisted")
		}
	}

	// Test Repository Data Persistence
	if len(glx.Repositories) == 0 {
		t.Error("No repositories imported - repository data not persisted")
	} else {
		foundFHL := false
		for _, repo := range glx.Repositories {
			if repo.Name == "Family History Library" {
				foundFHL = true
				t.Logf("✓ Repository 'Family History Library' persisted correctly")

				break
			}
		}
		if !foundFHL {
			t.Error("Failed to find expected repository - repository name data not persisted")
		}
	}

	t.Logf("✓ Sample.ged data persistence validated: %d persons, %d events, %d relationships, %d sources, %d repositories",
		len(glx.Persons), len(glx.Events), len(glx.Relationships), len(glx.Sources), len(glx.Repositories))
}
