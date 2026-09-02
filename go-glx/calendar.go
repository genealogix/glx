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

package glx

import (
	"strings"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// Calendar system constants for non-Gregorian date prefixes.
const (
	CalendarJulian  = glxdate.PrefixJulian
	CalendarHebrew  = glxdate.PrefixHebrew
	CalendarFrenchR = glxdate.PrefixFrenchRepublican
)

// knownCalendars maps GLX calendar prefixes to their GEDCOM escape sequences.
var knownCalendars = map[string]string{
	CalendarJulian:  "@#DJULIAN@",
	CalendarHebrew:  "@#DHEBREW@",
	CalendarFrenchR: "@#DFRENCH R@",
}

// gedcomEscapeToCalendar maps GEDCOM calendar escape names to GLX prefixes.
// The key is the text between @#D and @ (e.g., "JULIAN", "HEBREW", "FRENCH R").
var gedcomEscapeToCalendar = map[string]string{
	"JULIAN":    CalendarJulian,
	"HEBREW":    CalendarHebrew,
	"FRENCH R":  CalendarFrenchR,
	"GREGORIAN": "", // Gregorian is the default — no prefix needed
}

// extractCalendar extracts a GEDCOM calendar escape sequence from a date string,
// returning the GLX calendar prefix and the remaining date text.
// Returns ("", date) if no calendar escape is present or if the calendar is Gregorian.
func extractCalendar(date string) (string, string) {
	if date == "" {
		return "", ""
	}

	// Calendar escapes must appear at the start of the date string.
	// A mid-string escape (e.g., "ABT @#DJULIAN@...") is not a calendar prefix.
	trimmed := strings.TrimSpace(date)
	if !strings.HasPrefix(trimmed, "@#D") {
		return "", date
	}

	// Find the closing @. Search from after "@#D" (3 chars).
	escapeName, rest, found := strings.Cut(trimmed[3:], "@")
	if !found {
		return "", date
	}
	remainder := strings.TrimSpace(rest)

	calendar, known := gedcomEscapeToCalendar[escapeName]
	if !known {
		// Unknown calendar — normalize spaces to underscores for a single-token prefix
		// so ExtractCalendarPrefix can roundtrip it. calendarToGEDCOMEscape reverses this.
		calendar = strings.ReplaceAll(escapeName, " ", "_")
	}

	return calendar, remainder
}

// ExtractCalendarPrefix extracts a GLX calendar prefix from a DateString.
// Returns the calendar name and the remaining date without the prefix.
// Returns ("", original) if no calendar prefix is present.
// It delegates to glxdate.SplitCalendarPrefix, the single definition of
// what counts as a calendar prefix.
//
// Example:
//
//	ExtractCalendarPrefix("JULIAN 1731-03-15") → ("JULIAN", "1731-03-15")
//	ExtractCalendarPrefix("1731-03-15")        → ("", "1731-03-15")
//	ExtractCalendarPrefix("ABT 1731")          → ("", "ABT 1731")
func ExtractCalendarPrefix(date DateString) (string, DateString) {
	prefix, body := glxdate.SplitCalendarPrefix(string(date))
	if prefix == "" {
		return "", date
	}

	return prefix, DateString(body)
}

// calendarToGEDCOMEscape converts a GLX calendar prefix to a GEDCOM escape sequence.
// Returns "" for empty/Gregorian (no escape needed).
func calendarToGEDCOMEscape(calendar string) string {
	if calendar == "" {
		return ""
	}

	if escape, ok := knownCalendars[calendar]; ok {
		return escape
	}

	// Unknown calendar — reverse underscore normalization and construct escape.
	return "@#D" + strings.ReplaceAll(calendar, "_", " ") + "@"
}
