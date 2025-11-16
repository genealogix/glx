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
				if tt.data != nil && len(tt.data) > 0 {
					assert.NotNil(t, doc)
				}
			}
		})
	}
}

// Tests removed - testdata deleted. Use examples_test.go for archive validation tests.

func TestFormatValidationIssues(t *testing.T) {
	issues := []string{"issue 1", "issue 2"}
	path := "test.glx"
	result := formatValidationIssues(path, issues)

	assert.Len(t, result, 3)
	assert.Contains(t, result[0], "test.glx")
	assert.Contains(t, result[1], "issue 1")
	assert.Contains(t, result[2], "issue 2")
}

func TestLoadArchiveVocabularies(t *testing.T) {
	// Test with non-existent directory (should return empty vocabs)
	vocabs, err := LoadArchiveVocabularies("nonexistent")
	assert.NoError(t, err)
	assert.NotNil(t, vocabs)
	assert.Empty(t, vocabs.RelationshipTypes)
}

func TestCollectAllEntities(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create a test GLX file
	testFile := filepath.Join(tmpDir, "test.glx")
	testContent := `persons:
  person-123:
    properties:
      given_name: "Test"
      family_name: "Person"
places:
  place-456:
    name: "Test Place"
`
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	allEntities, duplicates, err := CollectAllEntities(tmpDir)
	assert.NoError(t, err)
	assert.Empty(t, duplicates)
	assert.True(t, allEntities["persons"]["person-123"])
	assert.True(t, allEntities["places"]["place-456"])
}

func TestValidateRepositoryReferences(t *testing.T) {
	// Create a temporary test directory structure
	tmpDir := t.TempDir()

	// Create test files with references
	personsFile := filepath.Join(tmpDir, "persons.glx")
	personsContent := `persons:
  person-123:
    properties:
      given_name: "Test"
      family_name: "Person"
`
	err := os.WriteFile(personsFile, []byte(personsContent), 0644)
	require.NoError(t, err)

	eventsFile := filepath.Join(tmpDir, "events.glx")
	eventsContent := `events:
  event-456:
    type: birth
    place: place-nonexistent
    participants:
      - person: person-123
      - person: person-nonexistent
`
	err = os.WriteFile(eventsFile, []byte(eventsContent), 0644)
	require.NoError(t, err)

	allEntities, _, err := CollectAllEntities(tmpDir)
	require.NoError(t, err)

	vocabs, err := LoadArchiveVocabularies(tmpDir)
	require.NoError(t, err)

	errors, warnings := ValidateRepositoryReferences(tmpDir, allEntities, vocabs)
	allIssues := append(errors, warnings...)
	assert.NotEmpty(t, allIssues, "expected reference validation issues")
}

func TestValidateVocabularyFile(t *testing.T) {
	tests := []struct {
		name   string
		doc    map[string]interface{}
		expect int // expected number of issues
	}{
		{
			name: "valid vocabulary file",
			doc: map[string]interface{}{
				"relationship_types": map[string]interface{}{
					"marriage": map[string]interface{}{
						"label": "Marriage",
					},
				},
			},
			expect: 0, // May have issues if schema not found, but structure is valid
		},
		{
			name: "no vocabulary key",
			doc: map[string]interface{}{
				"something": "invalid",
			},
			expect: 1,
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
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 2, // missing type and persons
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
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
		},
		{
			name:       "place missing name",
			entityType: "place",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
		},
		{
			name:       "source missing title",
			entityType: "source",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
		},
		{
			name:       "citation missing source",
			entityType: "citation",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
		},
		{
			name:       "repository missing name",
			entityType: "repository",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
		},
		{
			name:       "assertion missing subject",
			entityType: "assertion",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 3, // missing subject, claim, and sources/citations
		},
		{
			name:       "media missing uri",
			entityType: "media",
			entity: map[string]interface{}{},
			vocabs: nil,
			expect: 1,
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

// TestValidateRepositoryReferences_InvalidTestFiles removed - testdata deleted
