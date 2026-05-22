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

	"github.com/stretchr/testify/assert"
)

func TestTransliterateForSlug(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// German umlauts and ß use conventional digraph expansions.
		{"lower umlaut a", "ä", "ae"},
		{"lower umlaut o", "ö", "oe"},
		{"lower umlaut u", "ü", "ue"},
		{"sharp s", "ß", "ss"},
		{"upper umlaut A", "Ä", "Ae"},
		{"upper umlaut O", "Ö", "Oe"},
		{"upper umlaut U", "Ü", "Ue"},
		{"capital sharp s", "ẞ", "Ss"},
		{"german word Mueller", "Müller", "Mueller"},
		{"german word Groesse", "Größe", "Groesse"},
		{"german word Kirchenbuecher", "Kirchenbücher", "Kirchenbuecher"},
		// Decomposed (NFD) umlaut input: "u" followed by combining
		// diaeresis U+0308, written with an explicit \u0308 escape so an
		// editor cannot silently re-normalize it to the precomposed form. It
		// is composed to NFC first, so it still expands to the digraph rather
		// than being reduced to a bare vowel by the NFKD pass.
		{"decomposed umlaut u", "u\u0308", "ue"},
		{"decomposed umlaut word", "Mu\u0308ller", "Mueller"},
		// Other accented Latin letters are reduced to their base via NFKD.
		{"acute e", "é", "e"},
		{"ring a", "å", "a"},
		{"tilde n", "ñ", "n"},
		{"cedilla c", "ç", "c"},
		{"french Quebec", "Québec", "Quebec"},
		{"french Etienne", "Saint-Étienne", "Saint-Etienne"},
		{"spanish Nino", "Niño", "Nino"},
		{"portuguese Sao", "São", "Sao"},
		// Plain ASCII and unmapped scripts pass through unchanged (the caller's
		// slug step decides what to do with non-ASCII it cannot transliterate).
		{"plain ascii", "Hello World", "Hello World"},
		{"empty", "", ""},
		{"cjk passthrough", "東京", "東京"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, TransliterateForSlug(tt.in), "TransliterateForSlug(%q)", tt.in)
		})
	}
}
