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
	"gopkg.in/yaml.v3"
)

const researchLeadYAML = `research_logs:
  research-log-mary-green-parents:
    subject:
      person: person-mary-green
    objective: Identify the parents of Mary Green
    status: in_progress
    leads:
      - description: John H. Green of Wheeling, VA
        persons:
          - person-john-h-green
        status: active
        confidence: medium
        evidence_for:
          - Daughter aged 5-10 in the 1840 census
        evidence_against:
          - Father born in England, not connected to the NY Greens
        citations:
          - citation-1840-census
        assertions:
          - assertion-mary-father
        next_steps:
          - Search 1850 census, Ohio County, VA
        notes: Strongest remaining candidate
      - description: Luther Green of Springwater, NY
        status: eliminated
`

func TestResearchLeadRoundTrip(t *testing.T) {
	var glx GLXFile
	require.NoError(t, yaml.Unmarshal([]byte(researchLeadYAML), &glx))

	log := glx.ResearchLogs["research-log-mary-green-parents"]
	require.NotNil(t, log)
	require.Len(t, log.Leads, 2)
	lead := log.Leads[0]
	assert.Equal(t, "John H. Green of Wheeling, VA", lead.Description)
	assert.Equal(t, []string{"person-john-h-green"}, lead.Persons)
	assert.Equal(t, LeadStatusActive, lead.Status)
	assert.Equal(t, "medium", lead.Confidence)
	assert.Equal(t, []string{"Daughter aged 5-10 in the 1840 census"}, lead.EvidenceFor)
	assert.Equal(t, []string{"Father born in England, not connected to the NY Greens"}, lead.EvidenceAgainst)
	assert.Equal(t, []string{"citation-1840-census"}, lead.Citations)
	assert.Equal(t, []string{"assertion-mary-father"}, lead.Assertions)
	assert.Equal(t, []string{"Search 1850 census, Ohio County, VA"}, lead.NextSteps)
	assert.Equal(t, NoteList{"Strongest remaining candidate"}, lead.Notes)
	assert.Equal(t, LeadStatusEliminated, log.Leads[1].Status)

	out, err := yaml.Marshal(&glx)
	require.NoError(t, err)
	var glx2 GLXFile
	require.NoError(t, yaml.Unmarshal(out, &glx2))
	assert.Equal(t, log.Leads, glx2.ResearchLogs["research-log-mary-green-parents"].Leads)
}

func TestResearchLeadSerializerRoundTrip(t *testing.T) {
	var glx GLXFile
	require.NoError(t, yaml.Unmarshal([]byte(researchLeadYAML), &glx))

	s := NewSerializer(&SerializerOptions{Validate: false})
	files, err := s.SerializeMultiFileToMap(&glx)
	require.NoError(t, err)
	data, ok := files["research_logs/research-log-mary-green-parents.glx"]
	require.True(t, ok)

	back, _, err := s.DeserializeMultiFileFromMap(map[string][]byte{
		"research_logs/research-log-mary-green-parents.glx": data,
	})
	require.NoError(t, err)
	assert.Equal(t, glx.ResearchLogs["research-log-mary-green-parents"].Leads,
		back.ResearchLogs["research-log-mary-green-parents"].Leads)
}

func leadValidationArchive(lead *ResearchLead) *GLXFile {
	return &GLXFile{
		Persons:   map[string]*Person{"person-1": {}},
		Sources:   map[string]*Source{"source-1": {Title: "S"}},
		Citations: map[string]*Citation{"citation-1": {SourceID: "source-1"}},
		Assertions: map[string]*Assertion{"assertion-1": {
			Subject: EntityRef{Person: "person-1"}, Property: "name", Value: "x", Sources: []string{"source-1"},
		}},
		LeadStatuses: map[string]*VocabularyEntry{
			LeadStatusActive:     {Label: "Active"},
			LeadStatusEliminated: {Label: "Eliminated"},
		},
		ConfidenceLevels: map[string]*VocabularyEntry{"medium": {Label: "Medium"}},
		ResearchLogs: map[string]*ResearchLog{
			"log-1": {Leads: []ResearchLead{*lead}},
		},
	}
}

func TestResearchLeadValidation(t *testing.T) {
	t.Run("valid lead has no errors", func(t *testing.T) {
		result := leadValidationArchive(&ResearchLead{
			Persons:    []string{"person-1"},
			Status:     LeadStatusActive,
			Confidence: "medium",
			Citations:  []string{"citation-1"},
			Assertions: []string{"assertion-1"},
		}).Validate()
		assert.Empty(t, result.Errors)
	})

	cases := []struct {
		name    string
		lead    ResearchLead
		missing string
	}{
		{"unknown candidate person", ResearchLead{Persons: []string{"person-missing"}}, "person-missing"},
		{"status not in vocabulary", ResearchLead{Status: "maybe"}, "maybe"},
		{"confidence not in vocabulary", ResearchLead{Confidence: "certain"}, "certain"},
		{"unknown citation", ResearchLead{Citations: []string{"citation-missing"}}, "citation-missing"},
		{"unknown assertion", ResearchLead{Assertions: []string{"assertion-missing"}}, "assertion-missing"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := leadValidationArchive(&tc.lead).Validate()
			require.Len(t, result.Errors, 1, "%v", result.Errors)
			assert.Contains(t, result.Errors[0].Message, tc.missing)
		})
	}
}

func TestLeadStatusesStandardVocabularyLoaded(t *testing.T) {
	glx := &GLXFile{}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(glx))

	assert.Len(t, glx.LeadStatuses, 3)
	assert.Contains(t, glx.LeadStatuses, LeadStatusActive)
	assert.Contains(t, glx.LeadStatuses, LeadStatusEliminated)
	assert.Contains(t, glx.LeadStatuses, LeadStatusConfirmed)
	assert.Contains(t, StandardVocabularies(), "lead-statuses.glx")
}

func TestRenameEntity_ResearchLeadRefs(t *testing.T) {
	glx := &GLXFile{
		Persons: map[string]*Person{"person-old": {}},
		ResearchLogs: map[string]*ResearchLog{
			"rl-1": {Leads: []ResearchLead{
				{Persons: []string{"person-other", "person-old"}},
				{Persons: []string{"person-old"}},
			}},
		},
	}

	_, err := RenameEntity(glx, "person-old", "person-new")
	require.NoError(t, err)
	assert.Equal(t, []string{"person-other", "person-new"}, glx.ResearchLogs["rl-1"].Leads[0].Persons)
	assert.Equal(t, []string{"person-new"}, glx.ResearchLogs["rl-1"].Leads[1].Persons)
}

func TestThreeWayMerge_ResearchLog_Leads(t *testing.T) {
	base := &GLXFile{ResearchLogs: map[string]*ResearchLog{
		"rl1": {Leads: []ResearchLead{{Description: "A", Status: LeadStatusActive}}},
	}}

	t.Run("one-sided change wins", func(t *testing.T) {
		ours := &GLXFile{ResearchLogs: map[string]*ResearchLog{
			"rl1": {Leads: []ResearchLead{{Description: "A", Status: LeadStatusEliminated}}},
		}}
		merged, conflicts := ThreeWayMerge(base, ours, base)
		assert.False(t, HasUnresolvedConflict(conflicts), "%v", conflicts)
		assert.Equal(t, LeadStatusEliminated, merged.ResearchLogs["rl1"].Leads[0].Status)
	})

	t.Run("diverging changes conflict", func(t *testing.T) {
		ours := &GLXFile{ResearchLogs: map[string]*ResearchLog{
			"rl1": {Leads: []ResearchLead{{Description: "A", Status: LeadStatusEliminated}}},
		}}
		theirs := &GLXFile{ResearchLogs: map[string]*ResearchLog{
			"rl1": {Leads: []ResearchLead{{Description: "A", Status: LeadStatusConfirmed}}},
		}}
		_, conflicts := ThreeWayMerge(base, ours, theirs)
		assert.True(t, HasUnresolvedConflict(conflicts))
	})

	t.Run("one-sided lead_statuses vocabulary add survives", func(t *testing.T) {
		ours := &GLXFile{LeadStatuses: map[string]*VocabularyEntry{"parked": {Label: "Parked"}}}
		merged, conflicts := ThreeWayMerge(&GLXFile{}, ours, &GLXFile{})
		assert.Empty(t, conflicts)
		assert.Contains(t, merged.LeadStatuses, "parked")
	})
}

func TestResearchLogHelpers(t *testing.T) {
	log := &ResearchLog{
		Subject: &EntityRef{Person: "person-mary"},
		Searches: []Search{
			{Result: SearchResultNotSearched},
			{Result: SearchResultFound},
			{Result: SearchResultNotSearched},
		},
		Leads: []ResearchLead{
			{Persons: []string{"person-john", "person-mary"}, Status: LeadStatusActive},
			{Persons: []string{"person-luther"}, Status: LeadStatusEliminated},
			{Status: LeadStatusConfirmed},
			{},
		},
	}

	assert.Equal(t, []string{"person-mary", "person-john", "person-luther"}, log.PersonIDs())
	assert.True(t, log.ReferencesEntity("person-mary"))
	assert.True(t, log.ReferencesEntity("PERSON-LUTHER"))
	assert.False(t, log.ReferencesEntity("person-nobody"))
	assert.False(t, log.ReferencesEntity(""))
	assert.Equal(t, 2, log.CountSearchResults(SearchResultNotSearched))

	assert.True(t, log.Leads[0].IsOpen())
	assert.False(t, log.Leads[1].IsOpen())
	assert.False(t, log.Leads[2].IsOpen())
	assert.True(t, log.Leads[3].IsOpen(), "a lead with no status is still open")

	var nilLog *ResearchLog
	assert.Nil(t, nilLog.PersonIDs())
	assert.False(t, nilLog.ReferencesEntity("x"))
	assert.Zero(t, nilLog.CountSearchResults(SearchResultFound))

	placeLog := &ResearchLog{Subject: &EntityRef{Place: "place-leeds"}}
	assert.Empty(t, placeLog.PersonIDs())
	assert.True(t, placeLog.ReferencesEntity("place-leeds"))
}
