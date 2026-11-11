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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

// Tests for the role-aware behavior of glx coverage (#1329), glx migrations
// (#1330), and the evidence coverage shared by glx stats, glx serve and
// glx validate --report (#713).

// lewisLittleArchive is the shape of the archive behind #1329 and #1330:
// Lewis appears in his father-in-law's probate and his godson's baptism,
// sells land in North Carolina after moving to Indiana, and enters land in
// Illinois while still living in Indiana.
func lewisLittleArchive() *glxlib.GLXFile {
	no := false

	return &glxlib.GLXFile{
		ParticipantRoles: map[string]*glxlib.VocabularyEntry{
			"absent_heir": {Label: "Absent heir", ImpliesPresence: &no},
		},
		Persons: map[string]*glxlib.Person{
			"person-lewis": {Properties: map[string]any{
				"name": "Lewis Little",
				"residence": []any{
					map[string]any{"value": "place-rowan-nc", "date": "FROM 1790 TO 1808"},
					map[string]any{"value": "place-dearborn-in", "date": "FROM 1809 TO 1810"},
					map[string]any{"value": "place-wayne-in", "date": "1818"},
					map[string]any{"value": "place-crawford-il", "date": "FROM 1820 TO 1826"},
				},
			}},
			"person-caspar": {Properties: map[string]any{"name": "Caspar Stoehr"}},
			"person-adam":   {Properties: map[string]any{"name": "Adam Little"}},
		},
		Places: map[string]*glxlib.Place{
			"place-us":          {Name: "United States", Type: glxlib.PlaceTypeCountry},
			"place-nc":          {Name: "North Carolina", Type: "state", ParentID: "place-us"},
			"place-in":          {Name: "Indiana Territory", Type: "state", ParentID: "place-us"},
			"place-il":          {Name: "Illinois", Type: "state", ParentID: "place-us"},
			"place-rowan-nc":    {Name: "Rowan County", Type: "county", ParentID: "place-nc"},
			"place-dearborn-in": {Name: "Dearborn County", Type: "county", ParentID: "place-in"},
			"place-wayne-in":    {Name: "Wayne County", Type: "county", ParentID: "place-in"},
			"place-crawford-il": {Name: "Crawford County", Type: "county", ParentID: "place-il"},
		},
		Sources: map[string]*glxlib.Source{
			"src-estate-files": {Title: "North Carolina, Estate Files"},
		},
		Events: map[string]*glxlib.Event{
			"ev-birth-lewis": {
				Type: glxlib.EventTypeBirth, Date: "1765",
				Participants: []glxlib.Participant{{Person: "person-lewis", Role: glxlib.ParticipantRolePrincipal}},
			},
			"ev-estate-stoehr-1818": {
				Type: glxlib.EventTypeProbate, Date: "1818-04-03", PlaceID: "place-rowan-nc",
				Participants: []glxlib.Participant{
					{Person: "person-caspar", Role: glxlib.ParticipantRolePrincipal},
					{Person: "person-lewis", Role: "legatee"},
				},
			},
			"ev-bapt-adam-little": {
				Type: glxlib.EventTypeBaptism, Date: "1812", PlaceID: "place-dearborn-in",
				Participants: []glxlib.Participant{
					{Person: "person-adam", Role: glxlib.ParticipantRolePrincipal},
					{Person: "person-lewis", Role: glxlib.ParticipantRoleGodparent},
				},
			},
			"ev-deed-1810": {
				Type: glxlib.EventTypeGeneric, Title: "Deed", Date: "1810-10-19", PlaceID: "place-rowan-nc",
				Participants: []glxlib.Participant{{Person: "person-lewis", Role: "grantor"}},
			},
			"ev-land-entry-1818": {
				Type: glxlib.EventTypeGeneric, Title: "Land entry", Date: "1818-09-04", PlaceID: "place-crawford-il",
				Participants: []glxlib.Participant{{Person: "person-lewis", Role: "grantee"}},
			},
			"ev-sale-1808": {
				Type: glxlib.EventTypeGeneric, Title: "Sale of farm", Date: "1808-12-19", PlaceID: "place-rowan-nc",
				Participants: []glxlib.Participant{{Person: "person-lewis", Role: glxlib.ParticipantRolePrincipal}},
			},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-estate-date": {
				Subject: glxlib.EntityRef{Event: "ev-estate-stoehr-1818"}, Property: "date", Value: "1818-04-03",
				Sources: []string{"src-estate-files"},
			},
			"a-bapt-date": {
				Subject: glxlib.EntityRef{Event: "ev-bapt-adam-little"}, Property: "date", Value: "1812",
				Sources: []string{"src-estate-files"},
			},
		},
	}
}

func findCoverageRecord(t *testing.T, result *coverageResult, label string) coverageRecord {
	t.Helper()
	for _, r := range result.Records {
		if r.Label == label {
			return r
		}
	}
	t.Fatalf("no coverage record %q in %+v", label, result.Records)

	return coverageRecord{}
}

func TestBuildCoverage_OtherPeoplesRecordsDoNotCount(t *testing.T) {
	archive := lewisLittleArchive()
	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)

	probate := findCoverageRecord(t, result, "Probate/will")
	assert.False(t, probate.Found, "a legatee's appearance in a father-in-law's probate is not the person's own probate")
	assert.NotContains(t, probate.Description, "ev-estate-stoehr-1818")

	church := findCoverageRecord(t, result, "Church records")
	assert.False(t, church.Found, "a godparent's appearance at a godson's baptism is not the person's own church record")

	require.Len(t, result.AppearsIn, 4)
	byID := map[string]coverageAppearance{}
	for _, a := range result.AppearsIn {
		byID[a.EventID] = a
	}
	assert.Equal(t, "Probate of Caspar Stoehr", byID["ev-estate-stoehr-1818"].Label)
	assert.Equal(t, "legatee", byID["ev-estate-stoehr-1818"].Role)
	assert.Equal(t, "Baptism of Adam Little", byID["ev-bapt-adam-little"].Label)
	assert.Equal(t, "godparent", byID["ev-bapt-adam-little"].Role)
	assert.Equal(t, "Deed", byID["ev-deed-1810"].Label, "an event with no principal is named by its title")
	assert.NotContains(t, byID, "ev-birth-lewis")
	assert.NotContains(t, byID, "ev-sale-1808")
}

func TestBuildCoverage_OwnProbateStillCounts(t *testing.T) {
	archive := lewisLittleArchive()
	archive.Events["ev-estate-lewis-1827"] = &glxlib.Event{
		Type: glxlib.EventTypeProbate, Date: "1827",
		Participants: []glxlib.Participant{{Person: "person-lewis", Role: "decedent"}},
	}
	archive.Assertions["a-estate-lewis"] = &glxlib.Assertion{
		Subject: glxlib.EntityRef{Event: "ev-estate-lewis-1827"}, Property: "date", Value: "1827",
		Sources: []string{"src-estate-files"},
	}

	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)

	probate := findCoverageRecord(t, result, "Probate/will")
	assert.True(t, probate.Found)
	assert.Equal(t, "ev-estate-lewis-1827", probate.SourceRef)
}

func TestBuildCoverage_CensusHouseholdRoleCounts(t *testing.T) {
	setCensusFallback(t, countryUnitedStates)
	archive := lewisLittleArchive()
	archive.Events["ev-census-1820"] = &glxlib.Event{
		Type: glxlib.EventTypeCensus, Date: "1820",
		Participants: []glxlib.Participant{
			{Person: "person-adam", Role: glxlib.ParticipantRoleHouseholdHead},
			{Person: "person-lewis", Role: "father"},
		},
	}
	archive.Assertions["a-census-1820"] = &glxlib.Assertion{
		Subject: glxlib.EntityRef{Event: "ev-census-1820"}, Property: "date", Value: "1820",
		Sources: []string{"src-estate-files"},
	}

	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)

	census := findCoverageRecord(t, result, "1820 US Census (age ~55)")
	assert.True(t, census.Found, "a named household member's census is their own record")
	assert.Equal(t, "ev-census-1820", census.SourceRef)
}

func TestBuildCoverage_WitnessBurialDoesNotInferDeath(t *testing.T) {
	archive := lewisLittleArchive()
	archive.Events["ev-burial-caspar"] = &glxlib.Event{
		Type: glxlib.EventTypeBurial, Date: "1817",
		Participants: []glxlib.Participant{
			{Person: "person-caspar", Role: glxlib.ParticipantRolePrincipal},
			{Person: "person-lewis", Role: glxlib.ParticipantRoleWitness},
		},
	}

	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)

	// Someone else's burial must not cap the person's life at 1817
	findCoverageRecord(t, result, "1820 US Census (age ~55)")
}

func TestPrintCoverageText_AppearsIn(t *testing.T) {
	archive := lewisLittleArchive()
	result := mustBuildCoverage("person-lewis", archive.Persons["person-lewis"], archive)

	out := captureStdout(t, func() { printCoverageText(result) })

	assert.Contains(t, out, "Appears in (other people's records, not counted above):")
	assert.Contains(t, out, "Probate of Caspar Stoehr (legatee) -- ev-estate-stoehr-1818, 1818-04-03")
	assert.Contains(t, out, "[ ] Probate/will")
}

func migrationEntryByLabel(t *testing.T, entries []migrationEntry, label string) migrationEntry {
	t.Helper()
	for i := range entries {
		if entries[i].Label == label {
			return entries[i]
		}
	}
	t.Fatalf("no migration entry %q in %+v", label, entries)

	return migrationEntry{}
}

func TestCollectMigrationEntries_RolesAndResidencePeriods(t *testing.T) {
	archive := lewisLittleArchive()
	entries := collectMigrationEntries("person-lewis", archive)

	deed := migrationEntryByLabel(t, entries, "Deed")
	assert.True(t, deed.Excluded, "a grantor selling land from another state was not there")
	assert.Equal(t, "grantor", deed.Role)
	assert.Contains(t, deed.Note, `role "grantor" does not imply presence`)

	land := migrationEntryByLabel(t, entries, "Land entry")
	assert.True(t, land.Excluded)

	probate := migrationEntryByLabel(t, entries, "Probate")
	assert.True(t, probate.Excluded, "an absent legatee was not at the estate's place")

	// The baptism is in Dearborn, where Lewis lived only to 1810; no
	// bounded residence covers 1812, so a present godparent counts
	baptism := migrationEntryByLabel(t, entries, "Baptism")
	assert.False(t, baptism.Excluded)

	sale := migrationEntryByLabel(t, entries, "Sale of farm")
	assert.False(t, sale.Excluded, "a principal is present")

	movements := computeMovements(entries)
	require.Len(t, movements, 3, "movements: %+v", movements)
	assert.Equal(t, "North Carolina", movements[0].FromRegion)
	assert.Equal(t, "Indiana", movements[0].ToRegion)
	// Two counties of one state are two places (#1370)
	assert.Equal(t, "Dearborn County", movements[1].FromRegion)
	assert.Equal(t, "Wayne County", movements[1].ToRegion)
	assert.Equal(t, "Indiana", movements[2].FromRegion)
	assert.Equal(t, "Illinois", movements[2].ToRegion)
	assert.Equal(t, "FROM 1820 TO 1826", movements[2].ToDate, "the land entry made from Wayne County is not the move")
}

func TestCollectMigrationEntries_ResidenceWinsOverEventPlace(t *testing.T) {
	archive := lewisLittleArchive()
	// Lewis is present (a witness) at a wedding back in North Carolina while
	// his residence places him in Dearborn County
	archive.Events["ev-wedding-1809"] = &glxlib.Event{
		Type: glxlib.EventTypeMarriage, Title: "Wedding of a cousin", Date: "1809-06-01", PlaceID: "place-rowan-nc",
		Participants: []glxlib.Participant{{Person: "person-lewis", Role: glxlib.ParticipantRoleWitness}},
	}
	// A census in his residence region during the same period agrees with it
	archive.Events["ev-census-1810"] = &glxlib.Event{
		Type: glxlib.EventTypeCensus, Title: "1810 Census", Date: "1810", PlaceID: "place-dearborn-in",
		Participants: []glxlib.Participant{{Person: "person-lewis", Role: glxlib.ParticipantRoleSubject}},
	}

	entries := collectMigrationEntries("person-lewis", archive)

	wedding := migrationEntryByLabel(t, entries, "Wedding of a cousin")
	assert.True(t, wedding.Excluded)
	assert.Equal(t, "inside residence FROM 1809 TO 1810 Dearborn County, Indiana Territory, United States", wedding.Note)

	census := migrationEntryByLabel(t, entries, "1810 Census")
	assert.False(t, census.Excluded)

	for _, m := range computeMovements(entries) {
		assert.NotEqual(t, "North Carolina", m.ToRegion, "no move back to North Carolina: %+v", m)
	}
}

func TestCollectMigrationEntries_VocabularyImpliesPresence(t *testing.T) {
	archive := lewisLittleArchive()
	archive.Events["ev-absent"] = &glxlib.Event{
		Type: glxlib.EventTypeGeneric, Title: "Partition suit", Date: "1830", PlaceID: "place-rowan-nc",
		Participants: []glxlib.Participant{{Person: "person-lewis", Role: "absent_heir"}},
	}
	entries := collectMigrationEntries("person-lewis", archive)
	assert.True(t, migrationEntryByLabel(t, entries, "Partition suit").Excluded)

	// An archive can also say its grantors were there
	yes := true
	archive.ParticipantRoles["grantor"] = &glxlib.VocabularyEntry{Label: "Grantor", ImpliesPresence: &yes}
	entries = collectMigrationEntries("person-lewis", archive)
	deed := migrationEntryByLabel(t, entries, "Deed")
	// Present now, but still inside the Dearborn residence period
	assert.True(t, deed.Excluded)
	assert.Contains(t, deed.Note, "inside residence FROM 1809 TO 1810")
}

func TestPrintMigrationReportText_ExcludedNote(t *testing.T) {
	archive := lewisLittleArchive()
	report := buildMigrationReport(archive, "person-lewis", "")

	io, out, _ := TestIOStreams()
	printMigrationReportText(io, &report)

	assert.Contains(t, out.String(), `(Deed) [not counted as a move: role "grantor" does not imply presence]`)
}

func TestBoundedMigrationDate(t *testing.T) {
	tests := []struct {
		date   string
		ok     bool
		inside []string
		out    []string
	}{
		{date: "FROM 1809 TO 1810", ok: true, inside: []string{"1809", "1810-10-19"}, out: []string{"1808-12-31", "1811"}},
		{date: "BET 1820 AND 1826", ok: true, inside: []string{"1823"}, out: []string{"1827"}},
		{date: "1818", ok: true, inside: []string{"1818-09-04"}, out: []string{"1819"}},
		{date: "ABT 1818", ok: true, inside: []string{"1818"}},
		{date: "FROM 1820", ok: false},
		{date: "TO 1820", ok: false},
		{date: "BEF 1820", ok: false},
		{date: "", ok: false},
		{date: "sometime", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			date, ok := boundedMigrationDate(tt.date)
			require.Equal(t, tt.ok, ok)
			for _, d := range tt.inside {
				other, bounded := boundedMigrationDate(d)
				require.True(t, bounded)
				assert.True(t, date.Timing().Outer.Contains(other.Timing().Outer), "%s should be inside %s", d, tt.date)
			}
			for _, d := range tt.out {
				other, bounded := boundedMigrationDate(d)
				require.True(t, bounded)
				assert.False(t, date.Timing().Outer.Contains(other.Timing().Outer), "%s should be outside %s", d, tt.date)
			}
		})
	}
}

func TestBuildConfidenceReport_EvidenceCoverage(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-bride":   {},
			"person-witness": {},
			"person-sponsor": {},
			"person-child":   {},
			"person-alone":   {},
		},
		Events: map[string]*glxlib.Event{
			"event-marriage": {Type: glxlib.EventTypeMarriage, Participants: []glxlib.Participant{
				{Person: "person-bride", Role: glxlib.ParticipantRoleBride},
				{Person: "person-witness", Role: glxlib.ParticipantRoleWitness},
			}},
			"event-baptism": {Type: glxlib.EventTypeBaptism},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-parentage": {Type: glxlib.RelationshipTypeParentChild, Participants: []glxlib.Participant{
				{Person: "person-bride", Role: glxlib.ParticipantRoleParent},
				{Person: "person-child", Role: glxlib.ParticipantRoleChild},
			}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-marriage": {Subject: glxlib.EntityRef{Event: "event-marriage"}, Property: "date", Value: "1800", Confidence: "high"},
			"a-sponsor": {
				Subject:     glxlib.EntityRef{Event: "event-baptism"},
				Participant: &glxlib.Participant{Person: "person-sponsor", Role: glxlib.ParticipantRoleGodparent},
			},
			"a-parentage": {Subject: glxlib.EntityRef{Relationship: "rel-parentage"}, Property: "certainty", Value: "proven"},
		},
	}

	report := buildConfidenceReport(archive)

	assert.Equal(t, []string{"person-alone"}, report.UnbackedPersons,
		"participants of asserted events and relationships, and participant.person, are backed")
	assert.Empty(t, report.UnbackedEvents)
	assert.Empty(t, report.UnbackedRelations)
}

func TestPrintEntityCoverage_DirectAndEvidence(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"person-a": {}, "person-b": {}},
		Places: map[string]*glxlib.Place{
			"place-country": {Name: "Country", Type: glxlib.PlaceTypeCountry},
			"place-town":    {Name: "Town", ParentID: "place-country"},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {Type: glxlib.EventTypeBirth, PlaceID: "place-town", Participants: []glxlib.Participant{
				{Person: "person-a", Role: glxlib.ParticipantRolePrincipal},
			}},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-birth":    {Subject: glxlib.EntityRef{Event: "event-birth"}, Property: "date", Value: "1850"},
			"a-dangling": {Subject: glxlib.EntityRef{Person: "person-missing"}, Property: "name", Value: "X"},
		},
	}

	out := captureStdout(t, func() { printEntityCoverage(archive) })

	direct, evidence, found := strings.Cut(out, "Evidence coverage")
	require.True(t, found, out)
	assert.Contains(t, direct, "Persons         0/2  (0.0%)", "a dangling subject is not coverage")
	assert.Contains(t, direct, "Places          0/2  (0.0%)")
	assert.Contains(t, evidence, "Persons         1/2  (50.0%)")
	assert.Contains(t, evidence, "Places          2/2  (100.0%)")
}
