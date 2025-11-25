package main

import (
	"testing"
)

// TestRunValidate_TemporalPropertiesValid tests valid temporal property usage
func TestRunValidate_TemporalPropertiesValid(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "simple temporal values",
			path: "testdata/valid/temporal-properties-simple",
		},
		{
			name: "temporal lists with dates",
			path: "testdata/valid/temporal-properties-list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive, _, err := LoadArchive(tt.path)
			if err != nil {
				t.Fatalf("LoadArchive failed: %v", err)
			}

			result := archive.Validate()
			if len(result.Errors) != 0 {
				t.Errorf("Expected 0 errors for %s, got %d: %v", tt.name, len(result.Errors), result.Errors)
			}
			if len(result.Warnings) != 0 {
				t.Errorf("Expected 0 warnings for %s, got %d: %v", tt.name, len(result.Warnings), result.Warnings)
			}
		})
	}
}

// TestRunValidate_TemporalPropertiesMalformed tests malformed temporal property usage
func TestRunValidate_TemporalPropertiesMalformed(t *testing.T) {
	archive, _, err := LoadArchive("testdata/invalid/temporal-properties-malformed")
	if err != nil {
		t.Fatalf("LoadArchive failed: %v", err)
	}

	result := archive.Validate()

	// Expected errors:
	// 1. Non-temporal property (birth_year) with list value -> WARNING
	// 2. Temporal list item missing 'value' field (name) -> ERROR
	// Total: 1 error, 2 warnings minimum

	// Actually checking:
	// - birth_year with list -> 1 warning (non-temporal with list)
	// - name missing value -> 1 error
	// - occupation missing date -> 1 warning
	// Total expected: 1 error, 2 warnings

	if len(result.Errors) < 1 {
		t.Errorf("Expected at least 1 error, got %d", len(result.Errors))
		for i, e := range result.Errors {
			t.Logf("Error %d: %s", i+1, e.Message)
		}
	}

	if len(result.Warnings) < 2 {
		t.Errorf("Expected at least 2 warnings, got %d", len(result.Warnings))
		for i, w := range result.Warnings {
			t.Logf("Warning %d: %s", i+1, w.Message)
		}
	}

	// Log all errors and warnings for debugging
	t.Logf("Total errors: %d", len(result.Errors))
	for i, e := range result.Errors {
		t.Logf("  Error %d: [%s][%s] %s", i+1, e.SourceType, e.SourceID, e.Message)
	}
	t.Logf("Total warnings: %d", len(result.Warnings))
	for i, w := range result.Warnings {
		t.Logf("  Warning %d: [%s][%s] %s", i+1, w.SourceType, w.SourceID, w.Message)
	}
}
