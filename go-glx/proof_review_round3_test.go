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

func parentChild(parents, children []string) *Relationship {
	r := &Relationship{Type: RelationshipTypeBiologicalParentChild}
	for _, p := range parents {
		r.Participants = append(r.Participants, Participant{Person: p, Role: ParticipantRoleParent})
	}
	for _, c := range children {
		r.Participants = append(r.Participants, Participant{Person: c, Role: ParticipantRoleChild})
	}

	return r
}

func proven(subject EntityRef) *Assertion {
	return &Assertion{Subject: subject, Confidence: "high", Status: statusProven}
}

// A disputed baptism claim corroborating a disproven father link goes with
// the excluded father: it leaves no conflict behind.
func TestBuildProof_CorroborationLeavesWithExcludedCandidate(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale), "m": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{
			"pf": parentChild([]string{"f"}, []string{"c"}),
			"pm": parentChild([]string{"m"}, []string{"c"}),
		},
		Events: map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{{Person: "c", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{
			"pf":   {Subject: EntityRef{Relationship: "pf"}, Confidence: "high", Status: statusDisproven},
			"pm":   proven(EntityRef{Relationship: "pm"}),
			"bapt": claim(EntityRef{Event: "bapt"}, "f", ParticipantRoleParent, standingStatusDisputed),
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"m"}, answerIDs(result))
	assert.Equal(t, []string{"f"}, excludedIDs(result))
}

// A witness to a marriage is nobody's spouse, in a recorded relationship or
// a claimed one.
func TestBuildProof_WitnessIsNoSpouse(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale), "z": proofPerson("Peter", SexMale), "y": proofPerson("Jakob", SexMale)},
		Relationships: map[string]*Relationship{"r": {Type: RelationshipTypeMarriage, Participants: []Participant{
			{Person: "a", Role: ParticipantRoleGroom}, {Person: "w", Role: ParticipantRoleBride}, {Person: "z", Role: ParticipantRoleWitness},
		}}},
		Assertions: map[string]*Assertion{"r": proven(EntityRef{Relationship: "r"})},
	}
	result := mustProof(t, archive, "z", QuestionMarriage)
	assert.Empty(t, result.Answers)
	result = mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, []string{"w"}, answerIDs(result))

	// Claimed into the relationship as spouse, y is coupled with the couple,
	// never with the witness.
	archive.Assertions["y"] = claim(EntityRef{Relationship: "r"}, "y", ParticipantRoleSpouse, statusProven)
	result = mustProof(t, archive, "y", QuestionMarriage)
	assert.NotContains(t, answerIDs(result), "z")
}

// A spouse named only by a claim on a relationship that lists the person is a
// candidate beside the person's other spouses.
func TestBuildProof_ClaimedSpouseInListedRelationship(t *testing.T) {
	archive := marriageProofArchive()
	archive.Assertions["a2"].Status = statusProven
	archive.Relationships["r3"] = &Relationship{Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "a", Role: ParticipantRoleSpouse}}}
	archive.Persons["v"] = proofPerson("Eva", SexFemale)
	archive.Assertions["a3"] = claim(EntityRef{Relationship: "r3"}, "v", ParticipantRoleSpouse, statusProven)
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.ElementsMatch(t, []string{"w", "x", "v"}, answerIDs(result))

	archive.Assertions["a3"].Status = statusDisproven
	result = mustProof(t, archive, "a", QuestionMarriage)
	assert.ElementsMatch(t, []string{"w", "x"}, answerIDs(result))
	assert.Equal(t, []string{"v"}, excludedIDs(result))
}

// Relationship claims that the person is a parent, or that someone is the
// child, make child candidates even when the relationship does not list them.
func TestBuildProof_ClaimOnlyChildRelationships(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "k": proofPerson("Eva", SexFemale), "j": proofPerson("Jakob", SexMale)},
		Relationships: map[string]*Relationship{
			"pk": parentChild(nil, []string{"k"}),
			"pj": parentChild([]string{"c"}, nil),
		},
		Assertions: map[string]*Assertion{
			"pk": claim(EntityRef{Relationship: "pk"}, "c", ParticipantRoleParent, statusProven),
			"pj": claim(EntityRef{Relationship: "pj"}, "j", ParticipantRoleChild, statusProven),
		},
		ResearchLogs: map[string]*ResearchLog{"log": {Subject: &EntityRef{Person: "c"}, Properties: map[string]any{ResearchLogQuestionProperty: "children"}, Searches: []Search{{Result: SearchResultNotFound}}}},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.ElementsMatch(t, []string{"k", "j"}, answerIDs(result))
	assert.NotContains(t, result.Summary, "No children found")

	archive.Assertions["pk"].Status = statusDisproven
	archive.Assertions["pj"].Status = statusDisproven
	result = mustProof(t, archive, "c", QuestionChildren)
	assert.ElementsMatch(t, []string{"k", "j"}, excludedIDs(result))
}

// A child named only by a claim of a baptism that lists the person as parent
// is a candidate beside the person's other children.
func TestBuildProof_ClaimedChildOnBirthRecord(t *testing.T) {
	bapt := EntityRef{Event: "bapt"}
	archive := &GLXFile{
		Persons:       map[string]*Person{"c": proofPerson("Catharina", SexFemale), "k": proofPerson("Eva", SexFemale), "j": proofPerson("Jakob", SexMale)},
		Relationships: map[string]*Relationship{"pj": parentChild([]string{"c"}, []string{"j"})},
		Events:        map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{{Person: "c", Role: ParticipantRoleParent}}}},
		Assertions: map[string]*Assertion{
			"pj": proven(EntityRef{Relationship: "pj"}),
			"k":  claim(bapt, "k", ParticipantRoleSubject, statusProven),
		},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	assert.ElementsMatch(t, []string{"j", "k"}, answerIDs(result))

	archive.Assertions["k"].Status = statusDisproven
	result = mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, []string{"j"}, answerIDs(result))
	assert.Equal(t, []string{"k"}, excludedIDs(result))
}

// The person baptized as godchild is the baptism's principal: a child of the
// parents it names, with a parentage of their own, and the subject of logs
// about the baptism.
func TestBuildProof_GodchildIsBaptismPrincipal(t *testing.T) {
	bapt := EntityRef{Event: "bapt"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "k": proofPerson("Eva", SexFemale)},
		Events: map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{
			{Person: "c", Role: ParticipantRoleParent}, {Person: "k", Role: ParticipantRoleGodchild},
		}}},
		Assertions: map[string]*Assertion{"c": claim(bapt, "c", ParticipantRoleParent, statusProven)},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, []string{"k"}, answerIDs(result))
	result = mustProof(t, archive, "k", QuestionParentage)
	assert.Equal(t, []string{"c"}, answerIDs(result))
	questions, about := researchLogQuestions(&ResearchLog{Subject: &bapt}, "k", archive)
	assert.True(t, about)
	assert.Contains(t, questions, topicParentage)
}

// Two same-person links between one pair, proven and disproven, give
// opposing answers to one question: a conflict.
func TestBuildProof_OpposingLinksToOnePerson(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale), "q": proofPerson("Johann", SexMale)},
		Relationships: map[string]*Relationship{
			"s1": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "p"}, {Person: "q"}}},
			"s2": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "p"}, {Person: "q"}}},
		},
		Assertions: map[string]*Assertion{
			"s1": proven(EntityRef{Relationship: "s1"}),
			"s2": {Subject: EntityRef{Relationship: "s2"}, Confidence: "high", Status: statusDisproven},
		},
	}
	result := mustProof(t, archive, "p", QuestionIdentity)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	assert.NotContains(t, result.Summary, "Distinguished from")
}

// A birth date proven by one assertion and disproven by another is a conflict.
func TestBuildProof_VitalValueProvenAndDisproven(t *testing.T) {
	birth := EntityRef{Event: "birth"}
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale)},
		Events:  map[string]*Event{"birth": {Type: EventTypeBirth, Date: "1700", Participants: []Participant{{Person: "p", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{
			"yes": {Subject: birth, Property: conflictDateType, Value: "1700", Confidence: "high", Status: statusProven},
			"no":  {Subject: birth, Property: conflictDateType, Value: "1700", Confidence: "high", Status: statusDisproven},
		},
	}
	result := mustProof(t, archive, "p", QuestionBirth)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	delete(archive.Assertions, "no")
	result = mustProof(t, archive, "p", QuestionBirth)
	assert.Equal(t, ConclusionProven, result.Conclusion)
}

// Only not_found is a negative result; an archive's own result is outstanding.
func TestNegativeSearchSummary_CustomResultIsOutstanding(t *testing.T) {
	_, negative := negativeSearchSummary([]ProofSearch{{Result: SearchResultNotFound}, {Result: "pending_review"}})
	assert.False(t, negative)
	_, negative = negativeSearchSummary([]ProofSearch{{Result: SearchResultNotFound}})
	assert.True(t, negative)
}

// A claim naming the person as child of a relationship that lists only the
// parents evidences those parents.
func TestBuildProof_ClaimedChildOfParentsOnlyRelationship(t *testing.T) {
	archive := &GLXFile{
		Persons:       map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale)},
		Relationships: map[string]*Relationship{"pc": parentChild([]string{"f"}, nil)},
		Assertions:    map[string]*Assertion{"c": claim(EntityRef{Relationship: "pc"}, "c", ParticipantRoleChild, statusProven)},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Equal(t, []string{"f"}, answerIDs(result))
	require.Len(t, result.Evidence, 1)
}

// A probable and a disputed claim of one spouse resolve to probable; the
// dispute is not judged again as a conflict.
func TestBuildProof_ResolverGovernsParticipationDisputes(t *testing.T) {
	archive := marriageProofArchive()
	delete(archive.Relationships, "r2")
	delete(archive.Assertions, "a2")
	archive.Assertions["a1"].Status, archive.Assertions["a1"].Confidence = statusProbable, "medium"
	archive.Assertions["a1b"] = claim(EntityRef{Relationship: "r1"}, "w", ParticipantRoleSpouse, standingStatusDisputed)
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, ProofStatusProbable, result.Answers[0].Status)
}

// Two jobs held on the same day do not contradict each other; one job proven
// and disproven for the same day does.
func TestBuildProof_OccupationsHeldTogether(t *testing.T) {
	occ := func(value, status string) *Assertion {
		return &Assertion{Subject: EntityRef{Person: "p"}, Property: PersonPropertyOccupation, Value: value, Date: "1851-01-01", Confidence: "high", Status: status}
	}
	archive := &GLXFile{
		Persons:    map[string]*Person{"p": proofPerson("Johann", SexMale)},
		Assertions: map[string]*Assertion{"farmer": occ("farmer", statusProven), "miller": occ("miller", statusProven)},
	}
	for _, question := range []string{QuestionOccupation, QuestionIdentity} {
		result := mustProof(t, archive, "p", question)
		assert.NotEqual(t, ConclusionConflicted, result.Conclusion, question)
	}
	assert.Len(t, mustProof(t, archive, "p", QuestionOccupation).Answers, 2)

	archive.Assertions["not-miller"] = occ("miller", statusDisproven)
	assert.Equal(t, ConclusionConflicted, mustProof(t, archive, "p", QuestionOccupation).Conclusion)
}

// Birth parents named only by claims are compared with each other and with a
// linked birth parent.
func TestBuildProof_ClaimOnlyParentsCompete(t *testing.T) {
	birth := EntityRef{Event: "birth"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale), "g": proofPerson("Peter", SexMale)},
		Events:  map[string]*Event{"birth": {Type: EventTypeBirth, Participants: []Participant{{Person: "c", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{
			"f": claim(birth, "f", ParticipantRoleParent, statusProven),
			"g": claim(birth, "g", ParticipantRoleParent, statusProven),
		},
	}
	assert.Equal(t, ConclusionConflicted, mustProof(t, archive, "c", QuestionParentage).Conclusion)

	// One claimed father beside a recorded biological father.
	delete(archive.Assertions, "f")
	archive.Relationships = map[string]*Relationship{"pf": parentChild([]string{"f"}, []string{"c"})}
	archive.Assertions["pf"] = proven(EntityRef{Relationship: "pf"})
	assert.Equal(t, ConclusionConflicted, mustProof(t, archive, "c", QuestionParentage).Conclusion)

	// A claimed adoptive father is not a birth parent.
	archive.Assertions["g"].Participant.Role = ParticipantRoleAdoptiveParent
	assert.Equal(t, ConclusionProven, mustProof(t, archive, "c", QuestionParentage).Conclusion)
}

// A parent who survives through an adoption keeps no excluded biological
// link in the comparison of birth fathers.
func TestBuildProof_ExcludedBiologicalLinkDoesNotCompete(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "h": proofPerson("Hans", SexMale), "p": proofPerson("Peter", SexMale)},
		Relationships: map[string]*Relationship{
			"adopt": {Type: RelationshipTypeAdoptiveParentChild, Participants: []Participant{{Person: "h", Role: ParticipantRoleAdoptiveParent}, {Person: "c", Role: ParticipantRoleAdoptedChild}}},
			"hbio":  parentChild([]string{"h"}, []string{"c"}),
			"pbio":  parentChild([]string{"p"}, []string{"c"}),
		},
		Assertions: map[string]*Assertion{
			"adopt": proven(EntityRef{Relationship: "adopt"}),
			"hbio":  {Subject: EntityRef{Relationship: "hbio"}, Confidence: "low", Status: standingStatusDisputed},
			"pbio":  proven(EntityRef{Relationship: "pbio"}),
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.ElementsMatch(t, []string{"h", "p"}, answerIDs(result))
}

// A child's baptism dates and godparents are context for the children
// question, not disputes about which children there were.
func TestBuildProof_ChildBirthRecordDetailsAreContext(t *testing.T) {
	bapt := EntityRef{Event: "bapt"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "k": proofPerson("Eva", SexFemale), "g": proofPerson("Anna", SexFemale)},
		Events: map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{
			{Person: "c", Role: ParticipantRoleParent}, {Person: "k", Role: ParticipantRoleSubject}, {Person: "g", Role: ParticipantRoleGodparent},
		}}},
		Assertions: map[string]*Assertion{
			"c":    claim(bapt, "c", ParticipantRoleParent, statusProven),
			"d1":   {Subject: bapt, Property: conflictDateType, Value: "1700", Confidence: "high", Status: statusProven},
			"d2":   {Subject: bapt, Property: conflictDateType, Value: "1701", Confidence: "high", Status: statusProven},
			"godp": claim(bapt, "g", ParticipantRoleGodparent, standingStatusDisputed),
		},
	}
	result := mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Len(t, result.Evidence, 4)
}

// A proven same-person link establishes the identity from either endpoint,
// whichever one its assertion names.
func TestBuildProof_IdentityLinkFromEitherEndpoint(t *testing.T) {
	archive := &GLXFile{
		Persons:       map[string]*Person{"p": proofPerson("Johann", SexMale), "q": proofPerson("Johann", SexMale)},
		Relationships: map[string]*Relationship{"same": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "p"}, {Person: "q"}}}},
		Assertions:    map[string]*Assertion{"q": claim(EntityRef{Relationship: "same"}, "q", "", statusProven)},
	}
	assert.Equal(t, ConclusionProven, mustProof(t, archive, "p", QuestionIdentity).Conclusion)
	assert.Equal(t, ConclusionProven, mustProof(t, archive, "q", QuestionIdentity).Conclusion)
}

// The answer keeps the relationship that supplies its status.
func TestBuildProof_MergedAnswerKeepsStrongerRelationship(t *testing.T) {
	couple := []Participant{{Person: "a", Role: ParticipantRoleSpouse}, {Person: "w", Role: ParticipantRoleSpouse}}
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{
			"rel-a": {Type: RelationshipTypeMarriage, Participants: couple},
			"rel-b": {Type: RelationshipTypeMarriage, Participants: couple},
		},
		Assertions: map[string]*Assertion{"b": proven(EntityRef{Relationship: "rel-b"})},
	}
	result := mustProof(t, archive, "a", QuestionMarriage)
	require.Len(t, result.Answers, 1)
	assert.Equal(t, ProofStatusProven, result.Answers[0].Status)
	assert.Equal(t, "rel-b", result.Answers[0].Relationship)
}

// A candidate conflict's facts are keyed by each assertion's own subject.
func TestBuildProof_CandidateConflictFactKeys(t *testing.T) {
	birth := EntityRef{Event: "birth"}
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Eva", SexFemale), "f": proofPerson("Hans", SexMale)},
		Events:  map[string]*Event{"birth": {Type: EventTypeBirth, Participants: []Participant{{Person: "c", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{
			"yes": claim(birth, "f", ParticipantRoleParent, statusProven),
			"no":  claim(birth, "f", ParticipantRoleParent, statusDisproven),
		},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	require.Equal(t, ConclusionConflicted, result.Conclusion)
	found := false
	for _, conflict := range result.Conflicts {
		for _, fact := range conflict.Facts {
			found = true
			assert.Equal(t, EntityTypeEvents.String()+":birth", fact.FactKey)
		}
	}
	assert.True(t, found)
}

// Every entry of a participant's temporal occupation is a candidate, each
// with its own date.
func TestBuildProof_ParticipantOccupationEntries(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale)},
		Events:  map[string]*Event{"census": {Type: EventTypeCensus, Participants: []Participant{{Person: "p", Role: ParticipantRoleSubject}}}},
		Assertions: map[string]*Assertion{"occ": {
			Subject: EntityRef{Event: "census"}, Date: "1770", Confidence: "high", Status: statusProven,
			Participant: &Participant{Person: "p", Role: ParticipantRoleSubject, Properties: map[string]any{PersonPropertyOccupation: []any{
				map[string]any{temporalValueField: "church elder", temporalDateField: "1755"},
				map[string]any{temporalValueField: "miller"},
			}}},
		}},
	}
	result := mustProof(t, archive, "p", QuestionOccupation)
	labels := make([]string, 0, len(result.Answers))
	for i := range result.Answers {
		labels = append(labels, result.Answers[i].Label)
	}
	assert.ElementsMatch(t, []string{"church elder (1755)", "miller (1770)"}, labels)
}

// A namesake holding the office only on an event participation, recorded or
// claimed, shares it.
func TestBuildProof_SharedHolderOnEventParticipation(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": proofPerson("Johann", SexMale), "q": proofPerson("Hans", SexMale), "r": proofPerson("Jakob", SexMale)},
		Events: map[string]*Event{"census": {Type: EventTypeCensus, Date: "1760", Participants: []Participant{
			{Person: "q", Role: ParticipantRoleSubject, Properties: map[string]any{PersonPropertyOccupation: "miller"}},
		}}},
		Assertions: map[string]*Assertion{
			"p": {Subject: EntityRef{Person: "p"}, Property: PersonPropertyOccupation, Value: "miller", Confidence: "high", Status: statusProven},
			"r": {
				Subject: EntityRef{Event: "census"}, Date: "1761", Confidence: "high", Status: statusProven,
				Participant: &Participant{Person: "r", Role: ParticipantRoleSubject, Properties: map[string]any{PersonPropertyOccupation: "Miller"}},
			},
		},
	}
	result := mustProof(t, archive, "p", QuestionOccupation)
	require.Len(t, result.Answers, 1)
	assert.ElementsMatch(t, []string{"Hans (q)", "Jakob (r) 1761"}, result.Answers[0].SharedWith)
}

// A census log naming the person as its subject serves the question it names;
// one they only witnessed is not about them.
func TestResearchLogQuestions_NonVitalPrincipal(t *testing.T) {
	archive := &GLXFile{Events: map[string]*Event{"census": {Type: EventTypeCensus, Participants: []Participant{
		{Person: "p", Role: ParticipantRoleSubject}, {Person: "w", Role: ParticipantRoleInformant},
	}}}}
	log := &ResearchLog{Subject: &EntityRef{Event: "census"}, Properties: map[string]any{ResearchLogQuestionProperty: "occupation"}}
	questions, about := researchLogQuestions(log, "p", archive)
	assert.True(t, about)
	assert.Equal(t, []string{topicOccupation}, questions)
	_, about = researchLogQuestions(log, "w", archive)
	assert.False(t, about)
}

// A search covers a census gap only for its year, and a marriage-record gap
// only from a log about that marriage.
func TestMarkSearchedGaps_ScopedToYearAndMarriage(t *testing.T) {
	archive := &GLXFile{
		Sources: map[string]*Source{"s1850": {Title: "1850 US Census", Type: SourceTypeCensus}},
		Relationships: map[string]*Relationship{
			"r1": {Type: RelationshipTypeMarriage, StartEvent: "wed1"},
			"r2": {Type: RelationshipTypeMarriage},
		},
		Events: map[string]*Event{"wed1": {Type: EventTypeMarriage}},
	}
	gaps := []ProofGap{
		{Label: "1850 US Census (age ~30)", Priority: severityHigh, year: 1850},
		{Label: "1860 US Census (age ~40)", Priority: severityHigh, year: 1860},
		{Label: "Marriage record — Anna", relationshipID: "r1"},
		{Label: "Marriage record — Eva", relationshipID: "r2"},
	}
	searches := []ProofSearch{
		{SourceType: SourceTypeCensus, Result: SearchResultNotFound, years: proofSearchYears(&Search{SourceID: "s1850"}, archive)},
		{SourceType: SourceTypeCensus, Result: SearchResultNotFound},
		{SourceType: SourceTypeChurchRegister, Result: SearchResultNotFound, event: "wed1"},
		{SourceType: SourceTypeChurchRegister, Result: SearchResultNotFound},
	}
	markSearchedGaps(gaps, searches, archive)
	assert.Equal(t, "1 search: 1 not_found", gaps[0].Searched)
	assert.Empty(t, gaps[1].Searched)
	assert.Equal(t, "1 search: 1 not_found", gaps[2].Searched)
	assert.Empty(t, gaps[3].Searched)
	assert.Contains(t, appendNextSteps("", gaps), "1860")
}
