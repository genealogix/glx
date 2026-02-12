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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	glxlib "github.com/genealogix/glx/go-glx"
)

// TestLoadArchive tests the LoadArchive function that loads and merges multi-file archives
func TestLoadArchive(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (rootPath string, cleanup func())
		wantErr       bool
		errorContains string
		checkFunc     func(glx *glxlib.GLXFile, duplicates []string) error
	}{
		{
			name: "load valid multi-file archive",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create GLX files
				person1 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "Alice",
							},
						},
					},
				}
				person2 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-2": {
							Properties: map[string]any{
								"primary_name": "Bob",
							},
						},
					},
				}

				// Write files
				data1, _ := yaml.Marshal(person1)
				os.WriteFile(filepath.Join(tmpDir, "person1.glx"), data1, 0o644)

				data2, _ := yaml.Marshal(person2)
				os.WriteFile(filepath.Join(tmpDir, "person2.glx"), data2, 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			checkFunc: func(glx *glxlib.GLXFile, duplicates []string) error {
				if len(glx.Persons) != 2 {
					return &testError{"expected 2 persons, got %d", []any{len(glx.Persons)}}
				}
				if len(duplicates) != 0 {
					return &testError{"expected no duplicates, got %d", []any{len(duplicates)}}
				}

				return nil
			},
		},
		{
			name: "load archive with duplicate IDs",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create two files with same person ID
				person1 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "Alice",
							},
						},
					},
				}
				person2 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": { // Same ID as person1
							Properties: map[string]any{
								"primary_name": "Bob",
							},
						},
					},
				}

				data1, _ := yaml.Marshal(person1)
				os.WriteFile(filepath.Join(tmpDir, "person1.glx"), data1, 0o644)

				data2, _ := yaml.Marshal(person2)
				os.WriteFile(filepath.Join(tmpDir, "person2.glx"), data2, 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			checkFunc: func(glx *glxlib.GLXFile, duplicates []string) error {
				if len(glx.Persons) != 1 {
					return &testError{"expected 1 person (after merge), got %d", []any{len(glx.Persons)}}
				}
				if len(duplicates) != 1 {
					return &testError{"expected 1 duplicate, got %d", []any{len(duplicates)}}
				}
				if duplicates[0] != "duplicate persons ID: person-1" {
					return &testError{"expected duplicate to be 'duplicate persons ID: person-1', got %s", []any{duplicates[0]}}
				}

				return nil
			},
		},
		{
			name: "handle YAML parse error",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create invalid YAML file
				os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte("invalid: yaml: content:"), 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr:       true,
			errorContains: "multiple files failed validation",
		},
		{
			name: "handle structural validation error",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create YAML file with invalid structure
				invalidData := []byte(`
unknown_field: value
persons:
  person-1:
    invalid_field: test
`)
				os.WriteFile(filepath.Join(tmpDir, "invalid.glx"), invalidData, 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr:       true,
			errorContains: "multiple files failed validation",
		},
		{
			name: "handle I/O error - directory not found",
			setupFunc: func() (string, func()) {
				return "/nonexistent/directory", func() {}
			},
			wantErr:       true,
			errorContains: "no such file or directory",
		},
		{
			name: "load archive with all entity types",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create GLX file with all entity types
				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: make(map[string]any)},
						"person-2": {Properties: make(map[string]any)},
					},
					Events: map[string]*glxlib.Event{
						"event-1": {Type: "birth", Participants: []glxlib.Participant{{Person: "person-1", Role: "principal"}}},
					},
					Relationships: map[string]*glxlib.Relationship{
						"rel-1": {
							Type: "parent_child",
							Participants: []glxlib.Participant{
								{Person: "person-1", Role: "child"},
								{Person: "person-2", Role: "parent"},
							},
						},
					},
					Places: map[string]*glxlib.Place{
						"place-1": {Name: "Boston"},
					},
					Sources: map[string]*glxlib.Source{
						"source-1": {Title: "Test Source"},
					},
					Citations: map[string]*glxlib.Citation{
						"citation-1": {SourceID: "source-1"},
					},
					Repositories: map[string]*glxlib.Repository{
						"repo-1": {Name: "Test Repo"},
					},
					Assertions: map[string]*glxlib.Assertion{
						"assert-1": {
							Subject:  glxlib.EntityRef{Person: "person-1"},
							Sources:  []string{"source-1"},
							Property: "born_on",
						},
					},
					Media: map[string]*glxlib.Media{
						"media-1": {URI: "http://example.com"},
					},
					// Vocabularies (include referenced values for validation)
					EventTypes: map[string]*glxlib.EventType{
						"birth":  {Label: "Birth"},
						"custom": {Label: "Custom"},
					},
					ParticipantRoles: map[string]*glxlib.ParticipantRole{
						"principal": {Label: "Principal"},
						"child":     {Label: "Child"},
						"parent":    {Label: "Parent"},
					},
					RelationshipTypes: map[string]*glxlib.RelationshipType{
						"parent_child": {Label: "Parent-Child"},
					},
					PlaceTypes: map[string]*glxlib.PlaceType{
						"custom": {Label: "Custom"},
					},
				}

				data, _ := yaml.Marshal(glx)
				os.WriteFile(filepath.Join(tmpDir, "complete.glx"), data, 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			checkFunc: func(glx *glxlib.GLXFile, duplicates []string) error {
				if len(glx.Persons) != 2 {
					return &testError{"expected 2 persons", nil}
				}
				if len(glx.Events) != 1 {
					return &testError{"expected 1 event", nil}
				}
				if len(glx.Relationships) != 1 {
					return &testError{"expected 1 relationship", nil}
				}
				if len(glx.Places) != 1 {
					return &testError{"expected 1 place", nil}
				}
				if len(glx.Sources) != 1 {
					return &testError{"expected 1 source", nil}
				}
				if len(glx.Citations) != 1 {
					return &testError{"expected 1 citation", nil}
				}
				if len(glx.Repositories) != 1 {
					return &testError{"expected 1 repository", nil}
				}
				if len(glx.Assertions) != 1 {
					return &testError{"expected 1 assertion", nil}
				}
				if len(glx.Media) != 1 {
					return &testError{"expected 1 media", nil}
				}
				if len(glx.EventTypes) != 2 {
					return &testError{"expected 2 event types", nil}
				}
				if len(glx.ParticipantRoles) != 3 {
					return &testError{"expected 3 participant roles", nil}
				}
				if len(glx.RelationshipTypes) != 1 {
					return &testError{"expected 1 relationship type", nil}
				}
				if len(glx.PlaceTypes) != 1 {
					return &testError{"expected 1 place type", nil}
				}

				return nil
			},
		},
		{
			name: "skip non-GLX files",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create various file types
				glxFile := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: make(map[string]any)},
					},
				}
				data, _ := yaml.Marshal(glxFile)
				os.WriteFile(filepath.Join(tmpDir, "valid.glx"), data, 0o644)
				os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), data, 0o644)
				os.WriteFile(filepath.Join(tmpDir, "valid.yml"), data, 0o644)

				// These should be ignored
				os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0o644)
				os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			checkFunc: func(glx *glxlib.GLXFile, duplicates []string) error {
				// Should load 3 files (glx, yaml, yml) each with same person, resulting in duplicates
				if len(glx.Persons) != 1 {
					return &testError{"expected 1 person", nil}
				}
				if len(duplicates) != 2 { // 2 duplicates since 3 files have same person ID
					return &testError{"expected 2 duplicates, got %d", []any{len(duplicates)}}
				}

				return nil
			},
		},
		{
			name: "handle nested directories",
			setupFunc: func() (string, func()) {
				tmpDir := t.TempDir()

				// Create nested structure
				subDir := filepath.Join(tmpDir, "subdir")
				os.MkdirAll(subDir, 0o755)

				glxFile1 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: make(map[string]any)},
					},
				}
				glxFile2 := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-2": {Properties: make(map[string]any)},
					},
				}

				data1, _ := yaml.Marshal(glxFile1)
				os.WriteFile(filepath.Join(tmpDir, "root.glx"), data1, 0o644)

				data2, _ := yaml.Marshal(glxFile2)
				os.WriteFile(filepath.Join(subDir, "nested.glx"), data2, 0o644)

				return tmpDir, func() {
					os.RemoveAll(tmpDir)
				}
			},
			wantErr: false,
			checkFunc: func(glx *glxlib.GLXFile, duplicates []string) error {
				if len(glx.Persons) != 2 {
					return &testError{"expected 2 persons from nested dirs, got %d", []any{len(glx.Persons)}}
				}

				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootPath, cleanup := tt.setupFunc()
			defer cleanup()

			glx, duplicates, err := LoadArchive(rootPath)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadArchive() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil && tt.errorContains != "" {
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("LoadArchive() error = %v, should contain %q", err, tt.errorContains)
				}
			}

			if !tt.wantErr && tt.checkFunc != nil {
				if err := tt.checkFunc(glx, duplicates); err != nil {
					t.Errorf("LoadArchive() check failed: %v", err)
				}
			}
		})
	}
}

// TestReadWriteSingleFileArchive tests reading and writing single-file archives
func TestReadWriteSingleFileArchive(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func() (path string, glx *glxlib.GLXFile, cleanup func())
		validate      bool
		wantReadErr   bool
		wantWriteErr  bool
		errorContains string
	}{
		{
			name: "successful read and write",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "test.glx")

				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "Test Person",
							},
						},
					},
				}

				return path, glx, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:     false,
			wantReadErr:  false,
			wantWriteErr: false,
		},
		{
			name: "read non-existent file",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				return "/nonexistent/file.glx", nil, func() {}
			},
			validate:      false,
			wantReadErr:   true,
			errorContains: "failed to read file",
		},
		{
			name: "write to invalid path",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: make(map[string]any)},
					},
				}

				return "/nonexistent/dir/file.glx", glx, func() {}
			},
			validate:      false,
			wantWriteErr:  true,
			errorContains: "failed to write GLX file",
		},
		{
			name: "roundtrip without validation",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				tmpDir := t.TempDir()
				path := filepath.Join(tmpDir, "test.glx")

				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {
							Properties: map[string]any{
								"primary_name": "Valid Person",
							},
						},
					},
					Events: map[string]*glxlib.Event{
						"event-1": {
							Type: "birth",
							Date: glxlib.DateString("1950-01-01"),
						},
					},
				}

				return path, glx, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:     false, // Don't validate - just test I/O
			wantReadErr:  false,
			wantWriteErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, glx, cleanup := tt.setupFunc()
			defer cleanup()

			// Test write
			if glx != nil {
				err := writeSingleFileArchive(path, glx, tt.validate)
				if (err != nil) != tt.wantWriteErr {
					t.Errorf("writeSingleFileArchive() error = %v, wantWriteErr %v", err, tt.wantWriteErr)
				}
				if err != nil && tt.errorContains != "" {
					if !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("writeSingleFileArchive() error = %v, should contain %q", err, tt.errorContains)
					}
				}
			}

			// Test read (only if write was successful or testing read-only)
			if glx == nil || !tt.wantWriteErr {
				loaded, err := readSingleFileArchive(path, tt.validate)
				if (err != nil) != tt.wantReadErr {
					t.Errorf("readSingleFileArchive() error = %v, wantReadErr %v", err, tt.wantReadErr)
				}
				if err != nil && tt.errorContains != "" {
					if !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("readSingleFileArchive() error = %v, should contain %q", err, tt.errorContains)
					}
				}

				// Verify roundtrip if both succeeded
				if glx != nil && loaded != nil && !tt.wantWriteErr && !tt.wantReadErr {
					if len(loaded.Persons) != len(glx.Persons) {
						t.Errorf("Roundtrip failed: person count mismatch")
					}
					if len(loaded.Events) != len(glx.Events) {
						t.Errorf("Roundtrip failed: event count mismatch")
					}
				}
			}
		})
	}
}

// TestReadWriteMultiFileArchive tests reading and writing multi-file archives
func TestReadWriteMultiFileArchive(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func() (dirPath string, glx *glxlib.GLXFile, cleanup func())
		validate     bool
		wantReadErr  bool
		wantWriteErr bool
	}{
		{
			name: "successful multi-file read and write",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				tmpDir := t.TempDir()
				archiveDir := filepath.Join(tmpDir, "archive")
				os.MkdirAll(archiveDir, 0o755)

				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: map[string]any{"primary_name": "Alice"}},
						"person-2": {Properties: map[string]any{"primary_name": "Bob"}},
					},
					Events: map[string]*glxlib.Event{
						"event-1": {Type: "birth", Date: glxlib.DateString("1950-01-01")},
						"event-2": {Type: "death", Date: glxlib.DateString("2020-01-01")},
					},
					Places: map[string]*glxlib.Place{
						"place-1": {Name: "New York", Type: "city"},
					},
				}

				return archiveDir, glx, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:     false,
			wantReadErr:  false,
			wantWriteErr: false,
		},
		{
			name: "read from non-existent directory",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				return "/nonexistent/directory", nil, func() {}
			},
			validate:    false,
			wantReadErr: true,
		},
		{
			name: "write to invalid directory",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"person-1": {Properties: make(map[string]any)},
					},
				}

				return "/root/invalid/dir", glx, func() {}
			},
			validate:     false,
			wantWriteErr: true,
		},
		{
			name: "roundtrip with complex archive",
			setupFunc: func() (string, *glxlib.GLXFile, func()) {
				tmpDir := t.TempDir()
				archiveDir := filepath.Join(tmpDir, "complex")
				os.MkdirAll(archiveDir, 0o755)

				glx := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{
						"p1": {Properties: map[string]any{"name": "Person 1"}},
						"p2": {Properties: map[string]any{"name": "Person 2"}},
						"p3": {Properties: map[string]any{"name": "Person 3"}},
					},
					Events: map[string]*glxlib.Event{
						"e1": {Type: "birth"},
						"e2": {Type: "death"},
					},
					Relationships: map[string]*glxlib.Relationship{
						"r1": {
							Type: "parent_child",
							Participants: []glxlib.Participant{
								{Person: "p1"},
								{Person: "p2"},
							},
						},
					},
					Places: map[string]*glxlib.Place{
						"pl1": {Name: "Boston"},
						"pl2": {Name: "Chicago"},
					},
					EventTypes: map[string]*glxlib.EventType{
						"custom": {Label: "Custom Event"},
					},
				}

				return archiveDir, glx, func() {
					os.RemoveAll(tmpDir)
				}
			},
			validate:     false,
			wantReadErr:  false,
			wantWriteErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirPath, glx, cleanup := tt.setupFunc()
			defer cleanup()

			// Test write
			if glx != nil {
				err := writeMultiFileArchive(dirPath, glx, tt.validate)
				if (err != nil) != tt.wantWriteErr {
					t.Errorf("writeMultiFileArchive() error = %v, wantWriteErr %v", err, tt.wantWriteErr)
				}
			}

			// Test read (only if write was successful or testing read-only)
			if glx == nil || !tt.wantWriteErr {
				loaded, _, err := LoadArchiveWithOptions(dirPath, tt.validate)
				if (err != nil) != tt.wantReadErr {
					t.Errorf("LoadArchiveWithOptions() error = %v, wantReadErr %v", err, tt.wantReadErr)
				}

				// Verify roundtrip if both succeeded
				if glx != nil && loaded != nil && !tt.wantWriteErr && !tt.wantReadErr {
					if len(loaded.Persons) != len(glx.Persons) {
						t.Errorf("Roundtrip failed: person count mismatch: got %d, want %d",
							len(loaded.Persons), len(glx.Persons))
					}
					if len(loaded.Events) != len(glx.Events) {
						t.Errorf("Roundtrip failed: event count mismatch")
					}
					if len(loaded.Relationships) != len(glx.Relationships) {
						t.Errorf("Roundtrip failed: relationship count mismatch")
					}
					if len(loaded.Places) != len(glx.Places) {
						t.Errorf("Roundtrip failed: place count mismatch")
					}
				}
			}
		})
	}
}

// TestCreateSerializer tests the createSerializer function
func TestCreateSerializer(t *testing.T) {
	tests := []struct {
		name      string
		validate  bool
		pretty    bool
		indent    string
		checkOpts func(s *glxlib.DefaultSerializer) bool
	}{
		{
			name:     "default options",
			validate: false,
			pretty:   false,
			indent:   "",
		},
		{
			name:     "with validation",
			validate: true,
			pretty:   false,
			indent:   "",
		},
		{
			name:     "with pretty print",
			validate: false,
			pretty:   true,
			indent:   "  ",
		},
		{
			name:     "all options enabled",
			validate: true,
			pretty:   true,
			indent:   "    ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serializer := createSerializer(tt.validate, tt.pretty, tt.indent)
			if serializer == nil {
				t.Error("createSerializer() returned nil")
			}
			// We can't directly check the options as they're private,
			// but we can verify the serializer was created
		})
	}
}

// TestLoadArchiveAndJoinProduceSameResult verifies both deserialization paths
// produce equivalent results from the same multi-file archive on disk.
func TestLoadArchiveAndJoinProduceSameResult(t *testing.T) {
	// Create a known archive with multiple entity types
	glx := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"name": map[string]any{"value": "Alice"}}},
			"person-2": {Properties: map[string]any{"name": map[string]any{"value": "Bob"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {Type: "birth", Participants: []glxlib.Participant{{Person: "person-1", Role: "principal"}}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-1"}, {Person: "person-2"}}},
		},
		Places:       map[string]*glxlib.Place{"place-1": {Name: "London"}},
		Sources:      make(map[string]*glxlib.Source),
		Citations:    make(map[string]*glxlib.Citation),
		Repositories: make(map[string]*glxlib.Repository),
		Media:        make(map[string]*glxlib.Media),
		Assertions:   make(map[string]*glxlib.Assertion),
	}

	// Write to multi-file archive
	dirPath := t.TempDir()
	if err := writeMultiFileArchive(dirPath, glx, false); err != nil {
		t.Fatalf("writeMultiFileArchive failed: %v", err)
	}

	// Load with schema validation (used by validate command)
	loaded1, duplicates, err := LoadArchiveWithOptions(dirPath, true)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions(validate=true) failed: %v", err)
	}
	if len(duplicates) > 0 {
		t.Errorf("Unexpected duplicates: %v", duplicates)
	}

	// Load without schema validation (used by join command)
	loaded2, _, err := LoadArchiveWithOptions(dirPath, false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions(validate=false) failed: %v", err)
	}

	// Both should produce the same entity counts
	if len(loaded1.Persons) != len(loaded2.Persons) {
		t.Errorf("Person count mismatch: validated=%d, unvalidated=%d", len(loaded1.Persons), len(loaded2.Persons))
	}
	if len(loaded1.Events) != len(loaded2.Events) {
		t.Errorf("Event count mismatch: validated=%d, unvalidated=%d", len(loaded1.Events), len(loaded2.Events))
	}
	if len(loaded1.Relationships) != len(loaded2.Relationships) {
		t.Errorf("Relationship count mismatch: validated=%d, unvalidated=%d", len(loaded1.Relationships), len(loaded2.Relationships))
	}
	if len(loaded1.Places) != len(loaded2.Places) {
		t.Errorf("Place count mismatch: validated=%d, unvalidated=%d", len(loaded1.Places), len(loaded2.Places))
	}

	// Verify specific entities exist
	if _, ok := loaded1.Persons["person-1"]; !ok {
		t.Error("Missing person-1")
	}
}

// TestJoinPreservesEntities verifies that split→join round-trip preserves all entities.
func TestJoinPreservesEntities(t *testing.T) {
	glx := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{"name": map[string]any{"value": "Alice"}}},
			"person-2": {Properties: map[string]any{"name": map[string]any{"value": "Bob"}}},
		},
		Events: map[string]*glxlib.Event{
			"event-1": {Type: "birth", Participants: []glxlib.Participant{{Person: "person-1", Role: "principal"}}},
			"event-2": {Type: "death", Participants: []glxlib.Participant{{Person: "person-2", Role: "principal"}}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "person-1"}, {Person: "person-2"}}},
		},
		Places:       map[string]*glxlib.Place{"place-1": {Name: "London"}},
		Sources:      make(map[string]*glxlib.Source),
		Citations:    make(map[string]*glxlib.Citation),
		Repositories: make(map[string]*glxlib.Repository),
		Media:        make(map[string]*glxlib.Media),
		Assertions:   make(map[string]*glxlib.Assertion),
	}

	// Write to multi-file
	splitDir := t.TempDir()
	if err := writeMultiFileArchive(splitDir, glx, false); err != nil {
		t.Fatalf("writeMultiFileArchive failed: %v", err)
	}

	// Join back to single file
	joinedPath := filepath.Join(t.TempDir(), "joined.glx")
	if err := joinArchive(splitDir, joinedPath, false, false, 0); err != nil {
		t.Fatalf("joinArchive failed: %v", err)
	}

	// Read the joined file
	joined, err := readSingleFileArchive(joinedPath, false)
	if err != nil {
		t.Fatalf("readSingleFileArchive failed: %v", err)
	}

	// Verify all entity counts
	if len(joined.Persons) != 2 {
		t.Errorf("Expected 2 persons, got %d", len(joined.Persons))
	}
	if len(joined.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(joined.Events))
	}
	if len(joined.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(joined.Relationships))
	}
	if len(joined.Places) != 1 {
		t.Errorf("Expected 1 place, got %d", len(joined.Places))
	}

	// Verify specific entities
	if _, ok := joined.Persons["person-1"]; !ok {
		t.Error("Joined archive missing person-1")
	}
	if _, ok := joined.Events["event-2"]; !ok {
		t.Error("Joined archive missing event-2")
	}
}

// testError is a helper type for test error formatting
type testError struct {
	msg  string
	args []any
}

func (e *testError) Error() string {
	if len(e.args) > 0 {
		return fmt.Sprintf(e.msg, e.args...)
	}

	return e.msg
}
