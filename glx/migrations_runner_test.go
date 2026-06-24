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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// migrationArchive builds the issue #114 scenario: Jane Miller born in
// Florida Territory ~1832, in Millbrook, Hartford Co., Wisconsin by ~1852
// (eldest child's birth), with 1855 and 1860 census events and a residence
// property. A second person who never left Wisconsin exercises the negative
// pattern-match path.
func migrationArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane-webb": {Properties: map[string]any{
				"name": "Jane Miller",
				"residence": []any{
					map[string]any{"value": "place-millbrook", "date": 1855},
				},
			}},
			"person-robert-webb": {Properties: map[string]any{"name": "Robert Webb"}},
			"person-anna-lane":   {Properties: map[string]any{"name": "Anna Lane"}},
		},
		Places: map[string]*glxlib.Place{
			"place-florida-territory": {Name: "Florida Territory", Type: "region"},
			"place-usa":               {Name: "United States", Type: "country"},
			"place-wisconsin":         {Name: "Wisconsin", Type: "region", ParentID: "place-usa"},
			"place-hartford-co":       {Name: "Hartford Co.", Type: "county", ParentID: "place-wisconsin"},
			"place-millbrook":         {Name: "Millbrook", Type: "town", ParentID: "place-hartford-co"},
		},
		Events: map[string]*glxlib.Event{
			"event-jane-birth": {
				Type: "birth", Date: "ABT 1832", PlaceID: "place-florida-territory",
				Participants: []glxlib.Participant{{Person: "person-jane-webb", Role: "subject"}},
			},
			"event-robert-birth": {
				Type: "birth", Date: "ABT 1852", PlaceID: "place-millbrook",
				Participants: []glxlib.Participant{{Person: "person-robert-webb", Role: "subject"}},
			},
			"event-1860-census": {
				Type: "census", Title: "1860 US Census", Date: "1860", PlaceID: "place-millbrook",
				Participants: []glxlib.Participant{
					{Person: "person-jane-webb", Role: "subject"},
					{Person: "person-anna-lane", Role: "subject"},
				},
			},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-jane-robert": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-jane-webb", Role: "parent"},
					{Person: "person-robert-webb", Role: "child"},
				},
			},
		},
	}
}

func TestCollectMigrationEntries_OrderAndSources(t *testing.T) {
	entries := collectMigrationEntries("person-jane-webb", migrationArchive())

	if len(entries) != 4 {
		t.Fatalf("len(entries) = %d, want 4: %+v", len(entries), entries)
	}

	want := []struct {
		date   string
		region string
		label  string
	}{
		{"ABT 1832", "Florida", "Birth"},
		{"ABT 1852", "Wisconsin", "Birth of child (Robert Webb)"},
		{"1855", "Wisconsin", "Residence"},
		{"1860", "Wisconsin", "1860 US Census"},
	}
	for i, w := range want {
		e := entries[i]
		if e.Date != w.date || e.Region != w.region || e.Label != w.label {
			t.Errorf("entries[%d] = {%q, %q, %q}, want {%q, %q, %q}",
				i, e.Date, e.Region, e.Label, w.date, w.region, w.label)
		}
	}
}

func TestCollectMigrationEntries_ChildBirthDirectParticipationLabeled(t *testing.T) {
	archive := migrationArchive()
	// Jane participates in her child's birth event directly (as parent role).
	// The event must appear once and keep its child-birth provenance label —
	// a bare "Birth" would read as Jane's own birth.
	archive.Events["event-robert-birth"].Participants = append(
		archive.Events["event-robert-birth"].Participants,
		glxlib.Participant{Person: "person-jane-webb", Role: "parent"},
	)

	entries := collectMigrationEntries("person-jane-webb", archive)

	var matched []migrationEntry
	for _, e := range entries {
		if e.Date == "ABT 1852" {
			matched = append(matched, e)
		}
	}
	if len(matched) != 1 {
		t.Fatalf("child birth event appears %d times, want 1: %+v", len(matched), matched)
	}
	if matched[0].Label != "Birth of child (Robert Webb)" {
		t.Errorf("label = %q, want Birth of child (Robert Webb)", matched[0].Label)
	}
}

func TestComputeMovements_RegionChangeOnly(t *testing.T) {
	entries := collectMigrationEntries("person-jane-webb", migrationArchive())
	movements := computeMovements(entries)

	if len(movements) != 1 {
		t.Fatalf("len(movements) = %d, want 1: %+v", len(movements), movements)
	}
	m := movements[0]
	if m.FromRegion != "Florida" || m.ToRegion != "Wisconsin" {
		t.Errorf("movement = %s → %s, want Florida → Wisconsin", m.FromRegion, m.ToRegion)
	}
	if m.FromDate != "ABT 1832" || m.ToDate != "ABT 1852" {
		t.Errorf("movement dates = %q – %q, want ABT 1832 – ABT 1852", m.FromDate, m.ToDate)
	}
}

func TestComputeMovements_SkipsUndated(t *testing.T) {
	entries := []migrationEntry{
		{Date: "1850", Region: "Florida", sortKey: dateSortKey("1850")},
		{Date: "", Region: "Ohio", sortKey: dateSortKey("")},
		{Date: "1860", Region: "Florida", sortKey: dateSortKey("1860")},
	}
	if movements := computeMovements(entries); len(movements) != 0 {
		t.Errorf("undated entry caused movements: %+v", movements)
	}
}

func TestRegionForPlace(t *testing.T) {
	archive := migrationArchive()

	tests := []struct {
		placeID string
		want    string
	}{
		{"place-millbrook", "Wisconsin"},       // topmost non-country ancestor
		{"place-wisconsin", "Wisconsin"},       // region itself
		{"place-florida-territory", "Florida"}, // territory normalization
		{"place-usa", "United States"},         // all-country chain falls back to root
	}
	for _, tt := range tests {
		if got := regionForPlace(tt.placeID, archive.Places); got != tt.want {
			t.Errorf("regionForPlace(%q) = %q, want %q", tt.placeID, got, tt.want)
		}
	}
}

func TestRegionForPlace_ContinentRoot(t *testing.T) {
	// Hierarchies rooted in a continent (custom type) must resolve to the
	// level below it, not the continent itself.
	places := map[string]*glxlib.Place{
		"place-westeros":   {Name: "Westeros", Type: "continent"},
		"place-the-north":  {Name: "The North", Type: "kingdom", ParentID: "place-westeros"},
		"place-winterfell": {Name: "Winterfell", Type: "castle", ParentID: "place-the-north"},
	}
	if got := regionForPlace("place-winterfell", places); got != "The North" {
		t.Errorf("regionForPlace(winterfell) = %q, want The North", got)
	}
	if got := regionForPlace("place-westeros", places); got != "Westeros" {
		t.Errorf("regionForPlace(westeros) = %q, want Westeros (fallback)", got)
	}
}

func TestRegionForPlace_CycleSafe(t *testing.T) {
	places := map[string]*glxlib.Place{
		"place-a": {Name: "A", ParentID: "place-b"},
		"place-b": {Name: "B", ParentID: "place-a"},
	}
	if got := regionForPlace("place-a", places); got != "B" {
		t.Errorf("regionForPlace on cycle = %q, want B", got)
	}
}

func TestRegionFromFreeform(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Millbrook, Hartford Co., WI", "WI"},
		{"Florida Territory", "Florida"},
		{"Wisconsin", "Wisconsin"},
		{"", ""},
		{"  , ,  ", ""},
	}
	for _, tt := range tests {
		if got := regionFromFreeform(tt.input); got != tt.want {
			t.Errorf("regionFromFreeform(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMatchMigrationPattern(t *testing.T) {
	report := matchMigrationPattern(migrationArchive(), []string{"Florida", "Wisconsin"})

	if len(report.Matches) != 1 {
		t.Fatalf("len(Matches) = %d, want 1: %+v", len(report.Matches), report.Matches)
	}
	match := report.Matches[0]
	if match.Person != "person-jane-webb" {
		t.Errorf("matched person = %q, want person-jane-webb", match.Person)
	}
	if len(match.Stops) != 2 || match.Stops[0].Region != "Florida" || match.Stops[1].Region != "Wisconsin" {
		t.Errorf("stops = %+v, want Florida then Wisconsin", match.Stops)
	}
}

func TestMatchMigrationPattern_OrderMatters(t *testing.T) {
	report := matchMigrationPattern(migrationArchive(), []string{"Wisconsin", "Florida"})

	if len(report.Matches) != 0 {
		t.Errorf("reversed pattern matched %d persons, want 0", len(report.Matches))
	}
}

func TestMatchMigrationPattern_MatchesHierarchyNames(t *testing.T) {
	// "Millbrook" is a town inside the place path, not a region
	report := matchMigrationPattern(migrationArchive(), []string{"Florida", "Millbrook"})

	if len(report.Matches) != 1 {
		t.Errorf("hierarchy-name pattern matched %d persons, want 1", len(report.Matches))
	}
}

func TestBuildMigrationReport_FamilyRelations(t *testing.T) {
	archive := migrationArchive()
	report := buildMigrationReport(archive, "person-jane-webb", "")
	for _, rel := range findRelatedPersons("person-jane-webb", archive) {
		report.Family = append(report.Family, buildMigrationReport(archive, rel.PersonID, rel.Relation))
	}

	if len(report.Family) != 1 {
		t.Fatalf("len(Family) = %d, want 1", len(report.Family))
	}
	child := report.Family[0]
	if child.Person != "person-robert-webb" || child.Relation != "child" {
		t.Errorf("family member = %q (%s), want person-robert-webb (child)", child.Person, child.Relation)
	}
}

func TestPrintMigrationReportText(t *testing.T) {
	archive := migrationArchive()
	report := buildMigrationReport(archive, "person-jane-webb", "")

	io, out, _ := TestIOStreams()
	printMigrationReportText(io, &report)

	output := out.String()
	for _, want := range []string{
		"Migration timeline for Jane Miller (person-jane-webb)",
		"Florida Territory",
		"Millbrook, Hartford Co., Wisconsin, United States",
		"(Birth of child (Robert Webb))",
		"(1860 US Census)",
		"Movements:",
		"Florida → Wisconsin (ABT 1832 – ABT 1852)",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestResidenceMigrationEntries_Shapes(t *testing.T) {
	archive := migrationArchive()

	tests := []struct {
		name      string
		raw       any
		wantPlace string
		wantDate  string
	}{
		{"plain string place ID", "place-millbrook", "Millbrook, Hartford Co., Wisconsin, United States", ""},
		{"structured map", map[string]any{"value": "place-millbrook", "date": "1855"}, "Millbrook, Hartford Co., Wisconsin, United States", "1855"},
		{"freeform string in list", []any{"Millbrook, Hartford Co., WI"}, "Millbrook, Hartford Co., WI", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := residenceMigrationEntries(tt.raw, archive)
			if len(entries) != 1 {
				t.Fatalf("len(entries) = %d, want 1", len(entries))
			}
			if entries[0].Place != tt.wantPlace || entries[0].Date != tt.wantDate {
				t.Errorf("entry = {%q, %q}, want {%q, %q}",
					entries[0].Place, entries[0].Date, tt.wantPlace, tt.wantDate)
			}
		})
	}

	if entries := residenceMigrationEntries(nil, archive); entries != nil {
		t.Errorf("nil property produced entries: %+v", entries)
	}
	if entries := residenceMigrationEntries(map[string]any{"date": "1855"}, archive); entries != nil {
		t.Errorf("valueless map produced entries: %+v", entries)
	}
}

func TestPadPlaceColumn(t *testing.T) {
	long := strings.Repeat("Pohl-Göns, ", 8) // 88 runes, multibyte
	got := padPlaceColumn(long, migrationsPlaceColumnCap)
	if n := len([]rune(got)); n != migrationsPlaceColumnCap {
		t.Errorf("truncated rune length = %d, want %d (%q)", n, migrationsPlaceColumnCap, got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("truncated place missing ellipsis: %q", got)
	}

	if got := padPlaceColumn("Köln", 10); len([]rune(got)) != 10 {
		t.Errorf("padded rune length = %d, want 10 (%q)", len([]rune(got)), got)
	}
}

func TestPrintMigrationReportText_TruncatesLongPlaces(t *testing.T) {
	report := migrationReport{
		Person:     "person-x",
		PersonName: "X",
		Entries: []migrationEntry{
			{Date: "1850", Place: strings.Repeat("Very Long Place Name, ", 5), Label: "Census", sortKey: dateSortKey("1850")},
		},
	}

	io, out, _ := TestIOStreams()
	printMigrationReportText(io, &report)

	for line := range strings.SplitSeq(out.String(), "\n") {
		if strings.Contains(line, "Very Long") && !strings.Contains(line, "…") {
			t.Errorf("long place not truncated: %q", line)
		}
	}
}

func TestPrintMigrationReportText_UndatedSection(t *testing.T) {
	archive := migrationArchive()
	archive.Persons["person-jane-webb"].Properties["residence"] = "place-wisconsin" // undated

	report := buildMigrationReport(archive, "person-jane-webb", "")

	io, out, _ := TestIOStreams()
	printMigrationReportText(io, &report)

	output := out.String()
	if !strings.Contains(output, "Undated:") || !strings.Contains(output, "(no date)") {
		t.Errorf("undated section missing from output:\n%s", output)
	}
}

func TestPrintMigrationReportText_NoObservations(t *testing.T) {
	report := buildMigrationReport(migrationArchive(), "person-anna-lane", "")
	// Anna only appears in the 1860 census; strip it to make her observation-free
	report.Entries = nil
	report.Movements = nil

	io, out, _ := TestIOStreams()
	printMigrationReportText(io, &report)

	if !strings.Contains(out.String(), "No place observations found.") {
		t.Errorf("empty-report message missing:\n%s", out.String())
	}
}

func TestPrintMigrationPatternText_Matches(t *testing.T) {
	report := matchMigrationPattern(migrationArchive(), []string{"Florida", "Wisconsin"})

	io, out, _ := TestIOStreams()
	printMigrationPatternText(io, report)

	output := out.String()
	for _, want := range []string{
		"People with Florida → Wisconsin migration:",
		"Jane Miller (person-jane-webb) — Florida ABT 1832, Wisconsin ABT 1852",
		"1 match",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q:\n%s", want, output)
		}
	}
}

func TestPrintMigrationPatternText_NoMatches(t *testing.T) {
	report := matchMigrationPattern(migrationArchive(), []string{"Texas", "Oregon"})

	io, out, _ := TestIOStreams()
	printMigrationPatternText(io, report)

	output := out.String()
	if !strings.Contains(output, "People with Texas → Oregon migration:") ||
		!strings.Contains(output, "(no matches found)") {
		t.Errorf("unexpected no-match output:\n%s", output)
	}
}

func TestMigrationsCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		pattern string
		family  bool
		wantErr error
	}{
		{"no person no pattern", nil, "", false, ErrMigrationsPersonOrPattern},
		{"person and pattern", []string{"person-jane-webb"}, "Florida,Wisconsin", false, ErrMigrationsPersonAndPattern},
		{"family with pattern", nil, "Florida,Wisconsin", true, ErrMigrationsFamilyWithPattern},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrationsCommand(tt.args, ".", tt.pattern, tt.family, "text")
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestShowMigrations_ValidationErrors(t *testing.T) {
	io, _, _ := TestIOStreams()

	if err := showMigrations(io, ".", "x", "", false, "yaml"); !errors.Is(err, ErrMigrationsUnknownFormat) {
		t.Errorf("bad format error = %v, want ErrMigrationsUnknownFormat", err)
	}
	// A nonexistent archive path proves invocation errors surface before
	// the archive is loaded.
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if err := showMigrations(io, missing, "", "Florida", false, "text"); !errors.Is(err, ErrMigrationsPatternTooShort) {
		t.Errorf("short pattern error = %v, want ErrMigrationsPatternTooShort", err)
	}
}

// writeMigrationTestArchive serializes the test archive to a single-file
// archive on disk so showMigrations can exercise its load path end to end.
func writeMigrationTestArchive(t *testing.T) string {
	t.Helper()

	data, err := createSerializer(false, false, "").SerializeSingleFileBytes(migrationArchive())
	if err != nil {
		t.Fatalf("failed to serialize archive: %v", err)
	}
	path := filepath.Join(t.TempDir(), "migration-test.glx")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write archive: %v", err)
	}

	return path
}

func TestShowMigrations_JSONOutput(t *testing.T) {
	path := writeMigrationTestArchive(t)

	io, out, _ := TestIOStreams()
	if err := showMigrations(io, path, "person-jane-webb", "", false, "json"); err != nil {
		t.Fatalf("showMigrations: %v", err)
	}

	var report migrationReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if report.Person != "person-jane-webb" || report.PersonName != "Jane Miller" {
		t.Errorf("report header = %q/%q, want person-jane-webb/Jane Miller", report.Person, report.PersonName)
	}
	if len(report.Entries) != 4 {
		t.Errorf("len(Entries) = %d, want 4", len(report.Entries))
	}
	if len(report.Movements) != 1 || report.Movements[0].FromRegion != "Florida" {
		t.Errorf("Movements = %+v, want one Florida → Wisconsin", report.Movements)
	}
}

func TestShowMigrations_PatternJSON(t *testing.T) {
	path := writeMigrationTestArchive(t)

	io, out, _ := TestIOStreams()
	if err := showMigrations(io, path, "", "Florida,Wisconsin", false, "json"); err != nil {
		t.Fatalf("showMigrations: %v", err)
	}

	var report migrationPatternReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(report.Matches) != 1 || report.Matches[0].Person != "person-jane-webb" {
		t.Errorf("Matches = %+v, want person-jane-webb only", report.Matches)
	}
}

func TestShowMigrations_FamilyText(t *testing.T) {
	path := writeMigrationTestArchive(t)

	io, out, _ := TestIOStreams()
	if err := showMigrations(io, path, "person-jane-webb", "", true, "text"); err != nil {
		t.Fatalf("showMigrations: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Migration timeline for Robert Webb (person-robert-webb) — child:") {
		t.Errorf("family timeline missing from output:\n%s", output)
	}
}
