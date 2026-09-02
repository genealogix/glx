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
// YYYY-MM-DD), the keyword qualifiers ABT/BEF/AFT/CAL/INT, the range forms
// BET…AND, FROM…TO and open-ended FROM, and calendar prefixes (JULIAN, HEBREW,
// FRENCH_R, or any preserved unknown calendar name).
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
//   - Keywords are matched case-insensitively ("Abt 1850", "Bet 1880 and 1890").
//   - Numeric day/month forms ("15/01/1900", "01-15-1900") are never
//     interpreted: the spec forbids guessing day-vs-month order. Such dates,
//     dual years ("1731/32"), BCE dates and other free text are preserved
//     verbatim and reported as invalid, while [Date.Year] still returns the
//     best-effort year (a 4-digit token is preferred, so a day of month is
//     never mistaken for the year).
//   - Hebrew, French Republican and unknown-calendar bodies keep their raw
//     month names; only the year (the last number) is extracted.
//   - No calendar conversion is ever performed.
//
// [Date.Valid] reports whether the input is in canonical GLX form, which is
// what validation warns about. [Date.String] renders the canonical form
// whenever the components were determined, and the raw text otherwise.
package glxdate
