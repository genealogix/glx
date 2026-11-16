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
	"slices"
	"sort"
	"strings"
)

// Structural fields are denormalized copies. Once a field is asserted, only a
// surviving claim that agrees with the stored value permits that copy to answer
// a proof question. The assertions themselves can still supply a different value.
func structuralVitalFieldAvailable(archive *GLXFile, personID, eventID, property, value string) bool {
	if value == "" {
		return false
	}
	event := archive.Events[eventID]
	if event == nil {
		return false
	}
	asserted := false
	for _, claim := range archive.Assertions {
		if claim == nil || claim.Participant != nil {
			continue
		}
		kind, field := legacyVitalProperty(claim.Property)
		direct := claim.Subject.Event == eventID && claim.Property == property
		legacy := claim.Subject.Person == personID && kind == event.Type && field == property
		definition := ConflictProperty(archive, claim.Subject, claim.Property)
		if (!direct && !legacy) || !hasVitalSemantics(definition, property) {
			continue
		}
		comparison := compareFacts([]FactValue{{Value: claim.Value, Date: claim.Date}, {Value: value}}, definition, archive.Places, ComparisonOptions{})
		agrees := len(comparison) > 0 && comparison[0].Verdict == VerdictAgree
		// A legacy claim about a different value does not reject this
		// independent structural observation. Normal fact comparison handles it.
		if direct || agrees {
			asserted = true
		}
		if agrees && !AssertionIsDisproven(claim) {
			return true
		}
	}

	return !asserted
}

func proofPrimaryEvent(archive *GLXFile, personID, eventType string) (string, *Event) {
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || event.Type != eventType {
			continue
		}
		for _, participant := range event.Participants {
			if participant.Person == personID && isVitalPrincipal(eventType, participant.Role) {
				return id, event
			}
		}
	}

	return "", nil
}

func primaryStructuralVitalDate(archive *GLXFile, personID, topic string) bool {
	id, event := proofPrimaryEvent(archive, personID, topic)

	return event != nil && structuralVitalFieldAvailable(archive, personID, id, conflictDateType, string(event.Date))
}

func hasPrimaryVitalDateAssertion(relevant []proofAssertion, topic string) bool {
	for i := range relevant {
		if primaryVitalDateAssertion(&relevant[i], topic) {
			return true
		}
	}

	return false
}

func isProxyVitalType(topic, eventType string) bool {
	return topic == topicBirth && (eventType == EventTypeBaptism || eventType == EventTypeChristening) ||
		topic == topicDeath && (eventType == EventTypeBurial || eventType == EventTypeCremation)
}

// A rite's recorded data can describe a bound without supplying evidence or
// confidence. supportLevel continues to exclude every synthetic observation.
func proxyVitalAnswer(relevant []proofAssertion, archive *GLXFile, topic string) (string, bool) {
	seen := make(map[string]bool)
	var values []string
	hasDate := false
	for i := range relevant {
		claim := &relevant[i]
		if !isProxyVitalType(topic, claim.eventType) || claim.a.Participant != nil || AssertionIsDisproven(claim.a) {
			continue
		}
		if claim.a.Property != conflictDateType && claim.a.Property != eventFieldPlace {
			continue
		}
		value := proofAssertionValue(claim.a, archive)
		if value == "" {
			continue
		}
		if claim.a.Property == conflictDateType {
			hasDate = true
		}
		value = qualifyProxyVitalAnswer(topic, claim, value)
		if seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	if !hasDate {
		return "", false
	}
	sort.Strings(values)

	return strings.Join(values, "; ") + ".", true
}

// Proof can describe recorded proxy fields without adding them to the public
// evidence/claim index or promoting their confidence. Existing canonical fact
// keys stay intact for legacy comparisons; separate rites keep separate keys.
func appendProxyVitalProofFields(all []proofAssertion, archive *GLXFile, personID, topic string) []proofAssertion {
	if topic != topicBirth && topic != topicDeath {
		return all
	}
	all = slices.DeleteFunc(all, func(claim proofAssertion) bool {
		return claim.synthetic && claim.a.Subject.Event != "" &&
			!structuralVitalFieldAvailable(archive, personID, claim.a.Subject.Event, claim.a.Property, claim.a.Value)
	})
	for _, id := range sortedKeys(archive.Events) {
		event := archive.Events[id]
		if event == nil || !isProxyVitalType(topic, event.Type) {
			continue
		}
		principal := false
		for _, participant := range event.Participants {
			if participant.Person == personID && isVitalPrincipal(event.Type, participant.Role) {
				principal = true

				break
			}
		}
		if !principal {
			continue
		}
		for _, field := range []struct{ property, value string }{{conflictDateType, string(event.Date)}, {eventFieldPlace, event.PlaceID}} {
			seen := false
			for i := range all {
				claim := &all[i]
				if claim.a.Subject.Event == id && claim.a.Participant == nil && claim.a.Property == field.property {
					seen = true

					break
				}
			}
			if seen || !structuralVitalFieldAvailable(archive, personID, id, field.property, field.value) {
				continue
			}
			claim := &Assertion{Subject: EntityRef{Event: id}, Property: field.property, Value: field.value}
			all = append(all, proofAssertion{id: id + ":" + field.property, a: claim, subjectID: id, eventType: event.Type, factKey: "events:" + id, factProperty: field.property, synthetic: true})
		}
	}

	return all
}
