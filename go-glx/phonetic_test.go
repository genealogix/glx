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
)

func TestSoundex(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Robert", "R163"},
		{"Rupert", "R163"},
		{"Ashcraft", "A261"},
		{"Ashcroft", "A261"},
		{"Tymczak", "T522"},
		{"Pfister", "P236"},
		{"Smith", "S530"},
		{"Smyth", "S530"},
		{"Miller", "M460"},
		{"Myller", "M460"},
		{"Mueller", "M460"},
		{"Johannes", "J520"},
		{"Johannis", "J520"},
		{"Johanns", "J520"},
		{"Johnson", "J525"},
		{"Webb", "W100"},
		{"Wab", "W100"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Soundex(tt.name)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSoundexMatch(t *testing.T) {
	assert.True(t, SoundexMatch("Miller", "Myller"))
	assert.True(t, SoundexMatch("Smith", "Smyth"))
	assert.True(t, SoundexMatch("Johannes", "Johannis"))
	assert.False(t, SoundexMatch("Miller", "Smith"))
	assert.False(t, SoundexMatch("Johannes", "Johnson"))
	assert.False(t, SoundexMatch("", "Smith"))
}

func TestPhoneticPersonSearch(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-miller":  {Properties: map[string]any{"name": "Jane Miller"}},
			"person-myller":  {Properties: map[string]any{"name": "John Myller"}},
			"person-mueller": {Properties: map[string]any{"name": "Hans Mueller"}},
			"person-smith":   {Properties: map[string]any{"name": "Robert Smith"}},
		},
	}

	matches := PhoneticPersonSearch(archive, "Miller")
	assert.Len(t, matches, 3, "should match Miller, Myller, Mueller")

	ids := make(map[string]bool)
	for _, m := range matches {
		ids[m.PersonID] = true
	}
	assert.True(t, ids["person-miller"])
	assert.True(t, ids["person-myller"])
	assert.True(t, ids["person-mueller"])
	assert.False(t, ids["person-smith"])
}

func TestPhoneticPersonSearch_NoMatches(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-smith": {Properties: map[string]any{"name": "Robert Smith"}},
		},
	}

	matches := PhoneticPersonSearch(archive, "Miller")
	assert.Empty(t, matches)
}

func TestPhoneticPersonSearch_EmptyQuery(t *testing.T) {
	archive := &GLXFile{
		Persons: map[string]*Person{
			"person-smith": {Properties: map[string]any{"name": "Robert Smith"}},
		},
	}

	matches := PhoneticPersonSearch(archive, "")
	assert.Empty(t, matches)
}
