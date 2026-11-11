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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func importGEDCOMString(t *testing.T, ged string, opts ImportOptions) (*GLXFile, *ImportResult) {
	t.Helper()
	glxFile, result, err := ImportGEDCOMWithOptions(strings.NewReader(ged), nil, opts)
	require.NoError(t, err)

	return glxFile, result
}

func singlePerson(t *testing.T, glxFile *GLXFile) *Person {
	t.Helper()
	require.Len(t, glxFile.Persons, 1)
	for _, person := range glxFile.Persons {
		return person
	}

	return nil
}

func eventsOfType(glxFile *GLXFile, eventType string) []*Event {
	var events []*Event
	for _, event := range glxFile.Events {
		if event.Type == eventType {
			events = append(events, event)
		}
	}

	return events
}

func TestDetectGEDCOMPlatform(t *testing.T) {
	tests := []struct {
		name string
		head GEDCOMHeaderSource
		want string
	}{
		{"ancestry member tree", GEDCOMHeaderSource{SystemID: "Ancestry.com Family Trees", Name: "Ancestry.com Member Trees", Corporation: "Ancestry.com"}, "ancestry"},
		{"ancestry upper-case id", GEDCOMHeaderSource{SystemID: "ANCESTRY"}, "ancestry"},
		{"ftm by id, ancestry corp", GEDCOMHeaderSource{SystemID: "FTM", Name: "Family Tree Maker for Windows", Corporation: "Ancestry.com"}, "ftm"},
		{"ftm by name", GEDCOMHeaderSource{SystemID: "FTW", Name: "Family Tree Maker"}, "ftm"},
		{"rootsmagic", GEDCOMHeaderSource{SystemID: "RootsMagic", Name: "RootsMagic"}, "rootsmagic"},
		{"legacy", GEDCOMHeaderSource{SystemID: "Legacy", Name: "Legacy (R)"}, "legacy"},
		{"myheritage", GEDCOMHeaderSource{SystemID: "MYHERITAGE", Name: "MyHeritage Family Tree Builder"}, "myheritage"},
		{"findmypast", GEDCOMHeaderSource{SystemID: "FINDMYPAST"}, "findmypast"},
		{"corporation alone does not select ancestry", GEDCOMHeaderSource{SystemID: "OTHER", Corporation: "Ancestry.com"}, GEDCOMPlatformGeneric},
		{"unknown", GEDCOMHeaderSource{SystemID: "GRAMPS", Name: "Gramps"}, GEDCOMPlatformGeneric},
		{"empty", GEDCOMHeaderSource{}, GEDCOMPlatformGeneric},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, DetectGEDCOMPlatform(tt.head).Name())
		})
	}
}

func TestLookupGEDCOMPlatform(t *testing.T) {
	adapter, err := LookupGEDCOMPlatform("Ancestry")
	require.NoError(t, err)
	assert.Equal(t, "ancestry", adapter.Name())

	adapter, err = LookupGEDCOMPlatform(GEDCOMPlatformGeneric)
	require.NoError(t, err)
	assert.Equal(t, GEDCOMPlatformGeneric, adapter.Name())

	_, err = LookupGEDCOMPlatform("geni")
	require.ErrorIs(t, err, ErrUnknownGEDCOMPlatform)
	assert.Contains(t, err.Error(), "ancestry")
}

func TestGEDCOMPlatforms_ListsEveryAdapterOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, adapter := range GEDCOMPlatforms() {
		assert.False(t, seen[adapter.Name()], "duplicate adapter %q", adapter.Name())
		seen[adapter.Name()] = true
		assert.NotEmpty(t, adapter.Label())
	}
	for _, name := range []string{"ancestry", "ftm", "rootsmagic", "legacy", "myheritage", "findmypast", GEDCOMPlatformGeneric} {
		assert.True(t, seen[name], "missing adapter %q", name)
	}
}

const ancestryGEDCOM = `0 HEAD
1 SOUR Ancestry.com Family Trees
2 VERS (2010.3)
2 NAME Ancestry.com Member Trees
2 CORP Ancestry.com
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
0 @I1@ INDI
1 NAME John /Smith/
1 SEX M
1 BIRT
2 DATE 4 Jul 1890
1 DEAT
2 DATE 12 Mar 1952
1 _DCAUSE Pneumonia
1 _MILT US Army, World War I
2 DATE FROM 1917 TO 1919
2 PLAC France
2 SOUR @S1@
3 PAGE Roll 12
3 _APID 1,6482::4512345
1 NOTE <p>Served in the <b>AEF</b> &amp; returned home.</p><p>Second paragraph</
2 CONC p>
1 _UID 4F2A0B9C1D2E3F405162738495A6B7C8D9E0
0 @S1@ SOUR
1 TITL U.S., World War I Draft Registration Cards, 1917-1918
1 _APID 1,6482::0
0 TRLR
`

func TestAncestryAdapter_Import(t *testing.T) {
	glxFile, result := importGEDCOMString(t, ancestryGEDCOM, ImportOptions{})

	assert.Equal(t, "ancestry", result.Platform)
	assert.Equal(t, "Ancestry", result.PlatformLabel)
	assert.Equal(t, GEDCOMAdapterRewrites{"_MILT": 1, "_DCAUSE": 1, "_APID": 2, "_UID": 1, "NOTE (HTML)": 1}, result.PlatformRewrites)

	person := singlePerson(t, glxFile)

	t.Run("_MILT becomes a military_service event keeping its details", func(t *testing.T) {
		military := eventsOfType(glxFile, "military_service")
		require.Len(t, military, 1)
		event := military[0]
		assert.Equal(t, "US Army, World War I", event.Properties["event_subtype"])
		assert.Equal(t, DateString("FROM 1917 TO 1919"), event.Date)
		assert.NotEmpty(t, event.PlaceID)
		citations, _ := event.Properties[PropertyCitations].([]string)
		require.Len(t, citations, 1)

		citation := glxFile.Citations[citations[0]]
		require.NotNil(t, citation)
		assert.Equal(t, "https://www.ancestry.com/discoveryui-content/view/4512345:6482", citation.Properties[CitationPropertyURL])
		assert.Equal(t, []any{map[string]any{"value": "1,6482::4512345", "fields": map[string]any{"type": AncestryAPIDTypeURI}}},
			citation.Properties["external_ids"])
		assert.Equal(t, "Roll 12", citation.Properties["locator"])
	})

	t.Run("_DCAUSE becomes the death event's cause", func(t *testing.T) {
		deaths := eventsOfType(glxFile, EventTypeDeath)
		require.Len(t, deaths, 1)
		assert.Equal(t, "Pneumonia", deaths[0].Properties["cause"])
	})

	t.Run("source _APID becomes an external ID without a record link", func(t *testing.T) {
		require.Len(t, glxFile.Sources, 1)
		for _, source := range glxFile.Sources {
			assert.Equal(t, []any{map[string]any{"value": "1,6482::0", "fields": map[string]any{"type": AncestryAPIDTypeURI}}},
				source.Properties["external_ids"])
			assert.NotContains(t, source.Properties, CitationPropertyURL)
		}
	})

	t.Run("_UID becomes an external ID", func(t *testing.T) {
		assert.Equal(t, []any{map[string]any{"value": "4F2A0B9C1D2E3F405162738495A6B7C8D9E0", "fields": map[string]any{"type": GEDCOMUIDTypeURI}}},
			person.Properties["external_ids"])
	})

	t.Run("note HTML is stripped", func(t *testing.T) {
		require.Len(t, person.Notes, 1)
		assert.Equal(t, "Served in the AEF & returned home.\nSecond paragraph", person.Notes[0])
	})

	t.Run("nothing mapped is left as an extension", func(t *testing.T) {
		assert.NotContains(t, person.Properties, PropertyGEDCOMExtensions)
	})
}

func TestAncestryTags_WithoutAdapterArePreservedNotMapped(t *testing.T) {
	glxFile, result := importGEDCOMString(t, ancestryGEDCOM, ImportOptions{Platform: GEDCOMPlatformGeneric})

	assert.Equal(t, GEDCOMPlatformGeneric, result.Platform)
	assert.Empty(t, result.PlatformRewrites)
	assert.Empty(t, eventsOfType(glxFile, "military_service"))

	person := singlePerson(t, glxFile)
	extensions, _ := person.Properties[PropertyGEDCOMExtensions].([]any)
	tags := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		tags = append(tags, ext.(map[string]any)[ExtensionEntryTag].(string))
	}
	assert.ElementsMatch(t, []string{"_DCAUSE", "_MILT", "_UID"}, tags)
}

func TestForcedPlatform_OverridesDetection(t *testing.T) {
	ged := strings.Replace(ancestryGEDCOM, "1 SOUR Ancestry.com Family Trees\n2 VERS (2010.3)\n2 NAME Ancestry.com Member Trees\n2 CORP Ancestry.com\n", "1 SOUR GRAMPS\n", 1)

	_, result := importGEDCOMString(t, ged, ImportOptions{})
	assert.Equal(t, GEDCOMPlatformGeneric, result.Platform)

	glxFile, result := importGEDCOMString(t, ged, ImportOptions{Platform: "ancestry"})
	assert.Equal(t, "ancestry", result.Platform)
	assert.Len(t, eventsOfType(glxFile, "military_service"), 1)

	_, _, err := ImportGEDCOMWithOptions(strings.NewReader(ged), nil, ImportOptions{Platform: "nope"})
	require.ErrorIs(t, err, ErrUnknownGEDCOMPlatform)
}

func TestRewriteDeathCause(t *testing.T) {
	t.Run("creates DEAT when absent", func(t *testing.T) {
		indi := &GEDCOMRecord{Tag: GedcomTagIndi, SubRecords: []*GEDCOMRecord{
			{Tag: "_DCAUSE", Value: "Tuberculosis"},
		}}
		rewrites := GEDCOMAdapterRewrites{}
		rewriteDeathCause(indi, rewrites)
		require.Len(t, indi.SubRecords, 1)
		assert.Equal(t, GedcomTagDeat, indi.SubRecords[0].Tag)
		assert.Equal(t, []*GEDCOMRecord{{Tag: GedcomTagCaus, Value: "Tuberculosis"}}, indi.SubRecords[0].SubRecords)
		assert.Equal(t, 1, rewrites["_DCAUSE"])
	})

	t.Run("keeps an existing CAUS and preserves the _DCAUSE", func(t *testing.T) {
		indi := &GEDCOMRecord{Tag: GedcomTagIndi, SubRecords: []*GEDCOMRecord{
			{Tag: GedcomTagDeat, SubRecords: []*GEDCOMRecord{{Tag: GedcomTagCaus, Value: "Heart failure"}}},
			{Tag: "_DCAUSE", Value: "Pneumonia"},
		}}
		rewrites := GEDCOMAdapterRewrites{}
		rewriteDeathCause(indi, rewrites)
		require.Len(t, indi.SubRecords, 2)
		assert.Equal(t, "_DCAUSE", indi.SubRecords[1].Tag)
		assert.Empty(t, rewrites)
	})

	t.Run("leaves a _DCAUSE with citations alone", func(t *testing.T) {
		indi := &GEDCOMRecord{Tag: GedcomTagIndi, SubRecords: []*GEDCOMRecord{
			{Tag: "_DCAUSE", Value: "Pneumonia", SubRecords: []*GEDCOMRecord{{Tag: GedcomTagSour, Value: "@S1@"}}},
		}}
		rewrites := GEDCOMAdapterRewrites{}
		rewriteDeathCause(indi, rewrites)
		assert.Equal(t, "_DCAUSE", indi.SubRecords[0].Tag)
		assert.Empty(t, rewrites)
	})
}

func TestRewriteMilitaryService_WithoutValue(t *testing.T) {
	indi := &GEDCOMRecord{Tag: GedcomTagIndi, SubRecords: []*GEDCOMRecord{
		{Tag: "_MILT", SubRecords: []*GEDCOMRecord{{Tag: GedcomTagDate, Value: "1944"}}},
	}}
	rewriteMilitaryService(indi, GEDCOMAdapterRewrites{})
	milt := indi.SubRecords[0]
	assert.Equal(t, GedcomTagEven, milt.Tag)
	assert.Equal(t, []*GEDCOMRecord{{Tag: GedcomTagType, Value: "Military Service"}, {Tag: GedcomTagDate, Value: "1944"}}, milt.SubRecords)
}

func TestAncestryRecordURL(t *testing.T) {
	tests := map[string]string{
		"1,7602::12345678":   "https://www.ancestry.com/discoveryui-content/view/12345678:7602",
		"1,7602::12345678\t": "https://www.ancestry.com/discoveryui-content/view/12345678:7602",
		"1,7602::0":          "",
		"1,7602::":           "",
		"7602::12345678":     "",
		"1,abc::123":         "",
		"":                   "",
	}
	for apid, want := range tests {
		assert.Equal(t, want, AncestryRecordURL(apid), apid)
	}
}

func TestStripNoteHTML(t *testing.T) {
	tests := []struct {
		name, in, want string
		changed        bool
	}{
		{"plain text untouched", "Born at home & christened", "Born at home & christened", false},
		{"angle brackets that are not markup untouched", "Father <unknown>, mother &amp; aunt", "Father <unknown>, mother &amp; aunt", false},
		{"breaks become newlines", "Line one<br>Line two<br/>Line three", "Line one\nLine two\nLine three", true},
		{"paragraphs and entities", "<p>A &amp; B</p><p>C&nbsp;D</p>", "A & B\nC D", true},
		{"links keep their text", `See <a href="https://example.com">the record</a>.`, "See the record.", true},
		{"blank lines collapse", "<p>A</p><p></p><p></p><p>B</p>", "A\n\nB", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := stripNoteHTML(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.changed, changed)
		})
	}
}

func TestUIDAdapter_OnlyRewritesRecordsThatKeepExternalIDs(t *testing.T) {
	fam := &GEDCOMRecord{Tag: GedcomTagFam, SubRecords: []*GEDCOMRecord{{Tag: "_UID", Value: "ABC"}}}
	rewrites := GEDCOMAdapterRewrites{}
	rewriteUIDs(fam, rewrites)
	assert.Equal(t, "_UID", fam.SubRecords[0].Tag)
	assert.Empty(t, rewrites)
}

func TestRootsMagicAdapter_Import(t *testing.T) {
	ged := `0 HEAD
1 SOUR RootsMagic
2 NAME RootsMagic
2 VERS 9.1.3.0
1 GEDC
2 VERS 5.5.1
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Mary /Jones/
1 _UID 9B5F3C2A1E0D4B7A8C6F5E4D3C2B1A09A1B2
1 NOTE <p>kept as written</p>
0 TRLR
`
	glxFile, result := importGEDCOMString(t, ged, ImportOptions{})
	assert.Equal(t, "rootsmagic", result.Platform)
	assert.Equal(t, GEDCOMAdapterRewrites{"_UID": 1}, result.PlatformRewrites)

	person := singlePerson(t, glxFile)
	assert.Equal(t, []any{map[string]any{"value": "9B5F3C2A1E0D4B7A8C6F5E4D3C2B1A09A1B2", "fields": map[string]any{"type": GEDCOMUIDTypeURI}}},
		person.Properties["external_ids"])
	assert.Equal(t, NoteList{"<p>kept as written</p>"}, person.Notes, "RootsMagic notes are not HTML-stripped")
}

func TestGenericImport_ReportsGenericPlatform(t *testing.T) {
	ged := "0 HEAD\n1 SOUR GRAMPS\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n1 NAME A /B/\n0 TRLR\n"
	_, result := importGEDCOMString(t, ged, ImportOptions{})
	assert.Equal(t, GEDCOMPlatformGeneric, result.Platform)
	assert.Empty(t, result.PlatformRewrites)
}
