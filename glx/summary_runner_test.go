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
	"bytes"
	"io"
	"os"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": []any{
						map[string]any{
							"value":  "John Smith",
							"fields": map[string]any{"type": "birth", "given": "John", "surname": "Smith"},
						},
						map[string]any{
							"value":  "Johnny Smith",
							"fields": map[string]any{"type": "nickname"},
						},
					},
					"gender":  "male",
					"born_on": "1850-03-15",
					"born_at": "place-ny",
					"died_on": "1920-06-01",
					"died_at": "place-boston",
				},
			},
			"person-jane": {
				Properties: map[string]any{
					"name": []any{
						map[string]any{
							"value":  "Jane Doe",
							"fields": map[string]any{"type": "birth"},
						},
						map[string]any{
							"value":  "Jane Smith",
							"fields": map[string]any{"type": "married"},
						},
					},
					"gender": "female",
				},
			},
			"person-child": {
				Properties: map[string]any{
					"name":   "Alice Smith",
					"gender": "female",
				},
			},
			"person-child2": {
				Properties: map[string]any{
					"name":   "Bob Smith",
					"gender": "male",
				},
			},
			"person-neighbor": {
				Properties: map[string]any{
					"name": "Sam Wilson",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-john": {
				Type:         "birth",
				Date:         "1850-03-15",
				PlaceID:      "place-ny",
				Participants: []glxlib.Participant{{Person: "person-john"}},
			},
			"event-marriage": {
				Type:    "marriage",
				Date:    "1875",
				PlaceID: "place-boston",
				Participants: []glxlib.Participant{
					{Person: "person-john"},
					{Person: "person-jane"},
				},
			},
			"event-census": {
				Type:    "census",
				Date:    "1880",
				PlaceID: "place-boston",
				Participants: []glxlib.Participant{
					{Person: "person-john"},
					{Person: "person-jane"},
				},
			},
			"event-immigration": {
				Type:         "immigration",
				Date:         "1848",
				PlaceID:      "place-ny",
				Participants: []glxlib.Participant{{Person: "person-john"}},
			},
			"event-death-john": {
				Type:         "death",
				Date:         "1920-06-01",
				PlaceID:      "place-boston",
				Participants: []glxlib.Participant{{Person: "person-john"}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type: "marriage",
				Participants: []glxlib.Participant{
					{Person: "person-john"},
					{Person: "person-jane"},
				},
				StartEvent: "event-marriage",
			},
			"rel-parent-child-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "parent"},
					{Person: "person-jane", Role: "parent"},
					{Person: "person-child", Role: "child"},
				},
			},
			"rel-parent-child-2": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "parent"},
					{Person: "person-jane", Role: "parent"},
					{Person: "person-child2", Role: "child"},
				},
			},
			"rel-neighbor": {
				Type: "neighbor",
				Participants: []glxlib.Participant{
					{Person: "person-john"},
					{Person: "person-neighbor"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-ny":     {Name: "New York, New York"},
			"place-boston":  {Name: "Boston, Massachusetts"},
		},
	}
}

// ============================================================================
// findPersonByQuery tests
// ============================================================================

func TestFindPersonByQuery_ExactID(t *testing.T) {
	archive := newTestArchive()

	id, person, err := findPersonByQuery(archive, "person-john")
	require.NoError(t, err)
	assert.Equal(t, "person-john", id)
	assert.NotNil(t, person)
}

func TestFindPersonByQuery_NameSearch(t *testing.T) {
	archive := newTestArchive()

	id, _, err := findPersonByQuery(archive, "Alice")
	require.NoError(t, err)
	assert.Equal(t, "person-child", id)
}

func TestFindPersonByQuery_AlternateNameSearch(t *testing.T) {
	archive := newTestArchive()

	// "Johnny Smith" is a nickname variant of person-john
	id, _, err := findPersonByQuery(archive, "Johnny")
	require.NoError(t, err)
	assert.Equal(t, "person-john", id)
}

func TestFindPersonByQuery_NotFound(t *testing.T) {
	archive := newTestArchive()

	_, _, err := findPersonByQuery(archive, "Nonexistent Person")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}

func TestFindPersonByQuery_Ambiguous(t *testing.T) {
	archive := newTestArchive()

	// "Smith" matches multiple persons
	_, _, err := findPersonByQuery(archive, "Smith")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple persons match")
}

// ============================================================================
// extractAllNameVariants tests
// ============================================================================

func TestExtractAllNameVariants_TemporalList(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			"name": []any{
				map[string]any{"value": "Jane Miller", "fields": map[string]any{"type": "birth"}},
				map[string]any{"value": "Jane Webb", "fields": map[string]any{"type": "married"}},
			},
		},
	}

	variants := extractAllNameVariants(person)
	require.Len(t, variants, 2)
	assert.Equal(t, "Jane Miller", variants[0].Value)
	assert.Equal(t, "birth", variants[0].NameType)
	assert.Equal(t, "Jane Webb", variants[1].Value)
	assert.Equal(t, "married", variants[1].NameType)
}

func TestExtractAllNameVariants_SimpleString(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{"name": "John Smith"},
	}

	variants := extractAllNameVariants(person)
	require.Len(t, variants, 1)
	assert.Equal(t, "John Smith", variants[0].Value)
	assert.Equal(t, "", variants[0].NameType)
}

func TestExtractAllNameVariants_StructuredMap(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			"name": map[string]any{
				"value":  "Jane Doe",
				"fields": map[string]any{"type": "birth"},
			},
		},
	}

	variants := extractAllNameVariants(person)
	require.Len(t, variants, 1)
	assert.Equal(t, "Jane Doe", variants[0].Value)
	assert.Equal(t, "birth", variants[0].NameType)
}

func TestExtractAllNameVariants_NoName(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{"gender": "male"},
	}

	variants := extractAllNameVariants(person)
	assert.Nil(t, variants)
}

// ============================================================================
// Relationship finders tests
// ============================================================================

func TestFindSpouses(t *testing.T) {
	archive := newTestArchive()

	spouses := findSpouses("person-john", archive)
	require.Len(t, spouses, 1)
	assert.Equal(t, "person-jane", spouses[0].PersonID)
	assert.Equal(t, "Jane Doe", spouses[0].PersonName)
	assert.Equal(t, "1875", spouses[0].MarriageDate)
	assert.Equal(t, "Boston, Massachusetts", spouses[0].MarriagePlace)
}

func TestFindSpouses_None(t *testing.T) {
	archive := newTestArchive()

	spouses := findSpouses("person-neighbor", archive)
	assert.Empty(t, spouses)
}

func TestFindParentIDs(t *testing.T) {
	archive := newTestArchive()

	parents := findParentIDs("person-child", archive)
	require.Len(t, parents, 2)
	assert.Contains(t, parents, "person-john")
	assert.Contains(t, parents, "person-jane")
}

func TestFindParentIDs_None(t *testing.T) {
	archive := newTestArchive()

	parents := findParentIDs("person-john", archive)
	assert.Empty(t, parents)
}

func TestFindSiblingIDs(t *testing.T) {
	archive := newTestArchive()

	parentIDs := []string{"person-john", "person-jane"}
	siblings := findSiblingIDs("person-child", parentIDs, archive)
	require.Len(t, siblings, 1)
	assert.Equal(t, "person-child2", siblings[0])
}

func TestFindSiblingIDs_NoParents(t *testing.T) {
	siblings := findSiblingIDs("person-john", nil, newTestArchive())
	assert.Empty(t, siblings)
}

func TestFindOtherRelationships(t *testing.T) {
	archive := newTestArchive()

	rels := findOtherRelationships("person-john", archive)
	require.Len(t, rels, 1)
	assert.Equal(t, "neighbor", rels[0].RelType)
	assert.Equal(t, "Sam Wilson", rels[0].OtherName)
}

// ============================================================================
// Event helpers tests
// ============================================================================

func TestFindEventForPerson(t *testing.T) {
	archive := newTestArchive()

	result := findEventForPerson("person-john", "birth", archive)
	assert.Equal(t, "1850-03-15, New York, New York", result)
}

func TestFindEventForPerson_NotFound(t *testing.T) {
	archive := newTestArchive()

	result := findEventForPerson("person-john", "christening", archive)
	assert.Equal(t, "", result)
}

func TestFindMarriageEvent(t *testing.T) {
	archive := newTestArchive()

	date, place := findMarriageEvent("person-john", "person-jane", archive)
	assert.Equal(t, "1875", date)
	assert.Equal(t, "Boston, Massachusetts", place)
}

func TestResolvePlaceName(t *testing.T) {
	archive := newTestArchive()

	assert.Equal(t, "New York, New York", resolvePlaceName("place-ny", archive))
	assert.Equal(t, "place-unknown", resolvePlaceName("place-unknown", archive))
	assert.Equal(t, "", resolvePlaceName("", archive))
}

// ============================================================================
// Narrative helpers tests
// ============================================================================

func TestNarrativeDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1850", "in 1850"},
		{"ABT 1850", "about 1850"},
		{"BEF 1920", "before 1920"},
		{"AFT 1880", "after 1880"},
		{"BET 1880 AND 1890", "between 1880 and 1890"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, narrativeDate(tt.input))
		})
	}
}

func TestPronounFor(t *testing.T) {
	male := &glxlib.Person{Properties: map[string]any{"gender": "male"}}
	female := &glxlib.Person{Properties: map[string]any{"gender": "female"}}
	unknown := &glxlib.Person{Properties: map[string]any{}}

	subj, poss := pronounFor(male)
	assert.Equal(t, "He", subj)
	assert.Equal(t, "his", poss)

	subj, poss = pronounFor(female)
	assert.Equal(t, "She", subj)
	assert.Equal(t, "her", poss)

	subj, poss = pronounFor(unknown)
	assert.Equal(t, "They", subj)
	assert.Equal(t, "their", poss)
}

func TestJoinNames(t *testing.T) {
	assert.Equal(t, "", joinNames(nil))
	assert.Equal(t, "Alice", joinNames([]string{"Alice"}))
	assert.Equal(t, "Alice and Bob", joinNames([]string{"Alice", "Bob"}))
	assert.Equal(t, "Alice, Bob, and Carol", joinNames([]string{"Alice", "Bob", "Carol"}))
}

func TestGenerateLifeHistory(t *testing.T) {
	archive := newTestArchive()

	history := generateLifeHistory("person-john", archive.Persons["person-john"], archive)
	assert.Contains(t, history, "John Smith was born")
	assert.Contains(t, history, "New York")
	assert.Contains(t, history, "married Jane Doe")
	assert.Contains(t, history, "died")
}

// ============================================================================
// Display helpers tests
// ============================================================================

func TestFormatNameType(t *testing.T) {
	assert.Equal(t, "Birth Name", formatNameType("birth"))
	assert.Equal(t, "Married Name", formatNameType("married"))
	assert.Equal(t, "Maiden Name", formatNameType("maiden"))
	assert.Equal(t, "As Recorded", formatNameType("as_recorded"))
	assert.Equal(t, "Also Known As", formatNameType("aka"))
	assert.Equal(t, "Nickname", formatNameType("nickname"))
}

func TestSnakeCaseToTitle(t *testing.T) {
	assert.Equal(t, "Bar Mitzvah", snakeCaseToTitle("bar_mitzvah"))
	assert.Equal(t, "Military Service", snakeCaseToTitle("military_service"))
	assert.Equal(t, "Census", snakeCaseToTitle("census"))
	assert.Equal(t, "", snakeCaseToTitle(""))
}

func TestDisplayOrDash(t *testing.T) {
	assert.Equal(t, "—", displayOrDash(""))
	assert.Equal(t, "hello", displayOrDash("hello"))
}

func TestSectionHeader(t *testing.T) {
	header := sectionHeader("Test")
	assert.Contains(t, header, "── Test ")
	assert.Contains(t, header, "───")
}

// ============================================================================
// Integration test
// ============================================================================

func TestShowSummary_Integration(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := showSummary("../docs/examples/complete-family", "John")

	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	require.NoError(t, err)
	assert.Contains(t, output, "===")
	assert.Contains(t, output, "Name:")
	assert.Contains(t, output, "Vital Events")
}

func TestShowSummary_NotFound(t *testing.T) {
	err := showSummary("../docs/examples/complete-family", "ZZZ_NONEXISTENT_ZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no person found")
}
