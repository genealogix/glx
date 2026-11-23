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
	"cmp"
	"math"
	"slices"

	"github.com/genealogix/glx/go-glx/glxdate"
)

type placePathEdge struct {
	parent string
	window glxdate.Interval
}

type placePathMode uint8

const (
	placeDefaultPath placePathMode = iota
	placePossiblePath
	placeFactualPossiblePath
	placeCertainPath
	placeUnplacedDate
)

// PlacePathsAt returns all feasible paths starting with placeID, proceeding
// from the most specific place toward its ancestors. Every hop of a dated
// path must be supported by the same day. Missing periods or references can
// leave a path incomplete; a malformed date permits only scalar edges.
// A missing date follows the effective defaults instead.
func (glx *GLXFile) PlacePathsAt(placeID string, date DateString) [][]string {
	mode, window := placeDateScope(date)
	var paths [][]string
	glx.walkPlacePaths(placeID, mode, window, func(path []string, _ glxdate.Interval) {
		if !slices.ContainsFunc(paths, func(previous []string) bool { return slices.Equal(previous, path) }) {
			paths = append(paths, slices.Clone(path))
		}
	})

	return paths
}

// PlacePathAt returns the longest unambiguous prefix of the date-scoped
// paths. Use it to project a single historical label without inventing an
// ancestor for an ambiguous date.
func (glx *GLXFile) PlacePathAt(placeID string, date DateString) []string {
	path, _ := glx.placePathProjection(placeID, date)

	return path
}

func (glx *GLXFile) placePathProjection(placeID string, date DateString) ([]string, bool) {
	mode, window := placeDateScope(date)
	var prefix []string
	cycle := glx.walkPlacePaths(placeID, mode, window, func(path []string, _ glxdate.Interval) {
		if prefix == nil {
			prefix = slices.Clone(path)

			return
		}
		i := 0
		for i < min(len(prefix), len(path)) && prefix[i] == path[i] {
			i++
		}
		prefix = prefix[:i]
	})

	return prefix, cycle
}

// PlaceHasAncestorAt reports whether every possible day of the dated
// observation supports the ancestor, with no competing path lacking it.
// An empty date does not imply the default parent was historically true.
// Use PlaceHasAncestor for structural, all-period reachability instead.
func (glx *GLXFile) PlaceHasAncestorAt(placeID, ancestorID string, date DateString) bool {
	certain, _ := glx.placeAncestorForDates(placeID, ancestorID, date)

	return certain
}

// PlaceMayHaveAncestorAt reports whether any date-supported path reaches the
// ancestor. Unlike PlaceHasAncestorAt, it permits ambiguous paths.
func (glx *GLXFile) PlaceMayHaveAncestorAt(placeID, ancestorID string, date DateString) bool {
	if ancestorID == "" || placeID == ancestorID {
		return false
	}
	mode, window := placeDateScope(date)
	if mode == placeDefaultPath {
		mode = placePossiblePath
	}
	found := false
	glx.walkPlacePaths(placeID, mode, window, func(path []string, _ glxdate.Interval) {
		found = found || slices.Contains(path, ancestorID)
	})

	return found
}

// placeAncestorForDates tests factual refinement in the shared observation
// window. Undated facts do not imply the default parent was historically true.
func (glx *GLXFile) placeAncestorForDates(placeID, ancestorID string, dates ...DateString) (certain, possible bool) {
	window := glxdate.Interval{Start: math.MinInt, End: math.MaxInt}
	hasUnplacedDate := false
	for _, date := range dates {
		if date == "" {
			continue
		}
		bounds, ok := parentPeriodSpan(date)
		if !ok {
			hasUnplacedDate = true

			continue
		}
		window, ok = intersectPlaceWindows(window, bounds)
		if !ok {
			return false, false
		}
	}
	if hasUnplacedDate {
		if certain, _ := glx.placeAncestorIn(placeID, ancestorID, placeUnplacedDate, window); certain {
			return true, true
		}
		// An unplaced observation cannot confirm a temporal path but cannot
		// rule one out, either. Keep the restrictions from any placed dates.
		_, possible := glx.placeAncestorIn(placeID, ancestorID, placeFactualPossiblePath, window)

		return false, possible
	}

	return glx.placeAncestorIn(placeID, ancestorID, placeFactualPossiblePath, window)
}

func (glx *GLXFile) placeAncestorIn(placeID, ancestorID string, mode placePathMode, window glxdate.Interval) (certain, possible bool) {
	if ancestorID == "" || placeID == ancestorID {
		return false, false
	}
	allPathsMatch := true
	glx.walkPlacePaths(placeID, mode, window, func(path []string, _ glxdate.Interval) {
		matches := slices.Contains(path, ancestorID)
		possible = possible || matches
		allPathsMatch = allPathsMatch && matches
	})
	if !possible || !allPathsMatch {
		return false, possible
	}
	if mode != placeFactualPossiblePath {
		return true, true
	}
	var matches []glxdate.Interval
	glx.walkPlacePaths(placeID, placeCertainPath, window, func(path []string, window glxdate.Interval) {
		if slices.Contains(path, ancestorID) {
			matches = append(matches, window)
		}
	})

	return len(subtractPlaceWindows(window, matches)) == 0, true
}

func placeDateScope(date DateString) (placePathMode, glxdate.Interval) {
	unbounded := glxdate.Interval{Start: math.MinInt, End: math.MaxInt}
	if date == "" {
		return placeDefaultPath, unbounded
	}
	if window, ok := parentPeriodSpan(date); ok {
		return placePossiblePath, window
	}

	return placeUnplacedDate, unbounded
}

// walkPlacePaths uses path-local cycle detection: different branches may
// converge on the same parent through different date windows.
func (glx *GLXFile) walkPlacePaths(placeID string, mode placePathMode, window glxdate.Interval, visit func([]string, glxdate.Interval)) bool {
	if glx == nil || placeID == "" {
		return false
	}
	cycle := false
	var walk func([]string, glxdate.Interval)
	walk = func(path []string, window glxdate.Interval) {
		place := glx.Places[path[len(path)-1]]
		for _, edge := range place.pathEdges(mode, window) {
			closesCycle := edge.parent != "" && slices.Contains(path, edge.parent)
			if edge.parent == "" || closesCycle {
				cycle = cycle || closesCycle
				visit(path, edge.window)

				continue
			}
			walk(append(path, edge.parent), edge.window)
		}
	}
	walk([]string{placeID}, window)

	return cycle
}

func (p *Place) pathEdges(mode placePathMode, window glxdate.Interval) []placePathEdge {
	if !p.HasTemporalParent() || mode == placeDefaultPath {
		return []placePathEdge{{p.EffectiveParentID(), window}}
	}
	if mode == placeUnplacedDate {
		return []placePathEdge{{window: window}}
	}

	var edges []placePathEdge
	var occupied []glxdate.Interval
	var fallback string
	haveFallback := false
	for _, period := range p.ParentHistory {
		if period.Date == "" {
			fallback, haveFallback = period.Value, true

			break
		}
	}
	for _, period := range p.ParentHistory {
		if period.Date == "" {
			continue
		}
		outer, core, outerOK, coreOK := parentPeriodBounds(period.Date)
		if mode == placeFactualPossiblePath && !outerOK && unsupportedPlaceParentCalendar(period.Date) {
			// There may be a date for this edge, but without a conversion
			// we cannot narrow the window any further or exclude other edges.
			edges = append(edges, placePathEdge{period.Value, window})

			continue
		}
		span, known := outer, outerOK
		taken, takenOK := core, coreOK
		if mode == placeCertainPath {
			span, known = core, coreOK
		}
		if mode == placeCertainPath || !haveFallback {
			taken, takenOK = outer, outerOK
		}
		if clipped, ok := intersectPlaceWindows(window, span); known && ok {
			edges = append(edges, placePathEdge{period.Value, clipped})
		}
		if clipped, ok := intersectPlaceWindows(window, taken); takenOK && ok {
			occupied = append(occupied, clipped)
		}
	}
	for _, remaining := range subtractPlaceWindows(window, occupied) {
		edges = append(edges, placePathEdge{fallback, remaining})
	}

	return mergePlacePathEdges(edges)
}

func unsupportedPlaceParentCalendar(date DateString) bool {
	d, err := date.Parse()
	if err != nil || d.Timing().Reversed {
		return false
	}

	return d.Calendar() != glxdate.CalendarGregorian && d.Calendar() != glxdate.CalendarJulian
}

// Coalesce overlapping or neighboring windows with the same immediate parent.
// Identical edges in a history must not expand the same downstream path twice.
func mergePlacePathEdges(edges []placePathEdge) []placePathEdge {
	if len(edges) < 2 {
		return edges
	}
	order := make(map[string]int, len(edges))
	for i, edge := range edges {
		if _, exists := order[edge.parent]; !exists {
			order[edge.parent] = i
		}
	}
	slices.SortFunc(edges, func(a, b placePathEdge) int {
		if a.parent != b.parent {
			return cmp.Compare(order[a.parent], order[b.parent])
		}

		return cmp.Compare(a.window.Start, b.window.Start)
	})
	n := 1
	for _, edge := range edges[1:] {
		previous := &edges[n-1]
		if edge.parent == previous.parent && edge.window.Start <= previous.window.End {
			previous.window.End = max(previous.window.End, edge.window.End)

			continue
		}
		edges[n] = edge
		n++
	}

	return edges[:n]
}

func intersectPlaceWindows(a, b glxdate.Interval) (glxdate.Interval, bool) {
	clipped := glxdate.Interval{Start: max(a.Start, b.Start), End: min(a.End, b.End)}

	return clipped, clipped.Start < clipped.End
}

func subtractPlaceWindows(window glxdate.Interval, occupied []glxdate.Interval) []glxdate.Interval {
	slices.SortFunc(occupied, func(a, b glxdate.Interval) int {
		return cmp.Compare(a.Start, b.Start)
	})
	var remaining []glxdate.Interval
	for _, taken := range occupied {
		if taken.Start > window.Start {
			remaining = append(remaining, glxdate.Interval{Start: window.Start, End: taken.Start})
		}
		window.Start = max(window.Start, taken.End)
	}
	if window.Start < window.End {
		remaining = append(remaining, window)
	}

	return remaining
}
