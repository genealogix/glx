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
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// ErrRangeMismatch is returned by NewRange when the endpoints are not
// unqualified point dates in the same calendar.
var ErrRangeMismatch = errors.New("glxdate: range endpoints must be unqualified point dates in the same calendar")

// ParseError reports why a date string is not in canonical GLX form. The
// Date returned alongside it still carries the raw text and any components
// that could be recovered.
type ParseError struct {
	Input  string
	Reason string
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	return "invalid date " + strconv.Quote(e.Input) + ": " + e.Reason
}

const (
	// maxYearDigits is the width of a canonical year. Shorter years are
	// accepted and zero-padded on output (see #127).
	maxYearDigits = 4
	// maxRawYearDigits bounds the year token of raw-preserved bodies; Hebrew
	// years such as 5765 have four digits, and five leaves headroom.
	maxRawYearDigits = 5
	// componentDigits is the width of the MM and DD components.
	componentDigits = 2
	// maxYear is the largest year representable in canonical form.
	maxYear = 9999
	// maxDayDigits is the width of a day-of-month token in a named-month body.
	maxDayDigits = 2
)

var (
	// digitRunRegexp finds maximal runs of ASCII digits, including runs glued
	// to letters ("APR1828", "1893twin"): a 4-digit run is a year wherever it
	// sits.
	digitRunRegexp = regexp.MustCompile(`\d+`)
	// standaloneNumberRegexp finds runs of digits that form a whole token.
	// Short numbers glued to letters ("2nd", "10th") are ordinals, never years.
	standaloneNumberRegexp = regexp.MustCompile(`\b\d+\b`)
	// dayMonthRegexp matches a day of month followed by a Gregorian month
	// name or abbreviation, in any case ("15 MAR", "1 January", "3 sept.").
	dayMonthRegexp = regexp.MustCompile(`(?i)\b\d{1,2}\s+(?:JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)[A-Z]*\.?\b`)
)

// Parse parses a GLX date string. The empty string parses to the zero Date
// with no error.
//
// Parse never loses information: when the input is not in canonical form it
// returns a *ParseError explaining why together with a Date that preserves
// the raw text and exposes whatever could be recovered (in particular
// Date.Year). Callers that only need the year may ignore the error; callers
// that validate must not.
func Parse(s string) (Date, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Date{}, nil
	}

	v := parseDate(s)
	if !v.valid {
		return Date{v: v}, &ParseError{Input: s, Reason: v.reason}
	}

	return Date{v: v}, nil
}

// parseDate splits the calendar prefix and parses the body.
func parseDate(s string) *dateValue {
	prefix, body := SplitCalendarPrefix(s)
	v := &dateValue{raw: s, calendar: calendarForPrefix(prefix)}
	if v.calendar == CalendarOther {
		v.calendarName = prefix
	}

	tokens := strings.Fields(body)
	if len(tokens) == 0 {
		v.reason = "missing date body"

		return v
	}
	v.parseBody(tokens)

	return v
}

// parseBody recognizes the range and keyword forms, falling back to a single
// point (raw-preserved when unrecognized). tokens is non-empty.
func (d *dateValue) parseBody(tokens []string) {
	kw := keywordOf(tokens[0])
	rest := tokens[1:]
	reason := ""

	switch {
	case kw == keywordBetween:
		if i := indexKeyword(rest, keywordAnd); i > 0 && i < len(rest)-1 {
			d.setRange(rangeBetween, rest[:i], rest[i+1:])
			d.checkKeyword(tokens[0])
			d.checkKeyword(rest[i])

			return
		}
		reason = "BET requires two dates joined by AND"

	case kw == keywordFrom:
		i := indexKeyword(rest, keywordTo)
		switch {
		case i > 0 && i < len(rest)-1:
			d.setRange(rangeFromTo, rest[:i], rest[i+1:])
			d.checkKeyword(tokens[0])
			d.checkKeyword(rest[i])

			return
		case i == -1 && len(rest) > 0:
			d.setRange(rangeFrom, rest, nil)
			d.checkKeyword(tokens[0])

			return
		}
		reason = "FROM requires a start date, optionally followed by TO and an end date"

	case qualifierKeywords[kw] != QualifierNone:
		q := qualifierKeywords[kw]
		if q == QualifierInterpreted {
			rest, d.interpreted = splitInterpretedText(rest)
		}
		if len(rest) > 0 {
			d.qualifier = q
			d.setRange(rangeNone, rest, nil)
			d.checkKeyword(tokens[0])

			return
		}
		reason = kw + " requires a date"
	}

	d.setRange(rangeNone, tokens, nil)
	if reason != "" {
		d.reason = reason
	}
}

// setRange parses the start (and optional end) tokens in the date's calendar
// and derives validity from the components.
func (d *dateValue) setRange(kind rangeKind, startTokens, endTokens []string) {
	d.rng = kind
	d.start = parsePoint(d.calendar, startTokens)
	d.valid = d.start.canonical
	d.reason = d.start.reason

	if endTokens != nil {
		d.end = parsePoint(d.calendar, endTokens)
		if d.valid && !d.end.canonical {
			d.valid = false
			d.reason = d.end.reason
		}
		// "BET JUL AND SEP 1857": a start that is a bare month or day-month
		// shares the end's year. The range stays invalid, but Year() is
		// still right. Arbitrary text ("BET unknown AND 1857") inherits
		// nothing: a year that was never written must not be reported.
		if d.start.year == 0 && d.end.year != 0 && isPartialMonth(startTokens) {
			d.start.year = d.end.year
			d.start.precision = PrecisionYear
		}
	}
}

// isPartialMonth reports whether tokens are a Gregorian month name alone or
// preceded by a day of month ("JUL", "15 Jul"): a date component missing
// only its year.
func isPartialMonth(tokens []string) bool {
	switch len(tokens) {
	case 1:
		_, ok := MonthNumber(tokens[0])

		return ok
	case 2: //nolint:mnd // DD MONTH
		_, ok := MonthNumber(tokens[1])

		return ok && dayToken(tokens[0]) > 0
	}

	return false
}

// checkKeyword marks the date invalid when a keyword token is not written in
// its canonical upper-case, unpunctuated form.
func (d *dateValue) checkKeyword(tok string) {
	if canon := keywordOf(tok); d.valid && tok != canon {
		d.valid = false
		d.reason = "keyword " + tok + " must be written " + canon
	}
}

// keywordOf normalizes a token for keyword matching: upper case, without a
// trailing period ("Abt." → "ABT").
func keywordOf(tok string) string {
	return strings.ToUpper(strings.TrimSuffix(tok, "."))
}

// indexKeyword returns the index of the first token equal to kw
// (case-insensitively), or -1.
func indexKeyword(tokens []string, kw string) int {
	for i, tok := range tokens {
		if strings.EqualFold(tok, kw) {
			return i
		}
	}

	return -1
}

// splitInterpretedText separates the trailing "(original text)" of an INT
// date from its date tokens. Without a parenthesized suffix, or with an
// empty one ("INT 1850 ()"), the tokens are returned unchanged with empty
// text, so the empty parentheses stay in the raw body and the date is
// reported invalid rather than silently rewritten as "INT 1850".
func splitInterpretedText(tokens []string) ([]string, string) {
	if len(tokens) == 0 || !strings.HasSuffix(tokens[len(tokens)-1], ")") {
		return tokens, ""
	}
	for i, tok := range tokens {
		if strings.HasPrefix(tok, "(") {
			text := strings.Join(tokens[i:], " ")
			text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "("), ")"))
			if text == "" {
				return tokens, ""
			}

			return tokens[:i], text
		}
	}

	return tokens, ""
}

// parsePoint parses one date component. Gregorian and Julian bodies are
// structured (ISO, or tolerated month names); all other calendars preserve
// the body raw and take the last number as the year.
func parsePoint(cal Calendar, tokens []string) point {
	p := point{raw: strings.Join(tokens, " ")}

	if !cal.hasStructuredMonths() {
		p.year = trailingYear(tokens)
		if p.year > 0 {
			p.precision = PrecisionYear
			p.canonical = true
		} else {
			p.reason = "the year must be the last token of " + strconv.Quote(p.raw)
		}

		return p
	}

	if len(tokens) == 1 && parseISO(tokens[0], &p) {
		return p
	}
	if parseNamedMonth(tokens, &p) {
		return p
	}

	p.year = heuristicYear(p.raw)
	if p.year > 0 {
		p.precision = PrecisionYear
	}
	p.reason = "date body must be YYYY, YYYY-MM, or YYYY-MM-DD"

	return p
}

// parseISO recognizes the ISO shapes Y…Y, Y…Y-MM, Y…Y-MM-DD (1–4 digit year)
// and fills p from them. It reports whether tok has an ISO shape; component
// range errors are recorded on p rather than reported as a shape mismatch.
func parseISO(tok string, p *point) bool {
	parts := strings.Split(tok, "-")
	if len(parts) > 3 || !allDigits(parts[0]) || len(parts[0]) > maxYearDigits {
		return false
	}
	for _, part := range parts[1:] {
		if len(part) != componentDigits || !allDigits(part) {
			return false
		}
	}

	p.year, _ = strconv.Atoi(parts[0])
	p.precision = PrecisionYear
	if len(parts) > 1 {
		p.month, _ = strconv.Atoi(parts[1])
		p.precision = PrecisionMonth
	}
	if len(parts) > 2 {
		p.day, _ = strconv.Atoi(parts[2])
		p.precision = PrecisionDay
	}
	p.exact = checkComponents(p)
	p.canonical = p.exact

	return true
}

// parseNamedMonth recognizes tolerated Gregorian bodies written with a month
// name: "MONTH YYYY", "DD MONTH YYYY", and "MONTH DD, YYYY". Matching is
// case-insensitive and accepts abbreviations or full names. Such a body is
// exact (its canonical form is known) but not canonical as written.
func parseNamedMonth(tokens []string, p *point) bool {
	toks := make([]string, len(tokens))
	for i, tok := range tokens {
		toks[i] = strings.TrimSuffix(tok, ",")
	}

	var month, year, day int
	var ok bool
	switch len(toks) {
	case 2: //nolint:mnd // MONTH YYYY
		month, ok = MonthNumber(toks[0])
		year = yearToken(toks[1])
		p.precision = PrecisionMonth
	case 3: //nolint:mnd // DD MONTH YYYY or MONTH DD, YYYY
		year = yearToken(toks[2])
		if month, ok = MonthNumber(toks[1]); ok {
			day = dayToken(toks[0])
		} else if month, ok = MonthNumber(toks[0]); ok {
			day = dayToken(toks[1])
		}
		ok = ok && day > 0
		p.precision = PrecisionDay
	}
	if !ok || year == 0 {
		p.precision = PrecisionNone

		return false
	}

	p.year, p.month, p.day = year, month, day
	p.exact = checkComponents(p)
	if p.exact {
		p.reason = "month names are not canonical; write " + p.String()
	}

	return true
}

// checkComponents validates year/month/day against the point's precision,
// recording a reason and reporting false when a component is out of range.
func checkComponents(p *point) bool {
	switch {
	case p.year < 1 || p.year > maxYear:
		p.reason = "year must be between 0001 and 9999"
	case p.precision >= PrecisionMonth && (p.month < 1 || p.month > monthsPerYear):
		p.reason = "month must be between 01 and 12"
	case p.precision == PrecisionDay && !validDay(p.month, p.day):
		p.reason = "day is not valid for the month"
	default:
		return true
	}

	return false
}

// heuristicYear extracts a best-effort year from a raw Gregorian/Julian body
// that did not parse. A standalone 4-digit number is preferred, so a day of
// month is never mistaken for the year; otherwise "DD MONTH" pairs are
// stripped and the first remaining number of at most four digits is used
// ("5 JAN 476" → 476, "(10 Aug)" → 0). It returns 0 when nothing plausible
// is found.
func heuristicYear(raw string) int {
	for _, run := range digitRunRegexp.FindAllString(raw, -1) {
		if len(run) == maxYearDigits {
			return atoi(run)
		}
	}

	cleaned := dayMonthRegexp.ReplaceAllString(raw, "")
	for _, run := range standaloneNumberRegexp.FindAllString(cleaned, -1) {
		if len(run) <= maxYearDigits {
			return atoi(run)
		}
	}

	return 0
}

// trailingYear returns the year of a raw-preserved calendar body, or 0. The
// year is the last token and must be a number of at most five digits ("15
// TSH 5765", "1 VEND 0012"); an incomplete body ("15 TSH") has no year, and
// its day of month is never promoted to one.
func trailingYear(tokens []string) int {
	last := tokens[len(tokens)-1]
	if !allDigits(last) || len(last) > maxRawYearDigits {
		return 0
	}

	return atoi(last)
}

// yearToken parses a 1–4 digit year token, returning 0 for anything else.
func yearToken(tok string) int {
	if !allDigits(tok) || len(tok) > maxYearDigits {
		return 0
	}

	return atoi(tok)
}

// dayToken parses a 1–2 digit day token, returning 0 for anything else.
func dayToken(tok string) int {
	if !allDigits(tok) || len(tok) > maxDayDigits {
		return 0
	}

	return atoi(tok)
}

// allDigits reports whether s is non-empty and consists only of ASCII digits.
func allDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return s != ""
}

// atoi converts a string already known to be all digits.
func atoi(s string) int {
	n, _ := strconv.Atoi(s)

	return n
}
