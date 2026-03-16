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

import "strings"

// isoDateMonths maps month numbers to names.
var isoDateMonths = map[string]string{
	"01": "January", "02": "February", "03": "March",
	"04": "April", "05": "May", "06": "June",
	"07": "July", "08": "August", "09": "September",
	"10": "October", "11": "November", "12": "December",
}

// displayDate normalizes a date string for tabular display.
// Converts ISO dates to readable form; passes other formats through unchanged.
// Returns "(no date)" for empty strings.
func displayDate(date string) string {
	if date == "" {
		return "(no date)"
	}
	return formatReadableDate(date)
}

// formatReadableDate converts ISO dates to readable text:
//   - "1863-06-18" → "June 18, 1863"
//   - "1850-03"    → "March 1850"
//
// Returns the input unchanged for other formats.
func formatReadableDate(s string) string {
	s = strings.TrimSpace(s)
	// Full date: YYYY-MM-DD
	if isFullDate(s) {
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
