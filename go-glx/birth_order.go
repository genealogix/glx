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
	"math"
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
	keys map[string]int // person ID -> first civil day of the dating event
}

// NewBirthOrder indexes each person's birth date, falling back to their
// christening and then their baptism when the birth is missing or undated.
// Only events where the person is the subject count, and among several dated
// events of one type the lowest event ID wins, as in FindPersonEvent.
func NewBirthOrder(archive *GLXFile) *BirthOrder {
	bo := &BirthOrder{keys: map[string]int{}}
	if archive == nil {
		return bo
	}

	ids := make([]string, 0, len(archive.Events))
	for id := range archive.Events {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// byType[t][person] is the key from the first dated event of type t.
	byType := make(map[string]map[string]int, len(birthOrderEventTypes))
	for _, t := range birthOrderEventTypes {
		byType[t] = map[string]int{}
	}
	for _, id := range ids {
		event := archive.Events[id]
		if event == nil {
			continue
		}
		found, ok := byType[event.Type]
		if !ok {
			continue
		}
		key, dated := birthOrderKey(event.Date)
		if !dated {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" || (!isSubjectRole(p.Role) && !isResearchPrincipal(event.Type, p.Role)) {
				continue
			}
			if _, seen := found[p.Person]; !seen {
				found[p.Person] = key
			}
		}
	}

	for _, t := range birthOrderEventTypes {
		for person, key := range byType[t] {
			if _, ok := bo.keys[person]; !ok {
				bo.keys[person] = key
			}
		}
	}

	return bo
}

// birthOrderKey returns the first civil day a date can fall on, or false when
// the date is empty or cannot be placed.
func birthOrderKey(date DateString) (int, bool) {
	if date == "" {
		return 0, false
	}
	parsed, _ := date.Parse()
	bounds := parsed.Timing()
	if !bounds.Known || bounds.Outer.Start == math.MinInt {
		return 0, false
	}

	return bounds.Outer.Start, true
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
	sort.SliceStable(ids, func(i, j int) bool { return bo.Less(ids[i], ids[j]) })
}
