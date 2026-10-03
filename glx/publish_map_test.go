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
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// locatedRow builds a place index row carrying coordinates.
func locatedRow(name string, lat, lon float64, events int) placeRow {
	return placeRow{
		ID:         "place-" + name,
		Anchor:     "place-" + name,
		Name:       name,
		FullName:   name,
		HasCoords:  true,
		Lat:        lat,
		Lon:        lon,
		EventCount: events,
	}
}

// markerNamed returns the marker whose tooltip names the place, or nil.
func markerNamed(pm *placeMap, name string) *mapMarker {
	for i := range pm.Markers {
		if strings.HasPrefix(pm.Markers[i].Title, name) {
			return &pm.Markers[i]
		}
	}

	return nil
}

func TestBuildPlaceMap_NilWithoutEnoughLocatedPlaces(t *testing.T) {
	cases := map[string][]placeRow{
		"no places":         nil,
		"none located":      {{Name: "Boston"}, {Name: "Leeds"}},
		"only one located":  {locatedRow("Boston", 42.36, -71.06, 3), {Name: "Leeds"}},
		"one located twice": {locatedRow("Boston", 42.36, -71.06, 3)},
	}
	for name, rows := range cases {
		if pm := buildPlaceMap(rows); pm != nil {
			t.Errorf("%s: expected no map, got one with %d marker(s)", name, len(pm.Markers))
		}
	}
}

func TestBuildPlaceMap_PlotsEveryLocatedPlace(t *testing.T) {
	rows := []placeRow{
		locatedRow("Boston", 42.36, -71.06, 12),
		locatedRow("Leeds", 53.80, -1.55, 3),
		{Name: "Unlocated parish"},
	}

	pm := buildPlaceMap(rows)
	if pm == nil {
		t.Fatal("expected a map for two located places")
	}
	if got, want := len(pm.Markers), 2; got != want {
		t.Fatalf("marker count = %d, want %d (the unlocated place is not plottable)", got, want)
	}
	if pm.Located != 2 || pm.Total != 3 {
		t.Errorf("caption counts = %d of %d, want 2 of 3", pm.Located, pm.Total)
	}

	boston, leeds := markerNamed(pm, "Boston"), markerNamed(pm, "Leeds")
	if boston == nil || leeds == nil {
		t.Fatalf("expected both places on the map, got %+v", pm.Markers)
	}
	if boston.CX >= leeds.CX {
		t.Error("Boston is west of Leeds and must plot to its left")
	}
	if boston.CY <= leeds.CY {
		t.Error("Boston is south of Leeds and must plot below it")
	}
	if boston.R <= leeds.R {
		t.Errorf("Boston has more events (radius %.1f) than Leeds (radius %.1f); its marker should be larger",
			boston.R, leeds.R)
	}
	if boston.Href != "#place-Boston" {
		t.Errorf("marker links to %q, want the place's row anchor", boston.Href)
	}
	if !strings.Contains(boston.Title, "12 events") {
		t.Errorf("tooltip %q should report the event count", boston.Title)
	}

	for _, marker := range pm.Markers {
		if marker.CX < float64(pm.Frame.X) || marker.CX > float64(pm.Frame.X+pm.Frame.W) ||
			marker.CY < float64(pm.Frame.Y) || marker.CY > float64(pm.Frame.Y+pm.Frame.H) {
			t.Errorf("marker %q at (%.1f, %.1f) falls outside the frame", marker.Title, marker.CX, marker.CY)
		}
	}
}

func TestBuildPlaceMap_KeepsTheProjectionUndistorted(t *testing.T) {
	// Three places around 60°N, where a degree of longitude is half a degree
	// of latitude wide.
	rows := []placeRow{
		locatedRow("West", 60, -2, 1),
		locatedRow("East", 60, 2, 1),
		locatedRow("North", 61, 0, 1),
	}

	pm := buildPlaceMap(rows)
	if pm == nil {
		t.Fatal("expected a map")
	}
	west, east, north := markerNamed(pm, "West"), markerNamed(pm, "East"), markerNamed(pm, "North")
	if west == nil || east == nil || north == nil {
		t.Fatal("expected all three places on the map")
	}

	// 4° of longitude at 60°N covers the same ground as 2° of latitude, so the
	// two distances must render at the same scale.
	perLon := (east.CX - west.CX) / 4
	perLat := (west.CY - north.CY) / 1
	want := perLat * math.Cos(60*math.Pi/180)
	if math.Abs(perLon-want) > want*0.05 {
		t.Errorf("longitude renders at %.2f units/degree, want ~%.2f (latitude scale × cos 60°)", perLon, want)
	}
	if pm.Frame.H < mapMinHeight || pm.Frame.H > mapMaxHeight {
		t.Errorf("frame height %d outside the %d–%d bounds", pm.Frame.H, mapMinHeight, mapMaxHeight)
	}
}

func TestBoundingBox_WidensATightCluster(t *testing.T) {
	// Two places a few meters apart must not zoom the map to one street.
	box := boundingBox([]placeRow{
		locatedRow("Church", 53.8000, -1.5500, 1),
		locatedRow("Vicarage", 53.8004, -1.5504, 1),
	})

	if span := box.MaxLat - box.MinLat; span < mapMinSpanDegrees {
		t.Errorf("latitude span %.4f°, want at least %.2f°", span, mapMinSpanDegrees)
	}
	if span := box.MaxLon - box.MinLon; span < mapMinSpanDegrees {
		t.Errorf("longitude span %.4f°, want at least %.2f°", span, mapMinSpanDegrees)
	}
}

func TestBuildGraticule_LabelsRoundDegreesWithHemispheres(t *testing.T) {
	pm := buildPlaceMap([]placeRow{
		locatedRow("Quito", -0.18, -78.47, 1),
		locatedRow("Nairobi", -1.29, 36.82, 1),
	})
	if pm == nil {
		t.Fatal("expected a map")
	}

	labels := make([]string, 0, len(pm.Grid))
	for _, line := range pm.Grid {
		labels = append(labels, line.Label)
		if line.X1 < pm.Frame.X || line.X2 > pm.Frame.X+pm.Frame.W {
			t.Errorf("grid line %q runs outside the frame", line.Label)
		}
	}
	joined := strings.Join(labels, " ")
	for _, want := range []string{"°W", "°E", "°S", "0°"} {
		if !strings.Contains(joined, want) {
			t.Errorf("graticule labels %q missing %q", joined, want)
		}
	}
	for _, label := range labels {
		if label == "0°E" || label == "0°N" {
			t.Errorf("the prime meridian and the equator belong to no hemisphere; got %q in %q", label, joined)
		}
	}
}

func TestFormatDegrees_CarriesOnlyTheDecimalsTheStepNeeds(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"half-degree step keeps one decimal", formatLatitude(42.5, 0.5), "42.5°N"},
		{"whole-degree step drops decimals", formatLatitude(42, 1), "42°N"},
		{"southern latitudes carry S", formatLatitude(-33.87, 0.05), "33.87°S"},
		{"western longitudes carry W", formatLongitude(-71.25, 0.25), "71.25°W"},
		{"trailing zeros are trimmed", formatLongitude(5.5, 0.05), "5.5°E"},
		{"the prime meridian has no hemisphere", formatLongitude(0, 1), "0°"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestBuildMarkers_DropsLabelsThatWouldOverlap(t *testing.T) {
	// Two places at practically the same point: only the first keeps a label.
	rows := []placeRow{
		locatedRow("Leeds", 53.800, -1.550, 5),
		locatedRow("Leeds Minster", 53.801, -1.551, 1),
		locatedRow("York", 53.960, -1.080, 1),
	}

	pm := buildPlaceMap(rows)
	if pm == nil {
		t.Fatal("expected a map")
	}
	leeds, minster := markerNamed(pm, "Leeds ·"), markerNamed(pm, "Leeds Minster")
	if leeds == nil || minster == nil {
		t.Fatalf("expected both Leeds places on the map, got %+v", pm.Markers)
	}
	if leeds.Label == "" {
		t.Error("the busier of two overlapping places should keep its label")
	}
	if minster.Label != "" {
		t.Errorf("overlapping label %q should have been dropped", minster.Label)
	}
	if york := markerNamed(pm, "York"); york == nil || york.Label == "" {
		t.Error("a place with room for its label should keep it")
	}
}

func TestRenderSite_DrawsThePlaceMap(t *testing.T) {
	archive := buildModelTestArchive()
	// The test archive locates Boston only; give Massachusetts coordinates too
	// so the map has two points.
	lat, lon := 42.41, -71.38
	archive.Places["place-mass"].Latitude = &lat
	archive.Places["place-mass"].Longitude = &lon

	model := buildSiteModel(archive, siteModelOptions{Title: "Smith Family"})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	places := readFile(t, filepath.Join(out, "places", "index.html"))
	for _, want := range []string{
		`<figure class="placemap">`,
		`class="placemap-frame"`,
		`<circle cx=`,
		`place(s) in this archive carry coordinates`,
	} {
		if !strings.Contains(places, want) {
			t.Errorf("place index missing %q", want)
		}
	}
	for _, marker := range model.PlaceMap.Markers {
		if !strings.Contains(places, `href="`+marker.Href+`"`) {
			t.Errorf("marker link %q missing from the page", marker.Href)
		}
	}
}

func TestRenderSite_OmitsThePlaceMapWithoutCoordinates(t *testing.T) {
	archive := buildModelTestArchive()
	archive.Places["place-boston"].Latitude = nil
	archive.Places["place-boston"].Longitude = nil

	model := buildSiteModel(archive, siteModelOptions{})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	places := readFile(t, filepath.Join(out, "places", "index.html"))
	if strings.Contains(places, `<figure class="placemap">`) {
		t.Error("an archive with no located places should not get a map")
	}
	if !strings.Contains(places, `<table class="index-table">`) {
		t.Error("the place table must still render without a map")
	}
}
