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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// littleFamilyArchive models the Lewis Little family from #1325, #1326,
// #1327, #1331 and #208:
//
//   - Lewis's parents: Daniel and Annie (low confidence, unresearched), plus
//     two candidate fathers the archive has disproven (Jacob, John);
//   - Lewis married the widow Elizabeth Starr, mother of Peggy and Rachel Call
//     by Adam Call; Lewis is their step-parent (step_parent relationship);
//   - Jasper is Lewis's son by his first wife, Nancy his daughter by Elizabeth;
//   - Lewis is Rachel's guardian and Adam Little's godparent, and Ned's
//     neighbor.
func littleFamilyArchive() *glxlib.GLXFile {
	person := func(name, sex string) *glxlib.Person {
		return &glxlib.Person{Properties: map[string]any{"name": map[string]any{"value": name}, "sex": sex}}
	}
	pc := func(relType string, participants ...glxlib.Participant) *glxlib.Relationship {
		return &glxlib.Relationship{Type: relType, Participants: participants}
	}
	parent := func(id string) glxlib.Participant { return glxlib.Participant{Person: id, Role: "parent"} }
	child := func(id string) glxlib.Participant { return glxlib.Participant{Person: id, Role: "child"} }

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-lewis":   person("Lewis Little", "male"),
			"person-rachel":  person("Rachel Call", "female"),
			"person-peggy":   person("Peggy Call", "female"),
			"person-adam":    person("Adam Little", "male"),
			"person-call":    person("Adam Call", "male"),
			"person-starr":   person("Elizabeth Starr", "female"),
			"person-first":   person("Mary Smith", "female"),
			"person-jasper":  person("Jasper Little", "male"),
			"person-nancy":   person("Nancy Little", "female"),
			"person-daniel":  person("Johannes Daniel Little", "male"),
			"person-annie":   person("Annie Mary Lewis", "female"),
			"person-jacob":   person("Jacob Little", "male"),
			"person-john":    person("John Little", "male"),
			"person-ned":     person("Ned Neighbor", "male"),
			"person-ezekiel": person("Ezekiel Little", "male"),
		},
		Events: map[string]*glxlib.Event{
			"ev-death-rachel": {Type: "death", Date: "1843-12", Participants: []glxlib.Participant{{Person: "person-rachel", Role: "principal"}}},
			"ev-death-jacob":  {Type: "death", Date: "1800", Participants: []glxlib.Participant{{Person: "person-jacob", Role: "principal"}}},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-guardian": pc("guardian",
				glxlib.Participant{Person: "person-lewis", Role: "guardian"},
				glxlib.Participant{Person: "person-rachel", Role: "ward"}),
			"rel-godparent": pc("godparent",
				glxlib.Participant{Person: "person-lewis", Role: "godparent"},
				glxlib.Participant{Person: "person-adam", Role: "godchild"}),
			"rel-neighbor": pc("neighbor",
				glxlib.Participant{Person: "person-lewis", Role: "associate"},
				glxlib.Participant{Person: "person-ned", Role: "associate"}),
			"rel-pc-call":    pc("parent_child", parent("person-call"), parent("person-starr"), child("person-rachel"), child("person-peggy")),
			"rel-step":       pc("step_parent", parent("person-lewis"), child("person-rachel"), child("person-peggy")),
			"rel-pc-first":   pc("parent_child", parent("person-lewis"), parent("person-first"), child("person-jasper")),
			"rel-pc-starr":   pc("parent_child", parent("person-lewis"), parent("person-starr"), child("person-nancy")),
			"rel-pc-daniel":  pc("parent_child", parent("person-daniel"), parent("person-annie"), child("person-lewis")),
			"rel-pc-jacob":   pc("parent_child", parent("person-jacob"), child("person-lewis")),
			"rel-pc-john":    pc("parent_child", parent("person-john"), child("person-lewis")),
			"rel-marriage":   pc("marriage", glxlib.Participant{Person: "person-lewis", Role: "spouse"}, glxlib.Participant{Person: "person-starr", Role: "spouse"}),
			"rel-marriage-x": pc("marriage", glxlib.Participant{Person: "person-lewis", Role: "spouse"}, glxlib.Participant{Person: "person-first", Role: "spouse"}),
			"rel-neighbor-x": pc("neighbor",
				glxlib.Participant{Person: "person-lewis", Role: "associate"},
				glxlib.Participant{Person: "person-ezekiel", Role: "associate"}),
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-daniel":     {Subject: glxlib.EntityRef{Relationship: "rel-pc-daniel"}, Confidence: "low", Status: "unresearched"},
			"a-jacob":      {Subject: glxlib.EntityRef{Relationship: "rel-pc-jacob"}, Confidence: "high", Status: "disproven"},
			"a-john":       {Subject: glxlib.EntityRef{Relationship: "rel-pc-john"}, Confidence: "medium", Status: "disproven"},
			"a-marriage-x": {Subject: glxlib.EntityRef{Relationship: "rel-marriage-x"}, Confidence: "high", Status: "speculative"},
			"a-neighbor-x": {Subject: glxlib.EntityRef{Relationship: "rel-neighbor-x"}, Status: "disproven"},
		},
	}
}

func TestFamilyLinks_ParentsChildrenExcluded(t *testing.T) {
	fam := newFamilyLinks(littleFamilyArchive())

	assert.Equal(t, []familyEdge{
		{PersonID: "person-daniel", Hypothetical: true},
		{PersonID: "person-annie", Hypothetical: true},
	}, fam.parents("person-lewis"), "disproven fathers are not parents")
	assert.Equal(t, []string{"person-jacob", "person-john"}, fam.excludedParents("person-lewis"))

	assert.Equal(t, []familyEdge{
		{PersonID: "person-call"},
		{PersonID: "person-starr"},
		{PersonID: "person-lewis", Step: true},
	}, fam.parents("person-rachel"))

	own, step := splitStep(fam.children("person-lewis"))
	assert.Equal(t, []string{"person-jasper", "person-nancy"}, edgeIDs(own))
	assert.Equal(t, []string{"person-peggy", "person-rachel"}, edgeIDs(step))
}

func TestFamilyLinks_SiblingKinds(t *testing.T) {
	fam := newFamilyLinks(littleFamilyArchive())

	kinds := map[string]string{}
	for _, s := range fam.siblings("person-rachel") {
		kinds[s.PersonID] = s.Kind
	}
	assert.Equal(t, map[string]string{
		"person-peggy":  siblingKindFull, // same two parents
		"person-nancy":  siblingKindHalf, // shares Elizabeth only
		"person-jasper": siblingKindStep, // linked only through stepfather Lewis
	}, kinds)

	// Nancy's only other recorded parent is shared with Jasper: Lewis. Each has
	// a mother the other lacks, so they are half-siblings.
	for _, s := range fam.siblings("person-nancy") {
		if s.PersonID == "person-jasper" {
			assert.Equal(t, siblingKindHalf, s.Kind)
		}
	}
}

func TestPrintFamilySection_StepHalfAndExcluded(t *testing.T) {
	archive := littleFamilyArchive()

	rachel := captureStdout(t, func() { printFamilySection("person-rachel", archive) })
	assert.Contains(t, rachel, "Father:           Adam Call\n")
	assert.Contains(t, rachel, "Mother:           Elizabeth Starr\n")
	assert.Contains(t, rachel, "Stepfather:       Lewis Little\n")
	assert.NotContains(t, rachel, "Father:           Lewis Little")
	assert.Contains(t, rachel, "Siblings:         Peggy Call\n")
	assert.Contains(t, rachel, "Half-siblings:    Nancy Little\n")
	assert.Contains(t, rachel, "Step-siblings:    Jasper Little\n")

	lewis := captureStdout(t, func() { printFamilySection("person-lewis", archive) })
	assert.Contains(t, lewis, "Father:           Johannes Daniel Little  (?)\n")
	assert.Contains(t, lewis, "Mother:           Annie Mary Lewis  (?)\n")
	assert.Contains(t, lewis, "Excluded:         Jacob Little, John Little  (disproven parent)\n")
	assert.NotContains(t, lewis, "Father:           Jacob Little")
	assert.Contains(t, lewis, "Spouse:           Mary Smith  (?)\n", "speculative marriage is hypothetical")
}

func TestPrintOtherRelationships_LabelsByOtherRole(t *testing.T) {
	archive := littleFamilyArchive()

	lewis := captureStdout(t, func() { printOtherRelationshipsSection("person-lewis", archive) })
	assert.Contains(t, lewis, "Godchild:         Adam Little\n")
	assert.Contains(t, lewis, "Ward:             Rachel Call\n")
	assert.Contains(t, lewis, "Neighbor:         Ned Neighbor\n", "symmetric roles keep the type label")
	assert.NotContains(t, lewis, "Ezekiel", "disproven relationship is left out")

	rachel := captureStdout(t, func() { printOtherRelationshipsSection("person-rachel", archive) })
	assert.Contains(t, rachel, "Guardian:         Lewis Little\n")

	adam := captureStdout(t, func() { printOtherRelationshipsSection("person-adam", archive) })
	assert.Contains(t, adam, "Godparent:        Lewis Little\n")
}

func TestOtherRelationshipLabel(t *testing.T) {
	archive := &glxlib.GLXFile{ParticipantRoles: map[string]*glxlib.VocabularyEntry{
		"enslaved_person": {Label: "Enslaved Person"},
	}}
	assert.Equal(t, "Godchild", otherRelationshipLabel("godparent", "godparent", "godchild", archive))
	assert.Equal(t, "Enslaved Person", otherRelationshipLabel("enslavement", "enslaver", "enslaved_person", archive))
	assert.Equal(t, "Associate", otherRelationshipLabel("associate", "associate", "associate", archive))
	assert.Equal(t, "Neighbor", otherRelationshipLabel("neighbor", "", "", archive))
	// Legacy guardianships recorded with parent/child roles.
	assert.Equal(t, "Ward", otherRelationshipLabel("guardian", "parent", "child", archive))
	assert.Equal(t, "Guardian", otherRelationshipLabel("guardian", "child", "parent", archive))
}

func TestGenerateLifeHistory_StepAndHypothetical(t *testing.T) {
	archive := littleFamilyArchive()

	lewis := generateLifeHistory("person-lewis", archive.Persons["person-lewis"], archive)
	assert.Contains(t, lewis, "He was the child of Johannes Daniel Little (?) and Annie Mary Lewis (?).")
	assert.NotContains(t, lewis, "Jacob")
	assert.Contains(t, lewis, "He had two children: Jasper and Nancy, and two stepdaughters, Peggy and Rachel Call.")

	rachel := generateLifeHistory("person-rachel", archive.Persons["person-rachel"], archive)
	assert.Contains(t, rachel, "She was the child of Adam Call and Elizabeth Starr. Her stepfather was Lewis Little.")
}

func TestJoinStepChildNames(t *testing.T) {
	archive := littleFamilyArchive()
	edges := []familyEdge{{PersonID: "person-peggy"}, {PersonID: "person-jasper"}}
	assert.Equal(t, "Peggy Call and Jasper Little", joinStepChildNames(edges, archive), "different surnames stay whole")
	assert.Equal(t, "stepchildren", stepChildWord(edges, archive))
	assert.Equal(t, "stepson", stepChildWord([]familyEdge{{PersonID: "person-jasper"}}, archive))
}

func TestTreeIndexes_StandingAndStep(t *testing.T) {
	tc := newTreeContext(littleFamilyArchive())

	parents := findParents(tc, "person-lewis")
	require.Len(t, parents, 2, "disproven candidate fathers are not in the tree")
	for _, p := range parents {
		assert.True(t, p.hypothetical, p.personID)
	}

	root := buildDescendantTree(tc, "person-lewis", 1, 0, map[string]bool{})
	out := captureStdout(t, func() { printTree(root, "", true) })
	assert.Contains(t, out, "Peggy Call  person-peggy  [step]")
	assert.Contains(t, out, "Jasper Little  person-jasper\n")

	anc := buildAncestorTree(tc, "person-lewis", 1, 0, map[string]bool{})
	out = captureStdout(t, func() { printTree(anc, "", true) })
	assert.Contains(t, out, "Johannes Daniel Little  person-daniel  (?)")
	assert.NotContains(t, out, "Jacob")
}

func TestTimeline_GuardianAndDisproven(t *testing.T) {
	archive := littleFamilyArchive()

	entries := collectTimelineEntries("person-lewis", archive, true)
	labels := make([]string, 0, len(entries))
	for _, e := range entries {
		labels = append(labels, e.Label)
	}
	joined := strings.Join(labels, "\n")
	assert.Contains(t, joined, "Death of ward (Rachel Call)")
	assert.NotContains(t, joined, "Death of parent (Rachel Call)")
	assert.NotContains(t, joined, "Jacob Little", "a disproven father's death is not on the timeline")
}

func TestBuildRelationshipHypotheticalIndex_Status(t *testing.T) {
	index := buildRelationshipHypotheticalIndex(littleFamilyArchive())
	assert.True(t, index["rel-pc-daniel"], "low confidence")
	assert.True(t, index["rel-marriage-x"], "high confidence but speculative status")
	assert.True(t, index["rel-pc-jacob"], "disproven is not established either")
	assert.False(t, index["rel-pc-call"])
}

func TestBuildProof_ParentageExcludesDisproven(t *testing.T) {
	archive := littleFamilyArchive()
	// The #1327 reproduction: Jacob disproven, John speculative.
	delete(archive.Relationships, "rel-pc-daniel")
	archive.Assertions["a-john"] = &glxlib.Assertion{
		Subject: glxlib.EntityRef{Relationship: "rel-pc-john"}, Sources: []string{"src"}, Confidence: "low", Status: "speculative",
	}
	archive.Assertions["a-jacob"].Sources = []string{"src"}

	result := buildProof("person-lewis", archive.Persons["person-lewis"], topicParentage, archive)
	assert.Empty(t, result.Conflicts)
	assert.Equal(t, []proofExcludedAlternative{{PersonID: "person-jacob", Name: "Jacob Little", Relationship: "rel-pc-jacob"}}, result.Excluded)
	assert.Equal(t, proofConclusionPossible, result.Conclusion)
	assert.Contains(t, result.Summary, "Parents identified: John Little.")
	assert.NotContains(t, result.Summary, "Jacob")

	streams, out, _ := TestIOStreams()
	printProofText(streams, result)
	assert.Contains(t, out.String(), "Alternatives Excluded:\n    x Jacob Little (rel-pc-jacob) -- all assertions disproven")

	streams, out, _ = TestIOStreams()
	printProofMarkdown(streams, result)
	assert.Contains(t, out.String(), "## Alternatives Excluded")
}

func TestBuildProof_ParentageCompetingFathersConflict(t *testing.T) {
	archive := littleFamilyArchive()
	// Revive John as a surviving candidate alongside Daniel.
	archive.Assertions["a-john"].Status = "speculative"

	result := buildProof("person-lewis", archive.Persons["person-lewis"], topicParentage, archive)
	require.Len(t, result.Conflicts, 1)
	c := result.Conflicts[0]
	assert.Equal(t, "father (competing parent_child relationships)", c.Property)
	assert.False(t, c.Resolved)
	require.Len(t, c.Values, 2)
	assert.Equal(t, "Johannes Daniel Little", c.Values[0].Value)
	assert.Equal(t, "John Little", c.Values[1].Value)
	assert.Equal(t, proofConclusionConflicted, result.Conclusion)
	// Annie is the only mother: no mother conflict.
	for _, conflict := range result.Conflicts {
		assert.NotContains(t, conflict.Property, "mother")
	}
}

func TestBuildProof_ParentageAdoptiveFatherIsNoConflict(t *testing.T) {
	archive := littleFamilyArchive()
	archive.Relationships["rel-adoptive"] = &glxlib.Relationship{Type: "adoptive_parent_child", Participants: []glxlib.Participant{
		{Person: "person-ned", Role: "adoptive_parent"}, {Person: "person-lewis", Role: "adopted_child"},
	}}

	result := buildProof("person-lewis", archive.Persons["person-lewis"], topicParentage, archive)
	assert.Empty(t, result.Conflicts, "an adoptive father does not compete with a birth father")
	assert.Contains(t, result.Summary, "Ned Neighbor")
}

func TestPublishFamilyGroups(t *testing.T) {
	archive := littleFamilyArchive()
	idx := newSiteIndex(archive)

	names := func(links []personLink) []string {
		out := make([]string, 0, len(links))
		for _, l := range links {
			out = append(out, l.Name)
		}

		return out
	}

	rachel := buildPersonPage("person-rachel", archive.Persons["person-rachel"], archive, idx)
	assert.Equal(t, []string{"Adam Call", "Elizabeth Starr"}, names(rachel.Parents))
	assert.Equal(t, []string{"Lewis Little"}, names(rachel.StepParents))
	assert.Equal(t, []string{"Peggy Call"}, names(rachel.Siblings))
	assert.Equal(t, []string{"Nancy Little"}, names(rachel.HalfSiblings))
	assert.Equal(t, []string{"Jasper Little"}, names(rachel.StepSiblings))
	// The pedigree chart draws birth parents only.
	require.NotNil(t, rachel.Pedigree)
	for _, n := range rachel.Pedigree.Nodes {
		assert.NotEqual(t, "person-lewis", n.ID, "a stepfather is not drawn as an ancestor")
	}

	lewis := buildPersonPage("person-lewis", archive.Persons["person-lewis"], archive, idx)
	assert.Equal(t, []string{"Annie Mary Lewis", "Johannes Daniel Little"}, names(lewis.Parents))
	for _, p := range lewis.Parents {
		assert.True(t, p.Hypothetical, p.Name)
	}
	assert.Equal(t, []string{"Jasper Little", "Nancy Little"}, names(lewis.Children))
	assert.Equal(t, []string{"Peggy Call", "Rachel Call"}, names(lewis.StepChildren))
	require.NotNil(t, lewis.Pedigree)
	chartNames := make([]string, 0, len(lewis.Pedigree.Nodes))
	for _, n := range lewis.Pedigree.Nodes {
		chartNames = append(chartNames, n.Name)
		assert.NotEqual(t, "person-jacob", n.ID, "a disproven father is not drawn")
	}
	assert.Contains(t, chartNames, "Annie Mary Lewis (?)")
}
