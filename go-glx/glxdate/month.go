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
	"strings"
)

// monthsPerYear is the number of months in a Gregorian or Julian year.
const monthsPerYear = 12

// monthAbbreviations are the three-letter month abbreviations (index 1–12),
// the spelling GEDCOM uses on output.
var monthAbbreviations = [monthsPerYear + 1]string{
	"", "JAN", "FEB", "MAR", "APR", "MAY", "JUN",
	"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
}

// monthNames maps upper-case Gregorian month names and abbreviations to
// month numbers. Lookups go through MonthNumber, which folds case and strips
// a trailing period.
var monthNames = buildMonthNames()

// buildMonthNames joins the abbreviations with the full and dialect names.
func buildMonthNames() map[string]int {
	names := map[string]int{
		"JANUARY": 1, "FEBRUARY": 2, "MARCH": 3, "APRIL": 4, "JUNE": 6,
		"JULY": 7, "AUGUST": 8, "SEPT": 9, "SEPTEMBER": 9, "OCTOBER": 10,
		"NOVEMBER": 11, "DECEMBER": 12,
	}
	for m := 1; m <= monthsPerYear; m++ {
		names[monthAbbreviations[m]] = m
	}

	return names
}

// daysInMonth is the maximum day of each month (index 1–12). February allows
// 29 unconditionally, on purpose: a date with no calendar prefix is only
// nominally Gregorian, since records before a jurisdiction's adoption of the
// Gregorian calendar (1752 in Britain and its colonies) are Julian dates
// written without an escape, and 29 FEB 1700 is a real Julian date. Applying
// the Gregorian leap rule would reject genuine evidence, so GLX preserves
// the date as recorded and leaves plausibility to temporal validation.
var daysInMonth = [monthsPerYear + 1]int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// MonthNumber returns the month number (1–12) for a Gregorian month name or
// abbreviation in any letter case ("MAR", "March", "sept."). It reports false
// for anything else, including Hebrew and French Republican month names.
func MonthNumber(name string) (int, bool) {
	m, ok := monthNames[strings.ToUpper(strings.TrimSuffix(name, "."))]

	return m, ok
}

// validDay reports whether day is a possible day of the given month.
func validDay(month, day int) bool {
	return month >= 1 && month <= monthsPerYear && day >= 1 && day <= daysInMonth[month]
}
