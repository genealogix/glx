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

func TestResearchAPI_MergeKeepsSurvivingFixedClaim(t *testing.T) {
	for _, sameValue := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			invalid := map[string]any{"value": "1850", "status": "disproven"}
			valid := map[string]any{"value": "1860", "status": "proven"}
			if sameValue {
				valid["value"] = "1850"
			}
			keep, drop := invalid, valid
			if reverse {
				keep, drop = drop, keep
			}
			archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
				"keep": {Properties: map[string]any{"born_on": keep}},
				"drop": {Properties: map[string]any{"born_on": drop}},
			}}
			result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{KeepOldest: true})
			require.NoError(t, err)
			require.Empty(t, result.Conflicts)
			require.Equal(t, valid, archive.Persons["keep"].Properties["born_on"])
			require.NotContains(t, archive.Persons, "drop")
		}
	}
}

func TestResearchAPI_MergePreservesStructuredFields(t *testing.T) {
	for _, collision := range []bool{false, true} {
		keep := map[string]any{"value": "123", "fields": map[string]any{"system": "A"}}
		drop := map[string]any{"value": "123", "fields": map[string]any{"issuer": "B"}}
		if collision {
			drop["fields"] = map[string]any{"system": "B"}
		}
		archive := &glxlib.GLXFile{
			Persons: map[string]*glxlib.Person{"keep": {Properties: map[string]any{"identifier": keep}}, "drop": {Properties: map[string]any{"identifier": drop}}},
			PersonProperties: map[string]*glxlib.PropertyDefinition{"identifier": {Label: "Identifier", ValueType: "string", Fields: map[string]*glxlib.FieldDefinition{
				"system": {Label: "System", ValueType: "string"}, "issuer": {Label: "Issuer", ValueType: "string"},
			}}},
		}
		result, err := glxlib.MergePersons(archive, "keep", "drop", glxlib.MergePersonsOptions{})
		require.NoError(t, err)
		if collision {
			require.Len(t, result.Conflicts, 1)
			require.Equal(t, glxlib.VerdictDefinite, result.Conflicts[0].Verdict)
			require.Equal(t, drop, result.Conflicts[0].DropValue)
			require.Equal(t, "123 [system: A]", glxlib.FormatPropertyValue(result.Conflicts[0].KeepValue))
			require.Equal(t, "123 [system: B]", glxlib.FormatPropertyValue(result.Conflicts[0].DropValue))
		} else {
			require.Empty(t, result.Conflicts)
			require.Equal(t, map[string]any{"value": "123", "fields": map[string]any{"system": "A", "issuer": "B"}}, archive.Persons["keep"].Properties["identifier"])
			require.Equal(t, map[string]any{"system": "A"}, keep["fields"], "caller-held field maps remain intact")
		}
	}
}

func TestResearchAPI_ProofAndCoverageRespectRoles(t *testing.T) {
	cases := []struct{ kind, topic, role, record string }{
		{"birth", "birth", "parent", "Birth record"},
		{"birth", "birth", "witness", "Birth record"},
		{"baptism", "birth", "parent", "Church records"},
		{"christening", "birth", "witness", "Church records"},
		{"death", "death", "informant", "Death record"},
		{"death", "death", "child", "Death record"},
		{"burial", "death", "witness", "Death record"},
		{"cremation", "death", "parent", "Death record"},
		{"marriage", "marriage", "witness", ""},
		{"birth", "parentage", "parent", ""},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.topic+"/"+tc.role, func(t *testing.T) {
			archive := &glxlib.GLXFile{
				Persons:    map[string]*glxlib.Person{"p": {}, "q": {}},
				Events:     map[string]*glxlib.Event{"event": {Type: tc.kind, Date: "1900", Participants: []glxlib.Participant{{Person: "p", Role: tc.role}, {Person: "q", Role: "subject"}}}},
				Sources:    map[string]*glxlib.Source{"source": {Title: "Register"}},
				Assertions: map[string]*glxlib.Assertion{"claim": {Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1900", Confidence: "high", Sources: []string{"source"}}},
			}
			if tc.topic == "parentage" {
				archive.Assertions["claim"].Property = "father"
				archive.Assertions["claim"].Value = "q"
			}
			before := snapshot(t, archive)
			proof, err := glxlib.BuildProof(archive, "p", tc.topic, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion)
			require.Empty(t, proof.Evidence)
			require.Empty(t, proof.Undated)
			coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
			require.NoError(t, err)
			for _, record := range coverage.Records {
				if record.Label == tc.record {
					require.False(t, record.Found)
				}
			}
			if tc.record == "Birth record" || tc.record == "Death record" {
				gapFound := false
				for _, gap := range proof.Gaps {
					gapFound = gapFound || gap.Label == tc.record
				}
				require.True(t, gapFound)
			}
			require.JSONEq(t, before, snapshot(t, archive))
		})
	}
}

func TestResearchAPI_PrincipalAndCensusParticipationRemainEvidence(t *testing.T) {
	for _, kind := range []string{"birth", "baptism", "christening", "death", "burial", "cremation", "marriage", "census"} {
		role, topic, label := "child", glxlib.QuestionBirth, "Church records"
		switch kind {
		case "birth":
			label = "Birth record"
		case "death", "burial", "cremation":
			role, topic, label = "deceased", glxlib.QuestionDeath, "Death record"
		case "marriage":
			role, topic, label = "bride", glxlib.QuestionMarriage, ""
		case "census":
			role, label = "witness", "1870 US Census"
		}
		archive := &glxlib.GLXFile{
			Persons: map[string]*glxlib.Person{"p": {}},
			Events: map[string]*glxlib.Event{
				"event": {Type: kind, Date: "1870", Participants: []glxlib.Participant{{Person: "p", Role: "witness"}, {Person: "p", Role: role}}},
				"birth": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}},
			},
			Media:      map[string]*glxlib.Media{"record": {URI: "record.jpg"}},
			Assertions: map[string]*glxlib.Assertion{"claim": {Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1870", Confidence: "high", Media: []string{"record"}}},
		}
		if kind != "census" {
			delete(archive.Events, "birth")
			proof, err := glxlib.BuildProof(archive, "p", topic, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Len(t, proof.Evidence, 1, "a preceding witness role must not hide the principal role")
		}
		coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: glxlib.CensusCountryUnitedStates})
		require.NoError(t, err)
		matched := kind == "marriage" || kind == "burial" || kind == "cremation"
		for _, record := range coverage.Records {
			if record.Label == label && kind != "burial" && kind != "cremation" {
				require.True(t, record.Found)
				matched = true
			}
			if kind == "census" && record.Category == "census" && record.SourceRef == "event" {
				require.True(t, record.Found)
				matched = true
			}
		}
		require.True(t, matched, "expected principal or census record for %s", kind)
	}
}

func TestResearchAPI_SyntheticFactsOnlyCompareConflicts(t *testing.T) {
	for _, kind := range []string{"birth", "death"} {
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Events: map[string]*glxlib.Event{
			"a": {Type: kind, Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}},
			"b": {Type: kind, Date: "1850-03-02", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}},
		}}
		proof, err := glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion)
		require.Empty(t, proof.Evidence)
		require.Empty(t, proof.Undated)
		archive.Events["b"].Date = "1900"
		proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, glxlib.ConclusionConflicted, proof.Conclusion)
		require.Empty(t, proof.Evidence)
		require.True(t, proof.Conflicts[0].Facts[0].Synthetic)
		archive.Events["b"].Date = "1850-03-02"
		archive.Assertions = map[string]*glxlib.Assertion{"real": {Subject: glxlib.EntityRef{Event: "a"}, Property: "date", Value: "1850", Confidence: "high"}}
		proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, glxlib.ConclusionProven, proof.Conclusion)
		require.Len(t, proof.Evidence, 1)
		require.Equal(t, "real", proof.Evidence[0].AssertionID)
	}
}

func TestResearchAPI_VitalSourceSelectionIncludesMedia(t *testing.T) {
	for _, media := range []bool{false, true} {
		archive := &glxlib.GLXFile{
			Persons: map[string]*glxlib.Person{"p": {}},
			Sources: map[string]*glxlib.Source{
				"death-source": {Title: "Death certificate", Type: "vital_record"},
				"birth-source": {Title: "Birth certificate", Type: "vital_record"},
			},
			Media: map[string]*glxlib.Media{"death-media": {URI: "death.jpg", Source: "death-source"}, "birth-media": {URI: "birth.jpg", Source: "birth-source"}},
			Assertions: map[string]*glxlib.Assertion{
				"a-death": {Subject: glxlib.EntityRef{Person: "p"}, Sources: []string{"death-source"}},
				"z-birth": {Subject: glxlib.EntityRef{Person: "p"}, Sources: []string{"birth-source"}},
			},
		}
		if media {
			archive.Assertions["a-death"].Sources = nil
			archive.Assertions["a-death"].Media = []string{"death-media"}
			archive.Assertions["z-birth"].Sources = nil
			archive.Assertions["z-birth"].Media = []string{"birth-media"}
		}
		before := snapshot(t, archive)
		for range 16 {
			coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
			require.NoError(t, err)
			for _, record := range coverage.Records {
				switch record.Label {
				case "Birth record":
					require.True(t, record.Found)
					require.Equal(t, "birth-source", record.SourceRef)
				case "Death record":
					require.True(t, record.Found)
					require.Equal(t, "death-source", record.SourceRef)
				}
			}
		}
		require.JSONEq(t, before, snapshot(t, archive))
		if media {
			archive.Sources["birth-source"] = nil
			coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{})
			require.NoError(t, err)
			for _, record := range coverage.Records {
				if record.Label == "Birth record" {
					require.False(t, record.Found)
					require.Empty(t, record.SourceRef)
				}
			}
		}
	}
}

func TestResearchAPI_MediaLinkedPersonCensusSource(t *testing.T) {
	backing := []string{"unrelated", "sentinel"}
	archive := &glxlib.GLXFile{
		Persons:    map[string]*glxlib.Person{"p": {}},
		Events:     map[string]*glxlib.Event{"birth": {Type: "birth", Date: "1850", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}},
		Sources:    map[string]*glxlib.Source{"census-source": {Title: "1870 census", Type: "census", Date: "1870"}, "unrelated": {Title: "Other document"}},
		Media:      map[string]*glxlib.Media{"scan": {URI: "census.jpg", Source: "census-source"}},
		Assertions: map[string]*glxlib.Assertion{"claim": {Subject: glxlib.EntityRef{Person: "p"}, Sources: backing[:1], Media: []string{"scan"}}},
	}
	before := snapshot(t, archive)
	coverage, err := glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: glxlib.CensusCountryUnitedStates})
	require.NoError(t, err)
	found := false
	for _, record := range coverage.Records {
		found = found || record.Found && record.SourceRef == "census-source"
	}
	require.True(t, found)
	require.JSONEq(t, before, snapshot(t, archive))
	require.Equal(t, []string{"unrelated", "sentinel"}, backing)
	archive.Media["scan"] = nil
	coverage, err = glxlib.BuildCoverage(archive, "p", glxlib.CoverageOptions{CensusCountry: glxlib.CensusCountryUnitedStates})
	require.NoError(t, err)
	for _, record := range coverage.Records {
		require.NotEqual(t, "census-source", record.SourceRef)
	}
}

func TestResearchAPI_LegacyAliasesArePersonScoped(t *testing.T) {
	for _, subject := range []glxlib.EntityRef{{Event: "event"}, {Relationship: "relationship"}, {Place: "place"}} {
		archive := &glxlib.GLXFile{
			Persons:       map[string]*glxlib.Person{"p": {}},
			Events:        map[string]*glxlib.Event{"event": {Type: "birth"}},
			Relationships: map[string]*glxlib.Relationship{"relationship": {Type: "marriage"}},
			Places:        map[string]*glxlib.Place{"place": {Name: "Subject place"}, "paris": {Name: "Paris, France"}},
		}
		for _, property := range []string{"born_on", "birth_date", "born_at", "birth_place"} {
			first, second := "1850", "1850-02-01"
			if property == "born_at" || property == "birth_place" {
				first, second = "paris", "other"
			}
			archive.Assertions = map[string]*glxlib.Assertion{
				"first":  {Subject: subject, Property: property, Value: first},
				"second": {Subject: subject, Property: property, Value: second},
			}
			report, err := glxlib.BuildEvidenceReport(archive, subject, property, glxlib.ComparisonOptions{})
			require.NoError(t, err)
			require.Len(t, report.Conflicts, 1)
			require.Equal(t, glxlib.VerdictDefinite, report.Conflicts[0].Verdict)
			for _, group := range report.Groups {
				require.NotEqual(t, "Paris, France", group.Value, "unknown non-person aliases stay literal")
			}
		}
	}
	archive := &glxlib.GLXFile{
		Events:          map[string]*glxlib.Event{"event": {Type: "birth"}},
		EventProperties: map[string]*glxlib.PropertyDefinition{"birth_date": {ValueType: "date"}},
		Assertions: map[string]*glxlib.Assertion{
			"first":  {Subject: glxlib.EntityRef{Event: "event"}, Property: "birth_date", Value: "1850"},
			"second": {Subject: glxlib.EntityRef{Event: "event"}, Property: "birth_date", Value: "1850-02-01"},
		},
	}
	report, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Event: "event"}, "birth_date", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Empty(t, report.Conflicts, "an explicit event definition still controls its own property")
}

func TestResearchAPI_UnrelatedResolutionsPreserveVitalAnswers(t *testing.T) {
	for _, kind := range []string{"birth", "death"} {
		archive := &glxlib.GLXFile{
			Persons:         map[string]*glxlib.Person{"p": {}},
			Events:          map[string]*glxlib.Event{"event": {Type: kind, Date: "1900", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}},
			EventProperties: map[string]*glxlib.PropertyDefinition{"ceremony": {ValueType: "string"}},
			Assertions: map[string]*glxlib.Assertion{
				"old":             {Subject: glxlib.EntityRef{Event: "event"}, Property: "ceremony", Value: "church", Status: "disproven"},
				"live":            {Subject: glxlib.EntityRef{Event: "event"}, Property: "ceremony", Value: "home", Status: "proven", Confidence: "high"},
				"supporting-date": {Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1900", Confidence: "low"},
			},
		}
		proof, err := glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Contains(t, proof.Summary, "1900")
		require.NotContains(t, proof.Summary, "home")
		require.True(t, proof.Conflicts[0].Resolved)
		archive.Events["event"].Date = ""
		delete(archive.Assertions, "supporting-date")
		proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion, "a ceremony is not a date/place answer")
		archive.Events["event"].Date = "1900"
		archive.Assertions["old-date"] = &glxlib.Assertion{Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1900", Status: "disproven"}
		archive.Assertions["live-date"] = &glxlib.Assertion{Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1910", Status: "proven"}
		proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
		require.NoError(t, err)
		require.Contains(t, proof.Summary, "1910", "a resolved vital claim must replace stale structural data")
		require.NotContains(t, proof.Summary, "1900")
		require.NotContains(t, proof.Summary, "home")
	}
}

func TestResearchAPI_OnlyAnsweringClaimsStrengthenProof(t *testing.T) {
	for _, kind := range []string{"birth", "death", "marriage"} {
		t.Run(kind, func(t *testing.T) {
			archive := &glxlib.GLXFile{
				Persons:         map[string]*glxlib.Person{"p": {}, "q": {Properties: map[string]any{"name": "Alex"}}},
				Events:          map[string]*glxlib.Event{"event": {Type: kind, Date: "1900", Participants: []glxlib.Participant{{Person: "p", Role: "subject"}}}},
				EventProperties: map[string]*glxlib.PropertyDefinition{"ceremony": {ValueType: "string"}},
				Assertions:      map[string]*glxlib.Assertion{"context": {Subject: glxlib.EntityRef{Event: "event"}, Property: "ceremony", Value: "home", Confidence: "high", Status: "proven"}},
			}
			if kind == "marriage" {
				archive.Relationships = map[string]*glxlib.Relationship{"couple": {Type: "marriage", Participants: []glxlib.Participant{{Person: "p", Role: "spouse"}, {Person: "q", Role: "spouse"}}}}
			}
			proof, err := glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, glxlib.ConclusionInsufficient, proof.Conclusion)
			require.Len(t, proof.Evidence, 1, "related context remains available for inspection")
			archive.Assertions["date"] = &glxlib.Assertion{Subject: glxlib.EntityRef{Event: "event"}, Property: "date", Value: "1900", Confidence: "low"}
			proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, glxlib.ConclusionPossible, proof.Conclusion, "unrelated high confidence must not upgrade weak actual support")
			archive.Assertions["date"].Confidence = "high"
			proof, err = glxlib.BuildProof(archive, "p", kind, glxlib.ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, glxlib.ConclusionProven, proof.Conclusion)
		})
	}
}

func TestResearchAPI_MediaReportsPreserveIdentityAndDeduplicate(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Media:   map[string]*glxlib.Media{"a-media": {URI: "a.jpg", Title: "Record scan"}, "b-media": {URI: "b.jpg", Title: "Record scan"}, "c-media": {URI: "c.jpg", Title: "Record scan"}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "claim", Value: "A", Media: []string{"a-media"}, Confidence: "low"},
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "claim", Value: "A", Media: []string{"a-media", "a-media"}, Confidence: "high"},
			"b1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "claim", Value: "B", Media: []string{"c-media", "b-media"}, Confidence: "high"},
		},
	}
	before := snapshot(t, archive)
	report, err := glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Person: "p"}, "claim", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, report.TotalReports)
	require.Equal(t, "B", report.BestEvidence)
	require.Len(t, report.Groups, 2)
	require.Equal(t, 2, report.Groups[0].Reports)
	require.Equal(t, "b-media", report.Groups[0].Items[0].MediaID)
	require.Equal(t, "c-media", report.Groups[0].Items[1].MediaID)
	require.Equal(t, "Record scan", report.Groups[0].Items[0].Source)
	require.Equal(t, 1, report.Groups[1].Reports)
	require.Equal(t, "a-media", report.Groups[1].Items[0].MediaID)
	require.Equal(t, "high", report.Groups[1].Items[0].Confidence)
	report.Groups[1].Items[0].MediaID = "changed"
	require.JSONEq(t, before, snapshot(t, archive))
	// Reusing a media object for a distinct participation claim is independent support.
	archive.Events = map[string]*glxlib.Event{"event": {Type: "birth"}}
	archive.Assertions = map[string]*glxlib.Assertion{
		"parent":  {Subject: glxlib.EntityRef{Event: "event"}, Participant: &glxlib.Participant{Person: "p", Role: "parent"}, Media: []string{"a-media"}},
		"witness": {Subject: glxlib.EntityRef{Event: "event"}, Participant: &glxlib.Participant{Person: "p", Role: "witness"}, Media: []string{"a-media"}},
	}
	report, err = glxlib.BuildEvidenceReport(archive, glxlib.EntityRef{Event: "event"}, "", glxlib.ComparisonOptions{})
	require.NoError(t, err)
	require.Len(t, report.Groups, 2)
	require.Equal(t, 2, report.TotalReports)
}
