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
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

func TestFindPersonByID(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
					},
				},
			},
		},
	}

	id, person, err := findPerson(archive, "person-john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "person-john" {
		t.Errorf("expected id person-john, got %s", id)
	}
	if person == nil {
		t.Fatal("expected person, got nil")
	}
}

func TestFindPersonByName(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
					},
				},
			},
			"person-jane": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "Jane Miller",
					},
				},
			},
		},
	}

	id, _, err := findPerson(archive, "Jane")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "person-jane" {
		t.Errorf("expected id person-jane, got %s", id)
	}
}

func TestFindPersonNotFound(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": "John Smith",
				},
			},
		},
	}

	_, _, err := findPerson(archive, "Nobody")
	if err == nil {
		t.Fatal("expected error for missing person")
	}
}

func TestFindPersonAmbiguous(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane-webb": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "Jane Webb",
					},
				},
			},
			"person-jane-miller": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "Jane Miller",
					},
				},
			},
		},
	}

	_, _, err := findPerson(archive, "Jane")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
}

func TestCollectVitals(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-john": {
				Properties: map[string]any{
					"name": map[string]any{
						"value": "John Smith",
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
			"event-marriage-john": {
				Type: "marriage",
				Date:  "1875-05-10",
				Participants: []glxlib.Participant{
					{Person: "person-john", Role: "groom"},
				},
			},
		},
		Places: map[string]*glxlib.Place{
			"place-leeds":  {Name: "Leeds, Yorkshire, England"},
			"place-london": {Name: "London, England"},
		},
	}

	vitals := collectVitals("person-john", archive.Persons["person-john"], archive)

	// Check we have at least the standard 6 vitals + the marriage event
	if len(vitals) < 7 {
		t.Errorf("expected at least 7 vitals, got %d", len(vitals))
	}

	// Verify Name
	if vitals[0].Label != "Name" || vitals[0].Value != "John Smith" {
		t.Errorf("unexpected Name: %+v", vitals[0])
	}

	// Verify Sex
	if vitals[1].Label != "Sex" || vitals[1].Value != "male" {
		t.Errorf("unexpected Sex: %+v", vitals[1])
	}

	// Verify Birth
	if vitals[2].Label != "Birth" || vitals[2].Value != "1850-01-15, Leeds, Yorkshire, England" {
		t.Errorf("unexpected Birth: %+v", vitals[2])
	}

	// Verify Death
	if vitals[4].Label != "Death" || vitals[4].Value != "1920-03-10, London, England" {
		t.Errorf("unexpected Death: %+v", vitals[4])
	}
}

