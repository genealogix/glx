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
	"bufio"
	"bytes"
	_ "embed"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gedcomDatesCorpus is a sample of real-world GEDCOM DATE values: up to two
// examples of every distinct digit-shape found across ~700k DATE lines from
// ~250 GEDCOM files (exporter samples, torture tests, family trees). Lines
// are verbatim GEDCOM payloads, including calendar escapes and free text.
//
//go:embed testdata/gedcom_dates.txt
var gedcomDatesCorpus []byte

// corpusLines returns the corpus one date per line, skipping blanks.
func corpusLines(t *testing.T) []string {
	t.Helper()

	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(gedcomDatesCorpus))
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			lines = append(lines, line)
		}
	}
	require.NoError(t, scanner.Err())
	require.Greater(t, len(lines), 5000, "corpus should be large")

	return lines
}

// firstComponent returns the part of a GEDCOM date before AND/TO, i.e. the
// start of a range, ignoring an INT date's parenthesized text.
func firstComponent(s string) string {
	if i := strings.Index(s, " ("); i >= 0 {
		s = s[:i]
	}
	upper := strings.ToUpper(s)
	for _, sep := range []string{" AND ", " TO "} {
		if i := strings.Index(upper, sep); i >= 0 {
			s = s[:i]
			upper = upper[:i]
		}
	}

	return s
}

var fourDigitRegexp = regexp.MustCompile(`(?:^|[^0-9])(\d{4})(?:[^0-9]|$)`)

// oracleYear is an independent, deliberately simple oracle: when the start
// component of a date contains a 4-digit number, that number is its year.
// Real-world GEDCOM years are overwhelmingly 4 digits, and a day of month
// never is, so this pins down exactly the class of bug reported in #1025.
// It returns 0 when the oracle does not apply.
func oracleYear(s string) int {
	first := firstComponent(s)
	if m := fourDigitRegexp.FindStringSubmatch(first); m != nil {
		n, _ := strconv.Atoi(m[1])
		if bceOracleRegexp.MatchString(first) {
			return -n
		}

		return n
	}

	return 0
}

// bceOracleRegexp recognizes an era suffix; such a year is negative.
var bceOracleRegexp = regexp.MustCompile(`(?i)\b(B\.?C\.?E?\.?|BCE)$`)

// TestCorpus_YearMatchesOracle: for every corpus date whose start component
// contains a 4-digit year, Year must return it — never the day of month.
func TestCorpus_YearMatchesOracle(t *testing.T) {
	checked := 0
	for _, line := range corpusLines(t) {
		want := oracleYear(line)
		if want == 0 {
			continue
		}
		checked++
		d, _ := Parse(line)
		assert.Equal(t, want, d.Year(), "Year(%q)", line)
	}
	assert.Greater(t, checked, 4000, "oracle should apply to most of the corpus")
}

// TestCorpus_StringIsIdempotent: String never loses information — parsing
// the rendered form again renders the same text with the same year, and a
// valid date stays valid with the same components. (An invalid but tolerated
// date may legitimately become valid: that is canonicalization.)
func TestCorpus_StringIsIdempotent(t *testing.T) {
	for _, line := range corpusLines(t) {
		d, err := Parse(line)
		out := d.String()
		again, err2 := Parse(out)
		assert.Equal(t, out, again.String(), "String not idempotent for %q", line)
		assert.Equal(t, d.Year(), again.Year(), "Year changed through String for %q", line)
		if err == nil {
			require.NoError(t, err2, "valid date %q became invalid as %q", line, out)
			assert.Equal(t, d.Calendar(), again.Calendar(), line)
			assert.Equal(t, d.Precision(), again.Precision(), line)
			assert.Equal(t, d.Qualifier(), again.Qualifier(), line)
			assert.Equal(t, d.IsRange(), again.IsRange(), line)
		}
	}
}

// standardGEDCOMRegexp matches a standard single GEDCOM Gregorian date body:
// optional qualifier, optional day, optional English month name, 3–4 digit year.
var standardGEDCOMRegexp = regexp.MustCompile(`(?i)^(?:(?:ABT|BEF|AFT|CAL)\s+)?(?:(?:\d{1,2}\s+)?(?:JAN(?:UARY)?|FEB(?:RUARY)?|MAR(?:CH)?|APR(?:IL)?|MAY|JUN(?:E)?|JUL(?:Y)?|AUG(?:UST)?|SEP(?:T|TEMBER)?|OCT(?:OBER)?|NOV(?:EMBER)?|DEC(?:EMBER)?)\.?\s+)?\d{3,4}$`)

// TestCorpus_StandardFormsCanonicalize: every standard GEDCOM date body in
// the corpus, whatever its letter case, canonicalizes to a valid GLX date
// with the oracle's year and a precision matching its components.
func TestCorpus_StandardFormsCanonicalize(t *testing.T) {
	checked := 0
	for _, line := range corpusLines(t) {
		if !standardGEDCOMRegexp.MatchString(line) {
			continue
		}
		want := oracleYear(line)
		if want == 0 {
			continue // year 0000 or a 3-digit year: not covered by the oracle
		}
		checked++
		d, _ := Parse(line)
		canon, err := Parse(d.String())
		require.NoError(t, err, "standard GEDCOM date %q did not canonicalize (got %q)", line, d.String())
		assert.Equal(t, want, canon.Year(), line)

		fields := strings.Fields(firstComponent(line))
		body := len(fields)
		if qualifierKeywords[keywordOf(fields[0])] != QualifierNone {
			body--
		}
		wantPrecision := []Precision{PrecisionYear, PrecisionMonth, PrecisionDay}[body-1]
		assert.Equal(t, wantPrecision, canon.Precision(), line)
	}
	assert.Greater(t, checked, 500, "corpus should contain many standard GEDCOM dates")
}

// FuzzParse checks the parser's invariants on arbitrary input: no panic,
// String is idempotent, and a valid date round-trips with the same year.
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"1850", "1850-03-15", "ABT 1850", "BET 1880 AND 1890", "FROM 1900",
		"INT 1850 (text)", "HEBREW 15 TSH 5765", "1 JANUARY 1900", "15/01/1900",
		"JULIAN BET 1700 AND 1710", "MYCAL 12", "BET AND", "FROM TO", "(", "-",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		d, err := Parse(input)
		out := d.String()
		again, err2 := Parse(out)
		if again.String() != out {
			t.Fatalf("String not idempotent: %q → %q → %q", input, out, again.String())
		}
		if err == nil && err2 != nil {
			t.Fatalf("valid %q rendered as invalid %q: %v", input, out, err2)
		}
		if err == nil && again.Year() != d.Year() {
			t.Fatalf("year changed through String: %q", input)
		}
		if strings.TrimSpace(input) == "" && !d.IsZero() {
			t.Fatalf("blank input must parse to the zero Date")
		}
	})
}
