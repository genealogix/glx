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

// TestParse_Canonical covers every form the spec's Date Format Standard
// defines. Each input must parse as valid and round-trip through String.
func TestParse_Canonical(t *testing.T) {
	tests := []struct {
		input     string
		calendar  Calendar
		qualifier Qualifier
		precision Precision
		year      int
		month     int
		day       int
		isRange   bool
		openEnded bool
		endYear   int
	}{
		{input: "1850", precision: PrecisionYear, year: 1850},
		{input: "1850-03", precision: PrecisionMonth, year: 1850, month: 3},
		{input: "1850-03-15", precision: PrecisionDay, year: 1850, month: 3, day: 15},
		{input: "0047", precision: PrecisionYear, year: 47},
		{input: "ABT 1850", qualifier: QualifierAbout, precision: PrecisionYear, year: 1850},
		{input: "BEF 1920-01-15", qualifier: QualifierBefore, precision: PrecisionDay, year: 1920, month: 1, day: 15},
		{input: "AFT 1880-06", qualifier: QualifierAfter, precision: PrecisionMonth, year: 1880, month: 6},
		{input: "CAL 1850", qualifier: QualifierCalculated, precision: PrecisionYear, year: 1850},
		{input: "INT 1850-03-15 (15th March 1850)", qualifier: QualifierInterpreted, precision: PrecisionDay, year: 1850, month: 3, day: 15},
		{input: "BET 1880 AND 1890", precision: PrecisionYear, year: 1880, isRange: true, endYear: 1890},
		{input: "FROM 1900 TO 1950", precision: PrecisionYear, year: 1900, isRange: true, endYear: 1950},
		{input: "FROM 1900", precision: PrecisionYear, year: 1900, isRange: true, openEnded: true},
		{input: "JULIAN 1731-03-15", calendar: CalendarJulian, precision: PrecisionDay, year: 1731, month: 3, day: 15},
		{input: "JULIAN ABT 1731", calendar: CalendarJulian, qualifier: QualifierAbout, precision: PrecisionYear, year: 1731},
		{input: "HEBREW 15 TSH 5765", calendar: CalendarHebrew, precision: PrecisionYear, year: 5765},
		{input: "HEBREW ABT 5765", calendar: CalendarHebrew, qualifier: QualifierAbout, precision: PrecisionYear, year: 5765},
		{input: "HEBREW BET 15 TSH 5765 AND 15 TSH 5766", calendar: CalendarHebrew, precision: PrecisionYear, year: 5765, isRange: true, endYear: 5766},
		{input: "FRENCH_R 1 VEND 0012", calendar: CalendarFrenchRepublican, precision: PrecisionYear, year: 12},
		{input: "FRENCH_R FROM 1 VEND 0010 TO 1 VEND 0012", calendar: CalendarFrenchRepublican, precision: PrecisionYear, year: 10, isRange: true, endYear: 12},
		{input: "ROMAN 12", calendar: CalendarOther, precision: PrecisionYear, year: 12},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := Parse(tt.input)
			require.NoError(t, err)
			assert.True(t, d.Valid())
			assert.Equal(t, tt.input, d.String(), "canonical input must round-trip")
			assert.Equal(t, tt.input, d.Raw())
			assert.Equal(t, tt.calendar, d.Calendar())
			assert.Equal(t, tt.qualifier, d.Qualifier())
			assert.Equal(t, tt.precision, d.Precision())
			assert.Equal(t, tt.year, d.Year())
			assert.Equal(t, tt.isRange, d.IsRange())
			assert.Equal(t, tt.openEnded, d.IsOpenEnded())

			month, hasMonth := d.Month()
			assert.Equal(t, tt.month != 0, hasMonth)
			assert.Equal(t, tt.month, month)
			day, hasDay := d.Day()
			assert.Equal(t, tt.day != 0, hasDay)
			assert.Equal(t, tt.day, day)

			if tt.isRange {
				assert.Equal(t, tt.year, d.Start().Year())
				assert.False(t, d.Start().IsRange())
				assert.Equal(t, tt.endYear, d.End().Year())
				assert.Equal(t, tt.openEnded, d.End().IsZero())
			} else {
				assert.True(t, d.Equal(d.Start()))
				assert.True(t, d.End().IsZero())
			}
		})
	}
}

// TestParse_Tolerated covers inputs that are not canonical but whose
// components are unambiguous. They parse with an error, expose the right
// components, and String renders the canonical form.
func TestParse_Tolerated(t *testing.T) {
	tests := []struct {
		input     string
		canonical string
		year      int
		precision Precision
	}{
		// The bug in #1025: full and mixed-case month names.
		{"1 JANUARY 1900", "1900-01-01", 1900, PrecisionDay},
		{"15 March 2020", "2020-03-15", 2020, PrecisionDay},
		{"15 MAR 1850", "1850-03-15", 1850, PrecisionDay},
		{"15 mar 1850", "1850-03-15", 1850, PrecisionDay},
		{"3 Sept. 1850", "1850-09-03", 1850, PrecisionDay},
		{"MAR 1850", "1850-03", 1850, PrecisionMonth},
		{"March 1850", "1850-03", 1850, PrecisionMonth},
		{"APRIL 1688", "1688-04", 1688, PrecisionMonth},
		{"March 15, 1850", "1850-03-15", 1850, PrecisionDay},
		{"15 MAR 800", "0800-03-15", 800, PrecisionDay},
		{"5 JAN 476", "0476-01-05", 476, PrecisionDay},
		// Keyword case and punctuation.
		{"Abt 1850", "ABT 1850", 1850, PrecisionYear},
		{"abt 1850", "ABT 1850", 1850, PrecisionYear},
		{"ABT. 1850", "ABT 1850", 1850, PrecisionYear},
		{"Bet 1880 and 1890", "BET 1880 AND 1890", 1880, PrecisionYear},
		{"Aft 12 May 1880", "AFT 1880-05-12", 1880, PrecisionDay},
		{"From Jan 1900 to Feb 1901", "FROM 1900-01 TO 1901-02", 1900, PrecisionMonth},
		{"JULIAN 15 MAR 1731", "JULIAN 1731-03-15", 1731, PrecisionDay},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := Parse(tt.input)
			require.Error(t, err)
			assert.False(t, d.Valid())
			assert.Equal(t, tt.canonical, d.String())
			assert.Equal(t, tt.year, d.Year())
			assert.Equal(t, tt.precision, d.Precision())

			canon, err := Parse(d.String())
			require.NoError(t, err, "canonical form must be valid")
			assert.Equal(t, tt.year, canon.Year())
			assert.Equal(t, tt.precision, canon.Precision())
		})
	}
}

// TestParse_Whitespace: surrounding and repeated whitespace is not an error,
// and String normalizes it.
func TestParse_Whitespace(t *testing.T) {
	for input, want := range map[string]string{
		"  1850-03-15  ":      "1850-03-15",
		"ABT    1850":         "ABT 1850",
		"BET 1880  AND\t1890": "BET 1880 AND 1890",
	} {
		d, err := Parse(input)
		require.NoError(t, err, input)
		assert.Equal(t, want, d.String())
		assert.Equal(t, strings.TrimSpace(input), d.Raw())
	}
}

// TestParse_Preserved covers malformed input that must be preserved
// verbatim (never guessed) while still yielding a best-effort year.
func TestParse_Preserved(t *testing.T) {
	tests := []struct {
		input string
		year  int
	}{
		// Numeric day/month order is ambiguous: never interpreted, but the
		// 4-digit year is still recovered.
		{"15/01/1900", 1900},
		{"01/15/1900", 1900},
		{"15-03-1850", 1850},
		{"1850/03/15", 1850},
		{"1850.03.15", 1850},
		// Dual years, dual-year BCE, glued or doubtful text, free text.
		{"1731/32", 1731},
		{"ABT 1731/32", 1731},
		{"1401/8 B.C.", -1401},
		{"by 1850", 1850},
		{"sometime in 1850", 1850},
		{"1850?", 1850},
		{"(c.1850)", 1850},
		{"1880 - 1890", 1880},
		{"1850 or 1851", 1850},
		{"BET 1880", 1880},
		{"FROM 1900 TO", 1900},
		// A start with no year of its own shares the end's year, but only
		// when the end is exact does the range canonicalize (see
		// TestParse_KeywordSynonyms); a raw end keeps the range raw.
		{"BET JUL AND SEP 1857?", 1857},
		{"FROM MAR TO MAY 1900?", 1900},
		// Digits glued to letters and day-of-month tokens are never a year.
		{"(10 Aug)", 0},
		{"(2nd son)", 0},
		// A 4-digit run is a year even when glued to letters.
		{"10 APR1828", 1828},
		{"(c.1893twin)", 1893},
		{"2/18/1600or 1601", 1600},
		{"ABT", 0},
		{"unknown", 0},
		{"", 0},
		{"18500", 0},
		{"1850-3", 1850},
		{"1850-13", 1850},
		{"1850-02-30", 1850},
		{"0000", 0},
		{"HEBREW TSH", 0},
		{"TO 1900", 1900},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := Parse(tt.input)
			if tt.input == "" {
				require.NoError(t, err)
				assert.True(t, d.IsZero())

				return
			}
			require.Error(t, err)
			var perr *ParseError
			require.ErrorAs(t, err, &perr)
			assert.Equal(t, tt.input, perr.Input)
			assert.NotEmpty(t, perr.Reason)
			assert.Equal(t, tt.year, d.Year())
			assert.Equal(t, tt.input, d.String(), "preserved input must render verbatim")
		})
	}
}

// TestYear_IssueTable is the reproduction table from #1025.
func TestYear_IssueTable(t *testing.T) {
	tests := map[string]int{
		"1900-01-15":     1900,
		"1 JANUARY 1900": 1900,
		"15 March 2020":  2020,
		"15/01/1900":     1900,
	}
	for input, want := range tests {
		d, _ := Parse(input)
		assert.Equal(t, want, d.Year(), input)
	}
}

func TestDate_InterpretedText(t *testing.T) {
	d := MustParse("INT 1850-03-15 (15th March 1850)")
	text, ok := d.InterpretedText()
	assert.True(t, ok)
	assert.Equal(t, "15th March 1850", text)

	d = MustParse("INT 1850")
	_, ok = d.InterpretedText()
	assert.False(t, ok)
	assert.Equal(t, "INT 1850", d.String())

	d, err := Parse("INT (no date)")
	require.Error(t, err)
	assert.Equal(t, 0, d.Year())
}

func TestDate_CalendarName(t *testing.T) {
	assert.Empty(t, MustParse("1850").CalendarName())
	assert.Equal(t, "JULIAN", MustParse("JULIAN 1850").CalendarName())
	assert.Equal(t, "MYCAL", MustParse("MYCAL 1850").CalendarName())
	assert.Equal(t, CalendarOther, MustParse("MYCAL 1850").Calendar())

	rng := MustParse("JULIAN BET 1700 AND 1710")
	assert.Equal(t, "JULIAN 1700", rng.Start().String())
	assert.Equal(t, "JULIAN 1710", rng.End().String())
	assert.Equal(t, CalendarJulian, rng.End().Calendar())
}

func TestNew(t *testing.T) {
	d := New(CalendarGregorian, 1850, 3, 15)
	assert.True(t, d.Valid())
	assert.Equal(t, "1850-03-15", d.String())
	assert.Equal(t, PrecisionDay, d.Precision())

	assert.Equal(t, "1850-03", New(CalendarGregorian, 1850, 3, 0).String())
	assert.Equal(t, "JULIAN 0850", New(CalendarJulian, 850, 0, 0).String())
	assert.Equal(t, "HEBREW 5765", New(CalendarHebrew, 5765, 3, 15).String())

	bad := New(CalendarGregorian, 1850, 13, 1)
	assert.False(t, bad.Valid())
	assert.Equal(t, 1850, bad.Year())

	assert.True(t, d.Equal(MustParse(d.String())), "New and Parse agree")
}

func TestNewRange(t *testing.T) {
	r, err := NewRange(MustParse("1880"), MustParse("1890"))
	require.NoError(t, err)
	assert.Equal(t, "BET 1880 AND 1890", r.String())
	assert.True(t, r.Valid())
	assert.True(t, r.Equal(MustParse(r.String())))

	open, err := NewRange(MustParse("1900-05"), Date{})
	require.NoError(t, err)
	assert.Equal(t, "FROM 1900-05", open.String())
	assert.True(t, open.IsOpenEnded())

	_, err = NewRange(MustParse("JULIAN 1700"), MustParse("1710"))
	require.ErrorIs(t, err, ErrRangeMismatch)
	_, err = NewRange(r, MustParse("1900"))
	require.ErrorIs(t, err, ErrRangeMismatch)
	_, err = NewRange(Date{}, MustParse("1900"))
	require.ErrorIs(t, err, ErrRangeMismatch)
}

func TestEqual(t *testing.T) {
	assert.True(t, MustParse("1850").Equal(MustParse("1850")))
	assert.True(t, MustParse("1850").Equal(New(CalendarGregorian, 1850, 0, 0)))
	assert.False(t, MustParse("1850").Equal(MustParse("1850-01")))
	assert.False(t, MustParse("1850").Equal(MustParse("JULIAN 1850")))
	var zero Date
	assert.True(t, zero.Equal(Date{}))
	assert.False(t, zero.Equal(MustParse("1850")))

	d, _ := Parse("15 March 1850")
	assert.True(t, d.Equal(MustParse("1850-03-15")), "Equal compares canonical forms")
}

func TestMustParse_Panics(t *testing.T) {
	assert.Panics(t, func() { MustParse("15 March 2020") })
	assert.NotPanics(t, func() { MustParse("2020-03-15") })
}

func TestParseError_Message(t *testing.T) {
	_, err := Parse("March 1850")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"March 1850"`)
	assert.Contains(t, err.Error(), "1850-03")

	_, err = Parse("Abt 1850")
	assert.Contains(t, err.Error(), "must be written ABT")

	_, err = Parse("1850-13")
	assert.Contains(t, err.Error(), "month must be between 01 and 12")

	var perr *ParseError
	require.ErrorAs(t, err, &perr)
}

func TestMonthNumber(t *testing.T) {
	for _, name := range []string{"JAN", "Jan", "jan", "January", "JANUARY", "Jan."} {
		m, ok := MonthNumber(name)
		assert.True(t, ok, name)
		assert.Equal(t, 1, m, name)
	}
	m, ok := MonthNumber("SEPT")
	assert.True(t, ok)
	assert.Equal(t, 9, m)
	for _, name := range []string{"TSH", "VEND", "", "Mars", "1", "JANUAR"} {
		_, ok := MonthNumber(name)
		assert.False(t, ok, name)
	}
}

func TestSplitCalendarPrefix(t *testing.T) {
	tests := []struct {
		input  string
		prefix string
		body   string
	}{
		{"JULIAN 1731-03-15", "JULIAN", "1731-03-15"},
		{"HEBREW 15 TSH 5765", "HEBREW", "15 TSH 5765"},
		{"FRENCH_R 1 VEND 0012", "FRENCH_R", "1 VEND 0012"},
		{"MYCAL 1850", "MYCAL", "1850"},
		{"1731-03-15", "", "1731-03-15"},
		{"ABT 1731", "", "ABT 1731"},
		{"BET 1880 AND 1890", "", "BET 1880 AND 1890"},
		{"GREGORIAN 1850", "", "GREGORIAN 1850"},
		{"SPRING 1850", "", "SPRING 1850"},
		{"MAR 1850", "", "MAR 1850"},
		{"March 1850", "", "March 1850"},
		{"APRIL 1688", "", "APRIL 1688"},
		{"SEPTEMBER 1850", "", "SEPTEMBER 1850"},
		{"JULIAN", "", "JULIAN"},
		{"", "", ""},
	}
	for _, tt := range tests {
		prefix, body := SplitCalendarPrefix(tt.input)
		assert.Equal(t, tt.prefix, prefix, tt.input)
		assert.Equal(t, tt.body, body, tt.input)
	}
}

func TestEnumStrings(t *testing.T) {
	assert.Equal(t, "GREGORIAN", CalendarGregorian.String())
	assert.Empty(t, CalendarGregorian.Prefix())
	assert.Equal(t, "JULIAN", CalendarJulian.Prefix())
	assert.Equal(t, "OTHER", CalendarOther.String())
	assert.Empty(t, CalendarOther.Prefix())
	assert.Equal(t, "day", PrecisionDay.String())
	assert.Equal(t, "ABT", QualifierAbout.String())
	assert.Empty(t, QualifierNone.Keyword())
}
