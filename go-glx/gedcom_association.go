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
	"regexp"
	"slices"
	"strings"
)

var gedcom551ContextPattern = regexp.MustCompile(`^Event: (.+) \(([^;()]+); ([^)]+)\)(.*)$`)

// importIndividualAssociations waits until both INDI and FAM events exist.
func importIndividualAssociations(records []*GEDCOMRecord, conv *ConversionContext) {
	for _, individual := range records {
		hostID := conv.PersonIDMap[individual.XRef]
		for _, association := range individual.SubRecords {
			if association.Tag != GedcomTagAsso {
				continue
			}
			participant := convertASSOToParticipant(association, conv)
			if participant == nil {
				continue
			}
			event := matchAssociationEvent(hostID, association, conv)
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

func matchAssociationEvent(hostID string, association *GEDCOMRecord, conv *ConversionContext) *Event {
	context := associationEventContext(association, conv)
	if context == nil {
		return nil
	}
	label, tag := context[1], context[2]
	attributes := make(map[string]string)
	for part := range strings.SplitSeq(context[4], "; ") {
		if key, value, ok := strings.Cut(part, ": "); ok {
			attributes[key] = value
		}
	}
	var match *Event
	for _, event := range conv.GLX.Events {
		if event == nil || !associationEventHasHost(event, hostID) {
			continue
		}
		if tag == GedcomTagEven {
			if vocabularyLabel(conv.GLX.EventTypes, event.Type) != label {
				continue
			}
		} else if conv.GEDCOMIndex.EventTypes[tag] != event.Type {
			continue
		}
		if string(event.Date) != string(parseGEDCOMDate(attributes["date"])) {
			continue
		}
		if place := attributes["place"]; place != "" {
			export := &ExportContext{GLX: conv.GLX, PlaceStrings: make(map[string]string)}
			if resolvePlaceString(event.PlaceID, export) != place {
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

func associationEventHasHost(event *Event, hostID string) bool {
	for _, participant := range event.Participants {
		if participant.Person == hostID && isFamilyEventMemberRole(participant.Role) {
			return true
		}
	}

	return slices.Contains(eventHostIDs(event), hostID)
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
