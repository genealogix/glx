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
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// secureIntn returns a cryptographically random int in [0, n).
func secureIntn(n int) int {
	r, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(r.Int64())
}

// randomSexValue picks a random canonical sex_types vocabulary value.
// gofakeit.Gender() returns human labels ("Male"/"Female") which are
// out-of-vocab; this helper returns the lowercase keys (`male`/`female`)
// so generated archives pass validation cleanly.
var datagenSexValues = []string{SexMale, SexFemale}

func randomSexValue() string {
	return datagenSexValues[secureIntn(len(datagenSexValues))]
}

// GenerateTestData creates a complete GLXFile structure with plausible test data
// for a specified number of people.
func GenerateTestData(numPeople int) (*GLXFile, error) {
	gofakeit.GlobalFaker = gofakeit.New(0)

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
		ResearchLogs:  make(map[string]*ResearchLog),
		Studies:       make(map[string]*Study),
	}

	// Generate a repository for the sources
	repoID := EntityIDPrefixRepository + gofakeit.UUID()
	glxFile.Repositories[repoID] = &Repository{
		Name:    gofakeit.Company(),
		Address: gofakeit.Address().Address,
		City:    gofakeit.Address().City,
		Country: gofakeit.Address().Country,
	}

	// Generate people
	personIDs := make([]string, 0, numPeople)
	for i := range numPeople {
		personID := EntityIDPrefixPerson + gofakeit.UUID()
		personIDs = append(personIDs, personID)
		living := gofakeit.Bool()

		glxFile.Persons[personID] = &Person{
			Properties: map[string]any{
				"primary_name":    gofakeit.Name(),
				PersonPropertySex: randomSexValue(),
				"living":          living,
			},
		}

		// Generate a birth event for each person
		birthDate := gofakeit.DateRange(time.Now().AddDate(-80, 0, 0), time.Now().AddDate(-20, 0, 0))
		eventID := EntityIDPrefixEvent + "birth-" + personID
		placeID := generatePlace(glxFile)

		participants := []Participant{
			{Person: personID, Role: "principal"},
		}

		// Add parents to birth event if we have other people
		if i > 1 {
			parent1 := personIDs[secureIntn(i)]
			parent2 := personIDs[secureIntn(i)]
			if parent1 != parent2 {
				participants = append(participants,
					Participant{Person: parent1, Role: "parent"},
					Participant{Person: parent2, Role: "parent"},
				)
			}
		}

		glxFile.Events[eventID] = &Event{
			Type:         "birth",
			PlaceID:      placeID,
			Date:         DateString(birthDate.Format("2006-01-02")),
			Participants: participants,
		}

		// Generate a source, citation, and assertion for the birth
		generateEvidenceChain(glxFile, eventID, repoID)
	}

	// Generate some relationships
	if numPeople > 1 {
		relationshipTypes := []string{"marriage", "sibling", "partner", "friend"}
		for range numPeople / 2 {
			p1 := personIDs[secureIntn(len(personIDs))]
			p2 := personIDs[secureIntn(len(personIDs))]
			if p1 == p2 {
				continue // Don't create a relationship with oneself
			}

			relID := EntityIDPrefixRelationship + gofakeit.UUID()
			relType := relationshipTypes[secureIntn(len(relationshipTypes))]

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

	generateStudy(glxFile)
	generateResearchLog(glxFile, repoID)

	return glxFile, nil
}

// generateStudy adds one umbrella Study scoping the generated archive — a
// family-reconstruction covering every place and source generated above.
// Demonstrates the Study entity shape; tooling can use this to exercise
// Study-aware features against test data.
func generateStudy(glxFile *GLXFile) {
	if len(glxFile.Persons) == 0 {
		return
	}

	studyID := "study-" + gofakeit.UUID()

	placeRefs := make([]string, 0, len(glxFile.Places))
	for id := range glxFile.Places {
		placeRefs = append(placeRefs, id)
	}

	sourceRefs := make([]string, 0, len(glxFile.Sources))
	for id := range glxFile.Sources {
		sourceRefs = append(sourceRefs, id)
	}

	glxFile.Studies[studyID] = &Study{
		Title:     gofakeit.LastName() + " family reconstruction",
		Type:      "family_reconstruction",
		Status:    "active",
		DateRange: DateString("FROM 1900 TO 2000"),
		Places:    placeRefs,
		Sources:   sourceRefs,
	}
}

// generateResearchLog adds one ResearchLog with a Search entry paired with the
// first generated citation, demonstrating the "documented search" workflow.
func generateResearchLog(glxFile *GLXFile, repoID string) {
	var firstCitationID, firstSourceID string
	for id, c := range glxFile.Citations {
		firstCitationID = id
		firstSourceID = c.SourceID

		break
	}

	if firstCitationID == "" {
		return
	}

	logID := "research-log-" + gofakeit.UUID()
	glxFile.ResearchLogs[logID] = &ResearchLog{
		Title:      "Birth record search for generated test data",
		Objective:  "Locate the birth record cited in the generated evidence chain",
		Status:     "complete",
		Researcher: gofakeit.Name(),
		Searches: []Search{
			{
				RepositoryID: repoID,
				SourceID:     firstSourceID,
				Query:        "birth records",
				Result:       "found",
				CitationID:   firstCitationID,
			},
		},
		Conclusions: "Citation produced and attached to the corresponding birth event assertion.",
	}
}

// generatePlace creates a new place and adds it to the GLXFile, returning its ID.
func generatePlace(glxFile *GLXFile) string {
	placeID := EntityIDPrefixPlace + gofakeit.UUID()
	glxFile.Places[placeID] = &Place{
		Name: fmt.Sprintf("%s, %s", gofakeit.City(), gofakeit.Country()),
		Type: "city",
	}

	return placeID
}

// generateEvidenceChain creates a source, citation, and an existential
// assertion attesting the given event (no property/value pair — the assertion
// records that the event happened and is backed by the citation).
func generateEvidenceChain(glxFile *GLXFile, subjectID, repoID string) {
	// Source
	sourceID := EntityIDPrefixSource + gofakeit.UUID()
	glxFile.Sources[sourceID] = &Source{
		Title:        gofakeit.Word() + " Certificate",
		Type:         "vital_record",
		RepositoryID: repoID,
	}

	// Citation
	citationID := EntityIDPrefixCitation + gofakeit.UUID()
	glxFile.Citations[citationID] = &Citation{
		SourceID: sourceID,
		Properties: map[string]any{
			"locator": "Record ID: " + gofakeit.UUID(),
		},
	}

	// Existential assertion — attests that the subject event occurred.
	// (A property/value pair would also be schema-valid, but existential
	// matches the semantics of "the birth happened, here's the source".)
	assertionID := EntityIDPrefixAssertion + gofakeit.UUID()
	glxFile.Assertions[assertionID] = &Assertion{
		Subject:   EntityRef{Event: subjectID},
		Citations: []string{citationID},
	}
}
