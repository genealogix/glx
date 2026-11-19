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

package glx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// warningsOn returns the messages of the warnings about one field.
func warningsOn(result *ValidationResult, field string) []string {
	var messages []string
	for _, w := range result.Warnings {
		if w.Field == field {
			messages = append(messages, w.Message)
		}
	}

	return messages
}

func TestValidateCensusSourceTypes(t *testing.T) {
	sourceTypes := map[string]*VocabularyEntry{
		SourceTypeCensus: {Label: "Census"},
		"book":           {Label: "Book"},
	}
	tests := []struct {
		name   string
		source *Source
		want   string // substring of the warning; "" means no warning
	}{
		{"census title, census type", &Source{Title: "1880 US Census", Type: SourceTypeCensus}, ""},
		{"census title, other type", &Source{Title: "1880 US Census", Type: "book"}, `has type "book"`},
		{"census title, no type", &Source{Title: "Leeds CENSUS 1851"}, "has no type"},
		{"no census in title", &Source{Title: "Parish register", Type: "book"}, ""},
		{"census type, other title", &Source{Title: "Enumeration", Type: SourceTypeCensus}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := &GLXFile{
				Sources:     map[string]*Source{"source-1": tt.source},
				SourceTypes: sourceTypes,
			}
			warnings := warningsOn(archive.Validate(), "type")
			if tt.want == "" {
				assert.Empty(t, warnings)

				return
			}
			if assert.Len(t, warnings, 1) {
				assert.Contains(t, warnings[0], tt.want)
				assert.Contains(t, warnings[0], "set type: census")
			}
		})
	}
}

func TestValidateMediaHashes(t *testing.T) {
	sha256 := strings.Repeat("a1", 32)
	tests := []struct {
		name string
		hash string
		want string // substring of the warning; "" means no warning
	}{
		{"no hash", "", ""},
		{"sha256", "sha256:" + sha256, ""},
		{"upper-case hex and algorithm", "SHA256:" + strings.ToUpper(sha256), ""},
		{"md5", "md5:" + strings.Repeat("0", 32), ""},
		{"unknown algorithm, any length", "blake3:abc123", ""},
		{"no algorithm", sha256, "should be algorithm:hexstring"},
		{"not hex", "sha256:xyz", "should be algorithm:hexstring"},
		{"empty digest", "sha256:", "should be algorithm:hexstring"},
		{"urn form", "urn:hash:sha256:" + sha256, "should be algorithm:hexstring"},
		{"sha256 too short", "sha256:abc123", "has 6 hex digits; a sha256 digest has 64"},
		{"sha1 too long", "sha1:" + sha256, "has 64 hex digits; a sha1 digest has 40"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := &GLXFile{Media: map[string]*Media{"media-1": {URI: "a.jpg", Hash: tt.hash}}}
			result := archive.Validate()
			assert.Empty(t, result.Errors, "a hash warns, never errors")
			warnings := warningsOn(result, "hash")
			if tt.want == "" {
				assert.Empty(t, warnings)

				return
			}
			if assert.Len(t, warnings, 1) {
				assert.Contains(t, warnings[0], tt.want)
			}
		})
	}
}

// A key inside `properties` that names one of the entity's own fields was
// meant to sit beside `properties` (#648).
func TestValidateMisplacedPropertyFields(t *testing.T) {
	sourceProps := map[string]*PropertyDefinition{"url": {Label: "URL", ValueType: "string"}}

	t.Run("source type and date inside properties", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{"source-1": {Title: "Births", Properties: map[string]any{
				"type": "vital_record", "date": "1900", "url": "https://example.org",
			}}},
			SourceProperties: sourceProps,
		}
		result := archive.Validate()
		for _, field := range []string{"type", "date"} {
			warnings := warningsOn(result, "properties."+field)
			if assert.Len(t, warnings, 1, field) {
				assert.Contains(t, warnings[0], "'"+field+"' is a field of the entity, not a property")
				assert.NotContains(t, warnings[0], "unknown property", "one warning per key, not two")
			}
		}
		assert.Empty(t, warningsOn(result, "properties.url"))
	})

	t.Run("a property the vocabulary defines is not misplaced", func(t *testing.T) {
		archive := &GLXFile{
			Sources:          map[string]*Source{"source-1": {Title: "Births", Properties: map[string]any{"date": "1900"}}},
			SourceProperties: map[string]*PropertyDefinition{"date": {Label: "Date", ValueType: "string"}},
		}
		assert.Empty(t, archive.Validate().Warnings)
	})

	t.Run("without a vocabulary the misplaced field is still named", func(t *testing.T) {
		archive := &GLXFile{
			Sources: map[string]*Source{"source-1": {Title: "Births", Properties: map[string]any{"type": "book"}}},
		}
		result := archive.Validate()
		assert.Len(t, warningsOn(result, "properties.type"), 1)
		assert.Empty(t, warningsOn(result, "properties"),
			"a map holding only misplaced fields does not also warn that the vocabulary is missing")
	})

	t.Run("participant fields are checked against the participant", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{"person-1": {}},
			Events: map[string]*Event{"event-1": {Type: EventTypeBirth, Participants: []Participant{
				{Person: "person-1", Properties: map[string]any{"role": "principal"}},
			}}},
			EventTypes:      map[string]*VocabularyEntry{EventTypeBirth: {Label: "Birth"}},
			EventProperties: map[string]*PropertyDefinition{"age_at_event": {Label: "Age", ValueType: "string"}},
		}
		warnings := warningsOn(archive.Validate(), "properties.role")
		if assert.Len(t, warnings, 1) {
			assert.Contains(t, warnings[0], "event-1 participants[0]")
		}
	})
}
