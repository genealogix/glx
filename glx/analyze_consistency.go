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

// extractEventYear looks up a person's event (e.g. birth or death) and returns
// the first year from its date, or 0 if no matching event exists.
func extractEventYear(archive *glxlib.GLXFile, personID, eventType string) int {
	_, event := glxlib.FindPersonEvent(archive, personID, eventType)
	if event == nil {
		return 0
	}
	return glxlib.ExtractFirstYear(string(event.Date))
}

// analyzeConsistency checks for chronological inconsistencies across the archive.
func analyzeConsistency(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	issues = append(issues, checkBirthAfterDeath(archive)...)
	issues = append(issues, checkParentYoungerThanChild(archive)...)
	issues = append(issues, checkEventAfterDeath(archive)...)
	issues = append(issues, checkImplausibleLifespan(archive)...)
	issues = append(issues, checkMarriageBeforeBirth(archive)...)
	issues = append(issues, checkDuplicateSiblingNames(archive)...)

	return issues
}

// checkBirthAfterDeath reports persons whose death year precedes their birth year.
func checkBirthAfterDeath(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		deathYear := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if birthYear == 0 || deathYear == 0 {
			continue
		}

		if deathYear < birthYear {
			name := personName(archive, id)
			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "high",
				Person:   id,
				Message:  fmt.Sprintf("%s — death year (%d) before birth year (%d)", name, deathYear, birthYear),
			})
		}
	}

	return issues
}

// checkParentYoungerThanChild reports parent-child relationships where the
// parent was born after the child.
func checkParentYoungerThanChild(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if !isParentChildType(rel.Type) {
			continue
		}

		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case glxlib.ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case glxlib.ParticipantRoleChild:
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, parentID := range parentIDs {
			parent := archive.Persons[parentID]
			if parent == nil {
				continue
			}
			parentBirth := extractEventYear(archive, parentID, glxlib.EventTypeBirth)
			if parentBirth == 0 {
				continue
			}

			for _, childID := range childIDs {
				child := archive.Persons[childID]
				if child == nil {
					continue
				}
				childBirth := extractEventYear(archive, childID, glxlib.EventTypeBirth)
				if childBirth == 0 {
					continue
				}

				if parentBirth > childBirth {
					parentName := personName(archive, parentID)
					childName := personName(archive, childID)
					issues = append(issues, AnalysisIssue{
						Category: "consistency",
						Severity: "high",
						Person:   childID,
						Message: fmt.Sprintf("Parent %s (born %d) born after child %s (born %d)",
							parentName, parentBirth, childName, childBirth),
					})
				}
			}
		}
	}

	return issues
}

// checkEventAfterDeath reports persons who participate in events dated after
// their death (excluding burial/cremation/probate/will events).
func checkEventAfterDeath(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build death year index
	deathYears := make(map[string]int)
	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		year := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if year > 0 {
			deathYears[id] = year
		}
	}

	// Events that can legitimately occur after death
	postDeathEvents := map[string]bool{
		glxlib.EventTypeBurial:    true,
		glxlib.EventTypeCremation: true,
		glxlib.EventTypeProbate:   true,
		glxlib.EventTypeWill:      true,
	}

	var issues []AnalysisIssue

	for _, event := range archive.Events {
		if event == nil {
			continue
		}
		if postDeathEvents[event.Type] {
			continue
		}

		eventYear := glxlib.ExtractFirstYear(string(event.Date))
		if eventYear == 0 {
			continue
		}

		for _, p := range event.Participants {
			deathYear, ok := deathYears[p.Person]
			if !ok {
				continue
			}

			if eventYear > deathYear {
				name := personName(archive, p.Person)
				issues = append(issues, AnalysisIssue{
					Category: "consistency",
					Severity: "high",
					Person:   p.Person,
					Message: fmt.Sprintf("%s — %s event (%d) after death (%d)",
						name, event.Type, eventYear, deathYear),
				})
			}
		}
	}

	return issues
}

// checkImplausibleLifespan reports persons with a lifespan exceeding 110 years.
func checkImplausibleLifespan(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		deathYear := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if birthYear == 0 || deathYear == 0 {
			continue
		}

		age := deathYear - birthYear
		if age > 110 {
			name := personName(archive, id)
			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "medium",
				Person:   id,
				Message:  fmt.Sprintf("%s — implausible lifespan of %d years (%d–%d)", name, age, birthYear, deathYear),
			})
		}
	}

	return issues
}

// checkMarriageBeforeBirth reports marriage events that occur before a
// participant's birth.
func checkMarriageBeforeBirth(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeMarriage {
			continue
		}

		eventYear := glxlib.ExtractFirstYear(string(event.Date))
		if eventYear == 0 {
			continue
		}

		for _, p := range event.Participants {
			person := archive.Persons[p.Person]
			if person == nil {
				continue
			}

			birthYear := extractEventYear(archive, p.Person, glxlib.EventTypeBirth)
			if birthYear == 0 {
				continue
			}

			if eventYear < birthYear {
				name := personName(archive, p.Person)
				issues = append(issues, AnalysisIssue{
					Category: "consistency",
					Severity: "high",
					Person:   p.Person,
					Message:  fmt.Sprintf("%s — marriage (%d) before birth (%d)", name, eventYear, birthYear),
				})
			}
		}
	}

	return issues
}

// checkDuplicateSiblingNames flags parents whose children share the same given name.
// siblingInfo holds an ID and display name for duplicate-name detection.
type siblingInfo struct {
	id       string
	fullName string
}

func checkDuplicateSiblingNames(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build parent → children index
	parentChildren := make(map[string][]string)
	for _, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		var parents, children []string
		for _, p := range rel.Participants {
			if p.Role == glxlib.ParticipantRoleParent {
				parents = append(parents, p.Person)
			} else if p.Role == glxlib.ParticipantRoleChild {
				children = append(children, p.Person)
			}
		}
		for _, parentID := range parents {
			parentChildren[parentID] = append(parentChildren[parentID], children...)
		}
	}

	var issues []AnalysisIssue

	for _, parentID := range sortedPersonIDs(archive.Persons) {
		childIDs := parentChildren[parentID]
		if len(childIDs) < 2 {
			continue
		}

		uniqueChildren := dedupeStrings(childIDs)

		// Map lowercase given name → siblings with that name
		givenNames := make(map[string][]siblingInfo)
		// Track original capitalization for display
		givenDisplay := make(map[string]string)

		for _, cid := range uniqueChildren {
			person, ok := archive.Persons[cid]
			if !ok || person == nil {
				continue
			}
			fullName := glxlib.PersonDisplayName(person)
			given := extractGivenName(person)
			if given == "" {
				continue
			}
			key := strings.ToLower(given)
			givenNames[key] = append(givenNames[key], siblingInfo{id: cid, fullName: fullName})
			if _, exists := givenDisplay[key]; !exists {
				givenDisplay[key] = given
			}
		}

		parentName := personName(archive, parentID)
		for key, siblings := range givenNames {
			if len(siblings) < 2 {
				continue
			}

			if allReplacementPattern(siblings, archive) {
				continue
			}

			var names []string
			for _, c := range siblings {
				names = append(names, fmt.Sprintf("%s (%s)", c.id, c.fullName))
			}
			sort.Strings(names)

			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "medium",
				Person:   parentID,
				Message: fmt.Sprintf("%s — children share given name %q: %s",
					parentName, givenDisplay[key], strings.Join(names, " and ")),
			})
		}
	}

	return issues
}

// extractGivenName returns the given (first) name from a person's name property.
// Uses the "given" field if available in structured name data; otherwise splits
// the display name and takes the first token.
func extractGivenName(person *glxlib.Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}
	// Try structured name fields first
	raw, ok := person.Properties["name"]
	if ok {
		if m, ok := raw.(map[string]any); ok {
			if fields, ok := m["fields"].(map[string]any); ok {
				if given, ok := fields["given"].(string); ok && given != "" {
					return strings.Fields(given)[0]
				}
			}
		}
	}
	// Fall back to first token of display name
	fullName := glxlib.PersonDisplayName(person)
	parts := strings.Fields(fullName)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// allReplacementPattern returns true if all duplicate-named siblings follow the
// "replacement" pattern: each earlier child died before or in the same year the
// next was born.
func allReplacementPattern(siblings []siblingInfo, archive *glxlib.GLXFile) bool {
	type yearPair struct {
		birth int
		death int
	}
	var pairs []yearPair
	for _, c := range siblings {
		_, ok := archive.Persons[c.id]
		if !ok {
			return false
		}
		birth := extractEventYear(archive, c.id, glxlib.EventTypeBirth)
		death := extractEventYear(archive, c.id, glxlib.EventTypeDeath)
		pairs = append(pairs, yearPair{birth: birth, death: death})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].birth < pairs[j].birth })

	for i := 0; i < len(pairs)-1; i++ {
		if pairs[i].death == 0 || pairs[i+1].birth == 0 {
			return false
		}
		// If the earlier child died strictly after the next was born,
		// they overlapped — not a replacement. Same year (death == birth)
		// is allowed since infant death + replacement in the same year
		// was a common historical pattern.
		if pairs[i].death > pairs[i+1].birth {
			return false
		}
	}
	return true
}

// dedupeStrings returns unique strings preserving order.
func dedupeStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
