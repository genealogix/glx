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
	"fmt"
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

// usCensusLosses records total and extensive partial losses of early US
// federal census schedules (#1333). Historical territory names and retained
// modern-state aliases support archives that use either name; the lookup
// does not reconstruct historical boundaries. Primary record guidance:
//
//   - 1790: Delaware, Georgia, Kentucky, New Jersey, Tennessee (Southwest
//     Territory) and Virginia (with West Virginia) are lost. Virginia's
//     published "1790 census" is a reconstruction from state enumerations
//     and tax lists.
//     https://www.archives.gov/research/census/1790
//   - 1800: Georgia, Kentucky, Mississippi Territory, New Jersey, Tennessee
//     and Virginia are listed as missing by NARA. Washington County's federal
//     schedules survive for the Northwest Territory (Ohio); M1804 also
//     contains a separate 1803 Ohio enumeration. Georgia's surviving 1800
//     Oglethorpe record is a state census, not a federal census exception.
//     https://www.archives.gov/research/census/1800
//     https://www.georgiaarchives.org/research/census_records
//   - 1810: the District of Columbia, Georgia, Indiana Territory, Mississippi
//     Territory, Michigan Territory, New Jersey and Ohio are missing.
//     Tennessee retains Rutherford County; Illinois Territory retains
//     Randolph County (NARA provides a later typescript). These are partial
//     exceptions, unlike Ohio and Michigan Territory.
//     https://www.archives.gov/research/census/1810
//   - 1820: Arkansas Territory, Missouri Territory and New Jersey are missing.
//     Alabama is unclassified: NARA's FAQ lists it neither among surviving
//     states nor among whole-state losses. That omission establishes neither
//     survival nor loss; state censuses cannot establish federal survival.
//     https://www.archives.gov/research/census/1820
//
// Losses confined to individual counties in otherwise surviving years are
// deliberately not listed: the table is keyed by state or territory. An
// absent entry means no established loss here, not proven full survival.
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
			"Georgia", "Indiana Territory", "Indiana", "Illinois", "Kentucky",
			"Mississippi Territory", censusJurisdictionMississippi, censusJurisdictionNewJersey, "Tennessee",
			"Virginia", "West Virginia",
		},
		[]string{"Northwest Territory", "Ohio"},
	),
	1810: jurisdictionLosses(
		[]string{
			"District of Columbia", "Georgia", "Indiana Territory", "Indiana",
			"Michigan Territory", "Michigan", "Ohio",
			"Mississippi Territory", censusJurisdictionMississippi, censusJurisdictionNewJersey,
		},
		[]string{
			"Tennessee", "Illinois Territory", "Illinois",
		},
	),
	1820: jurisdictionLosses(
		[]string{"Arkansas Territory", "Arkansas", "Missouri Territory", "Missouri", censusJurisdictionNewJersey},
		nil,
	),
}

// jurisdictionKey normalizes a state or territory name for lookup in a
// schedule's loss table: lower-cased, with whitespace collapsed.
func jurisdictionKey(name string) string {
	return strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// jurisdictionOnPath finds the nearest state or territory before the country.
func jurisdictionOnPath(path []string, archive *GLXFile) (name, placeType string) {
	for _, id := range path {
		p := archive.Places[id]
		if p == nil || p.Type == PlaceTypeCountry {
			break
		}
		if p.Type == PlaceTypeState || p.Type == placeTypeTerritory {
			return p.Name, p.Type
		}
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
	// around the year has a documented total loss. A missing census is then
	// neither suggested nor counted as missing; an already found record is
	// retained by coverage.
	Lost bool
	// Note annotates a census that is partly lost for the person: lost or
	// mostly lost in some of the jurisdictions they may have been in.
	Note string
}

// Survival checks where the person is recorded when only the observation
// year is known. Use SurvivalAt to preserve more precise observation dates.
func (s *CensusSchedule) Survival(year int, places []CensusDatedPlace, archive *GLXFile) CensusYearSurvival {
	if s == nil || archive == nil || len(s.losses[year]) == 0 {
		return CensusYearSurvival{}
	}
	obs := make([]CensusPlaceObservation, 0, len(places))
	for _, p := range places {
		if p.Year > 0 {
			obs = append(obs, CensusPlaceObservation{PlaceID: p.PlaceID, Date: censusYearDate(p.Year)})
		}
	}

	return s.SurvivalAt(year, obs, archive)
}

// SurvivalAt checks the latest observations at or before a census year and
// the earliest at or after it on the schedule's side. For each location, both
// its recorded jurisdiction and its jurisdiction in the census year must be
// known total losses to omit the lead. A change in the same place's hierarchy
// can keep a lead open but cannot establish the person's census-day residence.
func (s *CensusSchedule) SurvivalAt(year int, places []CensusPlaceObservation, archive *GLXFile) CensusYearSurvival {
	if s == nil || archive == nil {
		return CensusYearSurvival{}
	}
	losses := s.losses[year]
	if len(losses) == 0 {
		return CensusYearSurvival{}
	}

	var before, after censusLossBracket
	censusDate := censusYearDate(year)
	for _, p := range places {
		observedYear := ExtractFirstYear(string(p.Date))
		if observedYear <= 0 || (observedYear < before.year || (after.year != 0 && observedYear > after.year)) {
			continue
		}
		local, foreign := s.locationLosses(losses, p.PlaceID, p.Date, archive)
		if len(local) == 0 {
			continue // Observation is known to be in a different country.
		}
		if p.Date != censusDate {
			// Keep the observation date's uncertainty: a path that was
			// foreign then still means the person may have left.
			projected, projectedForeign := s.locationLosses(losses, p.PlaceID, censusDate, archive)
			local = append(local, projected...)
			foreign = foreign || projectedForeign
		}
		if foreign {
			// A changed country does not prove this schedule's records were lost.
			local = append(local, censusJurisdictionLoss{})
		}
		if observedYear <= year {
			before.add(observedYear, local)
		}
		if observedYear >= year {
			after.add(observedYear, local)
		}
	}
	if before.year == 0 && after.year == 0 {
		return CensusYearSurvival{}
	}

	return summarizeCensusLoss(before, after)
}

func summarizeCensusLoss(brackets ...censusLossBracket) CensusYearSurvival {
	allLost := true
	seen := make(map[string]bool)
	var notes []string
	for _, bracket := range brackets {
		for _, loss := range bracket.losses {
			if loss.note == "" || loss.partial {
				allLost = false
			}
			if loss.note != "" && !seen[loss.note] {
				seen[loss.note] = true
				notes = append(notes, loss.note)
			}
		}
	}
	if allLost {
		return CensusYearSurvival{Lost: true}
	}
	sort.Strings(notes)

	return CensusYearSurvival{Note: strings.Join(notes, "; ")}
}

type censusLossBracket struct {
	year   int
	losses []censusJurisdictionLoss
}

func (b *censusLossBracket) add(year int, losses []censusJurisdictionLoss) {
	if year != b.year {
		b.year = year
		b.losses = nil
	}
	b.losses = append(b.losses, losses...)
}

func censusYearDate(year int) DateString {
	return DateString(fmt.Sprintf("%04d", year))
}

func (s *CensusSchedule) locationLosses(losses map[string]censusJurisdictionLoss, placeID string, date DateString, archive *GLXFile) (local []censusJurisdictionLoss, foreign bool) {
	paths := archive.PlacePathsAt(placeID, date)
	if len(paths) == 0 {
		return []censusJurisdictionLoss{{}}, false // Unresolved location isn't evidence of loss.
	}

	for _, path := range paths {
		country := countryOnPath(path, archive)
		if country != "" && canonicalCountry(country) != s.Country {
			foreign = true

			continue
		}
		name, placeType := jurisdictionOnPath(path, archive)
		local = append(local, lookupJurisdictionLoss(losses, name, placeType))
	}

	return local, foreign
}

// lookupJurisdictionLoss also accepts a territory-typed name without the
// suffix, such as "Indiana" rendered as "Indiana Territory" in the table.
func lookupJurisdictionLoss(losses map[string]censusJurisdictionLoss, name, placeType string) censusJurisdictionLoss {
	if name == "" {
		return censusJurisdictionLoss{}
	}
	key := jurisdictionKey(name)
	if placeType == placeTypeTerritory && !strings.HasSuffix(key, " territory") {
		if loss, ok := losses[key+" territory"]; ok {
			return loss
		}
	}

	return losses[key]
}
