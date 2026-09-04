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
	"strconv"
	"strings"
)

// gedcomEscapeStart opens a GEDCOM calendar escape: "@#DJULIAN@ 15 MAR 1731".
const gedcomEscapeStart = "@#D"

// gedcomFrenchRepublicanEscape is the GEDCOM escape name for FRENCH_R; the
// Julian and Hebrew escape names equal their GLX prefixes.
const gedcomFrenchRepublicanEscape = "FRENCH R"

// gedcomEra551 is the era suffix of GEDCOM 5.5.1 ("44 B.C."); GEDCOM 7 and
// GLX write BCE.
const gedcomEra551 = "B.C."

// gedcomEscapes maps GLX calendar prefixes to their GEDCOM escape names (the
// text between "@#D" and "@"). Gregorian has no prefix and no escape.
var gedcomEscapes = map[string]string{
	PrefixJulian:           PrefixJulian,
	PrefixHebrew:           PrefixHebrew,
	PrefixFrenchRepublican: gedcomFrenchRepublicanEscape,
}

// gedcomEscapeNames is the inverse of gedcomEscapes plus GREGORIAN, which
// maps to the empty prefix.
var gedcomEscapeNames = map[string]string{
	PrefixJulian:                 PrefixJulian,
	PrefixHebrew:                 PrefixHebrew,
	gedcomFrenchRepublicanEscape: PrefixFrenchRepublican,
	calendarGregorianName:        "",
}

// Canonicalize returns the canonical GLX form of a date string when its
// components can be determined without guessing, and the input (trimmed)
// unchanged otherwise. A date is only rewritten when its canonical form
// itself parses as valid, so a raw-preserved body is never half-rewritten.
//
// This is the importer's policy: [Parse] is strict about what it calls
// valid, but a tolerated spelling ("15 March 1850", "Abt 1850") is stored in
// canonical form rather than flagged.
func Canonicalize(s string) string {
	s = strings.TrimSpace(s)
	d, err := Parse(s)
	if err == nil {
		return d.String()
	}

	if canon := d.String(); canon != s {
		if _, err := Parse(canon); err == nil {
			return canon
		}
	}

	return s
}

// FromGEDCOM converts a GEDCOM DATE payload to a GLX date string.
//
// A leading calendar escape (@#DJULIAN@, @#DHEBREW@, @#DFRENCH R@,
// @#DGREGORIAN@, or any other @#D…@ name) becomes the GLX calendar prefix,
// and the body is passed through [Canonicalize]: standard GEDCOM bodies
// ("15 MAR 1850", "ABT 1850", "BET 1880 AND 1890", "FROM 1900 TO 1950") and
// the common dialect variants with full or mixed-case month names and
// keywords ("1 January 1900", "Bet 1880 and 1890", "ABT. 1850") come out in
// canonical GLX form. Anything that cannot be canonicalized without guessing
// (numeric day/month forms, dual years, free text) is preserved verbatim so
// it survives a roundtrip; validation later flags it.
//
// A non-standard calendar escape becomes an extension prefix in GEDCOM 7
// form: an underscore, then the name with spaces folded to underscores
// ("@#DROMAN@" → "_ROMAN", "@#DNEW CAL@" → "_NEW_CAL"). An escape with no
// date body ("@#DJULIAN@") is preserved verbatim.
func FromGEDCOM(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	prefix, body, found := splitGEDCOMEscape(s)
	if !found {
		return Canonicalize(s)
	}
	if body == "" {
		return s
	}
	if prefix != "" {
		body = prefix + " " + body
	}

	return Canonicalize(body)
}

// ParseGEDCOM parses a GEDCOM DATE payload. It is [Parse] applied to
// [FromGEDCOM], so the error reports whether the stored GLX form is valid.
func ParseGEDCOM(s string) (Date, error) {
	return Parse(FromGEDCOM(s))
}

// splitGEDCOMEscape splits a leading GEDCOM calendar escape from s, returning
// the GLX calendar prefix ("" for Gregorian), the trimmed remainder, and
// whether an escape was present. Unknown escape names become extension
// prefixes: an underscore, then the name with spaces folded to underscores.
func splitGEDCOMEscape(s string) (string, string, bool) {
	if !strings.HasPrefix(s, gedcomEscapeStart) {
		return "", s, false
	}

	name, rest, found := strings.Cut(s[len(gedcomEscapeStart):], "@")
	if !found {
		return "", s, false
	}

	prefix, known := gedcomEscapeNames[name]
	if !known {
		prefix = strings.ReplaceAll(name, " ", "_")
		if !strings.HasPrefix(prefix, "_") {
			prefix = "_" + prefix
		}
	}

	return prefix, strings.TrimSpace(rest), true
}

// gedcomEscape renders a GLX calendar prefix as a GEDCOM calendar escape.
// The empty (Gregorian) prefix has no escape; an extension prefix is
// written as-is ("_ROMAN" → "@#D_ROMAN@").
func gedcomEscape(prefix string) string {
	if prefix == "" {
		return ""
	}

	name, known := gedcomEscapes[prefix]
	if !known {
		name = prefix
	}

	return gedcomEscapeStart + name + "@"
}

// GEDCOM renders the date as a GEDCOM 7 DATE payload: the calendar prefix
// as an escape and every exact component in GEDCOM spelling ("15 MAR 1850",
// "ABT 1850", "BET 1 JAN 1880 AND 31 DEC 1890", "44 BCE"). A body whose
// components were not determined is rendered as [Date.String] would, so
// nothing is invented on export that was not understood on import.
func (d Date) GEDCOM() string {
	if d.IsZero() {
		return ""
	}

	return d.val().render(renderGEDCOM7)
}

// GEDCOM551 renders the date as a GEDCOM 5.5.1 DATE payload. It differs
// from [Date.GEDCOM] only in the era suffix: 5.5.1 writes "B.C." where
// GEDCOM 7 writes "BCE".
func (d Date) GEDCOM551() string {
	if d.IsZero() {
		return ""
	}

	return d.val().render(renderGEDCOM551)
}

// gedcom renders an exact point in GEDCOM spelling, and a raw point as-is.
func (p point) gedcom(mode renderMode) string {
	if !p.exact {
		return p.raw
	}

	year := strconv.Itoa(p.year)
	for len(year) < maxYearDigits {
		year = "0" + year
	}
	if p.bce {
		era := keywordBCE
		if mode == renderGEDCOM551 {
			era = gedcomEra551
		}
		year += " " + era
	}

	switch p.precision {
	case PrecisionDay:
		return strconv.Itoa(p.day) + " " + monthAbbreviations[p.month] + " " + year
	case PrecisionMonth:
		return monthAbbreviations[p.month] + " " + year
	case PrecisionYear, PrecisionNone:
		return year
	}

	return p.raw
}
