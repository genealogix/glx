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
	"testing"

	"github.com/stretchr/testify/assert"
)

func birthOrderEvent(eventType, date, person, role string) *Event {
	return &Event{
		Type:         eventType,
		Date:         DateString(date),
		Participants: []Participant{{Person: person, Role: role}},
	}
}

func TestBirthOrder_Sort(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{
		// Birth wins over an earlier baptism record.
		"e1": birthOrderEvent(EventTypeBirth, "1700-10-24", "p-birth", ParticipantRolePrincipal),
		"e2": birthOrderEvent(EventTypeBaptism, "1650", "p-birth", ParticipantRolePrincipal),
		// Baptism only, before civil registration.
		"e3": birthOrderEvent(EventTypeBaptism, "1688-11-04", "p-bapt", ParticipantRolePrincipal),
		// Christening beats baptism when there is no birth.
		"e4": birthOrderEvent(EventTypeChristening, "1711-05-03", "p-chr", ""),
		"e5": birthOrderEvent(EventTypeBaptism, "1600", "p-chr", ""),
		// An undated birth falls back to the dated baptism.
		"e6": birthOrderEvent(EventTypeBirth, "", "p-undated-birth", ParticipantRolePrincipal),
		"e7": birthOrderEvent(EventTypeBaptism, "1705-01-01", "p-undated-birth", ParticipantRolePrincipal),
		// A child-role birth is the child's own, as FindPersonEvent reads it.
		"e11": birthOrderEvent(EventTypeBirth, "1690-02-02", "p-child-role", ParticipantRoleChild),
		// Of two dated births, the lowest event ID wins.
		"e12": birthOrderEvent(EventTypeBirth, "1720", "p-two", ParticipantRolePrincipal),
		"e13": birthOrderEvent(EventTypeBirth, "1600", "p-two", ParticipantRolePrincipal),
		// A witness's own date is not the witness's birth.
		"e8": birthOrderEvent(EventTypeBirth, "1600", "p-witness", ParticipantRoleWitness),
		// A year-only date sorts at the start of its year; same day ties by ID.
		"e9":  birthOrderEvent(EventTypeBirth, "1706", "p-year", ParticipantRolePrincipal),
		"e10": birthOrderEvent(EventTypeBirth, "1688-11-04", "p-twin", ParticipantRolePrincipal),
	}}

	ids := []string{"p-two", "p-child-role", "p-witness", "p-chr", "p-birth", "p-none", "p-undated-birth", "p-year", "p-twin", "p-bapt"}
	NewBirthOrder(archive).Sort(ids)

	assert.Equal(t, []string{
		"p-bapt", "p-twin", "p-child-role", "p-birth", "p-undated-birth", "p-year", "p-chr",
		"p-two", "p-none", "p-witness",
	}, ids)
}

func TestBirthOrder_NilArchive(t *testing.T) {
	ids := []string{"b", "a"}
	NewBirthOrder(nil).Sort(ids)
	assert.Equal(t, []string{"a", "b"}, ids)
}

// The date a person sorts by is the one FindPersonEvent shows: its lowest-ID
// event of the type. When that event is undated the next type is used, never a
// later dated event of the same type the views do not display.
func TestBirthOrder_MatchesFindPersonEvent(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{
		"e1": birthOrderEvent(EventTypeBirth, "", "p-a", ParticipantRolePrincipal),
		"e2": birthOrderEvent(EventTypeBirth, "1600", "p-a", ParticipantRolePrincipal),
		"e3": birthOrderEvent(EventTypeBirth, "1700", "p-b", ParticipantRolePrincipal),
	}}
	id, _ := FindPersonEvent(archive, "p-a", EventTypeBirth)
	assert.Equal(t, "e1", id)

	ids := []string{"p-a", "p-b"}
	NewBirthOrder(archive).Sort(ids)
	assert.Equal(t, []string{"p-b", "p-a"}, ids, "p-a's shown birth is undated, so p-a is undated")
}

// A BEF birth sorts just before the day it names rather than last, and a
// JULIAN date is converted before it is compared with a Gregorian one.
func TestBirthOrder_OpenStartAndJulian(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{
		"e1": birthOrderEvent(EventTypeBirth, "BEF 1690", "p-bef", ParticipantRolePrincipal),
		"e2": birthOrderEvent(EventTypeBirth, "1690", "p-1690", ParticipantRolePrincipal),
		"e3": birthOrderEvent(EventTypeBirth, "1710", "p-1710", ParticipantRolePrincipal),
		// JULIAN 20 Feb 1700 is Gregorian 3 Mar 1700, after Gregorian 25 Feb.
		"e4": birthOrderEvent(EventTypeBirth, "JULIAN 1700-02-20", "p-julian", ParticipantRolePrincipal),
		"e5": birthOrderEvent(EventTypeBirth, "1700-02-25", "p-greg", ParticipantRolePrincipal),
	}}

	ids := []string{"p-1710", "p-julian", "p-greg", "p-1690", "p-bef"}
	NewBirthOrder(archive).Sort(ids)
	assert.Equal(t, []string{"p-bef", "p-1690", "p-greg", "p-julian", "p-1710"}, ids)
}

func TestSortByBirth(t *testing.T) {
	type kid struct{ id string }
	archive := &GLXFile{Events: map[string]*Event{
		"e1": birthOrderEvent(EventTypeBirth, "1700", "b", ParticipantRolePrincipal),
		"e2": birthOrderEvent(EventTypeBirth, "1690", "a", ParticipantRolePrincipal),
	}}
	kids := []kid{{"b"}, {"a"}, {"c"}}
	SortByBirth(NewBirthOrder(archive), kids, func(k kid) string { return k.id })
	assert.Equal(t, []kid{{"a"}, {"b"}, {"c"}}, kids)
}
