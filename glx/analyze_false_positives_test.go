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
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// --- #1323: research logs, studies and media reference citations and sources ---

func TestAnalyzeEvidence_ResearchLogReferencesCitation(t *testing.T) {
	archive := &glxlib.GLXFile{
		Sources: map[string]*glxlib.Source{"src-fag": {Title: "Find a Grave"}},
		Citations: map[string]*glxlib.Citation{
			"cit-search":   {SourceID: "src-fag"},
			"cit-loglevel": {SourceID: "src-fag"},
			"cit-orphan":   {SourceID: "src-fag"},
		},
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"rl-death": {
				Searches:  []glxlib.Search{{SourceID: "src-fag", Result: "not_found", CitationID: "cit-search"}},
				Citations: []string{"cit-loglevel"},
			},
		},
	}

	issues := analyzeEvidence(archive)
	assert.Nil(t, findIssueByEntity(issues, "cit-search"), "a search's citation is referenced")
	assert.Nil(t, findIssueByEntity(issues, "cit-loglevel"), "a log-level citation is referenced")
	assert.NotNil(t, findIssueByEntity(issues, "cit-orphan"), "an unreferenced citation is still orphaned")
}

func TestAnalyzeEvidence_SourcesReferencedOutsideCitations(t *testing.T) {
	archive := &glxlib.GLXFile{
		Sources: map[string]*glxlib.Source{
			"src-searched": {Title: "Searched"},
			"src-study":    {Title: "In study scope"},
			"src-media":    {Title: "Media source"},
			"src-orphan":   {Title: "Nothing refers to me"},
		},
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"rl-1": {Searches: []glxlib.Search{{SourceID: "src-searched", Result: "not_found"}}},
		},
		Studies: map[string]*glxlib.Study{"study-1": {Title: "ONS", Sources: []string{"src-study"}}},
		Media:   map[string]*glxlib.Media{"media-1": {URI: "x.jpg", Source: "src-media"}},
	}

	issues := analyzeEvidence(archive)
	assert.Nil(t, findIssueByEntity(issues, "src-searched"))
	assert.Nil(t, findIssueByEntity(issues, "src-study"))
	assert.Nil(t, findIssueByEntity(issues, "src-media"))
	assert.NotNil(t, findIssueByEntity(issues, "src-orphan"))
}

// --- #1324: a birthplace on the same branch of the place hierarchy ---

// hierarchyBirthplaceArchive is the issue's Little family: three children
// born in Rowan County and one (James) born in the given place.
func hierarchyBirthplaceArchive(jamesPlace string) *glxlib.GLXFile {
	archive := siblingBirthplaceArchive("person-lewis", "", []struct{ id, place string }{
		{"person-jasper", "place-rowan"},
		{"person-sarah", "place-rowan"},
		{"person-elizabeth", "place-rowan"},
		{"person-james", jamesPlace},
	})
	archive.Places["place-usa"] = &glxlib.Place{Name: "United States", Type: glxlib.PlaceTypeCountry}
	archive.Places["place-nc"] = &glxlib.Place{Name: "North Carolina", Type: glxlib.PlaceTypeState, ParentID: "place-usa"}
	archive.Places["place-rowan"] = &glxlib.Place{Name: "Rowan County", Type: glxlib.PlaceTypeCounty, ParentID: "place-nc"}
	archive.Places["place-salisbury"] = &glxlib.Place{Name: "Salisbury", Type: glxlib.PlaceTypeCity, ParentID: "place-rowan"}
	archive.Places["place-surry"] = &glxlib.Place{Name: "Surry County", Type: glxlib.PlaceTypeCounty, ParentID: "place-nc"}

	return archive
}

func TestAnalyzeConsistency_SiblingBirthplace_CoarserPlaceIsInfo(t *testing.T) {
	issues := analyzeConsistency(hierarchyBirthplaceArchive("place-nc"))

	found := findIssueByMessage(issues, "person-james", "less specific")
	require.NotNil(t, found, "expected an info prompt for the coarser birthplace")
	assert.Equal(t, "info", found.Severity)
	assert.Contains(t, found.Message, "North Carolina")
	assert.Contains(t, found.Message, "Rowan County")
	assert.Nil(t, findIssueByMessage(issues, "person-james", "born in North Carolina while"),
		"the coarser place must not be reported as an outlier")
}

func TestAnalyzeConsistency_SiblingBirthplace_FinerPlaceNotReported(t *testing.T) {
	issues := analyzeConsistency(hierarchyBirthplaceArchive("place-salisbury"))

	for _, issue := range issues {
		assert.NotEqual(t, "person-james", issue.Person, "a birthplace inside the majority place agrees with it: %s", issue.Message)
	}
}

func TestAnalyzeConsistency_SiblingBirthplace_SiblingCountyStillOutlier(t *testing.T) {
	issues := analyzeConsistency(hierarchyBirthplaceArchive("place-surry"))

	found := findIssueByMessage(issues, "person-james", "born in Surry County")
	require.NotNil(t, found, "a different county of the same state is still an outlier")
	assert.Equal(t, "medium", found.Severity)
}

// --- #1338: license, banns, contract and settlement events ---

func licenseOnlyArchive(eventType, date string, viaStartEvent bool) *glxlib.GLXFile {
	rel := &glxlib.Relationship{
		Type: "marriage",
		Participants: []glxlib.Participant{
			{Person: "person-adam", Role: "spouse"},
			{Person: "person-elizabeth", Role: "spouse"},
		},
	}
	event := &glxlib.Event{Type: eventType, Date: glxlib.DateString(date), PlaceID: "place-surry"}
	if viaStartEvent {
		rel.StartEvent = "ev-bond"
		event.Participants = []glxlib.Participant{{Person: "person-adam", Role: "groom"}}
	} else {
		event.Participants = []glxlib.Participant{
			{Person: "person-adam", Role: "groom"},
			{Person: "person-elizabeth", Role: "bride"},
		}
	}

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-adam":      {Properties: map[string]any{"name": "Adam Call"}},
			"person-elizabeth": {Properties: map[string]any{"name": "Elizabeth Starr"}},
		},
		Places:        map[string]*glxlib.Place{"place-surry": {Name: "Surry County"}},
		Events:        map[string]*glxlib.Event{"ev-bond": event},
		Relationships: map[string]*glxlib.Relationship{"rel-marr": rel},
	}
}

func TestAnalyzeGaps_LicenseOnlyMarriageIsInfo(t *testing.T) {
	for _, kind := range []string{glxlib.EventTypeMarriageLicense, glxlib.EventTypeMarriageBanns, glxlib.EventTypeMarriageContract, glxlib.EventTypeMarriageSettlement} {
		for _, viaStart := range []bool{false, true} {
			for _, placedOnly := range []bool{false, true} {
				t.Run(kind+"/start="+strconv.FormatBool(viaStart)+"/place="+strconv.FormatBool(placedOnly), func(t *testing.T) {
					archive := licenseOnlyArchive(kind, "1785-12-17", viaStart)
					if placedOnly {
						archive.Events["ev-bond"].Date = ""
					} else {
						archive.Events["ev-bond"].PlaceID = ""
					}
					issues := analyzeGaps(archive)
					for _, person := range []string{"person-adam", "person-elizabeth"} {
						assert.Nil(t, findIssueByMessage(issues, person, "no marriage event"))
						found := findIssueByMessage(issues, person, "known from "+preliminaryMarriageRecordName(kind)+" only")
						require.NotNil(t, found)
						assert.Equal(t, "info", found.Severity)
					}
				})
			}
		}
	}
}

func TestAnalyzeGaps_UndatedUnplacedLicenseStillGap(t *testing.T) {
	archive := licenseOnlyArchive(glxlib.EventTypeMarriageLicense, "", true)
	archive.Events["ev-bond"].PlaceID = ""

	issues := analyzeGaps(archive)
	found := findIssueByMessage(issues, "person-adam", "no marriage event for")
	require.NotNil(t, found)
	assert.Equal(t, "medium", found.Severity)
}

func TestAnalyzeGaps_CeremonyBeatsLicense(t *testing.T) {
	archive := licenseOnlyArchive(glxlib.EventTypeMarriageLicense, "1785-12-17", false)
	archive.Events["ev-marriage"] = &glxlib.Event{
		Type: glxlib.EventTypeMarriage,
		Date: "1785-12-20",
		Participants: []glxlib.Participant{
			{Person: "person-adam", Role: "groom"},
			{Person: "person-elizabeth", Role: "bride"},
		},
	}

	issues := analyzeGaps(archive)
	assert.Nil(t, findIssueByMessage(issues, "person-adam", "no marriage event"))
	assert.Nil(t, findIssueByMessage(issues, "person-adam", "known from"))
}

func TestAnalyzeGaps_EngagementIsNotMarriageEvidence(t *testing.T) {
	archive := licenseOnlyArchive(glxlib.EventTypeEngagement, "1785-06-01", true)

	issues := analyzeGaps(archive)
	assert.NotNil(t, findIssueByMessage(issues, "person-adam", "no marriage event for"))
}

// --- #1333: census schedules lost for a state or territory ---

// censusLossArchive builds a US place hierarchy with Dearborn County under a
// territory-typed Indiana Territory, Rowan County under North Carolina, a
// person born in Rowan County in 1770, and their residences at the given
// (place, year) pairs.
func censusLossArchive(residences map[string]int) *glxlib.GLXFile {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-lewis": {Properties: map[string]any{"name": "Lewis Little"}}},
		Places: map[string]*glxlib.Place{
			"place-usa":      {Name: "United States", Type: glxlib.PlaceTypeCountry},
			"place-int":      {Name: "Indiana Territory", Type: placeTypeTerritory, ParentID: "place-usa"},
			"place-dearborn": {Name: "Dearborn County", Type: glxlib.PlaceTypeCounty, ParentID: "place-int"},
			"place-nc":       {Name: "North Carolina", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
			"place-rowan":    {Name: "Rowan County", Type: glxlib.PlaceTypeCounty, ParentID: "place-nc"},
			"place-oh":       {Name: "Ohio", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
			"place-tn":       {Name: "Tennessee", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
			"place-ind":      {Name: "Indiana", Type: placeTypeTerritory, ParentID: "place-usa"},
		},
		Events: map[string]*glxlib.Event{
			"ev-birth": {
				Type: glxlib.EventTypeBirth, Date: "1770", PlaceID: "place-rowan",
				Participants: []glxlib.Participant{{Person: "person-lewis", Role: "principal"}},
			},
		},
	}
	for place, year := range residences {
		archive.Events["ev-res-"+place] = &glxlib.Event{
			Type: glxlib.EventTypeResidence, Date: glxlib.DateString(strconv.Itoa(year)), PlaceID: place,
			Participants: []glxlib.Participant{{Person: "person-lewis", Role: "principal"}},
		}
	}

	return archive
}

func hasRecordLabel(records []coverageRecord, prefix string) bool {
	for _, rec := range records {
		if strings.HasPrefix(rec.Label, prefix) {
			return true
		}
	}

	return false
}

func censusSuggestionFor(issues []AnalysisIssue, year string) *AnalysisIssue {
	return findIssueByMessage(issues, "person-lewis", "search "+year+" US census")
}

func TestAnalyzeSuggestions_CensusLostForTerritoryNotSuggested(t *testing.T) {
	archive := censusLossArchive(map[string]int{"place-dearborn": 1810})

	issues := suggestCensusSearches(archive)
	assert.Nil(t, censusSuggestionFor(issues, "1810"), "1810 Indiana Territory schedules are lost")
	assert.NotNil(t, censusSuggestionFor(issues, "1820"), "1820 Indiana survives")
}

func TestAnalyzeSuggestions_TerritoryNamedWithoutSuffix(t *testing.T) {
	archive := censusLossArchive(map[string]int{"place-ind": 1810})

	assert.Nil(t, censusSuggestionFor(suggestCensusSearches(archive), "1810"),
		`a territory-typed "Indiana" matches the Indiana Territory entry`)
}

func TestAnalyzeSuggestions_PartialLossKeepsSuggestionWithNote(t *testing.T) {
	archive := censusLossArchive(map[string]int{"place-tn": 1810})

	found := censusSuggestionFor(suggestCensusSearches(archive), "1810")
	require.NotNil(t, found, "1810 Tennessee retains Rutherford County")
	assert.Contains(t, found.Message, "schedules mostly lost for Tennessee")
}

func TestAnalyzeSuggestions_MixedJurisdictionsKeepSuggestionWithNote(t *testing.T) {
	// Recorded in North Carolina in 1805 and Indiana Territory in 1815: in
	// 1810 he was in one or the other, and North Carolina's 1810 survives.
	archive := censusLossArchive(map[string]int{"place-rowan": 1805, "place-dearborn": 1815})

	found := censusSuggestionFor(suggestCensusSearches(archive), "1810")
	require.NotNil(t, found)
	assert.Contains(t, found.Message, "schedules lost for Indiana Territory")
}

func TestAnalyzeSuggestions_NoStateKeepsSuggestion(t *testing.T) {
	archive := censusLossArchive(nil)
	archive.Events["ev-birth"].PlaceID = "place-usa"

	assert.NotNil(t, censusSuggestionFor(suggestCensusSearches(archive), "1810"),
		"a place with no state or territory cannot show the census is lost")
}

func TestCoverage_LostCensusNotCountedAsMissing(t *testing.T) {
	archive := censusLossArchive(map[string]int{"place-dearborn": 1810})

	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)
	for _, rec := range result.Records {
		assert.NotContains(t, rec.Label, "1810 US Census", "a lost census is not an expected record")
	}
	assert.True(t, hasRecordLabel(result.Records, "1820 US Census"))
}

func TestCoverage_FoundLostCensusStillShown(t *testing.T) {
	archive := censusLossArchive(map[string]int{"place-dearborn": 1810})
	archive.Sources = map[string]*glxlib.Source{"src-1810": {Type: glxlib.SourceTypeCensus, Date: "1810"}}
	archive.Assertions = map[string]*glxlib.Assertion{"as-census": {Subject: glxlib.EntityRef{Person: "person-lewis"}, Sources: []string{"src-1810"}}}
	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)
	assert.True(t, hasRecordLabel(result.Records, "1810 US Census"), "a census the user found anyway keeps its row")
}

func TestCensusSurvival_TotalLossTable(t *testing.T) {
	archive := &glxlib.GLXFile{Places: map[string]*glxlib.Place{
		"place-usa": {Name: "United States", Type: glxlib.PlaceTypeCountry},
	}}
	cases := []struct {
		year     int
		state    string
		wantLost bool
		wantNote string
	}{
		{1790, "Virginia", true, ""},
		{1790, "North Carolina", false, ""},
		{1800, "Kentucky", true, ""},
		{1800, "Georgia", true, ""},
		{1800, "Northwest Territory", false, "schedules mostly lost for Northwest Territory"},
		{1800, "Ohio", false, "schedules mostly lost for Ohio"},
		{1810, "District of Columbia", true, ""},
		{1810, "Ohio", true, ""},
		{1810, "Michigan Territory", true, ""},
		{1810, "Michigan", true, ""},
		{1810, "Tennessee", false, "schedules mostly lost for Tennessee"},
		{1810, "Illinois Territory", false, "schedules mostly lost for Illinois Territory"},
		{1810, "Louisiana", false, ""},
		{1820, "Arkansas Territory", true, ""},
		{1820, "New Jersey", true, ""},
		{1820, "Alabama", false, ""}, // Unclassified federal survival keeps the suggestion without a factual loss claim.
		{1830, "Virginia", false, ""},
	}
	us := scheduleForCountry(censusSchedulesForPlaces([]string{"place-usa"}, archive), countryUnitedStates)
	for _, tc := range cases {
		archive.Places["place-x"] = &glxlib.Place{Name: tc.state, Type: glxlib.PlaceTypeState, ParentID: "place-usa"}
		got := us.Survival(tc.year, []datedPlace{{Year: tc.year, PlaceID: "place-x"}}, archive)
		assert.Equal(t, tc.wantLost, got.Lost, "%d %s lost", tc.year, tc.state)
		if tc.wantNote != "" {
			assert.Contains(t, got.Note, tc.wantNote, "%d %s note", tc.year, tc.state)
		} else {
			assert.Empty(t, got.Note, "%d %s note", tc.year, tc.state)
		}
	}
}

func TestCensusSurvival_OtherCountryPlacesIgnored(t *testing.T) {
	archive := &glxlib.GLXFile{Places: map[string]*glxlib.Place{
		"place-usa": {Name: "United States", Type: glxlib.PlaceTypeCountry},
		"place-va":  {Name: "Virginia", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
		"place-ie":  {Name: "Ireland", Type: glxlib.PlaceTypeCountry},
	}}

	us := scheduleForCountry(censusSchedulesForPlaces([]string{"place-usa"}, archive), countryUnitedStates)
	// The 1790 bracket is Virginia (1785) on one side and Ireland (1790) on
	// the other; Ireland is outside the US schedule, so Virginia decides.
	got := us.Survival(1790, []datedPlace{{Year: 1785, PlaceID: "place-va"}, {Year: 1790, PlaceID: "place-ie"}}, archive)
	assert.True(t, got.Lost)
}

func TestSiblingBirthplace_StandingIntegration(t *testing.T) {
	archive := hierarchyBirthplaceArchive("place-surry")
	var relID string
	for id, rel := range archive.Relationships {
		for _, p := range rel.Participants {
			if p.Person == "person-james" && p.Role == "child" {
				relID = id
			}
		}
	}
	require.NotEmpty(t, relID)
	archive.Assertions = map[string]*glxlib.Assertion{
		"weak":   {Subject: glxlib.EntityRef{Relationship: relID}, Confidence: "low"},
		"proven": {Subject: glxlib.EntityRef{Relationship: relID}, Status: "proven"},
	}
	found := findIssueByMessage(checkSiblingBirthplaceOutlier(archive), "person-james", "born in Surry County")
	require.NotNil(t, found)
	require.Equal(t, "medium", found.Severity, "a proven survivor overrides weak standing")
	delete(archive.Assertions, "proven")
	found = findIssueByMessage(checkSiblingBirthplaceOutlier(archive), "person-james", "born in Surry County")
	require.NotNil(t, found)
	require.Equal(t, "high", found.Severity)
	archive.Events["event-birth-person-james"].PlaceID = "place-nc"
	found = findIssueByMessage(checkSiblingBirthplaceOutlier(archive), "person-james", "less specific")
	require.NotNil(t, found)
	require.Equal(t, "info", found.Severity, "coarser evidence still agrees despite weak parentage")
}
