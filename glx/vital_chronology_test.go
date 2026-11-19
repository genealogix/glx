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
	"testing"

	"github.com/stretchr/testify/assert"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestEventAfterDeath_RolesCanMentionADeceasedPerson(t *testing.T) {
	for _, scenario := range []struct {
		role, eventType string
		want            bool
	}{
		{"parent", "marriage", false},
		{"parent", "burial", false},
		{"godparent", "baptism", true},
		{"witness", "marriage", true},
		{"witness", "burial", true},
		{"principal", "probate", false},
	} {
		t.Run(scenario.role+scenario.eventType, func(t *testing.T) {
			archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Events: map[string]*glxlib.Event{
				"death": {Type: "death", Date: "1632", Participants: []glxlib.Participant{{Person: "p", Role: "principal"}}},
				"later": {Type: scenario.eventType, Date: "1634", Participants: []glxlib.Participant{{Person: "p", Role: scenario.role}}},
			}}
			assert.Equal(t, scenario.want, len(checkEventAfterDeath(archive)) > 0)
			no := false
			archive.ParticipantRoles = map[string]*glxlib.VocabularyEntry{scenario.role: {ImpliesPresence: &no}}
			assert.Empty(t, checkEventAfterDeath(archive))
		})
	}
}

func TestEventAfterDeath_UsesFullDateBounds(t *testing.T) {
	for _, scenario := range []struct {
		death, activity string
		want            bool
	}{
		{"1653-01-01", "1653-02-01", true},
		{"1653", "1653-02-01", false},
		{"AFT 1653", "1660", false},
		{"ABT 1653", "1654", false},
		{"JULIAN 1645-10-12", "1645-10-22", false},
	} {
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Events: map[string]*glxlib.Event{
			"death":    {Type: "death", Date: glxlib.DateString(scenario.death), Participants: []glxlib.Participant{{Person: "p", Role: "principal"}}},
			"activity": {Type: "baptism", Date: glxlib.DateString(scenario.activity), Participants: []glxlib.Participant{{Person: "p", Role: "godparent"}}},
		}}
		assert.Equal(t, scenario.want, len(checkEventAfterDeath(archive)) > 0, scenario)
	}
}
