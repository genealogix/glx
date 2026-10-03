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

	glxlib "github.com/genealogix/glx/go-glx"
)

const (
	countryUnitedStates  = glxlib.CensusCountryUnitedStates
	countryUnitedKingdom = glxlib.CensusCountryUnitedKingdom
	countryCanada        = glxlib.CensusCountryCanada
	countryIreland       = glxlib.CensusCountryIreland
	maxLifespan          = 100
	censusPrimeAgeMax    = 25
)

type (
	censusSchedule = glxlib.CensusSchedule
)

// censusCountryFlagUsage is the --country flag help shared by the commands
// that suggest census records.
const censusCountryFlagUsage = "Country whose census schedule to assume for persons " +
	"whose places name no country (default \"none\", which suggests no census records)"

// censusCountryNone is the --country value that turns census suggestions off
// for persons whose places do not name a country.
const censusCountryNone = "none"

// censusCountryFallback is the country assumed for a person no place in the
// archive puts in one. It is empty by default, so an archive that names no
// country for someone draws no census suggestions at all: guessing the United
// States from silence is the US-centric default #186 was filed about. Set it
// with `--country` to research an archive whose places stop short of the
// country level (#186).
var censusCountryFallback = ""

// applyCensusCountry validates a --country flag value and records it as the
// fallback census country for this run.
func applyCensusCountry(value string) error {
	country, err := parseCensusCountry(value)
	if err != nil {
		return err
	}
	censusCountryFallback = country

	return nil
}

func parseCensusCountry(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return censusCountryFallback, nil
	}

	return glxlib.CensusCountry(value)
}
func canonicalCountry(name string) string { return glxlib.CanonicalCountry(name) }
func resolveCountryFromPlace(placeRef string, archive *glxlib.GLXFile) string {
	return glxlib.PlaceCountry(archive, placeRef)
}

func censusSchedulesForPlaces(refs []string, archive *glxlib.GLXFile) []*censusSchedule {
	// applyCensusCountry validates this CLI-only flag state before command work.
	schedules, _ := glxlib.CensusSchedulesForPlaces(archive, refs, glxlib.CoverageOptions{CensusCountry: censusCountryFallback})

	return schedules
}

func censusSchedulesForPerson(archive *glxlib.GLXFile, personID string) []*censusSchedule {
	schedules, _ := glxlib.CensusSchedulesForPerson(archive, personID, glxlib.CoverageOptions{CensusCountry: censusCountryFallback})

	return schedules
}
func allCensusYears() []int { return glxlib.CensusYears() }
func buildPersonPlaceIndex(archive *glxlib.GLXFile) map[string][]string {
	index, _ := glxlib.PersonPlaceIndex(archive)

	return index
}

func scheduleForCountry(schedules []*censusSchedule, country string) *censusSchedule {
	for _, schedule := range schedules {
		if schedule.Country == country {
			return schedule
		}
	}

	return nil
}
