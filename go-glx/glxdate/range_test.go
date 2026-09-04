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

// TestParse_ToRange: GEDCOM's open-start period is a GLX form; the only
// year present is the end year, and Year reports it.
func TestParse_ToRange(t *testing.T) {
	d, err := Parse("TO 1950")
	require.NoError(t, err)
	assert.True(t, d.IsRange())
	assert.True(t, d.IsOpenEnded())
	assert.True(t, d.IsOpenStart())
	assert.True(t, d.Start().IsZero())
	assert.Equal(t, 1950, d.End().Year())
	assert.Equal(t, 1950, d.Year())
	assert.Equal(t, "TO 1950", d.String())
	assert.Equal(t, "TO 1950", d.GEDCOM())

	assert.Equal(t, "TO 1950-12-31", Canonicalize("to 31 Dec 1950"))
	assert.Equal(t, "JULIAN TO 1700", FromGEDCOM("@#DJULIAN@ TO 1700"))

	for _, bad := range []string{"TO", "TO 1900 TO 1950", "TO 1900 AND 1950"} {
		_, err := Parse(bad)
		require.Error(t, err, bad)
	}
}

// TestParse_RangeBorrowGuard: a yearless start borrows the end's year only
// when it can plausibly fall in it, and canonicalizes only when the end has
// at least month precision.
func TestParse_RangeBorrowGuard(t *testing.T) {
	for input, want := range map[string]struct {
		canonical string
		year      int
	}{
		"BET JUL AND SEP 1857":       {"BET 1857-07 AND 1857-09", 1857},
		"BET 07 OCT AND 08 NOV 1260": {"BET 1260-10-07 AND 1260-11-08", 1260},
		"BET 15 DEC AND 5 JAN 1860":  {"BET 15 DEC AND 5 JAN 1860", 0}, // December of the previous year
		"FROM DEC TO JAN 1860":       {"FROM DEC TO JAN 1860", 0},
		"BET 15 JUL AND 1857":        {"BET 15 JUL AND 1857", 1857},        // year-only end: no invented day precision
		"BET 31 MAR AND MAR 1900":    {"BET 1900-03-31 AND 1900-03", 1900}, // within the end's month
		"BET 1 APR AND MAR 1900":     {"BET 1 APR AND MAR 1900", 0},        // after the end's month
		"BET JUL AND SEP 0044 BCE":   {"BET JUL AND SEP 0044 BCE", -44},
		"BET 5 JAN AND 15 JAN 1860":  {"BET 1860-01-05 AND 1860-01-15", 1860},
	} {
		assert.Equal(t, want.canonical, Canonicalize(input), input)
		d, _ := Parse(input)
		assert.Equal(t, want.year, d.Year(), input)
	}
}

// TestMonthDay_OnlyWhenExact: out-of-range components are never reported.
func TestMonthDay_OnlyWhenExact(t *testing.T) {
	for _, input := range []string{"1850-13", "1850-00", "1850-02-30", "31 FEB 1850"} {
		d, err := Parse(input)
		require.Error(t, err, input)
		_, ok := d.Month()
		assert.False(t, ok, input)
		_, ok = d.Day()
		assert.False(t, ok, input)
		assert.Equal(t, 1850, d.Year(), input)
	}
}

// TestParse_RawCalendarAnnotatedYear: a raw-calendar body with a trailing
// annotation is invalid, but its year is still the last long number, never
// a day of month.
func TestParse_RawCalendarAnnotatedYear(t *testing.T) {
	d, err := Parse("HEBREW 15 TSH 5765 (approx)")
	require.Error(t, err)
	assert.Equal(t, 5765, d.Year())

	d, err = Parse("HEBREW 15 TSH")
	require.Error(t, err)
	assert.Equal(t, 0, d.Year())
}

// TestParse_EraMustFollowYear: an era marker elsewhere in the text is not an
// era.
func TestParse_EraMustFollowYear(t *testing.T) {
	for input, want := range map[string]int{
		"1900, Vancouver BC":     1900,
		"1850 (BC registration)": 1850,
		"abt. 1317 BC (or 934)":  -1317,
		"1401/8 B.C.":            -1401,
		"5? BC":                  -5,
	} {
		d, _ := Parse(input)
		assert.Equal(t, want, d.Year(), input)
	}
}

// TestNew_BCERawCalendar: a raw calendar cannot carry an era.
func TestNew_BCERawCalendar(t *testing.T) {
	assert.False(t, New(CalendarHebrew, -44, 0, 0).Valid())
	assert.True(t, New(CalendarJulian, -44, 0, 0).Valid())
}
