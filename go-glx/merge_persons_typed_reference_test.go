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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	mergeTestReferenceID  string
	mergeTestReferenceKey string
	mergeTestReferenceMap map[mergeTestReferenceKey]mergeTestReferenceID
)

func mergeTypedReferenceCases() []struct {
	name  string
	value func(string) any
} {
	return []struct {
		name  string
		value func(string) any
	}{
		{"named-string", func(id string) any { return mergeTestReferenceID(id) }},
		{"named-map-value", func(id string) any { return map[string]mergeTestReferenceID{"value": mergeTestReferenceID(id)} }},
		{"named-map-key", func(id string) any { return map[mergeTestReferenceKey]string{"value": id} }},
		{"named-map-key-and-value", func(id string) any { return mergeTestReferenceMap{"value": mergeTestReferenceID(id)} }},
		{"pointer-string", func(id string) any {
			value := mergeTestReferenceID(id)

			return &value
		}},
		{"pointer-map-value", func(id string) any {
			value := mergeTestReferenceID(id)

			return map[string]*mergeTestReferenceID{"value": &value}
		}},
		{"interface-pointer-map-value", func(id string) any {
			value := mergeTestReferenceID(id)

			return map[string]any{"value": &value, "date": "1850"}
		}},
		{"pointer-map", func(id string) any {
			value := mergeTestReferenceMap{"value": mergeTestReferenceID(id)}

			return &value
		}},
		{"pointer-interface", func(id string) any {
			var value any = mergeTestReferenceMap{"value": mergeTestReferenceID(id)}

			return &value
		}},
		{"named-string-slice", func(id string) any { return []mergeTestReferenceID{mergeTestReferenceID(id)} }},
		{"typed-map-slice", func(id string) any { return []mergeTestReferenceMap{{"value": mergeTestReferenceID(id)}} }},
		{"mixed-slice", func(id string) any {
			return []any{mergeTestReferenceID(id), mergeTestReferenceMap{"value": mergeTestReferenceID(id)}}
		}},
		{"temporal-named-string", func(id string) any { return TemporalValue{Value: mergeTestReferenceID(id), Date: "1850"} }},
		{"temporal-pointer-string", func(id string) any {
			value := mergeTestReferenceID(id)

			return TemporalValue{Value: &value, Date: "1850"}
		}},
		{"pointer-temporal-named-string", func(id string) any { return &TemporalValue{Value: mergeTestReferenceID(id), Date: "1850"} }},
	}
}

func TestMergePersonsTypedReferenceRewrite(t *testing.T) {
	for _, tc := range mergeTypedReferenceCases() {
		for _, rewrite := range []bool{true, false} {
			action := "rewrite"
			id := "person-drop"
			if !rewrite {
				action = "noop"
				id = "person-keep"
			}
			t.Run(tc.name+"/"+action, func(t *testing.T) {
				original := tc.value(id)
				untouched := tc.value(id)
				archive := newTwoPersonArchive()
				archive.Persons["person-third"] = &Person{Properties: map[string]any{"related_person": original}}
				archive.PersonProperties = map[string]*PropertyDefinition{"related_person": {Label: "Related person", ReferenceType: "persons", MultiValue: new(true)}}
				require.NoError(t, MergeStandardVocabularies(archive))
				require.NotPanics(t, func() {
					_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
					require.NoError(t, err)
				})
				assert.Equal(t, tc.value("person-keep"), archive.Persons["person-third"].Properties["related_person"], "SDK concrete occurrence and wrapper types must survive")
				assert.Equal(t, untouched, original, "detached merge must not mutate caller-owned reference values")
			})
		}
	}
}

func TestMergePersonsTypedReferencePruning(t *testing.T) {
	for _, tc := range mergeTypedReferenceCases() {
		t.Run(tc.name, func(t *testing.T) {
			original := tc.value("candidate")
			untouched := tc.value("candidate")
			archive := newTwoPersonArchive()
			archive.Persons["person-third"] = &Person{Properties: map[string]any{"identity_case": original}}
			archive.PersonProperties = map[string]*PropertyDefinition{"identity_case": {Label: "Identity case", ReferenceType: "relationships", MultiValue: new(true)}}
			archive.Relationships = map[string]*Relationship{"candidate": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-keep"}, {Person: "person-drop"}}}}
			require.NoError(t, MergeStandardVocabularies(archive))
			require.NotPanics(t, func() {
				_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
				require.NoError(t, err)
			})
			assert.NotContains(t, archive.Persons["person-third"].Properties, "identity_case")
			assert.Equal(t, untouched, original)
		})
	}
}

func TestMergePersonsTypedReferenceFailureLeavesInputUnchanged(t *testing.T) {
	archive := newTwoPersonArchive()
	archive.Persons["person-third"] = &Person{Properties: map[string]any{"related_person": mergeTestReferenceMap{"value": "person-drop"}}}
	archive.PersonProperties = map[string]*PropertyDefinition{"related_person": {Label: "Related person", ReferenceType: "persons"}}
	require.NoError(t, MergeStandardVocabularies(archive))
	before := cloneMergeArchive(archive)
	require.NotPanics(t, func() {
		_, err := MergePersons(archive, "person-keep", "person-keep", MergePersonsOptions{})
		require.Error(t, err)
	})
	assert.Equal(t, before, archive)
}

func TestMergePersonsTypedReferencePruningRetainsOtherOccurrences(t *testing.T) {
	for _, tc := range mergeTypedReferenceCases() {
		t.Run(tc.name, func(t *testing.T) {
			original := []any{tc.value("candidate"), tc.value("other")}
			untouched := []any{tc.value("candidate"), tc.value("other")}
			archive := newTwoPersonArchive()
			archive.Persons["person-third"] = &Person{Properties: map[string]any{"identity_case": original}}
			archive.PersonProperties = map[string]*PropertyDefinition{"identity_case": {Label: "Identity case", ReferenceType: "relationships", MultiValue: new(true)}}
			archive.Relationships = map[string]*Relationship{
				"candidate": {Type: RelationshipTypePossiblySamePerson, Participants: []Participant{{Person: "person-keep"}, {Person: "person-drop"}}},
				"other":     {Type: RelationshipTypeAssociate, Participants: []Participant{{Person: "person-keep"}, {Person: "person-third"}}},
			}
			require.NoError(t, MergeStandardVocabularies(archive))
			require.NotPanics(t, func() {
				_, err := MergePersons(archive, "person-keep", "person-drop", MergePersonsOptions{})
				require.NoError(t, err)
			})
			assert.Equal(t, []any{tc.value("other")}, archive.Persons["person-third"].Properties["identity_case"])
			assert.Equal(t, untouched, original)
		})
	}
}
