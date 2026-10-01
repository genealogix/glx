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
	"sort"
	"strings"
)

// householdRoles are the participant roles that place a person in a census
// household. An event with no principal is written under the participants who
// hold one of these roles, since each of them is a subject of the record.
var householdRoles = map[string]bool{
	ParticipantRoleHouseholdHead: true,
	ParticipantRoleBoarder:       true,
}

// gedcomAssociationRoles maps GLX participant roles to GEDCOM 7.0 ROLE
// enumeration values for ASSO structures. A role absent here exports as
// ROLE OTHER with a PHRASE naming it. A `gedcom:` value on the role's
// participant-roles vocabulary entry overrides this table (#524).
var gedcomAssociationRoles = map[string]string{
	ParticipantRoleWitness:   GedcomRoleWitn,
	ParticipantRoleGodparent: GedcomRoleGodp,
	ParticipantRoleOfficiant: GedcomRoleOfficiator,
	ParticipantRoleChild:     GedcomRoleChil,
	ParticipantRoleParent:    GedcomRoleParent,
	ParticipantRoleSpouse:    GedcomRoleSpou,
	ParticipantRoleGroom:     GedcomRoleHusb,
	ParticipantRoleBride:     GedcomRoleWife,
}

// gedcomRoleEnumeration is the set of GEDCOM 7.0 ROLE enumeration values. A
// role outside it exports as OTHER, with a PHRASE carrying the role's label.
var gedcomRoleEnumeration = map[string]bool{
	GedcomRoleChil: true, GedcomRoleClergy: true, GedcomRoleFath: true, GedcomRoleFriend: true,
	GedcomRoleGodp: true, GedcomRoleHusb: true, GedcomRoleMoth: true, GedcomRoleMultiple: true,
	GedcomRoleNghbr: true, GedcomRoleOfficiator: true, GedcomRoleOther: true, GedcomRoleParent: true,
	GedcomRoleSpou: true, GedcomRoleWife: true, GedcomRoleWitn: true,
}

// eventHostIDs returns the person IDs whose INDI record carries an event: its
// principals (role principal, subject or unset). An event with no principal is
// carried by its household-role participants (a census entered with only a
// household_head). The result is deduplicated and in participant order; it is
// empty when no participant can host the event.
func eventHostIDs(event *Event) []string {
	var principals, household []string
	for _, p := range event.Participants {
		if p.Person == "" {
			continue
		}
		switch {
		case isSubjectRole(p.Role):
			if !slices.Contains(principals, p.Person) {
				principals = append(principals, p.Person)
			}
		case householdRoles[p.Role]:
			if !slices.Contains(household, p.Person) {
				household = append(household, p.Person)
			}
		}
	}

	if len(principals) > 0 {
		return principals
	}

	return household
}

// genericEventTypeRecord returns the TYPE subrecord of a generic EVEN for an
// event whose type has no GEDCOM tag of its own (#1320): the type's vocabulary
// label, followed by ": <subtype>" when the event also has an event subtype.
// Returns nil for an event whose type has its own tag.
func genericEventTypeRecord(event *Event, expCtx *ExportContext) *GEDCOMRecord {
	if !expCtx.ExportIndex.GenericEventTypes[event.Type] {
		return nil
	}

	value := vocabularyLabel(expCtx.GLX.EventTypes, event.Type)
	if subtype := eventSubtype(event, expCtx); subtype != "" {
		value += genericEventSubtypeSeparator + subtype
	}

	return &GEDCOMRecord{Tag: GedcomTagType, Value: value}
}

// genericEventSubtypeSeparator joins the event type label and the event
// subtype in a generic EVEN's TYPE. Import splits on it (see
// resolveGenericEventType).
const genericEventSubtypeSeparator = ": "

// eventSubtype returns the event's value for the event property that maps to
// GEDCOM TYPE (event_subtype in the standard vocabulary), or "".
func eventSubtype(event *Event, expCtx *ExportContext) string {
	for key, tag := range expCtx.ExportIndex.EventProperties {
		if tag != GedcomTagType {
			continue
		}
		if s, ok := event.Properties[key].(string); ok && s != "" {
			return s
		}
	}

	return ""
}

// vocabularyLabel returns the label of a vocabulary entry, falling back to the
// key in title case when the entry is missing or unlabelled.
func vocabularyLabel(vocabulary map[string]*VocabularyEntry, key string) string {
	if entry, ok := vocabulary[key]; ok && entry != nil && entry.Label != "" {
		return entry.Label
	}

	words := strings.Split(key, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}

	return strings.Join(words, " ")
}

// exportEventAssociations builds an ASSO subrecord for each event participant
// that skip does not exclude (#1321): the people who took part in the event
// without being the subject of the record it is written under, such as a
// witness, godparent or informant.
//
// GEDCOM 7.0 writes ASSO @Ix@ with a ROLE from its enumeration, or ROLE OTHER
// with a PHRASE naming the role. GEDCOM 5.5.1 writes ASSO @Ix@ with a RELA
// naming the role. Participant notes become NOTE under the ASSO. GEDCOM 5.5.1
// defines ASSO only at INDI level; the event-level ASSO follows GEDCOM 5.5.5
// and 7.0 practice, and is what glx import reads back.
func exportEventAssociations(event *Event, skip func(Participant) bool, expCtx *ExportContext) []*GEDCOMRecord {
	var records []*GEDCOMRecord
	for _, p := range event.Participants {
		if p.Person == "" || skip(p) {
			continue
		}
		// A participant who is not an exported person (a dangling reference,
		// which validation reports) has no XREF to point at.
		xref := expCtx.PersonXRefMap[p.Person]
		if xref == "" {
			continue
		}

		asso := &GEDCOMRecord{Tag: GedcomTagAsso, Value: xref}
		if expCtx.Version == GEDCOM70 {
			role := &GEDCOMRecord{Tag: GedcomTagRole, Value: gedcomAssociationRole(p, expCtx)}
			if role.Value == GedcomRoleOther {
				role.SubRecords = []*GEDCOMRecord{{
					Tag:   GedcomTagPhrase,
					Value: vocabularyLabel(expCtx.GLX.ParticipantRoles, p.Role),
				}}
			}
			asso.SubRecords = append(asso.SubRecords, role)
		} else {
			asso.SubRecords = append(asso.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagRela,
				Value: vocabularyLabel(expCtx.GLX.ParticipantRoles, p.Role),
			})
		}
		for _, note := range p.Notes {
			asso.SubRecords = append(asso.SubRecords, &GEDCOMRecord{Tag: GedcomTagNote, Value: note})
		}

		records = append(records, asso)
	}

	return records
}

// gedcomAssociationRole maps a participant's role to a GEDCOM 7.0 ROLE value:
// the role vocabulary's `gedcom:` value when it names an enumeration value,
// else the built-in table (parent and spouse narrowed to FATH/MOTH and
// HUSB/WIFE by recorded sex), else OTHER.
func gedcomAssociationRole(p Participant, expCtx *ExportContext) string {
	if entry, ok := expCtx.GLX.ParticipantRoles[p.Role]; ok && entry != nil {
		if value := strings.ToUpper(strings.TrimSpace(entry.GEDCOM)); gedcomRoleEnumeration[value] {
			return value
		}
	}

	role, ok := gedcomAssociationRoles[p.Role]
	if !ok {
		return GedcomRoleOther
	}

	sex := getPersonSex(p.Person, expCtx)
	switch {
	case role == GedcomRoleParent && sex == SexMale:
		return GedcomRoleFath
	case role == GedcomRoleParent && sex == SexFemale:
		return GedcomRoleMoth
	case role == GedcomRoleSpou && sex == SexMale:
		return GedcomRoleHusb
	case role == GedcomRoleSpou && sex == SexFemale:
		return GedcomRoleWife
	}

	return role
}

// isFamilyEventMemberRole reports whether a participant of a FAM-level event
// is one of the couple (or a principal) rather than an associate who gets an
// ASSO.
func isFamilyEventMemberRole(role string) bool {
	switch role {
	case ParticipantRoleSpouse, ParticipantRoleBride, ParticipantRoleGroom:
		return true
	}

	return isSubjectRole(role)
}

// markFamilyEventExported records that an event was written under a FAM.
func (expCtx *ExportContext) markFamilyEventExported(eventID string) {
	if expCtx.familyEventsExported == nil {
		expCtx.familyEventsExported = make(map[string]bool)
	}
	expCtx.familyEventsExported[eventID] = true
}

// reportUnexportedEvents adds an export warning for every event that no INDI
// or FAM record carries, so nothing is dropped silently (#1320, #1321).
func reportUnexportedEvents(expCtx *ExportContext) {
	for _, eventID := range sortedKeys(expCtx.GLX.Events) {
		event := expCtx.GLX.Events[eventID]
		if event == nil || expCtx.familyEventsExported[eventID] {
			continue
		}

		_, mapped := expCtx.ExportIndex.EventTypes[event.Type]
		hosted := false
		for _, personID := range eventHostIDs(event) {
			if expCtx.GLX.Persons[personID] != nil {
				hosted = true

				break
			}
		}
		if mapped && hosted {
			continue
		}

		var roles []string
		for _, p := range event.Participants {
			if p.Person != "" && !slices.Contains(roles, p.Role) {
				roles = append(roles, p.Role)
			}
		}
		sort.Strings(roles)

		var reason string
		switch {
		case !mapped:
			reason = fmt.Sprintf("event type %q is not in the event_types vocabulary", event.Type)
		case len(roles) == 0:
			reason = "event has no person participants"
		default:
			reason = fmt.Sprintf("event has no principal or household participant (roles: %s) "+
				"and is not a family event of an exported FAM", strings.Join(roles, ", "))
		}
		expCtx.addExportWarning(EntityTypeEvents, eventID, reason+"; event not exported")
	}
}
