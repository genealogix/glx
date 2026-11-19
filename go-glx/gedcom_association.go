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
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	gedcom551ContextPattern          = regexp.MustCompile(`^Event: (.+?) \(([^;()]+); ([^)]+)\)(.*)$`)
	gedcom551ContextAttributePattern = regexp.MustCompile(`; (date|place|subjects): `)
)

const gedcom551EncodedContextPrefix = "; context: "

// associationMatcher indexes the imported events by host person once, so
// matching an association's context does not rescan every event, and shares
// one place-string cache across all associations.
type associationMatcher struct {
	conv         *ConversionContext
	eventsByHost map[string][]*Event
	places       *ExportContext
}

func newAssociationMatcher(conv *ConversionContext) *associationMatcher {
	matcher := &associationMatcher{
		conv:         conv,
		eventsByHost: make(map[string][]*Event),
		places:       &ExportContext{GLX: conv.GLX, PlaceStrings: make(map[string]string)},
	}
	for _, event := range conv.GLX.Events {
		if event == nil {
			continue
		}
		var hosts []string
		for _, participant := range event.Participants {
			if participant.Person != "" && isFamilyEventMemberRole(participant.Role) && !slices.Contains(hosts, participant.Person) {
				hosts = append(hosts, participant.Person)
			}
		}
		for _, host := range eventHostIDs(event) {
			if !slices.Contains(hosts, host) {
				hosts = append(hosts, host)
			}
		}
		for _, host := range hosts {
			matcher.eventsByHost[host] = append(matcher.eventsByHost[host], event)
		}
	}

	return matcher
}

// importIndividualAssociations waits until both INDI and FAM events exist.
func importIndividualAssociations(records []*GEDCOMRecord, conv *ConversionContext) {
	var matcher *associationMatcher
	for _, individual := range records {
		hostID, ok := conv.PersonIDMap[individual.XRef]
		if !ok || hostID == "" || conv.GLX.Persons[hostID] == nil {
			for _, association := range individual.SubRecords {
				if association.Tag == GedcomTagAsso {
					conv.addWarning(association.Line, GedcomTagAsso, fmt.Sprintf("Association on unconverted individual %s skipped", individual.XRef))
				}
			}

			continue
		}
		for _, association := range individual.SubRecords {
			if association.Tag != GedcomTagAsso {
				continue
			}
			participant := convertASSOToParticipant(association, conv)
			if participant == nil {
				continue
			}
			if matcher == nil {
				matcher = newAssociationMatcher(conv)
			}
			event := matcher.match(hostID, association)
			if event != nil {
				restoreAssociationRole(participant, association, conv)
				participant.Notes = slices.DeleteFunc(participant.Notes, func(note string) bool {
					return gedcom551ContextPattern.MatchString(note)
				})
				// A family event can be described under both spouses. Merge the
				// repeated projection without creating duplicate participants.
				appendAssociationParticipant(event, *participant)

				continue
			}
			// An unscoped association is a social connection, not evidence of
			// attendance at a guessed event. Retain RELA and all original notes.
			notes := slices.Clone(participant.Notes)
			for _, child := range association.SubRecords {
				if child.Tag == GedcomTagRela && child.Value != "" {
					note := "GEDCOM RELA: " + child.Value
					if !slices.Contains(notes, note) {
						notes = append(notes, note)
					}
				}
			}
			if hostID == participant.Person {
				conv.addWarning(association.Line, GedcomTagAsso, "Self association retained in person notes")
				conv.GLX.Persons[hostID].Notes = append(conv.GLX.Persons[hostID].Notes, notes...)

				continue
			}
			id := generateRelationshipID(conv)
			conv.GLX.Relationships[id] = &Relationship{
				Type:         "associate",
				Participants: []Participant{{Person: hostID, Role: ParticipantRoleAssociate}, {Person: participant.Person, Role: ParticipantRoleAssociate}},
				Notes:        notes,
			}
			conv.Stats.RelationshipsCreated++
			conv.addWarning(association.Line, GedcomTagAsso, fmt.Sprintf("Association with %s retained as an associate relationship; no unique event context", participant.Person))
		}
	}
}

func (matcher *associationMatcher) match(hostID string, association *GEDCOMRecord) *Event {
	conv := matcher.conv
	context := associationEventContext(association, conv)
	if context == nil {
		return nil
	}
	label, tag := context[1], context[2]
	attributes, valid := associationContextAttributes(context[4])
	if !valid {
		return nil
	}
	var match *Event
	for _, event := range matcher.eventsByHost[hostID] {
		if tag == GedcomTagEven {
			if vocabularyLabel(conv.GLX.EventTypes, event.Type) != label {
				continue
			}
		} else if conv.GEDCOMIndex.EventTypes[tag] != event.Type {
			continue
		}
		if string(event.Date) != string(parseGEDCOMDate(attributes[conflictDateType])) {
			continue
		}
		if place := attributes[eventFieldPlace]; place != "" {
			if resolvePlaceString(event.PlaceID, matcher.places) != place {
				continue
			}
		}
		if match != nil {
			return nil
		} // ambiguous context: preserve as a relationship
		match = event
	}

	return match
}

func associationContextAttributes(tail string) (map[string]string, bool) {
	attributes := make(map[string]string)
	if encoded, ok := strings.CutPrefix(tail, gedcom551EncodedContextPrefix); ok {
		if err := json.Unmarshal([]byte(encoded), &attributes); err != nil || attributes == nil {
			return nil, false
		}

		return attributes, true
	}
	// Older notes remain readable. Only a known attribute begins a new field;
	// an unlabelled semicolon belongs to its place or subject text.
	fields := gedcom551ContextAttributePattern.FindAllStringSubmatchIndex(tail, -1)
	for i, field := range fields {
		end := len(tail)
		if i+1 < len(fields) {
			end = fields[i+1][0]
		}
		attributes[tail[field[2]:field[3]]] = tail[field[1]:end]
	}

	return attributes, true
}

func restoreAssociationRole(participant *Participant, association *GEDCOMRecord, conv *ConversionContext) {
	neutral := false
	for _, child := range association.SubRecords {
		if child.Tag == GedcomTagRela && child.Value == gedcom551NeutralRelation {
			neutral = true

			break
		}
	}
	if !neutral {
		return
	}
	for _, note := range participant.Notes {
		if roleLabel, ok := strings.CutPrefix(note, gedcom551EventRolePrefix); ok {
			if role, found := lookupLabelIndex(conv.GEDCOMIndex.ParticipantRoles, roleLabel); found {
				participant.Role = role
				participant.Notes = slices.DeleteFunc(participant.Notes, func(value string) bool { return value == note || value == "GEDCOM RELA: "+gedcom551NeutralRelation })

				return
			}
		}
	}
}

func appendAssociationParticipant(event *Event, participant Participant) {
	for i, existing := range event.Participants {
		if existing.Person == participant.Person && existing.Role == participant.Role {
			for _, note := range participant.Notes {
				if !slices.Contains(event.Participants[i].Notes, note) {
					event.Participants[i].Notes = append(event.Participants[i].Notes, note)
				}
			}

			return
		}
	}
	event.Participants = append(event.Participants, participant)
}

func associationEventContext(association *GEDCOMRecord, conv *ConversionContext) []string {
	var context []string
	for _, child := range association.SubRecords {
		if child.Tag == GedcomTagNote {
			if match := gedcom551ContextPattern.FindStringSubmatch(extractNoteText(child, conv)); match != nil {
				if context != nil {
					return nil
				}
				context = match
			}
		}
	}

	return context
}
