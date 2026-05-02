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

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// arkNAAN is the Name Assigning Authority Number that identifies FamilySearch
// as the issuer of the ARK identifiers resolved by this command.
const arkNAAN = "61903"

// arkTypeURI is the URI form of the ARK prefix. GEDCOM 7 EXID records written
// by FamilySearch set TYPE to this exact string, so `glx link` produces the
// same external_ids shape for interoperability with imported GEDCOM files.
const arkTypeURI = "https://www.familysearch.org/ark:/" + arkNAAN + "/"

// arkURLPattern matches a FamilySearch ARK URL or bare ARK identifier.
// Accepts:
//
//	https://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2
//	https://familysearch.org/ark:/61903/1:1:C4H8-2DW2
//	http://www.familysearch.org/ark:/61903/1:1:C4H8-2DW2
//	ark:/61903/1:1:C4H8-2DW2
//
// The NOID (opaque identifier after the NAAN) is captured as group 1. Record
// ARKs from FamilySearch use the form "1:1:XXXX-XXXX"; the regex permits any
// ASCII alphanumeric/colon/hyphen sequence so other NOID shapes are not
// rejected out of hand.
var arkURLPattern = regexp.MustCompile(`^(?:https?://(?:www\.)?familysearch\.org/)?ark:/` + arkNAAN + `/([A-Za-z0-9:-]+)$`)

// ARK is a parsed FamilySearch ARK identifier.
type ARK struct {
	// NOID is the opaque identifier following "ark:/61903/", e.g. "1:1:C4H8-2DW2".
	// Original casing is preserved.
	NOID string
	// CanonicalURL is the input normalized to https://www.familysearch.org/ark:/61903/<NOID>.
	CanonicalURL string
}

// ParseFamilySearchARK validates the input as a FamilySearch ARK URL or bare
// identifier and returns a parsed ARK. Returns an error if the input is empty
// or does not match the expected form.
func ParseFamilySearchARK(input string) (*ARK, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, ErrEmptyARK
	}
	m := arkURLPattern.FindStringSubmatch(input)
	if m == nil {
		return nil, fmt.Errorf("%w: %q", ErrInvalidARK, input)
	}
	noid := m[1]

	return &ARK{
		NOID:         noid,
		CanonicalURL: fmt.Sprintf("https://www.familysearch.org/ark:/%s/%s", arkNAAN, noid),
	}, nil
}

// CitationIDSlug returns a deterministic, lowercase, filename-safe slug suitable
// for use as the stable portion of a citation ID.
//
// FamilySearch record-level ARKs carry the "1:1:" prefix by convention; it is
// redundant for our ID purposes and stripped. Other NOID shapes keep their
// contents with colons replaced by hyphens.
//
// Examples:
//
//	"1:1:C4H8-2DW2"  -> "c4h8-2dw2"
//	"2:1:ZZZ-TEST"   -> "2-1-zzz-test"
//	"FOO"            -> "foo"
func (a *ARK) CitationIDSlug() string {
	noid := a.NOID
	noid = strings.TrimPrefix(noid, "1:1:")
	noid = strings.ReplaceAll(noid, ":", "-")

	return strings.ToLower(noid)
}
