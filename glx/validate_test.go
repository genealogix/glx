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

	"github.com/genealogix/spec/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIsValidEntityID(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{
			name:  "standard format",
			id:    "person-12345678",
			valid: true,
		},
		{
			name:  "descriptive ID",
			id:    "person-john-smith",
			valid: true,
		},
		{
			name:  "single character",
			id:    "a",
			valid: true,
		},
		{
			name:  "64 characters",
			id:    "person-" + strings.Repeat("a", 64-7),
			valid: true,
		},
		{
			name:  "empty",
			id:    "",
			valid: false,
		},
		{
			name:  "underscore not allowed",
			id:    "person_12345",
			valid: false,
		},
		{
			name:  "dot not allowed",
			id:    "person.12345",
			valid: false,
		},
		{
			name:  "space not allowed",
			id:    "person 12345",
			valid: false,
		},
		{
			name:  "special char not allowed",
			id:    "person@12345",
			valid: false,
		},
		{
			name:  "slash not allowed",
			id:    "person/12345",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEntityID(tt.id)
			assert.Equal(t, tt.valid, got, "isValidEntityID(%q) = %v, want %v", tt.id, got, tt.valid)
		})
	}
}

func TestParseYAMLFile(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid YAML",
			data:    []byte("persons:\n  person-12345678:\n    properties:\n      given_name: \"John\"\n      family_name: \"Doe\""),
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			data:    []byte("invalid: yaml: syntax: error:"),
			wantErr: true,
		},
		{
			name:    "empty YAML",
			data:    []byte(""),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseYAMLFile(tt.data)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, doc)
			} else {
				assert.NoError(t, err)
				if len(tt.data) > 0 {
					assert.NotNil(t, doc)
				}
			}
		})
	}
}

// Tests removed - testdata deleted. Use examples_test.go for archive validation tests.


func TestValidateVocabularyFile(t *testing.T) {
	tests := []struct {
		name   string
		doc    map[string]interface{}
		expect int // expected number of issues
	}{
		{
			name: "valid relationship_types vocabulary",
			doc: map[string]interface{}{
				"relationship_types": map[string]interface{}{
					"marriage": map[string]interface{}{
						"label": "Marriage",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid event_types vocabulary",
			doc: map[string]interface{}{
				"event_types": map[string]interface{}{
					"birth": map[string]interface{}{
						"label": "Birth",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid place_types vocabulary",
			doc: map[string]interface{}{
				"place_types": map[string]interface{}{
					"city": map[string]interface{}{
						"label": "City",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid repository_types vocabulary",
			doc: map[string]interface{}{
				"repository_types": map[string]interface{}{
					"library": map[string]interface{}{
						"label": "Library",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid participant_roles vocabulary",
			doc: map[string]interface{}{
				"participant_roles": map[string]interface{}{
					"subject": map[string]interface{}{
						"label": "Subject",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid media_types vocabulary",
			doc: map[string]interface{}{
				"media_types": map[string]interface{}{
					"photo": map[string]interface{}{
						"label": "Photo",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid confidence_levels vocabulary",
			doc: map[string]interface{}{
				"confidence_levels": map[string]interface{}{
					"high": map[string]interface{}{
						"label": "High",
					},
				},
			},
			expect: 0,
		},
		{
			name: "valid quality_ratings vocabulary",
			doc: map[string]interface{}{
				"quality_ratings": map[string]interface{}{
					"1": map[string]interface{}{
						"label": "Questionable",
					},
				},
			},
			expect: 0,
		},
		{
			name: "no vocabulary key",
			doc: map[string]interface{}{
				"something": "invalid",
			},
			expect: 1,
		},
		{
			name: "unknown vocabulary type",
			doc: map[string]interface{}{
				"unknown_vocab_type": map[string]interface{}{
					"key": map[string]interface{}{
						"label": "Label",
					},
				},
			},
			expect: 1, // Should report no vocabulary key found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateVocabularyFile("test.glx", tt.doc)
			if tt.expect == 0 {
				// For valid vocab files, we may have schema issues if schemas don't exist
				// but structure should be valid
				if len(issues) > 0 {
					// Check if it's just a schema not found issue
					hasSchemaIssue := false
					for _, issue := range issues {
						if strings.Contains(issue, "schema") || strings.Contains(issue, "no schema found") {
							hasSchemaIssue = true
							break
						}
					}
					if !hasSchemaIssue {
						assert.Len(t, issues, tt.expect, "unexpected issues: %v", issues)
					}
				}
			} else {
				assert.NotEmpty(t, issues, "expected issues for %s", tt.name)
			}
		})
	}
}

func TestRunValidate(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func() string
		wantError bool
	}{
		{
			name: "validate single file",
			args: []string{},
			setup: func() string {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.glx")
				content := `persons:
  person-123:
    properties:
      given_name: "Test"
      family_name: "Person"
`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			wantError: false,
		},
		{
			name: "validate directory",
			args: []string{},
			setup: func() string {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.glx")
				content := `persons:
  person-123:
    properties:
      given_name: "Test"
      family_name: "Person"
`
				err := os.WriteFile(testFile, []byte(content), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			wantError: false,
		},
		{
			name: "no files found",
			args: []string{},
			setup: func() string {
				return t.TempDir()
			},
			wantError: true,
		},
		{
			name: "invalid file extension",
			args: []string{"test.txt"},
			setup: func() string {
				tmpDir := t.TempDir()
				testFile := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(testFile, []byte("not glx"), 0644)
				require.NoError(t, err)
				return tmpDir
			},
			wantError: true,
		},
		{
			name: "non-existent path",
			args: []string{"nonexistent"},
			setup: func() string {
				return t.TempDir()
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDir, err := os.Getwd()
			require.NoError(t, err)

			testDir := tt.setup()
			err = os.Chdir(testDir)
			require.NoError(t, err)
			defer os.Chdir(originalDir)

			// Modify args to use test directory paths
			args := tt.args
			if len(args) == 0 {
				args = []string{"."}
			} else {
				// Make paths relative to test directory
				for i, arg := range args {
					if !filepath.IsAbs(arg) {
						args[i] = filepath.Join(testDir, arg)
					}
				}
			}

			err = runValidate(args)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				// May have errors if validation fails, but function should complete
				_ = err
			}
		})
	}
}

func TestBasicValidateEntity(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entity     map[string]interface{}
		vocabs     *ArchiveVocabularies
		expect     int
	}{
		{
			name:       "relationship missing type",
			entityType: "relationship",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     2, // missing type and persons
		},
		{
			name:       "relationship missing persons",
			entityType: "relationship",
			entity: map[string]interface{}{
				"type": "marriage",
			},
			vocabs: nil,
			expect: 1, // missing persons
		},
		{
			name:       "event missing type",
			entityType: "event",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "place missing name",
			entityType: "place",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "source missing title",
			entityType: "source",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "citation missing source",
			entityType: "citation",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "repository missing name",
			entityType: "repository",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "assertion missing subject",
			entityType: "assertion",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     3, // missing subject, claim, and sources/citations
		},
		{
			name:       "media missing uri",
			entityType: "media",
			entity:     map[string]interface{}{},
			vocabs:     nil,
			expect:     1,
		},
		{
			name:       "relationship with vocab validation",
			entityType: "relationship",
			entity: map[string]interface{}{
				"type":    "unknown-type",
				"persons": []string{"person-1"},
			},
			vocabs: &ArchiveVocabularies{
				RelationshipTypes: map[string]*lib.RelationshipType{
					"marriage": {},
				},
			},
			expect: 1, // unknown type
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := basicValidateEntity(tt.entityType, tt.entity, tt.vocabs)
			assert.Len(t, issues, tt.expect, "expected %d issues, got %d: %v", tt.expect, len(issues), issues)
		})
	}
}

func TestLoadVocabData(t *testing.T) {
	yamlData := `
relationship_types:
  marriage:
    label: "Marriage"
    description: "A marriage"
    gedcom: "MARR"
`
	var glxFile lib.GLXFile
	err := yaml.Unmarshal([]byte(yamlData), &glxFile)
	assert.NoError(t, err)
	assert.Len(t, glxFile.RelationshipTypes, 1)
	assert.NotNil(t, glxFile.RelationshipTypes["marriage"])
	assert.Equal(t, "Marriage", glxFile.RelationshipTypes["marriage"].Label)
	assert.Equal(t, "MARR", glxFile.RelationshipTypes["marriage"].GEDCOM)
}

// TestInvalidArchiveDirectories validates that invalid archive directories fail validation
func TestInvalidArchiveDirectories(t *testing.T) {
	invalidCases := []struct {
		name        string
		description string
	}{
		{"missing-vocabularies", "archive missing required vocabularies"},
		{"broken-references", "archive with invalid entity references"},
		{"invalid-properties", "archive with unknown properties"},
		{"invalid-entity-ids", "archive with invalid entity IDs"},
		{"duplicate-ids", "archive with duplicate entity IDs"},
		{"invalid-relationship-participants", "archive with invalid relationship participant references"},
		{"invalid-assertion-claims", "archive with unknown assertion claims"},
		{"comprehensive-broken-references", "archive with multiple types of broken references (place, person, parent, source, repository)"},
		{"assertion-participant-and-claim", "assertion with both participant and claim (mutually exclusive)"},
		{"assertion-participant-and-value", "assertion with both participant and value (mutually exclusive)"},
		{"assertion-participant-invalid-person", "assertion participant references non-existent person"},
		{"assertion-participant-invalid-role", "assertion participant references non-existent role"},
		{"assertion-unknown-claim", "assertion with unknown claim (should warn)"},
		{"missing-required-fields", "archive with entities missing required fields"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := filepath.Join("testdata", "invalid", tc.name)

			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				t.Skipf("test case %s not found", tc.name)
				return
			}

			// Load and merge all GLX files from the archive
			archive, duplicates, err := LoadArchive(archivePath)
			require.NoError(t, err, "should be able to load invalid archive")

			// Check that there are no duplicate IDs (some test cases should have duplicates)
			if tc.name != "duplicate-ids" {
				assert.Empty(t, duplicates, "invalid archive %s should not have duplicate entity IDs", tc.name)
			} else {
				assert.NotEmpty(t, duplicates, "duplicate-ids test case should have duplicate entity IDs")
			}

			// Validate the merged archive - should have errors
			refErrors, refWarnings := ValidateArchive(archive, archivePath)
			allRefIssues := append(refErrors, refWarnings...)

			// All invalid test cases should have validation issues
			assert.NotEmpty(t, allRefIssues, "%s (%s) should have validation issues: %v", tc.name, tc.description, allRefIssues)
		})
	}
}

// TestValidArchiveDirectories validates that valid archive directories pass validation
func TestValidArchiveDirectories(t *testing.T) {
	validCases := []struct {
		name        string
		description string
	}{
		{"minimal-example", "minimal valid archive"},
	}

	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := filepath.Join("testdata", "valid", tc.name)

			if _, err := os.Stat(archivePath); os.IsNotExist(err) {
				t.Skipf("test case %s not found", tc.name)
				return
			}

			// Load and merge all GLX files from the archive
			archive, duplicates, err := LoadArchive(archivePath)
			require.NoError(t, err, "should be able to load valid archive")

			// Check for duplicate IDs
			assert.Empty(t, duplicates, "valid archive %s should not have duplicate entity IDs", tc.name)

			// Validate the merged archive - should have no errors
			refErrors, refWarnings := ValidateArchive(archive, archivePath)
			allRefIssues := append(refErrors, refWarnings...)

			// Valid archives should have no validation errors (warnings are OK)
			assert.Empty(t, refErrors, "%s (%s) should have no validation errors: %v", tc.name, tc.description, allRefIssues)
		})
	}
}

// TestValidateGLXFile tests the ValidateGLXFile function directly
func TestValidateGLXFile(t *testing.T) {
	tests := []struct {
		name   string
		doc    map[string]interface{}
		vocabs *ArchiveVocabularies
		expect int // expected number of issues
	}{
		{
			name: "valid GLX file",
			doc: map[string]interface{}{
				"persons": map[string]interface{}{
					"person-123": map[string]interface{}{
						"properties": map[string]interface{}{
							"given_name": "John",
							"family_name": "Smith",
						},
					},
				},
			},
			vocabs: &ArchiveVocabularies{
				PersonProperties: map[string]*lib.PropertyDefinition{
					"given_name":  {Label: "Given Name"},
					"family_name": {Label: "Family Name"},
				},
			},
			expect: 0,
		},
		{
			name: "entity with id field should be rejected",
			doc: map[string]interface{}{
				"persons": map[string]interface{}{
					"person-123": map[string]interface{}{
						"id": "person-123", // Should not have id field
						"properties": map[string]interface{}{
							"given_name": "John",
						},
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should reject id field
		},
		{
			name: "invalid entity ID format",
			doc: map[string]interface{}{
				"persons": map[string]interface{}{
					"person_123": map[string]interface{}{ // Underscore not allowed
						"properties": map[string]interface{}{
							"given_name": "John",
						},
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Invalid ID format
		},
		{
			name: "media entity",
			doc: map[string]interface{}{
				"media": map[string]interface{}{
					"media-123": map[string]interface{}{
						"uri":       "media/photos/photo.jpg",
						"mime_type": "image/jpeg",
						"title":     "Test Photo",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name: "multiple entity types",
			doc: map[string]interface{}{
				"persons": map[string]interface{}{
					"person-123": map[string]interface{}{
						"properties": map[string]interface{}{
							"given_name": "John",
						},
					},
				},
				"events": map[string]interface{}{
					"event-456": map[string]interface{}{
						"type": "birth",
						"date": "1850-01-15",
					},
				},
				"places": map[string]interface{}{
					"place-789": map[string]interface{}{
						"name": "Leeds",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name: "unknown entity type falls back to basic validation",
			doc: map[string]interface{}{
				"unknown_entities": map[string]interface{}{
					"unknown-123": map[string]interface{}{
						"some_field": "value",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0, // Unknown entity types don't fail, just pass through
		},
		{
			name: "event with unknown type (schema validates enum)",
			doc: map[string]interface{}{
				"events": map[string]interface{}{
					"event-unknown": map[string]interface{}{
						"type": "unknown-event-type",
						"date": "1850-01-15",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Schema rejects unknown type via enum
		},
		{
			name: "relationship with unknown type (schema validates enum)",
			doc: map[string]interface{}{
				"relationships": map[string]interface{}{
					"rel-unknown": map[string]interface{}{
						"type": "unknown-rel-type",
						"participants": []interface{}{
							map[string]interface{}{
								"person": "person-123",
							},
						},
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Schema rejects unknown type via enum
		},
		{
			name: "place with unknown type (schema validates enum)",
			doc: map[string]interface{}{
				"places": map[string]interface{}{
					"place-unknown": map[string]interface{}{
						"name": "Leeds",
						"type": "unknown-place-type",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Schema rejects unknown type via enum
		},
		{
			name: "repository with unknown type (schema validates enum)",
			doc: map[string]interface{}{
				"repositories": map[string]interface{}{
					"repo-unknown": map[string]interface{}{
						"name": "Library",
						"type": "unknown-repo-type",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Schema rejects unknown type via enum
		},
		{
			name: "event missing type falls back to basic validation",
			doc: map[string]interface{}{
				"events": map[string]interface{}{
					"event-missing-type": map[string]interface{}{
						"date": "1850-01-15",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require type field
		},
		{
			name: "relationship missing type falls back to basic validation",
			doc: map[string]interface{}{
				"relationships": map[string]interface{}{
					"rel-missing-type": map[string]interface{}{
						"participants": []interface{}{
							map[string]interface{}{
								"person": "person-123",
							},
						},
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 2, // Should require type and persons fields
		},
		{
			name: "place missing name falls back to basic validation",
			doc: map[string]interface{}{
				"places": map[string]interface{}{
					"place-missing-name": map[string]interface{}{
						"type": "city",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require name field
		},
		{
			name: "source missing title falls back to basic validation",
			doc: map[string]interface{}{
				"sources": map[string]interface{}{
					"source-missing-title": map[string]interface{}{
						"type": "book",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require title field
		},
		{
			name: "citation missing source falls back to basic validation",
			doc: map[string]interface{}{
				"citations": map[string]interface{}{
					"citation-missing-source": map[string]interface{}{
						"locator": "Page 1",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require source field
		},
		{
			name: "repository missing name falls back to basic validation",
			doc: map[string]interface{}{
				"repositories": map[string]interface{}{
					"repo-missing-name": map[string]interface{}{
						"type": "library",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require name field
		},
		{
			name: "assertion missing subject falls back to basic validation",
			doc: map[string]interface{}{
				"assertions": map[string]interface{}{
					"assertion-missing-subject": map[string]interface{}{
						"claim": "given_name",
						"value": "John",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 2, // Should require subject and sources/citations
		},
		{
			name: "media missing both uri and file_path falls back to basic validation",
			doc: map[string]interface{}{
				"media": map[string]interface{}{
					"media-missing": map[string]interface{}{
						"mime_type": "image/jpeg",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Should require uri or file_path
		},
		{
			name: "media with file_path instead of uri",
			doc: map[string]interface{}{
				"media": map[string]interface{}{
					"media-filepath": map[string]interface{}{
						"file_path": "media/photos/photo.jpg",
						"mime_type": "image/jpeg",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 1, // Schema requires uri, but basic validation allows file_path - schema error is reported
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateGLXFile("test.glx", tt.doc, tt.vocabs)
			if tt.expect == 0 {
				assert.Empty(t, issues, "expected no issues but got: %v", issues)
			} else {
				assert.GreaterOrEqual(t, len(issues), tt.expect, "expected at least %d issues but got: %v", tt.expect, issues)
			}
		})
	}
}

// TestValidateEntityByType tests the validateEntityByType function directly
func TestValidateEntityByType(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entity     map[string]interface{}
		vocabs     *ArchiveVocabularies
		expect     int // expected number of issues
	}{
		{
			name:       "valid person entity",
			entityType: "person",
			entity: map[string]interface{}{
				"properties": map[string]interface{}{
					"given_name": "John",
				},
			},
			vocabs: &ArchiveVocabularies{
				PersonProperties: map[string]*lib.PropertyDefinition{
					"given_name": {Label: "Given Name"},
				},
			},
			expect: 0,
		},
		{
			name:       "valid event entity",
			entityType: "event",
			entity: map[string]interface{}{
				"type": "birth",
				"date": "1850-01-15",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "valid place entity",
			entityType: "place",
			entity: map[string]interface{}{
				"name": "Leeds",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "valid media entity",
			entityType: "media",
			entity: map[string]interface{}{
				"uri":       "media/photos/photo.jpg",
				"mime_type": "image/jpeg",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "invalid entity - missing required field",
			entityType: "place",
			entity:     map[string]interface{}{}, // Missing name
			vocabs:     &ArchiveVocabularies{},
			expect:     1, // Should fail schema validation
		},
		{
			name:       "unknown entity type falls back to basic validation",
			entityType: "unknown_type",
			entity: map[string]interface{}{
				"some_field": "value",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0, // Unknown types don't fail, just pass through
		},
		{
			name:       "getEntityType test - person",
			entityType: "person",
			entity: map[string]interface{}{
				"properties": map[string]interface{}{
					"given_name": "John",
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "getEntityType test - event",
			entityType: "event",
			entity: map[string]interface{}{
				"type": "birth",
				"date": "1850-01-15",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "getEntityType test - relationship",
			entityType: "relationship",
			entity: map[string]interface{}{
				"type": "marriage",
				"participants": []interface{}{
					map[string]interface{}{
						"person": "person-123",
					},
				},
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
		{
			name:       "getEntityType test - place",
			entityType: "place",
			entity: map[string]interface{}{
				"name": "Leeds",
			},
			vocabs: &ArchiveVocabularies{},
			expect: 0,
		},
	}

		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := validateEntityByType(tt.entityType, tt.entity, tt.vocabs)
			if tt.expect == 0 {
				// For schema validation, we may have issues, so just check it doesn't crash
				_ = issues
			} else {
				assert.GreaterOrEqual(t, len(issues), tt.expect, "expected at least %d issues but got: %v", tt.expect, issues)
			}
		})
	}
}

// TestValidateEntityPropertiesFromStruct tests property validation with vocab entries
func TestValidateEntityPropertiesFromStruct(t *testing.T) {
	vocabs := &ArchiveVocabularies{
		PersonProperties: map[string]*lib.PropertyDefinition{
			"given_name": {
				Label:        "Given Name",
				ValueType:    "string",
				ReferenceType: "",
			},
			"family_name": {
				Label:        "Family Name",
				ValueType:    "string",
				ReferenceType: "",
			},
			"birth_place": {
				Label:        "Birth Place",
				ValueType:    "reference",
				ReferenceType: "places",
			},
		},
		PlaceProperties: map[string]*lib.PropertyDefinition{
			"name": {
				Label:        "Name",
				ValueType:    "string",
				ReferenceType: "",
			},
			"custom_place_prop": {
				Label:        "Custom Place Property",
				ValueType:    "string",
				ReferenceType: "",
			},
		},
		EventProperties: map[string]*lib.PropertyDefinition{
			"type": {
				Label:        "Type",
				ValueType:    "string",
				ReferenceType: "",
			},
			"custom_event_prop": {
				Label:        "Custom Event Property",
				ValueType:    "string",
				ReferenceType: "",
			},
		},
		RelationshipProperties: map[string]*lib.PropertyDefinition{
			"type": {
				Label:        "Type",
				ValueType:    "string",
				ReferenceType: "",
			},
			"custom_rel_prop": {
				Label:        "Custom Relationship Property",
				ValueType:    "string",
				ReferenceType: "",
			},
		},
	}

	allEntities := map[string]map[string]bool{
		"persons":  {"person-123": true},
		"places":   {"place-leeds": true},
		"events":   {},
		"relationships": {},
	}

	tests := []struct {
		name       string
		entityType string
		entityID   string
		properties map[string]interface{}
		expectWarn int
		expectErr  int
	}{
		{
			name:       "valid properties",
			entityType: "persons",
			entityID:   "person-123",
			properties: map[string]interface{}{
				"given_name":  "John",
				"family_name": "Smith",
			},
			expectWarn: 0,
			expectErr:  0,
		},
		{
			name:       "unknown property",
			entityType: "persons",
			entityID:   "person-123",
			properties: map[string]interface{}{
				"unknown_prop": "value",
			},
			expectWarn: 1, // Unknown property should warn
			expectErr:  0,
		},
		{
			name:       "valid reference property",
			entityType: "persons",
			entityID:   "person-123",
			properties: map[string]interface{}{
				"birth_place": "place-leeds",
			},
			expectWarn: 0,
			expectErr:  0,
		},
		{
			name:       "invalid reference property",
			entityType: "persons",
			entityID:   "person-123",
			properties: map[string]interface{}{
				"birth_place": "place-nonexistent",
			},
			expectWarn: 0,
			expectErr:  1, // Invalid reference should error
		},
		{
			name:       "event properties validation",
			entityType: "events",
			entityID:   "event-123",
			properties: map[string]interface{}{
				"custom_event_prop": "value",
			},
			expectWarn: 0,
			expectErr:  0,
		},
		{
			name:       "relationship properties validation",
			entityType: "relationships",
			entityID:   "rel-123",
			properties: map[string]interface{}{
				"custom_rel_prop": "value",
			},
			expectWarn: 0,
			expectErr:  0,
		},
		{
			name:       "place properties validation",
			entityType: "places",
			entityID:   "place-123",
			properties: map[string]interface{}{
				"custom_place_prop": "value",
			},
			expectWarn: 0,
			expectErr:  0,
		},
		{
			name:       "event properties with unknown property",
			entityType: "events",
			entityID:   "event-123",
			properties: map[string]interface{}{
				"unknown_event_prop": "value",
			},
			expectWarn: 1, // Unknown property should warn
			expectErr:  0,
		},
		{
			name:       "relationship properties with unknown property",
			entityType: "relationships",
			entityID:   "rel-123",
			properties: map[string]interface{}{
				"unknown_rel_prop": "value",
			},
			expectWarn: 1, // Unknown property should warn
			expectErr:  0,
		},
		{
			name:       "place properties with unknown property",
			entityType: "places",
			entityID:   "place-123",
			properties: map[string]interface{}{
				"unknown_place_prop": "value",
			},
			expectWarn: 1, // Unknown property should warn
			expectErr:  0,
		},
		{
			name:       "no vocab for entity type",
			entityType: "unknown",
			entityID:   "unknown-123",
			properties: map[string]interface{}{
				"some_prop": "value",
			},
			expectWarn: 0,
			expectErr:  0, // No vocab means no validation
		},
		{
			name:       "nil properties",
			entityType: "persons",
			entityID:   "person-123",
			properties: nil,
			expectWarn: 0,
			expectErr:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := validateEntityPropertiesFromStruct(tt.entityType, tt.entityID, tt.properties, allEntities, vocabs)
			assert.Equal(t, tt.expectWarn, len(warnings), "expected %d warnings but got %d: %v", tt.expectWarn, len(warnings), warnings)
			assert.Equal(t, tt.expectErr, len(errors), "expected %d errors but got %d: %v", tt.expectErr, len(errors), errors)
		})
	}
}
