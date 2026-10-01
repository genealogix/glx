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

import "strings"

// Participation describes what an event participant's role says about the
// person: whether the event is the person's own record, and whether the role
// puts the person at the event's place.
//
// The two questions are independent. A witness is present at someone else's
// marriage; a grantor selling land from another state is a party to the deed
// without being there; a principal is both. A role that is neither (a legatee
// in a father-in-law's estate, an adjoining owner named in a deed) only
// mentions the person, which Mentioned reports.
type Participation struct {
	// OwnRecord is true when the event is a record of this person: their
	// own birth, marriage, probate, or a census in which they are a member of
	// the enumerated household. Witness, godparent, informant, legatee and the
	// like appear in someone else's record.
	OwnRecord bool

	// Present is true when the role implies the person was at the event's
	// place on its date, so the place is evidence of where they were.
	Present bool
}

// Mentioned reports whether the person is only named in the record: it is
// not their own record and does not place them there.
func (p Participation) Mentioned() bool {
	return !p.OwnRecord && !p.Present
}

// ownRecordRoles are roles that make an event the participant's own record
// whatever the event type. The empty role is the participant default
// (principal).
var ownRecordRoles = map[string]bool{
	"":                          true,
	ParticipantRolePrincipal:    true,
	ParticipantRoleSubject:      true,
	ParticipantRoleGroom:        true,
	ParticipantRoleBride:        true,
	ParticipantRoleGodchild:     true, // the person baptized
	ParticipantRoleAdoptedChild: true,
	"decedent":                  true,
	"deceased":                  true,
	"testator":                  true, // the will is theirs, signed where they were
	"testatrix":                 true,
}

// coupleEventTypes are event types whose spouse participants are the couple
// the record is about, so `spouse` (GEDCOM import's role for HUSB and WIFE) is
// an own-record role there.
var coupleEventTypes = map[string]bool{
	EventTypeMarriage:           true,
	EventTypeDivorce:            true,
	EventTypeDivorceFiled:       true,
	EventTypeEngagement:         true,
	EventTypeAnnulment:          true,
	EventTypeMarriageBanns:      true,
	EventTypeMarriageContract:   true,
	EventTypeMarriageLicense:    true,
	EventTypeMarriageSettlement: true,
	EventTypeLegalSeparation:    true,
}

// coupleRoles are the roles that name one of the couple in a couple event.
var coupleRoles = map[string]bool{
	ParticipantRoleSpouse: true,
	"husband":             true,
	"wife":                true,
}

// nonHouseholdCensusRoles are census participant roles that do not make the
// person a member of the enumerated household. Every other role on a census
// event (subject, household_head, head, wife, son, boarder, servant, a
// "counted but not named" member role, ...) does, so the list of household
// roles stays open for archives and later vocabulary additions.
var nonHouseholdCensusRoles = map[string]bool{
	ParticipantRoleWitness: true,
	"enumerator":           true,
	"neighbor":             true,
	"neighbour":            true,
	"mentioned":            true,
}

// nonPresenceRoles are roles that name a person in a record without implying
// they were at its place: parties to a conveyance (who may sell land in a
// county they left years before), heirs and legatees of an estate, and people
// merely mentioned. An archive overrides this list for its own roles with
// `implies_presence` on the participant role's vocabulary entry.
var nonPresenceRoles = map[string]bool{
	"grantor":         true,
	"grantee":         true,
	"adjoining_owner": true,
	"adjoiner":        true,
	"legatee":         true,
	"devisee":         true,
	"beneficiary":     true,
	"heir":            true,
	"executor":        true,
	"executrix":       true,
	"administrator":   true,
	"administratrix":  true,
	"creditor":        true,
	"debtor":          true,
	"mentioned":       true,
	"neighbor":        true,
	"neighbour":       true,
}

// ClassifyParticipation classifies a participant's role in an event of the
// given type. roles is the archive's participant role vocabulary (may be
// nil); an entry's `implies_presence` overrides the built-in presence default
// for that role.
//
// Rules, in order:
//   - principal, subject, the empty role, bride, groom, godchild,
//     adopted_child, decedent and testator: own record, present
//   - spouse, husband or wife on a marriage-type event: own record, present
//   - any role on a census event except witness, enumerator, neighbor and
//     mentioned: own record (a household member), present
//   - grantor, grantee, legatee, heir, beneficiary, executor and the other
//     nonPresenceRoles: mentioned only
//   - everything else (witness, officiant, informant, godparent, parent,
//     custom roles): present at someone else's record
func ClassifyParticipation(eventType, role string, roles map[string]*VocabularyEntry) Participation {
	declared := strings.TrimSpace(role)
	role = strings.ToLower(declared)
	eventType = strings.ToLower(strings.TrimSpace(eventType))

	var p Participation
	switch {
	case ownRecordRoles[role]:
		p = Participation{OwnRecord: true, Present: true}
	case coupleEventTypes[eventType] && coupleRoles[role]:
		p = Participation{OwnRecord: true, Present: true}
	case eventType == EventTypeCensus && !nonHouseholdCensusRoles[role]:
		p = Participation{OwnRecord: true, Present: true}
	case nonPresenceRoles[role]:
		p = Participation{}
	default:
		p = Participation{Present: true}
	}

	entry := roles[declared]
	if entry == nil {
		entry = roles[role]
	}
	if entry != nil && entry.ImpliesPresence != nil {
		p.Present = *entry.ImpliesPresence
	}

	return p
}

// EventParticipation classifies personID's participation in event, returning
// ok=false when the person is not a participant. A person listed more than
// once (an unusual but valid shape) gets the union of their roles'
// classifications, and the role returned is the most significant of them.
func EventParticipation(event *Event, personID string, roles map[string]*VocabularyEntry) (Participation, string, bool) {
	if event == nil {
		return Participation{}, "", false
	}

	var (
		result   Participation
		bestRole string
		bestRank = -1
	)
	for _, participant := range event.Participants {
		if participant.Person != personID {
			continue
		}
		c := ClassifyParticipation(event.Type, participant.Role, roles)
		if rank := c.rank(); rank > bestRank {
			bestRank = rank
			bestRole = participant.Role
		}
		result.OwnRecord = result.OwnRecord || c.OwnRecord
		result.Present = result.Present || c.Present
	}

	return result, bestRole, bestRank >= 0
}

// rank orders classifications so the most significant of a person's roles
// in one event is the one reported: own record, then present, then mentioned.
func (p Participation) rank() int {
	rank := 0
	if p.Present {
		rank++
	}
	if p.OwnRecord {
		rank += 2
	}

	return rank
}
