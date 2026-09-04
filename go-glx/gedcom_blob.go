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
	"errors"
	"fmt"
	"strings"
)

// BLOB decoding errors
var (
	ErrGEDCOMBlobEmpty  = errors.New("empty BLOB data")
	ErrGEDCOMBlobLength = errors.New("invalid BLOB data length")
	ErrGEDCOMBlobChar   = errors.New("invalid BLOB character")
)

// blobWhitespace strips the ASCII whitespace GEDCOM CONT/CONC line wrapping
// introduces (\r, \n), plus the padding bytes lenient exporters have been
// seen to insert (space, \t, \v, \f). All are outside the '.'-'m' range, so
// stripping them widens what decodes without changing any existing decode.
// Byte-oriented rather than rune-oriented (strings.Map) so that invalid UTF-8
// in a malformed BLOB passes through untouched and is reported at its true
// byte offset instead of as U+FFFD.
var blobWhitespace = strings.NewReplacer(
	" ", "", "\t", "", "\n", "", "\v", "", "\f", "", "\r", "",
)

// DecodeGEDCOMBlob decodes GEDCOM 5.5.1 BLOB-encoded text to raw bytes.
// GEDCOM BLOB uses a custom encoding where each character's value minus 0x2E ('.')
// gives a 6-bit value. Groups of 4 characters encode 3 bytes; a trailing group of
// 2 or 3 characters encodes 1 or 2 bytes. ASCII whitespace is stripped before
// decoding.
//
// The returned error matches [ErrGEDCOMBlobEmpty] when the text holds no encoded
// characters, [ErrGEDCOMBlobLength] when the character count cannot encode a
// whole number of bytes, and [ErrGEDCOMBlobChar] when a character falls outside
// the '.' to 'm' range, in each case under [errors.Is]. When the text contains
// both an out-of-range character and an invalid length, the character error
// takes precedence.
func DecodeGEDCOMBlob(blobText string) ([]byte, error) {
	cleaned := blobWhitespace.Replace(blobText)

	if len(cleaned) == 0 {
		return nil, ErrGEDCOMBlobEmpty
	}

	result := make([]byte, 0, len(cleaned)*3/4)
	fullGroups := (len(cleaned) / 4) * 4 //nolint:mnd // 4 chars per group

	for i := 0; i < fullGroups; i += 4 {
		// Validate each character is in valid GEDCOM BLOB range (0x2E '.' to 0x6D 'm')
		// This gives 6-bit values (0-63) after subtracting 0x2E
		for j := range 4 {
			char := cleaned[i+j]
			if char < '.' || char > 'm' {
				return nil, fmt.Errorf("%w at position %d: %q (0x%02X) (must be in range '.' to 'm')", ErrGEDCOMBlobChar, i+j, char, char)
			}
		}

		b1 := cleaned[i] - '.'
		b2 := cleaned[i+1] - '.'
		b3 := cleaned[i+2] - '.'
		b4 := cleaned[i+3] - '.'

		//nolint:mnd // well-known base64 bit shifts
		result = append(result,
			(b1<<2)|(b2>>4),
			(b2<<4)|(b3>>2),
			(b3<<6)|b4,
		)
	}

	// Handle trailing characters (2 chars → 1 byte, 3 chars → 2 bytes).
	// Validate before the length guard so an out-of-range byte is reported as
	// ErrGEDCOMBlobChar even when the trailing group is also too short to decode.
	trailing := len(cleaned) - fullGroups
	for j := range trailing {
		char := cleaned[fullGroups+j]
		if char < '.' || char > 'm' {
			return nil, fmt.Errorf("%w at position %d: %q (0x%02X) (must be in range '.' to 'm')", ErrGEDCOMBlobChar, fullGroups+j, char, char)
		}
	}
	if trailing == 1 {
		return nil, fmt.Errorf("%w: %d (trailing single character cannot encode a full byte)", ErrGEDCOMBlobLength, len(cleaned))
	}
	if trailing >= 2 { //nolint:mnd // trailing group sizes
		b1 := cleaned[fullGroups] - '.'
		b2 := cleaned[fullGroups+1] - '.'
		result = append(result, (b1<<2)|(b2>>4)) //nolint:mnd // well-known base64 bit shifts

		if trailing == 3 { //nolint:mnd // 3-char trailing group
			b3 := cleaned[fullGroups+2] - '.'
			result = append(result, (b2<<4)|(b3>>2)) //nolint:mnd // well-known base64 bit shifts
		}
	}

	return result, nil
}
