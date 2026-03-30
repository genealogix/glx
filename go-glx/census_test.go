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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func intPtr(v int) *int { return &v }

func TestBuildCensusEntities_Minimal(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", Age: intPtr(30), Sex: "male"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	// Should create: 1 place, 1 source, 1 citation, 2 events (census + birth), 1 person
	assert.Len(t, result.Place, 1)
	assert.Len(t, result.Source, 1)
	assert.Len(t, result.Citation, 1)
	assert.Len(t, result.Event, 2)
	assert.Len(t, result.Persons, 1)
	assert.Len(t, result.NewPersonIDs, 1)
	assert.Empty(t, result.MatchedIDs)

	// Check person
	person := result.Persons[result.NewPersonIDs[0]]
	require.NotNil(t, person)
	assert.Equal(t, "Daniel Lane", person.Properties[PersonPropertyName])
	assert.Equal(t, "male", person.Properties[PersonPropertyGender])

	// Check event
	event := result.Event[result.EventID]
	require.NotNil(t, event)
	assert.Equal(t, EventTypeCensus, event.Type)
	assert.Equal(t, "1860 Census — Lane Household", event.Title)
	assert.Equal(t, DateString("1860"), event.Date)
	assert.Len(t, event.Participants, 1)
	assert.Equal(t, "subject", event.Participants[0].Role)

	// Check assertions exist
	assert.NotEmpty(t, result.Assertions)
}

func TestBuildCensusEntities_MultipleMembers(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1870,
			Date:     "1870-06-15",
			Location: CensusLocation{Place: "Sumter County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "William Clark", Age: intPtr(45), Sex: "male", Occupation: "Farmer", Birthplace: "Virginia"},
					{Name: "Mary Clark", Age: intPtr(38), Sex: "female", Birthplace: "Virginia"},
					{Name: "John Clark", Age: intPtr(12), Sex: "male", Birthplace: "Florida"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	assert.Len(t, result.Persons, 3)
	assert.Len(t, result.NewPersonIDs, 3)
	assert.Len(t, result.Event[result.EventID].Participants, 3)

	// Check date uses the specific date
	event := result.Event[result.EventID]
	assert.Equal(t, DateString("1870-06-15"), event.Date)
}

func TestBuildCensusEntities_ExistingPerson(t *testing.T) {
	existing := &GLXFile{
		Persons: map[string]*Person{
			"person-daniel-lane": {
				Properties: map[string]any{
					PersonPropertyName: "Daniel Lane",
				},
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", PersonID: "person-daniel-lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Empty(t, result.Persons, "should not create new person for existing ID")
	assert.Empty(t, result.NewPersonIDs)
	assert.Equal(t, []string{"person-daniel-lane"}, result.MatchedIDs)
}

func TestBuildCensusEntities_ExistingPlace(t *testing.T) {
	existing := &GLXFile{
		Places: map[string]*Place{
			"place-marion-county-florida": {
				Name: "Marion County, Florida",
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Empty(t, result.Place, "should not create new place if existing matches")
	assert.Equal(t, "place-marion-county-florida", result.PlaceID)
}

func TestBuildCensusEntities_ExistingSource(t *testing.T) {
	existing := &GLXFile{
		Sources: map[string]*Source{
			"source-1860-census": {
				Title: "1860 U.S. Federal Census — Marion County, Florida",
				Type:  SourceTypeCensus,
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Empty(t, result.Source, "should not create new source if existing matches")
	assert.Equal(t, "source-1860-census", result.SourceID)
}

func TestBuildCensusEntities_ExplicitSourceID(t *testing.T) {
	existing := &GLXFile{
		Sources: map[string]*Source{
			"source-my-census": {
				Title: "My Census Source",
				Type:  SourceTypeCensus,
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Source:   CensusSourceRef{SourceID: "source-my-census"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Equal(t, "source-my-census", result.SourceID)
	assert.Empty(t, result.Source)
}

func TestBuildCensusEntities_ExplicitSourceID_NotFound(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Source:   CensusSourceRef{SourceID: "source-does-not-exist"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBuildCensusEntities_ExplicitPlaceID(t *testing.T) {
	existing := &GLXFile{
		Places: map[string]*Place{
			"place-marion-county": {Name: "Marion County"},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{PlaceID: "place-marion-county"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Equal(t, "place-marion-county", result.PlaceID)
	assert.Empty(t, result.Place, "should not create place when explicit ID given")
}

func TestBuildCensusEntities_ExplicitPlaceID_NotFound(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{PlaceID: "place-does-not-exist"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestBuildCensusEntities_Citation(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Citation: CensusCitationData{
				Locator:        "Page 42, Line 12",
				TextFromSource: "Daniel Lane, age 30, farmer",
				URL:            "https://example.com/census",
				Accessed:       "2025-01-15",
			},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	cit := result.Citation[result.CitationID]
	require.NotNil(t, cit)
	assert.Equal(t, result.SourceID, cit.SourceID)
	assert.Equal(t, "Page 42, Line 12", cit.Properties["locator"])
	assert.Equal(t, "Daniel Lane, age 30, farmer", cit.Properties["text_from_source"])
	assert.Equal(t, "https://example.com/census", cit.Properties["url"])
	assert.Equal(t, "2025-01-15", cit.Properties["accessed"])
}

func TestBuildCensusEntities_FANNotes(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
			FAN: &CensusFAN{Notes: "Previous page: John Smith. Next page: Robert Brown."},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	event := result.Event[result.EventID]
	assert.Contains(t, event.Notes, "FAN — Previous page: John Smith. Next page: Robert Brown.")
}

func TestBuildCensusEntities_Assertions(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{
						Name:       "Daniel Lane",
						Age:        intPtr(30),
						Sex:        "male",
						Birthplace: "Virginia",
						Occupation: "Farmer",
						Properties: map[string]any{"race": "white"},
					},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	// Should have assertions for: birth year, birthplace, gender, occupation, residence, race
	assert.GreaterOrEqual(t, len(result.Assertions), 6)

	// Check birth year assertion
	birthAssertion := result.Assertions["assertion-person-daniel-lane-birth-year-1860"]
	require.NotNil(t, birthAssertion, "should have birth year assertion")
	assert.Equal(t, "date", birthAssertion.Property)
	assert.Equal(t, "ABT 1830", birthAssertion.Value)
	assert.Equal(t, ConfidenceLevelLow, birthAssertion.Confidence)
	assert.Contains(t, birthAssertion.Notes, "age 30")

	// Check gender assertion
	genderAssertion := result.Assertions["assertion-person-daniel-lane-gender-1860"]
	require.NotNil(t, genderAssertion, "should have gender assertion")
	assert.Equal(t, PersonPropertyGender, genderAssertion.Property)
	assert.Equal(t, "male", genderAssertion.Value)
	assert.Equal(t, ConfidenceLevelHigh, genderAssertion.Confidence)

	// Check occupation assertion
	occAssertion := result.Assertions["assertion-person-daniel-lane-occupation-1860"]
	require.NotNil(t, occAssertion, "should have occupation assertion")
	assert.Equal(t, "occupation", occAssertion.Property)
	assert.Equal(t, "Farmer", occAssertion.Value)
	assert.Equal(t, ConfidenceLevelHigh, occAssertion.Confidence)
	assert.Equal(t, DateString("1860"), occAssertion.Date)

	// Check residence assertion
	resAssertion := result.Assertions["assertion-person-daniel-lane-residence-1860"]
	require.NotNil(t, resAssertion, "should have residence assertion")
	assert.Equal(t, PersonPropertyResidence, resAssertion.Property)
	assert.Equal(t, ConfidenceLevelHigh, resAssertion.Confidence)

	// Check custom property (race) assertion
	raceAssertion := result.Assertions["assertion-person-daniel-lane-race-1860"]
	require.NotNil(t, raceAssertion, "should have race assertion")
	assert.Equal(t, "race", raceAssertion.Property)
	assert.Equal(t, "white", raceAssertion.Value)
	assert.Equal(t, ConfidenceLevelHigh, raceAssertion.Confidence)
}

func TestBuildCensusEntities_MemberWithExplicitPersonID(t *testing.T) {
	existing := &GLXFile{
		Persons: map[string]*Person{
			"person-d-lane": {
				Properties: map[string]any{
					PersonPropertyName: "Daniel Lane",
				},
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", PersonID: "person-d-lane", Age: intPtr(30)},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	// Birth assertion should target a birth event, not a person property.
	// Assertion ID uses the resolved personID slug.
	birthAssertion := result.Assertions["assertion-person-d-lane-birth-year-1860"]
	require.NotNil(t, birthAssertion)
	assert.NotEmpty(t, birthAssertion.Subject.Event,
		"birth assertion should target a birth event")
	assert.Equal(t, "date", birthAssertion.Property)
}

func TestBuildCensusEntities_PersonIDNotFound(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Unknown", PersonID: "person-does-not-exist"},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBuildCensusEntities_NameMatchExisting(t *testing.T) {
	existing := &GLXFile{
		Persons: map[string]*Person{
			"person-daniel-lane": {
				Properties: map[string]any{
					PersonPropertyName: "Daniel Lane",
				},
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	assert.Empty(t, result.Persons, "should not create new person when name matches")
	assert.Equal(t, []string{"person-daniel-lane"}, result.MatchedIDs)
}

func TestBuildCensusEntities_NameMatchUsesResolvedIDInAssertions(t *testing.T) {
	// The existing person ID ("person-abc123") differs from what Slugify would
	// produce ("person-daniel-lane"). Name matching should find this person,
	// and assertions must use the resolved ID, not the slugified name.
	existing := &GLXFile{
		Persons: map[string]*Person{
			"person-abc123": {
				Properties: map[string]any{
					PersonPropertyName: "Daniel Lane",
				},
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", Age: intPtr(30)},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	// Should match existing person by name, not create a new one
	assert.Empty(t, result.Persons, "should not create new person when name matches")
	assert.Equal(t, []string{"person-abc123"}, result.MatchedIDs)

	// Birth assertion should target a birth event (created by census), not a person property.
	// Assertion IDs use the resolved personID slug for uniqueness.
	birthAssertion := result.Assertions["assertion-person-abc123-birth-year-1860"]
	require.NotNil(t, birthAssertion)
	assert.NotEmpty(t, birthAssertion.Subject.Event,
		"birth assertion should target a birth event")
	assert.Empty(t, birthAssertion.Subject.Person,
		"birth assertion should not target person directly")
	assert.Equal(t, "date", birthAssertion.Property)

	resAssertion := result.Assertions["assertion-person-abc123-residence-1860"]
	require.NotNil(t, resAssertion)
	assert.Equal(t, "person-abc123", resAssertion.Subject.Person)
}

func TestBuildCensusEntities_ParticipantAge(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", Age: intPtr(30)},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	event := result.Event[result.EventID]
	require.Len(t, event.Participants, 1)
	assert.Equal(t, 30, event.Participants[0].Properties["age_at_event"])
}

func TestValidateCensusTemplate_MissingYear(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "year")
}

func TestValidateCensusTemplate_MissingLocation(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year: 1860,
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "location")
}

func TestValidateCensusTemplate_NoMembers(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "members")
}

func TestValidateCensusTemplate_MemberMissingName(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: ""},
				},
			},
		},
	}

	_, err := BuildCensusEntities(tpl, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func Test_slugify(t *testing.T) {
	tests := []struct {
		prefix string
		name   string
		want   string
	}{
		{"person", "Daniel Lane", "person-daniel-lane"},
		{"event", "1860 Census", "event-1860-census"},
		{"", "Daniel Lane", "daniel-lane"},
		{"person", "Mary O'Brien", "person-mary-o-brien"},
		{"person", "  spaces  ", "person-spaces"},
		{"person", "", "person-unknown"},
	}

	for _, tt := range tests {
		got := slugify(tt.prefix, tt.name)
		assert.Equal(t, tt.want, got, "slugify(%q, %q)", tt.prefix, tt.name)
	}
}

func TestBuildCensusEntities_CustomEventTitle(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Title: "1860 Census — Lane Household (Ocala)",
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	event := result.Event[result.EventID]
	assert.Equal(t, "1860 Census — Lane Household (Ocala)", event.Title)
}

func TestBuildCensusEntities_SourceWithRepository(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Source: CensusSourceRef{
				Title:        "1860 Census, Marston",
				RepositoryID: "repo-nara",
				CallNumber:   "M653, Roll 108",
			},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	require.Len(t, result.Source, 1)
	source := result.Source[result.SourceID]
	assert.Equal(t, "1860 Census, Marston", source.Title)
	assert.Equal(t, "repo-nara", source.RepositoryID)
	assert.Equal(t, "M653, Roll 108", source.Properties["call_number"])
}

func TestBuildCensusEntities_NoAgeSkipsBirthYearAssertion(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane"}, // No age
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	// Should NOT have a birth year assertion
	_, hasBirth := result.Assertions["assertion-person-daniel-lane-birth-year-1860"]
	assert.False(t, hasBirth, "should not have birth year assertion without age")

	// Should still have residence assertion
	_, hasResidence := result.Assertions["assertion-person-daniel-lane-residence-1860"]
	assert.True(t, hasResidence, "should have residence assertion regardless")
}

func TestBuildCensusEntities_NilTemplate(t *testing.T) {
	_, err := BuildCensusEntities(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "census template is required")
}

func TestBuildCensusEntities_DuplicateNamesInBatch(t *testing.T) {
	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1850,
			Location: CensusLocation{Place: "Ironton, Ohio"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "John Smith", Age: intPtr(40), Sex: "male"},
					{Name: "John Smith", Age: intPtr(12), Sex: "male"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, nil)
	require.NoError(t, err)

	// Should create 2 distinct persons, not merge them
	assert.Len(t, result.Persons, 2)
	assert.Len(t, result.NewPersonIDs, 2)
	assert.NotEqual(t, result.NewPersonIDs[0], result.NewPersonIDs[1],
		"duplicate names should get distinct person IDs")
}

func TestBuildCensusEntities_PersonIDCollisionWithExisting(t *testing.T) {
	// Archive has person-john-smith but it's a DIFFERENT John Smith
	// (name search won't match because it's "Jonathan Smith")
	existing := &GLXFile{
		Persons: map[string]*Person{
			"person-john-smith": {
				Properties: map[string]any{
					PersonPropertyName: "Jonathan Smith",
				},
			},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1850,
			Location: CensusLocation{Place: "Ironton, Ohio"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "John Smith"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	// Should create a new person with a disambiguated ID
	assert.Len(t, result.Persons, 1)
	assert.Len(t, result.NewPersonIDs, 1)
	assert.NotEqual(t, "person-john-smith", result.NewPersonIDs[0],
		"should not reuse existing person ID for a different individual")
}

func TestBuildCensusEntities_ResolveBirthplaceFromExisting(t *testing.T) {
	existing := &GLXFile{
		Places: map[string]*Place{
			"place-virginia": {Name: "Virginia"},
		},
	}

	tpl := &CensusTemplate{
		Census: CensusData{
			Year:     1860,
			Location: CensusLocation{Place: "Marion County, Florida"},
			Household: CensusHousehold{
				Members: []CensusHouseholdMember{
					{Name: "Daniel Lane", Birthplace: "Virginia"},
				},
			},
		},
	}

	result, err := BuildCensusEntities(tpl, existing)
	require.NoError(t, err)

	// Birthplace assertion should reference the existing place ID
	bpAssertion := result.Assertions["assertion-person-daniel-lane-birthplace-1860"]
	require.NotNil(t, bpAssertion)
	assert.Equal(t, "place-virginia", bpAssertion.Value,
		"should resolve birthplace against existing archive places")
}

func TestTruncateID(t *testing.T) {
	assert.Equal(t, "short-id", truncateID("short-id"))
	long := "prefix-1860-census-this-is-a-very-long-place-name-that-exceeds-the-sixty-four-character-limit"
	result := truncateID(long)
	assert.LessOrEqual(t, len(result), 64)
	assert.False(t, result[len(result)-1] == '-', "truncated ID should not end with hyphen")
}

func TestUniquePersonID_LongBaseID(t *testing.T) {
	// When baseID > 64 chars and truncated candidate collides, the suffix must
	// be preserved to avoid an infinite loop.
	longBase := "person-" + strings.Repeat("a", 60) // 67 chars total
	existing := &GLXFile{
		Persons: map[string]*Person{
			truncateID(longBase): {Properties: map[string]any{PersonPropertyName: "Collider"}},
		},
	}
	result := &CensusResult{
		Persons: make(map[string]*Person),
	}

	id := uniquePersonID(longBase, existing, result)
	assert.NotEqual(t, truncateID(longBase), id, "should disambiguate from colliding ID")
	assert.LessOrEqual(t, len(id), 64, "result must be within 64-char limit")
	assert.Contains(t, id, "-2", "should have suffix")
}

func TestCensusSlugIDWithHousehold(t *testing.T) {
	loc := CensusLocation{Place: "Marion County, Florida"}
	id := censusSlugIDWithHousehold("event", 1860, loc, "Lane")
	assert.Contains(t, id, "lane")
	assert.Contains(t, id, "1860")
	assert.LessOrEqual(t, len(id), 64)
}
