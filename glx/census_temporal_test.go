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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestCensusLeadsUseFeasibleHistoricalPaths(t *testing.T) {
	// The first path through South is impossible: South is Ridge's parent
	// only after June, and South's UK link applies only before July.
	history := func(name string, parents ...glxlib.PlaceParentPeriod) *glxlib.Place {
		p := &glxlib.Place{Name: name}
		p.SetParentHistory(parents)

		return p
	}
	archive := censusTemporalArchive("1841", "1862")
	archive.Places = map[string]*glxlib.Place{
		"ridge":  history("Ridge", glxlib.PlaceParentPeriod{Value: "north", Date: "FROM 1841-01-01 TO 1841-06-30"}, glxlib.PlaceParentPeriod{Value: "south", Date: "FROM 1841-07-01"}),
		"north":  {Name: "North District", ParentID: "canada"},
		"south":  history("South District", glxlib.PlaceParentPeriod{Value: "uk", Date: "TO 1841-06-30"}, glxlib.PlaceParentPeriod{Value: "canada", Date: "FROM 1841-07-01 TO 1841-09-30"}, glxlib.PlaceParentPeriod{Value: "us", Date: "FROM 1841-10-01"}),
		"canada": {Name: "Canada", Type: glxlib.PlaceTypeCountry},
		"uk":     {Name: "United Kingdom", Type: glxlib.PlaceTypeCountry},
		"us":     {Name: "United States", Type: glxlib.PlaceTypeCountry},
	}
	archive.Events["death"].PlaceID = "" // Do not supply an independent US observation.
	for _, tc := range []struct {
		name, date string
		us, canada bool
	}{
		{name: "year spans two countries", date: "1841", us: true, canada: true},
		{name: "exact July event", date: "1841-07-01", canada: true},
		{name: "exact October event", date: "1841-10-01", us: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			archive.Events["birth"].Date = glxlib.DateString(tc.date)
			for command, leads := range censusTemporalLeads(t, archive) {
				require.Equal(t, tc.us, strings.Contains(leads, "1850 US census"), command)
				require.Equal(t, tc.canada, strings.Contains(leads, "1851 Canadian census"), command)
				require.NotContains(t, leads, "UK census", command)
			}
		})
	}
}

func TestCensusLossUsesObservationPrecision(t *testing.T) {
	// January observations were in Alexandria County (then part of DC).
	// The fictional July change provides a surviving alternative when all
	// we know is the birth year; it must not override an exact January date.
	archive := censusTemporalArchive("1807-01-01", "1813-01-01")
	archive.Places = map[string]*glxlib.Place{
		"ridge": {Name: "Alexandria County"},
		"dc":    {Name: "District of Columbia", Type: "territory", ParentID: "us"},
		"va":    {Name: "Virginia", Type: glxlib.PlaceTypeState, ParentID: "us"},
		"us":    {Name: "United States", Type: glxlib.PlaceTypeCountry},
	}
	archive.Places["ridge"].SetParentHistory([]glxlib.PlaceParentPeriod{
		{Value: "dc", Date: "TO 1807-06-30"},
		{Value: "va", Date: "FROM 1807-07-01 TO 1807-12-31"},
		{Value: "dc", Date: "FROM 1808 TO 1846"},
		{Value: "va", Date: "FROM 1847"},
	})

	for command, leads := range censusTemporalLeads(t, archive) {
		require.NotContains(t, leads, "1810 US census", command)
	}
	archive.Events["birth"].Date = "1807"
	for command, leads := range censusTemporalLeads(t, archive) {
		require.Contains(t, leads, "1810 US census", command)
	}
}

func TestCensusLossRetainsLeadAfterJurisdictionChanges(t *testing.T) {
	archive := censusTemporalArchive("1804-01-01", "1813-01-01")
	archive.Events["death"].PlaceID = ""
	archive.Places = map[string]*glxlib.Place{
		"ridge":    {Name: "Randolph County"},
		"indiana":  {Name: "Indiana Territory", Type: "territory", ParentID: "us"},
		"illinois": {Name: "Illinois Territory", Type: "territory", ParentID: "us"},
		"us":       {Name: "United States", Type: glxlib.PlaceTypeCountry},
	}
	randolph := archive.Places["ridge"]
	for _, tc := range []struct {
		name, newParent string
		retain          bool
	}{
		{name: "total loss in unchanged Indiana", newParent: "indiana"},
		{name: "partial loss in Illinois", newParent: "illinois", retain: true},
		{name: "unresolved jurisdiction in census year", retain: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parents := []glxlib.PlaceParentPeriod{{Value: "indiana", Date: "TO 1808"}}
			if tc.newParent != "" {
				parents = append(parents, glxlib.PlaceParentPeriod{Value: tc.newParent, Date: "FROM 1809"})
			}
			randolph.SetParentHistory(parents)
			for command, leads := range censusTemporalLeads(t, archive) {
				require.Equal(t, tc.retain, strings.Contains(leads, "1810 US census"), command)
			}
		})
	}
}

func censusTemporalArchive(birth, death glxlib.DateString) *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person": {}},
		Events: map[string]*glxlib.Event{
			"birth": {Type: glxlib.EventTypeBirth, Date: birth, PlaceID: "ridge", Participants: []glxlib.Participant{{Person: "person", Role: "subject"}}},
			"death": {Type: glxlib.EventTypeDeath, Date: death, PlaceID: "ridge", Participants: []glxlib.Participant{{Person: "person", Role: "subject"}}},
		},
	}
}

func censusTemporalLeads(t *testing.T, archive *glxlib.GLXFile) map[string]string {
	t.Helper()
	const person = "person"
	coverage, err := glxlib.BuildCoverage(archive, person, glxlib.CoverageOptions{})
	require.NoError(t, err)
	var coverageLeads []string
	for _, record := range coverage.Records {
		if record.Category == coverageCategoryCensus {
			coverageLeads = append(coverageLeads, strings.ReplaceAll(record.Label, "Census", "census"))
		}
	}
	issues := suggestCensusSearches(archive)
	analyzeLeads := make([]string, 0, len(issues))
	for i := range issues {
		analyzeLeads = append(analyzeLeads, issues[i].Message)
	}
	suggestions := suggestParentCensusRecords(person, archive.Persons[person], archive, nil)
	ancestorLeads := make([]string, 0, len(suggestions))
	for _, suggestion := range suggestions {
		ancestorLeads = append(ancestorLeads, suggestion.Message)
	}

	return map[string]string{
		"coverage":  strings.Join(coverageLeads, "\n"),
		"analyze":   strings.Join(analyzeLeads, "\n"),
		"ancestors": strings.Join(ancestorLeads, "\n"),
	}
}
