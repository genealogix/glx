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
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func explorationTestArchive() *glxlib.GLXFile {
	archive := threeGenArchive()
	archive.Sources = map[string]*glxlib.Source{"source-census": {Title: "Census", Type: "census"}}
	archive.Citations = map[string]*glxlib.Citation{"citation-one": {SourceID: "source-census"}}
	archive.Repositories = map[string]*glxlib.Repository{"repo-one": {Name: "County archive"}}
	archive.Media = map[string]*glxlib.Media{"media-one": {Title: "Image"}}
	archive.Studies = map[string]*glxlib.Study{"study-one": {Title: "Origins"}}
	archive.Places = map[string]*glxlib.Place{"place-ohio": {Name: "Ohio"}}
	archive.Events["event-birth-child"].PlaceID = "place-ohio"
	archive.Assertions = map[string]*glxlib.Assertion{
		"assertion-one": {Subject: glxlib.EntityRef{Person: "person-child"}, Property: "name", Value: "Child Smith", Confidence: "low", Status: "proven", Citations: []string{"citation-one"}},
	}
	archive.ResearchLogs = map[string]*glxlib.ResearchLog{"research-one": {Subject: &glxlib.EntityRef{Person: "person-child"}, Status: "in_progress", Objective: "Identify parents"}}

	return archive
}

func TestExplorationJSONQueryTypesAndFilters(t *testing.T) {
	archive := explorationTestArchive()
	for _, entityType := range queryEntityTypes {
		t.Run(string(entityType), func(t *testing.T) {
			result, err := buildQueryJSON(archive, entityType, &queryOpts{})
			require.NoError(t, err)
			require.Positive(t, result.Count)
			require.Len(t, result.Entities, result.Count)
			for i := 1; i < len(result.Entities); i++ {
				assert.Less(t, result.Entities[i-1].ID, result.Entities[i].ID)
			}
		})
	}
	cases := []struct {
		entityType glxlib.EntityType
		opts       queryOpts
		want       []string
	}{
		{"persons", queryOpts{Name: "CHILD", BornBefore: 1881, BornAfter: 1879, Birthplace: "ohio"}, []string{"person-child"}},
		{"persons", queryOpts{Name: "Smyth", Phonetic: true, BornAfter: 1879, BornBefore: 1881}, []string{"person-child"}},
		{"persons", queryOpts{Status: "hypothetical"}, []string{}},
		{"events", queryOpts{Type: "BIRTH", Before: 1881, After: 1879}, []string{"event-birth-child"}},
		{"assertions", queryOpts{Confidence: "LOW", Status: "PROVEN", Source: "source-census", Citation: "citation-one", Subject: "Child Smith"}, []string{"assertion-one"}},
		{"sources", queryOpts{Name: "CENS", Type: "CENSUS"}, []string{"source-census"}},
		{"relationships", queryOpts{Type: "adoptive_parent_child"}, []string{"rel-father-adopted"}},
		{"places", queryOpts{Name: "OHIO"}, []string{"place-ohio"}},
		{"repositories", queryOpts{Name: "COUNTY"}, []string{"repo-one"}},
		{"research_logs", queryOpts{Status: "IN_PROGRESS", Subject: "person-child"}, []string{"research-one"}},
	}
	for _, tt := range cases {
		result, err := buildQueryJSON(archive, tt.entityType, &tt.opts)
		require.NoError(t, err)
		ids := []string{}
		for _, entity := range result.Entities {
			ids = append(ids, entity.ID)
		}
		assert.Equal(t, tt.want, ids, "%s: %+v", tt.entityType, &tt.opts)
	}
	_, err := buildQueryJSON(archive, "persons", &queryOpts{Phonetic: true})
	require.ErrorIs(t, err, errPhoneticRequiresName)
	_, err = buildQueryJSON(archive, "media", &queryOpts{Name: "unsupported"})
	require.ErrorContains(t, err, "not supported")
	_, err = buildQueryJSON(archive, "unknown", &queryOpts{})
	require.Error(t, err)
	_, err = buildQueryJSON(nil, "persons", &queryOpts{})
	require.Error(t, err)
}

func TestExplorationJSONProfileAndEvidence(t *testing.T) {
	archive := explorationTestArchive()
	before, err := json.Marshal(archive)
	require.NoError(t, err)
	summary, err := buildSummaryJSON(archive, "person-child")
	require.NoError(t, err)
	assert.Contains(t, summary.Narrative, "1880")
	require.Len(t, summary.Assertions, 1)
	require.Len(t, summary.ResearchLogs, 1)
	assert.Equal(t, "assertion-one", summary.Assertions[0].ID)
	assert.Equal(t, []familyJSONEdge{{PersonID: "person-father"}, {PersonID: "person-mother"}}, summary.Parents)
	vitals, err := buildVitalsJSON(archive, "Child Smith")
	require.NoError(t, err)
	assert.Equal(t, "person-child", vitals.PersonID)
	require.Len(t, vitals.Events, 1)
	assert.Equal(t, "event-birth-child", vitals.Events[0].ID)
	direct, err := buildTimelineJSON(archive, "person-child", false)
	require.NoError(t, err)
	family, err := buildTimelineJSON(archive, "person-child", true)
	require.NoError(t, err)
	assert.Greater(t, len(family.Entries), len(direct.Entries))
	for _, entry := range direct.Entries {
		assert.False(t, entry.IsFamily)
	}
	after, err := json.Marshal(archive)
	require.NoError(t, err)
	assert.JSONEq(t, string(before), string(after), "read-only builders must not alter the archive")
}

func TestExplorationJSONResolutionAndTrees(t *testing.T) {
	archive := explorationTestArchive()
	for _, query := range []string{"", "missing", "Smith"} {
		_, err := buildVitalsJSON(archive, query)
		require.Error(t, err)
		_, err = buildTimelineJSON(archive, query, true)
		require.Error(t, err)
		_, err = buildSummaryJSON(archive, query)
		require.Error(t, err)
		_, err = buildTreeJSON(archive, query, 1, true)
		require.Error(t, err)
	}
	_, err := buildTreeJSON(archive, "person-child", -1, true)
	require.Error(t, err)
	tree, err := buildTreeJSON(archive, "person-child", 1, true)
	require.NoError(t, err)
	assert.Equal(t, "ancestors", tree.Direction)
	require.Len(t, tree.Root.Children, 2)
	for _, child := range tree.Root.Children {
		assert.Empty(t, child.Children)
	}
	descendants, err := buildTreeJSON(archive, "person-father", 1, false)
	require.NoError(t, err)
	assert.Equal(t, "descendants", descendants.Direction)
	require.Len(t, descendants.Root.Children, 2)
}

func TestExplorationJSONCitationAndEmptyResults(t *testing.T) {
	archive := explorationTestArchive()
	result, err := buildCitationJSON(archive, "citation-one", false)
	require.NoError(t, err)
	require.Len(t, result.Citations, 1)
	assert.Contains(t, result.Citations[0].Text, "Census")
	all, err := buildCitationJSON(archive, "", true)
	require.NoError(t, err)
	assert.Equal(t, result, all)
	_, err = buildCitationJSON(archive, "missing", false)
	require.Error(t, err)
	empty, err := buildQueryJSON(&glxlib.GLXFile{}, "persons", &queryOpts{})
	require.NoError(t, err)
	data, err := json.Marshal(empty)
	require.NoError(t, err)
	assert.JSONEq(t, `{"entity_type":"persons","count":0,"entities":[]}`, string(data))
}

func TestExplorationJSONRunnerOutput(t *testing.T) {
	archivePath := "../docs/examples/complete-family"
	archive, err := loadArchiveForRead(SystemIOStreams(), archivePath)
	require.NoError(t, err)
	personID := sortedKeys(archive.Persons)[0]
	tests := map[string]func() error{
		"query":       func() error { return queryEntities("persons", &queryOpts{Archive: archivePath, JSON: true}) },
		"vitals":      func() error { return showVitals(archivePath, personID, true) },
		"timeline":    func() error { return showTimeline(archivePath, personID, true, true) },
		"summary":     func() error { return showSummary(archivePath, personID, true) },
		"ancestors":   func() error { return showAncestors(archivePath, personID, 1, true) },
		"descendants": func() error { return showDescendants(archivePath, personID, 1, true) },
		"cite":        func() error { return showAllCitations(archivePath, true) },
	}
	for name, run := range tests {
		t.Run(name, func(t *testing.T) {
			output := captureStdout(t, func() { require.NoError(t, run()) })
			require.True(t, json.Valid([]byte(output)), output)
		})
	}
}

func TestExplorationJSONArchiveFieldNames(t *testing.T) {
	archive := explorationTestArchive()
	result, err := buildQueryJSON(archive, glxlib.EntityTypeCitations, &queryOpts{})
	require.NoError(t, err)
	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"source":`)
	assert.NotContains(t, string(data), `"SourceID"`)
	summary, err := buildSummaryJSON(archive, "person-child")
	require.NoError(t, err)
	data, err = json.Marshal(summary)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"properties":`)
	assert.NotContains(t, string(data), `"Properties"`)
	citation, err := buildCitationJSON(archive, "citation-one", false)
	require.NoError(t, err)
	data, err = json.Marshal(citation)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"source":`)
	assert.NotContains(t, string(data), `"SourceID"`)
}

func BenchmarkExplorationJSONPage(b *testing.B) {
	archive := &glxlib.GLXFile{Persons: map[string]*glxlib.Person{}}
	for i := range 24000 {
		id := "person-" + strconv.Itoa(i)
		archive.Persons[id] = &glxlib.Person{Properties: map[string]any{"name": "Example Person"}}
	}
	b.ResetTimer()
	for b.Loop() {
		result, err := buildQueryJSON(archive, glxlib.EntityTypePersons, &queryOpts{})
		if err != nil {
			b.Fatal(err)
		}
		result.Entities = result.Entities[:100]
		if _, err := json.Marshal(result); err != nil {
			b.Fatal(err)
		}
	}
}

func TestExplorationJSONAlternateNameResolution(t *testing.T) {
	archive := explorationTestArchive()
	archive.Persons["person-child"].Properties["name"] = []any{
		map[string]any{"value": "Child Smith"},
		map[string]any{"value": "Historical Alias"},
	}
	archive.Persons["person-null"] = nil
	result, err := buildSummaryJSON(archive, "Historical Alias")
	require.NoError(t, err)
	assert.Equal(t, "person-child", result.PersonID)
}

func TestExplorationJSONFamilyEvidenceStanding(t *testing.T) {
	archive := littleFamilyArchive()
	lewis, err := buildSummaryJSON(archive, "person-lewis")
	require.NoError(t, err)
	require.Equal(t, []familyJSONEdge{
		{PersonID: "person-daniel", Hypothetical: true},
		{PersonID: "person-annie", Hypothetical: true},
	}, lewis.Parents)
	assert.Equal(t, []excludedParentJSONEdge{
		{PersonID: "person-jacob", Reason: glxlib.RelationshipStandingDisproven},
		{PersonID: "person-john", Reason: glxlib.RelationshipStandingDisproven},
	}, lewis.ExcludedParents)
	for _, child := range lewis.Children {
		assert.Equal(t, child.PersonID == "person-peggy" || child.PersonID == "person-rachel", child.Step)
	}
	rachel, err := buildSummaryJSON(archive, "person-rachel")
	require.NoError(t, err)
	assert.Contains(t, rachel.Parents, familyJSONEdge{PersonID: "person-lewis", Step: true})
	assert.Contains(t, rachel.Siblings, siblingJSONEdge{PersonID: "person-nancy", Kind: siblingKindHalf})
	assert.Contains(t, rachel.Siblings, siblingJSONEdge{PersonID: "person-jasper", Kind: siblingKindStep})
	// Removing the disproof restores competing birth-parent hypotheses, not facts.
	archive.Assertions["a-jacob"].Status = "speculative"
	lewis, err = buildSummaryJSON(archive, "person-lewis")
	require.NoError(t, err)
	assert.Contains(t, lewis.Parents, familyJSONEdge{PersonID: "person-daniel", Hypothetical: true, Alternative: true})
	assert.Contains(t, lewis.Parents, familyJSONEdge{PersonID: "person-jacob", Hypothetical: true, Alternative: true})
	data, err := json.Marshal(lewis)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"hypothetical":true`)
	assert.Contains(t, string(data), `"alternative":true`)
	assert.Contains(t, string(data), `"excluded_parents":[{"person_id":"person-john","reason":"disproven"}]`)
}

func TestExplorationJSONVitalsNilEvent(t *testing.T) {
	archive := explorationTestArchive()
	archive.Events["event-null"] = nil
	result, err := buildVitalsJSON(archive, "person-child")
	require.NoError(t, err)
	require.Len(t, result.Events, 1)
	assert.False(t, isPrincipalParticipant("person-child", nil))
}
