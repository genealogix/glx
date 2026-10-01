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

	glxlib "github.com/genealogix/glx/go-glx"
)

// The parentage question's handling of competing and rejected parent
// candidates (#1327). It is kept apart from the general proof engine so the
// rules stay in one place: the candidates are the person's parent-child
// relationships as the shared family graph sees them (step-parents are not
// parentage candidates), classified by relationship standing.

// Competing-parent conflict properties, one per side of the family.
const (
	parentageFather = "father"
	parentageMother = "mother"
)

// proofExcludedAlternative is a candidate parent whose every assertion is
// `disproven`: a recorded, rejected alternative (GPS element 4), listed apart
// from the parents identified.
type proofExcludedAlternative struct {
	PersonID     string `json:"person_id"`
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
}

// parentCandidate is one surviving (non-disproven) birth or adoptive parent.
type parentCandidate struct {
	personID string
	name     string
	sex      string
	// birth is true when at least one link is a plain or biological
	// parent-child relationship, as opposed to adoptive or foster.
	birth bool
	// assertions are the surviving assertions about the links, used to show
	// each competing value's confidence and status.
	assertions []*glxlib.Assertion
}

// parentageCandidates splits a person's recorded parents into surviving
// candidates (in relationship order) and excluded alternatives.
func parentageCandidates(personID string, archive *glxlib.GLXFile) ([]parentCandidate, []proofExcludedAlternative) {
	fam := newFamilyLinks(archive)

	var survivors []parentCandidate
	pos := map[string]int{}
	for _, link := range fam.parentsOf[personID] {
		if link.Step {
			continue
		}
		birth := isBirthParentageType(link.RelationshipType)
		var surviving []*glxlib.Assertion
		for _, a := range fam.standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID) {
			if !glxlib.AssertionIsDisproven(a) {
				surviving = append(surviving, a)
			}
		}
		if i, ok := pos[link.ParentID]; ok {
			survivors[i].birth = survivors[i].birth || birth
			survivors[i].assertions = append(survivors[i].assertions, surviving...)

			continue
		}
		pos[link.ParentID] = len(survivors)
		survivors = append(survivors, parentCandidate{
			personID:   link.ParentID,
			name:       personName(archive, link.ParentID),
			sex:        summaryPersonSex(link.ParentID, archive),
			birth:      birth,
			assertions: surviving,
		})
	}

	var excluded []proofExcludedAlternative
	seen := map[string]bool{}
	for _, link := range fam.excluded[personID] {
		if link.Step || seen[link.ParentID] {
			continue
		}
		if _, surviving := pos[link.ParentID]; surviving {
			continue
		}
		seen[link.ParentID] = true
		excluded = append(excluded, proofExcludedAlternative{
			PersonID:     link.ParentID,
			Name:         personName(archive, link.ParentID),
			Relationship: link.RelationshipID,
		})
	}

	return survivors, excluded
}

// isBirthParentageType reports whether a relationship type claims birth
// parentage (as opposed to adoption or fostering, which can coexist with a
// different birth parent without conflict).
func isBirthParentageType(relType string) bool {
	switch strings.ToLower(relType) {
	case glxlib.RelationshipTypeParentChild, glxlib.RelationshipTypeBiologicalParentChild:
		return true
	}

	return false
}

// parentageConflicts reports competing birth parents: two or more surviving
// candidates of the same sex (fathers, or mothers) claimed through birth
// parent-child relationships. Candidates of unknown sex can't be compared and
// are left out. Every such conflict is unresolved: the archive still holds
// more than one surviving candidate, so the researcher has not yet ruled the
// others out.
func parentageConflicts(survivors []parentCandidate) []proofConflict {
	var conflicts []proofConflict
	for _, side := range []struct{ sex, property string }{
		{glxlib.SexMale, parentageFather},
		{glxlib.SexFemale, parentageMother},
	} {
		var values []proofConflictValue
		for i := range survivors {
			c := &survivors[i]
			if !c.birth || !strings.EqualFold(c.sex, side.sex) {
				continue
			}
			confidence, status := strongestClaim(c.assertions)
			values = append(values, proofConflictValue{Value: c.name, Confidence: confidence, Status: status})
		}
		if len(values) < 2 {
			continue
		}
		conflicts = append(conflicts, proofConflict{
			Property: side.property + " (competing parent_child relationships)",
			Values:   values,
		})
	}

	return conflicts
}

// strongestClaim returns the highest confidence among the assertions and
// their combined status (see combineStatus).
func strongestClaim(assertions []*glxlib.Assertion) (confidence, status string) {
	for i, a := range assertions {
		if i == 0 || confidenceRank(a.Confidence) < confidenceRank(confidence) {
			confidence = a.Confidence
		}
		status = combineStatus(status, a.Status)
	}

	return confidence, status
}

// candidateNames returns the surviving candidates' names, in order.
func candidateNames(survivors []parentCandidate) []string {
	names := make([]string, 0, len(survivors))
	for i := range survivors {
		names = append(names, survivors[i].name)
	}

	return names
}

// excludedAlternativeLine renders one excluded alternative for text output.
func excludedAlternativeLine(e *proofExcludedAlternative) string {
	return e.Name + " (" + e.Relationship + ") -- all assertions disproven"
}
