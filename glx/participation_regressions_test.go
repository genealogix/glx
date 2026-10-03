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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	glxlib "github.com/genealogix/glx/go-glx"
)

func migrationParentArchive() *glxlib.GLXFile {
	no := false

	return &glxlib.GLXFile{
		Persons:          map[string]*glxlib.Person{"p": {}, "child": {}},
		ParticipantRoles: map[string]*glxlib.VocabularyEntry{"parent": {ImpliesPresence: &no}},
		Relationships:    map[string]*glxlib.Relationship{"r": {Type: "parent_child", Participants: []glxlib.Participant{{Person: "p", Role: "parent"}, {Person: "child", Role: "child"}}}},
		Events:           map[string]*glxlib.Event{"birth": {Type: "birth", Date: "1850", PlaceID: "Away, Illinois", Participants: []glxlib.Participant{{Person: "child", Role: "child"}, {Person: "p", Role: "parent"}}}},
	}
}

func TestMigrationParentPresenceOverride(t *testing.T) {
	archive := migrationParentArchive()
	entries := collectMigrationEntries("p", archive)
	require.Len(t, entries, 1)
	require.True(t, entries[0].Excluded)
	require.Equal(t, "parent", entries[0].Role)
	require.Contains(t, entries[0].Note, "does not imply presence")
}

func TestMigrationChildWitnessIsNotChildBirth(t *testing.T) {
	archive := migrationParentArchive()
	archive.Events["birth"].Participants = []glxlib.Participant{{Person: "child", Role: "witness"}}
	require.Empty(t, collectMigrationEntries("p", archive))
}

func TestMigrationDuplicateRolesKeepPresentObservation(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"residence": []any{map[string]any{"value": "Home, Indiana", "date": "1840"}}}}},
		Events: map[string]*glxlib.Event{
			"a-remote":  {Type: "generic", Title: "Deed", Date: "1850", PlaceID: "Away, Illinois", Participants: []glxlib.Participant{{Person: "p", Role: "grantor"}}},
			"z-present": {Type: "generic", Title: "Deed", Date: "1850", PlaceID: "Away, Illinois", Participants: []glxlib.Participant{{Person: "p", Role: "principal"}}},
		},
	}
	entries := collectMigrationEntries("p", archive)
	require.Len(t, entries, 3)
	require.Len(t, computeMovements(entries), 1, "the excluded duplicate cannot hide real presence")
}

func TestMigrationNilEntitiesAndCycles(t *testing.T) {
	archive := migrationParentArchive()
	archive.Relationships["nil"] = nil
	archive.Events["nil"] = nil
	archive.Places = map[string]*glxlib.Place{"a": {Name: "A", ParentID: "b"}, "b": {Name: "B", ParentID: "a"}, "nil": nil}
	archive.Events["birth"].PlaceID = "a"
	require.NotPanics(t, func() { collectMigrationEntries("p", archive) })
}

func TestMigrationResidenceBoundsAndOverlaps(t *testing.T) {
	bounded := []any{map[string]any{"value": "Home, Indiana", "date": "FROM 1809 TO 1810"}}
	overlapping := []any{
		map[string]any{"value": "Home, Indiana", "date": "FROM 1809 TO 1812"},
		map[string]any{"value": "Away, Illinois", "date": "FROM 1810 TO 1813"},
	}
	for _, tt := range []struct {
		name       string
		residences []any
		date       string
		place      string
		excluded   bool
	}{
		{"start boundary", bounded, "1809-01-01", "Away, Illinois", true},
		{"end boundary", bounded, "1810-12-31", "Away, Illinois", true},
		{"after boundary", bounded, "1811-01-01", "Away, Illinois", false},
		{"overlap agrees first", overlapping, "1811", "Home, Indiana", false},
		{"overlap agrees second", overlapping, "1811", "Away, Illinois", false},
		{"overlap agrees neither", overlapping, "1811", "Other, Ohio", true},
		{"unbounded start", []any{map[string]any{"value": "Home, Indiana", "date": "FROM 1809"}}, "1810", "Away, Illinois", false},
		{"unbounded end", []any{map[string]any{"value": "Home, Indiana", "date": "TO 1810"}}, "1809", "Away, Illinois", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				residences := append([]any(nil), tt.residences...)
				if reverse {
					for i, j := 0, len(residences)-1; i < j; i, j = i+1, j-1 {
						residences[i], residences[j] = residences[j], residences[i]
					}
				}
				archive := &glxlib.GLXFile{
					Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"residence": residences}}},
					Events: map[string]*glxlib.Event{"event": {
						Type: "generic", Title: "Observation", Date: glxlib.DateString(tt.date), PlaceID: tt.place,
						Participants: []glxlib.Participant{{Person: "p", Role: "principal"}},
					}},
				}
				entries := collectMigrationEntries("p", archive)
				observation := migrationEntryByLabel(t, entries, "Observation")
				require.Equal(t, tt.excluded, observation.Excluded, "reversed residence order: %v", reverse)
			}
		})
	}
}

func TestConfidenceConsumersSkipNullAssertions(t *testing.T) {
	for _, name := range []string{"all null", "mixed"} {
		t.Run(name, func(t *testing.T) {
			mixed := name == "mixed"
			data := "assertions:\n  assertion-null: null\n"
			if mixed {
				data += "  assertion-valid:\n    subject: {person: p}\n    confidence: high\n"
			}
			path := filepath.Join(t.TempDir(), "archive.glx")
			require.NoError(t, os.WriteFile(path, []byte(data), 0o644))
			archive, err := loadArchiveForStats(path)
			require.NoError(t, err)
			require.Contains(t, archive.Assertions, "assertion-null")
			require.Nil(t, archive.Assertions["assertion-null"])
			report := buildConfidenceReport(archive)
			output := captureStdout(t, func() { printConfidenceDistribution(archive) })
			if mixed {
				require.Equal(t, 1, report.TotalAssertions)
				require.Equal(t, map[string]int{"high": 1}, report.ByConfidence)
				require.Contains(t, output, "100.0%")
			} else {
				require.Zero(t, report.TotalAssertions)
				require.Empty(t, report.ByConfidence)
				require.Empty(t, output)
			}
		})
	}
}
