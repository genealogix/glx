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

// These tests pin the backward-compatibility shim added for #667, when
// `description` moved from a top-level structural field on Source to the
// `properties.description` vocabulary property.

func TestSourceUnmarshalYAML_LegacyTopLevelDescriptionFolded(t *testing.T) {
	const doc = `
title: "Parish Register"
description: "Baptisms, marriages, burials"
`
	var src Source
	require.NoError(t, yaml.Unmarshal([]byte(doc), &src))

	assert.Equal(t, "Parish Register", src.Title)
	require.NotNil(t, src.Properties)
	assert.Equal(t, "Baptisms, marriages, burials", src.Properties["description"])
}

func TestSourceUnmarshalYAML_PropertiesDescriptionPreserved(t *testing.T) {
	const doc = `
title: "Parish Register"
properties:
  description: "From properties"
  abbreviation: "PR"
`
	var src Source
	require.NoError(t, yaml.Unmarshal([]byte(doc), &src))

	assert.Equal(t, "From properties", src.Properties["description"])
	assert.Equal(t, "PR", src.Properties["abbreviation"])
}

func TestSourceUnmarshalYAML_ExplicitPropertyWinsOverLegacy(t *testing.T) {
	const doc = `
title: "Parish Register"
description: "legacy top-level"
properties:
  description: "explicit property"
`
	var src Source
	require.NoError(t, yaml.Unmarshal([]byte(doc), &src))

	assert.Equal(t, "explicit property", src.Properties["description"],
		"an explicit properties.description must not be clobbered by the legacy field")
}

func TestSourceUnmarshalYAML_NoDescriptionLeavesPropertiesUntouched(t *testing.T) {
	const doc = `
title: "Parish Register"
type: church_register
`
	var src Source
	require.NoError(t, yaml.Unmarshal([]byte(doc), &src))

	_, ok := src.Properties["description"]
	assert.False(t, ok)
	assert.Nil(t, src.Properties, "no spurious properties map allocated when there is nothing to fold")
}

func TestSourceMarshalYAML_EmitsNoTopLevelDescription(t *testing.T) {
	src := Source{
		Title:      "Parish Register",
		Properties: map[string]any{"description": "Baptisms, marriages, burials"},
	}
	out, err := yaml.Marshal(&src)
	require.NoError(t, err)

	// Serialized form must keep description under properties, never re-emit it
	// at the top level.
	var raw map[string]any
	require.NoError(t, yaml.Unmarshal(out, &raw))
	_, hasTopLevel := raw["description"]
	assert.False(t, hasTopLevel, "marshaled source must not carry a top-level description")

	// And it round-trips back through the shim to the same location.
	var reparsed Source
	require.NoError(t, yaml.Unmarshal(out, &reparsed))
	assert.Equal(t, "Baptisms, marriages, burials", reparsed.Properties["description"])
}
