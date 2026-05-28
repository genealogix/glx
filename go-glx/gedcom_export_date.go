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
	"strconv"
	"strings"
)

// gedcomMonthNames maps month numbers (1-12) to GEDCOM 3-letter month abbreviations.
var gedcomMonthNames = [13]string{
	"", "JAN", "FEB", "MAR", "APR", "MAY", "JUN",
	"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
}

// formatGEDCOMDate converts a GLX DateString to GEDCOM date format.
// Examples:
//
//	"1850-03-15" -> "15 MAR 1850"
//	"1850-03"    -> "MAR 1850"
//	"1850"       -> "1850"
//	"ABT 1850-03-15" -> "ABT 15 MAR 1850"
//	"BET 1880 AND 1890" -> "BET 1880 AND 1890"
//	"FROM 1880 TO 1890" -> "FROM 1880 TO 1890"
func formatGEDCOMDate(date DateString) string {
	s := strings.TrimSpace(string(date))
	if s == "" {
		return ""
	}

	// Extract calendar prefix (e.g., "JULIAN 1731-03-15" → calendar="JULIAN", body="1731-03-15")
	calendar, body := ExtractCalendarPrefix(DateString(s))
	if calendar != "" {
		escape := calendarToGEDCOMEscape(calendar)
		bodyStr := strings.TrimSpace(string(body))
		gedcomBody := formatGEDCOMDateBody(bodyStr)
		return escape + " " + gedcomBody
	}

	// No calendar prefix — format as Gregorian
	return formatGEDCOMDateBody(s)
}

// formatGEDCOMDateBody formats a date body (without calendar prefix) to GEDCOM format.
func formatGEDCOMDateBody(s string) string {
	if s == "" {
		return ""
	}

	// Handle range: BET ... AND ...
	if strings.HasPrefix(s, "BET ") && strings.Contains(s, " AND ") {
		parts := strings.SplitN(s, " AND ", 2)
		start := strings.TrimPrefix(parts[0], "BET ")
		end := parts[1]

		return "BET " + convertSingleDate(strings.TrimSpace(start)) + " AND " + convertSingleDate(strings.TrimSpace(end))
	}

	// Handle range: FROM ... TO ...
	if strings.HasPrefix(s, "FROM ") && strings.Contains(s, " TO ") {
		parts := strings.SplitN(s, " TO ", 2)
		start := strings.TrimPrefix(parts[0], "FROM ")
		end := parts[1]

		return "FROM " + convertSingleDate(strings.TrimSpace(start)) + " TO " + convertSingleDate(strings.TrimSpace(end))
	}

	// Handle open-ended FROM
	if strings.HasPrefix(s, "FROM ") {
		dateStr := strings.TrimPrefix(s, "FROM ")

		return "FROM " + convertSingleDate(strings.TrimSpace(dateStr))
	}

	// Handle qualifiers: ABT, BEF, AFT, CAL
	qualifiers := []string{"ABT ", "CAL ", "BEF ", "AFT "}
	for _, qual := range qualifiers {
		if strings.HasPrefix(s, qual) {
			dateStr := strings.TrimPrefix(s, qual)

			return qual + convertSingleDate(strings.TrimSpace(dateStr))
		}
	}

	// Plain date
	return convertSingleDate(s)
}

// convertSingleDate converts a single GLX date (YYYY, YYYY-MM, or YYYY-MM-DD) to GEDCOM format.
func convertSingleDate(date string) string {
	if date == "" {
		return ""
	}

	parts := strings.Split(date, "-")

	switch len(parts) {
	case 1:
		// Year only: "1850" -> "1850"
		return date

	case 2:
		// Year-month: "1850-03" -> "MAR 1850"
		month, err := strconv.Atoi(parts[1])
		if err != nil || month < 1 || month > 12 {
			return date // Return as-is if unparseable
		}

		return gedcomMonthNames[month] + " " + parts[0]

	case 3:
		// Full date: "1850-03-15" -> "15 MAR 1850"
		month, err := strconv.Atoi(parts[1])
		if err != nil || month < 1 || month > 12 {
			return date
		}

		day, err := strconv.Atoi(parts[2])
		if err != nil || day < 1 || day > 31 {
			return date
		}

		return strconv.Itoa(day) + " " + gedcomMonthNames[month] + " " + parts[0]

	default:
		return date
	}
}
