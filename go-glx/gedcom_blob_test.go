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

func TestDecodeGEDCOMBlobErrors(t *testing.T) {
	// Empty blob
	if _, err := DecodeGEDCOMBlob(""); !errors.Is(err, ErrEmptyBlobData) {
		t.Errorf("Expected ErrEmptyBlobData for empty blob, got %v", err)
	}

	// Whitespace-only blob cleans to empty
	if _, err := DecodeGEDCOMBlob(" \n\r "); !errors.Is(err, ErrEmptyBlobData) {
		t.Errorf("Expected ErrEmptyBlobData for whitespace-only blob, got %v", err)
	}

	// Single char is invalid (not enough bits for a full byte)
	if _, err := DecodeGEDCOMBlob("."); !errors.Is(err, ErrInvalidBlobLength) {
		t.Errorf("Expected ErrInvalidBlobLength for single character, got %v", err)
	}

	// 5 chars (4 + 1 trailing) is invalid — trailing single char can't encode a byte
	if _, err := DecodeGEDCOMBlob("....."); !errors.Is(err, ErrInvalidBlobLength) {
		t.Errorf("Expected ErrInvalidBlobLength for 5 chars, got %v", err)
	}

	// Out-of-range character in a full group ('z' > 'm')
	if _, err := DecodeGEDCOMBlob("..z."); !errors.Is(err, ErrInvalidBlobChar) {
		t.Errorf("Expected ErrInvalidBlobChar for out-of-range char in full group, got %v", err)
	}

	// Out-of-range character in a trailing group
	if _, err := DecodeGEDCOMBlob("....z."); !errors.Is(err, ErrInvalidBlobChar) {
		t.Errorf("Expected ErrInvalidBlobChar for out-of-range char in trailing group, got %v", err)
	}
}
