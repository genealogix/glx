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

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildConfidenceReport_Empty(t *testing.T) {
	archive := &glxlib.GLXFile{}
	report := buildConfidenceReport(archive)

	assert.Equal(t, 0, report.TotalAssertions)
	assert.Empty(t, report.ByConfidence)
	assert.Empty(t, report.NoConfidence)
}

func TestBuildConfidenceReport_ConfidenceBreakdown(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {},
			"person-b": {},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {
				Subject:    glxlib.EntityRef{Person: "person-a"},
				Property:   "born_on",
				Value:      "1850",
				Confidence: "high",
				Citations:  []string{"c1"},
			},
			"a2": {
				Subject:    glxlib.EntityRef{Person: "person-a"},
				Property:   "name",
				Value:      "John Smith",
				Confidence: "high",
				Citations:  []string{"c1"},
			},
			"a3": {
				Subject:    glxlib.EntityRef{Person: "person-b"},
				Property:   "name",
				Value:      "Jane Webb",
				Confidence: "low",
				Citations:  []string{"c2"},
			},
			"a4": {
				Subject:    glxlib.EntityRef{Person: "person-b"},
				Property:   "born_on",
				Value:      "1860",
				Confidence: "disputed",
				Sources:    []string{"s1"},
			},
		},
	}

	report := buildConfidenceReport(archive)

	assert.Equal(t, 4, report.TotalAssertions)
	assert.Equal(t, 2, report.ByConfidence["high"])
	assert.Equal(t, 1, report.ByConfidence["low"])
	assert.Equal(t, 1, report.ByConfidence["disputed"])
	assert.Empty(t, report.NoConfidence)
}

func TestBuildConfidenceReport_NoConfidence(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a1": {
				Subject:  glxlib.EntityRef{Person: "person-a"},
				Property: "name",
				Value:    "Test",
				Sources:  []string{"s1"},
				// No confidence set
			},
		},
	}

	report := buildConfidenceReport(archive)

	assert.Equal(t, 1, report.TotalAssertions)
	assert.Equal(t, 1, report.ByConfidence["(unset)"])
	require.Len(t, report.NoConfidence, 1)
	assert.Equal(t, "a1", report.NoConfidence[0].ID)
}

func TestBuildConfidenceReport_NoCitations(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-with-citation": {
				Subject:    glxlib.EntityRef{Person: "person-a"},
				Property:   "name",
				Value:      "John",
				Confidence: "high",
				Citations:  []string{"c1"},
			},
			"a-source-only": {
				Subject:    glxlib.EntityRef{Person: "person-a"},
				Property:   "born_on",
				Value:      "1850",
				Confidence: "medium",
				Sources:    []string{"s1"},
				// No citations
			},
			"a-media-only": {
				Subject:    glxlib.EntityRef{Person: "person-a"},
				Property:   "gender",
				Value:      "male",
				Confidence: "low",
				Media:      []string{"m1"},
				// No citations
			},
		},
	}

	report := buildConfidenceReport(archive)

	require.Len(t, report.NoCitations, 2)
	ids := []string{report.NoCitations[0].ID, report.NoCitations[1].ID}
	assert.Contains(t, ids, "a-source-only")
	assert.Contains(t, ids, "a-media-only")
}

func TestBuildConfidenceReport_UnbackedEntities(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-backed":   {},
			"person-unbacked": {},
		},
		Events: map[string]*glxlib.Event{
			"event-backed":   {},
			"event-unbacked": {},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-unbacked": {},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {
				Subject:    glxlib.EntityRef{Person: "person-backed"},
				Property:   "name",
				Value:      "Test",
				Confidence: "high",
				Citations:  []string{"c1"},
			},
			"a2": {
				Subject:    glxlib.EntityRef{Event: "event-backed"},
				Property:   "date",
				Value:      "1850",
				Confidence: "high",
				Citations:  []string{"c1"},
			},
		},
	}

	report := buildConfidenceReport(archive)

	assert.Equal(t, []string{"person-unbacked"}, report.UnbackedPersons)
	assert.Equal(t, []string{"event-unbacked"}, report.UnbackedEvents)
	assert.Equal(t, []string{"rel-unbacked"}, report.UnbackedRelations)
}

func TestBuildConfidenceReport_ParticipantAssertion(t *testing.T) {
	archive := &glxlib.GLXFile{
		Assertions: map[string]*glxlib.Assertion{
			"a-participant": {
				Subject:    glxlib.EntityRef{Event: "event-1"},
				Participant: &glxlib.Participant{
					Person: "person-a",
					Role:   "witness",
				},
				Confidence: "medium",
				Citations:  []string{"c1"},
			},
		},
	}

	report := buildConfidenceReport(archive)

	assert.Equal(t, 1, report.TotalAssertions)
	assert.Equal(t, 1, report.ByConfidence["medium"])
}

func TestBuildConfidenceOrder(t *testing.T) {
	counts := map[string]int{
		"(unset)":  2,
		"high":     5,
		"low":      1,
		"custom":   3,
		"disputed": 1,
	}

	order := buildConfidenceOrder(counts)

	// Standard order is high, medium, low, disputed; medium is absent from counts,
	// so we expect high, low, disputed first, then custom, then (unset).
	assert.Equal(t, []string{"high", "low", "disputed", "custom", "(unset)"}, order)
}

func TestConfidenceReport_CompleteFamily(t *testing.T) {
	// Integration test against the complete-family example
	err := confidenceReport("../docs/examples/complete-family")
	require.NoError(t, err)
}

func TestConfidenceReport_NonExistentPath(t *testing.T) {
	err := confidenceReport("/nonexistent/path")
	require.Error(t, err)
}

func TestConfidenceReport_SingleFile(t *testing.T) {
	err := confidenceReport("../docs/examples/single-file/archive.glx")
	require.NoError(t, err)
}
