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
	glxlib "github.com/genealogix/glx/go-glx"
)

// personSex returns the person's recorded sex, with a safe back-compat
// fallback to `gender` for pre-#528 archives.
//
// Fallback rules: if `sex` is empty and `gender` holds a value that exists
// in the sex vocabulary (male, female, unknown, other, not_recorded), the
// gender value is used. Identity-only values (notably `nonbinary`) are
// never surfaced as Sex — they can't have come from a legacy archive and
// belong to the post-split `gender` (identity) semantics.
//
// Operators on pre-split archives can run `glx migrate --rename-gender-to-sex`
// to make the data explicit.
func personSex(person *glxlib.Person) string {
	if person == nil {
		return ""
	}
	if v := propertyString(person.Properties, glxlib.PersonPropertySex); v != "" {
		return v
	}
	legacy := propertyString(person.Properties, glxlib.PersonPropertyGender)
	if isLegacySexValue(legacy) {
		return legacy
	}

	return ""
}

// isLegacySexValue reports whether v could plausibly be a pre-#528 `gender`
// property value that actually denoted recorded sex. The canonical
// post-split identity-only value (`nonbinary`) is excluded.
func isLegacySexValue(v string) bool {
	switch v {
	case glxlib.SexMale, glxlib.SexFemale, glxlib.SexUnknown,
		glxlib.SexOther, glxlib.SexNotRecorded:
		return true
	}

	return false
}

// displayableGenderIdentity returns the gender value that should be shown as
// a distinct "Gender" row in vitals/summary output. Returns "" when the
// gender property either (a) is unset, or (b) would already be shown as Sex
// via the legacy-gender fallback path — this prevents duplicate rows on
// pre-split archives that only carry `gender: "male"`.
//
// The Gender row is surfaced when:
//   - both `sex` and `gender` are explicitly set (genuine dual archive), or
//   - `gender` holds an identity-only value (e.g. `nonbinary`) that the Sex
//     fallback would NOT pick up.
func displayableGenderIdentity(person *glxlib.Person) string {
	if person == nil {
		return ""
	}
	gender := propertyString(person.Properties, glxlib.PersonPropertyGender)
	if gender == "" {
		return ""
	}
	sex := propertyString(person.Properties, glxlib.PersonPropertySex)
	if sex == "" && isLegacySexValue(gender) {
		// Pre-split archive: `gender: "male"` already surfaces as Sex via
		// the legacy fallback. Showing a duplicate Gender row would print
		// the same value twice.
		return ""
	}

	return gender
}
