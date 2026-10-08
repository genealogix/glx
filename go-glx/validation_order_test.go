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
	"cmp"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unorderedFindingsArchive builds an archive whose findings span several
// entity types and many IDs, enough that map iteration order would show up.
func unorderedFindingsArchive() *GLXFile {
	archive := &GLXFile{
		Persons:          map[string]*Person{},
		Events:           map[string]*Event{},
		Places:           map[string]*Place{},
		PersonProperties: map[string]*PropertyDefinition{"born_on": {}},
	}
	for i := range 20 {
		archive.Persons[fmt.Sprintf("person-%02d", i)] = &Person{Properties: map[string]any{"shoe_size": i}}
		archive.Events[fmt.Sprintf("event-%02d", i)] = &Event{
			Type:         "land_transaction",
			Participants: []Participant{{Person: "person-missing", Role: "grantor"}},
		}
		archive.Places[fmt.Sprintf("place-%02d", i)] = &Place{Name: "Somewhere", ParentID: "place-missing"}
	}

	return archive
}

func TestValidateSortsFindings(t *testing.T) {
	result := unorderedFindingsArchive().Validate()
	require.NotEmpty(t, result.Errors)
	require.NotEmpty(t, result.Warnings)

	assert.True(t, slices.IsSortedFunc(result.Errors, func(a, b ValidationError) int {
		return cmp.Or(
			cmp.Compare(a.SourceType, b.SourceType),
			cmp.Compare(a.SourceID, b.SourceID),
			cmp.Compare(a.Message, b.Message),
		)
	}), "errors are not sorted by entity type, ID, message")
	assert.True(t, slices.IsSortedFunc(result.Warnings, func(a, b ValidationWarning) int {
		return cmp.Or(
			cmp.Compare(a.SourceType, b.SourceType),
			cmp.Compare(a.SourceID, b.SourceID),
			cmp.Compare(a.Message, b.Message),
		)
	}), "warnings are not sorted by entity type, ID, message")

	// Entity types group together: every events finding precedes every places one.
	assert.Equal(t, EntityTypeEvents, result.Errors[0].SourceType)
	assert.Equal(t, "event-00", result.Errors[0].SourceID)
	assert.Equal(t, EntityTypePlaces, result.Errors[len(result.Errors)-1].SourceType)
	assert.Equal(t, "place-19", result.Errors[len(result.Errors)-1].SourceID)
}

func TestValidateFindingsStableAcrossRuns(t *testing.T) {
	first := unorderedFindingsArchive().Validate()
	for range 10 {
		again := unorderedFindingsArchive().Validate()
		assert.Equal(t, first.Errors, again.Errors)
		assert.Equal(t, first.Warnings, again.Warnings)
	}
}
