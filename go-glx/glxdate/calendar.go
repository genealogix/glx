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
	"unicode"
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

// calendarGregorianName names the default calendar in Calendar.String and in
// the GEDCOM escape @#DGREGORIAN@. It is never a prefix.
const calendarGregorianName = "GREGORIAN"

// String returns the calendar's GLX prefix, or "GREGORIAN" / "OTHER" for the
// calendars that have no fixed prefix.
func (c Calendar) String() string {
	switch c {
	case CalendarGregorian:
		return calendarGregorianName
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
	i := strings.IndexFunc(s, unicode.IsSpace)
	if i < 0 {
		return "", s
	}
	candidate, body := s[:i], strings.TrimLeftFunc(s[i:], unicode.IsSpace)

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

// notCalendarWords are upper-case tokens that can legitimately start a date
// body and must never be read as an unknown calendar prefix: keywords and
// their spelled-out forms, month names, seasons, and the placeholder words
// seen at the start of free-text dates in real GEDCOM files ("LIVING 1515",
// "PRIOR 1855", "UNKNOWN 87"). "APRIL 1688" is a Gregorian date, not an
// APRIL calendar; "AFTER 1839 BEFORE 1840" is free text, not an AFTER calendar.
var notCalendarWords = map[string]bool{
	keywordAbout: true, keywordBefore: true, keywordAfter: true, keywordCalculated: true,
	keywordInterpreted: true, keywordBetween: true, keywordAnd: true, keywordFrom: true, keywordTo: true,
	calendarGregorianName: true,
	"ABOUT":               true, "AFTER": true, "BEFORE": true, "BETWEEN": true, "CIRCA": true, "AROUND": true,
	"CALCULATED": true, "ESTIMATED": true, "INTERPRETED": true, "EST": true,
	"EARLY": true, "LATE": true, "PRIOR": true, "SINCE": true, "UNTIL": true, "DURING": true,
	"SPRING": true, "SUMMER": true, "FALL": true, "AUTUMN": true, "WINTER": true,
	"LIVING": true, "UNKNOWN": true, "DECEASED": true, "CLEARED": true, "INFANT": true, "CHILD": true,
	"STILLBORN": true, "PRIVATE": true, "PRIVATIZED": true, "NOT": true, "NONE": true,
}

// isCalendarPrefix reports whether token looks like an unknown calendar prefix:
// all uppercase letters and underscores, at least minCalendarPrefixLen long,
// and not a word that can legitimately start a date body (see
// notCalendarWords).
func isCalendarPrefix(token string) bool {
	if notCalendarWords[token] {
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
