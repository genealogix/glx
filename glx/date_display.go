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

package main

import (
	"strconv"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/genealogix/glx/go-glx/glxdate"
)

// isoDateMonths maps month numbers to names.
var isoDateMonths = map[string]string{
	"01": "January", "02": "February", "03": "March",
	"04": "April", "05": "May", "06": "June",
	"07": "July", "08": "August", "09": "September",
	"10": "October", "11": "November", "12": "December",
}

// displayDate normalizes a date string for tabular display.
// Converts ISO dates to readable form; passes other formats through unchanged.
// Returns "(no date)" for empty or whitespace-only strings.
func displayDate(date string) string {
	date = strings.TrimSpace(date)
	if date == "" {
		return "(no date)"
	}

	return formatReadableDate(date)
}

// formatReadableDate converts ISO dates to readable text:
//   - "1863-06-18" → "June 18, 1863"
//   - "1850-03"    → "March 1850"
//
// A BCE suffix is kept after the readable body ("March 15, 0044 BCE").
// Returns the input unchanged for other formats, and for an ISO date the
// grammar rejects.
func formatReadableDate(s string) string {
	s = strings.TrimSpace(s)
	if body, isBCE := strings.CutSuffix(s, " BCE"); isBCE {
		return formatReadableDate(body) + " BCE"
	}
	// Full date: YYYY-MM-DD. One the grammar rejects (1643-02-30) stays raw,
	// so an impossible day is never printed as a real date (#1373).
	if isFullDate(s) {
		if _, err := glxdate.Parse(s); err != nil {
			return s
		}
		month := isoDateMonths[s[5:7]]
		if month == "" {
			return s
		}
		day := strings.TrimLeft(s[8:10], "0")
		if day == "" {
			day = "0"
		}

		return month + " " + day + ", " + s[:4]
	}
	// Year-month: YYYY-MM
	if len(s) == 7 && s[4] == '-' {
		month := isoDateMonths[s[5:7]]
		if month != "" {
			return month + " " + s[:4]
		}
	}

	return s
}

// isFullDate checks if a date string is a full YYYY-MM-DD date.
func isFullDate(s string) bool {
	return len(s) == 10 && s[4] == '-' && s[7] == '-'
}

// shortDateYears renders a GLX date as the compact year form used in one-line
// lifespans ("1850", "c. 1765", "1756/1774"). Unlike glxlib.ExtractFirstYear it keeps
// what the date says about its certainty, so a range never reads as a precise
// year (#1328):
//
//   - "1850-03-22"               → "1850"
//   - "44 BCE"                   → "44 BCE"
//   - "ABT 1765"                 → "c. 1765"
//   - "EST 1765" / "CAL 1765"    → "est. 1765" / "cal. 1765"
//   - "BEF 1860" / "AFT 1860"    → "bef. 1860" / "aft. 1860"
//   - "BET 1756 AND 1774"        → "1756/1774"
//   - "FROM 1850 TO 1860"        → "1850/1860"
//   - "BET 1826-05-16 AND 1826"  → "1826" (both ends in the same year)
//   - "FROM 1850" / "TO 1860"    → "from 1850" / "to 1860"
//
// A range end whose year cannot be determined shows as "?". It returns "" when
// the date has no year at all.
func shortDateYears(date string) string {
	d, _ := glxlib.DateString(date).Parse()
	if d.IsZero() {
		return ""
	}

	switch {
	case d.IsOpenStart():
		if y := displayYear(d.End().Year()); y != "" {
			return "to " + y
		}

		return ""
	case d.IsOpenEnded():
		if y := displayYear(d.Start().Year()); y != "" {
			return "from " + y
		}

		return ""
	case d.IsRange():
		start, end := displayYear(d.Start().Year()), displayYear(d.End().Year())
		switch {
		case start == "" && end == "":
			return ""
		case start == end:
			return start
		case start == "":
			start = "?"
		case end == "":
			end = "?"
		}

		return start + "/" + end
	}

	year := displayYear(d.Year())
	if year == "" {
		return ""
	}
	switch d.Qualifier() {
	case glxdate.QualifierAbout:
		return "c. " + year
	case glxdate.QualifierEstimated:
		return "est. " + year
	case glxdate.QualifierCalculated:
		return "cal. " + year
	case glxdate.QualifierBefore:
		return "bef. " + year
	case glxdate.QualifierAfter:
		return "aft. " + year
	case glxdate.QualifierNone, glxdate.QualifierInterpreted:
	}

	return year
}

// displayYear renders a signed year for display ("1850", "44 BCE"), or "" for
// the unknown year 0.
func displayYear(year int) string {
	switch {
	case year == 0:
		return ""
	case year < 0:
		return strconv.Itoa(-year) + " BCE"
	}

	return strconv.Itoa(year)
}

// formatLifeSpan joins birth and death dates into a one-line lifespan:
// "1850–1920", "b. c. 1765", "d. bef. 1860", or "" when neither has a year.
// When either end is more than a bare year ("c. 1765", "1756/1774") the
// separator is a spaced en dash, so it cannot be read as part of a range:
// "1756/1774 – 1826/1830".
func formatLifeSpan(birthDate, deathDate string) string {
	b, d := shortDateYears(birthDate), shortDateYears(deathDate)
	switch {
	case b != "" && d != "":
		if isBareYear(b) && isBareYear(d) {
			return b + "–" + d
		}

		return b + " – " + d
	case b != "":
		return "b. " + b
	case d != "":
		return "d. " + d
	default:
		return ""
	}
}

// isBareYear reports whether a shortDateYears result is a plain CE year.
func isBareYear(s string) bool {
	return s != "" && !strings.ContainsAny(s, " /")
}
