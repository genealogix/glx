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
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// germanDigraphs expands German umlauts and ß to their conventional ASCII
// digraphs. This MUST run before NFKD: NFKD decomposes ü into "u" + combining
// diaeresis (which the mark-stripper then reduces to "u"), so mapping ü→ue
// after NFKD is impossible — the umlaut is already gone. ß has no NFKD
// decomposition at all, so without this map it would be dropped by the slug
// regex. Both letter cases are handled so the function is correct standalone.
var germanDigraphs = strings.NewReplacer(
	"ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss",
	"Ä", "Ae", "Ö", "Oe", "Ü", "Ue", "ẞ", "Ss",
)

// TransliterateForSlug converts German umlauts/ß and accented Latin letters in
// s to conventional ASCII, so that downstream slug/ID generation produces
// readable identifiers instead of dropping non-ASCII letters and leaving stray
// hyphens. German umlauts use digraph expansions (ä→ae, ö→oe, ü→ue, ß→ss); all
// other diacritics are removed via NFKD. Characters with no ASCII
// transliteration (CJK, emoji, ligatures like æ/ø/ł) are returned unchanged —
// the caller's slug step decides what to do with them.
func TransliterateForSlug(s string) string {
	s = germanDigraphs.Replace(s)

	// NFKD-decompose, then drop the combining marks (Unicode category Mn) it
	// exposes, reducing accented Latin letters to their base form: é→e, å→a,
	// ñ→n, ç→c, etc. strings.Map drops a rune when the mapping returns < 0.
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}

		return r
	}, norm.NFKD.String(s))
}
