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
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergePersons_ReportsSurvivingEvidenceWithoutClaimDeletion(t *testing.T) {
	for _, eventClaim := range []bool{false, true} {
		t.Run(fmt.Sprintf("event=%t", eventClaim), func(t *testing.T) {
			archive := newTwoPersonArchive()
			archive.Sources = map[string]*Source{"s": {Title: "Birth certificate", Type: SourceTypeVitalRecord}}
			claim := &Assertion{Subject: EntityRef{Person: "person-drop"}, Property: "born_on", Value: "1850", Sources: []string{"s"}, Confidence: ConfidenceLevelHigh, Status: "proven"}
			if eventClaim {
				archive.Events = map[string]*Event{"birth": {Type: EventTypeBirth, Date: "1850", Participants: []Participant{{Person: "person-drop", Role: ParticipantRoleSubject}}}}
				claim.Subject = EntityRef{Event: "birth"}
				claim.Property = "date"
			}
			archive.Assertions = map[string]*Assertion{"claim": claim}
			require.NoError(t, MergeStandardVocabularies(archive))
			before := cloneMergeArchive(archive)
			oldProof, err := BuildProof(before, "person-keep", "birth", ProofOptions{})
			require.NoError(t, err)
			result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
			require.NoError(t, err)
			assert.Empty(t, result.RemovedAssertions)
			assert.Contains(t, archive.Assertions, "claim")
			newProof, err := BuildProof(archive, "person-keep", "birth", ProofOptions{})
			require.NoError(t, err)
			assert.NotEqual(t, oldProof.Conclusion, newProof.Conclusion)
			assert.Equal(t, proofConclusionProven, newProof.Conclusion)
			oldJSON, err := json.Marshal(oldProof)
			require.NoError(t, err)
			newJSON, err := json.Marshal(newProof)
			require.NoError(t, err)
			foundProof, foundEvidence, foundCoverage := false, false, false
			for _, change := range result.InterpretationChanges {
				if change.ID == "person-keep" && change.Aspect == "proof/birth" {
					assert.JSONEq(t, string(oldJSON), change.Before)
					assert.JSONEq(t, string(newJSON), change.After)
					foundProof = true
				}
				if change.ID == "person-keep" && change.Aspect == "evidence/born_on" {
					assert.Contains(t, change.Before, `"total_reports":0`)
					assert.Contains(t, change.After, `"total_reports":1`)
					foundEvidence = true
				}
				if change.ID == "person-keep" && change.Aspect == "coverage" {
					foundCoverage = true
				}
			}
			assert.True(t, foundProof, "surviving claims can change proof without a deletion")
			if eventClaim {
				assert.True(t, foundCoverage, "a transferred sourced event changes the survivor's checklist")
			} else {
				assert.True(t, foundEvidence, "survivor evidence must use its own pre-merge baseline")
			}
		})
	}
}

func TestMergePersons_RemovedDropClaimUsesSurvivorEvidenceBaseline(t *testing.T) {
	archive := identityEvidenceFixture(t)
	archive.Assertions = map[string]*Assertion{
		"a-drop": {Subject: EntityRef{Person: "person-drop"}, Property: "identity_case", Value: "r-resolved", Sources: []string{"s"}},
		"z-keep": {Subject: EntityRef{Person: "person-keep"}, Property: "identity_case", Value: "r-other", Sources: []string{"s"}},
	}
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"a-drop"}, result.RemovedAssertions)
	for _, change := range result.InterpretationChanges {
		assert.False(t, change.ID == "person-keep" && change.Aspect == "evidence/identity_case",
			"the survivor's evidence is unchanged; the removed person's claim is disclosed by the deletion report")
	}
}

func TestMergePersons_ReportsChangedCitationProofForUnrelatedPerson(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.Persons["person-third"] = &Person{}
	archive.Relationships = map[string]*Relationship{
		"r-resolved": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-keep"}, {Person: "person-drop"}}},
	}
	archive.Sources = map[string]*Source{"s": {Title: "Birth certificate", Type: SourceTypeVitalRecord}}
	archive.Citations = map[string]*Citation{"c": {SourceID: "s", Properties: map[string]any{"locator": "r-resolved"}}}
	archive.CitationProperties = map[string]*PropertyDefinition{"locator": {Label: "Locator", ReferenceType: "relationships"}}
	archive.Assertions = map[string]*Assertion{
		"third-birth": {Subject: EntityRef{Person: "person-third"}, Property: "born_on", Value: "1850", Citations: []string{"c"}, Confidence: ConfidenceLevelHigh, Status: "proven"},
	}
	require.NoError(t, MergeStandardVocabularies(archive))
	before := cloneMergeArchive(archive)
	oldProof, err := BuildProof(before, "person-third", "birth", ProofOptions{})
	require.NoError(t, err)
	oldJSON, err := json.Marshal(oldProof)
	require.NoError(t, err)
	assert.Contains(t, string(oldJSON), `"locator":"r-resolved"`)
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.RemovedAssertions)
	assert.Equal(t, before.Assertions["third-birth"], archive.Assertions["third-birth"])
	newProof, err := BuildProof(archive, "person-third", "birth", ProofOptions{})
	require.NoError(t, err)
	newJSON, err := json.Marshal(newProof)
	require.NoError(t, err)
	assert.NotContains(t, string(newJSON), `"locator"`)
	found := false
	for _, change := range result.InterpretationChanges {
		if change.ID == "person-third" && change.Aspect == "proof/birth" {
			assert.JSONEq(t, string(oldJSON), change.Before)
			assert.JSONEq(t, string(newJSON), change.After)
			found = true
		}
	}
	assert.True(t, found, "proof support changes must appear even when the person and claim are unchanged")
}

func identityEvidenceFixture(t *testing.T) *GLXFile {
	t.Helper()
	archive := newTwoPersonArchive()
	archive.Persons["person-third"] = &Person{}
	archive.Relationships = map[string]*Relationship{
		"r-resolved": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-drop"}, {Person: "person-keep"}}, Notes: NoteList{"original identity hypothesis"}},
		"r-other":    {Type: RelationshipTypeAssociate, Participants: []Participant{{Person: "person-keep"}, {Person: "person-third"}}},
	}
	archive.Events = map[string]*Event{"e": {Type: EventTypeBirth, Participants: []Participant{{Person: "person-third", Role: "subject"}}}}
	archive.Sources = map[string]*Source{"s": {Type: "book", Title: "Original source"}}
	archive.Citations = map[string]*Citation{"c": {SourceID: "s"}}
	archive.Media = map[string]*Media{"m": {Type: "image", URI: "record.jpg", Source: "s"}}
	archive.PersonProperties = map[string]*PropertyDefinition{"identity_case": {Label: "Identity case", ReferenceType: "relationships"}}
	archive.EventProperties = map[string]*PropertyDefinition{"identity_case": {Label: "Identity case", ReferenceType: "relationships"}}
	archive.RelationshipProperties = map[string]*PropertyDefinition{"identity_case": {Label: "Identity case", ReferenceType: "relationships", MultiValue: new(true)}}
	archive.Assertions = map[string]*Assertion{}
	for _, status := range []string{"speculative", "disproven", "proven", "accepted", "custom-status"} {
		archive.Assertions["a-"+status] = &Assertion{Subject: EntityRef{Relationship: "r-resolved"}, Status: status, Confidence: ConfidenceLevelHigh, Date: "1850", Citations: []string{"c"}, Sources: []string{"s"}, Media: []string{"m"}, Notes: NoteList{"do not repoint this claim"}}
	}
	archive.Assertions["a-value"] = &Assertion{Subject: EntityRef{Person: "person-third"}, Property: "identity_case", Value: "r-resolved", Status: "disproven", Citations: []string{"c"}}
	archive.Assertions["a-qualified"] = &Assertion{Subject: EntityRef{Relationship: "r-other"}, Participant: &Participant{Person: "person-third", Role: "associate", Properties: map[string]any{"identity_case": []any{"r-other", map[string]any{"value": "r-resolved", "date": "1850", "fields": map[string]any{"note": "qualifier"}}}}}, Status: "speculative", Citations: []string{"c"}}
	archive.Assertions["a-event-qualified"] = &Assertion{Subject: EntityRef{Event: "e"}, Participant: &Participant{Person: "person-third", Properties: map[string]any{"identity_case": "r-resolved"}}, Sources: []string{"s"}}
	archive.Assertions["a-unaffected"] = &Assertion{Subject: EntityRef{Person: "person-drop"}, Property: "name", Value: "person-drop", Status: "disproven", Citations: []string{"c"}}
	archive.Assertions["a-literal"] = &Assertion{Subject: EntityRef{Person: "person-third"}, Property: "name", Value: "r-resolved", Sources: []string{"s"}}
	archive.ResearchLogs = map[string]*ResearchLog{"l": {Title: "Negative research", Subject: &EntityRef{Relationship: "r-resolved"}, Researcher: "Researcher", Date: "1850", Status: "complete", Objective: "Compare records", Searches: []Search{{SourceID: "s", CitationID: "c", Result: "not_found"}}, Citations: []string{"c"}, Conclusions: "No identification found", Notes: NoteList{"original log"}, Properties: map[string]any{"literal": "r-resolved", "label": "person-drop"}}}
	require.NoError(t, MergeStandardVocabularies(archive))

	return archive
}

func TestMergePersons_CleansResolvedEvidenceWithoutChangingOtherClaims(t *testing.T) {
	archive := identityEvidenceFixture(t)
	before := cloneMergeArchive(archive)
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"r-resolved"}, result.RemovedRelationships)
	assert.Equal(t, []string{"a-accepted", "a-custom-status", "a-disproven", "a-event-qualified", "a-proven", "a-qualified", "a-speculative", "a-value"}, result.RemovedAssertions)
	assert.Len(t, archive.Assertions, 2)
	expectedAssertion := *before.Assertions["a-unaffected"]
	expectedAssertion.Subject.Person = "person-keep"
	assert.Equal(t, &expectedAssertion, archive.Assertions["a-unaffected"])
	assert.Equal(t, before.Assertions["a-literal"], archive.Assertions["a-literal"])
	expectedLog := *before.ResearchLogs["l"]
	expectedLog.Subject = nil
	assert.Equal(t, &expectedLog, archive.ResearchLogs["l"])
	assert.Equal(t, before.Sources, archive.Sources)
	assert.Equal(t, before.Citations, archive.Citations)
	assert.Equal(t, before.Media, archive.Media)
	assert.NotContains(t, archive.Persons, "person-drop")
	assert.Empty(t, archive.Validate().Errors)
	foundStanding := false
	for _, change := range result.InterpretationChanges {
		if change.EntityType == EntityTypeRelationships && change.ID == "r-other" && change.Aspect == "standing" {
			assert.Equal(t, `"hypothetical"`, change.Before)
			assert.Equal(t, `"accepted"`, change.After)
			foundStanding = true
		}
	}
	assert.True(t, foundStanding, "preview must disclose the existing no-assertions fallback")
	var removedClaim string
	for _, change := range result.Changes {
		if change.ID == "a-disproven" {
			removedClaim = change.Fields[0].OldValue
		}
	}
	assert.Contains(t, removedClaim, "status: disproven")
	assert.Contains(t, removedClaim, "do not repoint this claim")
	repeated := cloneMergeArchive(before)
	repeatedResult, err := MergePersons(repeated, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, result, repeatedResult, "complete reporting must be deterministic")
}

func TestMergePersons_PrunesTypedReferencesAcrossEveryPropertyVocabulary(t *testing.T) {
	archive := identityEvidenceFixture(t)
	archive.Assertions = map[string]*Assertion{}
	archive.Places = map[string]*Place{"p": {Name: "Place"}}
	archive.Repositories = map[string]*Repository{"repo": {Name: "Repository"}}
	owners := []struct {
		properties  *map[string]any
		definitions *map[string]*PropertyDefinition
	}{
		{&archive.Persons["person-third"].Properties, &archive.PersonProperties},
		{&archive.Events["e"].Properties, &archive.EventProperties},
		{&archive.Relationships["r-other"].Properties, &archive.RelationshipProperties},
		{&archive.Places["p"].Properties, &archive.PlaceProperties},
		{&archive.Sources["s"].Properties, &archive.SourceProperties},
		{&archive.Citations["c"].Properties, &archive.CitationProperties},
		{&archive.Repositories["repo"].Properties, &archive.RepositoryProperties},
		{&archive.Media["m"].Properties, &archive.MediaProperties},
	}
	kept := map[string]any{"value": "r-other", "date": "1860", "fields": map[string]any{"note": "keep whole entry"}}
	removed := map[string]any{"value": "r-resolved", "date": "1850", "fields": map[string]any{"note": "remove whole entry"}}
	for _, owner := range owners {
		*owner.properties = map[string]any{"scalar": "r-resolved", "structured": removed, "list": []any{"r-resolved", "r-other"}, "temporal": []any{removed, kept}, "all": []string{"r-resolved"}, "persons": []string{"person-drop", "person-third"}, "opaque": "r-resolved", "opaque-person": "person-drop"}
		for _, key := range []string{"scalar", "structured", "list", "temporal", "all"} {
			(*owner.definitions)[key] = &PropertyDefinition{Label: key, ReferenceType: "relationships", MultiValue: new(true)}
		}
		(*owner.definitions)["persons"] = &PropertyDefinition{Label: "Persons", ReferenceType: "persons", MultiValue: new(true)}
	}
	// Per-participant properties use the owning event/relationship vocabulary.
	archive.Events["e"].Participants[0].Properties = map[string]any{"scalar": "r-resolved", "persons": []any{"person-drop"}}
	archive.Relationships["r-other"].Participants[0].Properties = map[string]any{"scalar": "r-resolved"}
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	// Fresh pointers are intentional: the operation publishes a detached plan.
	actualOwners := []map[string]any{archive.Persons["person-third"].Properties, archive.Events["e"].Properties, archive.Relationships["r-other"].Properties, archive.Places["p"].Properties, archive.Sources["s"].Properties, archive.Citations["c"].Properties, archive.Repositories["repo"].Properties, archive.Media["m"].Properties}
	for _, properties := range actualOwners {
		assert.NotContains(t, properties, "scalar")
		assert.NotContains(t, properties, "structured")
		assert.NotContains(t, properties, "all")
		assert.Equal(t, []any{"r-other"}, properties["list"])
		assert.Equal(t, []any{kept}, properties["temporal"])
		assert.Equal(t, []string{"person-keep", "person-third"}, properties["persons"])
		assert.Equal(t, "r-resolved", properties["opaque"])
		assert.Equal(t, "person-drop", properties["opaque-person"])
	}
	assert.Equal(t, map[string]any{"persons": []any{"person-keep"}}, archive.Events["e"].Participants[0].Properties)
	assert.Empty(t, archive.Relationships["r-other"].Participants[0].Properties)
	assert.Len(t, result.ReferenceChanges, 52)
}

func TestMergePersons_SDKTemporalAndPointerReferenceForms(t *testing.T) {
	for _, value := range []any{TemporalValue{Value: "r-resolved", Date: "1850"}, &TemporalValue{Value: "r-resolved", Date: "1850"}, []TemporalValue{{Value: "r-resolved", Date: "1850"}, {Value: "r-other", Date: "1860"}}, []*TemporalValue{{Value: "r-resolved", Date: "1850"}, {Value: "r-other", Date: "1860"}}} {
		t.Run(reflect.TypeOf(value).String(), func(t *testing.T) {
			archive := identityEvidenceFixture(t)
			archive.Assertions = map[string]*Assertion{}
			archive.SourceProperties["case"] = &PropertyDefinition{Label: "Case", ReferenceType: "relationships"}
			archive.Sources["s"].Properties = map[string]any{"case": value}
			_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
			require.NoError(t, err)
			missing, err := missingMergeTargets(archive)
			require.NoError(t, err)
			assert.Empty(t, missing)
			if reflect.TypeOf(value).Kind() == reflect.Slice {
				assert.Len(t, archive.Sources["s"].Properties["case"], 1)
			} else {
				assert.NotContains(t, archive.Sources["s"].Properties, "case")
			}
		})
	}
}

func TestMergePersons_ValidationFailureDoesNotMutateInput(t *testing.T) {
	archive := identityEvidenceFixture(t)
	// Unsupported custom target category exercises post-plan integrity failure,
	// rather than creating an additional cascade rule beyond the specification.
	archive.PersonProperties["unsupported"] = &PropertyDefinition{ReferenceType: "assertions"}
	archive.Persons["person-third"].Properties = map[string]any{"unsupported": "a-speculative"}
	before := cloneMergeArchive(archive)
	_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.ErrorIs(t, err, ErrValidationFailed)
	assert.Equal(t, before, archive, "cleanup, property merges and cache state must remain untouched on failure")
}

func TestMergePersons_UsesSubjectVocabularyAndReferenceType(t *testing.T) {
	archive := identityEvidenceFixture(t)
	archive.Assertions = map[string]*Assertion{
		"a-event":        {Subject: EntityRef{Event: "e"}, Participant: &Participant{Person: "person-third", Properties: map[string]any{"only_relationship": "r-resolved"}}, Sources: []string{"s"}},
		"a-person-value": {Subject: EntityRef{Person: "person-third"}, Property: "mentor", Value: "person-drop", Sources: []string{"s"}},
	}
	archive.RelationshipProperties["only_relationship"] = &PropertyDefinition{ReferenceType: "relationships"}
	archive.PersonProperties["mentor"] = &PropertyDefinition{ReferenceType: "persons"}
	// Cross-type collisions must not turn a place reference into a person rewrite.
	archive.Places = map[string]*Place{"person-drop": {Name: "A place whose ID matches a person"}}
	archive.Events["e"].PlaceID = "person-drop"
	_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Contains(t, archive.Assertions, "a-event")
	assert.Equal(t, "r-resolved", archive.Assertions["a-event"].Participant.Properties["only_relationship"])
	assert.Equal(t, "person-keep", archive.Assertions["a-person-value"].Value)
	assert.Equal(t, "person-drop", archive.Events["e"].PlaceID)
}

func TestMergePersons_ReportsParticipantStandingWhenAggregateStaysAccepted(t *testing.T) {
	archive := identityEvidenceFixture(t)
	archive.Relationships["r-other"].Participants = []Participant{{Person: "person-drop"}, {Person: "person-third"}}
	archive.Assertions["a-qualified"].Participant.Person = "person-drop"
	archive.Assertions["a-other-proven"] = &Assertion{Subject: EntityRef{Relationship: "r-other"}, Participant: &Participant{Person: "person-third"}, Status: "proven"}
	require.Equal(t, RelationshipStandingAccepted, NewRelationshipStandingIndex(archive).Relationship("r-other"))
	require.Equal(t, RelationshipStandingHypothetical, NewRelationshipStandingIndex(archive).Link("r-other", "person-drop"))
	result, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
	require.NoError(t, err)
	assert.Equal(t, RelationshipStandingAccepted, NewRelationshipStandingIndex(archive).Relationship("r-other"))
	found := false
	for _, change := range result.InterpretationChanges {
		if change.ID == "r-other" && change.Aspect == "link/person-drop→person-keep" {
			assert.Equal(t, `"hypothetical"`, change.Before)
			assert.Equal(t, `"accepted"`, change.After)
			found = true
		}
	}
	assert.True(t, found, "participant standing must be disclosed even when aggregate standing stays accepted")
}
