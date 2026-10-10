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
	"strconv"
	"strings"
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
// from parent-child relationships. A marriage or parent-child link the archive
// has disproven is a rejected alternative, not a family tie, and is left out
// with an export warning.
func reconstructFamilies(expCtx *ExportContext) {
	expCtx.Families = nil
	expCtx.FamilyXRefMap = make(map[string]string)
	expCtx.PersonSpouseFamilies = make(map[string][]string)
	expCtx.PersonChildFamilies = make(map[string][]childFamilyRef)
	expCtx.standings = NewRelationshipStandingIndex(expCtx.GLX)

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
		if expCtx.standings.Relationship(relID) == RelationshipStandingDisproven {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"marriage is disproven by every assertion about it; not exported as a family")

			continue
		}

		// Extract spouse person IDs from participants
		spouseIDs := extractSpouseIDs(rel)
		if len(spouseIDs) == 0 {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"marriage relationship has no spouse participants")

			continue
		}

		// Determine HUSB/WIFE by recorded sex
		var husbandID, wifeID string
		if len(spouseIDs) >= 2 {
			husbandID, wifeID = assignHusbandWife(spouseIDs[0], spouseIDs[1], expCtx)
		} else {
			// Single-spouse marriage — assign by recorded sex
			sex := getPersonSex(spouseIDs[0], expCtx)
			if sex == SexFemale {
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

	// Step 2: Attach every child of every parent-child relationship to the
	// family of its parent set, synthesizing a FAM when no marriage matches.
	attachChildrenToFamilies(expCtx, relIDs, parentToFamilies, parentPairToFamily)

	// Step 3: Sort children within each family and assign XRefs
	for i, family := range expCtx.Families {
		sort.Strings(family.ChildIDs)
		xref := fmt.Sprintf("@F%d@", i+1)
		family.FamilyXRef = xref
		expCtx.FamilyXRefMap[family.RelationshipID] = xref

		// Build person-to-family reverse maps
		if family.HusbandID != "" {
			expCtx.PersonSpouseFamilies[family.HusbandID] = append(
				expCtx.PersonSpouseFamilies[family.HusbandID], xref,
			)
		}
		if family.WifeID != "" {
			expCtx.PersonSpouseFamilies[family.WifeID] = append(
				expCtx.PersonSpouseFamilies[family.WifeID], xref,
			)
		}
		for _, childID := range family.ChildIDs {
			pedi := family.ChildPedigrees[childID]
			expCtx.PersonChildFamilies[childID] = append(
				expCtx.PersonChildFamilies[childID], childFamilyRef{
					FamilyXRef: xref,
					Pedigree:   pedi,
				},
			)
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
				marrRecord := exportFamilyEvent(rel.StartEvent, GedcomTagMarr, expCtx, family.HusbandID, family.WifeID)
				if marrRecord != nil {
					record.SubRecords = append(record.SubRecords, marrRecord)
				}
			}

			// DIV from end_event
			if rel.EndEvent != "" {
				divRecord := exportFamilyEvent(rel.EndEvent, GedcomTagDiv, expCtx, family.HusbandID, family.WifeID)
				if divRecord != nil {
					record.SubRecords = append(record.SubRecords, divRecord)
				}
			}

			// Other family events: events where both spouses participate with role "spouse"
			familyEvents := findFamilyEvents(family.HusbandID, family.WifeID,
				rel.StartEvent, rel.EndEvent, expCtx)
			record.SubRecords = append(record.SubRecords, familyEvents...)

			// FAM-level relationship properties mapped to GEDCOM tags (e.g. NCHI)
			record.SubRecords = append(record.SubRecords, exportMappedRelationshipProperties(rel, expCtx)...)

			// NOTE from relationship
			for _, note := range rel.Notes {
				record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
					Tag:   GedcomTagNote,
					Value: note,
				})
			}

			// Preserved FAM-level extension tags from import round-trip
			record.SubRecords = append(record.SubRecords, exportExtensionTags(rel.Properties)...)
		}
	}

	return record
}

// exportMappedRelationshipProperties emits FAM-level subrecords for any
// relationship property that the loaded vocabulary maps to a GEDCOM tag
// (currently number_of_children -> NCHI). Driving emission off the same
// vocabulary mapping used on import keeps the two sides symmetric: whatever
// property key the vocabulary assigns to a GEDCOM tag on import is the key
// export reads back, so a custom vocabulary that renames the property still
// round-trips. Keys are sorted for deterministic output.
func exportMappedRelationshipProperties(rel *Relationship, expCtx *ExportContext) []*GEDCOMRecord {
	if len(rel.Properties) == 0 || len(expCtx.ExportIndex.RelationshipProperties) == 0 {
		return nil
	}

	keys := make([]string, 0, len(rel.Properties))
	for key := range rel.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var records []*GEDCOMRecord
	for _, key := range keys {
		gedcomTag, ok := expCtx.ExportIndex.RelationshipProperties[key]
		if !ok || gedcomTag == "" {
			continue
		}
		value := formatCountProperty(rel.Properties[key])
		if value == "" {
			continue
		}
		records = append(records, &GEDCOMRecord{
			Tag:   gedcomTag,
			Value: value,
		})
	}

	return records
}

// formatCountProperty renders an integer-typed property value as a GEDCOM
// payload. YAML decoding yields int (or int64/float64 depending on source), and
// a non-numeric value imported from a malformed NCHI is preserved as a string.
// Returns "" for absent or unsupported value shapes.
func formatCountProperty(value any) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// Render losslessly rather than truncating: an integral float (3.0)
		// becomes "3", while a fractional value (3.5) keeps its precision
		// instead of being silently truncated to "3". Fractional counts are
		// malformed for NCHI, but preserving the value beats silent data loss.
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	default:
		return ""
	}
}

// exportFamilyEvent creates a family event subrecord (MARR, DIV, etc.)
// from a GLX event ID, using the given GEDCOM tag.
func exportFamilyEvent(eventID, gedcomTag string, expCtx *ExportContext, hostIDs ...string) *GEDCOMRecord {
	event, ok := expCtx.GLX.Events[eventID]
	if !ok {
		return nil
	}
	expCtx.markFamilyEventExported(eventID)

	record := &GEDCOMRecord{
		Tag:        gedcomTag,
		SubRecords: []*GEDCOMRecord{},
	}

	// DATE
	if event.Date != "" {
		gedcomDate := formatGEDCOMDate(event.Date, expCtx.Version)
		if gedcomDate != "" {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagDate,
				Value: gedcomDate,
			})
		}
	}

	// PLAC
	placRecords := exportPlaceSubrecords(event.PlaceID, event.Date, expCtx)
	if placRecords != nil {
		record.SubRecords = append(record.SubRecords, placRecords...)
	}

	// ASSO for participants other than the couple: witnesses, officiants,
	// and other roles (#1321)
	record.SubRecords = append(record.SubRecords, exportEventAssociations(event, func(p Participant) bool {
		return isFamilyEventMemberRole(p.Role)
	}, expCtx)...)
	queueGEDCOM551EventAssociations(eventID, event, gedcomTag, hostIDs, func(p Participant) bool {
		return isFamilyEventMemberRole(p.Role)
	}, expCtx)

	// NOTE — emit one NOTE subrecord per note in the NoteList to preserve
	// note boundaries through roundtrip. Fall back to Properties map.
	if !event.Notes.IsEmpty() {
		for _, note := range event.Notes {
			record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
				Tag:   GedcomTagNote,
				Value: note,
			})
		}
	} else if propNotes, ok := event.Properties[PropertyNotes].(string); ok && propNotes != "" {
		record.SubRecords = append(record.SubRecords, &GEDCOMRecord{
			Tag:   GedcomTagNote,
			Value: propNotes,
		})
	}

	// TYPE: a generic EVEN names its event type (#1320); otherwise the
	// marriage_type property takes precedence, then event_subtype via exportEventPropertySubrecords
	hasExplicitType := false
	if genericType := genericEventTypeRecord(event, expCtx); genericType != nil && gedcomTag == GedcomTagEven {
		record.SubRecords = append(record.SubRecords, genericType)
		hasExplicitType = true
	} else if marriageType, ok := event.Properties[PropertyMarriageType].(string); ok && marriageType != "" {
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

	// SOUR references from event sources, citations and event-subject assertions
	exportEventEvidenceRefs(eventID, event, expCtx, record)

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
		famEventRecord := exportFamilyEvent(eventID, gedcomTag, expCtx, husbandID, wifeID)
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

// assignHusbandWife determines HUSB/WIFE assignment based on recorded sex.
// Male -> HUSB, Female -> WIFE. If both same or unknown, first -> HUSB, second -> WIFE.
func assignHusbandWife(personA, personB string, expCtx *ExportContext) (husbandID, wifeID string) {
	sexA := getPersonSex(personA, expCtx)
	sexB := getPersonSex(personB, expCtx)

	switch {
	case sexA == SexMale && sexB == SexFemale:
		return personA, personB
	case sexA == SexFemale && sexB == SexMale:
		return personB, personA
	default:
		// Same sex, both unknown, or mixed unknown — first is HUSB, second is WIFE
		return personA, personB
	}
}

// getPersonSex retrieves the recorded sex property for a person ID, falling
// back to the legacy gender property for pre-split archives.
//
// Identity-only gender values (notably `nonbinary`) are NEVER surfaced as sex
// — after the two-field split, `gender` may carry identity values that would
// corrupt sex-based export logic (HUSB/WIFE assignment, GEDCOM SEX emission)
// if treated as recorded sex. Only legacy values that are also valid in
// `sex_types` fall through from `gender`.
func getPersonSex(personID string, expCtx *ExportContext) string {
	person, ok := expCtx.GLX.Persons[personID]
	if !ok || person == nil {
		return ""
	}

	// sex and gender are both declared `temporal: true` in
	// person-properties, so the scalar extraction (map[value] / list-first)
	// must run before isLegacySexValue — otherwise HUSB/WIFE inference
	// silently degrades to first/second on any archive that stores sex as
	// a temporal shape.
	if sex, ok := getScalarProperty(person.Properties, PersonPropertySex); ok {
		return sex
	}
	if gender, ok := getScalarProperty(person.Properties, PersonPropertyGender); ok && isLegacySexValue(gender) {
		return gender
	}

	return ""
}

// isLegacySexValue reports whether a gender-property value could plausibly be
// a pre-split `gender:` that actually denoted recorded sex. Identity-only
// values (e.g. `nonbinary`) do not pass this filter.
func isLegacySexValue(v string) bool {
	switch v {
	case SexMale, SexFemale, SexUnknown, SexOther, SexNotRecorded:
		return true
	}

	return false
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

// extractParentChildIDs returns every parent and every child person ID named
// by a parent-child relationship, each deduplicated and in participant order.
// A relationship may name both parents of a child (the shape the relationship
// spec uses) and several children at once, so neither side is a single ID.
func extractParentChildIDs(rel *Relationship) (parentIDs, childIDs []string) {
	for _, p := range rel.Participants {
		if p.Person == "" {
			continue
		}
		switch p.Role {
		case ParticipantRoleParent:
			if !slices.Contains(parentIDs, p.Person) {
				parentIDs = append(parentIDs, p.Person)
			}
		case ParticipantRoleChild:
			if !slices.Contains(childIDs, p.Person) {
				childIDs = append(childIDs, p.Person)
			}
		}
	}

	return parentIDs, childIDs
}

// childParentSet is one set of parents a child is attached to in GEDCOM.
// An explicit set comes from a single relationship that names two or more
// parents together; the non-explicit set gathers every parent that a child's
// one-parent relationships name (the shape GEDCOM import produces, one
// relationship per FAM spouse).
type childParentSet struct {
	parents         []string
	pedi            string
	explicit        bool
	parentPedigrees map[string]string // one-parent relationships, before grouping
}

// survivingParents returns the parents of childID in relationship relID whose
// link to the child the archive has not disproven, plus the rejected parents,
// warning once for each disproven link left out of the export.
func survivingParents(expCtx *ExportContext, relID string, parentIDs []string, childID string) ([]string, []string) {
	parents := make([]string, 0, len(parentIDs))
	var disproven []string
	for _, parentID := range parentIDs {
		if expCtx.standings.Link(relID, parentID, childID) == RelationshipStandingDisproven {
			disproven = append(disproven, parentID)
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				fmt.Sprintf("parent %s of %s is disproven; link not exported", parentID, childID))

			continue
		}
		parents = append(parents, parentID)
	}

	return parents, disproven
}

// collectChildParentSets groups the parent-child relationships by child.
// Each child gets one explicit set per distinct parent set of its
// multi-parent relationships, plus at most one set holding all parents of its
// one-parent relationships. Shared parents remain available for pair matching.
// Rejected parents are kept by child so fallback cannot restore those links.
// Relationships missing a parent or a child are reported as export warnings.
func collectChildParentSets(expCtx *ExportContext, relIDs []string) (map[string][]*childParentSet, map[string][]string) {
	sets := make(map[string][]*childParentSet)
	disprovenParents := make(map[string][]string)
	explicitByKey := make(map[string]*childParentSet)
	singleSets := make(map[string]*childParentSet)

	for _, relID := range relIDs {
		rel := expCtx.GLX.Relationships[relID]
		if rel == nil {
			continue
		}
		pedi := relationshipTypeToPedi(rel.Type)
		if pedi == "" && !isParentChildType(rel.Type) {
			continue
		}

		parentIDs, childIDs := extractParentChildIDs(rel)
		if len(parentIDs) == 0 || len(childIDs) == 0 {
			expCtx.addExportWarning(EntityTypeRelationships, relID,
				"parent-child relationship missing parent or child participant")

			continue
		}

		for _, childID := range childIDs {
			parents, disproven := survivingParents(expCtx, relID, parentIDs, childID)
			if len(disproven) > 0 {
				disprovenParents[childID] = append(disprovenParents[childID], disproven...)
			}
			if len(parents) == 0 {
				continue
			}
			if len(parents) == 1 {
				addSingleParent(singleSets, childID, parents[0], pedi)

				continue
			}

			key := childID + "\x00" + strings.Join(slices.Sorted(slices.Values(parents)), "\x00")
			if set, ok := explicitByKey[key]; ok {
				if set.pedi == "" {
					set.pedi = pedi
				}

				continue
			}
			set := &childParentSet{parents: parents, pedi: pedi, explicit: true}
			explicitByKey[key] = set
			sets[childID] = append(sets[childID], set)
		}
	}

	// Retain shared parents: a singleton mother may also pair with a
	// stepfather even when an explicit set already names her with the father.
	// Fold known singleton pedigree into otherwise unspecified explicit sets.
	for childID, single := range singleSets {
		for _, explicit := range sets[childID] {
			inheritSingleParentPedigree(explicit, single)
		}
		sets[childID] = append(sets[childID], single)
	}

	return sets, disprovenParents
}

func inheritSingleParentPedigree(explicit, single *childParentSet) {
	if explicit.pedi != "" {
		return
	}
	for _, parentID := range explicit.parents {
		if pedi := single.parentPedigrees[parentID]; pedi != "" {
			explicit.pedi = pedi

			return
		}
	}
}

// addSingleParent adds the parent of a one-parent relationship to the child's
// one-parent set, keeping the first non-empty pedigree.
func addSingleParent(singleSets map[string]*childParentSet, childID, parentID, pedi string) {
	set := singleSets[childID]
	if set == nil {
		set = &childParentSet{parentPedigrees: make(map[string]string)}
		singleSets[childID] = set
	}
	if !slices.Contains(set.parents, parentID) {
		set.parents = append(set.parents, parentID)
	}
	if set.pedi == "" {
		set.pedi = pedi
	}
	if set.parentPedigrees[parentID] == "" {
		set.parentPedigrees[parentID] = pedi
	}
}

// attachChildrenToFamilies places every child of every parent-child
// relationship into the FAM of its parent set (#1319):
//
//   - A set whose parents are a married couple joins that couple's FAM. When a
//     parent has several marriages, the pair lookup picks the right one.
//   - An explicit set (one relationship naming two parents) whose parents have
//     no shared FAM gets a FAM synthesized for that couple, which their other
//     children reuse, instead of being filed under one parent's unrelated
//     marriage.
//   - The one-parent fallback joins that parent's first FAM that holds no
//     pair-matched children and would not restore a disproven parent, else a
//     synthesized single-parent FAM.
func attachChildrenToFamilies(expCtx *ExportContext, relIDs []string,
	parentToFamilies map[string][]int, parentPairToFamily map[string]int,
) {
	childSets, disprovenParents := collectChildParentSets(expCtx, relIDs)
	childIDs := make([]string, 0, len(childSets))
	for childID := range childSets {
		childIDs = append(childIDs, childID)
	}
	sort.Strings(childIDs)

	// Pre-scan: families that will hold pair-matched children. A one-parent
	// child is not merged into such a family, because it belongs to a
	// different family unit.
	familiesWithPairedChildren := make(map[int]bool)
	for _, childID := range childIDs {
		for _, set := range childSets[childID] {
			if !set.explicit && len(set.parents) != 2 {
				continue
			}
			for _, idx := range pairedFamilies(set.parents, parentPairToFamily) {
				familiesWithPairedChildren[idx] = true
			}
		}
	}

	for _, childID := range childIDs {
		for _, set := range childSets[childID] {
			matched := pairedFamilies(set.parents, parentPairToFamily)
			switch {
			case len(matched) > 0:
				// The parents' own FAM.
			case set.explicit:
				idx := createSyntheticCoupleFamily(set.parents[0], set.parents[1],
					expCtx, parentToFamilies, parentPairToFamily)
				familiesWithPairedChildren[idx] = true
				matched = []int{idx}
				warnExtraParents(childID, set, expCtx)
			default:
				// Pair matching above needs every singleton edge. Only after it
				// fails can edges already carried by an explicit family be
				// omitted from fallback, avoiding an unrelated extra family.
				set = uncoveredSingleParentSet(set, childSets[childID])
				if len(set.parents) == 0 {
					continue
				}
				matched = singleParentFallbackFamily(set, expCtx, parentToFamilies, familiesWithPairedChildren, disprovenParents[childID])
			}

			assignChildToFamilies(childID, set, matched, expCtx)
		}
	}
}

func assignChildToFamilies(childID string, set *childParentSet, matched []int, expCtx *ExportContext) {
	for _, familyIdx := range matched {
		family := expCtx.Families[familyIdx]
		if !containsString(family.ChildIDs, childID) {
			family.ChildIDs = append(family.ChildIDs, childID)
		}
		if set.pedi != "" && (set.explicit || family.ChildPedigrees[childID] == "") {
			family.ChildPedigrees[childID] = set.pedi
		}
	}
}

// uncoveredSingleParentSet removes only redundant fallback edges, retaining
// pedigree from the remaining parents rather than from an excluded parent.
func uncoveredSingleParentSet(single *childParentSet, sets []*childParentSet) *childParentSet {
	remainder := &childParentSet{}
	for _, parentID := range single.parents {
		if slices.ContainsFunc(sets, func(set *childParentSet) bool {
			return set.explicit && slices.Contains(set.parents, parentID)
		}) {
			continue
		}
		remainder.parents = append(remainder.parents, parentID)
		if remainder.pedi == "" {
			remainder.pedi = single.parentPedigrees[parentID]
		}
	}

	return remainder
}

// warnExtraParents reports a parent set of more than two parents that no
// family matched: the synthesized FAM holds only the first two.
func warnExtraParents(childID string, set *childParentSet, expCtx *ExportContext) {
	if len(set.parents) <= 2 {
		return
	}
	expCtx.addExportWarning(EntityTypePersons, childID, fmt.Sprintf(
		"parent-child relationship names %d parents with no shared family; "+
			"a GEDCOM FAM holds two, so only %s and %s were exported as parents",
		len(set.parents), set.parents[0], set.parents[1],
	))
}

// pairedFamilies returns the sorted indices of the families whose HUSB and
// WIFE are both in parentIDs.
func pairedFamilies(parentIDs []string, parentPairToFamily map[string]int) []int {
	var matched []int
	for i := range parentIDs {
		for j := i + 1; j < len(parentIDs); j++ {
			idx, ok := parentPairToFamily[makeParentPairKey(parentIDs[i], parentIDs[j])]
			if ok && !slices.Contains(matched, idx) {
				matched = append(matched, idx)
			}
		}
	}
	sort.Ints(matched)

	return matched
}

// singleParentFallbackFamily picks the family for a child's one-parent
// relationships when their parents share no FAM: the first parent's first
// family (skipping families with pair-matched children when the child has a
// single parent, or a spouse whose parent link was disproven), else a
// synthesized single-parent family for the first parent.
func singleParentFallbackFamily(set *childParentSet, expCtx *ExportContext,
	parentToFamilies map[string][]int, familiesWithPairedChildren map[int]bool,
	disprovenParents []string,
) []int {
	for _, parentID := range set.parents {
		for _, idx := range parentToFamilies[parentID] {
			if len(set.parents) == 1 && familiesWithPairedChildren[idx] {
				continue
			}
			if fallbackRestoresDisprovenParent(expCtx.Families[idx], set.parents, disprovenParents) {
				continue
			}

			return []int{idx}
		}
	}

	return []int{createSyntheticFamily(set.parents[0], expCtx, parentToFamilies)}
}

// fallbackRestoresDisprovenParent reports whether a family would add a
// rejected parent that the surviving parent set does not directly support.
// A separate surviving link to the same parent remains an accepted alternative.
func fallbackRestoresDisprovenParent(family *ExportFamily, parents, disprovenParents []string) bool {
	for _, spouseID := range []string{family.HusbandID, family.WifeID} {
		if slices.Contains(disprovenParents, spouseID) && !slices.Contains(parents, spouseID) {
			return true
		}
	}

	return false
}

// createSyntheticCoupleFamily creates a FAM for two parents who share no
// marriage relationship and registers it so their other children join it.
func createSyntheticCoupleFamily(parentA, parentB string, expCtx *ExportContext,
	parentToFamilies map[string][]int, parentPairToFamily map[string]int,
) int {
	husbandID, wifeID := assignHusbandWife(parentA, parentB, expCtx)

	familyIdx := len(expCtx.Families)
	expCtx.Families = append(expCtx.Families, &ExportFamily{
		HusbandID:      husbandID,
		WifeID:         wifeID,
		ChildPedigrees: make(map[string]string),
	})
	parentPairToFamily[makeParentPairKey(parentA, parentB)] = familyIdx
	parentToFamilies[parentA] = append(parentToFamilies[parentA], familyIdx)
	parentToFamilies[parentB] = append(parentToFamilies[parentB], familyIdx)

	return familyIdx
}

// createSyntheticFamily creates a single-parent FAM for a parent without a marriage.
func createSyntheticFamily(parentID string, expCtx *ExportContext, parentToFamilies map[string][]int) int {
	sex := getPersonSex(parentID, expCtx)

	family := &ExportFamily{
		ChildPedigrees: make(map[string]string),
	}

	if sex == SexFemale {
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
	return slices.Contains(slice, s)
}
