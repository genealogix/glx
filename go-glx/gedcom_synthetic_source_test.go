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

func TestSyntheticSourceDeduplication(t *testing.T) {
	sour := func(value string, subs ...*GEDCOMRecord) *GEDCOMRecord {
		return &GEDCOMRecord{Tag: GedcomTagSour, Value: value, SubRecords: subs}
	}
	text := func(value string) *GEDCOMRecord {
		return &GEDCOMRecord{Tag: GedcomTagText, Value: value}
	}
	note := func(value string) *GEDCOMRecord {
		return &GEDCOMRecord{Tag: GedcomTagNote, Value: value}
	}
	tests := []struct {
		name   string
		first  *GEDCOMRecord
		second *GEDCOMRecord
		same   bool
	}{
		{"empty placeholders", sour(""), sour(""), true},
		{"identical content", sour("Bible", text("Entry"), note("Copy")), sour("Bible", text("Entry"), note("Copy")), true},
		{"TEXT supplies title", sour("", text("Entry")), sour("", text("Entry")), true},
		{"continued title", sour("Family", &GEDCOMRecord{Tag: GedcomTagConc, Value: " Bible"}), sour("Family Bible"), true},
		{"continued description", sour("Bible", &GEDCOMRecord{Tag: GedcomTagText, Value: "Birth", SubRecords: []*GEDCOMRecord{{Tag: GedcomTagCont, Value: "entry"}}}), sour("Bible", text("Birth\nentry")), true},
		{"resolved notes", sour("Bible", note("@N1@")), sour("Bible", note("Copy")), true},
		{"different title", sour("Bible"), sour("Register"), false},
		{"case remains significant", sour("Bible"), sour("bible"), false},
		{"whitespace remains significant", sour("Bible"), sour("Bible "), false},
		{"different description", sour("Bible", text("Birth")), sour("Bible", text("Death")), false},
		{"different notes", sour("Bible", note("Original")), sour("Bible", note("Copy")), false},
		{"note order", sour("Bible", note("Original"), note("Copy")), sour("Bible", note("Copy"), note("Original")), false},
		{"note boundaries", sour("Bible", note("a\nb")), sour("Bible", note("a"), note("b")), false},
		{"embedded delimiter", sour("Bible", note("a\x00b")), sour("Bible", note("a"), note("b")), false},
		{"field boundaries", sour("a|b", text("c")), sour("a", text("b|c")), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := &ConversionContext{
				GLX:         &GLXFile{Sources: make(map[string]*Source)},
				Logger:      NewImportLogger(nil),
				SharedNotes: map[string]string{"@N1@": "Copy"},
			}
			firstID := createSyntheticSourceFromEmbeddedCitation(tt.first, conv)
			firstSource := conv.GLX.Sources[firstID]
			secondID := createSyntheticSourceFromEmbeddedCitation(tt.second, conv)
			wantCount := 2
			if tt.same {
				assert.Equal(t, firstID, secondID)
				assert.Same(t, firstSource, conv.GLX.Sources[secondID])
				wantCount = 1
			} else {
				assert.NotEqual(t, firstID, secondID)
			}
			assert.Len(t, conv.GLX.Sources, wantCount)
			assert.Equal(t, wantCount, conv.Stats.SourcesCreated)
			assert.Equal(t, wantCount, conv.SourceCounter, "reuse must not consume a source ID")
			assert.Equal(t, secondID, createSyntheticSourceFromEmbeddedCitation(tt.second, conv), "reuse the first matching ID")
			assert.Equal(t, wantCount, conv.Stats.SourcesCreated)
		})
	}
}

func TestImportEmbeddedSourcesPreservesEvidence(t *testing.T) {
	// The two explicit records deliberately match the synthetic source's content.
	// They must retain their XREF identities, while inline citations share a source
	// across individual and family events without losing their own details.
	for _, version := range []string{"5.5.1", "7.0"} {
		t.Run(version, func(t *testing.T) {
			gedcom := `0 HEAD
1 GEDC
2 VERS ` + version + `
0 @S1@ SOUR
1 TITL Family Bible
1 TEXT Transcription
1 NOTE Source created from embedded GEDCOM citation
1 NOTE Copy
0 @S2@ SOUR
1 TITL Family Bible
1 TEXT Transcription
1 NOTE Source created from embedded GEDCOM citation
1 NOTE Copy
0 @M1@ OBJE
1 FILE birth.jpg
0 @M2@ OBJE
1 FILE death.jpg
0 @I1@ INDI
1 NAME John /Smith/
2 SOUR @S1@
2 SOUR @S2@
1 BIRT
2 DATE 1 JAN 1850
2 SOUR Family Bible
3 TEXT Transcription
3 PAGE Page 1
3 DATA
4 DATE 2 JAN 1850
4 TEXT Birth entry
3 QUAY 3
3 NOTE Copy
3 OBJE @M1@
1 DEAT
2 DATE 1 JAN 1920
2 SOUR Family Bible
3 TEXT Transcription
3 PAGE Page 2
3 DATA
4 DATE 2 JAN 1920
4 TEXT Death entry
3 QUAY 1
3 NOTE Copy
3 OBJE @M2@
0 @I2@ INDI
1 NAME Mary /Jones/
2 SOUR
2 SOUR
1 BIRT
2 DATE 1 JAN 1855
2 SOUR
3 PAGE Unknown entry
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 1 JAN 1875
2 SOUR Family Bible
3 TEXT Transcription
3 PAGE Page 3
3 NOTE Copy
0 TRLR
`
			for range 2 { // Independent imports must each create their own sources.
				archive, result, err := ImportGEDCOM(strings.NewReader(gedcom), nil)
				require.NoError(t, err)
				require.Empty(t, result.Statistics.Errors)
				require.Len(t, archive.Sources, 4, "two explicit sources, one Bible synthetic, one placeholder")
				assert.Equal(t, 4, result.Statistics.SourcesCreated)
				assert.Equal(t, len(archive.Citations), result.Statistics.CitationsCreated)
				assert.Equal(t, archive.Sources["source-1"], archive.Sources["source-2"], "identical explicit source records remain distinct")
				assert.Equal(t, "Family Bible", archive.Sources["source-3"].Title)
				assert.Equal(t, "Transcription", archive.Sources["source-3"].Properties["description"])
				assert.Equal(t, NoteList{"Source created from embedded GEDCOM citation", "Copy"}, archive.Sources["source-3"].Notes)
				assert.Equal(t, "Embedded Citation (No Source Description)", archive.Sources["source-4"].Title)

				byPage := make(map[string]*Citation)
				for _, citation := range archive.Citations {
					page, _ := getStringProperty(citation.Properties, "locator")
					byPage[page] = citation
				}
				for _, page := range []string{"Page 1", "Page 2", "Page 3"} {
					require.Contains(t, byPage, page)
					assert.Equal(t, "source-3", byPage[page].SourceID)
				}
				assert.Equal(t, "Birth entry", byPage["Page 1"].Properties["text_from_source"])
				assert.Equal(t, "1850-01-02", byPage["Page 1"].Properties["source_date"])
				assert.Equal(t, NoteList{"GEDCOM QUAY: 3", "Copy"}, byPage["Page 1"].Notes)
				assert.Equal(t, []string{"media-1"}, byPage["Page 1"].Media)
				assert.Equal(t, "Death entry", byPage["Page 2"].Properties["text_from_source"])
				assert.Equal(t, "1920-01-02", byPage["Page 2"].Properties["source_date"])
				assert.Equal(t, NoteList{"GEDCOM QUAY: 1", "Copy"}, byPage["Page 2"].Notes)
				assert.Equal(t, []string{"media-2"}, byPage["Page 2"].Media)
				require.Contains(t, byPage, "Unknown entry")
				assert.Equal(t, "source-4", byPage["Unknown entry"].SourceID)

				var personSources [][]string
				for _, assertion := range archive.Assertions {
					if assertion.Subject.Person != "" {
						personSources = append(personSources, assertion.Sources)
					}
					for _, sourceID := range assertion.Sources {
						assert.Contains(t, archive.Sources, sourceID)
					}
					for _, citationID := range assertion.Citations {
						assert.Contains(t, archive.Citations, citationID)
					}
				}
				assert.Contains(t, personSources, []string{"source-1", "source-2"})
				assert.Contains(t, personSources, []string{"source-4"}, "repeated bare placeholder refs should collapse")
				for _, event := range archive.Events {
					citations, ok := event.Properties[PropertyCitations].([]string)
					require.True(t, ok, "each event must retain citation evidence")
					require.NotEmpty(t, citations)
					for _, citationID := range citations {
						assert.Contains(t, archive.Citations, citationID)
					}
				}
			}
		})
	}
}
