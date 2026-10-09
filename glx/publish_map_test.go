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
		t.Errorf("frame height %g outside the %d–%d bounds", pm.Frame.H, mapMinHeight, mapMaxHeight)
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

func TestBuildPlaceMap_FitsAcrossTheDateLine(t *testing.T) {
	rows := []placeRow{
		locatedRow("West Fiji", -16, 179.5, 2),
		locatedRow("East Fiji", -16, -179.5, 1),
	}
	box := boundingBox(rows)
	if got, want := box.MaxLon-box.MinLon, 1.24; math.Abs(got-want) > degreeEpsilon {
		t.Fatalf("longitude span = %.2f°, want %.2f° for places one degree apart", got, want)
	}
	pm := buildPlaceMap(rows)
	west, east := markerNamed(pm, "West Fiji"), markerNamed(pm, "East Fiji")
	if west.CX >= east.CX {
		t.Fatal("crossing the date line eastward should preserve marker order")
	}
	// The one-degree gap covers 1/1.24 of the padded regional frame, just as
	// it would for a cluster away from the date line.
	if got, want := (east.CX-west.CX)/pm.Frame.W, 1/1.24; math.Abs(got-want) > degreeEpsilon {
		t.Errorf("marker gap/frame width = %.4f, want %.4f", got, want)
	}
}

func TestLongitudeBounds_UsesTheSmallestCircularInterval(t *testing.T) {
	cases := []struct {
		name string
		lons []float64
		span float64
	}{
		{"ordinary Atlantic", []float64{-71.06, -1.55}, 69.51},
		{"crosses date line", []float64{-179.5, 179.5}, 1},
		{"same meridian aliases", []float64{-180, 180, -180}, 0},
		{"cluster with date line aliases", []float64{179, -180, 180, -179}, 2},
		{"equal gaps", []float64{-120, 0, 120}, 240},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := make([]placeRow, len(tc.lons))
			for i, lon := range tc.lons {
				rows[i] = locatedRow("Place", 0, lon, 1)
			}
			minLon, maxLon := longitudeBounds(rows)
			if got := maxLon - minLon; math.Abs(got-tc.span) > degreeEpsilon {
				t.Errorf("span = %g°, want %g°", got, tc.span)
			}
			box := boundingBox(rows)
			for _, lon := range tc.lons {
				if unwrapped := box.unwrapLongitude(lon); unwrapped < box.MinLon || unwrapped > box.MaxLon {
					t.Errorf("longitude %g° unwraps outside %+v", lon, box)
				}
			}
		})
	}
}

func TestBuildPlaceMap_DateLineAliasesCoincide(t *testing.T) {
	pm := buildPlaceMap([]placeRow{
		locatedRow("Negative", 10, -180, 1),
		locatedRow("Positive", 10, 180, 1),
	})
	negative, positive := markerNamed(pm, "Negative"), markerNamed(pm, "Positive")
	if math.IsNaN(negative.CX) || math.IsNaN(negative.CY) || math.IsInf(negative.CX, 0) || math.IsInf(negative.CY, 0) {
		t.Fatalf("coincident points must have a finite projection: %+v", negative)
	}
	if negative.CX != positive.CX || negative.CY != positive.CY {
		t.Errorf("±180° name the same location: %+v versus %+v", negative, positive)
	}
}

func TestBuildPlaceMap_PreservesScaleNearGeographicBounds(t *testing.T) {
	cases := []struct {
		name string
		rows []placeRow
	}{
		{"Atlantic control", []placeRow{locatedRow("Boston", 42.36, -71.06, 1), locatedRow("Leeds", 53.8, -1.55, 1)}},
		{"Anchorage-Honolulu", []placeRow{locatedRow("Anchorage", 61.2181, -149.9003, 1), locatedRow("Honolulu", 21.3099, -157.8581, 1)}},
		{"north pole", []placeRow{locatedRow("Pole", 90, 180, 1), locatedRow("North", 89.9, -179.9, 1)}},
		{"south pole", []placeRow{locatedRow("Pole", -90, -180, 1), locatedRow("South", -89.9, 179.9, 1)}},
		{"wide polar view", []placeRow{locatedRow("West", 80, -135, 1), locatedRow("Middle", 85, -45, 1), locatedRow("East", 90, 45, 1), locatedRow("Far east", 89, 135, 1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			box, height := fitBox(boundingBox(tc.rows), mapFrameWidth)
			pm := buildPlaceMap(tc.rows)
			assertMapReferenceScale(t, box, pm.Frame)
			if height < mapMinHeight || height > mapMaxHeight {
				t.Errorf("allocated height = %d, outside map limits", height)
			}
			if box.MinLat < -maxLatitude || box.MaxLat > maxLatitude || box.MaxLon-box.MinLon > longitudeTurn {
				t.Errorf("geographic range outside limits: %+v", box)
			}
			for _, marker := range pm.Markers {
				if marker.CX < pm.Frame.X-degreeEpsilon || marker.CX > pm.Frame.X+pm.Frame.W+degreeEpsilon ||
					marker.CY < pm.Frame.Y-degreeEpsilon || marker.CY > pm.Frame.Y+pm.Frame.H+degreeEpsilon {
					t.Errorf("marker %+v outside frame %+v", marker, pm.Frame)
				}
			}
		})
	}
}

func TestFitBox_LetterboxesAWorldViewAtThePole(t *testing.T) {
	box, height := fitBox(geoBox{MinLat: 89, MaxLat: 90, MinLon: -180, MaxLon: 180}, mapFrameWidth)
	frame := fittedMapFrame(box, height)
	assertMapReferenceScale(t, box, frame)
	if frame.H >= float64(height) || frame.Y <= mapInsetTop {
		t.Errorf("polar world view should letterbox inside its height, got %+v in %d", frame, height)
	}
	if box.MinLat < -maxLatitude || box.MaxLat > maxLatitude {
		t.Errorf("latitude padding must stay valid: %+v", box)
	}
	// Every grid line remains within the plot and the full circle gets no more
	// than the configured number of meridians.
	meridians := 0
	for _, line := range buildGraticule(box, frame) {
		if line.Anchor == "middle" {
			meridians++
			if line.X1 < frame.X || line.X1 > frame.X+frame.W {
				t.Errorf("meridian outside frame: %+v", line)
			}
		}
	}
	if meridians > mapMaxGridLines {
		t.Errorf("world view has %d meridians, limit %d", meridians, mapMaxGridLines)
	}
}

func TestFitBox_FullWorldKeepsEveryCoordinateInFrame(t *testing.T) {
	box, height := fitBox(geoBox{MinLat: -90, MaxLat: 90, MinLon: -180, MaxLon: 180}, mapFrameWidth)
	frame := fittedMapFrame(box, height)
	assertMapReferenceScale(t, box, frame)
	for _, lat := range []float64{-90, 0, 90} {
		for _, lon := range []float64{-180, 0, 180} {
			x, y := box.project(lat, box.unwrapLongitude(lon), frame)
			if x < frame.X || x > frame.X+frame.W || y < frame.Y || y > frame.Y+frame.H {
				t.Errorf("(%g, %g) projects outside full world frame: (%g, %g)", lat, lon, x, y)
			}
		}
	}
}

func TestPadLat_PreservesSpanAtEitherPole(t *testing.T) {
	for _, box := range []geoBox{{MinLat: 90, MaxLat: 90}, {MinLat: -90, MaxLat: -90}} {
		padded := padLat(box, mapMinSpanDegrees/2)
		if got := padded.MaxLat - padded.MinLat; got != mapMinSpanDegrees {
			t.Errorf("padded polar span = %g°, want %g°", got, mapMinSpanDegrees)
		}
		if padded.MinLat > box.MinLat || padded.MaxLat < box.MaxLat || padded.MinLat < -maxLatitude || padded.MaxLat > maxLatitude {
			t.Errorf("padding lost the place or exceeded a pole: %+v -> %+v", box, padded)
		}
	}
}

func assertMapReferenceScale(t *testing.T, box geoBox, frame mapFrame) {
	t.Helper()
	latScale := frame.H / (box.MaxLat - box.MinLat)
	lonScale := frame.W / (box.MaxLon - box.MinLon)
	if got, want := lonScale/latScale, box.lonScale(); math.Abs(got-want) > degreeEpsilon {
		t.Errorf("longitude/latitude scale = %g, want cosine correction %g", got, want)
	}
	if frame.X < mapInsetLeft-degreeEpsilon || frame.X+frame.W > mapWidth-mapInsetRight+degreeEpsilon ||
		frame.Y < mapInsetTop-degreeEpsilon || frame.H > mapMaxHeight+degreeEpsilon {
		t.Errorf("fitted frame outside allocated area: %+v", frame)
	}
}

func TestBuildGraticule_NormalizesUnwrappedLongitudeLabels(t *testing.T) {
	box := geoBox{MinLat: -16.2, MaxLat: -15.8, MinLon: 179.4, MaxLon: 180.6}
	frame := fittedMapFrame(box, mapMinHeight)
	var labels []string
	previousX := math.Inf(-1)
	for _, line := range buildGraticule(box, frame) {
		if line.Anchor != "middle" {
			continue
		}
		if line.X1 <= previousX {
			t.Errorf("unwrapped meridians must advance across the frame: %+v", line)
		}
		previousX = line.X1
		labels = append(labels, line.Label)
	}
	want := "179.5°E 179.75°E 180°W 179.75°W 179.5°W"
	if got := strings.Join(labels, " "); got != want {
		t.Errorf("longitude labels = %q, want %q", got, want)
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
