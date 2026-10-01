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
	"github.com/genealogix/glx/go-glx/glxdate"
)

// parseGEDCOMDate converts a GEDCOM DATE value to a GLX DateString. The
// calendar escape, canonicalization, and preservation rules all live in
// glxdate.FromGEDCOM; this is only the DateString adapter.
func parseGEDCOMDate(gedcomDate string) DateString {
	return DateString(glxdate.FromGEDCOM(gedcomDate))
}

// formatGEDCOMDate converts a GLX DateString to a GEDCOM DATE value in the
// spelling of the target version. GEDCOM 7 writes the calendar as a tag
// before each date and the era as BCE; 5.5.1 writes a calendar escape first
// and B.C. Anything other than GEDCOM70 gets the 5.5.1 spelling, matching
// ExportGEDCOM's header choice. Examples:
//
//	"1850-03-15"          -> "15 MAR 1850"
//	"ABT 1850-03-15"      -> "ABT 15 MAR 1850"
//	"JULIAN ABT 1731"     -> "ABT JULIAN 1731"        (7.0)
//	"JULIAN ABT 1731"     -> "@#DJULIAN@ ABT 1731"    (5.5.1)
//	"0044 BCE"            -> "0044 BCE" (7.0), "0044 B.C." (5.5.1)
//
// A date that is not in canonical form is rendered as glxdate would render
// it, so nothing is invented on export that was not understood on import.
// INT dates are written with their keyword in both versions; GEDCOM 7 moved
// interpreted text to a PHRASE substructure, which the exporter does not
// emit yet.
func formatGEDCOMDate(date DateString, version GEDCOMVersion) string {
	d, _ := date.Parse()
	if version == GEDCOM70 {
		return d.GEDCOM()
	}

	return d.GEDCOM551()
}
