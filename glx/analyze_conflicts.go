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
	"fmt"
	"math"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/genealogix/glx/go-glx/glxdate"
)

// analyzeConflicts detects assertions with conflicting values for the same
// person/property combination.
// conflictPropKey identifies a person+property combination for conflict detection.
type conflictPropKey struct {
	personID string
	property string
}

// conflictValueInfo holds a value, its confidence level and, for a temporal
// property, the assertion's date.
type conflictValueInfo struct {
	value      string
	confidence string
	date       string
}

func analyzeConflicts(archive *glxlib.GLXFile) []AnalysisIssue {
	propValues := make(map[conflictPropKey][]conflictValueInfo)

	ids := sortedKeys(archive.Assertions)
	for _, id := range ids {
		a := archive.Assertions[id]
		if a == nil {
			continue
		}
		personID := a.Subject.Person
		if personID == "" || a.Property == "" || a.Value == "" {
			continue
		}

		info := conflictValueInfo{value: a.Value, confidence: a.Confidence}
		if isTemporalProperty(archive, a.Property) {
			info.date = a.Date.String()
		}

		key := conflictPropKey{personID: personID, property: a.Property}
		propValues[key] = append(propValues[key], info)
	}

	var issues []AnalysisIssue

	// Collect and sort keys for deterministic output order
	var keys []conflictPropKey
	for key := range propValues {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].personID != keys[j].personID {
			return keys[i].personID < keys[j].personID
		}

		return keys[i].property < keys[j].property
	})

	// Find properties with multiple distinct values
	for _, key := range keys {
		values := propValues[key]
		if isTemporalProperty(archive, key.property) {
			values = overlappingValues(values)
		}
		distinct := distinctValues(values)
		if len(distinct) < 2 {
			continue
		}

		name := personName(archive, key.personID)
		var parts []string
		for _, d := range distinct {
			entry := resolveConflictValue(d.value, key.property, archive)
			if d.confidence != "" {
				entry += " [" + d.confidence + "]"
			}
			parts = append(parts, entry)
		}

		issues = append(issues, AnalysisIssue{
			Category: "conflict",
			Severity: "high",
			Person:   key.personID,
			Property: key.property,
			Message: fmt.Sprintf("%s — %s has %d conflicting values: %s",
				name, key.property, len(distinct), strings.Join(parts, ", ")),
		})
	}

	sortIssues(issues)

	return issues
}

func isTemporalProperty(archive *glxlib.GLXFile, property string) bool {
	def, ok := archive.PersonProperties[property]

	return ok && def != nil && def.Temporal != nil && *def.Temporal
}

// overlappingValues keeps the entries whose date overlaps the date of an
// entry with a different value. An entry without a year overlaps every other.
func overlappingValues(values []conflictValueInfo) []conflictValueInfo {
	var out []conflictValueInfo
	for i, v := range values {
		lo, hi := spanOf(v.date)
		for j, o := range values {
			olo, ohi := spanOf(o.date)
			if i != j && v.value != o.value && lo <= ohi && olo <= hi {
				out = append(out, v)

				break
			}
		}
	}

	return out
}

// spanOf returns the first and last day key a date covers: the year for
// "1851", the month for "1851-03", the range for BET and FROM…TO, an open
// end for FROM or TO alone, all time for a date without a year. A qualifier
// does not widen it.
func spanOf(date string) (lo, hi int) {
	d, _ := glxdate.Parse(date) //nolint:errcheck // best-effort components are defined even for non-canonical dates
	lo, hi = math.MinInt, math.MaxInt
	if d.Year() == 0 {
		return lo, hi
	}
	if !d.IsRange() {
		return pointSpan(d)
	}
	if !d.IsOpenStart() {
		lo, _ = pointSpan(d.Start())
	}
	if !d.IsOpenEnded() {
		_, hi = pointSpan(d.End())
	}

	return lo, hi
}

// lastDayKey closes a month or year span; no month has more days.
const lastDayKey = 31

// pointSpan returns the first and last day key a point date covers.
func pointSpan(d glxdate.Date) (lo, hi int) {
	y := d.Year()
	month, hasMonth := d.Month()
	day, hasDay := d.Day()
	switch {
	case hasDay:
		return dayKey(y, month, day), dayKey(y, month, day)
	case hasMonth:
		return dayKey(y, month, 1), dayKey(y, month, lastDayKey)
	default:
		return dayKey(y, 1, 1), dayKey(y, 12, lastDayKey)
	}
}

// dayKey orders dates as integers: year, then month, then day. A BCE year
// is negative.
func dayKey(year, month, day int) int {
	return year*10000 + month*100 + day
}

// resolveConflictValue converts entity IDs to display names for place-reference
// properties. For other properties, returns the raw value. Uses the shared
// placeRefProperties set from places_runner.go to stay in sync.
func resolveConflictValue(value, property string, archive *glxlib.GLXFile) string {
	if placeRefProperties[property] {
		if place, ok := archive.Places[value]; ok && place != nil {
			return place.Name
		}
	}

	return value
}

// distinctValues returns unique values from a list, preserving the highest
// confidence level for each distinct value.
func distinctValues(values []conflictValueInfo) []conflictValueInfo {
	seen := make(map[string]string) // value → best confidence
	var order []string

	for _, v := range values {
		if _, exists := seen[v.value]; !exists {
			order = append(order, v.value)
			seen[v.value] = v.confidence
		} else if confidenceRank(v.confidence) < confidenceRank(seen[v.value]) {
			seen[v.value] = v.confidence
		}
	}

	sort.Strings(order)
	result := make([]conflictValueInfo, len(order))
	for i, val := range order {
		result[i] = conflictValueInfo{value: val, confidence: seen[val]}
	}

	return result
}

// confidenceRank returns a numeric rank for confidence levels (lower = higher confidence).
func confidenceRank(c string) int {
	switch strings.ToLower(c) {
	case "high":
		return 0
	case "medium-high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
