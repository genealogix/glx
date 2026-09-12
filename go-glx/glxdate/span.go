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

import "math"

// Span is the inclusive run of days a [Date] denotes. It lets a caller ask
// how two dates relate in time without re-deriving the qualifier and
// precision rules at every call site. Build one with [Date.Span].
//
// A date maps to its span as follows:
//
//   - A simple date covers its written precision: "1900" is the whole year,
//     "1900-03" the whole month, "1900-03-15" the single day. ABT, EST, CAL
//     and INT do not widen a date; they denote the same span as the bare
//     date.
//   - BEF and AFT exclude the named date and are open on the far side:
//     "AFT 1900" runs from 1901-01-01 onward, "AFT 1900-03-15" from
//     1900-03-16, "BEF 1900-03" up to 1900-02-28. This is the reading the
//     analyze command already applies to relationship windows and death
//     years. The step to the neighbouring day follows the date's own
//     calendar, so "JULIAN AFT 1900-02-28" starts on the Julian leap day
//     1900-02-29 while the Gregorian "AFT 1900-02-28" starts on March 1,
//     and 1 BCE is followed directly by 1 CE.
//   - FROM…TO and BET…AND include both endpoints. "FROM 1870" is open at the
//     end and "TO 1860" is open at the start.
//   - A side whose year cannot be determined (free text, an empty string, a
//     range endpoint with no year) is unbounded. A range whose end precedes
//     its start spans all time rather than nothing, so a date the library
//     cannot interpret is never silently dropped from a comparison.
//   - No calendar conversion is performed: a Julian and a Gregorian date are
//     compared on the same number line, as everywhere else in the library.
//     Hebrew and French Republican dates keep their month names raw, so
//     they span their whole year.
//
// The relations are Allen's interval algebra reduced to what inclusive days
// need: [Span.Precedes], [Span.Overlaps], [Span.During] and [Span.Equals].
// For any two spans X and Y exactly one of X.Precedes(Y), Y.Precedes(X),
// X.Overlaps(Y), Y.Overlaps(X), X.During(Y), Y.During(X) and X.Equals(Y)
// holds. Allen's "meets" has no place on inclusive days: two spans that
// touch on a shared day overlap, and two on adjacent days with nothing
// shared precede one another. Allen's "starts" and "finishes" fold into
// During. [Span.Intersects] answers the question most callers have, whether
// two spans share any day at all.
type Span struct {
	lo, hi int
}

// allTime is the span of every day, used for a date that cannot be placed.
var allTime = Span{lo: math.MinInt, hi: math.MaxInt}

// Span returns the inclusive run of days the date denotes; see [Span].
func (d Date) Span() Span {
	if d.IsRange() {
		return d.rangeSpan()
	}
	if d.Year() == 0 {
		return allTime
	}
	first, last := d.pointBounds()
	switch d.Qualifier() {
	case QualifierBefore:
		return Span{lo: math.MinInt, hi: first.prev(d.Calendar()).key()}
	case QualifierAfter:
		return Span{lo: last.next(d.Calendar()).key(), hi: math.MaxInt}
	default:
		return Span{lo: first.key(), hi: last.key()}
	}
}

// rangeSpan maps a range to its span. Start and End return the zero Date
// for an open side, so both open forms and endpoints without a year fall
// out of the same year check.
func (d Date) rangeSpan() Span {
	s := allTime
	if start := d.Start(); start.Year() != 0 {
		first, _ := start.pointBounds()
		s.lo = first.key()
	}
	if end := d.End(); end.Year() != 0 {
		_, last := end.pointBounds()
		s.hi = last.key()
	}
	if s.lo > s.hi {
		return allTime
	}

	return s
}

// pointBounds returns the first and last day of a point date at its
// written precision. Month and day take part only when the parser resolved
// them exactly, which it does for Gregorian and Julian bodies.
func (d Date) pointBounds() (civilDay, civilDay) {
	year := d.Year()
	month, ok := d.Month()
	if !ok {
		return civilDay{year, 1, 1}, civilDay{year, 12, 31}
	}
	day, ok := d.Day()
	if !ok {
		return civilDay{year, month, 1}, civilDay{year, month, daysIn(d.Calendar(), year, month)}
	}

	return civilDay{year, month, day}, civilDay{year, month, day}
}

// civilDay is a calendar day with a signed year (negative for BCE).
type civilDay struct {
	year, month, day int
}

// key orders days on one number line. Month and day contribute less than
// 10000, so negative BCE years keep their order too: 45 BCE sorts before
// 44 BCE, and every day of 44 BCE before 1 CE.
func (c civilDay) key() int {
	return c.year*10000 + c.month*100 + c.day
}

// next returns the following day in the given calendar.
func (c civilDay) next(cal Calendar) civilDay {
	if c.day < daysIn(cal, c.year, c.month) {
		c.day++

		return c
	}
	c.day = 1
	if c.month < 12 {
		c.month++

		return c
	}
	c.month = 1
	c.year++
	if c.year == 0 {
		c.year = 1 // 1 BCE is followed by 1 CE.
	}

	return c
}

// prev returns the preceding day in the given calendar.
func (c civilDay) prev(cal Calendar) civilDay {
	if c.day > 1 {
		c.day--

		return c
	}
	if c.month > 1 {
		c.month--
	} else {
		c.month = 12
		c.year--
		if c.year == 0 {
			c.year = -1 // 1 CE is preceded by 1 BCE.
		}
	}
	c.day = daysIn(cal, c.year, c.month)

	return c
}

// daysIn returns the length of a month. Only the Gregorian and Julian
// calendars expose exact months, and they differ in leap years alone.
func daysIn(cal Calendar, year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(cal, year) {
			return 29
		}

		return 28
	default:
		return 31
	}
}

// isLeapYear applies the calendar's leap rule to a signed year. BCE years
// are shifted to astronomical numbering first, in which 1 BCE is year 0,
// so the four-year cycle continues unbroken across the era boundary.
func isLeapYear(cal Calendar, year int) bool {
	if year < 0 {
		year++
	}
	if cal == CalendarJulian {
		return year%4 == 0
	}

	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// Precedes reports whether s ends before o begins, sharing no day.
func (s Span) Precedes(o Span) bool {
	return s.hi < o.lo
}

// Overlaps reports whether s begins before o, shares at least one day with
// it, and ends before it does. This is Allen's strict partial overlap; use
// [Span.Intersects] to ask whether two spans share any day.
func (s Span) Overlaps(o Span) bool {
	return s.lo < o.lo && o.lo <= s.hi && s.hi < o.hi
}

// During reports whether every day of s lies within o and o is the larger
// of the two. It covers Allen's starts, during and finishes.
func (s Span) During(o Span) bool {
	return o.lo <= s.lo && s.hi <= o.hi && s != o
}

// Equals reports whether the two spans cover exactly the same days.
func (s Span) Equals(o Span) bool {
	return s == o
}

// Intersects reports whether the two spans share at least one day, which is
// to say neither precedes the other.
func (s Span) Intersects(o Span) bool {
	return !s.Precedes(o) && !o.Precedes(s)
}
