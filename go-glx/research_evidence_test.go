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
	"testing"
)

// brickwallArchive builds the issue #144 scenario, condensed: a person whose
// born_at property is reported as three different states across six census
// citations, with mixed confidence inside the leading group.
func brickwallArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-jane-webb": {Properties: map[string]any{"name": map[string]any{"value": "Jane Miller"}}},
		},
		Sources: map[string]*Source{
			"src-1860": {Title: "1860 US Census"},
			"src-1880": {Title: "1880 US Census"},
			"src-1900": {Title: "1900 US Census"},
			"src-1905": {Title: "1905 WI Census"},
			"src-1910": {Title: "1910 US Census"},
			"src-1920": {Title: "1920 US Census"},
		},
		Citations: map[string]*Citation{
			"cit-1860-webb":    {SourceID: "src-1860"},
			"cit-1880-clara":   {SourceID: "src-1880"},
			"cit-1905-anna":    {SourceID: "src-1905"},
			"cit-1910-clara":   {SourceID: "src-1910"},
			"cit-1900-william": {SourceID: "src-1900"},
			"cit-1920-william": {SourceID: "src-1920"},
		},
		Assertions: map[string]*Assertion{
			"a-va-1": birthplaceAssertion("VIRGINIA", "medium", "cit-1880-clara"),
			"a-va-2": birthplaceAssertion("VIRGINIA", "high", "cit-1905-anna"),
			"a-va-3": birthplaceAssertion("VIRGINIA", "medium", "cit-1910-clara"),
			"a-wi-1": birthplaceAssertion("WISCONSIN", "low", "cit-1900-william"),
			"a-wi-2": birthplaceAssertion("WISCONSIN", "medium", "cit-1920-william"),
			"a-fl-1": birthplaceAssertion("FLORIDA", "low", "cit-1860-webb"),
		},
	}
}

func birthplaceAssertion(value, confidence string, citations ...string) *Assertion {
	return &Assertion{
		Subject:    EntityRef{Person: "person-jane-webb"},
		Property:   "born_at",
		Value:      value,
		Confidence: confidence,
		Citations:  citations,
	}
}

func TestCollectEvidence_GroupsRankAndCount(t *testing.T) {
	report := collectEvidence(brickwallArchive(), EntityRef{Person: "person-jane-webb"}, "born_at")

	if report.PersonName != "Jane Miller" {
		t.Errorf("PersonName = %q, want Jane Miller", report.PersonName)
	}
	if report.TotalReports != 6 {
		t.Errorf("TotalReports = %d, want 6", report.TotalReports)
	}
	if len(report.Groups) != 3 {
		t.Fatalf("len(Groups) = %d, want 3", len(report.Groups))
	}
	if report.BestEvidence != "VIRGINIA" {
		t.Errorf("BestEvidence = %q, want VIRGINIA", report.BestEvidence)
	}

	want := []struct {
		value      string
		reports    int
		confidence string
	}{
		{"VIRGINIA", 3, "high"}, // best confidence is the highest in the group
		{"WISCONSIN", 2, "medium"},
		{"FLORIDA", 1, "low"},
	}
	for i, w := range want {
		g := report.Groups[i]
		if g.Value != w.value || g.Reports != w.reports || g.BestConfidence != w.confidence {
			t.Errorf("Groups[%d] = {%q, %d, %q}, want {%q, %d, %q}",
				i, g.Value, g.Reports, g.BestConfidence, w.value, w.reports, w.confidence)
		}
	}
}

func TestCollectEvidence_ItemsResolveSourceTitleAndSort(t *testing.T) {
	report := collectEvidence(brickwallArchive(), EntityRef{Person: "person-jane-webb"}, "born_at")

	va := report.Groups[0]
	if va.Value != "VIRGINIA" {
		t.Fatalf("Groups[0].Value = %q, want VIRGINIA", va.Value)
	}
	// Items sorted by citation ID: cit-1880-clara, cit-1905-anna, cit-1910-clara.
	wantCits := []string{"cit-1880-clara", "cit-1905-anna", "cit-1910-clara"}
	wantSrc := []string{"1880 US Census", "1905 WI Census", "1910 US Census"}
	for i := range wantCits {
		if va.Items[i].CitationID != wantCits[i] {
			t.Errorf("Items[%d].CitationID = %q, want %q", i, va.Items[i].CitationID, wantCits[i])
		}
		if va.Items[i].Source != wantSrc[i] {
			t.Errorf("Items[%d].Source = %q, want %q", i, va.Items[i].Source, wantSrc[i])
		}
	}
}

func TestCollectEvidence_ConfidenceBreaksReportTie(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "low"},
			"a2": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "low"},
			"a3": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
			"a4": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "prop")

	// Both values have 2 reports; BETA's higher confidence wins and sorts first.
	if report.Groups[0].Value != "BETA" {
		t.Errorf("Groups[0].Value = %q, want BETA (higher confidence)", report.Groups[0].Value)
	}
	if report.BestEvidence != "BETA" {
		t.Errorf("BestEvidence = %q, want BETA", report.BestEvidence)
	}
}

func TestCollectEvidence_InconclusiveTie(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "medium"},
			"a2": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "prop")

	if report.BestEvidence != "" {
		t.Errorf("BestEvidence = %q, want \"\" (tie on reports and confidence)", report.BestEvidence)
	}
}

func TestCollectEvidence_DedupsSameCitationForSameValue(t *testing.T) {
	archive := &GLXFile{
		Persons:   map[string]*Person{"p": {}},
		Sources:   map[string]*Source{"s": {Title: "Shared Source"}},
		Citations: map[string]*Citation{"c": {SourceID: "s"}},
		Assertions: map[string]*Assertion{
			// Two assertions cite the same record for the same value, with
			// different confidence. The report is counted once, but it keeps the
			// strongest confidence (high) — not whichever assertion was seen first.
			"a1": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "X", Confidence: "low", Citations: []string{"c"}},
			"a2": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "X", Confidence: "high", Citations: []string{"c"}},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "prop")

	if report.TotalReports != 1 {
		t.Errorf("TotalReports = %d, want 1 (same citation counted once)", report.TotalReports)
	}
	if len(report.Groups) != 1 {
		t.Fatalf("len(Groups) = %d, want 1", len(report.Groups))
	}
	g := report.Groups[0]
	if g.BestConfidence != "high" {
		t.Errorf("BestConfidence = %q, want high (strongest across dedup'd assertions)", g.BestConfidence)
	}
	if len(g.Items) != 1 || g.Items[0].Confidence != "high" {
		t.Errorf("Items = %+v, want a single item with confidence high", g.Items)
	}
}

func TestCollectEvidence_ValueResolution(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"p":            {},
			"person-clara": {Properties: map[string]any{"name": map[string]any{"value": "Clara Webb"}}},
		},
		Places: map[string]*Place{
			"place-richmond": {Name: "Richmond, Virginia"},
		},
		PersonProperties: map[string]*PropertyDefinition{
			"named_for": {Label: "Named For", ReferenceType: EntityTypePersons.String()},
			"residence": {ReferenceType: EntityTypePlaces.String()},
		},
		Assertions: map[string]*Assertion{
			// residence resolves through its place-reference definition.
			"a1": {Subject: EntityRef{Person: "p"}, Property: "residence", Value: "place-richmond", Confidence: "high"},
			// named_for resolves via the archive's PropertyDefinition (persons).
			"a2": {Subject: EntityRef{Person: "p"}, Property: "named_for", Value: "person-clara", Confidence: "high"},
			// occupation is free text — passes through unchanged.
			"a3": {Subject: EntityRef{Person: "p"}, Property: "occupation", Value: "Farmer", Confidence: "high"},
		},
	}

	cases := map[string]string{
		"residence":  "Richmond, Virginia",
		"named_for":  "Clara Webb",
		"occupation": "Farmer",
	}
	for property, wantValue := range cases {
		report := collectEvidence(archive, EntityRef{Person: "p"}, property)
		if len(report.Groups) != 1 {
			t.Errorf("%s: len(Groups) = %d, want 1", property, len(report.Groups))

			continue
		}
		if report.Groups[0].Value != wantValue {
			t.Errorf("%s: Value = %q, want %q", property, report.Groups[0].Value, wantValue)
		}
	}
}

func TestCollectEvidence_DirectSourceAndBareAssertion(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Sources: map[string]*Source{"src-deed": {Title: "1842 Deed Book"}},
		Assertions: map[string]*Assertion{
			// Direct source, no citation.
			"a1": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "SOURCED", Confidence: "medium", Sources: []string{"src-deed"}},
			// Neither citation nor source.
			"a2": {Subject: EntityRef{Person: "p"}, Property: "prop", Value: "BARE", Confidence: "low"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "prop")
	byValue := map[string]EvidenceGroup{}
	for _, g := range report.Groups {
		byValue[g.Value] = g
	}

	sourced := byValue["SOURCED"]
	if len(sourced.Items) != 1 || sourced.Items[0].Source != "1842 Deed Book" || sourced.Items[0].CitationID != "" {
		t.Errorf("SOURCED item = %+v, want source title with empty citation", sourced.Items)
	}

	bare := byValue["BARE"]
	if len(bare.Items) != 1 || bare.Items[0].Source != "(no citation)" {
		t.Errorf("BARE item = %+v, want \"(no citation)\"", bare.Items)
	}
}

func TestCollectEvidence_ExactPropertyMatchWins(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Person: "p"}, Property: "born_at", Value: "EXACT", Confidence: "high"},
			"a2": {Subject: EntityRef{Person: "p"}, Property: "Born_At", Value: "FUZZY", Confidence: "high"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "born_at")
	if len(report.Groups) != 1 || report.Groups[0].Value != "EXACT" {
		t.Errorf("Groups = %+v, want only the exact-match value EXACT", report.Groups)
	}
}

func TestCollectEvidence_CaseInsensitiveFallback(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Person: "p"}, Property: "Born_At", Value: "FUZZY", Confidence: "high"},
		},
	}

	// No exact "born_at" exists, so the case-insensitive match is used.
	report := collectEvidence(archive, EntityRef{Person: "p"}, "born_at")
	if len(report.Groups) != 1 || report.Groups[0].Value != "FUZZY" {
		t.Errorf("Groups = %+v, want case-insensitive fallback to FUZZY", report.Groups)
	}
	// Property reflects the canonical key stored on the assertion, not the query.
	if report.Property != "Born_At" {
		t.Errorf("Property = %q, want canonical key Born_At", report.Property)
	}
}

func TestCollectEvidence_CaseInsensitiveResolvesReferences(t *testing.T) {
	archive := &GLXFile{
		Persons:          map[string]*Person{"p": {}},
		Places:           map[string]*Place{"place-richmond": {Name: "Richmond, Virginia"}},
		PersonProperties: map[string]*PropertyDefinition{"residence": {ReferenceType: EntityTypePlaces.String()}},
		Assertions: map[string]*Assertion{
			// Stored property is "residence" (a place reference); the query uses
			// different casing and matches via the case-insensitive fallback.
			// Resolution must use the assertion's own property key, so the place
			// still resolves to its name.
			"a1": {Subject: EntityRef{Person: "p"}, Property: "residence", Value: "place-richmond", Confidence: "high"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "RESIDENCE")
	if len(report.Groups) != 1 || report.Groups[0].Value != "Richmond, Virginia" {
		t.Errorf("Groups = %+v, want value resolved to place name despite query casing", report.Groups)
	}
	// JSON/text consumers see the canonical "residence", not the "RESIDENCE" query.
	if report.Property != "residence" {
		t.Errorf("Property = %q, want canonical key residence", report.Property)
	}
}

func TestCollectEvidence_NoMatchingAssertions(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{"p": {Properties: map[string]any{"name": map[string]any{"value": "Pat"}}}},
		Assertions: map[string]*Assertion{
			"a1": {Subject: EntityRef{Person: "p"}, Property: "occupation", Value: "Farmer", Confidence: "high"},
		},
	}

	report := collectEvidence(archive, EntityRef{Person: "p"}, "born_at")
	if report.TotalReports != 0 || len(report.Groups) != 0 {
		t.Errorf("empty result expected, got TotalReports=%d Groups=%d", report.TotalReports, len(report.Groups))
	}
	if report.BestEvidence != "" {
		t.Errorf("BestEvidence = %q, want empty", report.BestEvidence)
	}
	// With no matches there is no canonical key, so the query is echoed back.
	if report.Property != "born_at" {
		t.Errorf("Property = %q, want the query echoed (born_at)", report.Property)
	}
}

// TestCitationSourceLabel_FallsBackToCitationID pins the citation→source-title
// resolution contract: when the citation's source is missing, untitled, or
// the SourceID is empty, the label falls back to the citation ID rather than
// the source ID (which would duplicate identifier noise next to the citation
// column).
func TestCitationSourceLabel_FallsBackToCitationID(t *testing.T) {
	archive := &GLXFile{
		Sources: map[string]*Source{
			"src-titled":   {Title: "1880 US Census"},
			"src-untitled": {},
		},
		Citations: map[string]*Citation{
			"cit-with-title":   {SourceID: "src-titled"},
			"cit-untitled-src": {SourceID: "src-untitled"},
			"cit-missing-src":  {SourceID: "src-does-not-exist"},
			"cit-no-source-id": {},
		},
	}

	cases := map[string]string{
		"cit-with-title":   "1880 US Census",   // happy path: title resolves
		"cit-untitled-src": "cit-untitled-src", // source exists but no title → citation ID
		"cit-missing-src":  "cit-missing-src",  // source ID set but unknown → citation ID
		"cit-no-source-id": "cit-no-source-id", // citation has no SourceID → citation ID
		"cit-unknown":      "cit-unknown",      // citation itself missing → echo input
	}
	for citID, want := range cases {
		got := citationSourceLabel(citID, archive)
		if got != want {
			t.Errorf("citationSourceLabel(%q) = %q, want %q", citID, got, want)
		}
	}
}

// parishArchive is the issue #1268 scenario: a single person whose contested
// date and place live on their birth and death events, the canonical
// evidence-first modeling, with nothing asserted on the person at all.
func parishArchive() *GLXFile {
	return &GLXFile{
		Persons: map[string]*Person{
			"person-hollnagel-michael-david": {
				Properties: map[string]any{"name": map[string]any{"value": "Michael David Hollnagel"}},
			},
		},
		Events: map[string]*Event{
			"event-birth-est":   {Type: EventTypeBirth},
			"event-death-1807":  {Title: "Death of Michael David Hollnagel", Type: EventTypeDeath},
			"event-burial-1807": {Type: EventTypeBurial},
		},
		Places: map[string]*Place{
			"place-liepen": {Name: "Liepen, Vorpommern"},
			"place-anklam": {Name: "Anklam, Vorpommern"},
		},
		Relationships: map[string]*Relationship{
			"relationship-marriage-hollnagel": {Type: RelationshipTypeMarriage},
		},
		Sources: map[string]*Source{
			"src-death":  {Title: "Liepen Death Register 1807"},
			"src-burial": {Title: "Liepen Burial Register 1807"},
		},
		Citations: map[string]*Citation{
			"cit-death-entry":  {SourceID: "src-death"},
			"cit-burial-entry": {SourceID: "src-burial"},
		},
		Assertions: map[string]*Assertion{
			// The birth date is an inference from the death entry's decadal age
			// band, reported differently by the two registers.
			"a-birth-1": {
				Subject: EntityRef{Event: "event-birth-est"}, Property: "date",
				Value: "ABT 1737", Confidence: "low", Citations: []string{"cit-death-entry"},
			},
			"a-birth-2": {
				Subject: EntityRef{Event: "event-birth-est"}, Property: "date",
				Value: "ABT 1742", Confidence: "low", Citations: []string{"cit-burial-entry"},
			},
			// The death place is a place reference on an event subject.
			"a-death-place": {
				Subject: EntityRef{Event: "event-death-1807"}, Property: "place",
				Value: "place-liepen", Confidence: "high", Citations: []string{"cit-death-entry"},
			},
			"a-burial-place": {
				Subject: EntityRef{Event: "event-burial-1807"}, Property: "place",
				Value: "place-anklam", Confidence: "medium", Citations: []string{"cit-burial-entry"},
			},
		},
	}
}

func TestSubjectLabel(t *testing.T) {
	archive := parishArchive()

	cases := []struct {
		subject EntityRef
		want    string
	}{
		{EntityRef{Person: "person-hollnagel-michael-david"}, "Michael David Hollnagel"},
		{EntityRef{Event: "event-death-1807"}, "Death of Michael David Hollnagel"},
		// No title: fall back to a label generated from the event type.
		{EntityRef{Event: "event-birth-est"}, "Birth"},
		{EntityRef{Place: "place-liepen"}, "Liepen, Vorpommern"},
		{EntityRef{Relationship: "relationship-marriage-hollnagel"}, RelationshipTypeMarriage},
		// Missing entities fall back to the ID.
		{EntityRef{Event: "event-missing"}, "event-missing"},
		{EntityRef{Place: "place-missing"}, "place-missing"},
		{EntityRef{Relationship: "relationship-missing"}, "relationship-missing"},
	}
	for _, c := range cases {
		if got := subjectLabel(archive, c.subject); got != c.want {
			t.Errorf("subjectLabel(%+v) = %q, want %q", c.subject, got, c.want)
		}
	}
}

func TestCollectEvidence_EventSubject(t *testing.T) {
	report := collectEvidence(parishArchive(), EntityRef{Event: "event-birth-est"}, "date")

	if report.Subject != "event-birth-est" || report.SubjectType != "event" {
		t.Errorf("subject = %q/%q, want event-birth-est/event", report.Subject, report.SubjectType)
	}
	if report.SubjectName != "Birth" {
		t.Errorf("SubjectName = %q, want Birth", report.SubjectName)
	}
	// Person/PersonName are the person-subject compatibility aliases only.
	if report.Person != "" || report.PersonName != "" {
		t.Errorf("Person/PersonName = %q/%q, want empty for an event subject", report.Person, report.PersonName)
	}
	if report.TotalReports != 2 || len(report.Groups) != 2 {
		t.Fatalf("TotalReports=%d Groups=%d, want 2 and 2", report.TotalReports, len(report.Groups))
	}
	// Two equally-supported low-confidence readings: exactly the unresolved
	// conflict the command exists to surface.
	if report.BestEvidence != "" {
		t.Errorf("BestEvidence = %q, want \"\" (tie between the two reckonings)", report.BestEvidence)
	}
}

func TestCollectEvidence_SubjectsDoNotLeakAcrossTypes(t *testing.T) {
	archive := parishArchive()

	// Every assertion in the fixture is event-subject, so the person has none.
	person := collectEvidence(archive, EntityRef{Person: "person-hollnagel-michael-david"}, "date")
	if person.TotalReports != 0 {
		t.Errorf("person date reports = %d, want 0 (all date assertions are event-subject)", person.TotalReports)
	}

	// And one event's assertions never appear under another's.
	death := collectEvidence(archive, EntityRef{Event: "event-death-1807"}, "place")
	if death.TotalReports != 1 || len(death.Groups) != 1 {
		t.Fatalf("death place: TotalReports=%d Groups=%d, want 1 and 1", death.TotalReports, len(death.Groups))
	}
	if death.Groups[0].Value != "Liepen, Vorpommern" {
		t.Errorf("death place = %q, want the burial event's place excluded and this one resolved",
			death.Groups[0].Value)
	}
}

func TestCollectEvidence_EventPlaceResolvesToPlaceName(t *testing.T) {
	// `place` on an event subject is the event's structural place field: no
	// vocabulary definition declares it a reference, so it needs its own rule.
	report := collectEvidence(parishArchive(), EntityRef{Event: "event-burial-1807"}, "place")
	if len(report.Groups) != 1 || report.Groups[0].Value != "Anklam, Vorpommern" {
		t.Errorf("Groups = %+v, want the place ID resolved to its name", report.Groups)
	}
}

func TestResolveAssertionValue_UsesTheSubjectTypesVocabulary(t *testing.T) {
	archive := &GLXFile{
		Places: map[string]*Place{"place-liepen": {Name: "Liepen, Vorpommern"}},
		// The same key means different things in the two vocabularies: a place
		// reference for relationships, free text for persons.
		PersonProperties: map[string]*PropertyDefinition{
			"settled": {Label: "Settled"},
		},
		RelationshipProperties: map[string]*PropertyDefinition{
			"settled": {Label: "Settled", ReferenceType: EntityTypePlaces.String()},
		},
	}

	rel := resolveAssertionValue("place-liepen", "settled", EntityRef{Relationship: "r"}, archive)
	if rel != "Liepen, Vorpommern" {
		t.Errorf("relationship subject = %q, want the relationship vocabulary's place reference resolved", rel)
	}

	person := resolveAssertionValue("place-liepen", "settled", EntityRef{Person: "p"}, archive)
	if person != "place-liepen" {
		t.Errorf("person subject = %q, want the raw value (person vocabulary declares no reference)", person)
	}

	// A place subject reads place_properties: a place asserted to sit inside
	// another place resolves that parent to its name.
	archive.Places["place-anklam"] = &Place{Name: "Anklam, Vorpommern"}
	archive.PlaceProperties = map[string]*PropertyDefinition{
		"administered_from": {Label: "Administered From", ReferenceType: EntityTypePlaces.String()},
	}
	place := resolveAssertionValue("place-anklam", "administered_from", EntityRef{Place: "place-liepen"}, archive)
	if place != "Anklam, Vorpommern" {
		t.Errorf("place subject = %q, want the place vocabulary's reference resolved", place)
	}

	// A subject with no field set has no vocabulary, and must not panic or
	// borrow another type's definitions.
	none := resolveAssertionValue("place-liepen", "settled", EntityRef{}, archive)
	if none != "place-liepen" {
		t.Errorf("empty subject = %q, want the raw value", none)
	}
}

func TestResolveAssertionValue_EventReferenceResolvesToTitle(t *testing.T) {
	archive := &GLXFile{
		Events: map[string]*Event{
			"event-marriage-1875": {Title: "Marriage of Michael and Anna", Type: EventTypeMarriage},
			"event-untitled":      {Type: EventTypeMarriage},
		},
		PersonProperties: map[string]*PropertyDefinition{
			"witnessed": {Label: "Witnessed", ReferenceType: EntityTypeEvents.String()},
		},
	}

	subject := EntityRef{Person: "p"}
	if got := resolveAssertionValue("event-marriage-1875", "witnessed", subject, archive); got != "Marriage of Michael and Anna" {
		t.Errorf("titled event = %q, want the event title", got)
	}
	// An untitled event has no better display form than its ID.
	if got := resolveAssertionValue("event-untitled", "witnessed", subject, archive); got != "event-untitled" {
		t.Errorf("untitled event = %q, want the raw ID", got)
	}
}

func TestSortEvidenceGroups_RawIdentityAndDate(t *testing.T) {
	groups := []EvidenceGroup{
		{Value: "John Smith", RawValue: "q2", Date: "1900", Reports: 1, BestConfidence: "high"},
		{Value: "John Smith", RawValue: "q1", Date: "1900", Reports: 1, BestConfidence: "high"},
		{Value: "John Smith", RawValue: "q1", Date: "1850", Reports: 1, BestConfidence: "high"},
	}
	sortEvidenceGroups(groups)
	if groups[0].RawValue != "q1" || groups[0].Date != "1850" || groups[1].RawValue != "q1" || groups[1].Date != "1900" || groups[2].RawValue != "q2" {
		t.Fatalf("unexpected tied group order: %#v", groups)
	}
}
