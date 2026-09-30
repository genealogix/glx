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

// newTestArchiveForProof builds an in-memory archive exercising parentage,
// a resolved birth conflict, an unresolved birth conflict, a death with
// medium-confidence support, and a research log.
func newTestArchiveForProof() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-jane":   {Properties: map[string]any{PersonPropertyName: "Jane Webb"}},
			"person-robert": {Properties: map[string]any{PersonPropertyName: "Robert Webb"}},
			"person-mary":   {Properties: map[string]any{PersonPropertyName: "Mary Webb"}},
			"person-conf":   {Properties: map[string]any{PersonPropertyName: "Conf Person"}},
			"person-lonely": {Properties: map[string]any{PersonPropertyName: "Lonely Person"}},
		},
		Events: map[string]*Event{
			"event-jane-birth": {
				Type: EventTypeBirth, Date: "1850", PlaceID: "place-fl",
				Participants: []Participant{{Person: "person-jane", Role: "subject"}},
			},
			"event-robert-birth": {
				Type: EventTypeBirth, Date: "1820",
				Participants: []Participant{{Person: "person-robert", Role: "subject"}},
			},
			"event-robert-death": {
				Type: EventTypeDeath, Date: "1890", PlaceID: "place-wi",
				Participants: []Participant{{Person: "person-robert", Role: "subject"}},
			},
			"event-conf-birth": {
				Type:         EventTypeBirth,
				Participants: []Participant{{Person: "person-conf", Role: "subject"}},
			},
		},
		Relationships: map[string]*Relationship{
			"rel-jane-robert": {
				Type: RelationshipTypeParentChild,
				Participants: []Participant{
					{Person: "person-robert", Role: ParticipantRoleParent},
					{Person: "person-jane", Role: ParticipantRoleChild},
				},
			},
			"rel-jane-mary": {
				Type: RelationshipTypeParentChild,
				Participants: []Participant{
					{Person: "person-mary", Role: ParticipantRoleParent},
					{Person: "person-jane", Role: ParticipantRoleChild},
				},
			},
			"rel-robert-mary-marriage": {
				Type: RelationshipTypeMarriage,
				Participants: []Participant{
					{Person: "person-robert", Role: ParticipantRoleSpouse},
					{Person: "person-mary", Role: ParticipantRoleSpouse},
				},
			},
		},
		Sources: map[string]*Source{
			"source-1880-census": {Title: "1880 US Census", Type: SourceTypeCensus},
			"source-birth-cert":  {Title: "Birth Certificate", Type: SourceTypeVitalRecord},
			"source-lore":        {Title: "Family lore", Type: "oral_history"},
			"source-death-rec":   {Title: "Death Record", Type: SourceTypeVitalRecord},
		},
		Citations: map[string]*Citation{
			"cit-1880":  {SourceID: "source-1880-census", Properties: map[string]any{"locator": "Sheet 3"}},
			"cit-birth": {SourceID: "source-birth-cert"},
			"cit-lore":  {SourceID: "source-lore"},
			"cit-death": {SourceID: "source-death-rec"},
		},
		Assertions: map[string]*Assertion{
			// Parentage: existential assertion on the parent-child relationship.
			"assertion-jane-parentage": {
				Subject:    EntityRef{Relationship: "rel-jane-robert"},
				Citations:  []string{"cit-1880"},
				Confidence: "high",
			},
			// Birth conflict, resolved: certificate (proven) vs lore (disproven).
			"assertion-robert-birth": {
				Subject:  EntityRef{Event: "event-robert-birth"},
				Property: "date", Value: "1820",
				Citations: []string{"cit-birth"}, Confidence: "high", Status: "proven",
				Notes: NoteList{"Primary source birth certificate."},
			},
			"assertion-robert-birth-lore": {
				Subject:  EntityRef{Event: "event-robert-birth"},
				Property: "date", Value: "1821",
				Citations: []string{"cit-lore"}, Confidence: "low", Status: "disproven",
			},
			// Death, medium confidence (probable).
			"assertion-robert-death": {
				Subject:  EntityRef{Event: "event-robert-death"},
				Property: "date", Value: "1890",
				Citations: []string{"cit-death"}, Confidence: "medium",
			},
			// Birth conflict, unresolved: two high-confidence competing values.
			"assertion-conf-birth-a": {
				Subject:  EntityRef{Event: "event-conf-birth"},
				Property: "date", Value: "1900",
				Citations: []string{"cit-1880"}, Confidence: "high",
			},
			"assertion-conf-birth-b": {
				Subject:  EntityRef{Event: "event-conf-birth"},
				Property: "date", Value: "1905",
				Citations: []string{"cit-birth"}, Confidence: "high",
			},
		},
		ResearchLogs: map[string]*ResearchLog{
			"log-jane": {
				Subject: &EntityRef{Person: "person-jane"},
				Searches: []Search{
					{Query: "Jane Webb 1850 census", SourceID: "source-1880-census", Result: SearchResultFound},
					{Query: "Jane Webb FL birth", Result: SearchResultNotFound},
				},
			},
		},
		Places: map[string]*Place{
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
	parentRel := &proofAssertion{a: &Assertion{}, relType: RelationshipTypeParentChild, personRole: ParticipantRoleChild}
	parentRelAsParent := &proofAssertion{a: &Assertion{}, relType: RelationshipTypeParentChild, personRole: ParticipantRoleParent}
	birthEvent := &proofAssertion{a: &Assertion{Property: "date"}, eventType: EventTypeBirth}
	deathEvent := &proofAssertion{a: &Assertion{Property: "date"}, eventType: EventTypeDeath}
	burialEvent := &proofAssertion{a: &Assertion{}, eventType: EventTypeBurial}
	marriageRel := &proofAssertion{a: &Assertion{}, relType: RelationshipTypeMarriage}
	partnerRel := &proofAssertion{a: &Assertion{}, relType: RelationshipTypePartner}
	nameProp := &proofAssertion{a: &Assertion{Property: PersonPropertyName}}
	occProp := &proofAssertion{a: &Assertion{Property: "occupation"}}

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
	result := testBuildProof("person-robert", archive.Persons["person-robert"], "birth", archive)

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
	result := testBuildProof("person-conf", archive.Persons["person-conf"], "birth", archive)

	require.Len(t, result.Conflicts, 1)
	assert.False(t, result.Conflicts[0].Resolved)
	assert.Equal(t, "birth event", result.Conflicts[0].Subject, "conflict is attributed to its subject")
	assert.Equal(t, proofConclusionConflicted, result.Conclusion)
}

func TestBuildProof_Parentage(t *testing.T) {
	archive := newTestArchiveForProof()
	result := testBuildProof("person-jane", archive.Persons["person-jane"], "parentage", archive)

	assert.Equal(t, proofConclusionProven, result.Conclusion)
	assert.Contains(t, result.Summary, "Robert Webb")
	assert.Contains(t, result.Summary, "Mary Webb")
	// The existential parentage assertion is collected as evidence.
	require.Len(t, result.Evidence, 1)
	assert.Equal(t, "assertion-jane-parentage", result.Evidence[0].AssertionID)
}

func TestBuildProof_DeathProbable(t *testing.T) {
	archive := newTestArchiveForProof()
	result := testBuildProof("person-robert", archive.Persons["person-robert"], "death", archive)

	assert.Equal(t, proofConclusionProbable, result.Conclusion)
	assert.Contains(t, result.Summary, "1890")
	assert.Contains(t, result.Summary, "Wisconsin")
}

func TestBuildProof_DeathInsufficient(t *testing.T) {
	archive := newTestArchiveForProof()
	// Jane has no death event and no death assertions.
	result := testBuildProof("person-jane", archive.Persons["person-jane"], "death", archive)

	assert.Empty(t, result.Evidence)
	assert.Equal(t, proofConclusionInsufficient, result.Conclusion)
}

func TestBuildProof_IdentityInsufficientWithoutAssertion(t *testing.T) {
	archive := newTestArchiveForProof()
	// Lonely Person has a name (answer derivable) but no supporting assertions.
	result := testBuildProof("person-lonely", archive.Persons["person-lonely"], "identity", archive)

	assert.Empty(t, result.Evidence)
	assert.Equal(t, proofConclusionInsufficient, result.Conclusion)
}

func TestCollectProofSearches(t *testing.T) {
	archive := newTestArchiveForProof()
	searches := collectProofSearches("person-jane", archive)

	require.Len(t, searches, 2)
	assert.Equal(t, SearchResultFound, searches[0].Result)
	assert.Equal(t, SearchResultNotFound, searches[1].Result)

	// A person with no research log has no searches.
	assert.Empty(t, collectProofSearches("person-robert", archive))
}

func TestResolveConflict(t *testing.T) {
	resolved, resolution := evaluateTestResolution([]ConflictValue{
		{Value: "1820", Status: "proven"},
		{Value: "1821", Status: "disproven"},
	})
	assert.True(t, resolved)
	assert.Equal(t, "1820", resolution)

	unresolved, _ := evaluateTestResolution([]ConflictValue{
		{Value: "1900", Confidence: "high"},
		{Value: "1905", Confidence: "high"},
	})
	assert.False(t, unresolved)

	disputed, _ := evaluateTestResolution([]ConflictValue{
		{Value: "A", Status: "disputed"},
		{Value: "B", Status: "disproven"},
	})
	assert.False(t, disputed, "a disputed value is never auto-resolved")
}

func TestCombineStatus(t *testing.T) {
	assert.Equal(t, "proven", combineStatus("", "proven"))
	assert.Equal(t, "proven", combineStatus("proven", ""))
	assert.Equal(t, "proven", combineStatus("proven", "proven"))
	assert.Empty(t, combineStatus("", ""))
	// Conflicting decisive statuses for the same value escalate to disputed.
	assert.Equal(t, statusDisputed, combineStatus("proven", "disproven"))
	assert.Equal(t, statusDisputed, combineStatus("proven", "disputed"))
}

// TestDetectProofConflicts_StatusEscalation verifies that when one assertion for
// a value marks it disputed (or two assertions give conflicting decisive
// statuses), the conflict is reported unresolved even if the competing value is
// disproven — distinctProofValues must not let a disproven sibling auto-resolve.
func TestDetectProofConflicts_StatusEscalation(t *testing.T) {
	relevant := []proofAssertion{
		{id: "a1", subjectID: "event-x", eventType: EventTypeBirth, a: &Assertion{Property: "date", Value: "1900", Confidence: "high", Status: "proven"}},
		{id: "a2", subjectID: "event-x", eventType: EventTypeBirth, a: &Assertion{Property: "date", Value: "1900", Confidence: "medium", Status: "disputed"}},
		{id: "a3", subjectID: "event-x", eventType: EventTypeBirth, a: &Assertion{Property: "date", Value: "1901", Status: "disproven"}},
	}
	conflicts := detectProofConflicts(relevant, &GLXFile{})

	require.Len(t, conflicts, 1)
	assert.False(t, conflicts[0].Resolved, "value 1900 is itself disputed, so the conflict must stay unresolved")
	// The conflict is attributed to its subject (GPS disambiguation).
	assert.Equal(t, "birth event", conflicts[0].Subject)
}

// TestDetectProofConflicts_DistinctPlacesSameName verifies that two different
// place IDs sharing a display name stay distinct (so the conflict is detected)
// and are qualified by their raw ID in the output, instead of collapsing into a
// single "Springfield" value that would hide the disagreement.
func TestDetectProofConflicts_DistinctPlacesSameName(t *testing.T) {
	archive := &GLXFile{
		Places: map[string]*Place{
			"place-il-springfield": {Name: "Springfield"},
			"place-mo-springfield": {Name: "Springfield"},
		},
	}
	relevant := []proofAssertion{
		{id: "a1", subjectID: "event-x", eventType: EventTypeBirth, a: &Assertion{Subject: EntityRef{Event: "event-x"}, Property: "place", Value: "place-il-springfield", Confidence: "high"}},
		{id: "a2", subjectID: "event-x", eventType: EventTypeBirth, a: &Assertion{Subject: EntityRef{Event: "event-x"}, Property: "place", Value: "place-mo-springfield", Confidence: "high"}},
	}
	conflicts := detectProofConflicts(relevant, archive)

	require.Len(t, conflicts, 1, "two distinct places must not collapse into one value")
	require.Len(t, conflicts[0].Values, 2)
	assert.False(t, conflicts[0].Resolved)
	assert.ElementsMatch(t,
		[]string{"Springfield (place-il-springfield)", "Springfield (place-mo-springfield)"},
		[]string{conflicts[0].Values[0].Value, conflicts[0].Values[1].Value},
		"colliding display names are disambiguated by raw ID",
	)
}

func TestSupportLevel(t *testing.T) {
	proven := []proofAssertion{{a: &Assertion{Status: "proven"}}}
	assert.Equal(t, supportStrong, supportLevel(proven))

	high := []proofAssertion{{a: &Assertion{Confidence: "high"}}}
	assert.Equal(t, supportStrong, supportLevel(high))

	medium := []proofAssertion{{a: &Assertion{Confidence: "medium"}}}
	assert.Equal(t, supportModerate, supportLevel(medium))

	low := []proofAssertion{{a: &Assertion{Confidence: "low"}}}
	assert.Equal(t, supportWeak, supportLevel(low))

	none := []proofAssertion{}
	assert.Equal(t, supportNone, supportLevel(none))

	// A disproven assertion provides no support.
	disproven := []proofAssertion{{a: &Assertion{Confidence: "high", Status: "disproven"}}}
	assert.Equal(t, supportNone, supportLevel(disproven))
}

func TestGapRelevant(t *testing.T) {
	census := &CoverageRecord{Category: "census", Label: "1880 US Census (age ~30)"}
	birth := &CoverageRecord{Category: "vital", Label: "Birth record"}
	death := &CoverageRecord{Category: "vital", Label: "Death record"}
	marriage := &CoverageRecord{Category: "vital", Label: "Marriage record — Mary"}

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
	assert.Equal(t, "Florida", resolveProofValue("place-fl", EntityRef{Event: "birth"}, "place", archive))
	assert.Equal(t, "1850", resolveProofValue("1850", EntityRef{Event: "birth"}, "date", archive))
	assert.Empty(t, resolveProofValue("", EntityRef{Event: "birth"}, "place", archive))
}

func TestParentNames(t *testing.T) {
	archive := newTestArchiveForProof()
	names := parentNames("person-jane", archive)
	assert.ElementsMatch(t, []string{"Robert Webb", "Mary Webb"}, names)
	assert.Empty(t, parentNames("person-robert", archive))
}

func TestFirstLine(t *testing.T) {
	assert.Equal(t, "first", firstLine("\n  first\nsecond"))
	assert.Empty(t, firstLine(""))
	assert.Equal(t, "only", firstLine("only"))
}

func TestIsMarriageEventType(t *testing.T) {
	assert.True(t, isMarriageEventType(EventTypeMarriage))
	assert.True(t, isMarriageEventType(EventTypeEngagement))
	assert.True(t, isMarriageEventType(EventTypeMarriageLicense))
	assert.False(t, isMarriageEventType(EventTypeBirth))
	assert.False(t, isMarriageEventType(""))
}

func TestDeriveProofAnswer(t *testing.T) {
	archive := newTestArchiveForProof()

	ans, found := deriveProofAnswer("marriage", "person-robert", archive)
	assert.True(t, found)
	assert.Contains(t, ans, "Mary Webb")

	_, found = deriveProofAnswer("marriage", "person-lonely", archive)
	assert.False(t, found, "no spouse relationship")

	ans, found = deriveProofAnswer("identity", "person-jane", archive)
	assert.True(t, found)
	assert.Contains(t, ans, "Jane Webb")

	_, found = deriveProofAnswer("birth", "person-lonely", archive)
	assert.False(t, found, "no birth event")

	_, found = deriveProofAnswer("death", "person-jane", archive)
	assert.False(t, found, "no death event")

	_, found = deriveProofAnswer("unknown-topic", "person-jane", archive)
	assert.False(t, found)
}

func TestSpouseNames(t *testing.T) {
	archive := newTestArchiveForProof()
	assert.Equal(t, []string{"Mary Webb"}, spouseNames("person-robert", archive))
	assert.Empty(t, spouseNames("person-lonely", archive))
}

// TestConcludeProof_EvidenceFallback verifies that gathered assertions drive the
// conclusion even when the archive's structural data yields no answer — a legacy
// person-subject born_on assertion with no birth event, or an event whose
// Date/PlaceID was never denormalized. Before the fallback these concluded
// INSUFFICIENT EVIDENCE despite evidence being collected and shown.
func TestConcludeProof_EvidenceFallback(t *testing.T) {
	t.Run("legacy person-subject born_on, no birth event", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-legacy": {Properties: map[string]any{PersonPropertyName: "Legacy Person"}},
			},
			Sources:   map[string]*Source{"src": {Title: "Family Bible"}},
			Citations: map[string]*Citation{"cit": {SourceID: "src"}},
			Assertions: map[string]*Assertion{
				"a-born": {
					Subject:  EntityRef{Person: "person-legacy"},
					Property: DeprecatedPropertyBornOn, Value: "1888",
					Citations: []string{"cit"}, Confidence: "high",
				},
			},
		}
		result := testBuildProof("person-legacy", archive.Persons["person-legacy"], "birth", archive)

		require.Len(t, result.Evidence, 1, "the legacy born_on assertion is collected as evidence")
		assert.Equal(t, proofConclusionProven, result.Conclusion, "high-confidence legacy birth assertion should prove birth")
		assert.Contains(t, result.Summary, "1888")
	})

	t.Run("birth event lacking Date/PlaceID, backed by a date assertion", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-x": {Properties: map[string]any{PersonPropertyName: "Person X"}},
			},
			Events: map[string]*Event{
				"event-x-birth": {
					Type:         EventTypeBirth,
					Participants: []Participant{{Person: "person-x", Role: "subject"}},
				},
			},
			Assertions: map[string]*Assertion{
				"a-date": {
					Subject:  EntityRef{Event: "event-x-birth"},
					Property: "date", Value: "1890", Confidence: "low",
				},
			},
		}
		result := testBuildProof("person-x", archive.Persons["person-x"], "birth", archive)

		require.NotEmpty(t, result.Evidence, "the date assertion on the dateless event is collected as evidence")
		assert.Equal(t, proofConclusionPossible, result.Conclusion, "low-confidence evidence yields POSSIBLE, not INSUFFICIENT")
		assert.Contains(t, result.Summary, "1890")
	})

	t.Run("only disproven evidence stays INSUFFICIENT", func(t *testing.T) {
		archive := &GLXFile{
			Persons: map[string]*Person{
				"person-d": {Properties: map[string]any{PersonPropertyName: "Person D"}},
			},
			Assertions: map[string]*Assertion{
				"a-bad": {
					Subject:  EntityRef{Person: "person-d"},
					Property: DeprecatedPropertyBornOn, Value: "1700",
					Confidence: "low", Status: "disproven",
				},
			},
		}
		result := testBuildProof("person-d", archive.Persons["person-d"], "birth", archive)

		assert.Equal(t, proofConclusionInsufficient, result.Conclusion, "disproven-only evidence must not derive an answer")
	})
}

// TestBuildProof_DisputedSingleAssertion verifies that a lone assertion flagged
// `disputed` concludes CONFLICTED even though only one value is recorded — the
// spec allows a single disputed assertion citing conflicting sources, so the
// disagreement must not silently fall through to PROVEN/PROBABLE/POSSIBLE.
func TestBuildProof_DisputedSingleAssertion(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-d": {Properties: map[string]any{PersonPropertyName: "Disputed Dan"}},
		},
		Events: map[string]*Event{
			"event-d-birth": {
				Type:         EventTypeBirth,
				Participants: []Participant{{Person: "person-d", Role: "subject"}},
			},
		},
		Sources:   map[string]*Source{"src": {Title: "Family Bible"}},
		Citations: map[string]*Citation{"cit": {SourceID: "src"}},
		Assertions: map[string]*Assertion{
			"a-disputed": {
				Subject:  EntityRef{Event: "event-d-birth"},
				Property: "date", Value: "1850",
				Citations: []string{"cit"}, Confidence: "medium", Status: "disputed",
			},
		},
	}
	result := testBuildProof("person-d", archive.Persons["person-d"], "birth", archive)

	require.Len(t, result.Evidence, 1, "the disputed assertion is still collected as evidence")
	require.Len(t, result.Conflicts, 1, "a single disputed value remains a known dispute")
	assert.Equal(t, VerdictDisputed, result.Conflicts[0].Verdict)
	assert.Equal(t, proofConclusionConflicted, result.Conclusion, "disputed status must conclude CONFLICTED")
	assert.Contains(t, result.Summary, "unresolved")
}

func TestBuildProof_MarriageWithoutEvidence(t *testing.T) {
	archive := newTestArchiveForProof()
	// A spouse relationship exists, but no assertion backs the marriage, so the
	// conclusion is insufficient even though the spouse can be named.
	result := testBuildProof("person-robert", archive.Persons["person-robert"], "marriage", archive)

	assert.Empty(t, result.Evidence)
	assert.Equal(t, proofConclusionInsufficient, result.Conclusion)
	assert.Contains(t, result.Summary, "No marriage established")
}

func testBuildProof(id string, person *Person, topic string, archive *GLXFile) *ProofResult {
	return buildProof(id, person, topic, archive, ProofOptions{})
}

func evaluateTestResolution(values []ConflictValue) (bool, string) {
	facts := make([]FactValue, len(values))
	for i, v := range values {
		facts[i] = FactValue{Value: v.Value, Status: v.Status, Confidence: v.Confidence}
	}
	result := evaluateFacts(facts, nil, nil, ComparisonOptions{})

	return result.Resolved, result.Resolution
}
