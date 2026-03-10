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
	"regexp"
	"strconv"
)

// temporalYearRegexp matches the first 4-digit year in a date string.
var temporalYearRegexp = regexp.MustCompile(`\b(\d{4})\b`)

// ExtractPropertyYear extracts the first 4-digit year from a person property.
// Handles simple string values, structured maps with a "value" key, and
// temporal lists where each entry has a "value" key.
func ExtractPropertyYear(props map[string]any, key string) int {
	raw, ok := props[key]
	if !ok {
		return 0
	}

	var dateStr string

	switch v := raw.(type) {
	case string:
		dateStr = v
	case map[string]any:
		if val, ok := v["value"]; ok {
			dateStr = fmt.Sprint(val)
		}
	case []any:
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if val, ok := m["value"]; ok {
					dateStr = fmt.Sprint(val)
				}
			}
		}
	}

	return ExtractFirstYear(dateStr)
}

// ExtractFirstYear extracts the first 4-digit year from a date string.
// Returns 0 if no year is found.
func ExtractFirstYear(dateStr string) int {
	if dateStr == "" {
		return 0
	}

	match := temporalYearRegexp.FindStringSubmatch(dateStr)
	if len(match) < 2 {
		return 0
	}

	year, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}

	return year
}
