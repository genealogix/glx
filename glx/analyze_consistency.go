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

	glxlib "github.com/genealogix/glx/go-glx"
)

// analyzeConsistency checks for chronological inconsistencies across the archive.
func analyzeConsistency(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	issues = append(issues, checkBirthAfterDeath(archive)...)
	issues = append(issues, checkParentYoungerThanChild(archive)...)
	issues = append(issues, checkEventAfterDeath(archive)...)
	issues = append(issues, checkImplausibleLifespan(archive)...)
	issues = append(issues, checkMarriageBeforeBirth(archive)...)

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

		birthYear := glxlib.ExtractPropertyYear(person.Properties, "born_on")
		deathYear := glxlib.ExtractPropertyYear(person.Properties, "died_on")
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
			parentBirth := glxlib.ExtractPropertyYear(parent.Properties, "born_on")
			if parentBirth == 0 {
				continue
			}

			for _, childID := range childIDs {
				child := archive.Persons[childID]
				if child == nil {
					continue
				}
				childBirth := glxlib.ExtractPropertyYear(child.Properties, "born_on")
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
		year := glxlib.ExtractPropertyYear(person.Properties, "died_on")
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

		birthYear := glxlib.ExtractPropertyYear(person.Properties, "born_on")
		deathYear := glxlib.ExtractPropertyYear(person.Properties, "died_on")
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

			birthYear := glxlib.ExtractPropertyYear(person.Properties, "born_on")
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
