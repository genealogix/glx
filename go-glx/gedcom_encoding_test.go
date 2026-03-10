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
	"bytes"
	"io"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToUTF8_CP1252(t *testing.T) {
	// "1 TITL König" in CP1252: ö = 0xF6
	input := []byte("0 HEAD\r\n1 CHAR ANSI\r\n0 @I1@ INDI\r\n1 TITL K\xf6nig\r\n")

	got := convertToUTF8(input)

	assert.Contains(t, string(got), "König")
	assert.NotContains(t, string(got), "\xf6")
}

func TestConvertToUTF8_CP1252Explicit(t *testing.T) {
	// CHAR cp1252 (as used by Gramps exports)
	input := []byte("0 HEAD\n1 CHAR cp1252\n0 @I1@ INDI\n1 TITL Z\xfcrich\n")

	got := convertToUTF8(input)

	assert.Contains(t, string(got), "Zürich")
}

func TestConvertToUTF8_ANSEL(t *testing.T) {
	// ANSEL copyright symbol: 0xC3 = ©
	input := []byte("0 HEAD\n1 CHAR ANSEL\n0 @N1@ NOTE \xc3 2024 by Author\n")

	got := convertToUTF8(input)

	assert.Contains(t, string(got), "© 2024 by Author")
}

func TestConvertToUTF8_ANSELCombiningDiacritics(t *testing.T) {
	// ANSEL combining hook above (0xE0) precedes base letter
	// 0xE0 + 'A' = À (A with hook above / grave accent)
	input := []byte("0 HEAD\n1 CHAR ANSEL\n0 @I1@ INDI\n1 NAME \xe0A\n")

	got := convertToUTF8(input)

	// The combining diacritical should produce a valid UTF-8 character
	assert.True(t, isValidUTF8(got), "output should be valid UTF-8")
}

func TestConvertToUTF8_UTF8Passthrough(t *testing.T) {
	// UTF-8 input should pass through unchanged
	input := []byte("0 HEAD\n1 CHAR UTF-8\n0 @I1@ INDI\n1 TITL König\n")

	got := convertToUTF8(input)

	assert.Equal(t, input, got)
}

func TestConvertToUTF8_ASCIIPassthrough(t *testing.T) {
	input := []byte("0 HEAD\n1 CHAR ASCII\n0 @I1@ INDI\n1 NAME John /Smith/\n")

	got := convertToUTF8(input)

	assert.Equal(t, input, got)
}

func TestConvertToUTF8_NoCHARDefaultsToPassthrough(t *testing.T) {
	// No CHAR header — assume UTF-8
	input := []byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 @I1@ INDI\n")

	got := convertToUTF8(input)

	assert.Equal(t, input, got)
}

func TestImport_CP1252PlaceNames(t *testing.T) {
	// Full GEDCOM with CP1252-encoded place name containing ß (0xDF)
	gedcom := "0 HEAD\r\n" +
		"1 GEDC\r\n" +
		"2 VERS 5.5\r\n" +
		"1 CHAR ANSI\r\n" +
		"0 @I1@ INDI\r\n" +
		"1 NAME Johann /Habsburg/\r\n" +
		"1 BIRT\r\n" +
		"2 DATE 15 MAR 1500\r\n" +
		"2 PLAC Stra\xdfburg\r\n" +
		"0 TRLR\r\n"

	glxFile, _, err := ImportGEDCOM(bytes.NewReader([]byte(gedcom)), io.Discard)
	require.NoError(t, err)

	// The place name should be valid UTF-8 "Straßburg"
	foundPlace := false
	for _, place := range glxFile.Places {
		if place.Name == "Straßburg" {
			foundPlace = true
			break
		}
	}
	assert.True(t, foundPlace, "should find place with properly decoded name 'Straßburg'")
}

func TestImport_CP1252PersonTitle(t *testing.T) {
	// CP1252 title with ö (0xF6)
	gedcom := "0 HEAD\r\n" +
		"1 GEDC\r\n" +
		"2 VERS 5.5\r\n" +
		"1 CHAR ANSI\r\n" +
		"0 @I1@ INDI\r\n" +
		"1 NAME Franz /Habsburg/\r\n" +
		"1 TITL K\xf6nig\r\n" +
		"0 TRLR\r\n"

	glxFile, _, err := ImportGEDCOM(bytes.NewReader([]byte(gedcom)), io.Discard)
	require.NoError(t, err)

	// Find person and check title property contains decoded UTF-8
	for _, person := range glxFile.Persons {
		if title, ok := person.Properties["title"]; ok {
			// Title may be stored as a string or temporal list with "value" key
			switch v := title.(type) {
			case string:
				assert.Equal(t, "König", v, "title should be decoded from CP1252 to UTF-8")
			case map[string]any:
				assert.Equal(t, "König", v["value"], "title value should be decoded from CP1252 to UTF-8")
			case []any:
				require.NotEmpty(t, v)
				m, ok := v[0].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, "König", m["value"], "title value should be decoded from CP1252 to UTF-8")
			default:
				t.Fatalf("unexpected title type: %T", title)
			}
			return
		}
	}
	t.Fatal("expected person with title property")
}

func TestImport_CP1252EventTitle(t *testing.T) {
	// Event title generated from CP1252-encoded name with ü (0xFC)
	gedcom := "0 HEAD\r\n" +
		"1 GEDC\r\n" +
		"2 VERS 5.5\r\n" +
		"1 CHAR ANSI\r\n" +
		"0 @I1@ INDI\r\n" +
		"1 NAME G\xfcnter /M\xfcller/\r\n" +
		"1 BIRT\r\n" +
		"2 DATE 1 JAN 1900\r\n" +
		"0 TRLR\r\n"

	glxFile, _, err := ImportGEDCOM(bytes.NewReader([]byte(gedcom)), io.Discard)
	require.NoError(t, err)

	// Event title should contain properly decoded name
	for _, event := range glxFile.Events {
		if event.Type == EventTypeBirth {
			assert.Contains(t, string(event.Title), "Günter", "event title should contain decoded name")
			assert.Contains(t, string(event.Title), "Müller", "event title should contain decoded surname")
			return
		}
	}
	t.Fatal("expected birth event")
}

// isValidUTF8 checks if all bytes form valid UTF-8.
func isValidUTF8(b []byte) bool {
	return utf8.Valid(b)
}
