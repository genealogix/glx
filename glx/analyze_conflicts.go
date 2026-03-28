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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// analyzeConflicts detects assertions with conflicting values for the same
// person/property combination.
// conflictPropKey identifies a person+property combination for conflict detection.
type conflictPropKey struct {
	personID string
	property string
}

// conflictValueInfo holds a value and its confidence level.
type conflictValueInfo struct {
	value      string
	confidence string
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

		key := conflictPropKey{personID: personID, property: a.Property}
		propValues[key] = append(propValues[key], conflictValueInfo{
			value:      a.Value,
			confidence: a.Confidence,
		})
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
	case "disputed":
		return 4
	default:
		return 5
	}
}
