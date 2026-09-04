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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/genealogix/glx/go-glx/glxdate"
)

// TestGEDCOMDateAdapters checks the DateString adapters delegate to glxdate;
// the behavior itself is pinned in glxdate/gedcom_test.go and the golden.
func TestGEDCOMDateAdapters(t *testing.T) {
	assert.Equal(t, DateString("JULIAN 1731-03-15"), parseGEDCOMDate("@#DJULIAN@ 15 March 1731"))
	assert.Equal(t, DateString("15/01/1900"), parseGEDCOMDate("15/01/1900"))
	assert.Equal(t, DateString(""), parseGEDCOMDate("  "))

	assert.Equal(t, "@#DJULIAN@ 15 MAR 1731", formatGEDCOMDate("JULIAN 1731-03-15", GEDCOM70))
	assert.Equal(t, "15/01/1900", formatGEDCOMDate("15/01/1900", GEDCOM551))
	assert.Empty(t, formatGEDCOMDate("", GEDCOM70))
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
