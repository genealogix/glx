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
// spelling of the target version; the two differ only in the era suffix
// ("44 BCE" in 7.0, "44 B.C." in 5.5.1). Examples:
//
//	"1850-03-15"          -> "15 MAR 1850"
//	"1850-03"             -> "MAR 1850"
//	"ABT 1850-03-15"      -> "ABT 15 MAR 1850"
//	"JULIAN 1731-03-15"   -> "@#DJULIAN@ 15 MAR 1731"
//	"BET 1880 AND 1890"   -> "BET 1880 AND 1890"
//
// A date that is not in canonical form is rendered as glxdate would render
// it, so nothing is invented on export that was not understood on import.
func formatGEDCOMDate(date DateString, version GEDCOMVersion) string {
	d, _ := date.Parse()
	if version == GEDCOM551 {
		return d.GEDCOM551()
	}

	return d.GEDCOM()
}
