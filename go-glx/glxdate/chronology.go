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
// Julian-day axis. Other calendars and dates with no day in their calendar
// return Known=false. Unlike Timing, these bounds can be compared across
// calendars. Approximation uses ValueBounds'
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
	start, startKnown := chronologicalPoint(d.Start(), calendar)
	end, endKnown := chronologicalPoint(d.End(), calendar)
	// The parser preserves nominal dates such as an unqualified 29 Feb
	// 1700. They have no position in that calendar, even in BEF or AFT.
	if (startKnown && start.Start >= start.End) || (endKnown && end.Start >= end.End) {
		return Bounds{}
	}
	bounds.Outer = Interval{chronologyEndpoint(bounds.Outer.Start, calendar), chronologyEndpoint(bounds.Outer.End, calendar)}
	if bounds.Inner.Start < bounds.Inner.End {
		if d.IsRange() {
			// Take the last actual day of the start component. Civil
			// ordinals reserve February 29 even in common years.
			bounds.Inner = Interval{start.End - 1, end.Start + 1}
		} else {
			bounds.Inner = bounds.Outer
		}
	}

	return bounds
}

func chronologicalPoint(d Date, calendar Calendar) (Interval, bool) {
	bounds, known := pointBounds(d)
	if !known {
		return Interval{}, false
	}

	return Interval{chronologyEndpoint(bounds.Start, calendar), chronologyEndpoint(bounds.End, calendar)}, true
}

func chronologyEndpoint(key int, calendar Calendar) int {
	if key == math.MinInt || key == math.MaxInt {
		return key
	}
	year, month, day := decodeOrdinal(key)

	return julianDayNumber(year, month, day, calendar == CalendarGregorian)
}
