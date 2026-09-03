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
	"github.com/genealogix/glx/go-glx/glxdate"
)

// Calendar system constants for non-Gregorian date prefixes.
const (
	CalendarJulian  = glxdate.PrefixJulian
	CalendarHebrew  = glxdate.PrefixHebrew
	CalendarFrenchR = glxdate.PrefixFrenchRepublican
)

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
