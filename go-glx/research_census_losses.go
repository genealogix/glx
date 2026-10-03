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
	"sort"
	"strings"
)

// censusJurisdictionLoss describes how one census year's schedules survive
// for one state or territory.
type censusJurisdictionLoss struct {
	// partial marks a jurisdiction whose schedules survive only for some
	// counties. A partial loss is shown as a note, never suppressed: the
	// person may have lived in a surviving county.
	partial bool
	// note is the human-readable annotation. For a total loss it is shown
	// only when the person may also have been somewhere whose schedules
	// survive; otherwise the census is not suggested at all.
	note string
}

// placeTypeTerritory is the place type of a territory (Indiana Territory,
// Arkansas Territory). The type is accepted here before every vocabulary
// carries it; a place of this type is a census jurisdiction just as a state
// is.
const placeTypeTerritory = "territory"

// censusLossSubstituteHint is appended to the note for a jurisdiction whose
// schedules are wholly lost.
const censusLossSubstituteHint = "try tax lists and state or territorial enumerations"

const (
	censusJurisdictionNewJersey   = "New Jersey"
	censusJurisdictionMississippi = "Mississippi"
)

// jurisdictionLosses builds one census year's loss entries: the
// jurisdictions whose schedules are wholly lost, and those whose schedules
// survive only for a few counties.
func jurisdictionLosses(lost, mostlyLost []string) map[string]censusJurisdictionLoss {
	losses := make(map[string]censusJurisdictionLoss, len(lost)+len(mostlyLost))
	for _, name := range lost {
		losses[jurisdictionKey(name)] = censusJurisdictionLoss{
			note: "schedules lost for " + name + "; " + censusLossSubstituteHint,
		}
	}
	for _, name := range mostlyLost {
		losses[jurisdictionKey(name)] = censusJurisdictionLoss{
			partial: true,
			note:    "schedules mostly lost for " + name + " (a few counties survive)",
		}
	}

	return losses
}

// usCensusLosses lists the early US federal census schedules lost for a whole
// state or territory (#1333). Each loss is listed under the name of the
// territory at the time and under the modern state names whose whole area it
// covered, because archives record both ("Indiana Territory", or "Indiana"
// for an 1810 event). Only well-established losses are listed, checked
// against NARA's "Guide to Genealogical Research in the National Archives"
// and Szucs and Luebking, "The Source" (census chapters):
//
//   - 1790: Delaware, Georgia, Kentucky, New Jersey, Tennessee (Southwest
//     Territory) and Virginia (with West Virginia) are lost. Virginia's
//     published "1790 census" is a reconstruction from state enumerations
//     and tax lists.
//   - 1800: Indiana Territory (modern Indiana and Illinois), Kentucky,
//     Mississippi Territory, New Jersey, Tennessee and Virginia are lost.
//     Georgia and the Northwest Territory (Ohio) survive only in part (a
//     single county each), so they are noted rather than suppressed.
//   - 1810: the District of Columbia, Georgia, Indiana Territory, Mississippi
//     Territory and New Jersey are lost. Ohio, Tennessee, Illinois Territory
//     and Michigan Territory survive only for one or a few counties.
//   - 1820: Arkansas Territory and Missouri are lost; Alabama survives only
//     for some counties.
//
// Losses confined to individual counties in otherwise surviving years are
// deliberately not listed: the table is keyed by state or territory.
//
//nolint:goconst // place names repeated across census years are table data, not shared logic
var usCensusLosses = map[int]map[string]censusJurisdictionLoss{
	1790: jurisdictionLosses(
		[]string{
			"Delaware", "Georgia", "Kentucky", censusJurisdictionNewJersey, "Tennessee",
			"Southwest Territory", "Virginia", "West Virginia",
		},
		nil,
	),
	1800: jurisdictionLosses(
		[]string{
			"Indiana Territory", "Indiana", "Illinois", "Kentucky",
			"Mississippi Territory", censusJurisdictionMississippi, censusJurisdictionNewJersey, "Tennessee",
			"Virginia", "West Virginia",
		},
		[]string{"Georgia", "Northwest Territory", "Ohio"},
	),
	1810: jurisdictionLosses(
		[]string{
			"District of Columbia", "Georgia", "Indiana Territory", "Indiana",
			"Mississippi Territory", censusJurisdictionMississippi, censusJurisdictionNewJersey,
		},
		[]string{
			"Ohio", "Tennessee", "Illinois Territory", "Illinois",
			"Michigan Territory", "Michigan",
		},
	),
	1820: jurisdictionLosses(
		[]string{"Arkansas Territory", "Arkansas", "Missouri Territory", "Missouri"},
		[]string{"Alabama"},
	),
}

// jurisdictionKey normalizes a state or territory name for lookup in a
// schedule's loss table: lower-cased, with whitespace collapsed.
func jurisdictionKey(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// resolveJurisdictionFromPlace walks the place hierarchy upward and returns
// the name and type of the first place typed `state` or `territory`. Both are
// empty when the walk reaches a country, or the root, before any state or
// territory.
func resolveJurisdictionFromPlace(placeRef string, archive *GLXFile) (jurisdiction, jurisdictionType string) {
	if placeRef == "" || archive == nil {
		return "", ""
	}

	visited := make(map[string]bool)
	current := placeRef

	for current != "" && !visited[current] {
		visited[current] = true
		place, ok := archive.Places[current]
		if !ok || place == nil || place.Type == PlaceTypeCountry {
			return "", ""
		}
		if place.Type == PlaceTypeState || place.Type == placeTypeTerritory {
			return place.Name, place.Type
		}
		current = place.ParentID
	}

	return "", ""
}

// CensusDatedPlace is one place a person is recorded at, with the year.
type CensusDatedPlace struct {
	Year    int
	PlaceID string
}

// eventDatedPlace returns the event's place and year, and false when the
// event lacks either.
func eventDatedPlace(event *Event) (CensusDatedPlace, bool) {
	if event == nil || event.PlaceID == "" {
		return CensusDatedPlace{}, false
	}
	year := ExtractFirstYear(string(event.Date))
	if year == 0 {
		return CensusDatedPlace{}, false
	}

	return CensusDatedPlace{Year: year, PlaceID: event.PlaceID}, true
}

// buildPersonDatedPlaceIndex returns, for every person, the dated places of
// the events they participate in. Undated events cannot say where someone
// was in a given census year and are left out.
func buildPersonDatedPlaceIndex(archive *GLXFile) map[string][]CensusDatedPlace {
	index := make(map[string][]CensusDatedPlace)
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		dp, ok := eventDatedPlace(event)
		if !ok {
			continue
		}
		for _, p := range event.Participants {
			if p.Person != "" {
				index[p.Person] = append(index[p.Person], dp)
			}
		}
	}

	return index
}

// datedPlacesForPerson returns one person's dated places. Callers looping
// over every person should build a buildPersonDatedPlaceIndex once instead.
func datedPlacesForPerson(archive *GLXFile, personID string) []CensusDatedPlace {
	var places []CensusDatedPlace
	for _, eventID := range sortedKeys(archive.Events) {
		event := archive.Events[eventID]
		dp, ok := eventDatedPlace(event)
		if !ok {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				places = append(places, dp)

				break
			}
		}
	}

	return places
}

// CensusYearSurvival is the outcome of checking one census year's schedules
// against where a person is recorded around that year.
type CensusYearSurvival struct {
	// Lost is true when every jurisdiction the person can be placed in
	// around the year lost its schedules outright. Such a census can never
	// be found, so it is neither suggested nor counted as missing.
	Lost bool
	// Note annotates a census that is partly lost for the person: lost or
	// mostly lost in some of the jurisdictions they may have been in.
	Note string
}

// Survival reports whether the schedules for year survive where the person
// was. The person's whereabouts are taken from their dated places in this
// schedule's country (or in no named country): the latest at or before the
// census year and the earliest at or after it, which bracket the census.
// Each of those places is resolved to its state or territory. The census is
// lost only when all of them are total losses; a place with no state or
// territory, or one whose schedules survive, keeps the census suggested.
func (s *CensusSchedule) Survival(year int, places []CensusDatedPlace, archive *GLXFile) CensusYearSurvival {
	if s == nil || archive == nil {
		return CensusYearSurvival{}
	}
	losses := s.losses[year]
	if len(losses) == 0 {
		return CensusYearSurvival{}
	}

	before, after := s.bracketingYears(year, places, archive)
	if before == 0 && after == 0 {
		return CensusYearSurvival{}
	}

	allLost := true
	seen := make(map[string]bool)
	var notes []string
	for _, dp := range places {
		if (dp.Year != before && dp.Year != after) || !s.placeInCountry(dp.PlaceID, archive) {
			continue
		}
		loss, known := lookupJurisdictionLoss(losses, dp.PlaceID, archive)
		if !known || loss.partial {
			allLost = false
		}
		if known && !seen[loss.note] {
			seen[loss.note] = true
			notes = append(notes, loss.note)
		}
	}
	if allLost {
		return CensusYearSurvival{Lost: true}
	}
	sort.Strings(notes)

	return CensusYearSurvival{Note: strings.Join(notes, "; ")}
}

// placeInCountry reports whether a place resolves to this schedule's country,
// or names no country at all (the schedule then applies through
// censusCountryFallback, and the place is still the best evidence of where
// the person was).
func (s *CensusSchedule) placeInCountry(placeID string, archive *GLXFile) bool {
	country := resolveCountryFromPlace(placeID, archive)

	return country == "" || canonicalCountry(country) == s.Country
}

// lookupJurisdictionLoss resolves a place to its state or territory and looks
// it up in one year's loss entries. A territory-typed place named without the
// word ("Indiana", type territory) also matches the "Indiana Territory" entry.
func lookupJurisdictionLoss(losses map[string]censusJurisdictionLoss, placeID string, archive *GLXFile) (censusJurisdictionLoss, bool) {
	name, placeType := resolveJurisdictionFromPlace(placeID, archive)
	if name == "" {
		return censusJurisdictionLoss{}, false
	}
	key := jurisdictionKey(name)
	if placeType == placeTypeTerritory && !strings.HasSuffix(key, " territory") {
		if loss, ok := losses[key+" territory"]; ok {
			return loss, true
		}
	}
	loss, ok := losses[key]

	return loss, ok
}

// bracketingYears finds the nearest dated locations in this schedule's country.
func (s *CensusSchedule) bracketingYears(year int, places []CensusDatedPlace, archive *GLXFile) (before, after int) {
	for _, dp := range places {
		if !s.placeInCountry(dp.PlaceID, archive) {
			continue
		}
		if dp.Year <= year && dp.Year > before {
			before = dp.Year
		}
		if dp.Year >= year && (after == 0 || dp.Year < after) {
			after = dp.Year
		}
	}

	return before, after
}
