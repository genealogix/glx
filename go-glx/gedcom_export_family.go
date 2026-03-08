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
		if rel.Type != RelationshipTypeMarriage {
			continue
		}

		// Extract the two spouse person IDs from participants
		spouseIDs := extractSpouseIDs(rel)
		if len(spouseIDs) < 2 {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"marriage relationship has fewer than 2 spouse participants")
			continue
		}

		// Determine HUSB/WIFE by gender
		husbandID, wifeID := assignHusbandWife(spouseIDs[0], spouseIDs[1], expCtx)

		familyIdx := len(expCtx.Families)
		family := &ExportFamily{
			HusbandID:      husbandID,
			WifeID:         wifeID,
			ChildPedigrees: make(map[string]string),
			RelationshipID: relID,
		}
		expCtx.Families = append(expCtx.Families, family)

		// Map parent pair to family
		pairKey := makeParentPairKey(husbandID, wifeID)
		parentPairToFamily[pairKey] = familyIdx

		// Map each parent to this family
		parentToFamilies[husbandID] = append(parentToFamilies[husbandID], familyIdx)
		parentToFamilies[wifeID] = append(parentToFamilies[wifeID], familyIdx)
	}

	// Step 2: Attach children from parent-child relationships
	for _, relID := range relIDs {
		rel := expCtx.GLX.Relationships[relID]
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

		// Find which family this parent belongs to
		familyIdx := findFamilyForParent(parentID, parentToFamilies, expCtx)
		if familyIdx < 0 {
			// Parent has no marriage family - create a synthetic single-parent FAM
			familyIdx = createSyntheticFamily(parentID, expCtx, parentToFamilies)
		}

		family := expCtx.Families[familyIdx]

		// Avoid duplicate children
		if !containsString(family.ChildIDs, childID) {
			family.ChildIDs = append(family.ChildIDs, childID)
		}

		// Store pedigree value (prefer more specific over existing)
		if pediValue != "" {
			family.ChildPedigrees[childID] = pediValue
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

	return record
}

// findFamilyEvents finds events where both spouses participate with role "spouse",
// excluding the start and end events (already exported as MARR/DIV).
func findFamilyEvents(husbandID, wifeID, startEventID, endEventID string, expCtx *ExportContext) []*GEDCOMRecord {
	var records []*GEDCOMRecord

	eventIDs := sortedKeys(expCtx.GLX.Events)
	for _, eventID := range eventIDs {
		// Skip start/end events already handled
		if eventID == startEventID || eventID == endEventID {
			continue
		}

		event := expCtx.GLX.Events[eventID]

		// Check if both spouses participate as "spouse"
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

		if !hasHusband || !hasWife {
			continue
		}

		// Map event type to GEDCOM tag
		gedcomTag, ok := expCtx.ExportIndex.EventTypes[event.Type]
		if !ok || gedcomTag == "" {
			continue
		}

		famEventRecord := &GEDCOMRecord{
			Tag:        gedcomTag,
			SubRecords: []*GEDCOMRecord{},
		}

		if event.Date != "" {
			gedcomDate := formatGEDCOMDate(event.Date)
			if gedcomDate != "" {
				famEventRecord.SubRecords = append(famEventRecord.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagDate,
					Value: gedcomDate,
				})
			}
		}

		placRecords := exportPlaceSubrecords(event.PlaceID, expCtx)
		if placRecords != nil {
			famEventRecord.SubRecords = append(famEventRecord.SubRecords, placRecords...)
		}

		records = append(records, famEventRecord)
	}

	return records
}

// extractSpouseIDs extracts person IDs from a marriage relationship's participants.
// Returns person IDs for participants with "spouse" role, or all participants if none have that role.
func extractSpouseIDs(rel *Relationship) []string {
	var spouseIDs []string
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

// findFamilyForParent finds the family index for a parent, preferring the first marriage family.
func findFamilyForParent(parentID string, parentToFamilies map[string][]int, expCtx *ExportContext) int {
	familyIndices, ok := parentToFamilies[parentID]
	if !ok || len(familyIndices) == 0 {
		return -1
	}

	// If parent has exactly one family, use it
	if len(familyIndices) == 1 {
		return familyIndices[0]
	}

	// Multiple families — return the first one (deterministic since relationships are sorted)
	_ = expCtx // available for future use
	return familyIndices[0]
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
