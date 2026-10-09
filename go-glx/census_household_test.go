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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headOnlyCensusArchive is a valid archive with an 1820 head-only census:
// James named, his wife and daughter counted as tick marks, a tally, and two
// page neighbors (one linked to a person).
func headOnlyCensusArchive(t *testing.T) *GLXFile {
	t.Helper()
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-james":   {Properties: map[string]any{PersonPropertyName: "James Little"}},
			"person-eliz":    {Properties: map[string]any{PersonPropertyName: "Elizabeth Starr"}},
			"person-mary":    {Properties: map[string]any{PersonPropertyName: "Mary Little"}},
			"person-jeffrey": {Properties: map[string]any{PersonPropertyName: "Henry Jeffries"}},
		},
		Places: map[string]*Place{"place-wythe": {Name: "Wythe County"}},
		Events: map[string]*Event{
			"event-census-1820": {
				Type:    EventTypeCensus,
				Date:    "1820",
				PlaceID: "place-wythe",
				Participants: []Participant{
					{Person: "person-james", Role: ParticipantRolePrincipal},
					{Person: "person-eliz", Role: ParticipantRoleHouseholdMember, Properties: map[string]any{ParticipantPropertyNamed: false}},
					{Person: "person-mary", Role: ParticipantRoleHouseholdMember, Properties: map[string]any{ParticipantPropertyNamed: false}},
				},
				Household: &Household{Tally: []HouseholdTallyRow{
					{Sex: "male", AgeFrom: new(45), Count: 1, Status: "free white"},
					{Sex: "female", AgeFrom: new(45), Count: 1, Status: "free white"},
					{Sex: "female", AgeFrom: new(10), AgeTo: new(15), Count: 2, Status: "free white"},
					{Count: 1, Status: "engaged in agriculture"},
				}},
				Neighbors: []EventNeighbor{
					{Name: "Henry Jeffries", Person: "person-jeffrey", Position: "previous_household", Page: "12", Line: "3"},
					{Name: "James M Clark", Position: "next_household"},
				},
			},
		},
	}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))

	return archive
}

func validationMessages(result *ValidationResult) (errs, warns []string) {
	for _, e := range result.Errors {
		errs = append(errs, e.Message)
	}
	for _, w := range result.Warnings {
		warns = append(warns, w.Message)
	}

	return errs, warns
}

func TestValidateCensusHousehold_Valid(t *testing.T) {
	archive := headOnlyCensusArchive(t)
	errs, warns := validationMessages(archive.Validate())
	assert.Empty(t, errs)
	assert.Empty(t, warns)
}

func TestValidateCensusHousehold_StructuralErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(e *Event)
		wantErr string
	}{
		{"zero count", func(e *Event) { e.Household.Tally[0].Count = 0 }, "household.tally[0]: count must be at least 1"},
		{"inverted bracket", func(e *Event) { e.Household.Tally[2].AgeFrom = new(20) }, "age_from (20) is greater than age_to (15)"},
		{"negative age", func(e *Event) { e.Household.Tally[1].AgeFrom = new(-1) }, "age_from must not be negative"},
		{"neighbor without name or person", func(e *Event) { e.Neighbors[1].Name = "" }, "neighbors[1]: a neighbor needs a name or a person reference"},
		{"neighbor dangling person", func(e *Event) { e.Neighbors[0].Person = "person-nobody" }, "references non-existent persons: person-nobody"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := headOnlyCensusArchive(t)
			tt.mutate(archive.Events["event-census-1820"])
			errs, _ := validationMessages(archive.Validate())
			assert.True(t, containsSubstring(errs, tt.wantErr), "want error containing %q, got %v", tt.wantErr, errs)
		})
	}
}

func TestValidateCensusHousehold_Warnings(t *testing.T) {
	t.Run("empty row", func(t *testing.T) {
		archive := headOnlyCensusArchive(t)
		archive.Events["event-census-1820"].Household.Tally[3].Status = ""
		_, warns := validationMessages(archive.Validate())
		assert.True(t, containsSubstring(warns, "does not say who was counted"), "%v", warns)
	})
	t.Run("unknown sex", func(t *testing.T) {
		archive := headOnlyCensusArchive(t)
		archive.Events["event-census-1820"].Household.Tally[0].Sex = "m"
		errs, warns := validationMessages(archive.Validate())
		assert.Empty(t, errs)
		assert.True(t, containsSubstring(warns, "household.tally[0].sex: value 'm' not found in sex_types vocabulary"), "%v", warns)
	})
	t.Run("non-census event", func(t *testing.T) {
		archive := headOnlyCensusArchive(t)
		archive.Events["event-census-1820"].Type = EventTypeResidence
		_, warns := validationMessages(archive.Validate())
		assert.True(t, containsSubstring(warns, "defined for census events"), "%v", warns)
	})
	t.Run("more unnamed participants than tally", func(t *testing.T) {
		archive := headOnlyCensusArchive(t)
		archive.Events["event-census-1820"].Household.Tally = []HouseholdTallyRow{{Sex: "male", Count: 1}}
		_, warns := validationMessages(archive.Validate())
		assert.True(t, containsSubstring(warns, "2 participants are marked named: false but the household tally counts only 1"), "%v", warns)
	})
}

func containsSubstring(list []string, sub string) bool {
	for _, s := range list {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}

func TestCensusHousehold_SerializerRoundTrip(t *testing.T) {
	archive := headOnlyCensusArchive(t)
	s := NewSerializer(nil)

	data, err := s.SerializeSingleFileBytes(archive)
	require.NoError(t, err)
	assert.Contains(t, string(data), "household:")
	assert.Contains(t, string(data), "tally:")
	assert.Contains(t, string(data), "neighbors:")

	loaded, err := s.DeserializeSingleFileBytes(data)
	require.NoError(t, err)
	want := archive.Events["event-census-1820"]
	got := loaded.Events["event-census-1820"]
	require.NotNil(t, got)
	assert.Equal(t, want.Household, got.Household)
	assert.Equal(t, want.Neighbors, got.Neighbors)
	assert.True(t, IsUnnamedParticipant(got.Participants[1]))
	assert.False(t, IsUnnamedParticipant(got.Participants[0]))

	files, err := s.SerializeMultiFileToMap(archive)
	require.NoError(t, err)
	multi, _, err := s.DeserializeMultiFileFromMap(files)
	require.NoError(t, err)
	assert.Equal(t, want.Household, multi.Events["event-census-1820"].Household)
	assert.Equal(t, want.Neighbors, multi.Events["event-census-1820"].Neighbors)
}

func TestIsUnnamedParticipant(t *testing.T) {
	assert.False(t, IsUnnamedParticipant(Participant{}))
	assert.False(t, IsUnnamedParticipant(Participant{Properties: map[string]any{ParticipantPropertyNamed: true}}))
	assert.False(t, IsUnnamedParticipant(Participant{Properties: map[string]any{ParticipantPropertyNamed: "false"}}))
	assert.True(t, IsUnnamedParticipant(Participant{Properties: map[string]any{ParticipantPropertyNamed: false}}))
}

func TestCensusHousehold_RenameUpdatesNeighborPerson(t *testing.T) {
	archive := headOnlyCensusArchive(t)
	_, err := RenameEntity(archive, "person-jeffrey", "person-henry-jeffries")
	require.NoError(t, err)
	assert.Equal(t, "person-henry-jeffries", archive.Events["event-census-1820"].Neighbors[0].Person)
}

func TestCensusHousehold_ThreeWayMerge(t *testing.T) {
	t.Run("one side changes the tally", func(t *testing.T) {
		base := headOnlyCensusArchive(t)
		ours := headOnlyCensusArchive(t)
		theirs := headOnlyCensusArchive(t)
		theirs.Events["event-census-1820"].Household.Tally[0].Count = 2
		theirs.Events["event-census-1820"].Neighbors = append(theirs.Events["event-census-1820"].Neighbors, EventNeighbor{Name: "Abram Baker"})

		merged, conflicts := ThreeWayMerge(base, ours, theirs)
		assert.False(t, HasUnresolvedConflict(conflicts), "%v", conflicts)
		ev := merged.Events["event-census-1820"]
		assert.Equal(t, 2, ev.Household.Tally[0].Count)
		assert.Len(t, ev.Neighbors, 3)
	})
	t.Run("both sides change the tally differently", func(t *testing.T) {
		base := headOnlyCensusArchive(t)
		ours := headOnlyCensusArchive(t)
		theirs := headOnlyCensusArchive(t)
		ours.Events["event-census-1820"].Household.Tally[0].Count = 2
		theirs.Events["event-census-1820"].Household.Tally[0].Count = 3

		_, conflicts := ThreeWayMerge(base, ours, theirs)
		require.True(t, HasUnresolvedConflict(conflicts))
		paths := make([]string, 0, len(conflicts))
		for _, c := range conflicts {
			paths = append(paths, c.Path)
		}
		assert.True(t, containsSubstring(paths, ".household"), "%v", paths)
	})
}

func TestCensusHousehold_PrivatizeLivingNeighbor(t *testing.T) {
	archive := headOnlyCensusArchive(t)
	archive.Persons["person-jeffrey"].Properties["living"] = true

	PrivatizeLiving(archive, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 100)

	n := archive.Events["event-census-1820"].Neighbors[0]
	assert.Equal(t, "Living", n.Name)
	assert.Equal(t, "person-jeffrey", n.Person)
	assert.Empty(t, n.Page)
	assert.Equal(t, "James M Clark", archive.Events["event-census-1820"].Neighbors[1].Name)
}

func TestBuildCensusEntities_HeadOnlySchedule(t *testing.T) {
	existing := headOnlyCensusArchive(t)
	tpl := &CensusTemplate{Census: CensusData{
		Year:     1800,
		Location: CensusLocation{PlaceID: "place-wythe"},
		Household: CensusHousehold{
			Members: []CensusHouseholdMember{
				{Name: "James Little", PersonID: "person-james", Role: ParticipantRolePrincipal},
				{Name: "Elizabeth Starr", PersonID: "person-eliz", Named: new(false)},
				{Name: "Sarah Little", Sex: "female", Named: new(false)},
			},
			Tally: []HouseholdTallyRow{
				{Sex: "male", AgeFrom: new(26), AgeTo: new(44), Count: 1},
				{Sex: "female", AgeTo: new(9), Count: 1},
			},
			Neighbors: []EventNeighbor{{Name: "Henry Jeffries", Person: "person-jeffrey", Position: "previous_household"}},
		},
	}}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	event := result.Event[result.EventID]
	require.NotNil(t, event)
	require.NotNil(t, event.Household)
	assert.Equal(t, tpl.Census.Household.Tally, event.Household.Tally)
	assert.Equal(t, tpl.Census.Household.Neighbors, event.Neighbors)

	require.Len(t, event.Participants, 3)
	assert.Equal(t, ParticipantRolePrincipal, event.Participants[0].Role)
	for _, p := range event.Participants[1:] {
		assert.Equal(t, ParticipantRoleHouseholdMember, p.Role)
		assert.True(t, IsUnnamedParticipant(p))
	}

	// Unnamed members get only a low-confidence residence assertion: no sex
	// assertion, though Sarah's sex seeds her new person record.
	sarahID := result.NewPersonIDs[0]
	assert.Equal(t, "female", result.Persons[sarahID].Properties[PersonPropertySex])
	for id, a := range result.Assertions {
		if a.Subject.Person == sarahID || a.Subject.Person == "person-eliz" {
			assert.Equal(t, PersonPropertyResidence, a.Property, id)
			assert.Equal(t, ConfidenceLevelLow, a.Confidence, id)
		}
	}
}

func TestBuildCensusEntities_HeadOnlyTemplateErrors(t *testing.T) {
	base := func() *CensusTemplate {
		return &CensusTemplate{Census: CensusData{
			Year:     1800,
			Location: CensusLocation{Place: "Wythe County"},
			Household: CensusHousehold{Members: []CensusHouseholdMember{
				{Name: "James Little"},
				{Name: "Sarah Little", Named: new(false)},
			}},
		}}
	}
	tests := []struct {
		name    string
		mutate  func(tpl *CensusTemplate)
		wantErr string
	}{
		{"unnamed head", func(tpl *CensusTemplate) { tpl.Census.Household.Members[0].Named = new(false) }, "must be named"},
		{"unnamed member with age", func(tpl *CensusTemplate) { tpl.Census.Household.Members[1].Age = new(5) }, "sets age"},
		{"bad tally row", func(tpl *CensusTemplate) {
			tpl.Census.Household.Tally = []HouseholdTallyRow{{Sex: "male", Count: 0}}
		}, "census.household.tally[0]: count must be at least 1"},
		{"neighbor without name", func(tpl *CensusTemplate) {
			tpl.Census.Household.Neighbors = []EventNeighbor{{Position: "next_household"}}
		}, "needs a name or person: census.household.neighbors[0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl := base()
			tt.mutate(tpl)
			_, err := BuildCensusEntities(tpl, nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
