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
	"strings"
)

// Soundex computes the American Soundex code for a name, following the
// NARA (National Archives) algorithm used by the US Census Bureau.
// Returns a 4-character code (letter + 3 digits) or empty string for empty input.
func Soundex(name string) string {
	if name == "" {
		return ""
	}

	// Normalize: uppercase, ASCII letters only (Soundex is defined for A-Z)
	var letters []byte
	for _, r := range strings.ToUpper(name) {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, byte(r))
		}
	}
	if len(letters) == 0 {
		return ""
	}

	// First letter is kept as-is
	code := []byte{letters[0]}

	// Map remaining letters to digits
	prev := soundexDigitByte(letters[0])
	for i := 1; i < len(letters) && len(code) < 4; i++ {
		digit := soundexDigitByte(letters[i])
		if digit == '0' {
			// H and W don't separate identical digits; vowels (AEIOUY) do
			if letters[i] == 'A' || letters[i] == 'E' || letters[i] == 'I' ||
				letters[i] == 'O' || letters[i] == 'U' || letters[i] == 'Y' {
				prev = '0'
			}

			continue
		}
		if digit != prev {
			code = append(code, digit)
			prev = digit
		}
	}

	// Pad with zeros to length 4
	for len(code) < 4 {
		code = append(code, '0')
	}

	return string(code)
}

// soundexDigitByte returns the Soundex digit for an uppercase ASCII letter, or '0' for vowels/H/W.
func soundexDigitByte(r byte) byte {
	switch rune(r) {
	case 'B', 'F', 'P', 'V':
		return '1'
	case 'C', 'G', 'J', 'K', 'Q', 'S', 'X', 'Z':
		return '2'
	case 'D', 'T':
		return '3'
	case 'L':
		return '4'
	case 'M', 'N':
		return '5'
	case 'R':
		return '6'
	default:
		return '0' // A, E, I, O, U, H, W, Y
	}
}

// SoundexMatch returns true if two names have the same Soundex code.
func SoundexMatch(a, b string) bool {
	sa := Soundex(a)
	sb := Soundex(b)

	return sa != "" && sb != "" && sa == sb
}

// PhoneticMatch represents a person found via phonetic search.
type PhoneticMatch struct {
	PersonID    string
	PersonName  string
	SoundexCode string
	MatchedPart string // which part of the name matched (given or surname)
}

// PhoneticPersonSearch searches all persons in the archive for names that
// phonetically match the query using Soundex. Matches against both given
// names and surnames.
func PhoneticPersonSearch(archive *GLXFile, query string) []PhoneticMatch {
	if archive == nil || query == "" {
		return nil
	}

	queryCode := Soundex(query)
	if queryCode == "" {
		return nil
	}

	var matches []PhoneticMatch

	for id, person := range archive.Persons {
		if person == nil {
			continue
		}

		name := PersonDisplayName(person)
		if name == "" {
			continue
		}

		// Check each word in the name against the query's Soundex code
		for part := range strings.FieldsSeq(name) {
			partCode := Soundex(part)
			if partCode == queryCode {
				matches = append(matches, PhoneticMatch{
					PersonID:    id,
					PersonName:  name,
					SoundexCode: partCode,
					MatchedPart: part,
				})

				break // one match per person is enough
			}
		}
	}

	return matches
}
