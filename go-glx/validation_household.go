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
)

// validateCensusHouseholds checks the structure of event household tallies
// and neighbor lists (#1332, #180). The neighbor person reference is checked
// by the reflection-based reference pass; this pass covers counts, bracket
// bounds, the fields that make an entry meaningful, and the tally sex, which
// like a person's sex property is checked against sex_types as a warning (many
// archives predate the sex_types vocabulary file).
//
// The check is deliberately structural. Brackets are not validated against a
// per-census column vocabulary (1800 "16 and under 26" etc.); that is left
// for a future census-schedule vocabulary.
func (glx *GLXFile) validateCensusHouseholds(result *ValidationResult) {
	for _, id := range sortedKeys(glx.Events) {
		event := glx.Events[id]
		if event == nil {
			continue
		}
		hasTally := event.Household != nil && len(event.Household.Tally) > 0
		if (hasTally || len(event.Neighbors) > 0) && event.Type != EventTypeCensus {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeEvents,
				SourceID:   id,
				Field:      "household",
				Message: fmt.Sprintf("%s[%s]: household and neighbors are defined for census events, but this event's type is %q",
					EntityTypeEvents, id, event.Type),
			})
		}
		if event.Household != nil {
			glx.validateTallyRows(id, event, result)
		}
		for i, n := range event.Neighbors {
			if n.Name == "" && n.Person == "" {
				result.Errors = append(result.Errors, ValidationError{
					SourceType:  EntityTypeEvents,
					SourceID:    id,
					SourceField: fmt.Sprintf("neighbors[%d]", i),
					Message: fmt.Sprintf("%s[%s].neighbors[%d]: a neighbor needs a name or a person reference",
						EntityTypeEvents, id, i),
				})
			}
		}
	}
}

// validateTallyRows checks each tally row's count and age bracket, and warns
// when more participants are marked as counted-but-unnamed than the tally
// holds.
func (glx *GLXFile) validateTallyRows(id string, event *Event, result *ValidationResult) {
	total := 0
	for i, row := range event.Household.Tally {
		field := fmt.Sprintf("household.tally[%d]", i)
		for _, msg := range tallyRowProblems(&row) {
			result.Errors = append(result.Errors, ValidationError{
				SourceType:  EntityTypeEvents,
				SourceID:    id,
				SourceField: field,
				Message:     fmt.Sprintf("%s[%s].%s: %s", EntityTypeEvents, id, field, msg),
			})
		}
		if row.Sex != "" && len(result.Vocabularies[VocabSexTypes]) > 0 {
			glx.checkVocabValue(EntityTypeEvents, id, field+".sex", VocabSexTypes, row.Sex, result.Vocabularies[VocabSexTypes], result)
		}
		if row.Sex == "" && row.AgeFrom == nil && row.AgeTo == nil && row.Status == "" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeEvents,
				SourceID:   id,
				Field:      field,
				Message: fmt.Sprintf("%s[%s].%s: row has no sex, age bracket, or status, so it does not say who was counted",
					EntityTypeEvents, id, field),
			})
		}
		if row.Count > 0 {
			total += row.Count
		}
	}

	if len(event.Household.Tally) == 0 {
		return
	}
	unnamed := 0
	for _, p := range event.Participants {
		if IsUnnamedParticipant(p) {
			unnamed++
		}
	}
	if unnamed > total {
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: EntityTypeEvents,
			SourceID:   id,
			Field:      "participants",
			Message: fmt.Sprintf("%s[%s]: %d participants are marked named: false but the household tally counts only %d persons",
				EntityTypeEvents, id, unnamed, total),
		})
	}
}

// tallyRowProblems returns the structural errors in one household tally row:
// a count below one, a negative age bound, or an inverted bracket.
func tallyRowProblems(row *HouseholdTallyRow) []string {
	var problems []string
	if row.Count < 1 {
		problems = append(problems, fmt.Sprintf("count must be at least 1, got %d", row.Count))
	}
	if row.AgeFrom != nil && *row.AgeFrom < 0 {
		problems = append(problems, fmt.Sprintf("age_from must not be negative, got %d", *row.AgeFrom))
	}
	if row.AgeTo != nil && *row.AgeTo < 0 {
		problems = append(problems, fmt.Sprintf("age_to must not be negative, got %d", *row.AgeTo))
	}
	if row.AgeFrom != nil && row.AgeTo != nil && *row.AgeFrom > *row.AgeTo {
		problems = append(problems, fmt.Sprintf("age_from (%d) is greater than age_to (%d)", *row.AgeFrom, *row.AgeTo))
	}

	return problems
}
