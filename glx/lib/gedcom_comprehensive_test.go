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
		// GEDCOM 5.5.1 - Already tested in E2E
		{"5.5.1/shakespeare-family/shakespeare.ged", 31, 77, "Comprehensive family tree"},
		{"5.5.1/kennedy-family/kennedy.ged", 20, 0, "Political family"},
		{"5.5.1/british-royalty/british-royalty.ged", 10, 0, "Royal lineage"},
		{"5.5.1/bullinger-family/bullinger.ged", 948, 0, "Large performance test"},
		{"5.5.1/torture-test-551/torture-test.ged", 0, 0, "Edge case stress test - large file with many edge cases"},

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
		{"5.5.5/spec-samples/sample.ged", 1, 0, "Spec sample file"},

		// GEDCOM 7.0 - Already tested in E2E
		{"7.0/minimal-valid/minimal70.ged", 0, 0, "Minimal 7.0 - header only"},
		{"7.0/comprehensive-spec/maximal70.ged", 1, 0, "Comprehensive 7.0"},
		{"7.0/date-formats/date-all.ged", 1, 0, "All date formats"},
		{"7.0/age-values/age-all.ged", 1, 0, "All age formats"},
		{"7.0/same-sex-marriage/same-sex-marriage.ged", 2, 0, "Same-sex marriage"},

		// GEDCOM 7.0 - New tests
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

			glx, result, err := ImportGEDCOMFromFile(fullPath, logPath)
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

			glx, result, err := ImportGEDCOMFromFile(fullPath, logPath)
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
