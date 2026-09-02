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

	"github.com/genealogix/glx/go-glx/glxdate"
)

// parseGEDCOMDate converts a GEDCOM date value to a GLX DateString.
//
// It is the GEDCOM → GLX adapter: the GEDCOM-specific calendar escape
// (@#DJULIAN@, @#DHEBREW@, @#DFRENCH R@, @#DGREGORIAN@) is translated to a
// GLX calendar prefix here, and everything else is delegated to glxdate,
// which owns the neutral date model. Standard GEDCOM bodies ("15 MAR 1850",
// "ABT 1850", "BET 1880 AND 1890", "FROM 1900 TO 1950") canonicalize to the
// GLX form ("1850-03-15", "ABT 1850", …); so do the common dialect variants
// with full or mixed-case month names and keywords ("1 January 1900",
// "Bet 1880 and 1890", "ABT. 1850").
//
// Anything that cannot be canonicalized without guessing — numeric
// day/month forms, dual years ("1731/32"), BCE dates, free text — is
// preserved verbatim so it survives roundtrip; validation later flags it.
func parseGEDCOMDate(gedcomDate string) DateString {
	date := strings.TrimSpace(gedcomDate)
	if date == "" {
		return ""
	}

	calendar, body := extractCalendar(date)
	body = strings.TrimSpace(body)
	if body == "" {
		// Calendar escape with no date body (e.g., "@#DJULIAN@" or "@#DGREGORIAN@").
		// Preserve the original raw GEDCOM string so roundtrip can re-emit it.
		return DateString(gedcomDate)
	}

	full := body
	if calendar != "" {
		full = calendar + " " + body
	}

	return DateString(canonicalizeDate(full))
}

// canonicalizeDate returns the canonical GLX form of a date string when its
// components can be determined without guessing, and the input unchanged
// otherwise. A date is only rewritten when its canonical form itself parses
// as valid, so raw-preserved bodies are never altered.
func canonicalizeDate(s string) string {
	d, err := glxdate.Parse(s)
	if err == nil {
		return d.String()
	}

	if canon := d.String(); canon != s {
		if _, err := glxdate.Parse(canon); err == nil {
			return canon
		}
	}

	return s
}
