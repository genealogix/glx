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
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Map geometry, in user units. The map is laid out at this logical width and
// scaled to the page by the stylesheet, so it reads the same on a phone and a
// desktop.
const (
	mapWidth     = 720
	mapMinHeight = 220
	mapMaxHeight = 460
)

// Margins around the plotted frame. The left margin holds the latitude labels
// and the bottom margin the longitude ones, so they are wider than the two
// sides that carry nothing.
const (
	mapInsetLeft   = 48
	mapInsetRight  = 20
	mapInsetTop    = 16
	mapInsetBottom = 34
)

// Marker sizing. A place's marker grows with the number of events recorded
// there, on a square-root scale so a place with 100 events does not swamp one
// with four.
const (
	mapDotBase = 4.0
	mapDotStep = 1.6
	mapDotMax  = 11.0
)

// mapFrameWidth is the plotted area's width, once the margins are taken out.
const mapFrameWidth = mapWidth - mapInsetLeft - mapInsetRight

// Geographic limits and conversions used by the projection.
const (
	maxLatitude    = 90
	maxLongitude   = 180
	degreesPerPi   = 180
	minLonScale    = 0.05 // never let a polar box collapse to zero width
	labelBaselineY = 15   // longitude labels sit this far below the frame
	degreeEpsilon  = 1e-9 // tolerance when testing a step for whole decimals
)

// mapMinPoints is the number of located places below which no map is drawn: a
// single point positions nothing the place's own coordinates do not.
const mapMinPoints = 2

// mapMinSpanDegrees keeps a cluster of places recorded in the same parish from
// being magnified into a map of one street, where the coordinates' own
// precision would be the only thing on display.
const mapMinSpanDegrees = 0.25

// mapPadFraction is the margin added around the places' bounding box so no
// marker sits on the frame.
const mapPadFraction = 0.12

// mapGraticuleSteps are the degree intervals the latitude/longitude grid may
// use, smallest first; the first one that yields at most mapMaxGridLines lines
// on the wider axis wins.
var mapGraticuleSteps = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 45}

// mapMaxGridLines bounds the grid lines drawn per axis, and degreeMaxDecimals
// the precision of their degree labels — the finest step in
// mapGraticuleSteps needs two.
const (
	mapMaxGridLines   = 6
	degreeMaxDecimals = 2
)

// Label placement. Text width cannot be measured without a font, so labels are
// budgeted by an average glyph width and dropped when they would collide.
const (
	mapLabelMax      = 14
	mapLabelDX       = 9
	mapLabelDY       = 4
	mapLabelHeight   = 13
	mapGlyphWidth    = 6.1
	mapLabelMaxRunes = 26
)

// placeMap is a laid-out map of the archive's located places: a graticule of
// round latitude/longitude lines with one marker per place. Coordinates are
// precomputed in user units so the template stays logic-free.
type placeMap struct {
	Label   string // accessible label for the SVG
	Width   int
	Height  int
	Frame   mapFrame
	Grid    []mapGridLine
	Markers []mapMarker
	// Located and Total report how many places carry coordinates, for the
	// caption under the map.
	Located int
	Total   int
}

// mapFrame is the plotted area's border box.
type mapFrame struct {
	X int
	Y int
	W int
	H int
}

// mapGridLine is one graticule line with its degree label.
type mapGridLine struct {
	X1    int
	Y1    int
	X2    int
	Y2    int
	TextX int
	TextY int
	Label string
	// Anchor is the SVG text-anchor for the label: parallels label at the
	// left edge, meridians centered under the bottom edge.
	Anchor string
}

// mapMarker is one located place.
type mapMarker struct {
	CX     float64
	CY     float64
	R      float64
	Href   string // fragment link to the place's row in the index
	Label  string // short name drawn beside the marker; "" when it was dropped
	LabelX float64
	LabelY float64
	Title  string // tooltip: full place name and event count
}

// geoBox is a latitude/longitude bounding box in degrees.
type geoBox struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// buildPlaceMap lays out the map for the place index, or returns nil when
// fewer than mapMinPoints places carry coordinates.
func buildPlaceMap(rows []placeRow) *placeMap {
	located := locatedRows(rows)
	if len(located) < mapMinPoints {
		return nil
	}

	box, height := fitBox(boundingBox(located), float64(mapFrameWidth))
	pm := &placeMap{
		Label:   fmt.Sprintf("Map of %d located place(s) in this archive", len(located)),
		Width:   mapWidth,
		Height:  height + mapInsetTop + mapInsetBottom,
		Frame:   mapFrame{X: mapInsetLeft, Y: mapInsetTop, W: mapFrameWidth, H: height},
		Located: len(located),
		Total:   len(rows),
	}
	pm.Grid = buildGraticule(box, pm.Frame)
	pm.Markers = buildMarkers(located, box, pm.Frame)

	return pm
}

// locatedRows returns the rows carrying coordinates, ordered by event count
// (descending) then by name, so the busiest places get first claim on a label.
func locatedRows(rows []placeRow) []placeRow {
	located := make([]placeRow, 0, len(rows))
	for i := range rows {
		if rows[i].HasCoords {
			located = append(located, rows[i])
		}
	}

	sort.SliceStable(located, func(i, j int) bool {
		if located[i].EventCount != located[j].EventCount {
			return located[i].EventCount > located[j].EventCount
		}

		return located[i].FullName < located[j].FullName
	})

	return located
}

// boundingBox is the padded latitude/longitude box covering every located
// place, widened to at least mapMinSpanDegrees on each axis.
func boundingBox(rows []placeRow) geoBox {
	box := geoBox{MinLat: rows[0].Lat, MaxLat: rows[0].Lat, MinLon: rows[0].Lon, MaxLon: rows[0].Lon}
	for i := range rows[1:] {
		row := &rows[i+1]
		box.MinLat = math.Min(box.MinLat, row.Lat)
		box.MaxLat = math.Max(box.MaxLat, row.Lat)
		box.MinLon = math.Min(box.MinLon, row.Lon)
		box.MaxLon = math.Max(box.MaxLon, row.Lon)
	}

	box = growBox(box, mapPadFraction)
	if span := box.MaxLat - box.MinLat; span < mapMinSpanDegrees {
		box = padLat(box, (mapMinSpanDegrees-span)/2)
	}
	if span := box.MaxLon - box.MinLon; span < mapMinSpanDegrees {
		box = padLon(box, (mapMinSpanDegrees-span)/2)
	}

	return box
}

// growBox expands a box by a fraction of its own span on every side.
func growBox(box geoBox, fraction float64) geoBox {
	return padLon(padLat(box, (box.MaxLat-box.MinLat)*fraction), (box.MaxLon-box.MinLon)*fraction)
}

func padLat(box geoBox, by float64) geoBox {
	box.MinLat = math.Max(-maxLatitude, box.MinLat-by)
	box.MaxLat = math.Min(maxLatitude, box.MaxLat+by)

	return box
}

func padLon(box geoBox, by float64) geoBox {
	box.MinLon = math.Max(-maxLongitude, box.MinLon-by)
	box.MaxLon = math.Min(maxLongitude, box.MaxLon+by)

	return box
}

// fitBox picks the plotted area's height for the box's own shape (bounded, so
// neither a world map nor a single valley takes over the page), then widens the
// box on one axis so its shape matches that area exactly — which is what keeps
// the projection undistorted. Longitude degrees are narrower than latitude
// degrees away from the equator, so the comparison is made in cosine-corrected
// units, the same correction the projection itself applies.
func fitBox(box geoBox, frameW float64) (fitted geoBox, frameH int) {
	frameH = clampInt(int(math.Round(frameW*box.aspect())), mapMinHeight, mapMaxHeight)

	// Target: geoW/geoH == frameW/frameH, with geoW in corrected units.
	want := frameW / float64(frameH)
	geoW, geoH := box.correctedWidth(), box.MaxLat-box.MinLat
	switch got := geoW / geoH; {
	case got < want:
		box = padLon(box, (want*geoH-geoW)/box.lonScale()/2)
	case got > want:
		box = padLat(box, (geoW/want-geoH)/2)
	}

	return box, frameH
}

// aspect is the box's height-to-width ratio in corrected units.
func (b geoBox) aspect() float64 {
	width := b.correctedWidth()
	if width == 0 {
		return 1
	}

	return (b.MaxLat - b.MinLat) / width
}

// correctedWidth is the box's longitude span scaled to latitude-comparable
// units at its middle latitude.
func (b geoBox) correctedWidth() float64 {
	return (b.MaxLon - b.MinLon) * b.lonScale()
}

// lonScale is how much a degree of longitude shrinks at the box's middle
// latitude. It never reaches zero, so a box over a pole still projects.
func (b geoBox) lonScale() float64 {
	return math.Max(math.Cos((b.MinLat+b.MaxLat)/2*math.Pi/degreesPerPi), minLonScale)
}

// project maps a coordinate onto the frame. North is up, so latitude counts
// down from the frame's top edge.
func (b geoBox) project(lat, lon float64, frame mapFrame) (x, y float64) {
	x = float64(frame.X) + (lon-b.MinLon)/(b.MaxLon-b.MinLon)*float64(frame.W)
	y = float64(frame.Y) + (b.MaxLat-lat)/(b.MaxLat-b.MinLat)*float64(frame.H)

	return x, y
}

// buildGraticule lays out the latitude and longitude grid lines that fall
// inside the box, labeled in degrees.
func buildGraticule(box geoBox, frame mapFrame) []mapGridLine {
	lines := make([]mapGridLine, 0, 2*mapMaxGridLines)

	latStep := graticuleStep(box.MaxLat - box.MinLat)
	for _, lat := range roundValues(box.MinLat, box.MaxLat, latStep) {
		_, y := box.project(lat, box.MinLon, frame)
		lines = append(lines, mapGridLine{
			X1: frame.X, Y1: int(math.Round(y)),
			X2: frame.X + frame.W, Y2: int(math.Round(y)),
			TextX: frame.X - 4, TextY: int(math.Round(y)) + 4,
			Label: formatLatitude(lat, latStep), Anchor: "end",
		})
	}

	lonStep := graticuleStep(box.MaxLon - box.MinLon)
	for _, lon := range roundValues(box.MinLon, box.MaxLon, lonStep) {
		x, _ := box.project(box.MinLat, lon, frame)
		lines = append(lines, mapGridLine{
			X1: int(math.Round(x)), Y1: frame.Y,
			X2: int(math.Round(x)), Y2: frame.Y + frame.H,
			TextX: int(math.Round(x)), TextY: frame.Y + frame.H + labelBaselineY,
			Label: formatLongitude(lon, lonStep), Anchor: "middle",
		})
	}

	return lines
}

// graticuleStep picks the grid interval for a span.
func graticuleStep(span float64) float64 {
	for _, step := range mapGraticuleSteps {
		if span/step <= float64(mapMaxGridLines) {
			return step
		}
	}

	return mapGraticuleSteps[len(mapGraticuleSteps)-1]
}

// roundValues lists the multiples of step lying strictly inside (from, to).
func roundValues(from, to, step float64) []float64 {
	var values []float64
	for i := math.Ceil(from / step); i*step < to; i++ {
		if value := i * step; value > from {
			values = append(values, value)
		}
	}

	return values
}

// buildMarkers places one marker per located place, labeling as many as fit
// without overlapping. Rows arrive ordered by event count, so the busiest
// places keep their labels when space runs out.
func buildMarkers(rows []placeRow, box geoBox, frame mapFrame) []mapMarker {
	markers := make([]mapMarker, 0, len(rows))
	placed := make([]labelBox, 0, mapLabelMax)
	for i := range rows {
		row := &rows[i]
		x, y := box.project(row.Lat, row.Lon, frame)
		marker := mapMarker{
			CX:    x,
			CY:    y,
			R:     markerRadius(row.EventCount),
			Href:  "#" + row.Anchor,
			Title: markerTitle(row),
		}

		if len(placed) < mapLabelMax {
			text := truncateRunes(row.Name, mapLabelMaxRunes)
			candidate := labelBoxFor(text, x+mapLabelDX, y+mapLabelDY)
			if candidate.fitsIn(frame) && !candidate.overlapsAny(placed) {
				placed = append(placed, candidate)
				marker.Label = text
				marker.LabelX = candidate.X
				marker.LabelY = y + mapLabelDY
			}
		}

		markers = append(markers, marker)
	}

	return markers
}

// markerRadius scales a marker with the number of events recorded at the place.
func markerRadius(events int) float64 {
	return math.Min(mapDotBase+mapDotStep*math.Sqrt(float64(events)), mapDotMax)
}

// markerTitle is the marker's tooltip.
func markerTitle(row *placeRow) string {
	title := row.FullName
	switch row.EventCount {
	case 0:
	case 1:
		title += " · 1 event"
	default:
		title += " · " + strconv.Itoa(row.EventCount) + " events"
	}

	return title
}

// labelBox is a label's estimated footprint, used for collision checks.
type labelBox struct {
	X float64
	Y float64
	W float64
	H float64
}

// labelBoxFor estimates the box a label occupies, given that text cannot be
// measured without a font.
func labelBoxFor(text string, x, y float64) labelBox {
	return labelBox{
		X: x,
		Y: y - mapLabelHeight,
		W: float64(len([]rune(text))) * mapGlyphWidth,
		H: mapLabelHeight,
	}
}

func (l labelBox) fitsIn(frame mapFrame) bool {
	return l.X+l.W <= float64(frame.X+frame.W) && l.Y >= float64(frame.Y)
}

func (l labelBox) overlapsAny(others []labelBox) bool {
	for _, other := range others {
		if l.X < other.X+other.W && other.X < l.X+l.W &&
			l.Y < other.Y+other.H && other.Y < l.Y+l.H {
			return true
		}
	}

	return false
}

// formatLatitude renders a parallel's label, e.g. "42.5°N".
func formatLatitude(lat, step float64) string {
	return formatDegrees(lat, step, "N", "S")
}

// formatLongitude renders a meridian's label, e.g. "71°W".
func formatLongitude(lon, step float64) string {
	return formatDegrees(lon, step, "E", "W")
}

// formatDegrees renders a signed degree value with the hemisphere letter,
// carrying only as many decimals as the grid step needs.
func formatDegrees(value, step float64, positive, negative string) string {
	hemisphere := positive
	switch {
	case value < 0:
		hemisphere = negative
		value = -value
	case value == 0:
		// The equator and the prime meridian belong to no hemisphere.
		hemisphere = ""
	}
	text := strconv.FormatFloat(value, 'f', degreeDecimals(step), 64)
	if strings.Contains(text, ".") {
		text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	}

	return text + "°" + hemisphere
}

// degreeDecimals is how many decimal places a grid step needs to render
// exactly: none for whole degrees, one for halves and tenths, two for
// quarters and hundredths. Labels are always multiples of the step, so this is
// enough to render every one of them without rounding.
func degreeDecimals(step float64) int {
	for decimals := range degreeMaxDecimals {
		scaled := step * math.Pow10(decimals)
		if math.Abs(scaled-math.Round(scaled)) < degreeEpsilon {
			return decimals
		}
	}

	return degreeMaxDecimals
}

// clampInt bounds v to [low, high].
func clampInt(v, low, high int) int {
	return min(max(v, low), high)
}
