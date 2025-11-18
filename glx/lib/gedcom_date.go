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

package lib

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

// parseGEDCOMDate parses a GEDCOM date string to ISO 8601 format
// Handles: exact dates, ranges, qualifiers (ABT, BEF, AFT, etc.)
func parseGEDCOMDate(gedcomDate string) string {
	if gedcomDate == "" {
		return ""
	}

	date := strings.TrimSpace(gedcomDate)

	// Handle date ranges
	if strings.Contains(date, "BET ") && strings.Contains(date, " AND ") {
		// BET date1 AND date2
		parts := strings.Split(date, " AND ")
		if len(parts) == 2 {
			start := strings.TrimPrefix(parts[0], "BET ")
			end := parts[1]
			startISO := parseExactDate(strings.TrimSpace(start))
			endISO := parseExactDate(strings.TrimSpace(end))
			if startISO != "" && endISO != "" {
				return startISO + "/" + endISO
			}
		}
	}

	if strings.HasPrefix(date, "FROM ") && strings.Contains(date, " TO ") {
		// FROM date1 TO date2
		parts := strings.Split(date, " TO ")
		if len(parts) == 2 {
			start := strings.TrimPrefix(parts[0], "FROM ")
			end := parts[1]
			startISO := parseExactDate(strings.TrimSpace(start))
			endISO := parseExactDate(strings.TrimSpace(end))
			if startISO != "" && endISO != "" {
				return startISO + "/" + endISO
			}
		}
	}

	// Handle qualifiers
	qualifiers := []string{"ABT ", "CAL ", "EST ", "BEF ", "AFT "}
	for _, qual := range qualifiers {
		if strings.HasPrefix(date, qual) {
			dateStr := strings.TrimPrefix(date, qual)
			exactDate := parseExactDate(strings.TrimSpace(dateStr))
			if exactDate != "" {
				// For now, just return the exact date
				// In a more sophisticated implementation, could return structured data
				return exactDate
			}
		}
	}

	// Try to parse as exact date
	return parseExactDate(date)
}

// parseExactDate parses an exact GEDCOM date to ISO 8601
// Formats: "DD MMM YYYY", "MMM YYYY", "YYYY"
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
		if err == nil && year > 0 && year < 3000 {
			return fmt.Sprintf("%04d", year)
		}

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

// parseGEDCOMTime parses GEDCOM 7.0 TIME value
func parseGEDCOMTime(timeStr string) string {
	// TIME format: hh:mm:ss[.fraction][Z|+hh:mm|-hh:mm]
	// Already in ISO 8601-compatible format, return as-is
	return strings.TrimSpace(timeStr)
}

// combineDateAndTime combines GEDCOM DATE and TIME into ISO 8601 datetime
func combineDateAndTime(dateStr string, timeStr string) string {
	date := parseGEDCOMDate(dateStr)
	if date == "" {
		return ""
	}

	if timeStr != "" {
		time := parseGEDCOMTime(timeStr)
		return date + "T" + time
	}

	return date
}
