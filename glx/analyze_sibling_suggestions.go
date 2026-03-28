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

	glxlib "github.com/genealogix/glx/go-glx"
)

// suggestSiblingRecords recommends searching siblings' census records when a
// person has no known parents (brickwall). Siblings are identified as other
// children of the same parent. The 1880+ federal censuses are particularly
// valuable because they list parents' birthplaces.
func suggestSiblingRecords(archive *glxlib.GLXFile) []AnalysisIssue {
	childHasParents := buildChildHasParentsIndex(archive)
	parentToChildren := buildParentToChildrenIndex(archive)

	// Precompute census year index once
	personCensusYears := make(map[string]map[int]bool)
	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeCensus {
			continue
		}
		year := glxlib.ExtractFirstYear(string(event.Date))
		if year == 0 {
			continue
		}
		for _, p := range event.Participants {
			if p.Person == "" {
				continue
			}
			if personCensusYears[p.Person] == nil {
				personCensusYears[p.Person] = make(map[int]bool)
			}
			personCensusYears[p.Person][year] = true
		}
	}

	var issues []AnalysisIssue

	// Find persons without parents (brickwalls)
	for _, personID := range sortedPersonIDs(archive.Persons) {
		if childHasParents[personID] {
			continue // has parents, not a brickwall
		}

		// Find this person's children
		children := findChildrenOfPerson(personID, parentToChildren)
		if len(children) == 0 {
			continue // no children, so no siblings to search
		}

		// Find siblings: other children of the same parents as this person's children
		// Wait — the brickwall person IS the parent. We want to suggest searching
		// the brickwall person's CHILDREN's census records, since children's 1880+
		// censuses list parents' birthplaces (which helps identify the brickwall's parents).
		//
		// Actually, re-reading the issue: "When a person has unknown parents (a brickwall),
		// suggest searching census records for known or probable siblings"
		// So the brickwall person has no parents. We need to find the brickwall person's
		// SIBLINGS (other children of the brickwall person's parents).
		// But if the brickwall has no parents, there are no known siblings via relationships.
		//
		// The real pattern is: find persons whose parents are unknown, then find their
		// children (the brickwall person's children), and suggest searching those children's
		// 1880+ census records — because those records list the parents' birthplaces,
		// which are the brickwall person's birthplace info.

		name := personName(archive, personID)

		for _, childID := range children {
			childPerson := archive.Persons[childID]
			if childPerson == nil {
				continue
			}

			childName := personName(archive, childID)
			childBirthYear := glxlib.ExtractPropertyYear(childPerson.Properties, glxlib.PersonPropertyBornOn)
			if childBirthYear == 0 {
				continue
			}

			existing := personCensusYears[childID]

			// Suggest 1880+ censuses for children (lists parents' birthplaces)
			for _, year := range usFederalCensusYears {
				if year < 1880 || year < childBirthYear {
					continue
				}
				if year > childBirthYear+maxLifespan {
					break
				}
				if existing[year] {
					continue
				}

				age := year - childBirthYear
				msg := fmt.Sprintf("%s — search %d census for %s (child, age ~%d) — lists parents' birthplaces",
					name, year, childName, age)

				issues = append(issues, AnalysisIssue{
					Category: "suggestion",
					Severity: "info",
					Person:   personID,
					Message:  msg,
				})
			}
		}
	}

	return issues
}

// buildParentToChildrenIndex returns a map from parent ID to their child IDs.
func buildParentToChildrenIndex(archive *glxlib.GLXFile) map[string][]string {
	index := make(map[string][]string)

	for _, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}

		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case "parent":
				parentIDs = append(parentIDs, p.Person)
			case "child":
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, parentID := range parentIDs {
			index[parentID] = append(index[parentID], childIDs...)
		}
	}

	// Deduplicate and sort for deterministic output
	for id := range index {
		index[id] = uniqueSorted(index[id])
	}

	return index
}

// findChildrenOfPerson returns the children of a person from the index.
func findChildrenOfPerson(personID string, index map[string][]string) []string {
	return index[personID]
}

// uniqueSorted returns a sorted, deduplicated copy of a string slice.
func uniqueSorted(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(s))
	var result []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	sort.Strings(result)
	return result
}
