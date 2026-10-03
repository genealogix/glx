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
	"maps"
	"slices"
)

// CensusCountry resolves a supported census country or alias. Empty and "none"
// select no fallback. Unknown countries return ErrUnknownCensusCountry.
func CensusCountry(value string) (string, error) { return parseCensusCountry(value) }

// CensusCountries lists supported canonical country names in sorted order.
func CensusCountries() []string { return knownCensusCountries() }

// CensusSchedulesForPlaces returns owned schedule snapshots for the countries
// identified by placeRefs, using opts only when no country is identified.
func CensusSchedulesForPlaces(archive *GLXFile, placeRefs []string, opts CoverageOptions) ([]*CensusSchedule, error) {
	if archive == nil {
		return nil, ErrNilArchive
	}
	country, err := parseCensusCountry(opts.CensusCountry)
	if err != nil {
		return nil, err
	}

	return cloneCensusSchedules(censusSchedulesForPlaces(placeRefs, archive, country)), nil
}

// CensusSchedulesForPerson returns schedules for an exact person ID.
func CensusSchedulesForPerson(archive *GLXFile, personID string, opts CoverageOptions) ([]*CensusSchedule, error) {
	if archive == nil {
		return nil, ErrNilArchive
	}
	if err := requirePerson(archive, personID, "person"); err != nil {
		return nil, err
	}
	country, err := parseCensusCountry(opts.CensusCountry)
	if err != nil {
		return nil, err
	}

	return cloneCensusSchedules(censusSchedulesForPerson(archive, personID, country)), nil
}

func cloneCensusSchedules(schedules []*CensusSchedule) []*CensusSchedule {
	out := make([]*CensusSchedule, 0, len(schedules))
	for _, schedule := range schedules {
		snapshot := *schedule
		snapshot.Years = slices.Clone(schedule.Years)
		snapshot.Notes = maps.Clone(schedule.Notes)
		out = append(out, &snapshot)
	}

	return out
}

// CensusYears returns every listed census year in sorted order.
func CensusYears() []int { return allCensusYears() }

// PersonPlaceIndex maps participant IDs to their event places. Returned maps and
// slices are owned by the caller and sorted independently of archive map order.
func PersonPlaceIndex(archive *GLXFile) (map[string][]string, error) {
	if archive == nil {
		return nil, ErrNilArchive
	}
	index := buildPersonPlaceIndex(archive)
	for _, refs := range index {
		slices.Sort(refs)
	}

	return index, nil
}

// PlaceCountry returns the nearest country ancestor's name, or empty when absent.
func PlaceCountry(archive *GLXFile, placeID string) string {
	if archive == nil {
		return ""
	}

	return resolveCountryFromPlace(placeID, archive)
}

// CanonicalCountry recognizes a free-text country name without requiring that
// a census schedule exists. Unrecognized names return empty.
func CanonicalCountry(name string) string { return canonicalCountry(name) }
