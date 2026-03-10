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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// reconstructFamilies tests
// ============================================================================

func TestReconstructFamilies_BasicMarriage(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-husband": {
					Properties: map[string]any{
						PersonPropertyGender: "male",
					},
				},
				"person-wife": {
					Properties: map[string]any{
						PersonPropertyGender: "female",
					},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-marriage": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-husband", Role: ParticipantRoleSpouse},
						{Person: "person-wife", Role: ParticipantRoleSpouse},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-husband": "@I1@",
			"person-wife":    "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	require.Len(t, expCtx.Families, 1)

	family := expCtx.Families[0]
	assert.Equal(t, "@F1@", family.FamilyXRef)
	assert.Equal(t, "person-husband", family.HusbandID)
	assert.Equal(t, "person-wife", family.WifeID)
	assert.Equal(t, "rel-marriage", family.RelationshipID)
	assert.Empty(t, family.ChildIDs)

	// Check person-to-spouse-family maps
	assert.Contains(t, expCtx.PersonSpouseFamilies["person-husband"], "@F1@")
	assert.Contains(t, expCtx.PersonSpouseFamilies["person-wife"], "@F1@")
}

func TestReconstructFamilies_WithChildren(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-father": {
					Properties: map[string]any{
						PersonPropertyGender: "male",
					},
				},
				"person-mother": {
					Properties: map[string]any{
						PersonPropertyGender: "female",
					},
				},
				"person-child1": {
					Properties: map[string]any{},
				},
				"person-child2": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-marriage": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleSpouse},
						{Person: "person-mother", Role: ParticipantRoleSpouse},
					},
				},
				"rel-parent-child1": {
					Type: RelationshipTypeBiologicalParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child1", Role: ParticipantRoleChild},
					},
				},
				"rel-parent-child2": {
					Type: RelationshipTypeBiologicalParentChild,
					Participants: []Participant{
						{Person: "person-mother", Role: ParticipantRoleParent},
						{Person: "person-child2", Role: ParticipantRoleChild},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-father": "@I1@",
			"person-mother": "@I2@",
			"person-child1": "@I3@",
			"person-child2": "@I4@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	require.Len(t, expCtx.Families, 1)

	family := expCtx.Families[0]
	assert.Equal(t, "person-father", family.HusbandID)
	assert.Equal(t, "person-mother", family.WifeID)
	assert.Len(t, family.ChildIDs, 2)
	assert.Contains(t, family.ChildIDs, "person-child1")
	assert.Contains(t, family.ChildIDs, "person-child2")

	// Both children should have "birth" pedigree
	assert.Equal(t, "birth", family.ChildPedigrees["person-child1"])
	assert.Equal(t, "birth", family.ChildPedigrees["person-child2"])

	// Child-to-family maps
	require.Len(t, expCtx.PersonChildFamilies["person-child1"], 1)
	assert.Equal(t, "@F1@", expCtx.PersonChildFamilies["person-child1"][0].FamilyXRef)
	assert.Equal(t, "birth", expCtx.PersonChildFamilies["person-child1"][0].Pedigree)
}

func TestReconstructFamilies_SingleParent(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-mother": {
					Properties: map[string]any{
						PersonPropertyGender: "female",
					},
				},
				"person-child": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-parent-child": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-mother", Role: ParticipantRoleParent},
						{Person: "person-child", Role: ParticipantRoleChild},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-mother": "@I1@",
			"person-child":  "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	// Should create a synthetic single-parent family
	require.Len(t, expCtx.Families, 1)

	family := expCtx.Families[0]
	// Mother is female, so should be WIFE
	assert.Equal(t, "", family.HusbandID)
	assert.Equal(t, "person-mother", family.WifeID)
	assert.Len(t, family.ChildIDs, 1)
	assert.Equal(t, "person-child", family.ChildIDs[0])
	assert.Equal(t, "", family.RelationshipID) // synthetic family has no marriage relationship
}

// TestReconstructFamilies_SingleParentWithExistingFamily verifies that when a parent
// appears in both a marriage family AND has children from another relationship (no spouse),
// a separate synthetic family is created for the spouse-less children instead of merging
// them into the marriage family.
func TestReconstructFamilies_SingleParentWithExistingFamily(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-father": {
					Properties: map[string]any{
						PersonPropertyGender: "male",
					},
				},
				"person-mother": {
					Properties: map[string]any{
						PersonPropertyGender: "female",
					},
				},
				"person-child-married": {
					Properties: map[string]any{},
				},
				"person-child-other1": {
					Properties: map[string]any{},
				},
				"person-child-other2": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-marriage": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleSpouse},
						{Person: "person-mother", Role: ParticipantRoleSpouse},
					},
				},
				// Child from the marriage
				"rel-pc-married-f": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child-married", Role: ParticipantRoleChild},
					},
				},
				"rel-pc-married-m": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-mother", Role: ParticipantRoleParent},
						{Person: "person-child-married", Role: ParticipantRoleChild},
					},
				},
				// Children from another relationship (father only, no mother)
				"rel-pc-other1": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child-other1", Role: ParticipantRoleChild},
					},
				},
				"rel-pc-other2": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child-other2", Role: ParticipantRoleChild},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-father":        "@I1@",
			"person-mother":        "@I2@",
			"person-child-married": "@I3@",
			"person-child-other1":  "@I4@",
			"person-child-other2":  "@I5@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:        make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	// Should create TWO families: one for the marriage, one synthetic for father-only children
	require.Len(t, expCtx.Families, 2, "expected 2 families: 1 marriage + 1 synthetic single-parent")

	// Find the marriage family and the synthetic family
	var marriageFamily, syntheticFamily *ExportFamily
	for _, f := range expCtx.Families {
		if f.HusbandID != "" && f.WifeID != "" {
			marriageFamily = f
		} else if f.WifeID == "" && f.HusbandID == "person-father" {
			syntheticFamily = f
		}
	}

	require.NotNil(t, marriageFamily, "marriage family not found")
	require.NotNil(t, syntheticFamily, "synthetic single-parent family not found")

	// Marriage family should have the child from the marriage
	assert.Contains(t, marriageFamily.ChildIDs, "person-child-married")
	assert.NotContains(t, marriageFamily.ChildIDs, "person-child-other1")
	assert.NotContains(t, marriageFamily.ChildIDs, "person-child-other2")

	// Synthetic family should have the father-only children
	assert.Contains(t, syntheticFamily.ChildIDs, "person-child-other1")
	assert.Contains(t, syntheticFamily.ChildIDs, "person-child-other2")
	assert.Equal(t, "", syntheticFamily.RelationshipID)
}

func TestReconstructFamilies_PedigreeTypes(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-father": {
					Properties: map[string]any{
						PersonPropertyGender: "male",
					},
				},
				"person-mother": {
					Properties: map[string]any{
						PersonPropertyGender: "female",
					},
				},
				"person-bio-child": {
					Properties: map[string]any{},
				},
				"person-adopted-child": {
					Properties: map[string]any{},
				},
				"person-foster-child": {
					Properties: map[string]any{},
				},
				"person-generic-child": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-marriage": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleSpouse},
						{Person: "person-mother", Role: ParticipantRoleSpouse},
					},
				},
				"rel-bio": {
					Type: RelationshipTypeBiologicalParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-bio-child", Role: ParticipantRoleChild},
					},
				},
				"rel-adopted": {
					Type: RelationshipTypeAdoptiveParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-adopted-child", Role: ParticipantRoleChild},
					},
				},
				"rel-foster": {
					Type: RelationshipTypeFosterParentChild,
					Participants: []Participant{
						{Person: "person-mother", Role: ParticipantRoleParent},
						{Person: "person-foster-child", Role: ParticipantRoleChild},
					},
				},
				"rel-generic": {
					Type: RelationshipTypeParentChild,
					Participants: []Participant{
						{Person: "person-mother", Role: ParticipantRoleParent},
						{Person: "person-generic-child", Role: ParticipantRoleChild},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-father":        "@I1@",
			"person-mother":        "@I2@",
			"person-bio-child":     "@I3@",
			"person-adopted-child": "@I4@",
			"person-foster-child":  "@I5@",
			"person-generic-child": "@I6@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	require.Len(t, expCtx.Families, 1)

	family := expCtx.Families[0]
	assert.Equal(t, "birth", family.ChildPedigrees["person-bio-child"])
	assert.Equal(t, "adopted", family.ChildPedigrees["person-adopted-child"])
	assert.Equal(t, "foster", family.ChildPedigrees["person-foster-child"])
	assert.Equal(t, "", family.ChildPedigrees["person-generic-child"])
}

// ============================================================================
// exportFamily tests
// ============================================================================

func TestExportFamily_Basic(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-husband": {
					Properties: map[string]any{PersonPropertyGender: "male"},
				},
				"person-wife": {
					Properties: map[string]any{PersonPropertyGender: "female"},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-husband", Role: ParticipantRoleSpouse},
						{Person: "person-wife", Role: ParticipantRoleSpouse},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-husband": "@I1@",
			"person-wife":    "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	family := &ExportFamily{
		FamilyXRef:     "@F1@",
		HusbandID:      "person-husband",
		WifeID:         "person-wife",
		ChildPedigrees: make(map[string]string),
		RelationshipID: "rel-1",
	}

	record := exportFamily(family, expCtx)

	assert.Equal(t, "@F1@", record.XRef)
	assert.Equal(t, GedcomTagFam, record.Tag)

	// Check HUSB and WIFE
	var foundHusb, foundWife bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagHusb {
			foundHusb = true
			assert.Equal(t, "@I1@", sub.Value)
		}
		if sub.Tag == GedcomTagWife {
			foundWife = true
			assert.Equal(t, "@I2@", sub.Value)
		}
	}
	assert.True(t, foundHusb, "missing HUSB")
	assert.True(t, foundWife, "missing WIFE")
}

func TestExportFamily_WithEvents(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-h": {
					Properties: map[string]any{PersonPropertyGender: "male"},
				},
				"person-w": {
					Properties: map[string]any{PersonPropertyGender: "female"},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
					StartEvent: "event-marriage",
					EndEvent:   "event-divorce",
				},
			},
			Events: map[string]*Event{
				"event-marriage": {
					Type: EventTypeMarriage,
					Date: "1950-06-15",
				},
				"event-divorce": {
					Type: EventTypeDivorce,
					Date: "1970-03-01",
				},
			},
			Places: make(map[string]*Place),
		},
		PersonXRefMap: map[string]string{
			"person-h": "@I1@",
			"person-w": "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				EventTypeMarriage: GedcomTagMarr,
				EventTypeDivorce:  GedcomTagDiv,
			},
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	family := &ExportFamily{
		FamilyXRef:     "@F1@",
		HusbandID:      "person-h",
		WifeID:         "person-w",
		ChildPedigrees: make(map[string]string),
		RelationshipID: "rel-1",
	}

	record := exportFamily(family, expCtx)

	// Check for MARR and DIV subrecords
	var foundMarr, foundDiv bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagMarr {
			foundMarr = true
			// Should have DATE
			var foundDate bool
			for _, marrSub := range sub.SubRecords {
				if marrSub.Tag == GedcomTagDate {
					foundDate = true
					assert.Equal(t, "15 JUN 1950", marrSub.Value)
				}
			}
			assert.True(t, foundDate, "MARR missing DATE")
		}
		if sub.Tag == GedcomTagDiv {
			foundDiv = true
			var foundDate bool
			for _, divSub := range sub.SubRecords {
				if divSub.Tag == GedcomTagDate {
					foundDate = true
					assert.Equal(t, "1 MAR 1970", divSub.Value)
				}
			}
			assert.True(t, foundDate, "DIV missing DATE")
		}
	}
	assert.True(t, foundMarr, "missing MARR record")
	assert.True(t, foundDiv, "missing DIV record")
}

func TestExportFamily_WithChildren(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-h": {
					Properties: map[string]any{PersonPropertyGender: "male"},
				},
				"person-w": {
					Properties: map[string]any{PersonPropertyGender: "female"},
				},
				"person-c1": {
					Properties: map[string]any{},
				},
				"person-c2": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-h":  "@I1@",
			"person-w":  "@I2@",
			"person-c1": "@I3@",
			"person-c2": "@I4@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	family := &ExportFamily{
		FamilyXRef:     "@F1@",
		HusbandID:      "person-h",
		WifeID:         "person-w",
		ChildIDs:       []string{"person-c1", "person-c2"},
		ChildPedigrees: make(map[string]string),
		RelationshipID: "rel-1",
	}

	record := exportFamily(family, expCtx)

	// Count CHIL subrecords
	var childRefs []string
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagChil {
			childRefs = append(childRefs, sub.Value)
		}
	}
	assert.Len(t, childRefs, 2)
	assert.Contains(t, childRefs, "@I3@")
	assert.Contains(t, childRefs, "@I4@")
}

// ============================================================================
// exportPerson FAMS/FAMC tests
// ============================================================================

func TestExportPerson_FAMS_FAMC(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events:   make(map[string]*Event),
			Citations: make(map[string]*Citation),
		},
		PersonXRefMap: map[string]string{
			"person-father": "@I1@",
			"person-child":  "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents: make(map[string][]string),
		// FAMS: person-father is a spouse in @F1@
		PersonSpouseFamilies: map[string][]string{
			"person-father": {"@F1@"},
		},
		// FAMC: person-child is a child in @F1@ with "birth" pedigree
		PersonChildFamilies: map[string][]childFamilyRef{
			"person-child": {
				{FamilyXRef: "@F1@", Pedigree: "birth"},
			},
		},
		Stats: ExportStatistics{},
	}

	// Test FAMS on the father
	father := &Person{
		Properties: map[string]any{
			PersonPropertyName: map[string]any{
				"value": "John Smith",
				"fields": map[string]any{
					"given":   "John",
					"surname": "Smith",
				},
			},
			PersonPropertyGender: "male",
		},
	}

	fatherRecord := exportPerson("person-father", father, expCtx)

	var foundFams bool
	for _, sub := range fatherRecord.SubRecords {
		if sub.Tag == GedcomTagFams {
			foundFams = true
			assert.Equal(t, "@F1@", sub.Value)
		}
	}
	assert.True(t, foundFams, "missing FAMS on father")

	// Test FAMC on the child
	child := &Person{
		Properties: map[string]any{
			PersonPropertyName: map[string]any{
				"value": "Junior Smith",
				"fields": map[string]any{
					"given":   "Junior",
					"surname": "Smith",
				},
			},
		},
	}

	childRecord := exportPerson("person-child", child, expCtx)

	var foundFamc bool
	for _, sub := range childRecord.SubRecords {
		if sub.Tag == GedcomTagFamc {
			foundFamc = true
			assert.Equal(t, "@F1@", sub.Value)
			// Should have PEDI subrecord
			require.Len(t, sub.SubRecords, 1)
			assert.Equal(t, GedcomTagPedi, sub.SubRecords[0].Tag)
			assert.Equal(t, "birth", sub.SubRecords[0].Value)
		}
	}
	assert.True(t, foundFamc, "missing FAMC on child")
}

func TestExportPerson_FAMC_NoPedigree(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events:   make(map[string]*Event),
			Citations: make(map[string]*Citation),
		},
		PersonXRefMap: map[string]string{
			"person-child": "@I1@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:       make(map[string]string),
			PersonProperties: make(map[string]string),
			EventProperties:  make(map[string]string),
		},
		PersonEvents:         make(map[string][]string),
		PersonSpouseFamilies: make(map[string][]string),
		PersonChildFamilies: map[string][]childFamilyRef{
			"person-child": {
				{FamilyXRef: "@F1@", Pedigree: ""},
			},
		},
		Stats: ExportStatistics{},
	}

	child := &Person{
		Properties: map[string]any{
			PersonPropertyName: map[string]any{
				"value": "Jane Smith",
				"fields": map[string]any{
					"given":   "Jane",
					"surname": "Smith",
				},
			},
		},
	}

	record := exportPerson("person-child", child, expCtx)

	var foundFamc bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagFamc {
			foundFamc = true
			assert.Equal(t, "@F1@", sub.Value)
			// No PEDI subrecord for generic parent-child
			assert.Empty(t, sub.SubRecords)
		}
	}
	assert.True(t, foundFamc, "missing FAMC on child")
}

// ============================================================================
// End-to-end test
// ============================================================================

func TestExportGEDCOM_FullFamily(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-father": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value": "John Smith",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Smith",
						},
					},
					PersonPropertyGender: "male",
				},
			},
			"person-mother": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value": "Jane Doe",
						"fields": map[string]any{
							"given":   "Jane",
							"surname": "Doe",
						},
					},
					PersonPropertyGender: "female",
				},
			},
			"person-child": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value": "Junior Smith",
						"fields": map[string]any{
							"given":   "Junior",
							"surname": "Smith",
						},
					},
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-marriage": {
				Type: RelationshipTypeMarriage,
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleSpouse},
					{Person: "person-mother", Role: ParticipantRoleSpouse},
				},
				StartEvent: "event-marriage",
			},
			"rel-parent-child": {
				Type: RelationshipTypeBiologicalParentChild,
				Participants: []Participant{
					{Person: "person-father", Role: ParticipantRoleParent},
					{Person: "person-child", Role: ParticipantRoleChild},
				},
			},
		},
		Events: map[string]*Event{
			"event-marriage": {
				Type: EventTypeMarriage,
				Date: "1950-06-15",
			},
		},
		Places:       make(map[string]*Place),
		Sources:      make(map[string]*Source),
		Repositories: make(map[string]*Repository),
		Media:        make(map[string]*Media),
		Citations:    make(map[string]*Citation),
		Assertions:   make(map[string]*Assertion),
	}

	data, result, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)
	require.NotNil(t, result)

	output := string(data)

	// Verify persons exported
	assert.Equal(t, 3, result.Statistics.PersonsExported)

	// Verify family exported
	assert.Equal(t, 1, result.Statistics.FamiliesExported)

	// Should have FAM record
	assert.Contains(t, output, "@F1@ FAM")

	// Should have HUSB and WIFE in FAM
	// Person IDs are sorted, so person-child=@I1@, person-father=@I2@, person-mother=@I3@
	assert.Contains(t, output, "HUSB @I2@")  // person-father
	assert.Contains(t, output, "WIFE @I3@")  // person-mother

	// Should have CHIL
	assert.Contains(t, output, "CHIL @I1@")  // person-child

	// Should have MARR with DATE
	assert.Contains(t, output, "MARR")
	assert.Contains(t, output, "15 JUN 1950")

	// INDI records should have FAMS/FAMC back-references
	assert.Contains(t, output, "FAMS @F1@")
	assert.Contains(t, output, "FAMC @F1@")

	// Child should have PEDI
	assert.Contains(t, output, "PEDI birth")

	// Should have HEAD and TRLR
	assert.Contains(t, output, "0 HEAD")
	assert.Contains(t, output, "0 TRLR")

	// Verify record ordering: HEAD, INDI records, FAM records, TRLR
	headIdx := strings.Index(output, "0 HEAD")
	famIdx := strings.Index(output, "0 @F1@ FAM")
	trlrIdx := strings.Index(output, "0 TRLR")

	assert.True(t, headIdx < famIdx, "HEAD should come before FAM")
	assert.True(t, famIdx < trlrIdx, "FAM should come before TRLR")
}

func TestExportGEDCOM_FamilyWithNotes(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{
			"person-h": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value": "Husband",
						"fields": map[string]any{
							"given": "Husband",
						},
					},
					PersonPropertyGender: "male",
				},
			},
			"person-w": {
				Properties: map[string]any{
					PersonPropertyName: map[string]any{
						"value": "Wife",
						"fields": map[string]any{
							"given": "Wife",
						},
					},
					PersonPropertyGender: "female",
				},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: RelationshipTypeMarriage,
				Participants: []Participant{
					{Person: "person-h", Role: ParticipantRoleSpouse},
					{Person: "person-w", Role: ParticipantRoleSpouse},
				},
				Notes: "Married at the local church",
			},
		},
		Events:       make(map[string]*Event),
		Places:       make(map[string]*Place),
		Sources:      make(map[string]*Source),
		Repositories: make(map[string]*Repository),
		Media:        make(map[string]*Media),
		Citations:    make(map[string]*Citation),
		Assertions:   make(map[string]*Assertion),
	}

	data, _, err := ExportGEDCOM(glx, GEDCOM551, nil)
	require.NoError(t, err)

	output := string(data)
	assert.Contains(t, output, "NOTE Married at the local church")
}

func TestExportFamily_WithFamilyEvents(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-h": {
					Properties: map[string]any{PersonPropertyGender: "male"},
				},
				"person-w": {
					Properties: map[string]any{PersonPropertyGender: "female"},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
					StartEvent: "event-marr",
				},
			},
			Events: map[string]*Event{
				"event-marr": {
					Type: EventTypeMarriage,
					Date: "1950-06-15",
				},
				"event-enga": {
					Type: EventTypeEngagement,
					Date: "1949-12-25",
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
				},
			},
			Places: make(map[string]*Place),
		},
		PersonXRefMap: map[string]string{
			"person-h": "@I1@",
			"person-w": "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				EventTypeMarriage:    GedcomTagMarr,
				EventTypeEngagement:  GedcomTagEnga,
			},
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	family := &ExportFamily{
		FamilyXRef:     "@F1@",
		HusbandID:      "person-h",
		WifeID:         "person-w",
		ChildPedigrees: make(map[string]string),
		RelationshipID: "rel-1",
	}

	record := exportFamily(family, expCtx)

	var foundMarr, foundEnga bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagMarr {
			foundMarr = true
		}
		if sub.Tag == GedcomTagEnga {
			foundEnga = true
			// Check date
			for _, engaSub := range sub.SubRecords {
				if engaSub.Tag == GedcomTagDate {
					assert.Equal(t, "25 DEC 1949", engaSub.Value)
				}
			}
		}
	}
	assert.True(t, foundMarr, "missing MARR event")
	assert.True(t, foundEnga, "missing ENGA family event")
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestAssignHusbandWife_MaleFemale(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"a": {Properties: map[string]any{PersonPropertyGender: "male"}},
				"b": {Properties: map[string]any{PersonPropertyGender: "female"}},
			},
		},
	}

	h, w := assignHusbandWife("a", "b", expCtx)
	assert.Equal(t, "a", h)
	assert.Equal(t, "b", w)
}

func TestAssignHusbandWife_FemaleFirst(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"a": {Properties: map[string]any{PersonPropertyGender: "female"}},
				"b": {Properties: map[string]any{PersonPropertyGender: "male"}},
			},
		},
	}

	h, w := assignHusbandWife("a", "b", expCtx)
	assert.Equal(t, "b", h) // male
	assert.Equal(t, "a", w) // female
}

func TestAssignHusbandWife_SameGender(t *testing.T) {
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"a": {Properties: map[string]any{PersonPropertyGender: "male"}},
				"b": {Properties: map[string]any{PersonPropertyGender: "male"}},
			},
		},
	}

	h, w := assignHusbandWife("a", "b", expCtx)
	assert.Equal(t, "a", h) // first → HUSB
	assert.Equal(t, "b", w) // second → WIFE
}

func TestRelationshipTypeToPedi(t *testing.T) {
	assert.Equal(t, "birth", relationshipTypeToPedi(RelationshipTypeBiologicalParentChild))
	assert.Equal(t, "adopted", relationshipTypeToPedi(RelationshipTypeAdoptiveParentChild))
	assert.Equal(t, "foster", relationshipTypeToPedi(RelationshipTypeFosterParentChild))
	assert.Equal(t, "", relationshipTypeToPedi(RelationshipTypeParentChild))
	assert.Equal(t, "", relationshipTypeToPedi(RelationshipTypeMarriage))
}

func TestIsParentChildType(t *testing.T) {
	assert.True(t, isParentChildType(RelationshipTypeParentChild))
	assert.True(t, isParentChildType(RelationshipTypeBiologicalParentChild))
	assert.True(t, isParentChildType(RelationshipTypeAdoptiveParentChild))
	assert.True(t, isParentChildType(RelationshipTypeFosterParentChild))
	assert.False(t, isParentChildType(RelationshipTypeMarriage))
	assert.False(t, isParentChildType(RelationshipTypeSibling))
}

func TestExportFamily_MarriageWithoutStartEvent_NoMarr(t *testing.T) {
	// A marriage relationship without a StartEvent should NOT emit MARR,
	// because we can't distinguish "FAM without MARR" from "FAM with empty MARR"
	// after import. The conservative choice avoids inflating MARR counts.
	glxFile := &GLXFile{
		Persons: map[string]*Person{
			"person-1": {Properties: map[string]any{"gender": "male", "name": map[string]any{"value": "John"}}},
			"person-2": {Properties: map[string]any{"gender": "female", "name": map[string]any{"value": "Jane"}}},
		},
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: RelationshipTypeMarriage,
				Participants: []Participant{
					{Person: "person-1", Role: ParticipantRoleSpouse},
					{Person: "person-2", Role: ParticipantRoleSpouse},
				},
			},
		},
		Events:            make(map[string]*Event),
		EventTypes:        make(map[string]*EventType),
		PersonProperties:  make(map[string]*PropertyDefinition),
		RelationshipTypes: make(map[string]*RelationshipType),
		Sources:           make(map[string]*Source),
		Citations:         make(map[string]*Citation),
		Repositories:      make(map[string]*Repository),
		Media:             make(map[string]*Media),
		Assertions:        make(map[string]*Assertion),
	}

	if err := LoadStandardVocabulariesIntoGLX(glxFile); err != nil {
		t.Fatal(err)
	}

	expCtx := &ExportContext{
		GLX:                      glxFile,
		Version:                  GEDCOM551,
		Logger:                   NewImportLogger(nil),
		ExportIndex:              buildExportIndex(glxFile),
		PersonXRefMap:            map[string]string{"person-1": "@I1@", "person-2": "@I2@"},
		SourceXRefMap:            make(map[string]string),
		RepositoryXRefMap:        make(map[string]string),
		MediaXRefMap:             make(map[string]string),
		PlaceStrings:             make(map[string]string),
		PersonEvents:             make(map[string][]string),
		PersonSpouseFamilies:     make(map[string][]string),
		PersonChildFamilies:      make(map[string][]childFamilyRef),
		PersonPropertyAssertions: make(map[string]map[string][]*Assertion),
		Families:                 []*ExportFamily{},
		FamilyXRefMap:            make(map[string]string),
	}

	reconstructFamilies(expCtx)
	require.Len(t, expCtx.Families, 1)

	record := exportFamily(expCtx.Families[0], expCtx)

	for _, sub := range record.SubRecords {
		assert.NotEqual(t, GedcomTagMarr, sub.Tag,
			"Should NOT emit MARR for marriage relationship without StartEvent")
	}
}

// ============================================================================
// Marriage TYPE export tests
// ============================================================================

func TestExportFamilyEvent_MarriageType(t *testing.T) {
	// marriage_type property should be exported as TYPE sub-record on MARR
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: map[string]*Event{
				"event-marr": {
					Type: EventTypeMarriage,
					Date: "1950-06-15",
					Properties: map[string]any{
						PropertyMarriageType: "civil",
					},
				},
			},
		},
		ExportIndex: &ExportIndex{
			EventTypes:      make(map[string]string),
			EventProperties: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
	}

	record := exportFamilyEvent("event-marr", GedcomTagMarr, expCtx)
	require.NotNil(t, record)

	var foundType bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagType {
			foundType = true
			assert.Equal(t, "civil", sub.Value)
		}
	}
	assert.True(t, foundType, "MARR should have TYPE sub-record from marriage_type property")
}

func TestExportFamilyEvent_EventSubtype(t *testing.T) {
	// event_subtype property should be exported as TYPE via exportEventPropertySubrecords
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Events: map[string]*Event{
				"event-even": {
					Type: EventTypeGeneric,
					Date: "2019",
					Properties: map[string]any{
						"event_subtype": "separation",
					},
				},
			},
		},
		ExportIndex: &ExportIndex{
			EventTypes: make(map[string]string),
			EventProperties: map[string]string{
				"event_subtype": GedcomTagType,
			},
		},
		PlaceStrings: make(map[string]string),
	}

	record := exportFamilyEvent("event-even", GedcomTagEven, expCtx)
	require.NotNil(t, record)

	var foundType bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagType {
			foundType = true
			assert.Equal(t, "separation", sub.Value)
		}
	}
	assert.True(t, foundType, "family event should have TYPE sub-record from event_subtype")
}

func TestExportFamily_FamilyEventsPreserveEventProperties(t *testing.T) {
	// findFamilyEvents should include event properties (TYPE, CAUS, etc.)
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-h": {Properties: map[string]any{PersonPropertyGender: "male"}},
				"person-w": {Properties: map[string]any{PersonPropertyGender: "female"}},
			},
			Relationships: map[string]*Relationship{
				"rel-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
					StartEvent: "event-marr",
				},
			},
			Events: map[string]*Event{
				"event-marr": {
					Type: EventTypeMarriage,
					Date: "1950-06-15",
				},
				"event-separation": {
					Type: EventTypeGeneric,
					Date: "1965",
					Properties: map[string]any{
						"event_subtype": "separation",
					},
					Participants: []Participant{
						{Person: "person-h", Role: ParticipantRoleSpouse},
						{Person: "person-w", Role: ParticipantRoleSpouse},
					},
				},
			},
			Places: make(map[string]*Place),
		},
		PersonXRefMap: map[string]string{
			"person-h": "@I1@",
			"person-w": "@I2@",
		},
		ExportIndex: &ExportIndex{
			EventTypes: map[string]string{
				EventTypeMarriage: GedcomTagMarr,
				EventTypeGeneric:  GedcomTagEven,
			},
			EventProperties: map[string]string{
				"event_subtype": GedcomTagType,
			},
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	family := &ExportFamily{
		FamilyXRef:     "@F1@",
		HusbandID:      "person-h",
		WifeID:         "person-w",
		ChildPedigrees: make(map[string]string),
		RelationshipID: "rel-1",
	}

	record := exportFamily(family, expCtx)

	var foundEven bool
	for _, sub := range record.SubRecords {
		if sub.Tag == GedcomTagEven {
			foundEven = true
			var foundType bool
			for _, evenSub := range sub.SubRecords {
				if evenSub.Tag == GedcomTagType {
					foundType = true
					assert.Equal(t, "separation", evenSub.Value)
				}
			}
			assert.True(t, foundType, "EVEN family event should have TYPE sub-record")
		}
	}
	assert.True(t, foundEven, "family should have EVEN event for separation")
}

func TestReconstructFamilies_MultipleSingleSpouseMarriages(t *testing.T) {
	// A person has two separate single-spouse marriages, each with a child.
	// Both families should be created and children should be in the correct family.
	expCtx := &ExportContext{
		GLX: &GLXFile{
			Persons: map[string]*Person{
				"person-father": {
					Properties: map[string]any{
						PersonPropertyGender: "male",
					},
				},
				"person-child-a": {
					Properties: map[string]any{},
				},
				"person-child-b": {
					Properties: map[string]any{},
				},
			},
			Relationships: map[string]*Relationship{
				"rel-marriage-1": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleSpouse},
					},
				},
				"rel-marriage-2": {
					Type: RelationshipTypeMarriage,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleSpouse},
					},
				},
				"rel-parent-child-a": {
					Type: RelationshipTypeBiologicalParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child-a", Role: ParticipantRoleChild},
					},
				},
				"rel-parent-child-b": {
					Type: RelationshipTypeBiologicalParentChild,
					Participants: []Participant{
						{Person: "person-father", Role: ParticipantRoleParent},
						{Person: "person-child-b", Role: ParticipantRoleChild},
					},
				},
			},
			Events: make(map[string]*Event),
		},
		PersonXRefMap: map[string]string{
			"person-father":  "@I1@",
			"person-child-a": "@I2@",
			"person-child-b": "@I3@",
		},
		ExportIndex: &ExportIndex{
			EventTypes:        make(map[string]string),
			RelationshipTypes: make(map[string]string),
		},
		PlaceStrings: make(map[string]string),
		Stats:        ExportStatistics{},
	}

	reconstructFamilies(expCtx)

	require.Len(t, expCtx.Families, 2, "should create two separate families")

	// Both families should have the father as husband
	for _, fam := range expCtx.Families {
		assert.Equal(t, "person-father", fam.HusbandID)
		assert.Empty(t, fam.WifeID)
	}

	// Each child should be in some family (not lost due to pair-key overwrite)
	allChildren := make(map[string]bool)
	for _, fam := range expCtx.Families {
		for _, cid := range fam.ChildIDs {
			allChildren[cid] = true
		}
	}
	assert.True(t, allChildren["person-child-a"],
		"child-a should be placed in a family")
	assert.True(t, allChildren["person-child-b"],
		"child-b should be placed in a family")
}
