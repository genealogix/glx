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

// eventRefsConflict reports whether two event references are both set and
// differ.
func eventRefsConflict(a, b string) bool {
	return a != "" && b != "" && a != b
}

// relationshipBoundsCompatible reports whether a and b could record the same
// relationship as far as their boundaries go: neither their start events nor
// their end events are both set and different. A couple who married twice
// (two start events), or whose two unions ended differently (two end events),
// are two relationships.
func relationshipBoundsCompatible(a, b *Relationship) bool {
	if a == nil || b == nil {
		return false
	}

	return !eventRefsConflict(a.StartEvent, b.StartEvent) && !eventRefsConflict(a.EndEvent, b.EndEvent)
}

// classifyDuplicates decides, within one group of relationships that share a
// signature (ids sorted), which relationships each one repeats.
//
// Boundary compatibility is not transitive: an undated marriage is compatible
// with each of two remarriages that are not compatible with each other. Such a
// relationship is ambiguous: it is reported as repeating every relationship it
// matches, and it is never itself the relationship another is said to repeat,
// so the remarriages are not reported as copies of it, or of each other
// through it. An unambiguous relationship repeats the earliest unambiguous
// relationship it matches that comes before it by ID; free, when set, may
// match one after it (a candidate that has no place in the order yet).
func classifyDuplicates(ids []string, rels map[string]*Relationship, free string) map[string][]string {
	matches := make(map[string][]string, len(ids))
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			if relationshipBoundsCompatible(rels[a], rels[b]) {
				matches[a] = append(matches[a], b)
				matches[b] = append(matches[b], a)
			}
		}
	}

	ambiguous := make(map[string]bool)
	for _, id := range ids {
		m := matches[id]
	scan:
		for i, p := range m {
			for _, q := range m[i+1:] {
				if !relationshipBoundsCompatible(rels[p], rels[q]) {
					ambiguous[id] = true

					break scan
				}
			}
		}
	}

	out := make(map[string][]string)
	for _, id := range ids {
		m := matches[id]
		if len(m) == 0 {
			continue
		}
		if ambiguous[id] {
			sorted := slices.Clone(m)
			sort.Strings(sorted)
			out[id] = sorted

			continue
		}
		best := ""
		for _, other := range m {
			if ambiguous[other] || (other > id && id != free) {
				continue
			}
			if best == "" || other < best {
				best = other
			}
		}
		if best != "" {
			out[id] = []string{best}
		}
	}

	return out
}

// relationshipGroups returns the archive's relationships grouped by
// signature, each group in ID order, with each signature computed once.
// Relationships the archive's assertions have disproven are left out: a
// recorded, rejected alternative is not a second copy of the relationship
// that replaced it. The relationship stored under excludeID is left out too.
func relationshipGroups(archive *GLXFile, excludeID string) map[string][]string {
	standings := NewRelationshipStandingIndex(archive)
	groups := make(map[string][]string)
	for _, id := range sortedKeys(archive.Relationships) {
		rel := archive.Relationships[id]
		if rel == nil || id == excludeID || standings.Relationship(id) == RelationshipStandingDisproven {
			continue
		}
		if sig := relationshipSignature(rel); sig != "" {
			groups[sig] = append(groups[sig], id)
		}
	}

	return groups
}

// FindDuplicateRelationship returns the IDs of existing relationships in the
// archive that rel would duplicate (same type, same participants in the same
// roles, no conflicting start or end event; see DuplicateRelationships), or
// nil when there is none. More than one ID means rel matches several
// relationships that are distinct from each other (an undated marriage beside
// two dated ones). The relationship stored under excludeID is ignored, so a
// caller replacing an entity in place does not find the entity itself.
func FindDuplicateRelationship(archive *GLXFile, rel *Relationship, excludeID string) []string {
	if archive == nil || rel == nil {
		return nil
	}
	sig := relationshipSignature(rel)
	if sig == "" {
		return nil
	}
	group := relationshipGroups(archive, excludeID)[sig]
	if len(group) == 0 {
		return nil
	}

	rels := make(map[string]*Relationship, len(group)+1)
	for _, id := range group {
		rels[id] = archive.Relationships[id]
	}
	rels[excludeID] = rel
	ids := append(slices.Clone(group), excludeID)
	sort.Strings(ids)

	return classifyDuplicates(ids, rels, excludeID)[excludeID]
}

// DuplicateRelationships maps the ID of each relationship that repeats
// another to the relationships it repeats. Two relationships repeat each other
// when they have the same type and the same participants in the same roles
// (types and roles compared ignoring case), and neither their start events nor
// their end events are both set and different. Disproven relationships are
// left out (#1506).
//
// A relationship normally maps to the single earliest relationship, by ID,
// that it repeats. One that matches several relationships which are distinct
// from each other maps to all of them, and is not itself named as the
// relationship the others repeat.
func DuplicateRelationships(archive *GLXFile) map[string][]string {
	if archive == nil {
		return nil
	}
	dups := make(map[string][]string)
	for _, group := range relationshipGroups(archive, "") {
		if len(group) < 2 {
			continue
		}
		for id, of := range classifyDuplicates(group, archive.Relationships, "") {
			dups[id] = of
		}
	}

	return dups
}

// RecordedParentChildLink is a parent → child link of one relationship that
// another relationship already records.
type RecordedParentChildLink struct {
	ParentID string
	ChildID  string
	// RecordedIn is the relationship that already records the link.
	RecordedIn string
}

// parentChildLinkKey identifies a parent → child link regardless of the
// relationship recording it.
type parentChildLinkKey struct {
	parent, child string
	step          bool
}

// ParentChildTypesOverlap reports whether two parent-child relationship types
// state the same link when they join the same parent and child: the same type
// (ignoring case), or the generic parent_child beside any other. Two specific
// types, such as biological and adoptive, are different facts and do not
// overlap.
func ParentChildTypesOverlap(a, b string) bool {
	return strings.EqualFold(a, b) ||
		strings.EqualFold(a, RelationshipTypeParentChild) ||
		strings.EqualFold(b, RelationshipTypeParentChild)
}

// linkCovers reports whether a link recorded by a relationship of type
// coverType makes the same link in a relationship of type relType redundant:
// the types overlap, and the generic parent_child never covers a more
// specific type (the specific one says more).
func linkCovers(coverType, relType string) bool {
	if !ParentChildTypesOverlap(coverType, relType) {
		return false
	}

	return !strings.EqualFold(coverType, RelationshipTypeParentChild) ||
		strings.EqualFold(relType, RelationshipTypeParentChild)
}

// RedundantParentChildLinks maps the ID of each parent-child relationship
// that records a parent → child link another relationship already records to
// those links (#1506). It catches what DuplicateRelationships cannot: a link
// repeated inside a relationship with other participants (one parent → child
// beside a two-parent relationship for the same child), and a generic
// parent_child beside a specific type for the same link, which is reported on
// the generic one. Between two relationships of the same type the later by ID
// is reported. Pairs DuplicateRelationships already reports, disproven links,
// and relationships whose start or end events conflict are left out.
func RedundantParentChildLinks(archive *GLXFile) map[string][]RecordedParentChildLink {
	if archive == nil {
		return nil
	}
	whole := DuplicateRelationships(archive)
	wholePair := func(a, b string) bool {
		return slices.Contains(whole[a], b) || slices.Contains(whole[b], a)
	}

	byKey := make(map[parentChildLinkKey][]ParentChildLink)
	var keys []parentChildLinkKey
	for _, link := range ParentChildLinks(archive, nil) {
		if link.Standing == RelationshipStandingDisproven {
			continue
		}
		key := parentChildLinkKey{link.ParentID, link.ChildID, link.Step}
		if _, seen := byKey[key]; !seen {
			keys = append(keys, key)
		}
		byKey[key] = append(byKey[key], link)
	}

	out := make(map[string][]RecordedParentChildLink)
	for _, key := range keys {
		links := byKey[key]
		if len(links) < 2 {
			continue
		}
		for _, l := range links {
			for _, cover := range links {
				if cover.RelationshipID == l.RelationshipID || !linkCovers(cover.RelationshipType, l.RelationshipType) {
					continue
				}
				// Same type both ways: only the later by ID is redundant.
				if strings.EqualFold(cover.RelationshipType, l.RelationshipType) && cover.RelationshipID > l.RelationshipID {
					continue
				}
				if wholePair(cover.RelationshipID, l.RelationshipID) ||
					!relationshipBoundsCompatible(archive.Relationships[cover.RelationshipID], archive.Relationships[l.RelationshipID]) {
					continue
				}
				out[l.RelationshipID] = append(out[l.RelationshipID], RecordedParentChildLink{
					ParentID: key.parent, ChildID: key.child, RecordedIn: cover.RelationshipID,
				})

				break
			}
		}
	}

	return out
}

// FindRecordedParentChildLinks returns the parent → child links of rel that a
// relationship already in the archive records (see RedundantParentChildLinks
// for when one link covers another), and how many links rel has. The
// relationship stored under excludeID is ignored. A rel that is not a
// parent-child relationship has no links.
func FindRecordedParentChildLinks(archive *GLXFile, rel *Relationship, excludeID string) ([]RecordedParentChildLink, int) {
	if archive == nil || rel == nil || !IsParentChildRelationshipType(rel.Type) {
		return nil, 0
	}
	own := ParentChildLinks(&GLXFile{Relationships: map[string]*Relationship{excludeID: rel}}, nil)
	if len(own) == 0 {
		return nil, 0
	}

	existing := make(map[parentChildLinkKey][]ParentChildLink)
	for _, link := range ParentChildLinks(archive, nil) {
		if link.RelationshipID == excludeID || link.Standing == RelationshipStandingDisproven {
			continue
		}
		key := parentChildLinkKey{link.ParentID, link.ChildID, link.Step}
		existing[key] = append(existing[key], link)
	}

	var recorded []RecordedParentChildLink
	for _, l := range own {
		for _, cover := range existing[parentChildLinkKey{l.ParentID, l.ChildID, l.Step}] {
			if linkCovers(cover.RelationshipType, l.RelationshipType) &&
				relationshipBoundsCompatible(archive.Relationships[cover.RelationshipID], rel) {
				recorded = append(recorded, RecordedParentChildLink{
					ParentID: l.ParentID, ChildID: l.ChildID, RecordedIn: cover.RelationshipID,
				})

				break
			}
		}
	}

	return recorded, len(own)
}

// formatRelationshipRefs renders relationship IDs as `relationships[a]` or
// `relationships[a] or relationships[b]`.
func formatRelationshipRefs(ids []string) string {
	refs := make([]string, len(ids))
	for i, id := range ids {
		refs[i] = fmt.Sprintf("%s[%s]", EntityTypeRelationships, id)
	}

	return strings.Join(refs, " or ")
}

// validateDuplicateRelationships warns on each relationship that repeats
// another, naming the one it repeats, and on each parent-child link recorded
// twice (#1506). A repeated parent-child link otherwise goes unnoticed and
// draws the parent twice in `glx ancestors`.
func (glx *GLXFile) validateDuplicateRelationships(result *ValidationResult) {
	for id, of := range DuplicateRelationships(glx) {
		rel := glx.Relationships[id]
		advice := "merge them or remove one"
		if len(of) > 1 {
			advice = "those differ in start_event or end_event, so give this one the start_event or end_event " +
				"of the one it repeats, or remove it"
		}
		result.Warnings = append(result.Warnings, ValidationWarning{
			SourceType: EntityTypeRelationships,
			SourceID:   id,
			Field:      fieldParticipants,
			Message: fmt.Sprintf("%s[%s]: duplicates %s (same type %q and the same participants in the same roles); %s",
				EntityTypeRelationships, id, formatRelationshipRefs(of), rel.Type, advice),
		})
	}

	for id, links := range RedundantParentChildLinks(glx) {
		for _, l := range links {
			result.Warnings = append(result.Warnings, ValidationWarning{
				SourceType: EntityTypeRelationships,
				SourceID:   id,
				Field:      fieldParticipants,
				Message: fmt.Sprintf("%s[%s]: the link from parent %s to child %s is already recorded by %s[%s]; "+
					"record it once",
					EntityTypeRelationships, id, l.ParentID, l.ChildID, EntityTypeRelationships, l.RecordedIn),
			})
		}
	}
}
