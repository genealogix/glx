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
	"fmt"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// stateCensusYears maps US state names to their known state census years.
var stateCensusYears = map[string][]int{
	"Wisconsin":    {1855, 1865, 1875, 1885, 1895, 1905},
	"New York":     {1825, 1835, 1845, 1855, 1865, 1875, 1892, 1905, 1915, 1925},
	"Iowa":         {1856, 1885, 1895, 1905, 1915, 1925},
	"Massachusetts": {1855, 1865},
	"New Jersey":   {1855, 1865, 1875, 1885, 1895, 1905, 1915},
	"Minnesota":    {1857, 1865, 1875, 1885, 1895, 1905},
	"Kansas":       {1855, 1865, 1875, 1885, 1895, 1905, 1915, 1925},
	"Rhode Island": {1865, 1875, 1885, 1905, 1915, 1925, 1935},
	"Mississippi":  {1853, 1860, 1866},
	"Colorado":     {1885},
	"Florida":      {1855, 1867, 1885, 1895, 1935, 1945},
	"Nebraska":     {1885},
	"North Dakota": {1885, 1915, 1925},
	"South Dakota": {1885, 1895, 1905, 1915, 1925, 1935, 1945},
	"Oregon":       {1845, 1849, 1853, 1855, 1856, 1857, 1858, 1859, 1865, 1875, 1885, 1895, 1905},
}

// resolveStateFromPlace walks the place hierarchy to find the US state name.
// Returns the empty string if no state-type ancestor is found.
func resolveStateFromPlace(placeRef string, archive *glxlib.GLXFile) string {
	if placeRef == "" {
		return ""
	}

	visited := make(map[string]bool)
	current := placeRef

	for current != "" && !visited[current] {
		visited[current] = true
		place, ok := archive.Places[current]
		if !ok || place == nil {
			return ""
		}
		if place.Type == glxlib.PlaceTypeState {
			return place.Name
		}
		current = place.ParentID
	}

	return ""
}

// collectPersonStates returns the unique US state names associated with a person
// via birthplace, death place, and event places.
func collectPersonStates(personID string, person *glxlib.Person, archive *glxlib.GLXFile, events []personSourceInfo) []string {
	stateSet := make(map[string]bool)

	if person != nil {
		bornAt := propertyString(person.Properties, glxlib.PersonPropertyBornAt)
		if s := resolveStateFromPlace(bornAt, archive); s != "" {
			stateSet[s] = true
		}
		diedAt := propertyString(person.Properties, glxlib.PersonPropertyDiedAt)
		if s := resolveStateFromPlace(diedAt, archive); s != "" {
			stateSet[s] = true
		}
	}

	// Check event places
	for _, ev := range events {
		if ev.Ref == "" {
			continue
		}
		event, ok := archive.Events[ev.Ref]
		if !ok || event == nil || event.PlaceID == "" {
			continue
		}
		if s := resolveStateFromPlace(event.PlaceID, archive); s != "" {
			stateSet[s] = true
		}
	}

	var states []string
	for s := range stateSet {
		states = append(states, s)
	}
	sort.Strings(states)
	return states
}

// buildStateCensusRecords generates expected state census records based on
// the person's associated states and birth/death years.
func buildStateCensusRecords(birthYear, deathYear int, states []string, sources []personSourceInfo, events []personSourceInfo) []coverageRecord {
	if birthYear == 0 {
		return nil
	}

	upperBound := deathYear
	if upperBound == 0 {
		upperBound = birthYear + maxLifespan
	}

	var records []coverageRecord

	for _, state := range states {
		years, ok := stateCensusYears[state]
		if !ok {
			continue
		}
		for _, year := range years {
			if year < birthYear || year > upperBound {
				continue
			}
			age := year - birthYear
			label := fmt.Sprintf("%d %s State Census (age ~%d)", year, state, age)

			rec := coverageRecord{
				Category: "census",
				Label:    label,
			}

			ref := findStateCensusMatch(year, state, sources, events)
			if ref != "" {
				rec.Found = true
				rec.SourceRef = ref
			}

			records = append(records, rec)
		}
	}

	return records
}

// findStateCensusMatch checks if a state census for a given year exists.
// State censuses can appear as census events or sources with matching years.
// Since state census years don't overlap with federal census years, a census
// event matching the year is a strong indicator.
func findStateCensusMatch(year int, state string, sources []personSourceInfo, events []personSourceInfo) string {
	lowerState := strings.ToLower(state)

	// Check events first — a census event at this year is likely the state census
	for _, e := range events {
		if e.EventType == glxlib.EventTypeCensus && e.Year == year {
			return e.Ref
		}
	}

	// Check sources — look for census source with matching year or state name
	for _, s := range sources {
		if s.Type == glxlib.SourceTypeCensus {
			if s.Year == year {
				return s.Ref
			}
			if strings.Contains(strings.ToLower(s.Title), lowerState) &&
				strings.Contains(s.Title, fmt.Sprintf("%d", year)) {
				return s.Ref
			}
		}
	}

	return ""
}
