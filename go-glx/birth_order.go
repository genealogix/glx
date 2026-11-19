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
)

// birthOrderEventTypes are the events that date a person's birth, in order of
// preference: a recorded birth first, then christening, then baptism, the
// usual record before civil registration.
var birthOrderEventTypes = []string{EventTypeBirth, EventTypeChristening, EventTypeBaptism}

// BirthOrder ranks people by when they were born so that every view lists a
// family's children and siblings in the same order. Build it once with
// NewBirthOrder and reuse it; lookups are constant time.
type BirthOrder struct {
	keys map[string]int // person ID -> Julian Day Number of the dating event
}

// NewBirthOrder indexes each person's birth date, falling back to their
// christening and then their baptism when the birth is missing or undated.
// For each type it takes the event FindPersonEvent returns, the lowest-ID
// event in which the person holds their own role, so the date a view shows
// for a person is the date that person sorts by. Dates are compared as Julian
// Day Numbers, so JULIAN dates sort among Gregorian ones; a BEF date sorts
// just before the day it names. It makes one pass over the archive's events.
func NewBirthOrder(archive *GLXFile) *BirthOrder {
	bo := &BirthOrder{keys: map[string]int{}}
	if archive == nil {
		return bo
	}

	// byType[t][person] is the person's lowest-ID event of type t, dated or not.
	type candidate struct {
		eventID string
		key     int
		dated   bool
	}
	byType := make(map[string]map[string]candidate, len(birthOrderEventTypes))
	for _, t := range birthOrderEventTypes {
		byType[t] = map[string]candidate{}
	}
	for id, event := range archive.Events {
		if event == nil {
			continue
		}
		found, ok := byType[event.Type]
		if !ok {
			continue
		}
		key, dated := birthOrderKey(event.Date)
		for _, p := range event.Participants {
			if p.Person == "" || !isPersonEventRole(event.Type, p.Role) {
				continue
			}
			if prev, seen := found[p.Person]; !seen || id < prev.eventID {
				found[p.Person] = candidate{eventID: id, key: key, dated: dated}
			}
		}
	}

	for _, t := range birthOrderEventTypes {
		for person, c := range byType[t] {
			if _, ok := bo.keys[person]; !ok && c.dated {
				bo.keys[person] = c.key
			}
		}
	}

	return bo
}

// birthOrderKey returns the date's ordering day (see glxdate.Date.OrderingDay),
// or false when the date is empty or cannot be placed.
func birthOrderKey(date DateString) (int, bool) {
	if date == "" {
		return 0, false
	}
	parsed, _ := date.Parse()

	return parsed.OrderingDay()
}

// Less reports whether person a sorts before person b: dated people by date,
// earliest first, then undated people, with the ID breaking every tie.
func (bo *BirthOrder) Less(a, b string) bool {
	ka, okA := bo.keys[a]
	kb, okB := bo.keys[b]
	switch {
	case okA != okB:
		return okA
	case okA && ka != kb:
		return ka < kb
	default:
		return a < b
	}
}

// Sort orders person IDs in place by birth order (see Less).
func (bo *BirthOrder) Sort(ids []string) {
	SortByBirth(bo, ids, func(id string) string { return id })
}

// SortByBirth orders items in place by the birth order of the person each one
// names, as id picks it out.
func SortByBirth[T any](bo *BirthOrder, items []T, id func(T) string) {
	sort.SliceStable(items, func(i, j int) bool { return bo.Less(id(items[i]), id(items[j])) })
}
