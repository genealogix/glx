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

// setCensusFallback overrides the fallback census country for one test and
// restores it afterwards, so tests that exercise the flag do not leak into
// the tests that rely on the default.
func setCensusFallback(t *testing.T, country string) {
	t.Helper()
	previous := censusCountryFallback
	censusCountryFallback = country
	t.Cleanup(func() { censusCountryFallback = previous })
}

// mecklenburgArchive is the #186 repro: a person born, married and buried in
// one Mecklenburg-Strelitz village, with no US connection at all.
func mecklenburgArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-hollnagel": {
				Properties: map[string]any{glxlib.PersonPropertyName: "Michael David Hollnagel"},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-hrr":                  {Name: "Heiliges Römisches Reich", Type: glxlib.PlaceTypeCountry},
			"place-mecklenburg-strelitz": {Name: "Mecklenburg-Strelitz", Type: glxlib.PlaceTypeState, ParentID: "place-hrr"},
			"place-liepen":               {Name: "Liepen", Type: glxlib.PlaceTypeLocality, ParentID: "place-mecklenburg-strelitz"},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type: glxlib.EventTypeBirth, Date: "1742", PlaceID: "place-liepen",
				Participants: []glxlib.Participant{{Person: "person-hollnagel", Role: "subject"}},
			},
			"event-death": {
				Type: glxlib.EventTypeDeath, Date: "1807-08-31", PlaceID: "place-liepen",
				Participants: []glxlib.Participant{{Person: "person-hollnagel", Role: "subject"}},
			},
		},
	}
}

func TestCanonicalCountry(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"exact name", "United States", countryUnitedStates},
		{"abbreviation", "USA", countryUnitedStates},
		{"dotted abbreviation", "U.S.", countryUnitedStates},
		{"case and spacing", "  united   states of america ", countryUnitedStates},
		{"constituent country", "Scotland", countryUnitedKingdom},
		{"northern ireland is UK", "Northern Ireland", countryUnitedKingdom},
		{"republic of ireland", "Éire", countryIreland},
		{"canada", "Dominion of Canada", countryCanada},
		{"unknown country", "Heiliges Römisches Reich", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, canonicalCountry(tc.input))
		})
	}
}

func TestResolveCountryFromPlace(t *testing.T) {
	archive := mecklenburgArchive()

	assert.Equal(t, "Heiliges Römisches Reich", resolveCountryFromPlace("place-liepen", archive),
		"should walk the parent chain to the country place")
	assert.Equal(t, "Heiliges Römisches Reich", resolveCountryFromPlace("place-hrr", archive),
		"a country place resolves to itself")
	assert.Empty(t, resolveCountryFromPlace("", archive))
	assert.Empty(t, resolveCountryFromPlace("place-missing", archive))
}

func TestResolveCountryFromPlace_NoCountryAncestor(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-state": {Name: "Illinois", Type: glxlib.PlaceTypeState},
			"place-city":  {Name: "Springfield", Type: glxlib.PlaceTypeCity, ParentID: "place-state"},
		},
	}

	assert.Empty(t, resolveCountryFromPlace("place-city", archive),
		"a hierarchy that stops at a state names no country")
}

func TestResolveCountryFromPlace_CycleTerminates(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-a": {Name: "A", Type: glxlib.PlaceTypeLocality, ParentID: "place-b"},
			"place-b": {Name: "B", Type: glxlib.PlaceTypeLocality, ParentID: "place-a"},
		},
	}

	assert.Empty(t, resolveCountryFromPlace("place-a", archive))
}

func TestCensusSchedulesForPlaces(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-us":      {Name: "United States", Type: glxlib.PlaceTypeCountry},
			"place-boston":  {Name: "Boston", Type: glxlib.PlaceTypeCity, ParentID: "place-us"},
			"place-uk":      {Name: "England", Type: glxlib.PlaceTypeCountry},
			"place-bath":    {Name: "Bath", Type: glxlib.PlaceTypeCity, ParentID: "place-uk"},
			"place-hrr":     {Name: "Heiliges Römisches Reich", Type: glxlib.PlaceTypeCountry},
			"place-liepen":  {Name: "Liepen", Type: glxlib.PlaceTypeLocality, ParentID: "place-hrr"},
			"place-untyped": {Name: "Somewhere"},
		},
	}

	t.Run("recognized country", func(t *testing.T) {
		schedules := censusSchedulesForPlaces([]string{"place-boston"}, archive)
		require.Len(t, schedules, 1)
		assert.Equal(t, countryUnitedStates, schedules[0].country)
	})

	t.Run("unrecognized country suggests nothing", func(t *testing.T) {
		assert.Empty(t, censusSchedulesForPlaces([]string{"place-liepen"}, archive),
			"a country with no schedule should not fall back to the US one")
	})

	t.Run("places spanning two countries get both", func(t *testing.T) {
		schedules := censusSchedulesForPlaces([]string{"place-bath", "place-boston"}, archive)
		require.Len(t, schedules, 2)
		assert.Equal(t, countryUnitedKingdom, schedules[0].country, "schedules are ordered by country name")
		assert.Equal(t, countryUnitedStates, schedules[1].country)
	})

	t.Run("no country at all falls back", func(t *testing.T) {
		schedules := censusSchedulesForPlaces([]string{"place-untyped"}, archive)
		require.Len(t, schedules, 1)
		assert.Equal(t, countryUnitedStates, schedules[0].country)
	})

	t.Run("fallback is configurable", func(t *testing.T) {
		setCensusFallback(t, countryUnitedKingdom)
		schedules := censusSchedulesForPlaces(nil, archive)
		require.Len(t, schedules, 1)
		assert.Equal(t, countryUnitedKingdom, schedules[0].country)
	})

	t.Run("fallback can be disabled", func(t *testing.T) {
		setCensusFallback(t, "")
		assert.Empty(t, censusSchedulesForPlaces(nil, archive))
	})

	t.Run("an unrecognized country still beats the fallback", func(t *testing.T) {
		setCensusFallback(t, countryUnitedKingdom)
		assert.Empty(t, censusSchedulesForPlaces([]string{"place-liepen"}, archive),
			"the archive says where the person was; the fallback is only for when it does not")
	})
}

func TestParseCensusCountry(t *testing.T) {
	setCensusFallback(t, countryUnitedStates)

	t.Run("empty keeps the default", func(t *testing.T) {
		country, err := parseCensusCountry("")
		require.NoError(t, err)
		assert.Equal(t, countryUnitedStates, country)
	})

	t.Run("none disables", func(t *testing.T) {
		country, err := parseCensusCountry("none")
		require.NoError(t, err)
		assert.Empty(t, country)
	})

	t.Run("alias resolves", func(t *testing.T) {
		country, err := parseCensusCountry("uk")
		require.NoError(t, err)
		assert.Equal(t, countryUnitedKingdom, country)
	})

	t.Run("unknown country is an error naming the known ones", func(t *testing.T) {
		_, err := parseCensusCountry("Atlantis")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "United States")
		assert.Contains(t, err.Error(), "none")
	})
}

func TestAllCensusYears_IsSortedAndDeduplicated(t *testing.T) {
	years := allCensusYears()
	require.NotEmpty(t, years)

	seen := make(map[int]bool, len(years))
	for i, year := range years {
		assert.False(t, seen[year], "year %d appears twice", year)
		seen[year] = true
		if i > 0 {
			assert.Greater(t, year, years[i-1], "years should ascend")
		}
	}
	assert.True(t, seen[1841], "UK years should be included")
	assert.True(t, seen[1850], "US years should be included")
}

// TestCoverage_NonUSPersonHasNoCensusRows is the #186 regression: a person
// with no US connection was shown a US federal census checklist.
func TestCoverage_NonUSPersonHasNoCensusRows(t *testing.T) {
	archive := mecklenburgArchive()

	result := buildCoverage("person-hollnagel", archive.Persons["person-hollnagel"], archive)

	for _, record := range result.Records {
		assert.NotEqual(t, "census", record.Category,
			"no census schedule applies to Mecklenburg-Strelitz; got %q", record.Label)
	}
}

// TestAnalyzeSuggestions_NonUSPersonGetsNoCensusSuggestions is the other half
// of the #186 regression.
func TestAnalyzeSuggestions_NonUSPersonGetsNoCensusSuggestions(t *testing.T) {
	archive := mecklenburgArchive()

	issues := suggestCensusSearches(archive)

	assert.Empty(t, issues, "a person who never left Mecklenburg needs no US census searches")
}

func TestAnalyzeSuggestions_UKPersonGetsUKCensusYears(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane": {Properties: map[string]any{glxlib.PersonPropertyName: "Jane Webb"}},
		},
		Places: map[string]*glxlib.Place{
			"place-uk":   {Name: "England", Type: glxlib.PlaceTypeCountry},
			"place-bath": {Name: "Bath", Type: glxlib.PlaceTypeCity, ParentID: "place-uk"},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type: glxlib.EventTypeBirth, Date: "1835", PlaceID: "place-bath",
				Participants: []glxlib.Participant{{Person: "person-jane", Role: "subject"}},
			},
			"event-death": {
				Type: glxlib.EventTypeDeath, Date: "1899", PlaceID: "place-bath",
				Participants: []glxlib.Participant{{Person: "person-jane", Role: "subject"}},
			},
		},
	}

	issues := suggestCensusSearches(archive)

	require.NotNil(t, findIssueByMessage(issues, "person-jane", "1841 UK census"),
		"the 1841 UK census should be suggested")
	assert.Nil(t, findIssueByMessage(issues, "person-jane", "US census"),
		"no US census should be suggested for a person who never left Bath")
}

func TestSuggestVitalRecords_ChurchRegisterCounts(t *testing.T) {
	archive := mecklenburgArchive()
	archive.Sources = map[string]*glxlib.Source{
		"source-kirchenbuch": {
			Title: "Kirchenbuch Liepen, Sterberegister 1807",
			Type:  glxlib.SourceTypeChurchRegister,
		},
	}
	archive.Assertions = map[string]*glxlib.Assertion{
		"assertion-death": {
			Subject:  glxlib.EntityRef{Person: "person-hollnagel"},
			Property: "death_date",
			Value:    "1807-08-31",
			Sources:  []string{"source-kirchenbuch"},
		},
	}

	issues := suggestVitalRecords(archive)

	assert.Empty(t, issues,
		"a parish register is the vital record where civil registration did not exist")
}

func TestSuggestVitalRecords_NoSourceStillSuggests(t *testing.T) {
	archive := mecklenburgArchive()

	issues := suggestVitalRecords(archive)

	require.Len(t, issues, 1)
	assert.Contains(t, issues[0].Message, "search vital records")
}
