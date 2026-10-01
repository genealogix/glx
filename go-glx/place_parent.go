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
	"fmt"
	"math"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// Temporal place parents (#225).
//
// A place's `parent` is either a plain place ID or a dated list:
//
//	parent:
//	  - value: place-indiana-territory
//	    date: "FROM 1811 TO 1816-12-10"
//	  - value: place-indiana
//	    date: "FROM 1816-12-11"
//
// Place.ParentID always holds the *default* parent (DefaultParentOf), which is
// what date-unaware code walks. Code that has a date in hand (an event's date)
// calls Place.ParentAt to get the parent that applied then.

// placeFieldParent is the YAML key of a place's parent.
const placeFieldParent = "parent"

// PlaceParentPeriod is one entry of a temporal place parent: Value is the
// parent place ID, Date the period it applied (any GLX date; typically a
// FROM/TO range). An entry with no Date is undated and acts as the default.
type PlaceParentPeriod struct {
	Value string     `yaml:"value"`
	Date  DateString `yaml:"date,omitempty"`
}

// DefaultParentOf returns the default parent of a temporal parent list — the
// parent date-unaware code uses:
//
//  1. the first undated entry, if any (an undated entry is an explicit default);
//  2. otherwise the most recent entry: the one whose period ends latest, an
//     open-ended period ("FROM 1816-12-11") counting as ending last, with a
//     later start breaking ties and then the later position in the list.
//
// Entries whose date cannot be placed on a timeline rank below every dated
// one. It returns "" for an empty list.
func DefaultParentOf(periods []PlaceParentPeriod) string {
	for _, p := range periods {
		if p.Date == "" {
			return p.Value
		}
	}

	best := -1
	var bestSpan daySpan
	for i, p := range periods {
		span, ok := parentPeriodSpan(p.Date)
		if !ok {
			span = daySpan{lo: math.MinInt64, hi: math.MinInt64}
		}
		if best < 0 || span.hi > bestSpan.hi || (span.hi == bestSpan.hi && span.lo >= bestSpan.lo) {
			best, bestSpan = i, span
		}
	}
	if best < 0 {
		return ""
	}

	return periods[best].Value
}

// HasTemporalParent reports whether the place records its parent as a dated
// list rather than a single place ID.
func (p *Place) HasTemporalParent() bool {
	return p != nil && len(p.ParentHistory) > 0
}

// SetParentHistory replaces the place's temporal parent list and recomputes
// ParentID as its default parent. A nil or empty list clears the history and
// leaves ParentID as it is.
func (p *Place) SetParentHistory(periods []PlaceParentPeriod) {
	if len(periods) == 0 {
		p.ParentHistory = nil

		return
	}
	p.ParentHistory = periods
	p.ParentID = DefaultParentOf(periods)
}

// ParentIDs returns every distinct parent place ID the place has ever had,
// default parent first, then the temporal entries in list order. Use it for
// date-independent questions over all parent edges: reference checks, cycle
// detection, "is X ever an ancestor of Y".
func (p *Place) ParentIDs() []string {
	if p == nil {
		return nil
	}
	var ids []string
	seen := make(map[string]struct{}, len(p.ParentHistory)+1)
	add := func(id string) {
		if id == "" {
			return
		}
		if _, dup := seen[id]; dup {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	add(p.ParentID)
	for _, period := range p.ParentHistory {
		add(period.Value)
	}

	return ids
}

// ParentAt returns the parent place that applied at date, for places whose
// parent changed over time. The rules:
//
//   - a place with a plain (non-temporal) parent, an empty date, or a date
//     that cannot be placed on a timeline gets ParentID (the default parent);
//   - otherwise the dated entry whose period overlaps date the most wins
//     ("1816" falls mostly in "FROM 1811 TO 1816-12-10"); ties go to the
//     earlier entry in the list;
//   - when no dated period overlaps date, an undated entry is used if there
//     is one, else the period nearest to date (a place's earliest parent for
//     a date before its first period).
func (p *Place) ParentAt(date DateString) string {
	if p == nil {
		return ""
	}
	if len(p.ParentHistory) == 0 || date == "" {
		return p.ParentID
	}
	at, ok := pointDateSpan(date)
	if !ok {
		return p.ParentID
	}

	best, bestOverlap := -1, int64(0)
	nearest, nearestGap := -1, int64(math.MaxInt64)
	undated := -1
	for i, period := range p.ParentHistory {
		if period.Date == "" {
			if undated < 0 {
				undated = i
			}

			continue
		}
		span, ok := parentPeriodSpan(period.Date)
		if !ok {
			continue
		}
		if overlap := span.overlap(at); overlap > bestOverlap {
			best, bestOverlap = i, overlap
		}
		if gap := span.gap(at); gap < nearestGap {
			nearest, nearestGap = i, gap
		}
	}

	switch {
	case best >= 0:
		return p.ParentHistory[best].Value
	case undated >= 0:
		return p.ParentHistory[undated].Value
	case nearest >= 0:
		return p.ParentHistory[nearest].Value
	}

	return p.ParentID
}

// ReplaceParentRef rewrites every parent reference equal to oldID (the default
// parent and each temporal entry) to newID and returns how many were changed.
func (p *Place) ReplaceParentRef(oldID, newID string) int {
	if p == nil || oldID == "" {
		return 0
	}
	count := 0
	if len(p.ParentHistory) > 0 {
		for i := range p.ParentHistory {
			if p.ParentHistory[i].Value == oldID {
				p.ParentHistory[i].Value = newID
				count++
			}
		}
		p.ParentID = DefaultParentOf(p.ParentHistory)

		return count
	}
	if p.ParentID == oldID {
		p.ParentID = newID
		count++
	}

	return count
}

// PlaceHasAncestor reports whether ancestorID is reachable from placeID by
// following parent edges of any period (so a county that moved from a
// territory to a state is a descendant of both). Cycle-safe.
func (glx *GLXFile) PlaceHasAncestor(placeID, ancestorID string) bool {
	if glx == nil || placeID == "" || ancestorID == "" {
		return false
	}
	visited := map[string]bool{placeID: true}
	queue := []string{placeID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		place, ok := glx.Places[current]
		if !ok || place == nil {
			continue
		}
		for _, parent := range place.ParentIDs() {
			if parent == ancestorID {
				return true
			}
			if !visited[parent] {
				visited[parent] = true
				queue = append(queue, parent)
			}
		}
	}

	return false
}

// UnmarshalYAML accepts `parent` as either a place ID or a list of
// {value, date} entries. For the list form ParentHistory is filled and
// ParentID is set to the default parent.
func (p *Place) UnmarshalYAML(value *yaml.Node) error {
	type placeAlias Place // distinct type: no UnmarshalYAML, so no recursion

	node := value
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}

	var history []PlaceParentPeriod
	hasHistory := false
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value != placeFieldParent {
				continue
			}
			parentNode := node.Content[i+1]
			if parentNode.Kind == yaml.AliasNode && parentNode.Alias != nil {
				parentNode = parentNode.Alias
			}
			if parentNode.Kind == yaml.SequenceNode {
				if err := parentNode.Decode(&history); err != nil {
					return fmt.Errorf("place parent: expected a place ID or a list of {value, date} entries: %w", err)
				}
				hasHistory = true
				// Decode the rest without the list, which the alias's string
				// ParentID cannot hold.
				stripped := *node
				stripped.Content = make([]*yaml.Node, 0, len(node.Content)-2)
				stripped.Content = append(stripped.Content, node.Content[:i]...)
				stripped.Content = append(stripped.Content, node.Content[i+2:]...)
				node = &stripped
			}

			break
		}
	}

	var aux placeAlias
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*p = Place(aux)
	if hasHistory {
		p.SetParentHistory(history)
	}

	return nil
}

// MarshalYAML writes `parent` as a plain place ID, or as the dated list when
// ParentHistory is non-empty, keeping the field in its usual position.
//
//nolint:gocritic // value receiver so both Place and *Place marshal through it
func (p Place) MarshalYAML() (any, error) {
	type placeAlias Place // distinct type: no MarshalYAML, so no recursion

	aux := placeAlias(p)
	if len(p.ParentHistory) == 0 {
		return aux, nil
	}
	// Make sure the parent key is emitted, then swap its value for the list.
	aux.ParentID = placeFieldParent

	var node yaml.Node
	if err := node.Encode(aux); err != nil {
		return nil, err
	}
	var list yaml.Node
	if err := list.Encode(p.ParentHistory); err != nil {
		return nil, err
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == placeFieldParent {
			node.Content[i+1] = &list

			break
		}
	}

	return &node, nil
}

// daySpan is an inclusive interval of days since the Unix epoch.
// math.MinInt64 / math.MaxInt64 stand for an open start / end.
type daySpan struct {
	lo, hi int64
}

// overlap returns the number of days two spans share (0 when disjoint).
// The receiver may be open-ended; other must be finite.
func (s daySpan) overlap(other daySpan) int64 {
	lo := max(s.lo, other.lo)
	hi := min(s.hi, other.hi)
	if hi < lo {
		return 0
	}

	return hi - lo + 1
}

// gap returns the number of days between two disjoint spans (0 when they
// overlap). The receiver may be open-ended; other must be finite.
func (s daySpan) gap(other daySpan) int64 {
	switch {
	case s.hi < other.lo:
		return other.lo - s.hi
	case s.lo > other.hi:
		return s.lo - other.hi
	}

	return 0
}

// pointDateSpan returns the days a date may denote, ignoring qualifiers: "1816"
// is the whole of 1816, "FROM 1811 TO 1816" spans both years, "BEF 1820" is
// treated as 1820. Used for the date being looked up. ok is false for dates
// with no year, or in a calendar whose years are not Gregorian-like.
func pointDateSpan(ds DateString) (daySpan, bool) {
	d, _ := ds.Parse()
	if !d.IsRange() {
		return dateBounds(d)
	}
	first := d.Start()
	if d.IsOpenStart() {
		first = d.End()
	}
	last := d.End()
	if d.IsOpenEnded() {
		last = d.Start()
	}
	lo, okLo := dateBounds(first)
	hi, okHi := dateBounds(last)
	if !okLo || !okHi {
		return daySpan{}, false
	}

	return daySpan{lo: lo.lo, hi: hi.hi}, true
}

// parentPeriodSpan returns the days a parent period may cover: "FROM a TO b"
// and "BET a AND b" run from the first day of a to the last day of b; an open
// "FROM a" / "AFT a" never ends; "TO b" / "BEF b" has no start; a single date
// covers just that date.
func parentPeriodSpan(ds DateString) (daySpan, bool) {
	d, _ := ds.Parse()
	if d.IsRange() {
		span := daySpan{lo: math.MinInt64, hi: math.MaxInt64}
		if !d.IsOpenStart() {
			start, ok := dateBounds(d.Start())
			if !ok {
				return daySpan{}, false
			}
			span.lo = start.lo
		}
		if !d.IsOpenEnded() {
			end, ok := dateBounds(d.End())
			if !ok {
				return daySpan{}, false
			}
			span.hi = end.hi
		}

		return span, true
	}

	bounds, ok := dateBounds(d)
	if !ok {
		return daySpan{}, false
	}
	switch d.Qualifier() {
	case glxdate.QualifierBefore:
		return daySpan{lo: math.MinInt64, hi: bounds.lo - 1}, true
	case glxdate.QualifierAfter:
		return daySpan{lo: bounds.hi + 1, hi: math.MaxInt64}, true
	default:
		return bounds, true
	}
}

// parentPeriodCore returns the days a parent period certainly covers: the
// period with its imprecise boundaries shrunk inward, so "FROM 1840 TO 1972"
// and "FROM 1972" (which share the year 1972 only at year precision) do not
// overlap. ok is false when the period is empty or cannot be placed.
func parentPeriodCore(ds DateString) (daySpan, bool) {
	d, _ := ds.Parse()
	if !d.IsRange() {
		return parentPeriodSpan(ds)
	}
	span := daySpan{lo: math.MinInt64, hi: math.MaxInt64}
	if !d.IsOpenStart() {
		start, ok := dateBounds(d.Start())
		if !ok {
			return daySpan{}, false
		}
		span.lo = start.hi
	}
	if !d.IsOpenEnded() {
		end, ok := dateBounds(d.End())
		if !ok {
			return daySpan{}, false
		}
		span.hi = end.lo
	}
	if span.hi < span.lo {
		return daySpan{}, false
	}

	return span, true
}

// dateBounds returns the first and last day a single (non-range) date can
// denote at its precision: "1816" → 1 Jan–31 Dec 1816, "1816-12" → December.
func dateBounds(d glxdate.Date) (daySpan, bool) {
	if d.IsZero() || (d.Calendar() != glxdate.CalendarGregorian && d.Calendar() != glxdate.CalendarJulian) {
		return daySpan{}, false
	}
	year := d.Year()
	if year == 0 {
		return daySpan{}, false
	}
	month, hasMonth := d.Month()
	if !hasMonth {
		return daySpan{lo: dayNumber(year, 1, 1), hi: dayNumber(year+1, 1, 1) - 1}, true
	}
	day, hasDay := d.Day()
	if !hasDay {
		return daySpan{lo: dayNumber(year, month, 1), hi: dayNumber(year, month+1, 1) - 1}, true
	}
	n := dayNumber(year, month, day)

	return daySpan{lo: n, hi: n}, true
}

// dayNumber returns days since the Unix epoch for a proleptic Gregorian date.
// time.Date normalizes month 13 to January of the next year.
func dayNumber(year, month, day int) int64 {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay
}

// secondsPerDay converts Unix seconds to days.
const secondsPerDay = 24 * 60 * 60
