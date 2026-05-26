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

package main

import (
	"fmt"
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// extractEventYear looks up a person's event (e.g. birth or death) and returns
// the first year from its date, or 0 if no matching event exists.
func extractEventYear(archive *glxlib.GLXFile, personID, eventType string) int {
	_, event := glxlib.FindPersonEvent(archive, personID, eventType)
	if event == nil {
		return 0
	}
	return glxlib.ExtractFirstYear(string(event.Date))
}

// analyzeConsistency checks for chronological inconsistencies across the archive.
func analyzeConsistency(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	issues = append(issues, checkBirthAfterDeath(archive)...)
	issues = append(issues, checkParentYoungerThanChild(archive)...)
	issues = append(issues, checkEventAfterDeath(archive)...)
	issues = append(issues, checkImplausibleLifespan(archive)...)
	issues = append(issues, checkMarriageBeforeBirth(archive)...)
	issues = append(issues, checkDuplicateSiblingNames(archive)...)
	issues = append(issues, checkSiblingBirthplaceOutlier(archive)...)

	return issues
}

// checkBirthAfterDeath reports persons whose death year precedes their birth year.
func checkBirthAfterDeath(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		deathYear := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if birthYear == 0 || deathYear == 0 {
			continue
		}

		if deathYear < birthYear {
			name := personName(archive, id)
			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "high",
				Person:   id,
				Message:  fmt.Sprintf("%s — death year (%d) before birth year (%d)", name, deathYear, birthYear),
			})
		}
	}

	return issues
}

// checkParentYoungerThanChild reports parent-child relationships where the
// parent was born after the child.
func checkParentYoungerThanChild(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, rel := range archive.Relationships {
		if rel == nil {
			continue
		}
		if !isParentChildType(rel.Type) {
			continue
		}

		var parentIDs, childIDs []string
		for _, p := range rel.Participants {
			switch p.Role {
			case glxlib.ParticipantRoleParent:
				parentIDs = append(parentIDs, p.Person)
			case glxlib.ParticipantRoleChild:
				childIDs = append(childIDs, p.Person)
			}
		}

		for _, parentID := range parentIDs {
			parent := archive.Persons[parentID]
			if parent == nil {
				continue
			}
			parentBirth := extractEventYear(archive, parentID, glxlib.EventTypeBirth)
			if parentBirth == 0 {
				continue
			}

			for _, childID := range childIDs {
				child := archive.Persons[childID]
				if child == nil {
					continue
				}
				childBirth := extractEventYear(archive, childID, glxlib.EventTypeBirth)
				if childBirth == 0 {
					continue
				}

				if parentBirth > childBirth {
					parentName := personName(archive, parentID)
					childName := personName(archive, childID)
					issues = append(issues, AnalysisIssue{
						Category: "consistency",
						Severity: "high",
						Person:   childID,
						Message: fmt.Sprintf("Parent %s (born %d) born after child %s (born %d)",
							parentName, parentBirth, childName, childBirth),
					})
				}
			}
		}
	}

	return issues
}

// checkEventAfterDeath reports persons who participate in events dated after
// their death (excluding burial/cremation/probate/will events).
func checkEventAfterDeath(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build death year index
	deathYears := make(map[string]int)
	for id, person := range archive.Persons {
		if person == nil {
			continue
		}
		year := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if year > 0 {
			deathYears[id] = year
		}
	}

	// Events that can legitimately occur after death
	postDeathEvents := map[string]bool{
		glxlib.EventTypeBurial:    true,
		glxlib.EventTypeCremation: true,
		glxlib.EventTypeProbate:   true,
		glxlib.EventTypeWill:      true,
	}

	var issues []AnalysisIssue

	for _, event := range archive.Events {
		if event == nil {
			continue
		}
		if postDeathEvents[event.Type] {
			continue
		}

		eventYear := glxlib.ExtractFirstYear(string(event.Date))
		if eventYear == 0 {
			continue
		}

		for _, p := range event.Participants {
			deathYear, ok := deathYears[p.Person]
			if !ok {
				continue
			}

			if eventYear > deathYear {
				name := personName(archive, p.Person)
				issues = append(issues, AnalysisIssue{
					Category: "consistency",
					Severity: "high",
					Person:   p.Person,
					Message: fmt.Sprintf("%s — %s event (%d) after death (%d)",
						name, event.Type, eventYear, deathYear),
				})
			}
		}
	}

	return issues
}

// checkImplausibleLifespan reports persons with a lifespan exceeding 110 years.
func checkImplausibleLifespan(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, id := range sortedPersonIDs(archive.Persons) {
		person := archive.Persons[id]
		if person == nil {
			continue
		}

		birthYear := extractEventYear(archive, id, glxlib.EventTypeBirth)
		deathYear := extractEventYear(archive, id, glxlib.EventTypeDeath)
		if birthYear == 0 || deathYear == 0 {
			continue
		}

		age := deathYear - birthYear
		if age > 110 {
			name := personName(archive, id)
			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "medium",
				Person:   id,
				Message:  fmt.Sprintf("%s — implausible lifespan of %d years (%d–%d)", name, age, birthYear, deathYear),
			})
		}
	}

	return issues
}

// checkMarriageBeforeBirth reports marriage events that occur before a
// participant's birth.
func checkMarriageBeforeBirth(archive *glxlib.GLXFile) []AnalysisIssue {
	var issues []AnalysisIssue

	for _, event := range archive.Events {
		if event == nil || event.Type != glxlib.EventTypeMarriage {
			continue
		}

		eventYear := glxlib.ExtractFirstYear(string(event.Date))
		if eventYear == 0 {
			continue
		}

		for _, p := range event.Participants {
			person := archive.Persons[p.Person]
			if person == nil {
				continue
			}

			birthYear := extractEventYear(archive, p.Person, glxlib.EventTypeBirth)
			if birthYear == 0 {
				continue
			}

			if eventYear < birthYear {
				name := personName(archive, p.Person)
				issues = append(issues, AnalysisIssue{
					Category: "consistency",
					Severity: "high",
					Person:   p.Person,
					Message:  fmt.Sprintf("%s — marriage (%d) before birth (%d)", name, eventYear, birthYear),
				})
			}
		}
	}

	return issues
}

// checkDuplicateSiblingNames flags parents whose children share the same given name.
// siblingInfo holds an ID and display name for duplicate-name detection.
type siblingInfo struct {
	id       string
	fullName string
}

func checkDuplicateSiblingNames(archive *glxlib.GLXFile) []AnalysisIssue {
	// Build parent → children index
	parentChildren := make(map[string][]string)
	for _, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		var parents, children []string
		for _, p := range rel.Participants {
			if p.Role == glxlib.ParticipantRoleParent {
				parents = append(parents, p.Person)
			} else if p.Role == glxlib.ParticipantRoleChild {
				children = append(children, p.Person)
			}
		}
		for _, parentID := range parents {
			parentChildren[parentID] = append(parentChildren[parentID], children...)
		}
	}

	var issues []AnalysisIssue

	for _, parentID := range sortedPersonIDs(archive.Persons) {
		childIDs := parentChildren[parentID]
		if len(childIDs) < 2 {
			continue
		}

		uniqueChildren := dedupeStrings(childIDs)

		// Map lowercase given name → siblings with that name
		givenNames := make(map[string][]siblingInfo)
		// Track original capitalization for display
		givenDisplay := make(map[string]string)

		for _, cid := range uniqueChildren {
			person, ok := archive.Persons[cid]
			if !ok || person == nil {
				continue
			}
			fullName := glxlib.PersonDisplayName(person)
			given := extractGivenName(person)
			if given == "" {
				continue
			}
			key := strings.ToLower(given)
			givenNames[key] = append(givenNames[key], siblingInfo{id: cid, fullName: fullName})
			if _, exists := givenDisplay[key]; !exists {
				givenDisplay[key] = given
			}
		}

		parentName := personName(archive, parentID)
		for key, siblings := range givenNames {
			if len(siblings) < 2 {
				continue
			}

			if allReplacementPattern(siblings, archive) {
				continue
			}

			var names []string
			for _, c := range siblings {
				names = append(names, fmt.Sprintf("%s (%s)", c.id, c.fullName))
			}
			sort.Strings(names)

			issues = append(issues, AnalysisIssue{
				Category: "consistency",
				Severity: "medium",
				Person:   parentID,
				Message: fmt.Sprintf("%s — children share given name %q: %s",
					parentName, givenDisplay[key], strings.Join(names, " and ")),
			})
		}
	}

	return issues
}

// extractGivenName returns the given (first) name from a person's name property.
// Uses the "given" field if available in structured name data; otherwise splits
// the display name and takes the first token.
func extractGivenName(person *glxlib.Person) string {
	if person == nil || person.Properties == nil {
		return ""
	}
	// Try structured name fields first
	raw, ok := person.Properties["name"]
	if ok {
		if m, ok := raw.(map[string]any); ok {
			if fields, ok := m["fields"].(map[string]any); ok {
				if given, ok := fields["given"].(string); ok && given != "" {
					return strings.Fields(given)[0]
				}
			}
		}
	}
	// Fall back to first token of display name
	fullName := glxlib.PersonDisplayName(person)
	parts := strings.Fields(fullName)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// allReplacementPattern returns true if all duplicate-named siblings follow the
// "replacement" pattern: each earlier child died before or in the same year the
// next was born.
func allReplacementPattern(siblings []siblingInfo, archive *glxlib.GLXFile) bool {
	type yearPair struct {
		birth int
		death int
	}
	var pairs []yearPair
	for _, c := range siblings {
		_, ok := archive.Persons[c.id]
		if !ok {
			return false
		}
		birth := extractEventYear(archive, c.id, glxlib.EventTypeBirth)
		death := extractEventYear(archive, c.id, glxlib.EventTypeDeath)
		pairs = append(pairs, yearPair{birth: birth, death: death})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].birth < pairs[j].birth })

	for i := 0; i < len(pairs)-1; i++ {
		if pairs[i].death == 0 || pairs[i+1].birth == 0 {
			return false
		}
		// If the earlier child died strictly after the next was born,
		// they overlapped — not a replacement. Same year (death == birth)
		// is allowed since infant death + replacement in the same year
		// was a common historical pattern.
		if pairs[i].death > pairs[i+1].birth {
			return false
		}
	}
	return true
}

// dedupeStrings returns unique strings preserving order.
func dedupeStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// hypotheticalConfidence is the set of confidence values that mark a
// parent-child relationship as "hypothetical" for outlier-severity bumping.
// "tentative" is the literal spec match; "low" and "disputed" are included
// because low-confidence or disputed parentage is genealogically equivalent
// — the link is not established. If false positives surface, narrow this set.
var hypotheticalConfidence = map[string]bool{
	"tentative": true,
	"low":       true,
	"disputed":  true,
}

// Severity levels used by the sibling-birthplace-outlier check. Extracted to
// avoid goconst-flagged string repetition; the global severity vocabulary
// otherwise lives inline at the AnalysisIssue call sites.
const (
	severityMedium = "medium"
	severityHigh   = "high"
)

// buildRelationshipConfidenceIndex walks archive.Assertions once and returns
// a map from relationship ID to the confidence of the first assertion whose
// Subject is that relationship. Used by the sibling-birthplace-outlier check
// to detect "hypothetical" parent-child relationships without re-scanning
// every assertion per outlier child.
func buildRelationshipConfidenceIndex(archive *glxlib.GLXFile) map[string]string {
	if archive == nil {
		return nil
	}
	index := make(map[string]string)
	for _, a := range archive.Assertions {
		if a == nil || a.Subject.Type() != glxlib.EntityTypeRelationships {
			continue
		}
		relID := a.Subject.ID()
		if _, seen := index[relID]; seen {
			continue
		}
		index[relID] = a.Confidence
	}

	return index
}

// siblingChildPlace pairs a child ID with its recorded birth PlaceID.
type siblingChildPlace struct {
	childID string
	placeID string
}

// parentChildIndex holds the three indexes needed for the sibling-birthplace
// outlier check: per-parent children, per-child relationships, and per-rel
// parents. Built once per archive.
type parentChildIndex struct {
	parentChildren map[string][]string // parentID → child person IDs (may duplicate)
	childToRelIDs  map[string][]string // childID → relIDs the child appears in as child
	relParents     map[string][]string // relID → parent person IDs in that relationship
}

// buildParentChildIndex walks the archive's relationships once and builds
// the three indexes used by checkSiblingBirthplaceOutlier.
func buildParentChildIndex(archive *glxlib.GLXFile) parentChildIndex {
	idx := parentChildIndex{
		parentChildren: make(map[string][]string),
		childToRelIDs:  make(map[string][]string),
		relParents:     make(map[string][]string),
	}
	for relID, rel := range archive.Relationships {
		if rel == nil || !isParentChildType(rel.Type) {
			continue
		}
		var parents, children []string
		for _, p := range rel.Participants {
			switch p.Role {
			case glxlib.ParticipantRoleParent:
				parents = append(parents, p.Person)
			case glxlib.ParticipantRoleChild:
				children = append(children, p.Person)
			}
		}
		idx.relParents[relID] = parents
		for _, cid := range children {
			idx.childToRelIDs[cid] = append(idx.childToRelIDs[cid], relID)
		}
		for _, pid := range parents {
			idx.parentChildren[pid] = append(idx.parentChildren[pid], children...)
		}
	}

	return idx
}

// childrenWithBirthPlace returns the subset of childIDs that have a birth
// event with a non-empty PlaceID, sorted by childID for deterministic output.
func childrenWithBirthPlace(archive *glxlib.GLXFile, childIDs []string) []siblingChildPlace {
	var withPlace []siblingChildPlace
	for _, cid := range childIDs {
		_, birth := glxlib.FindPersonEvent(archive, cid, glxlib.EventTypeBirth)
		if birth == nil || birth.PlaceID == "" {
			continue
		}
		withPlace = append(withPlace, siblingChildPlace{childID: cid, placeID: birth.PlaceID})
	}
	sort.Slice(withPlace, func(i, j int) bool { return withPlace[i].childID < withPlace[j].childID })

	return withPlace
}

// strictMajorityBirthplace returns the PlaceID shared by a strict majority
// (>50%) of the given children, and its count. If no strict majority exists,
// returns ("", 0). Ties at the maximum are broken by sorted PlaceID order
// for deterministic test output (a tie can't pass the strict-majority filter).
func strictMajorityBirthplace(children []siblingChildPlace) (string, int) {
	counts := make(map[string]int)
	for _, c := range children {
		counts[c.placeID]++
	}
	placeIDs := make([]string, 0, len(counts))
	for pid := range counts {
		placeIDs = append(placeIDs, pid)
	}
	sort.Strings(placeIDs)

	var majorityPlace string
	var majorityCount int
	for _, pid := range placeIDs {
		if counts[pid] > majorityCount {
			majorityCount = counts[pid]
			majorityPlace = pid
		}
	}
	if majorityCount*2 <= len(children) {
		return "", 0
	}

	return majorityPlace, majorityCount
}

// childRelationshipIsHypothetical returns true if any parent_child relationship
// involving childID has a confidence value in hypotheticalConfidence.
func childRelationshipIsHypothetical(childID string, idx parentChildIndex, relConfidence map[string]string) bool {
	for _, relID := range idx.childToRelIDs[childID] {
		if hypotheticalConfidence[relConfidence[relID]] {
			return true
		}
	}

	return false
}

// parentBirthplaceReinforces returns true if the majority place matches at
// least one parent's birth PlaceID, and the outlier place matches none of
// them. Parents with no birth event or no PlaceID are skipped — missing
// data is not a match. Parents are gathered across all parent_child
// relationships involving childID.
func parentBirthplaceReinforces(archive *glxlib.GLXFile, childID, majorityPlace, outlierPlace string, idx parentChildIndex) bool {
	parentSet := make(map[string]bool)
	for _, relID := range idx.childToRelIDs[childID] {
		for _, p := range idx.relParents[relID] {
			parentSet[p] = true
		}
	}
	var majorityMatches, outlierMatches bool
	for pid := range parentSet {
		_, parentBirth := glxlib.FindPersonEvent(archive, pid, glxlib.EventTypeBirth)
		if parentBirth == nil || parentBirth.PlaceID == "" {
			continue
		}
		if parentBirth.PlaceID == majorityPlace {
			majorityMatches = true
		}
		if parentBirth.PlaceID == outlierPlace {
			outlierMatches = true
		}
	}

	return majorityMatches && !outlierMatches
}

// buildSiblingOutlierIssue assembles the AnalysisIssue for a single outlier
// child, including severity bumps and annotation phrases.
func buildSiblingOutlierIssue(archive *glxlib.GLXFile, c siblingChildPlace, majorityPlace string, majorityCount, n int, idx parentChildIndex, relConfidence map[string]string) AnalysisIssue {
	severity := severityMedium
	var annotations []string

	if childRelationshipIsHypothetical(c.childID, idx, relConfidence) {
		severity = severityHigh
		annotations = append(annotations, "parent-child relationship hypothetical")
	}
	if parentBirthplaceReinforces(archive, c.childID, majorityPlace, c.placeID, idx) {
		severity = severityHigh
		annotations = append(annotations, "matches parent birthplace")
	}

	childName := personName(archive, c.childID)
	msg := fmt.Sprintf("%s — born in %s while %d of %d siblings with recorded birthplaces born in %s",
		childName,
		resolvePlaceName(c.placeID, archive),
		majorityCount, n,
		resolvePlaceName(majorityPlace, archive),
	)
	if len(annotations) > 0 {
		msg += " (" + strings.Join(annotations, "; ") + ")"
	}

	return AnalysisIssue{
		Category: "consistency",
		Severity: severity,
		Person:   c.childID,
		Message:  msg,
	}
}

// checkSiblingBirthplaceOutlier flags children whose birthplace differs from
// the strict majority shared by their siblings. Severity is "medium" by
// default, elevated to "high" when the outlier's parent-child relationship
// is hypothetical or when the majority is reinforced by a parent's birthplace.
//
// The check runs per parent in the archive. A parent's child set must have at
// least 3 children with recorded birth PlaceIDs, and a strict majority (>50%)
// must share a single PlaceID, before any outliers are flagged. Place
// comparison is exact PlaceID equality — siblings born in distinct but
// related places (e.g. different counties of the same state) are treated as
// outliers.
func checkSiblingBirthplaceOutlier(archive *glxlib.GLXFile) []AnalysisIssue {
	idx := buildParentChildIndex(archive)
	relConfidence := buildRelationshipConfidenceIndex(archive)

	// Dedup key for emitted issues: same child, same majority, same outlier
	// place should produce one flag (a child with two parents would otherwise
	// be flagged twice via parallel iterations).
	emitted := make(map[string]bool)
	var issues []AnalysisIssue

	for _, parentID := range sortedPersonIDs(archive.Persons) {
		childIDs := dedupeStrings(idx.parentChildren[parentID])
		if len(childIDs) < 3 {
			continue
		}
		withPlace := childrenWithBirthPlace(archive, childIDs)
		if len(withPlace) < 3 {
			continue
		}
		majorityPlace, majorityCount := strictMajorityBirthplace(withPlace)
		if majorityPlace == "" {
			continue
		}

		for _, c := range withPlace {
			if c.placeID == majorityPlace {
				continue
			}
			key := c.childID + "|" + majorityPlace + "|" + c.placeID
			if emitted[key] {
				continue
			}
			emitted[key] = true
			issues = append(issues, buildSiblingOutlierIssue(archive, c, majorityPlace, majorityCount, len(withPlace), idx, relConfidence))
		}
	}

	return issues
}
