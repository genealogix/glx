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
