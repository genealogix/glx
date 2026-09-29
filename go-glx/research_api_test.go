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

package glx_test

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func researchFixture() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"p": {Properties: map[string]any{"name": "Mary", "occupation": map[string]any{"value": "miller", "date": "1851"}}},
			"q": {Properties: map[string]any{"name": "Mary", "occupation": map[string]any{"value": "farmer", "date": "1875"}}},
		},
		Events:  map[string]*glxlib.Event{"birth": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}},
		Sources: map[string]*glxlib.Source{"source": {Title: "Register"}},
		Assertions: map[string]*glxlib.Assertion{
			"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Smith", Date: "1850", Sources: []string{"source"}, Confidence: "high"},
			"b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Jones", Date: "1890", Confidence: "high"},
			"c": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Green", Notes: glxlib.NoteList{"Undated name."}},
		},
	}
}

func snapshot(t *testing.T, archive *glxlib.GLXFile) string {
	t.Helper()
	data, err := json.Marshal(archive)
	require.NoError(t, err)

	return string(data)
}

func TestResearchAPI_DefaultsReadOnlyAndOwnedResults(t *testing.T) {
	archive := researchFixture()
	before := snapshot(t, archive)
	findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
	require.NoError(t, err)
	require.Empty(t, findings)
	evidence, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "name", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.True(t, evidence.Temporal)
	require.Len(t, evidence.Groups, 2)
	require.Len(t, evidence.Undated, 1)
	proof, err := glxlib.BuildProof(archive, "p", "WHO", glxlib.ProofOptions{})
	require.NoError(t, err)
	require.Equal(t, glxlib.QuestionIdentity, proof.Question)
	require.Equal(t, glxlib.ConclusionProven, proof.Conclusion)
	require.Len(t, proof.Undated, 1)
	require.NotEmpty(t, proof.Gaps)
	facts, err := glxlib.CollectPersonFacts(archive, "p")
	require.NoError(t, err)
	require.Len(t, facts, 3)
	facts[0].Sources[0] = "changed"
	facts[0].Fact.Value = "changed"
	evidence.Groups[0].Items[0].Source = "changed"
	proof.Undated[0].Value = "changed"
	require.JSONEq(t, before, snapshot(t, archive))
	require.Nil(t, archive.PersonProperties, "read APIs must not populate the caller's vocabulary maps")
	again, err := glxlib.BuildProof(archive, "p", glxlib.QuestionIdentity, glxlib.ProofOptions{})
	require.NoError(t, err)
	require.Equal(t, "Mary Green", again.Undated[0].Value)
}

func TestResearchAPI_CustomVocabularyAndProvenance(t *testing.T) {
	archive := researchFixture()
	temporal := false
	custom := &glxlib.PropertyDefinition{Temporal: &temporal, ValueType: "string"}
	archive.PersonProperties = map[string]*glxlib.PropertyDefinition{"name": custom}
	before := snapshot(t, archive)
	groups, err := glxlib.PersonConflicts(archive, "p", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, glxlib.VerdictDefinite, groups[0].Verdict)
	require.Len(t, groups[0].Facts, 3)
	require.Equal(t, "a", groups[0].Facts[0].ID)
	require.Equal(t, glxlib.EntityRef{Person: "p"}, groups[0].Facts[0].Subject)
	require.Equal(t, "Mary Smith", groups[0].Facts[0].Fact.Value)
	require.NotEmpty(t, groups[0].Evaluation.Selected)
	groups[0].Facts[0].Sources[0] = "changed"
	require.JSONEq(t, before, snapshot(t, archive))
	require.Same(t, custom, archive.PersonProperties["name"])
	require.NoError(t, glxlib.MergeStandardVocabularies(archive))
	require.Same(t, custom, archive.PersonProperties["name"])
	require.NotNil(t, archive.PersonProperties["occupation"])
}

func TestResearchAPI_MergeDefaultsAndValidationBeforeMutation(t *testing.T) {
	archive := researchFixture()
	result, err := glxlib.MergePersons(archive, "p", "q", glxlib.MergePersonsOptions{})
	require.NoError(t, err)
	require.Empty(t, result.Conflicts)
	require.Len(t, archive.Persons["p"].Properties["occupation"], 2)
	for _, width := range []int{-1, 10001} {
		archive := researchFixture()
		before := snapshot(t, archive)
		opts := glxlib.ComparisonOptions{ApproximationYears: &width}
		_, err = glxlib.CompareFacts(nil, nil, nil, opts)
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.EvaluateFacts(nil, nil, nil, opts)
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{Comparison: opts})
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "name", opts)
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.BuildProof(archive, "p", glxlib.QuestionIdentity, glxlib.ProofOptions{Comparison: opts})
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.PersonConflicts(archive, "p", opts)
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = glxlib.MergePersons(archive, "p", "q", glxlib.MergePersonsOptions{Comparison: opts})
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		_, err = opts.ApproximationWidth()
		require.ErrorIs(t, err, glxlib.ErrInvalidApproximation)
		require.JSONEq(t, before, snapshot(t, archive))
	}
}

func TestResearchAPI_ErrorsAndResolution(t *testing.T) {
	archive := researchFixture()
	_, err := glxlib.BuildProof(nil, "p", glxlib.QuestionIdentity, glxlib.ProofOptions{})
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
	_, err = glxlib.BuildProof(archive, "missing", glxlib.QuestionIdentity, glxlib.ProofOptions{})
	require.ErrorIs(t, err, glxlib.ErrPersonNotFound)
	_, err = glxlib.BuildProof(archive, "p", "unknown", glxlib.ProofOptions{})
	require.ErrorIs(t, err, glxlib.ErrUnknownResearchQuestion)
	_, err = glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: "unknown"})
	require.ErrorIs(t, err, glxlib.ErrUnknownCensusCountry)
	_, err = glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p", Event: "birth"}, "date", glxlib.ComparisonOptions{})
	require.ErrorIs(t, err, glxlib.ErrInvalidResearchSubject)
	_, err = glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "missing"}, "name", glxlib.ComparisonOptions{})
	require.ErrorIs(t, err, glxlib.ErrResearchSubjectNotFound)
	_, err = glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{PersonID: "missing"})
	require.ErrorIs(t, err, glxlib.ErrPersonNotFound)
	_, err = glxlib.MergePersons(nil, "p", "q", glxlib.MergePersonsOptions{})
	require.ErrorIs(t, err, glxlib.ErrNilArchive)
	values := []glxlib.FactValue{{Value: "1850", Status: "proven"}, {Value: "1852", Status: "disproven"}, {Value: "1853", Status: "disproven"}}
	evaluation, err := glxlib.EvaluateFacts(values, &glxlib.PropertyDefinition{ValueType: "date"}, nil, glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.True(t, evaluation.Resolved)
	require.Equal(t, []int{0}, evaluation.Surviving)
	require.Equal(t, "1850", evaluation.Resolution)
	for _, width := range []int{0, 2, 3, 10000} {
		opts := glxlib.ComparisonOptions{ApproximationYears: &width}
		got, err := opts.ApproximationWidth()
		require.NoError(t, err)
		require.Equal(t, width, got)
		pairs, err := glxlib.CompareFacts([]glxlib.FactValue{{Value: "ABT 1850"}, {Value: "1853"}}, &glxlib.PropertyDefinition{ValueType: "date"}, nil, opts)
		require.NoError(t, err)
		if width < 3 {
			require.Equal(t, glxlib.VerdictDefinite, pairs[0].Verdict)
		} else {
			require.Equal(t, glxlib.VerdictRefinement, pairs[0].Verdict)
		}
	}
}

func TestResearchAPI_CoverageOptionsAreConcurrentAndLocal(t *testing.T) {
	archive := researchFixture()
	before := snapshot(t, archive)
	var wg sync.WaitGroup
	for _, country := range []string{"", glxlib.CensusCountryUnitedStates} {
		wg.Go(func() {
			result, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: country})
			if err != nil {
				t.Error(err)

				return
			}
			census := 0
			for _, record := range result.Records {
				if record.Category == "census" {
					census++
				}
			}
			if (country == "") != (census == 0) {
				t.Errorf("country=%q census=%d", country, census)
			}
		})
	}
	wg.Wait()
	require.JSONEq(t, before, snapshot(t, archive))
	opts := glxlib.CoverageOptions{CensusCountry: glxlib.CensusCountryUnitedStates}
	schedules, err := glxlib.CensusSchedulesForPlaces(archive, nil, opts)
	require.NoError(t, err)
	schedules[0].Years[0] = 9999
	schedules[0].Notes[1850] = glxlib.CensusYearNote{Note: "changed"}
	next, err := glxlib.CensusSchedulesForPlaces(archive, nil, opts)
	require.NoError(t, err)
	require.NotEqual(t, 9999, next[0].Years[0])
	require.NotEqual(t, "changed", next[0].Notes[1850].Note)
}

func TestResearchAPI_MergeAcceptsNativeDateString(t *testing.T) {
	archive := researchFixture()
	archive.Persons["p"].Properties["occupation"] = map[string]any{"value": "miller", "date": glxlib.DateString("FROM 1850 TO 1900")}
	archive.Persons["q"].Properties["occupation"] = map[string]any{"value": "farmer", "date": glxlib.DateString("1875")}
	result, err := glxlib.MergePersons(archive, "p", "q", glxlib.MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)
	require.Len(t, result.Conflicts, 1)
	require.Equal(t, glxlib.ResolutionKeptNewest, result.Conflicts[0].Resolution)
	require.Equal(t, "farmer (1875)", glxlib.FormatPropertyValue(archive.Persons["p"].Properties["occupation"]))
}

func TestResearchAPI_CompatibleAggregateIgnoresInputOrder(t *testing.T) {
	temporal := true
	definition := &glxlib.PropertyDefinition{Temporal: &temporal, ReferenceType: "places"}
	places := map[string]*glxlib.Place{"leeds": {ParentID: "england"}, "london": {ParentID: "england"}, "england": {}}
	values := []glxlib.FactValue{{Value: "leeds", Date: "1850"}, {Value: "london", Date: "1870"}, {Value: "england", Date: "1870"}}
	for _, indices := range [][3]int{{0, 1, 2}, {2, 0, 1}, {1, 2, 0}} {
		ordered := []glxlib.FactValue{values[indices[0]], values[indices[1]], values[indices[2]]}
		result, err := glxlib.EvaluateFacts(ordered, definition, places, glxlib.ComparisonOptions{})
		require.NoError(t, err)
		require.Equal(t, glxlib.VerdictHistory, result.Verdict)
		require.Empty(t, result.Selected)
	}
}
