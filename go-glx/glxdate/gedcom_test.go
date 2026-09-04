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

// TestExtensionPrefix pins how GEDCOM calendar names (5.5.1 escape text or
// 7.0 tags) map to GLX prefixes.
func TestExtensionPrefix(t *testing.T) {
	for name, want := range map[string]string{
		"JULIAN": PrefixJulian, "HEBREW": PrefixHebrew, "FRENCH R": PrefixFrenchRepublican,
		"FRENCH_R": PrefixFrenchRepublican, "GREGORIAN": "", "": "",
		"ROMAN": "_ROMAN", "_ROMAN": "_ROMAN", "NEW CAL": "_NEW_CAL", "_NEW_CAL": "_NEW_CAL",
	} {
		assert.Equal(t, want, extensionPrefix(name), name)
	}
}

// TestFromGEDCOM_CalendarPlacement: calendars are recognized in both GEDCOM
// spellings wherever the grammar allows them, and lifted to the GLX prefix
// when every endpoint agrees.
func TestFromGEDCOM_CalendarPlacement(t *testing.T) {
	for input, want := range map[string]string{
		"@#DJULIAN@ 15 MAR 1731":                              "JULIAN 1731-03-15",
		"@#DJULIAN@15 MAR 1731":                               "JULIAN 1731-03-15",
		"ABT @#DJULIAN@ 15 MAR 1731":                          "JULIAN ABT 1731-03-15",
		"BET @#DJULIAN@ 1 JAN 1700 AND @#DJULIAN@ 1 JAN 1701": "JULIAN BET 1700-01-01 AND 1701-01-01",
		"@#DGREGORIAN@ 15 MAR 1731":                           "1731-03-15",
		"@#D@ 1850":                                           "1850",
		"@#DROMAN@ 15 MAR 1731":                               "_ROMAN 15 MAR 1731",
		"@#DNEW CAL@ 15 MAR 1731":                             "_NEW_CAL 15 MAR 1731",
		"@#DJULIAN@":                                          "@#DJULIAN@",
		// GEDCOM 7 tags.
		"JULIAN 15 MAR 1731":                        "JULIAN 1731-03-15",
		"ABT JULIAN 1731":                           "JULIAN ABT 1731",
		"BEF HEBREW 15 TSH 5765":                    "HEBREW BEF 15 TSH 5765",
		"GREGORIAN 12 AUG 1401":                     "1401-08-12",
		"ABT GREGORIAN 1401 BCE":                    "ABT 1401 BCE",
		"BET JULIAN 1700 AND JULIAN 1710":           "JULIAN BET 1700 AND 1710",
		"FROM GREGORIAN JAN 1689 TO GREGORIAN 1700": "FROM 1689-01 TO 1700",
		"ABT _ROMAN 1000":                           "_ROMAN ABT 1000",
		// Mixed calendars cannot be lifted and stay verbatim.
		"FROM JULIAN 1000 TO HEBREW 5000":        "FROM JULIAN 1000 TO HEBREW 5000",
		"BET GREGORIAN JUL 1950 AND JULIAN 1428": "BET GREGORIAN JUL 1950 AND JULIAN 1428",
	} {
		assert.Equal(t, want, FromGEDCOM(input), input)
	}
}

// TestGEDCOMEscape551 pins the prefix → 5.5.1 escape direction; an extension
// prefix drops its underscore so the escape it came from is written back.
func TestGEDCOMEscape551(t *testing.T) {
	assert.Equal(t, "@#DJULIAN@", gedcomEscape551(PrefixJulian))
	assert.Equal(t, "@#DHEBREW@", gedcomEscape551(PrefixHebrew))
	assert.Equal(t, "@#DFRENCH R@", gedcomEscape551(PrefixFrenchRepublican))
	assert.Equal(t, "@#DROMAN@", gedcomEscape551("_ROMAN"))
	assert.Equal(t, "@#DNEW_CAL@", gedcomEscape551("_NEW_CAL"))
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
		"@#DROMAN@ 15 MAR 1731":        "_ROMAN 15 MAR 1731",
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

		// Tolerated spellings render canonically; preserved bodies do not.
		"15 March 1850":        "15 MAR 1850",
		"15/01/1900":           "15/01/1900",
		"1731/32":              "1731/32",
		"EST 1850":             "EST 1850",
		"BET 1880 AND unknown": "BET 1880 AND unknown",
		"1850-3-5":             "1850-3-5",
		"31 FEB 1850":          "31 FEB 1850",
		"@#DJULIAN@":           "@#DJULIAN@",
	}
	for input, want := range tests {
		d, _ := Parse(input)
		assert.Equal(t, want, d.GEDCOM(), input)
		assert.Equal(t, want, d.GEDCOM551(), input)
	}

	// Calendars and eras are where the two GEDCOM versions differ: 5.5.1
	// puts an escape first, 7 puts a tag before each date.
	calendars := []struct{ glx, v551, v7 string }{
		{"JULIAN 1731-03-15", "@#DJULIAN@ 15 MAR 1731", "JULIAN 15 MAR 1731"},
		{"JULIAN ABT 1731", "@#DJULIAN@ ABT 1731", "ABT JULIAN 1731"},
		{"JULIAN BET 1700 AND 1710", "@#DJULIAN@ BET 1700 AND 1710", "BET JULIAN 1700 AND JULIAN 1710"},
		{"HEBREW 15 TSH 5765", "@#DHEBREW@ 15 TSH 5765", "HEBREW 15 TSH 5765"},
		{"FRENCH_R 1 VEND 0012", "@#DFRENCH R@ 1 VEND 0012", "FRENCH_R 1 VEND 0012"},
		{"_ROMAN 15 MAR 1731", "@#DROMAN@ 15 MAR 1731", "_ROMAN 15 MAR 1731"},
		{"_NEW_CAL 15 MAR 1731", "@#DNEW_CAL@ 15 MAR 1731", "_NEW_CAL 15 MAR 1731"},
		{"HEBREW BET 1 TSH AND 5765", "@#DHEBREW@ BET 1 TSH AND 5765", "BET HEBREW 1 TSH AND HEBREW 5765"},
		{"JULIAN FROM 1700", "@#DJULIAN@ FROM 1700", "FROM JULIAN 1700"},
		{"0044-03-15 BCE", "15 MAR 0044 B.C.", "15 MAR 0044 BCE"},
		{"JULIAN ABT 0044 BCE", "@#DJULIAN@ ABT 0044 B.C.", "ABT JULIAN 0044 BCE"},
		{"TO 1950", "TO 1950", "TO 1950"},
	}
	for _, tt := range calendars {
		d, _ := Parse(tt.glx)
		assert.Equal(t, tt.v551, d.GEDCOM551(), tt.glx)
		assert.Equal(t, tt.v7, d.GEDCOM(), tt.glx)
		assert.Equal(t, tt.glx, FromGEDCOM(tt.v551), "5.5.1 roundtrip of %q", tt.glx)
		assert.Equal(t, tt.glx, FromGEDCOM(tt.v7), "7.0 roundtrip of %q", tt.glx)
	}
}

// standardGEDCOMEscapedRegexp matches a standard single Gregorian GEDCOM
// date body, optionally preceded by a Gregorian or Julian calendar escape.
var standardGEDCOMEscapedRegexp = regexp.MustCompile(`(?i)^(?:@#D(?:GREGORIAN|JULIAN)@\s+)?` + standardGEDCOMBody)

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
