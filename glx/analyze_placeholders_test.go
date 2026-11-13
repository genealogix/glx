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

func placeholderPerson(name, status string) *glxlib.Person {
	return &glxlib.Person{Properties: map[string]any{
		glxlib.PersonPropertyName:   name,
		glxlib.PersonPropertyStatus: status,
	}}
}

func parentChild(parent, child string) *glxlib.Relationship {
	return &glxlib.Relationship{
		Type: glxlib.RelationshipTypeParentChild,
		Participants: []glxlib.Participant{
			{Person: parent, Role: glxlib.ParticipantRoleParent},
			{Person: child, Role: glxlib.ParticipantRoleChild},
		},
	}
}

func issueMessagesFor(issues []AnalysisIssue, person string) []string {
	var messages []string
	for _, issue := range issues {
		if issue.Person == person {
			messages = append(messages, issue.Message)
		}
	}

	return messages
}

func TestAnalyzeGaps_SkipsUnresolvedPlaceholder(t *testing.T) {
	for _, status := range []string{glxlib.PersonStatusHypothetical, glxlib.PersonStatusPredicted} {
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"person-unknown": placeholderPerson("Unknown #8", status)}}
		assert.Empty(t, analyzeGaps(archive), status)
	}
}

func TestAnalyzeGaps_ResolvedPersonIsOrdinary(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"person-a": placeholderPerson("William Green", glxlib.PersonStatusResolved)}}
	assert.NotEmpty(t, issueMessagesFor(analyzeGaps(archive), "person-a"))
}

func TestAnalyzeGaps_PlaceholderParentsDoNotCloseTheGap(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary":    {Properties: map[string]any{glxlib.PersonPropertyName: "Mary Green"}},
			"person-unknown": placeholderPerson("Unknown #8", glxlib.PersonStatusPredicted),
		},
		Relationships: map[string]*glxlib.Relationship{"rel-1": parentChild("person-unknown", "person-mary")},
	}

	assert.NotNil(t, findIssueByMessage(analyzeGaps(archive), "person-mary", "no identified parents"))
	assert.False(t, buildChildHasParentsIndex(archive)["person-mary"], "still a brick wall for suggestions")

	archive.Persons["person-unknown"] = placeholderPerson("William Green", glxlib.PersonStatusResolved)
	issues := analyzeGaps(archive)
	assert.Nil(t, findIssueByMessage(issues, "person-mary", "parents"), "a resolved parent closes the gap")
}

func TestAnalyzeGaps_NoMarriageGapForPlaceholderSpouse(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-mary":    {Properties: map[string]any{glxlib.PersonPropertyName: "Mary Green"}},
			"person-unknown": placeholderPerson("Unknown #8", glxlib.PersonStatusPredicted),
		},
		Relationships: map[string]*glxlib.Relationship{"rel-1": {
			Type: glxlib.RelationshipTypeMarriage,
			Participants: []glxlib.Participant{
				{Person: "person-mary", Role: glxlib.ParticipantRoleSpouse},
				{Person: "person-unknown", Role: glxlib.ParticipantRoleSpouse},
			},
		}},
	}

	assert.Nil(t, findIssueByMessage(analyzeGaps(archive), "person-mary", "no marriage event"))
}

func TestReportResearchPlaceholders(t *testing.T) {
	assert.Empty(t, reportResearchPlaceholders(&glxlib.GLXFile{Persons: map[string]*glxlib.Person{"person-a": {}}}))

	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{
		"person-a": {},
		"person-b": placeholderPerson("Unknown #8", glxlib.PersonStatusPredicted),
		"person-c": placeholderPerson("Unknown #9", glxlib.PersonStatusHypothetical),
		"person-d": placeholderPerson("William Green", glxlib.PersonStatusResolved),
	}}
	issues := reportResearchPlaceholders(archive)

	require.Len(t, issues, 3)
	assert.Equal(t, "person-b", issues[0].Person)
	assert.Contains(t, issues[0].Message, "Unknown #8 — predicted placeholder, not yet identified")
	assert.Equal(t, "person-c", issues[1].Person)
	assert.Contains(t, issues[2].Message, "Research placeholders: 2 open, 1 resolved")
	assert.Empty(t, issues[2].Person)
}

func TestAnalyzeSuggestions_NoRecordSearchesForPlaceholder(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-unknown": placeholderPerson("Unknown #8", glxlib.PersonStatusPredicted)},
		Events: map[string]*glxlib.Event{"event-birth": {
			Type: glxlib.EventTypeBirth, Date: "ABT 1820",
			Participants: []glxlib.Participant{{Person: "person-unknown", Role: glxlib.ParticipantRolePrincipal}},
		}},
	}

	messages := issueMessagesFor(analyzeSuggestions(archive), "person-unknown")
	assert.Equal(t, []string{"Unknown #8 — predicted placeholder, not yet identified"}, messages)
}
