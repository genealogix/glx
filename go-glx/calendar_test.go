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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCalendar(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantCalendar  string
		wantRemainder string
	}{
		{
			name:          "Julian escape with trailing space",
			input:         "@#DJULIAN@ 15 MAR 1731",
			wantCalendar:  CalendarJulian,
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "Hebrew escape",
			input:         "@#DHEBREW@ 15 TSH 5765",
			wantCalendar:  CalendarHebrew,
			wantRemainder: "15 TSH 5765",
		},
		{
			name:          "French Republican escape (space in name)",
			input:         "@#DFRENCH R@ 1 VEND 0012",
			wantCalendar:  CalendarFrenchR,
			wantRemainder: "1 VEND 0012",
		},
		{
			name:          "Gregorian escape (maps to empty — default)",
			input:         "@#DGREGORIAN@ 15 MAR 1731",
			wantCalendar:  "",
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "no escape (plain date)",
			input:         "15 MAR 1731",
			wantCalendar:  "",
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "escape without trailing space",
			input:         "@#DJULIAN@15 MAR 1731",
			wantCalendar:  CalendarJulian,
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "empty string",
			input:         "",
			wantCalendar:  "",
			wantRemainder: "",
		},
		{
			name:          "unknown calendar preserved",
			input:         "@#DROMAN@ 15 MAR 1731",
			wantCalendar:  "ROMAN",
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "unknown calendar with spaces normalized to underscores",
			input:         "@#DNEW CAL@ 15 MAR 1731",
			wantCalendar:  "NEW_CAL",
			wantRemainder: "15 MAR 1731",
		},
		{
			name:          "escape with empty remainder",
			input:         "@#DJULIAN@",
			wantCalendar:  CalendarJulian,
			wantRemainder: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calendar, remainder := extractCalendar(tt.input)
			assert.Equal(t, tt.wantCalendar, calendar)
			assert.Equal(t, tt.wantRemainder, remainder)
		})
	}
}

func TestParseGEDCOMDate_CalendarPreservation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Julian exact date preserves calendar prefix",
			input:    "@#DJULIAN@ 15 MAR 1731",
			expected: "JULIAN 1731-03-15",
		},
		{
			name:     "Julian year only",
			input:    "@#DJULIAN@ 1731",
			expected: "JULIAN 1731",
		},
		{
			name:     "Julian with qualifier",
			input:    "@#DJULIAN@ ABT 1731",
			expected: "JULIAN ABT 1731",
		},
		{
			name:     "Hebrew raw preserved with prefix",
			input:    "@#DHEBREW@ 15 TSH 5765",
			expected: "HEBREW 15 TSH 5765",
		},
		{
			name:     "French Republican raw preserved with prefix",
			input:    "@#DFRENCH R@ 1 VEND 0012",
			expected: "FRENCH_R 1 VEND 0012",
		},
		{
			name:     "Gregorian escape produces no prefix (default)",
			input:    "@#DGREGORIAN@ 15 MAR 1731",
			expected: "1731-03-15",
		},
		{
			name:     "no escape unchanged behavior",
			input:    "15 MAR 1731",
			expected: "1731-03-15",
		},
		{
			name:     "Julian dual date preserved raw",
			input:    "@#DJULIAN@ 11 FEB 1731/32",
			expected: "JULIAN 11 FEB 1731/32",
		},
		{
			name:     "Julian month-year",
			input:    "@#DJULIAN@ MAR 1731",
			expected: "JULIAN 1731-03",
		},
		{
			name:     "calendar escape with no date body preserves raw",
			input:    "@#DJULIAN@",
			expected: "@#DJULIAN@",
		},
		{
			name:     "Gregorian escape with no date body preserves raw",
			input:    "@#DGREGORIAN@",
			expected: "@#DGREGORIAN@",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGEDCOMDate(tt.input)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestFormatGEDCOMDate_CalendarPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Julian full date",
			input:    "JULIAN 1731-03-15",
			expected: "@#DJULIAN@ 15 MAR 1731",
		},
		{
			name:     "Julian year only",
			input:    "JULIAN 1731",
			expected: "@#DJULIAN@ 1731",
		},
		{
			name:     "Julian with qualifier",
			input:    "JULIAN ABT 1731",
			expected: "@#DJULIAN@ ABT 1731",
		},
		{
			name:     "Hebrew raw passthrough",
			input:    "HEBREW 15 TSH 5765",
			expected: "@#DHEBREW@ 15 TSH 5765",
		},
		{
			name:     "French Republican raw passthrough",
			input:    "FRENCH_R 1 VEND 0012",
			expected: "@#DFRENCH R@ 1 VEND 0012",
		},
		{
			name:     "Gregorian (no prefix) unchanged",
			input:    "1731-03-15",
			expected: "15 MAR 1731",
		},
		{
			name:     "Julian dual date raw passthrough",
			input:    "JULIAN 11 FEB 1731/32",
			expected: "@#DJULIAN@ 11 FEB 1731/32",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatGEDCOMDate(DateString(tt.input))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractGLXCalendarPrefix(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantCalendar  string
		wantRemainder string
	}{
		{
			name:          "Julian prefix",
			input:         "JULIAN 1731-03-15",
			wantCalendar:  CalendarJulian,
			wantRemainder: "1731-03-15",
		},
		{
			name:          "Hebrew prefix",
			input:         "HEBREW 15 TSH 5765",
			wantCalendar:  CalendarHebrew,
			wantRemainder: "15 TSH 5765",
		},
		{
			name:          "French Republican prefix",
			input:         "FRENCH_R 1 VEND 0012",
			wantCalendar:  CalendarFrenchR,
			wantRemainder: "1 VEND 0012",
		},
		{
			name:          "no prefix (Gregorian default)",
			input:         "1731-03-15",
			wantCalendar:  "",
			wantRemainder: "1731-03-15",
		},
		{
			name:          "qualifier not mistaken for calendar",
			input:         "ABT 1731",
			wantCalendar:  "",
			wantRemainder: "ABT 1731",
		},
		{
			name:          "BET range not mistaken for calendar",
			input:         "BET 1880 AND 1890",
			wantCalendar:  "",
			wantRemainder: "BET 1880 AND 1890",
		},
		{
			name:          "Julian with qualifier",
			input:         "JULIAN ABT 1731",
			wantCalendar:  CalendarJulian,
			wantRemainder: "ABT 1731",
		},
		{
			name:          "empty string",
			input:         "",
			wantCalendar:  "",
			wantRemainder: "",
		},
		{
			name:          "GREGORIAN not treated as calendar prefix",
			input:         "GREGORIAN 1731-03-15",
			wantCalendar:  "",
			wantRemainder: "GREGORIAN 1731-03-15",
		},
		{
			name:          "EST not treated as calendar prefix",
			input:         "EST 1900",
			wantCalendar:  "",
			wantRemainder: "EST 1900",
		},
		{
			name:          "unknown calendar with underscores roundtrips",
			input:         "NEW_CAL 15 MAR 1731",
			wantCalendar:  "NEW_CAL",
			wantRemainder: "15 MAR 1731",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calendar, remainder := ExtractCalendarPrefix(DateString(tt.input))
			assert.Equal(t, tt.wantCalendar, calendar)
			assert.Equal(t, tt.wantRemainder, string(remainder))
		})
	}
}

func TestCalendarToGEDCOMEscape(t *testing.T) {
	assert.Equal(t, "@#DJULIAN@", calendarToGEDCOMEscape(CalendarJulian))
	assert.Equal(t, "@#DHEBREW@", calendarToGEDCOMEscape(CalendarHebrew))
	assert.Equal(t, "@#DFRENCH R@", calendarToGEDCOMEscape(CalendarFrenchR))
	assert.Equal(t, "", calendarToGEDCOMEscape(""))
	assert.Equal(t, "@#DROMAN@", calendarToGEDCOMEscape("ROMAN"))
	assert.Equal(t, "@#DNEW CAL@", calendarToGEDCOMEscape("NEW_CAL"))
}
