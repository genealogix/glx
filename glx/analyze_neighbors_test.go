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

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// neighborArchive has the 1850 Baker household, with a wife, and the given page
// neighbors. person-henry and person-james exist but are unconnected.
func neighborArchive(neighbors ...glxlib.EventNeighbor) *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-abram": {Properties: map[string]any{glxlib.PersonPropertyName: "Abram Baker"}},
			"person-wife":  {Properties: map[string]any{glxlib.PersonPropertyName: "Mary Cochran"}},
			"person-henry": {Properties: map[string]any{glxlib.PersonPropertyName: "Henry Jeffries"}},
			"person-james": {Properties: map[string]any{glxlib.PersonPropertyName: "James Clark"}},
		},
		Events: map[string]*glxlib.Event{
			"event-census-1850": {
				Type: glxlib.EventTypeCensus,
				Date: "1850",
				Participants: []glxlib.Participant{
					{Person: "person-abram", Role: glxlib.ParticipantRolePrincipal},
					{Person: "person-wife", Role: glxlib.ParticipantRoleHouseholdMember},
				},
				Neighbors: neighbors,
			},
		},
	}
}

func TestSuggestCensusNeighbors_ListsNeighborsOnHead(t *testing.T) {
	archive := neighborArchive(
		glxlib.EventNeighbor{Name: "Henry Jeffries", Position: "previous_household"},
		glxlib.EventNeighbor{Person: "person-james", Position: "next_household"},
		glxlib.EventNeighbor{Name: "Ann Weir"},
	)

	issues := suggestCensusNeighbors(archive)

	require.Len(t, issues, 1)
	issue := issues[0]
	assert.Equal(t, analysisSuggestionCategory, issue.Category)
	assert.Equal(t, severityInfo, issue.Severity)
	assert.Equal(t, "person-abram", issue.Person)
	assert.Equal(t, "event-census-1850", issue.Entity)
	assert.Contains(t, issue.Message, "1850 Census")
	assert.Contains(t, issue.Message, "Henry Jeffries (previous household), James Clark (next household), Ann Weir;")
}

func TestSuggestCensusNeighbors_PageNameWinsOverLinkedPerson(t *testing.T) {
	archive := neighborArchive(glxlib.EventNeighbor{Name: "H. Jefferis", Person: "person-henry"})

	issues := suggestCensusNeighbors(archive)

	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "H. Jefferis")
	assert.NotContains(t, issues[0].Message, "Henry Jeffries")
}

func TestSuggestCensusNeighbors_NoNeighborsNoSuggestion(t *testing.T) {
	assert.Empty(t, suggestCensusNeighbors(neighborArchive()))
}

// A neighbor is dropped once tied to anyone in the household, not only the
// head: in the #180 research the tie ran through the wife's family.
func TestSuggestCensusNeighbors_TiedNeighborsCleared(t *testing.T) {
	henry := glxlib.EventNeighbor{Name: "Henry Jeffries", Person: "person-henry"}
	wedding := func(witness string) *glxlib.Event {
		return &glxlib.Event{
			Type: glxlib.EventTypeMarriage,
			Participants: []glxlib.Participant{
				{Person: "person-wife", Role: "bride"},
				{Person: witness, Role: "witness"},
			},
		}
	}

	t.Run("shared non-census event with a member", func(t *testing.T) {
		archive := neighborArchive(henry)
		archive.Events["event-wedding"] = wedding("person-henry")
		assert.Empty(t, suggestCensusNeighbors(archive))
	})

	t.Run("relationship with a member", func(t *testing.T) {
		archive := neighborArchive(henry)
		archive.Relationships = map[string]*glxlib.Relationship{
			"rel-assoc": {Type: "associate", Participants: []glxlib.Participant{
				{Person: "person-henry"}, {Person: "person-abram"},
			}},
		}
		assert.Empty(t, suggestCensusNeighbors(archive))
	})

	t.Run("neighbor is a household member", func(t *testing.T) {
		archive := neighborArchive(glxlib.EventNeighbor{Person: "person-wife"})
		assert.Empty(t, suggestCensusNeighbors(archive))
	})

	t.Run("another census does not count as a tie", func(t *testing.T) {
		archive := neighborArchive(henry)
		archive.Events["event-census-1860"] = &glxlib.Event{
			Type: glxlib.EventTypeCensus,
			Date: "1860",
			Participants: []glxlib.Participant{
				{Person: "person-abram", Role: glxlib.ParticipantRolePrincipal},
				{Person: "person-henry", Role: glxlib.ParticipantRoleHouseholdMember},
			},
		}
		assert.Len(t, suggestCensusNeighbors(archive), 1)
	})

	t.Run("name-only neighbor is never tied", func(t *testing.T) {
		archive := neighborArchive(glxlib.EventNeighbor{Name: "Henry Jeffries"})
		archive.Events["event-wedding"] = wedding("person-henry")
		assert.Len(t, suggestCensusNeighbors(archive), 1)
	})

	t.Run("a tie to a nonresident does not count", func(t *testing.T) {
		archive := neighborArchive(henry)
		census := archive.Events["event-census-1850"]
		census.Participants = append(census.Participants, glxlib.Participant{Person: "person-james", Role: "enumerator"})
		archive.Events["event-other"] = &glxlib.Event{
			Type: glxlib.EventTypeMarriage,
			Participants: []glxlib.Participant{
				{Person: "person-james", Role: "groom"},
				{Person: "person-henry", Role: "witness"},
			},
		}
		assert.Len(t, suggestCensusNeighbors(archive), 1)
	})

	t.Run("only the untied neighbor is listed", func(t *testing.T) {
		archive := neighborArchive(henry, glxlib.EventNeighbor{Person: "person-james"})
		archive.Events["event-wedding"] = wedding("person-henry")
		issues := suggestCensusNeighbors(archive)
		require.Len(t, issues, 1)
		assert.Contains(t, issues[0].Message, "James Clark")
		assert.NotContains(t, issues[0].Message, "Henry Jeffries")
	})
}

func TestSuggestCensusNeighbors_SkipsWhatItCannotAttribute(t *testing.T) {
	neighbors := []glxlib.EventNeighbor{{Name: "Henry Jeffries"}}

	t.Run("non-census event", func(t *testing.T) {
		archive := neighborArchive()
		archive.Events["event-other"] = &glxlib.Event{
			Type:         glxlib.EventTypeBirth,
			Participants: []glxlib.Participant{{Person: "person-abram", Role: glxlib.ParticipantRolePrincipal}},
			Neighbors:    neighbors,
		}
		assert.Empty(t, suggestCensusNeighbors(archive))
	})

	t.Run("neighbor with neither name nor person", func(t *testing.T) {
		archive := neighborArchive(glxlib.EventNeighbor{Position: "next_household"})
		assert.Empty(t, suggestCensusNeighbors(archive))
	})

	t.Run("census with no household", func(t *testing.T) {
		archive := neighborArchive()
		archive.Events["event-census-1860"] = &glxlib.Event{Type: glxlib.EventTypeCensus, Date: "1860", Neighbors: neighbors}
		assert.Empty(t, suggestCensusNeighbors(archive))
	})
}

func TestAnalyzeSuggestions_IncludesCensusNeighbors(t *testing.T) {
	archive := neighborArchive(glxlib.EventNeighbor{Name: "Henry Jeffries", Position: "previous_household"})

	issues := analyzeSuggestions(archive)

	assert.NotNil(t, findIssueByMessage(issues, "person-abram", "Cross-reference page neighbors"))
	filtered := filterByPerson(issues, "person-abram", archive)
	assert.NotNil(t, findIssueByMessage(filtered, "person-abram", "Henry Jeffries"))
}
