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

package glxdate

import (
	"math"
)

// Interval is a half-open span of civil days. MinInt and MaxInt represent open ends.
type Interval struct{ Start, End int }

// Intersects reports whether the spans share at least one day.
func (s Interval) Intersects(o Interval) bool { return s.Start < o.End && o.Start < s.End }

// Contains reports whether s covers all of o.
func (s Interval) Contains(o Interval) bool { return s.Start <= o.Start && s.End >= o.End }

// Bounds distinguishes where a date could fall from the period it certainly covers.
// Inner is empty for uncertain dates, including open ends and reversed ranges.
type Bounds struct {
	Outer     Interval
	Inner     Interval
	Known     bool
	Reversed  bool
	Uncertain bool
}

// Timing returns bounds for a property's period of validity. Approximation
// qualifiers retain their written precision; they never prove overlap.
func (d Date) Timing() Bounds { return d.bounds(0) }

// ValueBounds returns bounds for comparing date values, widening ABT, EST and
// CAL by approximationYears on either side. Negative widths are treated as zero.
func (d Date) ValueBounds(approximationYears int) Bounds { return d.bounds(max(0, approximationYears)) }

func (d Date) bounds(width int) Bounds {
	v := d.val()
	if d.IsZero() {
		return Bounds{}
	}
	start, known := pointBounds(d.Start())
	if d.IsOpenStart() {
		start, known = pointBounds(d.End())
	}
	b := Bounds{Outer: start, Known: known}
	if d.IsRange() {
		return d.rangeBounds(b)
	}
	if !known {
		return b
	}
	switch v.qualifier {
	case QualifierBefore:
		b.Outer = Interval{math.MinInt, start.Start}
		b.Uncertain = true
	case QualifierAfter:
		b.Outer = Interval{start.End, math.MaxInt}
		b.Uncertain = true
	case QualifierAbout, QualifierCalculated, QualifierEstimated:
		b.Uncertain = true
		if width > 0 {
			width = min(width, MaxApproximationYears)
			b.Outer.Start -= width * civilYearDays
			b.Outer.End += width * civilYearDays
		}

	case QualifierInterpreted:
		b.Uncertain = true
	default:
		if d.Precision() == PrecisionDay && v.start.exact {
			b.Inner = start
		}
	}
	if !v.start.exact {
		b.Uncertain = true
		b.Inner = Interval{}
	}

	return b
}

func (d Date) rangeBounds(b Bounds) Bounds {
	if d.IsOpenStart() {
		b.Outer.Start = math.MinInt
		b.Uncertain = true

		return b
	}
	if d.IsOpenEnded() {
		b.Outer.End = math.MaxInt
		b.Uncertain = true

		return b
	}
	end, endKnown := pointBounds(d.End())
	if !b.Known && !endKnown {
		return b
	}
	b.Known = true
	if d.Start().Year() == 0 {
		b.Outer.Start = math.MinInt
	}
	if !endKnown {
		end.End = math.MaxInt
	}
	b.Reversed = endKnown && b.Outer.Start >= end.End
	if b.Reversed {
		b.Outer = Interval{math.MinInt, math.MaxInt}
		b.Uncertain = true

		return b
	}
	start := b.Outer
	b.Outer.End = end.End
	b.Uncertain = !endKnown || d.Start().Year() == 0 || !d.val().start.exact || !d.val().end.exact
	// BET describes an uncertain point; FROM ... TO describes a duration.
	if d.val().rng == rangeBetween {
		b.Uncertain = true
	}
	if !b.Uncertain {
		b.Inner = Interval{start.End - 1, end.Start + 1}
	}

	return b
}

// MaxApproximationYears bounds caller-supplied widths to prevent integer overflow.
const MaxApproximationYears = 10000

// Use the parser's maximum month lengths, including February 29. These are
// ordinal civil-date keys, not elapsed days: no calendar conversion is implied,
// and a preserved Julian February 29 must not normalize to Gregorian March 1.
const civilYearDays = 366

func pointBounds(d Date) (Interval, bool) {
	if d.Year() == 0 {
		return Interval{}, false
	}
	y := d.Year()
	if y < 0 {
		y++
	} // astronomical year zero is 1 BCE
	m, hasMonth := d.Month()
	day, hasDay := d.Day()
	start := y * civilYearDays
	if !hasMonth {
		return Interval{start, start + civilYearDays}, true
	}
	for month := 1; month < m; month++ {
		start += daysInMonth[month]
	}
	if !hasDay {
		return Interval{start, start + daysInMonth[m]}, true
	}
	start += day - 1

	return Interval{start, start + 1}, true
}

// Overlap describes whether two recorded periods were certainly simultaneous.
type Overlap string

// Timing-overlap verdicts.
const (
	NoOverlap       Overlap = "none"
	PossibleOverlap Overlap = "possible"
	DefiniteOverlap Overlap = "definite"
)

// CompareTiming compares periods without widening approximations. Undated
// values cannot contradict a history. Closed periods sharing an endpoint touch.
func CompareTiming(a, b Date) Overlap {
	x, y := a.Timing(), b.Timing()
	if !x.Known || !y.Known {
		return NoOverlap
	}
	if !x.Outer.Intersects(y.Outer) {
		return NoOverlap
	}
	if !x.Reversed && !y.Reversed && touchingPeriods(a, b) {
		return NoOverlap
	}

	if x.Uncertain || y.Uncertain || a.Calendar() != b.Calendar() || a.CalendarName() != b.CalendarName() {
		return PossibleOverlap
	}
	if (x.Inner.Start < x.Inner.End && x.Inner.Contains(y.Outer)) ||
		(y.Inner.Start < y.Inner.End && y.Inner.Contains(x.Outer)) ||
		(x.Inner.Start < x.Inner.End && y.Inner.Start < y.Inner.End && x.Inner.Intersects(y.Inner)) {
		return DefiniteOverlap
	}

	return PossibleOverlap
}

func touchingPeriods(a, b Date) bool {
	if !a.IsRange() || !b.IsRange() || a.Start().Equal(a.End()) || b.Start().Equal(b.End()) {
		return false
	}

	return (!a.End().IsZero() && a.End().Equal(b.Start())) || (!b.End().IsZero() && b.End().Equal(a.Start()))
}
