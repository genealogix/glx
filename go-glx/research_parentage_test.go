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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parentageTestArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"child":       {Properties: map[string]any{"name": "Child"}},
			"father-a":    {Properties: map[string]any{"name": "Alex", "sex": SexMale}},
			"father-b":    {Properties: map[string]any{"name": "Alex", "sex": SexMale}},
			"mother":      {Properties: map[string]any{"name": "Mother", "sex": SexFemale}},
			"other-child": {},
		},
		Relationships: map[string]*Relationship{
			"rel-a": {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "father-a", Role: ParticipantRoleParent}, {Person: "child", Role: ParticipantRoleChild}}},
			"rel-b": {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "father-b", Role: ParticipantRoleParent}, {Person: "child", Role: ParticipantRoleChild}}},
		},
		Assertions: map[string]*Assertion{
			"a": {Subject: EntityRef{Relationship: "rel-a"}, Confidence: "low", Status: "speculative"},
			"b": {Subject: EntityRef{Relationship: "rel-b"}, Confidence: "high", Status: "disproven"},
		},
	}
}

func TestParentageProof_SharedStanding(t *testing.T) {
	t.Run("excluded alternatives remain evidence and JSON", func(t *testing.T) {
		archive := parentageTestArchive()
		before, err := json.Marshal(archive)
		require.NoError(t, err)
		result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, ConclusionPossible, result.Conclusion)
		require.Empty(t, result.Conflicts)
		require.Len(t, result.Evidence, 2)
		require.Equal(t, []ProofExcludedAlternative{{PersonID: "father-b", Name: "Alex", Relationship: "rel-b"}}, result.Excluded)
		require.Contains(t, result.Summary, "Parents identified: Alex.")
		data, err := json.Marshal(result)
		require.NoError(t, err)
		require.Contains(t, string(data), `"alternatives_excluded":[{"person_id":"father-b","name":"Alex","relationship":"rel-b"}]`)
		result.Excluded[0].Name = "changed"
		result.Evidence[0].Status = "changed"
		after, err := json.Marshal(archive)
		require.NoError(t, err)
		require.JSONEq(t, string(before), string(after))
	})
	t.Run("all disproven cannot supply a fallback answer", func(t *testing.T) {
		archive := parentageTestArchive()
		archive.Assertions["a"].Status = "disproven"
		result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, ConclusionInsufficient, result.Conclusion)
		require.Len(t, result.Excluded, 2)
		require.NotContains(t, result.Summary, "Alex")
	})
	t.Run("participant scope leaves the other parent", func(t *testing.T) {
		archive := parentageTestArchive()
		archive.Relationships["rel-a"].Participants = append(archive.Relationships["rel-a"].Participants, Participant{Person: "mother", Role: ParticipantRoleParent}, Participant{Person: "other-child", Role: ParticipantRoleChild})
		archive.Assertions["a"].Participant = &Participant{Person: "father-a", Role: ParticipantRoleParent}
		archive.Assertions["a"].Status = "disproven"
		archive.Assertions["mother"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Participant: &Participant{Person: "mother", Role: ParticipantRoleParent}, Confidence: "medium"}
		archive.Assertions["other"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Participant: &Participant{Person: "other-child", Role: ParticipantRoleChild}, Confidence: "high", Status: "disputed"}
		result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, ConclusionProbable, result.Conclusion)
		require.Contains(t, result.Summary, "Parents identified: Mother.")
		require.NotContains(t, result.Summary, "Alex")
		require.Len(t, result.Excluded, 2)
		require.Empty(t, result.Conflicts)
		for _, evidence := range result.Evidence {
			require.NotEqual(t, "other", evidence.AssertionID)
		}
	})
	t.Run("property claims cannot prove or reject parentage", func(t *testing.T) {
		archive := parentageTestArchive()
		archive.Assertions["date"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Property: "started_on", Value: "1900", Confidence: "high", Status: "proven"}
		archive.Assertions["date-rejected"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Property: "started_on", Value: "1901", Confidence: "high", Status: "disproven"}
		result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, ConclusionPossible, result.Conclusion)
		require.Empty(t, result.Conflicts)
		require.Len(t, result.Excluded, 1)
		require.Contains(t, result.Summary, "Parents identified: Alex.")
	})
	t.Run("proven survivor overrides weak and rejected claims", func(t *testing.T) {
		archive := parentageTestArchive()
		archive.Assertions["proven"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Confidence: "low", Status: "proven"}
		archive.Assertions["rejected"] = &Assertion{Subject: EntityRef{Relationship: "rel-a"}, Confidence: "high", Status: "disproven"}
		result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, ConclusionProven, result.Conclusion)
		require.Len(t, result.Excluded, 1)
		require.Equal(t, RelationshipStandingAccepted, NewRelationshipStandingIndex(archive).Link("rel-a", "child", "father-a"))
	})
}

func TestParentageProof_CompetingBirthParents(t *testing.T) {
	for _, sex := range []string{SexMale, SexFemale} {
		t.Run(sex, func(t *testing.T) {
			archive := parentageTestArchive()
			archive.Persons["father-a"].Properties[PersonPropertySex] = sex
			archive.Persons["father-b"].Properties[PersonPropertySex] = sex
			archive.Assertions["b"].Status = "proven"
			archive.Assertions["a"].Date = "1800"
			archive.Assertions["b"].Date = "1900"
			archive.Assertions["a"].Sources = []string{"source"}
			archive.Relationships["rel-copy"] = archive.Relationships["rel-a"]
			result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, ConclusionConflicted, result.Conclusion)
			require.Len(t, result.Conflicts, 1)
			conflict := result.Conflicts[0]
			require.True(t, conflict.Definite)
			require.False(t, conflict.Resolved)
			require.Equal(t, VerdictDefinite, conflict.Verdict)
			require.Equal(t, []ConflictValue{{Value: "Alex (father-a)", Confidence: "low", Status: "speculative"}, {Value: "Alex (father-b)", Confidence: "high", Status: "proven"}}, conflict.Values)
			require.Equal(t, conflict.Verdict, conflict.Evaluation.Verdict)
			require.NotEmpty(t, conflict.Evaluation.Selected)
			for _, comparison := range conflict.Evaluation.Comparisons {
				require.Less(t, comparison.Left, len(conflict.Facts))
				require.Less(t, comparison.Right, len(conflict.Facts))
			}
			found := false
			for _, fact := range conflict.Facts {
				if fact.ID == "a" {
					found = true
					assert.Equal(t, "father-a", fact.Fact.Value)
					assert.Equal(t, "rel-a", fact.Subject.Relationship)
					assert.Equal(t, []string{"source"}, fact.Sources)
				}
			}
			require.True(t, found, "parentage comparison must retain raw claim provenance")
		})
	}
}

func TestParentageProof_NonBirthLinks(t *testing.T) {
	cases := []struct {
		name, kind, parentRole, childRole string
		excluded                          bool
	}{
		{"adoptive", RelationshipTypeAdoptiveParentChild, ParticipantRoleAdoptiveParent, ParticipantRoleAdoptedChild, false},
		{"foster", RelationshipTypeFosterParentChild, "foster_parent", "foster_child", false},
		{"step type", RelationshipTypeStepParent, ParticipantRoleParent, ParticipantRoleChild, true},
		{"step parent role", RelationshipTypeParentChild, "step_parent", ParticipantRoleChild, true},
		{"step child role", RelationshipTypeParentChild, ParticipantRoleParent, "step_child", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := parentageTestArchive()
			archive.Persons["father-b"].Properties[PersonPropertyName] = "Other"
			archive.Relationships["rel-b"].Type = tc.kind
			archive.Relationships["rel-b"].Participants = []Participant{{Person: "father-b", Role: tc.parentRole}, {Person: "child", Role: tc.childRole}}
			archive.Assertions["b"].Status = "proven"
			result, err := BuildProof(archive, "child", topicParentage, ProofOptions{})
			require.NoError(t, err)
			require.Empty(t, result.Conflicts)
			require.Empty(t, result.Excluded)
			if tc.excluded {
				require.Equal(t, ConclusionPossible, result.Conclusion)
				require.NotContains(t, result.Summary, "Other")
				require.Len(t, result.Evidence, 1)
				archive.Assertions["b"].Status = "disproven"
				result, err = BuildProof(archive, "child", topicParentage, ProofOptions{})
				require.NoError(t, err)
				require.Empty(t, result.Excluded, "step links are not parentage alternatives")
			} else {
				require.Contains(t, result.Summary, "Other")
			}
		})
	}
}
