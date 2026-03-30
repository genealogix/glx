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

func TestDiffArchives_EmptyArchives(t *testing.T) {
	old := &GLXFile{}
	newArchive := &GLXFile{}
	result := DiffArchives(old, newArchive,"")
	assert.Empty(t, result.Changes)
	assert.Equal(t, 0, result.Stats.Added)
	assert.Equal(t, 0, result.Stats.Modified)
	assert.Equal(t, 0, result.Stats.Removed)
}

func TestDiffArchives_AddedPerson(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {
				Properties: map[string]any{
					"name":       "John Smith",
					"occupation": "blacksmith",
				},
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	assert.Equal(t, ChangeAdded, result.Changes[0].Kind)
	assert.Equal(t, EntityTypePersons, result.Changes[0].EntityType)
	assert.Equal(t, "person-john", result.Changes[0].ID)
	assert.Contains(t, result.Changes[0].Summary, "John Smith")
	assert.Equal(t, 1, result.Stats.Added)
}

func TestDiffArchives_RemovedPerson(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {
				Properties: map[string]any{"name": "Jane Webb"},
			},
		},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	assert.Equal(t, ChangeRemoved, result.Changes[0].Kind)
	assert.Equal(t, "person-mary", result.Changes[0].ID)
	assert.Equal(t, 1, result.Stats.Removed)
}

func TestDiffArchives_ModifiedPerson(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {
				Properties: map[string]any{
					"name":       "Jane Webb",
					"occupation": "weaver",
				},
			},
		},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {
				Properties: map[string]any{
					"name":       "Jane Webb",
					"occupation": "teacher",
					"residence":  "place-london",
				},
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	c := result.Changes[0]
	assert.Equal(t, ChangeModified, c.Kind)
	assert.Equal(t, "person-mary", c.ID)
	assert.Equal(t, 1, result.Stats.Modified)

	// Should have field changes for occupation and residence
	fieldPaths := make(map[string]bool)
	for _, f := range c.Fields {
		fieldPaths[f.Path] = true
	}
	assert.True(t, fieldPaths["properties.occupation"], "should detect occupation change")
	assert.True(t, fieldPaths["properties.residence"], "should detect residence addition")
}

func TestDiffArchives_UnchangedEntity(t *testing.T) {
	person := &Person{
		Properties: map[string]any{"name": "John Smith"},
	}
	old := &GLXFile{
		Persons: map[string]*Person{"person-john": person},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {
				Properties: map[string]any{"name": "John Smith"},
			},
		},
	}

	result := DiffArchives(old, newArchive,"")
	assert.Empty(t, result.Changes)
}

func TestDiffArchives_MultipleEntityTypes(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{},
		Sources: map[string]*Source{},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-john": {Properties: map[string]any{"name": "John"}},
		},
		Sources: map[string]*Source{
			"source-census": {Title: "1860 Census"},
		},
		Citations: map[string]*Citation{
			"citation-1": {SourceID: "source-census"},
		},
	}

	result := DiffArchives(old, newArchive,"")

	assert.Equal(t, 3, result.Stats.Added)
	assert.Equal(t, 1, result.Stats.NewSources)
	assert.Equal(t, 1, result.Stats.NewCitations)
}

func TestDiffArchives_ConfidenceUpgrade(t *testing.T) {
	old := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-birth": {
				Subject:    EntityRef{Event: "event-birth-mary"},
				Property:   "date",
				Value:      "1832",
				Confidence: ConfidenceLevelLow,
			},
		},
	}
	newArchive := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-birth": {
				Subject:    EntityRef{Event: "event-birth-mary"},
				Property:   "date",
				Value:      "1832",
				Confidence: ConfidenceLevelHigh,
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	assert.Equal(t, 1, result.Stats.ConfidenceUpgrades)
	assert.Equal(t, 0, result.Stats.ConfidenceDowngrades)
}

func TestDiffArchives_ConfidenceDowngrade(t *testing.T) {
	old := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-birth": {
				Subject:    EntityRef{Event: "event-birth-mary"},
				Property:   "date",
				Value:      "1832",
				Confidence: ConfidenceLevelHigh,
			},
		},
	}
	newArchive := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-birth": {
				Subject:    EntityRef{Event: "event-birth-mary"},
				Property:   "date",
				Value:      "1832",
				Confidence: ConfidenceLevelMedium,
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	assert.Equal(t, 0, result.Stats.ConfidenceUpgrades)
	assert.Equal(t, 1, result.Stats.ConfidenceDowngrades)
}

func TestDiffArchives_PersonFilter(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {Properties: map[string]any{"name": "Jane Webb"}},
			"person-john": {Properties: map[string]any{"name": "John Smith"}},
		},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {Properties: map[string]any{"name": "Jane Webb", "occupation": "weaver"}},
			"person-john": {Properties: map[string]any{"name": "John Smith", "occupation": "farmer"}},
		},
	}

	result := DiffArchives(old, newArchive,"person-mary")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "person-mary", result.Changes[0].ID)
}

func TestDiffArchives_PersonFilter_IncludesAssertions(t *testing.T) {
	old := &GLXFile{
		Assertions: map[string]*Assertion{},
	}
	newArchive := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-1": {
				Subject:  EntityRef{Person: "person-mary"},
				Property: "occupation",
				Value:    "weaver",
			},
			"assertion-2": {
				Subject:  EntityRef{Person: "person-john"},
				Property: "occupation",
				Value:    "farmer",
			},
		},
	}

	result := DiffArchives(old, newArchive,"person-mary")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "assertion-1", result.Changes[0].ID)
}

func TestDiffArchives_PersonFilter_IncludesEvents(t *testing.T) {
	old := &GLXFile{
		Events: map[string]*Event{},
	}
	newArchive := &GLXFile{
		Events: map[string]*Event{
			"event-1": {
				Type:         "census",
				Participants: []Participant{{Person: "person-mary", Role: "principal"}},
			},
			"event-2": {
				Type:         "birth",
				Participants: []Participant{{Person: "person-john", Role: "principal"}},
			},
		},
	}

	result := DiffArchives(old, newArchive,"person-mary")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "event-1", result.Changes[0].ID)
}

func TestDiffArchives_SortOrder(t *testing.T) {
	old := &GLXFile{}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-b": {Properties: map[string]any{"name": "B"}},
			"person-a": {Properties: map[string]any{"name": "A"}},
		},
		Sources: map[string]*Source{
			"source-x": {Title: "X"},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 3)
	// Persons should come before sources
	assert.Equal(t, EntityTypePersons, result.Changes[0].EntityType)
	assert.Equal(t, EntityTypePersons, result.Changes[1].EntityType)
	assert.Equal(t, EntityTypeSources, result.Changes[2].EntityType)
	// Persons should be sorted by ID
	assert.Equal(t, "person-a", result.Changes[0].ID)
	assert.Equal(t, "person-b", result.Changes[1].ID)
}

func TestCompareEntity_NoChanges(t *testing.T) {
	old := &Person{Properties: map[string]any{"name": "John"}}
	newPerson := &Person{Properties: map[string]any{"name": "John"}}

	fields := compareEntity(old, newPerson)
	assert.Empty(t, fields)
}

func TestCompareEntity_FieldAdded(t *testing.T) {
	old := &Person{Properties: map[string]any{"name": "John"}}
	newPerson := &Person{Properties: map[string]any{"name": "John", "occupation": "blacksmith"}}

	fields := compareEntity(old, newPerson)
	require.Len(t, fields, 1)
	assert.Equal(t, "properties.occupation", fields[0].Path)
	assert.Equal(t, "(none)", fields[0].OldValue)
	assert.Contains(t, fields[0].NewValue, "blacksmith")
}

func TestCompareEntity_FieldRemoved(t *testing.T) {
	old := &Person{Properties: map[string]any{"name": "John", "occupation": "blacksmith"}}
	newPerson := &Person{Properties: map[string]any{"name": "John"}}

	fields := compareEntity(old, newPerson)
	require.Len(t, fields, 1)
	assert.Equal(t, "properties.occupation", fields[0].Path)
	assert.Contains(t, fields[0].OldValue, "blacksmith")
	assert.Equal(t, "(none)", fields[0].NewValue)
}

func TestCompareEntity_FieldChanged(t *testing.T) {
	old := &Person{Properties: map[string]any{"name": "John", "occupation": "apprentice"}}
	newPerson := &Person{Properties: map[string]any{"name": "John", "occupation": "blacksmith"}}

	fields := compareEntity(old, newPerson)
	require.Len(t, fields, 1)
	assert.Equal(t, "properties.occupation", fields[0].Path)
	assert.Contains(t, fields[0].OldValue, "apprentice")
	assert.Contains(t, fields[0].NewValue, "blacksmith")
}

func TestCompareEntity_NotesChanged(t *testing.T) {
	old := &Person{Properties: map[string]any{"name": "John"}, Notes: "old notes"}
	newPerson := &Person{Properties: map[string]any{"name": "John"}, Notes: "new notes"}

	fields := compareEntity(old, newPerson)
	require.Len(t, fields, 1)
	assert.Equal(t, "notes", fields[0].Path)
}

func TestFormatValue_String(t *testing.T) {
	assert.Equal(t, `"hello"`, formatValue("hello"))
}

func TestFormatValue_Nil(t *testing.T) {
	assert.Equal(t, "(none)", formatValue(nil))
}

func TestFormatValue_Slice(t *testing.T) {
	val := []any{"a", "b"}
	result := formatValue(val)
	assert.Contains(t, result, `"a"`)
	assert.Contains(t, result, `"b"`)
}

func TestDiffArchives_AddedEvent(t *testing.T) {
	old := &GLXFile{
		Events: map[string]*Event{},
	}
	newArchive := &GLXFile{
		Events: map[string]*Event{
			"event-1860-census": {
				Type: "census",
				Date: "1860",
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	assert.Equal(t, ChangeAdded, result.Changes[0].Kind)
	assert.Equal(t, EntityTypeEvents, result.Changes[0].EntityType)
	assert.Contains(t, result.Changes[0].Summary, "census")
}

func TestDiffArchives_ModifiedEvent_NotesChange(t *testing.T) {
	old := &GLXFile{
		Events: map[string]*Event{
			"event-1": {Type: "census", Notes: "old"},
		},
	}
	newArchive := &GLXFile{
		Events: map[string]*Event{
			"event-1": {Type: "census", Notes: "new"},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	assert.Equal(t, ChangeModified, result.Changes[0].Kind)
	assert.Len(t, result.Changes[0].Fields, 1)
	assert.Equal(t, "notes", result.Changes[0].Fields[0].Path)
}

func TestDiffArchives_AddedRelationship(t *testing.T) {
	old := &GLXFile{
		Relationships: map[string]*Relationship{},
	}
	newArchive := &GLXFile{
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-mary", Role: "parent"},
					{Person: "person-john", Role: "child"},
				},
			},
		},
	}

	result := DiffArchives(old, newArchive,"")

	require.Len(t, result.Changes, 1)
	assert.Equal(t, ChangeAdded, result.Changes[0].Kind)
	assert.Equal(t, EntityTypeRelationships, result.Changes[0].EntityType)
	assert.Contains(t, result.Changes[0].Summary, "parent_child")
}

func TestDiffArchives_PersonFilter_IncludesRelationships(t *testing.T) {
	old := &GLXFile{
		Relationships: map[string]*Relationship{},
	}
	newArchive := &GLXFile{
		Relationships: map[string]*Relationship{
			"rel-1": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-mary", Role: "parent"},
					{Person: "person-john", Role: "child"},
				},
			},
			"rel-2": {
				Type: "parent_child",
				Participants: []Participant{
					{Person: "person-alice", Role: "parent"},
					{Person: "person-bob", Role: "child"},
				},
			},
		},
	}

	result := DiffArchives(old, newArchive,"person-mary")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "rel-1", result.Changes[0].ID)
}

func TestSummarizeModified_SingleField(t *testing.T) {
	fields := []FieldChange{
		{Path: "confidence", OldValue: `"low"`, NewValue: `"high"`},
	}
	s := summarizeModified(EntityTypeAssertions, "assertion-1", fields)
	assert.Contains(t, s, "confidence")
	assert.Contains(t, s, "→")
}

func TestSummarizeModified_MultipleFields(t *testing.T) {
	fields := []FieldChange{
		{Path: "confidence", OldValue: `"low"`, NewValue: `"high"`},
		{Path: "notes", OldValue: `"old"`, NewValue: `"new"`},
	}
	s := summarizeModified(EntityTypeAssertions, "assertion-1", fields)
	assert.Contains(t, s, "2 fields changed")
}

func TestDiffArchives_PersonFilter_StatsReflectFilteredSet(t *testing.T) {
	old := &GLXFile{
		Persons: map[string]*Person{},
	}
	newArchive := &GLXFile{
		Persons: map[string]*Person{
			"person-mary": {Properties: map[string]any{"name": "Mary"}},
			"person-john": {Properties: map[string]any{"name": "John"}},
			"person-jane": {Properties: map[string]any{"name": "Jane"}},
		},
	}

	result := DiffArchives(old, newArchive,"person-mary")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, 1, result.Stats.Added, "stats should reflect filtered set, not full archive")
}

func TestDiffArchives_ConfidenceUnknownLevel_NotCounted(t *testing.T) {
	old := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-1": {
				Subject:    EntityRef{Person: "person-mary"},
				Property:   "occupation",
				Value:      "weaver",
				Confidence: "custom_level",
			},
		},
	}
	newArchive := &GLXFile{
		Assertions: map[string]*Assertion{
			"assertion-1": {
				Subject:    EntityRef{Person: "person-mary"},
				Property:   "occupation",
				Value:      "weaver",
				Confidence: ConfidenceLevelHigh,
			},
		},
	}

	result := DiffArchives(old, newArchive,"")
	assert.Equal(t, 0, result.Stats.ConfidenceUpgrades, "unknown old confidence should not count as upgrade")
}

func TestDiffArchives_EventWithTitle(t *testing.T) {
	old := &GLXFile{Events: map[string]*Event{}}
	newArchive := &GLXFile{
		Events: map[string]*Event{
			"event-census": {
				Title: "1860 Census — Webb Household",
				Type:  "census",
				Date:  "1860",
			},
		},
	}

	result := DiffArchives(old, newArchive,"")
	require.Len(t, result.Changes, 1)
	assert.Equal(t, "1860 Census — Webb Household", result.Changes[0].Summary)
}
