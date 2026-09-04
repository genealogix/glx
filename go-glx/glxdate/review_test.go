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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSplitCalendarPrefix_Whitespace: any whitespace separates the prefix.
func TestSplitCalendarPrefix_Whitespace(t *testing.T) {
	prefix, body := SplitCalendarPrefix("JULIAN\t1731-03-15")
	assert.Equal(t, PrefixJulian, prefix)
	assert.Equal(t, "1731-03-15", body)

	d, err := Parse("JULIAN\t 1731-03-15")
	require.NoError(t, err)
	assert.Equal(t, CalendarJulian, d.Calendar())
	assert.Equal(t, "JULIAN 1731-03-15", d.String())
}

// TestParse_FreeTextIsNotACalendar: upper-case words that start free-text
// dates are not unknown calendar prefixes, so such dates are invalid and
// the year comes from the Gregorian heuristic (first, not last, number).
func TestParse_FreeTextIsNotACalendar(t *testing.T) {
	for input, wantYear := range map[string]int{
		"AFTER 1839 BEFORE 1840": 1839,
		"ABOUT 1839 OR 1840":     1839,
		"LIVING 1515":            1515,
		"PRIOR 1855":             1855,
		"UNKNOWN 87":             87,
		"CLEARED 11/88":          11,
	} {
		d, err := Parse(input)
		require.Error(t, err, input)
		assert.Equal(t, CalendarGregorian, d.Calendar(), input)
		assert.Equal(t, wantYear, d.Year(), input)
	}
	// Placeholder words render verbatim; a keyword synonym is normalized
	// even when the rest of the body stays raw.
	assert.Equal(t, "LIVING 1515", stringOf("LIVING 1515"))
	assert.Equal(t, "AFT 1839 BEFORE 1840", stringOf("AFTER 1839 BEFORE 1840"))

	d, err := Parse("_ROMAN 15 MAR 1731")
	require.NoError(t, err)
	assert.Equal(t, "_ROMAN", d.CalendarName())
}

// stringOf returns Parse(s).String(), ignoring the error.
func stringOf(s string) string {
	d, _ := Parse(s)

	return d.String()
}

// TestParse_RawCalendarNeedsTrailingYear: a raw-calendar body whose last
// token is not a number has no year; the day of month is never promoted.
func TestParse_RawCalendarNeedsTrailingYear(t *testing.T) {
	for _, input := range []string{"HEBREW 15 TSH", "FRENCH_R 1 VEND", "HEBREW TSH 5765 (approx)"} {
		d, err := Parse(input)
		require.Error(t, err, input)
		assert.Equal(t, 0, d.Year(), input)
		assert.Equal(t, PrecisionNone, d.Precision(), input)
		assert.Equal(t, input, d.String(), input)
	}

	d, err := Parse("HEBREW 15 TSH 5765")
	require.NoError(t, err)
	assert.Equal(t, 5765, d.Year())
}

// TestParse_RangeStartInheritsYearOnlyFromPartialMonth: "BET JUL AND SEP
// 1857" starts in 1857, but arbitrary text never borrows the end's year.
func TestParse_RangeStartInheritsYearOnlyFromPartialMonth(t *testing.T) {
	for input, wantYear := range map[string]int{
		"BET JUL AND SEP 1857":      1857,
		"BET 15 Jul AND SEP 1857":   1857,
		"BET unknown AND 1857":      0,
		"FROM nonsense TO 2020":     0,
		"BET CSA 16 MAY AND 1862":   0,
		"BET 1850 AND 1860":         1850,
		"BET JUL 1850 AND SEP 1860": 1850,
	} {
		d, _ := Parse(input)
		assert.Equal(t, wantYear, d.Year(), input)
	}
}

// TestParse_EmptyInterpretedText: empty parentheses are not dropped.
func TestParse_EmptyInterpretedText(t *testing.T) {
	for _, input := range []string{"INT 1850 ()", "INT 1850 ( )"} {
		d, err := Parse(input)
		require.Error(t, err, input)
		assert.Equal(t, input, d.String(), input)
		assert.Equal(t, 1850, d.Year(), input)
		_, ok := d.InterpretedText()
		assert.False(t, ok, input)
	}
	assert.Equal(t, "INT 1850 ()", Canonicalize("INT 1850 ()"))
}

// TestNew_InvalidComponentsRenderRaw: an out-of-range component makes the
// date invalid, and it renders as written in both GLX and GEDCOM form.
func TestNew_InvalidComponentsRenderRaw(t *testing.T) {
	d := New(CalendarGregorian, 1850, 13, 1)
	assert.False(t, d.Valid())
	assert.Equal(t, "1850-13-01", d.String())
	assert.NotPanics(t, func() { _ = d.GEDCOM() })
	assert.Equal(t, "1850-13-01", d.GEDCOM())

	d = New(CalendarGregorian, 1850, 2, 31)
	assert.False(t, d.Valid())
	assert.Equal(t, "1850-02-31", d.GEDCOM())

	ok := New(CalendarJulian, 1850, 2, 29)
	assert.True(t, ok.Valid())
	assert.Equal(t, "@#DJULIAN@ 29 FEB 1850", ok.GEDCOM())
}

// TestNew_OtherCalendarIsInvalid: New cannot name an unknown calendar.
func TestNew_OtherCalendarIsInvalid(t *testing.T) {
	d := New(CalendarOther, 1850, 0, 0)
	assert.False(t, d.Valid())
	assert.Equal(t, CalendarOther, d.Calendar())
	assert.False(t, d.Equal(New(CalendarGregorian, 1850, 0, 0)))
}

// TestEqual_ComparesCalendar: equality requires the same calendar and name.
func TestEqual_ComparesCalendar(t *testing.T) {
	assert.True(t, MustParse("_ROMAN 1000").Equal(MustParse("_ROMAN 1000")))
	assert.False(t, MustParse("_ROMAN 1000").Equal(MustParse("_MAYAN 1000")))
	assert.False(t, MustParse("JULIAN 1731").Equal(MustParse("1731")))
	assert.True(t, MustParse("1731").Equal(New(CalendarGregorian, 1731, 0, 0)))
}

// TestNewRange_Rejections: qualified endpoints and differing unknown
// calendars are rejected instead of silently relabeled.
func TestNewRange_Rejections(t *testing.T) {
	_, err := NewRange(MustParse("ABT 1880"), MustParse("1890"))
	require.ErrorIs(t, err, ErrRangeMismatch)

	_, err = NewRange(MustParse("1880"), MustParse("INT 1890 (text)"))
	require.ErrorIs(t, err, ErrRangeMismatch)

	_, err = NewRange(MustParse("_ROMAN 1000"), MustParse("_MAYAN 1100"))
	require.ErrorIs(t, err, ErrRangeMismatch)

	r, err := NewRange(MustParse("_ROMAN 1000"), MustParse("_ROMAN 1100"))
	require.NoError(t, err)
	assert.Equal(t, "_ROMAN BET 1000 AND 1100", r.String())
}
