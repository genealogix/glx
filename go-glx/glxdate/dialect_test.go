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

package glxdate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParse_EST: EST is a qualifier in its own right, a sibling of ABT and
// CAL (GEDCOM: estimated from another event's date), not a spelling of INT.
func TestParse_EST(t *testing.T) {
	d, err := Parse("EST 1850")
	require.NoError(t, err)
	assert.Equal(t, QualifierEstimated, d.Qualifier())
	assert.Equal(t, "EST", d.Qualifier().Keyword())
	assert.Equal(t, 1850, d.Year())
	assert.Equal(t, "EST 1850", d.GEDCOM())

	assert.Equal(t, "EST 1850-03-15", Canonicalize("est 15 Mar 1850"))
	assert.Equal(t, "EST 1850", Canonicalize("Estimated 1850"))
}

// TestParse_KeywordSynonyms: unambiguous dialect spellings canonicalize to
// the keyword; ambiguous ones stay raw.
func TestParse_KeywordSynonyms(t *testing.T) {
	for input, want := range map[string]string{
		"about 1850":                 "ABT 1850",
		"About 1850":                 "ABT 1850",
		"circa 1850":                 "ABT 1850",
		"CIRCA 1850":                 "ABT 1850",
		"CIR 1375":                   "ABT 1375",
		"ca. 1841":                   "ABT 1841",
		"CA 1841":                    "ABT 1841",
		"c. 1841":                    "ABT 1841",
		"C 1841":                     "ABT 1841",
		"C. NOV 1841":                "ABT 1841-11",
		"around 1850":                "ABT 1850",
		"before 1900":                "BEF 1900",
		"after 1900":                 "AFT 1900",
		"Between 1880 and 1890":      "BET 1880 AND 1890",
		"calculated 1850":            "CAL 1850",
		"BET 1675 - 1740":            "BET 1675 AND 1740",
		"BET. 1675 - 1740":           "BET 1675 AND 1740",
		"BET 07 OCT AND 08 NOV 1260": "BET 1260-10-07 AND 1260-11-08",
		"BET JUL AND SEP 1857":       "BET 1857-07 AND 1857-09",
		// Not synonyms: "by" is before-or-in, "?" is doubt, "1675-1740" has no keyword.
		"by 1850":                "by 1850",
		"abt 1600?":              "abt 1600?",
		"1675 - 1740":            "1675 - 1740",
		"Bet 1173":               "Bet 1173",
		"BET 6 MAY 1943 AND MAR": "BET 6 MAY 1943 AND MAR",
	} {
		assert.Equal(t, want, Canonicalize(input), input)
		d, err := Parse(input)
		if want != input {
			require.Error(t, err, "%q is tolerated, not canonical", input)
			assert.Positive(t, d.Year(), input)
		}
	}

	// Canonical spellings are valid as written.
	for _, input := range []string{"ABT 1850", "EST 1850", "BET 1675 AND 1740"} {
		_, err := Parse(input)
		require.NoError(t, err, input)
	}
}

// TestParse_BCE: the era suffix parses in every common spelling,
// canonicalizes to BCE, exports as BCE, and makes Year negative.
func TestParse_BCE(t *testing.T) {
	for input, want := range map[string]string{
		"510 BC":                        "0510 BCE",
		"510 B.C.":                      "0510 BCE",
		"510 BCE":                       "0510 BCE",
		"44 B.C.E.":                     "0044 BCE",
		"ABT 560 BC":                    "ABT 0560 BCE",
		"15 MAR 44 BC":                  "0044-03-15 BCE",
		"MAR 44 BC":                     "0044-03 BCE",
		"BET 100 BC AND 50 BC":          "BET 0100 BCE AND 0050 BCE",
		"FROM 100 BC TO 50":             "FROM 0100 BCE TO 0050",
		"JULIAN 15 MAR 44 BC":           "JULIAN 0044-03-15 BCE",
		"INT 44 B.C. (Ides)":            "INT 0044 BCE (Ides)",
		"0044-03-15 BCE":                "0044-03-15 BCE",
		"BC":                            "BC",
		"abt. 1317 BC (or abt. 934 BC)": "abt. 1317 BC (or abt. 934 BC)",
	} {
		assert.Equal(t, want, Canonicalize(input), input)
	}

	d, err := Parse("0044-03-15 BCE")
	require.NoError(t, err)
	assert.Equal(t, -44, d.Year())
	m, _ := d.Month()
	assert.Equal(t, 3, m)
	assert.Equal(t, PrecisionDay, d.Precision())
	assert.Equal(t, "15 MAR 0044 BCE", d.GEDCOM())

	d, err = Parse("510 BC")
	require.Error(t, err)
	assert.Equal(t, -510, d.Year())
	assert.Equal(t, "0510 BCE", d.String())

	d, err = Parse("BET 0100 BCE AND 0050 BCE")
	require.NoError(t, err)
	assert.Equal(t, -100, d.Year())
	assert.Equal(t, -50, d.End().Year())
	assert.Equal(t, "BET 100 BCE AND 50 BCE", canonicalGEDCOMYears(d.GEDCOM()))

	// Raw-preserved text that names an era still dates BCE.
	d, _ = Parse("abt. 1317 BC (or abt. 934 BC)")
	assert.Equal(t, -1317, d.Year())

	// Constructed BCE dates.
	n := New(CalendarGregorian, -44, 3, 15)
	assert.True(t, n.Valid())
	assert.Equal(t, "0044-03-15 BCE", n.String())
	assert.Equal(t, -44, n.Year())
	assert.True(t, n.Equal(MustParse("0044-03-15 BCE")))
}

// canonicalGEDCOMYears strips the zero padding GEDCOM output carries so the
// assertion reads naturally.
func canonicalGEDCOMYears(s string) string {
	return strings.NewReplacer("0100", "100", "0050", "50").Replace(s)
}
