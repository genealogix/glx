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
	"strconv"
	"strings"
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

func TestResearchAPI_SharedFactIndexPreservesScope(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}, "q": {}, "empty": {}, "nil": nil},
		Events: map[string]*glxlib.Event{
			"birth-a": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "witness"}, {Person: "p", Role: "subject"}, {Person: "q", Role: "parent"}}},
			"birth-b": {Type: "birth", Date: "1852", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}, {Person: "q", Role: "parent"}}},
			"shared":  {Type: "census", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}, {Person: "p", Role: "witness"}, {Person: "q", Role: "subject"}}},
			"nil":     nil,
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "p", Role: "child"}, {Person: "p", Role: "witness"}, {Person: "q", Role: "parent"}}},
			"nil": nil,
		},
		Assertions: map[string]*glxlib.Assertion{
			"direct-a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "fixed", Value: "first"},
			"direct-b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "fixed", Value: "second"},
			"other":    {Subject: glxlib.EntityRef{Person: "q"}, Property: "fixed", Value: "only"},
			"shared-a": {Subject: glxlib.EntityRef{Event: "shared"}, Property: "cause", Value: "first"},
			"shared-b": {Subject: glxlib.EntityRef{Event: "shared"}, Property: "cause", Value: "second"},
			"rel-a":    {Subject: glxlib.EntityRef{Relationship: "rel"}, Property: "description", Value: "first"},
			"rel-b":    {Subject: glxlib.EntityRef{Relationship: "rel"}, Property: "description", Value: "second"},
			"nil":      nil,
		},
	}
	before := snapshot(t, archive)
	findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
	require.NoError(t, err)
	require.Len(t, findings, 6)
	for personID, want := range map[string]int{"p": 4, "q": 2, "empty": 0} {
		selected, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{PersonID: personID})
		require.NoError(t, err)
		require.Len(t, selected, want)
		var matching []glxlib.ConflictFinding
		for i := range findings {
			if findings[i].PersonID == personID {
				matching = append(matching, findings[i])
			}
		}
		require.Equal(t, matching, selected, "shared archive indexing must match a single-person query")
	}
	facts, err := glxlib.CollectPersonFacts(archive, "p")
	require.NoError(t, err)
	ids := make(map[string]int)
	for _, fact := range facts {
		ids[fact.ID]++
		if fact.Subject.Relationship == "rel" {
			require.Equal(t, "child", fact.PersonRole)
		}
	}
	require.Equal(t, 1, ids["shared-a"], "multiple roles must not duplicate a claim")
	require.Zero(t, ids["other"])
	require.Equal(t, 1, ids["birth-a:date"], "a later principal role must still qualify")
	require.Equal(t, 1, ids["birth-b:date"])
	parentFacts, err := glxlib.CollectPersonFacts(archive, "q")
	require.NoError(t, err)
	for _, fact := range parentFacts {
		require.False(t, fact.Synthetic, "the parent's participation does not make this their birth")
	}
	require.JSONEq(t, before, snapshot(t, archive))
}

func TestResearchAPI_NamesakeEvidenceOrder(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons:          map[string]*glxlib.Person{"p": {}, "q1": {Properties: map[string]any{"name": "John Smith"}}, "q2": {Properties: map[string]any{"name": "John Smith"}}},
		PersonProperties: map[string]*glxlib.PropertyDefinition{"mentor": {ReferenceType: "persons"}},
		Assertions: map[string]*glxlib.Assertion{
			"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "mentor", Value: "q1", Confidence: "high"},
			"b": {Subject: glxlib.EntityRef{Person: "p"}, Property: "mentor", Value: "q2", Confidence: "high"},
		},
	}
	for range 30 {
		report, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "mentor", glxlib.ComparisonOptions{})
		require.NoError(t, err)
		require.Len(t, report.Groups, 2)
		require.Equal(t, report.Groups[0].Value, report.Groups[1].Value)
		require.Equal(t, "q1", report.Groups[0].RawValue)
		require.Equal(t, "q2", report.Groups[1].RawValue)
		require.Empty(t, report.BestEvidence, "ordering cannot turn equal support into a winner")
	}
}

func BenchmarkResearchAPI_AnalyzeConflicts(b *testing.B) {
	for _, size := range []int{500, 1000, 2000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}, Assertions: map[string]*glxlib.Assertion{}}
			for i := range size {
				id := "person-" + strconv.Itoa(i)
				archive.Persons[id] = &glxlib.Person{}
				for j := range 5 {
					property := "property-" + strconv.Itoa(j)
					archive.Assertions[id+"-"+property] = &glxlib.Assertion{Subject: glxlib.EntityRef{Person: id}, Property: property, Value: "value"}
				}
			}
			b.ReportAllocs()
			for b.Loop() {
				findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
				if err != nil {
					b.Fatal(err)
				}
				if len(findings) != 0 {
					b.Fatal("unexpected conflicts")
				}
			}
		})
	}
}

func TestResearchAPI_NilOverridesSuppressFallbackSemantics(t *testing.T) {
	cases := []struct {
		name, property, first, second string
		subject                       glxlib.EntityRef
	}{
		{"legacy date", "born_on", "ABT 1850", "1851", glxlib.EntityRef{Person: "p"}},
		{"event date", "date", "1850", "1850-03-02", glxlib.EntityRef{Event: "birth"}},
		{"legacy place", "born_at", "county", "state", glxlib.EntityRef{Person: "p"}},
		{"event place", "place", "county", "state", glxlib.EntityRef{Event: "birth"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			archive := researchFixture()
			archive.Places = map[string]*glxlib.Place{"county": {Name: "County", ParentID: "state"}, "state": {Name: "State"}}
			archive.Assertions = map[string]*glxlib.Assertion{
				"first":  {Subject: tc.subject, Property: tc.property, Value: tc.first, Confidence: "high"},
				"second": {Subject: tc.subject, Property: tc.property, Value: tc.second, Confidence: "high"},
			}
			original, err := glxlib.BuildEvidenceReport(archive, tc.subject, tc.property, glxlib.ComparisonOptions{})
			require.NoError(t, err)
			require.Empty(t, original.Conflicts, "absent definitions retain the fallback refinement")
			if tc.subject.Person != "" {
				archive.PersonProperties = map[string]*glxlib.PropertyDefinition{tc.property: nil}
			} else {
				archive.EventProperties = map[string]*glxlib.PropertyDefinition{tc.property: nil}
			}
			require.NoError(t, glxlib.MergeStandardVocabularies(archive))
			require.Nil(t, glxlib.ConflictProperty(archive, tc.subject, tc.property))
			before := snapshot(t, archive)
			report, err := glxlib.BuildEvidenceReport(archive, tc.subject, tc.property, glxlib.ComparisonOptions{})
			require.NoError(t, err)
			require.Len(t, report.Conflicts, 1)
			require.Equal(t, glxlib.VerdictDefinite, report.Conflicts[0].Verdict)
			proof, err := glxlib.BuildProof(archive, "p", glxlib.QuestionBirth, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, glxlib.ConclusionConflicted, proof.Conclusion)
			findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
			require.NoError(t, err)
			require.Len(t, findings, 1)
			require.Equal(t, "high", findings[0].Severity)
			require.JSONEq(t, before, snapshot(t, archive))
		})
	}
	archive := researchFixture()
	archive.PersonProperties = map[string]*glxlib.PropertyDefinition{"born_on": nil}
	archive.Persons["p"].Properties["born_on"] = "ABT 1850"
	archive.Persons["q"].Properties["born_on"] = "1851"
	merged, err := glxlib.MergePersons(archive, "p", "q", glxlib.MergePersonsOptions{})
	require.NoError(t, err)
	require.Len(t, merged.Conflicts, 1)
	require.Equal(t, "born_on", merged.Conflicts[0].Property)
}

func TestResearchAPI_MarriageCoverageOrder(t *testing.T) {
	archive := researchFixture()
	archive.Persons["s-a"] = &glxlib.Person{Properties: map[string]any{"name": "Alpha"}}
	archive.Persons["s-b"] = &glxlib.Person{Properties: map[string]any{"name": "Beta"}}
	archive.Persons["s-c"] = &glxlib.Person{Properties: map[string]any{"name": "Gamma"}}
	archive.Relationships = map[string]*glxlib.Relationship{
		"rel-c": {Type: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}, {Person: "s-c", Role: "spouse"}}},
		"rel-a": {Type: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}, {Person: "s-a", Role: "spouse"}}},
		"rel-b": {Type: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}, {Person: "s-b", Role: "spouse"}}},
		"nil":   nil,
	}
	want := []string{"Marriage record — Alpha", "Marriage record — Beta", "Marriage record — Gamma"}
	for range 30 {
		coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
		require.NoError(t, err)
		var marriages []string
		for _, record := range coverage.Records {
			if strings.HasPrefix(record.Label, "Marriage record") {
				marriages = append(marriages, record.Label)
			}
		}
		require.Equal(t, want, marriages)
		proof, err := glxlib.BuildProof(archive, "p", glxlib.QuestionMarriage, glxlib.ProofOptions{})
		require.NoError(t, err)
		gaps := make([]string, 0, len(proof.Gaps))
		for _, gap := range proof.Gaps {
			gaps = append(gaps, gap.Label)
		}
		require.Equal(t, want, gaps)
	}
}

func TestResearchAPI_UndatedDisputePreserved(t *testing.T) {
	for _, date := range []glxlib.DateString{"", "spring"} {
		for _, status := range []string{"", "disputed", "DISPUTED"} {
			t.Run(string(date)+"/"+status, func(t *testing.T) {
				archive := researchFixture()
				archive.Assertions = map[string]*glxlib.Assertion{"claim": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "Mary Green", Date: date, Confidence: "high", Status: status}}
				findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
				require.NoError(t, err)
				evidence, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "name", glxlib.ComparisonOptions{})
				require.NoError(t, err)
				require.Len(t, evidence.Undated, 1)
				proof, err := glxlib.BuildProof(archive, "p", glxlib.QuestionIdentity, glxlib.ProofOptions{})
				require.NoError(t, err)
				require.Len(t, proof.Undated, 1)
				if status == "" {
					require.Empty(t, findings)
					require.Empty(t, evidence.Conflicts)
					require.Empty(t, proof.Conflicts)
					require.Equal(t, glxlib.ConclusionProven, proof.Conclusion)
				} else {
					require.Len(t, findings, 1)
					require.Equal(t, "low", findings[0].Severity)
					require.Len(t, evidence.Conflicts, 1)
					require.Equal(t, glxlib.VerdictDisputed, evidence.Conflicts[0].Verdict)
					require.Len(t, proof.Conflicts, 1)
					require.Equal(t, glxlib.VerdictDisputed, proof.Conflicts[0].Verdict)
					require.False(t, proof.Conflicts[0].Definite)
					require.Equal(t, glxlib.ConclusionPossible, proof.Conclusion)
				}
			})
		}
	}
	temporal := true
	definition := &glxlib.PropertyDefinition{Temporal: &temporal}
	values := []glxlib.FactValue{{Value: "Mary Green", Status: "disputed"}, {Value: "Mary Jones", Date: "1890"}}
	comparisons, err := glxlib.CompareFacts(values, definition, nil, glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Len(t, comparisons, 2)
	require.Equal(t, glxlib.VerdictHistory, comparisons[0].Verdict, "undated claims still cannot imply an overlap")
	require.Equal(t, glxlib.VerdictDisputed, comparisons[1].Verdict)
	require.Equal(t, comparisons[1].Left, comparisons[1].Right, "the dispute is explicitly recorded, not inferred between claims")
}
