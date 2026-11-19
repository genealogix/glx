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

func TestDuplicatesFollowup_NonLatinNamesStillMatch(t *testing.T) {
	archive := &GLXFile{Persons: map[string]*Person{
		"a": duplicateTestPerson("Иван", "Петров", "male"),
		"b": duplicateTestPerson("Иван", "Петров", "male"),
	}}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Len(t, result.Pairs, 1, "identical names outside the Latin script have no Soundex code but are still identity evidence")
}

func TestDuplicatesFollowup_SourceAssertionSurvivesIDCollision(t *testing.T) {
	dest := &GLXFile{Persons: map[string]*Person{"a": duplicateTestPerson("Johann", "Roth", "male")}, Relationships: map[string]*Relationship{
		"ra": {Type: "parent_child", Participants: []Participant{{Person: "a", Role: "parent"}, {Person: "child", Role: "child"}}},
	}, Assertions: map[string]*Assertion{"a1": {Subject: EntityRef{Relationship: "ra"}, Status: "proven"}}}
	src := &GLXFile{Persons: map[string]*Person{"b": duplicateTestPerson("Johann", "Roth", "male")}, Relationships: map[string]*Relationship{
		"rb": {Type: "parent_child", Participants: []Participant{{Person: "b", Role: "parent"}, {Person: "child", Role: "child"}}},
	}, Assertions: map[string]*Assertion{"a1": {Subject: EntityRef{Relationship: "rb"}, Status: "disproven"}}}

	combined := buildCombinedView(dest, src)
	assert.Same(t, dest.Assertions["a1"], combined.Assertions["a1"], "destination keeps its own ID")
	assert.Len(t, combined.Assertions, 2, "the source claim is kept under a view-only key")
	standing := NewRelationshipStandingIndex(combined)
	assert.Equal(t, RelationshipStandingDisproven, standing.Relationship("rb"))
	assert.NotEqual(t, RelationshipStandingDisproven, standing.Relationship("ra"))

	// An identical claim under the same ID is one claim, not two.
	same := *dest.Assertions["a1"]
	src.Assertions["a1"] = &same
	src.Relationships["ra"] = dest.Relationships["ra"]
	assert.Len(t, buildCombinedView(dest, src).Assertions, 1)
}

func TestDuplicatesFollowup_ParentChildSubtypesAndRoleCaseShareAPeer(t *testing.T) {
	archive := &GLXFile{Relationships: map[string]*Relationship{
		"ra": {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "a", Role: "parent"}, {Person: "c", Role: "child"}}},
		"rb": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{{Person: "b", Role: " Parent"}, {Person: "c", Role: "Child"}}},
	}}
	score, detail, hasData := scoreSharedRelationships("a", "b", buildDuplicateIndex(archive))
	assert.True(t, hasData)
	assert.InDelta(t, 1, score, 1e-12, detail)

	// A spouse of C is still not the same kind of tie as a parent of C.
	archive.Relationships["rb"] = &Relationship{Type: "marriage", Participants: []Participant{{Person: "b", Role: "spouse"}, {Person: "c", Role: "spouse"}}}
	score, _, _ = scoreSharedRelationships("a", "b", buildDuplicateIndex(archive))
	assert.Zero(t, score)
}

func TestDuplicatesFollowup_GivenNameFromDisplayValue(t *testing.T) {
	partial := &Person{Properties: map[string]any{"name": map[string]any{"value": "Anna Roth", "fields": map[string]any{"surname": "Roth"}}}}
	given, surname := duplicateNameParts(partial)
	assert.Equal(t, "Anna", given)
	assert.Equal(t, "Roth", surname)

	bare := &Person{Properties: map[string]any{"name": map[string]any{"value": "Roth", "fields": map[string]any{"surname": "Roth"}}}}
	given, _ = duplicateNameParts(bare)
	assert.Empty(t, given, "a surname-only value has no given name to recover")

	mismatched := &Person{Properties: map[string]any{"name": map[string]any{"value": "Anna Weber", "fields": map[string]any{"surname": "Roth"}}}}
	given, _ = duplicateNameParts(mismatched)
	assert.Empty(t, given, "the display value is used only when it ends in the recorded surname")

	full := duplicateTestPerson("Anna", "Roth", "female")
	archive := &GLXFile{Persons: map[string]*Person{"a": partial, "b": full}}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Len(t, result.Pairs, 1)
}

func TestDuplicatesFollowup_PhoneticBlocksSplitByGivenInitial(t *testing.T) {
	archive := &GLXFile{Persons: map[string]*Person{
		"js": duplicateTestPerson("Johann", "Schmidt", "male"),
		"jt": duplicateTestPerson("Johann", "Smith", "male"),
		"ks": duplicateTestPerson("Karl", "Schmitt", "male"),
	}}
	blocks := buildSurnameBlocks(archive)
	assert.ElementsMatch(t, []string{"js", "jt"}, blocks["soundex:S530:J"])
	assert.ElementsMatch(t, []string{"ks"}, blocks["soundex:S530:K"])
	assert.ElementsMatch(t, []string{"js"}, blocks["exact:schmidt"])
}

func TestDuplicatesFollowup_IndexParsesEachEventDateOnce(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{
		"death":    {Type: "death", Date: "1674-06-17", Participants: []Participant{{Person: "a", Role: "principal"}}},
		"activity": {Type: "baptism", Date: "1685", Participants: []Participant{{Person: "b", Role: "godparent"}}},
		"undated":  {Type: "baptism", Participants: []Participant{{Person: "b", Role: "witness"}}},
	}}
	idx := buildDuplicateIndex(archive)
	assert.Len(t, idx.eventDates, 3)
	assert.True(t, idx.eventDates[archive.Events["death"]].parsed)
	assert.False(t, idx.eventDates[archive.Events["undated"]].calendarAgrees(idx.eventDates[archive.Events["death"]]))
	assert.NotEmpty(t, duplicateLifetimeConflict("a", "b", archive, idx))
}
