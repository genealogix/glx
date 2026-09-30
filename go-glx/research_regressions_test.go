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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestResearchAPI_MediaBackedCoverage(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons:       map[string]*glxlib.Person{"p": {}, "q": {}},
		Events:        map[string]*glxlib.Event{},
		Assertions:    map[string]*glxlib.Assertion{},
		Media:         map[string]*glxlib.Media{"record": {URI: "record.jpg", Title: "Record scan"}},
		Relationships: map[string]*glxlib.Relationship{"couple": {Type: "marriage", StartEvent: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}, {Person: "q", Role: "spouse"}}}},
	}
	for kind, date := range map[string]glxlib.DateString{"birth": "1850", "death": "1920", "census": "1870", "marriage": "1875", "probate": "1920", "baptism": "1850"} {
		archive.Events[kind] = &glxlib.Event{Type: kind, Date: date, Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}
		archive.Assertions["a-"+kind] = &glxlib.Assertion{Subject: glxlib.EntityRef{Event: kind}, Media: []string{"record"}, Confidence: "high"}
	}
	before := snapshot(t, archive)
	report, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: glxlib.CensusCountryUnitedStates})
	require.NoError(t, err)
	found := map[string]bool{}
	for _, record := range report.Records {
		if record.Found {
			found[record.SourceRef] = true
		}
	}
	require.Equal(t, map[string]bool{"birth": true, "death": true, "census": true, "marriage": true, "probate": true, "baptism": true}, found)
	proof, err := glxlib.BuildProof(archive, "p", glxlib.QuestionBirth, glxlib.ProofOptions{})
	require.NoError(t, err)
	for _, gap := range proof.Gaps {
		require.NotEqual(t, "Birth record", gap.Label)
	}
	require.Equal(t, "media", proof.Evidence[0].Support[0].Kind)
	require.Equal(t, "record", proof.Evidence[0].Support[0].Ref)
	require.Equal(t, "Record scan", proof.Evidence[0].Support[0].SourceTitle)
	facts, err := glxlib.CollectPersonFacts(archive, "p")
	require.NoError(t, err)
	facts[0].Media[0] = "changed"
	require.JSONEq(t, before, snapshot(t, archive))
	for _, media := range []map[string]*glxlib.Media{nil, {"record": nil}} {
		archive.Media = media
		report, err = glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
		require.NoError(t, err)
		require.Zero(t, report.Found, "dangling or nil media must not count as evidence")
	}
}

func TestResearchAPI_FixedRefinementPreservesPrecision(t *testing.T) {
	cases := []struct{ property, broad, narrow string }{
		{"born_on", "1850", "1850-02-01"},
		{"died_on", "ABT 1900", "1901-01-01"},
		{"born_at", "state", "county"},
	}
	for _, tc := range cases {
		for _, reverse := range []bool{false, true} {
			keep, drop := tc.broad, tc.narrow
			if reverse {
				keep, drop = drop, keep
			}
			archive := &glxlib.GLXFile{
				Persons: map[string]*glxlib.Person{"keep": {Properties: map[string]any{tc.property: keep}}, "drop": {Properties: map[string]any{tc.property: drop}}},
				Places:  map[string]*glxlib.Place{"state": {Name: "State"}, "county": {Name: "County", ParentID: "state"}},
			}
			result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
			require.NoError(t, err)
			require.Empty(t, result.Conflicts)
			require.Equal(t, tc.narrow, archive.Persons["keep"].Properties[tc.property])
		}
	}
	// Intersecting ranges without containment cannot silently discard either claim.
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"keep": {Properties: map[string]any{"born_on": "BET 1850 AND 1860"}},
		"drop": {Properties: map[string]any{"born_on": "BET 1855 AND 1870"}},
	}}
	result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
	require.NoError(t, err)
	require.Len(t, result.Conflicts, 1)
	require.Equal(t, "BET 1855 AND 1870", result.Conflicts[0].DropValue)
	require.Equal(t, glxlib.VerdictRefinement, result.Conflicts[0].Verdict)
}

func TestResearchAPI_RefinementRespectsResearcherStatus(t *testing.T) {
	broad := map[string]any{"value": "1850", "status": "proven"}
	narrow := map[string]any{"value": "1850-02-01", "status": "disproven"}
	for _, reverse := range []bool{false, true} {
		keep, drop := broad, narrow
		if reverse {
			keep, drop = drop, keep
		}
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
			"keep": {Properties: map[string]any{"born_on": keep}},
			"drop": {Properties: map[string]any{"born_on": drop}},
		}}
		result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
		require.NoError(t, err)
		require.Empty(t, result.Conflicts)
		require.Equal(t, broad, archive.Persons["keep"].Properties["born_on"])
	}
}

func TestResearchAPI_ProofResolvesOnlyReferences(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons:    map[string]*glxlib.Person{"p": {}},
		Places:     map[string]*glxlib.Place{"paris": {Name: "Paris, France"}},
		Assertions: map[string]*glxlib.Assertion{"a": {Subject: glxlib.EntityRef{Person: "p"}, Property: "name", Value: "paris", Confidence: "high"}},
	}
	proof, err := glxlib.BuildProof(archive, "p", glxlib.QuestionIdentity, glxlib.ProofOptions{})
	require.NoError(t, err)
	require.Equal(t, "paris", proof.Undated[0].Value)
	require.Contains(t, proof.Summary, "paris")
	require.NotContains(t, proof.Summary, "Paris, France")
	archive.Assertions["a"].Property = "born_at"
	proof, err = glxlib.BuildProof(archive, "p", glxlib.QuestionBirth, glxlib.ProofOptions{})
	require.NoError(t, err)
	require.Equal(t, "Paris, France", proof.Evidence[0].Value)
	archive.PersonProperties = map[string]*glxlib.PropertyDefinition{"born_at": nil}
	proof, err = glxlib.BuildProof(archive, "p", glxlib.QuestionBirth, glxlib.ProofOptions{})
	require.NoError(t, err)
	require.Equal(t, "paris", proof.Evidence[0].Value, "nil overrides also suppress reference display")
}

func TestResearchAPI_LegacyVitalFactsShareEventComparisons(t *testing.T) {
	cases := []struct{ kind, property, field string }{
		{"birth", "born_on", "date"},
		{"birth", "birth_date", "date"},
		{"birth", "born_at", "place"},
		{"birth", "birth_place", "place"},
		{"death", "died_on", "date"},
		{"death", "death_date", "date"},
		{"death", "died_at", "place"},
		{"death", "death_place", "place"},
		{"burial", "buried_on", "date"},
		{"burial", "burial_date", "date"},
		{"burial", "buried_at", "place"},
		{"burial", "burial_place", "place"},
	}
	for _, tc := range cases {
		t.Run(tc.property, func(t *testing.T) {
			for _, asserted := range []bool{false, true} {
				archive := &glxlib.GLXFile{
					Persons:    map[string]*glxlib.Person{"p": {}},
					Events:     map[string]*glxlib.Event{"vital": {Type: tc.kind, Date: "1900", PlaceID: "b", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}},
					Places:     map[string]*glxlib.Place{"a": {Name: "Alpha"}, "b": {Name: "Beta"}},
					Assertions: map[string]*glxlib.Assertion{"legacy": {Subject: glxlib.EntityRef{Person: "p"}, Property: tc.property, Value: "1850", Confidence: "high"}},
				}
				if tc.kind == "burial" {
					archive.Events["vital"].Participants[0].Role = "deceased"
				}
				value := "1900"
				if tc.field == "place" {
					archive.Assertions["legacy"].Value = "a"
					value = "b"
				}
				if asserted {
					archive.Assertions["event"] = &glxlib.Assertion{Subject: glxlib.EntityRef{Event: "vital"}, Property: tc.field, Value: value, Confidence: "low"}
				}
				before := snapshot(t, archive)
				topic := tc.kind
				if topic == "burial" {
					topic = glxlib.QuestionDeath
				}
				proof, err := glxlib.BuildProof(archive, "p", topic, glxlib.ProofOptions{})
				require.NoError(t, err)
				require.Equal(t, glxlib.ConclusionConflicted, proof.Conclusion)
				require.Len(t, proof.Conflicts, 1)
				require.Equal(t, tc.field, proof.Conflicts[0].Property)
				require.Len(t, proof.Conflicts[0].Facts, 2)
				findings, err := glxlib.AnalyzeConflicts(archive, glxlib.ConflictAnalysisOptions{})
				require.NoError(t, err)
				require.Len(t, findings, 1)
				require.Equal(t, "high", findings[0].Severity)
				facts, err := glxlib.CollectPersonFacts(archive, "p")
				require.NoError(t, err)
				require.Len(t, facts, 2)
				require.Equal(t, facts[0].FactKey, facts[1].FactKey)
				require.Equal(t, tc.field, facts[0].FactProperty)
				require.Equal(t, facts[0].FactProperty, facts[1].FactProperty)
				require.NotEqual(t, facts[0].Property, facts[1].Property, "raw aliases are retained")
				require.JSONEq(t, before, snapshot(t, archive))
				if asserted {
					archive.Assertions["event"].Status = "disproven"
					proof, err = glxlib.BuildProof(archive, "p", topic, glxlib.ProofOptions{})
					require.NoError(t, err)
					require.True(t, proof.Conflicts[0].Resolved)
					unwanted := value
					if tc.field == "place" {
						unwanted = "Beta"
					}
					require.NotContains(t, proof.Summary, unwanted, "structural values must not revive disproven evidence")
				}
			}
		})
	}
}

func TestResearchAPI_VitalAliasesRespectRolesAndOverrides(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}, "q": {}},
		Events:  map[string]*glxlib.Event{"birth": {Type: "birth", Date: "1900", Participants: []glxlib.Participant{{Person: "p", Role: "parent"}, {Person: "q", Role: "subject"}}}},
		Assertions: map[string]*glxlib.Assertion{
			"legacy": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_on", Value: "1850", Confidence: "high"},
			"event":  {Subject: glxlib.EntityRef{Event: "birth"}, Property: "date", Value: "1900"},
		},
	}
	groups, err := glxlib.PersonConflicts(archive, "p", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Empty(t, groups, "a parent's own birth must not be compared with their child's birth")
	archive.Events["birth"].Participants = []glxlib.Participant{{Person: "p", Role: "subject"}}
	for _, personOverride := range []bool{false, true} {
		archive.PersonProperties, archive.EventProperties = nil, nil
		if personOverride {
			archive.PersonProperties = map[string]*glxlib.PropertyDefinition{"born_on": nil}
		} else {
			archive.EventProperties = map[string]*glxlib.PropertyDefinition{"date": nil}
		}
		facts, err := glxlib.CollectPersonFacts(archive, "p")
		require.NoError(t, err)
		require.NotEqual(t, facts[0].FactKey, facts[1].FactKey, "custom definitions must not inherit another property's semantics")
	}
}

func TestResearchAPI_ParticipantProofEvidence(t *testing.T) {
	cases := []struct {
		topic, kind, personRole, claimedRole string
		event                                bool
	}{
		{"parentage", "parent_child", "child", "parent", false},
		{"parentage", "birth", "subject", "parent", true},
		{"marriage", "marriage", "spouse", "spouse", false},
		{"marriage", "marriage", "bride", "groom", true},
		{"marriage", "marriage", "spouse", "witness", false},
	}
	for _, tc := range cases {
		t.Run(strings.Join([]string{tc.topic, tc.kind, tc.claimedRole}, "/"), func(t *testing.T) {
			archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}, "q": {Properties: map[string]any{"name": "Alex"}}}}
			subject := glxlib.EntityRef{Relationship: "subject"}
			participants := []glxlib.Participant{{Person: "p", Role: tc.personRole}}
			if tc.kind == "birth" {
				participants = append([]glxlib.Participant{{Person: "p", Role: "witness"}}, participants...)
			}
			if tc.event {
				subject = glxlib.EntityRef{Event: "subject"}
				archive.Events = map[string]*glxlib.Event{"subject": {Type: tc.kind, Participants: participants}}
			} else {
				archive.Relationships = map[string]*glxlib.Relationship{"subject": {Type: tc.kind, Participants: participants}}
			}
			archive.Assertions = map[string]*glxlib.Assertion{"claim": {Subject: subject, Participant: &glxlib.Participant{Person: "q", Role: tc.claimedRole}, Confidence: "high"}}
			proof, err := glxlib.BuildProof(archive, "p", tc.topic, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Len(t, proof.Evidence, 1)
			require.Equal(t, "q", proof.Evidence[0].ParticipantPerson)
			require.Equal(t, tc.claimedRole, proof.Evidence[0].ParticipantRole)
			require.Equal(t, "Alex (q) — "+tc.claimedRole, proof.Evidence[0].Value)
			if tc.claimedRole == "witness" {
				require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion)
			} else {
				require.Equal(t, glxlib.ConclusionProven, proof.Conclusion)
				require.Contains(t, proof.Summary, "Alex (q)")
				archive.Assertions["claim"].Status = "disproven"
				proof, err = glxlib.BuildProof(archive, "p", tc.topic, glxlib.ProofOptions{})
				require.NoError(t, err)
				require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion)
			}
		})
	}
}

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
