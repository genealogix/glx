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

func TestVitalReview_RejectedDateCannotReturnThroughAPlaceAssertion(t *testing.T) {
	for _, topic := range []string{"birth", "death"} {
		archive := chronologyArchive("burial", "1653-11-03", "1653-11-01")
		if topic == "birth" {
			archive.Events["burial"].Type = "baptism"
			archive.Events["death"].Type = "birth"
		}
		archive.Assertions["death-date"].Status = "disproven"
		archive.Places = map[string]*Place{"place": {Name: "Pohl-Gons"}}
		archive.Events["death"].PlaceID = "place"
		archive.Assertions["primary-place"] = &Assertion{Subject: EntityRef{Event: "death"}, Property: "place", Value: "place", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
		require.NoError(t, MergeStandardVocabularies(archive))
		proof, err := BuildProof(archive, "p", topic, ProofOptions{})
		require.NoError(t, err)
		assert.NotContains(t, proof.Summary, "1653-11-01")
		assert.Contains(t, proof.Summary, "on or before 1653-11-03")
		assert.Contains(t, proof.Summary, "Pohl-Gons")
		assert.Equal(t, DateString("1653-11-01"), archive.Events["death"].Date, "the reporting path must not mutate the archive")
		found := false
		for _, gap := range proof.Gaps {
			if gap.Label == "Death date not established" || gap.Label == "Birth date not established" {
				found = true
			}
		}
		assert.True(t, found)
	}
}

func TestVitalReview_RejectedStructuralDateYieldsToAcceptedAssertion(t *testing.T) {
	archive := chronologyArchive("burial", "1653-11-03", "1653-11-01")
	archive.Assertions["death-date"].Status = "disproven"
	archive.Assertions["accepted"] = &Assertion{Subject: EntityRef{Event: "death"}, Property: "date", Value: "1653-11-02", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
	require.NoError(t, MergeStandardVocabularies(archive))
	proof, err := BuildProof(archive, "p", "death", ProofOptions{})
	require.NoError(t, err)
	assert.NotContains(t, proof.Summary, "1653-11-01")
	assert.Contains(t, proof.Summary, "1653-11-02")
}

func TestVitalReview_LegacyDeathAndBurialDatesParticipateInChronology(t *testing.T) {
	for _, property := range []string{"death_date", "died_on"} {
		archive := chronologyArchive("burial", "1635", "1653")
		delete(archive.Events, "death")
		delete(archive.Assertions, "death-date")
		archive.Assertions["legacy"] = &Assertion{Subject: EntityRef{Person: "p"}, Property: property, Value: "1653", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
		require.NoError(t, MergeStandardVocabularies(archive))
		conflicts := VitalChronologyConflicts(archive, "p")
		require.Len(t, conflicts, 1, property)
		assert.Contains(t, conflicts[0].Facts[1].Subject.Person, "p")
		assert.Equal(t, property, conflicts[0].Facts[1].Property)
		proof, err := BuildProof(archive, "p", "death", ProofOptions{})
		require.NoError(t, err)
		assert.Equal(t, "CONFLICTED", proof.Conclusion)
		archive.Assertions["legacy"].Status = "disproven"
		assert.Empty(t, VitalChronologyConflicts(archive, "p"))
	}
	archive := chronologyArchive("burial", "1635", "1653")
	archive.Events = nil
	archive.Assertions["burial-date"].Subject = EntityRef{Person: "p"}
	archive.Assertions["burial-date"].Property = "buried_on"
	archive.Assertions["death-date"].Subject = EntityRef{Person: "p"}
	archive.Assertions["death-date"].Property = "died_on"
	require.NoError(t, MergeStandardVocabularies(archive))
	assert.Len(t, VitalChronologyConflicts(archive, "p"), 1, "two legacy person-subject dates also define the chronology")
}

func TestVitalReview_ProxyStructuralDatesRemainBoundsWithoutInventingSupport(t *testing.T) {
	for _, kind := range []string{"baptism", "christening", "burial", "cremation"} {
		for _, supported := range []bool{false, true} {
			archive := chronologyArchive(kind, "1700-01-03", "")
			delete(archive.Events, "death")
			archive.Assertions = map[string]*Assertion{}
			if supported {
				archive.Assertions["participation"] = &Assertion{Subject: EntityRef{Event: "burial"}, Participant: &Participant{Person: "p", Role: "principal"}, Citations: []string{"c"}, Confidence: "high", Status: "proven"}
			}
			require.NoError(t, MergeStandardVocabularies(archive))
			topic := "death"
			if isBirthEventType(kind) {
				topic = "birth"
			}
			proof, err := BuildProof(archive, "p", topic, ProofOptions{})
			require.NoError(t, err)
			assert.Contains(t, proof.Summary, "on or before 1700-01-03", kind)
			if !supported {
				assert.Equal(t, "INSUFFICIENT EVIDENCE", proof.Conclusion)
				assert.Empty(t, proof.Evidence)
			}
			if supported {
				assert.Equal(t, "INSUFFICIENT EVIDENCE", proof.Conclusion, "a participation citation alone does not establish a vital date")
				assert.Len(t, proof.Evidence, 1)
			}
		}
	}
}

func TestVitalReview_ChronologyGroupsKeepAllSupportingFacts(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	burialCopy := *archive.Assertions["burial-date"]
	deathCopy := *archive.Assertions["death-date"]
	deathCopy2 := deathCopy
	archive.Assertions["burial-second-source"] = &burialCopy
	archive.Assertions["death-second-source"] = &deathCopy
	archive.Assertions["death-third-source"] = &deathCopy2
	require.NoError(t, MergeStandardVocabularies(archive))
	conflicts := VitalChronologyConflicts(archive, "p")
	require.Len(t, conflicts, 1)
	assert.Len(t, conflicts[0].Facts, 5)
	assert.Len(t, conflicts[0].Evaluation.Comparisons, 6)
	assert.Len(t, conflicts[0].Evaluation.Selected, 5)
	assert.Len(t, conflicts[0].Evaluation.Surviving, 5)
	assert.Len(t, conflicts[0].Values, 2)
}

func TestVitalReview_LegacyBurialAliasesAreDeathBounds(t *testing.T) {
	for _, property := range []string{"buried_on", "burial_date"} {
		archive := chronologyArchive("burial", "1700", "")
		archive.Events = nil
		archive.Places = map[string]*Place{"place": {Name: "Parish"}}
		archive.Assertions = map[string]*Assertion{
			"burial": {Subject: EntityRef{Person: "p"}, Property: property, Value: "1700", Citations: []string{"c"}, Confidence: "high", Status: "proven"},
			"place":  {Subject: EntityRef{Person: "p"}, Property: "buried_at", Value: "place", Citations: []string{"c"}, Confidence: "high", Status: "proven"},
		}
		require.NoError(t, MergeStandardVocabularies(archive))
		proof, err := BuildProof(archive, "p", "death", ProofOptions{})
		require.NoError(t, err)
		assert.Contains(t, proof.Summary, "died on or before 1700 (burial)")
		assert.Contains(t, proof.Summary, "burial place: Parish")
		assert.Contains(t, proof.Gaps, ProofGap{Label: "Death date not established", Description: "The recorded rite bounds the vital event; it does not establish its exact date."})
		assert.Equal(t, property, archive.Assertions["burial"].Property)
		archive.Assertions["death"] = &Assertion{Subject: EntityRef{Person: "p"}, Property: "died_on", Value: "1699", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
		proof, err = BuildProof(archive, "p", "death", ProofOptions{})
		require.NoError(t, err)
		assert.Contains(t, proof.Summary, "1699")
		assert.NotContains(t, proof.Summary, "1700", "a primary death date takes precedence over the proxy date")
		for _, gap := range proof.Gaps {
			assert.NotEqual(t, "Death date not established", gap.Label)
		}
	}
}

func TestVitalReview_DisprovenDifferentValueKeepsStoredField(t *testing.T) {
	archive := chronologyArchive("burial", "1653-11-03", "1653-11-01")
	archive.Assertions["death-date"].Value = "1700-01-01"
	archive.Assertions["death-date"].Status = "disproven"
	require.NoError(t, MergeStandardVocabularies(archive))
	proof, err := BuildProof(archive, "p", "death", ProofOptions{})
	require.NoError(t, err)
	assert.Contains(t, proof.Summary, "1653-11-01", "ruling out 1700 does not withdraw the stored 1653 date")
	// Disproving the stored date's year still withdraws the day within it.
	archive.Assertions["death-date"].Value = "1653"
	proof, err = BuildProof(archive, "p", "death", ProofOptions{})
	require.NoError(t, err)
	assert.NotContains(t, proof.Summary, "1653-11-01")
}

func TestVitalReview_DecedentRoleIsTheVitalPrincipal(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	archive.Events["burial"].Participants[0].Role = "decedent"
	require.NoError(t, MergeStandardVocabularies(archive))
	assert.Len(t, VitalChronologyConflicts(archive, "p"), 1)
}

func TestVitalReview_DisprovenLegacyDateDoesNotReturnThroughEventDate(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	delete(archive.Assertions, "death-date")
	archive.Assertions["legacy"] = &Assertion{Subject: EntityRef{Person: "p"}, Property: "died_on", Value: "1653", Citations: []string{"c"}, Confidence: "high", Status: "disproven"}
	require.NoError(t, MergeStandardVocabularies(archive))
	assert.Empty(t, VitalChronologyConflicts(archive, "p"), "the rejected 1653 death date is not rebuilt from the event field")
	proof, err := BuildProof(archive, "p", "death", ProofOptions{})
	require.NoError(t, err)
	assert.NotEqual(t, "CONFLICTED", proof.Conclusion)
}

func TestVitalReview_DeathDateRecordedOnBurialIsCompared(t *testing.T) {
	for _, property := range []string{"death_date", "died_on"} {
		archive := chronologyArchive("burial", "1700-01-01", "")
		delete(archive.Events, "death")
		delete(archive.Assertions, "death-date")
		archive.Assertions["death-on-burial"] = &Assertion{Subject: EntityRef{Event: "burial"}, Property: property, Value: "1700-01-03", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
		require.NoError(t, MergeStandardVocabularies(archive))
		conflicts := VitalChronologyConflicts(archive, "p")
		require.Len(t, conflicts, 1, property)
		proof, err := BuildProof(archive, "p", "death", ProofOptions{})
		require.NoError(t, err)
		assert.Equal(t, "CONFLICTED", proof.Conclusion, property)
	}
}

func TestVitalReview_LegacyAndEventBurialAreOneObservation(t *testing.T) {
	archive := chronologyArchive("burial", "1635", "1653")
	archive.Assertions["legacy-burial"] = &Assertion{Subject: EntityRef{Person: "p"}, Property: "buried_on", Value: "1635", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
	archive.Assertions["legacy-death"] = &Assertion{Subject: EntityRef{Person: "p"}, Property: "died_on", Value: "1653", Citations: []string{"c"}, Confidence: "high", Status: "proven"}
	require.NoError(t, MergeStandardVocabularies(archive))
	conflicts := VitalChronologyConflicts(archive, "p")
	require.Len(t, conflicts, 1, "one burial and one death give one conflict")
	assert.Len(t, conflicts[0].Facts, 4, "both raw facts of each observation are kept")
	// Two burial events stay separate observations; the legacy claim cannot
	// be assigned to either.
	archive.Events["second-burial"] = &Event{Type: "burial", Date: "1636", Participants: []Participant{{Person: "p", Role: "principal"}}}
	assert.Len(t, VitalChronologyConflicts(archive, "p"), 3)
}
