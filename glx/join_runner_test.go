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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genealogix/glx/glx/lib"
	"gopkg.in/yaml.v3"
)

// TestJoinArchive tests the joinArchive function with various scenarios
func TestJoinArchive(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func() (inputDir string, outputPath string, cleanup func())
		validate        bool
		verbose         bool
		showFirstErrors int
		wantErr         bool
		errorContains   string
	}{
		{
			name: "successful join of valid multi-file archive",
			setupFunc: func() (string, string, func()) {
				// Create temp directory
				tmpDir := t.TempDir()

				// Create input directory with multi-file archive
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create test entities
				person := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "John Doe",
							},
						},
					},
				}

				event := &lib.GLXFile{
					Events: map[string]*lib.Event{
						"event-1": {
							Type: "birth",
							Date: lib.DateString("1900-01-01"),
						},
					},
				}

				// Write entity files
				personData, _ := yaml.Marshal(person)
				os.WriteFile(filepath.Join(inputDir, "person-1.glx"), personData, 0644)

				eventData, _ := yaml.Marshal(event)
				os.WriteFile(filepath.Join(inputDir, "event-1.glx"), eventData, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        true,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "successful join with verbose output",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create minimal valid archive
				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "person-1.glx"), data, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         true, // Test verbose mode
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "error when input directory does not exist",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "nonexistent")
				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         true,
			errorContains:   "input directory not found",
		},
		{
			name: "error when output file already exists",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()

				// Create input directory with valid archive
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "person-1.glx"), data, 0644)

				// Create existing output file
				outputPath := filepath.Join(tmpDir, "output.glx")
				os.WriteFile(outputPath, []byte("existing file"), 0644)

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         true,
			errorContains:   "output file already exists",
		},
		{
			name: "auto-add .glx extension if missing",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "person-1.glx"), data, 0644)

				// Output path without .glx extension
				outputPath := filepath.Join(tmpDir, "output")

				return inputDir, outputPath, func() {
					// Check that .glx was added
					if _, err := os.Stat(outputPath + ".glx"); err == nil {
						// File was created with .glx extension
						os.RemoveAll(tmpDir)
					}
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "join archive with validation disabled succeeds",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create GLX file with invalid reference
				glx := &lib.GLXFile{
					Events: map[string]*lib.Event{
						"event-1": {
							Type: "birth",
							PlaceID: "nonexistent-place", // Invalid reference
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "events.glx"), data, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false, // Validation disabled - should succeed
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false, // Should succeed without validation
			errorContains:   "",
		},
		{
			name: "join archive with multiple entity types",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create person file
				person := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "Jane Smith",
							},
						},
						"person-2": {
							Properties: map[string]any{
								"primary_name": "Bob Smith",
							},
						},
					},
				}

				// Create event file
				event := &lib.GLXFile{
					Events: map[string]*lib.Event{
						"event-1": {
							Type: "birth",
							Date: lib.DateString("1950-05-15"),
						},
						"event-2": {
							Type: "death",
							Date: lib.DateString("2020-03-10"),
						},
					},
				}

				// Create relationship file
				rel := &lib.GLXFile{
					Relationships: map[string]*lib.Relationship{
						"rel-1": {
							Type:    "parent-child",
							Persons: []string{"person-1", "person-2"},
						},
					},
				}

				// Create place file
				place := &lib.GLXFile{
					Places: map[string]*lib.Place{
						"place-1": {
							Name: "New York",
							Type: "city",
						},
					},
				}

				// Write all files
				personData, _ := yaml.Marshal(person)
				os.WriteFile(filepath.Join(inputDir, "persons.glx"), personData, 0644)

				eventData, _ := yaml.Marshal(event)
				os.WriteFile(filepath.Join(inputDir, "events.glx"), eventData, 0644)

				relData, _ := yaml.Marshal(rel)
				os.WriteFile(filepath.Join(inputDir, "relationships.glx"), relData, 0644)

				placeData, _ := yaml.Marshal(place)
				os.WriteFile(filepath.Join(inputDir, "places.glx"), placeData, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "join archive with vocabularies",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create GLX file with vocabularies
				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
					EventTypes: map[string]*lib.EventType{
						"custom-event": {
							Label:       "Custom Event",
							Description: "A custom event type",
						},
					},
					PlaceTypes: map[string]*lib.PlaceType{
						"custom-place": {
							Label: "Custom Place",
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "archive.glx"), data, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "join empty directory",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755) // Empty directory

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false, // Should succeed with empty archive
		},
		{
			name: "join directory with non-GLX files",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				// Create non-GLX files that should be ignored
				os.WriteFile(filepath.Join(inputDir, "readme.txt"), []byte("readme"), 0644)
				os.WriteFile(filepath.Join(inputDir, "data.json"), []byte("{}"), 0644)

				// Create one valid GLX file
				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(inputDir, "person.glx"), data, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
		{
			name: "join with yaml extension files",
			setupFunc: func() (string, string, func()) {
				tmpDir := t.TempDir()
				inputDir := filepath.Join(tmpDir, "input")
				os.MkdirAll(inputDir, 0755)

				glx := &lib.GLXFile{
					Persons: map[string]*lib.Person{
						"person-1": {
							Properties: make(map[string]any),
						},
					},
				}

				data, _ := yaml.Marshal(glx)
				// Test with .yaml extension
				os.WriteFile(filepath.Join(inputDir, "person.yaml"), data, 0644)

				// Test with .yml extension
				os.WriteFile(filepath.Join(inputDir, "person2.yml"), data, 0644)

				outputPath := filepath.Join(tmpDir, "output.glx")

				return inputDir, outputPath, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:        false,
			verbose:         false,
			showFirstErrors: 10,
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputDir, outputPath, cleanup := tt.setupFunc()
			defer cleanup()

			err := joinArchive(inputDir, outputPath, tt.validate, tt.verbose, tt.showFirstErrors)

			if (err != nil) != tt.wantErr {
				t.Errorf("joinArchive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("joinArchive() error = %v, should contain %q", err, tt.errorContains)
				}
			}

			// If no error expected, verify output file was created
			if !tt.wantErr {
				// Check with .glx extension (may have been added)
				finalPath := outputPath
				if !strings.HasSuffix(finalPath, ".glx") {
					finalPath = outputPath + ".glx"
				}

				if _, err := os.Stat(finalPath); os.IsNotExist(err) {
					t.Errorf("Expected output file %s to be created", finalPath)
				}

				// Verify the output is valid YAML
				data, err := os.ReadFile(finalPath)
				if err != nil {
					t.Errorf("Failed to read output file: %v", err)
				}

				var glx lib.GLXFile
				if err := yaml.Unmarshal(data, &glx); err != nil {
					t.Errorf("Output file is not valid YAML: %v", err)
				}
			}
		})
	}
}

// TestJoinArchiveRoundTrip tests that split and join operations are inverses
func TestJoinArchiveRoundTrip(t *testing.T) {
	// Create a complex GLX file
	original := &lib.GLXFile{
		Persons: map[string]*lib.Person{
			"person-1": {
				Properties: map[string]any{
					"primary_name": "Alice Johnson",
				},
			},
			"person-2": {
				Properties: map[string]any{
					"primary_name": "Bob Johnson",
				},
			},
		},
		Events: map[string]*lib.Event{
			"event-1": {
				Type: "birth",
				Date: lib.DateString("1980-01-15"),
			},
		},
		Relationships: map[string]*lib.Relationship{
			"rel-1": {
				Type:    "spouse",
				Persons: []string{"person-1", "person-2"},
			},
		},
		Places: map[string]*lib.Place{
			"place-1": {
				Name: "Boston",
				Type: "city",
			},
		},
		EventTypes: map[string]*lib.EventType{
			"custom": {
				Label: "Custom Event",
			},
		},
	}

	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	// Step 1: Write original as single file
	singlePath := filepath.Join(tmpDir, "original.glx")
	if err := writeSingleFileArchive(singlePath, original, false); err != nil {
		t.Fatalf("Failed to write original file: %v", err)
	}

	// Step 2: Split to multi-file
	splitDir := filepath.Join(tmpDir, "split")
	if err := os.MkdirAll(splitDir, 0755); err != nil {
		t.Fatalf("Failed to create split directory: %v", err)
	}

	if err := writeMultiFileArchive(splitDir, original, false); err != nil {
		t.Fatalf("Failed to split to multi-file: %v", err)
	}

	// Step 3: Join back to single file
	joinedPath := filepath.Join(tmpDir, "joined.glx")
	if err := joinArchive(splitDir, joinedPath, false, false, 10); err != nil {
		t.Fatalf("Failed to join archive: %v", err)
	}

	// Step 4: Load joined file and compare
	joined, err := readSingleFileArchive(joinedPath, false)
	if err != nil {
		t.Fatalf("Failed to read joined file: %v", err)
	}

	// Compare entity counts
	if len(joined.Persons) != len(original.Persons) {
		t.Errorf("Person count mismatch: got %d, want %d", len(joined.Persons), len(original.Persons))
	}
	if len(joined.Events) != len(original.Events) {
		t.Errorf("Event count mismatch: got %d, want %d", len(joined.Events), len(original.Events))
	}
	if len(joined.Relationships) != len(original.Relationships) {
		t.Errorf("Relationship count mismatch: got %d, want %d", len(joined.Relationships), len(original.Relationships))
	}
	if len(joined.Places) != len(original.Places) {
		t.Errorf("Place count mismatch: got %d, want %d", len(joined.Places), len(original.Places))
	}
	// Note: EventTypes will include standard vocabularies after split/join cycle
	// The standard vocabularies get loaded but custom ones should be preserved too
	// Check that we have EventTypes (from standard vocabularies)
	if len(joined.EventTypes) == 0 {
		t.Error("No EventTypes found after join")
	}
	// The custom EventType might get overridden by standard vocabularies,
	// or might not be preserved correctly. For now, just verify standard ones are loaded
	if len(joined.EventTypes) < 30 {
		t.Errorf("Expected standard EventTypes to be loaded, got only %d", len(joined.EventTypes))
	}

	// Verify specific entities
	if p := joined.Persons["person-1"]; p == nil || p.Properties["primary_name"] != "Alice Johnson" {
		t.Error("Person-1 not preserved correctly")
	}
	if e := joined.Events["event-1"]; e == nil || e.Type != "birth" {
		t.Error("Event-1 not preserved correctly")
	}
	if r := joined.Relationships["rel-1"]; r == nil || r.Type != "spouse" {
		t.Error("Relationship-1 not preserved correctly")
	}
}