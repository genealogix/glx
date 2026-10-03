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
	"bytes"
	"io"
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

func TestProofFormatKeys(t *testing.T) {
	assert.Equal(t, []string{"text", "json", "markdown", "md"}, proofFormatKeys())
}

// --- End-to-end tests of showProof against the maintained example archive ---

func TestShowProof_JSON(t *testing.T) {
	streams, out, _ := TestIOStreams()
	require.NoError(t, showProof(streams, exampleArchive, "person-robert-chen", "birth", "json"))
	assert.Contains(t, out.String(), `"conclusion": "PROVEN"`)
	assert.Contains(t, out.String(), `"resolution": "1955-03-22"`)
}

func TestShowProof_Markdown(t *testing.T) {
	streams, out, _ := TestIOStreams()
	require.NoError(t, showProof(streams, exampleArchive, "person-alice-chen", "parentage", "markdown"))
	assert.Contains(t, out.String(), "# Proof Summary:")
	assert.Contains(t, out.String(), "## Evidence Collected")
}

func TestShowProof_Text(t *testing.T) {
	streams, out, _ := TestIOStreams()
	require.NoError(t, showProof(streams, exampleArchive, "person-robert-chen", "birth", ""))
	assert.Contains(t, out.String(), "Proof Summary:")
	assert.Contains(t, out.String(), "Conclusion: PROVEN")
	assert.Contains(t, out.String(), "RESOLVED: 1955-03-22")
}

// TestShowProof_StreamRouting verifies the IOStreams contract: human-readable
// text routes to the diagnostic stream (Out, silenced by --quiet), while
// machine-consumable JSON routes to MachineOut (survives --quiet for shell
// capture). Neither stream may leak into the other.
func TestShowProof_StreamRouting(t *testing.T) {
	t.Run("json to MachineOut", func(t *testing.T) {
		diag, machine := &bytes.Buffer{}, &bytes.Buffer{}
		streams := &IOStreams{Out: diag, MachineOut: machine, ErrOut: io.Discard}
		require.NoError(t, showProof(streams, exampleArchive, "person-robert-chen", "birth", "json"))
		assert.Contains(t, machine.String(), `"conclusion": "PROVEN"`)
		assert.Empty(t, diag.String(), "JSON must not leak to the diagnostic stream")
	})

	t.Run("text to Out", func(t *testing.T) {
		diag, machine := &bytes.Buffer{}, &bytes.Buffer{}
		streams := &IOStreams{Out: diag, MachineOut: machine, ErrOut: io.Discard}
		require.NoError(t, showProof(streams, exampleArchive, "person-robert-chen", "birth", "text"))
		assert.Contains(t, diag.String(), "Conclusion: PROVEN")
		assert.Empty(t, machine.String(), "diagnostic text must not leak to the machine stream")
	})
}

func TestShowProof_Errors(t *testing.T) {
	streams, _, _ := TestIOStreams()

	// Unknown question is rejected before loading the archive.
	err := showProof(streams, exampleArchive, "person-robert-chen", "spaceflight", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown research question")

	// Unknown format is rejected after a successful load, and the error advertises
	// the full accepted set including the `md` alias.
	err = showProof(streams, exampleArchive, "person-robert-chen", "birth", "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown format")
	assert.Contains(t, err.Error(), "markdown, md", "the md alias must be advertised in the valid-format set")

	// Unknown person.
	err = showProof(streams, exampleArchive, "person-nobody", "birth", "")
	require.Error(t, err)

	// Empty question.
	err = showProof(streams, exampleArchive, "person-robert-chen", "", "")
	require.Error(t, err)
}

func TestPrintProofText_NoEvidence(t *testing.T) {
	result := &proofResult{
		PersonID:     "person-x",
		QuestionText: "Who?",
		Conclusion:   proofConclusionInsufficient,
		Summary:      "Nothing known.",
	}
	streams, out, _ := TestIOStreams()
	printProofText(streams, result)
	assert.Contains(t, out.String(), "(none)")
	assert.Contains(t, out.String(), "None identified")
	assert.Contains(t, out.String(), "INSUFFICIENT EVIDENCE")
}

// TestRenderProof_FullResult exercises the text and Markdown renderers across
// every branch: multi-citation and uncited evidence, high-priority gaps, logged
// searches (with and without targets), and an unresolved subject-scoped conflict.
func TestRenderProof_FullResult(t *testing.T) {
	result := &proofResult{
		PersonID:     "person-x",
		PersonName:   "Test Person",
		Question:     "death",
		QuestionText: "When and where did Test Person die?",
		Evidence: []proofEvidence{
			{
				AssertionID: "a-cited",
				Subject:     "death event (1890)",
				Property:    "date",
				Value:       "1890",
				Confidence:  "high",
				Status:      "proven",
				Notes:       "From death certificate.",
				Support: []proofSupport{
					{Ref: "cit-1", SourceID: "src-1", SourceTitle: "Death Certificate", Locator: "p.3"},
					{Ref: "cit-2"},
				},
			},
			{AssertionID: "a-uncited", Property: "place", Value: "Boston"},
		},
		Gaps: []proofGap{
			{Label: "Probate/will", Priority: "high", Description: "names heirs"},
			{Label: "Church records"},
		},
		Searches: []proofSearch{
			{Query: "Test 1890 death", Source: "src-1", Result: "found"},
			{Collection: "Vital Index", Repository: "repo-1", Result: "not_found"},
			{},
		},
		Conflicts: []proofConflict{
			{
				Subject:  "death event (1890)",
				Property: "date",
				Values: []proofConflictValue{
					{Value: "1890", Confidence: "high"},
					{Value: "1891", Confidence: "medium"},
				},
				Resolved: false,
			},
		},
		Conclusion: proofConclusionConflicted,
		Summary:    "Conflicting evidence remains unresolved.",
	}

	textStreams, text, _ := TestIOStreams()
	printProofText(textStreams, result)
	assert.Contains(t, text.String(), "Death Certificate (cit-1) +1 more")
	assert.Contains(t, text.String(), "Uncited assertion (a-uncited)")
	assert.Contains(t, text.String(), "-- HIGH PRIORITY")
	assert.Contains(t, text.String(), "Reasonably Exhaustive Search")
	assert.Contains(t, text.String(), "Test 1890 death @ src-1 -> found")
	assert.Contains(t, text.String(), "(search)")
	assert.Contains(t, text.String(), "death event (1890) date:")
	assert.Contains(t, text.String(), "UNRESOLVED")

	mdStreams, md, _ := TestIOStreams()
	printProofMarkdown(mdStreams, result)
	assert.Contains(t, md.String(), "## Reasonably Exhaustive Search")
	assert.Contains(t, md.String(), "Vital Index")
	assert.Contains(t, md.String(), "## Conflicts")
	assert.Contains(t, md.String(), "death event (1890) date")
	assert.Contains(t, md.String(), "Unresolved")
	assert.Contains(t, md.String(), "**HIGH PRIORITY**")
}
