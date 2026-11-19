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
	"reflect"
	"slices"
	"strings"
)

// validateParticipantRoleContexts warns when a participant role is used in a
// context (event or relationship) that the role's applies_to excludes.
//
// A role whose applies_to is omitted or empty applies to all contexts. A role
// that is not in the vocabulary at all is reported as an error by the
// reference pass, so it is skipped here. The check is a warning rather than an
// error so that archives written before applies_to was enforced still
// validate.
func (glx *GLXFile) validateParticipantRoleContexts(result *ValidationResult) {
	if len(glx.ParticipantRoles) == 0 {
		return
	}

	for _, id := range sortedKeys(glx.Events) {
		event := glx.Events[id]
		if event == nil {
			continue
		}
		for i := range event.Participants {
			field := fmt.Sprintf("participants[%d].role", i)
			glx.checkRoleContext(EntityTypeEvents, id, field, event.Participants[i].Role, RoleContextEvent, result)
		}
	}

	for _, id := range sortedKeys(glx.Relationships) {
		rel := glx.Relationships[id]
		if rel == nil {
			continue
		}
		for i := range rel.Participants {
			field := fmt.Sprintf("participants[%d].role", i)
			glx.checkRoleContext(EntityTypeRelationships, id, field, rel.Participants[i].Role, RoleContextRelationship, result)
		}
	}

	// An assertion participant takes the context of the assertion's subject.
	for _, id := range sortedKeys(glx.Assertions) {
		assertion := glx.Assertions[id]
		if assertion == nil || assertion.Participant == nil {
			continue
		}
		var roleContext string
		switch {
		case assertion.Subject.Event != "":
			roleContext = RoleContextEvent
		case assertion.Subject.Relationship != "":
			roleContext = RoleContextRelationship
		default:
			continue
		}
		glx.checkRoleContext(EntityTypeAssertions, id, "participant.role", assertion.Participant.Role, roleContext, result)
	}
}

// checkRoleContext appends a warning when role is defined in the vocabulary
// with an applies_to list that does not include roleContext.
func (glx *GLXFile) checkRoleContext(entityType EntityType, entityID, field, role, roleContext string, result *ValidationResult) {
	if role == "" {
		return
	}
	entry, ok := glx.ParticipantRoles[role]
	if !ok || entry == nil || len(entry.AppliesTo) == 0 {
		return
	}
	if slices.Contains(entry.AppliesTo, roleContext) {
		return
	}

	result.Warnings = append(result.Warnings, ValidationWarning{
		SourceType: entityType,
		SourceID:   entityID,
		Field:      field,
		Message: fmt.Sprintf("%s[%s].%s: role '%s' is used on %s %s, but its applies_to is [%s]",
			entityType, entityID, field, role, articleFor(roleContext), roleContext, strings.Join(entry.AppliesTo, ", ")),
	})
}

// articleFor returns the indefinite article for a role context word.
func articleFor(word string) string {
	if word != "" && strings.ContainsRune("aeiou", rune(word[0])) {
		return "an"
	}

	return "a"
}

// participantRoleKey identifies the rows of a participant list that name one
// person in one role.
type participantRoleKey struct{ Person, Role string }

// participantRoleRows groups participant indexes by person and role, in order
// of first appearance. Rows without a person are skipped. It is the single
// definition of "the same person in the same role" shared by validation and
// the merge-persons report (#1493).
func participantRoleRows(participants []Participant) ([]participantRoleKey, map[participantRoleKey][]int) {
	var keys []participantRoleKey
	rows := map[participantRoleKey][]int{}
	for i, participant := range participants {
		if participant.Person == "" {
			continue
		}
		key := participantRoleKey{participant.Person, participant.Role}
		if _, seen := rows[key]; !seen {
			keys = append(keys, key)
		}
		rows[key] = append(rows[key], i)
	}

	return keys, rows
}

// DescribeParticipantRole renders a participant role for messages:
// "with role 'godparent'", or "with no role" when the role is empty.
func DescribeParticipantRole(role string) string {
	if role == "" {
		return "with no role"
	}

	return "with role '" + role + "'"
}

// sameParticipantRow reports whether two rows for one person and role also
// carry the same notes and properties. Rows that differ there may record
// distinct appearances (a GEDCOM ASSO per detail, say) and are left alone.
func sameParticipantRow(a, b Participant) bool {
	if len(a.Notes) != len(b.Notes) || len(a.Properties) != len(b.Properties) {
		return false
	}
	if len(a.Notes) != 0 && !reflect.DeepEqual(a.Notes, b.Notes) {
		return false
	}

	return len(a.Properties) == 0 || reflect.DeepEqual(a.Properties, b.Properties)
}

// validateDuplicateParticipants warns when an event or relationship lists one
// person more than once in the same role with the same notes and properties.
// Such rows usually come from merging two records of one person who were both
// read into the same event (#1493).
func (glx *GLXFile) validateDuplicateParticipants(result *ValidationResult) {
	check := func(entityType EntityType, id string, participants []Participant) {
		keys, rows := participantRoleRows(participants)
		for _, key := range keys {
			indexes := rows[key]
			for n, i := range indexes {
				for _, j := range indexes[:n] {
					if !sameParticipantRow(participants[j], participants[i]) {
						continue
					}
					result.Warnings = append(result.Warnings, ValidationWarning{
						SourceType: entityType,
						SourceID:   id,
						Field:      fmt.Sprintf("participants[%d]", i),
						Message: fmt.Sprintf("%s[%s].participants[%d]: person '%s' already appears %s at participants[%d] with the same notes and properties; fold the rows into one if they record the same appearance",
							entityType, id, i, key.Person, DescribeParticipantRole(key.Role), j),
					})

					break
				}
			}
		}
	}
	for _, id := range sortedKeys(glx.Events) {
		if event := glx.Events[id]; event != nil {
			check(EntityTypeEvents, id, event.Participants)
		}
	}
	for _, id := range sortedKeys(glx.Relationships) {
		if rel := glx.Relationships[id]; rel != nil {
			check(EntityTypeRelationships, id, rel.Participants)
		}
	}
}
