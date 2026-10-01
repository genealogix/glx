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
	"sort"
	"strings"
)

// RelationshipStanding is how far the archive's assertions support a
// relationship (or one participant's place in it). Display commands use it to
// mark hypotheses and to leave out links the researcher has ruled out.
type RelationshipStanding string

// Relationship standings, from established to ruled out.
const (
	// RelationshipStandingAccepted means no assertion casts doubt on the
	// relationship. A relationship with no assertions at all is accepted:
	// the archive records it as fact.
	RelationshipStandingAccepted RelationshipStanding = "accepted"
	// RelationshipStandingHypothetical means a surviving assertion records
	// it as a hypothesis (low/tentative/disputed confidence, or a
	// speculative/unresearched/disputed status) and none marks it proven.
	RelationshipStandingHypothetical RelationshipStanding = "hypothetical"
	// RelationshipStandingDisproven means every assertion about the
	// relationship has status `disproven`: it is a recorded, rejected
	// alternative, not a relationship.
	RelationshipStandingDisproven RelationshipStanding = "disproven"
)

// Assertion status and confidence tokens the standing rules read. Statuses are
// free text in the spec; these are the values the spec documents.
const (
	standingStatusProven       = "proven"
	standingStatusDisproven    = "disproven"
	standingStatusSpeculative  = "speculative"
	standingStatusUnresearched = "unresearched"
	standingStatusDisputed     = "disputed"
)

// hypotheticalAssertionConfidence holds the confidence values that mark an
// assertion as a hypothesis. "low" is the standard vocabulary's weakest level;
// "tentative" and "disputed" are legacy confidence tokens that carried the same
// meaning before disputes moved to `status`.
var hypotheticalAssertionConfidence = map[string]bool{
	"low":       true,
	"tentative": true,
	"disputed":  true,
}

// hypotheticalAssertionStatus holds the status values that mark an assertion as
// a hypothesis rather than a conclusion.
var hypotheticalAssertionStatus = map[string]bool{
	standingStatusSpeculative:  true,
	standingStatusUnresearched: true,
	standingStatusDisputed:     true,
}

// AssertionIsDisproven reports whether the assertion's status is `disproven`.
func AssertionIsDisproven(a *Assertion) bool {
	return a != nil && strings.EqualFold(strings.TrimSpace(a.Status), standingStatusDisproven)
}

// AssertionIsHypothetical reports whether an assertion records a hypothesis:
// its confidence is low (or a legacy tentative/disputed level), or its status
// is speculative, unresearched, or disputed. A `proven` status overrides a
// weak confidence, and a disproven assertion is not a hypothesis (it is ruled
// out; see AssertionIsDisproven).
func AssertionIsHypothetical(a *Assertion) bool {
	if a == nil || AssertionIsDisproven(a) {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(a.Status))
	if status == standingStatusProven {
		return false
	}

	return hypotheticalAssertionStatus[status] ||
		hypotheticalAssertionConfidence[strings.ToLower(strings.TrimSpace(a.Confidence))]
}

// StandingFromAssertions classifies a relationship from the assertions about it:
//
//   - no assertions: accepted (the archive records the link as fact);
//   - every assertion disproven: disproven;
//   - otherwise only the surviving (non-disproven) assertions count: a
//     `proven` one makes the link accepted; failing that, any hypothetical
//     one (see AssertionIsHypothetical) makes it hypothetical; else accepted.
func StandingFromAssertions(assertions []*Assertion) RelationshipStanding {
	var surviving []*Assertion
	for _, a := range assertions {
		if a != nil && !AssertionIsDisproven(a) {
			surviving = append(surviving, a)
		}
	}
	if len(surviving) == 0 {
		if len(assertions) > 0 {
			return RelationshipStandingDisproven
		}

		return RelationshipStandingAccepted
	}

	hypothetical := false
	for _, a := range surviving {
		if strings.EqualFold(strings.TrimSpace(a.Status), standingStatusProven) {
			return RelationshipStandingAccepted
		}
		if AssertionIsHypothetical(a) {
			hypothetical = true
		}
	}
	if hypothetical {
		return RelationshipStandingHypothetical
	}

	return RelationshipStandingAccepted
}

// RelationshipStandingIndex answers "how well supported is this relationship?"
// for every relationship in an archive. Build it once per command with
// NewRelationshipStandingIndex; lookups are map reads.
//
// Only assertions about the relationship's existence count: an assertion whose
// subject is the relationship and that has no `property` (an existential
// assertion), or that names a `participant` (it asserts that person's place in
// the relationship). An assertion about one of the relationship's properties
// (say a disproven start date) says nothing about whether the people were
// related, so it is ignored.
type RelationshipStandingIndex struct {
	whole       map[string][]*Assertion            // relID -> existential assertions
	participant map[string]map[string][]*Assertion // relID -> person ID -> participant assertions
}

// NewRelationshipStandingIndex indexes the archive's relationship assertions.
func NewRelationshipStandingIndex(archive *GLXFile) *RelationshipStandingIndex {
	idx := &RelationshipStandingIndex{
		whole:       map[string][]*Assertion{},
		participant: map[string]map[string][]*Assertion{},
	}
	if archive == nil {
		return idx
	}

	ids := make([]string, 0, len(archive.Assertions))
	for id := range archive.Assertions {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		a := archive.Assertions[id]
		if a == nil || a.Subject.Relationship == "" || a.Property != "" {
			continue
		}
		relID := a.Subject.Relationship
		if a.Participant != nil && a.Participant.Person != "" {
			byPerson := idx.participant[relID]
			if byPerson == nil {
				byPerson = map[string][]*Assertion{}
				idx.participant[relID] = byPerson
			}
			byPerson[a.Participant.Person] = append(byPerson[a.Participant.Person], a)

			continue
		}
		idx.whole[relID] = append(idx.whole[relID], a)
	}

	return idx
}

// Relationship returns the standing of a relationship as a whole: its
// existential assertions plus every participant assertion.
func (idx *RelationshipStandingIndex) Relationship(relID string) RelationshipStanding {
	if idx == nil {
		return RelationshipStandingAccepted
	}
	assertions := append([]*Assertion(nil), idx.whole[relID]...)
	byPerson := idx.participant[relID]
	persons := make([]string, 0, len(byPerson))
	for p := range byPerson {
		persons = append(persons, p)
	}
	sort.Strings(persons)
	for _, p := range persons {
		assertions = append(assertions, byPerson[p]...)
	}

	return StandingFromAssertions(assertions)
}

// Link returns the standing of the link between the given participants of a
// relationship: its existential assertions plus the participant assertions
// about those people. A parent disproven by a participant assertion is thereby
// dropped without dropping the other parent recorded in the same relationship.
func (idx *RelationshipStandingIndex) Link(relID string, personIDs ...string) RelationshipStanding {
	return StandingFromAssertions(idx.LinkAssertions(relID, personIDs...))
}

// LinkAssertions returns the assertions Link weighs for the given
// participants of a relationship: the relationship's existential assertions,
// then the participant assertions about each person, in assertion-ID order.
func (idx *RelationshipStandingIndex) LinkAssertions(relID string, personIDs ...string) []*Assertion {
	if idx == nil {
		return nil
	}
	assertions := append([]*Assertion(nil), idx.whole[relID]...)
	for _, p := range personIDs {
		assertions = append(assertions, idx.participant[relID][p]...)
	}

	return assertions
}

// Participant-role keys recognized on the parent and child sides of a
// parent-child relationship. The step/foster keys are matched by name so
// archives that define them (or a later standard vocabulary) work unchanged.
const (
	roleKeyStepParent   = "step_parent"
	roleKeyStepChild    = "step_child"
	roleKeyFosterParent = "foster_parent"
	roleKeyFosterChild  = "foster_child"
)

var (
	parentSideRoles = map[string]bool{
		ParticipantRoleParent:         true,
		ParticipantRoleAdoptiveParent: true,
		roleKeyStepParent:             true,
		roleKeyFosterParent:           true,
	}
	childSideRoles = map[string]bool{
		ParticipantRoleChild:        true,
		ParticipantRoleAdoptedChild: true,
		roleKeyStepChild:            true,
		roleKeyFosterChild:          true,
	}
)

// IsParentChildRelationshipType reports whether a relationship type links
// parents to children (including adoptive, foster, and step links).
func IsParentChildRelationshipType(relType string) bool {
	switch strings.ToLower(relType) {
	case RelationshipTypeParentChild, RelationshipTypeBiologicalParentChild,
		RelationshipTypeAdoptiveParentChild, RelationshipTypeFosterParentChild,
		RelationshipTypeStepParent:
		return true
	}

	return false
}

// ParentChildLink is one parent → child edge of a parent-child relationship.
type ParentChildLink struct {
	RelationshipID   string
	RelationshipType string
	ParentID         string
	ChildID          string
	// Step is true for a step-parent link: a step_parent relationship, or a
	// parent/child recorded with a step_parent/step_child role.
	Step bool
	// Standing is the link's standing (see RelationshipStandingIndex.Link).
	Standing RelationshipStanding
}

// ParentChildLinks returns every parent → child edge in the archive, in
// relationship-ID order and then participant order, with each edge's step flag
// and standing. Disproven edges are included; callers decide whether to drop
// them or list them as excluded alternatives.
func ParentChildLinks(archive *GLXFile, standings *RelationshipStandingIndex) []ParentChildLink {
	if archive == nil {
		return nil
	}
	if standings == nil {
		standings = NewRelationshipStandingIndex(archive)
	}

	relIDs := make([]string, 0, len(archive.Relationships))
	for id := range archive.Relationships {
		relIDs = append(relIDs, id)
	}
	sort.Strings(relIDs)

	var links []ParentChildLink
	for _, relID := range relIDs {
		rel := archive.Relationships[relID]
		if rel == nil || !IsParentChildRelationshipType(rel.Type) {
			continue
		}
		relIsStep := strings.EqualFold(rel.Type, RelationshipTypeStepParent)

		var parents, children []Participant
		for _, p := range rel.Participants {
			if p.Person == "" {
				continue
			}
			switch {
			case IsParentSideRole(p.Role):
				parents = append(parents, p)
			case IsChildSideRole(p.Role):
				children = append(children, p)
			}
		}

		for _, c := range children {
			for _, p := range parents {
				links = append(links, ParentChildLink{
					RelationshipID:   relID,
					RelationshipType: rel.Type,
					ParentID:         p.Person,
					ChildID:          c.Person,
					Step:             relIsStep || IsStepRole(p.Role) || IsStepRole(c.Role),
					Standing:         standings.Link(relID, p.Person, c.Person),
				})
			}
		}
	}

	return links
}

// IsParentSideRole reports whether a participant role puts the person on the
// parent side of a parent-child relationship (parent, adoptive_parent,
// step_parent, foster_parent).
func IsParentSideRole(role string) bool {
	return parentSideRoles[strings.ToLower(role)]
}

// IsChildSideRole reports whether a participant role puts the person on the
// child side of a parent-child relationship (child, adopted_child, step_child,
// foster_child).
func IsChildSideRole(role string) bool {
	return childSideRoles[strings.ToLower(role)]
}

// IsStepRole reports whether a participant role is a step role (step_parent or
// step_child).
func IsStepRole(role string) bool {
	return strings.EqualFold(role, roleKeyStepParent) || strings.EqualFold(role, roleKeyStepChild)
}
