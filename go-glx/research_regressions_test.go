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
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestResearchAPI_TypedTemporalMerges(t *testing.T) {
	shapes := []struct {
		name string
		wrap func(glxlib.TemporalValue) any
	}{
		{"value", func(v glxlib.TemporalValue) any { return v }},
		{"pointer", func(v glxlib.TemporalValue) any { return &v }},
		{"typed slice", func(v glxlib.TemporalValue) any { return []glxlib.TemporalValue{v} }},
		{"pointer slice", func(v glxlib.TemporalValue) any { return []*glxlib.TemporalValue{&v} }},
		{"mixed slice", func(v glxlib.TemporalValue) any { return []any{v} }},
		{"mixed pointer slice", func(v glxlib.TemporalValue) any { return []any{&v} }},
	}
	for _, shape := range shapes {
		t.Run(shape.name, func(t *testing.T) {
			for _, overlap := range []bool{false, true} {
				a := glxlib.TemporalValue{Value: "miller", Date: "1851"}
				b := glxlib.TemporalValue{Value: "farmer", Date: "1875"}
				if overlap {
					a.Date = "FROM 1850 TO 1900"
				}
				archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
					"keep": {Properties: map[string]any{"occupation": shape.wrap(a)}},
					"drop": {Properties: map[string]any{"occupation": shape.wrap(b)}},
				}}
				result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
				require.NoError(t, err)
				if overlap {
					require.Len(t, result.Conflicts, 1)
					require.Equal(t, glxlib.VerdictDefinite, result.Conflicts[0].Verdict)
					require.Equal(t, "miller (FROM 1850 TO 1900)", glxlib.FormatPropertyValue(archive.Persons["keep"].Properties["occupation"]))
				} else {
					require.Empty(t, result.Conflicts)
					require.Equal(t, []any{map[string]any{"value": "miller", "date": "1851"}, map[string]any{"value": "farmer", "date": "1875"}}, archive.Persons["keep"].Properties["occupation"])
				}
				require.NoError(t, glxlib.MergeStandardVocabularies(archive))
				validation := archive.Validate()
				require.Empty(t, validation.Errors)
				require.Empty(t, validation.Warnings)
			}
		})
	}
}

func TestResearchAPI_TypedTemporalCopyDedupeAndWinner(t *testing.T) {
	value := glxlib.TemporalValue{Value: "miller", Date: "1851"}
	for _, keep := range []any{nil, []glxlib.TemporalValue{value}, map[string]any{"value": "miller", "date": glxlib.DateString("1851")}} {
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"keep": {Properties: map[string]any{}}, "drop": {Properties: map[string]any{"occupation": []any{value, value}}}}}
		if keep != nil {
			archive.Persons["keep"].Properties["occupation"] = keep
		} else {
			archive.Persons["drop"].Properties["occupation"] = []glxlib.TemporalValue{value}
		}
		result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
		require.NoError(t, err)
		require.Empty(t, result.Conflicts)
		require.Equal(t, "miller (1851)", glxlib.FormatPropertyValue(archive.Persons["keep"].Properties["occupation"]))
		if keep != nil {
			require.Zero(t, result.PropertiesMerged)
		}
	}
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"keep": {Properties: map[string]any{"occupation": glxlib.TemporalValue{Value: "miller", Date: "FROM 1850 TO 1900"}}},
		"drop": {Properties: map[string]any{"occupation": glxlib.TemporalValue{Value: "farmer", Date: "1875"}}},
	}}
	result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)
	require.Len(t, result.Conflicts, 1)
	require.Equal(t, glxlib.ResolutionKeptNewest, result.Conflicts[0].Resolution)
	require.Equal(t, []any{map[string]any{"value": "farmer", "date": "1875"}}, archive.Persons["keep"].Properties["occupation"])
	// A keep-side typed property must also remain valid when drop has no properties.
	archive.Persons["drop"] = &glxlib.Person{}
	archive.Persons["keep"].Properties["occupation"] = value
	_, err = glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"value": "miller", "date": "1851"}, archive.Persons["keep"].Properties["occupation"])
}

func TestResearchAPI_CoverageSourceOrder(t *testing.T) {
	for _, citation := range []bool{false, true} {
		archive := &glxlib.GLXFile{
			Persons:   map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "Pat"}}},
			Sources:   map[string]*glxlib.Source{"source-z": {Title: "Birth register Z", Type: "vital_record"}, "source-a": {Title: "Birth register A", Type: "vital_record"}},
			Citations: map[string]*glxlib.Citation{"citation-z": {SourceID: "source-z"}, "citation-a": {SourceID: "source-a"}},
			Assertions: map[string]*glxlib.Assertion{
				"b":   {Subject: glxlib.EntityRef{Person: "p"}, Sources: []string{"source-a"}},
				"a":   {Subject: glxlib.EntityRef{Person: "p"}, Sources: []string{"source-z"}},
				"nil": nil,
			},
		}
		want := "source-z"
		if citation {
			archive.Assertions["a"].Sources = nil
			archive.Assertions["a"].Citations = []string{"citation-z"}
			archive.Assertions["b"].Sources = nil
			archive.Assertions["b"].Citations = []string{"citation-a"}
			want = "citation-z"
		}
		for range 64 {
			report, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
			require.NoError(t, err)
			found := false
			for _, record := range report.Records {
				if record.Label == "Birth record" {
					found = true
					require.Equal(t, want, record.SourceRef)
				}
			}
			require.True(t, found)
		}
	}
}

func TestResearchAPI_ParticipantEvidenceIdentity(t *testing.T) {
	subject := glxlib.EntityRef{Event: "event"}
	archive := &glxlib.GLXFile{
		Persons:   map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": "Alex"}}, "q": {Properties: map[string]any{"name": "Alex"}}},
		Events:    map[string]*glxlib.Event{"event": {Type: "birth"}},
		Sources:   map[string]*glxlib.Source{"source": {Title: "Register"}},
		Citations: map[string]*glxlib.Citation{"citation": {SourceID: "source"}},
		Assertions: map[string]*glxlib.Assertion{
			"p-parent":        {Subject: subject, Participant: &glxlib.Participant{Person: "p", Role: "parent"}, Citations: []string{"citation"}, Confidence: "high"},
			"p-parent-repeat": {Subject: subject, Participant: &glxlib.Participant{Person: "p", Role: "parent"}, Citations: []string{"citation"}, Confidence: "low"},
			"p-witness":       {Subject: subject, Participant: &glxlib.Participant{Person: "p", Role: "witness"}, Citations: []string{"citation"}},
			"q-parent":        {Subject: subject, Participant: &glxlib.Participant{Person: "q", Role: "parent"}, Citations: []string{"citation"}},
			"existence":       {Subject: subject, Citations: []string{"citation"}},
		},
	}
	before := snapshot(t, archive)
	report, err := glxlib.BuildEvidenceReport(archive, subject, "", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Len(t, report.Groups, 4)
	require.Equal(t, 4, report.TotalReports)
	require.Empty(t, report.Conflicts, "co-participants are not competing property values")
	require.Empty(t, report.BestEvidence)
	identities := map[string]bool{}
	labels := map[string]bool{}
	for _, group := range report.Groups {
		identities[group.ParticipantPerson+"/"+group.ParticipantRole] = true
		require.False(t, labels[group.Value], "namesakes and roles must remain distinguishable")
		labels[group.Value] = true
		require.Equal(t, 1, group.Reports, "citation deduplication must be scoped to one claim")
	}
	require.Equal(t, map[string]bool{"p/parent": true, "p/witness": true, "q/parent": true, "/": true}, identities)
	require.JSONEq(t, before, snapshot(t, archive))
	archive.Assertions["p-witness"].Status = "disputed"
	report, err = glxlib.BuildEvidenceReport(archive, subject, "", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Len(t, report.Conflicts, 1)
	require.Equal(t, glxlib.VerdictDisputed, report.Conflicts[0].Verdict)
	require.Equal(t, "p", report.Conflicts[0].ParticipantPerson)
	require.Equal(t, "witness", report.Conflicts[0].ParticipantRole)
	require.Equal(t, glxlib.AssertionFact(archive.Assertions["p-witness"]), report.Conflicts[0].Values[0])
}

func TestResearchAPI_VitalRolesMatchEventType(t *testing.T) {
	cases := []struct {
		kind, role string
		principal  bool
	}{
		{"birth", "child", true},
		{"death", "child", false},
		{"death", "deceased", true},
		{"birth", "deceased", false},
		{"birth", "subject", true},
		{"death", "subject", true},
		{"birth", "principal", true},
		{"death", "principal", true},
		{"birth", "", true},
		{"death", "", true},
		{"birth", "witness", false},
		{"death", "witness", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.role, func(t *testing.T) {
			archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Events: map[string]*glxlib.Event{
				"a": {Type: tc.kind, Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: tc.role}}},
				"b": {Type: tc.kind, Date: "1870", Participants: []glxlib.Participant{{Person: "p", Role: tc.role}}},
			}}
			facts, err := glxlib.CollectPersonFacts(archive, "p")
			require.NoError(t, err)
			findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
			require.NoError(t, err)
			proof, err := glxlib.BuildProof(archive, "p", tc.kind, glxlib.ProofOptions{})
			require.NoError(t, err)
			if tc.principal {
				require.Len(t, facts, 2)
				require.Len(t, findings, 1)
				require.Equal(t, glxlib.ConclusionConflicted, proof.Conclusion)
			} else {
				require.Empty(t, facts)
				require.Empty(t, findings)
				require.Empty(t, proof.Conflicts)
				require.NotEqual(t, glxlib.ConclusionConflicted, proof.Conclusion)
			}
		})
	}
}
