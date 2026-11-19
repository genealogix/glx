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

// OrderingDay returns a Julian Day Number for sorting dates chronologically,
// across the Gregorian and Julian calendars. It is the earliest day the date
// can fall on or, for a date open at the start (BEF 1700, TO 1700), the
// latest. A BEF date therefore sorts just before the day it names. ok is false
// when the date cannot be placed: empty, without a year, open at both ends,
// reversed, or in a calendar GLX does not convert (Hebrew, French Republican,
// extension calendars).
func (d Date) OrderingDay() (int, bool) {
	b := d.Timing()
	if !b.Known {
		return 0, false
	}
	var ordinal int
	switch {
	case b.Outer.Start != math.MinInt:
		ordinal = b.Outer.Start
	case b.Outer.End != math.MaxInt:
		ordinal = b.Outer.End - 1
	default:
		return 0, false
	}
	year, month, day := decodeOrdinal(ordinal)

	switch d.Calendar() {
	case CalendarGregorian:
		return julianDayNumber(year, month, day, true), true
	case CalendarJulian:
		return julianDayNumber(year, month, day, false), true
	default:
		return 0, false
	}
}

// decodeOrdinal inverts pointBounds: it splits a civil-day ordinal into an
// astronomical year, a month and a day.
func decodeOrdinal(ordinal int) (year, month, day int) {
	year = floorDiv(ordinal, civilYearDays)
	rest := ordinal - year*civilYearDays
	month = 1
	for month < 12 && rest >= daysInMonth[month] {
		rest -= daysInMonth[month]
		month++
	}

	return year, month, rest + 1
}

// julianDayNumber converts an astronomical year, month and day in the
// proleptic Gregorian (gregorian=true) or Julian calendar to a Julian Day
// Number. A day past the end of its month (29 Feb in a common year) rolls into
// the next month, which keeps the order.
func julianDayNumber(year, month, day int, gregorian bool) int {
	// Fliegel and Van Flandern's algorithm, counting years from March so the
	// leap day falls at the end; floorDiv keeps it right for BCE years.
	const (
		monthsPerYear   = 12
		marchShift      = 14 // 14 - month is 1 for Jan and Feb, 0 otherwise (after /12)
		monthsFromMarch = 3
		epochYears      = 4800 // shifts every year in range to a positive count
		daysPerYear     = 365
		leapCycle       = 4
		centuryCycle    = 100
		gregorianCycle  = 400
		gregorianEpoch  = 32045
		julianEpoch     = 32083
		monthDaysNum    = 153 // 153 days in each five-month run from March
		monthDaysRound  = 2
		monthDaysDen    = 5
	)
	a := floorDiv(marchShift-month, monthsPerYear)
	y := year + epochYears - a
	m := month + monthsPerYear*a - monthsFromMarch
	jdn := day + floorDiv(monthDaysNum*m+monthDaysRound, monthDaysDen) + daysPerYear*y + floorDiv(y, leapCycle)
	if gregorian {
		return jdn - floorDiv(y, centuryCycle) + floorDiv(y, gregorianCycle) - gregorianEpoch
	}

	return jdn - julianEpoch
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}

	return q
}
