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
	"sort"
	"strings"

	glxlib "github.com/genealogix/glx/go-glx"
)

// hypotheticalMarker is appended to a relative's name wherever a display
// command shows a relationship the archive records only as a hypothesis.
const hypotheticalMarker = "(?)"

// Sibling kinds reported by familyLinks.siblings.
const (
	siblingKindFull = ""     // shares every known birth parent (or the data can't tell)
	siblingKindHalf = "half" // shares one birth parent; each has one the other lacks
	siblingKindStep = "step" // connected only through a step-parent link
)

// familyEdge is one relative reached through parent-child links, merged across
// every relationship that links the same two people.
type familyEdge struct {
	PersonID string
	// Step is true when every link to this relative is a step link; a person
	// who is both a recorded parent and a step-parent counts as a parent.
	Step bool
	// Hypothetical is true when no link to this relative is accepted.
	Hypothetical bool
}

// siblingEdge is a sibling with how the two are related.
type siblingEdge struct {
	PersonID     string
	Kind         string // siblingKindFull, siblingKindHalf, or siblingKindStep
	Hypothetical bool
}

// familyLinks is the parent-child graph of an archive with each link's
// standing applied: disproven links are set aside (reported only by
// excludedParents), hypothetical ones are flagged, step links are marked.
// summary, publish, serve, and the tree commands all read family structure
// through it so they agree on which links count.
type familyLinks struct {
	archive    *glxlib.GLXFile
	standings  *glxlib.RelationshipStandingIndex
	surviving  []glxlib.ParentChildLink            // non-disproven links, in relationship order
	parentsOf  map[string][]glxlib.ParentChildLink // child ID -> surviving links
	childrenOf map[string][]glxlib.ParentChildLink // parent ID -> surviving links
	excluded   map[string][]glxlib.ParentChildLink // child ID -> disproven links
}

// newFamilyLinks indexes the archive's parent-child links.
func newFamilyLinks(archive *glxlib.GLXFile) *familyLinks {
	standings := glxlib.NewRelationshipStandingIndex(archive)
	f := &familyLinks{
		archive:    archive,
		standings:  standings,
		parentsOf:  map[string][]glxlib.ParentChildLink{},
		childrenOf: map[string][]glxlib.ParentChildLink{},
		excluded:   map[string][]glxlib.ParentChildLink{},
	}
	for _, link := range glxlib.ParentChildLinks(archive, standings) {
		if link.Standing == glxlib.RelationshipStandingDisproven {
			f.excluded[link.ChildID] = append(f.excluded[link.ChildID], link)

			continue
		}
		f.surviving = append(f.surviving, link)
		f.parentsOf[link.ChildID] = append(f.parentsOf[link.ChildID], link)
		f.childrenOf[link.ParentID] = append(f.childrenOf[link.ParentID], link)
	}

	return f
}

// mergeEdges folds links into one edge per relative, in first-seen order.
// other picks the relative's ID out of each link.
func mergeEdges(links []glxlib.ParentChildLink, other func(glxlib.ParentChildLink) string) []familyEdge {
	var edges []familyEdge
	pos := map[string]int{}
	for _, link := range links {
		id := other(link)
		hypo := link.Standing == glxlib.RelationshipStandingHypothetical
		if i, ok := pos[id]; ok {
			edges[i].Step = edges[i].Step && link.Step
			edges[i].Hypothetical = edges[i].Hypothetical && hypo

			continue
		}
		pos[id] = len(edges)
		edges = append(edges, familyEdge{PersonID: id, Step: link.Step, Hypothetical: hypo})
	}

	return edges
}

// parents returns the person's surviving parents (including step-parents) in
// relationship order.
func (f *familyLinks) parents(personID string) []familyEdge {
	return mergeEdges(f.parentsOf[personID], func(l glxlib.ParentChildLink) string { return l.ParentID })
}

// children returns the person's surviving children (including stepchildren),
// ordered by birth year (unknown last), then by ID.
func (f *familyLinks) children(personID string) []familyEdge {
	edges := mergeEdges(f.childrenOf[personID], func(l glxlib.ParentChildLink) string { return l.ChildID })
	sort.SliceStable(edges, func(i, j int) bool {
		yi, yj := birthYear(f.archive, edges[i].PersonID), birthYear(f.archive, edges[j].PersonID)
		if yi != yj {
			if yi == 0 {
				return false
			}
			if yj == 0 {
				return true
			}

			return yi < yj
		}

		return edges[i].PersonID < edges[j].PersonID
	})

	return edges
}

// excludedParents returns the people recorded as this person's parent only in
// relationships the archive has disproven, in relationship order. A person who
// is also a surviving parent is not listed.
func (f *familyLinks) excludedParents(personID string) []string {
	surviving := map[string]bool{}
	for _, l := range f.parentsOf[personID] {
		surviving[l.ParentID] = true
	}
	var out []string
	seen := map[string]bool{}
	for _, l := range f.excluded[personID] {
		if surviving[l.ParentID] || seen[l.ParentID] {
			continue
		}
		seen[l.ParentID] = true
		out = append(out, l.ParentID)
	}

	return out
}

// birthParentSet returns the person's surviving non-step parents.
func (f *familyLinks) birthParentSet(personID string) map[string]bool {
	set := map[string]bool{}
	for _, l := range f.parentsOf[personID] {
		if !l.Step {
			set[l.ParentID] = true
		}
	}

	return set
}

// siblings returns the person's siblings, sorted by ID, each classified as a
// full sibling, half-sibling, or step-sibling:
//
//   - two people sharing a birth (non-step) parent are half-siblings when each
//     has a birth parent the other lacks, and siblings otherwise (including
//     when the data only records the one shared parent);
//   - two people linked only through a step-parent are step-siblings;
//   - an explicit `sibling` relationship makes a sibling unless the parent
//     links already say half- or step-sibling.
//
// A sibling is hypothetical when no connection between the two is made only of
// accepted links.
func (f *familyLinks) siblings(personID string) []siblingEdge {
	cands := map[string]*siblingCandidate{}
	f.addParentSiblings(personID, cands)
	f.addExplicitSiblings(personID, cands)

	mine := f.birthParentSet(personID)
	ids := make([]string, 0, len(cands))
	for id := range cands {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	out := make([]siblingEdge, 0, len(ids))
	for _, id := range ids {
		c := cands[id]
		kind := siblingKindFull
		switch {
		case c.sharesBirthParent:
			theirs := f.birthParentSet(id)
			if hasExtra(mine, theirs) && hasExtra(theirs, mine) {
				kind = siblingKindHalf
			}
		case !c.explicit:
			kind = siblingKindStep
		}
		out = append(out, siblingEdge{PersonID: id, Kind: kind, Hypothetical: !c.accepted})
	}

	return out
}

// siblingCandidate accumulates what connects a person to a would-be sibling.
type siblingCandidate struct {
	sharesBirthParent bool // a common parent through non-step links on both sides
	accepted          bool // some connection is made only of accepted links
	explicit          bool // an explicit sibling relationship names them
}

// candidate returns the candidate for id, creating it on first use.
func candidate(cands map[string]*siblingCandidate, id string) *siblingCandidate {
	c := cands[id]
	if c == nil {
		c = &siblingCandidate{}
		cands[id] = c
	}

	return c
}

// addParentSiblings records the other children of each of the person's
// surviving parents (step-parents included).
func (f *familyLinks) addParentSiblings(personID string, cands map[string]*siblingCandidate) {
	for _, up := range f.parentsOf[personID] {
		for _, down := range f.childrenOf[up.ParentID] {
			if down.ChildID == personID {
				continue
			}
			c := candidate(cands, down.ChildID)
			if !up.Step && !down.Step {
				c.sharesBirthParent = true
			}
			if up.Standing == glxlib.RelationshipStandingAccepted && down.Standing == glxlib.RelationshipStandingAccepted {
				c.accepted = true
			}
		}
	}
}

// addExplicitSiblings records the people named with the person in a
// surviving `sibling` relationship.
func (f *familyLinks) addExplicitSiblings(personID string, cands map[string]*siblingCandidate) {
	for _, relID := range sortedKeys(f.archive.Relationships) {
		rel := f.archive.Relationships[relID]
		if rel == nil || !strings.EqualFold(rel.Type, glxlib.RelationshipTypeSibling) || !hasParticipant(personID, rel.Participants) {
			continue
		}
		standing := f.standings.Relationship(relID)
		if standing == glxlib.RelationshipStandingDisproven {
			continue
		}
		for _, p := range rel.Participants {
			if p.Person == "" || p.Person == personID {
				continue
			}
			c := candidate(cands, p.Person)
			c.explicit = true
			if standing == glxlib.RelationshipStandingAccepted {
				c.accepted = true
			}
		}
	}
}

// hasExtra reports whether a has a member that b lacks.
func hasExtra(a, b map[string]bool) bool {
	for id := range a {
		if !b[id] {
			return true
		}
	}

	return false
}

// edgeIDs returns the person IDs of edges, in order.
func edgeIDs(edges []familyEdge) []string {
	ids := make([]string, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.PersonID)
	}

	return ids
}

// splitStep partitions edges into non-step and step edges, keeping order.
func splitStep(edges []familyEdge) (own, step []familyEdge) {
	for _, e := range edges {
		if e.Step {
			step = append(step, e)
		} else {
			own = append(own, e)
		}
	}

	return own, step
}

// stepParentLabel returns "Stepfather", "Stepmother", or "Stepparent" for a
// step-parent of the given sex.
func stepParentLabel(sex string) string {
	switch strings.ToLower(sex) {
	case glxlib.SexMale:
		return "Stepfather"
	case glxlib.SexFemale:
		return "Stepmother"
	default:
		return "Stepparent"
	}
}

// parentLabel returns "Father", "Mother", or "Parent" for a parent of the
// given sex.
func parentLabel(sex string) string {
	switch strings.ToLower(sex) {
	case glxlib.SexMale:
		return "Father"
	case glxlib.SexFemale:
		return "Mother"
	default:
		return "Parent"
	}
}

// withHypotheticalMarker appends the "(?)" marker to a name when hypo is set.
func withHypotheticalMarker(name string, hypo bool) string {
	if hypo {
		return name + "  " + hypotheticalMarker
	}

	return name
}

// competingBirthParents identifies unresolved birth-parent alternatives for
// summary labels. Step, adoptive and foster links do not compete.
func (f *familyLinks) competingBirthParents(personID string) map[string]bool {
	bySex := map[string]map[string]bool{}
	for _, link := range f.parentsOf[personID] {
		if link.Step || (link.RelationshipType != glxlib.RelationshipTypeParentChild && link.RelationshipType != glxlib.RelationshipTypeBiologicalParentChild) {
			continue
		}
		sex := strings.ToLower(summaryPersonSex(link.ParentID, f.archive))
		if sex != glxlib.SexMale && sex != glxlib.SexFemale {
			continue
		}
		if bySex[sex] == nil {
			bySex[sex] = map[string]bool{}
		}
		bySex[sex][link.ParentID] = true
	}
	competing := map[string]bool{}
	for _, ids := range bySex {
		if len(ids) > 1 {
			for id := range ids {
				competing[id] = true
			}
		}
	}

	return competing
}
