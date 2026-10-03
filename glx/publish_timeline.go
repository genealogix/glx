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
	"strconv"
)

// Timeline strip geometry, in user units. The strip is drawn at this fixed
// logical width and scaled to the page by the stylesheet (width: 100%), so a
// long life and a short one are laid out identically and the labels never
// collide at small screen sizes.
const (
	stripWidth       = 720
	stripHeight      = 92
	stripInsetLeft   = 28
	stripInsetRight  = 28
	stripBarY        = 44
	stripBarH        = 10
	stripDotY        = stripBarY + stripBarH/2
	stripDotR        = 5
	stripTickY1      = stripBarY + stripBarH + 6
	stripTickY2      = stripTickY1 + 6
	stripTickLabelY  = stripTickY2 + 13
	stripMinDistinct = 2 // fewer distinct years than this has nothing to plot
)

// stripTickSteps are the year intervals a strip's axis may use, smallest
// first. The first step that yields at most stripMaxTicks labels wins, so the
// axis reads in round numbers (decades, half-centuries) rather than in even
// fractions of an arbitrary range.
var stripTickSteps = []int{1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000}

// stripMaxTicks bounds how many year labels the axis carries, and
// stripMinTickGap is the closest two labels may sit (in user units) before one
// is dropped — the span's own endpoints are always kept, so it is an interior
// label that gives way.
const (
	stripMaxTicks   = 6
	stripMinTickGap = 40
)

// timelineStrip is a laid-out visual timeline: a life bar spanning the
// person's dated events with one dot per event, over an axis of round years.
// Like the family charts, every coordinate is precomputed so the template
// carries no arithmetic.
type timelineStrip struct {
	Label  string // accessible label for the SVG
	Width  int
	Height int
	BarX   int
	BarY   int
	BarW   int
	BarH   int
	Ticks  []stripTick
	Dots   []stripDot
}

// stripTick is one year gridline label on the axis.
type stripTick struct {
	X     int
	Y1    int
	Y2    int
	TextY int
	Label string
}

// stripDot is one dated event plotted on the strip.
type stripDot struct {
	X     int
	Y     int
	R     int
	Title string // tooltip text: "1850 · Birth · Boston"
}

// buildTimelineStrip lays out the visual timeline for a person's rows, or
// returns nil when there is nothing meaningful to plot — no dated rows, or
// every row in the same year, where a proportional strip would say nothing the
// list below it does not.
func buildTimelineStrip(rows []timelineRow) *timelineStrip {
	first, last, distinct := stripYearRange(rows)
	if distinct < stripMinDistinct {
		return nil
	}

	strip := &timelineStrip{
		Label:  fmt.Sprintf("Timeline of dated events from %s to %s", displayYear(first), displayYear(last)),
		Width:  stripWidth,
		Height: stripHeight,
		BarX:   stripInsetLeft,
		BarY:   stripBarY,
		BarW:   stripWidth - stripInsetLeft - stripInsetRight,
		BarH:   stripBarH,
	}

	strip.Ticks = stripTicks(first, last)

	for _, row := range rows {
		if row.Year == 0 {
			continue
		}
		strip.Dots = append(strip.Dots, stripDot{
			X:     stripX(row.Year, first, last),
			Y:     stripDotY,
			R:     stripDotR,
			Title: stripDotTitle(row),
		})
	}

	return strip
}

// stripYearRange reports the earliest and latest plotted year and how many
// distinct years the rows cover.
func stripYearRange(rows []timelineRow) (first, last, distinct int) {
	seen := map[int]bool{}
	for _, row := range rows {
		if row.Year == 0 {
			continue
		}
		if len(seen) == 0 || row.Year < first {
			first = row.Year
		}
		if len(seen) == 0 || row.Year > last {
			last = row.Year
		}
		seen[row.Year] = true
	}

	return first, last, len(seen)
}

// stripX projects a year onto the strip's bar, rounding to the nearest whole
// unit. first and last are known to differ, because a single-year range never
// produces a strip.
func stripX(year, first, last int) int {
	offset := float64(year-first) / float64(last-first) * float64(stripWidthInner())

	return stripInsetLeft + int(math.Round(offset))
}

// stripWidthInner is the drawable width of the bar.
func stripWidthInner() int {
	return stripWidth - stripInsetLeft - stripInsetRight
}

// stripTicks lays out the axis labels: the span's endpoints, plus the round
// years between them that are far enough from their neighbors to stay
// legible. An interior label too close to the one before it — or to the
// closing endpoint — is dropped rather than drawn overlapping.
func stripTicks(first, last int) []stripTick {
	var ticks []stripTick
	add := func(year int) {
		ticks = append(ticks, stripTick{
			X:     stripX(year, first, last),
			Y1:    stripTickY1,
			Y2:    stripTickY2,
			TextY: stripTickLabelY,
			Label: displayYear(year),
		})
	}

	add(first)
	lastX := stripX(last, first, last)
	for _, year := range stripTickYears(first, last) {
		x := stripX(year, first, last)
		if x-ticks[len(ticks)-1].X < stripMinTickGap || lastX-x < stripMinTickGap {
			continue
		}
		add(year)
	}
	add(last)

	return ticks
}

// stripTickYears picks the round years that fall strictly inside the span. The
// step is the smallest one that keeps the label count within stripMaxTicks, so
// the axis reads in decades or half-centuries rather than in even fractions of
// an arbitrary range.
func stripTickYears(first, last int) []int {
	step := stripTickSteps[len(stripTickSteps)-1]
	for _, candidate := range stripTickSteps {
		if (last-first)/candidate+1 <= stripMaxTicks {
			step = candidate

			break
		}
	}

	var years []int
	// Round the first labeled year up to the next multiple of the step.
	for year := (first/step)*step + step; year < last; year += step {
		years = append(years, year)
	}

	return years
}

// stripDotTitle is the tooltip for one plotted event.
func stripDotTitle(row timelineRow) string {
	title := row.Date
	if title == "" {
		title = displayYear(row.Year)
	}
	if row.Label != "" {
		title += " · " + row.Label
	}
	if row.Detail != "" {
		title += " · " + row.Detail
	}

	return title
}

// displayYear renders a plotted year the way the rest of the site does, with
// negative years shown as BCE.
func displayYear(year int) string {
	if year < 0 {
		return strconv.Itoa(-year) + " BCE"
	}

	return strconv.Itoa(year)
}
