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

	"github.com/stretchr/testify/require"
)

func TestGLXFile_Merge_EmptyFiles(t *testing.T) {
	// Test merging two empty files
	g1 := &GLXFile{}
	g2 := &GLXFile{}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "merging empty files should have no duplicates")
}

func TestGLXFile_Merge_Persons_NoDuplicates(t *testing.T) {
	// Test merging persons without duplicates
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{"primary_name": "Person 1"}},
		},
	}
	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-2": {Properties: map[string]any{"primary_name": "Person 2"}},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")
	require.Len(t, g1.Persons, 2, "should have merged both persons")
	require.Contains(t, g1.Persons, "person-1")
	require.Contains(t, g1.Persons, "person-2")
}

func TestGLXFile_Merge_Persons_WithDuplicates(t *testing.T) {
	// Test merging persons with duplicates
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{"primary_name": "Person 1"}},
		},
	}
	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{"primary_name": "Different Person 1"}},
			"person-2": {Properties: map[string]any{"primary_name": "Person 2"}},
		},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 1, "should detect one duplicate")
	require.Contains(t, duplicates[0], "duplicate persons ID: person-1")

	// person-2 should still be merged
	require.Len(t, g1.Persons, 2, "should merge non-duplicate persons")
	require.Contains(t, g1.Persons, "person-2")
}

func TestGLXFile_Merge_AllEntityTypes(t *testing.T) {
	// Test merging all entity types
	g1 := &GLXFile{
		Persons:       map[string]*Person{"person-1": {}},
		Events:        map[string]*Event{"event-1": {}},
		Relationships: map[string]*Relationship{"rel-1": {}},
		Places:        map[string]*Place{"place-1": {}},
		Sources:       map[string]*Source{"source-1": {}},
		Citations:     map[string]*Citation{"citation-1": {}},
		Repositories:  map[string]*Repository{"repo-1": {}},
		Assertions:    map[string]*Assertion{"assertion-1": {}},
		Media:         map[string]*Media{"media-1": {}},
	}

	g2 := &GLXFile{
		Persons:       map[string]*Person{"person-2": {}},
		Events:        map[string]*Event{"event-2": {}},
		Relationships: map[string]*Relationship{"rel-2": {}},
		Places:        map[string]*Place{"place-2": {}},
		Sources:       map[string]*Source{"source-2": {}},
		Citations:     map[string]*Citation{"citation-2": {}},
		Repositories:  map[string]*Repository{"repo-2": {}},
		Assertions:    map[string]*Assertion{"assertion-2": {}},
		Media:         map[string]*Media{"media-2": {}},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// Verify all entity types were merged
	require.Len(t, g1.Persons, 2)
	require.Len(t, g1.Events, 2)
	require.Len(t, g1.Relationships, 2)
	require.Len(t, g1.Places, 2)
	require.Len(t, g1.Sources, 2)
	require.Len(t, g1.Citations, 2)
	require.Len(t, g1.Repositories, 2)
	require.Len(t, g1.Assertions, 2)
	require.Len(t, g1.Media, 2)
}

func TestGLXFile_Merge_Vocabularies_NoDuplicates(t *testing.T) {
	// Test merging vocabularies without duplicates
	g1 := &GLXFile{
		EventTypes: map[string]*EventType{
			"birth": {Label: "Birth"},
		},
		ParticipantRoles: map[string]*ParticipantRole{
			"parent": {Label: "Parent"},
		},
	}

	g2 := &GLXFile{
		EventTypes: map[string]*EventType{
			"death": {Label: "Death"},
		},
		ParticipantRoles: map[string]*ParticipantRole{
			"child": {Label: "Child"},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")
	require.Len(t, g1.EventTypes, 2)
	require.Len(t, g1.ParticipantRoles, 2)
}

func TestGLXFile_Merge_Vocabularies_WithDuplicates(t *testing.T) {
	// Test merging vocabularies with duplicates
	g1 := &GLXFile{
		EventTypes: map[string]*EventType{
			"birth": {Label: "Birth"},
		},
	}

	g2 := &GLXFile{
		EventTypes: map[string]*EventType{
			"birth": {Label: "Different Birth"},
			"death": {Label: "Death"},
		},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 1, "should detect one duplicate")
	require.Contains(t, duplicates[0], "duplicate event_types ID: birth")

	// death should still be merged
	require.Len(t, g1.EventTypes, 2)
	require.Contains(t, g1.EventTypes, "death")
}

func TestGLXFile_Merge_AllVocabularyTypes(t *testing.T) {
	// Test merging all vocabulary types
	g1 := &GLXFile{
		EventTypes:        map[string]*EventType{"type-1": {}},
		RelationshipTypes: map[string]*RelationshipType{"type-1": {}},
		PlaceTypes:        map[string]*PlaceType{"type-1": {}},
		SourceTypes:       map[string]*SourceType{"type-1": {}},
		RepositoryTypes:   map[string]*RepositoryType{"type-1": {}},
		MediaTypes:        map[string]*MediaType{"type-1": {}},
		ParticipantRoles:  map[string]*ParticipantRole{"role-1": {}},
		ConfidenceLevels:  map[string]*ConfidenceLevel{"level-1": {}},
	}

	g2 := &GLXFile{
		EventTypes:        map[string]*EventType{"type-2": {}},
		RelationshipTypes: map[string]*RelationshipType{"type-2": {}},
		PlaceTypes:        map[string]*PlaceType{"type-2": {}},
		SourceTypes:       map[string]*SourceType{"type-2": {}},
		RepositoryTypes:   map[string]*RepositoryType{"type-2": {}},
		MediaTypes:        map[string]*MediaType{"type-2": {}},
		ParticipantRoles:  map[string]*ParticipantRole{"role-2": {}},
		ConfidenceLevels:  map[string]*ConfidenceLevel{"level-2": {}},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// Verify all vocabulary types were merged
	require.Len(t, g1.EventTypes, 2)
	require.Len(t, g1.RelationshipTypes, 2)
	require.Len(t, g1.PlaceTypes, 2)
	require.Len(t, g1.SourceTypes, 2)
	require.Len(t, g1.RepositoryTypes, 2)
	require.Len(t, g1.MediaTypes, 2)
	require.Len(t, g1.ParticipantRoles, 2)
	require.Len(t, g1.ConfidenceLevels, 2)
}

func TestGLXFile_Merge_PropertyVocabularies_NoDuplicates(t *testing.T) {
	// Test merging property vocabularies without duplicates
	g1 := &GLXFile{
		PersonProperties: map[string]*PropertyDefinition{
			"prop-1": {ValueType: "string"},
		},
		EventProperties: map[string]*PropertyDefinition{
			"prop-1": {ValueType: "string"},
		},
	}

	g2 := &GLXFile{
		PersonProperties: map[string]*PropertyDefinition{
			"prop-2": {ValueType: "number"},
		},
		EventProperties: map[string]*PropertyDefinition{
			"prop-2": {ValueType: "number"},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")
	require.Len(t, g1.PersonProperties, 2)
	require.Len(t, g1.EventProperties, 2)
}

func TestGLXFile_Merge_PropertyVocabularies_WithDuplicates(t *testing.T) {
	// Test merging property vocabularies with duplicates
	g1 := &GLXFile{
		PersonProperties: map[string]*PropertyDefinition{
			"prop-1": {ValueType: "string"},
		},
	}

	g2 := &GLXFile{
		PersonProperties: map[string]*PropertyDefinition{
			"prop-1": {ValueType: "different"},
			"prop-2": {ValueType: "number"},
		},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 1, "should detect one duplicate")
	require.Contains(t, duplicates[0], "duplicate person_properties ID: prop-1")

	// prop-2 should still be merged
	require.Len(t, g1.PersonProperties, 2)
	require.Contains(t, g1.PersonProperties, "prop-2")
}

func TestGLXFile_Merge_AllPropertyVocabularies(t *testing.T) {
	// Test merging all property vocabulary types
	g1 := &GLXFile{
		PersonProperties:       map[string]*PropertyDefinition{"prop-1": {}},
		EventProperties:        map[string]*PropertyDefinition{"prop-1": {}},
		RelationshipProperties: map[string]*PropertyDefinition{"prop-1": {}},
		PlaceProperties:        map[string]*PropertyDefinition{"prop-1": {}},
	}

	g2 := &GLXFile{
		PersonProperties:       map[string]*PropertyDefinition{"prop-2": {}},
		EventProperties:        map[string]*PropertyDefinition{"prop-2": {}},
		RelationshipProperties: map[string]*PropertyDefinition{"prop-2": {}},
		PlaceProperties:        map[string]*PropertyDefinition{"prop-2": {}},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// Verify all property vocabulary types were merged
	require.Len(t, g1.PersonProperties, 2)
	require.Len(t, g1.EventProperties, 2)
	require.Len(t, g1.RelationshipProperties, 2)
	require.Len(t, g1.PlaceProperties, 2)
}

func TestGLXFile_Merge_MultipleDuplicates(t *testing.T) {
	// Test detecting multiple duplicates across different entity types
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {},
		},
		Events: map[string]*Event{
			"event-1": {},
		},
		EventTypes: map[string]*EventType{
			"birth": {},
		},
	}

	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {}, // duplicate
			"person-2": {},
		},
		Events: map[string]*Event{
			"event-1": {}, // duplicate
			"event-2": {},
		},
		EventTypes: map[string]*EventType{
			"birth": {}, // duplicate
			"death": {},
		},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 3, "should detect three duplicates")

	// Check that all duplicates are reported
	duplicateStr := ""
	var duplicateStrSb317 strings.Builder
	for _, d := range duplicates {
		duplicateStrSb317.WriteString(d + " ")
	}
	duplicateStr += duplicateStrSb317.String()
	require.Contains(t, duplicateStr, "person-1")
	require.Contains(t, duplicateStr, "event-1")
	require.Contains(t, duplicateStr, "birth")

	// Non-duplicates should still be merged
	require.Contains(t, g1.Persons, "person-2")
	require.Contains(t, g1.Events, "event-2")
	require.Contains(t, g1.EventTypes, "death")
}

func TestGLXFile_Merge_NilMaps(t *testing.T) {
	// Test merging when maps are nil
	g1 := &GLXFile{
		Persons: nil,
		Events:  nil,
	}

	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {},
		},
		Events: map[string]*Event{
			"event-1": {},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// nil maps should not be populated (mergeMap returns early if dest is nil)
	require.Nil(t, g1.Persons)
	require.Nil(t, g1.Events)
}

func TestGLXFile_Merge_SourceNilMaps(t *testing.T) {
	// Test merging when source maps are nil
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {},
		},
		Events: map[string]*Event{
			"event-1": {},
		},
	}

	g2 := &GLXFile{
		Persons: nil,
		Events:  nil,
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// Destination should remain unchanged
	require.Len(t, g1.Persons, 1)
	require.Len(t, g1.Events, 1)
}

func TestGLXFile_Merge_MixedEntitiesAndVocabularies(t *testing.T) {
	// Test merging a mix of entities and vocabularies
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {},
		},
		Events: map[string]*Event{}, // Initialize empty map
		EventTypes: map[string]*EventType{
			"birth": {},
		},
		PersonProperties: map[string]*PropertyDefinition{
			"prop-1": {},
		},
		PlaceProperties: map[string]*PropertyDefinition{}, // Initialize empty map
	}

	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-2": {},
		},
		Events: map[string]*Event{
			"event-1": {},
		},
		EventTypes: map[string]*EventType{
			"death": {},
		},
		PersonProperties: map[string]*PropertyDefinition{
			"prop-2": {},
		},
		PlaceProperties: map[string]*PropertyDefinition{
			"prop-1": {},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")

	// Verify everything merged correctly
	require.Len(t, g1.Persons, 2)
	require.Len(t, g1.Events, 1) // g1 had empty events map
	require.Len(t, g1.EventTypes, 2)
	require.Len(t, g1.PersonProperties, 2)
	require.Len(t, g1.PlaceProperties, 1) // g1 had empty place properties map
}

func TestGLXFile_Merge_PreservesExistingData(t *testing.T) {
	// Test that merging doesn't modify existing data in g1
	g1 := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{"primary_name": "Original Name"}},
		},
	}

	g2 := &GLXFile{
		Persons: map[string]*Person{
			"person-2": {Properties: map[string]any{"primary_name": "New Person"}},
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates)

	// Verify original data is preserved
	require.Equal(t, "Original Name", g1.Persons["person-1"].Properties["primary_name"])
	require.Equal(t, "New Person", g1.Persons["person-2"].Properties["primary_name"])
}

func TestGLXFile_Merge_DuplicateReporting(t *testing.T) {
	// Test that duplicate messages are correctly formatted
	g1 := &GLXFile{
		Persons:       map[string]*Person{"person-1": {}},
		Events:        map[string]*Event{"event-1": {}},
		Relationships: map[string]*Relationship{"rel-1": {}},
		Places:        map[string]*Place{"place-1": {}},
		Sources:       map[string]*Source{"source-1": {}},
		Citations:     map[string]*Citation{"citation-1": {}},
		Repositories:  map[string]*Repository{"repo-1": {}},
		Assertions:    map[string]*Assertion{"assertion-1": {}},
		Media:         map[string]*Media{"media-1": {}},
	}

	g2 := &GLXFile{
		Persons:       map[string]*Person{"person-1": {}},
		Events:        map[string]*Event{"event-1": {}},
		Relationships: map[string]*Relationship{"rel-1": {}},
		Places:        map[string]*Place{"place-1": {}},
		Sources:       map[string]*Source{"source-1": {}},
		Citations:     map[string]*Citation{"citation-1": {}},
		Repositories:  map[string]*Repository{"repo-1": {}},
		Assertions:    map[string]*Assertion{"assertion-1": {}},
		Media:         map[string]*Media{"media-1": {}},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 9, "should detect 9 duplicates")

	// Verify messages include the entity type and ID
	duplicateStr := ""
	var duplicateStrSb476 strings.Builder
	for _, d := range duplicates {
		duplicateStrSb476.WriteString(d + "\n")
	}
	duplicateStr += duplicateStrSb476.String()

	require.Contains(t, duplicateStr, "duplicate persons ID: person-1")
	require.Contains(t, duplicateStr, "duplicate events ID: event-1")
	require.Contains(t, duplicateStr, "duplicate relationships ID: rel-1")
	require.Contains(t, duplicateStr, "duplicate places ID: place-1")
	require.Contains(t, duplicateStr, "duplicate sources ID: source-1")
	require.Contains(t, duplicateStr, "duplicate citations ID: citation-1")
	require.Contains(t, duplicateStr, "duplicate repositories ID: repo-1")
	require.Contains(t, duplicateStr, "duplicate assertions ID: assertion-1")
	require.Contains(t, duplicateStr, "duplicate media ID: media-1")
}

func TestGLXFile_Merge_Metadata_AdoptFromOther(t *testing.T) {
	// When g1 has no metadata and g2 has metadata with content, g1 adopts it.
	g1 := &GLXFile{
		Persons: map[string]*Person{},
	}
	g2 := &GLXFile{
		ImportMetadata: &Metadata{
			SourceSystem: "MyApp",
			ExportDate:   "2026-01-15",
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")
	require.NotNil(t, g1.ImportMetadata, "metadata should be adopted from other")
	require.Equal(t, "MyApp", g1.ImportMetadata.SourceSystem)
	require.Equal(t, DateString("2026-01-15"), g1.ImportMetadata.ExportDate)
}

func TestGLXFile_Merge_Metadata_DuplicateDetected(t *testing.T) {
	// When both g1 and g2 have metadata with content, a duplicate is reported.
	g1 := &GLXFile{
		Persons: map[string]*Person{},
		ImportMetadata: &Metadata{
			SourceSystem: "AppA",
		},
	}
	g2 := &GLXFile{
		ImportMetadata: &Metadata{
			SourceSystem: "AppB",
		},
	}

	duplicates := g1.Merge(g2)
	require.Len(t, duplicates, 1, "should detect one metadata duplicate")
	require.Contains(t, duplicates[0], "duplicate metadata")

	// Original metadata is preserved (first one wins)
	require.Equal(t, "AppA", g1.ImportMetadata.SourceSystem)
}

func TestGLXFile_Merge_Metadata_EmptyMetadataIgnored(t *testing.T) {
	// When g2 has a non-nil Metadata but no content, it should not be adopted.
	g1 := &GLXFile{
		Persons: map[string]*Person{},
	}
	g2 := &GLXFile{
		ImportMetadata: &Metadata{}, // all fields empty
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates")
	require.Nil(t, g1.ImportMetadata, "empty metadata should not be adopted")
}

func TestGLXFile_Merge_Metadata_NilMetadataBothSides(t *testing.T) {
	// When both sides have nil metadata, nothing happens.
	g1 := &GLXFile{
		Persons: map[string]*Person{},
	}
	g2 := &GLXFile{
		Persons: map[string]*Person{},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates)
	require.Nil(t, g1.ImportMetadata)
}

func TestGLXFile_Merge_Metadata_ExistingEmptyDoesNotConflict(t *testing.T) {
	// When g1 has empty metadata (no content) and g2 has metadata with content,
	// g2's metadata is adopted (no conflict since g1's has no content).
	g1 := &GLXFile{
		Persons:        map[string]*Person{},
		ImportMetadata: &Metadata{}, // non-nil but empty
	}
	g2 := &GLXFile{
		ImportMetadata: &Metadata{
			SourceSystem: "NewApp",
		},
	}

	duplicates := g1.Merge(g2)
	require.Empty(t, duplicates, "should have no duplicates since g1 metadata has no content")
	require.NotNil(t, g1.ImportMetadata)
	require.Equal(t, "NewApp", g1.ImportMetadata.SourceSystem)
}
