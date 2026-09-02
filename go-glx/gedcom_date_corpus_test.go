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
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// loadGEDCOMDateCorpus returns the real-world GEDCOM DATE sample shared with
// the glxdate package tests (see glxdate/corpus_test.go for provenance).
func loadGEDCOMDateCorpus(t *testing.T) []string {
	t.Helper()

	f, err := os.Open("glxdate/testdata/gedcom_dates.txt")
	require.NoError(t, err)
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	require.NoError(t, scanner.Err())
	require.Greater(t, len(lines), 5000)

	return lines
}

var corpusFourDigitRegexp = regexp.MustCompile(`(?:^|[^0-9])(\d{4})(?:[^0-9]|$)`)

// corpusOracleYear is the independent year oracle from the glxdate corpus
// test: the first 4-digit number in the start component of a date.
func corpusOracleYear(s string) int {
	if i := strings.Index(s, " ("); i >= 0 {
		s = s[:i]
	}
	upper := strings.ToUpper(s)
	for _, sep := range []string{" AND ", " TO "} {
		if i := strings.Index(upper, sep); i >= 0 {
			s, upper = s[:i], upper[:i]
		}
	}
	if m := corpusFourDigitRegexp.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])

		return n
	}

	return 0
}

// corpusStandardGEDCOMRegexp matches a standard single Gregorian GEDCOM date
// body, optionally preceded by a calendar escape.
var corpusStandardGEDCOMRegexp = regexp.MustCompile(`(?i)^(?:@#D(?:GREGORIAN|JULIAN)@\s+)?(?:(?:ABT|BEF|AFT|CAL)\s+)?(?:(?:\d{1,2}\s+)?(?:JAN(?:UARY)?|FEB(?:RUARY)?|MAR(?:CH)?|APR(?:IL)?|MAY|JUN(?:E)?|JUL(?:Y)?|AUG(?:UST)?|SEP(?:T|TEMBER)?|OCT(?:OBER)?|NOV(?:EMBER)?|DEC(?:EMBER)?)\.?\s+)?\d{3,4}$`)

// TestParseGEDCOMDate_Corpus runs the GEDCOM → GLX date adapter over the
// real-world corpus and checks the end-to-end properties #1025 is about:
//
//   - the year read from the stored GLX date is the year of the GEDCOM
//     input (never the day of month);
//   - every standard GEDCOM body, in any letter case, is stored in
//     canonical GLX form;
//   - importing is stable: exporting the stored date back to GEDCOM and
//     importing it again reaches a fixed point without changing the year.
func TestParseGEDCOMDate_Corpus(t *testing.T) {
	yearChecked, canonChecked := 0, 0
	for _, line := range loadGEDCOMDateCorpus(t) {
		stored := parseGEDCOMDate(line)
		require.NotEmpty(t, stored, "non-empty GEDCOM date %q must not import as empty", line)

		if want := corpusOracleYear(line); want != 0 {
			yearChecked++
			assert.Equal(t, want, ExtractFirstYear(string(stored)), "year of %q stored as %q", line, stored)
		}

		if corpusStandardGEDCOMRegexp.MatchString(line) && corpusOracleYear(line) != 0 {
			canonChecked++
			_, err := stored.Parse()
			require.NoError(t, err, "standard GEDCOM date %q stored non-canonically as %q", line, stored)
		}

		// Export normalizes whitespace inside raw-preserved bodies, so the
		// fixed point is reached after one roundtrip.
		once := parseGEDCOMDate(formatGEDCOMDate(stored))
		twice := parseGEDCOMDate(formatGEDCOMDate(once))
		assert.Equal(t, once, twice, "import → export → import not stable for %q", line)
		assert.Equal(t, ExtractFirstYear(string(stored)), ExtractFirstYear(string(once)), "year changed on roundtrip of %q", line)
	}
	assert.Greater(t, yearChecked, 4000)
	assert.Greater(t, canonChecked, 500)
}

// TestParseGEDCOMDate_DialectTolerance pins the dialect variants the adapter
// normalizes (all unambiguous) and the malformed forms it preserves.
func TestParseGEDCOMDate_DialectTolerance(t *testing.T) {
	tests := map[string]DateString{
		"1 JANUARY 1900":                  "1900-01-01",
		"15 March 2020":                   "2020-03-15",
		"Abt 1850":                        "ABT 1850",
		"abt 1850":                        "ABT 1850",
		"ABT. 1850":                       "ABT 1850",
		"Bet 1880 and 1890":               "BET 1880 AND 1890",
		"From Jan 1900 to Feb 1901":       "FROM 1900-01 TO 1901-02",
		"March 15, 1850":                  "1850-03-15",
		"@#DJULIAN@ 15 March 1731":        "JULIAN 1731-03-15",
		"@#DHEBREW@ 15 TSH 5765":          "HEBREW 15 TSH 5765",
		"INT 15 MAR 1850 (Ides of March)": "INT 1850-03-15 (Ides of March)",
		"15/01/1900":                      "15/01/1900",
		"01/15/1900":                      "01/15/1900",
		"1731/32":                         "1731/32",
		"ABT 1731/32":                     "ABT 1731/32",
		"100 BC":                          "100 BC",
		"EST 1850":                        "EST 1850",
		"C. 1850":                         "C. 1850",
		"BET. 1880 - 1890":                "BET. 1880 - 1890",
		"31 FEB 1850":                     "31 FEB 1850",
		"BET 1880 AND unknown":            "BET 1880 AND unknown",
		"TO 1900":                         "TO 1900",
		"850":                             "0850",
	}
	for input, want := range tests {
		assert.Equal(t, want, parseGEDCOMDate(input), input)
	}
}

// TestDateString_Year pins the DateString convenience accessor.
func TestDateString_Year(t *testing.T) {
	assert.Equal(t, 1900, DateString("1 JANUARY 1900").Year())
	assert.Equal(t, 1880, DateString("BET 1880 AND 1890").Year())
	assert.Equal(t, 0, DateString("").Year())

	d, err := DateString("15 March 2020").Parse()
	require.Error(t, err)
	assert.Equal(t, 2020, d.Year())
	assert.Equal(t, glxdate.PrecisionDay, d.Precision())
}
