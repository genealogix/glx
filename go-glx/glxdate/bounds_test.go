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

	"github.com/stretchr/testify/require"
)

func TestTimingBounds(t *testing.T) {
	cases := []struct {
		a, b string
		want Overlap
	}{
		{"FROM 1850 TO 1870", "FROM 1870 TO 1900", NoOverlap},
		{"FROM 1850 TO 1900", "1875", DefiniteOverlap},
		{"1851", "1851", PossibleOverlap},
		{"1851-01-01", "1851-01-01", DefiniteOverlap},
		{"1851-01", "1851-02", NoOverlap},
		{"AFT 1900", "BEF 1901", NoOverlap},
		{"BEF 1900", "1880", PossibleOverlap},
		{"AFT 1900", "1950", PossibleOverlap},
		{"FROM 1900", "1950", PossibleOverlap},
		{"TO 1900", "1850", PossibleOverlap},
		{"ABT 1900", "1901", NoOverlap},
		{"CAL 1900", "1900", PossibleOverlap},
		{"INT 1900 (estimated)", "1900", PossibleOverlap},
		{"FROM 1920 TO 1870", "1900", PossibleOverlap},
		{"FROM spring TO 1850", "1800", PossibleOverlap},
		{"FROM 1850 TO spring", "1900", PossibleOverlap},
		{"FROM spring TO summer", "1900", NoOverlap},
		{"", "1900", NoOverlap},
		{"spring", "1900", NoOverlap},
		{"FROM 1900 TO 1900", "FROM 1900 TO 1900", PossibleOverlap},
		{"JULIAN 1900-02-29", "JULIAN 1900-03-01", NoOverlap},
		{"JULIAN 1900-01-01", "1900-01-01", PossibleOverlap},
		{"0001 BCE", "0001", NoOverlap},
		{"FROM 0044 BCE TO 0001 BCE", "0020 BCE", DefiniteOverlap},
	}
	for _, tc := range cases {
		t.Run(tc.a+"/"+tc.b, func(t *testing.T) {
			a, _ := Parse(tc.a)
			b, _ := Parse(tc.b)
			require.Equal(t, tc.want, CompareTiming(a, b))
			require.Equal(t, tc.want, CompareTiming(b, a))
		})
	}
}

func TestValueBoundsApproximation(t *testing.T) {
	for _, q := range []string{"ABT", "EST", "CAL"} {
		a, _ := Parse(q + " 1850")
		b, _ := Parse("1852")
		c, _ := Parse("1853")
		require.True(t, a.ValueBounds(2).Outer.Intersects(b.ValueBounds(2).Outer))
		require.False(t, a.ValueBounds(2).Outer.Intersects(c.ValueBounds(2).Outer))
		require.True(t, a.ValueBounds(3).Outer.Intersects(c.ValueBounds(3).Outer))
		require.Equal(t, a.ValueBounds(0), a.ValueBounds(-1))
	}
	a, _ := Parse("ABT 0001")
	b, _ := Parse("0001 BCE")
	require.True(t, a.ValueBounds(1).Outer.Intersects(b.ValueBounds(1).Outer))
}
