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
	"unicode/utf8"
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
// GEDCOM 7.0 writes ASSO @Ix@ with ROLE (and PHRASE for OTHER). GEDCOM
// 5.5.1 has no event-level ASSO; queueGEDCOM551EventAssociations puts its
// associations on INDI instead, with descriptive event context in NOTE.
func exportEventAssociations(event *Event, skip func(Participant) bool, expCtx *ExportContext) []*GEDCOMRecord {
	if expCtx.Version == GEDCOM551 {
		return nil
	}

	return eventAssociationRecords(event, skip, expCtx)
}

func eventAssociationRecords(event *Event, skip func(Participant) bool, expCtx *ExportContext) []*GEDCOMRecord {
	var records []*GEDCOMRecord
	for _, p := range event.Participants {
		if asso := eventAssociationRecord(p, skip, false, expCtx); asso != nil {
			records = append(records, asso)
		}
	}

	return records
}

// eventAssociationRecord builds the ASSO subrecord for one event participant,
// or returns nil when skip excludes it or it has no exported XREF. In 5.5.1,
// neutral writes RELA Participant with the role in a NOTE, instead of a RELA
// that would state the role as a relation to the INDI carrying the ASSO.
func eventAssociationRecord(p Participant, skip func(Participant) bool, neutral bool, expCtx *ExportContext) *GEDCOMRecord {
	if p.Person == "" || skip(p) {
		return nil
	}
	// A participant who is not an exported person (a dangling reference,
	// which validation reports) has no XREF to point at.
	xref := expCtx.PersonXRefMap[p.Person]
	if xref == "" {
		return nil
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
		label := vocabularyLabel(expCtx.GLX.ParticipantRoles, p.Role)
		// RELA is one line of at most 25 characters in 5.5.1, with no
		// CONT/CONC children. Preserve longer or multiline labels in NOTE.
		relation := label
		if neutral || strings.TrimSpace(relation) == "" || utf8.RuneCountInString(relation) > 25 || strings.ContainsAny(relation, "\r\n") {
			relation = gedcom551NeutralRelation
		}
		asso.SubRecords = append(asso.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagRela,
			Value: relation,
		})
		if relation != label {
			noteLabel := strings.ReplaceAll(strings.ReplaceAll(label, "\r\n", "\n"), "\r", "\n")
			asso.SubRecords = append(asso.SubRecords, &GEDCOMRecord{Tag: GedcomTagNote, Value: gedcom551EventRolePrefix + noteLabel})
		}
	}
	for _, note := range p.Notes {
		asso.SubRecords = append(asso.SubRecords, &GEDCOMRecord{Tag: GedcomTagNote, Value: note})
	}

	return asso
}

// gedcom551NeutralRelation is the RELA a 5.5.1 ASSO carries when the GLX role
// cannot be written as a relation to the INDI: a label RELA cannot hold, or a
// kinship role whose relative among the event's subjects is unknown. The role
// itself goes in a NOTE starting with gedcom551EventRolePrefix, which import
// reads back (#1372).
const (
	gedcom551NeutralRelation = "Participant"
	gedcom551EventRolePrefix = "Event role: "
)

// kinshipParticipantRoles are participant roles that state a relation to one
// particular person rather than to the event: the bride's father at a
// marriage is the bride's parent, not the groom's (#1372). A 5.5.1 RELA states
// the relation to the INDI that carries the ASSO, so these roles may only be
// written under the subject they relate to.
var kinshipParticipantRoles = map[string]bool{
	ParticipantRoleParent:         true,
	ParticipantRoleChild:          true,
	ParticipantRoleSpouse:         true,
	ParticipantRoleSibling:        true,
	ParticipantRoleGodparent:      true,
	ParticipantRoleGodchild:       true,
	ParticipantRoleGuardian:       true,
	ParticipantRoleWard:           true,
	ParticipantRoleAdoptiveParent: true,
	ParticipantRoleAdoptedChild:   true,
	ParticipantRoleFosterParent:   true,
	ParticipantRoleFosterChild:    true,
	ParticipantRoleStepParent:     true,
	ParticipantRoleStepChild:      true,
}

// gedcom551AssociationHosts returns the hosts whose INDI records receive a
// participant's 5.5.1 ASSO, and whether its role must be written neutrally.
// A kinship role on an event with several subjects relates to only one of
// them: the ASSO goes to the subjects a relationship links the participant
// to, and when no relationship says which, to every subject with a neutral
// RELA.
func gedcom551AssociationHosts(p Participant, hostIDs []string, expCtx *ExportContext) ([]string, bool) {
	if len(hostIDs) < 2 || !kinshipParticipantRoles[p.Role] {
		return hostIDs, false
	}

	var related []string
	for _, hostID := range hostIDs {
		if expCtx.personsRelated(p.Person, hostID) {
			related = append(related, hostID)
		}
	}
	if len(related) == 0 {
		return hostIDs, true
	}

	return related, false
}

// personsRelated reports whether some relationship has both persons as
// participants.
func (expCtx *ExportContext) personsRelated(a, b string) bool {
	if expCtx.relatedPersons == nil {
		expCtx.relatedPersons = make(map[[2]string]bool)
		for _, rel := range expCtx.GLX.Relationships {
			if rel == nil {
				continue
			}
			for _, x := range rel.Participants {
				for _, y := range rel.Participants {
					if x.Person != "" && y.Person != "" && x.Person != y.Person {
						expCtx.relatedPersons[[2]string{x.Person, y.Person}] = true
					}
				}
			}
		}
	}

	return expCtx.relatedPersons[[2]string{a, b}]
}

// queueGEDCOM551EventAssociations projects event participants onto the event
// hosts' INDI records. hostIDs are all of the event's subjects; writeIDs are
// the hosts this call writes for (the person being exported, for an INDI
// event). Notes identify the event without inventing a standard event link
// that 5.5.1 cannot represent. Repeated exports of the same event retain
// distinct participant notes and collapse only identical structures.
func queueGEDCOM551EventAssociations(eventID string, event *Event, gedcomTag string, hostIDs, writeIDs []string,
	skip func(Participant) bool, expCtx *ExportContext,
) {
	if expCtx.Version != GEDCOM551 {
		return
	}

	context := gedcom551EventContext(eventID, event, gedcomTag, hostIDs, expCtx)
	for _, p := range event.Participants {
		targets, neutral := gedcom551AssociationHosts(p, hostIDs, expCtx)
		association := eventAssociationRecord(p, skip, neutral, expCtx)
		if association == nil {
			continue
		}
		association.SubRecords = append(association.SubRecords, &GEDCOMRecord{Tag: GedcomTagNote, Value: context})
		key := eventID + "\x00" + string(serializeGEDCOMRecords([]*GEDCOMRecord{association}))
		for _, personID := range targets {
			if !slices.Contains(writeIDs, personID) {
				continue
			}
			xref := expCtx.PersonXRefMap[personID]
			if xref == "" || xref == association.Value {
				continue
			}
			if expCtx.personAssociations551 == nil {
				expCtx.personAssociations551 = make(map[string]map[string]*GEDCOMRecord)
			}
			if expCtx.personAssociations551[xref] == nil {
				expCtx.personAssociations551[xref] = make(map[string]*GEDCOMRecord)
			}
			expCtx.personAssociations551[xref][key] = association
		}
	}
}

func gedcom551EventContext(eventID string, event *Event, gedcomTag string, hostIDs []string, expCtx *ExportContext) string {
	context := "Event: " + vocabularyLabel(expCtx.GLX.EventTypes, event.Type) + " (" + gedcomTag + "; " + eventID + ")"
	if event.Date != "" {
		context += "; date: " + string(event.Date)
	}
	if place := expCtx.PlaceStrings[event.PlaceID]; place != "" {
		context += "; place: " + place
	}
	var names []string
	for _, personID := range hostIDs {
		if person := expCtx.GLX.Persons[personID]; person != nil {
			names = append(names, PersonDisplayName(person))
		}
	}
	if len(names) > 0 {
		context += "; subjects: " + strings.Join(names, ", ")
	}

	return context
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
