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
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConflictVerdicts(t *testing.T) {
	data, err := os.ReadFile("testdata/conflict-verdicts.json")
	require.NoError(t, err)
	var cases []struct {
		Name, Property, Kind, Reference, Vocabulary string
		Temporal                                    bool
		Width                                       *int
		Values                                      []FactValue
		Verdict                                     Verdict
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	places := map[string]*Place{"hartford": {Name: "Hartford County", ParentID: "connecticut"}, "connecticut": {Name: "Connecticut"}, "springfield-il": {Name: "Springfield"}, "springfield-ma": {Name: "Springfield"}}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			def := &PropertyDefinition{Temporal: &tc.Temporal, ValueType: tc.Kind, ReferenceType: tc.Reference, VocabularyType: tc.Vocabulary}
			result := CompareFacts(tc.Values, def, places, ComparisonOptions{ApproximationYears: tc.Width})
			require.Len(t, result, 1)
			require.Equal(t, tc.Verdict, result[0].Verdict)
			reversed := []FactValue{tc.Values[1], tc.Values[0]}
			require.Equal(t, result[0].Verdict, CompareFacts(reversed, def, places, ComparisonOptions{ApproximationYears: tc.Width})[0].Verdict)
		})
	}
}

func TestConflictCompatibilityDoesNotBridge(t *testing.T) {
	values := []FactValue{{Value: "1850-01-01"}, {Value: "1850"}, {Value: "1850-12-31"}}
	comparisons := CompareFacts(values, &PropertyDefinition{ValueType: "date"}, nil, ComparisonOptions{})
	require.Equal(t, VerdictDefinite, comparisons[1].Verdict)
}

func TestConflictPlaceCycle(t *testing.T) {
	places := map[string]*Place{"a": {ParentID: "b"}, "b": {ParentID: "a"}}
	result := CompareFacts([]FactValue{{Value: "a"}, {Value: "c"}}, &PropertyDefinition{ReferenceType: "places"}, places, ComparisonOptions{})
	require.Equal(t, VerdictDefinite, result[0].Verdict)
}

func TestMergeTemporalHistory(t *testing.T) {
	for _, shape := range []string{"maps", "mixed", "lists"} {
		t.Run(shape, func(t *testing.T) {
			a := map[string]any{"value": "Leeds", "date": "1851"}
			b := map[string]any{"value": "London", "date": "1875"}
			archive := &GLXFile{Persons: map[string]*Person{"keep": {Properties: map[string]any{"residence": a, "sex": "male"}}, "drop": {Properties: map[string]any{"residence": b, "sex": "male"}}}}
			require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))
			if shape != "maps" {
				archive.Persons["keep"].Properties["residence"] = []any{a}
			}
			if shape == "lists" {
				archive.Persons["drop"].Properties["residence"] = []any{b, b}
			}
			result, err := MergePersons(archive, "keep", "drop", MergePersonsOptions{})
			require.NoError(t, err)
			require.Empty(t, result.Conflicts)
			require.Equal(t, []any{a, b}, archive.Persons["keep"].Properties["residence"])
		})
	}
}

func TestMergeTemporalConflictPreservesUnrelatedHistory(t *testing.T) {
	a := map[string]any{"value": "Leeds", "date": "FROM 1850 TO 1880"}
	b := map[string]any{"value": "London", "date": "1875"}
	later := map[string]any{"value": "York", "date": "1900"}
	archive := &GLXFile{Persons: map[string]*Person{"keep": {Properties: map[string]any{"residence": []any{a, later}}}, "drop": {Properties: map[string]any{"residence": b}}}}
	require.NoError(t, LoadStandardVocabulariesIntoGLX(archive))
	result, err := MergePersons(archive, "keep", "drop", MergePersonsOptions{KeepNewest: true})
	require.NoError(t, err)
	require.Len(t, result.Conflicts, 1)
	require.Equal(t, []any{b, later}, archive.Persons["keep"].Properties["residence"])
	require.Equal(t, "London (1875)", FormatPropertyValue(b))
	require.Equal(t, "London (1875); York (1900)", FormatPropertyValue([]any{b, later}))
}

func TestValidateReversedRangeWarning(t *testing.T) {
	for _, date := range []string{"FROM 1920 TO 1870", "BET 1920 AND 1870"} {
		result := &ValidationResult{}
		archive := &GLXFile{}
		archive.validateDateFormat(EntityTypeEvents, "event", "date", date, result)
		require.Empty(t, result.Errors)
		require.Len(t, result.Warnings, 1)
		require.Contains(t, result.Warnings[0].Message, "reversed date range")
	}
}

func TestMergeTemporalScalarProducesValidHistory(t *testing.T) {
	temporal := true
	archive := &GLXFile{
		Persons:          map[string]*Person{"keep": {Properties: map[string]any{"trade": "miller"}}, "drop": {Properties: map[string]any{"trade": map[string]any{"value": "farmer", "date": "1875"}}}},
		PersonProperties: map[string]*PropertyDefinition{"trade": {Temporal: &temporal, ValueType: "string"}},
	}
	result, err := MergePersons(archive, "keep", "drop", MergePersonsOptions{})
	require.NoError(t, err)
	require.Empty(t, result.Conflicts)
	require.Equal(t, []any{map[string]any{"value": "farmer", "date": "1875"}, map[string]any{"value": "miller"}}, archive.Persons["keep"].Properties["trade"])
	validation := &ValidationResult{}
	archive.validatePropertyValue(EntityTypePersons, "keep", "trade", archive.Persons["keep"].Properties["trade"], archive.PersonProperties["trade"], validation)
	require.Empty(t, validation.Errors)
	require.Empty(t, validation.Warnings)
}
