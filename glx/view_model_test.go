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
	"unicode/utf8"

	glxlib "github.com/genealogix/glx/go-glx"
)

// buildModelTestArchive constructs an in-memory archive exercising names,
// vital events, place hierarchy, family relationships, evidence, and media.
func buildModelTestArchive() *glxlib.GLXFile {
	lat, lng := 42.36, -71.06

	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john-smith": {
				Properties: map[string]any{"name": "John Smith", "sex": "male"},
				Notes:      glxlib.NoteList{"A note about John."},
			},
			"person-jane-doe": {
				Properties: map[string]any{
					"name": []any{
						map[string]any{"value": "Jane Doe"},
						map[string]any{"value": "Jane Smith"},
					},
					"sex": "female",
				},
			},
			"person-mary-smith": {
				Properties: map[string]any{"name": "Mary Smith"},
			},
			"person-living-soul": {
				Properties: map[string]any{"name": "Living", "living": true},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-john-birth": {
				Type: "birth", Date: "1850-01-15", PlaceID: "place-boston",
				Participants: []glxlib.Participant{{Person: "person-john-smith", Role: "subject"}},
			},
			"event-john-death": {
				Type: "death", Date: "1920-03-10", PlaceID: "place-boston",
				Participants: []glxlib.Participant{{Person: "person-john-smith", Role: "subject"}},
			},
			"event-jane-birth": {
				Type: "birth", Date: "1855",
				Participants: []glxlib.Participant{{Person: "person-jane-doe", Role: "subject"}},
			},
			"event-marriage": {
				Type: "marriage", Date: "1875", PlaceID: "place-boston",
				Participants: []glxlib.Participant{
					{Person: "person-john-smith", Role: "spouse"},
					{Person: "person-jane-doe", Role: "spouse"},
				},
			},
			"event-mary-birth": {
				Type: "birth", Date: "1880",
				Participants: []glxlib.Participant{{Person: "person-mary-smith", Role: "subject"}},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-boston": {
				Name: "Boston", ParentID: "place-mass", Type: "city",
				Latitude: &lat, Longitude: &lng,
			},
			"place-mass": {Name: "Massachusetts", ParentID: "place-us", Type: "state"},
			"place-us":   {Name: "United States", Type: "country"},
		},
		Relationships: map[string]*glxlib.Relationship{
			"rel-marriage": {
				Type:       "marriage",
				StartEvent: "event-marriage",
				Participants: []glxlib.Participant{
					{Person: "person-john-smith", Role: "spouse"},
					{Person: "person-jane-doe", Role: "spouse"},
				},
			},
			"rel-john-mary": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-john-smith", Role: "parent"},
					{Person: "person-mary-smith", Role: "child"},
				},
			},
			"rel-jane-mary": {
				Type: "parent_child",
				Participants: []glxlib.Participant{
					{Person: "person-jane-doe", Role: "parent"},
					{Person: "person-mary-smith", Role: "child"},
				},
			},
		},
		Sources: map[string]*glxlib.Source{
			"source-census": {
				Title: "1880 US Census", Type: "census", Authors: []string{"US Government"},
				Properties: map[string]any{"url": "https://example.org/census"},
			},
		},
		Citations: map[string]*glxlib.Citation{
			"citation-census": {
				SourceID: "source-census",
				Properties: map[string]any{
					"locator":          "Sheet 12",
					"text_from_source": "John Smith, head of household",
				},
			},
		},
		Assertions: map[string]*glxlib.Assertion{
			"assertion-john": {
				Subject:    glxlib.EntityRef{Person: "person-john-smith"},
				Property:   "occupation",
				Value:      "farmer",
				Citations:  []string{"citation-census"},
				Confidence: "high",
				Status:     "proven",
				Media:      []string{"media-portrait"},
			},
		},
		Media: map[string]*glxlib.Media{
			"media-portrait": {
				URI: "portrait.jpg", MimeType: "image/jpeg", Title: "John's portrait",
				Properties: map[string]any{"description": "Studio portrait, c. 1900"},
			},
		},
	}
}

func TestBuildSiteModel_Stats(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})

	if model.Title != "GLX Family Archive" {
		t.Errorf("default title = %q, want %q", model.Title, "GLX Family Archive")
	}
	if model.Stats.Persons != 4 {
		t.Errorf("Stats.Persons = %d, want 4", model.Stats.Persons)
	}
	if model.Stats.Sources != 1 {
		t.Errorf("Stats.Sources = %d, want 1", model.Stats.Sources)
	}
	if model.Stats.Places != 3 {
		t.Errorf("Stats.Places = %d, want 3", model.Stats.Places)
	}
}

func TestBuildSiteModel_PersonOrderAndNames(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{Title: "Test"})

	// Surname-first ordering by sort key: "doe, jane" < "living" <
	// "smith, john" < "smith, mary".
	wantOrder := []string{"person-jane-doe", "person-living-soul", "person-john-smith", "person-mary-smith"}
	if len(model.Persons) != len(wantOrder) {
		t.Fatalf("got %d persons, want %d", len(model.Persons), len(wantOrder))
	}
	for i, want := range wantOrder {
		if model.Persons[i].ID != want {
			t.Errorf("person[%d].ID = %q, want %q", i, model.Persons[i].ID, want)
		}
	}

	jane := findPage(t, model, "person-jane-doe")
	if jane.Name != "Jane Doe" {
		t.Errorf("Jane primary name = %q, want %q", jane.Name, "Jane Doe")
	}
	if len(jane.AltNames) != 1 || jane.AltNames[0] != "Jane Smith" {
		t.Errorf("Jane AltNames = %v, want [Jane Smith]", jane.AltNames)
	}
}

func TestBuildPersonPage_VitalsAndFamily(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})
	john := findPage(t, model, "person-john-smith")

	if john.LifeSpan != "1850–1920" {
		t.Errorf("LifeSpan = %q, want %q", john.LifeSpan, "1850–1920")
	}
	if john.Sex != "male" {
		t.Errorf("Sex = %q, want male", john.Sex)
	}
	if john.BirthPlace != "Boston, Massachusetts, United States" {
		t.Errorf("BirthPlace = %q", john.BirthPlace)
	}
	if !containsLink(john.Spouses, "person-jane-doe") {
		t.Errorf("expected Jane as spouse, got %+v", john.Spouses)
	}
	if !containsLink(john.Children, "person-mary-smith") {
		t.Errorf("expected Mary as child, got %+v", john.Children)
	}
	if len(john.Timeline) == 0 {
		t.Error("expected a non-empty timeline")
	}
	if len(john.Notes) != 1 || john.Notes[0] != "A note about John." {
		t.Errorf("Notes = %v", john.Notes)
	}

	// Mary has two parents and no siblings.
	mary := findPage(t, model, "person-mary-smith")
	if len(mary.Parents) != 2 {
		t.Errorf("Mary parents = %d, want 2", len(mary.Parents))
	}
	if len(mary.Siblings) != 0 {
		t.Errorf("Mary siblings = %d, want 0", len(mary.Siblings))
	}
}

func TestBuildPersonPage_Sources(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})
	john := findPage(t, model, "person-john-smith")

	if len(john.Sources) != 1 {
		t.Fatalf("Sources = %d, want 1", len(john.Sources))
	}
	src := john.Sources[0]
	if src.Title != "1880 US Census" {
		t.Errorf("source title = %q", src.Title)
	}
	if !strings.Contains(src.Detail, "Sheet 12") || !strings.Contains(src.Detail, "John Smith, head") {
		t.Errorf("source detail missing locator/quote: %q", src.Detail)
	}
	if src.Confidence != "high" || src.Status != "proven" {
		t.Errorf("confidence/status = %q/%q", src.Confidence, src.Status)
	}
}

func TestBuildPersonPage_Media(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})
	john := findPage(t, model, "person-john-smith")

	if len(john.Media) != 1 {
		t.Fatalf("Media = %d, want 1", len(john.Media))
	}
	m := john.Media[0]
	if m.Kind != mediaKindImage {
		t.Errorf("portrait Kind = %q, want %q", m.Kind, mediaKindImage)
	}
	if m.Caption != "Studio portrait, c. 1900" {
		t.Errorf("caption = %q", m.Caption)
	}
}

func TestBuildSearchIndex(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), viewModelOptions{})

	var persons, sources, places int
	for _, e := range model.Search {
		switch e.Kind {
		case "person":
			persons++
		case "source":
			sources++
		case "place":
			places++
		}
	}
	if persons != 4 || sources != 1 || places != 3 {
		t.Errorf("search counts person/source/place = %d/%d/%d, want 4/1/3", persons, sources, places)
	}

	for _, e := range model.Search {
		if e.Kind == "person" && e.Name == "John Smith" {
			if e.URL != "persons/person-john-smith.html" {
				t.Errorf("John search URL = %q", e.URL)
			}
		}
	}
}

func TestPlaceFullName_CycleSafe(t *testing.T) {
	archive := &glxlib.GLXFile{
		Places: map[string]*glxlib.Place{
			"place-a": {Name: "A", ParentID: "place-b"},
			"place-b": {Name: "B", ParentID: "place-a"}, // cycle
		},
	}
	got := placeFullName("place-a", archive)
	if got != "A, B" {
		t.Errorf("placeFullName with cycle = %q, want %q", got, "A, B")
	}
}

func TestLifeSpan(t *testing.T) {
	cases := []struct {
		name         string
		birth, death string
		want         string
	}{
		{"both", "1850", "1920", "1850–1920"},
		{"birth only", "1850-06-01", "", "b. 1850"},
		{"death only", "", "1920", "d. 1920"},
		{"neither", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var b, d *glxlib.Event
			if tc.birth != "" {
				b = &glxlib.Event{Date: glxlib.DateString(tc.birth)}
			}
			if tc.death != "" {
				d = &glxlib.Event{Date: glxlib.DateString(tc.death)}
			}
			if got := lifeSpan(b, d); got != tc.want {
				t.Errorf("lifeSpan(%q,%q) = %q, want %q", tc.birth, tc.death, got, tc.want)
			}
		})
	}
}

func TestEventYear(t *testing.T) {
	cases := map[string]string{
		"1850":       "1850",
		"1850-03-22": "1850",
		"ABT 1850":   "1850",
		"":           "",
		"unknown":    "",
	}
	for in, want := range cases {
		if got := eventYear(in); got != want {
			t.Errorf("eventYear(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPersonSortName(t *testing.T) {
	cases := map[string]string{
		"John Smith":    "smith, john",
		"Jane Mary Doe": "doe, jane mary",
		"Madonna":       "madonna",
		"":              "￿",
	}
	for in, want := range cases {
		if got := personSortName(in); got != want {
			t.Errorf("personSortName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsLivingProperty(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"absent", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &glxlib.Person{Properties: map[string]any{}}
			if tc.val != nil {
				p.Properties[glxlib.PersonPropertyLiving] = tc.val
			}
			if got := isLivingProperty(p); got != tc.want {
				t.Errorf("isLivingProperty(%v) = %v, want %v", tc.val, got, tc.want)
			}
		})
	}
}

func TestVitalEvents_IgnoresWitnessRole(t *testing.T) {
	// A person who only witnessed someone else's birth must NOT inherit it.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-subject": {Properties: map[string]any{"name": "Subject"}},
			"person-witness": {Properties: map[string]any{"name": "Witness"}},
		},
		Events: map[string]*glxlib.Event{
			"event-birth": {
				Type: "birth", Date: "1850", PlaceID: "place-town",
				Participants: []glxlib.Participant{
					{Person: "person-subject", Role: "principal"},
					{Person: "person-witness", Role: "witness"},
				},
			},
		},
		Places: map[string]*glxlib.Place{"place-town": {Name: "Town"}},
	}
	model := buildSiteModel(archive, viewModelOptions{})

	if subj := findPage(t, model, "person-subject"); subj.LifeSpan != "b. 1850" {
		t.Errorf("subject LifeSpan = %q, want %q", subj.LifeSpan, "b. 1850")
	}
	wit := findPage(t, model, "person-witness")
	if wit.LifeSpan != "" || wit.BirthDate != "" || wit.BirthPlace != "" {
		t.Errorf("witness wrongly inherited birth: LifeSpan=%q BirthDate=%q BirthPlace=%q",
			wit.LifeSpan, wit.BirthDate, wit.BirthPlace)
	}
}

func TestHumanizeToken_Multibyte(t *testing.T) {
	got := humanizeToken("état_civil")
	if !utf8.ValidString(got) {
		t.Errorf("humanizeToken produced invalid UTF-8: %q", got)
	}
	if got != "État civil" {
		t.Errorf("humanizeToken(\"état_civil\") = %q, want %q", got, "État civil")
	}
}

func TestHumanizeToken(t *testing.T) {
	cases := map[string]string{
		"vital_record": "Vital record",
		"census":       "Census",
		"":             "",
	}
	for in, want := range cases {
		if got := humanizeToken(in); got != want {
			t.Errorf("humanizeToken(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOpenStreetMapURL(t *testing.T) {
	got := openStreetMapURL(42.36, -71.06)
	if !strings.Contains(got, "mlat=42.36") || !strings.Contains(got, "mlon=-71.06") {
		t.Errorf("openStreetMapURL = %q", got)
	}
}

// findPage returns the person page with the given ID, failing the test if absent.
func findPage(t *testing.T, model *siteModel, id string) *personPage {
	t.Helper()
	for _, p := range model.Persons {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("person page %q not found", id)

	return nil
}

// containsLink reports whether a person link slice contains the given ID.
func containsLink(links []personLink, id string) bool {
	for _, l := range links {
		if l.ID == id {
			return true
		}
	}

	return false
}
