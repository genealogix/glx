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

// --- Levenshtein tests ---

func TestLevenshteinDistance_IdenticalStrings(t *testing.T) {
	assert.Equal(t, 0, levenshteinDistance("hello", "hello"))
}

func TestLevenshteinDistance_EmptyStrings(t *testing.T) {
	assert.Equal(t, 5, levenshteinDistance("hello", ""))
	assert.Equal(t, 5, levenshteinDistance("", "hello"))
	assert.Equal(t, 0, levenshteinDistance("", ""))
}

func TestLevenshteinDistance_KnownValues(t *testing.T) {
	assert.Equal(t, 3, levenshteinDistance("kitten", "sitting"))
	assert.Equal(t, 1, levenshteinDistance("John", "Joan"))
}

func TestNormalizedLevenshtein_Range(t *testing.T) {
	score := normalizedLevenshtein("John", "Joan")
	assert.True(t, score >= 0.0 && score <= 1.0)
	assert.Equal(t, 1.0, normalizedLevenshtein("same", "same"))
	assert.Equal(t, 1.0, normalizedLevenshtein("", ""))
}

// --- Name scoring tests ---

func TestScoreNameSimilarity_ExactMatch(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, _ := scoreNameSimilarity(a, b)
	assert.Equal(t, 1.0, score)
}

func TestScoreNameSimilarity_SurnameMatchGivenDifferent(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "James Smith"}}
	score, _ := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.5, "same surname should give partial credit")
	assert.True(t, score < 1.0, "different given name should not be perfect")
}

func TestScoreNameSimilarity_NicknameVariant(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "William Smith"}}
	b := &Person{Properties: map[string]any{"name": "Bill Smith"}}
	score, detail := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.8, "nickname variant should score high, got %f", score)
	assert.Contains(t, detail, "surname exact")
}

func TestScoreNameSimilarity_InitialMatch(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "J. Smith"}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, _ := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.7, "initial match should give partial credit, got %f", score)
}

func TestScoreNameSimilarity_CompletelyDifferent(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "Mary Johnson"}}
	score, _ := scoreNameSimilarity(a, b)
	assert.True(t, score < 0.5, "completely different names should score low, got %f", score)
}

func TestScoreNameSimilarity_NoName(t *testing.T) {
	a := &Person{Properties: map[string]any{}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, detail := scoreNameSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no name", detail)
}

func TestScoreNameSimilarity_StructuredFields(t *testing.T) {
	a := &Person{Properties: map[string]any{
		"name": map[string]any{
			"value":  "Robert Webb",
			"fields": map[string]any{"given": "Robert", "surname": "Webb"},
		},
	}}
	b := &Person{Properties: map[string]any{
		"name": map[string]any{
			"value":  "Rob Webb",
			"fields": map[string]any{"given": "Rob", "surname": "Webb"},
		},
	}}
	score, detail := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.8, "Rob/Robert Webb should match well, got %f", score)
	assert.Contains(t, detail, "surname exact")
}

// --- Year scoring tests ---

func TestScoreYearSimilarity_ExactMatch(t *testing.T) {
	propsA := map[string]any{"born_on": "1850"}
	propsB := map[string]any{"born_on": "1850"}
	score, detail := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 1.0, score)
	assert.Equal(t, "exact match", detail)
}

func TestScoreYearSimilarity_WithinOneYear(t *testing.T) {
	propsA := map[string]any{"born_on": "1850"}
	propsB := map[string]any{"born_on": "1851"}
	score, _ := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 0.75, score)
}

func TestScoreYearSimilarity_WithinTwoYears(t *testing.T) {
	propsA := map[string]any{"born_on": "1850"}
	propsB := map[string]any{"born_on": "1852"}
	score, _ := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 0.5, score)
}

func TestScoreYearSimilarity_FarApart(t *testing.T) {
	propsA := map[string]any{"born_on": "1850"}
	propsB := map[string]any{"born_on": "1870"}
	score, _ := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 0.0, score)
}

func TestScoreYearSimilarity_MissingYear(t *testing.T) {
	propsA := map[string]any{"born_on": "1850"}
	propsB := map[string]any{}
	score, detail := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
}

func TestScoreYearSimilarity_DateQualifiers(t *testing.T) {
	propsA := map[string]any{"born_on": "ABT 1850"}
	propsB := map[string]any{"born_on": "1850"}
	score, _ := scoreYearSimilarity(propsA, propsB, "born_on")
	assert.Equal(t, 1.0, score, "ABT qualifier should still match year")
}

// --- Place scoring tests ---

func TestScorePlaceSimilarity_SamePlaceID(t *testing.T) {
	propsA := map[string]any{"born_at": "place-madison-wi"}
	propsB := map[string]any{"born_at": "place-madison-wi"}
	score, _ := scorePlaceSimilarity(propsA, propsB, "born_at", nil)
	assert.Equal(t, 1.0, score)
}

func TestScorePlaceSimilarity_DifferentPlaceID(t *testing.T) {
	propsA := map[string]any{"born_at": "place-madison-wi"}
	propsB := map[string]any{"born_at": "place-milwaukee-wi"}
	score, _ := scorePlaceSimilarity(propsA, propsB, "born_at", nil)
	assert.Equal(t, 0.0, score)
}

func TestScorePlaceSimilarity_MissingPlace(t *testing.T) {
	propsA := map[string]any{"born_at": "place-madison-wi"}
	propsB := map[string]any{}
	score, _ := scorePlaceSimilarity(propsA, propsB, "born_at", nil)
	assert.Equal(t, 0.0, score)
}

// --- Relationship/event scoring tests ---

func TestScoreSharedRelationships_CommonPeer(t *testing.T) {
	idx := &duplicateIndex{
		personRelPeers: map[string]map[string]bool{
			"person-a": {"person-x": true, "person-y": true},
			"person-b": {"person-x": true, "person-z": true},
		},
	}
	score, detail := scoreSharedRelationships("person-a", "person-b", idx)
	assert.True(t, score > 0, "should have positive score for shared peer")
	assert.Contains(t, detail, "1 shared")
}

func TestScoreSharedRelationships_NoOverlap(t *testing.T) {
	idx := &duplicateIndex{
		personRelPeers: map[string]map[string]bool{
			"person-a": {"person-x": true},
			"person-b": {"person-y": true},
		},
	}
	score, _ := scoreSharedRelationships("person-a", "person-b", idx)
	assert.Equal(t, 0.0, score)
}

func TestScoreSharedEvents_CommonEvent(t *testing.T) {
	idx := &duplicateIndex{
		personEvents: map[string][]string{
			"person-a": {"event-census-1860", "event-birth-a"},
			"person-b": {"event-census-1860", "event-birth-b"},
		},
	}
	score, detail := scoreSharedEvents("person-a", "person-b", idx)
	assert.True(t, score > 0, "should have positive score for shared event")
	assert.Contains(t, detail, "1 shared")
}

// --- Integration tests ---

func TestFindDuplicates_EmptyArchive(t *testing.T) {
	archive := &GLXFile{}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs)
}

func TestFindDuplicates_SinglePerson(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs)
}

func TestFindDuplicates_ObviousDuplicate(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-r-webb": {Properties: map[string]any{
				"name": "R Webb", "born_on": "1815", "born_at": "place-va",
			}},
			"person-robert-webb": {Properties: map[string]any{
				"name": "Robert Webb", "born_on": "1815", "born_at": "place-va",
			}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.5})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	assert.True(t, result.Pairs[0].Score >= 0.5)
}

func TestFindDuplicates_RelatedPersonsSkipped(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
			"person-john-jr": {Properties: map[string]any{"name": "John Smith", "born_on": "1875"}},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-john", Role: "parent"},
					{Person: "person-john-jr", Role: "child"},
				},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs, "parent-child pairs should be skipped")
}

func TestFindDuplicates_ThresholdFiltering(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
			"person-jane": {Properties: map[string]any{"name": "Jane Doe"}},
		},
	}
	// With high threshold, no matches expected
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.99})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs)
}

func TestFindDuplicates_PersonFilter(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
			"person-b": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
			"person-c": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.5, PersonFilter: "person-a"})
	require.NoError(t, err)
	// All pairs should include person-a
	for _, pair := range result.Pairs {
		assert.True(t, pair.PersonA == "person-a" || pair.PersonB == "person-a",
			"filtered pairs should include person-a")
	}
}

func TestFindDuplicates_SortedByScoreDescending(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-john-1": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
			"person-john-2": {Properties: map[string]any{"name": "John Smith", "born_on": "1850"}},
			"person-john-3": {Properties: map[string]any{"name": "Jon Smyth", "born_on": "1855"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.3})
	require.NoError(t, err)
	if len(result.Pairs) > 1 {
		for i := 1; i < len(result.Pairs); i++ {
			assert.True(t, result.Pairs[i-1].Score >= result.Pairs[i].Score,
				"pairs should be sorted by score descending")
		}
	}
}

// --- Nickname/initial tests ---

func TestAreNicknameVariants(t *testing.T) {
	assert.True(t, areNicknameVariants("william", "bill"))
	assert.True(t, areNicknameVariants("james", "jas"))
	assert.True(t, areNicknameVariants("elizabeth", "betsy"))
	assert.False(t, areNicknameVariants("john", "mary"))
	assert.False(t, areNicknameVariants("unknown", "other"))
}

func TestIsInitialMatch(t *testing.T) {
	assert.True(t, isInitialMatch("j", "john"))
	assert.True(t, isInitialMatch("j.", "james"))
	assert.True(t, isInitialMatch("daniel", "d"))
	assert.False(t, isInitialMatch("john", "james"))
	assert.False(t, isInitialMatch("j", "j"))
}

func TestSplitFullName(t *testing.T) {
	given, surname := splitFullName("John Smith")
	assert.Equal(t, "John", given)
	assert.Equal(t, "Smith", surname)

	given, surname = splitFullName("Mary Jane Johnson")
	assert.Equal(t, "Mary Jane", given)
	assert.Equal(t, "Johnson", surname)

	given, surname = splitFullName("Madonna")
	assert.Equal(t, "Madonna", given)
	assert.Equal(t, "", surname)
}

// --- Threshold validation tests ---

func TestFindDuplicates_ThresholdTooHigh(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "John Smith"}},
			"person-b": {Properties: map[string]any{"name": "Jane Doe"}},
		},
	}
	_, err := FindDuplicates(archive, DuplicateOptions{Threshold: 1.5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "threshold must be between 0.0 and 1.0")
}

func TestFindDuplicates_ThresholdNegative(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "John Smith"}},
			"person-b": {Properties: map[string]any{"name": "Jane Doe"}},
		},
	}
	_, err := FindDuplicates(archive, DuplicateOptions{Threshold: -0.1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "threshold must be between 0.0 and 1.0")
}

func TestFindDuplicates_ThresholdBoundary(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "John Smith"}},
			"person-b": {Properties: map[string]any{"name": "Jane Doe"}},
		},
	}
	// 0.0 and 1.0 are valid
	_, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	assert.NoError(t, err)
	_, err = FindDuplicates(archive, DuplicateOptions{Threshold: 1.0})
	assert.NoError(t, err)
}
