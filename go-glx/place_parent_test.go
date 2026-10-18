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
	"gopkg.in/yaml.v3"
)

// wayneCounty is the Indiana example from #225: Wayne County was created in
// Indiana Territory in 1811 and passed to the State of Indiana on 11 Dec 1816.
func wayneCounty() *Place {
	p := &Place{Name: "Wayne County", Type: "county"}
	p.SetParentHistory([]PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "FROM 1811 TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	})

	return p
}

func indianaArchive() *GLXFile {
	return &GLXFile{
		Places: map[string]*Place{
			"place-usa":               {Name: "United States", Type: "country"},
			"place-indiana-territory": {Name: "Indiana Territory", ParentID: "place-usa"},
			"place-indiana":           {Name: "Indiana", Type: "state", ParentID: "place-usa"},
			"place-wayne-in":          wayneCounty(),
			"place-richmond":          {Name: "Richmond", Type: "city", ParentID: "place-wayne-in"},
		},
	}
}

func TestPlaceParent_UnmarshalStringForm(t *testing.T) {
	var p Place
	require.NoError(t, yaml.Unmarshal([]byte("name: Leeds\nparent: place-yorkshire\ntype: city\n"), &p))

	assert.Equal(t, "place-yorkshire", p.ParentID)
	assert.Empty(t, p.ParentHistory)
	assert.False(t, p.HasTemporalParent())
	assert.Equal(t, "city", p.Type)
}

func TestPlaceParent_UnmarshalTemporalForm(t *testing.T) {
	// The flow-style form from the issue comment.
	input := `name: Wayne County
type: county
parent:
  - {value: place-indiana-territory, date: "FROM 1811 TO 1816-12-10"}
  - {value: place-indiana, date: "FROM 1816-12-11"}
notes: Created 1811
`
	var p Place
	require.NoError(t, yaml.Unmarshal([]byte(input), &p))

	assert.True(t, p.HasTemporalParent())
	assert.Equal(t, []PlaceParentPeriod{
		{Value: "place-indiana-territory", Date: "FROM 1811 TO 1816-12-10"},
		{Value: "place-indiana", Date: "FROM 1816-12-11"},
	}, p.ParentHistory)
	assert.Equal(t, "place-indiana", p.ParentID, "default parent is the most recent period")
	assert.Equal(t, "county", p.Type)
	assert.Equal(t, NoteList{"Created 1811"}, p.Notes)
}

func TestPlaceParent_UnmarshalRejectsMalformedList(t *testing.T) {
	var p Place
	err := yaml.Unmarshal([]byte("name: X\nparent:\n  - value: [a, b]\n"), &p)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "place parent")
}

func TestPlaceParent_MarshalRoundTrip(t *testing.T) {
	t.Run("string form is unchanged", func(t *testing.T) {
		data, err := yaml.Marshal(&Place{Name: "Leeds", ParentID: "place-yorkshire", Type: "city"})
		require.NoError(t, err)
		assert.Equal(t, "name: Leeds\nparent: place-yorkshire\ntype: city\n", string(data))
	})

	t.Run("temporal form keeps its position and dates", func(t *testing.T) {
		data, err := yaml.Marshal(wayneCounty())
		require.NoError(t, err)
		want := `name: Wayne County
parent:
    - value: place-indiana-territory
      date: "FROM 1811 TO 1816-12-10"
    - value: place-indiana
      date: "FROM 1816-12-11"
type: county
`
		assert.Equal(t, want, string(data))

		var back Place
		require.NoError(t, yaml.Unmarshal(data, &back))
		assert.Equal(t, *wayneCounty(), back)
	})

	t.Run("inside an archive map", func(t *testing.T) {
		in := map[string]map[string]*Place{"places": {"place-wayne-in": wayneCounty()}}
		data, err := yaml.Marshal(in)
		require.NoError(t, err)

		var out map[string]map[string]*Place
		require.NoError(t, yaml.Unmarshal(data, &out))
		assert.Equal(t, wayneCounty(), out["places"]["place-wayne-in"])
	})
}

func TestDefaultParentOf(t *testing.T) {
	tests := []struct {
		name    string
		periods []PlaceParentPeriod
		want    string
	}{
		{"empty", nil, ""},
		{"undated entry is the default", []PlaceParentPeriod{
			{Value: "a", Date: "FROM 1900"},
			{Value: "b"},
		}, "b"},
		{"open-ended period wins", []PlaceParentPeriod{
			{Value: "new", Date: "FROM 1816-12-11"},
			{Value: "old", Date: "FROM 1811 TO 1816-12-10"},
		}, "new"},
		{"latest end wins among closed periods", []PlaceParentPeriod{
			{Value: "a", Date: "FROM 1800 TO 1850"},
			{Value: "b", Date: "FROM 1850 TO 1900"},
			{Value: "c", Date: "TO 1820"},
		}, "b"},
		{"Pohl-Goens example", []PlaceParentPeriod{
			{Value: "place-amt-huettenberg", Date: "TO 1821"},
			{Value: "place-kreis-friedberg", Date: "FROM 1840 TO 1972"},
			{Value: "place-wetteraukreis", Date: "FROM 1972"},
		}, "place-wetteraukreis"},
		{"later start breaks a tie between open periods", []PlaceParentPeriod{
			{Value: "later", Date: "FROM 1900"},
			{Value: "earlier", Date: "FROM 1800"},
		}, "later"},
		{"unparseable dates rank last", []PlaceParentPeriod{
			{Value: "dated", Date: "FROM 1800 TO 1810"},
			{Value: "garbage", Date: "sometime"},
		}, "dated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DefaultParentOf(tt.periods))
		})
	}
}

func TestPlace_ParentAt(t *testing.T) {
	wayne := wayneCounty()
	tests := []struct {
		date DateString
		want string
	}{
		{"", "place-indiana"},                            // no date: default parent
		{"1814-05-02", "place-indiana-territory"},        // inside the territory period
		{"1813", "place-indiana-territory"},              // year precision
		{"1817", "place-indiana"},                        // after statehood
		{"1816-12-10", "place-indiana-territory"},        // last territorial day
		{"1816-12-11", "place-indiana"},                  // first day of statehood
		{"1816", "place-indiana-territory"},              // most of 1816 was territorial
		{"1816-12", "place-indiana"},                     // most of December 1816 was statehood
		{"ABT 1815", "place-indiana-territory"},          // qualifiers are ignored
		{"FROM 1812 TO 1815", "place-indiana-territory"}, // a range inside one period
		{"1805", "place-indiana-territory"},              // before the first period: nearest
		{"not a date", "place-indiana"},                  // unparseable: default parent
	}
	for _, tt := range tests {
		t.Run(string(tt.date), func(t *testing.T) {
			assert.Equal(t, tt.want, wayne.ParentAt(tt.date))
		})
	}

	t.Run("plain parent ignores the date", func(t *testing.T) {
		p := &Place{ParentID: "place-yorkshire"}
		assert.Equal(t, "place-yorkshire", p.ParentAt("1066"))
	})

	t.Run("undated entry is the fallback outside every period", func(t *testing.T) {
		p := &Place{}
		p.SetParentHistory([]PlaceParentPeriod{
			{Value: "a", Date: "FROM 1900 TO 1910"},
			{Value: "fallback"},
		})
		assert.Equal(t, "a", p.ParentAt("1905"))
		assert.Equal(t, "fallback", p.ParentAt("1850"))
	})

	t.Run("nil place", func(t *testing.T) {
		var p *Place
		assert.Empty(t, p.ParentAt("1900"))
	})
}

func TestPlace_ParentIDsAndReplace(t *testing.T) {
	p := &Place{}
	p.SetParentHistory([]PlaceParentPeriod{
		{Value: "a", Date: "TO 1800"},
		{Value: "b", Date: "FROM 1801 TO 1850"},
		{Value: "a", Date: "FROM 1851"},
	})
	assert.Equal(t, "a", p.ParentID)
	assert.Equal(t, []string{"a", "b"}, p.ParentIDs())

	assert.Equal(t, 1, p.ReplaceParentRef("b", "c"))
	assert.Equal(t, []string{"a", "c"}, p.ParentIDs())
	assert.Equal(t, 2, p.ReplaceParentRef("a", "z"))
	assert.Equal(t, "z", p.ParentID, "default parent follows the rename")

	plain := &Place{ParentID: "x"}
	assert.Equal(t, []string{"x"}, plain.ParentIDs())
	assert.Equal(t, 1, plain.ReplaceParentRef("x", "y"))
	assert.Equal(t, "y", plain.ParentID)
	assert.Empty(t, (&Place{}).ParentIDs())
}

func TestGLXFile_PlaceHasAncestor(t *testing.T) {
	archive := indianaArchive()

	assert.True(t, archive.PlaceHasAncestor("place-richmond", "place-indiana-territory"))
	assert.True(t, archive.PlaceHasAncestor("place-richmond", "place-indiana"))
	assert.True(t, archive.PlaceHasAncestor("place-richmond", "place-usa"))
	assert.False(t, archive.PlaceHasAncestor("place-indiana", "place-richmond"))
	assert.False(t, archive.PlaceHasAncestor("place-richmond", "place-richmond"))

	archive.Places["place-a"] = &Place{ParentID: "place-b"}
	archive.Places["place-b"] = &Place{ParentID: "place-a"}
	assert.False(t, archive.PlaceHasAncestor("place-a", "place-usa"), "cycles terminate")
}

func TestValidate_TemporalPlaceParent(t *testing.T) {
	placeErrors := func(result *ValidationResult) []string {
		var msgs []string
		for _, e := range result.Errors {
			// Place types are not under test (no vocabulary is loaded).
			if e.SourceType == EntityTypePlaces && e.SourceField != "Type" {
				msgs = append(msgs, e.Message)
			}
		}

		return msgs
	}
	placeWarnings := func(result *ValidationResult) []string {
		var msgs []string
		for _, w := range result.Warnings {
			if w.SourceType == EntityTypePlaces {
				msgs = append(msgs, w.Message)
			}
		}

		return msgs
	}

	t.Run("valid temporal parent", func(t *testing.T) {
		archive := indianaArchive()
		result := archive.Validate()
		assert.Empty(t, placeErrors(result))
		assert.Empty(t, placeWarnings(result))
	})

	t.Run("every period's parent must exist", func(t *testing.T) {
		archive := indianaArchive()
		delete(archive.Places, "place-indiana-territory")
		errs := placeErrors(archive.Validate())
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "places[place-wayne-in].parent[0] references non-existent places: place-indiana-territory")
	})

	t.Run("default parent is reported once", func(t *testing.T) {
		archive := indianaArchive()
		delete(archive.Places, "place-indiana")
		errs := placeErrors(archive.Validate())
		// place-wayne-in's default parent, via ParentID only (not again as parent[1]).
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "place-wayne-in")
	})

	t.Run("cycle through an earlier period", func(t *testing.T) {
		archive := indianaArchive()
		// Indiana Territory's parent points back at Wayne County, closing a
		// loop that only the territorial period's edge reveals.
		archive.Places["place-indiana-territory"].ParentID = "place-wayne-in"
		errs := placeErrors(archive.Validate())
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "cycle detected")
		assert.Contains(t, errs[0], "place-indiana-territory")
	})

	t.Run("overlapping periods warn", func(t *testing.T) {
		archive := indianaArchive()
		archive.Places["place-wayne-in"].SetParentHistory([]PlaceParentPeriod{
			{Value: "place-indiana-territory", Date: "FROM 1811 TO 1820"},
			{Value: "place-indiana", Date: "FROM 1816"},
		})
		warns := placeWarnings(archive.Validate())
		require.Len(t, warns, 1)
		assert.Contains(t, warns[0], "periods overlap")
	})

	t.Run("boundaries shared at year precision do not warn", func(t *testing.T) {
		archive := &GLXFile{Places: map[string]*Place{
			"place-amt-huettenberg": {Name: "Amt Hüttenberg"},
			"place-kreis-friedberg": {Name: "Kreis Friedberg"},
			"place-wetteraukreis":   {Name: "Wetteraukreis"},
			"place-pohl-goens":      {Name: "Pohl-Göns"},
		}}
		archive.Places["place-pohl-goens"].SetParentHistory([]PlaceParentPeriod{
			{Value: "place-amt-huettenberg", Date: "TO 1821"},
			{Value: "place-kreis-friedberg", Date: "FROM 1840 TO 1972"},
			{Value: "place-wetteraukreis", Date: "FROM 1972"},
		})
		assert.Empty(t, placeWarnings(archive.Validate()))
	})

	t.Run("invalid period date warns", func(t *testing.T) {
		archive := indianaArchive()
		archive.Places["place-wayne-in"].ParentHistory[0].Date = "sometime in 1811"
		warns := placeWarnings(archive.Validate())
		require.NotEmpty(t, warns)
		assert.Contains(t, strings.Join(warns, "\n"), "parent[0].date")
	})

	t.Run("entry without a value is an error", func(t *testing.T) {
		archive := indianaArchive()
		archive.Places["place-wayne-in"].ParentHistory[0].Value = ""
		errs := placeErrors(archive.Validate())
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0], "parent entry has no value")
	})
}

func TestExportGEDCOM_TemporalPlaceParent(t *testing.T) {
	archive := indianaArchive()
	archive.Persons = map[string]*Person{
		"person-1": {Properties: map[string]any{"name": "John Smith"}},
	}
	archive.Events = map[string]*Event{
		"event-birth": {
			Type: "birth", Date: "1814-03-01", PlaceID: "place-richmond",
			Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
		},
		"event-death": {
			Type: "death", Date: "1870", PlaceID: "place-richmond",
			Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
		},
		"event-burial": {
			Type: "burial", PlaceID: "place-wayne-in",
			Participants: []Participant{{Person: "person-1", Role: ParticipantRolePrincipal}},
		},
	}

	data, _, err := ExportGEDCOM(archive, GEDCOM551, nil)
	require.NoError(t, err)
	out := string(data)

	assert.Contains(t, out, "1 BIRT\n2 DATE 1 MAR 1814\n2 PLAC Richmond, Wayne County, Indiana Territory, United States\n")
	assert.Contains(t, out, "1 DEAT\n2 DATE 1870\n2 PLAC Richmond, Wayne County, Indiana, United States\n")
	assert.Contains(t, out, "1 BURI\n2 PLAC Wayne County, Indiana, United States\n", "undated events use the default parent")
}

func TestRenameEntity_TemporalPlaceParent(t *testing.T) {
	archive := indianaArchive()

	_, err := RenameEntity(archive, "place-indiana-territory", "place-in-territory")
	require.NoError(t, err)
	assert.Equal(t, "place-in-territory", archive.Places["place-wayne-in"].ParentHistory[0].Value)
	assert.Equal(t, "place-indiana", archive.Places["place-wayne-in"].ParentID)
}

func TestThreeWayMerge_TemporalPlaceParent(t *testing.T) {
	base := &GLXFile{Places: map[string]*Place{"place-wayne-in": {Name: "Wayne County", ParentID: "place-indiana"}}}
	ours := &GLXFile{Places: map[string]*Place{"place-wayne-in": wayneCounty()}}
	ours.Places["place-wayne-in"].Type = ""
	theirs := &GLXFile{Places: map[string]*Place{"place-wayne-in": {Name: "Wayne County", ParentID: "place-indiana", Notes: NoteList{"seat: Richmond"}}}}

	merged, conflicts := ThreeWayMerge(base, ours, theirs)
	require.Empty(t, conflicts)
	got := merged.Places["place-wayne-in"]
	assert.Equal(t, wayneCounty().ParentHistory, got.ParentHistory, "ours switched to the temporal form; theirs did not touch parent")
	assert.Equal(t, "place-indiana", got.ParentID)
	assert.Equal(t, NoteList{"seat: Richmond"}, got.Notes)

	t.Run("diverging parents conflict", func(t *testing.T) {
		theirs := &GLXFile{Places: map[string]*Place{"place-wayne-in": {Name: "Wayne County", ParentID: "place-ohio"}}}
		_, conflicts := ThreeWayMerge(base, ours, theirs)
		require.Len(t, conflicts, 1)
		assert.Equal(t, "places[place-wayne-in].parent", conflicts[0].Path)
	})
}
