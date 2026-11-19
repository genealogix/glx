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
	dates, ownEvents := vitalChronologyDates(archive, personID, assertions)
	groups := groupVitalChronologyDates(dates, ownEvents, opts)
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

// groupVitalChronologyDates gathers the date claims of each vital observation.
// A claim belongs to the observation of the kind its property names (a
// death_date recorded on a burial is a death date). A legacy person-level
// claim, or an event-scoped alias of another kind, joins the person's own
// event of that kind when there is exactly one, so the same burial is never
// compared twice; with none or several it stays a separate observation.
func groupVitalChronologyDates(dates []proofAssertion, ownEvents map[string][]string, opts ComparisonOptions) []vitalChronologyGroup {
	type groupKey struct{ kind, subjectType, subjectID string }
	positions := make(map[groupKey]int)
	var groups []vitalChronologyGroup
	for i := range dates {
		claim := &dates[i]
		kind := vitalChronologyKind(claim)
		key := groupKey{kind, claim.a.Subject.Type().String(), claim.subjectID}
		if claim.a.Subject.Person != "" || kind != claim.eventType {
			if ids := ownEvents[kind]; len(ids) == 1 {
				key = groupKey{kind, EntityTypeEvents.String(), ids[0]}
			}
		}
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

// vitalChronologyKind is the vital kind a date claim describes: the kind its
// property names (died_on, burial_date, ...) or else its event's type.
func vitalChronologyKind(claim *proofAssertion) string {
	if kind, field := legacyVitalProperty(claim.a.Property); field == conflictDateType {
		return kind
	}

	return claim.eventType
}

// vitalChronologyDates selects the person's death and burial date claims:
// legacy person-level dates, the date of each of their own death-type events,
// and death/burial date aliases recorded on those events. A stored event date
// that a surviving assertion no longer supports is not synthesized back.
func vitalChronologyDates(archive *GLXFile, personID string, assertions []proofAssertion) ([]proofAssertion, map[string][]string) {
	all := slices.Clone(assertions)
	canonicalLegacyVitalFacts(all, archive, personID)
	events := make(map[string]bool)
	ownEvents := make(map[string][]string)
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || !isDeathEventType(event.Type) || !isEventVitalPrincipal(event, personID) {
			continue
		}
		events[id] = true
		ownEvents[event.Type] = append(ownEvents[event.Type], id)
		all = appendVitalEventFields(all, archive, id, id, map[string]bool{conflictDateType: true})
	}
	var dates []proofAssertion
	for i := range all {
		if isVitalChronologyDate(archive, personID, events, &all[i]) {
			dates = append(dates, all[i])
		}
	}

	return dates, ownEvents
}

// isVitalChronologyDate reports whether a collected claim is one of the
// person's surviving death or burial dates.
func isVitalChronologyDate(archive *GLXFile, personID string, events map[string]bool, claim *proofAssertion) bool {
	if claim.a.Participant != nil || AssertionIsDisproven(claim.a) {
		return false
	}
	if claim.synthetic && claim.a.Subject.Event != "" &&
		!structuralVitalFieldAvailable(archive, personID, claim.a.Subject.Event, claim.a.Property, claim.a.Value) {
		return false
	}
	legacyKind, legacyField := legacyVitalProperty(claim.a.Property)
	deathOrBurial := (legacyKind == EventTypeDeath || legacyKind == EventTypeBurial) && legacyField == conflictDateType
	if claim.a.Subject.Person == personID {
		return deathOrBurial && claim.factProperty == conflictDateType
	}
	if claim.a.Subject.Event == "" || !events[claim.subjectID] {
		return false
	}

	return claim.a.Property == conflictDateType || deathOrBurial && vitalAliasIsDate(archive, claim)
}

// vitalAliasIsDate reports whether a died_on/death_date/buried_on alias on an
// event can be read as a date. Proof reads an undefined alias as the vital
// date it names; an archive that defines the property otherwise (a custom
// type, a vocabulary) keeps its own meaning.
func vitalAliasIsDate(archive *GLXFile, claim *proofAssertion) bool {
	definition := ConflictProperty(archive, claim.a.Subject, claim.a.Property)

	return definition == nil || hasVitalSemantics(definition, conflictDateType)
}
