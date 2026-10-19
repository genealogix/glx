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

func TestMergePersons_CascadesDeletedAssertionReferences(t *testing.T) {
	archive := identityEvidenceFixture(t)
	archive.PersonProperties["supporting_claim"] = &PropertyDefinition{Label: "Supporting claim", ReferenceType: "assertions", MultiValue: new(true)}
	archive.EventProperties["supporting_claim"] = &PropertyDefinition{Label: "Supporting claim", ReferenceType: "assertions", MultiValue: new(true)}
	archive.PersonProperties["literal"] = &PropertyDefinition{Label: "Literal", ValueType: "string"}
	archive.Assertions = map[string]*Assertion{
		"z-identity":    {Subject: EntityRef{Relationship: "r-resolved"}, Sources: []string{"s"}, Notes: NoteList{"Identity"}},
		"a-dependent":   {Subject: EntityRef{Person: "person-third"}, Property: "supporting_claim", Value: "z-identity", Sources: []string{"s"}, Status: "proven", Confidence: ConfidenceLevelHigh},
		"b-transitive":  {Subject: EntityRef{Person: "person-third"}, Property: "supporting_claim", Value: "a-dependent", Sources: []string{"s"}},
		"c-participant": {Subject: EntityRef{Event: "e"}, Participant: &Participant{Person: "person-third", Properties: map[string]any{"supporting_claim": "b-transitive"}}, Sources: []string{"s"}},
		"d-qualified":   {Subject: EntityRef{Event: "e"}, Participant: &Participant{Person: "person-third", Properties: map[string]any{"supporting_claim": []any{"retained", "c-participant"}}}, Sources: []string{"s"}},
		"reachable-a":   {Subject: EntityRef{Event: "e"}, Participant: &Participant{Person: "person-third", Properties: map[string]any{"supporting_claim": []any{"z-identity", "reachable-b"}}}, Sources: []string{"s"}},
		"reachable-b":   {Subject: EntityRef{Person: "person-third"}, Property: "supporting_claim", Value: "reachable-a", Sources: []string{"s"}},
		"retained":      {Subject: EntityRef{Person: "person-third"}, Sources: []string{"s"}, Notes: NoteList{"Unrelated"}},
		"cycle-a":       {Subject: EntityRef{Person: "person-third"}, Property: "supporting_claim", Value: "cycle-b", Sources: []string{"s"}},
		"cycle-b":       {Subject: EntityRef{Person: "person-third"}, Property: "supporting_claim", Value: "cycle-a", Sources: []string{"s"}},
		"literal":       {Subject: EntityRef{Person: "person-third"}, Property: "literal", Value: "z-identity", Sources: []string{"s"}},
	}
	archive.Persons["person-third"].Properties = map[string]any{"supporting_claim": []any{"b-transitive", "retained", "d-qualified"}}
	archive.ResearchLogs["l"].Leads = []ResearchLead{{Description: "Keep research", Assertions: []string{"a-dependent", "b-transitive", "c-participant", "d-qualified", "retained"}}}
	before := cloneMergeArchive(archive)
	require.Empty(t, archive.Validate().Errors)
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"a-dependent", "b-transitive", "c-participant", "d-qualified", "reachable-a", "reachable-b", "z-identity"}, result.RemovedAssertions)
	for _, id := range result.RemovedAssertions {
		assert.NotContains(t, archive.Assertions, id)
	}
	for _, id := range []string{"cycle-a", "cycle-b", "retained", "literal"} {
		assert.Equal(t, before.Assertions[id], archive.Assertions[id])
	}
	assert.Equal(t, []any{"retained"}, archive.Persons["person-third"].Properties["supporting_claim"])
	assert.Equal(t, []string{"retained"}, archive.ResearchLogs["l"].Leads[0].Assertions)
	assert.Equal(t, "Keep research", archive.ResearchLogs["l"].Leads[0].Description)
	require.Empty(t, archive.Validate().Errors)
	removedValues := map[string]bool{}
	for _, reference := range result.ReferenceChanges {
		if reference.Action == "removed-assertion" && reference.TargetType == EntityTypeAssertions {
			assert.Nil(t, reference.NewValue)
			removedValues[reference.ID] = true
		}
	}
	for _, id := range []string{"a-dependent", "b-transitive", "c-participant", "d-qualified", "reachable-a", "reachable-b"} {
		assert.True(t, removedValues[id], id)
	}
}
