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

// Calendar identifies the calendar system a date is recorded in.
type Calendar uint8

// Calendar systems. Gregorian is the zero value and the default.
const (
	CalendarGregorian Calendar = iota
	CalendarJulian
	CalendarHebrew
	CalendarFrenchRepublican
	// CalendarOther is any calendar prefix GLX does not know. The prefix text
	// is preserved verbatim and available via Date.CalendarName.
	CalendarOther
)

// GLX calendar prefixes as written in a DateString.
const (
	PrefixJulian           = "JULIAN"
	PrefixHebrew           = "HEBREW"
	PrefixFrenchRepublican = "FRENCH_R"
)

// minCalendarPrefixLen is the shortest token treated as an unknown calendar
// prefix. It excludes 3-letter month abbreviations and short keywords while
// accepting all known calendar names (JULIAN=6, HEBREW=6, FRENCH_R=8).
const minCalendarPrefixLen = 5

// knownPrefixes maps GLX calendar prefixes to calendars.
var knownPrefixes = map[string]Calendar{
	PrefixJulian:           CalendarJulian,
	PrefixHebrew:           CalendarHebrew,
	PrefixFrenchRepublican: CalendarFrenchRepublican,
}

// String returns the calendar's GLX prefix, or "GREGORIAN" / "OTHER" for the
// calendars that have no fixed prefix.
func (c Calendar) String() string {
	switch c {
	case CalendarGregorian:
		return "GREGORIAN"
	case CalendarJulian:
		return PrefixJulian
	case CalendarHebrew:
		return PrefixHebrew
	case CalendarFrenchRepublican:
		return PrefixFrenchRepublican
	case CalendarOther:
		return "OTHER"
	}

	return "OTHER"
}

// Prefix returns the prefix written before a date body in this calendar.
// It is empty for Gregorian (the default is never written) and for
// CalendarOther (whose name lives on the Date).
func (c Calendar) Prefix() string {
	switch c {
	case CalendarJulian, CalendarHebrew, CalendarFrenchRepublican:
		return c.String()
	case CalendarGregorian, CalendarOther:
		return ""
	}

	return ""
}

// hasStructuredMonths reports whether GLX parses this calendar's month names
// into numeric components. Only Gregorian and Julian dates are structured;
// all others preserve raw month names (spec design note #3).
func (c Calendar) hasStructuredMonths() bool {
	return c == CalendarGregorian || c == CalendarJulian
}

// SplitCalendarPrefix splits a GLX calendar prefix from a date string.
// It returns the prefix and the remaining body, or ("", s) when no prefix is
// present. Known prefixes (JULIAN, HEBREW, FRENCH_R) are recognized directly;
// any other all-uppercase token of at least five letters that is not a date
// keyword is treated as an unknown calendar prefix.
//
//	SplitCalendarPrefix("JULIAN 1731-03-15") → ("JULIAN", "1731-03-15")
//	SplitCalendarPrefix("ABT 1731")          → ("", "ABT 1731")
func SplitCalendarPrefix(s string) (string, string) {
	candidate, body, found := strings.Cut(s, " ")
	if !found {
		return "", s
	}

	if _, ok := knownPrefixes[candidate]; ok || isCalendarPrefix(candidate) {
		return candidate, body
	}

	return "", s
}

// calendarForPrefix maps a prefix returned by SplitCalendarPrefix to a Calendar.
func calendarForPrefix(prefix string) Calendar {
	if prefix == "" {
		return CalendarGregorian
	}
	if cal, ok := knownPrefixes[prefix]; ok {
		return cal
	}

	return CalendarOther
}

// isCalendarPrefix reports whether token looks like an unknown calendar prefix:
// all uppercase letters and underscores, at least minCalendarPrefixLen long,
// and not a keyword, month name, or seasonal term that can legitimately start
// a date body ("APRIL 1688" is a Gregorian date, not an APRIL calendar).
func isCalendarPrefix(token string) bool {
	switch token {
	case keywordAbout, keywordBefore, keywordAfter, keywordCalculated, keywordInterpreted,
		keywordBetween, keywordAnd, keywordFrom, keywordTo,
		"GREGORIAN", "EST", "SPRING", "SUMMER", "FALL", "WINTER":
		return false
	}
	if _, isMonth := MonthNumber(token); isMonth {
		return false
	}

	for _, r := range token {
		if r != '_' && (r < 'A' || r > 'Z') {
			return false
		}
	}

	return len(token) >= minCalendarPrefixLen
}
