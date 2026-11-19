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

func TestOrderingDay(t *testing.T) {
	day := func(s string) int {
		t.Helper()
		d, err := Parse(s)
		require.NoError(t, err, s)
		n, ok := d.OrderingDay()
		require.True(t, ok, s)

		return n
	}

	// Known Julian Day Numbers.
	assert.Equal(t, 2451545, day("2000-01-01"))
	assert.Equal(t, 2299161, day("1582-10-15"))
	assert.Equal(t, 2299160, day("JULIAN 1582-10-04"), "the day before the Gregorian reform")

	// A year-only date starts on 1 January; BEF ends the day before.
	assert.Equal(t, day("1690-01-01"), day("1690"))
	assert.Equal(t, day("1689-12-31"), day("BEF 1690"))
	assert.Equal(t, day("1690-12-31"), day("TO 1690"), "TO includes the year it names")
	assert.Equal(t, day("1691-01-01"), day("AFT 1690"))
	assert.Equal(t, day("1690-01-01"), day("BET 1690 AND 1700"))

	// Feb 29 keeps its place even in a common year.
	assert.Less(t, day("1700-02-28"), day("1700-02-29"))
	assert.LessOrEqual(t, day("1700-02-29"), day("1700-03-01"))

	for _, s := range []string{"", "HEBREW 5460", "BET 1700 AND 1690"} {
		d, _ := Parse(s)
		_, ok := d.OrderingDay()
		assert.False(t, ok, s)
	}
}
