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

// monthNames maps upper-case Gregorian month names and abbreviations to
// month numbers. Lookups go through MonthNumber, which folds case and strips
// a trailing period.
var monthNames = map[string]int{
	"JAN": 1, "JANUARY": 1,
	"FEB": 2, "FEBRUARY": 2,
	"MAR": 3, "MARCH": 3,
	"APR": 4, "APRIL": 4,
	"MAY": 5,
	"JUN": 6, "JUNE": 6,
	"JUL": 7, "JULY": 7,
	"AUG": 8, "AUGUST": 8,
	"SEP": 9, "SEPT": 9, "SEPTEMBER": 9,
	"OCT": 10, "OCTOBER": 10,
	"NOV": 11, "NOVEMBER": 11,
	"DEC": 12, "DECEMBER": 12,
}

// daysInMonth is the maximum day of each month (index 1–12). February allows
// 29 unconditionally: leap-year rules differ between the Gregorian and Julian
// calendars and GLX preserves dates exactly as recorded.
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
