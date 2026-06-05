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
)

// TestMapSourceType verifies that every registered GEDCOM SOUR.TYPE value
// resolves to its mapped GLX source type, that matching is case-insensitive,
// and that unknown values fall back to "other". The full table is driven from
// gedcomSourceTypeMapping itself so new entries stay covered automatically; the
// #563 additions are additionally spot-checked against their expected types.
func TestMapSourceType(t *testing.T) {
	for gedcomType, want := range gedcomSourceTypeMapping {
		t.Run(gedcomType, func(t *testing.T) {
			assert.Equal(t, want, mapSourceType(gedcomType), "lowercase lookup")
			assert.Equal(t, want, mapSourceType(strings.ToUpper(gedcomType)), "uppercase is normalized")
		})
	}

	// Spot-check the #563 additions against their expected GLX types so the
	// pairings are pinned independently of the mapping table.
	assert.Equal(t, SourceTypeFamilyBible, mapSourceType("bible"))
	assert.Equal(t, SourceTypeDNATest, mapSourceType("dna"))
	assert.Equal(t, SourceTypeSocialMedia, mapSourceType("social"))

	// Unknown and empty values fall back to "other".
	assert.Equal(t, SourceTypeOther, mapSourceType("interpretive dance"))
	assert.Equal(t, SourceTypeOther, mapSourceType(""))
}

// TestInferSourceType verifies title-based source-type inference, including the
// types added in #563. The "map" cases deliberately assert that only the
// distinctive phrases "survey plat" / "plat map" trigger SourceTypeMap — a bare
// "map" substring (e.g. inside the place name "Mapleton") must NOT, to avoid
// false positives.
func TestInferSourceType(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		// Pre-existing inference rules.
		{"census record", "1850 Census of Leeds", SourceTypeCensus},
		{"birth certificate", "Birth Certificate of John Smith", SourceTypeVitalRecord},
		{"death certificate", "Death Certificate of Mary Brown", SourceTypeVitalRecord},
		{"baptism register", "Baptism Register, St. Mary's Parish", SourceTypeChurchRegister},
		{"parish register", "Parish Register of Burials", SourceTypeChurchRegister},
		{"military record", "Military Service Record", SourceTypeMilitary},
		{"newspaper archive", "Yorkshire Post Newspaper Archive", SourceTypeNewspaper},
		{"will", "Last Will and Testament of Jane Doe", SourceTypeProbate},
		{"probate file", "Probate File 1899", SourceTypeProbate},
		{"deed", "Property Deed, 1820", SourceTypeLand},
		{"land grant", "Land Grant Record", SourceTypeLand},
		{"population register", "Population Register of Amsterdam", SourceTypePopulationRegister},
		{"household register", "Household Register, Tokyo Ward", SourceTypePopulationRegister},
		{"tax roll", "Tax Roll of 1798", SourceTypeTaxRecord},
		{"tithe", "Tithe Apportionment Schedule", SourceTypeTaxRecord},
		{"notarial act", "Notarial Act of Sale", SourceTypeNotarialRecord},
		{"notary protocol", "Notary Protocol Book", SourceTypeNotarialRecord},

		// Rules added in #563.
		{"family bible", "Smith Family Bible", SourceTypeFamilyBible},
		{"bible record", "Bible Record of the Jones Family", SourceTypeFamilyBible},
		{"gravestone inscription", "Gravestone of John Doe", SourceTypeGravestone},
		{"headstone keyword", "Headstone Inscription, Highgate", SourceTypeGravestone},
		{"tombstone keyword", "Tombstone Transcription", SourceTypeGravestone},
		{"dna test phrase", "AncestryDNA Test Results", SourceTypeDNATest},
		{"y-dna marker", "Y-DNA 37-Marker Results", SourceTypeDNATest},
		{"mtdna sequence", "mtDNA Full Sequence", SourceTypeDNATest},
		{"autosomal match", "Autosomal DNA Match List", SourceTypeDNATest},
		{"memoir title", "Memoir of a Country Doctor", SourceTypeMemoir},
		{"diary title", "Diary of Anne Wexford", SourceTypeMemoir},
		{"manuscript title", "Unpublished Manuscript on the Wexford Line", SourceTypeManuscript},
		{"typescript draft", "Typescript Draft of Family History", SourceTypeManuscript},
		{"survey plat", "1872 Survey Plat of Section 12", SourceTypeMap},
		{"plat map", "Plat Map of Springfield Township", SourceTypeMap},

		// Negative cases: a bare "map" substring must not classify as map.
		{"bare map word is not map", "Map of Yorkshire, 1850", SourceTypeOther},
		{"map inside placename is not map", "Mapleton Cemetery Records", SourceTypeOther},
		{"unrelated title falls back to other", "Assorted Personal Papers", SourceTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, inferSourceType(tt.title))
		})
	}
}
