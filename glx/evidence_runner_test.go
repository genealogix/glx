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

func TestCollectEvidence_GroupsRankAndCount(t *testing.T) {
	report := collectEvidence(brickwallArchive(), "person-jane-webb", "born_at")

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
	report := collectEvidence(brickwallArchive(), "person-jane-webb", "born_at")

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
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "low"},
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "low"},
			"a3": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
			"a4": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
		},
	}

	report := collectEvidence(archive, "p", "prop")

	// Both values have 2 reports; BETA's higher confidence wins and sorts first.
	if report.Groups[0].Value != "BETA" {
		t.Errorf("Groups[0].Value = %q, want BETA (higher confidence)", report.Groups[0].Value)
	}
	if report.BestEvidence != "BETA" {
		t.Errorf("BestEvidence = %q, want BETA", report.BestEvidence)
	}
}

func TestCollectEvidence_InconclusiveTie(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "ALPHA", Confidence: "medium"},
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "BETA", Confidence: "medium"},
		},
	}

	report := collectEvidence(archive, "p", "prop")

	if report.BestEvidence != "" {
		t.Errorf("BestEvidence = %q, want \"\" (tie on reports and confidence)", report.BestEvidence)
	}
}

func TestCollectEvidence_DedupsSameCitationForSameValue(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons:   map[string]*glxlib.Person{"p": {}},
		Sources:   map[string]*glxlib.Source{"s": {Title: "Shared Source"}},
		Citations: map[string]*glxlib.Citation{"c": {SourceID: "s"}},
		Assertions: map[string]*glxlib.Assertion{
			// Two assertions cite the same record for the same value.
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "X", Confidence: "medium", Citations: []string{"c"}},
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "X", Confidence: "medium", Citations: []string{"c"}},
		},
	}

	report := collectEvidence(archive, "p", "prop")

	if report.TotalReports != 1 {
		t.Errorf("TotalReports = %d, want 1 (same citation counted once)", report.TotalReports)
	}
}

func TestCollectEvidence_ValueResolution(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{
			"p":            {},
			"person-clara": {Properties: map[string]any{"name": map[string]any{"value": "Clara Webb"}}},
		},
		Places: map[string]*glxlib.Place{
			"place-richmond": {Name: "Richmond, Virginia"},
		},
		PersonProperties: map[string]*glxlib.PropertyDefinition{
			"named_for": {Label: "Named For", ReferenceType: glxlib.EntityTypePersons},
		},
		Assertions: map[string]*glxlib.Assertion{
			// residence is a built-in place-ref property (placeRefProperties).
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "residence", Value: "place-richmond", Confidence: "high"},
			// named_for resolves via the archive's PropertyDefinition (persons).
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "named_for", Value: "person-clara", Confidence: "high"},
			// occupation is free text — passes through unchanged.
			"a3": {Subject: glxlib.EntityRef{Person: "p"}, Property: "occupation", Value: "Farmer", Confidence: "high"},
		},
	}

	cases := map[string]string{
		"residence":  "Richmond, Virginia",
		"named_for":  "Clara Webb",
		"occupation": "Farmer",
	}
	for property, wantValue := range cases {
		report := collectEvidence(archive, "p", property)
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
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Sources: map[string]*glxlib.Source{"src-deed": {Title: "1842 Deed Book"}},
		Assertions: map[string]*glxlib.Assertion{
			// Direct source, no citation.
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "SOURCED", Confidence: "medium", Sources: []string{"src-deed"}},
			// Neither citation nor source.
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "prop", Value: "BARE", Confidence: "low"},
		},
	}

	report := collectEvidence(archive, "p", "prop")
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
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "born_at", Value: "EXACT", Confidence: "high"},
			"a2": {Subject: glxlib.EntityRef{Person: "p"}, Property: "Born_At", Value: "FUZZY", Confidence: "high"},
		},
	}

	report := collectEvidence(archive, "p", "born_at")
	if len(report.Groups) != 1 || report.Groups[0].Value != "EXACT" {
		t.Errorf("Groups = %+v, want only the exact-match value EXACT", report.Groups)
	}
}

func TestCollectEvidence_CaseInsensitiveFallback(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "Born_At", Value: "FUZZY", Confidence: "high"},
		},
	}

	// No exact "born_at" exists, so the case-insensitive match is used.
	report := collectEvidence(archive, "p", "born_at")
	if len(report.Groups) != 1 || report.Groups[0].Value != "FUZZY" {
		t.Errorf("Groups = %+v, want case-insensitive fallback to FUZZY", report.Groups)
	}
}

func TestCollectEvidence_NoMatchingAssertions(t *testing.T) {
	archive := &glxlib.GLXFile{
		Persons: map[string]*glxlib.Person{"p": {Properties: map[string]any{"name": map[string]any{"value": "Pat"}}}},
		Assertions: map[string]*glxlib.Assertion{
			"a1": {Subject: glxlib.EntityRef{Person: "p"}, Property: "occupation", Value: "Farmer", Confidence: "high"},
		},
	}

	report := collectEvidence(archive, "p", "born_at")
	if report.TotalReports != 0 || len(report.Groups) != 0 {
		t.Errorf("empty result expected, got TotalReports=%d Groups=%d", report.TotalReports, len(report.Groups))
	}
	if report.BestEvidence != "" {
		t.Errorf("BestEvidence = %q, want empty", report.BestEvidence)
	}
}

func TestPrintEvidenceText_Output(t *testing.T) {
	report := collectEvidence(brickwallArchive(), "person-jane-webb", "born_at")
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
	printEvidenceText(streams, &EvidenceReport{Person: "p", PersonName: "Pat", Property: "born_at"})
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
	report := collectEvidence(brickwallArchive(), "person-jane-webb", "born_at")
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
