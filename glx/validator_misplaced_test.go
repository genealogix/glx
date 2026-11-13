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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A required field written inside properties is named in the schema error
// (#648).
func TestValidateGLXFileStructure_RequiredFieldInsideProperties(t *testing.T) {
	structureIssues := func(t *testing.T, yaml string) string {
		t.Helper()
		doc, err := ParseYAMLFile([]byte(yaml))
		require.NoError(t, err)

		return strings.Join(ValidateGLXFileStructure(doc), "\n")
	}

	t.Run("title inside properties", func(t *testing.T) {
		issues := structureIssues(t,
			"sources:\n  source-id:\n    properties:\n      title: My Source\n      type: vital_record\n")
		assert.Contains(t, issues, "/sources/source-id: missing property 'title' (found title inside properties; move it to the top level of the entity)")
	})

	t.Run("title absent everywhere", func(t *testing.T) {
		issues := structureIssues(t, "sources:\n  source-id:\n    properties:\n      url: https://example.org\n")
		assert.Contains(t, issues, "missing property 'title'")
		assert.NotContains(t, issues, "found title")
	})
}

func TestJSONPointerObject(t *testing.T) {
	doc := map[string]any{"a/b": map[string]any{"c~d": map[string]any{"x": 1}}}
	got, ok := jsonPointerObject(doc, "/a~1b/c~0d")
	require.True(t, ok)
	assert.Equal(t, map[string]any{"x": 1}, got)
	_, ok = jsonPointerObject(doc, "/missing")
	assert.False(t, ok)
	got, ok = jsonPointerObject(doc, "")
	require.True(t, ok)
	assert.Equal(t, doc, got)
}
