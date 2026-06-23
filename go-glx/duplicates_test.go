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
	"math"
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
	score, _, hasData := scoreNameSimilarity(a, b)
	assert.Equal(t, 1.0, score)
	assert.True(t, hasData)
}

func TestScoreNameSimilarity_SurnameMatchGivenDifferent(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "James Smith"}}
	score, _, hasData := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.5, "same surname should give partial credit")
	assert.True(t, score < 1.0, "different given name should not be perfect")
	assert.True(t, hasData)
}

func TestScoreNameSimilarity_NicknameVariant(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "William Smith"}}
	b := &Person{Properties: map[string]any{"name": "Bill Smith"}}
	score, detail, hasData := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.8, "nickname variant should score high, got %f", score)
	assert.Contains(t, detail, "surname exact")
	assert.True(t, hasData)
}

func TestScoreNameSimilarity_InitialMatch(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "J. Smith"}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, _, hasData := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.7, "initial match should give partial credit, got %f", score)
	assert.True(t, hasData)
}

func TestScoreNameSimilarity_CompletelyDifferent(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "Mary Johnson"}}
	score, _, hasData := scoreNameSimilarity(a, b)
	assert.True(t, score < 0.5, "completely different names should score low, got %f", score)
	assert.True(t, hasData, "completely-different names were still compared, so the dimension carries data")
}

func TestScoreNameSimilarity_NoName(t *testing.T) {
	a := &Person{Properties: map[string]any{}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, detail, hasData := scoreNameSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no name", detail)
	assert.False(t, hasData)
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
	score, detail, hasData := scoreNameSimilarity(a, b)
	assert.True(t, score > 0.8, "Rob/Robert Webb should match well, got %f", score)
	assert.Contains(t, detail, "surname exact")
	assert.True(t, hasData)
}

// --- Scoring boundary tests ---

func TestScoreNameSimilarity_BothEmpty(t *testing.T) {
	a := &Person{Properties: map[string]any{}}
	b := &Person{Properties: map[string]any{}}
	score, detail, hasData := scoreNameSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no name", detail)
	assert.False(t, hasData)
}

func TestScoreNameSimilarity_NilProperties(t *testing.T) {
	a := &Person{}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, detail, hasData := scoreNameSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no name", detail)
	assert.False(t, hasData)
}

func TestCompareSurnames_BothEmpty(t *testing.T) {
	assert.Equal(t, 0.0, compareSurnames("", ""))
}

func TestCompareSurnames_OneEmpty(t *testing.T) {
	assert.Equal(t, 0.0, compareSurnames("Smith", ""))
	assert.Equal(t, 0.0, compareSurnames("", "Smith"))
}

func TestCompareGivenNames_BothEmpty(t *testing.T) {
	assert.Equal(t, 0.0, compareGivenNames("", ""))
}

func TestCompareGivenNames_OneEmpty(t *testing.T) {
	assert.Equal(t, 0.0, compareGivenNames("John", ""))
	assert.Equal(t, 0.0, compareGivenNames("", "John"))
}

// --- Phonetic similarity tests (#704) ---

func TestScorePhoneticSimilarity_ExactMatch(t *testing.T) {
	// Schneider/Snider — the #704 motivating example. Same given on both
	// sides so both sub-comparisons run and both match.
	a := &Person{Properties: map[string]any{"name": "Hans Schneider"}}
	b := &Person{Properties: map[string]any{"name": "Hans Snider"}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 1.0, score, 1e-12)
	assert.True(t, hasData)
	assert.Contains(t, detail, "surname phonetic")
	assert.Contains(t, detail, "given phonetic")
}

func TestScorePhoneticSimilarity_GermanizedSurname(t *testing.T) {
	// Mueller/Miller Soundex both M460, same given Anna.
	a := &Person{Properties: map[string]any{"name": "Anna Mueller"}}
	b := &Person{Properties: map[string]any{"name": "Anna Miller"}}
	score, _, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 1.0, score, 1e-12)
	assert.True(t, hasData)
}

func TestScorePhoneticSimilarity_OnlySurnameMatches(t *testing.T) {
	// Schneider/Snider phonetic match, given names phonetically distinct.
	a := &Person{Properties: map[string]any{"name": "Hans Schneider"}}
	b := &Person{Properties: map[string]any{"name": "Karl Snider"}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 0.5, score, 1e-12)
	assert.True(t, hasData)
	assert.Equal(t, "surname phonetic", detail)
}

func TestScorePhoneticSimilarity_NoMatch(t *testing.T) {
	a := &Person{Properties: map[string]any{"name": "John Smith"}}
	b := &Person{Properties: map[string]any{"name": "Mary Johnson"}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 0.0, score, 1e-12)
	assert.True(t, hasData, "both sub-comparisons ran and disagreed; dimension carries data and must remain in the renormalization denominator")
	assert.Equal(t, "no match", detail)
}

func TestScorePhoneticSimilarity_BothEmpty(t *testing.T) {
	a := &Person{Properties: map[string]any{}}
	b := &Person{Properties: map[string]any{}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 0.0, score, 1e-12)
	assert.False(t, hasData)
	assert.Equal(t, noDataDetail, detail)
}

func TestScorePhoneticSimilarity_InitialVsFullGiven(t *testing.T) {
	// "J. Smith" vs "John Smith": Soundex on a single letter ("J000") cannot
	// meaningfully match a full name's code ("J500"). The given sub-comparison
	// is skipped (single-rune initial rule); the surname sub-comparison runs
	// and matches; signal averages over the one comparison that ran. Pins the
	// preservation of scoreNameSimilarity's existing isInitialMatch 0.6 credit.
	a := &Person{Properties: map[string]any{"name": "J. Smith"}}
	b := &Person{Properties: map[string]any{"name": "John Smith"}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 1.0, score, 1e-12)
	assert.True(t, hasData)
	assert.Equal(t, "surname phonetic", detail)
}

func TestScorePhoneticSimilarity_OneComponentOnlyPerSide(t *testing.T) {
	// Both persons have only a single-word name. splitFullName treats a
	// single-word string as the given, so given="Mueller"/"Miller" and
	// surname="" on both sides. Surname sub-comparison can't run (both
	// Soundex codes empty); given sub-comparison runs and matches. Signal
	// averages over the one sub-comparison that ran rather than penalizing
	// the absent surname — pins the "skip missing sub-comparisons" rule.
	a := &Person{Properties: map[string]any{"name": "Mueller"}}
	b := &Person{Properties: map[string]any{"name": "Miller"}}
	score, detail, hasData := scorePhoneticSimilarity(a, b)
	assert.InDelta(t, 1.0, score, 1e-12)
	assert.True(t, hasData)
	assert.Equal(t, "given phonetic", detail)
}

func TestScorePhoneticSimilarity_StructuredSurnameOnly(t *testing.T) {
	// Structured name with only the surname field populated on both sides.
	// Given sub-comparison can't run; surname sub-comparison runs and
	// matches. The mirror case of OneComponentOnlyPerSide above, exercising
	// the surname-only branch via the ExtractNameFields path.
	mk := func(surname string) *Person {
		return &Person{Properties: map[string]any{
			"name": map[string]any{
				"fields": map[string]any{
					"surname": surname,
				},
			},
		}}
	}
	score, detail, hasData := scorePhoneticSimilarity(mk("Schneider"), mk("Snider"))
	assert.InDelta(t, 1.0, score, 1e-12)
	assert.True(t, hasData)
	assert.Equal(t, "surname phonetic", detail)
}

func TestScoreEventYearSimilarity_NilEvents(t *testing.T) {
	score, detail, hasData := scoreEventYearSimilarity(nil, nil)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
	assert.False(t, hasData)
}

func TestScoreEventYearSimilarity_OneNil(t *testing.T) {
	e := &Event{Date: "1850"}
	score, detail, hasData := scoreEventYearSimilarity(e, nil)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
	assert.False(t, hasData)
}

func TestScoreEventYearSimilarity_EmptyDates(t *testing.T) {
	a := &Event{Date: ""}
	b := &Event{Date: "1850"}
	score, detail, hasData := scoreEventYearSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
	assert.False(t, hasData)
}

func TestScoreEventYearSimilarity_ExactYear(t *testing.T) {
	a := &Event{Date: "1850"}
	b := &Event{Date: "1850"}
	score, detail, hasData := scoreEventYearSimilarity(a, b)
	assert.Equal(t, 1.0, score)
	assert.Equal(t, "exact match", detail)
	assert.True(t, hasData)
}

func TestScoreEventYearSimilarity_Within1Year(t *testing.T) {
	a := &Event{Date: "1850"}
	b := &Event{Date: "1851"}
	score, detail, hasData := scoreEventYearSimilarity(a, b)
	assert.Equal(t, 0.75, score)
	assert.Equal(t, "within 1 year", detail)
	assert.True(t, hasData)
}

func TestScoreEventYearSimilarity_Within2Years(t *testing.T) {
	a := &Event{Date: "1850"}
	b := &Event{Date: "1852"}
	score, detail, hasData := scoreEventYearSimilarity(a, b)
	assert.Equal(t, 0.5, score)
	assert.Equal(t, "within 2 years", detail)
	assert.True(t, hasData)
}

func TestScoreEventYearSimilarity_FarApart(t *testing.T) {
	a := &Event{Date: "1850"}
	b := &Event{Date: "1900"}
	score, detail, hasData := scoreEventYearSimilarity(a, b)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "different", detail)
	assert.True(t, hasData, "years compared and differed by >2; the dimension carries real data and must stay in the renormalization denominator")
}

func TestScoreEventPlaceSimilarity_NilEvents(t *testing.T) {
	score, detail, hasData := scoreEventPlaceSimilarity(nil, nil, nil)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
	assert.False(t, hasData)
}

func TestScoreEventPlaceSimilarity_EmptyPlaces(t *testing.T) {
	a := &Event{PlaceID: ""}
	b := &Event{PlaceID: "place-x"}
	score, detail, hasData := scoreEventPlaceSimilarity(a, b, nil)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no data", detail)
	assert.False(t, hasData)
}

func TestScoreEventPlaceSimilarity_SamePlace(t *testing.T) {
	a := &Event{PlaceID: "place-london"}
	b := &Event{PlaceID: "place-london"}
	score, _, hasData := scoreEventPlaceSimilarity(a, b, nil)
	assert.Equal(t, 1.0, score)
	assert.True(t, hasData)
}

func TestScoreEventPlaceSimilarity_DifferentPlaces(t *testing.T) {
	a := &Event{PlaceID: "place-london"}
	b := &Event{PlaceID: "place-paris"}
	score, detail, hasData := scoreEventPlaceSimilarity(a, b, nil)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "different", detail)
	assert.True(t, hasData, "places compared and differed; the dimension carries real data and must stay in the renormalization denominator")
}

func TestScoreSharedRelationships_MissingPerson(t *testing.T) {
	idx := &duplicateIndex{
		personRelPeers: map[string]map[string]bool{
			"person-a": {"person-x": true},
		},
	}
	score, _, hasData := scoreSharedRelationships("person-a", "person-missing", idx)
	assert.Equal(t, 0.0, score)
	assert.False(t, hasData)
}

func TestScoreSharedEvents_MissingPerson(t *testing.T) {
	idx := &duplicateIndex{
		personEvents: map[string][]string{
			"person-a": {"event-1"},
		},
	}
	score, _, hasData := scoreSharedEvents("person-a", "person-missing", idx)
	assert.Equal(t, 0.0, score)
	assert.False(t, hasData)
}

// --- Relationship/event scoring tests ---

func TestScoreSharedRelationships_CommonPeer(t *testing.T) {
	idx := &duplicateIndex{
		personRelPeers: map[string]map[string]bool{
			"person-a": {"person-x": true, "person-y": true},
			"person-b": {"person-x": true, "person-z": true},
		},
	}
	score, detail, hasData := scoreSharedRelationships("person-a", "person-b", idx)
	assert.True(t, score > 0, "should have positive score for shared peer")
	assert.Contains(t, detail, "1 shared")
	assert.True(t, hasData)
}

func TestScoreSharedRelationships_NoOverlap(t *testing.T) {
	idx := &duplicateIndex{
		personRelPeers: map[string]map[string]bool{
			"person-a": {"person-x": true},
			"person-b": {"person-y": true},
		},
	}
	score, detail, hasData := scoreSharedRelationships("person-a", "person-b", idx)
	assert.Equal(t, 0.0, score)
	assert.Equal(t, "no overlap", detail)
	assert.True(t, hasData, "both persons have peers; no-overlap is a real negative signal, not missing data")
}

func TestScoreSharedEvents_CommonEvent(t *testing.T) {
	idx := &duplicateIndex{
		personEvents: map[string][]string{
			"person-a": {"event-census-1860", "event-birth-a"},
			"person-b": {"event-census-1860", "event-birth-b"},
		},
	}
	score, detail, hasData := scoreSharedEvents("person-a", "person-b", idx)
	assert.True(t, score > 0, "should have positive score for shared event")
	assert.Contains(t, detail, "1 shared")
	assert.True(t, hasData)
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
			"person-r-webb":      {Properties: map[string]any{"name": "R Webb"}},
			"person-robert-webb": {Properties: map[string]any{"name": "Robert Webb"}},
		},
		Events: map[string]*Event{
			"event-birth-r-webb": {
				Type: EventTypeBirth, Date: "1815", PlaceID: "place-va",
				Participants: []Participant{{Person: "person-r-webb", Role: ParticipantRolePrincipal}},
			},
			"event-birth-robert-webb": {
				Type: EventTypeBirth, Date: "1815", PlaceID: "place-va",
				Participants: []Participant{{Person: "person-robert-webb", Role: ParticipantRolePrincipal}},
			},
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
			"person-john":    {Properties: map[string]any{"name": "John Smith"}},
			"person-john-jr": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: map[string]*Event{
			"event-birth-john": {
				Type: EventTypeBirth, Date: "1850",
				Participants: []Participant{{Person: "person-john", Role: ParticipantRolePrincipal}},
			},
			"event-birth-john-jr": {
				Type: EventTypeBirth, Date: "1875",
				Participants: []Participant{{Person: "person-john-jr", Role: ParticipantRolePrincipal}},
			},
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
			"person-a": {Properties: map[string]any{"name": "John Smith"}},
			"person-b": {Properties: map[string]any{"name": "John Smith"}},
			"person-c": {Properties: map[string]any{"name": "John Smith"}},
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
			"person-john-1": {Properties: map[string]any{"name": "John Smith"}},
			"person-john-2": {Properties: map[string]any{"name": "John Smith"}},
			"person-john-3": {Properties: map[string]any{"name": "Jon Smyth"}},
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
	assert.False(t, isInitialMatch("j", "k"))
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

// --- Cross-archive duplicate tests ---

func TestFindCrossArchiveDuplicates_ObviousDuplicate(t *testing.T) {
	dest := &GLXFile{
		Persons: map[string]*Person{
			"person-dest-1": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}
	src := &GLXFile{
		Persons: map[string]*Person{
			"person-src-1": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}

	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.2})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	assert.Equal(t, "person-dest-1", result.Pairs[0].PersonA)
	assert.Equal(t, "person-src-1", result.Pairs[0].PersonB)
	assert.Greater(t, result.Pairs[0].Score, 0.2)
}

func TestFindCrossArchiveDuplicates_NoOverlap(t *testing.T) {
	dest := &GLXFile{
		Persons: map[string]*Person{
			"person-dest-1": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}
	src := &GLXFile{
		Persons: map[string]*Person{
			"person-src-1": {Properties: map[string]any{"name": "Jane Doe"}},
		},
		Events: make(map[string]*Event),
	}

	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.5})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs)
}

func TestFindCrossArchiveDuplicates_EmptyArchive(t *testing.T) {
	dest := &GLXFile{Persons: map[string]*Person{"p-1": {}}}
	src := &GLXFile{}

	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.5})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs)
}

func TestFindCrossArchiveDuplicates_SameIDSkipped(t *testing.T) {
	// Same ID in both archives is a merge conflict, not a duplicate
	dest := &GLXFile{
		Persons: map[string]*Person{
			"person-same": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}
	src := &GLXFile{
		Persons: map[string]*Person{
			"person-same": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}

	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs, "same ID should not be reported as duplicate")
}

func TestFindCrossArchiveDuplicates_InvalidThreshold(t *testing.T) {
	dest := &GLXFile{Persons: map[string]*Person{"p-1": {}}}
	src := &GLXFile{Persons: map[string]*Person{"p-2": {}}}

	_, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: -0.1})
	require.ErrorIs(t, err, ErrInvalidThreshold)

	_, err = FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 1.5})
	require.ErrorIs(t, err, ErrInvalidThreshold)
}

func TestFindCrossArchiveDuplicates_ThresholdFiltering(t *testing.T) {
	dest := &GLXFile{
		Persons: map[string]*Person{
			"person-dest-1": {Properties: map[string]any{"name": "John Smith"}},
		},
		Events: make(map[string]*Event),
	}
	src := &GLXFile{
		Persons: map[string]*Person{
			"person-src-1": {Properties: map[string]any{"name": "Jon Smyth"}},
		},
		Events: make(map[string]*Event),
	}

	// High threshold should filter out weak matches. Under renormalization (#716),
	// an exact name-only pair scores 1.0 on the available evidence; this fixture
	// uses similar-but-not-exact names so threshold 1.0 still filters it.
	result, err := FindCrossArchiveDuplicates(dest, src, DuplicateOptions{Threshold: 1.0})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs, "threshold 1.0 should filter out non-exact name matches")
}

// --- Age plausibility tests ---

func TestScoreAgePlausibility_NoData(t *testing.T) {
	idx := &duplicateIndex{
		personBirthYear:       map[string]int{},
		personFirstParentYear: map[string]int{},
	}
	score, detail := scoreAgePlausibility("person-a", "person-b", idx)
	assert.InDelta(t, 1.0, score, 1e-9)
	assert.Equal(t, "plausible", detail)
}

func TestScoreAgePlausibility_PlausibleParent(t *testing.T) {
	// A born 1700, parent at 1720 (age 20). B born 1702, parent at 1725 (age 23).
	// Both directions land inside [15, 100] — plausible.
	idx := &duplicateIndex{
		personBirthYear:       map[string]int{"person-a": 1700, "person-b": 1702},
		personFirstParentYear: map[string]int{"person-a": 1720, "person-b": 1725},
	}
	score, detail := scoreAgePlausibility("person-a", "person-b", idx)
	assert.InDelta(t, 1.0, score, 1e-9)
	assert.Equal(t, "plausible", detail)
}

func TestScoreAgePlausibility_InfantVsAdultParent(t *testing.T) {
	// A is documented as parent in 1720. B is born in 1725.
	// If A and B were the same person, B would be -5 at A's parenthood → impossible.
	idx := &duplicateIndex{
		personBirthYear:       map[string]int{"person-b": 1725},
		personFirstParentYear: map[string]int{"person-a": 1720},
	}
	score, detail := scoreAgePlausibility("person-a", "person-b", idx)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.Contains(t, detail, "person-a")
	assert.Contains(t, detail, "1720")
	assert.Contains(t, detail, "person-b")
	assert.Contains(t, detail, "1725")
	assert.Contains(t, detail, "-5")
}

func TestScoreAgePlausibility_OverhundredYearGap(t *testing.T) {
	// A is documented as parent in 1720. B is born in 1500.
	// If A and B were the same person, B would be 220 at A's parenthood → impossible.
	idx := &duplicateIndex{
		personBirthYear:       map[string]int{"person-b": 1500},
		personFirstParentYear: map[string]int{"person-a": 1720},
	}
	score, detail := scoreAgePlausibility("person-a", "person-b", idx)
	assert.InDelta(t, 0.0, score, 1e-9)
	assert.Contains(t, detail, "220")
}

func TestScoreAgePlausibility_SymmetricDirection(t *testing.T) {
	// B is documented as parent in 1720. A is born 1725.
	// Should still fire — direction shouldn't matter.
	idx := &duplicateIndex{
		personBirthYear:       map[string]int{"person-a": 1725},
		personFirstParentYear: map[string]int{"person-b": 1720},
	}
	score, _ := scoreAgePlausibility("person-a", "person-b", idx)
	assert.InDelta(t, 0.0, score, 1e-9)
}

func TestFindDuplicates_AgeImplausibleSuppressed(t *testing.T) {
	// Replicates issue #717: a putative father (no birth date) gets matched
	// against his newborn son with the same name. The son is principal of a
	// 1725 birth event; the father is documented as parent on a sibling's
	// 1720 birth event. Previously, this generationally impossible pair was
	// not suppressed and could still rank as a strong duplicate candidate.
	// The pair must score 0 (suppressed).
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-johann-juncker":     {Properties: map[string]any{"name": "Johann Peter Juncker"}},
			"person-johann-jungk-b1725": {Properties: map[string]any{"name": "Johann Peter Jungk"}},
			"person-sibling":            {Properties: map[string]any{"name": "Anna Maria Jungk"}},
		},
		Events: map[string]*Event{
			"event-birth-jungk-b1725": {
				Type: EventTypeBirth, Date: "1725-02-18",
				Participants: []Participant{
					{Person: "person-johann-jungk-b1725", Role: ParticipantRolePrincipal},
				},
			},
			"event-birth-sibling": {
				Type: EventTypeBirth, Date: "1720",
				Participants: []Participant{
					{Person: "person-sibling", Role: ParticipantRolePrincipal},
					{Person: "person-johann-juncker", Role: ParticipantRoleParent},
				},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)

	var found bool
	for _, pair := range result.Pairs {
		bothInvolved := (pair.PersonA == "person-johann-juncker" && pair.PersonB == "person-johann-jungk-b1725") ||
			(pair.PersonA == "person-johann-jungk-b1725" && pair.PersonB == "person-johann-juncker")
		if bothInvolved {
			found = true
			assert.InDelta(t, 0.0, pair.Score, 1e-9, "infant-vs-adult-parent pair must be suppressed")
			var hasAgeSignal bool
			for _, sig := range pair.Signals {
				if sig.Name == "Age plausibility" {
					hasAgeSignal = true
					assert.InDelta(t, 0.0, sig.Score, 1e-9)
					assert.Contains(t, sig.Detail, "1720")
				}
			}
			assert.True(t, hasAgeSignal, "suppressed pair must include Age plausibility signal in breakdown")
		}
	}
	require.True(t, found, "candidate pair (juncker, jungk-b1725) must appear in result.Pairs at threshold 0")
}

func TestFindDuplicates_AgeImplausibilityViaParentChildRelationship(t *testing.T) {
	// Same scenario but with the parent role established via a parent_child
	// Relationship rather than directly on a birth event. The relationship
	// has no own date, so the year is inferred from the child's birth event.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-father":  {Properties: map[string]any{"name": "Hans Schmidt"}},
			"person-newborn": {Properties: map[string]any{"name": "Hans Schmidt"}},
			"person-child":   {Properties: map[string]any{"name": "Anna Schmidt"}},
		},
		Events: map[string]*Event{
			"event-birth-newborn": {
				Type: EventTypeBirth, Date: "1730",
				Participants: []Participant{{Person: "person-newborn", Role: ParticipantRolePrincipal}},
			},
			"event-birth-child": {
				Type: EventTypeBirth, Date: "1725",
				Participants: []Participant{{Person: "person-child", Role: ParticipantRolePrincipal}},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-father-child": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleParent},
					{Person: "person-child", Role: ParticipantRoleChild},
				},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)

	var found bool
	for _, pair := range result.Pairs {
		bothInvolved := (pair.PersonA == "person-father" && pair.PersonB == "person-newborn") ||
			(pair.PersonA == "person-newborn" && pair.PersonB == "person-father")
		if bothInvolved {
			found = true
			assert.InDelta(t, 0.0, pair.Score, 1e-9, "father-vs-newborn pair must be suppressed when parent role established by relationship")
			var hasAgeSignal bool
			for _, sig := range pair.Signals {
				if sig.Name == "Age plausibility" {
					hasAgeSignal = true
					assert.InDelta(t, 0.0, sig.Score, 1e-9)
					assert.Contains(t, sig.Detail, "1725")
				}
			}
			assert.True(t, hasAgeSignal, "suppressed pair must include Age plausibility signal in breakdown")
		}
	}
	require.True(t, found, "candidate pair (father, newborn) must appear in result.Pairs at threshold 0")
}

// --- Renormalization tests (issue #716) ---
//
// These tests pin the behavior that when a dimension has no data on at least
// one side, it drops out of both the numerator and the denominator of the
// pair's weighted average. A name-only exact match must therefore score on
// the merit of its name comparison alone, not on name-weight (0.30) over a
// fixed denominator of 1.0.

func TestFindDuplicates_NameOnly_ExactMatch_ScoresHigh(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "Johann Peter Juncker"}},
			"person-b": {Properties: map[string]any{"name": "Johann Peter Juncker"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	assert.GreaterOrEqual(t, result.Pairs[0].Score, 0.9, "exact name match with no other data must renormalize to ~1.0")
}

func TestFindDuplicates_NameOnly_NearMatch_AboveThreshold(t *testing.T) {
	// The worked example from issue #716: 18th-century Enkirch father pair with
	// only names ("Johann Peter Juncker" / "Johann Peter Jungk"). Pre-fix this
	// scored 0.24 against the 0.60 default; post-fix it must surface.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-juncker": {Properties: map[string]any{"name": "Johann Peter Juncker"}},
			"person-jungk":   {Properties: map[string]any{"name": "Johann Peter Jungk"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1, "Juncker/Jungk must clear the default threshold post-renormalization")
	assert.GreaterOrEqual(t, result.Pairs[0].Score, 0.6)
}

func TestFindDuplicates_NameOnly_Mismatch_BelowThreshold(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "John Smith"}},
			"person-b": {Properties: map[string]any{"name": "Mary Johnson"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	assert.Empty(t, result.Pairs, "dissimilar name-only pairs must still fall below the default threshold")
}

func TestFindDuplicates_PartialData_NameAndBirthYearOnly(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "Robert Webb"}},
			"person-b": {Properties: map[string]any{"name": "Robert Webb"}},
		},
		Events: map[string]*Event{
			"event-birth-a": {
				Type: EventTypeBirth, Date: "1815",
				Participants: []Participant{{Person: "person-a", Role: ParticipantRolePrincipal}},
			},
			"event-birth-b": {
				Type: EventTypeBirth, Date: "1815",
				Participants: []Participant{{Person: "person-b", Role: ParticipantRolePrincipal}},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	assert.GreaterOrEqual(t, result.Pairs[0].Score, 0.9, "exact name + exact phonetic + exact birth year (effective weight 0.50: 0.20 name + 0.10 phonetic + 0.20 birth year) renormalizes to ~1.0")
}

func TestFindDuplicates_FullData_RegressionUnchanged(t *testing.T) {
	// Every weighted dimension has data on both sides (and the age gate does
	// not fire). effectiveWeight == 1.0, so the renormalized score must equal
	// the sum-of-weight*score contributions byte-for-byte — proving the fix
	// is a strict generalization of the old algorithm rather than a behavior
	// change on fully-documented pairs.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "William Smith"}},
			"person-b": {Properties: map[string]any{"name": "Bill Smith"}},
		},
		Events: map[string]*Event{
			"event-birth-a": {
				Type: EventTypeBirth, Date: "1850", PlaceID: "place-london",
				Participants: []Participant{{Person: "person-a", Role: ParticipantRolePrincipal}},
			},
			"event-birth-b": {
				Type: EventTypeBirth, Date: "1851", PlaceID: "place-london",
				Participants: []Participant{{Person: "person-b", Role: ParticipantRolePrincipal}},
			},
			"event-death-a": {
				Type: EventTypeDeath, Date: "1920", PlaceID: "place-london",
				Participants: []Participant{{Person: "person-a", Role: ParticipantRolePrincipal}},
			},
			"event-death-b": {
				Type: EventTypeDeath, Date: "1922", PlaceID: "place-paris",
				Participants: []Participant{{Person: "person-b", Role: ParticipantRolePrincipal}},
			},
			"event-shared": {
				Type: EventTypeBirth, Date: "1900",
				Participants: []Participant{
					{Person: "person-a", Role: "witness"},
					{Person: "person-b", Role: "witness"},
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-shared-peer-a": {
				Type: "spouse",
				Participants: []Participant{
					{Person: "person-a", Role: "spouse"},
					{Person: "person-shared-peer", Role: "spouse"},
				},
			},
			"rel-shared-peer-b": {
				Type: "spouse",
				Participants: []Participant{
					{Person: "person-b", Role: "spouse"},
					{Person: "person-shared-peer", Role: "spouse"},
				},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)

	var found bool
	for _, pair := range result.Pairs {
		if (pair.PersonA == "person-a" && pair.PersonB == "person-b") ||
			(pair.PersonA == "person-b" && pair.PersonB == "person-a") {
			found = true
			var weightedSum, effective float64
			for _, sig := range pair.Signals {
				if sig.Weight > 0 && sig.HasData {
					weightedSum += sig.Weight * sig.Score
					effective += sig.Weight
				}
			}
			require.InDelta(t, 1.0, effective, 1e-12, "fixture must cover all weighted dimensions; effective weight should be 1.0")
			// FindDuplicates rounds pair.Score to 2 decimal places (see duplicates.go).
			// Replicate that rounding so the comparison checks the algorithmic equivalence,
			// not the display-precision rounding. InDelta with delta=0 is the exact-equality
			// assertion the testifylint plugin lets us write for floats.
			expected := math.Round(weightedSum*100) / 100
			assert.InDelta(t, expected, pair.Score, 0, "with full data the renormalized score must equal the old weighted sum exactly (post-rounding)")
		}
	}
	require.True(t, found, "(person-a, person-b) must appear in result.Pairs")
}

func TestFindDuplicates_NoComparableData(t *testing.T) {
	// Both persons present, neither has a name or any events — no signal can
	// produce data. effectiveWeight == 0 path; the pair must still surface at
	// threshold 0 (FindDuplicates uses `>=`), and its score must be 0.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{}},
			"person-b": {Properties: map[string]any{}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)

	var found bool
	for _, pair := range result.Pairs {
		if (pair.PersonA == "person-a" && pair.PersonB == "person-b") ||
			(pair.PersonA == "person-b" && pair.PersonB == "person-a") {
			found = true
			assert.InDelta(t, 0.0, pair.Score, 1e-9, "no-data pair must score 0 via the effectiveWeight==0 branch")
		}
	}
	require.True(t, found, "(person-a, person-b) must appear in result.Pairs at threshold 0; otherwise the score assertion above would pass vacuously")
}

func TestFindDuplicates_OneSidedPartial_NameExactMatch_ScoresHigh(t *testing.T) {
	// Person A is fully documented; Person B has only a name (exact match to
	// A). Every non-name dimension hits `yearB == 0 || placeB == ""` etc. on
	// the B side and returns HasData=false, so only Name participates in the
	// weighted average. The pair therefore renormalizes to ~1.0.
	//
	// This is the deliberate trade-off of pure renormalization (issue #716,
	// option (a)): a thinly-documented record matches cleanly against a
	// well-documented one of the same name. Locked in by this test so the
	// behavior is visible to reviewers, not stumbled into. A confidence floor
	// based on effective weight is a possible follow-up.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-a": {Properties: map[string]any{"name": "Robert Webb"}},
			"person-b": {Properties: map[string]any{"name": "Robert Webb"}},
		},
		Events: map[string]*Event{
			"event-birth-a": {
				Type: EventTypeBirth, Date: "1815", PlaceID: "place-va",
				Participants: []Participant{{Person: "person-a", Role: ParticipantRolePrincipal}},
			},
			"event-death-a": {
				Type: EventTypeDeath, Date: "1885", PlaceID: "place-va",
				Participants: []Participant{{Person: "person-a", Role: ParticipantRolePrincipal}},
			},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.6})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)
	assert.GreaterOrEqual(t, result.Pairs[0].Score, 0.9, "name-only B must renormalize to ~1.0 against well-documented A")
}

func TestFindDuplicates_AgeImplausibilityParentYearEarliestWinsAcrossSources(t *testing.T) {
	// Parent role is established by two sources with conflicting years:
	// - relationship inferred from child's birth year (1720)
	// - direct event participation as parent in a later year (1740)
	// The earliest parent year must win.
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-father":  {Properties: map[string]any{"name": "Hans Schmidt"}},
			"person-newborn": {Properties: map[string]any{"name": "Hans Schmidt"}},
			"person-child":   {Properties: map[string]any{"name": "Anna Schmidt"}},
		},
		Events: map[string]*Event{
			"event-birth-newborn": {
				Type: EventTypeBirth, Date: "1730",
				Participants: []Participant{{Person: "person-newborn", Role: ParticipantRolePrincipal}},
			},
			"event-birth-child": {
				Type: EventTypeBirth, Date: "1720",
				Participants: []Participant{{Person: "person-child", Role: ParticipantRolePrincipal}},
			},
			"event-birth-late-parent-role": {
				Type: EventTypeBirth, Date: "1740",
				Participants: []Participant{
					{Person: "person-child", Role: ParticipantRolePrincipal},
					{Person: "person-father", Role: ParticipantRoleParent},
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-father-child": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleParent},
					{Person: "person-child", Role: ParticipantRoleChild},
				},
			},
		},
	}

	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)

	var found bool
	for _, pair := range result.Pairs {
		bothInvolved := (pair.PersonA == "person-father" && pair.PersonB == "person-newborn") ||
			(pair.PersonA == "person-newborn" && pair.PersonB == "person-father")
		if bothInvolved {
			found = true
			assert.InDelta(t, 0.0, pair.Score, 1e-9, "earliest parent year (1720) should suppress father-vs-newborn pair")
			var hasAgeSignal bool
			for _, sig := range pair.Signals {
				if sig.Name == "Age plausibility" {
					hasAgeSignal = true
					assert.InDelta(t, 0.0, sig.Score, 1e-9)
					assert.Contains(t, sig.Detail, "1720")
				}
			}
			assert.True(t, hasAgeSignal, "suppressed pair must include Age plausibility signal in breakdown")
		}
	}
	require.True(t, found, "candidate pair (father, newborn) must appear in result.Pairs at threshold 0")
}

// --- Phonetic end-to-end (#704) ---

func TestFindDuplicates_PhoneticSignalAppearsOnSchneiderSnider(t *testing.T) {
	// End-to-end claim: a Schneider/Snider pair surfaces a populated
	// Phonetic signal in pair.Signals, and the total pair score is strictly
	// greater than it would be if the Phonetic dimension were absent.
	//
	// Stated this way (rather than as a threshold-crossing assertion against
	// the 0.60 default) the regression is robust to future weight tuning:
	// what we're locking in is "phonetic contributes positively when names
	// are phonetically equivalent but lexically distant", not "this exact
	// pair clears this exact threshold under these exact weights".
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-schneider": {Properties: map[string]any{"name": "Hans Schneider"}},
			"person-snider":    {Properties: map[string]any{"name": "Hans Snider"}},
		},
	}
	result, err := FindDuplicates(archive, DuplicateOptions{Threshold: 0.0})
	require.NoError(t, err)
	require.Len(t, result.Pairs, 1)

	pair := result.Pairs[0]

	var (
		phonSig       DuplicateSignal
		phonFound     bool
		sumWithout    float64
		weightWithout float64
	)
	for _, sig := range pair.Signals {
		if sig.Name == "Phonetic" {
			phonSig = sig
			phonFound = true

			continue
		}
		if sig.Weight > 0 && sig.HasData {
			sumWithout += sig.Weight * sig.Score
			weightWithout += sig.Weight
		}
	}
	require.True(t, phonFound, "Phonetic signal must appear in pair.Signals")
	assert.InDelta(t, 1.0, phonSig.Score, 1e-12, "Schneider/Snider + Hans/Hans phonetically match on both components")
	assert.True(t, phonSig.HasData)
	assert.InDelta(t, weightPhonetic, phonSig.Weight, 1e-12)

	// Name signal always contributes on this fixture (both persons carry a
	// non-empty `name`), so weightWithout is guaranteed non-zero — the
	// division below is safe without a guard.
	scoreWithoutPhonetic := sumWithout / weightWithout
	assert.Greater(t, pair.Score, scoreWithoutPhonetic, "phonetic signal must lift the renormalized score on a Schneider/Snider pair (the #704 motivating case)")
}
