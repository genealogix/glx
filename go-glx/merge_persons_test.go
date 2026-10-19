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
	"fmt"
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

func TestMergePersons_RemovesResolvedIdentityCandidates(t *testing.T) {
	for _, keepID := range []string{"person-keep", "person-drop"} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("keep=%s/reverse=%t", keepID, reverse), func(t *testing.T) {
				archive := newTwoPersonArchive()
				dropID := "person-drop"
				if keepID == dropID {
					dropID = "person-keep"
				}
				first, second := "person-keep", "person-drop"
				if reverse {
					first, second = second, first
				}
				archive.Relationships = map[string]*Relationship{
					"rel-b": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
						{Person: first}, {Person: second},
					}},
					"rel-a": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
						{Person: second, Role: "subject"}, {Person: first, Role: "subject"},
					}},
				}

				result, err := MergePersons(archive, keepID, dropID, MergePersonsOptions{})
				require.NoError(t, err)

				assert.Empty(t, archive.Relationships, "resolved candidates must not become self-links")
				assert.Equal(t, []string{"rel-a", "rel-b"}, result.RemovedRelationships)
				assert.Zero(t, result.RefsUpdated, "discarded relationship references were not rewritten")
				assert.Contains(t, archive.Persons, keepID)
				assert.NotContains(t, archive.Persons, dropID)
			})
		}
	}
}

func TestMergePersons_PreservesOtherRelationshipsAndEvidence(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.Persons["person-other"] = &Person{}
	archive.Relationships = map[string]*Relationship{
		"rel-resolved": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "person-keep"}, {Person: "person-drop"},
		}},
		"rel-family": {Type: RelationshipTypeParentChild, Participants: []Participant{
			{Person: "person-drop", Role: "parent", Notes: NoteList{"original participant"}},
			{Person: "person-other", Role: "child"},
		}, Notes: NoteList{"family evidence"}},
		"rel-pair-other-type": {Type: RelationshipTypeAssociate, Participants: []Participant{
			{Person: "person-keep"}, {Person: "person-drop"},
		}},
		"rel-other-candidate": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "person-drop"}, {Person: "person-other"},
		}},
		"rel-other-keep": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "person-keep"}, {Person: "person-other"},
		}},
		"rel-group": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "person-drop"}, {Person: "person-keep"}, {Person: "person-other"},
		}},
		"rel-existing-self": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "person-drop"}, {Person: "person-drop"},
		}},
		"rel-incomplete": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-drop"}}},
		"rel-nil":        nil,
	}
	archive.Sources = map[string]*Source{"src-1": {Title: "Original source"}}
	archive.Citations = map[string]*Citation{"cit-1": {SourceID: "src-1"}}
	archive.Media = map[string]*Media{"media-1": {URI: "record.jpg", Source: "src-1"}}
	archive.Assertions = map[string]*Assertion{
		"a-family": {
			Subject: EntityRef{Relationship: "rel-family"}, Sources: []string{"src-1"},
			Citations: []string{"cit-1"}, Media: []string{"media-1"}, Status: "proven", Notes: NoteList{"family evidence"},
		},
		"a-person": {Subject: EntityRef{Person: "person-drop"}, Property: "occupation", Value: "smith", Citations: []string{"cit-1"}},
	}

	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{"rel-resolved"}, result.RemovedRelationships)
	assert.Len(t, archive.Relationships, 8)
	assert.Equal(t, 8, result.RefsUpdated)
	assert.Equal(t, &Relationship{Type: RelationshipTypeParentChild, Participants: []Participant{
		{Person: "person-keep", Role: "parent", Notes: NoteList{"original participant"}},
		{Person: "person-other", Role: "child"},
	}, Notes: NoteList{"family evidence"}}, archive.Relationships["rel-family"])
	for _, id := range []string{"rel-pair-other-type", "rel-other-candidate", "rel-group", "rel-existing-self", "rel-incomplete"} {
		assert.Equal(t, "person-keep", archive.Relationships[id].Participants[0].Person)
	}
	assert.Equal(t, "person-other", archive.Relationships["rel-other-candidate"].Participants[1].Person)
	assert.Equal(t, "person-other", archive.Relationships["rel-group"].Participants[2].Person)
	assert.Nil(t, archive.Relationships["rel-nil"])
	assert.Equal(t, &Source{Title: "Original source"}, archive.Sources["src-1"])
	assert.Equal(t, &Citation{SourceID: "src-1"}, archive.Citations["cit-1"])
	assert.Equal(t, &Media{URI: "record.jpg", Source: "src-1"}, archive.Media["media-1"])
	assert.Equal(t, &Assertion{
		Subject: EntityRef{Relationship: "rel-family"}, Sources: []string{"src-1"},
		Citations: []string{"cit-1"}, Media: []string{"media-1"}, Status: "proven", Notes: NoteList{"family evidence"},
	}, archive.Assertions["a-family"])
	assert.Equal(t, &Assertion{Subject: EntityRef{Person: "person-keep"}, Property: "occupation", Value: "smith", Citations: []string{"cit-1"}}, archive.Assertions["a-person"])
}

func TestMergePersons_NoResolvedIdentityCandidates(t *testing.T) {
	archive := newTwoPersonArchive()
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.RemovedRelationships)
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
	glx.Persons["person-keep"].Properties["fixed_trade"] = "blacksmith"
	glx.Persons["person-drop"].Properties["fixed_trade"] = "farmer"

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)

	assert.Equal(t, "blacksmith", glx.Persons["person-keep"].Properties["fixed_trade"])
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, "fixed_trade", result.Conflicts[0].Property)
	assert.Equal(t, ResolutionKeptKeep, result.Conflicts[0].Resolution)
}

func TestMergePersons_ConflictKeepNewest(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["fixed_residence"] = map[string]any{
		"value": "place-old", "date": "1800",
	}
	glx.Persons["person-drop"].Properties["fixed_residence"] = map[string]any{
		"value": "place-new", "date": "1820",
	}

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)

	res := glx.Persons["person-keep"].Properties["fixed_residence"].(map[string]any)
	assert.Equal(t, "place-new", res["value"])
	require.Len(t, result.Conflicts, 1)
	assert.Equal(t, ResolutionKeptNewest, result.Conflicts[0].Resolution)
}

func TestMergePersons_ConflictKeepOldest(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["fixed_residence"] = map[string]any{
		"value": "place-newer", "date": "1820",
	}
	glx.Persons["person-drop"].Properties["fixed_residence"] = map[string]any{
		"value": "place-older", "date": "1800",
	}

	_, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepOldest: true})
	require.NoError(t, err)

	res := glx.Persons["person-keep"].Properties["fixed_residence"].(map[string]any)
	assert.Equal(t, "place-older", res["value"])
}

func TestMergePersons_ConflictKeepNewestWithoutDates(t *testing.T) {
	glx := newTwoPersonArchive()
	glx.Persons["person-keep"].Properties["fixed_trade"] = "blacksmith"
	glx.Persons["person-drop"].Properties["fixed_trade"] = "farmer"

	result, err := MergePersons(glx, "person-keep", "person-drop",
		MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)

	assert.Equal(t, "blacksmith", glx.Persons["person-keep"].Properties["fixed_trade"],
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
	// A declared person reference follows the merge; ordinary strings and
	// references to other entity types must not be rewritten by ID equality.
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-keep":  {},
			"person-drop":  {},
			"person-other": {Properties: map[string]any{"mentor": "person-drop"}},
		},
		PersonProperties: map[string]*PropertyDefinition{"mentor": {ReferenceType: "persons"}},
	}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, "person-keep", glx.Persons["person-other"].Properties["mentor"])
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

func TestMergePersons_TemporalScalarMatchesWrappedEntry(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-keep": {Properties: map[string]any{"occupation": "Farmer"}},
			"person-drop": {Properties: map[string]any{"occupation": []any{map[string]any{"value": "Farmer"}}}},
		},
	}

	result, err := MergePersons(glx, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Farmer", glx.Persons["person-keep"].Properties["occupation"],
		"an undated scalar and its {value: ...} form are the same claim")
	assert.Equal(t, 0, result.PropertiesMerged)
	assert.Empty(t, result.Conflicts)
}
