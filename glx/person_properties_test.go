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
	"testing"

	"github.com/stretchr/testify/assert"

	glxlib "github.com/genealogix/glx/go-glx"
)

// TestPersonSex_PrefersSexOverLegacyGender verifies that when a person has
// both properties set, `sex` wins. This is the canonical post-split shape and
// `gender` is treated strictly as identity.
func TestPersonSex_PrefersSexOverLegacyGender(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			glxlib.PersonPropertySex:    "female",
			glxlib.PersonPropertyGender: "male",
		},
	}
	assert.Equal(t, "female", personSex(person))
}

// TestPersonSex_FallsBackToLegacyGenderMale is the back-compat contract — a
// pre-split archive carrying `gender: "male"` must still report "male" for
// Sex via `personSex` so the CLI keeps working until the user migrates.
func TestPersonSex_FallsBackToLegacyGenderMale(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			glxlib.PersonPropertyGender: "male",
		},
	}
	assert.Equal(t, "male", personSex(person))
}

// TestPersonSex_FallsBackToLegacyGenderFemale — same as above, female side.
func TestPersonSex_FallsBackToLegacyGenderFemale(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			glxlib.PersonPropertyGender: "female",
		},
	}
	assert.Equal(t, "female", personSex(person))
}

// TestPersonSex_FallsBackForSexVocabValues covers the full set of sex-vocab
// values (other, unknown, not_recorded) reached via the legacy-gender
// fallback path. Each is valid in `sex_types` and therefore safe to surface
// as Sex when only `gender` is populated on a pre-split archive.
func TestPersonSex_FallsBackForSexVocabValues(t *testing.T) {
	cases := []string{"other", "unknown", "not_recorded"}
	for _, v := range cases {
		t.Run(v, func(t *testing.T) {
			person := &glxlib.Person{
				Properties: map[string]any{glxlib.PersonPropertyGender: v},
			}
			assert.Equal(t, v, personSex(person))
		})
	}
}

// TestPersonSex_DoesNotFallBackForIdentityOnlyValues verifies that identity-
// only gender values (the canonical example: "nonbinary", which exists in
// `gender_types` but NOT in `sex_types`) are never surfaced as Sex. Doing so
// would corrupt identity data into the sex field.
func TestPersonSex_DoesNotFallBackForIdentityOnlyValues(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			glxlib.PersonPropertyGender: glxlib.GenderNonbinary,
		},
	}
	assert.Empty(t, personSex(person))
}

// TestPersonSex_EmptyWhenNeitherPropertySet covers the baseline: no sex, no
// gender, returns "".
func TestPersonSex_EmptyWhenNeitherPropertySet(t *testing.T) {
	person := &glxlib.Person{Properties: map[string]any{}}
	assert.Empty(t, personSex(person))
}

// TestPersonSex_EmptyStringSexFallsThroughToGender verifies that an explicit
// empty `sex: ""` still activates the fallback to `gender`, rather than
// short-circuiting to "".
func TestPersonSex_EmptyStringSexFallsThroughToGender(t *testing.T) {
	person := &glxlib.Person{
		Properties: map[string]any{
			glxlib.PersonPropertySex:    "",
			glxlib.PersonPropertyGender: "male",
		},
	}
	assert.Equal(t, "male", personSex(person))
}

// TestDisplayableGenderIdentity covers the predicate that decides whether a
// "Gender:" row is shown in vitals/summary. The central invariant: the row
// must NOT duplicate the Sex row. For pre-split archives where `gender`
// holds a legacy sex value like "male", the Sex row already surfaces it
// via fallback, so the Gender row is suppressed.
func TestDisplayableGenderIdentity(t *testing.T) {
	cases := []struct {
		name string
		sex  string
		gen  string
		want string
	}{
		// Legacy pre-split: gender is surfacing as Sex via fallback.
		// Gender row would duplicate the Sex row, so it must be suppressed.
		{"legacy_male_no_sex", "", "male", ""},
		{"legacy_female_no_sex", "", "female", ""},
		{"legacy_unknown_no_sex", "", "unknown", ""},
		{"legacy_other_no_sex", "", "other", ""},
		{"legacy_not_recorded_no_sex", "", "not_recorded", ""},

		// Post-split with both set: Gender adds independent information.
		{"dual_binary", "male", "female", "female"},
		{"dual_nonbinary", "male", "nonbinary", "nonbinary"},
		{"dual_same_value", "male", "male", "male"}, // distinct field, show both

		// Identity-only archive: Sex fallback would return "", so Gender
		// adds information even without an explicit sex.
		{"nonbinary_no_sex", "", "nonbinary", "nonbinary"},
		{"other_identity_no_sex", "", "two-spirit", "two-spirit"},

		// Neither set.
		{"empty", "", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := map[string]any{}
			if tc.sex != "" {
				props[glxlib.PersonPropertySex] = tc.sex
			}
			if tc.gen != "" {
				props[glxlib.PersonPropertyGender] = tc.gen
			}
			got := displayableGenderIdentity(&glxlib.Person{Properties: props})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDisplayableGenderIdentity_NilPerson(t *testing.T) {
	assert.Empty(t, displayableGenderIdentity(nil))
}

// TestPersonSex_TemporalShapes covers the three shapes `sex` can take on disk
// given its `temporal: true` declaration in person-properties. Each must
// resolve to the canonical vocab key rather than being fmt.Sprint'd to
// "map[...]" / "[...]" and silently breaking downstream logic.
func TestPersonSex_TemporalShapes(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want string
	}{
		{"plain_string", "male", "male"},
		{"single_temporal_map", map[string]any{"value": "female", "date": "1850"}, "female"},
		{"temporal_list_first_wins", []any{
			map[string]any{"value": "male", "date": "1850"},
			map[string]any{"value": "female", "date": "1860"},
		}, "male"},
		{"temporal_list_bare_strings", []any{"male"}, "male"},
		{"empty_map", map[string]any{}, ""},
		{"empty_list", []any{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			person := &glxlib.Person{
				Properties: map[string]any{glxlib.PersonPropertySex: tc.val},
			}
			assert.Equal(t, tc.want, personSex(person))
		})
	}
}

// TestDisplayableGenderIdentity_TemporalShapes mirrors the sex-side test:
// `gender` is also marked temporal, so the same shapes apply. The predicate
// must still suppress legacy-sex duplicates and surface identity values.
func TestDisplayableGenderIdentity_TemporalShapes(t *testing.T) {
	cases := []struct {
		name   string
		gender any
		sex    any
		want   string
	}{
		{
			"identity_temporal_map_no_sex",
			map[string]any{"value": "nonbinary", "date": "2024"},
			nil, "nonbinary",
		},
		{
			"identity_temporal_list_no_sex",
			[]any{map[string]any{"value": "nonbinary"}},
			nil, "nonbinary",
		},
		{
			"legacy_male_temporal_map_no_sex",
			map[string]any{"value": "male"},
			nil, "",
		},
		{
			"dual_temporal",
			map[string]any{"value": "nonbinary"},
			"male", "nonbinary",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := map[string]any{glxlib.PersonPropertyGender: tc.gender}
			if tc.sex != nil {
				props[glxlib.PersonPropertySex] = tc.sex
			}
			got := displayableGenderIdentity(&glxlib.Person{Properties: props})
			assert.Equal(t, tc.want, got)
		})
	}
}
