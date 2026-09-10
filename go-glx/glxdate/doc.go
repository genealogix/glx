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

// Package glxdate is the single source of truth for the GLX date format.
//
// It implements the "Date Format Standard" from the GLX specification
// (specification/2-core-concepts.md): ISO-style simple dates (YYYY, YYYY-MM,
// YYYY-MM-DD), the keyword qualifiers ABT/BEF/AFT/CAL/EST/INT, the range
// forms BET…AND, FROM…TO, open-ended FROM and open-start TO, the BCE era
// suffix on Gregorian and Julian dates ("0044-03-15 BCE"), and calendar
// prefixes (JULIAN, HEBREW, FRENCH_R, or an underscore-prefixed extension
// calendar such as _ROMAN).
//
// Every consumer that needs to parse, validate, canonicalize, or extract a
// component from a date string funnels through [Parse], so "parse vs extract
// vs validate" can never disagree.
//
// Parse is deliberately tolerant on input and strict on output:
//
//   - Gregorian and Julian bodies written with month names in any letter
//     case, abbreviated or in full ("15 March 1850", "1 JANUARY 1900",
//     "March 15, 1850"), are recognized and canonicalize to ISO form.
//     A month name is unambiguous, so this is recovery, not guessing.
//   - Keywords are matched case-insensitively and unambiguous spellings are
//     folded ("Abt 1850", "Bet 1880 and 1890", "circa 1850", "before 1900"),
//     as are the era spellings BC, B.C. and B.C.E. ("510 BC" → "0510 BCE").
//   - Numeric day/month forms ("15/01/1900", "01-15-1900") are never
//     interpreted: the spec forbids guessing day-vs-month order. Such dates,
//     dual years ("1731/32") and other free text are preserved verbatim and
//     reported as invalid, while [Date.Year] still returns the best-effort
//     year (the earliest 4-digit run or standalone 3-digit number, so a day
//     of month is never mistaken for the year).
//   - Hebrew, French Republican and extension-calendar bodies keep their raw
//     month names; only the year (the last token) is extracted.
//   - No calendar conversion is ever performed.
//
// [Date.Valid] reports whether the input is a well-formed GLX date, which is
// what validation warns about; whitespace and 1–3 digit years are normalized
// without being reported. [Date.String] renders the canonical form whenever
// the components were determined, and the raw text otherwise.
//
// The GEDCOM encoding of the same model lives here as well, so the GLX
// grammar is never re-implemented by a converter: [FromGEDCOM] turns a
// GEDCOM DATE payload (calendar escape, any tolerated spelling) into the
// GLX string an importer stores, via [Canonicalize]; [Date.GEDCOM] renders a
// Date as a GEDCOM DATE payload.
package glxdate
