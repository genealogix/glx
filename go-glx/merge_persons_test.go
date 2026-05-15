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

func newTwoPersonArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-keep": {Properties: map[string]any{}},
			"person-drop": {Properties: map[string]any{}},
		},
	}
}

func TestMergePersons_RewritesEventParticipant(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Events = map[string]*Event{
		"event-1": {
			Type: "census",
			Participants: []Participant{
				{Person: "person-drop", Role: "subject"},
			},
		},
	}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "person-keep", glx.Events["event-1"].Participants[0].Person)
	assert.Equal(t, 1, result.RefsUpdated)
}

func TestMergePersons_RewritesRelationshipParticipant(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-other"] = &Person{}
	glx.Relationships = map[string]*Relationship{
		"rel-1": {
			Type: "parent_child",
			Participants: []Participant{
				{Person: "person-drop", Role: "parent"},
				{Person: "person-other", Role: "child"},
			},
		},
	}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "person-keep", glx.Relationships["rel-1"].Participants[0].Person)
	assert.Equal(t, "person-other", glx.Relationships["rel-1"].Participants[1].Person)
}

func TestMergePersons_RewritesAssertionSubject(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Assertions = map[string]*Assertion{
		"a-1": {
			Subject:  EntityRef{Person: "person-drop"},
			Property: "occupation",
			Value:    "blacksmith",
		},
	}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "person-keep", glx.Assertions["a-1"].Subject.Person)
}

func TestMergePersons_RewritesAssertionParticipant(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Assertions = map[string]*Assertion{
		"a-1": {
			Subject:     EntityRef{Person: "person-keep"},
			Participant: &Participant{Person: "person-drop", Role: "witness"},
		},
	}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "person-keep", glx.Assertions["a-1"].Participant.Person)
}

func TestMergePersons_DeletesDropPerson(t *testing.T) {
	glx := newTwoPersonArchive()

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Contains(t, glx.Persons, "person-keep")
	assert.NotContains(t, glx.Persons, "person-drop")
}

func TestMergePersons_MergesNonOverlappingProperties(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["sex"] = "male"
	glx.Persons["person-drop"].Properties["occupation"] = "blacksmith"

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	keep := glx.Persons["person-keep"]
	assert.Equal(t, "male", keep.Properties["sex"])
	assert.Equal(t, "blacksmith", keep.Properties["occupation"])
	assert.Equal(t, 1, result.PropertiesMerged)
	assert.Empty(t, result.Conflicts)
}

func TestMergePersons_UnionsListProperties(t *testing.T) {
	glx := newTwoPersonArchive()
	keepName := map[string]any{"value": "Hans Juncker", "date": "1750"}
	sharedName := map[string]any{"value": "Johann Jungk", "date": "1760"}
	dropExtraName := map[string]any{"value": "Hans Jungk", "date": "1755"}
	glx.Persons["person-keep"].Properties["name"] = []any{keepName, sharedName}
	glx.Persons["person-drop"].Properties["name"] = []any{sharedName, dropExtraName}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	names, ok := glx.Persons["person-keep"].Properties["name"].([]any)
	require.True(t, ok)
	assert.Len(t, names, 3, "shared entry should be deduped, dropExtraName added")
	assert.Equal(t, 1, result.PropertiesMerged, "1 new entry added (shared was deduped)")
	assert.Empty(t, result.Conflicts, "list union should not produce conflicts")
}

func TestMergePersons_DedupesWithinDropList(t *testing.T) {
	// Regression: a list property where dropList contains internal duplicates
	// must collapse those duplicates as well, not just cross-list duplicates.
	glx := newTwoPersonArchive()
	entry := map[string]any{"value": "Hans Jungk", "date": "1755"}
	glx.Persons["person-keep"].Properties["name"] = []any{}
	glx.Persons["person-drop"].Properties["name"] = []any{entry, entry, entry}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	names := glx.Persons["person-keep"].Properties["name"].([]any)
	assert.Len(t, names, 1, "duplicate entries within dropList should be collapsed")
	assert.Equal(t, 1, result.PropertiesMerged)
}

func TestMergePersons_ExternalIdsDeduped(t *testing.T) {
	glx := newTwoPersonArchive()
	fsID := map[string]any{
		"value":  "ark:/61903/abc-123",
		"fields": map[string]any{"type": "familysearch"},
	}
	wikiID := map[string]any{
		"value":  "Smith-1234",
		"fields": map[string]any{"type": "wikitree"},
	}
	glx.Persons["person-keep"].Properties["external_ids"] = []any{fsID}
	glx.Persons["person-drop"].Properties["external_ids"] = []any{fsID, wikiID}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	ids := glx.Persons["person-keep"].Properties["external_ids"].([]any)
	assert.Len(t, ids, 2)
}

func TestMergePersons_ConflictDefaultKeepsKeep(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["occupation"] = "blacksmith"
	glx.Persons["person-drop"].Properties["occupation"] = "farmer"

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "blacksmith", glx.Persons["person-keep"].Properties["occupation"])
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "occupation", result.Conflicts[0].Property)
	assert.Equal(t, ResolutionKeptKeep, result.Conflicts[0].Resolution)
}

func TestMergePersons_ConflictKeepNewest(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["residence"] = map[string]any{
		"value": "place-old", "date": "1800",
	}
	glx.Persons["person-drop"].Properties["residence"] = map[string]any{
		"value": "place-new", "date": "1820",
	}

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)

	res := glx.Persons["person-keep"].Properties["residence"].(map[string]any)
	assert.Equal(t, "place-new", res["value"])
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, ResolutionKeptNewest, result.Conflicts[0].Resolution)
}

func TestMergePersons_ConflictKeepOldest(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["residence"] = map[string]any{
		"value": "place-newer", "date": "1820",
	}
	glx.Persons["person-drop"].Properties["residence"] = map[string]any{
		"value": "place-older", "date": "1800",
	}

	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepOldest: true})
	require.NoError(t, err)

	res := glx.Persons["person-keep"].Properties["residence"].(map[string]any)
	assert.Equal(t, "place-older", res["value"])
}

func TestMergePersons_ConflictKeepNewestWithoutDates(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["occupation"] = "blacksmith"
	glx.Persons["person-drop"].Properties["occupation"] = "farmer"

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)

	assert.Equal(t, "blacksmith", glx.Persons["person-keep"].Properties["occupation"],
		"undated conflict should fall back to keeping keep")
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, ResolutionKeptKeep, result.Conflicts[0].Resolution)
}

func TestMergePersons_NotesAppend(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Notes = NoteList{"keep note"}
	glx.Persons["person-drop"].Notes = NoteList{"drop note 1", "drop note 2"}

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{NotesStrategy: NotesStrategyAppend})
	require.NoError(t, err)

	assert.Equal(t, NoteList{"keep note", "drop note 1", "drop note 2"},
		glx.Persons["person-keep"].Notes)
	assert.Equal(t, 2, result.NotesMerged)
}

func TestMergePersons_NotesPreferKeep(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Notes = NoteList{"keep note"}
	glx.Persons["person-drop"].Notes = NoteList{"drop note"}

	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{NotesStrategy: NotesStrategyPreferKeep})
	require.NoError(t, err)

	assert.Equal(t, NoteList{"keep note"}, glx.Persons["person-keep"].Notes)
}

func TestMergePersons_NotesPreferDrop(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Notes = NoteList{"keep note 1", "keep note 2"}
	glx.Persons["person-drop"].Notes = NoteList{"drop note"}

	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{NotesStrategy: NotesStrategyPreferDrop})
	require.NoError(t, err)

	assert.Equal(t, NoteList{"drop note"}, glx.Persons["person-keep"].Notes)
}

func TestMergePersons_DefaultNotesStrategyIsAppend(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Notes = NoteList{"keep"}
	glx.Persons["person-drop"].Notes = NoteList{"drop"}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, NoteList{"keep", "drop"}, glx.Persons["person-keep"].Notes)
}

func TestMergePersons_ErrorSameID(t *testing.T) {
	glx := newTwoPersonArchive()
	_, err := MergePersons(glx, "person-keep", "person-keep", MergePersonsOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "with itself")
}

func TestMergePersons_ErrorMissingKeep(t *testing.T) {
	glx := newTwoPersonArchive()
	_, err := MergePersons(glx, "person-missing", "person-drop", MergePersonsOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMergePersons_ErrorMissingDrop(t *testing.T) {
	glx := newTwoPersonArchive()
	_, err := MergePersons(glx, "person-keep", "person-missing", MergePersonsOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMergePersons_ErrorBothNewestAndOldest(t *testing.T) {
	glx := newTwoPersonArchive()
	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepNewest: true, KeepOldest: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestMergePersons_ErrorInvalidNotesStrategy(t *testing.T) {
	glx := newTwoPersonArchive()
	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{NotesStrategy: NotesStrategy("nonsense")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notes strategy")
}

func TestMergePersons_ErrorNonPersonID(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{"person-keep": {}},
		Events:  map[string]*Event{"event-1": {Type: "birth"}},
	}
	_, err := MergePersons(glx, "person-keep", "event-1", MergePersonsOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exists in events, not persons")
}

func TestMergePersons_InvalidatesValidationCache(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.validation = &ValidationResult{validated: true}

	_, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Nil(t, glx.validation)
}

func TestMergePersons_RewritesPropertiesContainingDropID(t *testing.T) {
	// Person properties can hold place IDs; if a place was incorrectly named
	// after a person ID, updateAllRefs would rewrite it. We verify that
	// MergePersons routes through updateAllRefs by setting up a deliberate
	// reference in a third person's properties.
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-keep":  {},
			"person-drop":  {},
			"person-other": {Properties: map[string]any{"residence": "person-drop"}},
		},
	}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, "person-keep", glx.Persons["person-other"].Properties["residence"])
	assert.GreaterOrEqual(t, result.RefsUpdated, 1)
}

func TestMergePersons_NoteListPreservedWhenDropHasNone(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Notes = NoteList{"existing"}

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{NotesStrategy: NotesStrategyPreferDrop})
	require.NoError(t, err)
	assert.Equal(t, NoteList{"existing"}, glx.Persons["person-keep"].Notes,
		"prefer-drop with empty drop notes should not erase keep notes")
	assert.Equal(t, 0, result.NotesMerged)
}
