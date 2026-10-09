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

// Tests for reading a person's events by their role in them: migrations
// (#1370), timeline (#1363) and summary (#1362). The scenario is a
// 17th-century parish register: a man born in Niederkleen marries in
// Pohl-Göns, a neighboring village of the same district, and his infant son
// is buried in Butzbach, a war refuge in another territory.

func parishRegisterArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"p-jg":     {Properties: map[string]any{"name": "Johann Georg Schöpff"}},
			"p-bride":  {Properties: map[string]any{"name": "Elisabeth Roth"}},
			"p-child":  {Properties: map[string]any{"name": "Michell Schöpff"}},
			"p-father": {Properties: map[string]any{"name": "Wörner Roth"}},
			"p-other":  {Properties: map[string]any{"name": "Hans Other"}},
		},
		Places: map[string]*glxlib.Place{
			"pl-hre": {Name: "Heiliges Römisches Reich", Type: glxlib.PlaceTypeCountry},
			"pl-amt": {Name: "Amt Hüttenberg", Type: "district", ParentID: "pl-hre"},
			"pl-hd":  {Name: "Landgrafschaft Hessen-Darmstadt", Type: "state", ParentID: "pl-hre"},
			"pl-nk":  {Name: "Niederkleen", Type: "village", ParentID: "pl-amt"},
			"pl-pg":  {Name: "Pohl-Göns", Type: "village", ParentID: "pl-amt"},
			"pl-bz":  {Name: "Butzbach", Type: "town", ParentID: "pl-hd"},
		},
		Events: map[string]*glxlib.Event{
			"ev-birth": {
				Type: glxlib.EventTypeBirth, Date: "ABT 1621", PlaceID: "pl-nk",
				Participants: []glxlib.Participant{{Person: "p-jg", Role: glxlib.ParticipantRolePrincipal}},
			},
			"ev-bapt-other": {
				Type: glxlib.EventTypeBaptism, Date: "1642-07-03", PlaceID: "pl-pg",
				Participants: []glxlib.Participant{
					{Person: "p-other", Role: glxlib.ParticipantRolePrincipal},
					{Person: "p-jg", Role: glxlib.ParticipantRoleGodparent},
				},
			},
			"ev-marr": {
				Type: glxlib.EventTypeMarriage, Date: "1644-02-26", PlaceID: "pl-pg",
				Participants: []glxlib.Participant{
					{Person: "p-jg", Role: glxlib.ParticipantRoleGroom},
					{Person: "p-bride", Role: glxlib.ParticipantRoleBride},
					{Person: "p-father", Role: glxlib.ParticipantRoleParent},
				},
			},
			"ev-bapt": {
				Type: glxlib.EventTypeBaptism, Date: "1645-10-12", PlaceID: "pl-pg",
				Participants: []glxlib.Participant{
					{Person: "p-child", Role: glxlib.ParticipantRolePrincipal},
					{Person: "p-jg", Role: glxlib.ParticipantRoleParent},
				},
			},
			"ev-burial-child": {
				Type: glxlib.EventTypeBurial, Date: "1646-08-07", PlaceID: "pl-bz",
				Participants: []glxlib.Participant{
					{Person: "p-child", Role: glxlib.ParticipantRolePrincipal},
					{Person: "p-jg", Role: glxlib.ParticipantRoleParent},
				},
			},
			"ev-death-bride": {
				Type: glxlib.EventTypeDeath, Date: "1688-05-04", PlaceID: "pl-pg",
				Participants: []glxlib.Participant{
					{Person: "p-bride", Role: glxlib.ParticipantRolePrincipal},
					{Person: "p-jg", Role: glxlib.ParticipantRoleSpouse},
				},
			},
			"ev-death": {
				Type: glxlib.EventTypeDeath, Date: "1689-12-27", PlaceID: "pl-pg",
				Participants: []glxlib.Participant{{Person: "p-jg", Role: glxlib.ParticipantRolePrincipal}},
			},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marr": {
				Type: glxlib.RelationshipTypeMarriage,
				Participants: []glxlib.Participant{
					{Person: "p-jg", Role: glxlib.ParticipantRoleSpouse},
					{Person: "p-bride", Role: glxlib.ParticipantRoleSpouse},
				},
			},
		},
	}
}

func TestMigrations_ChildBurialElsewhereIsNotAMove(t *testing.T) {
	entries := collectMigrationEntries("p-jg", parishRegisterArchive())

	burial := migrationEntryByLabel(t, entries, "Burial")
	assert.True(t, burial.Excluded, "a child's burial places the child, not the father")
	assert.Equal(t, "parent", burial.Role)
	assert.Contains(t, burial.Note, `role "parent" on a burial`)

	// The father's own baptism record of his child still places him at home.
	assert.False(t, migrationEntryByLabel(t, entries, "Baptism").Excluded)
}

func TestMigrations_MoveBetweenVillagesOfOneDistrict(t *testing.T) {
	movements := computeMovements(collectMigrationEntries("p-jg", parishRegisterArchive()))

	require.Len(t, movements, 1, "movements: %+v", movements)
	assert.Equal(t, migrationMovement{
		FromRegion: "Niederkleen", ToRegion: "Pohl-Göns",
		FromDate: "ABT 1621", ToDate: "1642-07-03",
	}, movements[0])
}

func TestMigrations_PatternMatchesVillages(t *testing.T) {
	report := matchMigrationPattern(parishRegisterArchive(), []string{"Niederkleen", "Pohl-Göns"})

	require.Len(t, report.Matches, 1)
	assert.Equal(t, "p-jg", report.Matches[0].Person)
	// The child was baptized in Pohl-Göns and buried in Butzbach; his father
	// did not move there.
	refuge := matchMigrationPattern(parishRegisterArchive(), []string{"Pohl-Göns", "Butzbach"})
	require.Len(t, refuge.Matches, 1)
	assert.Equal(t, "p-child", refuge.Matches[0].Person)
}

func TestMigrations_ParentAtOwnRecordStillCounts(t *testing.T) {
	// A person who is both the decedent and, oddly, listed as parent is still
	// at their own death.
	event := &glxlib.Event{Type: glxlib.EventTypeDeath, Participants: []glxlib.Participant{
		{Person: "p", Role: glxlib.ParticipantRoleParent},
		{Person: "p", Role: glxlib.ParticipantRolePrincipal},
	}}
	assert.False(t, isParentAtDeathRecord(event, "p"))

	assert.True(t, isParentAtDeathRecord(&glxlib.Event{Type: "Funeral", Participants: []glxlib.Participant{
		{Person: "c"}, {Person: "p", Role: "Mother"},
	}}, "p"))
	assert.False(t, isParentAtDeathRecord(&glxlib.Event{Type: glxlib.EventTypeBaptism, Participants: []glxlib.Participant{
		{Person: "c"}, {Person: "p", Role: glxlib.ParticipantRoleParent},
	}}, "p"), "a baptism is evidence of the parents' parish")
}

func TestPlaceDivergence(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		from, to string
		moved    bool
	}{
		{"same place", []string{"WI", "Millbrook"}, []string{"wi", "millbrook"}, "", "", false},
		{"state then a town in it", []string{"Wisconsin"}, []string{"Wisconsin", "Hartford Co.", "Millbrook"}, "", "", false},
		{"town then its state", []string{"Wisconsin", "Hartford Co.", "Millbrook"}, []string{"Wisconsin"}, "", "", false},
		{"two villages of one district", []string{"Amt", "Niederkleen"}, []string{"Amt", "Pohl-Göns"}, "Niederkleen", "Pohl-Göns", true},
		{"two states", []string{"Florida"}, []string{"Wisconsin", "Hartford Co.", "Millbrook"}, "Florida", "Wisconsin", true},
		{"two counties of one state", []string{"IN", "Dearborn"}, []string{"IN", "Wayne", "Richmond"}, "Dearborn", "Wayne", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, moved := placeDivergence(tt.a, tt.b)
			assert.Equal(t, tt.moved, moved)
			assert.Equal(t, tt.from, from)
			assert.Equal(t, tt.to, to)
		})
	}
}

func TestMigrationPlacePath(t *testing.T) {
	archive := parishRegisterArchive()
	assert.Equal(t, []string{"Amt Hüttenberg", "Niederkleen"}, migrationPlacePath("pl-nk", archive.Places))
	assert.Equal(t, []string{"Heiliges Römisches Reich"}, migrationPlacePath("pl-hre", archive.Places),
		"a chain of containers keeps its names")
	assert.Equal(t, []string{"Florida", "Tampa"}, migrationFreeformPath("Tampa, Florida Territory"))
}

func TestMigrations_LessSpecificObservationIsNotAMove(t *testing.T) {
	// Millbrook, then a residence given only as the state, then Millbrook
	// again: no movement in either direction.
	archive := migrationArchive()
	archive.Persons["person-jane-webb"].Properties["residence"] = []any{
		map[string]any{"value": "place-wisconsin", "date": 1855},
	}
	archive.Events["event-jane-birth"].PlaceID = "place-millbrook"

	assert.Empty(t, computeMovements(collectMigrationEntries("person-jane-webb", archive)))
}

func TestTimeline_OtherPeoplesRecordsAreLabeled(t *testing.T) {
	entries := collectDirectEvents("p-jg", parishRegisterArchive())

	labels := map[string]string{}
	for _, e := range entries {
		labels[e.EventID] = e.Label
	}
	assert.Equal(t, "Birth", labels["ev-birth"])
	assert.Equal(t, "Marriage", labels["ev-marr"])
	assert.Equal(t, "Death", labels["ev-death"])
	assert.Equal(t, "Baptism of Hans Other (godparent)", labels["ev-bapt-other"])
	assert.Equal(t, "Baptism of Michell Schöpff (parent)", labels["ev-bapt"])
	assert.Equal(t, "Burial of Michell Schöpff (parent)", labels["ev-burial-child"])
	assert.Equal(t, "Death of Elisabeth Roth (spouse)", labels["ev-death-bride"])
}

func TestTimeline_FamilyEventsAreTheRelativesOwn(t *testing.T) {
	// The bride is named as spouse on her husband's death; her family events
	// must not report it as "Death of spouse (Johann Georg Schöpff)" twice,
	// nor his as hers.
	archive := parishRegisterArchive()
	archive.Events["ev-death"].Participants = append(archive.Events["ev-death"].Participants,
		glxlib.Participant{Person: "p-bride", Role: glxlib.ParticipantRoleSpouse})

	var deaths []string
	for _, e := range collectFamilyEvents("p-bride", archive) {
		if strings.HasPrefix(e.Label, "Death") {
			deaths = append(deaths, e.EventID+" "+e.Label)
		}
	}
	assert.Equal(t, []string{"ev-death Death of spouse (Johann Georg Schöpff)"}, deaths)
}

func TestSummary_SpousesComeFromTheCouple(t *testing.T) {
	archive := parishRegisterArchive()
	delete(archive.Relationships, "rel-marr")

	spouses := findSpouses("p-jg", archive)
	groomSpouses := make([]string, 0, len(spouses))
	for _, sp := range spouses {
		groomSpouses = append(groomSpouses, sp.PersonID)
	}
	assert.Equal(t, []string{"p-bride"}, groomSpouses, "the bride's father is not the groom's spouse")
	assert.Empty(t, findSpouses("p-father", archive), "a father at his daughter's wedding has no spouse in it")
}

func TestSummary_LifeHistoryDeathIsThePersonsOwn(t *testing.T) {
	archive := parishRegisterArchive()

	date, _ := findEventDatePlace("p-jg", "death", archive)
	assert.Equal(t, "1689-12-27", date, "not the wife's death he is named on")
	assert.Contains(t, generateLifeHistory("p-jg", archive.Persons["p-jg"], archive), "died on December 27, 1689")

	date, _ = findMarriageEvent("p-jg", "p-father", archive)
	assert.Empty(t, date, "a father named on the marriage is not of the couple")
}
