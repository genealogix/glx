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
	"fmt"
	"strings"
)

// Precision is how much of a date is known.
type Precision uint8

// Precision levels, in increasing order.
const (
	// PrecisionNone means no year could be determined.
	PrecisionNone Precision = iota
	PrecisionYear
	PrecisionMonth
	PrecisionDay
)

// String returns a human-readable precision name.
func (p Precision) String() string {
	switch p {
	case PrecisionNone:
		return "none"
	case PrecisionYear:
		return "year"
	case PrecisionMonth:
		return "month"
	case PrecisionDay:
		return "day"
	}

	return "none"
}

// Qualifier is the keyword that modifies a point date.
type Qualifier uint8

// Qualifiers, matching the spec's keyword list.
const (
	QualifierNone        Qualifier = iota
	QualifierAbout                 // ABT
	QualifierBefore                // BEF
	QualifierAfter                 // AFT
	QualifierCalculated            // CAL
	QualifierInterpreted           // INT
)

// qualifierKeywords maps keywords to qualifiers.
var qualifierKeywords = map[string]Qualifier{
	"ABT": QualifierAbout,
	"BEF": QualifierBefore,
	"AFT": QualifierAfter,
	"CAL": QualifierCalculated,
	"INT": QualifierInterpreted,
}

// Keyword returns the GLX keyword for the qualifier ("" for QualifierNone).
func (q Qualifier) Keyword() string {
	switch q {
	case QualifierNone:
		return ""
	case QualifierAbout:
		return "ABT"
	case QualifierBefore:
		return "BEF"
	case QualifierAfter:
		return "AFT"
	case QualifierCalculated:
		return "CAL"
	case QualifierInterpreted:
		return "INT"
	}

	return ""
}

// String returns the qualifier keyword.
func (q Qualifier) String() string {
	return q.Keyword()
}

// rangeKind distinguishes the range forms of the spec.
type rangeKind uint8

const (
	rangeNone    rangeKind = iota
	rangeBetween           // BET start AND end
	rangeFromTo            // FROM start TO end
	rangeFrom              // FROM start (open-ended)
)

// point is a single date within a Date: either a fully determined
// year/month/day (exact) or a raw-preserved body with a best-effort year.
type point struct {
	raw       string // component text with normalized whitespace
	year      int
	month     int
	day       int
	precision Precision
	exact     bool   // components fully determined; String renders ISO form
	canonical bool   // raw is already in canonical GLX form
	reason    string // why the point is not canonical (empty when it is)
}

// String renders the canonical ISO form of an exact point and the raw text
// of anything else.
func (p point) String() string {
	if !p.exact {
		return p.raw
	}

	switch p.precision {
	case PrecisionDay:
		return fmt.Sprintf("%04d-%02d-%02d", p.year, p.month, p.day)
	case PrecisionMonth:
		return fmt.Sprintf("%04d-%02d", p.year, p.month)
	case PrecisionYear, PrecisionNone:
		return fmt.Sprintf("%04d", p.year)
	}

	return p.raw
}

// Date is an immutable, calendar-aware genealogical date or date range.
// It has value semantics and is cheap to copy: a Date is a small handle to
// an immutable value, so pass and store it by value. The zero value is the
// unknown date (see IsZero). Compare dates with Equal, not ==.
//
// Date is a computation model, not a serialization type: GLX files store
// dates as plain strings, and Parse / String convert between the two.
type Date struct {
	v *dateValue
}

// dateValue is the immutable payload behind a Date.
type dateValue struct {
	raw          string
	calendar     Calendar
	calendarName string // prefix text for CalendarOther
	qualifier    Qualifier
	rng          rangeKind
	start        point
	end          point
	interpreted  string // original text of an INT date
	valid        bool
	reason       string // why the date is not valid (empty when it is)
}

// zeroValue backs the zero Date so accessors never dereference nil.
var zeroValue dateValue

// val returns the payload, or the shared zero payload for the zero Date.
func (d Date) val() *dateValue {
	if d.v == nil {
		return &zeroValue
	}

	return d.v
}

// IsZero reports whether d is the unknown date (the zero value).
func (d Date) IsZero() bool {
	return d.v == nil
}

// Raw returns the text the date was parsed from, trimmed of surrounding
// whitespace, or the canonical form for constructed dates.
func (d Date) Raw() string {
	return d.val().raw
}

// Calendar returns the calendar system the date is recorded in.
func (d Date) Calendar() Calendar {
	return d.val().calendar
}

// CalendarName returns the calendar prefix as written ("" for Gregorian).
// For known calendars it equals Calendar().Prefix(); for CalendarOther it is
// the preserved unknown prefix.
func (d Date) CalendarName() string {
	return d.val().prefix()
}

// prefix returns the calendar prefix to write before the body.
func (v *dateValue) prefix() string {
	if v.calendar == CalendarOther {
		return v.calendarName
	}

	return v.calendar.Prefix()
}

// Precision returns how much of the start date is known.
func (d Date) Precision() Precision {
	return d.val().start.precision
}

// Qualifier returns the keyword qualifier of a point date (QualifierNone for
// ranges and plain dates).
func (d Date) Qualifier() Qualifier {
	return d.val().qualifier
}

// InterpretedText returns the original text of an INT date and whether the
// date carries one.
func (d Date) InterpretedText() (string, bool) {
	v := d.val()

	return v.interpreted, v.qualifier == QualifierInterpreted && v.interpreted != ""
}

// IsRange reports whether the date is a BET…AND, FROM…TO, or open-ended FROM range.
func (d Date) IsRange() bool {
	return d.val().rng != rangeNone
}

// IsOpenEnded reports whether the date is a FROM range with no TO end.
func (d Date) IsOpenEnded() bool {
	return d.val().rng == rangeFrom
}

// Start returns the start of a range as a point date, or d itself when d is
// not a range.
func (d Date) Start() Date {
	if !d.IsRange() {
		return d
	}

	return d.val().pointDate(d.val().start)
}

// End returns the end of a range as a point date. It is the zero Date when d
// is not a range or is open-ended.
func (d Date) End() Date {
	v := d.val()
	if v.rng != rangeBetween && v.rng != rangeFromTo {
		return Date{}
	}

	return v.pointDate(v.end)
}

// pointDate wraps one component of a range as a standalone Date in the same calendar.
func (v *dateValue) pointDate(p point) Date {
	pv := &dateValue{
		raw:          p.raw,
		calendar:     v.calendar,
		calendarName: v.calendarName,
		start:        p,
		valid:        p.canonical,
		reason:       p.reason,
	}
	if name := pv.prefix(); name != "" {
		pv.raw = name + " " + p.raw
	}

	return Date{v: pv}
}

// Year returns the start year, or 0 when no year could be determined.
// For raw-preserved bodies this is a best-effort extraction that prefers a
// 4-digit token, so a day of month is never reported as the year.
func (d Date) Year() int {
	return d.val().start.year
}

// Month returns the start month (1–12) and whether it is known. It is never
// known for Hebrew, French Republican, or unknown-calendar dates, whose month
// names GLX preserves raw.
func (d Date) Month() (int, bool) {
	p := d.val().start
	if p.precision < PrecisionMonth {
		return 0, false
	}

	return p.month, true
}

// Day returns the start day of month and whether it is known.
func (d Date) Day() (int, bool) {
	p := d.val().start
	if p.precision < PrecisionDay {
		return 0, false
	}

	return p.day, true
}

// Valid reports whether the date is in canonical GLX form as defined by the
// specification's Date Format Standard. A date parsed from tolerated input
// ("15 March 1850") is not valid even though its components are known; its
// String form is.
func (d Date) Valid() bool {
	return d.val().valid
}

// Equal reports whether two dates have the same canonical form in the same
// calendar. It is textual equality of String(), not a temporal comparison:
// "1850" and "1850-01" are not equal.
func (d Date) Equal(o Date) bool {
	return d.String() == o.String()
}

// String returns the canonical GLX form of the date: an optional calendar
// prefix, then keyword and ISO components. Components that could not be
// determined are rendered as their raw text, so String never loses
// information. The zero Date renders as "".
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}

	return d.val().String()
}

// String renders the payload; see Date.String.
func (v *dateValue) String() string {
	var b strings.Builder
	if name := v.prefix(); name != "" {
		b.WriteString(name)
		b.WriteByte(' ')
	}

	switch v.rng {
	case rangeBetween:
		b.WriteString("BET " + v.start.String() + " AND " + v.end.String())
	case rangeFromTo:
		b.WriteString("FROM " + v.start.String() + " TO " + v.end.String())
	case rangeFrom:
		b.WriteString("FROM " + v.start.String())
	case rangeNone:
		if kw := v.qualifier.Keyword(); kw != "" {
			b.WriteString(kw + " ")
		}
		b.WriteString(v.start.String())
		if v.qualifier == QualifierInterpreted && v.interpreted != "" {
			b.WriteString(" (" + v.interpreted + ")")
		}
	}

	return b.String()
}

// New constructs an exact Gregorian or Julian date. Precision is inferred
// from zero month/day: New(cal, 1850, 0, 0) is year precision. Components
// outside their valid ranges produce a Date that is not Valid.
// For calendars whose months are preserved raw, only the year is used.
func New(cal Calendar, year, month, day int) Date {
	p := point{year: year, month: month, day: day, precision: PrecisionYear, exact: true}
	switch {
	case !cal.hasStructuredMonths():
		p.month, p.day = 0, 0
	case day != 0:
		p.precision = PrecisionDay
	case month != 0:
		p.precision = PrecisionMonth
	}
	p.canonical = checkComponents(&p)
	p.raw = p.String()

	v := &dateValue{calendar: cal, start: p, valid: p.canonical, reason: p.reason}
	v.raw = v.String()

	return Date{v: v}
}

// NewRange constructs a BET…AND range from two point dates in the same
// calendar. A zero end produces an open-ended FROM range. It returns
// ErrRangeMismatch when the calendars differ or either date is itself a range.
func NewRange(start, end Date) (Date, error) {
	if start.IsRange() || end.IsRange() || start.IsZero() {
		return Date{}, ErrRangeMismatch
	}
	if !end.IsZero() && end.Calendar() != start.Calendar() {
		return Date{}, ErrRangeMismatch
	}

	sv, ev := start.val(), end.val()
	v := &dateValue{
		calendar:     sv.calendar,
		calendarName: sv.calendarName,
		start:        sv.start,
		end:          ev.start,
		rng:          rangeFrom,
		valid:        sv.valid,
		reason:       sv.reason,
	}
	if !end.IsZero() {
		v.rng = rangeBetween
		v.valid = v.valid && ev.valid
		if v.reason == "" {
			v.reason = ev.reason
		}
	}
	v.raw = v.String()

	return Date{v: v}, nil
}

// MustParse is like Parse but panics on an invalid date. Intended for tests
// and package-level constants.
func MustParse(s string) Date {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}

	return d
}
