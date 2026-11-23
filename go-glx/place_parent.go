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
	"errors"
	"fmt"
	"math"
	"slices"

	"go.yaml.in/yaml/v3"

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
// Place.EffectiveParentID returns the default parent (DefaultParentOf).
// For date-scoped ancestry, use GLXFile.PlacePathsAt or GLXFile.PlaceHasAncestorAt.

// placeFieldParent is the YAML key of a place's parent.
const placeFieldParent = "parent"

var errDuplicatePlaceParent = errors.New("place parent: duplicate parent key")

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
	var bestSpan glxdate.Interval
	for i, p := range periods {
		span, ok := parentPeriodSpan(p.Date)
		if !ok {
			span = glxdate.Interval{Start: math.MinInt, End: math.MinInt}
		}
		if best < 0 || span.End > bestSpan.End || (span.End == bestSpan.End && span.Start >= bestSpan.Start) {
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
// ParentID as its default parent. A nil or empty list clears the history,
// retaining the effective default as the scalar parent.
func (p *Place) SetParentHistory(periods []PlaceParentPeriod) {
	if len(periods) == 0 {
		p.ParentID = p.EffectiveParentID()
		p.ParentHistory = nil

		return
	}
	p.ParentHistory = slices.Clone(periods)
	p.ParentID = DefaultParentOf(periods)
}

// EffectiveParentID returns the current default parent. Temporal history is
// authoritative, including for Places built or edited directly by SDK callers.
// ParentID remains available for scalar parents and compatibility.
func (p *Place) EffectiveParentID() string {
	if p == nil {
		return ""
	}
	if p.HasTemporalParent() {
		return DefaultParentOf(p.ParentHistory)
	}

	return p.ParentID
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
	add(p.EffectiveParentID())
	for _, period := range p.ParentHistory {
		add(period.Value)
	}

	return ids
}

// ParentAt returns the parent place that applied at date, for places whose
// parent changed over time. The rules:
//
//   - a place with a plain (non-temporal) parent, an empty date, or a date
//     that cannot be placed on a timeline gets its effective default parent;
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
		return p.EffectiveParentID()
	}
	at, ok := parentPeriodSpan(date)
	if !ok {
		return p.EffectiveParentID()
	}

	best, bestOverlap := -1, 0
	nearest, nearestGap := -1, math.MaxInt
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
		if overlap := chronologyOverlap(span, at); overlap > bestOverlap {
			best, bestOverlap = i, overlap
		}
		if gap := chronologyGap(span, at); gap < nearestGap {
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

	return ""
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
		parentIndex := -1
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == placeFieldParent {
				if parentIndex >= 0 {
					return errDuplicatePlaceParent
				}
				parentIndex = i
			}
		}
		if i := parentIndex; i >= 0 {
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

// chronologyOverlap counts shared elapsed days. An unbounded overlap ranks
// ahead of a finite one for the single-parent display projection.
func chronologyOverlap(a, b glxdate.Interval) int {
	lo, hi := max(a.Start, b.Start), min(a.End, b.End)
	if lo >= hi {
		return 0
	}
	if lo == math.MinInt || hi == math.MaxInt {
		return math.MaxInt
	}

	return hi - lo
}

// chronologyGap uses the bounds of disjoint half-open spans.
func chronologyGap(a, b glxdate.Interval) int {
	switch {
	case a.End <= b.Start:
		return b.Start - a.End
	case a.Start >= b.End:
		return a.Start - b.End
	default:
		return 0
	}
}

// parentChronology rejects malformed, reversed and empty dates instead of
// giving the parser's recoverable components historical force.
func parentChronology(ds DateString) (glxdate.Date, glxdate.Bounds, bool) {
	d, err := ds.Parse()
	if err != nil {
		return d, glxdate.Bounds{}, false
	}
	b := d.ChronologyBounds(0)
	if !b.Known || b.Reversed || b.Outer.Start >= b.Outer.End {
		return d, b, false
	}
	if d.IsRange() {
		// A nominal date can be preserved by the parser while denoting no
		// day in that calendar (e.g. unqualified February 29, 1900).
		for _, endpoint := range []glxdate.Date{d.Start(), d.End()} {
			if endpoint.IsZero() {
				continue
			}
			x := endpoint.ChronologyBounds(0)
			if !x.Known || x.Outer.Start >= x.Outer.End {
				return d, b, false
			}
		}
	}

	return d, b, true
}

func parentPeriodSpan(ds DateString) (glxdate.Interval, bool) {
	_, b, ok := parentChronology(ds)

	return b.Outer, ok
}

// parentPeriodCore returns days an entry certainly covers. A duration uses
// inclusive actual-calendar boundary days; a coarse point, BET or qualified
// event is not a guaranteed duration. FROM and TO supply one open boundary.
func parentPeriodCore(ds DateString) (glxdate.Interval, bool) {
	_, core, _, coreOK := parentPeriodBounds(ds)

	return core, coreOK
}

func parentPeriodBounds(ds DateString) (outer, core glxdate.Interval, outerOK, coreOK bool) {
	d, b, ok := parentChronology(ds)
	if !ok {
		return glxdate.Interval{}, glxdate.Interval{}, false, false
	}
	switch {
	case d.IsOpenStart():
		end := d.End().ChronologyBounds(0).Outer
		b.Inner = glxdate.Interval{Start: math.MinInt, End: end.Start + 1}
	case d.IsOpenEnded():
		start := d.Start().ChronologyBounds(0).Outer
		b.Inner = glxdate.Interval{Start: start.End - 1, End: math.MaxInt}
	}

	return b.Outer, b.Inner, true, b.Inner.Start < b.Inner.End
}
