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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// exampleArchive is the maintained assertion-workflow example, used for
// end-to-end tests of the I/O wrapper and format dispatch.
const exampleArchive = "../docs/examples/assertion-workflow/archive.glx"

// newTestArchiveForProof builds an in-memory archive exercising parentage,
// a resolved birth conflict, an unresolved birth conflict, a death with
// medium-confidence support, and a research log.
func newTestArchiveForProof() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane":   {Properties: map[string]any{glxlib.PersonPropertyName: "Jane Webb"}},
			"person-robert": {Properties: map[string]any{glxlib.PersonPropertyName: "Robert Webb"}},
			"person-mary":   {Properties: map[string]any{glxlib.PersonPropertyName: "Mary Webb"}},
			"person-conf":   {Properties: map[string]any{glxlib.PersonPropertyName: "Conf Person"}},
			"person-lonely": {Properties: map[string]any{glxlib.PersonPropertyName: "Lonely Person"}},
		},
		Events: map[string]*glxlib.Event{
			"event-jane-birth": {
				Type: glxlib.EventTypeBirth, Date: "1850", PlaceID: "place-fl",
				Participants: []glxlib.Participant{{Person: "person-jane", Role: "subject"}},
			},
			"event-robert-birth": {
				Type: glxlib.EventTypeBirth, Date: "1820",
				Participants: []glxlib.Participant{{Person: "person-robert", Role: "subject"}},
			},
			"event-robert-death": {
				Type: glxlib.EventTypeDeath, Date: "1890", PlaceID: "place-wi",
				Participants: []glxlib.Participant{{Person: "person-robert", Role: "subject"}},
			},
			"event-conf-birth": {
				Type:         glxlib.EventTypeBirth,
				Participants: []glxlib.Participant{{Person: "person-conf", Role: "subject"}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-jane-robert": {
				Type: glxlib.RelationshipTypeParentChild,
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: glxlib.ParticipantRoleParent},
					{Person: "person-jane", Role: glxlib.ParticipantRoleChild},
				},
			},
			"rel-jane-mary": {
				Type: glxlib.RelationshipTypeParentChild,
				Participants: []glxlib.Participant{
					{Person: "person-mary", Role: glxlib.ParticipantRoleParent},
					{Person: "person-jane", Role: glxlib.ParticipantRoleChild},
				},
			},
			"rel-robert-mary-marriage": {
				Type: glxlib.RelationshipTypeMarriage,
				Participants: []glxlib.Participant{
					{Person: "person-robert", Role: glxlib.ParticipantRoleSpouse},
					{Person: "person-mary", Role: glxlib.ParticipantRoleSpouse},
				},
			},
		},
		Sources: map[string]*glxlib.Source{
			"source-1880-census": {Title: "1880 US Census", Type: glxlib.SourceTypeCensus},
			"source-birth-cert":  {Title: "Birth Certificate", Type: glxlib.SourceTypeVitalRecord},
			"source-lore":        {Title: "Family lore", Type: "oral_history"},
			"source-death-rec":   {Title: "Death Record", Type: glxlib.SourceTypeVitalRecord},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-1880":  {SourceID: "source-1880-census", Properties: map[string]any{"locator": "Sheet 3"}},
			"cit-birth": {SourceID: "source-birth-cert"},
			"cit-lore":  {SourceID: "source-lore"},
			"cit-death": {SourceID: "source-death-rec"},
		},
		Assertions: map[string]*glxlib.Assertion{
			// Parentage: existential assertion on the parent-child relationship.
			"assertion-jane-parentage": {
				Subject:    glxlib.EntityRef{Relationship: "rel-jane-robert"},
				Citations:  []string{"cit-1880"},
				Confidence: "high",
			},
			// Birth conflict, resolved: certificate (proven) vs lore (disproven).
			"assertion-robert-birth": {
				Subject:  glxlib.EntityRef{Event: "event-robert-birth"},
				Property: "date", Value: "1820",
				Citations: []string{"cit-birth"}, Confidence: "high", Status: "proven",
				Notes: glxlib.NoteList{"Primary source birth certificate."},
			},
			"assertion-robert-birth-lore": {
				Subject:  glxlib.EntityRef{Event: "event-robert-birth"},
				Property: "date", Value: "1821",
				Citations: []string{"cit-lore"}, Confidence: "low", Status: "disproven",
			},
			// Death, medium confidence (probable).
			"assertion-robert-death": {
				Subject:  glxlib.EntityRef{Event: "event-robert-death"},
				Property: "date", Value: "1890",
				Citations: []string{"cit-death"}, Confidence: "medium",
			},
			// Birth conflict, unresolved: two high-confidence competing values.
			"assertion-conf-birth-a": {
				Subject:  glxlib.EntityRef{Event: "event-conf-birth"},
				Property: "date", Value: "1900",
				Citations: []string{"cit-1880"}, Confidence: "high",
			},
			"assertion-conf-birth-b": {
				Subject:  glxlib.EntityRef{Event: "event-conf-birth"},
				Property: "date", Value: "1905",
				Citations: []string{"cit-birth"}, Confidence: "high",
			},
		},
		ResearchLogs: map[string]*glxlib.ResearchLog{
			"log-jane": {
				Subject: &glxlib.EntityRef{Person: "person-jane"},
				Searches: []glxlib.Search{
					{Query: "Jane Webb 1850 census", SourceID: "source-1880-census", Result: glxlib.SearchResultFound},
					{Query: "Jane Webb FL birth", Result: glxlib.SearchResultNotFound},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-fl": {Name: "Florida"},
			"place-wi": {Name: "Wisconsin"},
		},
	}
}

func TestCanonicalQuestion(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"parentage", "parentage", true},
		{"PARENTS", "parentage", true},
		{"  Father ", "parentage", true},
		{"born", "birth", true},
		{"birthdate", "birth", true},
		{"died", "death", true},
		{"burial", "death", true},
		{"spouse", "marriage", true},
		{"name", "identity", true},
		{"", "", false},
		{"foobar", "", false},
	}
	for _, tc := range cases {
		got, ok := canonicalQuestion(tc.in)
		assert.Equal(t, tc.wantOK, ok, "ok for %q", tc.in)
		assert.Equal(t, tc.want, got, "topic for %q", tc.in)
	}
}

func TestProofQuestionKeys(t *testing.T) {
	keys := proofQuestionKeys()
	assert.Equal(t, []string{"parentage", "birth", "death", "marriage", "identity"}, keys)
}

func TestQuestionText(t *testing.T) {
	assert.Equal(t, "Who are the parents of Jane Webb?", questionText("parentage", "Jane Webb"))
	assert.Equal(t, "When and where did Robert Webb die?", questionText("death", "Robert Webb"))
	assert.Equal(t, "Jane Webb", questionText("unknown", "Jane Webb"))
}

func TestAssertionRelevant(t *testing.T) {
	parentRel := &proofAssertion{a: &glxlib.Assertion{}, relType: glxlib.RelationshipTypeParentChild, personRole: glxlib.ParticipantRoleChild}
	parentRelAsParent := &proofAssertion{a: &glxlib.Assertion{}, relType: glxlib.RelationshipTypeParentChild, personRole: glxlib.ParticipantRoleParent}
	birthEvent := &proofAssertion{a: &glxlib.Assertion{Property: "date"}, eventType: glxlib.EventTypeBirth}
	deathEvent := &proofAssertion{a: &glxlib.Assertion{Property: "date"}, eventType: glxlib.EventTypeDeath}
	burialEvent := &proofAssertion{a: &glxlib.Assertion{}, eventType: glxlib.EventTypeBurial}
	marriageRel := &proofAssertion{a: &glxlib.Assertion{}, relType: glxlib.RelationshipTypeMarriage}
	partnerRel := &proofAssertion{a: &glxlib.Assertion{}, relType: glxlib.RelationshipTypePartner}
	nameProp := &proofAssertion{a: &glxlib.Assertion{Property: glxlib.PersonPropertyName}}
	occProp := &proofAssertion{a: &glxlib.Assertion{Property: "occupation"}}

	assert.True(t, assertionRelevant("parentage", parentRel))
	assert.False(t, assertionRelevant("parentage", parentRelAsParent), "parent-as-parent is not the child's parentage evidence")
	assert.False(t, assertionRelevant("parentage", marriageRel))

	assert.True(t, assertionRelevant("birth", birthEvent))
	assert.False(t, assertionRelevant("birth", deathEvent))

	assert.True(t, assertionRelevant("death", deathEvent))
	assert.True(t, assertionRelevant("death", burialEvent))
	assert.False(t, assertionRelevant("death", birthEvent))

	assert.True(t, assertionRelevant("marriage", marriageRel))
	assert.True(t, assertionRelevant("marriage", partnerRel))

	assert.True(t, assertionRelevant("identity", nameProp))
	assert.False(t, assertionRelevant("identity", occProp))
}

func TestBuildProof_BirthResolvedConflict(t *testing.T) {
	archive := newTestArchiveForProof()
	result := buildProof("person-robert", archive.Persons["person-robert"], "birth", archive)

	require.Len(t, result.Evidence, 2)
	require.Len(t, result.Conflicts, 1)
	assert.True(t, result.Conflicts[0].Resolved)
	assert.Equal(t, "1820", result.Conflicts[0].Resolution)
	assert.Equal(t, proofConclusionProven, result.Conclusion)
	assert.Contains(t, result.Summary, "1820")

	// Citation is resolved to its source title and locator.
	ev := result.Evidence[0]
	require.NotEmpty(t, ev.Support)
	assert.Equal(t, "Birth Certificate", ev.Support[0].SourceTitle)
}

func TestBuildProof_UnresolvedConflict(t *testing.T) {
	archive := newTestArchiveForProof()
	result := buildProof("person-conf", archive.Persons["person-conf"], "birth", archive)

	require.Len(t, result.Conflicts, 1)
	assert.False(t, result.Conflicts[0].Resolved)
	assert.Equal(t, proofConclusionConflicted, result.Conclusion)
}

func TestBuildProof_Parentage(t *testing.T) {
	archive := newTestArchiveForProof()
	result := buildProof("person-jane", archive.Persons["person-jane"], "parentage", archive)

	assert.Equal(t, proofConclusionProven, result.Conclusion)
	assert.Contains(t, result.Summary, "Robert Webb")
	assert.Contains(t, result.Summary, "Mary Webb")
	// The existential parentage assertion is collected as evidence.
	require.Len(t, result.Evidence, 1)
	assert.Equal(t, "assertion-jane-parentage", result.Evidence[0].AssertionID)
}

func TestBuildProof_DeathProbable(t *testing.T) {
	archive := newTestArchiveForProof()
	result := buildProof("person-robert", archive.Persons["person-robert"], "death", archive)

	assert.Equal(t, proofConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "1890")
	assert.Contains(t, result.Summary, "Wisconsin")
}

func TestBuildProof_DeathInsufficient(t *testing.T) {
	archive := newTestArchiveForProof()
	// Jane has no death event and no death assertions.
	result := buildProof("person-jane", archive.Persons["person-jane"], "death", archive)

	assert.Empty(t, result.Evidence)
	assert.Equal(t, proofConclusionInsufficient, result.Conclusion)
}

func TestBuildProof_IdentityInsufficientWithoutAssertion(t *testing.T) {
	archive := newTestArchiveForProof()
	// Lonely Person has a name (answer derivable) but no supporting assertions.
	result := buildProof("person-lonely", archive.Persons["person-lonely"], "identity", archive)

	assert.Empty(t, result.Evidence)
	assert.Equal(t, proofConclusionInsufficient, result.Conclusion)
}

func TestCollectProofSearches(t *testing.T) {
	archive := newTestArchiveForProof()
	searches := collectProofSearches("person-jane", archive)

	require.Len(t, searches, 2)
	assert.Equal(t, glxlib.SearchResultFound, searches[0].Result)
	assert.Equal(t, glxlib.SearchResultNotFound, searches[1].Result)

	// A person with no research log has no searches.
	assert.Empty(t, collectProofSearches("person-robert", archive))
}

func TestResolveConflict(t *testing.T) {
	resolved, resolution := resolveConflict([]proofConflictValue{
		{Value: "1820", Status: "proven"},
		{Value: "1821", Status: "disproven"},
	})
	assert.True(t, resolved)
	assert.Equal(t, "1820", resolution)

	unresolved, _ := resolveConflict([]proofConflictValue{
		{Value: "1900", Confidence: "high"},
		{Value: "1905", Confidence: "high"},
	})
	assert.False(t, unresolved)

	disputed, _ := resolveConflict([]proofConflictValue{
		{Value: "A", Status: "disputed"},
		{Value: "B", Status: "disproven"},
	})
	assert.False(t, disputed, "a disputed value is never auto-resolved")
}

func TestSupportLevel(t *testing.T) {
	proven := []proofAssertion{{a: &glxlib.Assertion{Status: "proven"}}}
	assert.Equal(t, supportStrong, supportLevel(proven))

	high := []proofAssertion{{a: &glxlib.Assertion{Confidence: "high"}}}
	assert.Equal(t, supportStrong, supportLevel(high))

	medium := []proofAssertion{{a: &glxlib.Assertion{Confidence: "medium"}}}
	assert.Equal(t, supportModerate, supportLevel(medium))

	low := []proofAssertion{{a: &glxlib.Assertion{Confidence: "low"}}}
	assert.Equal(t, supportWeak, supportLevel(low))

	none := []proofAssertion{}
	assert.Equal(t, supportNone, supportLevel(none))

	// A disproven assertion provides no support.
	disproven := []proofAssertion{{a: &glxlib.Assertion{Confidence: "high", Status: "disproven"}}}
	assert.Equal(t, supportNone, supportLevel(disproven))
}

func TestGapRelevant(t *testing.T) {
	census := &coverageRecord{Category: "census", Label: "1880 US Census (age ~30)"}
	birth := &coverageRecord{Category: "vital", Label: "Birth record"}
	death := &coverageRecord{Category: "vital", Label: "Death record"}
	marriage := &coverageRecord{Category: "vital", Label: "Marriage record — Mary"}

	assert.True(t, gapRelevant("parentage", census))
	assert.True(t, gapRelevant("parentage", death))
	assert.False(t, gapRelevant("death", census))
	assert.True(t, gapRelevant("death", death))
	assert.True(t, gapRelevant("birth", birth))
	assert.True(t, gapRelevant("marriage", marriage))
	assert.False(t, gapRelevant("marriage", census))
	assert.True(t, gapRelevant("identity", census))
}

func TestResolveProofValue(t *testing.T) {
	archive := newTestArchiveForProof()
	assert.Equal(t, "Florida", resolveProofValue("place-fl", archive))
	assert.Equal(t, "1850", resolveProofValue("1850", archive))
	assert.Empty(t, resolveProofValue("", archive))
}

func TestParentNames(t *testing.T) {
	archive := newTestArchiveForProof()
	names := parentNames("person-jane", archive)
	assert.ElementsMatch(t, []string{"Robert Webb", "Mary Webb"}, names)
	assert.Empty(t, parentNames("person-robert", archive))
}

// --- End-to-end tests of showProof against the maintained example archive ---

func TestShowProof_JSON(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, showProof(exampleArchive, "person-robert-chen", "birth", "json"))
	})
	assert.Contains(t, out, `"conclusion": "PROVEN"`)
	assert.Contains(t, out, `"resolution": "1955-03-22"`)
}

func TestShowProof_Markdown(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, showProof(exampleArchive, "person-alice-chen", "parentage", "markdown"))
	})
	assert.Contains(t, out, "# Proof Summary:")
	assert.Contains(t, out, "## Evidence Collected")
}

func TestShowProof_Text(t *testing.T) {
	out := captureStdout(t, func() {
		require.NoError(t, showProof(exampleArchive, "person-robert-chen", "birth", ""))
	})
	assert.Contains(t, out, "Proof Summary:")
	assert.Contains(t, out, "Conclusion: PROVEN")
	assert.Contains(t, out, "RESOLVED: 1955-03-22")
}

func TestShowProof_Errors(t *testing.T) {
	// Unknown question is rejected before loading the archive.
	err := showProof(exampleArchive, "person-robert-chen", "spaceflight", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown research question")

	// Unknown format is rejected after a successful load.
	err = showProof(exampleArchive, "person-robert-chen", "birth", "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")

	// Unknown person.
	err = showProof(exampleArchive, "person-nobody", "birth", "")
	require.Error(t, err)

	// Empty question.
	err = showProof(exampleArchive, "person-robert-chen", "", "")
	require.Error(t, err)
}

func TestPrintProofText_NoEvidence(t *testing.T) {
	result := &proofResult{
		PersonID:     "person-x",
		QuestionText: "Who?",
		Conclusion:   proofConclusionInsufficient,
		Summary:      "Nothing known.",
	}
	out := captureStdout(t, func() { printProofText(result) })
	assert.Contains(t, out, "(none)")
	assert.Contains(t, out, "None identified")
	assert.Contains(t, out, "INSUFFICIENT EVIDENCE")
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "first", firstLine("\n  first\nsecond"))
	assert.Empty(t, firstLine(""))
	assert.Equal(t, "only", firstLine("only"))
}
