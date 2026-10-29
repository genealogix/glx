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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	glxlib "github.com/genealogix/glx/go-glx"
)

// brickwallArchive builds the issue #144 scenario, condensed: a person whose
// born_at property is reported as three different states across six census
// citations, with mixed confidence inside the leading group.
func brickwallArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-jane-webb": {Properties: map[string]any{"name": map[string]any{"value": "Jane Miller"}}},
		},
		Sources: map[string]*glxlib.Source{
			"src-1860": {Title: "1860 US Census"},
			"src-1880": {Title: "1880 US Census"},
			"src-1900": {Title: "1900 US Census"},
			"src-1905": {Title: "1905 WI Census"},
			"src-1910": {Title: "1910 US Census"},
			"src-1920": {Title: "1920 US Census"},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-1860-webb":    {SourceID: "src-1860"},
			"cit-1880-clara":   {SourceID: "src-1880"},
			"cit-1905-anna":    {SourceID: "src-1905"},
			"cit-1910-clara":   {SourceID: "src-1910"},
			"cit-1900-william": {SourceID: "src-1900"},
			"cit-1920-william": {SourceID: "src-1920"},
		},
		Assertions: map[string]*glxlib.Assertion{
			"a-va-1": birthplaceAssertion("VIRGINIA", "medium", "cit-1880-clara"),
			"a-va-2": birthplaceAssertion("VIRGINIA", "high", "cit-1905-anna"),
			"a-va-3": birthplaceAssertion("VIRGINIA", "medium", "cit-1910-clara"),
			"a-wi-1": birthplaceAssertion("WISCONSIN", "low", "cit-1900-william"),
			"a-wi-2": birthplaceAssertion("WISCONSIN", "medium", "cit-1920-william"),
			"a-fl-1": birthplaceAssertion("FLORIDA", "low", "cit-1860-webb"),
		},
	}
}

func birthplaceAssertion(value, confidence string, citations ...string) *glxlib.Assertion {
	return &glxlib.Assertion{
		Subject:    glxlib.EntityRef{Person: "person-jane-webb"},
		Property:   "born_at",
		Value:      value,
		Confidence: confidence,
		Citations:  citations,
	}
}























func TestPrintEvidenceText_Output(t *testing.T) {
	report := mustCollectEvidence(brickwallArchive(), glxlib.EntityRef{Person: "person-jane-webb"}, "born_at")
	streams, out, _ := TestIOStreams()

	printEvidenceText(streams, &report)
	got := out.String()

	for _, want := range []string{
		"Evidence for born_at of Jane Miller (person-jane-webb):",
		"6 reports across 3 values",
		"VIRGINIA — 3 reports, best confidence: high",
		"cit-1880-clara",
		"1880 US Census",
		"Best evidence: VIRGINIA (3 reports, high)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("text output missing %q\n--- got ---\n%s", want, got)
		}
	}
}

func TestPrintEvidenceText_EmptyAndTie(t *testing.T) {
	streams, out, _ := TestIOStreams()
	printEvidenceText(streams, &EvidenceReport{Subject: "p", SubjectName: "Pat", SubjectType: "person", Property: "born_at"})
	if !strings.Contains(out.String(), "No assertions found for born_at of Pat (p).") {
		t.Errorf("empty output = %q", out.String())
	}

	streams, out, _ = TestIOStreams()
	printEvidenceText(streams, &EvidenceReport{
		Property:     "prop",
		TotalReports: 2,
		Groups: []EvidenceGroup{
			{Value: "A", Reports: 1, BestConfidence: "medium"},
			{Value: "B", Reports: 1, BestConfidence: "medium"},
		},
	})
	if !strings.Contains(out.String(), "Best evidence: inconclusive") {
		t.Errorf("tie output missing inconclusive notice: %q", out.String())
	}
}

func TestPrintEvidenceJSON_RoundTrip(t *testing.T) {
	report := mustCollectEvidence(brickwallArchive(), glxlib.EntityRef{Person: "person-jane-webb"}, "born_at")
	streams, out, _ := TestIOStreams()

	if err := printEvidenceJSON(streams, &report); err != nil {
		t.Fatalf("printEvidenceJSON: %v", err)
	}

	var decoded EvidenceReport
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded.Property != "born_at" || decoded.TotalReports != 6 || decoded.BestEvidence != "VIRGINIA" {
		t.Errorf("decoded = %+v, want property=born_at total=6 best=VIRGINIA", decoded)
	}
	if len(decoded.Groups) != 3 || decoded.Groups[0].Value != "VIRGINIA" {
		t.Errorf("decoded.Groups = %+v", decoded.Groups)
	}
}

func TestPrintEvidenceJSON_GoesToMachineOut(t *testing.T) {
	report := mustCollectEvidence(brickwallArchive(), glxlib.EntityRef{Person: "person-jane-webb"}, "born_at")
	// Separate Out and MachineOut so we can prove JSON is on the machine stream
	// (which survives --quiet), not the diagnostic stream (which does not).
	var out, machine bytes.Buffer
	streams := &IOStreams{Out: &out, MachineOut: &machine, ErrOut: &out}

	if err := printEvidenceJSON(streams, &report); err != nil {
		t.Fatalf("printEvidenceJSON: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("JSON leaked to Out (would be silenced by --quiet): %q", out.String())
	}
	if err := json.Unmarshal(machine.Bytes(), &report); err != nil {
		t.Fatalf("MachineOut does not contain valid JSON: %v", err)
	}
}

// writeTempArchive serializes a GLXFile to a single .glx file and returns its path.
func writeTempArchive(t *testing.T, archive *glxlib.GLXFile) string {
	t.Helper()
	// Validate:false — the condensed fixtures intentionally omit vocab/schema
	// scaffolding irrelevant to evidence grouping.
	ser := glxlib.NewSerializer(&glxlib.SerializerOptions{Pretty: true, Indent: "  "})
	data, err := ser.SerializeSingleFileBytes(archive)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	path := filepath.Join(t.TempDir(), "archive.glx")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	return path
}

func TestShowEvidence_EndToEnd(t *testing.T) {
	path := writeTempArchive(t, brickwallArchive())

	t.Run("text", func(t *testing.T) {
		streams, out, _ := TestIOStreams()
		if err := showEvidence(streams, path, "person-jane-webb", "born_at", "text"); err != nil {
			t.Fatalf("showEvidence: %v", err)
		}
		if !strings.Contains(out.String(), "Best evidence: VIRGINIA") {
			t.Errorf("text output = %q", out.String())
		}
	})

	t.Run("json", func(t *testing.T) {
		streams, out, _ := TestIOStreams()
		if err := showEvidence(streams, path, "person-jane-webb", "born_at", "json"); err != nil {
			t.Fatalf("showEvidence: %v", err)
		}
		var decoded EvidenceReport
		if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
			t.Fatalf("json output invalid: %v", err)
		}
	})

	t.Run("unknown format", func(t *testing.T) {
		streams, _, _ := TestIOStreams()
		if err := showEvidence(streams, path, "person-jane-webb", "born_at", "xml"); err == nil {
			t.Error("expected error for unknown format, got nil")
		}
	})

	t.Run("person not found", func(t *testing.T) {
		streams, _, _ := TestIOStreams()
		if err := showEvidence(streams, path, "person-nobody", "born_at", "text"); err == nil {
			t.Error("expected error for unknown person, got nil")
		}
	})
}

// TestLoadArchiveForEvidence_SingleFileMergesVocabularies directly verifies
// the merge: a single-file archive without a vocabularies block leaves
// PersonProperties empty until mergeStandardVocabularies runs. After load,
// the standard "residence" definition must be present, otherwise the
// reference-type resolution path in resolveAssertionValue is dead code on
// single-file archives.
func TestLoadArchiveForEvidence_SingleFileMergesVocabularies(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")
	// Minimal single-file archive with no vocabularies block.
	if err := os.WriteFile(archivePath, []byte("persons: {}\n"), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	streams, _, _ := TestIOStreams()
	archive, err := loadArchiveForEvidence(streams, archivePath)
	if err != nil {
		t.Fatalf("loadArchiveForEvidence: %v", err)
	}
	if _, ok := archive.PersonProperties["residence"]; !ok {
		t.Errorf("PersonProperties[residence] missing — standard vocabularies not merged on single-file path")
	}
}

// TestShowEvidence_SingleFileResolvesPlaceReference exercises
// loadArchiveForEvidence's single-file path end-to-end: standard vocabularies
// are merged on load, so a place-reference property (residence) resolves to
// the place name rather than printing the raw place ID — matching the
// directory-archive behavior and loadArchiveForSummary.
func TestShowEvidence_SingleFileResolvesPlaceReference(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.glx")
	content := `persons:
  person-johannes-schepp:
    properties:
      name: "Johannes Schepp"
assertions:
  a1:
    subject:
      person: person-johannes-schepp
    property: residence
    value: place-pohlgoens
    confidence: high
places:
  place-pohlgoens:
    name: "Pohl-Göns"
    type: town
`
	if err := os.WriteFile(archivePath, []byte(content), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	streams, out, _ := TestIOStreams()
	if err := showEvidence(streams, archivePath, "person-johannes-schepp", "residence", "text"); err != nil {
		t.Fatalf("showEvidence: %v", err)
	}
	if !strings.Contains(out.String(), "Pohl-Göns") {
		t.Errorf("expected resolved place name in output, got %q", out.String())
	}
	if strings.Contains(out.String(), "place-pohlgoens") {
		t.Errorf("raw place ID leaked into output: %q", out.String())
	}
}



// ============================================================================
// Subject resolution (#1268): events, places, and relationships, not just
// persons.
// ============================================================================

// parishArchive is the issue #1268 scenario: a single person whose contested
// date and place live on their birth and death events, the canonical
// evidence-first modeling, with nothing asserted on the person at all.
func parishArchive() *glxlib.GLXFile {
	return &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-hollnagel-michael-david": {
				Properties: map[string]any{"name": map[string]any{"value": "Michael David Hollnagel"}},
			},
		},
		Events: map[string]*glxlib.Event{
			"event-birth-est":   {Type: glxlib.EventTypeBirth},
			"event-death-1807":  {Title: "Death of Michael David Hollnagel", Type: glxlib.EventTypeDeath},
			"event-burial-1807": {Type: glxlib.EventTypeBurial},
		},
		Places: map[string]*glxlib.Place{
			"place-liepen": {Name: "Liepen, Vorpommern"},
			"place-anklam": {Name: "Anklam, Vorpommern"},
		},
		Relationships: map[string]*glxlib.Relationship{
			"relationship-marriage-hollnagel": {Type: glxlib.RelationshipTypeMarriage},
		},
		Sources: map[string]*glxlib.Source{
			"src-death":  {Title: "Liepen Death Register 1807"},
			"src-burial": {Title: "Liepen Burial Register 1807"},
		},
		Citations: map[string]*glxlib.Citation{
			"cit-death-entry":  {SourceID: "src-death"},
			"cit-burial-entry": {SourceID: "src-burial"},
		},
		Assertions: map[string]*glxlib.Assertion{
			// The birth date is an inference from the death entry's decadal age
			// band, reported differently by the two registers.
			"a-birth-1": {
				Subject: glxlib.EntityRef{Event: "event-birth-est"}, Property: "date",
				Value: "ABT 1737", Confidence: "low", Citations: []string{"cit-death-entry"},
			},
			"a-birth-2": {
				Subject: glxlib.EntityRef{Event: "event-birth-est"}, Property: "date",
				Value: "ABT 1742", Confidence: "low", Citations: []string{"cit-burial-entry"},
			},
			// The death place is a place reference on an event subject.
			"a-death-place": {
				Subject: glxlib.EntityRef{Event: "event-death-1807"}, Property: "place",
				Value: "place-liepen", Confidence: "high", Citations: []string{"cit-death-entry"},
			},
			"a-burial-place": {
				Subject: glxlib.EntityRef{Event: "event-burial-1807"}, Property: "place",
				Value: "place-anklam", Confidence: "medium", Citations: []string{"cit-burial-entry"},
			},
		},
	}
}

func TestFindEvidenceSubject_ResolvesEverySubjectType(t *testing.T) {
	archive := parishArchive()

	cases := []struct {
		query string
		want  glxlib.EntityRef
	}{
		{"person-hollnagel-michael-david", glxlib.EntityRef{Person: "person-hollnagel-michael-david"}},
		{"event-death-1807", glxlib.EntityRef{Event: "event-death-1807"}},
		{"place-liepen", glxlib.EntityRef{Place: "place-liepen"}},
		{"relationship-marriage-hollnagel", glxlib.EntityRef{Relationship: "relationship-marriage-hollnagel"}},
		// Persons still resolve by name search when no exact ID matches.
		{"Michael David", glxlib.EntityRef{Person: "person-hollnagel-michael-david"}},
	}
	for _, c := range cases {
		got, err := findEvidenceSubject(archive, c.query)
		if err != nil {
			t.Errorf("findEvidenceSubject(%q): %v", c.query, err)

			continue
		}
		if got != c.want {
			t.Errorf("findEvidenceSubject(%q) = %+v, want %+v", c.query, got, c.want)
		}
	}
}

func TestFindEvidenceSubject_UnknownQueryNamesWhatItAccepts(t *testing.T) {
	_, err := findEvidenceSubject(parishArchive(), "nothing-at-all")
	if err == nil {
		t.Fatal("expected an error for an unresolvable subject, got nil")
	}
	if !errors.Is(err, ErrEvidenceNoSubject) {
		t.Errorf("error = %v, want it to wrap ErrEvidenceNoSubject", err)
	}
	// The old message ("no person found matching ...") read as "that person does
	// not exist"; the new one has to say the command takes more than a person.
	for _, want := range []string{"event", "place", "relationship"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err.Error(), want)
		}
	}
}

func TestFindEvidenceSubject_AmbiguousNameKeepsPersonListing(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"person-a": {Properties: map[string]any{"name": map[string]any{"value": "Jane Webb"}}},
			"person-b": {Properties: map[string]any{"name": map[string]any{"value": "Jane Miller"}}},
		},
	}

	_, err := findEvidenceSubject(archive, "Jane")
	if err == nil {
		t.Fatal("expected a disambiguation error, got nil")
	}
	if errors.Is(err, ErrEvidenceNoSubject) {
		t.Errorf("ambiguous name reported as no-subject: %v", err)
	}
	for _, want := range []string{"person-a", "person-b"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("disambiguation listing %q missing %q", err.Error(), want)
		}
	}
}

func TestFindEvidenceSubject_IDSharedAcrossTypesIsReported(t *testing.T) {
	// Entity IDs are unique archive-wide, but a hand-edited archive can break
	// that; resolving by map order would silently pick one.
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"shared-id": {}},
		Events:  map[string]*glxlib.Event{"shared-id": {Type: glxlib.EventTypeBirth}},
	}

	_, err := findEvidenceSubject(archive, "shared-id")
	if err == nil {
		t.Fatal("expected an ambiguity error for an ID used by two entity types")
	}
	for _, want := range []string{"person", "event"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name the %s candidate", err.Error(), want)
		}
	}
}













func TestPrintEvidenceText_EventSubjectHeader(t *testing.T) {
	report := mustCollectEvidence(parishArchive(), glxlib.EntityRef{Event: "event-death-1807"}, "place")
	streams, out, _ := TestIOStreams()

	printEvidenceText(streams, &report)

	want := "Evidence for place of Death of Michael David Hollnagel (event-death-1807):"
	if !strings.Contains(out.String(), want) {
		t.Errorf("text output missing %q\n--- got ---\n%s", want, out.String())
	}
}

func TestEvidenceJSON_SubjectFieldsAndPersonAliases(t *testing.T) {
	decode := func(t *testing.T, report EvidenceReport) map[string]any {
		t.Helper()
		streams, out, _ := TestIOStreams()
		if err := printEvidenceJSON(streams, &report); err != nil {
			t.Fatalf("printEvidenceJSON: %v", err)
		}
		var raw map[string]any
		if err := json.Unmarshal(out.Bytes(), &raw); err != nil {
			t.Fatalf("output is not valid JSON: %v", err)
		}

		return raw
	}

	t.Run("person subject keeps the pre-#1268 keys", func(t *testing.T) {
		raw := decode(t, mustCollectEvidence(brickwallArchive(), glxlib.EntityRef{Person: "person-jane-webb"}, "born_at"))
		for key, want := range map[string]string{
			"subject":      "person-jane-webb",
			"subject_type": "person",
			"subject_name": "Jane Miller",
			"person":       "person-jane-webb",
			"person_name":  "Jane Miller",
		} {
			if raw[key] != want {
				t.Errorf("%s = %v, want %q", key, raw[key], want)
			}
		}
	})

	t.Run("event subject omits them", func(t *testing.T) {
		raw := decode(t, mustCollectEvidence(parishArchive(), glxlib.EntityRef{Event: "event-death-1807"}, "place"))
		if raw["subject"] != "event-death-1807" || raw["subject_type"] != "event" {
			t.Errorf("subject = %v/%v, want event-death-1807/event", raw["subject"], raw["subject_type"])
		}
		if _, ok := raw["person"]; ok {
			t.Errorf("person key present for an event subject: %v", raw["person"])
		}
		if _, ok := raw["person_name"]; ok {
			t.Errorf("person_name key present for an event subject: %v", raw["person_name"])
		}
	})
}

func TestShowEvidence_EventSubjectEndToEnd(t *testing.T) {
	path := writeTempArchive(t, parishArchive())
	streams, out, _ := TestIOStreams()

	if err := showEvidence(streams, path, "event-birth-est", "date", "text"); err != nil {
		t.Fatalf("showEvidence: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"Evidence for date of Birth (event-birth-est):",
		"2 reports across 2 values",
		"ABT 1737",
		"ABT 1742",
		"Liepen Death Register 1807",
		"Best evidence: inconclusive",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("text output missing %q\n--- got ---\n%s", want, got)
		}
	}
}
