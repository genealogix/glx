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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoundex_LatinFolding(t *testing.T) {
	for _, pair := range [][2]string{{"Puß", "Puss"}, {"Puß", "Pusch"}, {"Joß", "Joss"}, {"Öhler", "Ohler"}, {"Übel", "Ubel"}, {"Émile", "Emile"}, {"E\u0301mile", "Emile"}, {"Łukasz", "Lukasz"}, {"ẞander", "Ssander"}} {
		assert.Equal(t, Soundex(pair[1]), Soundex(pair[0]), pair)
	}
	assert.Empty(t, Soundex("東京"))
}

func duplicateTestPerson(given, surname, sex string) *Person {
	return &Person{Properties: map[string]any{
		"name": map[string]any{"value": given + " " + surname, "fields": map[string]any{"given": given, "surname": surname}},
		"sex":  sex,
	}}
}

func TestDuplicates_RecordedSexAndSparseIdentity(t *testing.T) {
	for _, scenario := range []struct {
		name, givenA, givenB, sexA, sexB string
		want                             bool
	}{
		{"opposite sex", "Anna", "Anna", "male", "female", false},
		{"unknown neutral", "Anna", "Anna", "unknown", "female", true},
		{"unrecorded neutral", "Anna", "Anna", "not_recorded", "female", true},
		{"surname only", "", "Anna", "female", "female", false},
		{"full names", "Anna", "Anna", "female", "female", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			archive := &GLXFile{Persons: map[string]*Person{
				"a": duplicateTestPerson(scenario.givenA, "Roth", scenario.sexA),
				"b": duplicateTestPerson(scenario.givenB, "Roth", scenario.sexB),
			}}
			result, err := FindDuplicates(archive, DuplicateOptions{Threshold: .6})
			require.NoError(t, err)
			assert.Equal(t, scenario.want, len(result.Pairs) > 0)
		})
	}
	archive := &GLXFile{Persons: map[string]*Person{"a": {}, "b": {}}, Events: map[string]*Event{
		"e": {Type: "baptism", Participants: []Participant{{Person: "a", Role: "godparent"}, {Person: "b", Role: "godparent"}}},
	}}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: .6})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs, "a shared event alone must not produce a perfect match")
}

func TestDuplicates_PhoneticBlockingAtBoundaryAndAcrossArchives(t *testing.T) {
	for _, count := range []int{199, 200, 300} {
		archive := &GLXFile{Persons: map[string]*Person{
			"a": duplicateTestPerson("Johann Philipp", "Langstroff", "male"),
			"b": duplicateTestPerson("Johann Philipp", "Langsdorff", "male"),
		}}
		for i := 2; i < count; i++ {
			archive.Persons[fmt.Sprintf("f%d", i)] = duplicateTestPerson("Other", fmt.Sprintf("Filler%d", i), "female")
		}
		result, err := FindDuplicates(archive, DuplicateOptions{Threshold: .6, PersonFilter: ""})
		require.NoError(t, err)
		found := false
		for _, pair := range result.Pairs {
			if pair.PersonA == "a" && pair.PersonB == "b" {
				found = true
			}
		}
		assert.True(t, found, count)
		dest := &GLXFile{Persons: map[string]*Person{"a": archive.Persons["a"]}}
		delete(archive.Persons, "a")
		cross, err := FindCrossArchiveDuplicates(dest, archive, DuplicateOptions{Threshold: .6})
		require.NoError(t, err)
		found = false
		for _, pair := range cross.Pairs {
			if pair.PersonA == "a" && pair.PersonB == "b" {
				found = true
			}
		}
		assert.True(t, found, count)
	}
}

func TestDuplicates_SocialSignalsRespectRolesAndRejectedRelationships(t *testing.T) {
	archive := &GLXFile{Persons: map[string]*Person{
		"a":     duplicateTestPerson("Johann", "Schepp", "male"),
		"b":     duplicateTestPerson("Johann", "Muerckel", "male"),
		"child": duplicateTestPerson("Anna", "Schepp", "female"),
	}, Relationships: map[string]*Relationship{
		"parent":   {Type: "parent_child", Participants: []Participant{{Person: "a", Role: "parent"}, {Person: "child", Role: "child"}}},
		"marriage": {Type: "marriage", Participants: []Participant{{Person: "b", Role: "spouse"}, {Person: "child", Role: "spouse"}}},
	}, Events: map[string]*Event{
		"e": {Type: "death", Participants: []Participant{{Person: "child", Role: "principal"}, {Person: "a", Role: "parent"}, {Person: "b", Role: "spouse"}}},
	}}
	idx := buildDuplicateIndex(archive)
	score, _, _ := scoreSharedRelationships("a", "b", idx)
	assert.Zero(t, score)
	score, _, _ = scoreSharedEvents("a", "b", idx)
	assert.Zero(t, score)
	// Same roles to the same third person remain useful corroboration.
	archive.Relationships["marriage"].Type = "parent_child"
	archive.Relationships["marriage"].Participants[0].Role = "parent"
	archive.Relationships["marriage"].Participants[1].Role = "child"
	idx = buildDuplicateIndex(archive)
	score, _, _ = scoreSharedRelationships("a", "b", idx)
	assert.InDelta(t, 1, score, 1e-12)
	archive.Assertions = map[string]*Assertion{"rejected": {Subject: EntityRef{Relationship: "marriage"}, Status: "disproven"}}
	idx = buildDuplicateIndex(archive)
	score, _, _ = scoreSharedRelationships("a", "b", idx)
	assert.Zero(t, score)
}

func TestDuplicates_LifetimeContradictionRespectsRolesAndDates(t *testing.T) {
	for _, scenario := range []struct {
		role, eventType, date string
		want                  bool
	}{
		{"godparent", "baptism", "1685-04-19", false},
		{"parent", "marriage", "1685-04-19", true},
		{"principal", "burial", "1685-04-19", true},
		{"godparent", "baptism", "1674-06-17", true},
		{"godparent", "baptism", "ABT 1674", true},
	} {
		t.Run(scenario.role+scenario.eventType+scenario.date, func(t *testing.T) {
			archive := &GLXFile{Persons: map[string]*Person{"a": duplicateTestPerson("Johannes", "Schopp", "male"), "b": duplicateTestPerson("Johannes", "Schopp", "male")}, Events: map[string]*Event{
				"death":    {Type: "death", Date: "1674-06-17", Participants: []Participant{{Person: "a", Role: "principal"}}},
				"activity": {Type: scenario.eventType, Date: DateString(scenario.date), Participants: []Participant{{Person: "b", Role: scenario.role}}},
			}}
			idx := buildDuplicateIndex(archive)
			assert.Equal(t, scenario.want, duplicateLifetimeConflict("a", "b", archive, idx) == "")
		})
	}
}
