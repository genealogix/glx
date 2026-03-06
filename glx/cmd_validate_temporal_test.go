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
	// Load without validation since this test uses intentionally malformed data
	archive, _, err := LoadArchiveWithOptions("testdata/invalid/temporal-properties-malformed", false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions failed: %v", err)
	}

	result := archive.Validate()

	// Expected:
	// - birth_year with list -> 1 warning (non-temporal with list)
	// - name missing value -> 1 error
	// Total expected: 1 error, 1 warning minimum

	if len(result.Errors) < 1 {
		t.Errorf("Expected at least 1 error, got %d", len(result.Errors))
		for i, e := range result.Errors {
			t.Logf("Error %d: %s", i+1, e.Message)
		}
	}

	if len(result.Warnings) < 1 {
		t.Errorf("Expected at least 1 warning, got %d", len(result.Warnings))
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
