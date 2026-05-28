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

// These tests pin the backward-compatibility shim added for #894, when
// `description` moved from a top-level structural field on Media to the
// `properties.description` vocabulary property (mirroring the Source change
// from #667).

func TestMediaUnmarshalYAML_LegacyTopLevelDescriptionFolded(t *testing.T) {
	const doc = `
uri: "media/files/portrait.jpg"
title: "Portrait"
description: "Studio portrait, c.1880"
`
	var m Media
	require.NoError(t, yaml.Unmarshal([]byte(doc), &m))

	assert.Equal(t, "Portrait", m.Title)
	require.NotNil(t, m.Properties)
	assert.Equal(t, "Studio portrait, c.1880", m.Properties["description"])
}

func TestMediaUnmarshalYAML_PropertiesDescriptionPreserved(t *testing.T) {
	const doc = `
uri: "media/files/portrait.jpg"
title: "Portrait"
properties:
  description: "From properties"
  original_filename: "portrait.jpg"
`
	var m Media
	require.NoError(t, yaml.Unmarshal([]byte(doc), &m))

	assert.Equal(t, "From properties", m.Properties["description"])
	assert.Equal(t, "portrait.jpg", m.Properties["original_filename"])
}

func TestMediaUnmarshalYAML_ExplicitPropertyWinsOverLegacy(t *testing.T) {
	const doc = `
uri: "media/files/portrait.jpg"
title: "Portrait"
description: "legacy top-level"
properties:
  description: "explicit property"
`
	var m Media
	require.NoError(t, yaml.Unmarshal([]byte(doc), &m))

	assert.Equal(t, "explicit property", m.Properties["description"],
		"an explicit properties.description must not be clobbered by the legacy field")
}

func TestMediaUnmarshalYAML_NoDescriptionLeavesPropertiesUntouched(t *testing.T) {
	const doc = `
uri: "media/files/portrait.jpg"
title: "Portrait"
type: photograph
`
	var m Media
	require.NoError(t, yaml.Unmarshal([]byte(doc), &m))

	_, ok := m.Properties["description"]
	assert.False(t, ok)
	assert.Nil(t, m.Properties, "no spurious properties map allocated when there is nothing to fold")
}

func TestMediaMarshalYAML_EmitsNoTopLevelDescription(t *testing.T) {
	m := Media{
		URI:        "media/files/portrait.jpg",
		Title:      "Portrait",
		Properties: map[string]any{"description": "Studio portrait, c.1880"},
	}
	out, err := yaml.Marshal(&m)
	require.NoError(t, err)

	// Serialized form must keep description under properties, never re-emit it
	// at the top level.
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(out, &raw))
	_, hasTopLevel := raw["description"]
	assert.False(t, hasTopLevel, "marshaled media must not carry a top-level description")

	// And it round-trips back through the shim to the same location.
	var reparsed Media
	require.NoError(t, yaml.Unmarshal(out, &reparsed))
	assert.Equal(t, "Studio portrait, c.1880", reparsed.Properties["description"])
}
