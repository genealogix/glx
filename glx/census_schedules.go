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

// Canonical country names. These are the keys of censusSchedulesByCountry;
// the many spellings archives actually contain are mapped onto them by
// countryAliases.
const (
	countryUnitedStates  = "United States"
	countryUnitedKingdom = "United Kingdom"
	countryCanada        = "Canada"
	countryIreland       = "Ireland"
)

// censusYearNote carries the research annotations attached to one census year.
type censusYearNote struct {
	// note is appended to the record description for every person.
	note string
	// minorNote is appended when the person was under minorAgeUnder at the
	// census year.
	minorNote string
	// highPriority marks the year as high research priority when missing,
	// regardless of the person's age.
	highPriority bool
}

// censusSchedule is one country's national census schedule: the years a
// researcher can realistically search, plus the per-year notes shown
// alongside them.
type censusSchedule struct {
	// country is the canonical country name this schedule belongs to.
	country string
	// adjective is the country word used in labels — "US" in
	// "1880 US Census", "UK" in "1841 UK Census".
	adjective string
	// years are the census years, ascending.
	years []int
	// notes are the per-year research annotations, keyed by year.
	notes map[int]censusYearNote
	// losses are the census years whose schedules are lost for part of the
	// country, keyed by year and then by normalized state or territory name
	// (see jurisdictionKey). A year absent from the map survives wherever
	// the national entry says it does.
	losses map[int]map[string]censusJurisdictionLoss
}

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

// coverageLabel renders the coverage-checklist label for one census year,
// e.g. "1880 US Census (age ~30)".
func (s *censusSchedule) coverageLabel(year, age int) string {
	return fmt.Sprintf("%d %s Census (age ~%d)", year, s.adjective, age)
}

// suggestionLabel renders the analyze wording for one census year,
// e.g. "1880 US census".
func (s *censusSchedule) suggestionLabel(year int) string {
	return fmt.Sprintf("%d %s census", year, s.adjective)
}

// censusSchedulesByCountry holds the national census schedules GLX suggests
// records from. A year is listed when returns naming individuals survive and
// are open to researchers — a year a country never enumerated, or whose
// returns were destroyed outright, is not a record anyone can be asked to
// look for. The one exception is a year whose loss is itself worth telling a
// researcher about (US 1890), which is listed with a note saying so.
// Schedules lost for only part of a country are recorded per state or
// territory in the schedule's losses table (#1333).
//
// The table is deliberately small: adding a country is adding a claim about
// that country's archives, so each entry should come with a source. Countries
// absent from the table simply produce no census suggestions, which is the
// correct answer until someone contributes the schedule.
var censusSchedulesByCountry = map[string]*censusSchedule{
	// US federal censuses, 1790 onward. 1960 and later are closed by the
	// 72-year rule.
	countryUnitedStates: {
		country:   countryUnitedStates,
		adjective: "US",
		years: []int{
			1790, 1800, 1810, 1820, 1830, 1840, 1850, 1860, 1870, 1880,
			1890, 1900, 1910, 1920, 1930, 1940, 1950,
		},
		notes: map[int]censusYearNote{
			1850: {
				note:      "first census to list individual names",
				minorNote: "likely in parents' household",
			},
			1880: {
				note:         "first census to list parents' birthplaces",
				highPriority: true,
			},
			1890: {note: "mostly destroyed (1921 fire)"},
		},
		losses: usCensusLosses,
	},
	// UK censuses, 1841 onward. The 1801-1831 counts recorded no names,
	// the 1931 England and Wales returns burned in 1942, and no census was
	// taken in 1941.
	countryUnitedKingdom: {
		country:   countryUnitedKingdom,
		adjective: "UK",
		years:     []int{1841, 1851, 1861, 1871, 1881, 1891, 1901, 1911, 1921},
		notes: map[int]censusYearNote{
			1841: {note: "ages over 15 rounded down; birthplace only recorded as in-county or not"},
			1911: {note: "first census whose original household schedules survive"},
		},
	},
	// Canadian censuses, 1851 onward. 1941 and later are not yet open.
	countryCanada: {
		country:   countryCanada,
		adjective: "Canadian",
		years:     []int{1851, 1861, 1871, 1881, 1891, 1901, 1911, 1921, 1931},
	},
	// Irish censuses. Only 1901 and 1911 survive intact: the 1821-1851
	// returns were largely destroyed in the 1922 Four Courts fire and the
	// 1861-1891 returns were pulped during the First World War.
	countryIreland: {
		country:   countryIreland,
		adjective: "Irish",
		years:     []int{1901, 1911},
	},
}

// countryAliases maps the country spellings archives actually contain onto
// the canonical names censusSchedulesByCountry is keyed by. Keys are
// lower-cased and whitespace-normalized; canonicalCountry normalizes the
// place name the same way before looking it up.
//
// Constituent countries map to the state that ran the census: Scotland and
// Northern Ireland are listed under the United Kingdom because their returns
// are UK census returns, taken in the same years.
var countryAliases = map[string]string{
	"united states":            countryUnitedStates,
	"united states of america": countryUnitedStates,
	"usa":                      countryUnitedStates,
	"u.s.a.":                   countryUnitedStates,
	"us":                       countryUnitedStates,
	"u.s.":                     countryUnitedStates,
	"united kingdom":           countryUnitedKingdom,
	"united kingdom of great britain and ireland":          countryUnitedKingdom,
	"united kingdom of great britain and northern ireland": countryUnitedKingdom,
	"uk":                  countryUnitedKingdom,
	"u.k.":                countryUnitedKingdom,
	"great britain":       countryUnitedKingdom,
	"britain":             countryUnitedKingdom,
	"england":             countryUnitedKingdom,
	"england and wales":   countryUnitedKingdom,
	"wales":               countryUnitedKingdom,
	"scotland":            countryUnitedKingdom,
	"northern ireland":    countryUnitedKingdom,
	"canada":              countryCanada,
	"dominion of canada":  countryCanada,
	"ireland":             countryIreland,
	"republic of ireland": countryIreland,
	"irish free state":    countryIreland,
	"eire":                countryIreland,
	"éire":                countryIreland,
}

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

// parseCensusCountry resolves a --country flag value to a canonical country
// name. The empty string leaves the fallback alone (no fallback, unless one
// has already been set), "none" disables it, and anything else must name a
// country the schedule table knows about.
func parseCensusCountry(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return censusCountryFallback, nil
	}
	if strings.EqualFold(trimmed, censusCountryNone) {
		return "", nil
	}

	country := canonicalCountry(trimmed)
	if country == "" || censusSchedulesByCountry[country] == nil {
		return "", fmt.Errorf("%w %q (known: %s, or %q to disable)",
			ErrUnknownCensusCountry, value, strings.Join(knownCensusCountries(), ", "), censusCountryNone)
	}

	return country, nil
}

// knownCensusCountries returns the canonical names of every country with a
// census schedule, sorted for stable help and error text.
func knownCensusCountries() []string {
	names := make([]string, 0, len(censusSchedulesByCountry))
	for name := range censusSchedulesByCountry {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// canonicalCountry maps a free-text country name onto a canonical country
// name, or returns the empty string when the name is not one we recognize.
// Place names are free text in GLX, so an unrecognized name is expected: it
// means "somewhere we have no census schedule for", not "somewhere invalid".
func canonicalCountry(name string) string {
	key := strings.Join(strings.Fields(strings.ToLower(name)), " ")
	if key == "" {
		return ""
	}

	return countryAliases[key]
}

// resolveCountryFromPlace walks the place hierarchy upward and returns the
// name of the first country-type ancestor, or the empty string when the chain
// reaches its root without one. Only places explicitly typed `country` count,
// so a hierarchy that stops at a state or a city resolves to nothing rather
// than guessing from the root place's name.
func resolveCountryFromPlace(placeRef string, archive *glxlib.GLXFile) string {
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
		if place.Type == glxlib.PlaceTypeCountry {
			return place.Name
		}
		current = place.ParentID
	}

	return ""
}

// censusSchedulesForPlaces returns the census schedules that apply to a
// person, given the places their events happened in.
//
// A person whose places resolve to countries gets the schedule of every one
// of those countries that the table knows about — an emigrant enumerated on
// both sides of a move should be asked about both. A person whose places
// resolve to a country with no schedule (or to no country GLX recognizes)
// gets nothing: suggesting US census years for someone who never left
// Mecklenburg is worse than suggesting nothing. Only when no place puts the
// person in any country at all does censusCountryFallback apply.
func censusSchedulesForPlaces(placeRefs []string, archive *glxlib.GLXFile) []*censusSchedule {
	countries := make(map[string]bool)
	sawCountry := false

	for _, ref := range placeRefs {
		name := resolveCountryFromPlace(ref, archive)
		if name == "" {
			continue
		}
		sawCountry = true
		if canonical := canonicalCountry(name); canonical != "" {
			countries[canonical] = true
		}
	}

	if !sawCountry {
		if censusCountryFallback == "" {
			return nil
		}
		countries[censusCountryFallback] = true
	}

	names := make([]string, 0, len(countries))
	for name := range countries {
		names = append(names, name)
	}
	sort.Strings(names)

	schedules := make([]*censusSchedule, 0, len(names))
	for _, name := range names {
		if schedule := censusSchedulesByCountry[name]; schedule != nil {
			schedules = append(schedules, schedule)
		}
	}

	return schedules
}

// allCensusYears returns every year named by any schedule, ascending. It is
// used when scanning source titles for a census year, where the point is only
// to recognize that a title refers to some census — narrowing that scan to
// one country's schedule would make an already-cited record look missing.
func allCensusYears() []int {
	seen := make(map[int]bool)
	var years []int
	for _, schedule := range censusSchedulesByCountry {
		for _, year := range schedule.years {
			if !seen[year] {
				seen[year] = true
				years = append(years, year)
			}
		}
	}
	sort.Ints(years)

	return years
}

// buildPersonPlaceIndex returns, for every person, the place references of
// the events they participate in. Callers that need schedules for the whole
// archive build this once rather than rescanning the events per person.
func buildPersonPlaceIndex(archive *glxlib.GLXFile) map[string][]string {
	index := make(map[string][]string)
	for _, event := range archive.Events {
		if event == nil || event.PlaceID == "" {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" {
				continue
			}
			index[p.Person] = append(index[p.Person], event.PlaceID)
		}
	}

	return index
}

// censusSchedulesForPerson returns the census schedules that apply to one
// person. Callers looping over every person should build a
// buildPersonPlaceIndex once and call censusSchedulesForPlaces instead.
func censusSchedulesForPerson(archive *glxlib.GLXFile, personID string) []*censusSchedule {
	var placeRefs []string
	for _, event := range archive.Events {
		if event == nil || event.PlaceID == "" {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == personID {
				placeRefs = append(placeRefs, event.PlaceID)

				break
			}
		}
	}

	return censusSchedulesForPlaces(placeRefs, archive)
}

// scheduleForCountry returns the schedule for a canonical country name, or
// nil when the country has none.
func scheduleForCountry(schedules []*censusSchedule, country string) *censusSchedule {
	for _, schedule := range schedules {
		if schedule.country == country {
			return schedule
		}
	}

	return nil
}

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

// placeTypeTerritory is the place type of a territory (Indiana Territory,
// Arkansas Territory). The type is accepted here before every vocabulary
// carries it; a place of this type is a census jurisdiction just as a state
// is.
const placeTypeTerritory = "territory"

// censusLossSubstituteHint is appended to the note for a jurisdiction whose
// schedules are wholly lost.
const censusLossSubstituteHint = "try tax lists and state or territorial enumerations"

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
			"Delaware", "Georgia", "Kentucky", "New Jersey", "Tennessee",
			"Southwest Territory", "Virginia", "West Virginia",
		},
		nil,
	),
	1800: jurisdictionLosses(
		[]string{
			"Indiana Territory", "Indiana", "Illinois", "Kentucky",
			"Mississippi Territory", "Mississippi", "New Jersey", "Tennessee",
			"Virginia", "West Virginia",
		},
		[]string{"Georgia", "Northwest Territory", "Ohio"},
	),
	1810: jurisdictionLosses(
		[]string{
			"District of Columbia", "Georgia", "Indiana Territory", "Indiana",
			"Mississippi Territory", "Mississippi", "New Jersey",
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
func resolveJurisdictionFromPlace(placeRef string, archive *glxlib.GLXFile) (jurisdiction, jurisdictionType string) {
	if placeRef == "" || archive == nil {
		return "", ""
	}

	visited := make(map[string]bool)
	current := placeRef

	for current != "" && !visited[current] {
		visited[current] = true
		place, ok := archive.Places[current]
		if !ok || place == nil || place.Type == glxlib.PlaceTypeCountry {
			return "", ""
		}
		if place.Type == glxlib.PlaceTypeState || place.Type == placeTypeTerritory {
			return place.Name, place.Type
		}
		current = place.ParentID
	}

	return "", ""
}

// datedPlace is one place a person is recorded at, with the year.
type datedPlace struct {
	year    int
	placeID string
}

// eventDatedPlace returns the event's place and year, and false when the
// event lacks either.
func eventDatedPlace(event *glxlib.Event) (datedPlace, bool) {
	if event == nil || event.PlaceID == "" {
		return datedPlace{}, false
	}
	year := glxlib.ExtractFirstYear(string(event.Date))
	if year == 0 {
		return datedPlace{}, false
	}

	return datedPlace{year: year, placeID: event.PlaceID}, true
}

// buildPersonDatedPlaceIndex returns, for every person, the dated places of
// the events they participate in. Undated events cannot say where someone
// was in a given census year and are left out.
func buildPersonDatedPlaceIndex(archive *glxlib.GLXFile) map[string][]datedPlace {
	index := make(map[string][]datedPlace)
	for _, event := range archive.Events {
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
func datedPlacesForPerson(archive *glxlib.GLXFile, personID string) []datedPlace {
	var places []datedPlace
	for _, event := range archive.Events {
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

// censusYearSurvival is the outcome of checking one census year's schedules
// against where a person is recorded around that year.
type censusYearSurvival struct {
	// lost is true when every jurisdiction the person can be placed in
	// around the year lost its schedules outright. Such a census can never
	// be found, so it is neither suggested nor counted as missing.
	lost bool
	// note annotates a census that is partly lost for the person: lost or
	// mostly lost in some of the jurisdictions they may have been in.
	note string
}

// survival reports whether the schedules for year survive where the person
// was. The person's whereabouts are taken from their dated places in this
// schedule's country (or in no named country): the latest at or before the
// census year and the earliest at or after it, which bracket the census.
// Each of those places is resolved to its state or territory. The census is
// lost only when all of them are total losses; a place with no state or
// territory, or one whose schedules survive, keeps the census suggested.
func (s *censusSchedule) survival(year int, places []datedPlace, archive *glxlib.GLXFile) censusYearSurvival {
	losses := s.losses[year]
	if len(losses) == 0 {
		return censusYearSurvival{}
	}

	before, after := 0, 0
	for _, dp := range places {
		if !s.placeInCountry(dp.placeID, archive) {
			continue
		}
		if dp.year <= year && dp.year > before {
			before = dp.year
		}
		if dp.year >= year && (after == 0 || dp.year < after) {
			after = dp.year
		}
	}
	if before == 0 && after == 0 {
		return censusYearSurvival{}
	}

	allLost := true
	seen := make(map[string]bool)
	var notes []string
	for _, dp := range places {
		if (dp.year != before && dp.year != after) || !s.placeInCountry(dp.placeID, archive) {
			continue
		}
		loss, known := lookupJurisdictionLoss(losses, dp.placeID, archive)
		if !known || loss.partial {
			allLost = false
		}
		if known && !seen[loss.note] {
			seen[loss.note] = true
			notes = append(notes, loss.note)
		}
	}
	if allLost {
		return censusYearSurvival{lost: true}
	}
	sort.Strings(notes)

	return censusYearSurvival{note: strings.Join(notes, "; ")}
}

// placeInCountry reports whether a place resolves to this schedule's country,
// or names no country at all (the schedule then applies through
// censusCountryFallback, and the place is still the best evidence of where
// the person was).
func (s *censusSchedule) placeInCountry(placeID string, archive *glxlib.GLXFile) bool {
	country := resolveCountryFromPlace(placeID, archive)

	return country == "" || canonicalCountry(country) == s.country
}

// lookupJurisdictionLoss resolves a place to its state or territory and looks
// it up in one year's loss entries. A territory-typed place named without the
// word ("Indiana", type territory) also matches the "Indiana Territory" entry.
func lookupJurisdictionLoss(losses map[string]censusJurisdictionLoss, placeID string, archive *glxlib.GLXFile) (censusJurisdictionLoss, bool) {
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
