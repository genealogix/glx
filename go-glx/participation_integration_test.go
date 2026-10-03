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

func TestAssertionCoverageMissingEntitiesDoNotPropagate(t *testing.T) {
	for _, missing := range []string{"missing", "nil"} {
		t.Run(missing, func(t *testing.T) {
			archive := &GLXFile{
				Persons:       map[string]*Person{"p": {}, "nil": nil},
				Events:        map[string]*Event{"nil": nil},
				Relationships: map[string]*Relationship{"r": {StartEvent: missing, Participants: []Participant{{Person: "p"}}}},
				Assertions:    map[string]*Assertion{"a": {Subject: EntityRef{Event: missing}}},
			}
			coverage := ComputeAssertionCoverage(archive)
			require.Empty(t, coverage.Direct.Events)
			require.Empty(t, coverage.Evidence.Events)
			require.Empty(t, coverage.Evidence.Relationships)
			require.Empty(t, coverage.Evidence.Persons)
		})
	}
	archive := &GLXFile{
		Persons: map[string]*Person{"nil": nil}, Places: map[string]*Place{"nil": nil},
		Events:     map[string]*Event{"e": {PlaceID: "nil", Participants: []Participant{{Person: "nil"}, {Person: "missing"}}}},
		Assertions: map[string]*Assertion{"a": {Subject: EntityRef{Event: "e"}}, "p": {Subject: EntityRef{Person: "nil"}}, "v": {Subject: EntityRef{Event: "e"}, Property: "place", Value: "nil"}},
	}
	coverage := ComputeAssertionCoverage(archive)
	require.Empty(t, coverage.Direct.Persons)
	require.Empty(t, coverage.Evidence.Persons)
	require.Empty(t, coverage.Evidence.Places)
}

func TestAssertionCoverageRespectsPropertyDefinitionsAndDefaults(t *testing.T) {
	for _, def := range []*PropertyDefinition{nil, {ValueType: "string"}} {
		archive := &GLXFile{Events: map[string]*Event{"e": {}}, Places: map[string]*Place{"home": {}}, EventProperties: map[string]*PropertyDefinition{"place": def}, Assertions: map[string]*Assertion{"a": {Subject: EntityRef{Event: "e"}, Property: "place", Value: "home"}}}
		require.Empty(t, ComputeAssertionCoverage(archive).Evidence.Places)
	}
	archive := &GLXFile{Persons: map[string]*Person{"p": {}}, Places: map[string]*Place{"home": {}}, Assertions: map[string]*Assertion{"a": {Subject: EntityRef{Person: "p"}, Property: "residence", Value: "home"}}}
	require.Equal(t, map[string]bool{"home": true}, ComputeAssertionCoverage(archive).Evidence.Places)
	require.Nil(t, archive.PersonProperties, "coverage must not load defaults into the caller's archive")
}

func TestCoveragePresenceAndOwnRecordRemainIndependent(t *testing.T) {
	no := false
	archive := &GLXFile{
		Persons:          map[string]*Person{"p": {}},
		Places:           map[string]*Place{"us": {Name: "United States", Type: "country"}, "wi": {Name: "Wisconsin", Type: "state", ParentID: "us"}, "fr": {Name: "France", Type: "country"}},
		ParticipantRoles: map[string]*VocabularyEntry{"principal": {ImpliesPresence: &no}},
		Events:           map[string]*Event{"birth": {Type: "birth", Date: "1850", PlaceID: "fr", Participants: []Participant{{Person: "p", Role: "child"}}}, "remote": {Type: "probate", Date: "1870", PlaceID: "wi", Participants: []Participant{{Person: "p", Role: "principal"}}}},
		Sources:          map[string]*Source{"s": {}}, Assertions: map[string]*Assertion{"a": {Subject: EntityRef{Event: "remote"}, Sources: []string{"s"}}},
	}
	result, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	require.Equal(t, "1850", result.BirthDate)
	for _, record := range result.Records {
		require.NotEqual(t, "census", record.Category, "the remote own record must not imply US presence")
		if record.Label == "Probate/will" {
			require.True(t, record.Found)
		}
	}
}

func TestCoveragePresentWitnessRetainsSurvivingCensusGap(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Places:  map[string]*Place{"us": {Name: "United States", Type: "country"}, "oh": {Name: "Ohio", Type: "state", ParentID: "us"}, "pa": {Name: "Pennsylvania", Type: "state", ParentID: "us"}},
		Events:  map[string]*Event{"birth": {Type: "birth", Date: "1800", PlaceID: "oh", Participants: []Participant{{Person: "p", Role: "child"}}}, "death": {Type: "death", Date: "1820", PlaceID: "oh", Participants: []Participant{{Person: "p", Role: "decedent"}}}, "wedding": {Type: "marriage", Date: "1810", PlaceID: "pa", Participants: []Participant{{Person: "p", Role: "witness"}}}},
	}
	result, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	require.Equal(t, "1820", result.DeathDate)
	found := false
	for _, record := range result.Records {
		if record.Label == "1810 US Census (age ~10)" {
			found = true
			require.False(t, record.Found)
		}
	}
	require.True(t, found, "a present observation in surviving Pennsylvania keeps the research gap")
	proof, err := BuildProof(archive, "p", QuestionBirth, ProofOptions{})
	require.NoError(t, err)
	found = false
	for _, gap := range proof.Gaps {
		if gap.Label == "1810 US Census (age ~10)" {
			found = true
		}
	}
	require.True(t, found, "proof gaps must use the shared coverage calculation")
}

func TestCoverageWitnessMarriageDoesNotCountAsCouplesOwn(t *testing.T) {
	archive := &GLXFile{
		Persons:       map[string]*Person{"p": {}, "q": {}, "other": {}},
		Relationships: map[string]*Relationship{"r": {Type: "marriage", Participants: []Participant{{Person: "p", Role: "spouse"}, {Person: "q", Role: "spouse"}}}},
		Events:        map[string]*Event{"other-wedding": {Type: "marriage", Participants: []Participant{{Person: "p", Role: "witness"}, {Person: "q", Role: "witness"}, {Person: "other", Role: "principal"}}}},
		Sources:       map[string]*Source{"s": {}}, Assertions: map[string]*Assertion{"a": {Subject: EntityRef{Event: "other-wedding"}, Sources: []string{"s"}}},
	}
	result, err := BuildCoverage(archive, "p", CoverageOptions{})
	require.NoError(t, err)
	for _, record := range result.Records {
		if record.Category == "vital" {
			require.False(t, record.Found)
		}
	}
	require.Len(t, result.AppearsIn, 1)
}
