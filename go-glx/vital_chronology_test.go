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
	"github.com/stretchr/testify/require"
)

func chronologyArchive(burialType, burialDate, deathDate string) *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{"p": {Properties: map[string]any{"name": "Veronica Woerner"}}},
		Events: map[string]*Event{
			"burial": {Type: burialType, Date: DateString(burialDate), Participants: []Participant{{Person: "p", Role: "principal"}}},
			"death":  {Type: "death", Date: DateString(deathDate), Participants: []Participant{{Person: "p", Role: "principal"}}},
		},
		Sources:   map[string]*Source{"s": {Title: "Parish register", Type: "church_register"}},
		Citations: map[string]*Citation{"c": {SourceID: "s"}},
		Assertions: map[string]*Assertion{
			"burial-date": {Subject: EntityRef{Event: "burial"}, Property: "date", Value: burialDate, Citations: []string{"c"}, Confidence: "high", Status: "proven"},
			"death-date":  {Subject: EntityRef{Event: "death"}, Property: "date", Value: deathDate, Citations: []string{"c"}, Confidence: "high", Status: "proven"},
		},
	}
}

func TestVitalChronology_BoundsCalendarsAndRoles(t *testing.T) {
	for _, scenario := range []struct {
		burial, death string
		conflict      bool
	}{
		{"JULIAN 1635", "JULIAN 1653-11-01", true},
		{"1653-10-31", "1653-11-01", true},
		{"1653-11-01", "1653-11-01", false},
		{"1653-11-02", "1653-11-01", false},
		{"1653", "1653-11-01", false},
		{"BET 1635 AND 1655", "1653", false},
		{"BEF 1635", "1653", true},
		{"1635", "BEF 1653", false},
		{"ABT 1652", "1653", false},
		{"JULIAN 1645-10-12", "1645-10-22", false},
	} {
		for _, kind := range []string{"burial", "cremation"} {
			t.Run(kind+scenario.burial+scenario.death, func(t *testing.T) {
				archive := chronologyArchive(kind, scenario.burial, scenario.death)
				require.NoError(t, MergeStandardVocabularies(archive))
				conflicts := VitalChronologyConflicts(archive, "p")
				assert.Equal(t, scenario.conflict, len(conflicts) > 0)
				proof, err := BuildProof(archive, "p", "death", ProofOptions{})
				require.NoError(t, err)
				if scenario.conflict {
					assert.Equal(t, "CONFLICTED", proof.Conclusion)
				}
				archive.Events["burial"].Participants[0].Role = "parent"
				assert.Empty(t, VitalChronologyConflicts(archive, "p"), "another person's burial is not this person's death evidence")
			})
		}
	}
}

func TestVitalChronology_DisprovenDateDoesNotReappearAsStructuralFact(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	require.NoError(t, MergeStandardVocabularies(archive))
	archive.Assertions["burial-date"].Status = "disproven"
	assert.Empty(t, VitalChronologyConflicts(archive, "p"))
}

func TestProof_VitalParticipantEvidenceBelongsToTarget(t *testing.T) {
	for _, topic := range []string{"birth", "death"} {
		archive := chronologyArchive("burial", "1653", "1653")
		delete(archive.Events, "death")
		if topic == "birth" {
			archive.Events["burial"].Type = "baptism"
		}
		archive.Persons["godmother"] = &Person{Properties: map[string]any{"name": "Elsa"}}
		archive.Events["burial"].Participants = append(archive.Events["burial"].Participants, Participant{Person: "godmother", Role: "godparent"})
		archive.Assertions = map[string]*Assertion{"godmother": {Subject: EntityRef{Event: "burial"}, Participant: &Participant{Person: "godmother", Role: "godparent"}, Citations: []string{"c"}, Confidence: "high", Status: "proven"}}
		require.NoError(t, MergeStandardVocabularies(archive))
		proof, err := BuildProof(archive, "p", topic, ProofOptions{})
		require.NoError(t, err)
		assert.Empty(t, proof.Evidence)
		assert.Equal(t, "INSUFFICIENT EVIDENCE", proof.Conclusion)
	}
}

func TestProof_ProxyVitalDatesAreQualified(t *testing.T) {
	for _, scenario := range []struct{ eventType, question, expected string }{
		{"baptism", "birth", "born on or before 1690-01-03 (baptism)"},
		{"christening", "birth", "born on or before 1690-01-03 (christening)"},
		{"burial", "death", "died on or before 1690-01-03 (burial)"},
		{"cremation", "death", "died on or before 1690-01-03 (cremation)"},
	} {
		t.Run(scenario.eventType, func(t *testing.T) {
			archive := chronologyArchive(scenario.eventType, "1690-01-03", "")
			delete(archive.Events, "death")
			delete(archive.Assertions, "death-date")
			require.NoError(t, MergeStandardVocabularies(archive))
			proof, err := BuildProof(archive, "p", scenario.question, ProofOptions{})
			require.NoError(t, err)
			assert.Equal(t, "PROVEN", proof.Conclusion)
			assert.Contains(t, proof.Summary, scenario.expected)
			found := false
			for _, gap := range proof.Gaps {
				if strings.Contains(gap.Label, "date not established") {
					found = true
				}
			}
			assert.True(t, found)
		})
	}
}

func TestMergePersons_PreviewInterpretationExposesChronologyConflict(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	archive.Persons["q"] = &Person{Properties: map[string]any{"name": "Fronica Joachim"}}
	archive.Events["burial"].Participants[0].Person = "q"
	require.NoError(t, MergeStandardVocabularies(archive))
	result, err := MergePersons(archive, "p", "q", MergePersonsOptions{})
	require.NoError(t, err)
	found := false
	for _, change := range result.InterpretationChanges {
		if change.ID == "p" && change.Aspect == "proof/death" {
			assert.Contains(t, change.After, `"conclusion":"CONFLICTED"`)
			assert.Contains(t, change.After, "Death/burial chronology")
			found = true
		}
	}
	assert.True(t, found)
}

func TestProof_JulianAndGregorianEvidenceForTheSameDayAgrees(t *testing.T) {
	archive := chronologyArchive("birth", "JULIAN 1645-10-12", "1645-10-22")
	archive.Events["death"].Type = "birth"
	require.NoError(t, MergeStandardVocabularies(archive))
	proof, err := BuildProof(archive, "p", "birth", ProofOptions{})
	require.NoError(t, err)
	assert.Empty(t, proof.Conflicts)
	assert.Equal(t, "PROVEN", proof.Conclusion)
}

func TestProof_ExplicitBirthDateInBaptismAnswersExactly(t *testing.T) {
	archive := chronologyArchive("baptism", "1690-01-03", "")
	delete(archive.Events, "death")
	delete(archive.Assertions, "death-date")
	archive.Assertions["actual-birth"] = &Assertion{Subject: EntityRef{Event: "burial"}, Property: "birth_date", Value: "1690-01-01", Status: "proven", Confidence: "high", Citations: []string{"c"}}
	require.NoError(t, MergeStandardVocabularies(archive))
	proof, err := BuildProof(archive, "p", "birth", ProofOptions{})
	require.NoError(t, err)
	assert.Contains(t, proof.Summary, "1690-01-01")
	assert.NotContains(t, proof.Summary, "on or before")
	for _, gap := range proof.Gaps {
		assert.NotEqual(t, "Birth date not established", gap.Label)
	}
}
