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
	"slices"
)

// VitalChronologyConflicts reports burial/cremation dates that certainly precede
// a person's death. Partial dates, ranges, calendars and approximation tolerance
// use the shared date bounds. Disproven date assertions do not participate.
func VitalChronologyConflicts(archive *GLXFile, personID string) []ConflictGroup {
	if archive == nil {
		return nil
	}

	return vitalChronologyConflicts(archive, personID, collectPersonProofAssertions(archive, personID), ComparisonOptions{})
}

func vitalChronologyConflicts(archive *GLXFile, personID string, assertions []proofAssertion, opts ComparisonOptions) []ConflictGroup {
	dates := vitalChronologyDates(archive, personID, assertions)
	var conflicts []ConflictGroup
	for i := range dates {
		burial := &dates[i]
		if burial.eventType != EventTypeBurial && burial.eventType != EventTypeCremation {
			continue
		}
		before, err := DateString(burial.a.Value).Parse()
		if err != nil {
			continue
		}
		beforeBounds := before.ChronologyBounds(opts.approximationWidth())
		for j := range dates {
			death := &dates[j]
			if death.eventType != EventTypeDeath {
				continue
			}
			after, parseErr := DateString(death.a.Value).Parse()
			if parseErr != nil {
				continue
			}
			afterBounds := after.ChronologyBounds(opts.approximationWidth())
			if !beforeBounds.Known || !afterBounds.Known || beforeBounds.Reversed || afterBounds.Reversed || beforeBounds.Outer.End > afterBounds.Outer.Start {
				continue
			}
			comparisons := []FactComparison{{Left: 0, Right: 1, Verdict: VerdictDefinite, Definite: true}}
			conflicts = append(conflicts, ConflictGroup{
				Subject: "Death/burial chronology", Property: conflictDateType, Verdict: VerdictDefinite, Definite: true,
				Facts:      []ResearchFact{researchFact(burial), researchFact(death)},
				Evaluation: FactEvaluation{Comparisons: comparisons, Selected: []int{0, 1}, Surviving: []int{0, 1}, Verdict: VerdictDefinite, Definite: true},
				Values: []ConflictValue{
					{Value: fmt.Sprintf("%s %s (%s) before death", burial.eventType, burial.a.Value, burial.subjectID), Confidence: burial.a.Confidence, Status: burial.a.Status},
					{Value: fmt.Sprintf("death %s (%s)", death.a.Value, death.subjectID), Confidence: death.a.Confidence, Status: death.a.Status},
				},
			})
		}
	}

	return conflicts
}

func vitalChronologyDates(archive *GLXFile, personID string, assertions []proofAssertion) []proofAssertion {
	all := slices.Clone(assertions)
	events := make(map[string]bool)
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || !isDeathEventType(event.Type) {
			continue
		}
		for _, participant := range event.Participants {
			if participant.Person == personID && isVitalPrincipal(event.Type, participant.Role) {
				events[id] = true
				all = appendVitalEventFields(all, archive, id, id, map[string]bool{conflictDateType: true})

				break
			}
		}
	}
	var dates []proofAssertion
	for i := range all {
		assertion := &all[i]
		if events[assertion.subjectID] && assertion.a.Property == conflictDateType && assertion.a.Participant == nil && !AssertionIsDisproven(assertion.a) {
			dates = append(dates, *assertion)
		}
	}

	return dates
}
