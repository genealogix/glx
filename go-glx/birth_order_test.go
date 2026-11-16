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
		// A witness's own date is not the witness's birth.
		"e8": birthOrderEvent(EventTypeBirth, "1600", "p-witness", ParticipantRoleWitness),
		// A year-only date sorts at the start of its year; same day ties by ID.
		"e9":  birthOrderEvent(EventTypeBirth, "1706", "p-year", ParticipantRolePrincipal),
		"e10": birthOrderEvent(EventTypeBirth, "1688-11-04", "p-twin", ParticipantRolePrincipal),
	}}

	ids := []string{"p-witness", "p-chr", "p-birth", "p-none", "p-undated-birth", "p-year", "p-twin", "p-bapt"}
	NewBirthOrder(archive).Sort(ids)

	assert.Equal(t, []string{
		"p-bapt", "p-twin", "p-birth", "p-undated-birth", "p-year", "p-chr",
		"p-none", "p-witness",
	}, ids)
}

func TestBirthOrder_NilArchive(t *testing.T) {
	ids := []string{"b", "a"}
	NewBirthOrder(nil).Sort(ids)
	assert.Equal(t, []string{"a", "b"}, ids)
}
