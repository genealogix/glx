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
// Note: some state census years overlap with federal census years (e.g.,
// Mississippi 1860). The matching logic requires a state-specific signal
// to avoid confusing state and federal censuses.
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
	if placeRef == "" || archive == nil {
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

// placeRefsFromProperty extracts place reference IDs from a property value
// using the shared collectPlaceRefsFromProperty helper from places_runner.go.
func placeRefsFromProperty(v any) []string {
	refs := make(map[string]struct{})
	collectPlaceRefsFromProperty(v, refs)
	var result []string
	for ref := range refs {
		result = append(result, ref)
	}
	return result
}

// collectPersonStates returns the unique US state names associated with a person
// via birthplace, death place, and event places.
func collectPersonStates(person *glxlib.Person, archive *glxlib.GLXFile, events []personSourceInfo) []string {
	stateSet := make(map[string]bool)

	// Check event places (includes birth and death events)
	for _, ev := range events {
		if ev.PlaceID == "" {
			continue
		}
		if s := resolveStateFromPlace(ev.PlaceID, archive); s != "" {
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
// the person's associated states and birth/death years. The archive parameter
// is used for place-based matching of events; pass nil if not available.
func buildStateCensusRecords(birthYear, deathYear int, states []string, sources []personSourceInfo, events []personSourceInfo, archive *glxlib.GLXFile) []coverageRecord {
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

			ref := findStateCensusMatch(year, state, sources, events, archive)
			if ref != "" {
				rec.Found = true
				rec.SourceRef = ref
			}

			records = append(records, rec)
		}
	}

	return records
}

// findStateCensusMatch checks if a state census for a given year and state
// titleMatchesState checks if a title contains the state name as a distinct
// word/phrase, avoiding substring false positives (e.g., "Kansas" in "Arkansas").
// Scans all occurrences and returns true if any is at a word boundary.
func titleMatchesState(title, state string) bool {
	lowerTitle := strings.ToLower(title)
	lowerState := strings.ToLower(state)

	// Scan all occurrences — return true if any is at a word boundary
	start := 0
	for {
		idx := strings.Index(lowerTitle[start:], lowerState)
		if idx < 0 {
			return false
		}
		idx += start
		end := idx + len(lowerState)
		atStart := idx == 0 || !isAlpha(lowerTitle[idx-1])
		atEnd := end >= len(lowerTitle) || !isAlpha(lowerTitle[end])
		if atStart && atEnd {
			return true
		}
		start = idx + 1
	}
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// findStateCensusMatch checks if a state census for a given year and state
// exists in events or sources. Requires a state-specific signal to avoid
// confusing state and federal censuses on overlapping years: the title must
// mention the state name, or the event's place must resolve to the target state.
func findStateCensusMatch(year int, state string, sources []personSourceInfo, events []personSourceInfo, archive *glxlib.GLXFile) string {
	// Check events — require census type + year + state-specific signal
	for _, e := range events {
		if e.EventType != glxlib.EventTypeCensus || e.Year != year {
			continue
		}
		// Title must mention this specific state
		if titleMatchesState(e.Title, state) {
			return e.Ref
		}
		// Place-based matching: resolve event place to this state
		if archive != nil && e.PlaceID != "" {
			if resolveStateFromPlace(e.PlaceID, archive) == state {
				return e.Ref
			}
		}
	}

	// Check sources — require census type + year + state-specific signal
	for _, s := range sources {
		if s.Type != glxlib.SourceTypeCensus {
			continue
		}
		// Title must mention this specific state (not just generic "state census")
		if s.Year == year && titleMatchesState(s.Title, state) {
			return s.Ref
		}
		// Fallback: title explicitly mentions both state and year
		if titleMatchesState(s.Title, state) && strings.Contains(s.Title, fmt.Sprintf("%d", year)) {
			return s.Ref
		}
	}

	return ""
}
