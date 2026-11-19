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

// ChronologyBounds expresses Gregorian and Julian date bounds on the same
// Julian-day axis. Other calendars return Known=false. Unlike Timing, these
// bounds can be compared across calendars. Approximation uses ValueBounds'
// tolerance and does not make uncertain dates exact.
func (d Date) ChronologyBounds(approximationYears int) Bounds {
	calendar := d.Calendar()
	if calendar != CalendarGregorian && calendar != CalendarJulian {
		return Bounds{}
	}
	bounds := d.ValueBounds(approximationYears)
	if !bounds.Known {
		return bounds
	}
	bounds.Outer = Interval{chronologyEndpoint(bounds.Outer.Start, calendar), chronologyEndpoint(bounds.Outer.End, calendar)}
	if bounds.Inner.Start < bounds.Inner.End {
		bounds.Inner = Interval{chronologyEndpoint(bounds.Inner.Start, calendar), chronologyEndpoint(bounds.Inner.End, calendar)}
	}

	return bounds
}

func chronologyEndpoint(key int, calendar Calendar) int {
	if key == math.MinInt || key == math.MaxInt {
		return key
	}
	year := floorQuotient(key, civilYearDays)
	ordinal := key - year*civilYearDays
	month := 1
	for ordinal >= daysInMonth[month] && month < 12 {
		ordinal -= daysInMonth[month]
		month++
	}

	return calendarJulianDay(year, month, ordinal+1, calendar)
}

// calendarJulianDay uses astronomical years, including zero for 1 BCE.
//
//nolint:mnd // Standard proleptic Gregorian/Julian day-number formula constants.
func calendarJulianDay(year, month, day int, calendar Calendar) int {
	shift := (14 - month) / 12
	y, m := year+4800-shift, month+12*shift-3
	base := day + (153*m+2)/5 + 365*y + floorQuotient(y, 4)
	if calendar == CalendarJulian {
		return base - 32083
	}

	return base - floorQuotient(y, 100) + floorQuotient(y, 400) - 32045
}

func floorQuotient(a, b int) int {
	quotient := a / b
	if a%b < 0 {
		quotient--
	}

	return quotient
}
