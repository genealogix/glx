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
	"fmt"
	"math/rand"
	"time"

	"github.com/brianvoe/gofakeit/v6"
)

// GenerateTestData creates a complete GLXFile structure with plausible test data
// for a specified number of people.
func GenerateTestData(numPeople int) (*GLXFile, error) {
	gofakeit.Seed(0)

	glxFile := &GLXFile{
		Persons:       make(map[string]*Person),
		Relationships: make(map[string]*Relationship),
		Events:        make(map[string]*Event),
		Places:        make(map[string]*Place),
		Sources:       make(map[string]*Source),
		Citations:     make(map[string]*Citation),
		Repositories:  make(map[string]*Repository),
		Assertions:    make(map[string]*Assertion),
		Media:         make(map[string]*Media),
	}

	// Generate a repository for the sources
	repoID := "repo-" + gofakeit.UUID()
	glxFile.Repositories[repoID] = &Repository{
		Name:    gofakeit.Company(),
		Address: gofakeit.Address().Address,
		City:    gofakeit.Address().City,
		Country: gofakeit.Address().Country,
	}

	// Generate people
	personIDs := make([]string, 0, numPeople)
	for i := range numPeople {
		personID := "person-" + gofakeit.UUID()
		personIDs = append(personIDs, personID)
		living := gofakeit.Bool()

		glxFile.Persons[personID] = &Person{
			Properties: map[string]any{
				"primary_name": gofakeit.Name(),
				"gender":       gofakeit.Gender(),
				"living":       living,
			},
		}

		// Generate a birth event for each person
		birthDate := gofakeit.DateRange(time.Now().AddDate(-80, 0, 0), time.Now().AddDate(-20, 0, 0))
		eventID := "event-birth-" + personID
		placeID := generatePlace(glxFile)

		participants := []Participant{
			{Person: personID, Role: "principal"},
		}

		// Add parents to birth event if we have other people
		if i > 1 {
			parent1 := personIDs[rand.Intn(i)]
			parent2 := personIDs[rand.Intn(i)]
			if parent1 != parent2 {
				participants = append(participants, Participant{Person: parent1, Role: "parent"})
				participants = append(participants, Participant{Person: parent2, Role: "parent"})
			}
		}

		glxFile.Events[eventID] = &Event{
			Type:         "birth",
			PlaceID:      placeID,
			Date:         DateString(birthDate.Format("2006-01-02")),
			Participants: participants,
		}

		// Generate a source, citation, and assertion for the birth
		generateEvidenceChain(glxFile, eventID, "birth", repoID)
	}

	// Generate some relationships
	if numPeople > 1 {
		relationshipTypes := []string{"marriage", "sibling", "partner", "friend"}
		for range numPeople / 2 {
			p1 := personIDs[rand.Intn(len(personIDs))]
			p2 := personIDs[rand.Intn(len(personIDs))]
			if p1 == p2 {
				continue // Don't create a relationship with oneself
			}

			relID := "rel-" + gofakeit.UUID()
			relType := relationshipTypes[rand.Intn(len(relationshipTypes))]

			var roles []string
			switch relType {
			case "marriage":
				roles = []string{"spouse", "spouse"}
			case "sibling":
				roles = []string{"sibling", "sibling"}
			case "partner":
				roles = []string{"partner", "partner"}
			default:
				roles = []string{"friend", "friend"}
			}

			glxFile.Relationships[relID] = &Relationship{
				Type: relType,
				Participants: []Participant{
					{Person: p1, Role: roles[0]},
					{Person: p2, Role: roles[1]},
				},
			}
		}
	}

	return glxFile, nil
}

// generatePlace creates a new place and adds it to the GLXFile, returning its ID.
func generatePlace(glxFile *GLXFile) string {
	placeID := "place-" + gofakeit.UUID()
	glxFile.Places[placeID] = &Place{
		Name: fmt.Sprintf("%s, %s", gofakeit.City(), gofakeit.Country()),
		Type: "city",
	}

	return placeID
}

// generateEvidenceChain creates a source, citation, and assertion for a given event.
func generateEvidenceChain(glxFile *GLXFile, subjectID, propertyName, repoID string) {
	// Source
	sourceID := "source-" + gofakeit.UUID()
	glxFile.Sources[sourceID] = &Source{
		Title:        gofakeit.Word() + " Certificate",
		Type:         "vital_record",
		RepositoryID: repoID,
	}

	// Citation
	citationID := "citation-" + gofakeit.UUID()
	glxFile.Citations[citationID] = &Citation{
		SourceID: sourceID,
		Properties: map[string]any{
			"locator": "Record ID: " + gofakeit.UUID(),
		},
	}

	// Assertion
	assertionID := "assertion-" + gofakeit.UUID()
	glxFile.Assertions[assertionID] = &Assertion{
		Subject:   EntityRef{Event: subjectID},
		Property:  propertyName,
		Citations: []string{citationID},
	}
}
