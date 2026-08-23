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
	"errors"
	"strings"
	"testing"
)

func TestDecodeGEDCOMBlob(t *testing.T) {
	// GEDCOM BLOB encoding: each char minus '.' (0x2E) gives 6 bits.
	// 4 chars → 3 bytes.
	// '.' = 0, '/' = 1, '0' = 2, etc.

	// Simple test: 4 dots should produce 3 zero bytes
	decoded, err := DecodeGEDCOMBlob("....")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(decoded) != 3 {
		t.Fatalf("Expected 3 bytes, got %d", len(decoded))
	}
	for i, b := range decoded {
		if b != 0 {
			t.Errorf("Byte %d: expected 0, got %d", i, b)
		}
	}

	// Known non-zero value: 'm' is the max character (value 63 = 0b111111),
	// so "mmmm" decodes to three 0xFF bytes
	decodedMax, err := DecodeGEDCOMBlob("mmmm")
	if err != nil {
		t.Fatalf("Decode of max chars failed: %v", err)
	}
	if !bytes.Equal(decodedMax, []byte{0xFF, 0xFF, 0xFF}) {
		t.Errorf("Expected [0xFF 0xFF 0xFF], got %v", decodedMax)
	}

	// Test with newlines/whitespace (should be stripped)
	decoded2, err := DecodeGEDCOMBlob("....\n....")
	if err != nil {
		t.Fatalf("Decode with newlines failed: %v", err)
	}
	if len(decoded2) != 6 {
		t.Fatalf("Expected 6 bytes, got %d", len(decoded2))
	}

	// Trailing 3 chars (valid — produces 2 bytes)
	decoded3, err := DecodeGEDCOMBlob("...")
	if err != nil {
		t.Fatalf("Decode of 3 chars failed: %v", err)
	}
	if len(decoded3) != 2 {
		t.Errorf("Expected 2 bytes from 3 chars, got %d", len(decoded3))
	}

	// Trailing 2 chars (valid — produces 1 byte)
	decoded4, err := DecodeGEDCOMBlob("..")
	if err != nil {
		t.Fatalf("Decode of 2 chars failed: %v", err)
	}
	if len(decoded4) != 1 {
		t.Errorf("Expected 1 byte from 2 chars, got %d", len(decoded4))
	}

	// Mixed: 4 + 2 trailing chars (produces 3 + 1 = 4 bytes)
	decoded5, err := DecodeGEDCOMBlob("......")
	if err != nil {
		t.Fatalf("Decode of 6 chars failed: %v", err)
	}
	if len(decoded5) != 4 {
		t.Errorf("Expected 4 bytes from 6 chars, got %d", len(decoded5))
	}
}

// TestDecodeGEDCOMBlobTrailingValues pins the byte values the trailing-group
// paths produce, not just their length. All-zero and all-ones inputs survive a
// swapped shift direction, so these use mixed sextets to pin the
// (b1<<2)|(b2>>4) and (b2<<4)|(b3>>2) splits.
func TestDecodeGEDCOMBlobTrailingValues(t *testing.T) {
	cases := []struct {
		input string
		want  []byte
	}{
		{"mm", []byte{0xFF}},        // 63, 63
		{"m.", []byte{0xFC}},        // 63, 0  -> 111111|00
		{".m", []byte{0x03}},        // 0, 63  -> 000000|11
		{"mmm", []byte{0xFF, 0xFF}}, // 63, 63, 63
		{"m.m", []byte{0xFC, 0x0F}}, // 63, 0, 63
		{".m.", []byte{0x03, 0xF0}}, // 0, 63, 0
	}

	for _, tc := range cases {
		got, err := DecodeGEDCOMBlob(tc.input)
		if err != nil {
			t.Errorf("DecodeGEDCOMBlob(%q) failed: %v", tc.input, err)

			continue
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("DecodeGEDCOMBlob(%q) = % X, want % X", tc.input, got, tc.want)
		}
	}
}

// TestDecodeGEDCOMBlobWhitespace verifies every ASCII whitespace byte — the
// \r and \n of CONT/CONC line wrapping plus the padding lenient exporters
// insert — is stripped before decoding.
func TestDecodeGEDCOMBlobWhitespace(t *testing.T) {
	// Same four characters, split by each whitespace byte in turn.
	for _, ws := range []string{" ", "\t", "\n", "\v", "\f", "\r", "\r\n", " \t "} {
		got, err := DecodeGEDCOMBlob("mm" + ws + "mm")
		if err != nil {
			t.Errorf("DecodeGEDCOMBlob with %q separator failed: %v", ws, err)

			continue
		}
		if !bytes.Equal(got, []byte{0xFF, 0xFF, 0xFF}) {
			t.Errorf("DecodeGEDCOMBlob with %q separator = % X, want FF FF FF", ws, got)
		}
	}
}

func TestDecodeGEDCOMBlobErrors(t *testing.T) {
	// Empty blob
	if _, err := DecodeGEDCOMBlob(""); !errors.Is(err, ErrGEDCOMBlobEmpty) {
		t.Errorf("Expected ErrGEDCOMBlobEmpty for empty blob, got %v", err)
	}

	// Whitespace-only blob cleans to empty
	if _, err := DecodeGEDCOMBlob(" \t\n\v\f\r "); !errors.Is(err, ErrGEDCOMBlobEmpty) {
		t.Errorf("Expected ErrGEDCOMBlobEmpty for whitespace-only blob, got %v", err)
	}

	// Single char is invalid (not enough bits for a full byte)
	if _, err := DecodeGEDCOMBlob("."); !errors.Is(err, ErrGEDCOMBlobLength) {
		t.Errorf("Expected ErrGEDCOMBlobLength for single character, got %v", err)
	}

	// 5 chars (4 + 1 trailing) is invalid — trailing single char can't encode a byte
	if _, err := DecodeGEDCOMBlob("....."); !errors.Is(err, ErrGEDCOMBlobLength) {
		t.Errorf("Expected ErrGEDCOMBlobLength for 5 chars, got %v", err)
	}

	// An out-of-range character takes precedence over an invalid length: a lone
	// invalid byte — bare or after full groups — must report ErrGEDCOMBlobChar,
	// not the trailing-single-character length error.
	if _, err := DecodeGEDCOMBlob("z"); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for lone out-of-range char, got %v", err)
	}
	if _, err := DecodeGEDCOMBlob("....z"); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for out-of-range char in trailing single position, got %v", err)
	}

	// Out-of-range character in a full group ('z' > 'm')
	if _, err := DecodeGEDCOMBlob("..z."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for out-of-range char in full group, got %v", err)
	}

	// Out-of-range character in a trailing group
	if _, err := DecodeGEDCOMBlob("....z."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for out-of-range char in trailing group, got %v", err)
	}

	// Lower boundary: '-' is 0x2D, exactly one below '.' (0x2E)
	if _, err := DecodeGEDCOMBlob("-..."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for below-range char in full group, got %v", err)
	}
	if _, err := DecodeGEDCOMBlob("....-."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for below-range char in trailing group, got %v", err)
	}

	// Upper boundary: 'n' is 0x6E, exactly one above 'm' (0x6D). 'm' itself is
	// accepted — see the "mmmm" case in TestDecodeGEDCOMBlob.
	if _, err := DecodeGEDCOMBlob("n..."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for 'n' in full group, got %v", err)
	}
	if _, err := DecodeGEDCOMBlob("....n."); !errors.Is(err, ErrGEDCOMBlobChar) {
		t.Errorf("Expected ErrGEDCOMBlobChar for 'n' in trailing group, got %v", err)
	}
}

// TestDecodeGEDCOMBlobCharErrorReportsByte pins the hex rendering in the
// invalid-character message. %q formats a byte as a rune, so on its own it
// reports any byte >= 0x80 as the Latin-1 character with that code point
// rather than the byte actually in the input; the hex cannot lie.
func TestDecodeGEDCOMBlobCharErrorReportsByte(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"full group", "..\xFF."},
		{"trailing group", "....\xFF."},
	} {
		_, err := DecodeGEDCOMBlob(tc.input)
		if err == nil {
			t.Errorf("%s: expected an error for 0xFF, got nil", tc.name)

			continue
		}
		if !strings.Contains(err.Error(), "(0xFF)") {
			t.Errorf("%s: error %q does not report the offending byte as 0xFF", tc.name, err)
		}
	}
}
