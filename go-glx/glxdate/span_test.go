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
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// relationOf names the one relation that holds between a and b, failing the
// test unless exactly one of the seven does.
func relationOf(t *testing.T, a, b Span) string {
	t.Helper()
	held := []string{}
	for name, holds := range map[string]bool{
		"precedes":     a.Precedes(b),
		"precededBy":   b.Precedes(a),
		"overlaps":     a.Overlaps(b),
		"overlappedBy": b.Overlaps(a),
		"during":       a.During(b),
		"contains":     b.During(a),
		"equals":       a.Equals(b),
	} {
		if holds {
			held = append(held, name)
		}
	}
	require.Len(t, held, 1, "exactly one relation must hold between %+v and %+v, got %v", a, b, held)

	return held[0]
}

// spanOf parses best-effort: an invalid date still yields the components
// the parser could determine, which is what Span is defined over.
func spanOf(s string) Span {
	d, _ := Parse(s) //nolint:errcheck // best-effort components are the point of the test

	return d.Span()
}

func TestSpan_Relations(t *testing.T) {
	tests := []struct {
		a, b string
		want string
	}{
		// Simple dates at their written precision.
		{"1851", "1852", "precedes"},
		{"1852", "1851", "precededBy"},
		{"1851", "1851", "equals"},
		{"1851-03", "1851", "during"},
		{"1851", "1851-03", "contains"},
		{"1851-03-15", "1851-03", "during"},
		{"1851-03-15", "1851-03-16", "precedes"},
		{"1851-03", "1851-04", "precedes"},

		// Qualifiers other than BEF and AFT do not widen a date.
		{"ABT 1851", "1851", "equals"},
		{"EST 1851", "1851", "equals"},
		{"CAL 1851", "1851", "equals"},
		{"INT 1851", "1851", "equals"},
		{"ABT 1851", "1852", "precedes"},

		// Spellings and calendars.
		{"15 March 1851", "1851-03-15", "equals"},
		{"JULIAN 1851", "1851", "equals"},
		{"HEBREW 15 TSH 5765", "5765", "equals"},

		// Closed ranges include both endpoints; FROM…TO and BET…AND agree.
		{"FROM 1870 TO 1920", "1870", "contains"},
		{"FROM 1870 TO 1920", "1920", "contains"},
		{"FROM 1870 TO 1920", "1921", "precedes"},
		{"BET 1870 AND 1920", "FROM 1870 TO 1920", "equals"},
		{"BET 1840 AND 1855", "FROM 1850 TO 1860", "overlaps"},
		{"FROM 1850 TO 1860", "BET 1840 AND 1855", "overlappedBy"},
		{"FROM 298 TO 298", "298", "equals"},

		// Open ranges.
		{"FROM 1870", "1869", "precededBy"},
		{"FROM 1870", "1870", "contains"},
		{"TO 1860", "1861", "precedes"},
		{"TO 1860", "1860", "contains"},
		{"FROM 1870", "TO 1869", "precededBy"},
		{"FROM 1870", "TO 1870", "overlappedBy"},
		{"FROM 1870", "FROM 1880", "contains"},
		{"TO 1860", "TO 1850", "contains"},

		// BEF and AFT exclude the named year and are open on the far side.
		{"AFT 1900", "1900", "precededBy"},
		{"AFT 1900", "1901", "contains"},
		{"AFT 1900", "1950", "contains"},
		{"BEF 1900", "1900", "precedes"},
		{"BEF 1900", "1899", "contains"},
		{"BEF 1900", "1880", "contains"},
		{"AFT 1900", "AFT 1950", "contains"},
		{"BEF 1900", "BEF 1850", "contains"},
		{"BEF 1950", "AFT 1900", "overlaps"},
		{"AFT 1900", "BEF 1950", "overlappedBy"},
		{"AFT 1950", "BEF 1900", "precededBy"},
		{"AFT 1900", "BEF 1901", "precededBy"},
		{"AFT 1900", "BEF 1902", "overlappedBy"},

		// The excluded unit is the written precision, not the whole year.
		{"AFT 1900-03-15", "1900-03-15", "precededBy"},
		{"AFT 1900-03-15", "1900-03-16", "contains"},
		{"AFT 1900-03-15", "1900-03", "overlappedBy"},
		{"AFT 1900-01-01", "1900", "overlappedBy"},
		{"AFT 1900-03", "1900-03-31", "precededBy"},
		{"AFT 1900-03", "1900-04-01", "contains"},
		{"AFT 1900-12", "1900", "precededBy"},
		{"AFT 1900-12", "1901", "contains"},
		{"BEF 1900-03", "1900-03-01", "precedes"},
		{"BEF 1900-03", "1900-02", "contains"},
		{"BEF 1900-03", "1900-02-28", "contains"},
		{"BEF 1900-03-15", "1900-03-15", "precedes"},
		{"BEF 1900-03-15", "1900-03-14", "contains"},
		{"BEF 1900-01-01", "1900", "precedes"},
		{"BEF 1900-01-01", "1899", "contains"},

		// BEF and AFT against ranges.
		{"AFT 1900", "TO 1900", "precededBy"},
		{"AFT 1900", "TO 1901", "overlappedBy"},
		{"BEF 1900", "FROM 1900", "precedes"},
		{"BEF 1900", "FROM 1899", "overlaps"},
		{"AFT 1900", "FROM 1900 TO 1900", "precededBy"},
		{"AFT 1900", "BET 1900 AND 1901", "overlappedBy"},

		// Calendar prefixes and tolerated spellings keep the qualifier.
		{"JULIAN AFT 1900", "1950", "contains"},
		{"JULIAN AFT 1900", "1900", "precededBy"},
		{"JULIAN BEF 1900", "1900", "precedes"},
		{"bef 1900", "1900", "precedes"},
		{"AFTER 1900", "1900", "precededBy"},
		{"before 1900", "1900", "precedes"},

		// Unwidened qualifiers meet BEF and AFT exactly at the edge.
		{"AFT 1900", "ABT 1900", "precededBy"},
		{"BEF 1901", "ABT 1900", "contains"},

		// Shifted bounds land on real days, so facing BEF and AFT never meet
		// on a day that does not exist.
		{"AFT 1900-04-30", "BEF 1900-05-01", "precededBy"},
		{"AFT 1900-03-31", "1900-04-01", "contains"},
		{"AFT 1900-02-28", "1900-02-29", "precededBy"},
		{"AFT 1900-02-28", "1900-03-01", "contains"},
		{"AFT 1904-02-28", "1904-02-29", "contains"},
		{"JULIAN AFT 1900-02-28", "1900-02-29", "contains"},
		{"BEF 1900-03", "1900-02-28", "contains"},
		{"BEF 1900-03", "1900-02-29", "precedes"},
		{"BEF 1904-03", "1904-02-29", "contains"},

		// BCE years keep their order across the era boundary, and the step
		// from BEF or AFT skips the year 0 that neither calendar has.
		{"0045 BCE", "0044 BCE", "precedes"},
		{"0044-12-31 BCE", "0044-01-01 BCE", "precededBy"},
		{"0001 BCE", "0001", "precedes"},
		{"AFT 0001 BCE", "0001", "contains"},
		{"AFT 0001 BCE", "BEF 0001", "precededBy"},
		{"BEF 0001", "0001 BCE", "contains"},

		// Dates that cannot be placed span all time.
		{"", "1900", "contains"},
		{"", "", "equals"},
		{"spring", "AFT 1900", "contains"},
		{"FROM 1850 TO spring", "1900", "contains"},
		{"FROM 1850 TO spring", "1849", "precededBy"},
		{"FROM spring TO 1850", "1800", "contains"},
		{"FROM spring TO 1850", "1851", "precedes"},
		{"FROM 1920 TO 1870", "1900", "contains"},
		{"BET 1920 AND 1870", "", "equals"},
	}
	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			a, b := spanOf(tt.a), spanOf(tt.b)
			got := relationOf(t, a, b)
			require.Equal(t, tt.want, got)

			wantIntersects := got != "precedes" && got != "precededBy"
			require.Equal(t, wantIntersects, a.Intersects(b))
			require.Equal(t, wantIntersects, b.Intersects(a))
		})
	}
}

func TestSpan_Bounds(t *testing.T) {
	tests := []struct {
		date string
		want Span
	}{
		{"1900", Span{lo: 19000101, hi: 19001231}},
		{"1900-03", Span{lo: 19000301, hi: 19000331}},
		{"1900-02", Span{lo: 19000201, hi: 19000228}},
		{"1904-02", Span{lo: 19040201, hi: 19040229}},
		{"2000-02", Span{lo: 20000201, hi: 20000229}},
		{"JULIAN 1900-02", Span{lo: 19000201, hi: 19000229}},
		{"1900-03-15", Span{lo: 19000315, hi: 19000315}},
		{"AFT 1900", Span{lo: 19010101, hi: math.MaxInt}},
		{"AFT 1900-12", Span{lo: 19010101, hi: math.MaxInt}},
		{"AFT 1900-12-31", Span{lo: 19010101, hi: math.MaxInt}},
		{"AFT 1900-02-28", Span{lo: 19000301, hi: math.MaxInt}},
		{"JULIAN AFT 1900-02-28", Span{lo: 19000229, hi: math.MaxInt}},
		{"BEF 1900", Span{lo: math.MinInt, hi: 18991231}},
		{"BEF 1900-01", Span{lo: math.MinInt, hi: 18991231}},
		{"BEF 1900-01-01", Span{lo: math.MinInt, hi: 18991231}},
		{"BEF 1900-03-01", Span{lo: math.MinInt, hi: 19000228}},
		{"FROM 1870 TO 1920", Span{lo: 18700101, hi: 19201231}},
		{"FROM 1870", Span{lo: 18700101, hi: math.MaxInt}},
		{"TO 1860", Span{lo: math.MinInt, hi: 18601231}},
		{"0044-03-15 BCE", Span{lo: -440000 + 315, hi: -440000 + 315}},
		{"AFT 0001 BCE", Span{lo: 10101, hi: math.MaxInt}},
		{"BEF 0001", Span{lo: math.MinInt, hi: -10000 + 1231}},
		{"", allTime},
		{"FROM 1920 TO 1870", allTime},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			require.Equal(t, tt.want, spanOf(tt.date))
		})
	}
}

// TestSpan_RelationsAreExclusive checks the algebra itself on every pair of
// small spans, including open sides: exactly one relation holds each time,
// and Intersects agrees with the relation that does.
func TestSpan_RelationsAreExclusive(t *testing.T) {
	bounds := []int{math.MinInt, 0, 1, 2, 3, math.MaxInt}
	var spans []Span
	for _, lo := range bounds {
		for _, hi := range bounds {
			if lo <= hi {
				spans = append(spans, Span{lo: lo, hi: hi})
			}
		}
	}
	for _, a := range spans {
		for _, b := range spans {
			rel := relationOf(t, a, b)
			shareDay := a.lo <= b.hi && b.lo <= a.hi
			require.Equal(t, shareDay, a.Intersects(b), "%+v vs %+v (%s)", a, b, rel)
		}
	}
}
