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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func claim(subject EntityRef, person, role, status string) *Assertion {
	return &Assertion{Subject: subject, Participant: &Participant{Person: person, Role: role}, Confidence: "high", Status: status}
}

func excludedIDs(result *ProofResult) []string {
	ids := make([]string, 0, len(result.Excluded))
	for i := range result.Excluded {
		ids = append(ids, result.Excluded[i].PersonID)
	}

	return ids
}

func answerIDs(result *ProofResult) []string {
	ids := make([]string, 0, len(result.Answers))
	for i := range result.Answers {
		ids = append(ids, result.Answers[i].PersonID)
	}

	return ids
}

// A child known only from a baptism: every parent claim decides it, so a
// proven and a disproven claim conflict instead of the disproven one
// excluding the child alone.
func TestBuildProof_EventOnlyChildClaimsDecide(t *testing.T) {
	bapt := EntityRef{Event: "bapt"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "k": proofPerson("Eva", SexFemale)},
		Events: map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{
			{Person: "k", Role: ParticipantRoleSubject}, {Person: "c", Role: ParticipantRoleParent},
		}}},
		Assertions: map[string]*Assertion{
			"yes": claim(bapt, "c", ParticipantRoleParent, statusProven),
			"no":  claim(bapt, "c", ParticipantRoleParent, statusDisproven),
		},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	assert.Empty(t, result.Excluded)

	// A proven claim of a baptism that does not list the person as parent
	// makes the child a candidate too.
	archive.Events["bapt"].Participants = archive.Events["bapt"].Participants[:1]
	delete(archive.Assertions, "no")
	result = mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"k"}, answerIDs(result))
	archive.Assertions["yes"].Status = statusDisproven
	result = mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, []string{"k"}, excludedIDs(result))
}

// A probable same-person link caps the identity at PROBABLE, whatever else
// is proven about the person; a disputed link beside it stays open.
func TestBuildProof_IdentityLinkStrength(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale), "q": proofPerson("Johann", SexMale), "r": proofPerson("Hans", SexMale)},
		Relationships: map[string]*Relationship{
			"same": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "p"}, {Person: "q"}}},
		},
		Assertions: map[string]*Assertion{
			"same": {Subject: EntityRef{Relationship: "same"}, Confidence: "medium", Status: statusProbable},
			"occ":  {Subject: EntityRef{Person: "p"}, Property: PersonPropertyOccupation, Value: "miller", Confidence: "high", Status: statusProven},
		},
	}
	result := mustProof(t, archive, "p", QuestionIdentity)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "Same person as Johann (q).")

	// The disputed link to r is a conflict to resolve, not a namesake told
	// apart, although the link to q is accepted.
	archive.Relationships["also"] = &Relationship{Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "p"}, {Person: "r"}}}
	archive.Assertions["also"] = &Assertion{Subject: EntityRef{Relationship: "also"}, Confidence: "low", Status: standingStatusDisputed}
	result = mustProof(t, archive, "p", QuestionIdentity)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	assert.Empty(t, result.Excluded)
}

// A lone disputed candidate, even one disputed only by legacy confidence,
// leaves the question CONFLICTED rather than listing it as excluded.
func TestBuildProof_LoneDisputedCandidateConflicts(t *testing.T) {
	archive := marriageProofArchive()
	delete(archive.Relationships, "r2")
	delete(archive.Assertions, "a2")
	archive.Assertions["a1"].Status = ""
	archive.Assertions["a1"].Confidence = standingStatusDisputed
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	assert.Empty(t, result.Excluded)
}

// Two relationships to one parent are resolved one by one: a proven adoption
// and a disproven biological link are both true, so the parent is an answer,
// the biological link an excluded alternative, and nothing conflicts.
func TestBuildProof_TwoRelationshipsToOneParent(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale)},
		Relationships: map[string]*Relationship{
			"bio":   {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{{Person: "f", Role: ParticipantRoleParent}, {Person: "c", Role: ParticipantRoleChild}}},
			"adopt": {Type: RelationshipTypeAdoptiveParentChild, Participants: []Participant{{Person: "f", Role: ParticipantRoleAdoptiveParent}, {Person: "c", Role: ParticipantRoleAdoptedChild}}},
		},
		Assertions: map[string]*Assertion{
			"bio":   {Subject: EntityRef{Relationship: "bio"}, Confidence: "high", Status: statusDisproven},
			"adopt": {Subject: EntityRef{Relationship: "adopt"}, Confidence: "high", Status: statusProven},
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"f"}, answerIDs(result))
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, "bio", result.Excluded[0].Relationship)
}

// A proven marriage and a disproven second marriage to the same spouse: the
// spouse is an answer, and the accepted marriage keeps its record gap.
func TestBuildProof_TwoMarriagesToOneSpouse(t *testing.T) {
	archive := marriageProofArchive()
	archive.Relationships["r2"].Participants[1].Person = "w"
	archive.Assertions["a2"].Participant.Person = "w"
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"w"}, answerIDs(result))
	assert.Equal(t, []string{"w"}, excludedIDs(result))
	labels := make([]string, 0, len(result.Gaps))
	for _, g := range result.Gaps {
		labels = append(labels, g.Label)
	}
	assert.Contains(t, labels, "Marriage record — Catharina")
}

// An excluded spouse sharing an accepted spouse's name does not remove the
// accepted marriage's record gap.
func TestBuildProof_ExcludedNamesakeKeepsAcceptedGap(t *testing.T) {
	archive := marriageProofArchive()
	archive.Persons["x"] = proofPerson("Catharina", SexFemale)
	result := mustProof(t, archive, "a", QuestionMarriage)
	count := 0
	for _, g := range result.Gaps {
		if g.Label == "Marriage record — Catharina" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

// Parents named only by claims on the birth record are candidates decided by
// those claims, beside a linked parent or with no link at all.
func TestBuildProof_ClaimOnlyParents(t *testing.T) {
	birth := EntityRef{Event: "birth"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale), "m": proofPerson("Anna", SexFemale)},
		Events:  map[string]*Event{"birth": {Type: EventTypeBirth, Participants: []Participant{{Person: "c", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{
			"f": claim(birth, "f", ParticipantRoleParent, statusProven),
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"f"}, answerIDs(result))

	// A proven and a disproven claim about the same parent conflict.
	archive.Assertions["f2"] = claim(birth, "f", ParticipantRoleParent, statusDisproven)
	result = mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	delete(archive.Assertions, "f2")

	// With the mother linked, the father named only by the claim is kept.
	archive.Relationships = map[string]*Relationship{"pc": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{
		{Person: "m", Role: ParticipantRoleParent}, {Person: "c", Role: ParticipantRoleChild},
	}}}
	archive.Assertions["pc"] = &Assertion{Subject: EntityRef{Relationship: "pc"}, Confidence: "high", Status: statusProven}
	result = mustProof(t, archive, "c", QuestionParentage)
	assert.ElementsMatch(t, []string{"f", "m"}, answerIDs(result))
}

// A spouse claim on a marriage relationship that does not list the person
// makes the recorded counterpart a candidate, accepted or excluded.
func TestBuildProof_ClaimOnlyMarriageRelationship(t *testing.T) {
	rel := EntityRef{Relationship: "r"}
	archive := &GLXFile{
		Persons:       map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{"r": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "w", Role: ParticipantRoleSpouse}}}},
		Assertions:    map[string]*Assertion{"c": claim(rel, "a", ParticipantRoleSpouse, statusProven)},
	}
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"w"}, answerIDs(result))
	archive.Assertions["c"].Status = statusDisproven
	result = mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Equal(t, []string{"w"}, excludedIDs(result))
}

// Banns prepare a marriage; a witness takes no part in it. Neither makes a
// spouse.
func TestBuildProof_BannsAndWitnessMakeNoSpouse(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale), "x": proofPerson("Eva", SexFemale)},
		Events: map[string]*Event{
			"banns": {Type: EventTypeMarriageBanns, Date: "1700", Participants: []Participant{{Person: "a", Role: ParticipantRoleGroom}, {Person: "w", Role: ParticipantRoleBride}}},
			"wed":   {Type: EventTypeMarriage, Date: "1702", Participants: []Participant{{Person: "x", Role: ParticipantRoleBride}, {Person: "a", Role: ParticipantRoleWitness}}},
		},
		Assertions: map[string]*Assertion{
			"banns":   {Subject: EntityRef{Event: "banns"}, Confidence: "high", Status: statusProven},
			"date":    {Subject: EntityRef{Event: "banns"}, Property: conflictDateType, Value: "1700", Confidence: "high", Status: statusProven},
			"witness": claim(EntityRef{Event: "wed"}, "a", ParticipantRoleWitness, statusProven),
		},
	}
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Empty(t, result.Answers)
	assert.NotEmpty(t, result.Evidence, "the banns stay as contextual evidence")
}

// A wedding naming only the person is another marriage until a relationship
// names it: its proven record does not upgrade an unsupported spouse.
func TestBuildProof_UnlinkedWeddingStaysUnattributed(t *testing.T) {
	archive := &GLXFile{
		Persons:       map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{"r": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "a", Role: ParticipantRoleSpouse}, {Person: "w", Role: ParticipantRoleSpouse}}}},
		Events:        map[string]*Event{"wed": {Type: EventTypeMarriage, Participants: []Participant{{Person: "a", Role: ParticipantRoleGroom}}}},
		Assertions:    map[string]*Assertion{"wed": {Subject: EntityRef{Event: "wed"}, Confidence: "high", Status: statusProven}},
	}
	result := mustProof(t, archive, "a", QuestionMarriage)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, ProofStatusUnsupported, result.Answers[0].Status)
	archive.Relationships["r"].StartEvent = "wed"
	result = mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ProofStatusProven, result.Answers[0].Status)
}

// The same office disproven for one period and proven for another is two
// candidates, not a conflict; dates of typed temporal values are kept.
func TestBuildProof_OccupationPeriods(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale), "q": proofPerson("Hans", SexMale), "nil": nil},
		Assertions: map[string]*Assertion{
			"early": {Subject: EntityRef{Person: "p"}, Property: PersonPropertyOccupation, Value: "church elder", Date: "1755", Confidence: "high", Status: statusDisproven},
			"late":  {Subject: EntityRef{Person: "p"}, Property: PersonPropertyOccupation, Value: "church elder", Date: "1762", Confidence: "high", Status: statusProven},
		},
	}
	archive.Persons["q"].Properties[PersonPropertyOccupation] = []TemporalValue{{Value: "church elder", Date: "1770"}}
	result := mustProof(t, archive, "p", QuestionOccupation)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, "church elder (1762)", result.Answers[0].Label)
	assert.Equal(t, []string{"Hans (q) 1770"}, result.Answers[0].SharedWith)
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, "church elder (1755)", result.Excluded[0].Name)
}

// An occupation recorded as a typed temporal value, or claimed on an event
// participant's properties, is a candidate.
func TestBuildProof_OccupationForms(t *testing.T) {
	archive := &GLXFile{Persons: map[string]*Person{"p": proofPerson("Johann", SexMale)}}
	archive.Persons["p"].Properties[PersonPropertyOccupation] = []TemporalValue{{Value: "miller", Date: "1851"}}
	result := mustProof(t, archive, "p", QuestionOccupation)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, "miller (1851)", result.Answers[0].Label)

	delete(archive.Persons["p"].Properties, PersonPropertyOccupation)
	archive.Events = map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{{Person: "p", Role: ParticipantRoleGodparent}}}}
	archive.Assertions = map[string]*Assertion{"occ": {
		Subject: EntityRef{Event: "bapt"}, Participant: &Participant{Person: "p", Role: ParticipantRoleGodparent, Properties: map[string]any{PersonPropertyOccupation: "miller"}},
		Confidence: "high", Status: statusProven,
	}}
	result = mustProof(t, archive, "p", QuestionOccupation)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, []string{"occ"}, result.Answers[0].Assertions)
}

// A disputed type assertion for an alternative father is the alternative's
// status, not a conflict about the accepted father.
func TestBuildProof_RejectedAlternativeRaisesNoConflict(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale), "g": proofPerson("Peter", SexMale)},
		Relationships: map[string]*Relationship{
			"pf": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{{Person: "f", Role: ParticipantRoleParent}, {Person: "c", Role: ParticipantRoleChild}}},
			"pg": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{{Person: "g", Role: ParticipantRoleParent}, {Person: "c", Role: ParticipantRoleChild}}},
		},
		Assertions: map[string]*Assertion{
			"pf": {Subject: EntityRef{Relationship: "pf"}, Confidence: "high", Status: statusProven},
			"pg": {Subject: EntityRef{Relationship: "pg"}, Property: relationshipPropertyType, Value: RelationshipTypeBiologicalParentChild, Confidence: "low", Status: standingStatusDisputed},
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"g"}, excludedIDs(result))
}

// A children-proof assertion naming the co-parent is not about a child: the
// counterpart is the relationship's child.
func TestBuildProof_ChildrenCounterpartIsTheChild(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "h": proofPerson("Hans", SexMale), "k": proofPerson("Eva", SexFemale)},
		Relationships: map[string]*Relationship{"pc": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{
			{Person: "c", Role: ParticipantRoleParent}, {Person: "h", Role: ParticipantRoleParent}, {Person: "k", Role: ParticipantRoleChild},
		}}},
		Assertions: map[string]*Assertion{"h": claim(EntityRef{Relationship: "pc"}, "h", ParticipantRoleParent, statusProven)},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	require.Len(t, result.Evidence, 1)
	assert.Equal(t, []string{"k"}, result.Evidence[0].Counterparts)
}

// Excluded candidates settle the question: a ruled-out marriage's proven date
// supplies no answer.
func TestBuildProof_ExhaustedCandidatesAreAuthoritative(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale)},
		Events:  map[string]*Event{"wed": {Type: EventTypeMarriage, Date: "1700"}},
		Relationships: map[string]*Relationship{"r": {Type: RelationshipTypeMarriage, StartEvent: "wed", Participants: []Participant{
			{Person: "a", Role: ParticipantRoleSpouse}, {Person: "w", Role: ParticipantRoleSpouse},
		}}},
		Assertions: map[string]*Assertion{
			"r":    {Subject: EntityRef{Relationship: "r"}, Confidence: "high", Status: statusDisproven},
			"date": {Subject: EntityRef{Event: "wed"}, Property: conflictDateType, Value: "1700", Confidence: "high", Status: statusProven},
		},
	}
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Contains(t, result.Summary, "Excluded: Anna.")
}

// A proven date of a baptism the person only sponsored proves the baptism,
// not who the person was.
func TestBuildProof_IdentityIgnoresOthersEventFacts(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale)},
		Events:  map[string]*Event{"bapt": {Type: EventTypeBaptism, Date: "1720", Participants: []Participant{{Person: "p", Role: ParticipantRoleGodparent}}}},
		Assertions: map[string]*Assertion{
			"date": {Subject: EntityRef{Event: "bapt"}, Property: conflictDateType, Value: "1720", Confidence: "high", Status: statusProven},
		},
	}
	result := mustProof(t, archive, "p", QuestionIdentity)
	assert.NotEqual(t, ConclusionProven, result.Conclusion)
	assert.Len(t, result.Evidence, 1, "the baptism stays visible as context")
	archive.Assertions["godparent"] = claim(EntityRef{Event: "bapt"}, "p", ParticipantRoleGodparent, statusProven)
	result = mustProof(t, archive, "p", QuestionIdentity)
	assert.Equal(t, ConclusionProven, result.Conclusion)
}

// A planned search covers no gap.
func TestMarkSearchedGaps_PlannedSearchCoversNothing(t *testing.T) {
	gaps := []ProofGap{{Label: "Church records", Priority: severityHigh}}
	markSearchedGaps(gaps, []ProofSearch{{SourceType: SourceTypeChurchRegister, Result: SearchResultNotSearched}}, nil)
	assert.Empty(t, gaps[0].Searched)
	assert.Contains(t, appendNextSteps("", gaps), "Church records")
}
