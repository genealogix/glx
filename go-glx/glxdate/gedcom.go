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
	"regexp"
	"strings"
)

// gedcomEscapeStart opens a GEDCOM 5.5.1 calendar escape: "@#DJULIAN@ 15 MAR 1731".
const gedcomEscapeStart = "@#D"

// gedcomFrenchRepublicanEscape is the GEDCOM 5.5.1 escape name for FRENCH_R;
// the Julian and Hebrew escape names equal their GLX prefixes.
const gedcomFrenchRepublicanEscape = "FRENCH R"

// gedcomEra551 is the era suffix of GEDCOM 5.5.1 ("44 B.C."); GEDCOM 7 and
// GLX write BCE.
const gedcomEra551 = "B.C."

// gedcomEscapes maps GLX calendar prefixes to their GEDCOM 5.5.1 escape names
// (the text between "@#D" and "@"). Gregorian has no prefix and no escape.
var gedcomEscapes = map[string]string{
	PrefixJulian:           PrefixJulian,
	PrefixHebrew:           PrefixHebrew,
	PrefixFrenchRepublican: gedcomFrenchRepublicanEscape,
}

// gedcomEscapeNames is the inverse of gedcomEscapes plus GREGORIAN and the
// empty name, which map to the empty prefix.
var gedcomEscapeNames = map[string]string{
	PrefixJulian:                 PrefixJulian,
	PrefixHebrew:                 PrefixHebrew,
	gedcomFrenchRepublicanEscape: PrefixFrenchRepublican,
	calendarGregorianName:        "",
	"":                           "",
}

// gedcomEscapeRegexp matches a GEDCOM 5.5.1 calendar escape anywhere in a
// payload. The grammar puts it after the qualifier and inside each range
// endpoint ("ABT @#DJULIAN@ 15 MAR 1731"), though many writers put it first.
var gedcomEscapeRegexp = regexp.MustCompile(`@#D([^@]*)@`)

// Canonicalize returns the canonical GLX form of a date string when its
// components can be determined without guessing, and the input (trimmed)
// unchanged otherwise. Only a date whose every component is exact is
// rewritten, so a raw-preserved body is never half-rewritten; the corpus
// tests assert that every such rewrite parses back as valid.
//
// This is the importer's policy: [Parse] is strict about what it calls
// valid, but a tolerated spelling ("15 March 1850", "Abt 1850") is stored in
// canonical form rather than flagged.
func Canonicalize(s string) string {
	s = strings.TrimSpace(s)
	d, err := Parse(s)
	if err == nil || d.val().exactThroughout() {
		return d.String()
	}

	return s
}

// exactThroughout reports whether every point of the date was fully
// determined, which is when String renders canonical components.
func (v *dateValue) exactThroughout() bool {
	hasEnd := v.rng == rangeBetween || v.rng == rangeFromTo

	return v.start.exact && (!hasEnd || v.end.exact)
}

// FromGEDCOM converts a GEDCOM DATE payload to a GLX date string.
//
// Calendars are recognized in either GEDCOM spelling and at any position:
// 5.5.1 escapes (@#DJULIAN@, @#DHEBREW@, @#DFRENCH R@, @#DGREGORIAN@, or any
// other @#D…@ name) and GEDCOM 7 calendar tags (JULIAN, HEBREW, FRENCH_R,
// GREGORIAN, _EXTENSION), whether written first or, as both grammars
// specify, after the qualifier and before each range endpoint ("ABT JULIAN
// 1731", "BET @#DJULIAN@ 1700 AND @#DJULIAN@ 1710"). When every endpoint
// names the same calendar it becomes the GLX prefix; a range that mixes
// calendars is preserved verbatim. A non-standard calendar becomes an
// extension prefix in GEDCOM 7 form: an underscore, then the name with
// spaces folded to underscores ("@#DROMAN@" → "_ROMAN", "@#DNEW CAL@" →
// "_NEW_CAL").
//
// The body is then passed through [Canonicalize]: standard GEDCOM bodies
// ("15 MAR 1850", "ABT 1850", "BET 1880 AND 1890", "FROM 1900 TO 1950") and
// the common dialect variants with full or mixed-case month names and
// keywords ("1 January 1900", "Bet 1880 and 1890", "ABT. 1850") come out in
// canonical GLX form. Anything that cannot be canonicalized without guessing
// (numeric day/month forms, dual years, free text) is preserved verbatim so
// it survives a roundtrip; validation later flags it.
//
// An escape with no date body ("@#DJULIAN@") is preserved verbatim.
func FromGEDCOM(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	// 5.5.1 escapes become GEDCOM 7 calendar tags so one pass handles both.
	// A Gregorian or empty escape becomes the GREGORIAN tag so it still
	// counts as naming its endpoint's calendar.
	tagged := gedcomEscapeRegexp.ReplaceAllStringFunc(s, func(esc string) string {
		tag := extensionPrefix(gedcomEscapeRegexp.FindStringSubmatch(esc)[1])
		if tag == "" {
			tag = calendarGregorianName
		}

		return " " + tag + " "
	})

	tokens := strings.Fields(tagged)
	prefix, body, ok := liftCalendar(tokens)
	switch {
	case !ok:
		return s // calendars disagree between endpoints: nothing to lift
	case len(body) == 0:
		return s // an escape or tag with no date body
	case prefix != "":
		body = append([]string{prefix}, body...)
	}

	return Canonicalize(strings.Join(body, " "))
}

// extensionPrefix maps a GEDCOM calendar name (escape text or 7.0 tag) to a
// GLX prefix: "" for Gregorian, the known prefix for the standard calendars,
// and an underscore-prefixed extension name for anything else.
func extensionPrefix(name string) string {
	if prefix, known := gedcomEscapeNames[name]; known {
		return prefix
	}
	if _, known := knownPrefixes[name]; known {
		return name // a GEDCOM 7 tag spelled as the GLX prefix (FRENCH_R)
	}
	prefix := strings.ReplaceAll(name, " ", "_")
	if !strings.HasPrefix(prefix, "_") {
		prefix = "_" + prefix
	}

	return prefix
}

// isCalendarTag reports whether tok names a calendar in GEDCOM 7 form.
func isCalendarTag(tok string) bool {
	if tok == calendarGregorianName {
		return true
	}
	if _, known := knownPrefixes[tok]; known {
		return true
	}

	return isCalendarPrefix(tok)
}

// liftCalendar removes calendar tags from tokens and returns the single GLX
// prefix they agree on ("" when none or Gregorian) with the remaining
// tokens. A tag counts only in calendar position: first, or directly after
// a qualifier, BET, FROM, AND, or TO ("_MONTH" inside a body is an extension
// month, not a calendar). A leading tag covers the whole date; otherwise
// every endpoint of a range must name the same calendar, since an endpoint
// with no tag is Gregorian. It reports false when the calendars disagree.
func liftCalendar(tokens []string) (string, []string, bool) {
	if len(tokens) == 0 {
		return "", tokens, true
	}

	var (
		body      = make([]string, 0, len(tokens))
		leading   = isCalendarTag(tokens[0])
		prefix    string
		tagged    int
		endpoints = 1
	)
	for i, tok := range tokens {
		kw := ""
		if i > 0 {
			kw = keywordOf(tokens[i-1])
		}
		if kw == keywordAnd || kw == keywordTo && i > 1 {
			endpoints++
		}
		inPosition := i == 0 || kw == keywordAnd || kw == keywordTo || kw == keywordBetween ||
			kw == keywordFrom || qualifierKeywords[kw] != QualifierNone
		if !inPosition || !isCalendarTag(tok) {
			body = append(body, tok)

			continue
		}
		cal := extensionPrefix(tok)
		if tagged > 0 && cal != prefix {
			return "", tokens, false
		}
		prefix, tagged = cal, tagged+1
	}

	if tagged > 0 && !leading && tagged != endpoints {
		return "", tokens, false // one endpoint named a calendar, the other is Gregorian
	}

	return prefix, body, true
}

// ParseGEDCOM parses a GEDCOM DATE payload. It is [Parse] applied to
// [FromGEDCOM], so the error reports whether the stored GLX form is valid.
func ParseGEDCOM(s string) (Date, error) {
	return Parse(FromGEDCOM(s))
}

// gedcomEscape551 renders a GLX calendar prefix as a GEDCOM 5.5.1 calendar
// escape. An extension prefix drops its leading underscore, so a calendar
// read from "@#DROMAN@" is written back the same way.
func gedcomEscape551(prefix string) string {
	name, known := gedcomEscapes[prefix]
	if !known {
		name = strings.TrimPrefix(prefix, "_")
	}

	return gedcomEscapeStart + name + "@"
}

// GEDCOM renders the date as a GEDCOM 7 DATE payload: the calendar as a tag
// before each date ("ABT JULIAN 15 MAR 1731", "BET JULIAN 1700 AND JULIAN
// 1710"), every exact component in GEDCOM spelling, and "BCE" for the era.
// A body whose components were not determined is rendered as [Date.String]
// would, so nothing is invented on export that was not understood on import.
func (d Date) GEDCOM() string {
	return d.val().render(renderGEDCOM7)
}

// GEDCOM551 renders the date as a GEDCOM 5.5.1 DATE payload: a calendar
// escape first ("@#DJULIAN@ ABT 15 MAR 1731") and "B.C." for the era.
func (d Date) GEDCOM551() string {
	return d.val().render(renderGEDCOM551)
}
