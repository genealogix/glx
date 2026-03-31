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
	"gopkg.in/yaml.v3"
)

func TestNoteList_UnmarshalScalar(t *testing.T) {
	input := `notes: "This is a single note"`
	var out struct {
		Notes NoteList `yaml:"notes"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(input), &out))
	assert.Equal(t, NoteList{"This is a single note"}, out.Notes)
}

func TestNoteList_UnmarshalSequence(t *testing.T) {
	input := "notes:\n  - \"First note\"\n  - \"Second note\""
	var out struct {
		Notes NoteList `yaml:"notes"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(input), &out))
	assert.Equal(t, NoteList{"First note", "Second note"}, out.Notes)
}

func TestNoteList_UnmarshalEmpty(t *testing.T) {
	input := `name: "test"`
	var out struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(input), &out))
	assert.Nil(t, out.Notes)
}

func TestNoteList_MarshalSingle(t *testing.T) {
	in := struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}{
		Notes: NoteList{"Single note"},
	}
	data, err := yaml.Marshal(&in)
	require.NoError(t, err)
	assert.Contains(t, string(data), "notes: Single note")
	assert.NotContains(t, string(data), "- ")
}

func TestNoteList_MarshalMultiple(t *testing.T) {
	in := struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}{
		Notes: NoteList{"First", "Second"},
	}
	data, err := yaml.Marshal(&in)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "- First")
	assert.Contains(t, s, "- Second")
}

func TestNoteList_MarshalEmpty(t *testing.T) {
	in := struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}{}
	data, err := yaml.Marshal(&in)
	require.NoError(t, err)
	assert.Equal(t, "{}\n", string(data))
}

func TestNoteList_RoundtripSingle(t *testing.T) {
	original := struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}{
		Notes: NoteList{"A single note with special chars: <>&"},
	}
	data, err := yaml.Marshal(&original)
	require.NoError(t, err)

	var roundtripped struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}
	require.NoError(t, yaml.Unmarshal(data, &roundtripped))
	assert.Equal(t, original.Notes, roundtripped.Notes)
}

func TestNoteList_RoundtripMultiple(t *testing.T) {
	original := struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}{
		Notes: NoteList{"First note", "Second note", "Third note"},
	}
	data, err := yaml.Marshal(&original)
	require.NoError(t, err)

	var roundtripped struct {
		Notes NoteList `yaml:"notes,omitempty"`
	}
	require.NoError(t, yaml.Unmarshal(data, &roundtripped))
	assert.Equal(t, original.Notes, roundtripped.Notes)
}

func TestNoteList_String(t *testing.T) {
	n := NoteList{"First", "Second", "Third"}
	assert.Equal(t, "First\n\nSecond\n\nThird", n.String())
}

func TestNoteList_IsEmpty(t *testing.T) {
	assert.True(t, NoteList{}.IsEmpty())
	assert.True(t, NoteList(nil).IsEmpty())
	assert.False(t, NoteList{"note"}.IsEmpty())
}
