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
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errMergeReviewValueChanged = errors.New("caller value changed")

func TestMergePersonsRejectsParticipantCollapseWithoutMutation(t *testing.T) {
	for _, kind := range []string{RelationshipTypeMarriage, RelationshipTypeParentChild, RelationshipTypeSibling, RelationshipTypeAssociate, "custom_connection", RelationshipTypePossiblySamePerson} {
		for _, people := range [][]string{{"person-keep", "person-drop"}, {"person-drop", "person-keep", "third"}, {"person-drop", "person-keep", "person-drop"}} {
			if kind == RelationshipTypePossiblySamePerson && len(people) == 2 {
				continue
			}
			t.Run(fmt.Sprintf("%s/%v", kind, people), func(t *testing.T) {
				archive := newTwoPersonArchive()
				archive.Persons["third"] = &Person{}
				relationship := &Relationship{Type: kind, Notes: NoteList{"preserve this evidence"}}
				for _, id := range people {
					relationship.Participants = append(relationship.Participants, Participant{Person: id})
				}
				archive.Relationships = map[string]*Relationship{"relation": relationship}
				before := cloneMergeArchive(archive)
				_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
				require.ErrorIs(t, err, ErrValidationFailed)
				require.Contains(t, err.Error(), "relation")
				require.Contains(t, err.Error(), "collapsing participants")
				assert.Equal(t, before, archive, "refusal must retain the complete original evidence")
			})
		}
	}
}

func TestMergePersonsRetainsPreexistingSelfLink(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.Relationships = map[string]*Relationship{"old": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "person-drop"}, {Person: "person-drop"}}, Notes: NoteList{"existing research"}}}
	_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, []Participant{{Person: "person-keep"}, {Person: "person-keep"}}, archive.Relationships["old"].Participants)
	assert.Equal(t, NoteList{"existing research"}, archive.Relationships["old"].Notes)
}

func mergeParticipantReviewArchive() *GLXFile {
	archive := newTwoPersonArchive()
	archive.Persons["third"] = &Person{}
	archive.Relationships = map[string]*Relationship{
		"identity": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-keep"}, {Person: "person-drop"}}},
		"other":    {Type: RelationshipTypeAssociate, Participants: []Participant{{Person: "person-keep"}, {Person: "third"}}},
	}
	archive.EventProperties = map[string]*PropertyDefinition{
		"linked_person":       {Label: "Linked person", ReferenceType: "persons"},
		"linked_relationship": {Label: "Linked relationship", ReferenceType: "relationships"},
		"linked_assertion":    {Label: "Linked assertion", ReferenceType: "assertions"},
	}

	return archive
}

func TestMergePersonsAssertionParticipantUsesEventVocabulary(t *testing.T) {
	for _, dependent := range []bool{false, true} {
		t.Run(fmt.Sprintf("dependent=%t", dependent), func(t *testing.T) {
			archive := mergeParticipantReviewArchive()
			properties := map[string]any{"linked_person": "person-drop"}
			if dependent {
				properties["linked_relationship"] = "identity"
			}
			archive.Assertions = map[string]*Assertion{
				"claim":     {Subject: EntityRef{Relationship: "other"}, Participant: &Participant{Person: "person-keep", Properties: properties}},
				"dependent": {Subject: EntityRef{Relationship: "other"}, Participant: &Participant{Person: "person-keep", Properties: map[string]any{"linked_assertion": "claim"}}},
			}
			require.NoError(t, MergeStandardVocabularies(archive))
			require.Empty(t, archive.Validate().Errors, "fixture must be valid before merging")
			result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
			require.NoError(t, err)
			if dependent {
				assert.Equal(t, []string{"claim", "dependent"}, result.RemovedAssertions)
				assert.Empty(t, archive.Assertions)
			} else {
				assert.Empty(t, result.RemovedAssertions)
				assert.Equal(t, "person-keep", archive.Assertions["claim"].Participant.Properties["linked_person"])
				assert.Equal(t, "claim", archive.Assertions["dependent"].Participant.Properties["linked_assertion"])
			}
			require.Empty(t, archive.Validate().Errors, "no dangling participant qualifiers may be published")
		})
	}
}

func TestMergePersonsAssertionValueKeepsSubjectVocabulary(t *testing.T) {
	archive := mergeParticipantReviewArchive()
	archive.RelationshipProperties = map[string]*PropertyDefinition{"linked": {Label: "Linked person", ReferenceType: "persons"}}
	archive.EventProperties["linked"] = &PropertyDefinition{Label: "Linked source", ReferenceType: "sources"}
	archive.Sources = map[string]*Source{"source": {Title: "Unrelated record"}}
	archive.Assertions = map[string]*Assertion{
		"value":       {Subject: EntityRef{Relationship: "other"}, Property: "linked", Value: "person-drop"},
		"participant": {Subject: EntityRef{Relationship: "other"}, Participant: &Participant{Person: "person-keep", Properties: map[string]any{"linked": "source"}}},
	}
	require.NoError(t, MergeStandardVocabularies(archive))
	require.Empty(t, archive.Validate().Errors)
	_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, "person-keep", archive.Assertions["value"].Value)
	assert.Equal(t, "source", archive.Assertions["participant"].Participant.Properties["linked"])
	require.Empty(t, archive.Validate().Errors)
}

func TestMergeReferenceCheckDoesNotReplaceCallerValuesOnFailure(t *testing.T) {
	before := newTwoPersonArchive()
	before.PersonProperties = map[string]*PropertyDefinition{"typed": {ReferenceType: "persons"}}
	original := []string{"person-drop"}
	before.Persons["person-keep"].Properties["typed"] = original
	after := cloneMergeArchive(before)
	after.Persons["person-keep"].Properties["typed"] = []string{"missing"}
	require.ErrorIs(t, validateMergeReferences(before, after), ErrValidationFailed)
	retained := before.Persons["person-keep"].Properties["typed"].([]string)
	assert.Equal(t, reflect.ValueOf(original).Pointer(), reflect.ValueOf(retained).Pointer(), "even an equal replacement mutates caller-owned state")
	assert.Equal(t, original, retained)
}

func TestMergeReferenceCheckSupportsConcurrentReaders(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.PersonProperties = map[string]*PropertyDefinition{"typed": {ReferenceType: "persons"}}
	archive.Persons["person-keep"].Properties["typed"] = []string{"person-drop"}
	expected := cloneMergeArchive(archive)
	_, err := missingMergeTargets(archive)
	require.NoError(t, err)
	var workers sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		workers.Go(func() {
			for range 50 {
				if _, err := missingMergeTargets(archive); err != nil {
					failures <- err

					return
				}
				value := archive.Persons["person-keep"].Properties["typed"].([]string)
				if len(value) != 1 || value[0] != "person-drop" {
					failures <- errMergeReviewValueChanged

					return
				}
			}
		})
	}
	workers.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	assert.Equal(t, expected, archive)
}

func TestMergeReferenceCheckExcludesVocabularyCollections(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.Assertions = map[string]*Assertion{"claim": {Subject: EntityRef{Person: "person-keep"}, Property: "occupation", Value: "smith", Confidence: ConfidenceLevelHigh}}
	missing, err := missingMergeTargets(archive)
	require.NoError(t, err)
	assert.NotContains(t, missing, "confidence_levels:high")
	for _, kind := range []EntityType{"confidence_levels", "search_result_types", "person_properties", "metadata"} {
		assert.False(t, mergeEntityCollection(archive, kind).IsValid())
	}
}
