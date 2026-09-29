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
	"strings"

	"github.com/spf13/cobra"

	glxlib "github.com/genealogix/glx/go-glx"
	"github.com/genealogix/glx/go-glx/glxdate"
)

func comparisonOptions(options []glxlib.ComparisonOptions) glxlib.ComparisonOptions {
	if len(options) > 0 {
		return options[0]
	}

	return glxlib.ComparisonOptions{}
}

const approximationFlag = "approximation-years"

func init() {
	for _, cmd := range []*cobra.Command{analyzeCmd, proofCmd, evidenceCmd, mergePersonsCmd} {
		cmd.Flags().Int(approximationFlag, glxlib.DefaultApproximationYears, "Tolerance in years for ABT, EST and CAL date values (0–10000)")
	}
}

func commandComparisonOptions(cmd *cobra.Command) (glxlib.ComparisonOptions, error) {
	width, err := cmd.Flags().GetInt(approximationFlag)
	if err != nil {
		return glxlib.ComparisonOptions{}, err
	}
	if width < 0 || width > glxdate.MaxApproximationYears {
		return glxlib.ComparisonOptions{}, fmt.Errorf("%w: --%s must be between 0 and 10000", ErrInvalidApproximation, approximationFlag)
	}

	return glxlib.ComparisonOptions{ApproximationYears: &width}, nil
}

func comparedAssertions(assertions []*glxlib.Assertion, archive *glxlib.GLXFile, opts glxlib.ComparisonOptions) ([]*glxlib.Assertion, glxlib.Verdict, bool) {
	if len(assertions) == 0 {
		return nil, glxlib.VerdictAgree, false
	}
	facts := make([]glxlib.FactValue, len(assertions))
	for i, a := range assertions {
		facts[i] = glxlib.AssertionFact(a)
	}
	def := glxlib.ConflictProperty(archive, assertions[0].Subject, assertions[0].Property)
	comparisons := glxlib.CompareFacts(facts, def, archive.Places, opts)
	selected := make(map[int]bool)
	verdict := glxlib.VerdictAgree
	definite := false
	for _, c := range comparisons {
		if c.IsConflict() {
			selected[c.Left], selected[c.Right] = true, true
			if verdictRank(c.Verdict) > verdictRank(verdict) {
				verdict = c.Verdict
			}
			definite = definite || c.Definite
		}
	}
	if len(selected) == 0 {
		for _, c := range comparisons {
			if c.Verdict == glxlib.VerdictResolved {
				selected[c.Left], selected[c.Right] = true, true
				verdict = glxlib.VerdictResolved
			}
		}
	}
	var out []*glxlib.Assertion
	for i, a := range assertions {
		if selected[i] {
			out = append(out, a)
		}
	}

	return out, verdict, definite
}

func verdictRank(v glxlib.Verdict) int {
	switch v {
	case glxlib.VerdictDefinite:
		return 3
	case glxlib.VerdictPossible:
		return 2
	case glxlib.VerdictDisputed:
		return 1
	default:
		return 0
	}
}

// Non-property assertions cannot enter a property/value comparison, but an
// explicit dispute about participation or existence still needs resolution.
// Let the shared engine classify the status, just as for a singleton property.
func nonPropertyProofDispute(pa *proofAssertion, archive *glxlib.GLXFile, opts glxlib.ComparisonOptions) (proofConflict, bool) {
	comparisons := glxlib.CompareFacts([]glxlib.FactValue{glxlib.AssertionFact(pa.a)}, nil, archive.Places, opts)
	if len(comparisons) == 0 || !comparisons[0].IsConflict() {
		return proofConflict{}, false
	}
	subject := describeProofSubject(pa, archive)
	if subject == "" {
		subject = pa.subjectID
	}
	property, value := "existence", "existence asserted"
	if participant := pa.a.Participant; participant != nil {
		property = "participation"
		value = personName(archive, participant.Person)
		if participant.Role != "" {
			value += " (" + participant.Role + ")"
		}
	}

	return proofConflict{
		Subject: subject, Property: property,
		Verdict: comparisons[0].Verdict, Definite: comparisons[0].Definite,
		Values: []proofConflictValue{{Value: value, Confidence: pa.a.Confidence, Status: pa.a.Status}},
	}, true
}

// appendDuplicateEventFacts compares repeated birth/death events for the person
// in the principal role. Parents and witnesses do not acquire the child's birth.
// A structural field is used only when no assertion records that field, so a
// denormalized conclusion never revives a researcher's disproven claim.
func appendDuplicateEventFacts(out []proofAssertion, archive *glxlib.GLXFile, personID string) []proofAssertion {
	groups := duplicateVitalEventIDs(archive, personID)
	for _, eventType := range []string{glxlib.EventTypeBirth, glxlib.EventTypeDeath} {
		ids := groups[eventType]
		if len(ids) < 2 {
			continue
		}
		for _, id := range ids {
			seen := make(map[string]bool)
			key := personID + ":" + eventType
			for i := range out {
				if out[i].a.Subject.Event == id {
					out[i].factKey = key
					seen[out[i].a.Property] = true
				}
			}
			ev := archive.Events[id]
			for _, field := range []struct{ property, value string }{{"date", string(ev.Date)}, {eventFieldPlace, ev.PlaceID}} {
				if field.value == "" || seen[field.property] {
					continue
				}
				a := &glxlib.Assertion{Subject: glxlib.EntityRef{Event: id}, Property: field.property, Value: field.value}
				out = append(out, proofAssertion{id: id + ":" + field.property, a: a, subjectID: id, eventType: eventType, factKey: key})
			}
		}
	}

	return out
}

func factDisplay(value string, subject glxlib.EntityRef, property string, archive *glxlib.GLXFile) string {
	display := resolveAssertionValue(value, property, subject, archive)
	def := glxlib.ConflictProperty(archive, subject, property)
	if def != nil && def.ReferenceType == "places" {
		if p := archive.Places[value]; p != nil {
			for id, other := range archive.Places {
				if id != value && other != nil && strings.EqualFold(other.Name, p.Name) {
					return display + " (" + value + ")"
				}
			}
		}
	}

	return display
}

func duplicateVitalEventIDs(archive *glxlib.GLXFile, personID string) map[string][]string {
	groups := make(map[string][]string)
	for _, id := range sortedKeys(archive.Events) {
		ev := archive.Events[id]
		if ev == nil || (ev.Type != glxlib.EventTypeBirth && ev.Type != glxlib.EventTypeDeath) {
			continue
		}
		for _, p := range ev.Participants {
			if p.Person == personID && (p.Role == "subject" || p.Role == "principal" || p.Role == "" || p.Role == glxlib.ParticipantRoleChild || p.Role == "deceased") {
				groups[ev.Type] = append(groups[ev.Type], id)

				break
			}
		}
	}

	return groups
}
