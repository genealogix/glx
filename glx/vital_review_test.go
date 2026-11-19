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

func TestVitalReview_RoleOverridesUseDeclaredThenNormalizedKeys(t *testing.T) {
	yes, no := true, false
	mentioned := func(role string, roles map[string]*glxlib.VocabularyEntry) bool {
		participation := glxlib.ClassifyParticipation("marriage", role, roles)

		return !participation.Present || mentionedRelativeAtEvent("marriage", role, participation)
	}
	for _, role := range []string{"Parent", " parent ", " PARENT "} {
		assert.False(t, mentioned(role, map[string]*glxlib.VocabularyEntry{"parent": {ImpliesPresence: &yes}}), role)
		assert.True(t, mentioned(role, nil), role)
	}
	assert.True(t, mentioned(" Parent ", map[string]*glxlib.VocabularyEntry{"Parent": {ImpliesPresence: &no}, "parent": {ImpliesPresence: &yes}}), "a declared key takes precedence over its normalized fallback")
}

func TestVitalReview_PosthumousChildNamesTheLateFather(t *testing.T) {
	for _, eventType := range []string{"birth", "baptism", "christening"} {
		archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"father": {}, "child": {}}, Events: map[string]*glxlib.Event{
			"death":   {Type: "death", Date: "1653-03-01", Participants: []glxlib.Participant{{Person: "father", Role: "principal"}}},
			"baptism": {Type: eventType, Date: "1653-09-10", Participants: []glxlib.Participant{{Person: "child", Role: "principal"}, {Person: "father", Role: "father"}}},
		}}
		assert.Empty(t, checkEventAfterDeath(archive), eventType)
		// A godparent appearance after death is still reported.
		archive.Events["baptism"].Participants[1].Role = "godparent"
		assert.NotEmpty(t, checkEventAfterDeath(archive), eventType)
	}
}

func TestVitalReview_EventAfterDeathInOtherCalendars(t *testing.T) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{"p": {}}, Events: map[string]*glxlib.Event{
		"death":   {Type: "death", Date: "HEBREW 5600", Participants: []glxlib.Participant{{Person: "p", Role: "principal"}}},
		"witness": {Type: "marriage", Date: "HEBREW 5610", Participants: []glxlib.Participant{{Person: "p", Role: "witness"}}},
	}}
	assert.NotEmpty(t, checkEventAfterDeath(archive), "same-calendar dates are still compared")
	archive.Events["witness"].Date = "1860"
	assert.Empty(t, checkEventAfterDeath(archive), "a Hebrew date is not compared with a Gregorian one without conversion")
}

func TestVitalReview_UnconvertibleCalendarsFollowChronologicalDates(t *testing.T) {
	assert.Less(t, dateSortKey("2010"), dateSortKey("HEBREW 15 TSH 5765"))
	assert.Less(t, dateSortKey("1850"), dateSortKey("HEBREW 15 TSH 5600"))
	assert.Less(t, dateSortKey("HEBREW 15 TSH 5600"), dateSortKey("HEBREW 15 TSH 5765"))
	assert.Less(t, dateSortKey("HEBREW 15 TSH 5765"), dateSortKey(""))
}

func TestVitalReview_BEFDatesSortByTheirDistinctBounds(t *testing.T) {
	assert.Less(t, dateSortKey("BEF 1900"), dateSortKey("BEF 1920"))
	assert.Less(t, dateSortKey("BEF 1900-01"), dateSortKey("BEF 1900-02"))
	assert.Less(t, dateSortKey("BEF 1900-01-01"), dateSortKey("BEF 1900-01-02"))
	assert.Less(t, dateSortKey("BEF 0100 BCE"), dateSortKey("BEF 0050 BCE"))
	assert.Less(t, dateSortKey("JULIAN BEF 1645-10-12"), dateSortKey("1645-10-22"))
}
