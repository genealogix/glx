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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChronologyBounds_CalendarEquivalenceAndIntervals(t *testing.T) {
	for _, pair := range [][2]string{
		{"JULIAN 1645-10-12", "1645-10-22"},
		{"JULIAN 1700-02-29", "1700-03-11"},
		{"JULIAN 1582-10-04", "1582-10-14"},
		{"JULIAN 1900-02-28", "1900-03-12"},
		{"JULIAN 0001-01-01", "0001-12-30 BCE"},
	} {
		first, err := Parse(pair[0])
		require.NoError(t, err)
		second, err := Parse(pair[1])
		require.NoError(t, err)
		a, b := first.ChronologyBounds(2), second.ChronologyBounds(2)
		assert.True(t, a.Known)
		assert.Equal(t, b.Outer, a.Outer, pair)
	}
	date, err := Parse("JULIAN BEF 1645-10-12")
	require.NoError(t, err)
	bounds := date.ChronologyBounds(2)
	assert.Equal(t, math.MinInt, bounds.Outer.Start)
	day, _ := Parse("1645-10-22")
	assert.Equal(t, day.ChronologyBounds(2).Outer.Start, bounds.Outer.End)
	hebrew, _ := Parse("HEBREW 5765")
	assert.False(t, hebrew.ChronologyBounds(2).Known)
	julianYear, _ := Parse("JULIAN 1700")
	assert.Equal(t, 366, julianYear.ChronologyBounds(2).Outer.End-julianYear.ChronologyBounds(2).Outer.Start)
	gregorianYear, _ := Parse("1700")
	assert.Equal(t, 365, gregorianYear.ChronologyBounds(2).Outer.End-gregorianYear.ChronologyBounds(2).Outer.Start)
}

func TestChronologyBounds_CertainDurationStartsOnCalendarValidDay(t *testing.T) {
	for _, test := range []struct {
		duration string
		firstDay string
	}{
		{"FROM 1900-02 TO 1900-03", "1900-02-28"},
		{"FROM 2000-02 TO 2000-03", "2000-02-29"},
		{"JULIAN FROM 1900-02 TO 1900-03", "JULIAN 1900-02-29"},
	} {
		t.Run(test.duration, func(t *testing.T) {
			duration, err := Parse(test.duration)
			require.NoError(t, err)
			firstDay, err := Parse(test.firstDay)
			require.NoError(t, err)
			assert.Equal(t, firstDay.ChronologyBounds(0).Outer.Start, duration.ChronologyBounds(0).Inner.Start)
		})
	}
}

func TestChronologyBounds_NominalDayCannotAnchorAnInterval(t *testing.T) {
	for _, date := range []string{"1900-02-29", "BEF 1900-02-29", "AFT 1900-02-29", "FROM 1900-02-29 TO 1900-03-15"} {
		d, err := Parse(date)
		require.NoError(t, err, "the date remains readable as written")
		assert.False(t, d.ChronologyBounds(0).Known, date)
	}
	assert.True(t, MustParse("JULIAN 1900-02-29").ChronologyBounds(0).Known)
}
