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

func proofPerson(given, sex string) *Person {
	return &Person{Properties: map[string]any{PersonPropertyName: given, PersonPropertySex: sex}}
}

func mustProof(t *testing.T, archive *GLXFile, personID, question string) *ProofResult {
	t.Helper()
	result, err := BuildProof(archive, personID, question, ProofOptions{})
	require.NoError(t, err)

	return result
}

func TestResolveProofStatus(t *testing.T) {
	a := func(confidence, status string) *Assertion { return &Assertion{Confidence: confidence, Status: status} }
	cases := []struct {
		name string
		in   []*Assertion
		want string
	}{
		{"none", nil, ProofStatusUnsupported},
		{"nil only", []*Assertion{nil}, ProofStatusUnsupported},
		{"proven status", []*Assertion{a("low", "proven")}, ProofStatusProven},
		{"status wins over confidence", []*Assertion{a("high", "probable")}, ProofStatusProbable},
		{"confidence alone", []*Assertion{a("medium", "")}, ProofStatusProbable},
		{"high confidence alone", []*Assertion{a("high", "")}, ProofStatusProven},
		{"unrated", []*Assertion{a("", "")}, ProofStatusSpeculative},
		{"speculative", []*Assertion{a("high", "speculative")}, ProofStatusSpeculative},
		{"unresearched", []*Assertion{a("", "unresearched")}, ProofStatusSpeculative},
		{"all disproven", []*Assertion{a("high", "disproven"), a("low", "Disproven")}, ProofStatusDisproven},
		{"all disputed", []*Assertion{a("low", "disputed")}, ProofStatusDisputed},
		{"legacy disputed confidence", []*Assertion{a("disputed", "")}, ProofStatusDisputed},
		{"disputed beside probable", []*Assertion{a("low", "disputed"), a("medium", "probable")}, ProofStatusProbable},
		{"proven and disproven", []*Assertion{a("high", "proven"), a("high", "disproven")}, ProofStatusConflicting},
		{"probable survives a disproven claim", []*Assertion{a("medium", "probable"), a("high", "disproven")}, ProofStatusProbable},
		{"strongest survivor", []*Assertion{a("low", ""), a("medium", "probable")}, ProofStatusProbable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ResolveProofStatus(tc.in))
		})
	}
	assert.Equal(t, ProofStatusUnsupported, AssertionProofStatus(nil))
	assert.Equal(t, proofConclusionConflicted, statusConclusion(ProofStatusConflicting))
	assert.Equal(t, proofConclusionPossible, statusConclusion(ProofStatusSpeculative))
	assert.Equal(t, proofConclusionInsufficient, statusConclusion(ProofStatusUnsupported))
}

// A cited assertion of a relationship's own type is evidence for the parents
// it names (#1548).
func TestBuildProof_RelationshipTypeAssertionEvidencesParentage(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("William", SexMale), "f": proofPerson("Isaac", SexMale), "m": proofPerson("Mary", SexFemale)},
		Relationships: map[string]*Relationship{"rel": {Type: RelationshipTypeBiologicalParentChild, Participants: []Participant{
			{Person: "f", Role: ParticipantRoleParent}, {Person: "m", Role: ParticipantRoleParent}, {Person: "c", Role: ParticipantRoleChild},
		}}},
		Sources:   map[string]*Source{"s": {Title: "County history", Type: SourceTypeBook}},
		Citations: map[string]*Citation{"cit": {SourceID: "s"}},
		Assertions: map[string]*Assertion{"type": {
			Subject: EntityRef{Relationship: "rel"}, Property: "type", Value: RelationshipTypeBiologicalParentChild,
			Confidence: "medium", Status: "probable", Citations: []string{"cit"},
		}},
	}
	result := mustProof(t, archive, "c", QuestionParentage)
	require.Len(t, result.Evidence, 1)
	assert.Equal(t, "type", result.Evidence[0].AssertionID)
	assert.Equal(t, []string{"f", "m"}, result.Evidence[0].Counterparts)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	require.Len(t, result.Answers, 2)
	assert.Equal(t, "father", result.Answers[0].Role)
	assert.Equal(t, "mother", result.Answers[1].Role)

	// A disproven type assertion excludes the relationship like an existential one.
	archive.Assertions["type"].Status = statusDisproven
	result = mustProof(t, archive, "c", QuestionParentage)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Len(t, result.Excluded, 2)
	// A type assertion naming another type is a property, not standing.
	archive.Assertions["type"].Value = RelationshipTypeAdoptiveParentChild
	assert.Equal(t, RelationshipStandingAccepted, NewRelationshipStandingIndex(archive).Relationship("rel"))
}

func marriageProofArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann", SexMale), "w": proofPerson("Catharina", SexFemale), "x": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{
			"r1": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "a", Role: ParticipantRoleSpouse}, {Person: "w", Role: ParticipantRoleSpouse}}},
			"r2": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "a", Role: ParticipantRoleSpouse}, {Person: "x", Role: ParticipantRoleSpouse}}},
		},
		Sources:   map[string]*Source{"s": {Title: "Register", Type: SourceTypeChurchRegister}},
		Citations: map[string]*Citation{"c1": {SourceID: "s"}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Relationship: "r1"}, Participant: &Participant{Person: "w", Role: ParticipantRoleSpouse}, Citations: []string{"c1"}, Confidence: "high", Status: statusProven},
			"a2": {Subject: EntityRef{Relationship: "r2"}, Participant: &Participant{Person: "x", Role: ParticipantRoleSpouse}, Citations: []string{"c1"}, Confidence: "high", Status: statusDisproven, Notes: NoteList{"not his wife"}},
		},
	}
}

// A disproven spouse is an excluded alternative: not "Married:", no gap (#1520).
func TestBuildProof_DisprovenSpouseExcluded(t *testing.T) {
	archive := marriageProofArchive()
	result := mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	assert.Contains(t, result.Summary, "Married: Catharina.")
	assert.NotContains(t, result.Summary, "Anna")
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, ProofExcludedAlternative{
		PersonID: "x", Name: "Anna", Relationship: "r2", Role: ParticipantRoleSpouse, Status: ProofStatusDisproven,
		Assertions: []string{"a2"}, Citations: []string{"c1"}, Notes: "not his wife",
	}, result.Excluded[0])
	for _, gap := range result.Gaps {
		assert.NotEqual(t, "Marriage record — Anna", gap.Label)
	}

	// A proven and a disproven assertion about one marriage is a conflict.
	archive.Assertions["a3"] = &Assertion{Subject: EntityRef{Relationship: "r2"}, Participant: &Participant{Person: "x", Role: ParticipantRoleSpouse}, Confidence: "high", Status: statusProven}
	result = mustProof(t, archive, "a", QuestionMarriage)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
	require.NotEmpty(t, result.Conflicts)
	assert.Contains(t, result.Conflicts[len(result.Conflicts)-1].Property, "proven and disproven")
}

// A disproven claim that the person is a recorded bride is shown as an
// excluded alternative naming the groom and the event (#1514).
func TestBuildProof_DisprovenBrideClaimExcluded(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"anna": proofPerson("Anna Pusch", SexFemale), "other": proofPerson("Anna Pusch", SexFemale), "groom": proofPerson("Matthias Morckel", SexMale)},
		Events: map[string]*Event{"ev-m": {Type: EventTypeMarriage, Date: "1719", Participants: []Participant{
			{Person: "groom", Role: ParticipantRoleGroom}, {Person: "other", Role: ParticipantRoleBride},
		}}},
		Sources:   map[string]*Source{"s": {Title: "Parish register", Type: SourceTypeChurchRegister}},
		Citations: map[string]*Citation{"cit-m": {SourceID: "s"}},
		Assertions: map[string]*Assertion{"not-anna": {
			Subject: EntityRef{Event: "ev-m"}, Participant: &Participant{Person: "anna", Role: ParticipantRoleBride},
			Confidence: "high", Status: statusDisproven, Citations: []string{"cit-m"},
		}},
	}
	result := mustProof(t, archive, "anna", QuestionMarriage)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Empty(t, result.Conflicts)
	require.Len(t, result.Evidence, 1)
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, "groom", result.Excluded[0].PersonID)
	assert.Equal(t, "ev-m", result.Excluded[0].Event)
	assert.Equal(t, []string{"cit-m"}, result.Excluded[0].Citations)
	assert.Contains(t, result.Summary, "Excluded: Matthias Morckel.")
}

// Identity gathers the person's own assertions, participations and
// possibly_same_person links (#1508).
func TestBuildProof_IdentityEvidenceAndNamesakes(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"elder": proofPerson("Johann Philipp", SexMale), "younger": proofPerson("Johann Philipp", SexMale)},
		Events:  map[string]*Event{"bapt": {Type: EventTypeBaptism, Date: "1720-03-10", Participants: []Participant{{Person: "elder", Role: ParticipantRoleParent}}}},
		Relationships: map[string]*Relationship{"same": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{
			{Person: "elder"}, {Person: "younger"},
		}}},
		Assertions: map[string]*Assertion{
			"bapt": {Subject: EntityRef{Event: "bapt"}, Participant: &Participant{Person: "elder", Role: ParticipantRoleParent}, Confidence: "high", Status: statusProven},
			"occ":  {Subject: EntityRef{Person: "elder"}, Property: PersonPropertyOccupation, Value: "Kirchen-Senior", Date: "FROM 1720", Confidence: "high", Status: statusProven},
			"same": {Subject: EntityRef{Relationship: "same"}, Participant: &Participant{Person: "elder"}, Confidence: "high", Status: statusDisproven, Notes: NoteList{"elder vs junior"}},
		},
	}
	result := mustProof(t, archive, "elder", QuestionIdentity)
	ids := make([]string, 0, len(result.Evidence)+len(result.Undated))
	for _, ev := range append(result.Evidence, result.Undated...) {
		ids = append(ids, ev.AssertionID)
	}
	assert.ElementsMatch(t, []string{"bapt", "occ", "same"}, ids)
	assert.Equal(t, ConclusionProven, result.Conclusion)
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, "distinguished_from", result.Excluded[0].Role)
	assert.Contains(t, result.Summary, "Distinguished from Johann Philipp (younger).")

	// An accepted link names the same person.
	archive.Assertions["same"].Status = statusProven
	result = mustProof(t, archive, "elder", QuestionIdentity)
	assert.Contains(t, result.Summary, "Same person as Johann Philipp (younger).")
	require.Len(t, result.Answers, 1)

	// An open link is a conflict to resolve.
	archive.Assertions["same"].Status = "speculative"
	result = mustProof(t, archive, "elder", QuestionIdentity)
	assert.Equal(t, ConclusionConflicted, result.Conclusion)
}

// Occupation compiles a person property and names others holding it (#1512).
func TestBuildProof_Occupation(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Conrad Schepp", SexMale), "b": proofPerson("Conrad Schepp", SexMale)},
		Assertions: map[string]*Assertion{
			"a-occ":  {Subject: EntityRef{Person: "a"}, Property: PersonPropertyOccupation, Value: "church elder", Date: "1762", Confidence: "high", Status: statusProven},
			"b-occ":  {Subject: EntityRef{Person: "b"}, Property: PersonPropertyOccupation, Value: "Church  Elder", Date: "1774", Confidence: "high", Status: statusProven},
			"a-farm": {Subject: EntityRef{Person: "a"}, Property: PersonPropertyOccupation, Value: "farmer", Date: "1750", Confidence: "low"},
		},
	}
	for _, alias := range []string{QuestionOccupation, "office", "Trade"} {
		result := mustProof(t, archive, "a", alias)
		assert.Equal(t, QuestionOccupation, result.Question)
		require.Len(t, result.Answers, 2)
		// The weaker answer decides the question (#1484 applies to every question).
		assert.Equal(t, ConclusionPossible, result.Conclusion)
	}
	result := mustProof(t, archive, "a", QuestionOccupation)
	var elder *ProofAnswer
	for i := range result.Answers {
		if result.Answers[i].Value == "church elder" {
			elder = &result.Answers[i]
		}
	}
	require.NotNil(t, elder)
	assert.Equal(t, ProofStatusProven, elder.Status)
	assert.Equal(t, []string{"Conrad Schepp (b) 1774"}, elder.SharedWith)

	delete(archive.Assertions, "a-farm")
	archive.Persons["a"].Properties[PersonPropertyOccupation] = "schoolmaster"
	result = mustProof(t, archive, "a", QuestionOccupation)
	require.Len(t, result.Answers, 2)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion, "an occupation recorded without an assertion is unsupported")
	assert.Contains(t, result.Summary, "No assertion supports schoolmaster.")
}

func childrenProofArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{"c": proofPerson("Catharina", SexFemale), "h": proofPerson("Hans", SexMale), "k": proofPerson("Anna", SexFemale)},
		Relationships: map[string]*Relationship{
			"m":  {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "c", Role: ParticipantRoleSpouse}, {Person: "h", Role: ParticipantRoleSpouse}}},
			"pc": {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "c", Role: ParticipantRoleParent}, {Person: "k", Role: ParticipantRoleChild}}},
		},
		Sources: map[string]*Source{"kb": {Title: "Register", Type: SourceTypeChurchRegister}},
		Assertions: map[string]*Assertion{
			"pc": {Subject: EntityRef{Relationship: "pc"}, Participant: &Participant{Person: "k", Role: ParticipantRoleChild}, Confidence: "high", Status: statusDisproven},
		},
		ResearchLogs: map[string]*ResearchLog{"kids": {Subject: &EntityRef{Relationship: "m"}, Searches: []Search{
			{SourceID: "kb", Query: "Baptisms 1669-1680", Result: SearchResultNotFound},
			{SourceID: "kb", Query: "Baptisms 1681-1691", Result: SearchResultNotFound},
		}}},
	}
}

// A children question: a disproven child is excluded, and negative searches of
// a log about the marriage support "no children found" (#1499).
func TestBuildProof_Children(t *testing.T) {
	archive := childrenProofArchive()
	result := mustProof(t, archive, "c", "children")
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "No children found. None found in 2 logged searches.")
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, "k", result.Excluded[0].PersonID)
	require.Len(t, result.Searches, 2)
	for _, gap := range result.Gaps {
		if gap.Label == "Church records" {
			assert.Equal(t, "2 searches: 2 not_found", gap.Searched)
		}
	}

	// An outstanding search leaves the question open.
	archive.ResearchLogs["kids"].Searches[1].Result = SearchResultRequiresVisit
	result = mustProof(t, archive, "c", QuestionChildren)
	assert.Equal(t, ConclusionInsufficient, result.Conclusion)
	assert.Contains(t, result.Summary, "1 of 2 logged searches still outstanding.")

	// A child baptized with the person as parent is an answer.
	archive.Assertions["pc"].Status = statusProven
	archive.Persons["k2"] = proofPerson("Eva", SexFemale)
	archive.Events = map[string]*Event{"bapt": {Type: EventTypeBaptism, Date: "1675", Participants: []Participant{
		{Person: "k2", Role: ParticipantRoleSubject}, {Person: "c", Role: ParticipantRoleParent},
	}}}
	archive.Assertions["bapt"] = &Assertion{Subject: EntityRef{Event: "bapt"}, Participant: &Participant{Person: "c", Role: ParticipantRoleParent}, Confidence: "medium", Status: "probable"}
	result = mustProof(t, archive, "c", QuestionChildren)
	require.Len(t, result.Answers, 2)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "Children identified: Anna, Eva.")
}

// Two parents of different strength: the question takes the weaker (#1484).
func TestBuildProof_ParentageWeakestParent(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"kid": proofPerson("Anna", SexFemale), "jb": proofPerson("Johann", SexMale), "mom": proofPerson("Anna Rommel", SexFemale)},
		Relationships: map[string]*Relationship{
			"pc":   {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "jb", Role: ParticipantRoleParent}, {Person: "kid", Role: ParticipantRoleChild}}},
			"pc-m": {Type: RelationshipTypeParentChild, Participants: []Participant{{Person: "mom", Role: ParticipantRoleParent}, {Person: "kid", Role: ParticipantRoleChild}}},
		},
		Assertions: map[string]*Assertion{
			"a-pc":   {Subject: EntityRef{Relationship: "pc"}, Participant: &Participant{Person: "kid", Role: ParticipantRoleChild}, Confidence: "medium", Status: "probable"},
			"a-pc-m": {Subject: EntityRef{Relationship: "pc-m"}, Participant: &Participant{Person: "kid", Role: ParticipantRoleChild}, Confidence: "high", Status: statusProven},
		},
	}
	result := mustProof(t, archive, "kid", QuestionParentage)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "Father Johann: PROBABLE; Mother Anna Rommel: PROVEN.")

	// A baptism naming the father corroborates him.
	archive.Events = map[string]*Event{"bapt": {Type: EventTypeBaptism, Participants: []Participant{{Person: "kid", Role: ParticipantRoleSubject}, {Person: "jb", Role: ParticipantRoleParent}}}}
	archive.Assertions["bapt"] = &Assertion{Subject: EntityRef{Event: "bapt"}, Participant: &Participant{Person: "jb", Role: ParticipantRoleParent}, Confidence: "high", Status: statusProven}
	result = mustProof(t, archive, "kid", QuestionParentage)
	assert.Equal(t, ConclusionProven, result.Conclusion)
}

// A disputed-only marriage is an alternative, not a conflict about the
// accepted one; evidence lines name the other party (#1474, #1492).
func TestBuildProof_DisputedSpouseIsAlternative(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"eva": proofPerson("Eva", SexFemale), "hans": proofPerson("Hans Roth", SexMale), "jo": proofPerson("Johannes Roth", SexMale)},
		Relationships: map[string]*Relationship{
			"m-hans": {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "hans", Role: ParticipantRoleSpouse}, {Person: "eva", Role: ParticipantRoleSpouse}}},
			"m-jo":   {Type: RelationshipTypeMarriage, Participants: []Participant{{Person: "jo", Role: ParticipantRoleSpouse}, {Person: "eva", Role: ParticipantRoleSpouse}}},
		},
		Assertions: map[string]*Assertion{
			"a-hans": {Subject: EntityRef{Relationship: "m-hans"}, Participant: &Participant{Person: "eva", Role: ParticipantRoleSpouse}, Confidence: "medium", Status: "probable"},
			"a-jo":   {Subject: EntityRef{Relationship: "m-jo"}, Participant: &Participant{Person: "eva", Role: ParticipantRoleSpouse}, Confidence: "low", Status: "disputed"},
		},
	}
	result := mustProof(t, archive, "eva", QuestionMarriage)
	assert.Equal(t, ConclusionProbable, result.Conclusion)
	assert.Empty(t, result.Conflicts)
	require.Len(t, result.Excluded, 1)
	assert.Equal(t, ProofStatusDisputed, result.Excluded[0].Status)
	subjects := map[string]string{}
	for _, ev := range result.Evidence {
		subjects[ev.AssertionID] = ev.Subject
	}
	assert.Equal(t, "marriage to Hans Roth (hans) [m-hans]", subjects["a-hans"])
	assert.Equal(t, "marriage to Johannes Roth (jo) [m-jo]", subjects["a-jo"])

	// Two probable husbands are two answers, both standing.
	archive.Assertions["a-jo"].Status = "probable"
	archive.Assertions["a-jo"].Confidence = "medium"
	result = mustProof(t, archive, "eva", QuestionMarriage)
	assert.Len(t, result.Answers, 2)
	assert.Empty(t, result.Excluded)
}

func TestDistinguishPersonNames(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"a": proofPerson("Johann Anton Schepp", SexMale), "b": proofPerson("Johann Anton Schepp", SexMale), "c": proofPerson("Eva", SexFemale)},
		Events: map[string]*Event{
			"bur":  {Type: EventTypeBurial, Date: "1741", Participants: []Participant{{Person: "b", Role: ParticipantRoleSubject}}},
			"bapt": {Type: EventTypeBaptism, Date: "1690", Participants: []Participant{{Person: "a", Role: ParticipantRoleSubject}}},
		},
	}
	labels := DistinguishPersonNames(archive, []string{"a", "b", "c", "a"})
	assert.Equal(t, "Johann Anton Schepp (b. 1690) a", labels["a"])
	assert.Equal(t, "Johann Anton Schepp (d. 1741) b", labels["b"])
	assert.Equal(t, "Eva", labels["c"])
	archive.Events["bapt"].Participants = []Participant{{Person: "b", Role: ParticipantRoleSubject}}
	labels = DistinguishPersonNames(archive, []string{"a", "b"})
	assert.Equal(t, "Johann Anton Schepp (a)", labels["a"])
	assert.Equal(t, "Johann Anton Schepp (1690 – 1741) b", labels["b"])
}

func TestResearchLogQuestions(t *testing.T) {
	archive := childrenProofArchive()
	archive.Relationships["same"] = &Relationship{Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "c"}, {Person: "k"}}}
	archive.Events = map[string]*Event{
		"bapt": {Type: EventTypeBaptism, Participants: []Participant{{Person: "k", Role: ParticipantRoleSubject}, {Person: "c", Role: ParticipantRoleParent}}},
		"bur":  {Type: EventTypeBurial, Participants: []Participant{{Person: "c", Role: ParticipantRoleSubject}}},
		"wed":  {Type: EventTypeMarriage, Participants: []Participant{{Person: "c", Role: ParticipantRoleBride}}},
	}
	cases := []struct {
		subject EntityRef
		person  string
		props   map[string]any
		want    []string
		about   bool
	}{
		{EntityRef{Relationship: "m"}, "c", nil, []string{topicMarriage, topicChildren}, true},
		{EntityRef{Relationship: "pc"}, "k", nil, []string{topicParentage}, true},
		{EntityRef{Relationship: "pc"}, "c", nil, []string{topicChildren}, true},
		{EntityRef{Relationship: "same"}, "c", nil, []string{topicIdentity}, true},
		{EntityRef{Event: "bapt"}, "k", nil, []string{topicBirth, topicParentage}, true},
		{EntityRef{Event: "bapt"}, "c", nil, []string{topicChildren}, true},
		{EntityRef{Event: "bur"}, "c", nil, []string{topicDeath}, true},
		{EntityRef{Event: "wed"}, "c", nil, []string{topicMarriage}, true},
		{EntityRef{Person: "c"}, "c", nil, nil, true},
		{EntityRef{Person: "c"}, "c", map[string]any{"question": "parents, Death, bogus"}, []string{topicParentage, topicDeath}, true},
		{EntityRef{Person: "c"}, "c", map[string]any{"question": []string{"issue"}}, []string{topicChildren}, true},
		{EntityRef{Person: "c"}, "h", nil, nil, false},
		{EntityRef{Relationship: "missing"}, "c", nil, nil, false},
		{EntityRef{Event: "missing"}, "c", nil, nil, false},
	}
	for _, tc := range cases {
		got, about := researchLogQuestions(&ResearchLog{Subject: &tc.subject, Properties: tc.props}, tc.person, archive)
		assert.Equal(t, tc.about, about, "%+v", tc.subject)
		assert.Equal(t, tc.want, got, "%+v", tc.subject)
	}
	_, about := researchLogQuestions(&ResearchLog{}, "c", archive)
	assert.False(t, about)
}

// Searched gaps carry their results and leave the next steps (#1368).
func TestMarkSearchedGapsAndProbatePriority(t *testing.T) {
	gaps := []ProofGap{{Label: "Church records"}, {Label: "Probate/will", Priority: severityHigh}, {Label: "1850 US Census"}, {Label: "Land records"}, {Label: "Military records"}, {Label: "Birth record"}, {Label: "Unknown"}}
	markSearchedGaps(gaps, []ProofSearch{
		{SourceType: SourceTypeChurchRegister, Result: SearchResultNotFound},
		{SourceType: SourceTypeChurchRegister, Result: SearchResultUnavailable},
		{SourceType: SourceTypeProbate},
		{SourceType: SourceTypeCensus, Result: SearchResultNotFound},
	})
	assert.Equal(t, "2 searches: 1 not_found, 1 unavailable", gaps[0].Searched)
	assert.Equal(t, "1 search: 1 searched", gaps[1].Searched)
	assert.Equal(t, "1 search: 1 not_found", gaps[2].Searched)
	assert.Empty(t, gaps[3].Searched)
	assert.Empty(t, gaps[4].Searched)
	assert.Equal(t, gaps[0].Searched, gaps[5].Searched)
	assert.Empty(t, gaps[6].Searched)
	assert.Equal(t, "Done.", appendNextSteps("Done.", gaps), "a searched high-priority gap no longer heads the next steps")

	// The person's own probate is not a high-priority parentage lead.
	archive := newTestArchiveForProof()
	archive.Persons["person-robert"].Properties[PersonPropertySex] = SexMale
	result := mustProof(t, archive, "person-robert", QuestionParentage)
	for _, gap := range result.Gaps {
		if gap.Label == "Probate/will" {
			assert.NotEqual(t, severityHigh, gap.Priority)
		}
	}

	_, ok := negativeSearchSummary(nil)
	assert.False(t, ok)
	_, ok = negativeSearchSummary([]ProofSearch{{Result: SearchResultFound}})
	assert.False(t, ok)
}
