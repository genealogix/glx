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

// fieldParticipants is the `participants` field name that validation
// findings point at.
const fieldParticipants = "participants"

// relationshipSignature returns a key shared by relationships of the same type
// whose participants are the same people in the same roles, regardless of the
// order the participants are listed in. Type and role comparison ignores case.
// It returns "" for a relationship with no identified participant.
func relationshipSignature(rel *Relationship) string {
	parts := make([]string, 0, len(rel.Participants))
	for _, p := range rel.Participants {
		if p.Person == "" {
			continue
		}
		parts = append(parts, p.Person+"\x00"+strings.ToLower(p.Role))
	}
	if len(parts) == 0 {
		return ""
	}
	sort.Strings(parts)
	parts = slices.Compact(parts)

	return strings.ToLower(rel.Type) + "\x01" + strings.Join(parts, "\x01")
}

// relationshipsDuplicate reports whether a and b record the same relationship:
// same type and the same participants in the same roles. Two relationships
// whose start events are both set and differ are not duplicates (a couple who
// married, divorced, and remarried).
func relationshipsDuplicate(a, b *Relationship) bool {
	if a == nil || b == nil {
		return false
	}
	if a.StartEvent != "" && b.StartEvent != "" && a.StartEvent != b.StartEvent {
		return false
	}
	sig := relationshipSignature(a)

	return sig != "" && sig == relationshipSignature(b)
}

// FindDuplicateRelationship returns the ID of an existing relationship in the
// archive that duplicates rel (same type, same participants in the same roles;
// see DuplicateRelationships), or "" when there is none. The relationship
// stored under excludeID is ignored, so a caller replacing an entity in place
// does not find the entity itself. When several match, the lowest ID wins.
func FindDuplicateRelationship(archive *GLXFile, rel *Relationship, excludeID string) string {
	if archive == nil || rel == nil {
		return ""
	}
	for _, id := range sortedKeys(archive.Relationships) {
		if id == excludeID {
			continue
		}
		if relationshipsDuplicate(archive.Relationships[id], rel) {
			return id
		}
	}

	return ""
}

// DuplicateRelationships maps the ID of each relationship that repeats an
// earlier one to the ID of the earliest relationship it repeats, earliest by
// ID order. Two relationships repeat each other when they have the same type
// and the same participants in the same roles, and their start events do not
// conflict (#1506).
func DuplicateRelationships(archive *GLXFile) map[string]string {
	if archive == nil {
		return nil
	}
	groups := make(map[string][]string)
	ids := sortedKeys(archive.Relationships)
	for _, id := range ids {
		rel := archive.Relationships[id]
		if rel == nil {
			continue
		}
		if sig := relationshipSignature(rel); sig != "" {
			groups[sig] = append(groups[sig], id)
		}
	}

	dups := make(map[string]string)
	for _, group := range groups {
		for i := 1; i < len(group); i++ {
			for _, earlier := range group[:i] {
				if relationshipsDuplicate(archive.Relationships[earlier], archive.Relationships[group[i]]) {
					dups[group[i]] = earlier

					break
				}
			}
		}
	}

	return dups
}

// validateDuplicateRelationships warns on each relationship that repeats an
// earlier one, naming the one it repeats (#1506). A repeated parent-child link
// otherwise goes unnoticed and draws the parent twice in `glx ancestors`.
func (glx *GLXFile) validateDuplicateRelationships(result *ValidationResult) {
	for id, earlier := range DuplicateRelationships(glx) {
		rel := glx.Relationships[id]
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: EntityTypeRelationships,
			SourceID:   id,
			Field:      fieldParticipants,
			Message: fmt.Sprintf("%s[%s]: duplicates %s[%s] (same type %q and the same participants in the same roles); "+
				"merge them or remove one",
				EntityTypeRelationships, id, EntityTypeRelationships, earlier, rel.Type),
		})
	}
}
