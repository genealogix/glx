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
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// readBackArchive returns the loaded archive for assertion. Helper used by
// every TestAdd_* test so the table-driven assertions can read from disk.
func readBackArchive(t *testing.T, dir string) *glxlib.GLXFile {
	t.Helper()
	archive, _, err := LoadArchiveWithOptions(dir, false)
	if err != nil {
		t.Fatalf("LoadArchiveWithOptions: %v", err)
	}

	return archive
}

// trailingLine returns the last non-empty line written to out — the
// invariant from finalizeAdd is that the new entity ID is the final line of
// stdout so $(glx add …) substitution works.
func trailingLine(s string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) == 0 {
		return ""
	}

	return lines[len(lines)-1]
}

// =============================================================================
// add person
// =============================================================================

func TestAdd_PersonHappyPath(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()

	err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Johann Peter",
		Surname:          "Jungk",
		Sex:              "male",
	})
	if err != nil {
		t.Fatalf("addPerson: %v", err)
	}

	want := "person-johann-peter-jungk"
	if got := trailingLine(out.String()); got != want {
		t.Errorf("trailing stdout line: got %q, want %q", got, want)
	}

	archive := readBackArchive(t, dir)
	person, ok := archive.Persons[want]
	if !ok {
		t.Fatalf("person %s missing from archive", want)
	}
	if got := person.Properties[glxlib.PersonPropertySex]; got != "male" {
		t.Errorf("sex property: got %v, want male", got)
	}
	name, ok := person.Properties[glxlib.PersonPropertyName].(map[string]any)
	if !ok {
		t.Fatalf("name property: got %T, want map", person.Properties[glxlib.PersonPropertyName])
	}
	if got := name["value"]; got != "Johann Peter Jungk" {
		t.Errorf("name.value: got %v, want %q", got, "Johann Peter Jungk")
	}
	fields, _ := name["fields"].(map[string]any)
	if fields["given"] != "Johann Peter" {
		t.Errorf("name.fields.given: got %v", fields["given"])
	}
}

func TestAdd_PersonRequiresAName(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
	})
	if !errors.Is(err, ErrAddPersonNameRequired) {
		t.Errorf("expected ErrAddPersonNameRequired, got %v", err)
	}
}

func TestAdd_PersonBadVocabRejected(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Jane",
		Sex:              "bogus",
	})
	if !errors.Is(err, ErrAddVocabKeyUnknown) {
		t.Errorf("expected ErrAddVocabKeyUnknown, got %v", err)
	}
}

// =============================================================================
// add place
// =============================================================================

func TestAdd_PlaceWithParent(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	if err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Germany",
		Type:             "country",
	}); err != nil {
		t.Fatalf("addPlace germany: %v", err)
	}
	if err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Enkirch",
		Type:             "city",
		Parent:           "place-germany",
	}); err != nil {
		t.Fatalf("addPlace enkirch: %v", err)
	}

	archive := readBackArchive(t, dir)
	enkirch, ok := archive.Places["place-enkirch"]
	if !ok {
		t.Fatalf("place-enkirch missing")
	}
	if enkirch.ParentID != "place-germany" {
		t.Errorf("parent: got %q, want place-germany", enkirch.ParentID)
	}
}

func TestAdd_PlaceUnknownParentRejected(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Enkirch",
		Parent:           "place-nope",
	})
	if !errors.Is(err, ErrAddRefNotFound) {
		t.Errorf("expected ErrAddRefNotFound, got %v", err)
	}
}

// =============================================================================
// add event
// =============================================================================

func TestAdd_EventWithPrincipalAndPlace(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Johann",
		Surname:          "Jungk",
	}); err != nil {
		t.Fatalf("addPerson: %v", err)
	}
	if err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Enkirch",
	}); err != nil {
		t.Fatalf("addPlace: %v", err)
	}
	if err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "christening",
		Date:             "1725-02-25",
		Place:            "place-enkirch",
		Principal:        "person-johann-jungk",
	}); err != nil {
		t.Fatalf("addEvent: %v", err)
	}

	archive := readBackArchive(t, dir)
	event, ok := archive.Events["event-christening-johann-jungk"]
	if !ok {
		t.Fatalf("event-christening-johann-jungk missing")
	}
	if event.Type != "christening" || event.PlaceID != "place-enkirch" {
		t.Errorf("event fields wrong: %+v", event)
	}
	if got := string(event.Date); got != "1725-02-25" {
		t.Errorf("date: got %q", got)
	}
	if len(event.Participants) != 1 || event.Participants[0].Person != "person-johann-jungk" ||
		event.Participants[0].Role != glxlib.ParticipantRolePrincipal {
		t.Errorf("participants: %+v", event.Participants)
	}
}

func TestAdd_EventRequiresType(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
	})
	if !errors.Is(err, ErrAddEventTypeRequired) {
		t.Errorf("expected ErrAddEventTypeRequired, got %v", err)
	}
}

func TestAdd_EventRejectsBadParticipantFlag(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "marriage",
		Participants:     []string{""},
	})
	if !errors.Is(err, ErrAddParticipantFormat) {
		t.Errorf("expected ErrAddParticipantFormat, got %v", err)
	}
}

// =============================================================================
// add repository / source / citation
// =============================================================================

func TestAdd_RepositorySourceCitationChain(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	if err := addRepository(io, &addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "FamilySearch",
		Type:             "database",
		Website:          "https://www.familysearch.org",
	}); err != nil {
		t.Fatalf("addRepository: %v", err)
	}
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "Deutschland Geburten",
		Type:             "database",
		Repository:       "repository-familysearch",
	}); err != nil {
		t.Fatalf("addSource: %v", err)
	}
	if err := addCitation(io, &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Source:           "source-deutschland-geburten",
		URL:              "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2",
		Accessed:         "2026-04-20",
		ExternalIDs:      []string{"familysearch:ark:/61903/1:1:C4H8-2DW2"},
	}); err != nil {
		t.Fatalf("addCitation: %v", err)
	}

	archive := readBackArchive(t, dir)
	if _, ok := archive.Repositories["repository-familysearch"]; !ok {
		t.Errorf("repository missing")
	}
	src, ok := archive.Sources["source-deutschland-geburten"]
	if !ok {
		t.Fatalf("source missing")
	}
	if src.RepositoryID != "repository-familysearch" {
		t.Errorf("source repository: got %q", src.RepositoryID)
	}

	// Pick out the citation by source prefix — its full ID has a hash suffix.
	var foundCitation *glxlib.Citation
	for id, c := range archive.Citations {
		if strings.HasPrefix(id, "citation-deutschland-geburten-") {
			foundCitation = c

			break
		}
	}
	if foundCitation == nil {
		t.Fatalf("citation missing; archive citations=%+v", archive.Citations)
	}
	if foundCitation.SourceID != "source-deutschland-geburten" {
		t.Errorf("citation source: got %q", foundCitation.SourceID)
	}
	if got := foundCitation.Properties["url"]; got != "https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2" {
		t.Errorf("citation url: got %v", got)
	}
	extIDs, ok := foundCitation.Properties["external_ids"].([]any)
	if !ok || len(extIDs) != 1 {
		t.Fatalf("external_ids: got %T (%v)", foundCitation.Properties["external_ids"], foundCitation.Properties["external_ids"])
	}
	entry, ok := extIDs[0].(map[string]any)
	if !ok {
		t.Fatalf("external_ids[0] not a structured entry: %T", extIDs[0])
	}
	if entry["value"] != "ark:/61903/1:1:C4H8-2DW2" {
		t.Errorf("external_ids[0].value: got %v", entry["value"])
	}
	fields, _ := entry["fields"].(map[string]any)
	if fields["type"] != "familysearch" {
		t.Errorf("external_ids[0].fields.type: got %v", fields["type"])
	}
}

func TestAdd_PersonRequiresResidencePlaceRef(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Jane",
		Residence:        "place-nonexistent",
	})
	if !errors.Is(err, ErrAddRefNotFound) {
		t.Errorf("expected ErrAddRefNotFound, got %v", err)
	}
}

func TestAdd_CitationRequiresDistinguisher(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "S",
	}); err != nil {
		t.Fatalf("setup source: %v", err)
	}
	err := addCitation(io, &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Source:           "source-s",
	})
	if !errors.Is(err, ErrAddCitationDistinguisherRequired) {
		t.Errorf("expected ErrAddCitationDistinguisherRequired, got %v", err)
	}
}

func TestAdd_CitationDeterministicIDsAreIdempotent(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "S",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	opts := &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Source:           "source-s",
		URL:              "https://example.com/record/1",
	}
	if err := addCitation(io, opts); err != nil {
		t.Fatalf("first add: %v", err)
	}
	first := citationIDs(readBackArchive(t, dir))

	// Second invocation with identical inputs derives the same slug, hits
	// the collision path, and auto-suffixes to citation-...-2. The point is
	// that both IDs are stable — repeating yet again would produce -3, not
	// a brand-new hash from time.Now().
	if err := addCitation(io, opts); err != nil {
		t.Fatalf("second add: %v", err)
	}
	second := citationIDs(readBackArchive(t, dir))
	if len(second) != 2 {
		t.Fatalf("expected 2 citation IDs after second add, got %v", second)
	}
	for _, id := range first {
		if !slices.Contains(second, id) {
			t.Errorf("first-pass citation %q vanished from second-pass archive (non-idempotent)", id)
		}
	}
}

func citationIDs(archive *glxlib.GLXFile) []string {
	out := make([]string, 0, len(archive.Citations))
	for k := range archive.Citations {
		out = append(out, k)
	}

	return out
}

func TestAdd_CitationRequiresSource(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	err := addCitation(io, &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
	})
	if !errors.Is(err, ErrAddCitationSourceRequired) {
		t.Errorf("expected ErrAddCitationSourceRequired, got %v", err)
	}
}

func TestAdd_CitationBadExternalIDFlag(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addRepository(io, &addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "R",
	}); err != nil {
		t.Fatalf("setup repo: %v", err)
	}
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "S",
	}); err != nil {
		t.Fatalf("setup source: %v", err)
	}
	err := addCitation(io, &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Source:           "source-s",
		ExternalIDs:      []string{"missing-colon"},
	})
	if !errors.Is(err, ErrAddExternalIDFormat) {
		t.Errorf("expected ErrAddExternalIDFormat, got %v", err)
	}
}

// =============================================================================
// add relationship / assertion
// =============================================================================

func TestAdd_RelationshipParentChild(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	for _, given := range []string{"Father", "Son"} {
		if err := addPerson(io, &addPersonOptions{
			addCommonOptions: addCommonOptions{ArchivePath: dir},
			Given:            given,
			Surname:          "Smith",
		}); err != nil {
			t.Fatalf("addPerson %s: %v", given, err)
		}
	}
	if err := addRelationship(io, &addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "biological_parent_child",
		Parents:          []string{"person-father-smith"},
		Children:         []string{"person-son-smith"},
	}); err != nil {
		t.Fatalf("addRelationship: %v", err)
	}

	archive := readBackArchive(t, dir)
	rel, ok := archive.Relationships["relationship-biological-parent-child-father-smith-son-smith"]
	if !ok {
		t.Fatalf("relationship missing; have keys: %v", relKeys(archive))
	}
	if rel.Type != "biological_parent_child" {
		t.Errorf("type: %q", rel.Type)
	}
	if len(rel.Participants) != 2 {
		t.Fatalf("participants: %+v", rel.Participants)
	}
}

func relKeys(archive *glxlib.GLXFile) []string {
	out := make([]string, 0, len(archive.Relationships))
	for k := range archive.Relationships {
		out = append(out, k)
	}

	return out
}

func TestAdd_RelationshipRequiresTwoParticipants(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Lonely",
	}); err != nil {
		t.Fatalf("addPerson: %v", err)
	}
	err := addRelationship(io, &addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "marriage",
		Spouses:          []string{"person-lonely"},
	})
	if !errors.Is(err, ErrAddRelationshipParticipantsRequired) {
		t.Errorf("expected ErrAddRelationshipParticipantsRequired, got %v", err)
	}
}

func TestAdd_AssertionFactOnEvent(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Johann",
	}); err != nil {
		t.Fatalf("addPerson: %v", err)
	}
	if err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "christening",
		Principal:        "person-johann",
	}); err != nil {
		t.Fatalf("addEvent: %v", err)
	}
	if err := addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		SubjectEvent:     "event-christening-johann",
		Property:         "date",
		Value:            "1725-02-18",
		Confidence:       "medium",
	}); err != nil {
		t.Fatalf("addAssertion: %v", err)
	}

	archive := readBackArchive(t, dir)
	assertion, ok := archive.Assertions["assertion-christening-johann-date"]
	if !ok {
		t.Fatalf("assertion missing; have keys: %v", assertionKeys(archive))
	}
	if assertion.Subject.Event != "event-christening-johann" {
		t.Errorf("subject.event: %q", assertion.Subject.Event)
	}
	if assertion.Property != "date" || assertion.Value != "1725-02-18" {
		t.Errorf("assertion fields: %+v", assertion)
	}
}

func assertionKeys(archive *glxlib.GLXFile) []string {
	out := make([]string, 0, len(archive.Assertions))
	for k := range archive.Assertions {
		out = append(out, k)
	}

	return out
}

func TestAdd_AssertionParticipantPayload(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	for _, given := range []string{"Subject", "Godparent"} {
		if err := addPerson(io, &addPersonOptions{
			addCommonOptions: addCommonOptions{ArchivePath: dir},
			Given:            given,
		}); err != nil {
			t.Fatalf("addPerson %s: %v", given, err)
		}
	}
	if err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "christening",
		Principal:        "person-subject",
	}); err != nil {
		t.Fatalf("addEvent: %v", err)
	}
	if err := addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		SubjectEvent:     "event-christening-subject",
		Participant:      "person-godparent:godparent",
	}); err != nil {
		t.Fatalf("addAssertion: %v", err)
	}

	archive := readBackArchive(t, dir)
	assertion, ok := archive.Assertions["assertion-christening-subject-godparent"]
	if !ok {
		t.Fatalf("expected assertion-christening-subject-godparent; have %v", assertionKeys(archive))
	}
	if assertion.Participant == nil {
		t.Fatalf("participant is nil")
	}
	if got := assertion.Participant.Person; got != "person-godparent" {
		t.Errorf("participant.person: %q", got)
	}
	if got := assertion.Participant.Role; got != "godparent" {
		t.Errorf("participant.role: %q", got)
	}
	if assertion.Property != "" || assertion.Value != "" {
		t.Errorf("participant-only assertion should not have property/value; got prop=%q val=%q", assertion.Property, assertion.Value)
	}
}

func TestAdd_AssertionRejectsPartialFactPayload(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "P",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, tc := range []struct {
		name     string
		property string
		value    string
	}{
		{"property only", "date", ""},
		{"value only", "", "1850"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := addAssertion(io, &addAssertionOptions{
				addCommonOptions: addCommonOptions{ArchivePath: dir},
				SubjectPerson:    "person-p",
				Property:         tc.property,
				Value:            tc.value,
			})
			if !errors.Is(err, ErrAddAssertionPayloadRequired) {
				t.Errorf("expected ErrAddAssertionPayloadRequired, got %v", err)
			}
		})
	}
}

func TestAdd_AssertionRequiresExactlyOneSubject(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	// Missing
	err := addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Property:         "p",
		Value:            "v",
	})
	if !errors.Is(err, ErrAddSubjectRequired) {
		t.Errorf("missing: expected ErrAddSubjectRequired, got %v", err)
	}

	// Two
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "P",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "birth",
		Principal:        "person-p",
	}); err != nil {
		t.Fatalf("setup event: %v", err)
	}
	err = addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		SubjectPerson:    "person-p",
		SubjectEvent:     "event-birth-p",
		Property:         "p",
		Value:            "v",
	})
	if !errors.Is(err, ErrAddSubjectConflict) {
		t.Errorf("two: expected ErrAddSubjectConflict, got %v", err)
	}
}

func TestAdd_AssertionPayloadConflict(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "P",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		SubjectPerson:    "person-p",
		Property:         "name",
		Value:            "P",
		Participant:      "person-p:witness",
	})
	if !errors.Is(err, ErrAddAssertionPayloadConflict) {
		t.Errorf("expected ErrAddAssertionPayloadConflict, got %v", err)
	}
}

// =============================================================================
// shared behaviors (ID override, --force, --dry-run, --skip-validate)
// =============================================================================

func TestAdd_ExplicitIDCollisionErrorsWithoutForce(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	for i, given := range []string{"A", "B"} {
		opts := &addPersonOptions{
			addCommonOptions: addCommonOptions{ArchivePath: dir, OverrideID: "person-shared"},
			Given:            given,
		}
		err := addPerson(io, opts)
		if i == 0 {
			if err != nil {
				t.Fatalf("first add: %v", err)
			}
		} else {
			if !errors.Is(err, ErrAddEntityExists) {
				t.Errorf("second add: expected ErrAddEntityExists, got %v", err)
			}
		}
	}
}

func TestAdd_ExplicitIDCollisionForceOverwrites(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, OverrideID: "person-shared"},
		Given:            "A",
	}); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, OverrideID: "person-shared", Force: true},
		Given:            "B",
	}); err != nil {
		t.Fatalf("force: %v", err)
	}
	archive := readBackArchive(t, dir)
	name, ok := archive.Persons["person-shared"].Properties[glxlib.PersonPropertyName].(map[string]any)
	if !ok {
		t.Fatalf("name not a map: %T", archive.Persons["person-shared"].Properties[glxlib.PersonPropertyName])
	}
	if name["value"] != "B" {
		t.Errorf("expected overwrite to B, got %v", name["value"])
	}
}

func TestAdd_DerivedIDsAutoSuffixOnCollision(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	wantIDs := []string{"person-jane", "person-jane-2", "person-jane-3"}
	for i := range wantIDs {
		if err := addPerson(io, &addPersonOptions{
			addCommonOptions: addCommonOptions{ArchivePath: dir},
			Given:            "Jane",
		}); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	archive := readBackArchive(t, dir)
	for _, want := range wantIDs {
		if _, ok := archive.Persons[want]; !ok {
			t.Errorf("expected %s to exist; persons=%v", want, personKeys(archive))
		}
	}
}

func personKeys(archive *glxlib.GLXFile) []string {
	out := make([]string, 0, len(archive.Persons))
	for k := range archive.Persons {
		out = append(out, k)
	}

	return out
}

func TestAdd_DryRunWritesNothing(t *testing.T) {
	dir := initArchiveDir(t)
	io, out, _ := TestIOStreams()
	before := countFiles(t, filepath.Join(dir, "persons"))

	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, DryRun: true},
		Given:            "Dryrun",
	}); err != nil {
		t.Fatalf("addPerson dry: %v", err)
	}

	after := countFiles(t, filepath.Join(dir, "persons"))
	if before != after {
		t.Errorf("dry-run wrote files: before=%d after=%d", before, after)
	}
	if !strings.Contains(out.String(), "(dry run") {
		t.Errorf("output should mention dry-run; got %q", out.String())
	}
	if got := trailingLine(out.String()); got != "person-dryrun" {
		t.Errorf("trailing line: got %q, want person-dryrun", got)
	}
}

func TestAdd_QuietSuppressesDiagnosticsButKeepsIDEcho(t *testing.T) {
	dir := initArchiveDir(t)

	machineOut := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	// Mirror the SystemIOStreams shape when --quiet is set: Out discarded,
	// MachineOut still routed to a real writer.
	streams := &IOStreams{Out: io.Discard, MachineOut: machineOut, ErrOut: errOut}

	if err := addPerson(streams, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "Quiet",
		Surname:          "Person",
	}); err != nil {
		t.Fatalf("addPerson: %v", err)
	}

	if got := strings.TrimSpace(machineOut.String()); got != "person-quiet-person" {
		t.Errorf("MachineOut: got %q, want person-quiet-person", got)
	}
}

func TestAdd_RejectsUnsafeOverrideID(t *testing.T) {
	dir := initArchiveDir(t)
	cases := map[string]string{
		"path traversal":      "../escape",
		"control char":        "person-x\x00bad",
		"too long":            "person-" + strings.Repeat("a", glxlib.MaxEntityIDLength),
		"leading hyphen":      "-leading-hyphen",
		"forbidden character": "person@bad",
	}
	for name, badID := range cases {
		t.Run(name, func(t *testing.T) {
			io, _, _ := TestIOStreams()
			err := addPerson(io, &addPersonOptions{
				addCommonOptions: addCommonOptions{ArchivePath: dir, OverrideID: badID},
				Given:            "Jane",
			})
			if !errors.Is(err, ErrAddInvalidID) {
				t.Errorf("%s: expected ErrAddInvalidID, got %v", name, err)
			}
		})
	}
}

func TestDeriveOrOverrideID_RejectsUnsafeDerivedBase(t *testing.T) {
	// base is contractually pre-slugified via glxlib.EntityID, so this never
	// fires in normal flows — but deriveOrOverrideID promises a validated ID,
	// so a malformed base (e.g. a caller bug producing an uncapped or invalid
	// value from user input) must be rejected, not returned for writing.
	cases := map[string]string{
		"spaces and punctuation": "Bad ID!",
		"path traversal":         "../escape",
		"too long":               strings.Repeat("a", glxlib.MaxEntityIDLength+1),
		"leading hyphen":         "-leading",
	}
	for name, badBase := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := deriveOrOverrideID(badBase, "", map[string]struct{}{}, false)
			if !errors.Is(err, ErrAddInvalidID) {
				t.Errorf("%s: expected ErrAddInvalidID for unsafe base, got %v", name, err)
			}
		})
	}
}

func TestAdd_SkipValidateAllowsStaleArchive(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	// Hand-craft a stale assertion file pointing at a non-existent event so
	// the whole-archive validation pass fails. Without --skip-validate the
	// add must error; with --skip-validate it must succeed.
	stale := "" +
		"assertions:\n" +
		"  assertion-stale:\n" +
		"    subject:\n" +
		"      event: event-missing\n" +
		"    property: date\n" +
		"    value: \"1850\"\n"
	staleDir := filepath.Join(dir, "assertions")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staleDir, "assertion-stale.glx"), []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale: %v", err)
	}

	// Without skip-validate: should fail because the archive doesn't validate
	// post-merge (the new place is fine; the stale assertion is the problem).
	err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "WithValidate",
	})
	if err == nil {
		t.Errorf("expected whole-archive validation to surface the stale assertion")
	}

	// With skip-validate: should succeed.
	err = addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Name:             "SkipValidate",
	})
	if err != nil {
		t.Errorf("with --skip-validate: %v", err)
	}
}

// =============================================================================
// helpers
// =============================================================================

func TestParseParticipantFlag(t *testing.T) {
	tests := []struct {
		in         string
		wantPerson string
		wantRole   string
		wantErr    bool
	}{
		{"person-x:role", "person-x", "role", false},
		{"person-x", "person-x", "", false},
		{"  person-x : role  ", "person-x", "role", false},
		{":role", "", "", true},
		{"person-x:", "", "", true}, // trailing colon == empty role, rejected
		{"", "", "", true},
	}
	for _, tc := range tests {
		gotPerson, gotRole, err := parseParticipantFlag(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", tc.in)
			}

			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)

			continue
		}
		if gotPerson != tc.wantPerson || gotRole != tc.wantRole {
			t.Errorf("%q: got (%q,%q), want (%q,%q)", tc.in, gotPerson, gotRole, tc.wantPerson, tc.wantRole)
		}
	}
}

func TestParseExternalIDFlag(t *testing.T) {
	tests := []struct {
		in      string
		wantT   string
		wantV   string
		wantErr bool
	}{
		{"familysearch:ark:/61903/1:1:C4H8-2DW2", "familysearch", "ark:/61903/1:1:C4H8-2DW2", false},
		{"type:value", "type", "value", false},
		{"missing-colon", "", "", true},
		{":value", "", "", true},
		{"type:", "", "", true},
	}
	for _, tc := range tests {
		gotT, gotV, err := parseExternalIDFlag(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: expected error", tc.in)
			}

			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)

			continue
		}
		if gotT != tc.wantT || gotV != tc.wantV {
			t.Errorf("%q: got (%q,%q), want (%q,%q)", tc.in, gotT, gotV, tc.wantT, tc.wantV)
		}
	}
}

func TestAdd_RepositoryHappyPath(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addRepository(io, &addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Virginia State Library",
		Type:             "library",
		Address:          "800 E. Broad St",
		City:             "Richmond",
		State:            "VA",
		PostalCode:       "23219",
		Country:          "USA",
		Website:          "https://lva.virginia.gov",
	}); err != nil {
		t.Fatalf("addRepository: %v", err)
	}
	archive := readBackArchive(t, dir)
	repo, ok := archive.Repositories["repository-virginia-state-library"]
	if !ok {
		t.Fatalf("repository missing")
	}
	if repo.Type != "library" || repo.City != "Richmond" || repo.State != "VA" {
		t.Errorf("repository fields wrong: %+v", repo)
	}
}

func TestAdd_SourceHappyPath(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addRepository(io, &addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "FamilySearch",
		Type:             "database",
	}); err != nil {
		t.Fatalf("setup repo: %v", err)
	}
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "Virginia Marriage Records",
		Type:             "database",
		Repository:       "repository-familysearch",
		Authors:          []string{"FamilySearch"},
		Date:             "1850",
		Description:      "Indexed VA marriages",
		Language:         "en",
	}); err != nil {
		t.Fatalf("addSource: %v", err)
	}
	archive := readBackArchive(t, dir)
	source, ok := archive.Sources["source-virginia-marriage-records"]
	if !ok {
		t.Fatalf("source missing")
	}
	if source.RepositoryID != "repository-familysearch" || source.Properties["description"] != "Indexed VA marriages" {
		t.Errorf("source fields wrong: %+v", source)
	}
}

func TestAdd_AssertionSubjects(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	// Seed: person, event, relationship (between two persons), place.
	for _, given := range []string{"A", "B"} {
		if err := addPerson(io, &addPersonOptions{
			addCommonOptions: addCommonOptions{ArchivePath: dir},
			Given:            given,
		}); err != nil {
			t.Fatalf("addPerson %s: %v", given, err)
		}
	}
	if err := addEvent(io, &addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "marriage",
		Date:             "1850",
		Participants:     []string{"person-a:groom", "person-b:bride"},
	}); err != nil {
		t.Fatalf("addEvent: %v", err)
	}
	if err := addRelationship(io, &addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "marriage",
		Spouses:          []string{"person-a", "person-b"},
	}); err != nil {
		t.Fatalf("addRelationship: %v", err)
	}
	if err := addPlace(io, &addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "Richmond",
	}); err != nil {
		t.Fatalf("addPlace: %v", err)
	}

	type subjectCase struct {
		name, subjectPerson, subjectEvent, subjectRelationship, subjectPlace string
		wantPersonField, wantEventField                                      string
		wantRelField, wantPlaceField                                         string
	}
	cases := []subjectCase{
		{name: "subject-person", subjectPerson: "person-a", wantPersonField: "person-a"},
		{name: "subject-event", subjectEvent: "event-marriage-1850", wantEventField: "event-marriage-1850"},
		{name: "subject-relationship", subjectRelationship: "relationship-marriage-a-b", wantRelField: "relationship-marriage-a-b"},
		{name: "subject-place", subjectPlace: "place-richmond", wantPlaceField: "place-richmond"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := addAssertion(io, &addAssertionOptions{
				addCommonOptions:    addCommonOptions{ArchivePath: dir},
				SubjectPerson:       tc.subjectPerson,
				SubjectEvent:        tc.subjectEvent,
				SubjectRelationship: tc.subjectRelationship,
				SubjectPlace:        tc.subjectPlace,
				Property:            "note",
				Value:               tc.name,
			}); err != nil {
				t.Fatalf("addAssertion: %v", err)
			}
			archive := readBackArchive(t, dir)
			var found *glxlib.Assertion
			for _, a := range archive.Assertions {
				if a.Value == tc.name {
					found = a

					break
				}
			}
			if found == nil {
				t.Fatalf("assertion missing for %s", tc.name)
			}
			if found.Subject.Person != tc.wantPersonField {
				t.Errorf("subject.person: got %q, want %q", found.Subject.Person, tc.wantPersonField)
			}
			if found.Subject.Event != tc.wantEventField {
				t.Errorf("subject.event: got %q, want %q", found.Subject.Event, tc.wantEventField)
			}
			if found.Subject.Relationship != tc.wantRelField {
				t.Errorf("subject.relationship: got %q, want %q", found.Subject.Relationship, tc.wantRelField)
			}
			if found.Subject.Place != tc.wantPlaceField {
				t.Errorf("subject.place: got %q, want %q", found.Subject.Place, tc.wantPlaceField)
			}
		})
	}
}

func TestAdd_AssertionRefValidation(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addPerson(io, &addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Given:            "P",
	}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, tc := range []struct {
		name      string
		sources   []string
		citations []string
		media     []string
	}{
		{name: "bad source", sources: []string{"source-nope"}},
		{name: "bad citation", citations: []string{"citation-nope"}},
		{name: "bad media", media: []string{"media-nope"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := addAssertion(io, &addAssertionOptions{
				addCommonOptions: addCommonOptions{ArchivePath: dir},
				SubjectPerson:    "person-p",
				Property:         "note",
				Value:            "x",
				Sources:          tc.sources,
				Citations:        tc.citations,
				Media:            tc.media,
			})
			if !errors.Is(err, ErrAddRefNotFound) {
				t.Errorf("expected ErrAddRefNotFound, got %v", err)
			}
		})
	}
}

func TestAdd_RequiredFlagErrors(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	type call func() error
	cases := []struct {
		name  string
		run   call
		errIs error
	}{
		{
			name: "place without --name",
			run: func() error {
				return addPlace(io, &addPlaceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}})
			},
			errIs: ErrAddPlaceNameRequired,
		},
		{
			name: "repository without --name",
			run: func() error {
				return addRepository(io, &addRepositoryOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}})
			},
			errIs: ErrAddRepositoryNameRequired,
		},
		{
			name: "source without --title",
			run: func() error {
				return addSource(io, &addSourceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}})
			},
			errIs: ErrAddSourceTitleRequired,
		},
		{
			name: "relationship without --type",
			run: func() error {
				return addRelationship(io, &addRelationshipOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}})
			},
			errIs: ErrAddRelationshipTypeRequired,
		},
		{
			name: "citation without --source",
			run: func() error {
				return addCitation(io, &addCitationOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}})
			},
			errIs: ErrAddCitationSourceRequired,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if !errors.Is(err, tc.errIs) {
				t.Errorf("expected %v, got %v", tc.errIs, err)
			}
		})
	}
}

func TestAdd_BadVocabKeys(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	cases := []struct {
		name string
		run  func() error
	}{
		{name: "place bad type", run: func() error {
			return addPlace(io, &addPlaceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Name: "X", Type: "bogus"})
		}},
		{name: "repository bad type", run: func() error {
			return addRepository(io, &addRepositoryOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Name: "X", Type: "bogus"})
		}},
		{name: "source bad type", run: func() error {
			return addSource(io, &addSourceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Title: "X", Type: "bogus"})
		}},
		{name: "event bad type", run: func() error {
			return addEvent(io, &addEventOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "bogus"})
		}},
		{name: "relationship bad type", run: func() error {
			return addRelationship(io, &addRelationshipOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "bogus"})
		}},
		{name: "person bad gender", run: func() error {
			return addPerson(io, &addPersonOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Given: "X", Gender: "bogus"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if !errors.Is(err, ErrAddVocabKeyUnknown) {
				t.Errorf("expected ErrAddVocabKeyUnknown, got %v", err)
			}
		})
	}
}

func TestAdd_BadReferences(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	cases := []struct {
		name string
		run  func() error
	}{
		{name: "event bad place", run: func() error {
			return addEvent(io, &addEventOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "birth", Place: "place-nope"})
		}},
		{name: "event bad principal", run: func() error {
			return addEvent(io, &addEventOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "birth", Principal: "person-nope"})
		}},
		{name: "event bad participant ref", run: func() error {
			return addEvent(io, &addEventOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "birth", Participants: []string{"person-nope:principal"}})
		}},
		{name: "source bad repository", run: func() error {
			return addSource(io, &addSourceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Title: "X", Repository: "repository-nope"})
		}},
		{name: "citation bad source", run: func() error {
			return addCitation(io, &addCitationOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Source: "source-nope", URL: "x"})
		}},
		{name: "relationship bad start-event", run: func() error {
			return addRelationship(io, &addRelationshipOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "marriage", StartEvent: "event-nope"})
		}},
		{name: "relationship bad parent person", run: func() error {
			return addRelationship(io, &addRelationshipOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Type: "biological_parent_child", Parents: []string{"person-nope"}, Children: []string{"person-nope2"}})
		}},
		{name: "assertion bad subject person", run: func() error {
			return addAssertion(io, &addAssertionOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, SubjectPerson: "person-nope", Property: "p", Value: "v"})
		}},
		{name: "assertion participant bad person", run: func() error {
			// Seed a real subject so subject resolution passes.
			if err := addPerson(io, &addPersonOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Given: "Real"}); err != nil {
				return err
			}

			return addAssertion(io, &addAssertionOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, SubjectPerson: "person-real", Participant: "person-nope:witness"})
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if !errors.Is(err, ErrAddRefNotFound) {
				t.Errorf("expected ErrAddRefNotFound, got %v", err)
			}
		})
	}
}

func TestAdd_RelationshipParticipantFlagAndBadRole(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	for _, given := range []string{"X", "Y"} {
		if err := addPerson(io, &addPersonOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Given: given}); err != nil {
			t.Fatalf("setup person %s: %v", given, err)
		}
	}

	// Happy path: --participant person-id:role twice.
	if err := addRelationship(io, &addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "sibling",
		Participants:     []string{"person-x:sibling", "person-y:sibling"},
	}); err != nil {
		t.Fatalf("addRelationship via participants: %v", err)
	}

	// Bad role rejected.
	err := addRelationship(io, &addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Type:             "sibling",
		Participants:     []string{"person-x:bogusrole", "person-y:bogusrole"},
	})
	if !errors.Is(err, ErrAddVocabKeyUnknown) {
		t.Errorf("expected ErrAddVocabKeyUnknown for bad role, got %v", err)
	}
}

func TestAdd_PlaceWithCoordinates(t *testing.T) {
	dir := initArchiveDir(t)

	addPlaceLat = 49.97
	addPlaceLng = 7.13
	addPlaceLatSet = true
	addPlaceLngSet = true
	addPlaceOpts = addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Name:             "Enkirch",
		Type:             "city",
	}
	t.Cleanup(func() {
		addPlaceLat, addPlaceLng = 0, 0
		addPlaceLatSet, addPlaceLngSet = false, false
		addPlaceOpts = addPlaceOptions{}
	})

	if err := runAddPlace(nil, nil); err != nil {
		t.Fatalf("runAddPlace: %v", err)
	}

	archive := readBackArchive(t, dir)
	place, ok := archive.Places["place-enkirch"]
	if !ok {
		t.Fatalf("place-enkirch missing")
	}
	if place.Latitude == nil || *place.Latitude != 49.97 {
		t.Errorf("latitude: %v", place.Latitude)
	}
	if place.Longitude == nil || *place.Longitude != 7.13 {
		t.Errorf("longitude: %v", place.Longitude)
	}
}

func TestAdd_AssertionWithEvidence(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()

	if err := addPerson(io, &addPersonOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Given: "P"}); err != nil {
		t.Fatalf("setup person: %v", err)
	}
	if err := addRepository(io, &addRepositoryOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Name: "R"}); err != nil {
		t.Fatalf("setup repo: %v", err)
	}
	if err := addSource(io, &addSourceOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Title: "S"}); err != nil {
		t.Fatalf("setup source: %v", err)
	}
	if err := addCitation(io, &addCitationOptions{addCommonOptions: addCommonOptions{ArchivePath: dir}, Source: "source-s", URL: "https://x"}); err != nil {
		t.Fatalf("setup citation: %v", err)
	}
	citationID := ""
	for id := range readBackArchive(t, dir).Citations {
		citationID = id

		break
	}

	if err := addAssertion(io, &addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, Notes: []string{"researcher note"}},
		SubjectPerson:    "person-p",
		Property:         "note",
		Value:            "evidence",
		Sources:          []string{"source-s"},
		Citations:        []string{citationID},
		Confidence:       "high",
		Status:           "accepted",
	}); err != nil {
		t.Fatalf("addAssertion: %v", err)
	}

	archive := readBackArchive(t, dir)
	var a *glxlib.Assertion
	for _, ent := range archive.Assertions {
		if ent.Property == "note" && ent.Value == "evidence" {
			a = ent

			break
		}
	}
	if a == nil {
		t.Fatalf("evidence assertion missing")
	}
	if len(a.Sources) != 1 || a.Sources[0] != "source-s" {
		t.Errorf("sources: %v", a.Sources)
	}
	if len(a.Citations) != 1 || a.Citations[0] != citationID {
		t.Errorf("citations: %v", a.Citations)
	}
	if a.Confidence != "high" || a.Status != "accepted" {
		t.Errorf("confidence/status: %q / %q", a.Confidence, a.Status)
	}
	if len(a.Notes) != 1 || a.Notes[0] != "researcher note" {
		t.Errorf("notes: %v", a.Notes)
	}
}

func TestAdd_CitationWithRepository(t *testing.T) {
	dir := initArchiveDir(t)
	io, _, _ := TestIOStreams()
	if err := addRepository(io, &addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Name:             "R",
	}); err != nil {
		t.Fatalf("setup repo: %v", err)
	}
	if err := addSource(io, &addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Title:            "S",
	}); err != nil {
		t.Fatalf("setup source: %v", err)
	}
	if err := addCitation(io, &addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir},
		Source:           "source-s",
		Repository:       "repository-r",
		Locator:          "p. 42",
		SourceDate:       "1850",
	}); err != nil {
		t.Fatalf("addCitation: %v", err)
	}
	archive := readBackArchive(t, dir)
	var cit *glxlib.Citation
	for _, c := range archive.Citations {
		cit = c

		break
	}
	if cit == nil {
		t.Fatalf("citation missing")
	}
	if cit.RepositoryID != "repository-r" {
		t.Errorf("citation.repository: got %q", cit.RepositoryID)
	}
	if cit.Properties["locator"] != "p. 42" || cit.Properties["source_date"] != "1850" {
		t.Errorf("citation properties wrong: %+v", cit.Properties)
	}
}

// TestAdd_CobraWrappersDelegateCleanly drives each of the eight runAdd* cobra
// handlers end-to-end so they aren't dead weight in coverage. Each handler is
// a thin pass-through to its addX runner; this test sets the package-level
// option vars and invokes the handler directly. The actual entity-add
// behavior is covered by the TestAdd_* tests above; this test only asserts
// that no handler returns an error on a valid invocation, locking in the
// "thin wrapper" contract from glx/CLAUDE.md.
func TestAdd_CobraWrappersDelegateCleanly(t *testing.T) {
	dir := initArchiveDir(t)

	addPersonOpts = addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Given:            "Wrapper",
	}
	if err := runAddPerson(nil, nil); err != nil {
		t.Fatalf("runAddPerson: %v", err)
	}

	addPlaceOpts = addPlaceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Name:             "Wrapperville",
	}
	if err := runAddPlace(nil, nil); err != nil {
		t.Fatalf("runAddPlace: %v", err)
	}

	addEventOpts = addEventOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Type:             "birth",
		Principal:        "person-wrapper",
	}
	if err := runAddEvent(nil, nil); err != nil {
		t.Fatalf("runAddEvent: %v", err)
	}

	addRepoOpts = addRepositoryOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Name:             "Wrapper Repo",
	}
	if err := runAddRepository(nil, nil); err != nil {
		t.Fatalf("runAddRepository: %v", err)
	}

	addSourceOpts = addSourceOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Title:            "Wrapper Source",
	}
	if err := runAddSource(nil, nil); err != nil {
		t.Fatalf("runAddSource: %v", err)
	}

	addCitationOpts = addCitationOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Source:           "source-wrapper-source",
		Locator:          "p. 1",
	}
	if err := runAddCitation(nil, nil); err != nil {
		t.Fatalf("runAddCitation: %v", err)
	}

	// Need a second person for the relationship.
	addPersonOpts = addPersonOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Given:            "Other",
	}
	if err := runAddPerson(nil, nil); err != nil {
		t.Fatalf("runAddPerson (second): %v", err)
	}

	addRelOpts = addRelationshipOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		Type:             "sibling",
		Participants:     []string{"person-wrapper:sibling", "person-other:sibling"},
	}
	if err := runAddRelationship(nil, nil); err != nil {
		t.Fatalf("runAddRelationship: %v", err)
	}

	addAssertionOpts = addAssertionOptions{
		addCommonOptions: addCommonOptions{ArchivePath: dir, SkipValidate: true},
		SubjectPerson:    "person-wrapper",
		Property:         "note",
		Value:            "wrapped",
	}
	if err := runAddAssertion(nil, nil); err != nil {
		t.Fatalf("runAddAssertion: %v", err)
	}

	// Reset package-level option vars so subsequent tests start clean.
	t.Cleanup(func() {
		addPersonOpts = addPersonOptions{}
		addPlaceOpts = addPlaceOptions{}
		addEventOpts = addEventOptions{}
		addRepoOpts = addRepositoryOptions{}
		addSourceOpts = addSourceOptions{}
		addCitationOpts = addCitationOptions{}
		addRelOpts = addRelationshipOptions{}
		addAssertionOpts = addAssertionOptions{}
	})
}

func TestStripEntityPrefix(t *testing.T) {
	cases := map[string]string{
		"person-x":       "x",
		"event-y":        "y",
		"relationship-z": "z",
		"place-foo":      "foo",
		"unprefixed":     "unprefixed",
		"":               "",
	}
	for in, want := range cases {
		if got := stripEntityPrefix(in); got != want {
			t.Errorf("stripEntityPrefix(%q) = %q, want %q", in, got, want)
		}
	}
}
