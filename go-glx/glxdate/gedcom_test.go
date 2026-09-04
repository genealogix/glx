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
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplitGEDCOMEscape pins how calendar escapes map to GLX prefixes.
func TestSplitGEDCOMEscape(t *testing.T) {
	tests := []struct {
		input      string
		wantPrefix string
		wantBody   string
		wantFound  bool
	}{
		{"@#DJULIAN@ 15 MAR 1731", PrefixJulian, "15 MAR 1731", true},
		{"@#DHEBREW@ 15 TSH 5765", PrefixHebrew, "15 TSH 5765", true},
		{"@#DFRENCH R@ 1 VEND 0012", PrefixFrenchRepublican, "1 VEND 0012", true},
		{"@#DGREGORIAN@ 15 MAR 1731", "", "15 MAR 1731", true},
		{"@#DJULIAN@15 MAR 1731", PrefixJulian, "15 MAR 1731", true},
		{"@#DROMAN@ 15 MAR 1731", "ROMAN", "15 MAR 1731", true},
		{"@#DNEW CAL@ 15 MAR 1731", "NEW_CAL", "15 MAR 1731", true},
		{"@#DJULIAN@", PrefixJulian, "", true},
		{"15 MAR 1731", "", "15 MAR 1731", false},
		{"@#DJULIAN 15 MAR 1731", "", "@#DJULIAN 15 MAR 1731", false},
		{"", "", "", false},
	}
	for _, tt := range tests {
		prefix, body, found := splitGEDCOMEscape(tt.input)
		assert.Equal(t, tt.wantPrefix, prefix, tt.input)
		assert.Equal(t, tt.wantBody, body, tt.input)
		assert.Equal(t, tt.wantFound, found, tt.input)
	}
}

// TestGEDCOMEscape pins the prefix → escape direction, including the
// underscore folding that keeps unknown calendar names a single token.
func TestGEDCOMEscape(t *testing.T) {
	assert.Equal(t, "@#DJULIAN@", gedcomEscape(PrefixJulian))
	assert.Equal(t, "@#DHEBREW@", gedcomEscape(PrefixHebrew))
	assert.Equal(t, "@#DFRENCH R@", gedcomEscape(PrefixFrenchRepublican))
	assert.Empty(t, gedcomEscape(""))
	assert.Equal(t, "@#DROMAN@", gedcomEscape("ROMAN"))
	assert.Equal(t, "@#DNEW CAL@", gedcomEscape("NEW_CAL"))
}

// TestFromGEDCOM pins the dialect variants the importer normalizes (all
// unambiguous), the calendars it preserves, and the malformed forms it
// keeps verbatim.
func TestFromGEDCOM(t *testing.T) {
	tests := map[string]string{
		"1 JAN 1900":                      "1900-01-01",
		"JAN 1900":                        "1900-01",
		"1900":                            "1900",
		"850":                             "0850",
		"BEF 15 JAN 1900":                 "BEF 1900-01-15",
		"1 JANUARY 1900":                  "1900-01-01",
		"15 March 2020":                   "2020-03-15",
		"Abt 1850":                        "ABT 1850",
		"abt 1850":                        "ABT 1850",
		"ABT. 1850":                       "ABT 1850",
		"Bet 1880 and 1890":               "BET 1880 AND 1890",
		"From Jan 1900 to Feb 1901":       "FROM 1900-01 TO 1901-02",
		"FROM 1900":                       "FROM 1900",
		"March 15, 1850":                  "1850-03-15",
		"INT 15 MAR 1850 (Ides of March)": "INT 1850-03-15 (Ides of March)",
		"  1850  ":                        "1850",
		"":                                "",

		// Calendars.
		"@#DJULIAN@ 15 March 1731":     "JULIAN 1731-03-15",
		"@#DJULIAN@ ABT 1731":          "JULIAN ABT 1731",
		"@#DGREGORIAN@ 15 MAR 1731":    "1731-03-15",
		"@#DHEBREW@ 15 TSH 5765":       "HEBREW 15 TSH 5765",
		"@#DFRENCH R@ 1 VEND 0012":     "FRENCH_R 1 VEND 0012",
		"@#DROMAN@ 15 MAR 1731":        "ROMAN 15 MAR 1731",
		"@#DJULIAN@":                   "@#DJULIAN@",
		"@#DJULIAN@ BET 1700 AND 1710": "JULIAN BET 1700 AND 1710",

		// Recovered dialect: unambiguous keyword spellings and eras.
		"100 BC":           "0100 BCE",
		"EST 1850":         "EST 1850",
		"C. 1850":          "ABT 1850",
		"BET. 1880 - 1890": "BET 1880 AND 1890",

		// Preserved verbatim: canonicalizing would mean guessing.
		"15/01/1900":           "15/01/1900",
		"01/15/1900":           "01/15/1900",
		"1731/32":              "1731/32",
		"ABT 1731/32":          "ABT 1731/32",
		"31 FEB 1850":          "31 FEB 1850",
		"BET 1880 AND unknown": "BET 1880 AND unknown",
		"TO 1900":              "TO 1900",
		"1850-3-5":             "1850-3-5",
	}
	for input, want := range tests {
		assert.Equal(t, want, FromGEDCOM(input), input)
	}
}

// TestParseGEDCOM checks the parsed view of a GEDCOM payload.
func TestParseGEDCOM(t *testing.T) {
	d, err := ParseGEDCOM("@#DJULIAN@ 15 March 1731")
	require.NoError(t, err)
	assert.Equal(t, CalendarJulian, d.Calendar())
	assert.Equal(t, 1731, d.Year())
	assert.Equal(t, PrecisionDay, d.Precision())

	d, err = ParseGEDCOM("15/01/1900")
	require.Error(t, err)
	assert.Equal(t, "15/01/1900", d.String())
	assert.Equal(t, 1900, d.Year())
}

// TestDate_GEDCOM pins the GLX → GEDCOM rendering.
func TestDate_GEDCOM(t *testing.T) {
	tests := map[string]string{
		"":                               "",
		"1850":                           "1850",
		"0850":                           "0850",
		"1850-03":                        "MAR 1850",
		"1850-03-15":                     "15 MAR 1850",
		"ABT 1850-03-15":                 "ABT 15 MAR 1850",
		"ABT 1850":                       "ABT 1850",
		"BEF 1920-01-15":                 "BEF 15 JAN 1920",
		"AFT 1900-12":                    "AFT DEC 1900",
		"CAL 1880-06-01":                 "CAL 1 JUN 1880",
		"INT 1850-03-15 (Ides of March)": "INT 15 MAR 1850 (Ides of March)",
		"BET 1880 AND 1890":              "BET 1880 AND 1890",
		"BET 1880-01-01 AND 1890-12-31":  "BET 1 JAN 1880 AND 31 DEC 1890",
		"FROM 1880 TO 1890":              "FROM 1880 TO 1890",
		"FROM 1900-06 TO 1950-12":        "FROM JUN 1900 TO DEC 1950",
		"FROM 1900":                      "FROM 1900",
		"2000-12-25":                     "25 DEC 2000",

		// Calendars.
		"JULIAN 1731-03-15":         "@#DJULIAN@ 15 MAR 1731",
		"JULIAN ABT 1731":           "@#DJULIAN@ ABT 1731",
		"JULIAN BET 1700 AND 1710":  "@#DJULIAN@ BET 1700 AND 1710",
		"HEBREW 15 TSH 5765":        "@#DHEBREW@ 15 TSH 5765",
		"FRENCH_R 1 VEND 0012":      "@#DFRENCH R@ 1 VEND 0012",
		"ROMAN 15 MAR 1731":         "@#DROMAN@ 15 MAR 1731",
		"NEW_CAL 15 MAR 1731":       "@#DNEW CAL@ 15 MAR 1731",
		"@#DJULIAN@":                "@#DJULIAN@",
		"HEBREW BET 1 TSH AND 5765": "@#DHEBREW@ BET 1 TSH AND 5765",

		// Tolerated spellings render canonically; preserved bodies do not.
		"15 March 1850":        "15 MAR 1850",
		"15/01/1900":           "15/01/1900",
		"1731/32":              "1731/32",
		"EST 1850":             "EST 1850",
		"BET 1880 AND unknown": "BET 1880 AND unknown",
		"1850-3-5":             "1850-3-5",
		"31 FEB 1850":          "31 FEB 1850",
	}
	for input, want := range tests {
		d, _ := Parse(input)
		assert.Equal(t, want, d.GEDCOM(), input)
	}
}

// standardGEDCOMEscapedRegexp matches a standard single Gregorian GEDCOM
// date body, optionally preceded by a Gregorian or Julian calendar escape.
var standardGEDCOMEscapedRegexp = regexp.MustCompile(`(?i)^(?:@#D(?:GREGORIAN|JULIAN)@\s+)?(?:(?:ABT|BEF|AFT|CAL)\s+)?(?:(?:\d{1,2}\s+)?(?:JAN(?:UARY)?|FEB(?:RUARY)?|MAR(?:CH)?|APR(?:IL)?|MAY|JUN(?:E)?|JUL(?:Y)?|AUG(?:UST)?|SEP(?:T|TEMBER)?|OCT(?:OBER)?|NOV(?:EMBER)?|DEC(?:EMBER)?)\.?\s+)?\d{3,4}$`)

// TestCorpus_GEDCOMRoundtrip runs the GEDCOM adapter over the real-world
// corpus and checks the end-to-end properties #1025 is about:
//
//   - the year read from the stored GLX date is the year of the GEDCOM
//     input (never the day of month);
//   - every standard GEDCOM body, in any letter case, is stored in
//     canonical GLX form;
//   - importing is stable: exporting the stored date back to GEDCOM and
//     importing it again reaches a fixed point without changing the year.
func TestCorpus_GEDCOMRoundtrip(t *testing.T) {
	yearChecked, canonChecked := 0, 0
	for _, line := range corpusLines(t) {
		stored := FromGEDCOM(line)
		require.NotEmpty(t, stored, "non-empty GEDCOM date %q must not import as empty", line)
		d, err := Parse(stored)

		if want := oracleYear(line); want != 0 {
			yearChecked++
			assert.Equal(t, want, d.Year(), "year of %q stored as %q", line, stored)
		}

		if standardGEDCOMEscapedRegexp.MatchString(line) && oracleYear(line) != 0 {
			canonChecked++
			require.NoError(t, err, "standard GEDCOM date %q stored non-canonically as %q", line, stored)
		}

		// Export normalizes whitespace and keyword case inside raw-preserved
		// bodies, so the fixed point is reached after one roundtrip.
		once, _ := ParseGEDCOM(d.GEDCOM())
		twice, _ := ParseGEDCOM(once.GEDCOM())
		assert.Equal(t, once.String(), twice.String(), "import → export → import not stable for %q", line)
		assert.Equal(t, d.Year(), once.Year(), "year changed on roundtrip of %q", line)
	}
	assert.Greater(t, yearChecked, 4000)
	assert.Greater(t, canonChecked, 500)
}
