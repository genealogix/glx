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

	"github.com/genealogix/glx/go-glx/glxdate"
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

type vitalChronologyGroup struct {
	kind   string
	facts  []proofAssertion
	bounds []glxdate.Bounds
}

func vitalChronologyConflicts(archive *GLXFile, personID string, assertions []proofAssertion, opts ComparisonOptions) []ConflictGroup {
	groups := groupVitalChronologyDates(vitalChronologyDates(archive, personID, assertions), opts)
	var conflicts []ConflictGroup
	for i := range groups {
		burial := &groups[i]
		if burial.kind != EventTypeBurial && burial.kind != EventTypeCremation {
			continue
		}
		for j := range groups {
			death := &groups[j]
			if death.kind != EventTypeDeath {
				continue
			}
			if conflict, found := compareVitalChronologyGroups(burial, death); found {
				conflicts = append(conflicts, conflict)
			}
		}
	}

	return conflicts
}

func compareVitalChronologyGroups(burial, death *vitalChronologyGroup) (ConflictGroup, bool) {
	facts := append(slices.Clone(burial.facts), death.facts...)
	var comparisons []FactComparison
	selected := make(map[int]bool)
	for i, before := range burial.bounds {
		for j, after := range death.bounds {
			if !before.Known || !after.Known || before.Reversed || after.Reversed || before.Outer.End > after.Outer.Start {
				continue
			}
			right := len(burial.facts) + j
			comparisons = append(comparisons, FactComparison{Left: i, Right: right, Verdict: VerdictDefinite, Definite: true})
			selected[i], selected[right] = true, true
		}
	}
	if len(comparisons) == 0 {
		return ConflictGroup{}, false
	}
	result := ConflictGroup{
		Subject: "Death/burial chronology", Property: conflictDateType, Verdict: VerdictDefinite, Definite: true,
		Evaluation: FactEvaluation{Comparisons: comparisons, Verdict: VerdictDefinite, Definite: true},
	}
	seen := make(map[string]bool)
	for i := range facts {
		claim := &facts[i]
		result.Facts = append(result.Facts, researchFact(claim))
		result.Evaluation.Surviving = append(result.Evaluation.Surviving, i)
		if !selected[i] {
			continue
		}
		result.Evaluation.Selected = append(result.Evaluation.Selected, i)
		kind := death.kind
		if i < len(burial.facts) {
			kind = burial.kind
		}
		label := fmt.Sprintf("%s %s (%s:%s)", kind, claim.a.Value, claim.a.Subject.Type(), claim.subjectID)
		if i < len(burial.facts) {
			label += " before death"
		}
		if seen[label] {
			continue
		}
		seen[label] = true
		result.Values = append(result.Values, ConflictValue{Value: label, Confidence: claim.a.Confidence, Status: claim.a.Status})
	}

	return result, true
}

func groupVitalChronologyDates(dates []proofAssertion, opts ComparisonOptions) []vitalChronologyGroup {
	type groupKey struct{ kind, subjectType, subjectID string }
	positions := make(map[groupKey]int)
	var groups []vitalChronologyGroup
	for i := range dates {
		claim := &dates[i]
		kind := claim.eventType
		if claim.a.Subject.Person != "" {
			kind, _ = legacyVitalProperty(claim.a.Property)
		}
		key := groupKey{kind, claim.a.Subject.Type().String(), claim.subjectID}
		position, exists := positions[key]
		if !exists {
			position = len(groups)
			positions[key] = position
			groups = append(groups, vitalChronologyGroup{kind: kind})
		}
		date, err := DateString(claim.a.Value).Parse()
		var bounds glxdate.Bounds
		if err == nil {
			bounds = date.ChronologyBounds(opts.approximationWidth())
		}
		groups[position].facts = append(groups[position].facts, *claim)
		groups[position].bounds = append(groups[position].bounds, bounds)
	}

	return groups
}

func vitalChronologyDates(archive *GLXFile, personID string, assertions []proofAssertion) []proofAssertion {
	all := slices.Clone(assertions)
	canonicalLegacyVitalFacts(all, archive, personID)
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
		claim := &all[i]
		if claim.a.Participant != nil || AssertionIsDisproven(claim.a) {
			continue
		}
		legacyKind, _ := legacyVitalProperty(claim.a.Property)
		legacy := claim.a.Subject.Person == personID && claim.factProperty == conflictDateType && (legacyKind == EventTypeDeath || legacyKind == EventTypeBurial)
		structural := claim.a.Subject.Event != "" && events[claim.subjectID] && claim.a.Property == conflictDateType
		if structural || legacy {
			dates = append(dates, *claim)
		}
	}

	return dates
}
