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
)

const (
	parentageFather = "father"
	parentageMother = "mother"
)

// parentCandidate merges surviving non-step links to the same parent. Only
// birth links supply facts for comparison with other birth parents.
type parentCandidate struct {
	personID   string
	name       string
	sex        string
	birthFacts []ResearchFact
}

func parentageCandidates(personID string, archive *GLXFile, links []ParentChildLink, standings *RelationshipStandingIndex) ([]parentCandidate, []ProofExcludedAlternative) {
	var candidates []parentCandidate
	positions := map[string]int{}
	assertionIDs := map[*Assertion]string{}
	for _, id := range sortedKeys(archive.Assertions) {
		assertionIDs[archive.Assertions[id]] = id
	}
	for _, link := range links {
		if link.ChildID != personID || link.Step || link.Standing == RelationshipStandingDisproven {
			continue
		}
		pos, seen := positions[link.ParentID]
		if !seen {
			pos = len(candidates)
			positions[link.ParentID] = pos
			sex := ""
			if parent := archive.Persons[link.ParentID]; parent != nil {
				sex = propertyString(parent.Properties, PersonPropertySex)
			}
			candidates = append(candidates, parentCandidate{personID: link.ParentID, name: personName(archive, link.ParentID), sex: sex})
		}
		if link.RelationshipType == RelationshipTypeParentChild || link.RelationshipType == RelationshipTypeBiologicalParentChild {
			candidates[pos].birthFacts = append(candidates[pos].birthFacts, parentageLinkFacts(&link, standings, assertionIDs)...)
		}
	}
	var excluded []ProofExcludedAlternative
	seen := map[string]bool{}
	for _, link := range links {
		if link.ChildID != personID || link.Step || link.Standing != RelationshipStandingDisproven || seen[link.ParentID] {
			continue
		}
		if _, survives := positions[link.ParentID]; survives {
			continue
		}
		seen[link.ParentID] = true
		excluded = append(excluded, ProofExcludedAlternative{PersonID: link.ParentID, Name: personName(archive, link.ParentID), Relationship: link.RelationshipID})
	}

	return candidates, excluded
}

func parentageLinkFacts(link *ParentChildLink, standings *RelationshipStandingIndex, ids map[*Assertion]string) []ResearchFact {
	var facts []ResearchFact
	for _, a := range standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID) {
		if AssertionIsDisproven(a) {
			continue
		}
		fact := researchFact(&proofAssertion{id: ids[a], a: a, subjectID: link.RelationshipID, relType: link.RelationshipType, personRole: ParticipantRoleChild})
		fact.Fact.Value = link.ParentID
		facts = append(facts, fact)
	}
	if len(facts) == 0 {
		facts = append(facts, ResearchFact{
			ID:               link.RelationshipID + ":" + link.ParentID + ":" + link.ChildID,
			Subject:          EntityRef{Relationship: link.RelationshipID},
			RelationshipType: link.RelationshipType, PersonRole: ParticipantRoleChild,
			Fact: FactValue{Value: link.ParentID}, Synthetic: true,
		})
	}

	return facts
}

// Competing birth fathers or mothers are mutually exclusive regardless of the
// periods attached to the relationships. Compare raw person IDs so identical
// names remain separate; keep original assertion context and provenance.
func parentageConflicts(personID string, candidates []parentCandidate, archive *GLXFile, opts ComparisonOptions) []ConflictGroup {
	var conflicts []ConflictGroup
	for _, side := range []struct{ sex, property string }{{SexMale, parentageFather}, {SexFemale, parentageMother}} {
		var facts []ResearchFact
		var values []ConflictValue
		var rawIDs []string
		for _, candidate := range candidates {
			if !strings.EqualFold(candidate.sex, side.sex) || len(candidate.birthFacts) == 0 {
				continue
			}
			confidence, status := "", ""
			for i := range candidate.birthFacts {
				fact := candidate.birthFacts[i]
				fact.FactKey = personID + ":parentage"
				fact.FactProperty = side.property
				facts = append(facts, fact)
				if i == 0 || researchConfidenceRank(fact.Fact.Confidence) < researchConfidenceRank(confidence) {
					confidence = fact.Fact.Confidence
				}
				status = combineStatus(status, fact.Fact.Status)
			}
			values = append(values, ConflictValue{Value: candidate.name, Confidence: confidence, Status: status})
			rawIDs = append(rawIDs, candidate.personID)
		}
		if len(values) < 2 {
			continue
		}
		rawFacts := make([]FactValue, len(facts))
		for i := range facts {
			rawFacts[i] = facts[i].Fact
		}
		evaluation := evaluateFacts(rawFacts, &PropertyDefinition{ReferenceType: EntityTypePersons.String()}, archive.Places, opts)
		disambiguateConflictValues(values, rawIDs)
		conflicts = append(conflicts, ConflictGroup{
			Facts: facts, Evaluation: evaluation,
			Verdict: evaluation.Verdict, Definite: evaluation.Definite,
			Subject: personName(archive, personID), Property: side.property + " (competing parent_child relationships)",
			Values: values,
		})
	}

	return conflicts
}

// Relationship properties and step links are context rather than parentage
// evidence. Participant claims apply only to the edges naming that person.
func parentageLinkRelevant(pa *proofAssertion, personID string, links []ParentChildLink) bool {
	if pa.a.Subject.Relationship == "" {
		return true
	}
	if pa.a.Property != "" {
		return false
	}
	for _, link := range links {
		if link.ChildID != personID || link.Step || link.RelationshipID != pa.subjectID {
			continue
		}
		if pa.a.Participant == nil || pa.a.Participant.Person == link.ParentID || pa.a.Participant.Person == link.ChildID {
			return true
		}
	}
	// #1313 also accepts participant evidence before that parent has been
	// denormalized into the relationship's participant list.

	return parentageClaimOnly(pa, personID, links)
}

func parentageClaimOnly(pa *proofAssertion, personID string, links []ParentChildLink) bool {
	participant := pa.a.Participant
	if participant == nil || !IsParentSideRole(participant.Role) || IsStepRole(participant.Role) || participant.Person == personID || participant.Person == "" || IsStepRole(pa.personRole) || strings.EqualFold(pa.relType, RelationshipTypeStepParent) {
		return false
	}
	for _, link := range links {
		if link.RelationshipID == pa.subjectID && link.ChildID == personID && link.ParentID == participant.Person {
			return false // recorded edges were handled above, including steps
		}
	}

	return true
}

// Retain rejected assertions in Evidence, but never let them or a rejected
// participant strengthen or supply a fallback answer to the conclusion.
func parentageSupport(relevant []proofAssertion, personID string, links []ParentChildLink, standings *RelationshipStandingIndex) []proofAssertion {
	supported := map[*Assertion]bool{}
	for _, link := range links {
		if link.ChildID != personID || link.Step || link.Standing == RelationshipStandingDisproven {
			continue
		}
		for _, a := range standings.LinkAssertions(link.RelationshipID, link.ParentID, link.ChildID) {
			supported[a] = true
		}
	}
	var result []proofAssertion
	for i := range relevant {
		pa := &relevant[i]
		if pa.a.Subject.Relationship == "" || supported[pa.a] || (parentageClaimOnly(pa, personID, links) && standings.Link(pa.subjectID, personID, pa.a.Participant.Person) != RelationshipStandingDisproven) {
			result = append(result, *pa)
		}
	}

	return result
}
