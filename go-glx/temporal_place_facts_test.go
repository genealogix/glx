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

	"github.com/stretchr/testify/require"
)

// Both sources assert the place, but neither assertion repeats the date of
// the event. The comparison may use that date; the source claims must not be
// rewritten to look as though they asserted it.
func TestHistoricalPlaceClaimsUseEventDate(t *testing.T) {
	for _, tc := range []struct {
		date       DateString
		conclusion string
		conflict   Verdict
	}{
		{"1840-06-01", proofConclusionProven, ""},
		{"1850-06-01", proofConclusionConflicted, VerdictDefinite},
		{"1841", proofConclusionPossible, VerdictPossible},
	} {
		t.Run(string(tc.date), func(t *testing.T) {
			archive := &GLXFile{
				Persons: map[string]*Person{"rowan": {}},
				Places: map[string]*Place{
					"village": {Name: "Alder Village", ParentID: "district"},
					"district": {Name: "Alder District", ParentHistory: []PlaceParentPeriod{
						{Value: "old", Date: "TO 1841-06-30"},
						{Value: "new", Date: "FROM 1841-07-01"},
					}},
					"old": {Name: "Old County"}, "new": {Name: "New County"},
				},
				Events: map[string]*Event{"birth": {
					Type: EventTypeBirth, Date: tc.date, PlaceID: "village",
					Participants: []Participant{{Person: "rowan", Role: ParticipantRoleSubject}},
				}},
				Sources:   map[string]*Source{"church": {Title: "Parish register"}, "civil": {Title: "County register"}},
				Citations: map[string]*Citation{"a": {SourceID: "church"}, "b": {SourceID: "civil"}},
				Assertions: map[string]*Assertion{
					"town-claim":   {Subject: EntityRef{Event: "birth"}, Property: eventFieldPlace, Value: "village", Citations: []string{"a"}, Confidence: "high"},
					"county-claim": {Subject: EntityRef{Event: "birth"}, Property: eventFieldPlace, Value: "old", Citations: []string{"b"}, Confidence: "high"},
				},
			}
			result, err := BuildProof(archive, "rowan", QuestionBirth, ProofOptions{})
			require.NoError(t, err)
			require.Equal(t, tc.conclusion, result.Conclusion)
			evidence, err := BuildEvidenceReport(archive, EntityRef{Event: "birth"}, eventFieldPlace, ComparisonOptions{})
			require.NoError(t, err)
			if tc.conflict == "" {
				require.Empty(t, result.Conflicts)
				require.Empty(t, evidence.Conflicts)
			} else {
				require.Len(t, result.Conflicts, 1)
				require.Equal(t, tc.conflict, result.Conflicts[0].Verdict)
				require.Len(t, evidence.Conflicts, 1)
				require.Equal(t, tc.conflict, evidence.Conflicts[0].Verdict)
				for _, claim := range evidence.Conflicts[0].Values {
					require.Empty(t, claim.Date)
				}
			}
			require.Empty(t, archive.Assertions["county-claim"].Date)
		})
	}
}

func TestMergeDatedResidenceUsesContemporaneousAncestor(t *testing.T) {
	for _, tc := range []struct {
		date      string
		collision bool
	}{{"1814-01-02", true}, {"1820-01-02", false}} {
		t.Run(tc.date, func(t *testing.T) {
			archive := indianaTerritoryArchive()
			location := func(id string) *Person {
				return &Person{Properties: map[string]any{PersonPropertyResidence: map[string]any{"value": id, "date": tc.date}}}
			}
			archive.Persons = map[string]*Person{"a": location("place-wayne-in"), "b": location("place-indiana")}
			require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))
			r, err := MergePersons(archive, "a", "b", MergePersonsOptions{})
			require.NoError(t, err)
			if !tc.collision {
				require.Empty(t, r.Conflicts)
			} else {
				require.Len(t, r.Conflicts, 1)
				require.Equal(t, VerdictDefinite, r.Conflicts[0].Verdict)
			}
		})
	}
}

func TestGEDCOMPlaceProjectionCannotCombineDisjointEdges(t *testing.T) {
	archive := indianaTerritoryArchive()
	archive.Places["place-bridge"] = &Place{Name: "Bridge", ParentHistory: []PlaceParentPeriod{
		{Value: "place-wayne-in", Date: "FROM 1816-12-11"},
		{Value: "place-usa", Date: "TO 1816-12-10"},
	}}
	exp := &ExportContext{GLX: archive}
	require.Equal(t, "Bridge", buildPlaceString("place-bridge", "1816", exp))
	require.Equal(t, "Bridge, Wayne County, Indiana, United States", buildPlaceString("place-bridge", "1816-12-11", exp))
	require.Equal(t, "Bridge, United States", buildPlaceString("place-bridge", "1816-12-10", exp))
}

func TestFixedPlaceClaimsWithDifferentRecordedDates(t *testing.T) {
	a := indianaTerritoryArchive()
	claims := []FactValue{{Value: "place-wayne-in", Date: "1814-01-01"}, {Value: "place-usa", Date: "1820-01-01"}}
	def := &PropertyDefinition{ReferenceType: EntityTypePlaces.String()}
	for _, tc := range []struct {
		place string
		want  Verdict
	}{
		{"place-usa", VerdictRefinement}, {"place-indiana", VerdictPossible}, {"other", VerdictDefinite},
	} {
		claims[1].Value = tc.place
		comparison, err := CompareFacts(claims, def, a.Places, ComparisonOptions{})
		require.NoError(t, err)
		require.Equal(t, tc.want, comparison[0].Verdict, tc.place)
	}
	temporal := true
	def.Temporal = &temporal
	claims[1].Value = "place-usa"
	comparison, err := CompareFacts(claims, def, a.Places, ComparisonOptions{})
	require.NoError(t, err)
	require.Equal(t, VerdictHistory, comparison[0].Verdict)
}
