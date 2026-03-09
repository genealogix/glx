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
	"strings"
	"testing"
	"time"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestGenerateWikiTreeBio_BasicPerson(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Smith",
						},
					},
					"gender":  "male",
					"born_on": "1850-01-15",
					"born_at": "place-leeds",
				},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-death-john": {
				Type:    "death",
				Date:    "1920-03-10",
				PlaceID: "place-london",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-leeds":  {Name: "Leeds, Yorkshire, England"},
			"place-london": {Name: "London, England"},
		},
		Assertions:    map[string]*glxlib.Assertion{},
		Citations:     map[string]*glxlib.Citation{},
		Sources:       map[string]*glxlib.Source{},
		Relationships: map[string]*glxlib.Relationship{},
	}

	bio := generateWikiTreeBio("person-john", archive.Persons["person-john"], archive)

	// Check required sections
	if !strings.Contains(bio, "== Biography ==") {
		t.Error("missing Biography section")
	}
	if !strings.Contains(bio, "== Sources ==") {
		t.Error("missing Sources section")
	}
	if !strings.Contains(bio, "<references />") {
		t.Error("missing <references /> tag")
	}

	// Check birth info
	if !strings.Contains(bio, "'''John Smith'''") {
		t.Error("missing bold name in opening sentence")
	}
	if !strings.Contains(bio, "1850-01-15") {
		t.Error("missing birth date")
	}
	if !strings.Contains(bio, "Leeds, Yorkshire, England") {
		t.Error("missing birth place")
	}

	// Check death section
	if !strings.Contains(bio, "=== Death and Burial ===") {
		t.Error("missing Death and Burial subsection")
	}
	if !strings.Contains(bio, "London, England") {
		t.Error("missing death place")
	}
}

func TestGenerateWikiTreeBio_WithCitations(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane": {
				Properties: map[string]any{
					"name":    "Jane Doe",
					"gender":  "female",
					"born_on": "ABT 1832",
				},
			},
		},
		Events:        map[string]*glxlib.Event{},
		Places:        map[string]*glxlib.Place{},
		Relationships: map[string]*glxlib.Relationship{},
		Citations: map[string]*glxlib.Citation{
			"cit-census": {
				SourceID: "src-census",
				Properties: map[string]any{
					"citation_text": "1860 U.S. Census, Wisconsin, Sauk County",
				},
			},
		},
		Sources: map[string]*glxlib.Source{
			"src-census": {Title: "1860 United States Federal Census"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assert-birth": {
				Subject:    glxlib.EntityRef{Person: "person-jane"},
				Property:   "born_on",
				Value:      "ABT 1832",
				Citations:  []string{"cit-census"},
				Confidence: "medium",
				Notes:      "Birth year estimated from census age",
			},
		},
	}

	bio := generateWikiTreeBio("person-jane", archive.Persons["person-jane"], archive)

	// Check inline citation
	if !strings.Contains(bio, `<ref name="cit-census">`) {
		t.Error("missing inline citation ref tag")
	}
	if !strings.Contains(bio, "1860 U.S. Census, Wisconsin, Sauk County") {
		t.Error("missing citation text in ref")
	}

	// Check approximate date rendering
	if !strings.Contains(bio, "about 1832") {
		t.Error("expected 'about 1832' for ABT date")
	}

	// Check research notes (medium confidence assertion with notes)
	if !strings.Contains(bio, "== Research Notes ==") {
		t.Error("missing Research Notes section for medium-confidence assertion")
	}
}

func TestGenerateWikiTreeBio_Children(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-parent": {
				Properties: map[string]any{
					"name":   "Mary Lane",
					"gender": "female",
				},
			},
			"person-child1": {
				Properties: map[string]any{
					"name":    "Alice Lane",
					"born_on": "1855",
				},
			},
			"person-child2": {
				Properties: map[string]any{
					"name":    "Bob Lane",
					"born_on": "1858",
				},
			},
		},
		Events:     map[string]*glxlib.Event{},
		Places:     map[string]*glxlib.Place{},
		Citations:  map[string]*glxlib.Citation{},
		Sources:    map[string]*glxlib.Source{},
		Assertions: map[string]*glxlib.Assertion{},
		Relationships: map[string]*glxlib.Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child1", Role: "child"},
				},
			},
			"rel-2": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-parent", Role: "parent"},
					{Person: "person-child2", Role: "child"},
				},
			},
		},
	}

	bio := generateWikiTreeBio("person-parent", archive.Persons["person-parent"], archive)

	if !strings.Contains(bio, "=== Children ===") {
		t.Error("missing Children subsection")
	}
	if !strings.Contains(bio, "Alice Lane (b. 1855)") {
		t.Error("missing child Alice with birth year")
	}
	if !strings.Contains(bio, "Bob Lane (b. 1858)") {
		t.Error("missing child Bob with birth year")
	}
}

func TestNarrativeDateWT(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"1850-01-15", "on 1850-01-15"},
		{"ABT 1832", "about 1832"},
		{"BEF 1920", "before 1920"},
		{"AFT 1860", "after 1860"},
		{"BET 1861 AND 1865", "between 1861 and 1865"},
		{"", ""},
	}

	for _, tt := range tests {
		result := narrativeDateWT(tt.input)
		if result != tt.expected {
			t.Errorf("narrativeDateWT(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractPersonNameShort(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
						"fields": map[string]any{
							"given":   "John",
							"surname": "Smith",
						},
					},
				},
			},
			"person-simple": {
				Properties: map[string]any{
					"name": "Mary Green",
				},
			},
		},
	}

	if got := extractPersonNameShort("person-john", archive); got != "John" {
		t.Errorf("expected 'John', got %q", got)
	}
	if got := extractPersonNameShort("person-simple", archive); got != "Mary" {
		t.Errorf("expected 'Mary', got %q", got)
	}
}

func TestRefTracker(t *testing.T) {
	archive := &glxlib.GLXFile{
		Citations: map[string]*glxlib.Citation{
			"cit-1": {
				SourceID: "src-1",
				Properties: map[string]any{
					"citation_text": "Full citation here",
				},
			},
		},
		Sources: map[string]*glxlib.Source{
			"src-1": {Title: "Test Source"},
		},
	}

	refs := &refTracker{citations: make(map[string]int)}

	// First use should emit full ref
	first := refs.ref("cit-1", archive)
	if !strings.Contains(first, `<ref name="cit-1">Full citation here</ref>`) {
		t.Errorf("first ref should contain full citation, got: %s", first)
	}

	// Second use should emit short ref
	second := refs.ref("cit-1", archive)
	if second != `<ref name="cit-1" />` {
		t.Errorf("second ref should be short form, got: %s", second)
	}
}

func TestRecordAndLoadWikiTreeTracking(t *testing.T) {
	tmpDir := t.TempDir()

	// Record a generation
	err := recordWikiTreeGeneration(tmpDir, "person-john")
	if err != nil {
		t.Fatalf("unexpected error recording generation: %v", err)
	}

	// Verify tracking file was created
	trackingPath := filepath.Join(tmpDir, wikiTreeTrackingFile)
	if _, err := os.Stat(trackingPath); os.IsNotExist(err) {
		t.Fatal("tracking file was not created")
	}

	// Load and verify
	tracking := loadWikiTreeTracking(tmpDir)
	ts, ok := tracking["person-john"]
	if !ok {
		t.Fatal("person-john not found in tracking")
	}

	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		t.Fatalf("invalid timestamp format: %v", err)
	}

	// Should be within the last minute
	if time.Since(parsed) > time.Minute {
		t.Errorf("timestamp too old: %v", parsed)
	}
}

func TestLoadWikiTreeTracking_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	tracking := loadWikiTreeTracking(tmpDir)

	if len(tracking) != 0 {
		t.Errorf("expected empty tracking for missing file, got %d entries", len(tracking))
	}
}

func TestFindStaleFiles(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {},
		},
		Events: map[string]*glxlib.Event{
			"event-census": {
				Type: "census",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "subject"},
				},
			},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assert-birth": {
				Subject: glxlib.EntityRef{Person: "person-john"},
			},
		},
	}

	// Build the person entity index
	relatedIDs := buildPersonEntityIndex(archive)

	genTime := time.Now().Add(-time.Hour) // Generated 1 hour ago

	// Files modified after generation
	fileMtimes := map[string]time.Time{
		"persons/person-john.glx":     time.Now(), // Modified after gen
		"events/event-census.glx":     time.Now(), // Modified after gen
		"assertions/assert-birth.glx": time.Now(), // Modified after gen
		"events/event-unrelated.glx":  time.Now(), // Unrelated
		"persons/person-other.glx":    time.Now(), // Unrelated
	}

	stale := findStaleFiles("person-john", relatedIDs["person-john"], fileMtimes, genTime)

	if len(stale) < 2 {
		t.Errorf("expected at least 2 stale files (person + event or assertion), got %d: %v", len(stale), stale)
	}

	// Should not include unrelated files
	for _, f := range stale {
		if f == "events/event-unrelated.glx" || f == "persons/person-other.glx" {
			t.Errorf("should not include unrelated file: %s", f)
		}
	}
}
