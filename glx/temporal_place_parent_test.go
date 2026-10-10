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

// temporalParentArchive is the Indiana example from #225: Wayne County was
// in Indiana Territory until statehood on 11 Dec 1816, then in Indiana.
func temporalParentArchive() *glxlib.GLXFile {
	wayne := &glxlib.Place{Name: "Wayne County", Type: glxlib.PlaceTypeCounty}
	wayne.SetParentHistory([]glxlib.PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "FROM 1811 TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	})

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-1": {Properties: map[string]any{glxlib.PersonPropertyName: "John Smith"}},
		},
		Places: map[string]*glxlib.Place{
			"place-usa":               {Name: "United States", Type: glxlib.PlaceTypeCountry},
			"place-indiana-territory": {Name: "Indiana Territory", ParentID: "place-usa"},
			"place-indiana":           {Name: "Indiana", Type: glxlib.PlaceTypeState, ParentID: "place-usa"},
			"place-wayne-in":          wayne,
		},
		Events: map[string]*glxlib.Event{
			"event-marriage": {
				Type: glxlib.EventTypeMarriage, Date: "1814-04-02", PlaceID: "place-wayne-in",
				Participants: []glxlib.Participant{{Person: "person-1", Role: "groom"}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{},
		Sources:       map[string]*glxlib.Source{},
		Citations:     map[string]*glxlib.Citation{},
		Assertions:    map[string]*glxlib.Assertion{},
	}
}

func TestPlaceFullNameAt_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.Equal(t, "Wayne County, Indiana Territory, United States", placeFullNameAt("place-wayne-in", "1814", archive))
	assert.Equal(t, "Wayne County, Indiana, United States", placeFullNameAt("place-wayne-in", "1817", archive))
	assert.Equal(t, "Wayne County, Indiana, United States", placeFullName("place-wayne-in", archive), "no date: default parent")
}

func TestBuildCanonicalPathAt_TemporalParent(t *testing.T) {
	places := temporalParentArchive().Places

	assert.Equal(t, "Wayne County, Indiana Territory, United States", buildCanonicalPathAt("place-wayne-in", "1813", places))
	assert.Equal(t, "Wayne County, Indiana, United States", buildCanonicalPath("place-wayne-in", places))
}

func TestMigrationEntry_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.Equal(t, "Wayne County, Indiana Territory, United States", newMigrationEntry("1814", "place-wayne-in", "Marriage", archive).Place)
	assert.Equal(t, "Wayne County, Indiana, United States", newMigrationEntry("1830", "place-wayne-in", "Residence", archive).Place)
}

func TestPlaceIsDescendant_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()

	assert.True(t, placeIsDescendant("place-wayne-in", "place-indiana-territory", archive))
	assert.True(t, placeIsDescendant("place-wayne-in", "place-indiana", archive))
	assert.True(t, placeIsDescendant("place-wayne-in", "place-usa", archive))
	assert.False(t, placeIsDescendant("place-indiana", "place-wayne-in", archive))
}

func TestBuildPlaceAnalysis_TemporalParent(t *testing.T) {
	archive := temporalParentArchive()
	archive.Events = map[string]*glxlib.Event{}

	a := buildPlaceAnalysis(archive)
	assert.NotContains(t, a.Unreferenced, "place-indiana-territory", "a parent in an earlier period is referenced")
	assert.Empty(t, a.DanglingParent)

	delete(archive.Places, "place-indiana-territory")
	a = buildPlaceAnalysis(archive)
	assert.Equal(t, []string{"place-wayne-in"}, a.DanglingParent)
	assert.Equal(t, "place-indiana-territory", a.DanglingParentIDs["place-wayne-in"])
}
