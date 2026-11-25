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

func TestPlacePathsAtCarriesOneSharedObservationWindow(t *testing.T) {
	archive := &GLXFile{Places: map[string]*Place{
		"ridge": {ParentHistory: []PlaceParentPeriod{
			{Value: "north", Date: "TO 1841-06-30"},
			{Value: "south", Date: "FROM 1841-07-01"},
		}},
		"north": {ParentID: "canada"},
		"south": {ParentHistory: []PlaceParentPeriod{
			{Value: "uk", Date: "TO 1841-06-30"},
			{Value: "canada", Date: "FROM 1841-07-01 TO 1841-09-30"},
			{Value: "usa", Date: "FROM 1841-10-01"},
		}},
		"canada": {}, "uk": {}, "usa": {},
	}}
	assert.Equal(t, [][]string{
		{"ridge", "north", "canada"},
		{"ridge", "south", "canada"},
		{"ridge", "south", "usa"},
	}, archive.PlacePathsAt("ridge", "1841"))
	assert.Equal(t, []string{"ridge"}, archive.PlacePathAt("ridge", "1841"))
	assert.Equal(t, []string{"ridge", "south", "usa"}, archive.PlacePathAt("ridge", "1841-10-01"))
	assert.True(t, archive.PlaceHasAncestor("ridge", "uk"))
	assert.False(t, archive.PlaceMayHaveAncestorAt("ridge", "uk", "1841"))
	assert.True(t, archive.PlaceMayHaveAncestorAt("ridge", "canada", "1841"))
	assert.False(t, archive.PlaceHasAncestorAt("ridge", "canada", "1841"))
	assert.True(t, archive.PlaceHasAncestorAt("ridge", "canada", "BET 1841-06-01 AND 1841-09-30"))
	assert.False(t, archive.PlaceHasAncestorAt("ridge", "canada", "1841-10-01"))
	assert.Equal(t, []string{"ridge", "south", "usa"}, archive.PlacePathAt("ridge", "FROM 1850 TO 1841"), "unplaced: default path")
	assert.Equal(t, []string{"ridge", "south", "usa"}, archive.PlacePathAt("ridge", ""))

	certain, possible := archive.placeAncestorForDates("ridge", "canada", "1841", "1841-07-01")
	assert.True(t, certain)
	assert.True(t, possible)
	certain, possible = archive.placeAncestorForDates("ridge", "canada", "1841", "1842")
	assert.False(t, certain)
	assert.False(t, possible)
	certain, possible = archive.placeAncestorForDates("ridge", "usa")
	assert.False(t, certain)
	assert.True(t, possible)
	certain, possible = archive.placeAncestorForDates("ridge", "uk")
	assert.False(t, certain)
	assert.False(t, possible)
	assert.False(t, archive.PlaceHasAncestorAt("ridge", "usa", ""))
	assert.True(t, archive.PlaceMayHaveAncestorAt("ridge", "usa", ""))
	for _, unplaced := range []DateString{"FROM 1850 TO 1841", "HEBREW 10 TSH 5600"} {
		// A projection follows the defaults; a "may" query rules out only
		// what no common day supports, as placeAncestorForDates does.
		assert.Equal(t, []string{"ridge", "south", "usa"}, archive.PlacePathAt("ridge", unplaced))
		assert.True(t, archive.PlaceMayHaveAncestorAt("ridge", "usa", unplaced))
		assert.True(t, archive.PlaceMayHaveAncestorAt("ridge", "canada", unplaced))
		assert.False(t, archive.PlaceMayHaveAncestorAt("ridge", "uk", unplaced))
		certain, possible = archive.placeAncestorForDates("ridge", "usa", unplaced)
		assert.False(t, certain)
		assert.True(t, possible)
		certain, possible = archive.placeAncestorForDates("ridge", "uk", unplaced)
		assert.False(t, certain)
		assert.False(t, possible, "structural-only path has no common day")
		certain, possible = archive.placeAncestorForDates("ridge", "usa", unplaced, "1841", "1841-10-01")
		assert.False(t, certain)
		assert.True(t, possible)
		certain, possible = archive.placeAncestorForDates("ridge", "canada", unplaced, "1841", "1841-10-01")
		assert.False(t, certain)
		assert.False(t, possible)
		certain, possible = archive.placeAncestorForDates("ridge", "usa", unplaced, "1841", "1842")
		assert.False(t, certain)
		assert.False(t, possible)
		certain, possible = archive.placeAncestorForDates("north", "canada", unplaced)
		assert.True(t, certain)
		assert.True(t, possible)
	}
}

func TestPlacePathsAtRetainsUnknownAndFallbackIntervals(t *testing.T) {
	archive := &GLXFile{Places: map[string]*Place{
		"village": {ParentHistory: []PlaceParentPeriod{
			{Value: "district", Date: "FROM 1841-02-01 TO 1841-03-31"},
			{Value: "invalid", Date: "1841-02-30"},
			{Value: "reversed", Date: "FROM 1860 TO 1840"},
		}},
		"district": {}, "default": {},
	}}
	village := archive.Places["village"]
	assert.Equal(t, [][]string{{"village", "district"}, {"village"}}, archive.PlacePathsAt("village", "1841"))
	assert.Equal(t, []string{"village"}, archive.PlacePathAt("village", "1850"))
	assert.Equal(t, []string{"village", "district"}, archive.PlacePathAt("village", "1841-02-01"))
	assert.True(t, archive.PlaceMayHaveAncestorAt("village", "district", "1841"))
	assert.False(t, archive.PlaceHasAncestorAt("village", "district", "1841"))
	village.ParentHistory = append(village.ParentHistory, PlaceParentPeriod{Value: "default"})
	assert.Equal(t, [][]string{{"village", "district"}, {"village", "default"}}, archive.PlacePathsAt("village", "1841"))
	assert.Equal(t, []string{"village", "default"}, archive.PlacePathAt("village", "1850"))
}

func TestPlacePathsAtHandlesOpenQualifiersAndCycles(t *testing.T) {
	archive := &GLXFile{Places: map[string]*Place{
		"village": {ParentHistory: []PlaceParentPeriod{
			{Value: "old", Date: "TO 1816-12-10"},
			{Value: "new", Date: "FROM 1816-12-11"},
		}},
		"old": {}, "new": {},
		"a": {ParentID: "b"},
		"b": {ParentID: "a"},
	}}
	assert.Equal(t, []string{"village", "old"}, archive.PlacePathAt("village", "BEF 1816-12-11"))
	assert.Equal(t, []string{"village", "new"}, archive.PlacePathAt("village", "AFT 1816-12-10"))
	path, cycle := archive.placePathProjection("a", "1971")
	assert.True(t, cycle)
	assert.Equal(t, []string{"a", "b"}, path)
}

func TestPlacePathsAtCoalescesDuplicateHistory(t *testing.T) {
	place := &Place{ParentHistory: []PlaceParentPeriod{
		{Value: "district", Date: "FROM 1841-01-01 TO 1841-03-31"},
		{Value: "district", Date: "FROM 1841-01-01 TO 1841-03-31"},
		{Value: "district", Date: "FROM 1841-03-31 TO 1841-12-31"},
	}}
	window, ok := parentPeriodSpan("1841")
	assert.True(t, ok)
	assert.Equal(t, []placePathEdge{{"district", window}}, place.pathEdges(placePossiblePath, window))
	assert.Equal(t, []placePathEdge{{"district", window}}, place.pathEdges(placeCertainPath, window))
}

func TestFactualPlaceAncestryAllowsUnsupportedParentCalendarOnlyAsPossible(t *testing.T) {
	archive := &GLXFile{Places: map[string]*Place{
		"town": {ParentHistory: []PlaceParentPeriod{
			{Value: "old", Date: "HEBREW TO 5601"},
			{Value: "new", Date: "FROM 1842"},
			{Value: "malformed", Date: "HEBREW TO"},
			{Value: "invalid", Date: "FROM 1900-02-29"},
			{Value: "reversed", Date: "HEBREW FROM 5601 TO 5500"},
		}},
		"village": {ParentHistory: []PlaceParentPeriod{{Value: "unknown", Date: "TO 1840-12-31"}}},
		"unknown": {ParentHistory: []PlaceParentPeriod{{Value: "north", Date: "HEBREW TO 5601"}}},
		"north":   {ParentHistory: []PlaceParentPeriod{{Value: "country", Date: "FROM 1841-01-01"}}},
	}}
	certain, possible := archive.placeAncestorForDates("town", "old", "1839-09-18")
	assert.False(t, certain)
	assert.True(t, possible)
	assert.Equal(t, []string{"town"}, archive.PlacePathAt("town", "1839-09-18"))
	assert.False(t, archive.PlaceMayHaveAncestorAt("town", "old", "1839-09-18"))
	assert.False(t, archive.PlaceHasAncestorAt("town", "new", "1845"), "the unplaced old jurisdiction remains a competing possibility")
	for _, invalid := range []string{"malformed", "invalid", "reversed"} {
		certain, possible = archive.placeAncestorForDates("town", invalid, "1901")
		assert.False(t, certain)
		assert.False(t, possible, invalid)
	}
	certain, possible = archive.placeAncestorForDates("village", "north", "1840")
	assert.False(t, certain)
	assert.True(t, possible)
	certain, possible = archive.placeAncestorForDates("village", "country")
	assert.False(t, certain)
	assert.False(t, possible, "unsupported edge must not reset the known window")
}
