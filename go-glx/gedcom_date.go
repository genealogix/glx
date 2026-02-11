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
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Month names to numbers
var monthMap = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

// parseGEDCOMDate parses a GEDCOM date string to GLX format
// Handles: exact dates, ranges, qualifiers (ABT, BEF, AFT, etc.)
// Returns: DateString in GLX format (e.g., "ABT 1850", "BEF 1920-01-15", "BET 1880 AND 1890")
func parseGEDCOMDate(gedcomDate string) DateString {
	if gedcomDate == "" {
		return ""
	}

	date := strings.TrimSpace(gedcomDate)

	// Handle date ranges - convert to GLX keyword format
	if strings.Contains(date, "BET ") && strings.Contains(date, " AND ") {
		// BET date1 AND date2 (e.g., "BET 1880 AND 1890")
		parts := strings.Split(date, " AND ")
		if len(parts) == 2 {
			start := strings.TrimPrefix(parts[0], "BET ")
			end := parts[1]
			startDate := parseExactDate(strings.TrimSpace(start))
			endDate := parseExactDate(strings.TrimSpace(end))
			if startDate != "" && endDate != "" {
				return DateString("BET " + startDate + " AND " + endDate)
			}
		}
	}

	if after, ok := strings.CutPrefix(date, "FROM "); ok {
		if strings.Contains(date, " TO ") {
			// FROM date1 TO date2 (e.g., "FROM 1900 TO 1950")
			parts := strings.Split(date, " TO ")
			if len(parts) == 2 {
				start := strings.TrimPrefix(parts[0], "FROM ")
				end := parts[1]
				startDate := parseExactDate(strings.TrimSpace(start))
				endDate := parseExactDate(strings.TrimSpace(end))
				if startDate != "" && endDate != "" {
					return DateString("FROM " + startDate + " TO " + endDate)
				}
			}
		} else {
			// FROM date (open-ended, e.g., "FROM 1900")
			dateStr := after
			startDate := parseExactDate(strings.TrimSpace(dateStr))
			if startDate != "" {
				return DateString("FROM " + startDate)
			}
		}
	}

	// Handle qualifiers - preserve them in GLX keyword format
	// GLX uses same keywords as GEDCOM: ABT, BEF, AFT, CAL
	qualifiers := []string{"ABT ", "CAL ", "BEF ", "AFT "}
	for _, qual := range qualifiers {
		if after, ok := strings.CutPrefix(date, qual); ok {
			dateStr := after
			exactDate := parseExactDate(strings.TrimSpace(dateStr))
			if exactDate != "" {
				// Return as "ABT 1850-03-15" format (keyword + YYYY-MM-DD)
				return DateString(qual + exactDate)
			}
		}
	}

	// Try to parse as exact date - return YYYY-MM-DD format
	return DateString(parseExactDate(date))
}

// parseExactDate parses an exact GEDCOM date to YYYY-MM-DD format
// Formats: "DD MMM YYYY" -> "YYYY-MM-DD", "MMM YYYY" -> "YYYY-MM", "YYYY" -> "YYYY"
//
//nolint:gocyclo
func parseExactDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	parts := strings.Fields(dateStr)
	if len(parts) == 0 {
		return ""
	}

	// Try different formats
	switch len(parts) {
	case 1:
		// Just year: "1900"
		year, err := strconv.Atoi(parts[0])
		if err != nil || year <= 0 || year >= 3000 {
			return ""
		}

		return fmt.Sprintf("%04d", year)

	case 2:
		// Month and year: "JAN 1900"
		month, ok := monthMap[strings.ToUpper(parts[0])]
		if !ok {
			return ""
		}
		year, err := strconv.Atoi(parts[1])
		if err != nil || year <= 0 || year >= 3000 {
			return ""
		}

		return fmt.Sprintf("%04d-%02d", year, month)

	case 3:
		// Day, month, year: "15 JAN 1900"
		day, err := strconv.Atoi(parts[0])
		if err != nil || day <= 0 || day > 31 {
			return ""
		}
		month, ok := monthMap[strings.ToUpper(parts[1])]
		if !ok {
			return ""
		}
		year, err := strconv.Atoi(parts[2])
		if err != nil || year <= 0 || year >= 3000 {
			return ""
		}

		// Validate the date
		date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if date.Year() == year && int(date.Month()) == month && date.Day() == day {
			return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		}
	}

	return ""
}
