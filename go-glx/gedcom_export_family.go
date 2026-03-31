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
	"sort"
)

// ExportFamily represents a reconstructed GEDCOM FAM record from GLX relationships.
type ExportFamily struct {
	FamilyXRef     string            // @F1@, @F2@, etc.
	HusbandID      string            // GLX person ID for HUSB
	WifeID         string            // GLX person ID for WIFE
	ChildIDs       []string          // GLX person IDs (sorted)
	ChildPedigrees map[string]string // child person ID -> PEDI value
	RelationshipID string            // GLX relationship ID (marriage), empty for synthetic
}

// childFamilyRef stores a child's family reference with pedigree info.
type childFamilyRef struct {
	FamilyXRef string
	Pedigree   string // "birth", "adopted", "foster", or "" for generic
}

// reconstructFamilies scans relationships to build ExportFamily structures.
// It creates FAM records from marriage relationships and attaches children
// from parent-child relationships.
func reconstructFamilies(expCtx *ExportContext) {
	expCtx.Families = nil
	expCtx.FamilyXRefMap = make(map[string]string)
	expCtx.PersonSpouseFamilies = make(map[string][]string)
	expCtx.PersonChildFamilies = make(map[string][]childFamilyRef)

	// Step 1: Create families from marriage relationships
	// parentToFamilies maps a person ID to the indices of families they're a spouse in
	parentToFamilies := make(map[string][]int)
	// parentPairToFamily maps sorted spouse pair key to family index
	parentPairToFamily := make(map[string]int)

	relIDs := sortedKeys(expCtx.GLX.Relationships)
	for _, relID := range relIDs {
		rel := expCtx.GLX.Relationships[relID]
		if rel == nil || rel.Type != RelationshipTypeMarriage {
			continue
		}

		// Extract spouse person IDs from participants
		spouseIDs := extractSpouseIDs(rel)
		if len(spouseIDs) == 0 {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"marriage relationship has no spouse participants")
			continue
		}

		// Determine HUSB/WIFE by gender
		var husbandID, wifeID string
		if len(spouseIDs) >= 2 {
			husbandID, wifeID = assignHusbandWife(spouseIDs[0], spouseIDs[1], expCtx)
		} else {
			// Single-spouse marriage — assign by gender
			gender := getPersonGender(spouseIDs[0], expCtx)
			if gender == GenderFemale {
				wifeID = spouseIDs[0]
			} else {
				husbandID = spouseIDs[0]
			}
		}

		familyIdx := len(expCtx.Families)
		family := &ExportFamily{
			HusbandID:      husbandID,
			WifeID:         wifeID,
			ChildPedigrees: make(map[string]string),
			RelationshipID: relID,
		}
		expCtx.Families = append(expCtx.Families, family)

		// Map parent pair to family (only when both spouses are known;
		// single-spouse families use the parentToFamilies fallback)
		if husbandID != "" && wifeID != "" {
			pairKey := makeParentPairKey(husbandID, wifeID)
			parentPairToFamily[pairKey] = familyIdx
		}

		// Map each parent to this family
		if husbandID != "" {
			parentToFamilies[husbandID] = append(parentToFamilies[husbandID], familyIdx)
		}
		if wifeID != "" {
			parentToFamilies[wifeID] = append(parentToFamilies[wifeID], familyIdx)
		}
	}

	// Step 2: Collect all parent-child relationships per child.
	// A child may have two parent-child relationships (one per parent).
	// We use this to match children to the correct family when a parent has
	// multiple marriages.
	type childParentInfo struct {
		relID    string
		parentID string
		childID  string
		pedi     string
	}

	// childParents maps child ID -> list of parent relationships
	childParents := make(map[string][]childParentInfo)

	for _, relID := range relIDs {
		rel := expCtx.GLX.Relationships[relID]
		if rel == nil {
			continue
		}
		pediValue := relationshipTypeToPedi(rel.Type)
		if pediValue == "" && !isParentChildType(rel.Type) {
			continue
		}

		parentID, childID := extractParentChildIDs(rel)
		if parentID == "" || childID == "" {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"parent-child relationship missing parent or child participant")
			continue
		}

		childParents[childID] = append(childParents[childID], childParentInfo{
			relID:    relID,
			parentID: parentID,
			childID:  childID,
			pedi:     pediValue,
		})
	}

	// Now attach each child to the correct family.
	// When a child has two parents, use parentPairToFamily to find the family
	// containing both parents (handles multi-marriage cases correctly).
	childIDs := make([]string, 0, len(childParents))
	for childID := range childParents {
		childIDs = append(childIDs, childID)
	}
	sort.Strings(childIDs)

	// Pre-scan: identify families that will have pair-matched children (both parents known).
	// This lets us avoid merging single-parent children into a two-parent family that
	// already has properly paired children from a different family unit.
	familiesWithPairedChildren := make(map[int]bool)
	for _, cID := range childIDs {
		cParents := childParents[cID]
		cParentIDs := make(map[string]bool)
		for _, cp := range cParents {
			cParentIDs[cp.parentID] = true
		}
		if len(cParentIDs) == 2 {
			pids := make([]string, 0, 2)
			for pid := range cParentIDs {
				pids = append(pids, pid)
			}
			pairKey := makeParentPairKey(pids[0], pids[1])
			if idx, ok := parentPairToFamily[pairKey]; ok {
				familiesWithPairedChildren[idx] = true
			}
		}
	}

	for _, childID := range childIDs {
		parents := childParents[childID]

		// Collect unique parent IDs and best pedigree value
		parentIDs := make(map[string]string) // parent ID -> pedi
		for _, cp := range parents {
			if existing, ok := parentIDs[cp.parentID]; !ok || (existing == "" && cp.pedi != "") {
				parentIDs[cp.parentID] = cp.pedi
			}
		}

		// Best pedigree: prefer non-empty
		bestPedi := ""
		for _, pedi := range parentIDs {
			if pedi != "" {
				bestPedi = pedi
				break
			}
		}

		// Find ALL matching families for this child.
		// A child may belong to multiple families (e.g., birth family + step-family).
		matchedFamilies := make(map[int]bool)

		// Try all parent pairs against known families
		parentList := make([]string, 0, len(parentIDs))
		for pid := range parentIDs {
			parentList = append(parentList, pid)
		}
		for i := 0; i < len(parentList); i++ {
			for j := i + 1; j < len(parentList); j++ {
				pairKey := makeParentPairKey(parentList[i], parentList[j])
				if idx, ok := parentPairToFamily[pairKey]; ok {
					matchedFamilies[idx] = true
				}
			}
		}

		// Fallback: if no pair matches, use the first parent's family
		if len(matchedFamilies) == 0 {
			for _, cp := range parents {
				familyIndices := parentToFamilies[cp.parentID]
				for _, idx := range familyIndices {
					// If child has only one parent and the family already has
					// pair-matched children, this child belongs to a different
					// family unit — don't merge it in.
					if len(parentIDs) == 1 && familiesWithPairedChildren[idx] {
						continue
					}
					matchedFamilies[idx] = true
					break
				}
				if len(matchedFamilies) > 0 {
					break
				}
			}
		}

		// Still no family: create a synthetic single-parent FAM
		if len(matchedFamilies) == 0 && len(parents) > 0 {
			idx := createSyntheticFamily(parents[0].parentID, expCtx, parentToFamilies)
			if idx >= 0 {
				matchedFamilies[idx] = true
			}
		}

		// Place child in all matched families
		for familyIdx := range matchedFamilies {
			family := expCtx.Families[familyIdx]

			if !containsString(family.ChildIDs, childID) {
				family.ChildIDs = append(family.ChildIDs, childID)
			}

			if bestPedi != "" {
				family.ChildPedigrees[childID] = bestPedi
			}
		}
	}

	// Step 3: Sort children within each family and assign XRefs
	for i, family := range expCtx.Families {
		sort.Strings(family.ChildIDs)
		xref := fmt.Sprintf("@F%d@", i+1)
		family.FamilyXRef = xref
		expCtx.FamilyXRefMap[family.RelationshipID] = xref

		// Build person-to-family reverse maps
		if family.HusbandID != "" {
			expCtx.PersonSpouseFamilies[family.HusbandID] = append(
				expCtx.PersonSpouseFamilies[family.HusbandID], xref)
		}
		if family.WifeID != "" {
			expCtx.PersonSpouseFamilies[family.WifeID] = append(
				expCtx.PersonSpouseFamilies[family.WifeID], xref)
		}
		for _, childID := range family.ChildIDs {
			pedi := family.ChildPedigrees[childID]
			expCtx.PersonChildFamilies[childID] = append(
				expCtx.PersonChildFamilies[childID], childFamilyRef{
					FamilyXRef: xref,
					Pedigree:   pedi,
				})
		}
	}
}

// exportFamily converts an ExportFamily to a GEDCOM FAM record.
func exportFamily(family *ExportFamily, expCtx *ExportContext) *GEDCOMRecord {
	record := &GEDCOMRecord{
		XRef:       family.FamilyXRef,
		Tag:        GedcomTagFam,
		SubRecords: []*GEDCOMRecord{},
	}

	// HUSB
	if family.HusbandID != "" {
		if xref, ok := expCtx.PersonXRefMap[family.HusbandID]; ok {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagHusb,
				Value: xref,
			})
		}
	}

	// WIFE
	if family.WifeID != "" {
		if xref, ok := expCtx.PersonXRefMap[family.WifeID]; ok {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagWife,
				Value: xref,
			})
		}
	}

	// CHIL (sorted by child person ID)
	for _, childID := range family.ChildIDs {
		if xref, ok := expCtx.PersonXRefMap[childID]; ok {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagChil,
				Value: xref,
			})
		}
	}

	// Marriage event (from relationship's StartEvent)
	if family.RelationshipID != "" {
		rel, ok := expCtx.GLX.Relationships[family.RelationshipID]
		if ok {
			// MARR from start_event
			if rel.StartEvent != "" {
				marrRecord := exportFamilyEvent(rel.StartEvent, GedcomTagMarr, expCtx)
				if marrRecord != nil {
					record.SubRecords = append(record.SubRecords, marrRecord)
				}
			}

			// DIV from end_event
			if rel.EndEvent != "" {
				divRecord := exportFamilyEvent(rel.EndEvent, GedcomTagDiv, expCtx)
				if divRecord != nil {
					record.SubRecords = append(record.SubRecords, divRecord)
				}
			}

			// Other family events: events where both spouses participate with role "spouse"
			familyEvents := findFamilyEvents(family.HusbandID, family.WifeID,
				rel.StartEvent, rel.EndEvent, expCtx)
			for _, fe := range familyEvents {
				record.SubRecords = append(record.SubRecords, fe)
			}

			// NOTE from relationship
			if rel.Notes != "" {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagNote,
					Value: rel.Notes,
				})
			}
		}
	}

	return record
}

// exportFamilyEvent creates a family event subrecord (MARR, DIV, etc.)
// from a GLX event ID, using the given GEDCOM tag.
func exportFamilyEvent(eventID, gedcomTag string, expCtx *ExportContext) *GEDCOMRecord {
	event, ok := expCtx.GLX.Events[eventID]
	if !ok {
		return nil
	}

	record := &GEDCOMRecord{
		Tag:        gedcomTag,
		SubRecords: []*GEDCOMRecord{},
	}

	// DATE
	if event.Date != "" {
		gedcomDate := formatGEDCOMDate(event.Date)
		if gedcomDate != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagDate,
				Value: gedcomDate,
			})
		}
	}

	// PLAC
	placRecords := exportPlaceSubrecords(event.PlaceID, expCtx)
	if placRecords != nil {
		record.SubRecords = append(record.SubRecords, placRecords...)
	}

	// NOTE - check both struct field and Properties map
	famEventNoteText := event.Notes
	if famEventNoteText == "" {
		if propNotes, ok := event.Properties[PropertyNotes].(string); ok {
			famEventNoteText = propNotes
		}
	}
	if famEventNoteText != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: famEventNoteText,
		})
	}

	// TYPE (marriage_type property takes precedence, then event_subtype via exportEventPropertySubrecords)
	hasExplicitType := false
	if marriageType, ok := event.Properties[PropertyMarriageType].(string); ok && marriageType != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagType,
			Value: marriageType,
		})
		hasExplicitType = true
	}

	// Other event properties (event_subtype, cause, age_at_event, etc.)
	for _, propRec := range exportEventPropertySubrecords(event, expCtx) {
		if hasExplicitType && propRec.Tag == GedcomTagType {
			continue // Skip duplicate TYPE
		}
		record.SubRecords = append(record.SubRecords, propRec)
	}

	// SOUR references from event sources and citations
	exportEventSourceRefs(event, expCtx, record)

	return record
}

// findFamilyEvents finds events where the family's spouses participate with role "spouse",
// excluding the start and end events (already exported as MARR/DIV).
// For single-spouse families, only the known spouse needs to participate.
func findFamilyEvents(husbandID, wifeID, startEventID, endEventID string, expCtx *ExportContext) []*GEDCOMRecord {
	var records []*GEDCOMRecord

	eventIDs := sortedKeys(expCtx.GLX.Events)
	for _, eventID := range eventIDs {
		// Skip start/end events already handled
		if eventID == startEventID || eventID == endEventID {
			continue
		}

		event := expCtx.GLX.Events[eventID]
		if event == nil {
			continue
		}

		// Check if the family's spouses participate as "spouse".
		// For single-spouse families, only require the known spouse.
		var hasHusband, hasWife bool
		for _, p := range event.Participants {
			if p.Role == ParticipantRoleSpouse {
				if p.Person == husbandID {
					hasHusband = true
				}
				if p.Person == wifeID {
					hasWife = true
				}
			}
		}

		needHusband := husbandID != ""
		needWife := wifeID != ""
		if (needHusband && !hasHusband) || (needWife && !hasWife) {
			continue
		}
		if !hasHusband && !hasWife {
			continue
		}

		// Map event type to GEDCOM tag
		gedcomTag, ok := expCtx.ExportIndex.EventTypes[event.Type]
		if !ok || gedcomTag == "" {
			continue
		}

		// Reuse exportFamilyEvent to get full sub-record export (DATE, PLAC, NOTE, SOUR, properties)
		famEventRecord := exportFamilyEvent(eventID, gedcomTag, expCtx)
		if famEventRecord != nil {
			records = append(records, famEventRecord)
		}
	}

	return records
}

// extractSpouseIDs extracts person IDs from a marriage relationship's participants.
// Returns person IDs for participants with "spouse" role, or all participants if none have that role.
func extractSpouseIDs(rel *Relationship) []string {
	var spouseIDs []string

	// First, collect participants explicitly marked as "spouse"
	for _, p := range rel.Participants {
		if p.Role == ParticipantRoleSpouse && p.Person != "" {
			spouseIDs = append(spouseIDs, p.Person)
		}
	}

	// If any spouse-role participants were found, return only those
	if len(spouseIDs) > 0 {
		return spouseIDs
	}

	// Fallback: no explicit spouse-role participants; return all participants
	for _, p := range rel.Participants {
		if p.Person != "" {
			spouseIDs = append(spouseIDs, p.Person)
		}
	}

	return spouseIDs
}

// assignHusbandWife determines HUSB/WIFE assignment based on gender.
// Male -> HUSB, Female -> WIFE. If both same or unknown, first -> HUSB, second -> WIFE.
func assignHusbandWife(personA, personB string, expCtx *ExportContext) (husbandID, wifeID string) {
	genderA := getPersonGender(personA, expCtx)
	genderB := getPersonGender(personB, expCtx)

	switch {
	case genderA == GenderMale && genderB == GenderFemale:
		return personA, personB
	case genderA == GenderFemale && genderB == GenderMale:
		return personB, personA
	default:
		// Same gender, both unknown, or mixed unknown — first is HUSB, second is WIFE
		return personA, personB
	}
}

// getPersonGender retrieves the gender property for a person ID.
func getPersonGender(personID string, expCtx *ExportContext) string {
	person, ok := expCtx.GLX.Persons[personID]
	if !ok {
		return ""
	}

	gender, ok := getStringProperty(person.Properties, PersonPropertyGender)
	if !ok {
		return ""
	}

	return gender
}

// relationshipTypeToPedi maps GLX relationship types to GEDCOM PEDI values.
func relationshipTypeToPedi(relType string) string {
	switch relType {
	case RelationshipTypeBiologicalParentChild:
		return "birth"
	case RelationshipTypeAdoptiveParentChild:
		return "adopted"
	case RelationshipTypeFosterParentChild:
		return "foster"
	default:
		return ""
	}
}

// isParentChildType returns true if the relationship type is a parent-child variant.
func isParentChildType(relType string) bool {
	switch relType {
	case RelationshipTypeParentChild,
		RelationshipTypeBiologicalParentChild,
		RelationshipTypeAdoptiveParentChild,
		RelationshipTypeFosterParentChild:
		return true
	default:
		return false
	}
}

// extractParentChildIDs extracts parent and child person IDs from a parent-child relationship.
func extractParentChildIDs(rel *Relationship) (parentID, childID string) {
	for _, p := range rel.Participants {
		switch p.Role {
		case ParticipantRoleParent:
			parentID = p.Person
		case ParticipantRoleChild:
			childID = p.Person
		}
	}
	return parentID, childID
}

// createSyntheticFamily creates a single-parent FAM for a parent without a marriage.
func createSyntheticFamily(parentID string, expCtx *ExportContext, parentToFamilies map[string][]int) int {
	gender := getPersonGender(parentID, expCtx)

	family := &ExportFamily{
		ChildPedigrees: make(map[string]string),
	}

	if gender == GenderFemale {
		family.WifeID = parentID
	} else {
		family.HusbandID = parentID
	}

	familyIdx := len(expCtx.Families)
	expCtx.Families = append(expCtx.Families, family)
	parentToFamilies[parentID] = append(parentToFamilies[parentID], familyIdx)

	return familyIdx
}

// makeParentPairKey creates a deterministic key from two spouse IDs.
func makeParentPairKey(idA, idB string) string {
	if idA < idB {
		return idA + "|" + idB
	}
	return idB + "|" + idA
}

// containsString checks if a slice contains a string.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
