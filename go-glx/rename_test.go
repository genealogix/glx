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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameEntity_PersonMapKey(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-old": {Properties: map[string]any{"name": "Test"}},
		},
	}
	

	result, err := RenameEntity(glx, "person-old", "person-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Persons, "person-new")
	assert.NotContains(t, glx.Persons, "person-old")
	assert.Greater(t, result.RefsUpdated, 0)
}

func TestRenameEntity_PersonRefsInEvents(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-old": {Properties: map[string]any{"name": "Test"}},
		},
		Events: map[string]*Event{
			"event-1": {
				Type: "census",
				Participants: []Participant{
					{Person: "person-old", Role: "subject"},
				},
			},
		},
	}
	

	_, err := RenameEntity(glx, "person-old", "person-new")
	require.NoError(t, err)
	assert.Equal(t, "person-new", glx.Events["event-1"].Participants[0].Person)
}

func TestRenameEntity_PersonRefsInRelationships(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-old": {},
			"person-other": {},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-old", Role: "parent"},
					{Person: "person-other", Role: "child"},
				},
			},
		},
	}
	

	_, err := RenameEntity(glx, "person-old", "person-new")
	require.NoError(t, err)
	assert.Equal(t, "person-new", glx.Relationships["rel-1"].Participants[0].Person)
	assert.Equal(t, "person-other", glx.Relationships["rel-1"].Participants[1].Person)
}

func TestRenameEntity_PersonRefsInAssertions(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-old": {},
		},
		Assertions: map[string]*Assertion{
			"a-1": {
				Subject:  EntityRef{Person: "person-old"},
				Property: "occupation",
				Value:    "blacksmith",
			},
			"a-2": {
				Participant: &Participant{Person: "person-old", Role: "witness"},
			},
		},
	}
	

	_, err := RenameEntity(glx, "person-old", "person-new")
	require.NoError(t, err)
	assert.Equal(t, "person-new", glx.Assertions["a-1"].Subject.Person)
	assert.Equal(t, "person-new", glx.Assertions["a-2"].Participant.Person)
}

func TestRenameEntity_PlaceRefs(t *testing.T) {
	glx := &GLXFile{
		Places: map[string]*Place{
			"place-old":   {Name: "Old Place"},
			"place-child": {Name: "Child", ParentID: "place-old"},
		},
		Events: map[string]*Event{
			"event-1": {Type: "birth", PlaceID: "place-old"},
		},
		Assertions: map[string]*Assertion{
			"a-1": {Subject: EntityRef{Place: "place-old"}},
		},
	}
	

	_, err := RenameEntity(glx, "place-old", "place-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Places, "place-new")
	assert.Equal(t, "place-new", glx.Places["place-child"].ParentID)
	assert.Equal(t, "place-new", glx.Events["event-1"].PlaceID)
	assert.Equal(t, "place-new", glx.Assertions["a-1"].Subject.Place)
}

func TestRenameEntity_SourceRefs(t *testing.T) {
	glx := &GLXFile{
		Sources: map[string]*Source{
			"source-old": {Title: "Test Source"},
		},
		Citations: map[string]*Citation{
			"cit-1": {SourceID: "source-old"},
		},
		Assertions: map[string]*Assertion{
			"a-1": {Sources: []string{"source-old", "source-other"}},
		},
		Media: map[string]*Media{
			"media-1": {Source: "source-old"},
		},
	}
	

	_, err := RenameEntity(glx, "source-old", "source-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Sources, "source-new")
	assert.Equal(t, "source-new", glx.Citations["cit-1"].SourceID)
	assert.Equal(t, "source-new", glx.Assertions["a-1"].Sources[0])
	assert.Equal(t, "source-new", glx.Media["media-1"].Source)
}

func TestRenameEntity_EventRefs(t *testing.T) {
	glx := &GLXFile{
		Events: map[string]*Event{
			"event-old": {Type: "marriage"},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {StartEvent: "event-old", EndEvent: "event-old"},
		},
		Assertions: map[string]*Assertion{
			"a-1": {Subject: EntityRef{Event: "event-old"}},
		},
	}
	

	_, err := RenameEntity(glx, "event-old", "event-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Events, "event-new")
	assert.Equal(t, "event-new", glx.Relationships["rel-1"].StartEvent)
	assert.Equal(t, "event-new", glx.Relationships["rel-1"].EndEvent)
	assert.Equal(t, "event-new", glx.Assertions["a-1"].Subject.Event)
}

func TestRenameEntity_NotFound(t *testing.T) {
	glx := &GLXFile{}
	

	_, err := RenameEntity(glx, "nonexistent", "new-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenameEntity_TargetExists(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-old": {},
			"person-new": {},
		},
	}
	

	_, err := RenameEntity(glx, "person-old", "person-new")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRenameEntity_CitationAndRepoRefs(t *testing.T) {
	glx := &GLXFile{
		Citations: map[string]*Citation{
			"cit-old": {SourceID: "src-1"},
		},
		Assertions: map[string]*Assertion{
			"a-1": {Citations: []string{"cit-old"}},
		},
		Repositories: map[string]*Repository{
			"repo-old": {Name: "Archive"},
		},
		Sources: map[string]*Source{
			"src-1": {RepositoryID: "repo-old"},
		},
	}
	

	// Rename citation
	_, err := RenameEntity(glx, "cit-old", "cit-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Citations, "cit-new")
	assert.Equal(t, "cit-new", glx.Assertions["a-1"].Citations[0])

	// Rename repository
	_, err = RenameEntity(glx, "repo-old", "repo-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Repositories, "repo-new")
	assert.Equal(t, "repo-new", glx.Sources["src-1"].RepositoryID)
}

func TestRenameEntity_MediaRefs(t *testing.T) {
	glx := &GLXFile{
		Media: map[string]*Media{
			"media-old": {Type: "photo"},
		},
		Sources: map[string]*Source{
			"src-1": {Media: []string{"media-old", "media-other"}},
		},
		Citations: map[string]*Citation{
			"cit-1": {Media: []string{"media-old"}},
		},
		Assertions: map[string]*Assertion{
			"a-1": {Media: []string{"media-old"}},
		},
	}
	

	_, err := RenameEntity(glx, "media-old", "media-new")
	require.NoError(t, err)
	assert.Contains(t, glx.Media, "media-new")
	assert.Equal(t, "media-new", glx.Sources["src-1"].Media[0])
	assert.Equal(t, "media-new", glx.Citations["cit-1"].Media[0])
	assert.Equal(t, "media-new", glx.Assertions["a-1"].Media[0])
}

func TestRenameEntity_PropertyBasedRefs(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{
				"name":      "Jane",
				"residence": "place-old",
				"workplace": map[string]any{"value": "place-old"},
			}},
		},
		Places: map[string]*Place{
			"place-old": {Name: "Old Place"},
		},
	}

	_, err := RenameEntity(glx, "place-old", "place-new")
	require.NoError(t, err)
	assert.Equal(t, "place-new", glx.Persons["person-1"].Properties["residence"])
	workplace := glx.Persons["person-1"].Properties["workplace"].(map[string]any)
	assert.Equal(t, "place-new", workplace["value"])
}

func TestRenameEntity_AssertionValue(t *testing.T) {
	glx := &GLXFile{
		Places: map[string]*Place{
			"place-old": {Name: "Old Place"},
		},
		Assertions: map[string]*Assertion{
			"a-1": {
				Subject:  EntityRef{Person: "person-1"},
				Property: "residence",
				Value:    "place-old",
			},
		},
	}

	_, err := RenameEntity(glx, "place-old", "place-new")
	require.NoError(t, err)
	assert.Equal(t, "place-new", glx.Assertions["a-1"].Value)
}
