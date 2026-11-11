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

func TestMergePersons_ReportsUnchangedClaimWithRenamedPersonTarget(t *testing.T) {
	for _, participant := range []bool{false, true} {
		name := "typed value"
		if participant {
			name = "event participant"
		}
		t.Run(name, func(t *testing.T) {
			archive := newTwoPersonArchive()
			archive.Persons["person-drop"].Properties["name"] = "New Name"
			archive.Persons["third"] = &Person{Properties: map[string]any{"name": "Child"}}
			claim := &Assertion{Subject: EntityRef{Person: "third"}, Property: "father", Value: "person-keep"}
			if participant {
				archive.Events = map[string]*Event{"birth": {Type: EventTypeBirth, Participants: []Participant{{Person: "third", Role: ParticipantRoleSubject}}}}
				claim = &Assertion{Subject: EntityRef{Event: "birth"}, Participant: &Participant{Person: "person-keep", Role: ParticipantRoleParent}}
			}
			archive.Sources = map[string]*Source{"source": {Title: "Family register", Type: SourceTypeBook}}
			claim.Sources = []string{"source"}
			archive.Assertions = map[string]*Assertion{"claim": claim}
			require.NoError(t, MergeStandardVocabularies(archive))
			archive.PersonProperties["father"] = &PropertyDefinition{Label: "Father", ReferenceType: "persons"}
			before := cloneMergeArchive(archive)
			oldEvidence, err := BuildEvidenceReport(before, claim.Subject, claim.Property, ComparisonOptions{})
			require.NoError(t, err)
			oldProof, err := BuildProof(before, "third", "parentage", ProofOptions{})
			require.NoError(t, err)
			result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
			require.NoError(t, err)
			assert.Equal(t, before.Assertions["claim"], archive.Assertions["claim"], "claim itself stays unchanged")
			newEvidence, err := BuildEvidenceReport(archive, claim.Subject, claim.Property, ComparisonOptions{})
			require.NoError(t, err)
			newProof, err := BuildProof(archive, "third", "parentage", ProofOptions{})
			require.NoError(t, err)
			require.NotEqual(t, oldEvidence, newEvidence)
			require.NotEqual(t, oldProof, newProof)
			evidenceFound, proofFound := false, false
			for _, change := range result.InterpretationChanges {
				if change.EntityType == claim.Subject.Type() && change.ID == claim.Subject.ID() && change.Aspect == "evidence/"+claim.Property {
					oldJSON, _ := json.Marshal(oldEvidence)
					newJSON, _ := json.Marshal(newEvidence)
					assert.JSONEq(t, string(oldJSON), change.Before)
					assert.JSONEq(t, string(newJSON), change.After)
					evidenceFound = true
				}
				if change.ID == "third" && change.Aspect == "proof/parentage" {
					oldJSON, _ := json.Marshal(oldProof)
					newJSON, _ := json.Marshal(newProof)
					assert.JSONEq(t, string(oldJSON), change.Before)
					assert.JSONEq(t, string(newJSON), change.After)
					proofFound = true
				}
			}
			assert.True(t, evidenceFound, "report label changed despite unchanged assertion")
			assert.True(t, proofFound, "proof label changed for another person")
		})
	}
}
